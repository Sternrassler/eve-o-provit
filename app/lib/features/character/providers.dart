/// Riverpod providers for the character feature.
///
/// Provider graph (no cycles):
///   dioProvider → characterApiProvider
///   characterApiProvider → characterProvider, activeShipProvider, skillsProvider
library;

import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../api/dio_client.dart';
import 'character_api.dart';
import 'character_models.dart';

// ---------------------------------------------------------------------------
// CharacterApi provider
// ---------------------------------------------------------------------------

/// Provides a [CharacterApi] backed by the interceptor-equipped [Dio].
final characterApiProvider = Provider<CharacterApi>((ref) {
  return CharacterApi(ref.watch(dioProvider));
});

// ---------------------------------------------------------------------------
// Data providers
// ---------------------------------------------------------------------------

/// Fetches the authenticated character's info (id, name, scopes, portrait).
///
/// Cached for the lifetime of the container.
final characterProvider = FutureProvider<CharacterInfo>((ref) async {
  final api = ref.watch(characterApiProvider);
  return api.me();
});

/// Fetches the character's currently active ship.
///
/// Cached for the lifetime of the container.
final activeShipProvider = FutureProvider<CharacterShip>((ref) async {
  final api = ref.watch(characterApiProvider);
  return api.activeShip();
});

/// Fetches the character's skills, keyed by [characterId].
///
/// Uses [FutureProvider.family] so callers pass the character ID.
final skillsProvider =
    FutureProvider.family<CharacterSkillsData, int>((ref, characterId) async {
  final api = ref.watch(characterApiProvider);
  return api.skills(characterId);
});
