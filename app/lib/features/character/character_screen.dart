/// CharacterScreen — shows portrait, active ship, fitting summary and skills.
///
/// Single-pane, Card-based layout matching the direction-C card style used
/// throughout the app.
library;

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'character_models.dart';
import 'providers.dart';

// ---------------------------------------------------------------------------
// Public entry point
// ---------------------------------------------------------------------------

/// Screen showing authenticated character information.
///
/// Watches [characterProvider] and [activeShipProvider]; derives the character
/// ID from the character data to drive [skillsProvider].
class CharacterScreen extends ConsumerWidget {
  const CharacterScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final characterAsync = ref.watch(characterProvider);

    return Scaffold(
      appBar: AppBar(
        title: const Text('Character'),
        centerTitle: false,
        elevation: 0,
        scrolledUnderElevation: 1,
      ),
      body: characterAsync.when(
        loading: () => const Center(child: CircularProgressIndicator()),
        error: (err, _) => _ErrorView(message: err.toString()),
        data: (character) => _CharacterContent(character: character),
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Main content — loaded after character data arrives
// ---------------------------------------------------------------------------

class _CharacterContent extends ConsumerWidget {
  const _CharacterContent({required this.character});

  final CharacterInfo character;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final activeShipAsync = ref.watch(activeShipProvider);
    final skillsAsync = ref.watch(skillsProvider(character.characterId));

    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        // ── Character header ────────────────────────────────────────────────
        _CharacterHeader(character: character),
        const SizedBox(height: 16),

        // ── Active ship card ────────────────────────────────────────────────
        activeShipAsync.when(
          loading: () => const _LoadingCard(label: 'Aktives Schiff'),
          error: (err, _) =>
              _ErrorCard(label: 'Aktives Schiff', message: err.toString()),
          data: (ship) => _ActiveShipCard(ship: ship),
        ),
        const SizedBox(height: 16),

        // ── Skills card ──────────────────────────────────────────────────────
        skillsAsync.when(
          loading: () => const _LoadingCard(label: 'Skills'),
          error: (err, _) =>
              _ErrorCard(label: 'Skills', message: err.toString()),
          data: (skillsData) => _SkillsCard(skillsData: skillsData),
        ),
      ],
    );
  }
}

// ---------------------------------------------------------------------------
// Character header
// ---------------------------------------------------------------------------

class _CharacterHeader extends StatelessWidget {
  const _CharacterHeader({required this.character});

  final CharacterInfo character;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Row(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            // Portrait
            ClipRRect(
              borderRadius: BorderRadius.circular(8),
              child: character.portraitUrl.isNotEmpty
                  ? Image.network(
                      character.portraitUrl,
                      width: 80,
                      height: 80,
                      fit: BoxFit.cover,
                      errorBuilder: (ctx, err, stack) => _PortraitPlaceholder(),
                    )
                  : _PortraitPlaceholder(),
            ),
            const SizedBox(width: 16),
            // Name + ID
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    character.characterName,
                    style: theme.textTheme.titleLarge?.copyWith(
                      fontWeight: FontWeight.bold,
                    ),
                  ),
                  const SizedBox(height: 4),
                  Text(
                    'ID: ${character.characterId}',
                    style: theme.textTheme.bodySmall?.copyWith(
                      color: theme.colorScheme.onSurface.withAlpha(153),
                    ),
                  ),
                  if (character.scopes.isNotEmpty) ...[
                    const SizedBox(height: 8),
                    Text(
                      '${character.scopes.length} Scopes',
                      style: theme.textTheme.bodySmall?.copyWith(
                        color: theme.colorScheme.primary,
                      ),
                    ),
                  ],
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _PortraitPlaceholder extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    return Container(
      width: 80,
      height: 80,
      color: Theme.of(context).colorScheme.surfaceContainerHighest,
      child: const Icon(Icons.person, size: 40),
    );
  }
}

// ---------------------------------------------------------------------------
// Active ship card
// ---------------------------------------------------------------------------

class _ActiveShipCard extends StatelessWidget {
  const _ActiveShipCard({required this.ship});

  final CharacterShip ship;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Icon(
                  Icons.rocket_launch_rounded,
                  size: 18,
                  color: theme.colorScheme.primary,
                ),
                const SizedBox(width: 8),
                Text(
                  'Aktives Schiff',
                  style: theme.textTheme.titleMedium?.copyWith(
                    fontWeight: FontWeight.w600,
                  ),
                ),
              ],
            ),
            const SizedBox(height: 12),
            _InfoRow(label: 'Name', value: ship.shipName),
            const SizedBox(height: 4),
            _InfoRow(label: 'Typ', value: ship.shipTypeName),
            const SizedBox(height: 4),
            _InfoRow(
              label: 'Laderaum',
              value: '${ship.cargoCapacity.toStringAsFixed(1)} m³',
            ),
            const SizedBox(height: 4),
            _InfoRow(
              label: 'Typ-ID',
              value: ship.shipTypeId.toString(),
            ),
          ],
        ),
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Skills card
// ---------------------------------------------------------------------------

/// Shows total SP and the relevant cargo/navigation skills found in the list.
///
/// The interesting skill IDs for traders:
///   - 3456 = Trade
///   - 3443 = Navigation
///   - 3428 = Warp Drive Operation
///   - 3327 = Gallente Hauler (example)
///   - 3340 = Spaceship Command
///   - 3463 = Evasive Maneuvering
class _SkillsCard extends StatelessWidget {
  const _SkillsCard({required this.skillsData});

  final CharacterSkillsData skillsData;

  // Highlight skill IDs relevant to hauling / trading
  static const _highlightedSkillIds = <int>{
    3340, // Spaceship Command
    3428, // Warp Drive Operation
    3463, // Evasive Maneuvering
    3443, // Navigation
  };

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    final highlighted = skillsData.skills
        .where((s) => _highlightedSkillIds.contains(s.skillId))
        .toList()
      ..sort((a, b) => a.skillId.compareTo(b.skillId));

    final totalSpStr = _formatSP(skillsData.totalSp);

    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Icon(
                  Icons.psychology_rounded,
                  size: 18,
                  color: theme.colorScheme.primary,
                ),
                const SizedBox(width: 8),
                Text(
                  'Skills',
                  style: theme.textTheme.titleMedium?.copyWith(
                    fontWeight: FontWeight.w600,
                  ),
                ),
              ],
            ),
            const SizedBox(height: 8),
            _InfoRow(label: 'Gesamt-SP', value: totalSpStr),
            _InfoRow(
              label: 'Trainierte Skills',
              value: skillsData.skills.length.toString(),
            ),
            if (highlighted.isNotEmpty) ...[
              const Divider(height: 20),
              Text(
                'Navigations- & Cargo-Skills',
                style: theme.textTheme.labelMedium?.copyWith(
                  color: theme.colorScheme.onSurface.withAlpha(153),
                ),
              ),
              const SizedBox(height: 8),
              ...highlighted.map(
                (skill) => Padding(
                  padding: const EdgeInsets.only(bottom: 4),
                  child: Row(
                    children: [
                      Expanded(
                        child: Text(
                          'Skill ${skill.skillId}',
                          style: theme.textTheme.bodySmall,
                        ),
                      ),
                      _SkillLevelDots(level: skill.activeSkillLevel),
                    ],
                  ),
                ),
              ),
            ],
          ],
        ),
      ),
    );
  }

  static String _formatSP(int sp) {
    if (sp >= 1000000) {
      return '${(sp / 1000000).toStringAsFixed(1)}M SP';
    }
    if (sp >= 1000) {
      return '${(sp / 1000).toStringAsFixed(0)}K SP';
    }
    return '$sp SP';
  }
}

/// Five dots indicating a skill level (0–5).
class _SkillLevelDots extends StatelessWidget {
  const _SkillLevelDots({required this.level});

  final int level;

  @override
  Widget build(BuildContext context) {
    final active = Theme.of(context).colorScheme.primary;
    final inactive = Theme.of(context).colorScheme.outline.withAlpha(80);

    return Row(
      mainAxisSize: MainAxisSize.min,
      children: List.generate(
        5,
        (i) => Container(
          width: 10,
          height: 10,
          margin: const EdgeInsets.only(left: 3),
          decoration: BoxDecoration(
            shape: BoxShape.circle,
            color: i < level ? active : inactive,
          ),
        ),
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Shared helpers
// ---------------------------------------------------------------------------

class _InfoRow extends StatelessWidget {
  const _InfoRow({required this.label, required this.value});

  final String label;
  final String value;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Row(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        SizedBox(
          width: 110,
          child: Text(
            label,
            style: theme.textTheme.bodySmall?.copyWith(
              color: theme.colorScheme.onSurface.withAlpha(153),
            ),
          ),
        ),
        Expanded(
          child: Text(
            value,
            style: theme.textTheme.bodyMedium,
          ),
        ),
      ],
    );
  }
}

class _LoadingCard extends StatelessWidget {
  const _LoadingCard({required this.label});

  final String label;

  @override
  Widget build(BuildContext context) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(24),
        child: Row(
          children: [
            const SizedBox(
              width: 18,
              height: 18,
              child: CircularProgressIndicator(strokeWidth: 2),
            ),
            const SizedBox(width: 12),
            Text(label),
          ],
        ),
      ),
    );
  }
}

class _ErrorCard extends StatelessWidget {
  const _ErrorCard({required this.label, required this.message});

  final String label;
  final String message;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              label,
              style: theme.textTheme.titleSmall,
            ),
            const SizedBox(height: 4),
            Text(
              message,
              style: TextStyle(color: theme.colorScheme.error),
            ),
          ],
        ),
      ),
    );
  }
}

class _ErrorView extends StatelessWidget {
  const _ErrorView({required this.message});

  final String message;

  @override
  Widget build(BuildContext context) {
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(32),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(
              Icons.error_outline_rounded,
              size: 48,
              color: Theme.of(context).colorScheme.error,
            ),
            const SizedBox(height: 16),
            Text(
              'Fehler beim Laden',
              style: Theme.of(context).textTheme.titleMedium,
            ),
            const SizedBox(height: 8),
            Text(
              message,
              textAlign: TextAlign.center,
              style: TextStyle(
                color: Theme.of(context).colorScheme.onSurface.withAlpha(153),
              ),
            ),
          ],
        ),
      ),
    );
  }
}
