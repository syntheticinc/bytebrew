/// Status of a single plan step.
enum PlanStepStatus { pending, inProgress, completed }

/// A step within an agent's execution plan.
class PlanStep {
  const PlanStep({
    required this.index,
    required this.description,
    required this.status,
    this.completedAt,
  });

  final int index;
  final String description;
  final PlanStepStatus status;
  final DateTime? completedAt;

  PlanStep copyWith({
    int? index,
    String? description,
    PlanStepStatus? status,
    DateTime? completedAt,
  }) {
    return PlanStep(
      index: index ?? this.index,
      description: description ?? this.description,
      status: status ?? this.status,
      completedAt: completedAt ?? this.completedAt,
    );
  }
}

/// An agent's execution plan with a goal and ordered steps.
class PlanData {
  const PlanData({required this.goal, required this.steps});

  final String goal;
  final List<PlanStep> steps;

  /// Completion progress from 0.0 to 1.0.
  double get progress {
    if (steps.isEmpty) {
      return 0;
    }
    final completedCount = steps
        .where((s) => s.status == PlanStepStatus.completed)
        .length;
    return completedCount / steps.length;
  }
}
