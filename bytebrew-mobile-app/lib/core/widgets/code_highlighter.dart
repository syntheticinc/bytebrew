import 'package:flutter/material.dart';
import 'package:flutter_markdown_plus/flutter_markdown_plus.dart';
import 'package:highlight/highlight.dart' as hl;

/// Brand-themed syntax highlighter for markdown code blocks.
///
/// Uses the `highlight` package to parse code and applies brand colors
/// to different token types (keywords, strings, types, comments, etc.).
class CodeHighlighter implements SyntaxHighlighter {
  CodeHighlighter({required this.isDark});

  final bool isDark;

  // The top-level `highlight` from the package already has all languages.
  static final hl.Highlight _highlight = hl.highlight;

  @override
  TextSpan format(String source) {
    final result = _highlight.parse(source, autoDetection: true);
    if (result.nodes == null || result.nodes!.isEmpty) {
      return TextSpan(text: source, style: _baseStyle);
    }

    return TextSpan(children: _buildSpans(result.nodes!), style: _baseStyle);
  }

  List<TextSpan> _buildSpans(List<hl.Node> nodes) {
    final spans = <TextSpan>[];
    for (final node in nodes) {
      if (node.value != null) {
        spans.add(
          TextSpan(text: node.value, style: _styleForClass(node.className)),
        );
      } else if (node.children != null) {
        spans.add(
          TextSpan(
            children: _buildSpans(node.children!),
            style: _styleForClass(node.className),
          ),
        );
      }
    }
    return spans;
  }

  TextStyle? _styleForClass(String? className) {
    if (className == null) return null;
    return isDark ? _darkTheme[className] : _lightTheme[className];
  }

  TextStyle get _baseStyle => TextStyle(
    color: isDark ? const Color(0xFFF7F8F1) : const Color(0xFF111111),
    fontSize: 13,
    height: 1.5,
  );

  // Dark theme — brand-aligned syntax colors
  static const _darkTheme = <String, TextStyle>{
    'keyword': TextStyle(color: Color(0xFFD7513E)), // accent
    'built_in': TextStyle(color: Color(0xFFE5C07B)), // warm yellow
    'type': TextStyle(color: Color(0xFF4CAF50)), // green
    'literal': TextStyle(color: Color(0xFF61AFEF)), // blue
    'number': TextStyle(color: Color(0xFF61AFEF)), // blue
    'string': TextStyle(color: Color(0xFFE5C07B)), // warm yellow
    'comment': TextStyle(color: Color(0xFF87867F), fontStyle: FontStyle.italic),
    'doctag': TextStyle(color: Color(0xFF87867F)),
    'function': TextStyle(color: Color(0xFF61AFEF)), // blue
    'title': TextStyle(color: Color(0xFF4CAF50)), // green
    'class': TextStyle(color: Color(0xFF4CAF50)),
    'params': TextStyle(color: Color(0xFFF7F8F1)), // light
    'attr': TextStyle(color: Color(0xFFD7513E)), // accent
    'attribute': TextStyle(color: Color(0xFFD7513E)),
    'meta': TextStyle(color: Color(0xFF87867F)),
    'tag': TextStyle(color: Color(0xFFD7513E)),
    'name': TextStyle(color: Color(0xFFD7513E)),
    'selector-tag': TextStyle(color: Color(0xFFD7513E)),
    'selector-id': TextStyle(color: Color(0xFF61AFEF)),
    'selector-class': TextStyle(color: Color(0xFF4CAF50)),
    'regexp': TextStyle(color: Color(0xFFE5C07B)),
    'symbol': TextStyle(color: Color(0xFF61AFEF)),
    'variable': TextStyle(color: Color(0xFFF7F8F1)),
    'template-variable': TextStyle(color: Color(0xFFD7513E)),
    'link': TextStyle(color: Color(0xFF61AFEF)),
    'addition': TextStyle(color: Color(0xFF4CAF50)), // diff green
    'deletion': TextStyle(color: Color(0xFFD7513E)), // diff red
    'section': TextStyle(color: Color(0xFF61AFEF), fontWeight: FontWeight.bold),
    'subst': TextStyle(color: Color(0xFFF7F8F1)),
  };

  // Light theme — darker tones
  static const _lightTheme = <String, TextStyle>{
    'keyword': TextStyle(color: Color(0xFFC0392B)),
    'built_in': TextStyle(color: Color(0xFFB8860B)),
    'type': TextStyle(color: Color(0xFF2E7D32)),
    'literal': TextStyle(color: Color(0xFF1565C0)),
    'number': TextStyle(color: Color(0xFF1565C0)),
    'string': TextStyle(color: Color(0xFFB8860B)),
    'comment': TextStyle(color: Color(0xFF87867F), fontStyle: FontStyle.italic),
    'doctag': TextStyle(color: Color(0xFF87867F)),
    'function': TextStyle(color: Color(0xFF1565C0)),
    'title': TextStyle(color: Color(0xFF2E7D32)),
    'class': TextStyle(color: Color(0xFF2E7D32)),
    'params': TextStyle(color: Color(0xFF111111)),
    'attr': TextStyle(color: Color(0xFFC0392B)),
    'attribute': TextStyle(color: Color(0xFFC0392B)),
    'meta': TextStyle(color: Color(0xFF87867F)),
    'tag': TextStyle(color: Color(0xFFC0392B)),
    'name': TextStyle(color: Color(0xFFC0392B)),
    'selector-tag': TextStyle(color: Color(0xFFC0392B)),
    'selector-id': TextStyle(color: Color(0xFF1565C0)),
    'selector-class': TextStyle(color: Color(0xFF2E7D32)),
    'regexp': TextStyle(color: Color(0xFFB8860B)),
    'symbol': TextStyle(color: Color(0xFF1565C0)),
    'variable': TextStyle(color: Color(0xFF111111)),
    'template-variable': TextStyle(color: Color(0xFFC0392B)),
    'link': TextStyle(color: Color(0xFF1565C0)),
    'addition': TextStyle(color: Color(0xFF2E7D32)),
    'deletion': TextStyle(color: Color(0xFFC0392B)),
    'section': TextStyle(color: Color(0xFF1565C0), fontWeight: FontWeight.bold),
    'subst': TextStyle(color: Color(0xFF111111)),
  };
}
