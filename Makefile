APP_NAME := PicFetch
PACKAGE_ID := io.github.frathe.picfetch
BIN_NAME := picfetch
ICON     := assets/appIcon.png
BIN_DIR  := bin
WIN_ARCHES := amd64 arm64
LINUX_ARCHES := amd64 arm64

RELEASE_BRANCH := main

# Module path; goimports -local puts these imports in their own group after
# third-party packages (stdlib / fyne.io+others / github.com/frathe/picfetch).
GOIMPORTS_LOCAL := github.com/frathe/picfetch
# ubuntu-latest + race + Fyne's software renderer: internal/ui is ~10 minutes.
# go test's default 10m per-package timeout is no longer enough.
TEST_TIMEOUT := 30m
TEST_IMAGE := ubuntu:24.04
TEST_CONTAINER_LABEL := io.github.frathe.picfetch.test=true
TEST_RACE :=
TEST_RACE_FLAGS := -race -count=1 -timeout $(TEST_TIMEOUT)
TEST_LOCALE := en_US.UTF-8
TEST_SHARD_MANIFEST := .github/testshards/internal-ui.tsv
TEST_SHARD_PACKAGE := ./internal/ui
TEST_PARTITION :=
TEST_CAPTURE ?= /tmp/picfetch-test-$(TEST_PARTITION).json

.PHONY: all build build-linux-all run fmt fmt-check vet test update-test-image enter-test-container test-native test-race test-race-direct test-race-non-ui-direct test-race-ui-direct verify golden tidy clean package-mac package-windows package-windows-store package-windows-debug package-linux package-linux-debug build-all install-tools install-linux-tools security security-govulncheck security-github bump-version release check-tuf-root sync-tuf-root sync-qodana-test-exclusions check-qodana-test-exclusions check-test-shards check-test-shards-direct help

all: build

build: ## Build a native binary for the current OS/arch into bin/ (stripped, no debug symbols)
	mkdir -p $(BIN_DIR)
	go build -trimpath -ldflags="-s -w" -o $(BIN_DIR)/$(BIN_NAME) .

run: ## Run the app directly (go run .)
	go run .

fmt: ## Format all Go source files (gofmt + import groups via goimports -local)
	go tool goimports -local $(GOIMPORTS_LOCAL) -w .

fmt-check: ## Fail if any Go file differs from goimports -local (the CI format gate)
	@unformatted=$$(go tool goimports -local $(GOIMPORTS_LOCAL) -l .); \
	if [ -n "$$unformatted" ]; then \
		echo "These files need goimports (run 'make fmt'):"; echo "$$unformatted"; exit 1; \
	fi

check-tuf-root: ## Fail if the embedded GitHub TUF root is expired or has fewer than 60 days remaining (offline)
	go run ./scripts/synctuf --check

sync-tuf-root: ## Fetch and TUF-verify a newer GitHub root into the embed (needs network)
	go run ./scripts/synctuf --write

sync-qodana-test-exclusions: ## Synchronize Qodana's duplication exclusions with every *_test.go file
	@set -eu; \
	listed=$$(mktemp); \
	entries=$$(mktemp); \
	updated=$$(mktemp); \
	trap 'rm -f "$$listed" "$$entries" "$$updated"' 0 1 2 3 15; \
	git ls-files --cached --others --exclude-standard -- '*_test.go' > "$$listed"; \
	while IFS= read -r test_file; do [ -f "$$test_file" ] && printf '%s\n' "$$test_file"; done < "$$listed" | \
		LC_ALL=C sort -u | \
		awk '{ printf "      - \"%s\"\n", $$0 }' > "$$entries"; \
	awk -v entries="$$entries" '\
		function emit_entries(entry) { \
			while ((getline entry < entries) > 0) print entry; \
			close(entries); \
		} \
		{ \
			if ($$0 == "exclude:") { in_exclude = 1; print; next } \
			if (in_paths) { \
				if ($$0 ~ /^      - /) { \
					if ($$0 !~ /_test\.go"$$/) print; \
					next; \
				} \
				in_paths = 0; \
				in_duplicate = 0; \
			} \
			if (in_exclude && $$0 == "  - name: DuplicatedCode") in_duplicate = 1; \
			if (in_duplicate && $$0 == "    paths:") { \
				print; \
				emit_entries(); \
				in_paths = 1; \
				replaced = 1; \
				next; \
			} \
			if (in_exclude && $$0 !~ /^ /) in_exclude = 0; \
			print; \
		} \
		END { \
			if (!replaced) { \
				print "DuplicatedCode exclusion paths block not found in qodana.yaml" > "/dev/stderr"; \
				exit 1; \
			} \
		}' qodana.yaml > "$$updated"; \
	if cmp -s qodana.yaml "$$updated"; then \
		echo "Qodana test exclusions are already synchronized."; \
	else \
		cp "$$updated" qodana.yaml; \
		echo "Updated qodana.yaml test exclusions."; \
	fi; \
	$(MAKE) --no-print-directory check-qodana-test-exclusions

check-qodana-test-exclusions: ## Fail if qodana.yaml does not exclude every *_test.go from duplication checks
	@set -eu; \
	listed=$$(mktemp); \
	test_files=$$(mktemp); \
	excluded_files=$$(mktemp); \
	trap 'rm -f "$$listed" "$$test_files" "$$excluded_files"' 0 1 2 3 15; \
	git ls-files --cached --others --exclude-standard -- '*_test.go' > "$$listed"; \
	while IFS= read -r test_file; do [ -f "$$test_file" ] && printf '%s\n' "$$test_file"; done < "$$listed" | \
		LC_ALL=C sort -u > "$$test_files"; \
	sed -nE 's/^      - "([^"]+_test\.go)"$$/\1/p' qodana.yaml | LC_ALL=C sort -u > "$$excluded_files"; \
	missing=$$(comm -23 "$$test_files" "$$excluded_files"); \
	stale=$$(comm -13 "$$test_files" "$$excluded_files"); \
	if [ -n "$$missing$$stale" ]; then \
		if [ -n "$$missing" ]; then printf 'Missing Qodana test exclusions:\n%s\n' "$$missing"; fi; \
		if [ -n "$$stale" ]; then printf 'Stale Qodana test exclusions:\n%s\n' "$$stale"; fi; \
		echo "Run 'make sync-qodana-test-exclusions' to update qodana.yaml."; \
		exit 1; \
	fi

vet: ## Run go vet
	go vet ./...

check-test-shards: ## Validate the UI shard manifest against the live Linux/amd64 test inventory
	docker run --rm --platform linux/amd64 \
		--label "$(TEST_CONTAINER_LABEL)" \
		-v "$(CURDIR):/work" -w /work \
		-v picfetch-go-build-linux-amd64:/root/.cache/go-build \
		-v picfetch-go-mod-linux-amd64:/root/go/pkg/mod \
		$(TEST_IMAGE) bash -c '\
			set -e; \
			apt-get update -qq; \
			apt-get install -y -qq make gcc libgl1-mesa-dev xorg-dev libwayland-dev libxkbcommon-dev golang-go ca-certificates >/dev/null; \
			make --no-print-directory check-test-shards-direct \
		'

# Internal entry point for a prepared Linux/amd64 runner. Use the public Docker
# target above for canonical validation from any host.
check-test-shards-direct:
	go run ./scripts/testshards check -package "$(TEST_SHARD_PACKAGE)" -manifest "$(TEST_SHARD_MANIFEST)"

# Internal entry points for a prepared Linux/amd64 runner. The public test-race
# target enters Docker once, checks the manifest, then runs all partitions
# concurrently there; hosted CI can call the partition targets directly.
test-race-direct:
	@$(MAKE) --no-print-directory check-test-shards-direct
	@bash -c '\
		set -u; \
		pids=""; \
		$(MAKE) --no-print-directory test-race-non-ui-direct & pids="$$pids $$!"; \
		$(MAKE) --no-print-directory test-race-ui-direct TEST_SHARD=ui-1 & pids="$$pids $$!"; \
		$(MAKE) --no-print-directory test-race-ui-direct TEST_SHARD=ui-2 & pids="$$pids $$!"; \
		$(MAKE) --no-print-directory test-race-ui-direct TEST_SHARD=ui-3 & pids="$$pids $$!"; \
		status=0; \
		for pid in $$pids; do \
			if ! wait "$$pid"; then status=1; fi; \
		done; \
		exit "$$status" \
	'

test-race-non-ui-direct: override TEST_PARTITION := non-ui
test-race-non-ui-direct:
	bash -c '\
		set -eu -o pipefail; \
		packages="$$(go run ./scripts/testshards partition -package "$(TEST_SHARD_PACKAGE)")"; \
		LANG="$(TEST_LOCALE)" go test $(TEST_RACE_FLAGS) -json $$packages | \
			go run ./scripts/testshards capture -out "$(TEST_CAPTURE)" -partition "$(TEST_PARTITION)" \
	'

test-race-ui-direct: override TEST_PARTITION = $(TEST_SHARD)
test-race-ui-direct:
	bash -c '\
		set -eu -o pipefail; \
		case "$(TEST_SHARD)" in ui-1|ui-2|ui-3) ;; *) echo "TEST_SHARD must be one of ui-1, ui-2, or ui-3" >&2; exit 2;; esac; \
		filter="$$(go run ./scripts/testshards regex -manifest "$(TEST_SHARD_MANIFEST)" -shard "$(TEST_SHARD)")"; \
		LANG="$(TEST_LOCALE)" go test $(TEST_RACE_FLAGS) -json -run "$$filter" $(TEST_SHARD_PACKAGE) | \
			go run ./scripts/testshards capture -out "$(TEST_CAPTURE)" -partition "$(TEST_PARTITION)" \
	'

update-test-image: ## Pull the latest Linux/amd64 Ubuntu image used by Docker tests
	docker pull --platform linux/amd64 "$(TEST_IMAGE)"

test: ## Run tests in Linux/amd64 Docker, matching CI and golden rendering (needs Docker)
	docker run --rm --platform linux/amd64 \
		--label "$(TEST_CONTAINER_LABEL)" \
		-v "$(CURDIR):/work" -w /work \
		-v picfetch-go-build-linux-amd64:/root/.cache/go-build \
		-v picfetch-go-mod-linux-amd64:/root/go/pkg/mod \
		-e HOST_UID=$$(id -u) -e HOST_GID=$$(id -g) \
		$(TEST_IMAGE) bash -c '\
			set -e; \
			apt-get update -qq; \
			apt-get install -y -qq make gcc libgl1-mesa-dev xorg-dev libwayland-dev libxkbcommon-dev golang-go ca-certificates locales procps htop >/dev/null; \
			locale-gen en_US.UTF-8 >/dev/null; \
			export LANG=en_US.UTF-8; \
			status=0; \
			go test -timeout $(TEST_TIMEOUT) $(TEST_RACE) ./... || status=$$?; \
			if [ -d internal/ui/testdata/failed ]; then chown -R "$$HOST_UID:$$HOST_GID" internal/ui/testdata/failed; fi; \
			exit $$status \
		'

enter-test-container: ## Open Bash in the running test container (htop/top available)
	@container_ids=$$(docker ps --filter "label=$(TEST_CONTAINER_LABEL)" --format '{{.ID}}') || exit $$?; \
	set -- $$container_ids; \
	if [ "$$#" -eq 0 ]; then \
		echo "No PicFetch test container is running. Start one with 'make test' or 'make verify'."; \
		exit 1; \
	fi; \
	if [ "$$#" -gt 1 ]; then \
		echo "Multiple PicFetch test containers are running; stop all but the one you want to enter:"; \
		docker ps --filter "label=$(TEST_CONTAINER_LABEL)" --format '  {{.ID}}\t{{.Names}}\t{{.Status}}'; \
		exit 1; \
	fi; \
	exec docker exec -it "$$1" bash -lc '\
		if ! command -v htop >/dev/null || ! command -v top >/dev/null; then \
			echo "Waiting for test container setup to finish..."; \
			until command -v htop >/dev/null && command -v top >/dev/null; do sleep 1; done; \
		fi; \
		exec bash \
	'

test-native: ## Run tests directly on the current OS/architecture
	go test -timeout $(TEST_TIMEOUT) ./...

test-race: ## Run the guarded race partitions concurrently in one Linux/amd64 Docker container
	docker run --rm --platform linux/amd64 \
		--label "$(TEST_CONTAINER_LABEL)" \
		-v "$(CURDIR):/work" -w /work \
		-v picfetch-go-build-linux-amd64:/root/.cache/go-build \
		-v picfetch-go-mod-linux-amd64:/root/go/pkg/mod \
		-e HOST_UID=$$(id -u) -e HOST_GID=$$(id -g) \
		$(TEST_IMAGE) bash -c '\
			set -e; \
			apt-get update -qq; \
			apt-get install -y -qq make gcc libgl1-mesa-dev xorg-dev libwayland-dev libxkbcommon-dev golang-go ca-certificates locales procps htop >/dev/null; \
			locale-gen $(TEST_LOCALE) >/dev/null; \
			status=0; \
			make --no-print-directory test-race-direct || status=$$?; \
			if [ -d internal/ui/testdata/failed ]; then chown -R "$$HOST_UID:$$HOST_GID" internal/ui/testdata/failed; fi; \
			exit $$status \
		'

verify: fmt-check check-tuf-root check-qodana-test-exclusions ## Run the same checks CI does (format, TUF root, Qodana exclusions, vet, build, race tests)
	go vet ./...
	go build ./...
	$(MAKE) test-race

golden: ## Regenerate the e2e golden-master screenshots via Docker (linux/amd64, matching CI exactly - needs Docker)
	@# Fyne's software rasterizer renders slightly different anti-aliased
	@# pixels depending on CPU architecture - fyne.io/fyne/v2's own test
	@# harness even special-cases darwin/arm64 for it. A master captured by
	@# running `go test` directly on a non-amd64-Linux machine can pass there
	@# and still fail in CI, which runs on ubuntu-latest/amd64 with no such
	@# leniency - this target renders in the same environment CI does so the
	@# result is never machine-dependent. See CONTRIBUTING.md for the full
	@# accept-a-new-master workflow.
	docker run --rm --platform linux/amd64 \
		-v "$(CURDIR):/work" -w /work \
		-e HOST_UID=$$(id -u) -e HOST_GID=$$(id -g) \
		$(TEST_IMAGE) bash -c '\
			set -e; \
			apt-get update -qq; \
			apt-get install -y -qq gcc libgl1-mesa-dev xorg-dev libwayland-dev libxkbcommon-dev golang-go ca-certificates >/dev/null; \
			go test -run TestE2E ./internal/ui/... -v || true; \
			if [ -d internal/ui/testdata/failed ]; then chown -R "$$HOST_UID:$$HOST_GID" internal/ui/testdata/failed; fi \
		'
	@echo "Inspect internal/ui/testdata/failed/*.png (if any), and if they look right, copy the ones you want over the matching internal/ui/testdata/*.png to accept them as the new baseline."

tidy: ## Tidy go.mod / go.sum
	go mod tidy

security-govulncheck: ## Scan dependencies with the module-pinned govulncheck
	go tool govulncheck ./...

security-github: ## List open GitHub Dependabot alerts for this repo (needs `gh auth login`)
	gh api "repos/$$(gh repo view --json nameWithOwner -q .nameWithOwner)/dependabot/alerts" \
		--jq '.[] | select(.state=="open") | "\(.security_advisory.severity)\t\(.dependency.package.name)\t\(.security_advisory.summary)"'

security: security-govulncheck security-github ## Run all security checks (govulncheck + GitHub Dependabot alerts)

clean: ## Remove all build artifacts
	rm -rf $(BIN_DIR) fyne-cross "$(APP_NAME).app" "$(BIN_NAME).zip"

package-mac: ## Package a macOS .app bundle (native, no Docker) into bin/
	fyne package -os darwin -icon $(ICON) -name "$(APP_NAME)" -appID $(PACKAGE_ID) -release
	go run ./scripts/plistdoctypes "$(APP_NAME).app/Contents/Info.plist"
	mkdir -p $(BIN_DIR)
	rm -rf "$(BIN_DIR)/$(APP_NAME).app"
	mv "$(APP_NAME).app" "$(BIN_DIR)/"

package-windows: ## Cross-compile Windows .exe files via fyne-cross (needs Docker) into bin/, one per arch in WIN_ARCHES (stripped by default)
	mkdir -p $(BIN_DIR)
	for arch in $(WIN_ARCHES); do \
		fyne-cross windows -arch=$$arch -icon $(ICON) -name $(BIN_NAME) -app-id $(PACKAGE_ID) -env GOTOOLCHAIN=auto || exit 1; \
		cp fyne-cross/bin/windows-$$arch/$(BIN_NAME).exe $(BIN_DIR)/$(BIN_NAME)-windows-$$arch.exe; \
	done

package-windows-store: ## Cross-compile Microsoft Store-managed Windows .exe files into bin/ (MSIX packaging runs on Windows in CI)
	mkdir -p $(BIN_DIR)
	for arch in $(WIN_ARCHES); do \
		fyne-cross windows -arch=$$arch -icon $(ICON) -name $(BIN_NAME) -app-id $(PACKAGE_ID) -tags microsoftstore -env GOTOOLCHAIN=auto || exit 1; \
		cp fyne-cross/bin/windows-$$arch/$(BIN_NAME).exe $(BIN_DIR)/$(BIN_NAME)-microsoft-store-$$arch.exe; \
	done

package-windows-debug: ## Cross-compile console-subsystem, unstripped Windows .exe files for diagnosing startup failures, one per arch in WIN_ARCHES
	mkdir -p $(BIN_DIR)
	for arch in $(WIN_ARCHES); do \
		fyne-cross windows -arch=$$arch -icon $(ICON) -name $(BIN_NAME)-debug -app-id $(PACKAGE_ID) -env GOTOOLCHAIN=auto -console -no-strip-debug || exit 1; \
		cp fyne-cross/bin/windows-$$arch/$(BIN_NAME)-debug.exe $(BIN_DIR)/$(BIN_NAME)-debug-windows-$$arch.exe; \
	done

package-linux: ## Cross-compile Linux binaries via fyne-cross (needs Docker) into bin/, one per arch in LINUX_ARCHES (stripped by default)
	mkdir -p $(BIN_DIR)
	for arch in $(LINUX_ARCHES); do \
		fyne-cross linux -arch=$$arch -icon $(ICON) -name $(BIN_NAME) -app-id $(PACKAGE_ID) -env GOTOOLCHAIN=auto || exit 1; \
		cp fyne-cross/bin/linux-$$arch/* $(BIN_DIR)/$(BIN_NAME)-linux-$$arch; \
	done

package-linux-debug: ## Cross-compile unstripped Linux binaries for diagnosing startup failures, one per arch in LINUX_ARCHES
	mkdir -p $(BIN_DIR)
	for arch in $(LINUX_ARCHES); do \
		fyne-cross linux -arch=$$arch -icon $(ICON) -name $(BIN_NAME)-debug -app-id $(PACKAGE_ID) -env GOTOOLCHAIN=auto -no-strip-debug || exit 1; \
		cp fyne-cross/bin/linux-$$arch/* $(BIN_DIR)/$(BIN_NAME)-debug-linux-$$arch; \
	done

build-linux-all: package-linux ## Alias for package-linux: cross-compile Linux binaries for all LINUX_ARCHES via fyne-cross (needs Docker)

build-all: package-mac package-windows package-linux ## Build release artifacts for macOS, Windows, and Linux

install-tools: ## Install the fyne and fyne-cross packaging tools
	go install fyne.io/fyne/v2/cmd/fyne@latest
	go install github.com/fyne-io/fyne-cross@latest

install-linux-tools: ## Install apt dev headers needed to build natively on Linux (OpenGL, X11, Wayland; needs sudo)
	sudo apt-get update
	sudo apt-get install -y gcc libgl1-mesa-dev xorg-dev libwayland-dev libxkbcommon-dev

bump-version: ## Bump FyneApp.toml's Version/Build only (PART=major|minor|patch, default patch); no commit, no tag
	@scripts/bump_version.sh $${PART:-patch} >/dev/null
	@echo "FyneApp.toml was updated but NOT committed. Use 'make release' for the full flow."

release: ## Full release: verify, bump version, commit, tag, push (PART=major|minor|patch, default patch; YES=1 skips the prompt)
	@# The tag must contain its own version bump, so this target commits the
	@# FyneApp.toml edit before tagging - the one place the Makefile writes to
	@# git history. Publishing happens in .github/workflows/release.yml, which
	@# is triggered by the tag push and re-runs CI as a gate, so a red run
	@# leaves the tag orphaned rather than shipping a broken build.
	@# A GitHub TUF root bump, if any, is a separate commit before Release.
	@# Release notes come from todos.md ## Done (empty categories dropped);
	@# they are written to .github/release-notes.md in the Release commit so
	@# the workflow can attach them, then Done items are cleared.
	@set -e; \
	part=$${PART:-patch}; \
	branch=$$(git rev-parse --abbrev-ref HEAD); \
	if [ "$$branch" != "$(RELEASE_BRANCH)" ]; then \
		echo "On branch '$$branch', expected '$(RELEASE_BRANCH)' (override with RELEASE_BRANCH=)"; exit 1; \
	fi; \
	if [ -n "$$(git status --porcelain)" ]; then \
		echo "Working tree is dirty - commit or stash first:"; git status --short; exit 1; \
	fi; \
	git fetch --quiet origin "$(RELEASE_BRANCH)"; \
	if [ "$$(git rev-parse HEAD)" != "$$(git rev-parse origin/$(RELEASE_BRANCH))" ]; then \
		echo "HEAD and origin/$(RELEASE_BRANCH) have diverged - pull/push first"; exit 1; \
	fi; \
	prev_version=$$(sed -nE 's/^Version = "(.*)"/\1/p' FyneApp.toml); \
	new_version=$$(scripts/bump_version.sh $$part --dry-run); \
	tag="v$$new_version"; \
	if git rev-parse -q --verify "refs/tags/$$tag" >/dev/null || \
	   [ -n "$$(git ls-remote --tags origin "refs/tags/$$tag")" ]; then \
		echo "Tag $$tag already exists"; exit 1; \
	fi; \
	notes=$$(go run ./scripts/releasenotes --prev "$$prev_version" --next "$$new_version") || exit 1; \
	printf '%s\n\n' "$$notes"; \
	$(MAKE) sync-tuf-root; \
	tuf_root="internal/update/embed/tuf-repo.github.com/root.json"; \
	tuf_changed=$$(git status --porcelain -- "$$tuf_root"); \
	if [ -z "$$YES" ]; then \
		extra=""; \
		if [ -n "$$tuf_changed" ]; then \
			extra=" plus a GitHub TUF root commit"; \
		fi; \
		printf "Release %s from %s (%s)%s? [y/N] " "$$tag" "$(RELEASE_BRANCH)" "$$(git rev-parse --short HEAD)" "$$extra"; \
		read answer; \
		case $$answer in y|Y|yes|YES) ;; *) echo "Aborted."; exit 1 ;; esac; \
	fi; \
	$(MAKE) verify; \
	if [ -n "$$tuf_changed" ]; then \
		git add "$$tuf_root"; \
		git commit -m "Update GitHub TUF root"; \
	fi; \
	go run ./scripts/releasenotes --prev "$$prev_version" --next "$$new_version" --write .github/release-notes.md --clear-done; \
	scripts/bump_version.sh $$part >/dev/null; \
	git add FyneApp.toml .github/release-notes.md todos.md; \
	git commit -m "Release $$tag"; \
	git tag -a "$$tag" -m "Release $$tag"; \
	git push origin "$(RELEASE_BRANCH)"; \
	git push origin "$$tag"; \
	echo "Pushed $$tag - .github/workflows/release.yml now builds and publishes the artifacts."; \
	scripts/watch_release.sh "$$tag"

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*##' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*##"}; {printf "  %-16s %s\n", $$1, $$2}'
