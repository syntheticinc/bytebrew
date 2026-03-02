import 'package:bytebrew_mobile/core/domain/chat_message.dart';
import 'package:bytebrew_mobile/core/domain/tool_call.dart';
import 'package:bytebrew_mobile/core/theme/app_colors.dart';
import 'package:bytebrew_mobile/features/chat/presentation/widgets/tool_call_widget.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('ToolCallWidget', () {
    ChatMessage buildToolMessage({ToolCallData? toolCall}) {
      return ChatMessage(
        id: 'tc-msg-1',
        type: ChatMessageType.toolCall,
        content: '',
        timestamp: DateTime.now(),
        toolCall: toolCall,
      );
    }

    Widget buildWidget(ChatMessage message) {
      return MaterialApp(
        home: Scaffold(body: ToolCallWidget(message: message)),
      );
    }

    testWidgets('renders SizedBox.shrink when toolCall is null', (
      tester,
    ) async {
      final message = buildToolMessage(toolCall: null);
      await tester.pumpWidget(buildWidget(message));
      await tester.pumpAndSettle();

      final sizedBox = tester.widget<SizedBox>(find.byType(SizedBox).first);
      expect(sizedBox.width, 0.0);
      expect(sizedBox.height, 0.0);
    });

    testWidgets('displays tool name', (tester) async {
      final message = buildToolMessage(
        toolCall: const ToolCallData(
          id: 'tc-1',
          toolName: 'read_file',
          arguments: {'path': 'main.go'},
          status: ToolCallStatus.completed,
          result: '50 lines',
        ),
      );

      await tester.pumpWidget(buildWidget(message));
      await tester.pumpAndSettle();

      expect(find.text('read_file'), findsOneWidget);
    });

    testWidgets('shows first argument value in parentheses', (tester) async {
      final message = buildToolMessage(
        toolCall: const ToolCallData(
          id: 'tc-1',
          toolName: 'search_code',
          arguments: {'query': 'auth handler'},
          status: ToolCallStatus.completed,
          result: '5 results',
        ),
      );

      await tester.pumpWidget(buildWidget(message));
      await tester.pumpAndSettle();

      expect(find.text('(auth handler)'), findsOneWidget);
    });

    testWidgets('does not show arguments when empty', (tester) async {
      final message = buildToolMessage(
        toolCall: const ToolCallData(
          id: 'tc-1',
          toolName: 'get_context',
          arguments: {},
          status: ToolCallStatus.completed,
          result: 'done',
        ),
      );

      await tester.pumpWidget(buildWidget(message));
      await tester.pumpAndSettle();

      expect(find.text('get_context'), findsOneWidget);
      // No parenthesized arguments text.
      expect(find.textContaining('('), findsNothing);
    });

    testWidgets('shows result line when result is present', (tester) async {
      final message = buildToolMessage(
        toolCall: const ToolCallData(
          id: 'tc-1',
          toolName: 'read_file',
          arguments: {'path': 'main.go'},
          status: ToolCallStatus.completed,
          result: '50 lines',
        ),
      );

      await tester.pumpWidget(buildWidget(message));
      await tester.pumpAndSettle();

      expect(find.text('50 lines'), findsOneWidget);
    });

    testWidgets('does not show result line when result is null', (
      tester,
    ) async {
      final message = buildToolMessage(
        toolCall: const ToolCallData(
          id: 'tc-1',
          toolName: 'write_file',
          arguments: {'path': 'test.go'},
          status: ToolCallStatus.running,
        ),
      );

      await tester.pumpWidget(buildWidget(message));
      // Use pump() because running status has CircularProgressIndicator.
      await tester.pump();

      expect(find.text('write_file'), findsOneWidget);
      // The result tree connector should not be rendered.
      expect(find.textContaining('\u2514'), findsNothing);
    });

    testWidgets('shows spinner for running status', (tester) async {
      final message = buildToolMessage(
        toolCall: const ToolCallData(
          id: 'tc-1',
          toolName: 'search_code',
          arguments: {'query': 'test'},
          status: ToolCallStatus.running,
        ),
      );

      await tester.pumpWidget(buildWidget(message));
      await tester.pump();

      expect(find.byType(CircularProgressIndicator), findsOneWidget);

      final indicator = tester.widget<CircularProgressIndicator>(
        find.byType(CircularProgressIndicator),
      );
      expect(indicator.strokeWidth, 1.5);
    });

    testWidgets('does not show spinner for completed status', (tester) async {
      final message = buildToolMessage(
        toolCall: const ToolCallData(
          id: 'tc-1',
          toolName: 'read_file',
          arguments: {'path': 'main.go'},
          status: ToolCallStatus.completed,
          result: 'done',
        ),
      );

      await tester.pumpWidget(buildWidget(message));
      await tester.pumpAndSettle();

      expect(find.byType(CircularProgressIndicator), findsNothing);
    });

    testWidgets('bullet color is green for completed status', (tester) async {
      final message = buildToolMessage(
        toolCall: const ToolCallData(
          id: 'tc-1',
          toolName: 'read_file',
          arguments: {'path': 'x'},
          status: ToolCallStatus.completed,
          result: 'ok',
        ),
      );

      await tester.pumpWidget(buildWidget(message));
      await tester.pumpAndSettle();

      // The bullet character is rendered as a Text widget with \u25CF.
      final bulletFinder = find.text('\u25CF');
      expect(bulletFinder, findsOneWidget);

      final bulletWidget = tester.widget<Text>(bulletFinder);
      expect(bulletWidget.style?.color, AppColors.statusActive);
    });

    testWidgets('bullet color is accent (red) for failed status', (
      tester,
    ) async {
      final message = buildToolMessage(
        toolCall: const ToolCallData(
          id: 'tc-1',
          toolName: 'run_command',
          arguments: {'cmd': 'make test'},
          status: ToolCallStatus.failed,
          result: 'exit code 1',
        ),
      );

      await tester.pumpWidget(buildWidget(message));
      await tester.pumpAndSettle();

      final bulletWidget = tester.widget<Text>(find.text('\u25CF'));
      expect(bulletWidget.style?.color, AppColors.accent);
    });

    testWidgets('bullet color is shade3 for running status', (tester) async {
      final message = buildToolMessage(
        toolCall: const ToolCallData(
          id: 'tc-1',
          toolName: 'search',
          arguments: {'q': 'x'},
          status: ToolCallStatus.running,
        ),
      );

      await tester.pumpWidget(buildWidget(message));
      await tester.pump();

      final bulletWidget = tester.widget<Text>(find.text('\u25CF'));
      expect(bulletWidget.style?.color, AppColors.shade3);
    });

    testWidgets('tap on widget triggers InkWell (no crash)', (tester) async {
      final message = buildToolMessage(
        toolCall: const ToolCallData(
          id: 'tc-1',
          toolName: 'read_file',
          arguments: {'path': 'main.go'},
          status: ToolCallStatus.completed,
          result: '50 lines',
          fullResult: 'Full content of file...',
        ),
      );

      await tester.pumpWidget(buildWidget(message));
      await tester.pumpAndSettle();

      // Tapping opens ToolDetailSheet (modal bottom sheet).
      await tester.tap(find.text('read_file'));
      await tester.pumpAndSettle();

      // The bottom sheet should appear with the tool name as header.
      expect(find.text('read_file'), findsWidgets);
    });
  });
}
