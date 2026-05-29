/// Widget tests for RoiCalculatorScreen adaptive layout.
///
/// 1. Small view (<840 dp) → single-pane: no VerticalDivider.
/// 2. Large view (≥840 dp) → two-pane: at least one VerticalDivider.
/// Plus: result table renders from a seeded portfolio, and an empty `items`
/// result shows the "no viable portfolio" empty-state.
library;

import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:eve_o_provit/api/portfolio_models.dart';
import 'package:eve_o_provit/api/trading_api.dart';
import 'package:eve_o_provit/api/trading_models.dart';
import 'package:eve_o_provit/core/theme.dart';
import 'package:eve_o_provit/features/character/character_api.dart';
import 'package:eve_o_provit/features/character/character_models.dart';
import 'package:eve_o_provit/features/character/providers.dart'
    show characterApiProvider;
import 'package:eve_o_provit/features/trading/providers.dart';
import 'package:eve_o_provit/features/trading/roi_calculator_screen.dart';
import 'package:eve_o_provit/features/trading/roi_providers.dart';

// ---------------------------------------------------------------------------
// Canned data + fakes
// ---------------------------------------------------------------------------

PortfolioResult _fakeResult() => const PortfolioResult(
      items: [
        PortfolioItem(
          typeId: 34,
          name: 'Tritanium',
          capitalUsed: 1.5e8,
          units: 120000,
          tripsPerDay: 4,
          dailyProfit: 2.5e7,
          roiPercent: 16.7,
          buySystemName: 'Jita',
          buyStationName: 'Jita IV - Moon 4 - Caldari Navy Assembly Plant',
          buyStationId: 60003760,
          sellSystemName: 'Amarr',
          sellStationName: 'Amarr VIII (Oris) - Emperor Family Academy',
          sellStationId: 60008494,
        ),
        PortfolioItem(
          typeId: 35,
          name: 'Pyerite',
          capitalUsed: 1.2e8,
          units: 60000,
          tripsPerDay: 3,
          dailyProfit: 1.8e7,
          roiPercent: 15.0,
          buySystemName: 'Jita',
          buyStationName: 'Jita IV - Moon 4 - Caldari Navy Assembly Plant',
          buyStationId: 60003760,
          sellSystemName: 'Dodixie',
          sellStationName: 'Dodixie IX - Moon 20 - Federation Navy Assembly',
          sellStationId: 60011866,
        ),
      ],
      totalCapitalUsed: 4.8e8,
      totalDailyProfit: 6.1e7,
      timeUsedMin: 118,
      diversificationScore: 72,
      skillsApplied: SkillsApplied(
        applied: true,
        accounting: 5,
        brokerRelations: 5,
        salesTaxRate: 0.025,
        brokerFeeRate: 0.015,
      ),
    );

PortfolioResult _emptyResult() => const PortfolioResult(
      items: [],
      totalCapitalUsed: 0,
      totalDailyProfit: 0,
      timeUsedMin: 0,
      diversificationScore: 0,
      skillsApplied: SkillsApplied(
        applied: false,
        accounting: 0,
        brokerRelations: 0,
        salesTaxRate: 0,
        brokerFeeRate: 0,
      ),
    );

class _FakeTradingApi extends TradingApi {
  _FakeTradingApi() : super(Dio());

  @override
  Future<RegionsResponse> regions() async => RegionsResponse(
        regions: [const Region(id: 10000002, name: 'The Forge')],
        count: 1,
      );

  @override
  Future<PortfolioResult> optimizePortfolio(PortfolioRequest request) async =>
      _fakeResult();
}

class _FakeCharacterApi extends CharacterApi {
  _FakeCharacterApi() : super(Dio());

  @override
  Future<CharacterShipsResponse> ships() async =>
      const CharacterShipsResponse(ships: [], count: 0);
}

class _StubNotifier extends RoiOptimizeNotifier {
  @override
  Future<PortfolioResult?> build() async => _fakeResult();
}

class _EmptyNotifier extends RoiOptimizeNotifier {
  @override
  Future<PortfolioResult?> build() async => _emptyResult();
}

Future<void> _pumpScreen(
  WidgetTester tester,
  double width, {
  double height = 1200,
  RoiOptimizeNotifier Function() notifier = _StubNotifier.new,
}) async {
  tester.view.physicalSize = Size(width, height);
  tester.view.devicePixelRatio = 1.0;
  addTearDown(tester.view.resetPhysicalSize);
  addTearDown(tester.view.resetDevicePixelRatio);

  await tester.pumpWidget(
    ProviderScope(
      overrides: [
        tradingApiProvider.overrideWithValue(_FakeTradingApi()),
        characterApiProvider.overrideWithValue(_FakeCharacterApi()),
        roiOptimizeProvider.overrideWith(notifier),
      ],
      child: MaterialApp(
        theme: buildTheme(Brightness.light),
        home: const RoiCalculatorScreen(),
      ),
    ),
  );
  await tester.pumpAndSettle();
}

void main() {
  testWidgets('Single-pane (width 800): no VerticalDivider', (tester) async {
    await _pumpScreen(tester, 800);

    expect(find.byType(VerticalDivider), findsNothing);
    // Input form + result both present.
    expect(find.text('Region'), findsOneWidget);
    expect(find.text('Empfohlenes Portfolio'), findsOneWidget);
  });

  testWidgets('Two-pane (width 1280): at least one VerticalDivider',
      (tester) async {
    await _pumpScreen(tester, 1280);

    expect(find.byType(VerticalDivider), findsAtLeastNWidgets(1));
    expect(find.text('Region'), findsOneWidget);
    expect(find.text('Empfohlenes Portfolio'), findsOneWidget);
  });

  testWidgets('Allocation table renders item names + totals', (tester) async {
    await _pumpScreen(tester, 1280);

    expect(find.byKey(const Key('roi-allocation-table')), findsOneWidget);
    expect(find.text('Tritanium'), findsOneWidget);
    expect(find.text('Pyerite'), findsOneWidget);
    // Diversification summary tile.
    expect(find.text('Diversifikation'), findsOneWidget);
    expect(find.text('72/100'), findsOneWidget);
  });

  testWidgets('Allocation rows render Route text + waypoint button',
      (tester) async {
    await _pumpScreen(tester, 1280);

    // Compact "buy → sell" route label.
    expect(find.text('Jita → Amarr'), findsOneWidget);
    expect(find.text('Jita → Dodixie'), findsOneWidget);
    // One waypoint button per allocation row.
    expect(find.byKey(const Key('roi-waypoint-34')), findsOneWidget);
    expect(find.byKey(const Key('roi-waypoint-35')), findsOneWidget);
  });

  testWidgets('Empty items result shows the no-viable-portfolio empty-state',
      (tester) async {
    await _pumpScreen(tester, 1280, notifier: _EmptyNotifier.new);

    expect(find.byKey(const Key('roi-empty-state')), findsOneWidget);
    expect(
      find.text('Kein sinnvolles Portfolio für dieses Budget'),
      findsOneWidget,
    );
    expect(find.byKey(const Key('roi-allocation-table')), findsNothing);
  });
}
