/// Riverpod providers for the Multi-Hub Comparison feature.
///
/// Provider graph (reuses [tradingApiProvider] from providers.dart):
///   tradingApiProvider → hubComparisonProvider (AsyncNotifier)
///
/// State leaves:
///   selectedSearchItemProvider — the item the user picked from search results.
library;

import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../api/hub_comparison_models.dart';
import 'providers.dart' show tradingApiProvider;

// ---------------------------------------------------------------------------
// Selected search item
// ---------------------------------------------------------------------------

/// Notifier for the currently selected search result; null until a pick.
class SelectedSearchItemNotifier extends Notifier<ItemSearchResult?> {
  @override
  ItemSearchResult? build() => null;

  /// Sets the selected item.
  void select(ItemSearchResult? item) => state = item;
}

/// Provider for the currently selected [ItemSearchResult].
final selectedSearchItemProvider =
    NotifierProvider<SelectedSearchItemNotifier, ItemSearchResult?>(
  SelectedSearchItemNotifier.new,
);

// ---------------------------------------------------------------------------
// HubComparisonNotifier — AsyncNotifier<HubComparisonResponse?>
// ---------------------------------------------------------------------------

/// Owns the hub-comparison lifecycle.
///
/// [build] returns null — no comparison has been performed yet. Call
/// [search] with an item type id to trigger a comparison; it sets
/// [AsyncLoading] then guards the API call.
class HubComparisonNotifier extends AsyncNotifier<HubComparisonResponse?> {
  @override
  Future<HubComparisonResponse?> build() async => null;

  /// Runs a hub comparison for [typeId].
  Future<void> search(int typeId) async {
    state = const AsyncLoading();
    state = await AsyncValue.guard(() async {
      final api = ref.read(tradingApiProvider);
      return api.compareHubs(typeId);
    });
  }
}

/// Provider for [HubComparisonNotifier].
final hubComparisonProvider =
    AsyncNotifierProvider<HubComparisonNotifier, HubComparisonResponse?>(
  HubComparisonNotifier.new,
);
