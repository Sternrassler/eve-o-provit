/// Riverpod providers for the character feature.
///
/// Provider graph (no cycles):
///   dioProvider → characterApiProvider
///   characterApiProvider → characterProvider, activeShipProvider, skillsProvider
library;

import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../api/dio_client.dart';
import '../../auth/auth_controller.dart';
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

/// The character's CURRENT (flown) ship — the single source of truth for every
/// trading view now that the ship dropdown is gone.
///
/// Mirrors the web `useCurrentShip` hook: gated on auth (returns null while
/// unauthenticated so a logged-out mount never fires a request), and re-runs
/// when auth flips. Errors are NOT swallowed — they propagate as [AsyncError]
/// so the UI can FAIL LOUD ("Aktuelles Schiff konnte nicht geladen werden")
/// instead of silently rendering "no ship".
///
/// Refresh by invalidating this provider (`ref.invalidate(currentShipProvider)`).
final currentShipProvider = FutureProvider<CharacterShip?>((ref) async {
  final auth = ref.watch(authControllerProvider).value;
  if (auth is! Authenticated) return null;
  final api = ref.watch(characterApiProvider);
  return api.activeShip();
});

/// Derives the `cargo_capacity` override to send with a calculation request,
/// from the current ship. Returns the effective (fitted) cargo only when it is
/// known and > 0 and NOT flagged unavailable; otherwise null (omit the field so
/// the backend falls back to its per-type computation). Mirrors the web's
/// `cargoOverride` / `buildPortfolioRequest` logic exactly.
double? cargoOverrideForShip(CharacterShip? ship) {
  if (ship == null) return null;
  if (ship.effectiveCargoUnavailable) return null;
  final eff = ship.effectiveCargoCapacity;
  if (eff == null || eff <= 0) return null;
  return eff;
}

/// Character's wallet balance in ISK; null when not authenticated or on error
/// (e.g. the character authorized before the wallet scope was added). Used to
/// pre-fill the ROI capital field.
final walletBalanceProvider = FutureProvider<double?>((ref) async {
  final auth = ref.watch(authControllerProvider).value;
  if (auth is! Authenticated) return null;
  final api = ref.watch(characterApiProvider);
  try {
    return await api.walletBalance();
  } catch (_) {
    return null;
  }
});

/// Region ID of the character's current solar system; null on error.
/// Used to pre-fill region selectors with the current region.
final currentRegionIdProvider = FutureProvider<int?>((ref) async {
  // Gate on auth: only fetch once authenticated, and re-run when auth flips so a
  // login transition refetches with a valid token (no stale pre-auth null cached).
  final auth = ref.watch(authControllerProvider).value;
  if (auth is! Authenticated) return null;
  final api = ref.watch(characterApiProvider);
  try {
    return await api.currentRegionId();
  } catch (_) {
    return null;
  }
});

/// Fetches the character's skills, keyed by [characterId].
///
/// Uses [FutureProvider.family] so callers pass the character ID.
final skillsProvider =
    FutureProvider.family<CharacterSkillsData, int>((ref, characterId) async {
  final api = ref.watch(characterApiProvider);
  return api.skills(characterId);
});
