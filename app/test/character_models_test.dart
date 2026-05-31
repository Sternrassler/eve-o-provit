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
      // Effective-cargo fields default to absent / available.
      expect(ship.effectiveCargoCapacity, isNull);
      expect(ship.effectiveCargoUnavailable, isFalse);
    });

    test('parses effective_cargo_capacity + effective_cargo_unavailable', () {
      final json = {
        'ship_type_id': 650,
        'ship_name': 'My Nereus',
        'ship_item_id': 1000000000002,
        'ship_type_name': 'Nereus',
        'cargo_capacity': 2700.0,
        'effective_cargo_capacity': 9656.9,
        'effective_cargo_unavailable': false,
      };

      final ship = CharacterShip.fromJson(json);

      expect(ship.effectiveCargoCapacity, closeTo(9656.9, 0.01));
      expect(ship.effectiveCargoUnavailable, isFalse);
    });

    test('effective_cargo_unavailable=true is parsed (fitting errored)', () {
      final json = {
        'ship_type_id': 650,
        'ship_name': 'My Nereus',
        'ship_item_id': 1000000000002,
        'ship_type_name': 'Nereus',
        'cargo_capacity': 2700.0,
        'effective_cargo_unavailable': true,
      };

      final ship = CharacterShip.fromJson(json);

      expect(ship.effectiveCargoUnavailable, isTrue);
      expect(ship.effectiveCargoCapacity, isNull);
    });
  });

  group('CharacterSkillsData.fromJson', () {
    test('parses the flat TradingSkills map (PascalCase keys) from handler', () {
      // REAL shape: the handler serializes *TradingSkills (no json tags), so
      // the "skills" value is a FLAT skill-name → value map.
      // Mirrors backend/internal/handlers/character_test.go (Accounting=5,
      // BrokerRelations=4, Navigation=5).
      final json = {
        'character_id': 12345,
        'skills': {
          'Accounting': 5,
          'BrokerRelations': 4,
          'AdvancedBrokerRelations': 0,
          'FactionStanding': 0,
          'CorpStanding': 0,
          'SpaceshipCommand': 5,
          'Navigation': 5,
          'EvasiveManeuvering': 4,
          'GallenteIndustrial': 3,
          'CaldariIndustrial': 0,
          'AmarrIndustrial': 0,
          'MinmatarIndustrial': 0,
          'GallenteHauler': 2,
          'CaldariHauler': 0,
          'AmarrHauler': 0,
          'MinmatarHauler': 0,
        },
      };

      final data = CharacterSkillsData.fromJson(json);

      expect(data.characterId, equals(12345));
      // Real assertions against the actual backend test values.
      expect(data.level('Accounting'), equals(5));
      expect(data.level('BrokerRelations'), equals(4));
      expect(data.level('Navigation'), equals(5));
      expect(data.level('SpaceshipCommand'), equals(5));
      expect(data.level('EvasiveManeuvering'), equals(4));
      expect(data.level('GallenteIndustrial'), equals(3));
      expect(data.level('GallenteHauler'), equals(2));
      // The raw map is preserved as a flat name → value map.
      expect(data.skills['CaldariHauler'], equals(0));
    });

    test('level() returns 0 for an absent skill key', () {
      final data = CharacterSkillsData.fromJson({
        'character_id': 1,
        'skills': {'Navigation': 3},
      });
      expect(data.level('Navigation'), equals(3));
      expect(data.level('DoesNotExist'), equals(0));
    });

    test('parses float standing values without crashing', () {
      // FactionStanding / CorpStanding are float64 on the Go side.
      final data = CharacterSkillsData.fromJson({
        'character_id': 7,
        'skills': {'FactionStanding': 2.5, 'CorpStanding': -1.0},
      });
      expect(data.skills['FactionStanding'], closeTo(2.5, 0.001));
      // level() truncates the float to an int.
      expect(data.level('FactionStanding'), equals(2));
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
