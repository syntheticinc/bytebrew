import 'package:bytebrew_mobile/core/domain/agent_info.dart';
import 'package:bytebrew_mobile/features/chat/application/agent_provider.dart';
import 'package:bytebrew_mobile/features/chat/presentation/widgets/agent_selector_bar.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

const _sessionId = 'test-session';

final _testAgents = [
  AgentInfo(
    agentId: 'agent-coder',
    status: AgentStatus.running,
    description: 'Code Writer',
    lastActivityAt: DateTime(2026, 3, 1, 12, 0),
  ),
  AgentInfo(
    agentId: 'agent-reviewer',
    status: AgentStatus.completed,
    description: 'Code Reviewer',
    lastActivityAt: DateTime(2026, 3, 1, 12, 5),
  ),
];

void main() {
  group('AgentSelectorBar', () {
    testWidgets('hidden when no agents (renders SizedBox.shrink)',
        (tester) async {
      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            agentsProvider(_sessionId).overrideWithValue(<AgentInfo>[]),
            selectedAgentProvider(_sessionId).overrideWithValue(null),
          ],
          child: const MaterialApp(
            home: Scaffold(
              body: AgentSelectorBar(sessionId: _sessionId),
            ),
          ),
        ),
      );

      await tester.pumpAndSettle();

      // SizedBox.shrink produces a zero-size box.
      final sizedBox = tester.widget<SizedBox>(find.byType(SizedBox).first);
      expect(sizedBox.width, 0.0);
      expect(sizedBox.height, 0.0);

      // No FilterChip should be rendered.
      expect(find.byType(FilterChip), findsNothing);
    });

    testWidgets('shows Supervisor chip and agent chips when agents exist',
        (tester) async {
      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            agentsProvider(_sessionId).overrideWithValue(_testAgents),
            selectedAgentProvider(_sessionId).overrideWithValue(null),
          ],
          child: const MaterialApp(
            home: Scaffold(
              body: AgentSelectorBar(sessionId: _sessionId),
            ),
          ),
        ),
      );

      await tester.pumpAndSettle();

      // Should render 3 chips: Supervisor + 2 agents.
      expect(find.byType(FilterChip), findsNWidgets(3));
      expect(find.text('Supervisor'), findsOneWidget);
      expect(find.text('Code Writer'), findsOneWidget);
      expect(find.text('Code Reviewer'), findsOneWidget);
    });

    testWidgets('Supervisor chip is selected when selectedAgent is null',
        (tester) async {
      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            agentsProvider(_sessionId).overrideWithValue(_testAgents),
            selectedAgentProvider(_sessionId).overrideWithValue(null),
          ],
          child: const MaterialApp(
            home: Scaffold(
              body: AgentSelectorBar(sessionId: _sessionId),
            ),
          ),
        ),
      );

      await tester.pumpAndSettle();

      // The Supervisor chip should be selected.
      final chips = tester.widgetList<FilterChip>(find.byType(FilterChip));
      final supervisorChip = chips.first;
      expect(supervisorChip.selected, isTrue);
    });

    testWidgets('agent chip is selected when selectedAgent matches',
        (tester) async {
      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            agentsProvider(_sessionId).overrideWithValue(_testAgents),
            selectedAgentProvider(_sessionId).overrideWithValue('agent-coder'),
          ],
          child: const MaterialApp(
            home: Scaffold(
              body: AgentSelectorBar(sessionId: _sessionId),
            ),
          ),
        ),
      );

      await tester.pumpAndSettle();

      final chips =
          tester.widgetList<FilterChip>(find.byType(FilterChip)).toList();

      // Supervisor (index 0) should NOT be selected.
      expect(chips[0].selected, isFalse);

      // Code Writer (index 1) should be selected.
      expect(chips[1].selected, isTrue);

      // Code Reviewer (index 2) should NOT be selected.
      expect(chips[2].selected, isFalse);
    });

    testWidgets('renders status indicators for agents', (tester) async {
      final agentsWithFailed = [
        AgentInfo(
          agentId: 'agent-coder',
          status: AgentStatus.running,
          description: 'Code Writer',
          lastActivityAt: DateTime(2026, 3, 1, 12, 0),
        ),
        AgentInfo(
          agentId: 'agent-reviewer',
          status: AgentStatus.failed,
          description: 'Code Reviewer',
          lastActivityAt: DateTime(2026, 3, 1, 12, 5),
        ),
      ];

      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            agentsProvider(_sessionId).overrideWithValue(agentsWithFailed),
            selectedAgentProvider(_sessionId).overrideWithValue(null),
          ],
          child: const MaterialApp(
            home: Scaffold(
              body: AgentSelectorBar(sessionId: _sessionId),
            ),
          ),
        ),
      );

      await tester.pumpAndSettle();

      // Running agent has a small circle icon (Icons.circle, size 8).
      expect(find.byIcon(Icons.circle), findsOneWidget);

      // Failed agent has an error icon (Icons.error, size 12).
      expect(find.byIcon(Icons.error), findsOneWidget);
    });

    testWidgets('completed agent shows check_circle icon', (tester) async {
      final agentsWithCompleted = [
        AgentInfo(
          agentId: 'agent-coder',
          status: AgentStatus.completed,
          description: 'Code Writer',
          lastActivityAt: DateTime(2026, 3, 1, 12, 0),
        ),
      ];

      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            agentsProvider(_sessionId).overrideWithValue(agentsWithCompleted),
            selectedAgentProvider(_sessionId).overrideWithValue(null),
          ],
          child: const MaterialApp(
            home: Scaffold(
              body: AgentSelectorBar(sessionId: _sessionId),
            ),
          ),
        ),
      );

      await tester.pumpAndSettle();

      expect(find.byIcon(Icons.check_circle), findsOneWidget);
    });

    testWidgets('long description is truncated to 20 chars', (tester) async {
      final agentsWithLongDesc = [
        AgentInfo(
          agentId: 'agent-long',
          status: AgentStatus.running,
          description: 'Very Long Agent Description That Exceeds Twenty Characters',
          lastActivityAt: DateTime(2026, 3, 1, 12, 0),
        ),
      ];

      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            agentsProvider(_sessionId).overrideWithValue(agentsWithLongDesc),
            selectedAgentProvider(_sessionId).overrideWithValue(null),
          ],
          child: const MaterialApp(
            home: Scaffold(
              body: AgentSelectorBar(sessionId: _sessionId),
            ),
          ),
        ),
      );

      await tester.pumpAndSettle();

      // The label should be truncated: first 17 chars + "..."
      expect(find.text('Very Long Agent D...'), findsOneWidget);
      expect(
        find.text(
            'Very Long Agent Description That Exceeds Twenty Characters'),
        findsNothing,
      );
    });
  });
}
