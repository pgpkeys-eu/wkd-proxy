PROJECTPATH = $(dir $(realpath $(firstword $(MAKEFILE_LIST))))
export GOPATH := $(PROJECTPATH)
export GOCACHE := $(GOPATH)/.gocache
export SRCDIR := $(PROJECTPATH)src/wkd-proxy
VERSION ?= $(shell git describe --tags 2>/dev/null)
TIMESTAMP = $(shell date -Iseconds -u)

project = wkd-proxy

prefix = /usr
statedir = /var/lib/$(project)

commands = \
	$(project)

all: test build

build:

clean: clean-go
	rm -rf debian/{.debhelper/,$(project).debhelper.log,$(project).postinst.debhelper,$(project).postrm.debhelper,$(project).prerm.debhelper,$(project).substvars,$(project)/}

clean-go:
	-chmod -R u+rwX pkg
	rm -rf $(PROJECTPATH)/.gocache
	rm -rf $(PROJECTPATH)/bin
	rm -rf $(PROJECTPATH)/pkg

dch:
	gbp dch --debian-tag='%(version)s' -D trixie --git-log --first-parent

deb-src:
	debuild -S -sa -I

install:
	mkdir -p -m 0755 $(DESTDIR)$(prefix)/bin
	cp -a bin/$(project)* $(DESTDIR)$(prefix)/bin
	mkdir -p -m 0755 $(DESTDIR)/etc/$(project)
	cp -a etc/$(project).conf* $(DESTDIR)/etc/$(project)

install-build-depends:
	sudo apt install -y \
	    debhelper \
		dh-systemd \
	    git-buildpackage \
	    golang

lint: lint-go

lint-go:
	cd $(SRCDIR) && ! go fmt $(project)/... | awk '/./ {print "ERROR: go fmt made unexpected changes:", $$0}' | grep .
	cd $(SRCDIR) && go vet $(project)/...

test: test-go

test-coverage:
	cd $(SRCDIR) && go test -coverprofile=${PROJECTPATH}/cover.out $(project)/...
	cd $(SRCDIR) && go tool cover -html=${PROJECTPATH}/cover.out
	rm cover.out

test-go:
	cd $(SRCDIR) && go test $(project)/... -count=1

#
# Generate targets to build Go commands.
#
define make-go-cmd-target
	$(eval cmd_name := $1)
	$(eval cmd_package := $(project)/cmd/$(cmd_name))
	$(eval cmd_target := $(cmd_name))

$(cmd_target):
	cd $(SRCDIR) && \
	go install -ldflags " \
			-X $(project)/server.Version=$(VERSION) \
			-X $(project)/server.BuiltAt=$(TIMESTAMP) \
		" $(cmd_package)

build: $(cmd_target)

endef

$(foreach command,$(commands),$(eval $(call make-go-cmd-target,$(command))))
