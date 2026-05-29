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
import '../character/providers.dart';
import 'providers.dart';

mixin CurrentSelectionPrefill<T extends ConsumerStatefulWidget>
    on ConsumerState<T> {
  bool _shipPrefilled = false;
  bool _regionPrefilled = false;

  /// Kicks off the one-time region + ship pre-fill after the first frame.
  void startSelectionPrefill() {
    WidgetsBinding.instance.addPostFrameCallback((_) {
      _tryPrefillShip();
      // Region depends on two async sources (current region id + region list);
      // set up listeners once, then re-check on every change.
      ref.listenManual(currentRegionIdProvider, (_, _) => _tryPrefillRegion());
      ref.listenManual(regionsProvider, (_, _) => _tryPrefillRegion());
      _tryPrefillRegion();
    });
  }

  void _tryPrefillShip() {
    if (_shipPrefilled) return;
    final async = ref.read(activeShipTypeIdProvider);
    if (async.hasValue) {
      _applyShip(async.value);
    } else if (async is AsyncError) {
      _applyShip(null);
    } else {
      ref.listenManual(activeShipTypeIdProvider, (_, next) {
        if (_shipPrefilled) return;
        if (next.hasValue) {
          _applyShip(next.value);
        } else if (next is AsyncError) {
          _applyShip(null);
        }
      });
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
