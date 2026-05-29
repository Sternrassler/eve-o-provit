/// Verifies the CurrentSelectionPrefill mixin seeds the shared region + ship
/// selection from the character's current region / active ship — once, and
/// without overwriting an explicit prior choice.
library;

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:eve_o_provit/api/trading_models.dart';
import 'package:eve_o_provit/auth/auth_controller.dart';
import 'package:eve_o_provit/features/character/character_models.dart';
import 'package:eve_o_provit/features/character/providers.dart';
import 'package:eve_o_provit/features/trading/current_selection_prefill.dart';
import 'package:eve_o_provit/features/trading/providers.dart';

CharacterShip _ship(int typeId) => CharacterShip(
      shipTypeId: typeId,
      shipName: 'Test',
      shipItemId: 1,
      shipTypeName: 'Test Ship',
      cargoCapacity: 1000,
    );

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

  testWidgets('seeds region + ship from current values', (tester) async {
    final c = ProviderContainer(overrides: [
      authControllerProvider.overrideWith(_AuthedController.new),
      activeShipProvider.overrideWith((ref) async => _ship(12005)),
      currentRegionIdProvider.overrideWith((ref) async => 10000042),
      regionsProvider.overrideWith((ref) async => regions),
    ]);
    addTearDown(c.dispose);
    await _pump(tester, c);

    expect(c.read(selectedShipTypeIdProvider), 12005);
    expect(c.read(selectedRegionProvider)?.id, 10000042);
  });

  testWidgets('region stays null when no current region is available',
      (tester) async {
    final c = ProviderContainer(overrides: [
      authControllerProvider.overrideWith(_AuthedController.new),
      activeShipProvider.overrideWith((ref) async => _ship(12005)),
      currentRegionIdProvider.overrideWith((ref) async => null),
      regionsProvider.overrideWith((ref) async => regions),
    ]);
    addTearDown(c.dispose);
    await _pump(tester, c);

    // Ship still seeds; region has no current value → left unset (no fallback).
    expect(c.read(selectedShipTypeIdProvider), 12005);
    expect(c.read(selectedRegionProvider), isNull);
  });

  // Regression guard: a mount before authentication must NOT prematurely apply
  // the 648 fallback / leave region empty for the session (the original bug —
  // the prefill fired during the SSO login transition and cached null).
  testWidgets('does not prefill while unauthenticated', (tester) async {
    final c = ProviderContainer(overrides: [
      authControllerProvider.overrideWith(_UnauthedController.new),
      activeShipProvider.overrideWith((ref) async => _ship(12005)),
      currentRegionIdProvider.overrideWith((ref) async => 10000042),
      regionsProvider.overrideWith((ref) async => regions),
    ]);
    addTearDown(c.dispose);
    await _pump(tester, c);

    expect(c.read(selectedShipTypeIdProvider), isNull);
    expect(c.read(selectedRegionProvider), isNull);
  });
}
