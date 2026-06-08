/// Mining Ore-Ranking screen.
///
/// Shows ranked ore types for the character's current region + a selected
/// sec band (High/Low/Null). The CurrentShipCard at the top reflects the
/// active mining ship. Pressing "Erze berechnen" posts
/// POST /api/v1/mining/ore-ranking with region_id=0 (current region).
///
/// Layout (via [isTwoPane] / [kTwoPaneBreakpoint] = 840 dp):
///   • <840 dp  → single-pane: controls above the result table.
///   • ≥840 dp  → two-pane: controls on the left, table on the right.
library;

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../api/mining_models.dart';
import '../../core/breakpoint.dart';
import '../../core/format.dart';
import '../trading/current_ship_card.dart';
import 'mining_providers.dart';
import 'sold_loads_tally.dart';

// ---------------------------------------------------------------------------
// Public entry point
// ---------------------------------------------------------------------------

/// The Mining Ore-Ranking screen.
class MiningScreen extends ConsumerWidget {
  const MiningScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Mining'),
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
                  width: 300,
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
// Input form — ship card + sec-band selector + calculate button
// ---------------------------------------------------------------------------

class _InputForm extends ConsumerStatefulWidget {
  const _InputForm();

  @override
  ConsumerState<_InputForm> createState() => _InputFormState();
}

class _InputFormState extends ConsumerState<_InputForm> {
  bool _allowLowSec = false;

  Future<void> _calculate() async {
    final request = OreRankingRequest(
      regionId: 0,
      allowLowSec: _allowLowSec,
    );
    await ref.read(oreRankingProvider.notifier).run(request);
  }

  @override
  Widget build(BuildContext context) {
    final busy = ref.watch(oreRankingProvider).isLoading;

    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      mainAxisSize: MainAxisSize.min,
      children: [
        // ── Ship (read-only current ship + refresh) ───────────────────────
        const CurrentShipCard(),
        const SizedBox(height: 16),

        // ── Low-Sec toggle ────────────────────────────────────────────────
        SwitchListTile(
          value: _allowLowSec,
          onChanged: busy ? null : (v) => setState(() => _allowLowSec = v),
          title: const Text(
            'Auch Low-Sec abbauen/verkaufen',
            style: TextStyle(fontSize: 13, fontWeight: FontWeight.w500),
          ),
          subtitle: Text(
            _allowLowSec ? 'An: High-Sec + Low-Sec' : 'Aus: nur High-Sec',
            style: const TextStyle(fontSize: 12),
          ),
          contentPadding: EdgeInsets.zero,
          dense: true,
        ),

        const SizedBox(height: 16),

        // ── Calculate button ───────────────────────────────────────────────
        FilledButton.icon(
          onPressed: busy ? null : _calculate,
          icon: busy
              ? const SizedBox(
                  width: 16,
                  height: 16,
                  child: CircularProgressIndicator(strokeWidth: 2),
                )
              : const Icon(Icons.diamond_rounded),
          label: const Text('Erze berechnen'),
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
    final async = ref.watch(oreRankingProvider);

    return async.when(
      loading: () => const Center(child: CircularProgressIndicator()),
      error: (err, _) => Center(
        child: Padding(
          padding: const EdgeInsets.all(24),
          child: Text(
            'Berechnung fehlgeschlagen.\n$err',
            textAlign: TextAlign.center,
            style: TextStyle(color: Theme.of(context).colorScheme.error),
          ),
        ),
      ),
      data: (result) {
        if (result == null) {
          return const _IdleHint();
        }
        if (result.noMiningSetup) {
          return const _NoMiningSetup();
        }
        if (result.notAvailableReason != null && result.isEmpty) {
          return _NotAvailable(reason: result.notAvailableReason!);
        }
        if (result.isEmpty) {
          return const _EmptyResult();
        }
        final degradedReason = result.degradedReason;
        return Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            if (degradedReason != null && degradedReason.isNotEmpty)
              _DegradedBanner(reason: degradedReason),
            Expanded(child: OreRankingTable(result: result)),
            const Padding(
              padding: EdgeInsets.only(top: 8),
              child: Text(
                'Hinweis: ISK/h berücksichtigt Schiffs-/Skill-Boni und (best-case) '
                'Erz-Crystals. Zeilen mit ≈ sind Schätzwerte (Schiffs-Bonus oder '
                'Crystal nicht auflösbar). Roh-vs-Refine-Verdict unberührt.'
                ' ISK/h ist effektiv inkl. Füllen + Flug (ein Zyklus ab deinem System).',
                style: TextStyle(fontSize: 11, color: Colors.grey),
              ),
            ),
          ],
        );
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
              Icons.diamond_rounded,
              size: 56,
              color: Theme.of(context).colorScheme.onSurface.withAlpha(80),
            ),
            const SizedBox(height: 16),
            Text(
              'Zone konfigurieren und Erze berechnen',
              textAlign: TextAlign.center,
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

class _NoMiningSetup extends StatelessWidget {
  const _NoMiningSetup();

  @override
  Widget build(BuildContext context) {
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(32),
        child: Column(
          key: const Key('mining-no-setup'),
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(
              Icons.warning_amber_rounded,
              size: 56,
              color: Theme.of(context).colorScheme.error.withAlpha(200),
            ),
            const SizedBox(height: 16),
            Text(
              'Kein Mining-Setup',
              textAlign: TextAlign.center,
              style: Theme.of(context).textTheme.titleMedium?.copyWith(
                    color: Theme.of(context).colorScheme.error,
                    fontWeight: FontWeight.w700,
                  ),
            ),
            const SizedBox(height: 8),
            Text(
              'Das aktive Schiff hat keine Mining-Laser ausgerüstet.',
              textAlign: TextAlign.center,
              style: Theme.of(context).textTheme.bodyMedium?.copyWith(
                    color:
                        Theme.of(context).colorScheme.onSurface.withAlpha(153),
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
          key: const Key('mining-empty-state'),
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(
              Icons.sentiment_dissatisfied_rounded,
              size: 56,
              color: Theme.of(context).colorScheme.onSurface.withAlpha(80),
            ),
            const SizedBox(height: 16),
            Text(
              'Keine abbaubaren Erze für diese Zone gefunden',
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

class _NotAvailable extends StatelessWidget {
  const _NotAvailable({required this.reason});

  final String reason;

  @override
  Widget build(BuildContext context) {
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(32),
        child: Column(
          key: const Key('mining-not-available'),
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(
              Icons.info_outline_rounded,
              size: 56,
              color: Theme.of(context).colorScheme.onSurface.withAlpha(80),
            ),
            const SizedBox(height: 16),
            Text(
              reason,
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

/// Visible warning shown above the result list when the backend computed the
/// ranking with degraded inputs (skills and/or standings could not be loaded).
/// Unlike [_NotAvailable], this renders even when [rows] are present, so the
/// numbers below are never mistaken for reliable values.
class _DegradedBanner extends StatelessWidget {
  const _DegradedBanner({required this.reason});

  final String reason;

  // Amber/orange — consistent with the ≈ estimate marker used elsewhere.
  static const Color _warnColor = Color(0xFFFFB300);

  @override
  Widget build(BuildContext context) {
    return Padding(
      key: const Key('mining-degraded-banner'),
      padding: const EdgeInsets.fromLTRB(16, 16, 16, 0),
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
        decoration: BoxDecoration(
          color: _warnColor.withAlpha(30),
          borderRadius: BorderRadius.circular(8),
          border: Border.all(color: _warnColor.withAlpha(150)),
        ),
        child: Row(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const Icon(
              Icons.warning_amber_rounded,
              size: 20,
              color: _warnColor,
            ),
            const SizedBox(width: 8),
            Expanded(
              child: Text(
                reason,
                style: Theme.of(context).textTheme.bodySmall?.copyWith(
                      color: Theme.of(context).colorScheme.onSurface,
                      fontWeight: FontWeight.w500,
                    ),
              ),
            ),
          ],
        ),
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Ore ranking table
// ---------------------------------------------------------------------------

/// Scrollable list of ranked ore rows. Each ore is an [ExpansionTile]: the
/// header shows the summary (name, ISK/h, verdict, tax, Δ); expanding reveals
/// where to reprocess (best NPC station — system) and where to sell — a
/// per-mineral breakdown for the refine path, or the raw ore location for the
/// raw path. Extracted as a public widget so widget tests can import it.
class OreRankingTable extends StatelessWidget {
  const OreRankingTable({super.key, required this.result});

  final OreRankingResponse result;

  @override
  Widget build(BuildContext context) {
    return SingleChildScrollView(
      padding: const EdgeInsets.all(16),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Text(
            'Erz-Ranking',
            style: Theme.of(context).textTheme.titleLarge,
          ),
          const SizedBox(height: 4),
          Text(
            'Region ${result.regionId}'
            '${result.quarter.isNotEmpty ? ' · ${result.quarter}' : ''}'
            ' · ${result.systemSecurity.toStringAsFixed(1)}',
            style: Theme.of(context).textTheme.bodySmall,
          ),
          if (result.originSystemName != null) ...[
            const SizedBox(height: 4),
            Text(
              result.originStationName != null
                  ? 'Standort: ${result.originStationName} (${result.originSystemName})'
                  : 'Standort: ${result.originSystemName}',
              style: Theme.of(context).textTheme.bodySmall,
            ),
          ],
          const SizedBox(height: 16),
          Column(
            key: const Key('mining-ore-table'),
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              for (final row in result.rows) _OreRankTile(row: row),
            ],
          ),
        ],
      ),
    );
  }
}

/// One expandable ore card: collapsed header + reprocess/sell detail panel.
class _OreRankTile extends StatelessWidget {
  const _OreRankTile({required this.row});

  final OreRankRow row;

  @override
  Widget build(BuildContext context) {
    final isRefine = row.best == 'refine';
    final verdictColor = isRefine
        ? const Color(0xFF66BB6A) // green — refining wins
        : Theme.of(context).colorScheme.primary;
    final verdictLabel = isRefine ? 'Raffinieren' : 'Roh verkaufen';

    return Card(
      key: ValueKey('mining-ore-row-${row.oreTypeId}'),
      margin: const EdgeInsets.symmetric(vertical: 4),
      child: ExpansionTile(
        key: ValueKey('mining-ore-expand-${row.oreTypeId}'),
        tilePadding: const EdgeInsets.symmetric(horizontal: 16, vertical: 4),
        childrenPadding: const EdgeInsets.fromLTRB(16, 0, 16, 16),
        expandedCrossAxisAlignment: CrossAxisAlignment.start,
        title: Row(
          children: [
            Expanded(
              child: Text(
                row.oreName,
                style: const TextStyle(fontWeight: FontWeight.w600),
              ),
            ),
            if (row.isEstimate)
              Padding(
                key: ValueKey('mining-estimate-${row.oreTypeId}'),
                padding: const EdgeInsets.only(right: 6),
                child: Tooltip(
                  message: row.estimateReason ?? 'Schätzwert',
                  child: const Text(
                    '≈',
                    style: TextStyle(
                      color: Color(0xFFFFB300),
                      fontWeight: FontWeight.w700,
                    ),
                  ),
                ),
              ),
            _verdictChip(context, verdictColor, verdictLabel),
          ],
        ),
        subtitle: Padding(
          padding: const EdgeInsets.only(top: 4),
          child: Text(
            '${row.effectiveIskPerHour > 0 ? 'eff ${fmtIsk(row.effectiveIskPerHour)} · ' : ''}'
            'm³/h ${fmtVolume(row.miningM3PerHour)} · '
            'roh ${fmtIsk(row.rawIskPerHour)} · '
            'refine ${fmtIsk(row.refineIskPerHour)} · '
            'Steuer ${(row.bestStationTax * 100).toStringAsFixed(1)}% · '
            'Δ ${fmtIsk(row.deltaIskPerHour)}',
            style: Theme.of(context).textTheme.bodySmall,
          ),
        ),
        children: [isRefine ? _refineDetail(context) : _rawDetail(context)],
      ),
    );
  }

  Widget _verdictChip(BuildContext context, Color color, String label) {
    return Container(
      key: Key('mining-verdict-${row.oreTypeId}'),
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
      decoration: BoxDecoration(
        color: color.withAlpha(40),
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: color.withAlpha(150)),
      ),
      child: Text(
        label,
        style: TextStyle(
          color: color,
          fontWeight: FontWeight.w600,
          fontSize: 12,
        ),
      ),
    );
  }

  Widget _refineDetail(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        if (row.cycleMinutes > 0)
          Padding(
            padding: const EdgeInsets.only(bottom: 6),
            child: Text(
              'Zyklus ${row.cycleMinutes.toStringAsFixed(1)} min · ${row.routeJumps} Jumps'
              '${row.sellSystemName != null ? ' · Verkauf in ${row.sellSystemName}' : ''}',
              style: Theme.of(context).textTheme.bodySmall,
            ),
          ),
        if (row.marketLoads != null && row.marketLoads! > 0)
          Padding(
            padding: const EdgeInsets.only(bottom: 6),
            child: row.marketLoads! >= 1
                ? Text(
                    'Markt: ~${row.marketLoads!.toStringAsFixed(1)} volle Ladungen',
                    style: Theme.of(context).textTheme.bodySmall,
                  )
                : Text(
                    '⚠ Bestpreis nur ~${row.marketLoads!.toStringAsFixed(2)} Ladungen — ISK/h optimistisch',
                    style: const TextStyle(
                      color: Color(0xFFFFB300),
                      fontWeight: FontWeight.w500,
                      fontSize: 12,
                    ),
                  ),
          ),
        if (row.marketLoads != null && row.marketLoads! >= 1)
          MiningLoadsTally(
            oreTypeId: row.oreTypeId,
            total: row.marketLoads!.floor(),
          ),
        _detailLine(
          context,
          'Aufbereiten bei',
          _formatStation(row.bestStationName, row.bestStationSystem),
        ),
        const SizedBox(height: 8),
        Text(
          'Raffinate verkaufen:',
          style: Theme.of(context).textTheme.bodySmall,
        ),
        const SizedBox(height: 4),
        for (final m in row.materials) _materialRow(context, m),
      ],
    );
  }

  Widget _materialRow(BuildContext context, RefineMaterial m) {
    return Padding(
      key: ValueKey('mining-material-${row.oreTypeId}-${m.materialTypeId}'),
      padding: const EdgeInsets.symmetric(vertical: 3),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Expanded(
            flex: 3,
            child: Text(
              m.materialName.isEmpty ? 'Typ ${m.materialTypeId}' : m.materialName,
            ),
          ),
          Expanded(
            flex: 2,
            child: Text(
              fmtUnits(m.effectiveQty.toDouble()),
              textAlign: TextAlign.right,
            ),
          ),
          Expanded(
            flex: 2,
            child: Text(fmtIsk(m.buyPrice), textAlign: TextAlign.right),
          ),
          Expanded(
            flex: 4,
            child: Padding(
              padding: const EdgeInsets.only(left: 8),
              child: Text(
                _formatSell(m.sell),
                style: Theme.of(context).textTheme.bodySmall,
              ),
            ),
          ),
        ],
      ),
    );
  }

  Widget _rawDetail(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        if (row.cycleMinutes > 0)
          Padding(
            padding: const EdgeInsets.only(bottom: 6),
            child: Text(
              'Zyklus ${row.cycleMinutes.toStringAsFixed(1)} min · ${row.routeJumps} Jumps'
              '${row.sellSystemName != null ? ' · Verkauf in ${row.sellSystemName}' : ''}',
              style: Theme.of(context).textTheme.bodySmall,
            ),
          ),
        if (row.marketLoads != null && row.marketLoads! > 0)
          Padding(
            padding: const EdgeInsets.only(bottom: 6),
            child: row.marketLoads! >= 1
                ? Text(
                    'Markt: ~${row.marketLoads!.toStringAsFixed(1)} volle Ladungen',
                    style: Theme.of(context).textTheme.bodySmall,
                  )
                : Text(
                    '⚠ Bestpreis nur ~${row.marketLoads!.toStringAsFixed(2)} Ladungen — ISK/h optimistisch',
                    style: const TextStyle(
                      color: Color(0xFFFFB300),
                      fontWeight: FontWeight.w500,
                      fontSize: 12,
                    ),
                  ),
          ),
        if (row.marketLoads != null && row.marketLoads! >= 1)
          MiningLoadsTally(
            oreTypeId: row.oreTypeId,
            total: row.marketLoads!.floor(),
          ),
        _detailLine(context, 'Roh verkaufen bei', _formatSell(row.rawSell)),
      ],
    );
  }

  Widget _detailLine(BuildContext context, String label, String value) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 2),
      child: Text.rich(
        TextSpan(
          children: [
            TextSpan(
              text: '$label: ',
              style: TextStyle(
                color: Theme.of(context).colorScheme.onSurface.withAlpha(160),
              ),
            ),
            TextSpan(
              text: value,
              style: const TextStyle(fontWeight: FontWeight.w500),
            ),
          ],
        ),
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Small helpers
// ---------------------------------------------------------------------------


/// "Station — System", dropping empty parts; '—' when both are absent.
String _formatStation(String? name, String? system) {
  final parts = [name, system]
      .where((s) => s != null && s.isNotEmpty)
      .cast<String>()
      .toList();
  return parts.isEmpty ? '—' : parts.join(' — ');
}

/// Human label for a sell location. Citadels (SDE can't name them) render as
/// "Player-Struktur"; otherwise "Station — System".
String _formatSell(SellLocation? loc) {
  if (loc == null) return '—';
  if (loc.isStructure) return 'Player-Struktur';
  final parts = [loc.stationName, loc.systemName]
      .where((s) => s != null && s.isNotEmpty)
      .cast<String>()
      .toList();
  return parts.isEmpty ? 'unbekannt' : parts.join(' — ');
}
