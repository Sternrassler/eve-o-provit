# Makefile — eve-o-provit Flutter app
# Run `make help` to list all targets.
.PHONY: help test e2e-test analyze build-debug android-install

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

# ── Release install (real device) ──────────────────────────────────────────────
#
# Builds the release APK with the production dart-defines and installs it on the
# connected device. The mobile client ID is read from ../deployments/.env
# (EVE_MOBILE_CLIENT_ID) so a release APK can never ship with an empty client_id
# (which silently breaks the SSO login). Afterwards the install is verified via
# dumpsys (INTERNET permission present + lastUpdateTime).
#
# Optional: set ANDROID_DEVICE=<serial> when more than one device is attached.
PROD_API_BASE_URL ?= https://eveonline.sternrassler.de
DEPLOY_ENV         := ../deployments/.env
RELEASE_APK        := build/app/outputs/flutter-apk/app-release.apk
ANDROID_PACKAGE    := de.sternrassler.eve_o_provit
ANDROID_DEVICE_ARG := $(if $(ANDROID_DEVICE),-s $(ANDROID_DEVICE),)

android-install: ## Build release APK with prod dart-defines and install it via adb (verified)
	@set -e; \
	test -f "$(DEPLOY_ENV)" || { echo "ERROR: $(DEPLOY_ENV) not found (provides EVE_MOBILE_CLIENT_ID)" >&2; exit 1; }; \
	CLIENT_ID="$$(grep -E '^EVE_MOBILE_CLIENT_ID=' "$(DEPLOY_ENV)" | head -n1 | cut -d= -f2-)"; \
	test -n "$$CLIENT_ID" || { echo "ERROR: EVE_MOBILE_CLIENT_ID missing/empty in $(DEPLOY_ENV)" >&2; exit 1; }; \
	echo "Building release APK (API_BASE_URL=$(PROD_API_BASE_URL), EVE_CLIENT_ID from deployments/.env) ..."; \
	flutter build apk --release \
		--dart-define=API_BASE_URL=$(PROD_API_BASE_URL) \
		--dart-define=EVE_CLIENT_ID="$$CLIENT_ID"; \
	adb $(ANDROID_DEVICE_ARG) install -r "$(RELEASE_APK)"; \
	echo "Verifying installed package ..."; \
	adb $(ANDROID_DEVICE_ARG) shell dumpsys package $(ANDROID_PACKAGE) | grep -E 'android.permission.INTERNET|lastUpdateTime='; \
	echo "OK: $(ANDROID_PACKAGE) installed (INTERNET permission verified)."

# ── Real-backend on-device validation (manual) ─────────────────────────────────
#
# The automated E2E tests mock all backend/auth layers. `make android-install`
# (above) is the canonical way to put a production release build on a device.
# To iterate against a non-production backend, run:
#
#   flutter run \
#     --dart-define=API_BASE_URL=https://your-backend.example.com \
#     --dart-define=EVE_CLIENT_ID=your_client_id
#
# This requires:
#   • A running eve-o-provit backend (see ../backend/)
#   • A registered EVE developer application (client ID + redirect URI)
#   • A connected Android/iOS device or running emulator
