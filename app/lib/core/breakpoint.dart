/// Single device-tuned breakpoint: Galaxy Tab portrait ~800dp (single-pane),
/// landscape ~1280dp (two-pane). 840 sits cleanly between the two.
const double kTwoPaneBreakpoint = 840;

bool isTwoPane(double widthDp) => widthDp >= kTwoPaneBreakpoint;
