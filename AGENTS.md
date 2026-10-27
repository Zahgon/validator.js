# AGENTS.md

This file provides guidance to AI coding agents working with this repository.

## Project map

- validator.js is a Go module (`github.com/validatorjs/validator.js`) for string validation and sanitization; it reproduces the behaviour of the JavaScript library exactly, errors included.
- `validator.go` defines the public exports (`Call`, `Names`, `Version`); one Go file per area holds the validators, sanitizers, normalizers and converters, and the `*_data.go` files hold the static locale and format tables.
- `js/` models the JavaScript values and conversions the library depends on; `internal/jsre` is the JavaScript regular expression engine, `internal/jsdate` the Date arithmetic and `Date.parse`, and `internal/wire` the tagged JSON form of values.
- `cmd/validator` is the JSON Lines command line.
- `validators_test.go` and the fixtures in `testdata/` hold the recorded calls of the original test suites; `util_test.go` and `exports_test.go` cover the helpers and the export list.
- `README.md` is the primary public API reference. Update it when documented API behavior or options change.
- validator.js has no dependencies outside the Go standard library.

## Branching and pull requests

- Normal changes target `master`.
- Keep changes focused on the reported behavior. Avoid unrelated API, formatting, generated-file, or data-table changes.
- Bug fixes should include a focused regression test that fails before the fix and passes afterward.
- Follow the style of recent commit and pull-request titles; the repository does not define a separate title convention in `CONTRIBUTING.md`.

## Install and build

- Use Go 1.23 or newer; the build needs no network access and no module beyond the standard library.
- Build the command with `go build ./cmd/validator`.

## Test

- `go test ./...` is the broad local check; also run `gofmt -l .` and `go vet ./...`.
- For a focused run, use `go test -run '<TestName>/<subtest>' .`.
- When changing a shared helper, identify and test its public callers. A helper-level test alone does not establish public API behavior.
- If a command fails during setup, distinguish toolchain, filesystem and environment failures from product-code failures before changing source.

## Change discipline

- Preserve documented behavior and compatibility outside the stated scope of a change.
- Prefer the smallest root-cause fix over special-casing a reported input.
- Keep static locale and format data changes separate from algorithm changes when practical.
- Do not commit, push, publish a package or advisory, open an issue or pull request, or disclose private security material unless explicitly authorized.

## Documentation and threat model

- Update `README.md` when a public API, option, return contract, or documented limitation changes.
- Read `SECURITY.md` for the supported-version and vulnerability-reporting policy.
- Read `THREAT_MODEL.md` before triaging a suspected vulnerability.
- Do not place private advisories, unpublished findings, reporter information, or private bug-hunt results in public documentation.

## Security analysis

- Before describing a finding as novel, check open and closed issues, pull requests, published advisories, available approved private advisory material, tests, documentation, changelog entries, and relevant source history.
- Prefer rejecting speculative candidates over presenting unsupported security conclusions.

## Security reproduction standards

- Start from a documented or reasonably supported public API invocation. An internal-helper test alone does not demonstrate public behavior unless that helper is itself public.
- State the exact arguments, options, environment, expected documented behavior, observed behavior, independent security effect, and relevant preconditions.
- Add a focused regression test that expresses the expected correct behavior rather than an incidental implementation detail.
- Run the test against the unchanged implementation and confirm that it fails for the stated reason. If the behavior does not reproduce, stop and report the evidence instead of modifying source.
- After a fix, confirm that the focused test passes and run `go test ./...`, `gofmt -l .` and `go vet ./...`.
- Keep reproductions safe, local, bounded, self-contained, and readable. Do not contact external targets or run unbounded denial-of-service payloads.
- Report the root cause, files changed, before-and-after results, compatibility effects, assumptions, and remaining limitations.
