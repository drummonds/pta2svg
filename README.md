# pta2svg

Generate styled SVG flowcharts from plain text accounting (go-luca) files showing money flows between accounts.

## Install

```bash
go install github.com/drummonds/pta2svg/cmd/pta2svg@latest
```

## Usage

```bash
pta2svg [flags] <file.pta>
  -o string     output file (default: stdout)
  -animate      enable CSS flow animations
  -title string SVG title
  -layout string layout engine (default "lr")
```

## Example

```bash
pta2svg testdata/simple.pta > output.svg
pta2svg -animate -title "Bank Deposit" testdata/simple.pta > animated.svg
```

## Input Format

```
2024-01-15 * Salary deposit
  +Income:Salary -> Asset:Bank salary payment 5000.00 GBP

2024-01-20 * Pay rent
  Asset:Bank -> Expense:Rent monthly rent 1200.00 GBP
```

Lines: `[+]from -> to [description] amount [commodity]`

The `+` prefix links movements for staggered animation. Arrow variants: `>`, `->`, `//`, `→`.
