// Unit tests for CharacterInfo and CharacterShip DTOs.
//
// JSON samples are taken from the real backend response shapes documented in
// backend/internal/models/api_responses.go and backend/internal/models/trading.go.

import 'package:eve_o_provit/features/character/character_models.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('CharacterInfo.fromJson', () {
    test('parses all fields from realistic backend sample', () {
      // Shape from cmd/api/main.go handleCharacterInfo
      final json = {
        'character_id': 12345678,
        'character_name': 'Karsten Flache',
        'scopes': [
          'esi-skills.read_skills.v1',
          'esi-location.read_location.v1',
        ],
        'portrait_url':
            'https://images.evetech.net/characters/12345678/portrait?size=128',
      };

      final info = CharacterInfo.fromJson(json);

      expect(info.characterId, equals(12345678));
      expect(info.characterName, equals('Karsten Flache'));
      expect(info.scopes, containsAll(['esi-skills.read_skills.v1']));
      expect(info.portraitUrl, contains('12345678'));
    });

    test('handles scopes as a space-separated string (legacy format)', () {
      final json = {
        'character_id': 99,
        'character_name': 'Test Pilot',
        'scopes': 'esi-skills.read_skills.v1 esi-location.read_location.v1',
        'portrait_url': '',
      };

      final info = CharacterInfo.fromJson(json);

      expect(info.scopes.length, equals(2));
      expect(info.scopes, contains('esi-skills.read_skills.v1'));
    });

    test('handles missing portrait_url gracefully', () {
      final json = {
        'character_id': 1,
        'character_name': 'X',
        'scopes': <String>[],
      };

      final info = CharacterInfo.fromJson(json);
      expect(info.portraitUrl, equals(''));
    });

    test('equality is based on characterId', () {
      final a = CharacterInfo.fromJson({
        'character_id': 7,
        'character_name': 'A',
        'scopes': <String>[],
        'portrait_url': '',
      });
      final b = CharacterInfo.fromJson({
        'character_id': 7,
        'character_name': 'B',
        'scopes': <String>[],
        'portrait_url': '',
      });

      expect(a, equals(b));
    });
  });

  group('CharacterShip.fromJson', () {
    test('parses all fields from realistic backend sample', () {
      // Shape from backend/internal/models/trading.go CharacterShip
      final json = {
        'ship_type_id': 650,
        'ship_name': 'My Nereus',
        'ship_item_id': 1000000000002,
        'ship_type_name': 'Nereus',
        'cargo_capacity': 2700.0,
      };

      final ship = CharacterShip.fromJson(json);

      expect(ship.shipTypeId, equals(650));
      expect(ship.shipName, equals('My Nereus'));
      expect(ship.shipItemId, equals(1000000000002));
      expect(ship.shipTypeName, equals('Nereus'));
      expect(ship.cargoCapacity, closeTo(2700.0, 0.01));
    });

    test('handles missing optional fields with defaults', () {
      final json = {
        'ship_type_id': 648,
        'ship_name': 'Badger',
        'ship_item_id': 9001,
      };

      final ship = CharacterShip.fromJson(json);

      expect(ship.shipTypeName, equals(''));
      expect(ship.cargoCapacity, equals(0.0));
    });
  });

  group('CharacterSkillsData.fromJson', () {
    test('parses nested skills structure from backend handler', () {
      // The character handler returns:
      // { "character_id": X, "skills": { "total_sp": Y, "skills": [...] } }
      final json = {
        'character_id': 12345678,
        'skills': {
          'total_sp': 50000000,
          'skills': [
            {
              'skill_id': 3340,
              'active_skill_level': 5,
              'trained_skill_level': 5,
              'skillpoints_in_skill': 256000,
            },
            {
              'skill_id': 3428,
              'active_skill_level': 4,
              'trained_skill_level': 4,
              'skillpoints_in_skill': 90510,
            },
          ],
        },
      };

      final data = CharacterSkillsData.fromJson(json);

      expect(data.characterId, equals(12345678));
      expect(data.totalSp, equals(50000000));
      expect(data.skills.length, equals(2));

      final first = data.skills.first;
      expect(first.skillId, equals(3340));
      expect(first.activeSkillLevel, equals(5));
      expect(first.skillpointsInSkill, equals(256000));
    });
  });

  group('CharacterFitting.fromJson', () {
    test('parses fitting response from backend handler', () {
      // Shape from backend/internal/handlers/fitting.go GetCharacterFitting
      final json = {
        'character_id': 12345678,
        'ship_type_id': 650,
        'effective_cargo_m3': 9656.9,
        'warp_speed_au_s': 6.87,
        'align_time_seconds': 4.82,
        'base_cargo_hold_m3': 2700.0,
        'base_warp_speed_au_s': 3.0,
        'fitted_modules': 5,
        'bonuses': {
          'cargo_bonus_m3': 6956.9,
          'warp_speed_multiplier': 2.29,
          'inertia_modifier': 0.8,
          'skills_bonus_m3': 675.0,
          'skills_bonus_pct': 25.0,
          'modules_bonus_m3': 4656.9,
        },
        'cached': true,
      };

      final fitting = CharacterFitting.fromJson(json);

      expect(fitting.shipTypeId, equals(650));
      expect(fitting.effectiveCargoM3, closeTo(9656.9, 0.01));
      expect(fitting.warpSpeedAuS, closeTo(6.87, 0.001));
      expect(fitting.fittedModules, equals(5));
      expect(fitting.cached, isTrue);
      expect(fitting.bonuses.skillsBonusPct, closeTo(25.0, 0.01));
    });
  });
}
