/// Verifies the CurrentSelectionPrefill mixin seeds the shared region selection
/// from the character's current region — once, and without overwriting an
/// explicit prior choice.
///
/// (Ship pre-fill was removed: every trading view now reads the character's
/// CURRENT ship directly via currentShipProvider — there is no ship selection
/// state to seed.)
library;

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:eve_o_provit/api/trading_models.dart';
import 'package:eve_o_provit/auth/auth_controller.dart';
import 'package:eve_o_provit/features/character/providers.dart';
import 'package:eve_o_provit/features/trading/current_selection_prefill.dart';
import 'package:eve_o_provit/features/trading/providers.dart';

// Auth controller stub that reports an authenticated session (the prefill is
// gated on Authenticated).
class _AuthedController extends AuthController {
  @override
  Future<AuthState> build() async => const Authenticated();
}

class _UnauthedController extends AuthController {
  @override
  Future<AuthState> build() async => const Unauthenticated();
}

// Minimal host widget that mounts the mixin.
class _Host extends ConsumerStatefulWidget {
  const _Host();
  @override
  ConsumerState<_Host> createState() => _HostState();
}

class _HostState extends ConsumerState<_Host> with CurrentSelectionPrefill {
  @override
  void initState() {
    super.initState();
    startSelectionPrefill();
  }

  @override
  Widget build(BuildContext context) => const SizedBox.shrink();
}

Future<void> _pump(WidgetTester tester, ProviderContainer container) async {
  await tester.pumpWidget(
    UncontrolledProviderScope(
      container: container,
      child: const MaterialApp(home: _Host()),
    ),
  );
  await tester.pumpAndSettle();
}

void main() {
  const regions = [
    Region(id: 10000002, name: 'The Forge'),
    Region(id: 10000042, name: 'Metropolis'),
  ];

  testWidgets('seeds region from the current region', (tester) async {
    final c = ProviderContainer(overrides: [
      authControllerProvider.overrideWith(_AuthedController.new),
      currentRegionIdProvider.overrideWith((ref) async => 10000042),
      regionsProvider.overrideWith((ref) async => regions),
    ]);
    addTearDown(c.dispose);
    await _pump(tester, c);

    expect(c.read(selectedRegionProvider)?.id, 10000042);
  });

  testWidgets('region stays null when no current region is available',
      (tester) async {
    final c = ProviderContainer(overrides: [
      authControllerProvider.overrideWith(_AuthedController.new),
      currentRegionIdProvider.overrideWith((ref) async => null),
      regionsProvider.overrideWith((ref) async => regions),
    ]);
    addTearDown(c.dispose);
    await _pump(tester, c);

    // No current region → left unset (no hard fallback).
    expect(c.read(selectedRegionProvider), isNull);
  });

  testWidgets('does not override an explicit prior region choice',
      (tester) async {
    final c = ProviderContainer(overrides: [
      authControllerProvider.overrideWith(_AuthedController.new),
      currentRegionIdProvider.overrideWith((ref) async => 10000042),
      regionsProvider.overrideWith((ref) async => regions),
    ]);
    addTearDown(c.dispose);

    // User picks The Forge before the prefill resolves.
    c.read(selectedRegionProvider.notifier).select(regions[0]);
    await _pump(tester, c);

    // The explicit choice is preserved (not overwritten by the current region).
    expect(c.read(selectedRegionProvider)?.id, 10000002);
  });

  // Regression guard: a mount before authentication must NOT prematurely seed
  // the region (the original bug — the prefill fired during the SSO login
  // transition and cached null).
  testWidgets('does not prefill while unauthenticated', (tester) async {
    final c = ProviderContainer(overrides: [
      authControllerProvider.overrideWith(_UnauthedController.new),
      currentRegionIdProvider.overrideWith((ref) async => 10000042),
      regionsProvider.overrideWith((ref) async => regions),
    ]);
    addTearDown(c.dispose);
    await _pump(tester, c);

    expect(c.read(selectedRegionProvider), isNull);
  });
}
