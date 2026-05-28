/// Widget tests for HubComparisonScreen adaptive layout.
///
/// 1. Small view (<840 dp) → single-pane: no VerticalDivider.
/// 2. Large view (≥840 dp) → two-pane: at least one VerticalDivider.
library;

import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:eve_o_provit/api/hub_comparison_models.dart';
import 'package:eve_o_provit/api/trading_api.dart';
import 'package:eve_o_provit/core/theme.dart';
import 'package:eve_o_provit/features/trading/hub_comparison_providers.dart';
import 'package:eve_o_provit/features/trading/hub_comparison_screen.dart';
import 'package:eve_o_provit/features/trading/providers.dart';

// ---------------------------------------------------------------------------
// Canned data + fakes
// ---------------------------------------------------------------------------

HubComparisonResponse _fakeResponse() => const HubComparisonResponse(
      typeId: 34,
      itemName: 'Tritanium',
      bestHubRegionId: 10000032,
      skillsApplied: SkillsApplied(
        applied: true,
        accounting: 5,
        brokerRelations: 5,
        salesTaxRate: 0.025,
        brokerFeeRate: 0.015,
      ),
      hubs: [
        HubRow(
          regionId: 10000032,
          regionName: 'Sinq Laison',
          hubName: 'Dodixie',
          systemId: 30002659,
          tier: 'secondary',
          hasData: true,
          buyPrice: 100,
          sellPrice: 110,
          spreadPercent: 10,
          netMarginPercent: 3.4,
          netProfitPerUnit: 4.1,
          dailyVolume: 125000,
          liquidityScore: 72,
          competition: CompetitionInfo(
            changesPerHour: 42.5,
            source: 'baseline',
          ),
        ),
        HubRow(
          regionId: 10000030,
          regionName: 'Heimatar',
          hubName: 'Rens',
          systemId: 30002510,
          tier: 'tertiary',
          hasData: false,
          buyPrice: 0,
          sellPrice: 0,
          spreadPercent: 0,
          netMarginPercent: 0,
          netProfitPerUnit: 0,
          dailyVolume: 0,
          liquidityScore: 0,
        ),
      ],
    );

class _FakeApi extends TradingApi {
  _FakeApi() : super(Dio());

  @override
  Future<HubComparisonResponse> compareHubs(int typeId) async =>
      _fakeResponse();
}

/// Stub notifier that seeds the comparison response immediately.
class _StubNotifier extends HubComparisonNotifier {
  @override
  Future<HubComparisonResponse?> build() async => _fakeResponse();
}

Future<void> _pumpScreen(
  WidgetTester tester,
  double width, {
  double height = 1000,
}) async {
  tester.view.physicalSize = Size(width, height);
  tester.view.devicePixelRatio = 1.0;
  addTearDown(tester.view.resetPhysicalSize);
  addTearDown(tester.view.resetDevicePixelRatio);

  await tester.pumpWidget(
    ProviderScope(
      overrides: [
        tradingApiProvider.overrideWithValue(_FakeApi()),
        hubComparisonProvider.overrideWith(_StubNotifier.new),
      ],
      child: MaterialApp(
        theme: buildTheme(Brightness.light),
        home: const HubComparisonScreen(),
      ),
    ),
  );
  await tester.pumpAndSettle();
}

void main() {
  testWidgets('Single-pane (width 800): no VerticalDivider', (tester) async {
    await _pumpScreen(tester, 800);

    expect(find.byType(VerticalDivider), findsNothing);
    // Search field + table both present.
    expect(find.text('Item suchen'), findsOneWidget);
    expect(find.text('Tritanium'), findsOneWidget);
  });

  testWidgets('Two-pane (width 1280): at least one VerticalDivider',
      (tester) async {
    await _pumpScreen(tester, 1280);

    expect(find.byType(VerticalDivider), findsAtLeastNWidgets(1));
    expect(find.text('Item suchen'), findsOneWidget);
    expect(find.text('Tritanium'), findsOneWidget);
  });

  testWidgets('Table renders hub names + recommended star + no-data row',
      (tester) async {
    await _pumpScreen(tester, 1280);

    expect(find.text('Dodixie'), findsOneWidget);
    expect(find.text('Rens'), findsOneWidget);
    // No-data hub shows the placeholder.
    expect(find.text('keine Daten'), findsOneWidget);
    // Recommended hub (Dodixie) is highlighted with a star icon.
    expect(find.byIcon(Icons.star_rounded), findsOneWidget);
  });
}
