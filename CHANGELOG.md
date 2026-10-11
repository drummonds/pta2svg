# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/).

## [Unreleased]

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
