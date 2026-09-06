# Agent guidelines

Repository-wide working rules. See `.github/instructions/architecture.instructions.md` for pipeline architecture and design contracts.

## Go source files

- Every Go file must contain exactly one `package` declaration, on the first line.
- After creating or editing a Go file, re-read the top of the file to confirm the header was not duplicated or split.

## Before reporting a change complete

- `go build ./...`
- `go vet ./...`
- `go test ./...`
- `gofmt -l .` must print nothing.
