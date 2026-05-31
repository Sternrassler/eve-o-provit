/// Widget tests for HaulingScreen adaptive layout + route list + detail.
///
/// 1. Small view (<840 dp) → single-pane: no VerticalDivider.
/// 2. Large view (≥840 dp) → two-pane: at least one VerticalDivider.
/// Plus: route list renders from a seeded response, the security-risk chip uses
/// the right colour, the cargo detail table renders on tap, and an empty
/// `routes` result shows the "no profitable routes" empty-state.
library;

import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:eve_o_provit/api/hauling_models.dart';
import 'package:eve_o_provit/api/trading_api.dart';
import 'package:eve_o_provit/core/theme.dart';
import 'package:eve_o_provit/core/security_risk.dart';
import 'package:eve_o_provit/features/character/character_api.dart';
import 'package:eve_o_provit/features/character/character_models.dart';
import 'package:eve_o_provit/features/character/providers.dart'
    show characterApiProvider, currentShipProvider;
import 'package:eve_o_provit/features/trading/hauling_providers.dart';
import 'package:eve_o_provit/features/trading/hauling_screen.dart';
import 'package:eve_o_provit/features/trading/providers.dart';

// ---------------------------------------------------------------------------
// Canned data + fakes
// ---------------------------------------------------------------------------

HaulingResponse _fakeResponse() => const HaulingResponse(
      originRegionId: 10000002,
      regionsScanned: [10000002, 10000016],
      routes: [
        HaulingRoute(
          buySystemName: 'Osmon',
          buyStationName: 'Osmon V - Moon 5 - Station',
          buyStationId: 60000001,
          sellSystemName: 'Jita',
          sellStationName: 'Jita IV - Moon 4 - Caldari Navy Assembly Plant',
          sellStationId: 60003760,
          jumps: 7,
          travelTimeMin: 12.5,
          securityRisk: 'safe',
          totalProfit: 1.2e8,
          totalCapital: 3.4e8,
          totalVolume: 2400,
          iskPerHour: 4.8e8,
          items: [
            HaulingItem(
              typeId: 34,
              name: 'Meta-4 Module A',
              quantity: 50,
              buyPrice: 1.0e6,
              sellPrice: 1.9e6,
              unitVolume: 10,
              profit: 4.5e7,
              profitPerM3: 90000,
            ),
          ],
        ),
        HaulingRoute(
          buySystemName: 'Hek',
          buyStationName: 'Hek VIII - Moon 12 - Boundless Creation',
          buyStationId: 60000002,
          sellSystemName: 'Rens',
          sellStationName: 'Rens VI - Moon 8 - Brutor Tribe Treasury',
          sellStationId: 60004588,
          jumps: 3,
          travelTimeMin: 5.0,
          securityRisk: 'danger',
          totalProfit: 6.0e7,
          totalCapital: 1.0e8,
          totalVolume: 1200,
          iskPerHour: 2.0e8,
          items: [
            HaulingItem(
              typeId: 35,
              name: 'Meta-4 Module B',
              quantity: 20,
              buyPrice: 5.0e5,
              sellPrice: 1.1e6,
              unitVolume: 5,
              profit: 1.2e7,
              profitPerM3: 120000,
            ),
          ],
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

HaulingResponse _emptyResponse() => const HaulingResponse(
      originRegionId: 10000002,
      regionsScanned: [10000002],
      routes: [],
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
  Future<HaulingResponse> findHaulingRoutes(HaulingRequest request) async =>
      _fakeResponse();
}

class _FakeCharacterApi extends CharacterApi {
  _FakeCharacterApi() : super(Dio());
}

const _currentShip = CharacterShip(
  shipTypeId: 650,
  shipName: 'My Nereus',
  shipItemId: 1000000000002,
  shipTypeName: 'Nereus',
  cargoCapacity: 2700.0,
  effectiveCargoCapacity: 9656.9,
);

class _StubNotifier extends HaulingRoutesNotifier {
  @override
  Future<HaulingResponse?> build() async => _fakeResponse();
}

class _EmptyNotifier extends HaulingRoutesNotifier {
  @override
  Future<HaulingResponse?> build() async => _emptyResponse();
}

Future<void> _pumpScreen(
  WidgetTester tester,
  double width, {
  double height = 1400,
  HaulingRoutesNotifier Function() notifier = _StubNotifier.new,
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
        currentShipProvider.overrideWith((ref) async => _currentShip),
        haulingRoutesProvider.overrideWith(notifier),
      ],
      child: MaterialApp(
        theme: buildTheme(Brightness.light),
        home: const HaulingScreen(),
      ),
    ),
  );
  await tester.pumpAndSettle();
}

void main() {
  testWidgets('Single-pane (width 800): no VerticalDivider', (tester) async {
    await _pumpScreen(tester, 800);

    expect(find.byType(VerticalDivider), findsNothing);
    expect(find.text('Kapital'), findsOneWidget);
    expect(find.text('Gefundene Routen'), findsOneWidget);
  });

  testWidgets('Two-pane (width 1280): at least one VerticalDivider',
      (tester) async {
    await _pumpScreen(tester, 1280);

    expect(find.byType(VerticalDivider), findsAtLeastNWidgets(1));
    expect(find.text('Kapital'), findsOneWidget);
    expect(find.text('Gefundene Routen'), findsOneWidget);
  });

  testWidgets('Route list renders buy → sell headers', (tester) async {
    await _pumpScreen(tester, 1280);

    expect(find.byKey(const Key('hauling-route-list')), findsOneWidget);
    expect(find.text('Osmon → Jita'), findsOneWidget);
    expect(find.text('Hek → Rens'), findsOneWidget);
  });

  testWidgets('Security risk chips use the correct colour', (tester) async {
    await _pumpScreen(tester, 1280);

    final safeChip = tester.widget<Container>(
      find.byKey(const Key('hauling-risk-chip-safe')).first,
    );
    final safeDeco = safeChip.decoration as BoxDecoration;
    expect(safeDeco.color, securityRiskColor('safe').withAlpha(40));
    expect(securityRiskColor('safe'), const Color(0xFF66BB6A)); // green

    final dangerChip = tester.widget<Container>(
      find.byKey(const Key('hauling-risk-chip-danger')).first,
    );
    final dangerDeco = dangerChip.decoration as BoxDecoration;
    expect(dangerDeco.color, securityRiskColor('danger').withAlpha(40));
    expect(securityRiskColor('danger'), const Color(0xFFF44336)); // red

    // caution maps to amber (no caution row in fixture, assert mapping directly)
    expect(securityRiskColor('caution'), const Color(0xFFFF9800));
  });

  testWidgets('Tapping a route opens the cargo detail table', (tester) async {
    await _pumpScreen(tester, 1280);

    await tester.tap(
      find.byKey(const Key('hauling-route-60000001-60003760')),
    );
    await tester.pumpAndSettle();

    expect(find.byKey(const Key('hauling-cargo-table')), findsOneWidget);
    expect(find.text('Meta-4 Module A'), findsOneWidget);
    // Cargo column headers.
    expect(find.text('Gewinn/m³'), findsOneWidget);
  });

  testWidgets('Empty routes result shows the empty-state', (tester) async {
    await _pumpScreen(tester, 1280, notifier: _EmptyNotifier.new);

    expect(find.byKey(const Key('hauling-empty-state')), findsOneWidget);
    expect(find.text('Keine profitablen Routen im Umkreis'), findsOneWidget);
    expect(find.byKey(const Key('hauling-route-list')), findsNothing);
  });
}
