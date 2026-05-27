/// RouteList — a scrollable list of [RouteCard]s driven by [routesProvider].
///
/// Handles loading, empty-state, and populated list. Tapping a card updates
/// [selectedRouteProvider] and optionally triggers [onRouteTap] for
/// single-pane navigation.
library;

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../api/trading_models.dart';
import '../trading/providers.dart';
import 'route_card.dart';

/// A list of [RouteCard] widgets driven by [routesProvider].
///
/// [onRouteTap] is called with the tapped [TradingRoute] after updating
/// [selectedRouteProvider]. In two-pane mode the caller ignores this callback;
/// in single-pane mode it uses it to push a detail page.
class RouteList extends ConsumerWidget {
  const RouteList({super.key, this.onRouteTap});

  /// Optional callback for single-pane navigation. Receives the tapped route.
  final void Function(TradingRoute route)? onRouteTap;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final routesAsync = ref.watch(routesProvider);
    final selectedRoute = ref.watch(selectedRouteProvider);
    // Routes filtered CLIENT-SIDE by the active sec-zone / threshold toggles
    // (the backend has no security-zone request field — see providers.dart).
    final filteredRoutes = ref.watch(filteredRoutesProvider);

    return routesAsync.when(
      loading: () => const Center(child: CircularProgressIndicator()),
      error: (err, _) => Center(
        child: Padding(
          padding: const EdgeInsets.all(24),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              Icon(
                Icons.error_outline,
                size: 48,
                color: Theme.of(context).colorScheme.error,
              ),
              const SizedBox(height: 12),
              Text(
                'Fehler beim Laden der Routen',
                style: Theme.of(context).textTheme.titleMedium,
                textAlign: TextAlign.center,
              ),
              const SizedBox(height: 4),
              Text(
                err.toString(),
                style: Theme.of(context).textTheme.bodySmall,
                textAlign: TextAlign.center,
              ),
            ],
          ),
        ),
      ),
      data: (response) {
        // Empty when: no calculation yet, backend returned 0 routes, or all
        // routes were filtered out client-side by the sec-zone toggles.
        if (response == null || filteredRoutes.isEmpty) {
          // Distinguish "filtered everything out" from "backend had nothing".
          final filteredOut =
              response != null && response.routes.isNotEmpty;
          final message = filteredOut
              ? 'Keine Routen für die gewählten Sicherheitszonen'
              : 'Keine Routen gefunden';

          return Center(
            child: Padding(
              padding: const EdgeInsets.all(32),
              child: Column(
                mainAxisSize: MainAxisSize.min,
                children: [
                  Icon(
                    Icons.search_off_rounded,
                    size: 56,
                    color: Theme.of(context)
                        .colorScheme
                        .onSurface
                        .withAlpha(100),
                  ),
                  const SizedBox(height: 16),
                  Text(
                    message,
                    style: Theme.of(context).textTheme.titleMedium?.copyWith(
                          color: Theme.of(context)
                              .colorScheme
                              .onSurface
                              .withAlpha(153),
                        ),
                    textAlign: TextAlign.center,
                  ),
                  if (!filteredOut && response?.warning != null) ...[
                    const SizedBox(height: 8),
                    Text(
                      response!.warning!,
                      style: Theme.of(context).textTheme.bodySmall?.copyWith(
                            color: Theme.of(context)
                                .colorScheme
                                .onSurface
                                .withAlpha(120),
                          ),
                      textAlign: TextAlign.center,
                    ),
                  ],
                ],
              ),
            ),
          );
        }

        final routes = filteredRoutes;
        return ListView.separated(
          padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 12),
          itemCount: routes.length,
          separatorBuilder: (context, index) => const SizedBox(height: 6),
          itemBuilder: (context, index) {
            final route = routes[index];
            final isSelected =
                selectedRoute?.itemTypeId == route.itemTypeId &&
                    selectedRoute?.buyStationId == route.buyStationId &&
                    selectedRoute?.sellStationId == route.sellStationId;

            return RouteCard(
              route: route,
              selected: isSelected,
              onTap: () {
                ref.read(selectedRouteProvider.notifier).select(route);
                onRouteTap?.call(route);
              },
            );
          },
        );
      },
    );
  }
}
