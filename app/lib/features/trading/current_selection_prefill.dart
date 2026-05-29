/// Mixin that pre-fills the shared region + ship selection from the character's
/// CURRENT region and active ship — once, and only while the selection is still
/// empty (an explicit user choice is never overwritten).
///
/// Mirrors the web's `override ?? current ?? default` pattern: the fallback
/// (ship 648) is applied only once the active ship has resolved (data OR error),
/// never prematurely while it's still loading — otherwise a late-resolving
/// active ship (e.g. after a 401 → token refresh) could be blocked. The region
/// is seeded from the current region when available; if unavailable (not logged
/// in / error) the region is left untouched (no hard fallback).
///
/// Screens call [startSelectionPrefill] from `initState`.
library;

import 'package:flutter/widgets.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../api/trading_models.dart';
import '../../auth/auth_controller.dart';
import '../character/providers.dart';
import 'providers.dart';

mixin CurrentSelectionPrefill<T extends ConsumerStatefulWidget>
    on ConsumerState<T> {
  bool _shipPrefilled = false;
  bool _regionPrefilled = false;
  bool _refreshedForAuth = false;

  /// Kicks off the one-time region + ship pre-fill after the first frame.
  ///
  /// The prefill only finalizes once the user is [Authenticated] — otherwise a
  /// mount during the SSO login transition (token not yet available) would
  /// resolve the data providers to null and prematurely apply the fallback. We
  /// listen to the auth state and the data providers and re-attempt on each
  /// change; the gated providers refetch when auth flips, so the real values
  /// arrive and get applied.
  void startSelectionPrefill() {
    ref.listenManual(authControllerProvider, (_, _) => _attempt());
    ref.listenManual(currentRegionIdProvider, (_, _) => _attempt());
    ref.listenManual(regionsProvider, (_, _) => _attempt());
    ref.listenManual(activeShipProvider, (_, _) => _attempt());
    WidgetsBinding.instance.addPostFrameCallback((_) => _attempt());
  }

  void _attempt() {
    // Wait until authenticated before finalizing anything (incl. the fallback).
    if (ref.read(authControllerProvider).value is! Authenticated) return;

    // On the first authenticated tick, discard any value the gated providers
    // computed during the pre-auth window. Without this, a mount right after the
    // SSO login transition reads a stale AsyncData(null) (the dependent provider
    // hasn't re-run yet) and prematurely applies the ship fallback. Forcing a
    // fresh computation under auth makes the real values arrive before we apply.
    if (!_refreshedForAuth) {
      _refreshedForAuth = true;
      // Discard any value these providers computed during the pre-auth window
      // so the real values are fetched under auth before we apply.
      ref.invalidate(activeShipProvider);
      ref.invalidate(currentRegionIdProvider);
      return; // the providers' listeners re-invoke _attempt once they resolve
    }

    _tryPrefillShip();
    _tryPrefillRegion();
  }

  void _tryPrefillShip() {
    if (_shipPrefilled) return;
    // Read the active ship directly (the same provider ShipSelect uses). Act
    // only once it has resolved (data or error); _attempt() re-runs on changes.
    final async = ref.read(activeShipProvider);
    if (async.hasValue) {
      _applyShip(async.value?.shipTypeId);
    } else if (async is AsyncError) {
      _applyShip(null);
    }
  }

  void _applyShip(int? activeTypeId) {
    if (_shipPrefilled) return;
    _shipPrefilled = true;
    if (ref.read(selectedShipTypeIdProvider) == null) {
      ref.read(selectedShipTypeIdProvider.notifier).select(activeTypeId ?? 648);
    }
  }

  void _tryPrefillRegion() {
    if (_regionPrefilled) return;
    final idAsync = ref.read(currentRegionIdProvider);
    final regionsAsync = ref.read(regionsProvider);
    final idReady = idAsync.hasValue || idAsync is AsyncError;
    final regionsReady = regionsAsync.hasValue || regionsAsync is AsyncError;
    if (!idReady || !regionsReady) return;

    _regionPrefilled = true;
    // Never override an explicit user choice.
    if (ref.read(selectedRegionProvider) != null) return;

    final regionId = idAsync.value;
    final regions = regionsAsync.value;
    if (regionId == null || regions == null) return; // no current → leave empty

    Region? match;
    for (final r in regions) {
      if (r.id == regionId) {
        match = r;
        break;
      }
    }
    if (match != null) {
      ref.read(selectedRegionProvider.notifier).select(match);
    }
  }
}
