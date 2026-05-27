import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'core/router.dart';
import 'core/theme.dart';

void main() {
  runApp(
    const ProviderScope(
      child: EveOProvitApp(),
    ),
  );
}

/// Root widget of the EVE-O Provit app.
///
/// Reads [goRouterProvider] for routing and applies [buildTheme] for both
/// light and dark modes.
class EveOProvitApp extends ConsumerWidget {
  const EveOProvitApp({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final router = ref.watch(goRouterProvider);

    return MaterialApp.router(
      title: 'EVE-O Provit',
      routerConfig: router,
      theme: buildTheme(Brightness.light),
      darkTheme: buildTheme(Brightness.dark),
      themeMode: ThemeMode.system,
    );
  }
}
