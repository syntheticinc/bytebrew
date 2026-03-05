import 'dart:ui';

import 'package:bytebrew_mobile/core/domain/session.dart';

/// Brand color palette from ByteBrew brand guidelines.
abstract final class AppColors {
  // Brand
  static const Color accent = Color(0xFFD7513E);
  static const Color dark = Color(0xFF111111);
  static const Color darkAlt = Color(0xFF1F1F1F);
  static const Color light = Color(0xFFF7F8F1);
  static const Color shade1 = Color(0xFFDFD8D0);
  static const Color shade2 = Color(0xFFCBC9BC);
  static const Color shade3 = Color(0xFF87867F);
  static const Color white = Color(0xFFFFFFFF);
  static const Color black = Color(0xFF000000);

  // Semantic status
  static const Color statusActive = Color(0xFF4CAF50);
  static const Color statusNeedsAttention = Color(0xFFD7513E); // = accent
  static const Color statusIdle = Color(0xFF87867F); // = shade3

  /// Returns the semantic color for a given [SessionStatus].
  static Color statusColor(SessionStatus status) {
    return switch (status) {
      SessionStatus.needsAttention => statusNeedsAttention,
      SessionStatus.active => statusActive,
      SessionStatus.idle => statusIdle,
    };
  }
}
