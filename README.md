# pta2svg

Draws plain text accounting as an SVG flow diagram: accounts as boxes carrying their balances, movements as arrows. Reads go-luca's `.goluca` format, the simpler `.pta` movement format and beancount. Runs as a CLI, a Go library or in the browser as WebAssembly.

## Install

```bash
go install git.bytestone.uk/hum3/pta2svg/cmd/pta2svg@latest
```

## Usage

```bash
pta2svg [flags] <file>
  -o string       output file (default: stdout)
  -format string  input format: auto, pta, goluca, beancount (default "auto", by extension)
  -layout string  layout engine: lr, sugiyama (default "lr")
  -title string   SVG title
  -animate        enable CSS flow animations
```

```bash
pta2svg testdata/examples/simple.goluca > simple.svg
pta2svg -animate -title "Interest" testdata/examples/movement.goluca > movement.svg
```

A file the grammar rejects is reported with its line number rather than drawn empty.

## Input format

```
2026-01-31 balance "Liability:Savings:Alice Lovelace:easy-access" 1000.00 GBP

2026-01-31 * Interest applied
  Expense:Interest -> "Liability:Savings:Alice Lovelace:easy-access" "interest" 1.27 GBP
```

Movements: `[+]from -> to ["description"] amount [commodity]`. In `.goluca` the description is quoted; in `.pta` it is bare words. Arrow variants: `>`, `->`, `//`, `→`. The `+` prefix links movements for staggered animation.

Account paths are `Type:Segment[:Segment...]`; segments may carry hyphens and a name with spaces is quoted. The first segment (Asset, Liability, Equity, Income, Expense) sets the colour and column.

A `balance` directive states what an account held before the movements, so a single movement can be drawn with the balances it leaves. Without one, balances are the net of the movements in the file.

## Library

```go
import (
	"git.bytestone.uk/hum3/pta2svg"
	"git.bytestone.uk/hum3/pta2svg/parse"
)

err := parse.Render(w, src, parse.Goluca, pta2svg.Options{Title: "January"})
```

`parse.Parse` returns the `Diagram`; `pta2svg.RenderDiagram` draws one built without text (`NewDiagram`, `GetOrCreateAccount`, `Transactions`, `Opening`, then `ComputeBalances`). The text parsers live in `parse` because their tree-sitter grammars are most of the size of a browser build; a program that holds the ledger as data imports only the root package.

## Browser

`task wasm:build` compiles `cmd/pta2svg-wasm` to `docs/pta2svg.wasm`. Loaded with Go's `wasm_exec.js`, it installs:

```js
pta2svg.render(text, {format: "goluca", title: "", animate: false, layout: "lr"}) // -> SVG string, throws on error
```

The docs site's Try-it page is this build.

## Links

| | |
|---|---|
| Source | https://git.bytestone.uk/hum3/pta2svg |
| Mirror (GitHub) | https://github.com/drummonds/pta2svg |
