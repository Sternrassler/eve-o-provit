import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'token_store.dart';

/// Shared, single source of truth for the [TokenStore] instance.
///
/// Both [authRepositoryProvider] (auth_controller.dart) and [dioProvider]
/// (api/dio_client.dart) read this provider so they operate on the *same*
/// secure-storage-backed store.  Lives in this neutral file (no dependency on
/// either consumer) so importing it never introduces a circular import.
///
/// Override in tests to inject an in-memory / fake [TokenStore].
final tokenStoreProvider = Provider<TokenStore>((ref) => TokenStore());
