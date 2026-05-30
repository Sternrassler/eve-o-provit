/// ShipSelect — dropdown of the character's ships.
///
/// Options = the character's **active ship** (so the current-ship pre-fill is
/// always selectable, even when you're flying a ship that isn't sitting in your
/// hangar) merged with the hangar ships from [characterShipsProvider]. Falls
/// back to a list of common haulers when no character ships are available.
/// Selecting a ship writes to [selectedShipTypeIdProvider].
library;

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../character/providers.dart';
import '../trading/providers.dart';

// ---------------------------------------------------------------------------
// Fallback ships (mirrors web mock-data/ships — common haulers)
// ---------------------------------------------------------------------------

class _FallbackShip {
  const _FallbackShip(this.typeId, this.name, this.cargoM3);
  final int typeId;
  final String name;
  final double cargoM3;
}

const List<_FallbackShip> _fallbackShips = [
  _FallbackShip(648, 'Badger', 9656.0),
  _FallbackShip(650, 'Nereus', 9656.0),
  _FallbackShip(655, 'Iteron Mark V', 63338.0),
  _FallbackShip(657, 'Mammoth', 27289.0),
  _FallbackShip(4363, 'Tayra', 26500.0),
  _FallbackShip(33693, 'Miasmos', 63338.0),
];

/// A uniform dropdown option. [cargoM3] is the base hull cargo;
/// [effectiveM3] is the optimizer-value (hull + skills + modules) when known.
class _ShipOpt {
  const _ShipOpt(this.typeId, this.name, this.cargoM3, [this.effectiveM3]);
  final int typeId;
  final String name;
  final double cargoM3;
  final double? effectiveM3;
}

// ---------------------------------------------------------------------------
// ShipSelect widget
// ---------------------------------------------------------------------------

class ShipSelect extends ConsumerStatefulWidget {
  const ShipSelect({super.key, this.enabled = true});

  /// Whether the control is interactive (e.g. false while calculating).
  final bool enabled;

  @override
  ConsumerState<ShipSelect> createState() => _ShipSelectState();
}

class _ShipSelectState extends ConsumerState<ShipSelect> {
  void _onDropdownChanged(int? typeId) {
    if (typeId == null) return;
    ref.read(selectedShipTypeIdProvider.notifier).select(typeId);
  }

  @override
  Widget build(BuildContext context) {
    final shipsAsync = ref.watch(characterShipsProvider);
    final activeAsync = ref.watch(activeShipProvider);
    final selectedTypeId = ref.watch(selectedShipTypeIdProvider);

    final loading = shipsAsync.isLoading;

    // Base list: the character's hangar ships, or the fallback haulers if none.
    final hangar = shipsAsync.value ?? const [];
    final base = <_ShipOpt>[];
    if (hangar.isNotEmpty) {
      for (final s in hangar) {
        base.add(_ShipOpt(
          s.typeId,
          s.typeName,
          s.cargoCapacity,
          s.effectiveCargoCapacity,
        ));
      }
    } else {
      for (final f in _fallbackShips) {
        base.add(_ShipOpt(f.typeId, f.name, f.cargoM3));
      }
    }

    // Merge the active ship in first (deduplicated): the ship you're currently
    // flying isn't necessarily in the hangar list, but the current-ship pre-fill
    // points at it, so it must be a selectable option.
    final opts = <_ShipOpt>[];
    final seen = <int>{};
    final active = activeAsync.value;
    if (active != null && active.shipTypeId > 0) {
      final name =
          active.shipTypeName.isNotEmpty ? active.shipTypeName : active.shipName;
      opts.add(_ShipOpt(
        active.shipTypeId,
        name,
        active.cargoCapacity,
        active.effectiveCargoCapacity,
      ));
      seen.add(active.shipTypeId);
    }
    for (final o in base) {
      if (seen.add(o.typeId)) opts.add(o);
    }

    final typeIds = opts.map((o) => o.typeId).toSet();
    final currentValue =
        typeIds.contains(selectedTypeId) ? selectedTypeId : null;

    return DropdownButtonFormField<int>(
      initialValue: currentValue,
      isExpanded: true,
      decoration: InputDecoration(
        labelText: loading ? 'Lade Schiffe…' : 'Schiff',
        border: const OutlineInputBorder(),
        contentPadding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
        isDense: true,
      ),
      hint: const Text('Schiff wählen…'),
      items: opts
          .map(
            (o) => DropdownMenuItem<int>(
              value: o.typeId,
              child: Text(
                _labelForShip(o),
                overflow: TextOverflow.ellipsis,
              ),
            ),
          )
          .toList(),
      onChanged: (widget.enabled && !loading) ? _onDropdownChanged : null,
    );
  }
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

/// Dropdown label: prefer the effective cargo (the optimizer's value, matches
/// EVE in-game); fall back to base with a "Basis" prefix so the bare number
/// can't be misread as the calc input.
String _labelForShip(_ShipOpt o) {
  final eff = o.effectiveM3;
  if (eff != null && eff > 0) {
    return '${o.name} (${_formatCargo(eff)})';
  }
  return '${o.name} (Basis ${_formatCargo(o.cargoM3)})';
}

String _formatCargo(double m3) {
  if (m3 >= 1000) {
    return '${(m3 / 1000).toStringAsFixed(1)}k m³';
  }
  return '${m3.round()} m³';
}
