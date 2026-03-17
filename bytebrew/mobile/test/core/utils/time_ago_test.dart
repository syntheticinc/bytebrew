import 'package:flutter_test/flutter_test.dart';

import 'package:bytebrew_mobile/core/utils/time_ago.dart';

void main() {
  group('timeAgo', () {
    test('returns "just now" for less than 60 seconds ago', () {
      final date = DateTime.now().subtract(const Duration(seconds: 30));
      expect(timeAgo(date), 'just now');
    });

    test('returns "just now" for 0 seconds ago', () {
      final date = DateTime.now();
      expect(timeAgo(date), 'just now');
    });

    test('returns minutes ago for 1-59 minutes', () {
      final oneMin = DateTime.now().subtract(const Duration(minutes: 1));
      expect(timeAgo(oneMin), '1m ago');

      final fiveMin = DateTime.now().subtract(const Duration(minutes: 5));
      expect(timeAgo(fiveMin), '5m ago');

      final fiftyNineMin = DateTime.now().subtract(const Duration(minutes: 59));
      expect(timeAgo(fiftyNineMin), '59m ago');
    });

    test('returns hours ago for 1-23 hours', () {
      final oneHour = DateTime.now().subtract(const Duration(hours: 1));
      expect(timeAgo(oneHour), '1h ago');

      final twelveHours = DateTime.now().subtract(const Duration(hours: 12));
      expect(timeAgo(twelveHours), '12h ago');

      final twentyThreeHours = DateTime.now().subtract(
        const Duration(hours: 23),
      );
      expect(timeAgo(twentyThreeHours), '23h ago');
    });

    test('returns days ago for 24+ hours', () {
      final oneDay = DateTime.now().subtract(const Duration(days: 1));
      expect(timeAgo(oneDay), '1d ago');

      final sevenDays = DateTime.now().subtract(const Duration(days: 7));
      expect(timeAgo(sevenDays), '7d ago');

      final thirtyDays = DateTime.now().subtract(const Duration(days: 30));
      expect(timeAgo(thirtyDays), '30d ago');
    });
  });
}
