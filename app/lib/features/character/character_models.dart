/// DTOs for the character API endpoints.
///
/// JSON field names are mapped exactly from the backend models/handlers:
///
/// GET /api/v1/character          → CharacterInfo   (handlers/main.go handleCharacterInfo)
/// GET /api/v1/character/ship     → CharacterShip   (models/trading.go CharacterShip)
/// GET /api/v1/character/ships    → CharacterShipsResponse (models/trading.go)
/// GET /api/v1/characters/:id/skills → CharacterSkillsData (services/character_helper.go)
/// GET /api/v1/characters/:id/fitting/:shipTypeId → CharacterFitting (models/api_responses.go)
library;

import 'package:flutter/foundation.dart';

// ---------------------------------------------------------------------------
// CharacterInfo — GET /api/v1/character
// ---------------------------------------------------------------------------

/// Response from `GET /api/v1/character`.
///
/// Backend shape (handleCharacterInfo in cmd/api/main.go):
/// ```json
/// {
///   "character_id": 12345678,
///   "character_name": "John Doe",
///   "scopes": ["esi-skills.read_skills.v1"],
///   "portrait_url": "https://images.evetech.net/characters/12345678/portrait?size=128"
/// }
/// ```
@immutable
class CharacterInfo {
  const CharacterInfo({
    required this.characterId,
    required this.characterName,
    required this.scopes,
    required this.portraitUrl,
  });

  final int characterId;
  final String characterName;
  final List<String> scopes;
  final String portraitUrl;

  factory CharacterInfo.fromJson(Map<String, dynamic> json) {
    final rawScopes = json['scopes'];
    final List<String> parsedScopes;
    if (rawScopes is List) {
      parsedScopes = rawScopes.cast<String>();
    } else if (rawScopes is String && rawScopes.isNotEmpty) {
      parsedScopes = rawScopes.split(' ');
    } else {
      parsedScopes = const [];
    }
    return CharacterInfo(
      characterId: (json['character_id'] as num).toInt(),
      characterName: json['character_name'] as String,
      scopes: parsedScopes,
      portraitUrl: json['portrait_url'] as String? ?? '',
    );
  }

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is CharacterInfo && characterId == other.characterId;

  @override
  int get hashCode => characterId.hashCode;

  @override
  String toString() => 'CharacterInfo(id: $characterId, name: $characterName)';
}

// ---------------------------------------------------------------------------
// CharacterShip — GET /api/v1/character/ship
// ---------------------------------------------------------------------------

/// Response from `GET /api/v1/character/ship`.
///
/// Backend shape (models/trading.go CharacterShip):
/// ```json
/// {
///   "ship_type_id": 650,
///   "ship_name": "Nereus",
///   "ship_item_id": 1000000000002,
///   "ship_type_name": "Nereus",
///   "cargo_capacity": 2700.0
/// }
/// ```
@immutable
class CharacterShip {
  const CharacterShip({
    required this.shipTypeId,
    required this.shipName,
    required this.shipItemId,
    required this.shipTypeName,
    required this.cargoCapacity,
  });

  final int shipTypeId;
  final String shipName;
  final int shipItemId;
  final String shipTypeName;
  final double cargoCapacity;

  factory CharacterShip.fromJson(Map<String, dynamic> json) {
    return CharacterShip(
      shipTypeId: (json['ship_type_id'] as num).toInt(),
      shipName: json['ship_name'] as String? ?? '',
      shipItemId: (json['ship_item_id'] as num).toInt(),
      shipTypeName: json['ship_type_name'] as String? ?? '',
      cargoCapacity: (json['cargo_capacity'] as num?)?.toDouble() ?? 0.0,
    );
  }

  @override
  String toString() =>
      'CharacterShip(typeId: $shipTypeId, name: $shipName, itemId: $shipItemId)';
}

// ---------------------------------------------------------------------------
// CharacterAssetShip — element in GET /api/v1/character/ships
// ---------------------------------------------------------------------------

/// A single ship in the character's asset hangar.
///
/// Backend shape (models/trading.go CharacterAssetShip):
/// ```json
/// {
///   "item_id": 1000000000002,
///   "type_id": 650,
///   "type_name": "Nereus",
///   "location_id": 60003760,
///   "location_name": "Jita IV",
///   "location_flag": "Hangar",
///   "cargo_capacity": 2700.0,
///   "is_singleton": true
/// }
/// ```
@immutable
class CharacterAssetShip {
  const CharacterAssetShip({
    required this.itemId,
    required this.typeId,
    required this.typeName,
    required this.locationId,
    required this.locationName,
    required this.locationFlag,
    required this.cargoCapacity,
    required this.isSingleton,
  });

  final int itemId;
  final int typeId;
  final String typeName;
  final int locationId;
  final String locationName;
  final String locationFlag;
  final double cargoCapacity;
  final bool isSingleton;

  factory CharacterAssetShip.fromJson(Map<String, dynamic> json) {
    return CharacterAssetShip(
      itemId: (json['item_id'] as num).toInt(),
      typeId: (json['type_id'] as num).toInt(),
      typeName: json['type_name'] as String? ?? '',
      locationId: (json['location_id'] as num).toInt(),
      locationName: json['location_name'] as String? ?? '',
      locationFlag: json['location_flag'] as String? ?? '',
      cargoCapacity: (json['cargo_capacity'] as num?)?.toDouble() ?? 0.0,
      isSingleton: json['is_singleton'] as bool? ?? false,
    );
  }
}

// ---------------------------------------------------------------------------
// CharacterShipsResponse — GET /api/v1/character/ships
// ---------------------------------------------------------------------------

/// Response from `GET /api/v1/character/ships`.
///
/// Backend shape (models/trading.go CharacterShipsResponse):
/// ```json
/// { "ships": [...], "count": 3 }
/// ```
@immutable
class CharacterShipsResponse {
  const CharacterShipsResponse({required this.ships, required this.count});

  final List<CharacterAssetShip> ships;
  final int count;

  factory CharacterShipsResponse.fromJson(Map<String, dynamic> json) {
    final rawShips = json['ships'] as List<dynamic>? ?? [];
    return CharacterShipsResponse(
      ships: rawShips
          .cast<Map<String, dynamic>>()
          .map(CharacterAssetShip.fromJson)
          .toList(),
      count: (json['count'] as num?)?.toInt() ?? 0,
    );
  }
}

// ---------------------------------------------------------------------------
// CharacterSkill + CharacterSkillsData — GET /api/v1/characters/:id/skills
// ---------------------------------------------------------------------------

/// A single skill entry.
///
/// Backend shape (services/character_helper.go CharacterSkill):
/// ```json
/// {
///   "skill_id": 3340,
///   "active_skill_level": 5,
///   "trained_skill_level": 5,
///   "skillpoints_in_skill": 256000
/// }
/// ```
@immutable
class CharacterSkill {
  const CharacterSkill({
    required this.skillId,
    required this.activeSkillLevel,
    required this.trainedSkillLevel,
    required this.skillpointsInSkill,
  });

  final int skillId;
  final int activeSkillLevel;
  final int trainedSkillLevel;
  final int skillpointsInSkill;

  factory CharacterSkill.fromJson(Map<String, dynamic> json) {
    return CharacterSkill(
      skillId: (json['skill_id'] as num).toInt(),
      activeSkillLevel: (json['active_skill_level'] as num).toInt(),
      trainedSkillLevel: (json['trained_skill_level'] as num).toInt(),
      skillpointsInSkill: (json['skillpoints_in_skill'] as num).toInt(),
    );
  }
}

/// Response body from `GET /api/v1/characters/:characterId/skills`.
///
/// Handler returns:
/// ```json
/// {
///   "character_id": 12345678,
///   "skills": {
///     "total_sp": 50000000,
///     "skills": [...]
///   }
/// }
/// ```
@immutable
class CharacterSkillsData {
  const CharacterSkillsData({
    required this.characterId,
    required this.totalSp,
    required this.skills,
  });

  final int characterId;
  final int totalSp;
  final List<CharacterSkill> skills;

  factory CharacterSkillsData.fromJson(Map<String, dynamic> json) {
    // The handler wraps CharacterSkills under the key "skills"
    final skillsMap = json['skills'] as Map<String, dynamic>? ?? {};
    final rawSkills = skillsMap['skills'] as List<dynamic>? ?? [];
    return CharacterSkillsData(
      characterId: (json['character_id'] as num).toInt(),
      totalSp: (skillsMap['total_sp'] as num?)?.toInt() ?? 0,
      skills: rawSkills
          .cast<Map<String, dynamic>>()
          .map(CharacterSkill.fromJson)
          .toList(),
    );
  }
}

// ---------------------------------------------------------------------------
// FittingBonuses + CharacterFitting — GET /api/v1/characters/:id/fitting/:shipTypeId
// ---------------------------------------------------------------------------

/// Bonus sub-object in the fitting response.
@immutable
class FittingBonuses {
  const FittingBonuses({
    required this.cargoBonusM3,
    required this.warpSpeedMultiplier,
    required this.inertiaModifier,
    required this.skillsBonusM3,
    required this.skillsBonusPct,
    required this.modulesBonusM3,
  });

  final double cargoBonusM3;
  final double warpSpeedMultiplier;
  final double inertiaModifier;
  final double skillsBonusM3;
  final double skillsBonusPct;
  final double modulesBonusM3;

  factory FittingBonuses.fromJson(Map<String, dynamic> json) {
    return FittingBonuses(
      cargoBonusM3: (json['cargo_bonus_m3'] as num?)?.toDouble() ?? 0.0,
      warpSpeedMultiplier:
          (json['warp_speed_multiplier'] as num?)?.toDouble() ?? 1.0,
      inertiaModifier: (json['inertia_modifier'] as num?)?.toDouble() ?? 1.0,
      skillsBonusM3: (json['skills_bonus_m3'] as num?)?.toDouble() ?? 0.0,
      skillsBonusPct: (json['skills_bonus_pct'] as num?)?.toDouble() ?? 0.0,
      modulesBonusM3: (json['modules_bonus_m3'] as num?)?.toDouble() ?? 0.0,
    );
  }
}

/// Response from `GET /api/v1/characters/:characterId/fitting/:shipTypeId`.
///
/// Backend shape (handlers/fitting.go GetCharacterFitting):
/// ```json
/// {
///   "character_id": 12345678,
///   "ship_type_id": 650,
///   "effective_cargo_m3": 9656.9,
///   "warp_speed_au_s": 6.87,
///   "align_time_seconds": 4.82,
///   "base_cargo_hold_m3": 2700.0,
///   "base_warp_speed_au_s": 3.0,
///   "fitted_modules": 5,
///   "bonuses": { ... },
///   "cached": true
/// }
/// ```
@immutable
class CharacterFitting {
  const CharacterFitting({
    required this.characterId,
    required this.shipTypeId,
    required this.effectiveCargoM3,
    required this.warpSpeedAuS,
    required this.alignTimeSeconds,
    required this.baseCargoHoldM3,
    required this.baseWarpSpeedAuS,
    required this.fittedModules,
    required this.bonuses,
    required this.cached,
  });

  final int characterId;
  final int shipTypeId;
  final double effectiveCargoM3;
  final double warpSpeedAuS;
  final double alignTimeSeconds;
  final double baseCargoHoldM3;
  final double baseWarpSpeedAuS;
  final int fittedModules;
  final FittingBonuses bonuses;
  final bool cached;

  factory CharacterFitting.fromJson(Map<String, dynamic> json) {
    final bonusesMap =
        json['bonuses'] as Map<String, dynamic>? ?? {};
    return CharacterFitting(
      characterId: (json['character_id'] as num).toInt(),
      shipTypeId: (json['ship_type_id'] as num).toInt(),
      effectiveCargoM3:
          (json['effective_cargo_m3'] as num?)?.toDouble() ?? 0.0,
      warpSpeedAuS: (json['warp_speed_au_s'] as num?)?.toDouble() ?? 0.0,
      alignTimeSeconds:
          (json['align_time_seconds'] as num?)?.toDouble() ?? 0.0,
      baseCargoHoldM3:
          (json['base_cargo_hold_m3'] as num?)?.toDouble() ?? 0.0,
      baseWarpSpeedAuS:
          (json['base_warp_speed_au_s'] as num?)?.toDouble() ?? 0.0,
      fittedModules: (json['fitted_modules'] as num?)?.toInt() ?? 0,
      bonuses: FittingBonuses.fromJson(bonusesMap),
      cached: json['cached'] as bool? ?? false,
    );
  }
}
