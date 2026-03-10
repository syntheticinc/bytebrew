import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/domain/session.dart';
import '../../../core/infrastructure/ws/ws_connection.dart';
import '../../../core/infrastructure/ws/ws_connection_manager.dart';
import '../../../core/infrastructure/ws/ws_providers.dart';
import '../../../core/theme/app_colors.dart';
import '../../../core/widgets/status_indicator.dart';
import '../../settings/application/settings_provider.dart';
import '../application/auto_connect_provider.dart';
import '../application/sessions_provider.dart';
import 'widgets/session_group.dart';

/// The order in which session status groups are displayed.
const _statusDisplayOrder = [
  SessionStatus.needsAttention,
  SessionStatus.active,
  SessionStatus.idle,
];

/// Main screen showing all agent sessions grouped by status.
class SessionsScreen extends ConsumerWidget {
  const SessionsScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    // Trigger auto-connect to all paired servers on first build.
    ref.watch(sessionsAutoConnectProvider);

    final sessionsAsync = ref.watch(sessionsProvider);
    final grouped = ref.watch(groupedSessionsProvider);
    final manager = ref.watch(connectionManagerProvider);
    final servers = ref.watch(serversProvider);
    final hasActiveConnection = manager.activeConnections.isNotEmpty;

    return Scaffold(
      appBar: AppBar(title: const Text('Activity')),
      body: RefreshIndicator(
        onRefresh: () => ref.read(sessionsProvider.notifier).refresh(),
        child: sessionsAsync.when(
          loading: () => const Center(child: CircularProgressIndicator()),
          error: (error, _) => _ErrorBody(
            message: error.toString(),
            onRetry: () => ref.read(sessionsProvider.notifier).refresh(),
          ),
          data: (_) {
            if (grouped.isEmpty && !hasActiveConnection) {
              return const _EmptyBody();
            }

            // Resolve active server name for the status bar.
            // Prefer resolvedName (from ping) over persisted server.name.
            final activeConn = manager.activeConnections.firstOrNull;
            final activeServerName =
                activeConn?.resolvedName ?? activeConn?.server.name;

            if (grouped.isEmpty && hasActiveConnection) {
              return _ConnectedEmptyBody(serverName: activeServerName ?? 'Server');
            }

            final allSessions = grouped.values.expand((s) => s).toList();
            final serverIds = allSessions.map((s) => s.serverId).toSet();
            final multipleServers = serverIds.length > 1 || servers.length > 1;

            if (multipleServers) {
              return _ServerGroupedList(
                grouped: grouped,
                serverName: activeServerName,
                manager: manager,
              );
            }

            return _SessionsList(
              grouped: grouped,
              serverName: activeServerName,
            );
          },
        ),
      ),
    );
  }
}

/// Scrollable list of session groups ordered by [_statusDisplayOrder],
/// with a compact summary header showing counts per status.
class _SessionsList extends StatelessWidget {
  const _SessionsList({
    required this.grouped,
    this.serverName,
  });

  final Map<SessionStatus, List<Session>> grouped;
  final String? serverName;

  @override
  Widget build(BuildContext context) {
    final visibleStatuses = _statusDisplayOrder
        .where((s) => grouped.containsKey(s))
        .toList();

    final hasStatus = serverName != null;
    final extraItems = hasStatus ? 1 : 0;

    return ListView.builder(
      itemCount: visibleStatuses.length + extraItems,
      itemBuilder: (context, index) {
        if (hasStatus && index == 0) {
          return _ConnectionStatusBar(serverName: serverName!);
        }
        final adjustedIndex = index - extraItems;
        final status = visibleStatuses[adjustedIndex];
        return SessionGroup(status: status, sessions: grouped[status]!);
      },
    );
  }
}

/// Sessions list grouped by server name, each with an online/offline badge.
class _ServerGroupedList extends StatelessWidget {
  const _ServerGroupedList({
    required this.grouped,
    this.serverName,
    required this.manager,
  });

  final Map<SessionStatus, List<Session>> grouped;
  final String? serverName;
  final WsConnectionManager manager;

  @override
  Widget build(BuildContext context) {
    // Flatten all sessions then group by serverId.
    final allSessions = <Session>[];
    for (final status in _statusDisplayOrder) {
      final sessions = grouped[status];
      if (sessions != null) {
        allSessions.addAll(sessions);
      }
    }

    final byServer = <String, List<Session>>{};
    for (final session in allSessions) {
      byServer.putIfAbsent(session.serverId, () => []).add(session);
    }

    final serverIds = byServer.keys.toList();

    // Build a flat list of widgets: (live status), then server headers + groups.
    final widgets = <Widget>[];

    if (serverName != null) {
      widgets.add(_ConnectionStatusBar(serverName: serverName!));
    }

    for (final serverId in serverIds) {
      final sessions = byServer[serverId]!;
      final serverName = sessions.first.serverName;
      final connection = manager.getConnection(serverId);
      final isOnline = connection != null &&
          connection.status == WsConnectionStatus.connected;

      widgets.add(
        _ServerHeader(serverName: serverName, isOnline: isOnline),
      );

      // Group this server's sessions by status.
      final serverGrouped = <SessionStatus, List<Session>>{};
      for (final session in sessions) {
        serverGrouped.putIfAbsent(session.status, () => []).add(session);
      }
      for (final status in _statusDisplayOrder) {
        final statusSessions = serverGrouped[status];
        if (statusSessions != null) {
          widgets.add(
            SessionGroup(status: status, sessions: statusSessions),
          );
        }
      }
    }

    return ListView(children: widgets);
  }
}

/// Server name header with online/offline status badge.
class _ServerHeader extends StatelessWidget {
  const _ServerHeader({required this.serverName, required this.isOnline});

  final String serverName;
  final bool isOnline;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final statusColor = isOnline ? AppColors.statusActive : AppColors.shade3;

    return Padding(
      padding: const EdgeInsets.fromLTRB(16, 16, 16, 4),
      child: Row(
        children: [
          StatusIndicator(color: statusColor, size: 8),
          const SizedBox(width: 8),
          Text(
            serverName,
            style: theme.textTheme.titleSmall?.copyWith(
              fontWeight: FontWeight.w600,
            ),
          ),
        ],
      ),
    );
  }
}

/// Compact connection status bar shown at the top of the sessions list
/// when there is an active server connection.
class _ConnectionStatusBar extends StatelessWidget {
  const _ConnectionStatusBar({required this.serverName});

  final String serverName;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return Padding(
      padding: const EdgeInsets.fromLTRB(16, 12, 16, 8),
      child: Row(
        children: [
          const Icon(Icons.circle, color: AppColors.statusActive, size: 8),
          const SizedBox(width: 8),
          Expanded(
            child: Text(
              serverName,
              style: theme.textTheme.bodySmall?.copyWith(
                color: AppColors.shade3,
              ),
              overflow: TextOverflow.ellipsis,
            ),
          ),
        ],
      ),
    );
  }
}

/// Empty state shown when there are no sessions.
class _EmptyBody extends StatelessWidget {
  const _EmptyBody();

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return CustomScrollView(
      slivers: [
        SliverFillRemaining(
          child: Center(
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                Icon(
                  Icons.chat_bubble_outline,
                  size: 64,
                  color: theme.colorScheme.onSurfaceVariant.withValues(
                    alpha: 0.5,
                  ),
                ),
                const SizedBox(height: 16),
                Text(
                  'No sessions yet',
                  style: theme.textTheme.titleMedium?.copyWith(
                    color: theme.colorScheme.onSurfaceVariant,
                  ),
                ),
                const SizedBox(height: 4),
                Text(
                  'Your agent sessions will appear here',
                  style: theme.textTheme.bodySmall?.copyWith(
                    color: theme.colorScheme.onSurfaceVariant.withValues(
                      alpha: 0.7,
                    ),
                  ),
                ),
              ],
            ),
          ),
        ),
      ],
    );
  }
}

/// Empty state shown when connected to a server but no sessions exist yet.
class _ConnectedEmptyBody extends StatelessWidget {
  const _ConnectedEmptyBody({required this.serverName});

  final String serverName;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return CustomScrollView(
      slivers: [
        SliverFillRemaining(
          child: Center(
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                Row(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    const Icon(
                      Icons.circle,
                      color: AppColors.statusActive,
                      size: 8,
                    ),
                    const SizedBox(width: 8),
                    Text(
                      serverName,
                      style: theme.textTheme.bodySmall?.copyWith(
                        color: AppColors.shade3,
                      ),
                    ),
                  ],
                ),
                const SizedBox(height: 24),
                Icon(
                  Icons.chat_bubble_outline,
                  size: 64,
                  color: theme.colorScheme.onSurfaceVariant.withValues(
                    alpha: 0.5,
                  ),
                ),
                const SizedBox(height: 16),
                Text(
                  'No active sessions',
                  style: theme.textTheme.titleMedium?.copyWith(
                    color: theme.colorScheme.onSurfaceVariant,
                  ),
                ),
                const SizedBox(height: 4),
                Text(
                  'Start a session from CLI or send a message',
                  style: theme.textTheme.bodySmall?.copyWith(
                    color: theme.colorScheme.onSurfaceVariant.withValues(
                      alpha: 0.7,
                    ),
                  ),
                ),
              ],
            ),
          ),
        ),
      ],
    );
  }
}

/// Error state with a message and retry button.
class _ErrorBody extends StatelessWidget {
  const _ErrorBody({required this.message, required this.onRetry});

  final String message;
  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return CustomScrollView(
      slivers: [
        SliverFillRemaining(
          child: Center(
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                Text(
                  message,
                  style: theme.textTheme.bodyMedium?.copyWith(
                    color: theme.colorScheme.error,
                  ),
                  textAlign: TextAlign.center,
                ),
                const SizedBox(height: 16),
                FilledButton.tonal(
                  onPressed: onRetry,
                  child: const Text('Retry'),
                ),
              ],
            ),
          ),
        ),
      ],
    );
  }
}
