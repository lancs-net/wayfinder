# SPDX-License-Identifier: BSD-3-Clause
#
# Authors: Alexander Jung <a.jung@lancs.ac.uk>
#
# Copyright (c) 2020, Lancaster University.  All rights reserved.
#
# Redistribution and use in source and binary forms, with or without
# modification, are permitted provided that the following conditions
# are met:
#
# 1. Redistributions of source code must retain the above copyright
#    notice, this list of conditions and the following disclaimer.
# 2. Redistributions in binary form must reproduce the above copyright
#    notice, this list of conditions and the following disclaimer in the
#    documentation and/or other materials provided with the distribution.
# 3. Neither the name of the copyright holder nor the names of its
#    contributors may be used to endorse or promote products derived from
#    this software without specific prior written permission.
#
# THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
# AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
# IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE
# ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE
# LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR
# CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF
# SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS
# INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN
# CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE)
# ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE
# POSSIBILITY OF SUCH DAMAGE.

# Directories
WORKDIR     ?= $(CURDIR)
TESTDIR     ?= $(WORKDIR)/tests
DISTDIR     ?= $(WORKDIR)/dist
INSTALLDIR  ?= /usr/local/bin/

# Arguments
REGISTRY    ?= ghcr.io
ORG         ?= lancs-net
BIN         ?= wayfinder
IMAGE_TAG   ?= latest
IMAGE       ?= $(REGISTRY)/$(ORG)/$(BIN):$(IMAGE_TAG)


ifeq ($(HASH),)
HASH_COMMIT ?= HEAD
HASH        ?= $(shell git update-index -q --refresh && \
                       git describe --tags)
# Others can't be dirty by definition
ifneq ($(HASH_COMMIT),HEAD)
HASH_COMMIT ?= HEAD
endif
DIRTY       ?= $(shell git update-index -q --refresh && \
                       git diff-index --quiet HEAD -- $(WORKDIR) || \
                       echo "-dirty")
endif
APP_VERSION ?= $(HASH)$(DIRTY)
GIT_SHA     ?= $(shell git update-index -q --refresh && \
                       git rev-parse --short HEAD)


# Tools
DOCKER      ?= docker
DOCKER_RUN  ?= $(DOCKER) run --rm $(1) \
               -w /go/src/github.com/$(ORG)/$(BIN) \
               -v $(WORKDIR):/go/src/github.com/$(ORG)/$(BIN) \
               $(REGISTRY)/$(ORG)/$(BIN):$(IMAGE_TAG) \
                 $(2)
GO          ?= go

# Misc
Q           ?= @

# If run with DOCKER= or within a container, unset DOCKER_RUN so all commands
# are not proxied via docker container.
ifeq ($(DOCKER),)
DOCKER_RUN  :=
else ifneq ($(wildcard /.dockerenv),)
DOCKER_RUN  :=
endif
.PROXY      :=
ifneq ($(DOCKER_RUN),)
.PROXY      := docker-proxy-
$(MAKECMDGOALS):
	$(info Running target via Docker ($(IMAGE)...))
	$(Q)$(call DOCKER_RUN,,$(MAKE) $@)
endif

# Targets
.PHONY: all
$(.PROXY)all: build

.PHONY: build
ifeq ($(DEBUG),y)
$(.PROXY)build: GO_GCFLAGS ?= -N -l
endif
$(.PROXY)build: GO_LDFLAGS ?= -s -w
$(.PROXY)build: GO_LDFLAGS += -X "main.version=$(APP_VERSION)"
$(.PROXY)build: GO_LDFLAGS += -X "main.commit=$(GIT_SHA)"
$(.PROXY)build: GO_LDFLAGS += -X "main.buildTime=$(shell date)"
$(.PROXY)build:
	$(GO) build \
		-ldflags='$(GO_GCFLAGS)' \
		-ldflags='$(GO_LDFLAGS)' \
		-o $(DISTDIR)/$(BIN)

# Create an environment where we can build
.PHONY: container
container: GO_VERSION         ?= 1.14
container: DOCKER_BUILD_EXTRA ?=
container:
	$(DOCKER) build \
		--build-arg ORG=$(ORG) \
		--build-arg BIN=$(BIN) \
		--build-arg GO_VERSION=$(GO_VERSION) \
		--tag $(IMAGE) \
		$(DOCKER_BUILD_EXTRA) $(WORKDIR)

# Run an environment where we can build
.PHONY: devenv
devenv: DOCKER_RUN_EXTRA ?= -it --name $(BIN)-devenv
devenv:
	$(Q)$(call DOCKER_RUN,$(DOCKER_RUN_EXTRA),bash)


# For CI
.PHONY: ci-install-ci-tools
ci-install-ci-tools:
	curl -sfL https://install.goreleaser.com/github.com/goreleaser/goreleaser.sh | sh -s -- -b /usr/local/bin/ "v0.146.0"


.PHONY: ci-publish-release
ci-publish-release:
	goreleaser --rm-dist

.PHONY: ci-build-snapshot-packages
ci-build-snapshot-packages:
	goreleaser \
		--snapshot \
		--skip-publish \
		--rm-dist

.PHONY: ci-release
ci-release:
	goreleaser release --rm-dist


.PHONY: ci-test-deb-package-install
ci-test-deb-package-install:
	docker run \
		-v //var/run/docker.sock://var/run/docker.sock \
		-v /${PWD}://src \
		-w //src \
		ubuntu:latest \
			/bin/bash -x -c "\
				apt update && \
				apt install ./dist/$(BIN)_*_linux_amd64.deb -y && \
				$(BIN) version \
			"

.PHONY: ci-test-rpm-package-install
ci-test-rpm-package-install:
	docker run \
		-v //var/run/docker.sock://var/run/docker.sock \
		-v /${PWD}://src \
		-w //src \
		fedora:latest \
			/bin/bash -x -c "\
				dnf install ./dist/$(BIN)_*_linux_amd64.rpm -y && \
				$(BIN) version \
			"

.PHONY: ci-test-linux-run
ci-test-linux-run:
	chmod 755 ./dist/$(BIN)_linux_amd64/$(BIN) && \
	./dist/$(BIN)_linux_amd64/$(BIN) version
