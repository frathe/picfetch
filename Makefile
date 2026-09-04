GO ?= go
SITE_SOURCE ?= website.md
SITE_TEMPLATES ?= site/templates
SITE_TRANSLATIONS ?= site/translations/de.json
SITE_OUTPUT_DIR ?= docs
SITE_LOCALES ?= en,de
SITE_FORMATS ?= regular,amp
DEEPL_API_URL ?=
DEEPL_ENV_FILE ?= .env.local
NODE ?= node
AMP_VALIDATOR ?= site/tools/validate-amp.cjs

export SITE_SOURCE
export SITE_TEMPLATES
export SITE_TRANSLATIONS
export SITE_OUTPUT_DIR
export SITE_LOCALES
export SITE_FORMATS
export DEEPL_API_URL
export DEEPL_ENV_FILE
export NODE
export AMP_VALIDATOR

.PHONY: build translate update check-generated check validate-amp test-browser
build:
	env -u DEEPL_API_KEY $(GO) run ./cmd/sitegen build

translate:
	$(GO) run ./cmd/sitegen translate

update:
	$(GO) run ./cmd/sitegen update

check-generated:
	env -u DEEPL_API_KEY $(GO) run ./cmd/sitegen check

check:
	env -u DEEPL_API_KEY $(GO) test ./...
	$(MAKE) check-generated
	$(MAKE) validate-amp
	$(MAKE) test-browser

validate-amp:
	env -u DEEPL_API_KEY node site/tools/validate-amp.cjs

test-browser:
	env -u DEEPL_API_KEY node site/tools/test-language.cjs
