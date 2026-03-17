import 'dart:async';
import 'dart:io';
import 'dart:math';

/// Starts real backend components for Flutter E2E integration tests.
///
/// Manages three subprocesses:
/// 1. **testserver** (Go) — gRPC server with MockChatModel
/// 2. **bridge** (Go) — WebSocket relay
/// 3. **headless-bridge-server** (Bun/TS) — CLI container that connects to
///    both testserver (gRPC) and bridge (WS), handles mobile requests
///
/// The CLI generates a pairing token on startup. The Flutter app connects to
/// the bridge as a mobile device and communicates through the full chain:
/// ```
/// Flutter App --WS--> Bridge --WS--> CLI --gRPC--> Server(MockLLM)
/// ```
///
/// ## Prerequisites
///
/// Requires pre-built binaries. Call [buildAll] once in `setUpAll`:
/// - `bytebrew-srv/bin/testserver[.exe]`
/// - `bytebrew-bridge/bin/bridge[.exe]`
/// - Bun runtime on PATH
///
/// ## Usage
///
/// ```dart
/// late BackendFixture backend;
///
/// setUpAll(() async {
///   await BackendFixture.buildAll();
///   backend = BackendFixture();
///   await backend.start(scenario: 'echo');
/// });
///
/// tearDownAll(() async {
///   await backend.stop();
/// });
///
/// testWidgets('sends message through real chain', (tester) async {
///   await tester.runAsync(() async {
///     // Use backend.bridgeUrl, backend.serverId, backend.pairingToken
///     // to configure the app with real backend connectivity.
///   });
/// });
/// ```
class BackendFixture {
  Process? _serverProcess;
  Process? _bridgeProcess;
  Process? _cliProcess;

  int _serverPort = 0;
  int _bridgePort = 0;
  String _bridgeAuthToken = '';
  String _serverId = '';
  String _sessionId = '';
  String _pairingToken = '';

  /// The gRPC port of the testserver.
  int get serverPort => _serverPort;

  /// The HTTP/WS port of the bridge relay.
  int get bridgePort => _bridgePort;

  /// UUID identifying this CLI instance on the bridge.
  String get serverId => _serverId;

  /// The active session ID from the CLI container.
  String get sessionId => _sessionId;

  /// WebSocket URL for the bridge relay.
  String get bridgeUrl => 'ws://localhost:$_bridgePort';

  /// Pre-generated pairing token for mobile device registration.
  String get pairingToken => _pairingToken;

  // ---------------------------------------------------------------------------
  // Path resolution
  // ---------------------------------------------------------------------------

  /// Returns the mono-repo root directory.
  ///
  /// In Flutter integration tests, [Directory.current] is typically
  /// `bytebrew-mobile-app/`. The mono-repo root is one level up.
  static String _monoRepoRoot() {
    final dir = Directory.current.path;
    if (dir.endsWith('bytebrew-mobile-app')) {
      return Directory(dir).parent.path;
    }
    // Fallback: assume we're already at the root
    return dir;
  }

  static String _ext() => Platform.isWindows ? '.exe' : '';

  static String _serverBinaryPath() =>
      '${_monoRepoRoot()}/bytebrew-srv/bin/testserver${_ext()}';

  static String _bridgeBinaryPath() =>
      '${_monoRepoRoot()}/bytebrew-bridge/bin/bridge${_ext()}';

  static String _cliDir() => '${_monoRepoRoot()}/bytebrew-cli';

  // ---------------------------------------------------------------------------
  // Build
  // ---------------------------------------------------------------------------

  /// Builds all required binaries. Call once in `setUpAll`.
  ///
  /// Compiles testserver and bridge Go binaries. Skips if the binary already
  /// exists (delete the binary to force a rebuild).
  ///
  /// Typical timeout: 60-120 seconds for first build.
  static Future<void> buildAll() async {
    final root = _monoRepoRoot();

    await Future.wait([
      _buildGoBinary(
        workingDir: '$root/bytebrew-srv',
        outputPath: 'bin/testserver${_ext()}',
        packagePath: './cmd/testserver',
        label: 'testserver',
      ),
      _buildGoBinary(
        workingDir: '$root/bytebrew-bridge',
        outputPath: 'bin/bridge${_ext()}',
        packagePath: './cmd/bridge',
        label: 'bridge',
      ),
    ]);
  }

  static Future<void> _buildGoBinary({
    required String workingDir,
    required String outputPath,
    required String packagePath,
    required String label,
  }) async {
    final fullPath = '$workingDir/$outputPath';
    if (File(fullPath).existsSync()) {
      return;
    }

    // Ensure bin/ directory exists
    final binDir = File(fullPath).parent;
    if (!binDir.existsSync()) {
      binDir.createSync(recursive: true);
    }

    final result = await Process.run('go', [
      'build',
      '-o',
      outputPath,
      packagePath,
    ], workingDirectory: workingDir);

    if (result.exitCode != 0) {
      throw Exception(
        'Failed to build $label (exit ${result.exitCode}):\n${result.stderr}',
      );
    }
  }

  // ---------------------------------------------------------------------------
  // Start / Stop
  // ---------------------------------------------------------------------------

  /// Starts all backend components with the given [scenario].
  ///
  /// Available scenarios (defined in `bytebrew-srv/cmd/testserver`):
  /// `echo`, `server-tool`, `reasoning`, `error`, `ask-user`,
  /// `multi-agent`, `cancel-during-stream`, etc.
  ///
  /// Resolves when all three processes are ready and the CLI has registered
  /// with the bridge and generated a pairing token.
  Future<void> start({String scenario = 'echo'}) async {
    _serverId = _generateUuidV4();
    _bridgeAuthToken = _generateUuidV4();

    // 1. Start testserver (gRPC)
    await _startServer(scenario);

    // 2. Start bridge relay (WS)
    await _startBridge();

    // 3. Start CLI headless bridge server (connects to both)
    await _startCli();
  }

  /// Stops all backend components in reverse order.
  ///
  /// Sends SIGTERM (or `kill()` on Windows), waits up to 3 seconds per
  /// process, then force-kills if needed.
  Future<void> stop() async {
    // Stop in reverse order: CLI -> Bridge -> Server
    await _killProcess(_cliProcess, 'cli');
    _cliProcess = null;

    await _killProcess(_bridgeProcess, 'bridge');
    _bridgeProcess = null;

    await _killProcess(_serverProcess, 'testserver');
    _serverProcess = null;
  }

  // ---------------------------------------------------------------------------
  // Private: Start individual processes
  // ---------------------------------------------------------------------------

  Future<void> _startServer(String scenario) async {
    final binary = _serverBinaryPath();
    if (!File(binary).existsSync()) {
      throw Exception(
        'testserver binary not found at $binary. '
        'Call BackendFixture.buildAll() first.',
      );
    }

    _serverProcess = await Process.start(binary, [
      '--scenario',
      scenario,
      '--port',
      '0',
    ]);

    final completer = Completer<void>();
    final buffer = StringBuffer();

    _serverProcess!.stdout.transform(const SystemEncoding().decoder).listen((
      data,
    ) {
      buffer.write(data);
      final match = RegExp(r'READY:(\d+)').firstMatch(buffer.toString());
      if (match != null && !completer.isCompleted) {
        _serverPort = int.parse(match.group(1)!);
        completer.complete();
      }
    });

    _serverProcess!.stderr.transform(const SystemEncoding().decoder).listen((
      data,
    ) {
      stderr.write('[testserver] $data');
    });

    _serverProcess!.exitCode.then((code) {
      if (!completer.isCompleted) {
        completer.completeError(
          Exception('testserver exited with code $code before READY'),
        );
      }
    });

    await completer.future.timeout(
      const Duration(seconds: 30),
      onTimeout: () {
        throw TimeoutException('testserver did not emit READY within 30s');
      },
    );
  }

  Future<void> _startBridge() async {
    final binary = _bridgeBinaryPath();
    if (!File(binary).existsSync()) {
      throw Exception(
        'bridge binary not found at $binary. '
        'Call BackendFixture.buildAll() first.',
      );
    }

    // Find a free ephemeral port (bridge rejects port 0)
    _bridgePort = await _findFreePort();

    _bridgeProcess = await Process.start(
      binary,
      [],
      environment: {
        ...Platform.environment,
        'BRIDGE_PORT': '$_bridgePort',
        'BRIDGE_AUTH_TOKEN': _bridgeAuthToken,
      },
    );

    _bridgeProcess!.stderr.transform(const SystemEncoding().decoder).listen((
      data,
    ) {
      stderr.write('[bridge] $data');
    });

    _bridgeProcess!.exitCode.then((code) {
      // Only log if bridge exits unexpectedly during test
      if (_bridgeProcess != null) {
        stderr.writeln('[bridge] exited with code $code');
      }
    });

    // Poll /health until the bridge is ready
    await _waitForHealth(
      'http://localhost:$_bridgePort/health',
      timeout: const Duration(seconds: 15),
    );
  }

  Future<void> _startCli() async {
    final cliDir = _cliDir();
    final scriptPath = '$cliDir/src/test-utils/headless-bridge-server.ts';

    if (!File(scriptPath).existsSync()) {
      throw Exception('headless-bridge-server.ts not found at $scriptPath');
    }

    // Use bun to run the TypeScript script directly
    final bunExe = Platform.isWindows ? 'bun.exe' : 'bun';

    _cliProcess = await Process.start(bunExe, [
      scriptPath,
      '--server-port',
      '$_serverPort',
      '--bridge-port',
      '$_bridgePort',
      '--bridge-auth-token',
      _bridgeAuthToken,
      '--server-id',
      _serverId,
    ], workingDirectory: cliDir);

    // Parse READY:{serverId}:{sessionId}:{pairingToken} from stdout
    final completer = Completer<void>();
    final buffer = StringBuffer();

    _cliProcess!.stdout.transform(const SystemEncoding().decoder).listen((
      data,
    ) {
      buffer.write(data);
      final lines = buffer.toString().split('\n');
      for (final line in lines) {
        final match = RegExp(
          r'READY:([^:]+):([^:]+):(.+)',
        ).firstMatch(line.trim());
        if (match != null && !completer.isCompleted) {
          _serverId = match.group(1)!;
          _sessionId = match.group(2)!;
          _pairingToken = match.group(3)!;
          completer.complete();
        }
      }
    });

    _cliProcess!.stderr.transform(const SystemEncoding().decoder).listen((
      data,
    ) {
      stderr.write('[cli] $data');
    });

    _cliProcess!.exitCode.then((code) {
      if (!completer.isCompleted) {
        completer.completeError(
          Exception('CLI exited with code $code before READY'),
        );
      }
    });

    await completer.future.timeout(
      const Duration(seconds: 30),
      onTimeout: () {
        throw TimeoutException(
          'CLI headless-bridge-server did not emit READY within 30s',
        );
      },
    );
  }

  // ---------------------------------------------------------------------------
  // Private: Helpers
  // ---------------------------------------------------------------------------

  static Future<void> _killProcess(Process? process, String label) async {
    if (process == null) return;

    process.kill(); // SIGTERM on Unix, terminates on Windows

    try {
      await process.exitCode.timeout(
        const Duration(seconds: 3),
        onTimeout: () {
          process.kill(ProcessSignal.sigkill);
          return -1;
        },
      );
    } catch (_) {
      // Process already dead or SIGKILL not supported (Windows)
    }
  }

  /// Finds a free ephemeral port by binding to port 0, reading the assigned
  /// port, then closing the listener.
  ///
  /// There is a small race window between closing and the bridge binding,
  /// but for local tests this is acceptable (same approach as
  /// `bytebrew-cli/src/test-utils/BridgeHelper.ts`).
  static Future<int> _findFreePort() async {
    final server = await ServerSocket.bind(InternetAddress.loopbackIPv4, 0);
    final port = server.port;
    await server.close();
    return port;
  }

  /// Polls a health endpoint until it returns HTTP 200 or timeout.
  static Future<void> _waitForHealth(
    String url, {
    Duration timeout = const Duration(seconds: 15),
  }) async {
    final deadline = DateTime.now().add(timeout);
    final client = HttpClient();

    try {
      while (DateTime.now().isBefore(deadline)) {
        try {
          final request = await client.getUrl(Uri.parse(url));
          final response = await request.close();
          await response.drain<void>();
          if (response.statusCode == 200) {
            return;
          }
        } catch (_) {
          // Connection refused — bridge not ready yet
        }
        await Future<void>.delayed(const Duration(milliseconds: 100));
      }

      throw TimeoutException('Health check timeout for $url');
    } finally {
      client.close();
    }
  }

  /// Generates a UUID v4 string using [Random.secure].
  static String _generateUuidV4() {
    final random = Random.secure();
    final bytes = List<int>.generate(16, (_) => random.nextInt(256));

    // Set version (4) and variant (10xx) bits per RFC 4122
    bytes[6] = (bytes[6] & 0x0F) | 0x40; // version 4
    bytes[8] = (bytes[8] & 0x3F) | 0x80; // variant 10xx

    final hex = bytes.map((b) => b.toRadixString(16).padLeft(2, '0')).join();

    return '${hex.substring(0, 8)}-'
        '${hex.substring(8, 12)}-'
        '${hex.substring(12, 16)}-'
        '${hex.substring(16, 20)}-'
        '${hex.substring(20, 32)}';
  }
}
