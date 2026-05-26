import 'package:flutter/foundation.dart';

/// Immutable model representing an authenticated EVE Online character.
@immutable
class Character {
  const Character({
    required this.id,
    required this.name,
    required this.portraitUrl,
    required this.scopes,
  });

  final int id;
  final String name;
  final String portraitUrl;
  final List<String> scopes;

  factory Character.fromJson(Map<String, dynamic> json) {
    final rawScopes = json['scopes'];
    final List<String> parsedScopes;
    if (rawScopes is List) {
      parsedScopes = rawScopes.cast<String>();
    } else if (rawScopes is String && rawScopes.isNotEmpty) {
      parsedScopes = rawScopes.split(' ');
    } else {
      parsedScopes = const [];
    }

    return Character(
      id: (json['character_id'] as num).toInt(),
      name: json['character_name'] as String,
      portraitUrl: json['portrait_url'] as String? ?? '',
      scopes: parsedScopes,
    );
  }

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is Character && runtimeType == other.runtimeType && id == other.id;

  @override
  int get hashCode => id.hashCode;

  @override
  String toString() => 'Character(id: $id, name: $name)';
}
