import 'package:flutter/material.dart';

const _accent = Color(0xFF1F6FB0); // blue accent: primary / active / profit highlight

ThemeData buildTheme(Brightness brightness) {
  final scheme = ColorScheme.fromSeed(
    seedColor: _accent,
    brightness: brightness,
  ).copyWith(
    surface: brightness == Brightness.light
        ? const Color(0xFFFFFFFF)
        : const Color(0xFF242424),
    onSurface: brightness == Brightness.light
        ? const Color(0xFF333333)
        : const Color(0xFFFAFAFA),
  );
  return ThemeData(
    useMaterial3: true,
    colorScheme: scheme,
    cardTheme: CardThemeData(
      elevation: 1,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(10),
      ),
    ),
  );
}
