import 'package:flutter_test/flutter_test.dart';
import 'package:eve_o_provit/core/breakpoint.dart';

void main() {
  test('two-pane at/above 840dp, single-pane below', () {
    expect(isTwoPane(800), isFalse);
    expect(isTwoPane(840), isTrue);
    expect(isTwoPane(1280), isTrue);
  });
}
