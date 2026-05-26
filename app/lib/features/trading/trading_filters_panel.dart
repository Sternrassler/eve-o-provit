/// TradingFiltersPanel — collapsible filter panel matching the web TradingFilters.
///
/// Controls:
///   • Min. Spread slider (0–50, step 1, shows "X%")
///   • Min. Gewinn number input (step 10000)
///   • Max. Reisezeit slider (0–60, step 5, shows "X min")
///   • High / Low / Null sec checkboxes
///   • Zurücksetzen button
///
/// Wires directly to [filtersProvider].
library;

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'providers.dart';

// ---------------------------------------------------------------------------
// Web-matching defaults (mirror page.tsx defaultFilters)
// ---------------------------------------------------------------------------

const _defaultFilters = TradingFilters(
  minSpread: 5.0,
  minProfit: 100000.0,
  maxTravelTime: 30.0,
  highSec: true,
  lowSec: false,
  nullSec: false,
);

// ---------------------------------------------------------------------------
// TradingFiltersPanel
// ---------------------------------------------------------------------------

/// Collapsible filter panel, suitable for the left-column sidebar on tablets.
class TradingFiltersPanel extends ConsumerStatefulWidget {
  const TradingFiltersPanel({super.key});

  @override
  ConsumerState<TradingFiltersPanel> createState() =>
      _TradingFiltersPanelState();
}

class _TradingFiltersPanelState extends ConsumerState<TradingFiltersPanel> {
  bool _collapsed = false;

  /// Text controller for the Min-Profit input field — kept in sync with provider.
  final _profitController = TextEditingController();
  bool _profitEditing = false;

  @override
  void initState() {
    super.initState();
    final f = ref.read(filtersProvider);
    _profitController.text = f.minProfit.toStringAsFixed(0);
  }

  @override
  void dispose() {
    _profitController.dispose();
    super.dispose();
  }

  void _reset() {
    ref
        .read(filtersProvider.notifier)
        .update((_) => _defaultFilters);
    setState(() {
      _profitController.text = _defaultFilters.minProfit.toStringAsFixed(0);
    });
  }

  void _onProfitSubmit(String value) {
    final v = double.tryParse(value.replaceAll('.', '').replaceAll(',', ''));
    if (v != null && v >= 0) {
      ref
          .read(filtersProvider.notifier)
          .update((f) => f.copyWith(minProfit: v));
    } else {
      // Reset to provider value on invalid input.
      final current = ref.read(filtersProvider).minProfit;
      _profitController.text = current.toStringAsFixed(0);
    }
    setState(() => _profitEditing = false);
  }

  @override
  Widget build(BuildContext context) {
    final filters = ref.watch(filtersProvider);

    // Keep the text field in sync when provider changes externally (e.g. reset),
    // but only when the field is not being actively edited.
    if (!_profitEditing) {
      final expected = filters.minProfit.toStringAsFixed(0);
      if (_profitController.text != expected) {
        _profitController.text = expected;
      }
    }

    return Card(
      margin: EdgeInsets.zero,
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          // ── Header ────────────────────────────────────────────────────────
          Padding(
            padding:
                const EdgeInsets.symmetric(horizontal: 4, vertical: 2),
            child: Row(
              children: [
                const SizedBox(width: 12),
                const Expanded(
                  child: Text(
                    'Filter',
                    style: TextStyle(fontWeight: FontWeight.w600),
                  ),
                ),
                TextButton.icon(
                  onPressed: _reset,
                  icon: const Icon(Icons.refresh_rounded, size: 16),
                  label: const Text('Zurücksetzen'),
                  style: TextButton.styleFrom(
                    visualDensity: VisualDensity.compact,
                    padding: const EdgeInsets.symmetric(
                        horizontal: 8, vertical: 4),
                  ),
                ),
                IconButton(
                  onPressed: () => setState(() => _collapsed = !_collapsed),
                  icon: Icon(
                    _collapsed
                        ? Icons.expand_more_rounded
                        : Icons.expand_less_rounded,
                  ),
                  visualDensity: VisualDensity.compact,
                ),
              ],
            ),
          ),

          if (!_collapsed) ...[
            const Divider(height: 1),
            Padding(
              padding: const EdgeInsets.all(16),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  // ── Min. Spread ──────────────────────────────────────────
                  _FilterRow(
                    label: 'Min. Spread',
                    valueLabel: '${filters.minSpread.toInt()}%',
                  ),
                  Slider(
                    value: filters.minSpread,
                    min: 0,
                    max: 50,
                    divisions: 50,
                    label: '${filters.minSpread.toInt()}%',
                    onChanged: (v) => ref
                        .read(filtersProvider.notifier)
                        .update((f) => f.copyWith(minSpread: v)),
                  ),

                  const SizedBox(height: 8),

                  // ── Min. Gewinn ──────────────────────────────────────────
                  const Text(
                    'Min. Gewinn (ISK)',
                    style: TextStyle(fontSize: 13, fontWeight: FontWeight.w500),
                  ),
                  const SizedBox(height: 6),
                  TextFormField(
                    controller: _profitController,
                    keyboardType: TextInputType.number,
                    inputFormatters: [FilteringTextInputFormatter.digitsOnly],
                    decoration: const InputDecoration(
                      border: OutlineInputBorder(),
                      contentPadding:
                          EdgeInsets.symmetric(horizontal: 12, vertical: 8),
                      isDense: true,
                      suffixText: 'ISK',
                    ),
                    onTap: () => setState(() => _profitEditing = true),
                    onFieldSubmitted: _onProfitSubmit,
                    onEditingComplete: () =>
                        _onProfitSubmit(_profitController.text),
                  ),

                  const SizedBox(height: 16),

                  // ── Max. Reisezeit ───────────────────────────────────────
                  _FilterRow(
                    label: 'Max. Reisezeit',
                    valueLabel: '${filters.maxTravelTime.toInt()} min',
                  ),
                  Slider(
                    value: filters.maxTravelTime,
                    min: 0,
                    max: 60,
                    divisions: 12,
                    label: '${filters.maxTravelTime.toInt()} min',
                    onChanged: (v) => ref
                        .read(filtersProvider.notifier)
                        .update((f) => f.copyWith(maxTravelTime: v)),
                  ),

                  const SizedBox(height: 8),

                  // ── Sicherheitszonen ─────────────────────────────────────
                  const Text(
                    'Sicherheitszonen',
                    style: TextStyle(fontSize: 13, fontWeight: FontWeight.w500),
                  ),
                  const SizedBox(height: 8),
                  _SecCheckbox(
                    label: 'High Sec (≥ 0.5)',
                    value: filters.highSec,
                    onChanged: (v) => ref
                        .read(filtersProvider.notifier)
                        .update((f) => f.copyWith(highSec: v)),
                  ),
                  _SecCheckbox(
                    label: 'Low Sec (< 0.5)',
                    value: filters.lowSec,
                    activeColor: const Color(0xFFFF9800),
                    onChanged: (v) => ref
                        .read(filtersProvider.notifier)
                        .update((f) => f.copyWith(lowSec: v)),
                  ),
                  _SecCheckbox(
                    label: 'Null Sec (≤ 0.0)',
                    value: filters.nullSec,
                    activeColor: const Color(0xFFF44336),
                    onChanged: (v) => ref
                        .read(filtersProvider.notifier)
                        .update((f) => f.copyWith(nullSec: v)),
                  ),
                ],
              ),
            ),
          ],
        ],
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Small helpers
// ---------------------------------------------------------------------------

class _FilterRow extends StatelessWidget {
  const _FilterRow({required this.label, required this.valueLabel});
  final String label;
  final String valueLabel;

  @override
  Widget build(BuildContext context) {
    return Row(
      children: [
        Expanded(
          child: Text(
            label,
            style: const TextStyle(fontSize: 13, fontWeight: FontWeight.w500),
          ),
        ),
        const SizedBox(width: 8),
        Text(
          valueLabel,
          style: TextStyle(
            fontSize: 13,
            color: Theme.of(context).colorScheme.onSurface.withAlpha(153),
          ),
        ),
      ],
    );
  }
}

class _SecCheckbox extends StatelessWidget {
  const _SecCheckbox({
    required this.label,
    required this.value,
    required this.onChanged,
    this.activeColor,
  });

  final String label;
  final bool value;
  final ValueChanged<bool> onChanged;
  final Color? activeColor;

  @override
  Widget build(BuildContext context) {
    return CheckboxListTile(
      value: value,
      onChanged: (v) => onChanged(v ?? false),
      title: Text(label, style: const TextStyle(fontSize: 13)),
      controlAffinity: ListTileControlAffinity.leading,
      dense: true,
      contentPadding: EdgeInsets.zero,
      activeColor: activeColor ?? Theme.of(context).colorScheme.primary,
    );
  }
}
