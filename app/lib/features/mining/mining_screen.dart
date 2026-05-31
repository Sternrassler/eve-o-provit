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
  String _secBand = 'high';

  Future<void> _calculate() async {
    final request = OreRankingRequest(
      regionId: 0,
      secBand: _secBand,
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

        // ── Sec-band selector ──────────────────────────────────────────────
        const Text(
          'Sicherheitszone',
          style: TextStyle(fontSize: 13, fontWeight: FontWeight.w500),
        ),
        const SizedBox(height: 6),
        RadioGroup<String>(
          groupValue: _secBand,
          onChanged: busy ? (_) {} : (v) => setState(() => _secBand = v ?? 'high'),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              _SecBandTile(
                label: 'High-Sec (≥ 0.5)',
                value: 'high',
                enabled: !busy,
              ),
              _SecBandTile(
                label: 'Low-Sec (< 0.5)',
                value: 'low',
                enabled: !busy,
                activeColor: const Color(0xFFFF9800),
              ),
              _SecBandTile(
                label: 'Null-Sec (≤ 0.0)',
                value: 'null',
                enabled: !busy,
                activeColor: const Color(0xFFF44336),
              ),
            ],
          ),
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
        if (result.isEmpty) {
          return const _EmptyResult();
        }
        return Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Expanded(child: OreRankingTable(result: result)),
            const Padding(
              padding: EdgeInsets.only(top: 8),
              child: Text(
                'Hinweis: ISK/h ist eine skills-basierte Untergrenze — Schiffs-Rollenboni '
                '(Mining Barge/Exhumer) und Erz-Crystals fehlen noch. '
                'Roh-vs-Refine-Verdict und Ranking sind unberührt.',
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
              'Sicherheitszone wählen und Erze berechnen',
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
            'Region ${result.regionId} · ${_secBandLabel(result.secBand)}',
            style: Theme.of(context).textTheme.bodySmall,
          ),
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
            _verdictChip(context, verdictColor, verdictLabel),
          ],
        ),
        subtitle: Padding(
          padding: const EdgeInsets.only(top: 4),
          child: Text(
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
    return _detailLine(context, 'Roh verkaufen bei', _formatSell(row.rawSell));
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

class _SecBandTile extends StatelessWidget {
  const _SecBandTile({
    required this.label,
    required this.value,
    this.enabled = true,
    this.activeColor,
  });

  final String label;
  final String value;
  final bool enabled;
  final Color? activeColor;

  @override
  Widget build(BuildContext context) {
    final color = activeColor ?? Theme.of(context).colorScheme.primary;
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 2),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Radio<String>(
            value: value,
            activeColor: color,
            materialTapTargetSize: MaterialTapTargetSize.shrinkWrap,
            visualDensity: VisualDensity.compact,
            enabled: enabled,
          ),
          const SizedBox(width: 4),
          Flexible(
            child: Text(label, style: const TextStyle(fontSize: 13)),
          ),
        ],
      ),
    );
  }
}

String _secBandLabel(String band) {
  switch (band) {
    case 'low':
      return 'Low-Sec';
    case 'null':
      return 'Null-Sec';
    default:
      return 'High-Sec';
  }
}

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
