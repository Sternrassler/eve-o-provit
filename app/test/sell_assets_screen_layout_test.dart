/// Widget tests for SellAssetsScreen adaptive layout + option list + detail.
///
/// 1. Small view (<840 dp) → single-pane: no VerticalDivider.
/// 2. Large view (≥840 dp) → two-pane: at least one VerticalDivider.
/// Plus: option list renders from a seeded response (system → station), the
/// security-risk chip uses the right colour, tapping an option opens the detail
/// with the waypoint button, and an empty `options` result shows the empty
/// state. Mirrors hauling_screen_layout_test.dart.
library;

import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:eve_o_provit/api/asset_models.dart';
import 'package:eve_o_provit/api/trading_api.dart';
import 'package:eve_o_provit/core/theme.dart';
import 'package:eve_o_provit/features/trading/providers.dart';
import 'package:eve_o_provit/features/trading/sell_assets_providers.dart';
import 'package:eve_o_provit/features/trading/sell_assets_screen.dart';

// ---------------------------------------------------------------------------
// Canned data + fakes
// ---------------------------------------------------------------------------

AssetsResponse _fakeAssets() => const AssetsResponse(
      count: 1,
      assets: [
        AssetItem(
          typeId: 34,
          name: 'Tritanium',
          quantity: 120000,
          locationId: 60003760,
          locationName: 'Jita IV - Moon 4 - CNAP',
          systemId: 30000142,
          regionId: 10000002,
          marketable: true,
        ),
      ],
    );

SellOptionsResponse _fakeResponse() => const SellOptionsResponse(
      typeId: 34,
      name: 'Tritanium',
      quantity: 120000,
      originSystemId: 30000142,
      best: SellOption(
        scope: 'hub',
        regionId: 10000002,
        regionName: 'The Forge',
        stationId: 60003760,
        stationName: 'Jita IV - Moon 4 - CNAP',
        systemName: 'Jita',
        buyPrice: 5.5,
        unitNet: 5.36,
        totalNet: 643200,
        jumps: 0,
        travelTimeMin: 0,
        securityRisk: 'safe',
        hasData: true,
      ),
      options: [
        SellOption(
          scope: 'hub',
          regionId: 10000002,
          regionName: 'The Forge',
          stationId: 60003760,
          stationName: 'Jita IV - Moon 4 - CNAP',
          systemName: 'Jita',
          buyPrice: 5.5,
          unitNet: 5.36,
          totalNet: 643200,
          jumps: 0,
          travelTimeMin: 0,
          securityRisk: 'safe',
          hasData: true,
        ),
        SellOption(
          scope: 'current_region',
          regionId: 10000002,
          regionName: 'The Forge',
          stationId: 60000002,
          stationName: 'Sobaseki Outpost',
          systemName: 'Sobaseki',
          buyPrice: 5.1,
          unitNet: 4.97,
          totalNet: 596400,
          jumps: 3,
          travelTimeMin: 5.0,
          securityRisk: 'danger',
          hasData: true,
        ),
        SellOption(
          scope: 'current_region',
          regionId: 10000002,
          regionName: 'The Forge',
          stationId: 60000003,
          stationName: 'Nowhere Station',
          systemName: 'Nowhere',
          buyPrice: 0,
          unitNet: 0,
          totalNet: 0,
          jumps: 0,
          travelTimeMin: 0,
          securityRisk: 'caution',
          hasData: false,
        ),
      ],
      skillsApplied: SkillsApplied(
        applied: true,
        accounting: 5,
        brokerRelations: 5,
        salesTaxRate: 0.025,
        brokerFeeRate: 0.015,
      ),
    );

SellOptionsResponse _emptyResponse() => const SellOptionsResponse(
      typeId: 34,
      name: 'Tritanium',
      quantity: 120000,
      originSystemId: 30000142,
      best: null,
      options: [],
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
  Future<AssetsResponse> listAssets() async => _fakeAssets();

  @override
  Future<SellOptionsResponse> findSellOptions(
    SellOptionsRequest request,
  ) async =>
      _fakeResponse();
}

class _MultiAssetApi extends TradingApi {
  _MultiAssetApi() : super(Dio());

  @override
  Future<AssetsResponse> listAssets() async => const AssetsResponse(
        count: 3,
        assets: [
          AssetItem(typeId: 1, name: 'Tritanium', quantity: 5, locationId: 60000001, locationName: 'A', systemId: 1, regionId: 1, marketable: true),
          AssetItem(typeId: 2, name: 'Arkonor', quantity: 100, locationId: 60000001, locationName: 'A', systemId: 1, regionId: 1, marketable: true),
          AssetItem(typeId: 3, name: 'Mexallon', quantity: 50, locationId: 60000001, locationName: 'A', systemId: 1, regionId: 1, marketable: true),
        ],
      );

  @override
  Future<SellOptionsResponse> findSellOptions(SellOptionsRequest request) async =>
      _emptyResponse();
}

class _StubNotifier extends SellOptionsNotifier {
  @override
  Future<SellOptionsResponse?> build() async => _fakeResponse();
}

class _EmptyNotifier extends SellOptionsNotifier {
  @override
  Future<SellOptionsResponse?> build() async => _emptyResponse();
}

class _IdleNotifier extends SellOptionsNotifier {
  @override
  Future<SellOptionsResponse?> build() async => null;
}

Future<void> _pumpScreen(
  WidgetTester tester,
  double width, {
  double height = 1400,
  SellOptionsNotifier Function() notifier = _StubNotifier.new,
  TradingApi Function() api = _FakeTradingApi.new,
}) async {
  tester.view.physicalSize = Size(width, height);
  tester.view.devicePixelRatio = 1.0;
  addTearDown(tester.view.resetPhysicalSize);
  addTearDown(tester.view.resetDevicePixelRatio);

  await tester.pumpWidget(
    ProviderScope(
      overrides: [
        tradingApiProvider.overrideWithValue(api()),
        sellOptionsProvider.overrideWith(notifier),
      ],
      child: MaterialApp(
        theme: buildTheme(Brightness.light),
        home: const SellAssetsScreen(),
      ),
    ),
  );
  await tester.pumpAndSettle();
}

void main() {
  testWidgets('Single-pane (width 800): no VerticalDivider', (tester) async {
    await _pumpScreen(tester, 800);

    expect(find.byType(VerticalDivider), findsNothing);
    expect(find.byKey(const Key('sell-asset-search')), findsOneWidget);
  });

  testWidgets('Two-pane (width 1280): at least one VerticalDivider',
      (tester) async {
    await _pumpScreen(tester, 1280);

    expect(find.byType(VerticalDivider), findsAtLeastNWidgets(1));
    expect(find.byKey(const Key('sell-asset-search')), findsOneWidget);
  });

  testWidgets('Asset picker lists owned items with quantity + location',
      (tester) async {
    await _pumpScreen(tester, 1280);

    expect(find.byKey(const Key('sell-asset-list')), findsOneWidget);
    expect(find.text('Tritanium'), findsWidgets);
    expect(find.textContaining('Jita IV - Moon 4 - CNAP'), findsWidgets);
  });

  testWidgets('Option list renders system → station headers', (tester) async {
    await _pumpScreen(tester, 1280);

    expect(find.byKey(const Key('sell-option-list')), findsOneWidget);
    // system_name headers (best is also Jita, so findsWidgets).
    expect(find.text('Jita'), findsWidgets);
    expect(find.text('Sobaseki'), findsOneWidget);
    // station_name subtitle.
    expect(find.text('Sobaseki Outpost'), findsOneWidget);
  });

  testWidgets('has_data:false option renders muted "kein Marktpreis"',
      (tester) async {
    await _pumpScreen(tester, 1280);

    expect(find.text('kein Marktpreis'), findsOneWidget);
    expect(find.text('Nowhere'), findsOneWidget);
  });

  testWidgets('Security risk chips use the correct colour', (tester) async {
    await _pumpScreen(tester, 1280);

    final safeChip = tester.widget<Container>(
      find.byKey(const Key('sell-risk-chip-safe')).first,
    );
    final safeDeco = safeChip.decoration as BoxDecoration;
    expect(safeDeco.color, sellSecurityRiskColor('safe').withAlpha(40));
    expect(sellSecurityRiskColor('safe'), const Color(0xFF66BB6A)); // green

    final dangerChip = tester.widget<Container>(
      find.byKey(const Key('sell-risk-chip-danger')).first,
    );
    final dangerDeco = dangerChip.decoration as BoxDecoration;
    expect(dangerDeco.color, sellSecurityRiskColor('danger').withAlpha(40));
    expect(sellSecurityRiskColor('danger'), const Color(0xFFF44336)); // red

    // caution maps to amber.
    expect(sellSecurityRiskColor('caution'), const Color(0xFFFF9800));
  });

  testWidgets('Scope badge renders Hub and Region labels', (tester) async {
    await _pumpScreen(tester, 1280);

    expect(find.byKey(const Key('sell-scope-badge-hub')), findsWidgets);
    expect(
        find.byKey(const Key('sell-scope-badge-current_region')), findsWidgets);
    expect(find.text('Hub'), findsWidgets);
    expect(find.text('Region'), findsWidgets);
  });

  testWidgets('Tapping an option opens the detail with the waypoint button',
      (tester) async {
    await _pumpScreen(tester, 1280);

    // Tap the second (current_region) option; the best+first share station id.
    await tester.tap(find.text('Sobaseki Outpost'));
    await tester.pumpAndSettle();

    expect(find.byKey(const Key('sell-waypoint-button')), findsOneWidget);
    expect(find.text('Route an EVE übertragen'), findsOneWidget);
  });

  testWidgets('Empty options result shows the empty-state', (tester) async {
    await _pumpScreen(tester, 1280, notifier: _EmptyNotifier.new);

    expect(find.byKey(const Key('sell-empty-state')), findsOneWidget);
    expect(find.text('Keine Verkaufsorte gefunden'), findsOneWidget);
    expect(find.byKey(const Key('sell-option-list')), findsNothing);
  });

  testWidgets('Idle state shows the hint before any search', (tester) async {
    await _pumpScreen(tester, 1280, notifier: _IdleNotifier.new);

    expect(find.text('Item wählen und Verkaufsorte suchen'), findsOneWidget);
    expect(find.byKey(const Key('sell-option-list')), findsNothing);
  });

  testWidgets('Asset list sorts by name (default) then by quantity',
      (tester) async {
    await _pumpScreen(tester, 1280,
        notifier: _IdleNotifier.new, api: _MultiAssetApi.new);

    double dy(String name) => tester.getTopLeft(find.text(name)).dy;

    // Default: name ascending → Arkonor, Mexallon, Tritanium.
    expect(dy('Arkonor') < dy('Mexallon'), isTrue);
    expect(dy('Mexallon') < dy('Tritanium'), isTrue);

    // Toggle direction → name descending.
    await tester.tap(find.byKey(const Key('sell-sort-dir')));
    await tester.pumpAndSettle();
    expect(dy('Tritanium') < dy('Mexallon'), isTrue);
    expect(dy('Mexallon') < dy('Arkonor'), isTrue);

    // Sort by quantity ascending (5 Tritanium, 50 Mexallon, 100 Arkonor).
    await tester.tap(find.byKey(const Key('sell-sort-quantity')));
    await tester.tap(find.byKey(const Key('sell-sort-dir'))); // back to asc
    await tester.pumpAndSettle();
    expect(dy('Tritanium') < dy('Mexallon'), isTrue);
    expect(dy('Mexallon') < dy('Arkonor'), isTrue);
  });
}
