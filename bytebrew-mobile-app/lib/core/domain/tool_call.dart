/// Status of a tool call execution.
enum ToolCallStatus { running, completed, failed }

/// Data for a tool call made by the agent.
class ToolCallData {
  const ToolCallData({
    required this.id,
    required this.toolName,
    required this.arguments,
    required this.status,
    this.result,
    this.fullResult,
    this.error,
  });

  final String id;
  final String toolName;
  final Map<String, String> arguments;
  final ToolCallStatus status;
  final String? result;
  final String? fullResult;
  final String? error;

  ToolCallData copyWith({
    String? id,
    String? toolName,
    Map<String, String>? arguments,
    ToolCallStatus? status,
    String? result,
    String? fullResult,
    String? error,
  }) {
    return ToolCallData(
      id: id ?? this.id,
      toolName: toolName ?? this.toolName,
      arguments: arguments ?? this.arguments,
      status: status ?? this.status,
      result: result ?? this.result,
      fullResult: fullResult ?? this.fullResult,
      error: error ?? this.error,
    );
  }
}
