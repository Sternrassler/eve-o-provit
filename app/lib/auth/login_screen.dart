import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'auth_controller.dart';

/// Login screen displayed when the user is not authenticated.
///
/// Shows the app title and a "Login mit EVE" button.  While the login flow is
/// in progress a [CircularProgressIndicator] replaces the button.  Errors are
/// shown in a [SnackBar].
class LoginScreen extends ConsumerWidget {
  const LoginScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final authState = ref.watch(authControllerProvider);

    // Show SnackBar once when an error surfaces.
    ref.listen<AsyncValue<AuthState>>(authControllerProvider, (previous, next) {
      if (next is AsyncError) {
        final messenger = ScaffoldMessenger.of(context);
        messenger.clearSnackBars();
        messenger.showSnackBar(
          SnackBar(
            content: Text('Login fehlgeschlagen: ${next.error}'),
            backgroundColor: Theme.of(context).colorScheme.error,
          ),
        );
      }
    });

    final isLoading = authState.isLoading;

    return Scaffold(
      body: SafeArea(
        child: Center(
          child: Padding(
            padding: const EdgeInsets.symmetric(horizontal: 32),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                Text(
                  'EVE-O Provit',
                  style: Theme.of(context).textTheme.headlineLarge?.copyWith(
                        color: Theme.of(context).colorScheme.primary,
                        fontWeight: FontWeight.bold,
                      ),
                ),
                const SizedBox(height: 8),
                Text(
                  'Trading Profit Optimizer',
                  style: Theme.of(context).textTheme.bodyMedium?.copyWith(
                        color: Theme.of(context)
                            .colorScheme
                            .onSurface
                            .withValues(alpha: 0.6),
                      ),
                ),
                const SizedBox(height: 48),
                if (isLoading)
                  const CircularProgressIndicator()
                else
                  FilledButton.icon(
                    onPressed: () =>
                        ref.read(authControllerProvider.notifier).login(),
                    icon: const Icon(Icons.login),
                    label: const Text('Login mit EVE'),
                    style: FilledButton.styleFrom(
                      minimumSize: const Size(220, 48),
                    ),
                  ),
                if (authState is AsyncError && !isLoading)
                  Padding(
                    padding: const EdgeInsets.only(top: 16),
                    child: Text(
                      'Fehler: ${authState.error}',
                      style: TextStyle(
                        color: Theme.of(context).colorScheme.error,
                      ),
                      textAlign: TextAlign.center,
                    ),
                  ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}
