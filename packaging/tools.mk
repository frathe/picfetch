# Reviewed packaging inputs. Upgrade these together with the packaging smoke
# procedure in docs/packaging-inputs.md; image digests identify multiarch indexes.
FYNE_VERSION := v1.7.2
FYNE_CROSS_VERSION := v1.6.3
FYNE_CROSS_WINDOWS_IMAGE ?= fyneio/fyne-cross-images:windows@sha256:5663afcd79d447e56c57226c3a06cc895fd83a4aa99d2d1974cf0338e9d08f36
FYNE_CROSS_LINUX_IMAGE ?= fyneio/fyne-cross-images:linux@sha256:7502500e2224dbbc207df49b13c98b9116a6f6967ff3f8ceab6798be75918706
FYNE_CROSS_ENGINE ?= docker
FYNE_CROSS_CACHE ?= $(dir $(shell go env GOCACHE))fyne-cross
FYNE_BIN := .tools/fyne-$(FYNE_VERSION)/fyne
FYNE_CROSS_BIN := .tools/fyne-cross-$(FYNE_CROSS_VERSION)/fyne-cross
