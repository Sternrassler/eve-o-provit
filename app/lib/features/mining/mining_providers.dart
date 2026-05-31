/// Riverpod providers for the Mining Ore-Ranking feature.
///
/// Provider graph (mirrors hauling_providers.dart / roi_providers.dart):
///   dioProvider → miningApiProvider → oreRankingProvider (AsyncNotifier)
library;

import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../api/dio_client.dart';
import '../../api/mining_models.dart';
import 'mining_api.dart';

// ---------------------------------------------------------------------------
// MiningApi provider
// ---------------------------------------------------------------------------

/// Provides a [MiningApi] backed by the interceptor-equipped [Dio].
final miningApiProvider = Provider<MiningApi>((ref) {
  return MiningApi(ref.watch(dioProvider));
});

// ---------------------------------------------------------------------------
// OreRankingNotifier — AsyncNotifier<OreRankingResponse?>
// ---------------------------------------------------------------------------

/// Owns the ore-ranking lifecycle.
///
/// [build] returns null — no ranking has been performed yet. Call [run] with
/// an [OreRankingRequest] to trigger one; it sets [AsyncLoading] then guards
/// the API call. An empty [rows] list on the resulting [OreRankingResponse]
/// is a valid successful state (possibly with [noMiningSetup] = true) — it is
/// NOT treated as an error.
class OreRankingNotifier extends AsyncNotifier<OreRankingResponse?> {
  @override
  Future<OreRankingResponse?> build() async => null;

  /// Runs an ore ranking for [request].
  Future<void> run(OreRankingRequest request) async {
    state = const AsyncLoading();
    state = await AsyncValue.guard(() async {
      final api = ref.read(miningApiProvider);
      return api.oreRanking(request);
    });
  }
}

/// Provider for [OreRankingNotifier].
final oreRankingProvider =
    AsyncNotifierProvider<OreRankingNotifier, OreRankingResponse?>(
  OreRankingNotifier.new,
);
