# Keep

Keep is a minimal Discord bot which saves any URLs parsed from messages visible
to the configured account on the Wayback Machine.

A local cache of saved URLs is kept to prevent duplicate availability API
requests.

## Installation

Keep can be compiled with `make` or `go build`, and installed system-wide by
running `make install` with root-level permissions. Tests can be run with `make
test`.

## Usage

```
Usage of ./keep:
  -config string
        path to configuration file (default "/home/jordan/.keep/keep.json")
```
