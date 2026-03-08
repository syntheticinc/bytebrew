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

            final allSessions = grouped.values.expand((s) => s).toList();
            final serverIds = allSessions.map((s) => s.serverId).toSet();
            final multipleServers = serverIds.length > 1 || servers.length > 1;

            if (multipleServers) {
              return _ServerGroupedList(
                grouped: grouped,
                showLiveSession: hasActiveConnection,
                manager: manager,
              );
            }

            return _SessionsList(
              grouped: grouped,
              showLiveSession: hasActiveConnection,
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
  const _SessionsList({required this.grouped, this.showLiveSession = false});

  final Map<SessionStatus, List<Session>> grouped;
  final bool showLiveSession;

  @override
  Widget build(BuildContext context) {
    final visibleStatuses = _statusDisplayOrder
        .where((s) => grouped.containsKey(s))
        .toList();

    final extraItems = showLiveSession ? 1 : 0;

    // +1 for summary header, +extraItems for live session card.
    return ListView.builder(
      itemCount: visibleStatuses.length + 1 + extraItems,
      itemBuilder: (context, index) {
        if (index == 0) {
          return _SummaryHeader(grouped: grouped);
        }
        if (showLiveSession && index == 1) {
          return const _LiveSessionCard();
        }
        final adjustedIndex = index - 1 - extraItems;
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
    this.showLiveSession = false,
    required this.manager,
  });

  final Map<SessionStatus, List<Session>> grouped;
  final bool showLiveSession;
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

    // Build a flat list of widgets: summary, (live), then server headers + groups.
    final widgets = <Widget>[
      _SummaryHeader(grouped: grouped),
    ];

    if (showLiveSession) {
      widgets.add(const _LiveSessionCard());
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

/// Section header + status line shown when there is an active CLI connection.
///
/// Uses the same monospace uppercase header style as [SessionGroup] headers
/// (e.g. "IN PROGRESS", "ACTION REQUIRED") for visual consistency.
class _LiveSessionCard extends StatelessWidget {
  const _LiveSessionCard();

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Padding(
          padding: const EdgeInsets.fromLTRB(16, 12, 16, 4),
          child: Text(
            'LIVE',
            style: theme.textTheme.labelSmall?.copyWith(
              fontWeight: FontWeight.w600,
              letterSpacing: 2,
              color: AppColors.statusActive,
            ),
          ),
        ),
        Padding(
          padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 2),
          child: Row(
            children: [
              Icon(Icons.circle, color: AppColors.statusActive, size: 8),
              const SizedBox(width: 8),
              Text(
                'Connected to CLI',
                style: theme.textTheme.bodySmall?.copyWith(
                  color: AppColors.shade3,
                ),
              ),
            ],
          ),
        ),
      ],
    );
  }
}

/// Compact summary row: "2 active . 1 needs attention . 3 idle"
/// with colored counts.
class _SummaryHeader extends StatelessWidget {
  const _SummaryHeader({required this.grouped});

  final Map<SessionStatus, List<Session>> grouped;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;

    final activeCount = grouped[SessionStatus.active]?.length ?? 0;
    final attentionCount = grouped[SessionStatus.needsAttention]?.length ?? 0;
    final idleCount = grouped[SessionStatus.idle]?.length ?? 0;

    final spans = <InlineSpan>[];

    void addSpan(int count, String label, Color color) {
      if (spans.isNotEmpty) {
        spans.add(
          TextSpan(
            text: '  \u00B7  ',
            style: theme.textTheme.bodySmall?.copyWith(
              color: colorScheme.onSurfaceVariant,
            ),
          ),
        );
      }
      spans.add(
        TextSpan(
          text: '$count',
          style: theme.textTheme.bodySmall?.copyWith(
            color: color,
            fontWeight: FontWeight.w600,
          ),
        ),
      );
      spans.add(
        TextSpan(
          text: ' $label',
          style: theme.textTheme.bodySmall?.copyWith(
            color: colorScheme.onSurfaceVariant,
          ),
        ),
      );
    }

    if (activeCount > 0) {
      addSpan(activeCount, 'active', AppColors.statusActive);
    }
    if (attentionCount > 0) {
      addSpan(
        attentionCount,
        'needs attention',
        AppColors.statusNeedsAttention,
      );
    }
    if (idleCount > 0) {
      addSpan(idleCount, 'idle', AppColors.statusIdle);
    }

    return Padding(
      padding: const EdgeInsets.fromLTRB(16, 12, 16, 0),
      child: Text.rich(TextSpan(children: spans)),
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
