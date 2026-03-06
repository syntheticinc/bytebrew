import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:mobile_scanner/mobile_scanner.dart';

import '../../../../core/theme/app_colors.dart';

/// Data extracted from a scanned QR code during pairing.
class QrPairingData {
  const QrPairingData({
    required this.bridgeUrl,
    required this.serverId,
    required this.pairingToken,
    this.serverPublicKey,
  });

  /// Bridge relay URL (e.g. "ws://bridge.bytebrew.ai:8080").
  final String bridgeUrl;

  /// Unique server identifier for Bridge routing.
  final String serverId;

  /// Pairing token.
  final String pairingToken;

  /// Server's X25519 public key (base64-encoded in QR, decoded here).
  final String? serverPublicKey;

  /// Parses [QrPairingData] from a JSON string embedded in a QR code.
  ///
  /// Expected JSON format:
  /// ```json
  /// {
  ///   "bridge_url": "ws://bridge.bytebrew.ai:8080",
  ///   "server_id": "server-uuid",
  ///   "token": "pairing-token",
  ///   "server_public_key": "base64-encoded-key"
  /// }
  /// ```
  ///
  /// Throws [FormatException] if the JSON is invalid or required fields
  /// are missing.
  factory QrPairingData.fromJson(String jsonString) {
    final Map<String, dynamic> json;
    try {
      json = jsonDecode(jsonString) as Map<String, dynamic>;
    } on FormatException {
      throw const FormatException('Invalid QR code: not valid JSON');
    }

    final bridgeUrl = json['bridge_url'] as String?;
    if (bridgeUrl == null || bridgeUrl.isEmpty) {
      throw const FormatException(
        'Invalid QR code: missing "bridge_url" field',
      );
    }

    final serverId = json['server_id'] as String?;
    if (serverId == null || serverId.isEmpty) {
      throw const FormatException('Invalid QR code: missing "server_id" field');
    }

    final token = json['token'] as String?;
    if (token == null || token.isEmpty) {
      throw const FormatException('Invalid QR code: missing "token" field');
    }

    return QrPairingData(
      bridgeUrl: bridgeUrl,
      serverId: serverId,
      pairingToken: token,
      serverPublicKey: json['server_public_key'] as String?,
    );
  }
}

/// Camera-based QR code scanner widget for server pairing.
///
/// Displays a live camera preview with a viewfinder overlay.
/// When a valid ByteBrew pairing QR code is detected, calls [onScanned]
/// with the parsed data and stops scanning.
class QrScannerWidget extends StatefulWidget {
  const QrScannerWidget({super.key, required this.onScanned});

  /// Callback invoked when a valid QR code is successfully scanned.
  final void Function(QrPairingData data) onScanned;

  @override
  State<QrScannerWidget> createState() => _QrScannerWidgetState();
}

class _QrScannerWidgetState extends State<QrScannerWidget> {
  final _controller = MobileScannerController(
    detectionSpeed: DetectionSpeed.normal,
    facing: CameraFacing.back,
  );

  bool _scanned = false;
  String? _errorMessage;

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  void _handleDetection(BarcodeCapture capture) {
    if (_scanned) return;

    final barcode = capture.barcodes.firstOrNull;
    if (barcode?.rawValue == null) return;

    final rawValue = barcode!.rawValue!;

    try {
      final data = QrPairingData.fromJson(rawValue);
      _scanned = true;
      _controller.stop();
      widget.onScanned(data);
    } on FormatException catch (e) {
      setState(() {
        _errorMessage = e.message;
      });
      // Clear error after a delay so user can try scanning another code.
      Future.delayed(const Duration(seconds: 3), () {
        if (mounted) {
          setState(() => _errorMessage = null);
        }
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return ClipRRect(
      borderRadius: BorderRadius.circular(16),
      child: SizedBox(
        width: 280,
        height: 280,
        child: Stack(
          children: [
            MobileScanner(controller: _controller, onDetect: _handleDetection),
            // Viewfinder overlay
            _ViewfinderOverlay(theme: theme),
            // Error message overlay
            if (_errorMessage != null)
              Positioned(
                bottom: 0,
                left: 0,
                right: 0,
                child: Container(
                  padding: const EdgeInsets.symmetric(
                    horizontal: 12,
                    vertical: 8,
                  ),
                  color: AppColors.dark.withValues(alpha: 0.85),
                  child: Text(
                    _errorMessage!,
                    style: theme.textTheme.bodySmall?.copyWith(
                      color: AppColors.statusNeedsAttention,
                    ),
                    textAlign: TextAlign.center,
                    maxLines: 2,
                    overflow: TextOverflow.ellipsis,
                  ),
                ),
              ),
          ],
        ),
      ),
    );
  }
}

/// Semi-transparent overlay with a centered viewfinder cutout.
class _ViewfinderOverlay extends StatelessWidget {
  const _ViewfinderOverlay({required this.theme});

  final ThemeData theme;

  @override
  Widget build(BuildContext context) {
    return Positioned.fill(
      child: CustomPaint(
        painter: _ViewfinderPainter(
          cornerColor: AppColors.accent,
          cornerLength: 24,
          cornerWidth: 3,
        ),
      ),
    );
  }
}

/// Paints corner brackets to frame the QR scan area.
class _ViewfinderPainter extends CustomPainter {
  _ViewfinderPainter({
    required this.cornerColor,
    required this.cornerLength,
    required this.cornerWidth,
  });

  final Color cornerColor;
  final double cornerLength;
  final double cornerWidth;

  @override
  void paint(Canvas canvas, Size size) {
    final paint = Paint()
      ..color = cornerColor
      ..strokeWidth = cornerWidth
      ..style = PaintingStyle.stroke
      ..strokeCap = StrokeCap.round;

    final inset = size.width * 0.15;
    final rect = Rect.fromLTRB(
      inset,
      inset,
      size.width - inset,
      size.height - inset,
    );

    // Top-left corner
    canvas.drawLine(
      rect.topLeft,
      Offset(rect.left + cornerLength, rect.top),
      paint,
    );
    canvas.drawLine(
      rect.topLeft,
      Offset(rect.left, rect.top + cornerLength),
      paint,
    );

    // Top-right corner
    canvas.drawLine(
      rect.topRight,
      Offset(rect.right - cornerLength, rect.top),
      paint,
    );
    canvas.drawLine(
      rect.topRight,
      Offset(rect.right, rect.top + cornerLength),
      paint,
    );

    // Bottom-left corner
    canvas.drawLine(
      rect.bottomLeft,
      Offset(rect.left + cornerLength, rect.bottom),
      paint,
    );
    canvas.drawLine(
      rect.bottomLeft,
      Offset(rect.left, rect.bottom - cornerLength),
      paint,
    );

    // Bottom-right corner
    canvas.drawLine(
      rect.bottomRight,
      Offset(rect.right - cornerLength, rect.bottom),
      paint,
    );
    canvas.drawLine(
      rect.bottomRight,
      Offset(rect.right, rect.bottom - cornerLength),
      paint,
    );
  }

  @override
  bool shouldRepaint(covariant _ViewfinderPainter oldDelegate) {
    return cornerColor != oldDelegate.cornerColor ||
        cornerLength != oldDelegate.cornerLength ||
        cornerWidth != oldDelegate.cornerWidth;
  }
}
