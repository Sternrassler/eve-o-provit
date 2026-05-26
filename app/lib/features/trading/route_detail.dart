/// RouteDetail — full detail pane for a selected [TradingRoute].
///
/// Shows all relevant route fields. The "Route in EVE setzen" button pushes
/// buy and sell waypoints via [TradingApi.setWaypoint] and shows a SnackBar
/// with the outcome.
library;

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../api/trading_models.dart';
import '../trading/providers.dart';
import 'route_card.dart' show formatIskPerHour;

/// Full detail view for a [TradingRoute].
class RouteDetail extends ConsumerWidget {
  const RouteDetail({super.key, required this.route});

  final TradingRoute route;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final colorScheme = Theme.of(context).colorScheme;
    final accent = colorScheme.primary;
    final muted = colorScheme.onSurface.withAlpha(153);

    return SingleChildScrollView(
      padding: const EdgeInsets.all(24),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          // ── Header label ───────────────────────────────────────────────────
          Text(
            'Handelsroute',
            style: Theme.of(context).textTheme.labelLarge?.copyWith(
                  color: muted,
                  letterSpacing: 1.1,
                ),
          ),
          const SizedBox(height: 4),

          // ── Item name ──────────────────────────────────────────────────────
          Text(
            route.itemName,
            style: Theme.of(context).textTheme.headlineSmall?.copyWith(
                  fontWeight: FontWeight.bold,
                ),
          ),
          const SizedBox(height: 8),

          // ── ISK/h — large accent ───────────────────────────────────────────
          Text(
            formatIskPerHour(route.iskPerHour),
            style: Theme.of(context).textTheme.headlineMedium?.copyWith(
                  color: accent,
                  fontWeight: FontWeight.bold,
                ),
          ),
          const SizedBox(height: 20),

          // ── Route meta block ───────────────────────────────────────────────
          _MetaCard(route: route),
          const SizedBox(height: 24),

          // ── Action button ──────────────────────────────────────────────────
          SizedBox(
            width: double.infinity,
            child: FilledButton.icon(
              onPressed: () => _setWaypoint(context, ref),
              icon: const Icon(Icons.navigation_rounded),
              label: const Text('Route in EVE setzen'),
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
      // Clear existing route; set buy system first, then sell system.
      await api.setWaypoint(
        destinationId: route.buySystemId,
        clearOtherWaypoints: true,
        addToBeginning: false,
      );
      await api.setWaypoint(
        destinationId: route.sellSystemId,
        clearOtherWaypoints: false,
        addToBeginning: false,
      );
      if (context.mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
            content: Text(
              'Route gesetzt: ${route.buySystemName} → ${route.sellSystemName}',
            ),
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
// _MetaCard — helper that renders all the numeric detail rows
// ---------------------------------------------------------------------------

class _MetaCard extends StatelessWidget {
  const _MetaCard({required this.route});

  final TradingRoute route;

  @override
  Widget build(BuildContext context) {
    final buyPrice = _iskStr(route.buyPrice);
    final sellPrice = _iskStr(route.sellPrice);

    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            // ── Buy side ────────────────────────────────────────────────────
            _SectionLabel('Kaufen'),
            _Row('System', route.buySystemName),
            _Row('Station', route.buyStationName, muted: true),
            _Row('Preis', buyPrice),
            const SizedBox(height: 12),

            // ── Sell side ───────────────────────────────────────────────────
            _SectionLabel('Verkaufen'),
            _Row('System', route.sellSystemName),
            _Row('Station', route.sellStationName, muted: true),
            _Row('Preis', sellPrice),
            const SizedBox(height: 12),

            // ── Financials ──────────────────────────────────────────────────
            _SectionLabel('Kennzahlen'),
            _Row('Netto-Gewinn', _iskStr(route.netProfit)),
            _Row('Spread', '${route.spreadPercent.toStringAsFixed(2)} %'),
            _Row('Menge', '${route.quantity} Stk.'),
            _Row('Volumen', '${route.itemVolume.toStringAsFixed(2)} m³/Stk.'),
            _Row('Sprünge', '${route.jumps}'),
            _Row('Gebühren', _iskStr(route.totalFees)),
            _Row('Investition', _iskStr(route.totalInvestment)),
            _Row('Cargo-Auslastung',
                '${route.cargoUtilization.toStringAsFixed(1)} %'),
          ],
        ),
      ),
    );
  }
}

class _SectionLabel extends StatelessWidget {
  const _SectionLabel(this.label);
  final String label;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.only(bottom: 6),
      child: Text(
        label,
        style: Theme.of(context).textTheme.labelMedium?.copyWith(
              color: Theme.of(context).colorScheme.primary,
              fontWeight: FontWeight.w700,
              letterSpacing: 0.8,
            ),
      ),
    );
  }
}

class _Row extends StatelessWidget {
  const _Row(this.label, this.value, {this.muted = false});
  final String label;
  final String value;
  final bool muted;

  @override
  Widget build(BuildContext context) {
    final textTheme = Theme.of(context).textTheme;
    final onSurface = Theme.of(context).colorScheme.onSurface;
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 2),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          SizedBox(
            width: 140,
            child: Text(
              label,
              style: textTheme.bodySmall?.copyWith(
                color: onSurface.withAlpha(muted ? 120 : 178),
              ),
            ),
          ),
          Expanded(
            child: Text(
              value,
              style: textTheme.bodySmall?.copyWith(
                fontWeight: FontWeight.w500,
                color: muted ? onSurface.withAlpha(153) : null,
              ),
            ),
          ),
        ],
      ),
    );
  }
}

String _iskStr(double isk) {
  // Show full value with thousand separators (simplified).
  if (isk >= 1_000_000_000) {
    return '${(isk / 1_000_000_000).toStringAsFixed(2)} B ISK';
  } else if (isk >= 1_000_000) {
    return '${(isk / 1_000_000).toStringAsFixed(2)} M ISK';
  } else if (isk >= 1_000) {
    return '${(isk / 1_000).toStringAsFixed(2)} K ISK';
  }
  return '${isk.toStringAsFixed(2)} ISK';
}
