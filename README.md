# Keep

Keep is a minimal Discord bot which archives URLs visible to the configured
account (sent by anyone, anywhere) on the Wayback Machine.

A local cache of saved URLs is kept to prevent duplicate availability API
requests.

## Installation

Keep can be compiled with `make` or `go build`, and installed system-wide by
running `make install` with root-level permissions. Tests can be run with `make
test`.

## Usage

Create `~/.keep`, copy and populate `keep.json`, then start `./keep`. An index
of processed URLs can be found at `127.0.0.1:9099`.

```
Usage of ./keep:
  -config string
        path to configuration file (default "~/.keep/keep.json")
```
