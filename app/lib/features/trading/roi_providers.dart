/// Riverpod providers for the ROI Calculator / Capital Allocation feature.
///
/// Provider graph (reuses [tradingApiProvider] from providers.dart):
///   tradingApiProvider → roiOptimizeProvider (AsyncNotifier)
///
/// Region selection reuses the trading-feature [selectedRegionProvider]; the
/// ship is the character's CURRENT ship (currentShipProvider). The screen builds
/// the [PortfolioRequest] (incl. cargo override) and calls [optimize].
library;

import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../api/portfolio_models.dart';
import 'providers.dart' show tradingApiProvider;

// ---------------------------------------------------------------------------
// RoiOptimizeNotifier — AsyncNotifier<PortfolioResult?>
// ---------------------------------------------------------------------------

/// Owns the portfolio-optimization lifecycle.
///
/// [build] returns null — no optimization has been performed yet. Call
/// [optimize] with a [PortfolioRequest] to trigger one; it sets
/// [AsyncLoading] then guards the API call. An empty `items` list on the
/// resulting [PortfolioResult] is a valid, successful "no viable portfolio"
/// state — it is NOT treated as an error.
class RoiOptimizeNotifier extends AsyncNotifier<PortfolioResult?> {
  @override
  Future<PortfolioResult?> build() async => null;

  /// Runs a portfolio optimization for [request].
  Future<void> optimize(PortfolioRequest request) async {
    state = const AsyncLoading();
    state = await AsyncValue.guard(() async {
      final api = ref.read(tradingApiProvider);
      return api.optimizePortfolio(request);
    });
  }
}

/// Provider for [RoiOptimizeNotifier].
final roiOptimizeProvider =
    AsyncNotifierProvider<RoiOptimizeNotifier, PortfolioResult?>(
  RoiOptimizeNotifier.new,
);
