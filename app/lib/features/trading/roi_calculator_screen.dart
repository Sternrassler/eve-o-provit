/// Adaptive ROI Calculator & Capital Allocation Optimizer screen.
///
/// The user picks a Region + Ship and enters their available Capital (ISK),
/// trading Time per day (minutes), a Liquidity-Cap %, a Max % of capital per
/// item and the security zones to include. The backend then suggests a capital
/// allocation across items (a portfolio) that maximizes daily profit, plus a
/// diversification score.
///
/// Layout (via [isTwoPane] / [kTwoPaneBreakpoint] = 840 dp):
///   • <840 dp  → single-pane: input form above a scrollable result table.
///   • ≥840 dp  → two-pane: input form on the left, result on the right.
library;

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../api/portfolio_models.dart';
import '../../api/trading_models.dart';
import '../../core/breakpoint.dart';
import '../../core/format.dart';
import '../character/providers.dart';
import 'current_selection_prefill.dart';
import 'providers.dart';
import 'roi_providers.dart';
import 'ship_select.dart';

// ---------------------------------------------------------------------------
// Public entry point
// ---------------------------------------------------------------------------

/// The ROI Calculator screen.
class RoiCalculatorScreen extends ConsumerWidget {
  const RoiCalculatorScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('ROI-Rechner'),
        centerTitle: false,
        elevation: 0,
        scrolledUnderElevation: 1,
      ),
      body: LayoutBuilder(
        builder: (context, constraints) {
          final twoPane = isTwoPane(constraints.maxWidth);
          if (twoPane) {
            return Row(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: const [
                SizedBox(
                  width: 340,
                  child: SingleChildScrollView(
                    padding: EdgeInsets.all(16),
                    child: _InputForm(),
                  ),
                ),
                VerticalDivider(width: 1),
                Expanded(child: _ResultPane()),
              ],
            );
          }
          return Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: const [
              Flexible(
                child: SingleChildScrollView(
                  padding: EdgeInsets.all(16),
                  child: _InputForm(),
                ),
              ),
              Divider(height: 1),
              Expanded(child: _ResultPane()),
            ],
          );
        },
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Input form — region/ship pickers + numeric inputs + sec-zone toggles
// ---------------------------------------------------------------------------

class _InputForm extends ConsumerStatefulWidget {
  const _InputForm();

  @override
  ConsumerState<_InputForm> createState() => _InputFormState();
}

class _InputFormState extends ConsumerState<_InputForm>
    with CurrentSelectionPrefill {
  // Numeric inputs.
  final _capitalController = TextEditingController(text: '500000000');
  double _timeBudgetMin = 120;
  double _liquidityCapPct = 10;
  double _maxItemPct = 30;

  // Sec-zone toggles (default: high only — mirrors the trading filter default).
  bool _highSec = true;
  bool _lowSec = false;
  bool _nullSec = false;

  String? _error;

  // Wallet → capital prefill: apply once, and never over an explicit user edit
  // (mirrors the region/ship `override ?? current ?? default` semantics).
  bool _capitalPrefilled = false;
  bool _capitalUserEdited = false;

  @override
  void initState() {
    super.initState();
    // Pre-fill region + ship with the character's current region/active ship.
    startSelectionPrefill();
    // Pre-fill capital with the character's wallet balance.
    ref.listenManual(walletBalanceProvider, (_, _) => _tryPrefillCapital());
    WidgetsBinding.instance
        .addPostFrameCallback((_) => _tryPrefillCapital());
  }

  void _tryPrefillCapital() {
    if (_capitalPrefilled || _capitalUserEdited) return;
    final async = ref.read(walletBalanceProvider);
    if (!async.hasValue && async is! AsyncError) return; // still loading
    _capitalPrefilled = true;
    final balance = async.value;
    if (balance == null || balance <= 0) return; // no wallet → keep default
    _capitalController.text = balance.floor().toString();
  }

  @override
  void dispose() {
    _capitalController.dispose();
    super.dispose();
  }

  List<String> get _secZones => [
        if (_highSec) 'high',
        if (_lowSec) 'low',
        if (_nullSec) 'null',
      ];

  Future<void> _optimize() async {
    final region = ref.read(selectedRegionProvider);
    final shipTypeId = ref.read(selectedShipTypeIdProvider);
    final capital =
        double.tryParse(_capitalController.text.replaceAll(RegExp(r'[^0-9.]'), ''));

    if (region == null) {
      setState(() => _error = 'Bitte eine Region auswählen.');
      return;
    }
    if (shipTypeId == null) {
      setState(() => _error = 'Bitte ein Schiff auswählen.');
      return;
    }
    if (capital == null || capital <= 0) {
      setState(() => _error = 'Bitte ein gültiges Kapital eingeben.');
      return;
    }
    if (_secZones.isEmpty) {
      setState(() => _error = 'Mindestens eine Sicherheitszone wählen.');
      return;
    }

    setState(() => _error = null);

    final request = PortfolioRequest(
      regionId: region.id,
      shipTypeId: shipTypeId,
      capital: capital,
      timeBudgetMin: _timeBudgetMin.round(),
      liquidityCapPct: _liquidityCapPct,
      maxItemPct: _maxItemPct,
      secZones: _secZones,
    );

    await ref.read(roiOptimizeProvider.notifier).optimize(request);
  }

  @override
  Widget build(BuildContext context) {
    final regionsAsync = ref.watch(regionsProvider);
    final selectedRegion = ref.watch(selectedRegionProvider);
    final busy = ref.watch(roiOptimizeProvider).isLoading;

    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      mainAxisSize: MainAxisSize.min,
      children: [
        // ── Region ─────────────────────────────────────────────────────────
        regionsAsync.when(
          loading: () => const LinearProgressIndicator(),
          error: (err, _) => const Text('Regionen nicht verfügbar'),
          data: (regions) => DropdownButtonFormField<Region>(
            initialValue:
                regions.contains(selectedRegion) ? selectedRegion : null,
            isExpanded: true,
            decoration: const InputDecoration(
              labelText: 'Region',
              border: OutlineInputBorder(),
              isDense: true,
              contentPadding:
                  EdgeInsets.symmetric(horizontal: 12, vertical: 8),
            ),
            hint: const Text('Region wählen…'),
            items: regions
                .map(
                  (r) => DropdownMenuItem(
                    value: r,
                    child: Text(r.name, overflow: TextOverflow.ellipsis),
                  ),
                )
                .toList(),
            onChanged: busy
                ? null
                : (r) =>
                    ref.read(selectedRegionProvider.notifier).select(r),
          ),
        ),
        const SizedBox(height: 12),

        // ── Ship ───────────────────────────────────────────────────────────
        ShipSelect(enabled: !busy),
        const SizedBox(height: 12),

        // ── Capital ────────────────────────────────────────────────────────
        TextField(
          controller: _capitalController,
          enabled: !busy,
          keyboardType: TextInputType.number,
          inputFormatters: [FilteringTextInputFormatter.digitsOnly],
          onChanged: (_) => _capitalUserEdited = true,
          decoration: const InputDecoration(
            labelText: 'Kapital',
            border: OutlineInputBorder(),
            isDense: true,
            suffixText: 'ISK',
            contentPadding:
                EdgeInsets.symmetric(horizontal: 12, vertical: 8),
          ),
        ),
        const SizedBox(height: 16),

        // ── Time budget ────────────────────────────────────────────────────
        _SliderRow(
          label: 'Zeit/Tag',
          valueLabel: '${_timeBudgetMin.round()} min',
        ),
        Slider(
          value: _timeBudgetMin,
          min: 15,
          max: 480,
          divisions: 31,
          label: '${_timeBudgetMin.round()} min',
          onChanged:
              busy ? null : (v) => setState(() => _timeBudgetMin = v),
        ),

        // ── Liquidity cap ──────────────────────────────────────────────────
        _SliderRow(
          label: 'Liquiditäts-Cap',
          valueLabel: '${_liquidityCapPct.round()}%',
        ),
        Slider(
          value: _liquidityCapPct,
          min: 1,
          max: 100,
          divisions: 99,
          label: '${_liquidityCapPct.round()}%',
          onChanged:
              busy ? null : (v) => setState(() => _liquidityCapPct = v),
        ),

        // ── Max % per item ─────────────────────────────────────────────────
        _SliderRow(
          label: 'Max. % / Item',
          valueLabel: '${_maxItemPct.round()}%',
        ),
        Slider(
          value: _maxItemPct,
          min: 1,
          max: 100,
          divisions: 99,
          label: '${_maxItemPct.round()}%',
          onChanged: busy ? null : (v) => setState(() => _maxItemPct = v),
        ),

        const SizedBox(height: 8),

        // ── Sec zones ──────────────────────────────────────────────────────
        const Text(
          'Sicherheitszonen',
          style: TextStyle(fontSize: 13, fontWeight: FontWeight.w500),
        ),
        const SizedBox(height: 4),
        _SecCheckbox(
          label: 'High Sec (≥ 0.5)',
          value: _highSec,
          enabled: !busy,
          onChanged: (v) => setState(() => _highSec = v),
        ),
        _SecCheckbox(
          label: 'Low Sec (< 0.5)',
          value: _lowSec,
          enabled: !busy,
          activeColor: const Color(0xFFFF9800),
          onChanged: (v) => setState(() => _lowSec = v),
        ),
        _SecCheckbox(
          label: 'Null Sec (≤ 0.0)',
          value: _nullSec,
          enabled: !busy,
          activeColor: const Color(0xFFF44336),
          onChanged: (v) => setState(() => _nullSec = v),
        ),

        if (_error != null) ...[
          const SizedBox(height: 8),
          Text(
            _error!,
            style: TextStyle(color: Theme.of(context).colorScheme.error),
          ),
        ],

        const SizedBox(height: 16),

        // ── Optimize button ────────────────────────────────────────────────
        FilledButton.icon(
          onPressed: busy ? null : _optimize,
          icon: busy
              ? const SizedBox(
                  width: 16,
                  height: 16,
                  child: CircularProgressIndicator(strokeWidth: 2),
                )
              : const Icon(Icons.auto_graph_rounded),
          label: const Text('Optimieren'),
        ),
      ],
    );
  }
}

// ---------------------------------------------------------------------------
// Result pane — async-aware
// ---------------------------------------------------------------------------

class _ResultPane extends ConsumerWidget {
  const _ResultPane();

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final async = ref.watch(roiOptimizeProvider);

    return async.when(
      loading: () => const Center(child: CircularProgressIndicator()),
      error: (err, _) => Center(
        child: Padding(
          padding: const EdgeInsets.all(24),
          child: Text(
            'Optimierung fehlgeschlagen.\n$err',
            textAlign: TextAlign.center,
            style: TextStyle(color: Theme.of(context).colorScheme.error),
          ),
        ),
      ),
      data: (result) {
        if (result == null) {
          return const _IdleHint();
        }
        if (result.isEmpty) {
          return const _EmptyResult();
        }
        return _PortfolioTable(result: result);
      },
    );
  }
}

class _IdleHint extends StatelessWidget {
  const _IdleHint();

  @override
  Widget build(BuildContext context) {
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(32),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(
              Icons.auto_graph_rounded,
              size: 56,
              color: Theme.of(context).colorScheme.onSurface.withAlpha(80),
            ),
            const SizedBox(height: 16),
            Text(
              'Parameter wählen und optimieren',
              style: Theme.of(context).textTheme.titleMedium?.copyWith(
                    color:
                        Theme.of(context).colorScheme.onSurface.withAlpha(120),
                  ),
            ),
          ],
        ),
      ),
    );
  }
}

class _EmptyResult extends StatelessWidget {
  const _EmptyResult();

  @override
  Widget build(BuildContext context) {
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(32),
        child: Column(
          key: const Key('roi-empty-state'),
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(
              Icons.sentiment_dissatisfied_rounded,
              size: 56,
              color: Theme.of(context).colorScheme.onSurface.withAlpha(80),
            ),
            const SizedBox(height: 16),
            Text(
              'Kein sinnvolles Portfolio für dieses Budget',
              textAlign: TextAlign.center,
              style: Theme.of(context).textTheme.titleMedium?.copyWith(
                    color:
                        Theme.of(context).colorScheme.onSurface.withAlpha(140),
                  ),
            ),
          ],
        ),
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Portfolio table + totals + diversification
// ---------------------------------------------------------------------------

class _PortfolioTable extends ConsumerWidget {
  const _PortfolioTable({required this.result});

  final PortfolioResult result;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final skills = result.skillsApplied;
    return SingleChildScrollView(
      padding: const EdgeInsets.all(16),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Text(
            'Empfohlenes Portfolio',
            style: Theme.of(context).textTheme.titleLarge,
          ),
          const SizedBox(height: 4),
          Text(
            skills.applied
                ? 'Skill-bereinigt (Accounting ${skills.accounting}, '
                    'Broker Relations ${skills.brokerRelations})'
                : 'Ohne Skill-Bereinigung',
            style: Theme.of(context).textTheme.bodySmall,
          ),
          const SizedBox(height: 16),

          // ── Summary tiles ──────────────────────────────────────────────
          Wrap(
            spacing: 12,
            runSpacing: 12,
            children: [
              _SummaryTile(
                label: 'Tagesgewinn',
                value: fmtIsk(result.totalDailyProfit),
                accent: Theme.of(context).colorScheme.primary,
              ),
              _SummaryTile(
                label: 'Kapital genutzt',
                value: fmtIsk(result.totalCapitalUsed),
              ),
              _SummaryTile(
                label: 'Zeit genutzt',
                value: '${result.timeUsedMin.round()} min',
              ),
              _SummaryTile(
                label: 'Diversifikation',
                value: '${result.diversificationScore.round()}/100',
              ),
            ],
          ),
          const SizedBox(height: 20),

          // ── Allocation table ───────────────────────────────────────────
          SingleChildScrollView(
            scrollDirection: Axis.horizontal,
            child: DataTable(
              key: const Key('roi-allocation-table'),
              columns: const [
                DataColumn(label: Text('Item')),
                DataColumn(label: Text('Route')),
                DataColumn(label: Text('Kapital'), numeric: true),
                DataColumn(label: Text('Units'), numeric: true),
                DataColumn(label: Text('Fahrten/Tag'), numeric: true),
                DataColumn(label: Text('Tagesgewinn'), numeric: true),
                DataColumn(label: Text('ROI%'), numeric: true),
                DataColumn(label: Text('EVE')),
              ],
              rows: [
                for (final item in result.items)
                  DataRow(
                    cells: [
                      DataCell(Text(item.name)),
                      DataCell(_RouteCell(item: item)),
                      DataCell(Text(fmtIsk(item.capitalUsed))),
                      DataCell(Text(fmtUnits(item.units))),
                      DataCell(Text(item.tripsPerDay.toStringAsFixed(1))),
                      DataCell(Text(fmtIsk(item.dailyProfit))),
                      DataCell(Text('${item.roiPercent.toStringAsFixed(1)}%')),
                      DataCell(_WaypointButton(item: item)),
                    ],
                  ),
              ],
            ),
          ),
        ],
      ),
    );
  }

}

// ---------------------------------------------------------------------------
// Route cell — compact "buy → sell" system names with station tooltip
// ---------------------------------------------------------------------------

class _RouteCell extends StatelessWidget {
  const _RouteCell({required this.item});

  final PortfolioItem item;

  @override
  Widget build(BuildContext context) {
    final hasRoute =
        item.buySystemName.isNotEmpty || item.sellSystemName.isNotEmpty;
    if (!hasRoute) {
      return const Text('—');
    }
    final label = '${item.buySystemName} → ${item.sellSystemName}';
    final tooltip = 'Kaufen: ${item.buyStationName}\n'
        'Verkaufen: ${item.sellStationName}';
    return Tooltip(
      message: tooltip,
      child: Text(label),
    );
  }
}

// ---------------------------------------------------------------------------
// Waypoint button — sets in-game autopilot waypoints (buy then sell)
//
// Reuses the exact pattern from route_detail.dart's `_setWaypoint`: clear the
// route and set the buy station first, then add the sell station, then show a
// success/error SnackBar.
// ---------------------------------------------------------------------------

class _WaypointButton extends ConsumerWidget {
  const _WaypointButton({required this.item});

  final PortfolioItem item;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final enabled = item.buyStationId > 0 && item.sellStationId > 0;
    return IconButton(
      key: Key('roi-waypoint-${item.typeId}'),
      icon: const Icon(Icons.navigation_rounded),
      tooltip: 'Route an EVE übertragen',
      visualDensity: VisualDensity.compact,
      onPressed: enabled ? () => _setWaypoint(context, ref) : null,
    );
  }

  Future<void> _setWaypoint(BuildContext context, WidgetRef ref) async {
    final api = ref.read(tradingApiProvider);
    try {
      // Clear existing route; set buy station first, then sell station.
      await api.setWaypoint(
        destinationId: item.buyStationId,
        clearOtherWaypoints: true,
        addToBeginning: false,
      );
      await api.setWaypoint(
        destinationId: item.sellStationId,
        clearOtherWaypoints: false,
        addToBeginning: false,
      );
      if (context.mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(
            content: Text('Route an EVE übertragen'),
            behavior: SnackBarBehavior.floating,
          ),
        );
      }
    } catch (e) {
      if (context.mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
            content: Text('Fehler beim Setzen der Route: $e'),
            backgroundColor: Theme.of(context).colorScheme.error,
            behavior: SnackBarBehavior.floating,
          ),
        );
      }
    }
  }
}

// ---------------------------------------------------------------------------
// Small helpers
// ---------------------------------------------------------------------------

class _SliderRow extends StatelessWidget {
  const _SliderRow({required this.label, required this.valueLabel});
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
    this.enabled = true,
    this.activeColor,
  });

  final String label;
  final bool value;
  final ValueChanged<bool> onChanged;
  final bool enabled;
  final Color? activeColor;

  @override
  Widget build(BuildContext context) {
    return CheckboxListTile(
      value: value,
      onChanged: enabled ? (v) => onChanged(v ?? false) : null,
      title: Text(label, style: const TextStyle(fontSize: 13)),
      controlAffinity: ListTileControlAffinity.leading,
      dense: true,
      contentPadding: EdgeInsets.zero,
      activeColor: activeColor ?? Theme.of(context).colorScheme.primary,
    );
  }
}

class _SummaryTile extends StatelessWidget {
  const _SummaryTile({
    required this.label,
    required this.value,
    this.accent,
  });

  final String label;
  final String value;
  final Color? accent;

  @override
  Widget build(BuildContext context) {
    final color = accent ?? Theme.of(context).colorScheme.onSurface;
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
      decoration: BoxDecoration(
        color: Theme.of(context).colorScheme.surfaceContainerHighest,
        borderRadius: BorderRadius.circular(8),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        mainAxisSize: MainAxisSize.min,
        children: [
          Text(
            label,
            style: Theme.of(context).textTheme.bodySmall,
          ),
          const SizedBox(height: 4),
          Text(
            value,
            style: Theme.of(context).textTheme.titleMedium?.copyWith(
                  fontWeight: FontWeight.w700,
                  color: color,
                ),
          ),
        ],
      ),
    );
  }
}
