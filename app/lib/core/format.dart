/// Shared number formatters for the trading screens (ISK, unit counts, volume).
/// Previously duplicated per screen — keep a single source so all masks format
/// identically.
library;

/// Formats an ISK amount with a B/M/K suffix (2 decimals for B/M, 1 for K).
String fmtIsk(double v) {
  if (v >= 1e9) return '${(v / 1e9).toStringAsFixed(2)}B';
  if (v >= 1e6) return '${(v / 1e6).toStringAsFixed(2)}M';
  if (v >= 1e3) return '${(v / 1e3).toStringAsFixed(1)}K';
  return v.toStringAsFixed(2);
}

/// Formats a unit count with a M/K suffix (1 decimal), whole numbers below 1000.
String fmtUnits(double v) {
  if (v >= 1e6) return '${(v / 1e6).toStringAsFixed(1)}M';
  if (v >= 1e3) return '${(v / 1e3).toStringAsFixed(1)}K';
  return v.toStringAsFixed(0);
}

/// Formats a volume (m³) with a B/M/K suffix (1 decimal), whole below 1000.
String fmtVolume(double v) {
  if (v >= 1e9) return '${(v / 1e9).toStringAsFixed(1)}B';
  if (v >= 1e6) return '${(v / 1e6).toStringAsFixed(1)}M';
  if (v >= 1e3) return '${(v / 1e3).toStringAsFixed(1)}K';
  return v.toStringAsFixed(0);
}
