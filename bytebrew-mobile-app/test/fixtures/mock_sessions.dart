import 'package:bytebrew_mobile/core/domain/session.dart';

import 'mock_servers.dart';

final mockSessions = [
  Session(
    id: 'session-1',
    serverId: mockServers[1].id,
    serverName: mockServers[1].name,
    projectName: 'api-gateway',
    status: SessionStatus.needsAttention,
    hasAskUser: true,
    lastActivityAt: DateTime.now().subtract(const Duration(minutes: 1)),
  ),
  Session(
    id: 'session-2',
    serverId: mockServers[0].id,
    serverName: mockServers[0].name,
    projectName: 'bytebrew-srv',
    status: SessionStatus.active,
    currentTask: 'Рефакторинг auth модуля',
    hasAskUser: false,
    lastActivityAt: DateTime.now().subtract(const Duration(minutes: 2)),
  ),
  Session(
    id: 'session-3',
    serverId: mockServers[1].id,
    serverName: mockServers[1].name,
    projectName: 'test-project',
    status: SessionStatus.idle,
    hasAskUser: false,
    lastActivityAt: DateTime.now().subtract(const Duration(minutes: 15)),
  ),
  Session(
    id: 'session-4',
    serverId: mockServers[0].id,
    serverName: mockServers[0].name,
    projectName: 'mobile-app',
    status: SessionStatus.idle,
    hasAskUser: false,
    lastActivityAt: DateTime.now().subtract(const Duration(hours: 2)),
  ),
  Session(
    id: 'session-5',
    serverId: mockServers[0].id,
    serverName: mockServers[0].name,
    projectName: 'data-pipeline',
    status: SessionStatus.active,
    currentTask: 'Adding error handling',
    hasAskUser: false,
    lastActivityAt: DateTime.now().subtract(const Duration(minutes: 5)),
  ),
];
