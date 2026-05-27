/// ShipFittingCard — shows effective cargo m³, warp speed AU/s, align time s,
/// and bonuses for the selected ship/character combination.
///
/// Mirrors the web ShipFittingCard (ShipFittingCard.tsx).  Data is fetched
/// via [fittingProvider] (FutureProvider.family keyed on (characterId, shipTypeId)).
library;

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'providers.dart';

// ---------------------------------------------------------------------------
// ShipFittingCard
// ---------------------------------------------------------------------------

/// Card that displays fitting data for [characterId] + [shipTypeId].
///
/// Shows loading skeleton, error, empty state, and the full metrics.
class ShipFittingCard extends ConsumerWidget {
  const ShipFittingCard({
    super.key,
    required this.characterId,
    required this.shipTypeId,
  });

  final int characterId;
  final int shipTypeId;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final fittingAsync =
        ref.watch(fittingProvider((characterId, shipTypeId)));

    return Card(
      margin: EdgeInsets.zero,
      child: fittingAsync.when(
        loading: () => _FittingLoading(),
        error: (err, _) => _FittingError(message: err.toString()),
        data: (fitting) {
          return Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            mainAxisSize: MainAxisSize.min,
            children: [
              // ── Header ─────────────────────────────────────────────────
              ListTile(
                dense: true,
                title: const Text(
                  'Schiff-Fitting',
                  style: TextStyle(fontWeight: FontWeight.w600),
                ),
                subtitle: Text(
                  '${fitting.fittedModules} Module gefittet'
                  '${fitting.cached ? ' (Cache)' : ''}',
                  style: const TextStyle(fontSize: 12),
                ),
              ),
              const Divider(height: 1),
              // ── Metrics ────────────────────────────────────────────────
              Padding(
                padding: const EdgeInsets.all(12),
                child: Column(
                  children: [
                    _MetricRow(
                      label: 'Cargo',
                      value:
                          '${_fmt1(fitting.effectiveCargoM3)} m³',
                      valueColor: _improvedColor(
                        context,
                        effective: fitting.effectiveCargoM3,
                        base: fitting.baseCargoHoldM3,
                        higherIsBetter: true,
                      ),
                      subtitle:
                          'Basis ${_fmt1(fitting.baseCargoHoldM3)} m³',
                    ),
                    const SizedBox(height: 8),
                    _MetricRow(
                      label: 'Warp Speed',
                      value: '${_fmt2(fitting.warpSpeedAuS)} AU/s',
                      valueColor: _improvedColor(
                        context,
                        effective: fitting.warpSpeedAuS,
                        base: fitting.baseWarpSpeedAuS,
                        higherIsBetter: true,
                      ),
                      subtitle:
                          'Basis ${_fmt2(fitting.baseWarpSpeedAuS)} AU/s',
                    ),
                    const SizedBox(height: 8),
                    _MetricRow(
                      label: 'Agility',
                      value: '${_fmt2(fitting.alignTimeSeconds)} s',
                    ),
                    // ── Bonuses detail ────────────────────────────────────
                    if (fitting.bonuses.cargoBonusM3 != 0 ||
                        fitting.bonuses.modulesBonusM3 != 0 ||
                        fitting.bonuses.skillsBonusM3 != 0) ...[
                      const Divider(height: 16),
                      const Align(
                        alignment: Alignment.centerLeft,
                        child: Text(
                          'Boni',
                          style: TextStyle(
                            fontSize: 12,
                            fontWeight: FontWeight.w600,
                          ),
                        ),
                      ),
                      const SizedBox(height: 4),
                      if (fitting.bonuses.skillsBonusM3 != 0)
                        _BonusLine(
                          label: 'Skills',
                          value:
                              '+${_fmt1(fitting.bonuses.skillsBonusM3)} m³'
                              ' (${_fmt1(fitting.bonuses.skillsBonusPct)}%)',
                        ),
                      if (fitting.bonuses.modulesBonusM3 != 0)
                        _BonusLine(
                          label: 'Module',
                          value:
                              '+${_fmt1(fitting.bonuses.modulesBonusM3)} m³',
                        ),
                    ],
                  ],
                ),
              ),
            ],
          );
        },
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Loading / error states
// ---------------------------------------------------------------------------

class _FittingLoading extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    return const Padding(
      padding: EdgeInsets.all(16),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        mainAxisSize: MainAxisSize.min,
        children: [
          Text(
            'Schiff-Fitting',
            style: TextStyle(fontWeight: FontWeight.w600),
          ),
          SizedBox(height: 8),
          LinearProgressIndicator(),
          SizedBox(height: 4),
          Text(
            'Lade Fitting-Daten…',
            style: TextStyle(fontSize: 12),
          ),
        ],
      ),
    );
  }
}

class _FittingError extends StatelessWidget {
  const _FittingError({required this.message});
  final String message;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.all(16),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        mainAxisSize: MainAxisSize.min,
        children: [
          const Text(
            'Schiff-Fitting',
            style: TextStyle(fontWeight: FontWeight.w600),
          ),
          const SizedBox(height: 8),
          Row(
            children: [
              Icon(
                Icons.info_outline_rounded,
                size: 16,
                color: Theme.of(context).colorScheme.error,
              ),
              const SizedBox(width: 8),
              Expanded(
                child: Text(
                  'Fitting nicht verfügbar',
                  style: TextStyle(
                    fontSize: 12,
                    color: Theme.of(context).colorScheme.error,
                  ),
                ),
              ),
            ],
          ),
        ],
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Small display helpers
// ---------------------------------------------------------------------------

class _MetricRow extends StatelessWidget {
  const _MetricRow({
    required this.label,
    required this.value,
    this.subtitle,
    this.valueColor,
  });

  final String label;
  final String value;
  final String? subtitle;
  final Color? valueColor;

  @override
  Widget build(BuildContext context) {
    return Row(
      children: [
        Expanded(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                label,
                style: TextStyle(
                  fontSize: 11,
                  color: Theme.of(context)
                      .colorScheme
                      .onSurface
                      .withAlpha(153),
                ),
              ),
              if (subtitle != null)
                Text(
                  subtitle!,
                  style: TextStyle(
                    fontSize: 10,
                    color: Theme.of(context)
                        .colorScheme
                        .onSurface
                        .withAlpha(100),
                  ),
                ),
            ],
          ),
        ),
        const SizedBox(width: 8),
        Flexible(
          child: Text(
            value,
            textAlign: TextAlign.end,
            overflow: TextOverflow.ellipsis,
            style: TextStyle(
              fontSize: 14,
              fontWeight: FontWeight.w600,
              color: valueColor,
            ),
          ),
        ),
      ],
    );
  }
}

class _BonusLine extends StatelessWidget {
  const _BonusLine({required this.label, required this.value});
  final String label;
  final String value;

  @override
  Widget build(BuildContext context) {
    return Row(
      children: [
        Expanded(
          child: Text(
            label,
            style: TextStyle(
              fontSize: 11,
              color:
                  Theme.of(context).colorScheme.onSurface.withAlpha(153),
            ),
          ),
        ),
        const SizedBox(width: 8),
        Flexible(
          child: Text(
            value,
            textAlign: TextAlign.end,
            overflow: TextOverflow.ellipsis,
            style: TextStyle(
              fontSize: 11,
              color: Theme.of(context).colorScheme.primary,
            ),
          ),
        ),
      ],
    );
  }
}

// ---------------------------------------------------------------------------
// Utility
// ---------------------------------------------------------------------------

Color? _improvedColor(
  BuildContext context, {
  required double effective,
  required double base,
  required bool higherIsBetter,
}) {
  if (base <= 0) return null;
  final improved =
      higherIsBetter ? effective > base : effective < base;
  final worse = higherIsBetter ? effective < base : effective > base;
  if (improved) return const Color(0xFF4CAF50); // green
  if (worse) return const Color(0xFFF44336); // red
  return null;
}

String _fmt1(double v) => v.toStringAsFixed(1);
String _fmt2(double v) => v.toStringAsFixed(2);
