/// Riverpod providers for the Sell-from-Assets feature.
///
/// Provider graph (reuses [tradingApiProvider] from providers.dart):
///   tradingApiProvider → characterAssetsProvider (FutureProvider)
///                      → sellOptionsProvider (AsyncNotifier)
library;

import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../api/asset_models.dart';
import 'providers.dart' show tradingApiProvider;

// ---------------------------------------------------------------------------
// characterAssetsProvider — FutureProvider<AssetsResponse>
// ---------------------------------------------------------------------------

/// Fetches the character's owned items (the picker source) via
/// GET /api/v1/trading/assets. Errors propagate so the UI can surface them.
final characterAssetsProvider = FutureProvider<AssetsResponse>((ref) async {
  final api = ref.watch(tradingApiProvider);
  return api.listAssets();
});

// ---------------------------------------------------------------------------
// SellOptionsNotifier — AsyncNotifier<SellOptionsResponse?>
// ---------------------------------------------------------------------------

/// Owns the sell-options-search lifecycle.
///
/// [build] returns null — no search has been performed yet. Call [find] with a
/// [SellOptionsRequest] to trigger one; it sets [AsyncLoading] then guards the
/// API call. An empty `options` list on the resulting [SellOptionsResponse] is
/// a valid, successful "no sell locations found" state — NOT an error.
class SellOptionsNotifier extends AsyncNotifier<SellOptionsResponse?> {
  @override
  Future<SellOptionsResponse?> build() async => null;

  /// Runs a sell-options search for [request].
  Future<void> find(SellOptionsRequest request) async {
    state = const AsyncLoading();
    state = await AsyncValue.guard(() async {
      final api = ref.read(tradingApiProvider);
      return api.findSellOptions(request);
    });
  }

  /// Clears the current result (e.g. when a different asset is selected).
  void reset() {
    state = const AsyncData(null);
  }
}

/// Provider for [SellOptionsNotifier].
final sellOptionsProvider =
    AsyncNotifierProvider<SellOptionsNotifier, SellOptionsResponse?>(
  SellOptionsNotifier.new,
);
