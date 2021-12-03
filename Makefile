.POSIX:
.SUFFIXES:

GO = go
RM = rm
GOFLAGS =
PREFIX = /usr/local
BINDIR = $(PREFIX)/bin
CONFIGDIR = $(HOME)/.keep

goflags = $(GOFLAGS)

all: keep

keep:
	$(GO) build $(goflags) -ldflags "-X main.buildPrefix=$(PREFIX)"

clean:
	$(RM) -f keep

test:
	$(GO) test -v ./...

install: all
	mkdir -p $(DESTDIR)$(BINDIR)
	mkdir -p $(DESTDIR)$(CONFIGDIR)
	cp -f keep $(DESTDIR)$(BINDIR)
	cp -n keep.json $(DESTDIR)$(CONFIGDIR)

