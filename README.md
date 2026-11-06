# httpcheck

> **Smoke-test an endpoint without opening a dashboard.**

`httpcheck` is a small Go CLI that requests one or more URLs, compares the status code with the expected one and reports latency. It supports retries, timeouts and JSON output for CI.

No dependencies outside the Go standard library.

## Why this exists

A deployment is not done when the container starts. It is done when `/health` answers with the status you expect. `httpcheck` makes that the last step of a pipeline instead of a manual click.

## Install

```sh
go install github.com/melinoeluna4/httpcheck@latest
```

## Usage

```sh
httpcheck https://melinoeluna4.github.io/
```

```text
httpcheck: 1 URL(s), expecting 200
  https://melinoeluna4.github.io/ 200 231ms
OK
```

Check a different status, retry a flaky endpoint or emit JSON:

```sh
httpcheck -expect 204 https://example.com/health
httpcheck -retries 3 -timeout 2s -interval 1s https://api.example.com/health
httpcheck -json https://example.com
```

### Flags

| Flag | Default | Meaning |
| --- | --- | --- |
| `-expect` | `200` | expected HTTP status |
| `-timeout` | `5s` | per-request timeout |
| `-retries` | `0` | retries per URL |
| `-interval` | `500ms` | delay between retries |
| `-json` | `false` | print a JSON array of results |

### Exit codes

| Code | Meaning |
| --- | --- |
| `0` | every URL returned the expected status |
| `1` | at least one URL failed |
| `2` | usage error |

## How it works

1. Build a `GET` request with a context and a client timeout.
2. Read and close the body so the connection can be reused.
3. Compare the status with `-expect`.
4. Retry only when the attempt fails or the status is unexpected.
5. Print one line per URL and a final `OK` / `FAIL`.

A network error is reported with its message; a status mismatch shows the expected code, the real code and the number of attempts.

## Development

```sh
go test ./...
go vet ./...
go build ./...
```

CI runs the same three commands on every push and pull request.

---

Part of the [Melinoe Luna portfolio](https://melinoeluna4.github.io/proyectos/httpcheck/) · Go 1.27 · standard library only.
