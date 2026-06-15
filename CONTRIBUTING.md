# Contributing

Thanks for your interest! AEWC26 is a small, focused project — contributions
that keep it simple and dependency-light are very welcome.

## Getting started

```sh
go vet ./...
go test ./...
go run .        # http://localhost:8090
```

The whole app is in `internal/app/` (single package). UI is server-rendered
`html/template` in `internal/app/web/templates`, assets in `web/static`,
strings in `i18n.go`.

## Guidelines

- Keep the standard-library-first, single-binary spirit. New third-party deps
  need a good reason.
- Run `go vet ./...` and `go test ./...` before opening a PR; add tests for any
  change to settlement/payout logic.
- Match the existing code style (`gofmt`).
- For UI strings, add both English and Vietnamese entries in `i18n.go`.
- One logical change per PR; describe what and why.

## Good first issues

- More languages in `i18n.go`.
- Light theme / theme toggle.
- Make the tournament data swappable (other competitions).
- Login rate-limiting and security headers.

By contributing you agree your work is licensed under the project's MIT license.
