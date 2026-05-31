/// Mixin that pre-fills the shared region selection from the character's CURRENT
/// region — once, and only while the selection is still empty (an explicit user
/// choice is never overwritten).
///
/// The ship is no longer pre-filled here: every trading view now reads the
/// character's CURRENT ship directly via [currentShipProvider] (there is no ship
/// dropdown / selection state any more). This mixin only seeds the region.
///
/// The region is seeded from the current region when available; if unavailable
/// (not logged in / error) the region is left untouched (no hard fallback).
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
  bool _regionPrefilled = false;
  bool _refreshedForAuth = false;

  /// Kicks off the one-time region pre-fill after the first frame.
  ///
  /// The prefill only finalizes once the user is [Authenticated] — otherwise a
  /// mount during the SSO login transition (token not yet available) would
  /// resolve the data providers to null and prematurely give up. We listen to
  /// the auth state and the data providers and re-attempt on each change; the
  /// gated providers refetch when auth flips, so the real values arrive and get
  /// applied.
  void startSelectionPrefill() {
    ref.listenManual(authControllerProvider, (_, _) => _attempt());
    ref.listenManual(currentRegionIdProvider, (_, _) => _attempt());
    ref.listenManual(regionsProvider, (_, _) => _attempt());
    WidgetsBinding.instance.addPostFrameCallback((_) => _attempt());
  }

  void _attempt() {
    // Wait until authenticated before finalizing anything.
    if (ref.read(authControllerProvider).value is! Authenticated) return;

    // On the first authenticated tick, discard any value the gated provider
    // computed during the pre-auth window. Without this, a mount right after the
    // SSO login transition reads a stale AsyncData(null) (the dependent provider
    // hasn't re-run yet). Forcing a fresh computation under auth makes the real
    // value arrive before we apply.
    if (!_refreshedForAuth) {
      _refreshedForAuth = true;
      ref.invalidate(currentRegionIdProvider);
      return; // the provider's listener re-invokes _attempt once it resolves
    }

    _tryPrefillRegion();
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
