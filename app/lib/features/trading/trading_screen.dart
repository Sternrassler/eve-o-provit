/// Adaptive TradingScreen — single-pane on phones / tablets in portrait,
/// two-pane side-by-side on wide displays (≥840 dp).
///
/// Controls area: region dropdown, ship type input, market staleness/refresh,
/// security-zone filter toggles, and the "Berechnen" action.
library;

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../api/trading_models.dart';
import '../../core/breakpoint.dart';
import '../trading/providers.dart';
import 'route_detail.dart';
import 'route_list.dart';

// ---------------------------------------------------------------------------
// Public entry point
// ---------------------------------------------------------------------------

/// The main trading screen.
///
/// Uses [LayoutBuilder] to switch between single-pane (list only) and
/// two-pane (list + detail side-by-side) layouts at the [kTwoPaneBreakpoint].
class TradingScreen extends ConsumerWidget {
  const TradingScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
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

    return Row(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        // Left pane: route list — fixed 360 dp
        SizedBox(
          width: 360,
          child: RouteList(),
        ),
        const VerticalDivider(width: 1),
        // Right pane: detail or placeholder
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
    return RouteList(
      onRouteTap: (route) => _pushDetail(context, route),
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

/// Horizontal (or wrapping) bar with all trading controls.
class _ControlsBar extends ConsumerStatefulWidget {
  @override
  ConsumerState<_ControlsBar> createState() => _ControlsBarState();
}

class _ControlsBarState extends ConsumerState<_ControlsBar> {
  final _shipController = TextEditingController(text: '648'); // Badger default
  String? _stalenessText;
  bool _refreshing = false;

  @override
  void initState() {
    super.initState();
    // Sync text field from existing provider state.
    final existing = ref.read(selectedShipTypeIdProvider);
    if (existing != null) {
      _shipController.text = existing.toString();
    }
    _shipController.addListener(_onShipTypeChanged);
    // Seed the default ship type (Badger 648) after the first frame so we
    // don't mutate providers during the build phase.
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted) return;
      if (ref.read(selectedShipTypeIdProvider) == null) {
        ref.read(selectedShipTypeIdProvider.notifier).select(648);
      }
    });
  }

  void _onShipTypeChanged() {
    final v = int.tryParse(_shipController.text);
    if (v != null && v > 0) {
      ref.read(selectedShipTypeIdProvider.notifier).select(v);
    }
  }

  @override
  void dispose() {
    _shipController.removeListener(_onShipTypeChanged);
    _shipController.dispose();
    super.dispose();
  }

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
      // refreshMarket takes a typeId; we use 34 (Tritanium) as a proxy trigger.
      await api.refreshMarket(region.id, 34);
      final staleness = await api.staleness(region.id);
      if (mounted) {
        setState(() {
          _stalenessText =
              '${staleness.ageMinutes} Min. alt · ${staleness.status}';
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
      _showSnack('Bitte eine Schiff-Typ-ID eingeben.');
      return;
    }
    await ref.read(routesProvider.notifier).calculate();
  }

  @override
  Widget build(BuildContext context) {
    final regionsAsync = ref.watch(regionsProvider);
    final selectedRegion = ref.watch(selectedRegionProvider);
    final filters = ref.watch(filtersProvider);
    final accent = Theme.of(context).colorScheme.primary;

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

            // ── Ship type ID input ──────────────────────────────────────────
            SizedBox(
              width: 140,
              child: TextFormField(
                controller: _shipController,
                keyboardType: TextInputType.number,
                decoration: const InputDecoration(
                  labelText: 'Schiff-Typ-ID',
                  border: OutlineInputBorder(),
                  contentPadding:
                      EdgeInsets.symmetric(horizontal: 12, vertical: 8),
                  isDense: true,
                  hintText: '648',
                ),
              ),
            ),

            // ── Market refresh + staleness ──────────────────────────────────
            _refreshing
                ? const SizedBox(
                    width: 24,
                    height: 24,
                    child: CircularProgressIndicator(strokeWidth: 2),
                  )
                : IconButton.outlined(
                    onPressed: _refreshMarket,
                    icon: const Icon(Icons.refresh_rounded),
                    tooltip: 'Marktdaten aktualisieren',
                  ),
            if (_stalenessText != null)
              Text(
                _stalenessText!,
                style: Theme.of(context).textTheme.bodySmall?.copyWith(
                      color: Theme.of(context)
                          .colorScheme
                          .onSurface
                          .withAlpha(153),
                    ),
              ),

            // ── Security-zone toggles ──────────────────────────────────────
            _SecZoneToggle(
              label: 'Lo',
              active: filters.lowSec,
              activeColor: const Color(0xFFFF9800),
              onChanged: (v) => ref
                  .read(filtersProvider.notifier)
                  .update((f) => f.copyWith(lowSec: v)),
            ),
            _SecZoneToggle(
              label: 'Null',
              active: filters.nullSec,
              activeColor: const Color(0xFFF44336),
              onChanged: (v) => ref
                  .read(filtersProvider.notifier)
                  .update((f) => f.copyWith(nullSec: v)),
            ),

            // ── Calculate button ───────────────────────────────────────────
            FilledButton(
              onPressed: _calculate,
              style: FilledButton.styleFrom(
                backgroundColor: accent,
                foregroundColor: Colors.white,
                padding:
                    const EdgeInsets.symmetric(horizontal: 20, vertical: 10),
              ),
              child: const Text(
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

// ---------------------------------------------------------------------------
// Sec-zone toggle pill
// ---------------------------------------------------------------------------

class _SecZoneToggle extends StatelessWidget {
  const _SecZoneToggle({
    required this.label,
    required this.active,
    required this.activeColor,
    required this.onChanged,
  });

  final String label;
  final bool active;
  final Color activeColor;
  final ValueChanged<bool> onChanged;

  @override
  Widget build(BuildContext context) {
    return GestureDetector(
      onTap: () => onChanged(!active),
      child: AnimatedContainer(
        duration: const Duration(milliseconds: 150),
        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
        decoration: BoxDecoration(
          color: active ? activeColor.withAlpha(30) : Colors.transparent,
          borderRadius: BorderRadius.circular(20),
          border: Border.all(
            color: active
                ? activeColor
                : Theme.of(context).colorScheme.outline.withAlpha(100),
          ),
        ),
        child: Text(
          label,
          style: TextStyle(
            color: active
                ? activeColor
                : Theme.of(context).colorScheme.onSurface.withAlpha(153),
            fontWeight: FontWeight.w600,
            fontSize: 13,
          ),
        ),
      ),
    );
  }
}
