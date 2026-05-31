/// Widget tests for MiningScreen adaptive layout + ore-ranking table.
///
/// 1. Small view (<840 dp) → single-pane: no VerticalDivider.
/// 2. Large view (≥840 dp) → two-pane: at least one VerticalDivider.
/// Plus: ore table renders ≥2 rows with verdict chips, and the
/// no-mining-setup state is displayed correctly.
library;

import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:eve_o_provit/api/mining_models.dart';
import 'package:eve_o_provit/core/theme.dart';
import 'package:eve_o_provit/features/character/character_api.dart';
import 'package:eve_o_provit/features/character/character_models.dart';
import 'package:eve_o_provit/features/character/providers.dart'
    show characterApiProvider, currentShipProvider;
import 'package:eve_o_provit/features/mining/mining_providers.dart';
import 'package:eve_o_provit/features/mining/mining_screen.dart';

// ---------------------------------------------------------------------------
// Canned data
// ---------------------------------------------------------------------------

OreRankingResponse _fakeResponse() => const OreRankingResponse(
      regionId: 10000002,
      secBand: 'high',
      noMiningSetup: false,
      rows: [
        OreRankRow(
          oreTypeId: 1230,
          oreName: 'Veldspar',
          miningM3PerHour: 3600,
          rawIskPerHour: 4200000,
          refineIskPerHour: 5100000,
          rawNetPerM3: 1166.67,
          refineNetPerM3: 1416.67,
          best: 'refine',
          deltaIskPerHour: 900000,
          bestStationId: 60003760,
          bestStationTax: 0.05,
        ),
        OreRankRow(
          oreTypeId: 1228,
          oreName: 'Scordite',
          miningM3PerHour: 3000,
          rawIskPerHour: 3100000,
          refineIskPerHour: 3400000,
          rawNetPerM3: 1033.33,
          refineNetPerM3: 1133.33,
          best: 'raw',
          deltaIskPerHour: 300000,
          bestStationTax: 0.04,
        ),
      ],
    );

OreRankingResponse _detailResponse() => const OreRankingResponse(
      regionId: 10000002,
      secBand: 'high',
      noMiningSetup: false,
      rows: [
        OreRankRow(
          oreTypeId: 1230,
          oreName: 'Veldspar',
          miningM3PerHour: 3600,
          rawIskPerHour: 4200000,
          refineIskPerHour: 5100000,
          rawNetPerM3: 1166.67,
          refineNetPerM3: 1416.67,
          best: 'refine',
          deltaIskPerHour: 900000,
          bestStationId: 60003760,
          bestStationTax: 0.05,
          bestStationName: 'Jita IV - Moon 4 - Caldari Navy Assembly Plant',
          bestStationSystem: 'Jita',
          materials: [
            RefineMaterial(
              materialTypeId: 34,
              materialName: 'Tritanium',
              effectiveQty: 12345,
              buyPrice: 5.5,
              sell: SellLocation(
                isStructure: false,
                stationName: 'Amarr VIII - Emperor Family Academy',
                systemName: 'Amarr',
              ),
            ),
          ],
        ),
        OreRankRow(
          oreTypeId: 1228,
          oreName: 'Scordite',
          miningM3PerHour: 3000,
          rawIskPerHour: 3100000,
          refineIskPerHour: 3400000,
          rawNetPerM3: 1033.33,
          refineNetPerM3: 1133.33,
          best: 'raw',
          deltaIskPerHour: 300000,
          bestStationTax: 0.04,
          rawSell: SellLocation(
            isStructure: false,
            stationName: 'Dodixie IX - Moon 20 - Federation Navy Assembly Plant',
            systemName: 'Dodixie',
          ),
        ),
        OreRankRow(
          oreTypeId: 1224,
          oreName: 'Pyroxeres',
          miningM3PerHour: 2000,
          rawIskPerHour: 2100000,
          refineIskPerHour: 1900000,
          rawNetPerM3: 700,
          refineNetPerM3: 633,
          best: 'raw',
          deltaIskPerHour: 200000,
          bestStationTax: 0.03,
          rawSell: SellLocation(isStructure: true),
        ),
      ],
    );

OreRankingResponse _noSetupResponse() => const OreRankingResponse(
      regionId: 10000002,
      secBand: 'high',
      noMiningSetup: true,
      rows: [],
    );

OreRankingResponse _emptyResponse() => const OreRankingResponse(
      regionId: 10000002,
      secBand: 'null',
      noMiningSetup: false,
      rows: [],
    );

class _FakeCharacterApi extends CharacterApi {
  _FakeCharacterApi() : super(Dio());
}

const _currentShip = CharacterShip(
  shipTypeId: 32880,
  shipName: 'My Retriever',
  shipItemId: 1000000000010,
  shipTypeName: 'Retriever',
  cargoCapacity: 2400.0,
  effectiveCargoCapacity: 27500.0,
);

class _StubNotifier extends OreRankingNotifier {
  @override
  Future<OreRankingResponse?> build() async => _fakeResponse();
}

class _DetailNotifier extends OreRankingNotifier {
  @override
  Future<OreRankingResponse?> build() async => _detailResponse();
}

class _NoSetupNotifier extends OreRankingNotifier {
  @override
  Future<OreRankingResponse?> build() async => _noSetupResponse();
}

class _EmptyNotifier extends OreRankingNotifier {
  @override
  Future<OreRankingResponse?> build() async => _emptyResponse();
}

Future<void> _pumpScreen(
  WidgetTester tester,
  double width, {
  double height = 1200,
  OreRankingNotifier Function() notifier = _StubNotifier.new,
}) async {
  tester.view.physicalSize = Size(width, height);
  tester.view.devicePixelRatio = 1.0;
  addTearDown(tester.view.resetPhysicalSize);
  addTearDown(tester.view.resetDevicePixelRatio);

  await tester.pumpWidget(
    ProviderScope(
      overrides: [
        characterApiProvider.overrideWithValue(_FakeCharacterApi()),
        currentShipProvider.overrideWith((ref) async => _currentShip),
        oreRankingProvider.overrideWith(notifier),
      ],
      child: MaterialApp(
        theme: buildTheme(Brightness.light),
        home: const MiningScreen(),
      ),
    ),
  );
  await tester.pumpAndSettle();
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

void main() {
  testWidgets('Single-pane (width 800): no VerticalDivider', (tester) async {
    await _pumpScreen(tester, 800);

    expect(find.byType(VerticalDivider), findsNothing);
    // Input controls present.
    expect(find.text('High-Sec (≥ 0.5)'), findsOneWidget);
    // Result table also visible in single-pane.
    expect(find.text('Erz-Ranking'), findsOneWidget);
  });

  testWidgets('Two-pane (width 1280): at least one VerticalDivider',
      (tester) async {
    await _pumpScreen(tester, 1280);

    expect(find.byType(VerticalDivider), findsAtLeastNWidgets(1));
    expect(find.text('High-Sec (≥ 0.5)'), findsOneWidget);
    expect(find.text('Erz-Ranking'), findsOneWidget);
  });

  testWidgets('Ore table renders ≥2 rows with ore names', (tester) async {
    await _pumpScreen(tester, 1280);

    expect(find.byKey(const Key('mining-ore-table')), findsOneWidget);
    expect(find.text('Veldspar'), findsOneWidget);
    expect(find.text('Scordite'), findsOneWidget);
  });

  testWidgets('Verdict chips are rendered per row', (tester) async {
    await _pumpScreen(tester, 1280);

    // Veldspar: best = 'refine' → chip text 'Raffinieren'
    final veldsparVerdict = find.byKey(const Key('mining-verdict-1230'));
    expect(veldsparVerdict, findsOneWidget);

    // Scordite: best = 'raw' → chip text 'Roh verkaufen'
    final scorditeVerdict = find.byKey(const Key('mining-verdict-1228'));
    expect(scorditeVerdict, findsOneWidget);

    expect(find.text('Raffinieren'), findsOneWidget);
    expect(find.text('Roh verkaufen'), findsOneWidget);
  });

  testWidgets('No-mining-setup state shows the visible notice', (tester) async {
    await _pumpScreen(tester, 1280, notifier: _NoSetupNotifier.new);

    expect(find.byKey(const Key('mining-no-setup')), findsOneWidget);
    expect(find.text('Kein Mining-Setup'), findsOneWidget);
    expect(find.byKey(const Key('mining-ore-table')), findsNothing);
  });

  testWidgets('Empty rows (no-mining-setup=false) shows empty-state',
      (tester) async {
    await _pumpScreen(tester, 1280, notifier: _EmptyNotifier.new);

    expect(find.byKey(const Key('mining-empty-state')), findsOneWidget);
    expect(
      find.text('Keine abbaubaren Erze für diese Zone gefunden'),
      findsOneWidget,
    );
    expect(find.byKey(const Key('mining-ore-table')), findsNothing);
  });

  testWidgets('CurrentShipCard is shown at the top of the input form',
      (tester) async {
    await _pumpScreen(tester, 1280);

    expect(find.byKey(const Key('current-ship-card')), findsOneWidget);
    // Ship name from the stub.
    expect(find.text('Schiff'), findsOneWidget);
  });

  testWidgets('Location detail is hidden until a row is expanded',
      (tester) async {
    await _pumpScreen(tester, 1280, notifier: _DetailNotifier.new);

    // Reprocess station + minerals live in the collapsed (lazy) detail panel.
    expect(find.textContaining('Aufbereiten bei'), findsNothing);
    expect(find.text('Tritanium'), findsNothing);
  });

  testWidgets('Expanding a refine row shows reprocess station + mineral sell',
      (tester) async {
    await _pumpScreen(tester, 1280, notifier: _DetailNotifier.new);

    await tester.tap(find.byKey(const ValueKey('mining-ore-expand-1230')));
    await tester.pumpAndSettle();

    expect(
      find.textContaining('Caldari Navy Assembly Plant — Jita'),
      findsOneWidget,
    );
    expect(find.text('Tritanium'), findsOneWidget);
    expect(
      find.textContaining('Emperor Family Academy — Amarr'),
      findsOneWidget,
    );
  });

  testWidgets('Expanding a raw row shows the raw ore sell location',
      (tester) async {
    await _pumpScreen(tester, 1280, notifier: _DetailNotifier.new);

    await tester.tap(find.byKey(const ValueKey('mining-ore-expand-1228')));
    await tester.pumpAndSettle();

    expect(
      find.textContaining('Federation Navy Assembly Plant — Dodixie'),
      findsOneWidget,
    );
  });

  testWidgets('Citadel sell location renders as Player-Struktur',
      (tester) async {
    await _pumpScreen(tester, 1280, notifier: _DetailNotifier.new);

    await tester.tap(find.byKey(const ValueKey('mining-ore-expand-1224')));
    await tester.pumpAndSettle();

    expect(find.textContaining('Player-Struktur'), findsOneWidget);
  });
}
