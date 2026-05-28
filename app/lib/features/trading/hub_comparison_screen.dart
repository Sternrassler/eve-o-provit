/// Adaptive Multi-Hub Comparison screen.
///
/// The user searches for an EVE item, picks one, and sees that item's
/// station-trading profitability compared across the major trade hubs
/// (Jita, Amarr, Dodixie, Rens, Hek): buy/sell price, spread%, skill-adjusted
/// net margin, volume and a competition indicator, with the recommended hub
/// highlighted.
///
/// Layout (via [isTwoPane] / [kTwoPaneBreakpoint] = 840 dp):
///   • <840 dp  → single-pane: search form above a scrollable comparison table.
///   • ≥840 dp  → two-pane: search form on the left, table on the right.
library;

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../api/hub_comparison_models.dart';
import '../../core/breakpoint.dart';
import 'hub_comparison_providers.dart';
import 'providers.dart' show tradingApiProvider;

// ---------------------------------------------------------------------------
// Public entry point
// ---------------------------------------------------------------------------

/// The hub-comparison screen.
class HubComparisonScreen extends ConsumerWidget {
  const HubComparisonScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Hub-Vergleich'),
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
                  width: 320,
                  child: SingleChildScrollView(
                    padding: EdgeInsets.all(16),
                    child: _SearchForm(),
                  ),
                ),
                VerticalDivider(width: 1),
                Expanded(child: _ComparisonTable()),
              ],
            );
          }
          return Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: const [
              Padding(
                padding: EdgeInsets.all(16),
                child: _SearchForm(),
              ),
              Divider(height: 1),
              Expanded(child: _ComparisonTable()),
            ],
          );
        },
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Search form — item lookup + pick + compare
// ---------------------------------------------------------------------------

class _SearchForm extends ConsumerStatefulWidget {
  const _SearchForm();

  @override
  ConsumerState<_SearchForm> createState() => _SearchFormState();
}

class _SearchFormState extends ConsumerState<_SearchForm> {
  final _controller = TextEditingController();
  List<ItemSearchResult> _results = const [];
  bool _searching = false;
  String? _error;

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  Future<void> _runSearch() async {
    final q = _controller.text.trim();
    if (q.length < 3) {
      setState(() {
        _error = 'Mindestens 3 Zeichen eingeben.';
        _results = const [];
      });
      return;
    }
    setState(() {
      _searching = true;
      _error = null;
    });
    try {
      final api = ref.read(tradingApiProvider);
      final resp = await api.searchItems(q);
      if (!mounted) return;
      setState(() => _results = resp.items);
    } catch (e) {
      if (!mounted) return;
      setState(() => _error = 'Suche fehlgeschlagen: $e');
    } finally {
      if (mounted) setState(() => _searching = false);
    }
  }

  Future<void> _pick(ItemSearchResult item) async {
    ref.read(selectedSearchItemProvider.notifier).select(item);
    setState(() => _results = const []);
    await ref.read(hubComparisonProvider.notifier).search(item.typeId);
  }

  @override
  Widget build(BuildContext context) {
    final selected = ref.watch(selectedSearchItemProvider);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      mainAxisSize: MainAxisSize.min,
      children: [
        TextField(
          controller: _controller,
          textInputAction: TextInputAction.search,
          onSubmitted: (_) => _runSearch(),
          decoration: InputDecoration(
            labelText: 'Item suchen',
            hintText: 'min. 3 Zeichen',
            border: const OutlineInputBorder(),
            isDense: true,
            suffixIcon: _searching
                ? const Padding(
                    padding: EdgeInsets.all(12),
                    child: SizedBox(
                      width: 16,
                      height: 16,
                      child: CircularProgressIndicator(strokeWidth: 2),
                    ),
                  )
                : IconButton(
                    icon: const Icon(Icons.search_rounded),
                    onPressed: _searching ? null : _runSearch,
                    tooltip: 'Suchen',
                  ),
          ),
        ),
        if (_error != null) ...[
          const SizedBox(height: 8),
          Text(
            _error!,
            style: TextStyle(
              color: Theme.of(context).colorScheme.error,
            ),
          ),
        ],
        if (_results.isNotEmpty) ...[
          const SizedBox(height: 8),
          ConstrainedBox(
            constraints: const BoxConstraints(maxHeight: 280),
            child: Material(
              color: Theme.of(context).colorScheme.surface,
              shape: RoundedRectangleBorder(
                side: BorderSide(
                  color: Theme.of(context).dividerColor,
                ),
                borderRadius: BorderRadius.circular(8),
              ),
              child: ListView.separated(
                shrinkWrap: true,
                itemCount: _results.length,
                separatorBuilder: (context, index) => const Divider(height: 1),
                itemBuilder: (context, i) {
                  final item = _results[i];
                  return ListTile(
                    dense: true,
                    title: Text(item.name),
                    subtitle: item.groupName.isEmpty
                        ? null
                        : Text(item.groupName),
                    onTap: () => _pick(item),
                  );
                },
              ),
            ),
          ),
        ],
        if (selected != null) ...[
          const SizedBox(height: 16),
          Text(
            'Ausgewählt: ${selected.name}',
            style: Theme.of(context).textTheme.titleSmall,
          ),
        ],
      ],
    );
  }
}

// ---------------------------------------------------------------------------
// Comparison table — async-aware
// ---------------------------------------------------------------------------

class _ComparisonTable extends ConsumerWidget {
  const _ComparisonTable();

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final async = ref.watch(hubComparisonProvider);

    return async.when(
      loading: () => const Center(child: CircularProgressIndicator()),
      error: (err, _) => Center(
        child: Padding(
          padding: const EdgeInsets.all(24),
          child: Text(
            'Vergleich fehlgeschlagen.\n$err',
            textAlign: TextAlign.center,
            style: TextStyle(color: Theme.of(context).colorScheme.error),
          ),
        ),
      ),
      data: (resp) {
        if (resp == null) {
          return const _EmptyHint();
        }
        return _HubDataTable(response: resp);
      },
    );
  }
}

class _EmptyHint extends StatelessWidget {
  const _EmptyHint();

  @override
  Widget build(BuildContext context) {
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(32),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(
              Icons.compare_arrows_rounded,
              size: 56,
              color: Theme.of(context).colorScheme.onSurface.withAlpha(80),
            ),
            const SizedBox(height: 16),
            Text(
              'Item suchen und auswählen',
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

// ---------------------------------------------------------------------------
// The actual data table
// ---------------------------------------------------------------------------

class _HubDataTable extends StatelessWidget {
  const _HubDataTable({required this.response});

  final HubComparisonResponse response;

  @override
  Widget build(BuildContext context) {
    final accent = Theme.of(context).colorScheme.primary;
    return SingleChildScrollView(
      padding: const EdgeInsets.all(16),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Text(
            response.itemName,
            style: Theme.of(context).textTheme.titleLarge,
          ),
          const SizedBox(height: 4),
          Text(
            response.skillsApplied.applied
                ? 'Skill-bereinigt (Accounting ${response.skillsApplied.accounting}, '
                    'Broker Relations ${response.skillsApplied.brokerRelations})'
                : 'Ohne Skill-Bereinigung',
            style: Theme.of(context).textTheme.bodySmall,
          ),
          const SizedBox(height: 16),
          if (response.hubs.any((h) => h.hasData) &&
              response.bestHubRegionId == 0)
            Container(
              key: const Key('no-profit-hint'),
              margin: const EdgeInsets.only(bottom: 12),
              padding: const EdgeInsets.all(12),
              decoration: BoxDecoration(
                color: Colors.amber.withAlpha(30),
                borderRadius: BorderRadius.circular(6),
                border: Border.all(color: Colors.amber.withAlpha(120)),
              ),
              child: const Text(
                'Kein profitabler Hub für dieses Item — nach Steuer und Broker-Fee '
                'ist die Station-Trading-Marge an allen Hubs negativ. Der beste '
                '(am wenigsten verlustreiche) Hub steht oben.',
              ),
            ),
          SingleChildScrollView(
            scrollDirection: Axis.horizontal,
            child: DataTable(
              columns: const [
                DataColumn(label: Text('Hub')),
                DataColumn(label: Text('Buy'), numeric: true),
                DataColumn(label: Text('Sell'), numeric: true),
                DataColumn(label: Text('Spread%'), numeric: true),
                DataColumn(label: Text('Net Margin%'), numeric: true),
                DataColumn(label: Text('Volume'), numeric: true),
                DataColumn(label: Text('Competition')),
              ],
              rows: [
                for (final hub in response.hubs)
                  _hubRow(context, hub, accent),
              ],
            ),
          ),
        ],
      ),
    );
  }

  DataRow _hubRow(BuildContext context, HubRow hub, Color accent) {
    final isBest =
        hub.hasData && hub.regionId == response.bestHubRegionId;
    final highlight = accent.withAlpha(30);

    if (!hub.hasData) {
      return DataRow(
        color: WidgetStateProperty.all(
          Theme.of(context).colorScheme.onSurface.withAlpha(8),
        ),
        cells: [
          DataCell(_hubName(context, hub, isBest: false)),
          const DataCell(Text('—')),
          const DataCell(Text('—')),
          const DataCell(Text('—')),
          const DataCell(Text('keine Daten')),
          const DataCell(Text('—')),
          const DataCell(Text('—')),
        ],
      );
    }

    return DataRow(
      color: isBest ? WidgetStateProperty.all(highlight) : null,
      cells: [
        DataCell(_hubName(context, hub, isBest: isBest)),
        DataCell(Text(_fmtIsk(hub.buyPrice))),
        DataCell(Text(_fmtIsk(hub.sellPrice))),
        DataCell(Text('${hub.spreadPercent.toStringAsFixed(1)}%')),
        DataCell(Text('${hub.netMarginPercent.toStringAsFixed(1)}%')),
        DataCell(Text(_fmtVolume(hub.dailyVolume))),
        DataCell(_competition(context, hub.competition)),
      ],
    );
  }

  Widget _hubName(BuildContext context, HubRow hub, {required bool isBest}) {
    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        if (isBest) ...[
          Icon(
            Icons.star_rounded,
            size: 16,
            color: Theme.of(context).colorScheme.primary,
          ),
          const SizedBox(width: 4),
        ],
        Text(
          hub.hubName,
          style: isBest
              ? const TextStyle(fontWeight: FontWeight.w700)
              : null,
        ),
      ],
    );
  }

  Widget _competition(BuildContext context, CompetitionInfo? comp) {
    if (comp == null) return const Text('—');
    final isLive = comp.source == 'live';
    final color = isLive
        ? const Color(0xFF66BB6A)
        : Theme.of(context).colorScheme.onSurface.withAlpha(120);
    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        Icon(Icons.circle, size: 8, color: color),
        const SizedBox(width: 6),
        Text(
          '${comp.changesPerHour.toStringAsFixed(1)}/h',
        ),
        const SizedBox(width: 4),
        Text(
          isLive ? 'live' : 'baseline',
          style: Theme.of(context).textTheme.bodySmall?.copyWith(color: color),
        ),
      ],
    );
  }

  // ── Formatting helpers ─────────────────────────────────────────────────────

  static String _fmtIsk(double v) {
    if (v >= 1e9) return '${(v / 1e9).toStringAsFixed(2)}B';
    if (v >= 1e6) return '${(v / 1e6).toStringAsFixed(2)}M';
    if (v >= 1e3) return '${(v / 1e3).toStringAsFixed(1)}K';
    return v.toStringAsFixed(2);
  }

  static String _fmtVolume(double v) {
    if (v >= 1e9) return '${(v / 1e9).toStringAsFixed(1)}B';
    if (v >= 1e6) return '${(v / 1e6).toStringAsFixed(1)}M';
    if (v >= 1e3) return '${(v / 1e3).toStringAsFixed(1)}K';
    return v.toStringAsFixed(0);
  }
}
