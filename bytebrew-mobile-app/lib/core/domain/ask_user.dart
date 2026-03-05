/// Status of an ask-user prompt.
enum AskUserStatus { pending, answered }

/// Data for a question the agent asks the user.
class AskUserData {
  const AskUserData({
    required this.id,
    required this.question,
    required this.options,
    required this.status,
    this.answer,
  });

  final String id;
  final String question;
  final List<String> options;
  final AskUserStatus status;
  final String? answer;

  AskUserData copyWith({
    String? id,
    String? question,
    List<String>? options,
    AskUserStatus? status,
    String? answer,
  }) {
    return AskUserData(
      id: id ?? this.id,
      question: question ?? this.question,
      options: options ?? this.options,
      status: status ?? this.status,
      answer: answer ?? this.answer,
    );
  }
}
