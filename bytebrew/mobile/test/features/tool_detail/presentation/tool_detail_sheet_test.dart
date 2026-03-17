import 'package:bytebrew_mobile/core/domain/tool_call.dart';
import 'package:bytebrew_mobile/features/tool_detail/presentation/tool_detail_sheet.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

/// Wraps [ToolDetailSheet] in a [MaterialApp] for testing.
///
/// The sheet is normally shown via [showModalBottomSheet]. For testing,
/// we instantiate it directly with a custom [ScrollController].
Widget _buildSheet(ToolCallData toolCall) {
  return MaterialApp(
    home: Scaffold(
      body: ToolDetailSheet.testOnly(
        toolCall: toolCall,
        scrollController: ScrollController(),
      ),
    ),
  );
}

void main() {
  testWidgets('ToolDetailSheet shows tool name', (tester) async {
    const toolCall = ToolCallData(
      id: 'tc-1',
      toolName: 'read_file',
      arguments: {'path': 'src/main.dart'},
      status: ToolCallStatus.completed,
      result: '42 lines',
    );

    await tester.pumpWidget(_buildSheet(toolCall));
    await tester.pumpAndSettle();

    expect(find.text('read_file'), findsOneWidget);
  });

  testWidgets('ToolDetailSheet shows arguments', (tester) async {
    const toolCall = ToolCallData(
      id: 'tc-2',
      toolName: 'search',
      arguments: {'query': 'auth', 'scope': 'project'},
      status: ToolCallStatus.completed,
      result: '5 results',
    );

    await tester.pumpWidget(_buildSheet(toolCall));
    await tester.pumpAndSettle();

    expect(find.text('ARGUMENTS'), findsOneWidget);
    expect(find.text('query: '), findsOneWidget);
    expect(find.text('auth'), findsOneWidget);
    expect(find.text('scope: '), findsOneWidget);
    expect(find.text('project'), findsOneWidget);
  });

  testWidgets('ToolDetailSheet shows result section', (tester) async {
    const toolCall = ToolCallData(
      id: 'tc-3',
      toolName: 'read_file',
      arguments: {'path': 'README.md'},
      status: ToolCallStatus.completed,
      result: 'File contents here',
      fullResult: 'Full file contents with more detail',
    );

    await tester.pumpWidget(_buildSheet(toolCall));
    await tester.pumpAndSettle();

    expect(find.text('RESULT'), findsOneWidget);
    // fullResult takes priority over result.
    expect(find.text('Full file contents with more detail'), findsOneWidget);
  });

  testWidgets('ToolDetailSheet shows result when fullResult is null', (
    tester,
  ) async {
    const toolCall = ToolCallData(
      id: 'tc-4',
      toolName: 'list_files',
      arguments: {'dir': '.'},
      status: ToolCallStatus.completed,
      result: '10 files found',
    );

    await tester.pumpWidget(_buildSheet(toolCall));
    await tester.pumpAndSettle();

    expect(find.text('RESULT'), findsOneWidget);
    expect(find.text('10 files found'), findsOneWidget);
  });

  testWidgets('ToolDetailSheet shows error section when error is present', (
    tester,
  ) async {
    const toolCall = ToolCallData(
      id: 'tc-5',
      toolName: 'execute',
      arguments: {'cmd': 'npm test'},
      status: ToolCallStatus.failed,
      error: 'Process exited with code 1',
    );

    await tester.pumpWidget(_buildSheet(toolCall));
    await tester.pumpAndSettle();

    expect(find.text('ERROR'), findsOneWidget);
    expect(find.text('Process exited with code 1'), findsOneWidget);
  });

  testWidgets('ToolDetailSheet hides result when result is null', (
    tester,
  ) async {
    const toolCall = ToolCallData(
      id: 'tc-6',
      toolName: 'write_file',
      arguments: {'path': 'out.txt'},
      status: ToolCallStatus.running,
    );

    await tester.pumpWidget(_buildSheet(toolCall));
    await tester.pumpAndSettle();

    expect(find.text('RESULT'), findsNothing);
  });

  testWidgets('ToolDetailSheet hides error when error is null', (tester) async {
    const toolCall = ToolCallData(
      id: 'tc-7',
      toolName: 'read_file',
      arguments: {'path': 'lib/main.dart'},
      status: ToolCallStatus.completed,
      result: 'OK',
    );

    await tester.pumpWidget(_buildSheet(toolCall));
    await tester.pumpAndSettle();

    expect(find.text('ERROR'), findsNothing);
  });

  testWidgets('ToolDetailSheet shows "Completed" status chip', (tester) async {
    const toolCall = ToolCallData(
      id: 'tc-8',
      toolName: 'read',
      arguments: {},
      status: ToolCallStatus.completed,
    );

    await tester.pumpWidget(_buildSheet(toolCall));
    await tester.pumpAndSettle();

    expect(find.text('Completed'), findsOneWidget);
  });

  testWidgets('ToolDetailSheet shows "Running" status chip', (tester) async {
    const toolCall = ToolCallData(
      id: 'tc-9',
      toolName: 'search',
      arguments: {},
      status: ToolCallStatus.running,
    );

    await tester.pumpWidget(_buildSheet(toolCall));
    await tester.pumpAndSettle();

    expect(find.text('Running'), findsOneWidget);
  });

  testWidgets('ToolDetailSheet shows "Failed" status chip', (tester) async {
    const toolCall = ToolCallData(
      id: 'tc-10',
      toolName: 'exec',
      arguments: {},
      status: ToolCallStatus.failed,
      error: 'timeout',
    );

    await tester.pumpWidget(_buildSheet(toolCall));
    await tester.pumpAndSettle();

    expect(find.text('Failed'), findsOneWidget);
  });
}
