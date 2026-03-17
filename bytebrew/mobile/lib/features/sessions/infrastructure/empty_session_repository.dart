import 'package:bytebrew_mobile/core/domain/session.dart';
import 'package:bytebrew_mobile/features/sessions/domain/session_repository.dart';

/// [SessionRepository] that always returns an empty list.
///
/// Used when no gRPC connection is active.
class EmptySessionRepository implements SessionRepository {
  const EmptySessionRepository();

  @override
  Future<List<Session>> listSessions() async => [];

  @override
  Future<void> refresh() async {}

  @override
  Stream<List<Session>>? watchSessions() => null;
}
