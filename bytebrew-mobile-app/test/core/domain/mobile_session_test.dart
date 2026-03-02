import 'package:bytebrew_mobile/core/domain/mobile_session.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('MobileSession.projectName', () {
    MobileSession _make({
      String projectRoot = '',
      String projectKey = 'fallback-key',
    }) {
      return MobileSession(
        sessionId: 'session-1',
        projectKey: projectKey,
        projectRoot: projectRoot,
        status: MobileSessionState.active,
        currentTask: '',
        startedAt: DateTime(2026, 1, 1),
        lastActivityAt: DateTime(2026, 1, 1),
        hasAskUser: false,
        platform: 'linux',
      );
    }

    test('returns projectKey when projectRoot is empty', () {
      final session = _make(projectRoot: '', projectKey: 'my-project');

      expect(session.projectName, 'my-project');
    });

    test('extracts last segment from Unix path', () {
      final session = _make(projectRoot: '/home/user/projects/bytebrew');

      expect(session.projectName, 'bytebrew');
    });

    test('extracts last segment from deeply nested Unix path', () {
      final session = _make(projectRoot: '/var/lib/data/apps/my-app');

      expect(session.projectName, 'my-app');
    });

    test('normalizes Windows backslash path and extracts last segment', () {
      final session = _make(
        projectRoot: r'C:\Users\dev\Projects\bytebrew-srv',
      );

      expect(session.projectName, 'bytebrew-srv');
    });

    test('handles Windows path with mixed separators', () {
      final session = _make(
        projectRoot: r'C:\Users\dev/Projects/my-app',
      );

      expect(session.projectName, 'my-app');
    });

    test('handles path with trailing slash', () {
      final session = _make(projectRoot: '/home/user/projects/bytebrew/');

      expect(session.projectName, 'bytebrew');
    });

    test('handles Windows path with trailing backslash', () {
      final session = _make(
        projectRoot: r'C:\Users\dev\Projects\app\',
      );

      expect(session.projectName, 'app');
    });

    test('handles single segment Unix path', () {
      final session = _make(projectRoot: '/root');

      expect(session.projectName, 'root');
    });

    test('handles single segment without leading slash', () {
      final session = _make(projectRoot: 'standalone-folder');

      expect(session.projectName, 'standalone-folder');
    });

    test('returns projectKey when path is only slashes', () {
      final session = _make(projectRoot: '/', projectKey: 'fallback');

      expect(session.projectName, 'fallback');
    });

    test('returns projectKey when path is only backslash', () {
      final session = _make(projectRoot: r'\', projectKey: 'fallback');

      expect(session.projectName, 'fallback');
    });

    test('handles path with multiple trailing slashes equivalent', () {
      // After split and filter, trailing slashes leave no extra segments.
      final session = _make(projectRoot: '/home/user/');

      expect(session.projectName, 'user');
    });
  });

  group('MobileSessionState', () {
    test('has all expected values', () {
      expect(MobileSessionState.values, containsAll([
        MobileSessionState.unspecified,
        MobileSessionState.active,
        MobileSessionState.idle,
        MobileSessionState.needsAttention,
        MobileSessionState.completed,
        MobileSessionState.failed,
      ]));
    });
  });

  group('MobileSession constructor', () {
    test('stores all fields correctly', () {
      final startedAt = DateTime(2026, 3, 1, 10, 0);
      final lastActivity = DateTime(2026, 3, 1, 10, 30);

      final session = MobileSession(
        sessionId: 'sess-abc',
        projectKey: 'proj-key',
        projectRoot: '/home/dev/project',
        status: MobileSessionState.needsAttention,
        currentTask: 'Analyzing code',
        startedAt: startedAt,
        lastActivityAt: lastActivity,
        hasAskUser: true,
        platform: 'darwin',
      );

      expect(session.sessionId, 'sess-abc');
      expect(session.projectKey, 'proj-key');
      expect(session.projectRoot, '/home/dev/project');
      expect(session.status, MobileSessionState.needsAttention);
      expect(session.currentTask, 'Analyzing code');
      expect(session.startedAt, startedAt);
      expect(session.lastActivityAt, lastActivity);
      expect(session.hasAskUser, isTrue);
      expect(session.platform, 'darwin');
    });
  });
}
