# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/).

## [Unreleased]

### Added
- `Options.Commodities`: how each commodity's amounts are written, by
  code: a symbol before the number and after its sign, the decimals
  shown and a thousands separator, so `{"GBP": {Symbol: "£", Decimals:
  2, Thousands: ","}}` writes `-£1,234.56`. A commodity not described
  is written the default way (see Changed). The column gap is sized to
  the amount as written.
- `Options.Palette`: the fill of each account type's boxes by type name,
  legend swatch included; a type not named keeps pta2svg's own colour.
- The browser builds read both from the options object:
  `renderModel(model, {commodities: {GBP: {symbol: "£", decimals: 2,
  thousands: ","}}, palette: {Asset: "#d1f0e2"}})`.
- Box formatting (#9): `Options.Shape` (`Stadium`
  pill boxes beside the default `Rounded`), `Options.Strokes` (stroke
  width by account type name, as `Palette`), `Options.BalanceBelowName`
  (the balance in bold beneath the name rather than small in the
  corner), and `Options.Style`. Its default, `afp`, is the look of
  bytestone.uk/afp: stadium boxes, Income and Expense stroked at 3, the
  balance beneath the name; a `Shape` or `Strokes` set alongside it
  applies on top. `"plain"` is the drawing without the preset; an
  unknown style is an error. The browser builds
  read `shape`, `strokes`, `balanceBelowName` and `style`, and the Try-it
  page has a style choice.
- Docs: each example shows its JSON model beside the source; the Try-it
  page takes the JSON model as a format, drawn through `renderModel`;
  the index lists the browser builds with their sizes, raw and gzipped,
  against the full build. A favicon: a to b, two boxes and an arrow.
  The example pages have layout (lr, sugiyama), style (afp, plain) and
  animate controls that redraw every diagram on the page in the browser
  from its JSON model with the TinyGo build.

### Changed
- The default layout is Sugiyama, the layered graph layout; LR is asked
  for by name (`Layout: pta2svg.LR`, `-layout lr`, `layout: "lr"`).
- The default drawing is the afp style; `Style: "plain"` is the old
  look.
- Money is written for people by default: GBP, USD, EUR and JPY with
  their symbols, every commodity with thousands separated (`-£1,234.56`,
  `¥123,456`, `1,234.50 XYZ`) where it was `-1234.56 GBP`.
  `Commodity.Suffix` writes text after the number, so `{"GBP": {Suffix:
  " GBP", Decimals: 2}}` keeps the old form. The TinyGo build grows by
  under 1 KB.

### Fixed
- A Sugiyama arrow's label is centred on the point halfway along the
  arrow by length, the amount under the description, with a white halo
  so the arrow passes behind the text instead of striking it out. It
  was split above and below the midpoint of the middle segment, which
  for a bent arrow is the bend.
- In the LR layout an arrow that jumps a column no longer runs through
  the boxes in between: it is routed along a lane of its own below them,
  leaving and entering through a box's bottom edge when nothing sits
  beneath it and through the column gap otherwise. The canvas grows to
  hold the lanes. An arrow back to an earlier column takes a lane the
  same way instead of arcing off the top of the canvas, and an arrow to
  a box in its own column is a bracket through the gap to its left
  instead of a loop through its own box.
- In the LR layout arrows meeting a box are spread along its side, in
  order of where their other ends lie, instead of meeting at one point;
  a run up or down a column gap keeps close to the box it serves, clear
  of the labels in the middle of the gap; forward hops are straight. No
  two arrows share a stretch of line.
- A label whose arrow is horizontal at its middle is written above and
  below the line rather than over it, and the first lane is far enough
  below the boxes for its label. Every arrow label is haloed, so a line
  crossing it passes behind the text.

## [0.4.0] - 2026-10-12

 - Label-aware column gap, the TinyGo lite build, direct SVG writing

### Added
- `pta2svg-tiny.wasm`, the lite build compiled with TinyGo: 0.7 MB against 4 MB, the same SVG, loaded with TinyGo's `wasm_exec_tinygo.js`. `task wasm:tiny:test`, run by `task check`, runs the root, render and wasm packages under TinyGo.

### Changed
- The SVG is written directly rather than through text/template, which TinyGo cannot run. The output is unchanged and pinned by golden files for every testdata ledger. The Go builds shrink too: lite from 6 MB to 4 MB, full from 26 MB to 24 MB.

### Fixed
- The Sugiyama layout drew nodes, and so the legend, in an order that changed from run to run.
- The gap between columns grows to fit the widest arrow label (the description or the formatted amount, estimated from its length at the edge font with clearance either side), so a long label such as a movement code no longer sits over the boxes. Both layouts take the gap; the goldens are re-pinned.
- A routed (Sugiyama) edge's label is drawn halfway along its path; it was drawn at the middle point of the polyline, which for a straight edge is its end, on the box.

## [0.3.0] - 2026-10-11

 - Diagram model as JSON and a 6 MB lite wasm build without the text parsers; text parsing moved to the parse subpackage

### Added
- `Model`, the `Diagram` as JSON: `ToModel` and `Model.Diagram`. Accounts are optional overrides of the type and short name inferred from the path; balances are computed on load (#8).
- `cmd/pta2svg-wasm-lite`, a 6 MB browser build without the text parsers, installing `pta2svg.renderModel(model, opts)` for a page that holds the ledger as data. The full build installs `renderModel` beside `render` (#8).
- `task deps:guard`, run by `task check`, fails if the root package or the lite build links gotreesitter.

### Fixed
- The browser build's `render` crashed the Go runtime on a parse error instead of throwing a JS `Error`, leaving the page dead.

### Changed
- Text parsing moved to the `parse` subpackage: `parse.Parse`, `parse.Render(w, src, format, opts)`, `parse.Format`, `DetectFormat`, `ParseFormat`. `Options` loses `Format`. The root package now links no tree-sitter grammars, which `task deps:guard` enforces, so a browser build that takes the ledger as data can stay small (#8).

## [0.2.0] - 2026-10-10

 - Library package, wasm build, opening balances, hyphens and quoted names in accounts, syntax errors reported

### Added
- Public library package at the module root: `Render`, `Parse`, `RenderDiagram`, `Options`, the `Diagram` model (#5).
- WebAssembly build (`cmd/pta2svg-wasm`, `task wasm:build`) installing `pta2svg.render` on the page, with a Try-it docs page (#6).
- `balance` directive seeding an account's opening balance, so one movement is drawn with the balances it leaves (#4).
- Hyphens in account path segments, and quoted account names for names with spaces (#2, #3).
- Every testdata file is parsed in its declared format by a test.

### Fixed
- A file the grammar rejects is reported as an error with its line number instead of being drawn as an empty diagram (#1).
- The goluca examples had unquoted descriptions, which is pta syntax; they are now valid goluca.

### Changed
- Requires gotreesitter with the regenerated pta and goluca grammars.

## [0.1.1] - 2026-03-02

 - Starting to work## [0.1.3] - 2026-03-03

 - Hope to update docs## [0.1.5] - 2026-03-05

 - Goluca syntax and mostly working
## [0.1.6] - 2026-03-06

 - Creating install and getting ready for task-plus

## [0.1.4] - 2026-03-04

 - Refactoring to use gotreesitter parser

## [0.1.2] - 2026-03-02

 - improving plain text format

## [0.1.0] - 2026-03-02

 - Working release but layout needs to be improved.

## [0.1.0] - 2026-02-28

### Added
- Initial implementation
- Line-oriented parser for go-luca plain text accounting
- Left-to-right column layout engine
- SVG renderer with text/template
- CSS flow animations (optional)
- CLI with -o, -animate, -title, -layout flags
