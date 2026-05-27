/// ShipSelect — dropdown of character's ships sourced from [characterShipsProvider].
///
/// Selecting a ship writes to [selectedShipTypeIdProvider].
/// Falls back gracefully when the character has no ships or the fetch fails:
/// the current selection is kept and the dropdown shows a manual-entry option.
library;

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../trading/providers.dart';

// ---------------------------------------------------------------------------
// Fallback ships (mirrors web mock-data/ships — common haulers)
// ---------------------------------------------------------------------------

/// A minimal fallback entry used when no character ships are available.
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

// ---------------------------------------------------------------------------
// ShipSelect widget
// ---------------------------------------------------------------------------

/// Dropdown + optional manual-ID fallback for selecting a ship type.
///
/// Shows the character's hangar ships when available; otherwise shows fallback
/// ships.  Selecting sets [selectedShipTypeIdProvider].
class ShipSelect extends ConsumerStatefulWidget {
  const ShipSelect({super.key, this.enabled = true});

  /// Whether the control is interactive (e.g. false while calculating).
  final bool enabled;

  @override
  ConsumerState<ShipSelect> createState() => _ShipSelectState();
}

class _ShipSelectState extends ConsumerState<ShipSelect> {
  /// True when the user taps "Manuell eingeben" to type a raw type-ID.
  bool _showManualEntry = false;
  final _manualController = TextEditingController();

  @override
  void dispose() {
    _manualController.dispose();
    super.dispose();
  }

  void _onDropdownChanged(int? typeId) {
    if (typeId == null) return;
    ref.read(selectedShipTypeIdProvider.notifier).select(typeId);
  }

  void _onManualSubmit() {
    final v = int.tryParse(_manualController.text.trim());
    if (v != null && v > 0) {
      ref.read(selectedShipTypeIdProvider.notifier).select(v);
      setState(() => _showManualEntry = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final shipsAsync = ref.watch(characterShipsProvider);
    final selectedTypeId = ref.watch(selectedShipTypeIdProvider);

    return shipsAsync.when(
      loading: () => _buildDropdown(
        context: context,
        ships: _fallbackShips,
        selectedTypeId: selectedTypeId,
        loading: true,
      ),
      error: (err, stackTrace) => _buildDropdown(
        context: context,
        ships: _fallbackShips,
        selectedTypeId: selectedTypeId,
        loading: false,
      ),
      data: (ships) {
        final hasFetched = ships.isNotEmpty;

        if (!hasFetched) {
          // No hangar ships — use fallback list.
          return _buildDropdown(
            context: context,
            ships: _fallbackShips,
            selectedTypeId: selectedTypeId,
            loading: false,
          );
        }

        // Build dropdown from character's hangar ships.
        if (_showManualEntry) {
          return _buildManualEntry(context);
        }

        // Deduplicate by type id — the hangar can hold several ships of the
        // same type (e.g. two Ventures). DropdownButtonFormField requires a
        // unique value per item, and the selection is a ship TYPE id.
        final seenTypeIds = <int>{};
        final dedupShips =
            ships.where((s) => seenTypeIds.add(s.typeId)).toList();

        // Determine current value: must be in the list.
        final typeIds = dedupShips.map((s) => s.typeId).toList();
        final currentValue =
            typeIds.contains(selectedTypeId) ? selectedTypeId : null;

        return Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            DropdownButtonFormField<int>(
              initialValue: currentValue,
              isExpanded: true,
              decoration: const InputDecoration(
                labelText: 'Schiff',
                border: OutlineInputBorder(),
                contentPadding:
                    EdgeInsets.symmetric(horizontal: 12, vertical: 8),
                isDense: true,
              ),
              hint: const Text('Schiff wählen…'),
              items: [
                ...dedupShips.map(
                  (s) => DropdownMenuItem<int>(
                    value: s.typeId,
                    child: Text(
                      '${s.typeName} (${_formatCargo(s.cargoCapacity)})',
                      overflow: TextOverflow.ellipsis,
                    ),
                  ),
                ),
              ],
              onChanged: widget.enabled ? _onDropdownChanged : null,
            ),
            TextButton(
              onPressed: widget.enabled
                  ? () => setState(() => _showManualEntry = true)
                  : null,
              child: const Text('Typ-ID manuell eingeben'),
            ),
          ],
        );
      },
    );
  }

  Widget _buildDropdown({
    required BuildContext context,
    required List<_FallbackShip> ships,
    required int? selectedTypeId,
    required bool loading,
  }) {
    if (_showManualEntry) return _buildManualEntry(context);

    final typeIds = ships.map((s) => s.typeId).toList();
    final currentValue =
        typeIds.contains(selectedTypeId) ? selectedTypeId : null;

    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        DropdownButtonFormField<int>(
          initialValue: currentValue,
          isExpanded: true,
          decoration: InputDecoration(
            labelText: loading ? 'Lade Schiffe…' : 'Schiff',
            border: const OutlineInputBorder(),
            contentPadding:
                const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
            isDense: true,
          ),
          hint: const Text('Schiff wählen…'),
          items: ships
              .map(
                (s) => DropdownMenuItem<int>(
                  value: s.typeId,
                  child: Text(
                    '${s.name} (${_formatCargo(s.cargoM3)})',
                    overflow: TextOverflow.ellipsis,
                  ),
                ),
              )
              .toList(),
          onChanged: (widget.enabled && !loading) ? _onDropdownChanged : null,
        ),
        TextButton(
          onPressed: widget.enabled
              ? () => setState(() => _showManualEntry = true)
              : null,
          child: const Text('Typ-ID manuell eingeben'),
        ),
      ],
    );
  }

  Widget _buildManualEntry(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        TextFormField(
          controller: _manualController,
          keyboardType: TextInputType.number,
          inputFormatters: [FilteringTextInputFormatter.digitsOnly],
          decoration: const InputDecoration(
            labelText: 'Schiff Typ-ID',
            border: OutlineInputBorder(),
            contentPadding: EdgeInsets.symmetric(horizontal: 12, vertical: 8),
            isDense: true,
            hintText: '648',
          ),
          onFieldSubmitted: (_) => _onManualSubmit(),
        ),
        Row(
          mainAxisAlignment: MainAxisAlignment.end,
          children: [
            TextButton(
              onPressed: () => setState(() => _showManualEntry = false),
              child: const Text('Abbrechen'),
            ),
            TextButton(
              onPressed: _onManualSubmit,
              child: const Text('Übernehmen'),
            ),
          ],
        ),
      ],
    );
  }
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

String _formatCargo(double m3) {
  if (m3 >= 1000) {
    return '${(m3 / 1000).toStringAsFixed(1)}k m³';
  }
  return '${m3.round()} m³';
}
