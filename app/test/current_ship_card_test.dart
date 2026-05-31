/// Widget tests for CurrentShipCard — the read-only "current ship" widget that
/// replaced the ship dropdown.
///
/// Verifies the cargo-display branches (effective / base / fitted-unbekannt),
/// the fail-loud error state, the "no ship" state, and that the refresh control
/// invalidates the provider (re-fetch).
library;

import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:eve_o_provit/core/theme.dart';
import 'package:eve_o_provit/features/character/character_models.dart';
import 'package:eve_o_provit/features/character/providers.dart'
    show currentShipProvider;
import 'package:eve_o_provit/features/trading/current_ship_card.dart';

CharacterShip _ship({
  String name = 'My Nereus',
  double cargo = 2700.0,
  double? effective,
  bool unavailable = false,
}) =>
    CharacterShip(
      shipTypeId: 650,
      shipName: name,
      shipItemId: 1,
      shipTypeName: 'Nereus',
      cargoCapacity: cargo,
      effectiveCargoCapacity: effective,
      effectiveCargoUnavailable: unavailable,
    );

Future<void> _pump(
  WidgetTester tester, {
  required FutureOr<CharacterShip?> Function(Ref ref) ship,
}) async {
  await tester.pumpWidget(
    ProviderScope(
      overrides: [currentShipProvider.overrideWith(ship)],
      child: MaterialApp(
        theme: buildTheme(Brightness.light),
        home: const Scaffold(body: CurrentShipCard()),
      ),
    ),
  );
  await tester.pumpAndSettle();
}

void main() {
  testWidgets('header label is "Schiff" with a refresh icon', (tester) async {
    await _pump(
      tester,
      ship: (ref) async => _ship(),
    );

    expect(find.text('Schiff'), findsOneWidget);
    expect(find.byKey(const Key('current-ship-refresh')), findsOneWidget);
  });

  testWidgets('shows effective cargo when known (> 0)', (tester) async {
    await _pump(
      tester,
      ship: (ref) async => _ship(effective: 9656.9),
    );

    // Effective cargo formatted as "9.7k m³" — no "Basis" prefix.
    expect(find.text('My Nereus (9.7k m³)'), findsOneWidget);
  });

  testWidgets('shows "Basis …" when only base cargo is known', (tester) async {
    await _pump(
      tester,
      ship: (ref) async => _ship(),
    );

    expect(find.text('My Nereus (Basis 2.7k m³)'), findsOneWidget);
  });

  testWidgets('shows "fitted unbekannt" when effective cargo unavailable',
      (tester) async {
    await _pump(
      tester,
      ship: (ref) async => _ship(unavailable: true),
    );

    expect(
      find.text('My Nereus (Basis 2.7k m³ — fitted unbekannt)'),
      findsOneWidget,
    );
  });

  testWidgets('fail-loud: surfaces an error when the fetch failed',
      (tester) async {
    await _pump(
      tester,
      ship: (ref) async => throw Exception('boom'),
    );

    expect(find.byKey(const Key('current-ship-error')), findsOneWidget);
    expect(
      find.text('Aktuelles Schiff konnte nicht geladen werden'),
      findsOneWidget,
    );
    // Must NOT silently render the "no ship" fallback on error.
    expect(find.text('Kein aktives Schiff'), findsNothing);
  });

  testWidgets('shows "Kein aktives Schiff" when there is no current ship',
      (tester) async {
    await _pump(
      tester,
      ship: (ref) async => null,
    );

    expect(find.text('Kein aktives Schiff'), findsOneWidget);
  });

  testWidgets('refresh button invalidates the provider (re-fetch)',
      (tester) async {
    var fetches = 0;
    await _pump(
      tester,
      ship: (ref) async {
        fetches++;
        return _ship(effective: 9656.9);
      },
    );

    expect(fetches, 1);

    await tester.tap(find.byKey(const Key('current-ship-refresh')));
    await tester.pumpAndSettle();

    // Invalidation triggers a second fetch.
    expect(fetches, 2);
  });
}
