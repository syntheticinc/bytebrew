import 'dart:io';

/// Temporary file logger for debugging transport chain on real device.
/// Writes to a file that can be read externally.
class DebugFileLogger {
  DebugFileLogger._();

  static final _instance = DebugFileLogger._();
  static DebugFileLogger get instance => _instance;

  File? _file;
  IOSink? _sink;

  void init() {
    // Write to app-accessible location
    final dir = Directory.systemTemp;
    _file = File('${dir.path}/bytebrew_debug.log');
    // Truncate on init
    _sink = _file!.openWrite(mode: FileMode.write);
    _sink!.writeln('[${_ts()}] DebugFileLogger initialized: ${_file!.path}');
    // Also print the path so we know where to look
    print('[DebugFileLogger] Writing to: ${_file!.path}');
  }

  void log(String msg) {
    final line = '[${_ts()}] $msg';
    _sink?.writeln(line);
    // NOTE: Do NOT call flush() here — it calls addStream() internally which
    // locks the IOSink. If another writeln() arrives before flush completes
    // (common during fast WS streaming), it throws:
    //   "Bad state: StreamSink is bound to a stream"
    // IOSink auto-flushes on its own schedule. For forced flush, use dispose().
    print(line);
  }

  String get filePath => _file?.path ?? 'not initialized';

  static String _ts() {
    final now = DateTime.now();
    return '${now.hour.toString().padLeft(2, '0')}:'
        '${now.minute.toString().padLeft(2, '0')}:'
        '${now.second.toString().padLeft(2, '0')}.'
        '${now.millisecond.toString().padLeft(3, '0')}';
  }
}

/// Shorthand
void dlog(String msg) => DebugFileLogger.instance.log(msg);
