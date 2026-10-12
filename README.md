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
  -layout string  layout engine: sugiyama, lr (default "sugiyama")
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

Money is written for people by default: GBP, USD, EUR and JPY with their symbols and thousands separated (`-£1,234.56`, `¥123,456`), any other commodity separated with its code after (`1,234.50 XYZ`). `Options` also takes how money is written and what colours the boxes are, so a drawing matches the page it sits on:

```go
pta2svg.Options{
	Commodities: map[string]pta2svg.Commodity{"GBP": {Suffix: " GBP", Decimals: 2}}, // 1234.56 GBP rather than the default £1,234.56
	Palette:     map[string]string{"Asset": "#d1f0e2", "Liability": "#fcd5dd"},                      // fill by account type; others keep their own
}
```

and how the boxes are drawn. The default style is `afp`, the look of bytestone.uk/afp: stadium boxes, Income and Expense outlined heavier, the balance in bold beneath the name. `Style: "plain"` drops the preset and leaves the three fields to set alone:

```go
pta2svg.Options{
	Style:            "plain",                          // no preset; "afp" is the default
	Shape:            pta2svg.Stadium,                  // pill-shaped boxes; Rounded is plain's
	Strokes:          map[string]float64{"Income": 3},  // stroke width by account type; others 1.5
	BalanceBelowName: true,                             // balance in bold beneath the name, not in the corner
}
```

## Browser

`task wasm:build` compiles three binaries into `docs/`. Each installs functions on the page that return an SVG string and throw a JS `Error` on failure:

| binary | compiler and shim | installs | size |
|---|---|---|---|
| `pta2svg.wasm` (`cmd/pta2svg-wasm`) | Go, `wasm_exec.js` | `render`, `renderModel` | 24 MB |
| `pta2svg-lite.wasm` (`cmd/pta2svg-wasm-lite`) | Go, `wasm_exec.js` | `renderModel` | 4 MB |
| `pta2svg-tiny.wasm` (`cmd/pta2svg-wasm-lite`) | TinyGo, `wasm_exec_tinygo.js` | `renderModel` | 0.7 MB |

The tiny build is the lite build compiled with [TinyGo](https://tinygo.org) and draws the same SVG. It must be loaded with TinyGo's shim, not Go's. `task wasm:tiny:test`, run by `task check`, runs what it links under TinyGo, so that code stays clear of the reflection TinyGo lacks (text/template, for one).

```js
pta2svg.render(text, {format: "goluca", title: "", animate: false, layout: "sugiyama"})
pta2svg.renderModel(model, {title: "", animate: false, layout: "sugiyama"})
```

Both take the library's `Options` as an object: `{commodities: {GBP: {symbol: "£", decimals: 2, thousands: ","}}, palette: {Asset: "#d1f0e2"}, shape: "stadium", strokes: {Income: 3}, balanceBelowName: true, style: "plain"}`.

The tree-sitter grammars behind `render` are the difference in size. A page that already holds the ledger as data uses the lite or tiny build and passes the model, as JSON text or as an object:

```json
{
  "accounts": [{"name": "Asset:Cash", "short": "Cash", "type": "Asset"}],
  "opening": {"Liability:Savings:alice": 1000.00},
  "transactions": [
    {"date": "2026-01-31", "flag": "*", "payee": "Interest",
     "movements": [{"from": "Expense:Interest", "to": "Liability:Savings:alice",
                    "amount": 1.27, "commodity": "GBP", "description": "interest"}]}
  ]
}
```

`accounts` is optional: accounts are created from the movements that name them, with type and short name inferred from the path, so an entry only overrides that. Balances are computed on load. In Go the same shape is `pta2svg.Model`; `ToModel` and `Model.Diagram` convert to and from a `Diagram`. `task deps:guard` keeps the parsers out of the root package and the lite build.

The docs site's Try-it page is the full build.

## Links

| | |
|---|---|
| Documentation | https://h3-pta2svg.docs.bytestone.uk/ |
| Source | https://git.bytestone.uk/hum3/pta2svg |
| Mirror (GitHub) | https://github.com/drummonds/pta2svg |
