import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'mining_providers.dart';

/// Above this many full loads, individual pips become unwieldy → counter + bar.
const int _pipCap = 24;

/// Per-ore tally for marking off loads already sold, so the miner keeps track
/// while selling. Sold counts live in [soldLoadsProvider] and reset on every
/// new ranking. Pips for small counts, a counter + progress bar for large ones.
class MiningLoadsTally extends ConsumerWidget {
  const MiningLoadsTally({
    super.key,
    required this.oreTypeId,
    required this.total,
  });

  final int oreTypeId;
  final int total;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    if (total < 1) return const SizedBox.shrink();

    final int sold =
        ((ref.watch(soldLoadsProvider)[oreTypeId] ?? 0).clamp(0, total)).toInt();
    final theme = Theme.of(context);
    final primary = theme.colorScheme.primary;

    void setSold(int n) {
      ref.read(soldLoadsProvider.notifier).setSold(oreTypeId, n.clamp(0, total).toInt());
    }

    return Padding(
      padding: const EdgeInsets.only(top: 4),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text('Verkauft: $sold / $total', style: theme.textTheme.bodySmall),
          const SizedBox(height: 4),
          if (total <= _pipCap)
            Wrap(
              spacing: 4,
              runSpacing: 4,
              children: [
                for (var i = 0; i < total; i++)
                  _Pip(
                    filled: i < sold,
                    color: primary,
                    // Tap fills up to here, or clears from here if already filled.
                    onTap: () => setSold(i < sold ? i : i + 1),
                  ),
              ],
            )
          else
            Row(
              children: [
                _StepButton(label: '−', onTap: () => setSold(sold - 1)),
                const SizedBox(width: 8),
                _StepButton(label: '+', onTap: () => setSold(sold + 1)),
                const SizedBox(width: 12),
                Expanded(
                  child: ClipRRect(
                    borderRadius: BorderRadius.circular(3),
                    child: LinearProgressIndicator(
                      value: total > 0 ? sold / total : 0,
                      minHeight: 6,
                      color: primary,
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

class _Pip extends StatelessWidget {
  const _Pip({required this.filled, required this.color, required this.onTap});

  final bool filled;
  final Color color;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    return InkWell(
      onTap: onTap,
      borderRadius: BorderRadius.circular(4),
      child: Container(
        width: 22,
        height: 22,
        decoration: BoxDecoration(
          borderRadius: BorderRadius.circular(4),
          border: Border.all(color: filled ? color : color.withAlpha(80)),
          color: filled ? color.withAlpha(40) : null,
        ),
        child: filled ? Icon(Icons.check, size: 14, color: color) : null,
      ),
    );
  }
}

class _StepButton extends StatelessWidget {
  const _StepButton({required this.label, required this.onTap});

  final String label;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final color = Theme.of(context).colorScheme.primary;
    return InkWell(
      onTap: onTap,
      borderRadius: BorderRadius.circular(4),
      child: Container(
        width: 28,
        height: 28,
        alignment: Alignment.center,
        decoration: BoxDecoration(
          borderRadius: BorderRadius.circular(4),
          border: Border.all(color: color.withAlpha(120)),
        ),
        child: Text(
          label,
          style: TextStyle(color: color, fontSize: 16, fontWeight: FontWeight.w600),
        ),
      ),
    );
  }
}
