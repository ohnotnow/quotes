# quotes

I used to keep quotes in MacOS sticky notes. Hundreds of them, piling up across desktops, impossible to search. This is the replacement: a small terminal app that stores quotes in SQLite and lets me actually find things again.

## What it does

Two modes. Run `quotes` with no arguments and you get a full-screen TUI for browsing, searching, and adding new quotes. Run `quotes darwin` or `quotes species that survives` and it prints matches straight to stdout so you can pipe them or just grab what you need.

It strips surrounding quotation marks automatically (straight, curly, guillemets), so you can paste from the web without cleaning up first.

The database lives at `~/.quotes/quotes.db` and gets created on first run.

## Prerequisites

- Go 1.25+

## Getting started

```bash
git clone https://github.com/ohnotnow/quotes.git
cd quotes
go build -o quotes .
```

Move the binary somewhere on your `$PATH` if you like:

```bash
mv quotes /usr/local/bin/
```

## Usage

### TUI mode

```bash
quotes
```

| Key | Action |
|-----|--------|
| `j` / `k` or arrow keys | Navigate the list |
| `h` / `l` or arrow keys | Previous / next page |
| `/` | Search |
| `a` | Add a new quote |
| `d` | Delete selected quote (with confirmation) |
| `enter` | View the full quote |
| `q` | Quit |

### Quick search

```bash
quotes darwin
quotes species that survives
quotes "responsive to change"
```

Searches the quote body and the "who" field. Prints matches to stdout, one per result.

## Running tests

```bash
go test ./...
```

## Contributing

Fork it, make your changes, run `go build .` to check it compiles, open a PR. That's it.

## Licence

[MIT](LICENSE)
