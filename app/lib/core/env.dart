class Env {
  static const apiBaseUrl = String.fromEnvironment(
    'API_BASE_URL',
    defaultValue: 'http://10.0.2.2:9001',
  );
  static const eveClientId = String.fromEnvironment('EVE_CLIENT_ID');
  static const redirectUri = 'eveauth-eveoprovit://callback';
  static const scopes = [
    'esi-location.read_location.v1',
    'esi-location.read_ship_type.v1',
    'esi-skills.read_skills.v1',
    'esi-clones.read_clones.v1',
    'esi-assets.read_assets.v1',
    'esi-ui.write_waypoint.v1',
  ];
}
