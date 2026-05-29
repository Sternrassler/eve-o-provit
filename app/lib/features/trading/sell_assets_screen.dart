/// Adaptive Sell-from-Assets screen (GitHub issue #107).
///
/// The user picks an item they already own (from their character's assets) and
/// gets a ranked answer to "where do I sell this for the most net ISK, and how
/// do I get there?" — a taker model (instant sell into a buy order, net of
/// sales tax) compared across the 5 major hubs + every station in the item's
/// current region, each with a route (jumps, travel time, security risk). The
/// user can push the chosen sell location to the in-game autopilot.
///
/// Layout (via [isTwoPane] / [kTwoPaneBreakpoint] = 840 dp):
///   • <840 dp  → single-pane: asset picker + form above a scrollable option
///                list; tapping an option pushes a detail page.
///   • ≥840 dp  → two-pane: picker + form on the left, option list on the
///                right; tapping an option shows its detail in a dialog.
library;

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../api/asset_models.dart';
import '../../core/breakpoint.dart';
import 'providers.dart';
import 'sell_assets_providers.dart';

// ---------------------------------------------------------------------------
// Security-risk → colour / label mapping (safe=green, caution=amber, danger=red)
// ---------------------------------------------------------------------------

const Color _kSafeColor = Color(0xFF66BB6A);
const Color _kCautionColor = Color(0xFFFF9800);
const Color _kDangerColor = Color(0xFFF44336);

/// Maps a backend `security_risk` value to its chip colour.
Color sellSecurityRiskColor(String risk) {
  switch (risk) {
    case 'danger':
      return _kDangerColor;
    case 'caution':
      return _kCautionColor;
    case 'safe':
    default:
      return _kSafeColor;
  }
}

/// Maps a backend `security_risk` value to a German chip label.
String sellSecurityRiskLabel(String risk) {
  switch (risk) {
    case 'danger':
      return 'Gefahr';
    case 'caution':
      return 'Vorsicht';
    case 'safe':
    default:
      return 'Sicher';
  }
}

// ---------------------------------------------------------------------------
// Public entry point
// ---------------------------------------------------------------------------

/// The Sell-from-Assets screen.
class SellAssetsScreen extends ConsumerWidget {
  const SellAssetsScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Aus Besitz verkaufen'),
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
                  width: 360,
                  child: _PickerPane(),
                ),
                VerticalDivider(width: 1),
                Expanded(child: _ResultPane(twoPane: true)),
              ],
            );
          }
          return Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: const [
              Flexible(child: _PickerPane()),
              Divider(height: 1),
              Expanded(child: _ResultPane(twoPane: false)),
            ],
          );
        },
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Picker pane — asset search + list, then quantity + avoid-low-sec + button
// ---------------------------------------------------------------------------

class _PickerPane extends ConsumerStatefulWidget {
  const _PickerPane();

  @override
  ConsumerState<_PickerPane> createState() => _PickerPaneState();
}

class _PickerPaneState extends ConsumerState<_PickerPane> {
  final _searchController = TextEditingController();
  final _quantityController = TextEditingController();
  String _query = '';
  AssetItem? _selected;
  bool _avoidLowSec = true;
  String? _error;

  @override
  void dispose() {
    _searchController.dispose();
    _quantityController.dispose();
    super.dispose();
  }

  void _select(AssetItem asset) {
    setState(() {
      _selected = asset;
      _quantityController.text = asset.quantity.toString();
      _error = null;
    });
    // A new selection invalidates any previous search result.
    ref.read(sellOptionsProvider.notifier).reset();
  }

  Future<void> _search() async {
    final asset = _selected;
    if (asset == null) {
      setState(() => _error = 'Bitte ein Item auswählen.');
      return;
    }
    final qty = int.tryParse(
        _quantityController.text.replaceAll(RegExp(r'[^0-9]'), ''));
    if (qty == null || qty <= 0) {
      setState(() => _error = 'Bitte eine gültige Menge eingeben.');
      return;
    }
    setState(() => _error = null);

    final request = SellOptionsRequest(
      typeId: asset.typeId,
      locationId: asset.locationId,
      quantity: qty,
      avoidLowSec: _avoidLowSec,
    );
    await ref.read(sellOptionsProvider.notifier).find(request);
  }

  @override
  Widget build(BuildContext context) {
    final busy = ref.watch(sellOptionsProvider).isLoading;
    final assetsAsync = ref.watch(characterAssetsProvider);

    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        // ── Search field ───────────────────────────────────────────────────
        Padding(
          padding: const EdgeInsets.fromLTRB(16, 16, 16, 8),
          child: TextField(
            key: const Key('sell-asset-search'),
            controller: _searchController,
            enabled: !busy,
            onChanged: (v) => setState(() => _query = v.trim().toLowerCase()),
            decoration: const InputDecoration(
              labelText: 'Item suchen',
              prefixIcon: Icon(Icons.search_rounded),
              border: OutlineInputBorder(),
              isDense: true,
              contentPadding:
                  EdgeInsets.symmetric(horizontal: 12, vertical: 8),
            ),
          ),
        ),

        // ── Asset list ─────────────────────────────────────────────────────
        Expanded(
          child: assetsAsync.when(
            loading: () =>
                const Center(child: CircularProgressIndicator()),
            error: (err, _) => Center(
              child: Padding(
                padding: const EdgeInsets.all(24),
                child: Text(
                  'Besitz konnte nicht geladen werden.\n$err',
                  textAlign: TextAlign.center,
                  style: TextStyle(
                      color: Theme.of(context).colorScheme.error),
                ),
              ),
            ),
            data: (resp) {
              final filtered = resp.assets
                  .where((a) =>
                      _query.isEmpty || a.name.toLowerCase().contains(_query))
                  .toList();
              if (resp.isEmpty) {
                return const Center(
                  child: Padding(
                    padding: EdgeInsets.all(24),
                    child: Text(
                      'Kein verkaufbarer Besitz gefunden.',
                      textAlign: TextAlign.center,
                    ),
                  ),
                );
              }
              return ListView.builder(
                key: const Key('sell-asset-list'),
                itemCount: filtered.length,
                itemBuilder: (context, i) {
                  final asset = filtered[i];
                  final selected = identical(asset, _selected) ||
                      (_selected != null &&
                          asset.typeId == _selected!.typeId &&
                          asset.locationId == _selected!.locationId);
                  return ListTile(
                    key: Key(
                        'sell-asset-${asset.typeId}-${asset.locationId}'),
                    enabled: asset.marketable && !busy,
                    selected: selected,
                    title: Text(
                      asset.name,
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                    ),
                    subtitle: Text(
                      '${_fmtUnits(asset.quantity.toDouble())} × '
                      '${asset.locationName}',
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                    ),
                    trailing: asset.marketable
                        ? null
                        : const Text('nicht handelbar',
                            style: TextStyle(fontSize: 11)),
                    onTap: asset.marketable && !busy
                        ? () => _select(asset)
                        : null,
                  );
                },
              );
            },
          ),
        ),

        // ── Selection form (quantity + avoid-low-sec + search) ───────────────
        if (_selected != null) ...[
          const Divider(height: 1),
          Padding(
            padding: const EdgeInsets.all(16),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              mainAxisSize: MainAxisSize.min,
              children: [
                Text(
                  _selected!.name,
                  style: Theme.of(context).textTheme.titleMedium,
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                ),
                const SizedBox(height: 8),
                TextField(
                  key: const Key('sell-quantity'),
                  controller: _quantityController,
                  enabled: !busy,
                  keyboardType: TextInputType.number,
                  inputFormatters: [
                    FilteringTextInputFormatter.digitsOnly
                  ],
                  decoration: const InputDecoration(
                    labelText: 'Menge',
                    border: OutlineInputBorder(),
                    isDense: true,
                    suffixText: 'Stk',
                    contentPadding:
                        EdgeInsets.symmetric(horizontal: 12, vertical: 8),
                  ),
                ),
                const SizedBox(height: 4),
                SwitchListTile(
                  key: const Key('sell-avoid-lowsec'),
                  value: _avoidLowSec,
                  onChanged:
                      busy ? null : (v) => setState(() => _avoidLowSec = v),
                  title: const Text(
                    'Low-Sec vermeiden',
                    style: TextStyle(fontSize: 14),
                  ),
                  contentPadding: EdgeInsets.zero,
                  dense: true,
                ),
                if (_error != null) ...[
                  const SizedBox(height: 4),
                  Text(
                    _error!,
                    style: TextStyle(
                        color: Theme.of(context).colorScheme.error),
                  ),
                ],
                const SizedBox(height: 12),
                FilledButton.icon(
                  key: const Key('sell-search-button'),
                  onPressed: busy ? null : _search,
                  icon: busy
                      ? const SizedBox(
                          width: 16,
                          height: 16,
                          child:
                              CircularProgressIndicator(strokeWidth: 2),
                        )
                      : const Icon(Icons.sell_rounded),
                  label: const Text('Verkaufsorte suchen'),
                ),
              ],
            ),
          ),
        ],
      ],
    );
  }
}

// ---------------------------------------------------------------------------
// Result pane — async-aware
// ---------------------------------------------------------------------------

class _ResultPane extends ConsumerWidget {
  const _ResultPane({required this.twoPane});

  final bool twoPane;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final async = ref.watch(sellOptionsProvider);

    return async.when(
      loading: () => const Center(child: CircularProgressIndicator()),
      error: (err, _) => Center(
        child: Padding(
          padding: const EdgeInsets.all(24),
          child: Text(
            'Suche fehlgeschlagen.\n$err',
            textAlign: TextAlign.center,
            style: TextStyle(color: Theme.of(context).colorScheme.error),
          ),
        ),
      ),
      data: (resp) {
        if (resp == null) {
          return const _IdleHint();
        }
        if (resp.isEmpty) {
          return const _EmptyResult();
        }
        return _OptionList(response: resp, twoPane: twoPane);
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
              Icons.sell_rounded,
              size: 56,
              color: Theme.of(context).colorScheme.onSurface.withAlpha(80),
            ),
            const SizedBox(height: 16),
            Text(
              'Item wählen und Verkaufsorte suchen',
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

class _EmptyResult extends StatelessWidget {
  const _EmptyResult();

  @override
  Widget build(BuildContext context) {
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(32),
        child: Column(
          key: const Key('sell-empty-state'),
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(
              Icons.sentiment_dissatisfied_rounded,
              size: 56,
              color: Theme.of(context).colorScheme.onSurface.withAlpha(80),
            ),
            const SizedBox(height: 16),
            Text(
              'Keine Verkaufsorte gefunden',
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
// Option list — `best` highlighted, then the ranked list
// ---------------------------------------------------------------------------

class _OptionList extends StatelessWidget {
  const _OptionList({required this.response, required this.twoPane});

  final SellOptionsResponse response;
  final bool twoPane;

  @override
  Widget build(BuildContext context) {
    final skills = response.skillsApplied;
    final best = response.best;
    return ListView(
      key: const Key('sell-option-list'),
      padding: const EdgeInsets.all(16),
      children: [
        Text(
          'Verkaufsorte für ${response.name}',
          style: Theme.of(context).textTheme.titleLarge,
        ),
        const SizedBox(height: 4),
        Text(
          skills.applied
              ? 'Skill-bereinigt (Accounting ${skills.accounting}, '
                  'Verkaufssteuer '
                  '${(skills.salesTaxRate * 100).toStringAsFixed(1)}%)'
              : 'Ohne Skill-Bereinigung',
          style: Theme.of(context).textTheme.bodySmall,
        ),
        const SizedBox(height: 12),
        if (best != null) ...[
          _OptionCard(option: best, twoPane: twoPane, isBest: true),
          const SizedBox(height: 4),
        ],
        for (final option in response.options)
          _OptionCard(option: option, twoPane: twoPane, isBest: false),
      ],
    );
  }
}

class _OptionCard extends StatelessWidget {
  const _OptionCard({
    required this.option,
    required this.twoPane,
    required this.isBest,
  });

  final SellOption option;
  final bool twoPane;
  final bool isBest;

  void _openDetail(BuildContext context) {
    if (twoPane) {
      showDialog<void>(
        context: context,
        builder: (_) => Dialog(
          child: ConstrainedBox(
            constraints: const BoxConstraints(maxWidth: 560, maxHeight: 720),
            child: SellOptionDetail(option: option, itemName: option.systemName),
          ),
        ),
      );
    } else {
      Navigator.of(context).push(
        MaterialPageRoute<void>(
          builder: (_) => Scaffold(
            appBar: AppBar(title: Text(option.systemName)),
            body: SellOptionDetail(option: option, itemName: option.systemName),
          ),
        ),
      );
    }
  }

  @override
  Widget build(BuildContext context) {
    final accent = Theme.of(context).colorScheme.primary;
    final muted = Theme.of(context).colorScheme.onSurface.withAlpha(153);
    final scheme = Theme.of(context).colorScheme;

    // has_data:false → muted "kein Marktpreis" card, not tappable.
    if (!option.hasData) {
      return Card(
        key: Key('sell-option-${option.stationId}'),
        margin: const EdgeInsets.only(bottom: 12),
        color: scheme.surfaceContainerHighest.withAlpha(80),
        child: Padding(
          padding: const EdgeInsets.all(16),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Row(
                children: [
                  Expanded(
                    child: Text(
                      option.systemName,
                      style: Theme.of(context).textTheme.titleMedium?.copyWith(
                            color: muted,
                          ),
                      overflow: TextOverflow.ellipsis,
                    ),
                  ),
                  _ScopeBadge(scope: option.scope),
                ],
              ),
              const SizedBox(height: 4),
              Text(
                option.stationName,
                style: Theme.of(context)
                    .textTheme
                    .bodySmall
                    ?.copyWith(color: muted),
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
              ),
              const SizedBox(height: 8),
              Text(
                'kein Marktpreis',
                style: Theme.of(context).textTheme.bodyMedium?.copyWith(
                      color: muted,
                      fontStyle: FontStyle.italic,
                    ),
              ),
            ],
          ),
        ),
      );
    }

    return Card(
      key: Key('sell-option-${option.stationId}'),
      margin: const EdgeInsets.only(bottom: 12),
      shape: isBest
          ? RoundedRectangleBorder(
              borderRadius: BorderRadius.circular(12),
              side: BorderSide(color: accent, width: 2),
            )
          : null,
      child: InkWell(
        onTap: () => _openDetail(context),
        borderRadius: BorderRadius.circular(12),
        child: Padding(
          padding: const EdgeInsets.all(16),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              // ── Header: system + best badge + scope + sec-risk ─────────────
              Row(
                children: [
                  if (isBest) ...[
                    Icon(Icons.star_rounded, size: 18, color: accent),
                    const SizedBox(width: 4),
                  ],
                  Expanded(
                    child: Text(
                      option.systemName,
                      style:
                          Theme.of(context).textTheme.titleMedium?.copyWith(
                                fontWeight: FontWeight.w700,
                              ),
                      overflow: TextOverflow.ellipsis,
                    ),
                  ),
                  const SizedBox(width: 8),
                  _ScopeBadge(scope: option.scope),
                  const SizedBox(width: 6),
                  _SecurityRiskChip(risk: option.securityRisk),
                ],
              ),
              const SizedBox(height: 2),
              Text(
                option.stationName,
                style: Theme.of(context)
                    .textTheme
                    .bodySmall
                    ?.copyWith(color: muted),
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
              ),
              const SizedBox(height: 8),

              // ── total_net — large accent (headline) ────────────────────────
              Text(
                '${_fmtIsk(option.totalNet)} ISK',
                style: Theme.of(context).textTheme.titleLarge?.copyWith(
                      color: accent,
                      fontWeight: FontWeight.bold,
                    ),
              ),
              const SizedBox(height: 8),

              // ── Meta row: buy_price, unit_net, jumps, travel time ──────────
              Wrap(
                spacing: 16,
                runSpacing: 4,
                children: [
                  _Meta(
                    icon: Icons.shopping_cart_rounded,
                    label: 'Kauf ${_fmtIsk(option.buyPrice)}',
                  ),
                  _Meta(
                    icon: Icons.attach_money_rounded,
                    label: 'Netto/Stk ${_fmtIsk(option.unitNet)}',
                  ),
                  _Meta(
                    icon: Icons.alt_route_rounded,
                    label: '${option.jumps} Sprünge',
                  ),
                  _Meta(
                    icon: Icons.schedule_rounded,
                    label: '${option.travelTimeMin.toStringAsFixed(1)} min',
                  ),
                ],
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class _Meta extends StatelessWidget {
  const _Meta({required this.icon, required this.label});
  final IconData icon;
  final String label;

  @override
  Widget build(BuildContext context) {
    final muted = Theme.of(context).colorScheme.onSurface.withAlpha(153);
    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        Icon(icon, size: 16, color: muted),
        const SizedBox(width: 4),
        Text(label, style: TextStyle(color: muted)),
      ],
    );
  }
}

// ---------------------------------------------------------------------------
// Scope badge (Hub / Region)
// ---------------------------------------------------------------------------

class _ScopeBadge extends StatelessWidget {
  const _ScopeBadge({required this.scope});

  final String scope;

  @override
  Widget build(BuildContext context) {
    final isHub = scope == 'hub';
    final color = isHub
        ? Theme.of(context).colorScheme.primary
        : Theme.of(context).colorScheme.tertiary;
    return Container(
      key: Key('sell-scope-badge-$scope'),
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
      decoration: BoxDecoration(
        color: color.withAlpha(40),
        borderRadius: BorderRadius.circular(10),
        border: Border.all(color: color.withAlpha(150)),
      ),
      child: Text(
        isHub ? 'Hub' : 'Region',
        style: TextStyle(
          color: color,
          fontWeight: FontWeight.w600,
          fontSize: 11,
        ),
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Security-risk chip (safe=green / caution=amber / danger=red)
// ---------------------------------------------------------------------------

class _SecurityRiskChip extends StatelessWidget {
  const _SecurityRiskChip({required this.risk});

  final String risk;

  @override
  Widget build(BuildContext context) {
    final color = sellSecurityRiskColor(risk);
    return Container(
      key: Key('sell-risk-chip-$risk'),
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
      decoration: BoxDecoration(
        color: color.withAlpha(40),
        borderRadius: BorderRadius.circular(12),
        border: Border.all(color: color.withAlpha(150)),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Icon(Icons.shield_rounded, size: 14, color: color),
          const SizedBox(width: 4),
          Text(
            sellSecurityRiskLabel(risk),
            style: TextStyle(
              color: color,
              fontWeight: FontWeight.w600,
              fontSize: 12,
            ),
          ),
        ],
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Option detail — full breakdown + "Route an EVE übertragen" button
// ---------------------------------------------------------------------------

/// Full detail view for a [SellOption]: a metrics breakdown plus a button that
/// pushes the sell station to the in-game autopilot (clearing other waypoints).
class SellOptionDetail extends ConsumerWidget {
  const SellOptionDetail({
    super.key,
    required this.option,
    required this.itemName,
  });

  final SellOption option;

  /// Used only for the snackbar confirmation message.
  final String itemName;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final accent = Theme.of(context).colorScheme.primary;
    final muted = Theme.of(context).colorScheme.onSurface.withAlpha(153);

    return SingleChildScrollView(
      padding: const EdgeInsets.all(24),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            'Verkaufsort',
            style: Theme.of(context).textTheme.labelLarge?.copyWith(
                  color: muted,
                  letterSpacing: 1.1,
                ),
          ),
          const SizedBox(height: 4),
          Row(
            children: [
              Expanded(
                child: Text(
                  option.systemName,
                  style: Theme.of(context).textTheme.headlineSmall?.copyWith(
                        fontWeight: FontWeight.bold,
                      ),
                ),
              ),
              const SizedBox(width: 8),
              _ScopeBadge(scope: option.scope),
              const SizedBox(width: 6),
              _SecurityRiskChip(risk: option.securityRisk),
            ],
          ),
          const SizedBox(height: 4),
          Text(
            option.stationName,
            style: Theme.of(context).textTheme.bodySmall?.copyWith(color: muted),
          ),
          const SizedBox(height: 12),

          Text(
            '${_fmtIsk(option.totalNet)} ISK netto',
            style: Theme.of(context).textTheme.headlineMedium?.copyWith(
                  color: accent,
                  fontWeight: FontWeight.bold,
                ),
          ),
          const SizedBox(height: 8),

          // ── Summary row ──────────────────────────────────────────────────
          Wrap(
            spacing: 16,
            runSpacing: 4,
            children: [
              _Meta(
                icon: Icons.shopping_cart_rounded,
                label: 'Kaufpreis ${_fmtIsk(option.buyPrice)}',
              ),
              _Meta(
                icon: Icons.attach_money_rounded,
                label: 'Netto/Stk ${_fmtIsk(option.unitNet)}',
              ),
              _Meta(
                icon: Icons.alt_route_rounded,
                label: '${option.jumps} Sprünge',
              ),
              _Meta(
                icon: Icons.schedule_rounded,
                label: '${option.travelTimeMin.toStringAsFixed(1)} min',
              ),
              _Meta(
                icon: Icons.public_rounded,
                label: option.regionName,
              ),
            ],
          ),
          const SizedBox(height: 24),

          // ── Action button ────────────────────────────────────────────────
          SizedBox(
            width: double.infinity,
            child: FilledButton.icon(
              key: const Key('sell-waypoint-button'),
              onPressed: () => _setWaypoint(context, ref),
              icon: const Icon(Icons.navigation_rounded),
              label: const Text('Route an EVE übertragen'),
              style: FilledButton.styleFrom(
                backgroundColor: accent,
                foregroundColor: Colors.white,
                padding: const EdgeInsets.symmetric(vertical: 14),
                textStyle: const TextStyle(
                  fontWeight: FontWeight.w600,
                  fontSize: 16,
                ),
              ),
            ),
          ),
        ],
      ),
    );
  }

  Future<void> _setWaypoint(BuildContext context, WidgetRef ref) async {
    final api = ref.read(tradingApiProvider);
    try {
      // Single sell destination; clear any existing route.
      await api.setWaypoint(
        destinationId: option.stationId,
        clearOtherWaypoints: true,
        addToBeginning: false,
      );
      if (context.mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
            content: Text('Route gesetzt: ${option.systemName}'),
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
// Formatting helpers (mirrors hauling/roi/hub screens)
// ---------------------------------------------------------------------------

String _fmtIsk(double v) {
  if (v >= 1e9) return '${(v / 1e9).toStringAsFixed(2)}B';
  if (v >= 1e6) return '${(v / 1e6).toStringAsFixed(2)}M';
  if (v >= 1e3) return '${(v / 1e3).toStringAsFixed(1)}K';
  return v.toStringAsFixed(2);
}

String _fmtUnits(double v) {
  if (v >= 1e6) return '${(v / 1e6).toStringAsFixed(1)}M';
  if (v >= 1e3) return '${(v / 1e3).toStringAsFixed(1)}K';
  return v.toStringAsFixed(0);
}
