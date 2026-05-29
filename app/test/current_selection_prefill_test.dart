/// Verifies the CurrentSelectionPrefill mixin seeds the shared region + ship
/// selection from the character's current region / active ship — once, and
/// without overwriting an explicit prior choice.
library;

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:eve_o_provit/api/trading_models.dart';
import 'package:eve_o_provit/features/character/providers.dart';
import 'package:eve_o_provit/features/trading/current_selection_prefill.dart';
import 'package:eve_o_provit/features/trading/providers.dart';

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
      activeShipTypeIdProvider.overrideWith((ref) async => 12005),
      currentRegionIdProvider.overrideWith((ref) async => 10000042),
      regionsProvider.overrideWith((ref) async => regions),
    ]);
    addTearDown(c.dispose);
    await _pump(tester, c);

    expect(c.read(selectedShipTypeIdProvider), 12005);
    expect(c.read(selectedRegionProvider)?.id, 10000042);
  });

  testWidgets('falls back to ship 648 when no active ship; region stays null',
      (tester) async {
    final c = ProviderContainer(overrides: [
      // Real provider returns null on error → mixin applies the 648 fallback.
      activeShipTypeIdProvider.overrideWith((ref) async => null),
      currentRegionIdProvider.overrideWith((ref) async => null),
      regionsProvider.overrideWith((ref) async => regions),
    ]);
    addTearDown(c.dispose);
    await _pump(tester, c);

    expect(c.read(selectedShipTypeIdProvider), 648);
    expect(c.read(selectedRegionProvider), isNull);
  });
}
