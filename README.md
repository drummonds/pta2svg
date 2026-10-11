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

`task wasm:build` compiles two binaries into `docs/`. Loaded with Go's `wasm_exec.js`, each installs functions on the page that return an SVG string and throw a JS `Error` on failure:

| binary | installs | size |
|---|---|---|
| `pta2svg.wasm` (`cmd/pta2svg-wasm`) | `render`, `renderModel` | 26 MB |
| `pta2svg-lite.wasm` (`cmd/pta2svg-wasm-lite`) | `renderModel` | 6 MB |

```js
pta2svg.render(text, {format: "goluca", title: "", animate: false, layout: "lr"})
pta2svg.renderModel(model, {title: "", animate: false, layout: "lr"})
```

The tree-sitter grammars behind `render` are the difference in size. A page that already holds the ledger as data uses the lite build and passes the model, as JSON text or as an object:

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
| Source | https://git.bytestone.uk/hum3/pta2svg |
| Mirror (GitHub) | https://github.com/drummonds/pta2svg |
