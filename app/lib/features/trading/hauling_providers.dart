/// Riverpod providers for the Neighborhood Hauling Routes feature.
///
/// Provider graph (reuses [tradingApiProvider] from providers.dart):
///   tradingApiProvider → haulingRoutesProvider (AsyncNotifier)
///
/// The ship is the character's CURRENT ship (currentShipProvider); the screen
/// builds the [HaulingRequest] (incl. cargo override) and calls [find].
library;

import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../api/hauling_models.dart';
import 'providers.dart' show tradingApiProvider;

// ---------------------------------------------------------------------------
// HaulingRoutesNotifier — AsyncNotifier<HaulingResponse?>
// ---------------------------------------------------------------------------

/// Owns the hauling-route-search lifecycle.
///
/// [build] returns null — no search has been performed yet. Call [find] with a
/// [HaulingRequest] to trigger one; it sets [AsyncLoading] then guards the API
/// call. An empty `routes` list on the resulting [HaulingResponse] is a valid,
/// successful "no profitable routes" state — it is NOT treated as an error.
class HaulingRoutesNotifier extends AsyncNotifier<HaulingResponse?> {
  @override
  Future<HaulingResponse?> build() async => null;

  /// Runs a hauling-route search for [request].
  Future<void> find(HaulingRequest request) async {
    state = const AsyncLoading();
    state = await AsyncValue.guard(() async {
      final api = ref.read(tradingApiProvider);
      return api.findHaulingRoutes(request);
    });
  }
}

/// Provider for [HaulingRoutesNotifier].
final haulingRoutesProvider =
    AsyncNotifierProvider<HaulingRoutesNotifier, HaulingResponse?>(
  HaulingRoutesNotifier.new,
);
