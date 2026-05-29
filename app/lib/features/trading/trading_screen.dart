/// Adaptive TradingScreen — single-pane on phones / tablets in portrait,
/// two-pane side-by-side on wide displays (≥840 dp).
///
/// Controls area: region dropdown, ship selector, market staleness/refresh,
/// and the "Berechnen" action.
///
/// Left-column sidebar (two-pane): TradingFiltersPanel + ShipFittingCard.
/// In single-pane the filters panel appears (collapsed) above the route list.
library;

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../api/trading_models.dart';
import '../../auth/auth_controller.dart';
import '../../core/breakpoint.dart';
import '../trading/providers.dart';
import 'current_selection_prefill.dart';
import 'route_detail.dart';
import 'route_list.dart';
import 'ship_fitting_card.dart';
import 'ship_select.dart';
import 'trading_filters_panel.dart';

// ---------------------------------------------------------------------------
// Public entry point
// ---------------------------------------------------------------------------

/// The main trading screen.
class TradingScreen extends ConsumerStatefulWidget {
  const TradingScreen({super.key});

  @override
  ConsumerState<TradingScreen> createState() => _TradingScreenState();
}

class _TradingScreenState extends ConsumerState<TradingScreen>
    with CurrentSelectionPrefill {
  @override
  void initState() {
    super.initState();
    // Seed region + ship from the character's current region/active ship once.
    startSelectionPrefill();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Trading'),
        centerTitle: false,
        elevation: 0,
        scrolledUnderElevation: 1,
      ),
      body: LayoutBuilder(
        builder: (context, constraints) {
          final twoPane = isTwoPane(constraints.maxWidth);
          return Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              // ── Controls ──────────────────────────────────────────────────
              _ControlsBar(),
              const Divider(height: 1),
              // ── Content ───────────────────────────────────────────────────
              Expanded(
                child: twoPane
                    ? _TwoPaneLayout()
                    : _SinglePaneLayout(),
              ),
            ],
          );
        },
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Two-pane layout
// ---------------------------------------------------------------------------

class _TwoPaneLayout extends ConsumerWidget {
  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final selectedRoute = ref.watch(selectedRouteProvider);
    final selectedShipTypeId = ref.watch(selectedShipTypeIdProvider);
    final authState = ref.watch(authControllerProvider).value;
    final character =
        authState is Authenticated ? authState.character : null;

    return Row(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        // Left pane: filters + fitting card — fixed 320 dp
        SizedBox(
          width: 320,
          child: SingleChildScrollView(
            padding: const EdgeInsets.all(12),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                const TradingFiltersPanel(),
                const SizedBox(height: 12),
                if (character != null && selectedShipTypeId != null)
                  ShipFittingCard(
                    characterId: character.id,
                    shipTypeId: selectedShipTypeId,
                  ),
              ],
            ),
          ),
        ),
        const VerticalDivider(width: 1),
        // Right pane: route list (360 dp) + detail
        SizedBox(
          width: 360,
          child: RouteList(),
        ),
        const VerticalDivider(width: 1),
        // Detail pane
        Expanded(
          child: selectedRoute != null
              ? RouteDetail(route: selectedRoute)
              : _DetailPlaceholder(),
        ),
      ],
    );
  }
}

// ---------------------------------------------------------------------------
// Single-pane layout
// ---------------------------------------------------------------------------

class _SinglePaneLayout extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    // Portrait has no side column, so surface the filter panel (collapsed by
    // default to save vertical space) above the route list.
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        const Padding(
          padding: EdgeInsets.fromLTRB(12, 12, 12, 0),
          child: TradingFiltersPanel(initiallyCollapsed: true),
        ),
        Expanded(
          child: RouteList(
            onRouteTap: (route) => _pushDetail(context, route),
          ),
        ),
      ],
    );
  }

  void _pushDetail(BuildContext context, TradingRoute route) {
    Navigator.of(context).push(
      MaterialPageRoute<void>(
        builder: (_) => _DetailPage(route: route),
      ),
    );
  }
}

/// A standalone page wrapping [RouteDetail] for single-pane navigation.
class _DetailPage extends StatelessWidget {
  const _DetailPage({required this.route});

  final TradingRoute route;

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: Text(route.itemName),
        centerTitle: false,
      ),
      body: RouteDetail(route: route),
    );
  }
}

// ---------------------------------------------------------------------------
// Placeholder shown in the right pane before any route is selected
// ---------------------------------------------------------------------------

class _DetailPlaceholder extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(32),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(
              Icons.route_rounded,
              size: 56,
              color:
                  Theme.of(context).colorScheme.onSurface.withAlpha(80),
            ),
            const SizedBox(height: 16),
            Text(
              'Route auswählen',
              style: Theme.of(context).textTheme.titleMedium?.copyWith(
                    color: Theme.of(context)
                        .colorScheme
                        .onSurface
                        .withAlpha(120),
                  ),
            ),
          ],
        ),
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Controls bar
// ---------------------------------------------------------------------------

/// Horizontal (or wrapping) bar with trading controls.
///
/// Contains: Region dropdown, ShipSelect, market refresh, Berechnen button.
class _ControlsBar extends ConsumerStatefulWidget {
  @override
  ConsumerState<_ControlsBar> createState() => _ControlsBarState();
}

class _ControlsBarState extends ConsumerState<_ControlsBar> {
  String? _stalenessText;
  Color? _stalenessColor;
  bool _refreshing = false;
  bool _calculating = false;

  Future<void> _refreshMarket() async {
    final region = ref.read(selectedRegionProvider);
    if (region == null) {
      _showSnack('Bitte zuerst eine Region auswählen.');
      return;
    }
    setState(() {
      _refreshing = true;
      _stalenessText = null;
    });
    try {
      final api = ref.read(tradingApiProvider);
      await api.refreshMarket(region.id, 34);
      final staleness = await api.staleness(region.id);
      if (mounted) {
        setState(() {
          // Mirror the web RegionStalenessIndicator: show only the age,
          // colour-coded by freshness (the backend sends no status field).
          _stalenessText = _formatAge(staleness.ageMinutes);
          _stalenessColor = _ageColor(staleness.ageMinutes);
        });
      }
    } catch (e) {
      if (mounted) {
        _showSnack('Refresh fehlgeschlagen: $e');
      }
    } finally {
      if (mounted) setState(() => _refreshing = false);
    }
  }

  /// Human-readable market-data age, mirroring the web's `formatAge`.
  String _formatAge(double? minutes) {
    if (minutes == null) return 'Keine Marktdaten';
    if (minutes < 1) return '< 1 Min.';
    if (minutes < 60) return '${minutes.floor()} Min. alt';
    final h = minutes ~/ 60;
    final m = (minutes % 60).floor();
    return '${h}h ${m}m alt';
  }

  /// Freshness colour, mirroring the web's `getStatusColor` thresholds.
  Color? _ageColor(double? minutes) {
    if (minutes == null) return null; // muted default
    if (minutes < 5) return const Color(0xFF66BB6A); // green
    if (minutes < 15) return const Color(0xFFFFB300); // amber
    return const Color(0xFFFF9800); // orange
  }

  void _showSnack(String msg) {
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(
        content: Text(msg),
        behavior: SnackBarBehavior.floating,
      ),
    );
  }

  Future<void> _calculate() async {
    final region = ref.read(selectedRegionProvider);
    final shipTypeId = ref.read(selectedShipTypeIdProvider);
    if (region == null) {
      _showSnack('Bitte eine Region auswählen.');
      return;
    }
    if (shipTypeId == null) {
      _showSnack('Bitte ein Schiff auswählen.');
      return;
    }
    setState(() => _calculating = true);
    try {
      await ref.read(routesProvider.notifier).calculate();
    } finally {
      if (mounted) setState(() => _calculating = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final regionsAsync = ref.watch(regionsProvider);
    final selectedRegion = ref.watch(selectedRegionProvider);
    final accent = Theme.of(context).colorScheme.primary;
    final busy = _refreshing || _calculating;

    return Material(
      color: Theme.of(context).colorScheme.surface,
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 10),
        child: Wrap(
          spacing: 12,
          runSpacing: 10,
          crossAxisAlignment: WrapCrossAlignment.center,
          children: [
            // ── Region dropdown ─────────────────────────────────────────────
            ConstrainedBox(
              constraints: const BoxConstraints(minWidth: 160, maxWidth: 220),
              child: regionsAsync.when(
                loading: () => const SizedBox(
                  height: 40,
                  child: Center(child: LinearProgressIndicator()),
                ),
                error: (err, stack) => const Text('Regionen nicht verfügbar'),
                data: (regions) => DropdownButtonFormField<Region>(
                  initialValue:
                      regions.contains(selectedRegion) ? selectedRegion : null,
                  isExpanded: true,
                  decoration: const InputDecoration(
                    labelText: 'Region',
                    border: OutlineInputBorder(),
                    contentPadding:
                        EdgeInsets.symmetric(horizontal: 12, vertical: 8),
                    isDense: true,
                  ),
                  items: regions
                      .map(
                        (r) => DropdownMenuItem(
                          value: r,
                          child: Text(
                            r.name,
                            overflow: TextOverflow.ellipsis,
                          ),
                        ),
                      )
                      .toList(),
                  onChanged: (r) {
                    ref.read(selectedRegionProvider.notifier).select(r);
                    setState(() => _stalenessText = null);
                  },
                ),
              ),
            ),

            // ── Ship selector ───────────────────────────────────────────────
            SizedBox(
              width: 220,
              child: ShipSelect(enabled: !busy),
            ),

            // ── Market refresh + staleness ──────────────────────────────────
            _refreshing
                ? const SizedBox(
                    width: 24,
                    height: 24,
                    child: CircularProgressIndicator(strokeWidth: 2),
                  )
                : IconButton.outlined(
                    onPressed: busy ? null : _refreshMarket,
                    icon: const Icon(Icons.refresh_rounded),
                    tooltip: 'Marktdaten aktualisieren',
                  ),
            if (_stalenessText != null)
              Text(
                _stalenessText!,
                style: Theme.of(context).textTheme.bodySmall?.copyWith(
                      color: _stalenessColor ??
                          Theme.of(context)
                              .colorScheme
                              .onSurface
                              .withAlpha(153),
                      fontWeight: _stalenessColor != null
                          ? FontWeight.w600
                          : null,
                    ),
              ),

            // ── Calculate button ───────────────────────────────────────────
            FilledButton(
              onPressed: busy ? null : _calculate,
              style: FilledButton.styleFrom(
                backgroundColor: accent,
                foregroundColor: Colors.white,
                padding:
                    const EdgeInsets.symmetric(horizontal: 20, vertical: 10),
              ),
              child: _calculating
                  ? const SizedBox(
                      width: 16,
                      height: 16,
                      child: CircularProgressIndicator(
                        strokeWidth: 2,
                        valueColor:
                            AlwaysStoppedAnimation<Color>(Colors.white),
                      ),
                    )
                  : const Text(
                      'Berechnen',
                      style: TextStyle(fontWeight: FontWeight.w600),
                    ),
            ),
          ],
        ),
      ),
    );
  }
}

