/// Shared security-risk → colour / label mapping for the trading screens
/// (safe=green, caution=amber, danger=red). Previously duplicated in the
/// hauling and sell-assets screens.
library;

import 'package:flutter/material.dart';

const Color kSafeColor = Color(0xFF66BB6A);
const Color kCautionColor = Color(0xFFFF9800);
const Color kDangerColor = Color(0xFFF44336);

/// Maps a backend `security_risk` value to its chip colour.
Color securityRiskColor(String risk) {
  switch (risk) {
    case 'danger':
      return kDangerColor;
    case 'caution':
      return kCautionColor;
    case 'safe':
    default:
      return kSafeColor;
  }
}

/// Maps a backend `security_risk` value to a German chip label.
String securityRiskLabel(String risk) {
  switch (risk) {
    case 'danger':
      return 'Gefahr';
    case 'caution':
      return 'Vorsicht';
    case 'safe':
    default:
      return 'Sicher';
  }
}
