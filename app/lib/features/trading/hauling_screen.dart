/// Adaptive Neighborhood Hauling Routes screen.
///
/// The user picks a Ship + enters their available Capital (ISK) + toggles
/// "avoid low-sec". The backend then scans the character's current region plus
/// adjacent regions and returns profitable station→station hauling routes
/// (buy cheap at one station, sell dear at another), each with an optimal cargo
/// load and an isk/hour metric. Routes are pre-sorted by isk/hour desc. The
/// user can push a route to the in-game autopilot.
///
/// Layout (via [isTwoPane] / [kTwoPaneBreakpoint] = 840 dp):
///   • <840 dp  → single-pane: input form above a scrollable route list;
///                tapping a route pushes a detail page.
///   • ≥840 dp  → two-pane: input form on the left, route list on the right;
///                tapping a route shows its detail in a dialog.
library;

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../api/hauling_models.dart';
import '../../core/breakpoint.dart';
import '../../core/format.dart';
import '../../core/security_risk.dart';
import '../character/providers.dart';
import 'current_ship_card.dart';
import 'hauling_providers.dart';

// ---------------------------------------------------------------------------
// Public entry point
// ---------------------------------------------------------------------------

/// The hauling-routes screen.
class HaulingScreen extends ConsumerWidget {
  const HaulingScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Hauling-Routen'),
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
                Expanded(child: _ResultPane(twoPane: true)),
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
              Expanded(child: _ResultPane(twoPane: false)),
            ],
          );
        },
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Input form — ship picker + capital + avoid-low-sec switch
// ---------------------------------------------------------------------------

class _InputForm extends ConsumerStatefulWidget {
  const _InputForm();

  @override
  ConsumerState<_InputForm> createState() => _InputFormState();
}

class _InputFormState extends ConsumerState<_InputForm> {
  final _capitalController = TextEditingController(text: '500000000');
  bool _avoidLowSec = true;
  String? _error;

  // No ship/region prefill needed: hauling uses the character's CURRENT ship
  // directly and derives its origin region backend-side (origin_region_id: 0).

  @override
  void dispose() {
    _capitalController.dispose();
    super.dispose();
  }

  Future<void> _search() async {
    final ship = ref.read(currentShipProvider).value;
    final capital = double.tryParse(
        _capitalController.text.replaceAll(RegExp(r'[^0-9.]'), ''));

    // Never send ship_type_id 0 — gated by the button, guarded here too.
    if (ship == null || ship.shipTypeId <= 0) {
      setState(() => _error = 'Aktuelles Schiff noch nicht geladen.');
      return;
    }
    if (capital == null || capital <= 0) {
      setState(() => _error = 'Bitte ein gültiges Kapital eingeben.');
      return;
    }

    setState(() => _error = null);

    // origin_region_id: 0 ⇒ backend uses the character's current region.
    final request = HaulingRequest(
      originRegionId: 0,
      shipTypeId: ship.shipTypeId,
      capital: capital,
      avoidLowSec: _avoidLowSec,
      maxRoutes: 15,
      cargoCapacity: cargoOverrideForShip(ship),
    );

    await ref.read(haulingRoutesProvider.notifier).find(request);
  }

  @override
  Widget build(BuildContext context) {
    final busy = ref.watch(haulingRoutesProvider).isLoading;
    // Gate "Routen suchen" until the current ship is loaded (valid type id).
    final ship = ref.watch(currentShipProvider).value;
    final shipReady = ship != null && ship.shipTypeId > 0;

    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      mainAxisSize: MainAxisSize.min,
      children: [
        // ── Ship (read-only current ship + refresh) ─────────────────────────
        const CurrentShipCard(),
        const SizedBox(height: 12),

        // ── Capital ────────────────────────────────────────────────────────
        TextField(
          controller: _capitalController,
          enabled: !busy,
          keyboardType: TextInputType.number,
          inputFormatters: [FilteringTextInputFormatter.digitsOnly],
          decoration: const InputDecoration(
            labelText: 'Kapital',
            border: OutlineInputBorder(),
            isDense: true,
            suffixText: 'ISK',
            contentPadding: EdgeInsets.symmetric(horizontal: 12, vertical: 8),
          ),
        ),
        const SizedBox(height: 8),

        // ── Avoid low-sec ──────────────────────────────────────────────────
        SwitchListTile(
          key: const Key('hauling-avoid-lowsec'),
          value: _avoidLowSec,
          onChanged: busy ? null : (v) => setState(() => _avoidLowSec = v),
          title: const Text(
            'Low-Sec vermeiden',
            style: TextStyle(fontSize: 14),
          ),
          contentPadding: EdgeInsets.zero,
          dense: true,
        ),

        if (_error != null) ...[
          const SizedBox(height: 8),
          Text(
            _error!,
            style: TextStyle(color: Theme.of(context).colorScheme.error),
          ),
        ],

        const SizedBox(height: 16),

        // ── Search button ──────────────────────────────────────────────────
        FilledButton.icon(
          onPressed: (busy || !shipReady) ? null : _search,
          icon: busy
              ? const SizedBox(
                  width: 16,
                  height: 16,
                  child: CircularProgressIndicator(strokeWidth: 2),
                )
              : const Icon(Icons.local_shipping_rounded),
          label: const Text('Routen suchen'),
        ),
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
    final async = ref.watch(haulingRoutesProvider);

    return async.when(
      loading: () => const Center(child: CircularProgressIndicator()),
      error: (err, _) => Center(
        child: Padding(
          padding: const EdgeInsets.all(24),
          child: Text(
            'Routensuche fehlgeschlagen.\n$err',
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
        return _RouteList(response: resp, twoPane: twoPane);
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
              Icons.local_shipping_rounded,
              size: 56,
              color: Theme.of(context).colorScheme.onSurface.withAlpha(80),
            ),
            const SizedBox(height: 16),
            Text(
              'Schiff + Kapital wählen und Routen suchen',
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
          key: const Key('hauling-empty-state'),
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(
              Icons.sentiment_dissatisfied_rounded,
              size: 56,
              color: Theme.of(context).colorScheme.onSurface.withAlpha(80),
            ),
            const SizedBox(height: 16),
            Text(
              'Keine profitablen Routen im Umkreis',
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
// Route list
// ---------------------------------------------------------------------------

class _RouteList extends StatelessWidget {
  const _RouteList({required this.response, required this.twoPane});

  final HaulingResponse response;
  final bool twoPane;

  @override
  Widget build(BuildContext context) {
    final skills = response.skillsApplied;
    return ListView(
      key: const Key('hauling-route-list'),
      padding: const EdgeInsets.all(16),
      children: [
        Text(
          'Gefundene Routen',
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
        const SizedBox(height: 12),
        for (final route in response.routes)
          _RouteCard(route: route, twoPane: twoPane),
      ],
    );
  }
}

class _RouteCard extends StatelessWidget {
  const _RouteCard({required this.route, required this.twoPane});

  final HaulingRoute route;
  final bool twoPane;

  void _openDetail(BuildContext context) {
    if (twoPane) {
      showDialog<void>(
        context: context,
        builder: (_) => Dialog(
          child: ConstrainedBox(
            constraints: const BoxConstraints(maxWidth: 560, maxHeight: 720),
            child: HaulingRouteDetail(route: route),
          ),
        ),
      );
    } else {
      Navigator.of(context).push(
        MaterialPageRoute<void>(
          builder: (_) => Scaffold(
            appBar: AppBar(
              title: Text('${route.buySystemName} → ${route.sellSystemName}'),
            ),
            body: HaulingRouteDetail(route: route),
          ),
        ),
      );
    }
  }

  @override
  Widget build(BuildContext context) {
    final accent = Theme.of(context).colorScheme.primary;
    return Card(
      margin: const EdgeInsets.only(bottom: 12),
      child: InkWell(
        key: Key('hauling-route-${route.buyStationId}-${route.sellStationId}'),
        onTap: () => _openDetail(context),
        borderRadius: BorderRadius.circular(12),
        child: Padding(
          padding: const EdgeInsets.all(16),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              // ── Route header: buy → sell + sec-risk chip ─────────────────
              Row(
                children: [
                  Expanded(
                    child: Text(
                      '${route.buySystemName} → ${route.sellSystemName}',
                      style: Theme.of(context).textTheme.titleMedium?.copyWith(
                            fontWeight: FontWeight.w700,
                          ),
                      overflow: TextOverflow.ellipsis,
                    ),
                  ),
                  const SizedBox(width: 8),
                  _SecurityRiskChip(risk: route.securityRisk),
                ],
              ),
              const SizedBox(height: 8),

              // ── ISK/h — large accent ─────────────────────────────────────
              Text(
                '${fmtIsk(route.iskPerHour)} ISK/h',
                style: Theme.of(context).textTheme.titleLarge?.copyWith(
                      color: accent,
                      fontWeight: FontWeight.bold,
                    ),
              ),
              const SizedBox(height: 8),

              // ── Meta row: jumps, travel time, total profit ───────────────
              Wrap(
                spacing: 16,
                runSpacing: 4,
                children: [
                  _Meta(
                    icon: Icons.alt_route_rounded,
                    label: '${route.jumps} Sprünge',
                  ),
                  _Meta(
                    icon: Icons.schedule_rounded,
                    label: '${route.travelTimeMin.toStringAsFixed(1)} min',
                  ),
                  _Meta(
                    icon: Icons.payments_rounded,
                    label: '${fmtIsk(route.totalProfit)} ISK',
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
// Security-risk chip (safe=green / caution=amber / danger=red)
// ---------------------------------------------------------------------------

class _SecurityRiskChip extends StatelessWidget {
  const _SecurityRiskChip({required this.risk});

  final String risk;

  @override
  Widget build(BuildContext context) {
    final color = securityRiskColor(risk);
    return Container(
      key: Key('hauling-risk-chip-$risk'),
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
            securityRiskLabel(risk),
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
// Route detail — cargo item table + "Route an EVE übertragen" button
// ---------------------------------------------------------------------------

/// Full detail view for a [HaulingRoute]: a cargo item table plus a button that
/// pushes the route to the in-game autopilot (buy station then sell station).
class HaulingRouteDetail extends ConsumerWidget {
  const HaulingRouteDetail({super.key, required this.route});

  final HaulingRoute route;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final accent = Theme.of(context).colorScheme.primary;
    final muted = Theme.of(context).colorScheme.onSurface.withAlpha(153);

    return SingleChildScrollView(
      padding: const EdgeInsets.all(24),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          // ── Header ─────────────────────────────────────────────────────
          Text(
            'Hauling-Route',
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
                  '${route.buySystemName} → ${route.sellSystemName}',
                  style: Theme.of(context).textTheme.headlineSmall?.copyWith(
                        fontWeight: FontWeight.bold,
                      ),
                ),
              ),
              const SizedBox(width: 8),
              _SecurityRiskChip(risk: route.securityRisk),
            ],
          ),
          const SizedBox(height: 4),
          Text(
            '${route.buyStationName}\n→ ${route.sellStationName}',
            style: Theme.of(context).textTheme.bodySmall?.copyWith(color: muted),
          ),
          const SizedBox(height: 12),

          Text(
            '${fmtIsk(route.iskPerHour)} ISK/h',
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
                icon: Icons.alt_route_rounded,
                label: '${route.jumps} Sprünge',
              ),
              _Meta(
                icon: Icons.schedule_rounded,
                label: '${route.travelTimeMin.toStringAsFixed(1)} min',
              ),
              _Meta(
                icon: Icons.payments_rounded,
                label: 'Gewinn ${fmtIsk(route.totalProfit)}',
              ),
              _Meta(
                icon: Icons.account_balance_wallet_rounded,
                label: 'Kapital ${fmtIsk(route.totalCapital)}',
              ),
              _Meta(
                icon: Icons.inventory_2_rounded,
                label: '${fmtVolume(route.totalVolume)} m³',
              ),
            ],
          ),
          const SizedBox(height: 20),

          // ── Cargo item table ─────────────────────────────────────────────
          Text(
            'Ladung',
            style: Theme.of(context).textTheme.titleMedium,
          ),
          const SizedBox(height: 8),
          SingleChildScrollView(
            scrollDirection: Axis.horizontal,
            child: DataTable(
              key: const Key('hauling-cargo-table'),
              columns: const [
                DataColumn(label: Text('Item')),
                DataColumn(label: Text('Menge'), numeric: true),
                DataColumn(label: Text('Kauf'), numeric: true),
                DataColumn(label: Text('Verkauf'), numeric: true),
                DataColumn(label: Text('Volumen'), numeric: true),
                DataColumn(label: Text('Gewinn'), numeric: true),
                DataColumn(label: Text('Gewinn/m³'), numeric: true),
              ],
              rows: [
                for (final item in route.items)
                  DataRow(
                    cells: [
                      DataCell(Text(item.name)),
                      DataCell(Text(fmtUnits(item.quantity.toDouble()))),
                      DataCell(Text(fmtIsk(item.buyPrice))),
                      DataCell(Text(fmtIsk(item.sellPrice))),
                      DataCell(Text(
                          '${fmtVolume(item.unitVolume * item.quantity)} m³')),
                      DataCell(Text(fmtIsk(item.profit))),
                      DataCell(Text(fmtIsk(item.profitPerM3))),
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

