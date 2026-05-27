/// RouteCard — a tappable card representing a single [TradingRoute].
///
/// Design direction C: whitespace, soft surfaces, ONE blue accent for ISK/h,
/// neutral muted text for meta-info. Card radius/elevation from theme.
library;

import 'package:flutter/material.dart';

import '../../api/trading_models.dart';

/// Formats an ISK value (absolute) into a compact human-readable string.
///
/// Examples:
///   195_000_000 → "195,0M ISK/h"
///   5_000_000   →   "5,0M ISK/h"
///     500_000   → "500,0K ISK/h"
String formatIskPerHour(double isk) {
  if (isk >= 1_000_000_000) {
    final b = isk / 1_000_000_000;
    return '${b.toStringAsFixed(1).replaceAll('.', ',')}B ISK/h';
  } else if (isk >= 1_000_000) {
    final m = isk / 1_000_000;
    return '${m.toStringAsFixed(1).replaceAll('.', ',')}M ISK/h';
  } else if (isk >= 1_000) {
    final k = isk / 1_000;
    return '${k.toStringAsFixed(1).replaceAll('.', ',')}K ISK/h';
  }
  return '${isk.toStringAsFixed(0)} ISK/h';
}

/// Formats a raw ISK amount (not per hour) to a compact string without suffix.
String _formatIsk(double isk) {
  if (isk >= 1_000_000_000) {
    return '${(isk / 1_000_000_000).toStringAsFixed(1).replaceAll('.', ',')}B ISK';
  } else if (isk >= 1_000_000) {
    return '${(isk / 1_000_000).toStringAsFixed(1).replaceAll('.', ',')}M ISK';
  } else if (isk >= 1_000) {
    return '${(isk / 1_000).toStringAsFixed(1).replaceAll('.', ',')}K ISK';
  }
  return '${isk.toStringAsFixed(0)} ISK';
}

/// Returns a colour representing the route's minimum security status.
Color _secColor(double sec) {
  if (sec >= 0.5) return const Color(0xFF4CAF50); // green — high-sec
  if (sec >= 0.0) return const Color(0xFFFF9800); // orange — low-sec
  return const Color(0xFFF44336); // red — null-sec
}

/// Human-readable sec-zone label for the route's minimum security status.
String _secLabel(double sec) {
  if (sec >= 0.5) return 'Hi';
  if (sec >= 0.0) return 'Low';
  return 'Null';
}

/// A card displaying summary information for a [TradingRoute].
///
/// Shows item name, security zone chip, accent-coloured ISK/h, and a muted
/// meta line (spread · profit · buy→sell). Highlights the border when
/// [selected]. Calls [onTap] when tapped.
class RouteCard extends StatelessWidget {
  const RouteCard({
    super.key,
    required this.route,
    required this.selected,
    required this.onTap,
  });

  final TradingRoute route;
  final bool selected;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final colorScheme = Theme.of(context).colorScheme;
    final accent = colorScheme.primary;
    final muted = colorScheme.onSurface.withAlpha(153); // ~60% opacity

    final secColor = _secColor(route.minRouteSecurityStatus);
    final secLabel = _secLabel(route.minRouteSecurityStatus);

    return Card(
      // Apply an accent outline when selected; no extra border otherwise.
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(10),
        side: selected
            ? BorderSide(color: accent, width: 2)
            : BorderSide.none,
      ),
      clipBehavior: Clip.antiAlias,
      child: InkWell(
        onTap: onTap,
        child: Padding(
          padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 14),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            mainAxisSize: MainAxisSize.min,
            children: [
              // ── Top row: item name + sec-zone chip ──────────────────────────
              Row(
                crossAxisAlignment: CrossAxisAlignment.center,
                children: [
                  Expanded(
                    child: Text(
                      route.itemName,
                      style: Theme.of(context).textTheme.titleMedium?.copyWith(
                            fontWeight: FontWeight.bold,
                          ),
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                    ),
                  ),
                  const SizedBox(width: 8),
                  // Security-zone chip
                  Container(
                    padding:
                        const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
                    decoration: BoxDecoration(
                      color: secColor.withAlpha(30),
                      borderRadius: BorderRadius.circular(20),
                      border: Border.all(color: secColor, width: 1),
                    ),
                    child: Text(
                      secLabel,
                      style: TextStyle(
                        color: secColor,
                        fontSize: 11,
                        fontWeight: FontWeight.w600,
                      ),
                    ),
                  ),
                ],
              ),
              const SizedBox(height: 8),

              // ── ISK/h — large, accent-coloured ──────────────────────────────
              Text(
                formatIskPerHour(route.iskPerHour),
                style: Theme.of(context).textTheme.titleLarge?.copyWith(
                      color: accent,
                      fontWeight: FontWeight.bold,
                    ),
              ),
              const SizedBox(height: 6),

              // ── Muted meta line: spread · profit · route ─────────────────────
              Text(
                '${route.spreadPercent.toStringAsFixed(1)}% spread'
                ' · ${_formatIsk(route.netProfit)}'
                ' · ${route.buySystemName}→${route.sellSystemName}',
                style: TextStyle(color: muted, fontSize: 12),
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
              ),
            ],
          ),
        ),
      ),
    );
  }
}
