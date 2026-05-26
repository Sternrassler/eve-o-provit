/// Thin HTTP client for the character endpoints.
///
/// All methods use the interceptor-equipped [Dio] (bearer token + 401 retry).
/// Business logic lives in the Riverpod layer; this class only maps HTTP ↔ DTOs.
library;

import 'package:dio/dio.dart';

import 'character_models.dart';

class CharacterApi {
  CharacterApi(this._dio);

  final Dio _dio;

  // ── GET /api/v1/character ──────────────────────────────────────────────────

  /// Returns the authenticated character's info (id, name, scopes, portrait).
  Future<CharacterInfo> me() async {
    final response = await _dio.get<Map<String, dynamic>>('/api/v1/character');
    return CharacterInfo.fromJson(response.data!);
  }

  // ── GET /api/v1/character/ship ─────────────────────────────────────────────

  /// Returns the character's currently active ship.
  Future<CharacterShip> activeShip() async {
    final response =
        await _dio.get<Map<String, dynamic>>('/api/v1/character/ship');
    return CharacterShip.fromJson(response.data!);
  }

  // ── GET /api/v1/character/ships ────────────────────────────────────────────

  /// Returns all ships in the character's current hangar.
  Future<CharacterShipsResponse> ships() async {
    final response =
        await _dio.get<Map<String, dynamic>>('/api/v1/character/ships');
    return CharacterShipsResponse.fromJson(response.data!);
  }

  // ── GET /api/v1/characters/:characterId/skills ────────────────────────────

  /// Returns the character's trained skills and total skill points.
  Future<CharacterSkillsData> skills(int characterId) async {
    final response = await _dio.get<Map<String, dynamic>>(
      '/api/v1/characters/$characterId/skills',
    );
    return CharacterSkillsData.fromJson(response.data!);
  }

  // ── GET /api/v1/characters/:characterId/fitting/:shipTypeId ───────────────

  /// Returns fitting details (effective cargo, warp speed, align time) for
  /// the given character/ship combination.
  Future<CharacterFitting> fitting(int characterId, int shipTypeId) async {
    final response = await _dio.get<Map<String, dynamic>>(
      '/api/v1/characters/$characterId/fitting/$shipTypeId',
    );
    return CharacterFitting.fromJson(response.data!);
  }
}
