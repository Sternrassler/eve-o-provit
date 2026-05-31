/// Thin API client for the mining endpoints.
///
/// Follows the same pattern as [TradingApi]: takes the interceptor-equipped
/// [Dio] and maps HTTP ↔ DTOs. Business logic lives in the Riverpod layer.
library;

import 'package:dio/dio.dart';

import '../../api/mining_models.dart';

class MiningApi {
  MiningApi(this._dio);

  final Dio _dio;

  // ── POST /api/v1/mining/ore-ranking ────────────────────────────────────────

  /// Returns ore types ranked by best ISK/hour for the given region + sec band.
  /// An empty [rows] list with [noMiningSetup] = true means no mining lasers.
  Future<OreRankingResponse> oreRanking(OreRankingRequest request) async {
    final response = await _dio.post<Map<String, dynamic>>(
      '/api/v1/mining/ore-ranking',
      data: request.toJson(),
    );
    return OreRankingResponse.fromJson(response.data!);
  }
}
