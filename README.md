# SuperACH

Cross-platform native desktop viewer & editor for [NACHA ACH](https://www.nacha.org) files, built in Go with [Fyne](https://fyne.io) and powered by [moov-io/ach](https://github.com/moov-io/ach).

One binary, no runtime dependencies beyond the OS graphics stack. Runs on macOS (arm64 + amd64), Windows (amd64), and Linux (amd64 + arm64).

## Features

- **Open / view / save** NACHA `.ach` files and JSON representations
- **Field-level editing** with live moov-io/ach validation
- **Structure tree** (File → Batches → Entries → Addenda) with addenda 05 / 98 / 99 / IAT 10–18
- **Add / remove** batches and entries
- **ACH Return wizard** — pick R01–R85, attach Addenda99, mark dishonored/contested
- **NOC wizard** — pick C01–C14, guided corrected-data entry in a COR batch
- **IAT** — IAT batch + entry + all seven mandatory addenda + optional 17/18
- **CSV import / export** — flatten entries to CSV and round-trip back
- **JSON import / export** — full file round-trip via moov-io/ach's JSON codec
- **Undo** — 20-step deep-copy snapshot stack

## Build from source

Prereqs: Go 1.25+, a C toolchain, and Fyne's OS build deps ([list](https://docs.fyne.io/started/)).

```bash
make run          # dev run
make test         # unit tests
make build        # host binary into ./dist/
make build-all    # cross-compile for mac/win/linux via fyne-cross (needs Docker)
```

Release binaries are produced automatically by GitHub Actions on `v*` tags (see `.github/workflows/release.yml`).

## Project layout

```
cmd/superach/         entrypoint
internal/achio/       moov-io/ach wrapper: read/write, JSON/CSV, mutate, returns, NOC, IAT
internal/ui/          Fyne app shell, state, tree, detail router, menu
internal/ui/forms/    per-record-type form constructors
internal/ui/dialogs/  modal wizards (new batch / return / NOC / about)
testdata/             sample .ach files (from moov-io/ach)
```

## License

Apache 2.0, matching moov-io/ach. Sample `.ach` files in `testdata/` are derived from the moov-io/ach test corpus (see their LICENSE).
