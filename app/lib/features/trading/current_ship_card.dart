/// CurrentShipCard — read-only display of the character's CURRENT (flown) ship.
///
/// Mirrors the web `CurrentShipCard`: the ship is no longer user-selectable, so
/// every trading view shows the current ship (name + cargo) with a REFRESH
/// control instead of a dropdown.
///
/// Cargo display logic (mirrors the web):
///   • effective cargo known (> 0)        → just the figure, e.g. "9.7k m³"
///   • only base cargo known              → "Basis 2.7k m³"
///   • effective_cargo_unavailable (error)→ "Basis 2.7k m³ — fitted unbekannt"
///
/// FAIL LOUD: when the fetch errored we surface a visible error message — never
/// silently render "no ship".
library;

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../character/character_models.dart';
import '../character/providers.dart';

/// Read-only "Schiff" widget with a refresh icon.
class CurrentShipCard extends ConsumerWidget {
  const CurrentShipCard({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final async = ref.watch(currentShipProvider);
    final loading = async.isLoading;
    final muted = Theme.of(context).colorScheme.onSurface.withAlpha(153);

    return Column(
      key: const Key('current-ship-card'),
      crossAxisAlignment: CrossAxisAlignment.start,
      mainAxisSize: MainAxisSize.min,
      children: [
        // ── Header: "Schiff" + refresh ──────────────────────────────────────
        Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            const Text(
              'Schiff',
              style: TextStyle(fontSize: 13, fontWeight: FontWeight.w600),
            ),
            const SizedBox(width: 4),
            IconButton(
              key: const Key('current-ship-refresh'),
              visualDensity: VisualDensity.compact,
              padding: EdgeInsets.zero,
              constraints: const BoxConstraints(minWidth: 28, minHeight: 28),
              iconSize: 16,
              tooltip: 'Aktuelles Schiff neu laden',
              onPressed:
                  loading ? null : () => ref.invalidate(currentShipProvider),
              icon: loading
                  ? const SizedBox(
                      width: 14,
                      height: 14,
                      child: CircularProgressIndicator(strokeWidth: 2),
                    )
                  : const Icon(Icons.refresh_rounded),
            ),
          ],
        ),
        // ── Body: error / loading / ship ────────────────────────────────────
        async.when(
          // FAIL LOUD — a load error is shown, never collapsed into "no ship".
          error: (err, _) => Text(
            'Aktuelles Schiff konnte nicht geladen werden',
            key: const Key('current-ship-error'),
            style: TextStyle(
              fontSize: 13,
              color: Theme.of(context).colorScheme.error,
            ),
          ),
          loading: () => Text(
            'Lade aktuelles Schiff…',
            style: TextStyle(fontSize: 13, color: muted),
          ),
          data: (ship) {
            if (ship == null || ship.shipTypeId <= 0) {
              return Text(
                'Kein aktives Schiff',
                style: TextStyle(fontSize: 13, color: muted),
              );
            }
            return Text(
              '${ship.shipName} (${_cargoDisplay(ship)})',
              style: TextStyle(fontSize: 13, color: muted),
            );
          },
        ),
      ],
    );
  }
}

/// Builds the cargo label, mirroring the web `cargoDisplay`.
String _cargoDisplay(CharacterShip ship) {
  final eff = ship.effectiveCargoCapacity;
  final useEffective = eff != null && eff > 0;
  final value = useEffective ? eff : ship.cargoCapacity;
  final cargoFmt = _formatCargo(value);
  if (ship.effectiveCargoUnavailable) {
    return 'Basis $cargoFmt — fitted unbekannt';
  }
  return useEffective ? cargoFmt : 'Basis $cargoFmt';
}

String _formatCargo(double m3) {
  if (m3 >= 1000) {
    return '${(m3 / 1000).toStringAsFixed(1)}k m³';
  }
  return '${m3.round()} m³';
}
