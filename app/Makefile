# Makefile — eve-o-provit Flutter app
# Run `make help` to list all targets.
.PHONY: help test e2e-test analyze build-debug

# ── Help ───────────────────────────────────────────────────────────────────────
help: ## Show this help
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) \
		| awk 'BEGIN {FS = ":.*?## "}; {printf "  %-18s %s\n", $$1, $$2}'

# ── Tests ──────────────────────────────────────────────────────────────────────
test: ## Run ALL tests (unit + widget + E2E) via flutter test — no device needed
	flutter test

e2e-test: ## Run only the E2E / integration-style flow tests (deviceless)
	flutter test test/e2e/

# ── Code quality ───────────────────────────────────────────────────────────────
analyze: ## Run flutter analyze (must be clean for CI)
	flutter analyze

# ── Build ──────────────────────────────────────────────────────────────────────
build-debug: ## Build a debug APK (sanity-check compilation)
	flutter build apk --debug

# ── Real-backend on-device validation (manual) ─────────────────────────────────
#
# The automated E2E tests mock all backend/auth layers. To validate the real SSO
# login flow and live route calculation on a physical device or emulator, run:
#
#   flutter run \
#     --dart-define=API_BASE_URL=https://your-backend.example.com \
#     --dart-define=EVE_CLIENT_ID=your_client_id
#
# This requires:
#   • A running eve-o-provit backend (see ../backend/)
#   • A registered EVE developer application (client ID + redirect URI)
#   • A connected Android/iOS device or running emulator
