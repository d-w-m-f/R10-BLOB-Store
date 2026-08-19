# R10 Blob Store

Distributed object store (S3-like). Go gateway + Go storage daemons + Angular control
plane + PostgreSQL for metadata. ~2.5k lines of Go; small enough to read directly.

## Shape

- `apps/gateway` — REST API, Reed-Solomon 8+4 encoding, placement, blob catalog. Port 8080.
- `apps/wkr10` — storage daemon. One process multiplexes many logical "machines";
  4 daemons serve 38 machines on ports 8081-8084. Block engine = one file per chunk;
  inline engine = append-only volume, returns a byte offset.
- `apps/web` — Angular 18 control plane. Port 4200.
- Machines are addressed by an 8-char `namespace`; directories live under `/tmp/r10_cluster`.

Storage routing by file size (`docs/erasure-coding.md`, implemented in
`internal/services/storage_service.go`): `<128KB` inline whole · `128KB-32MB` one block
chunk, no erasure coding · `>32MB` full 32MB blocks striped 8+4 across 12 distinct
machines, trailing partial block stored whole.

## Running and testing

`./scripts/r10` is the single entrypoint — `./scripts/r10 help` for verbs.
Always `./scripts/r10 preflight` before trusting any test result; it catches the
environment traps that produce misleading passes and failures.

```
./scripts/r10 up          # build + start daemons and gateway
./scripts/r10 preflight   # verify the environment is sane AND current
./scripts/r10 all         # preflight -> matrix -> fault injection
./scripts/r10 explain <blob-id>   # how one blob is physically stored
```

The e2e matrix lives in `scripts/e2e_cases.json` as data: add a case there rather than
writing new test logic. Results are JSON lines in `/tmp/r10_logs/{matrix,fault}.jsonl`.

E2E is deliberately **not** run per-commit — it is too slow and heavy for that. It runs
locally on demand, and is intended for stage 2 of a future 3-stage CI pipeline, as the
only job in that stage, immediately before merging to main.

We keep a **failure catalog** of historic e2e events in `docs/failure-catalog.md`.
Read it when e2e fails — failures here tend to rhyme with past ones — and add an entry
whenever e2e turns up something new.

## Conventions

- Comments and docs in this repo are a mix of English and Portuguese; match the file
  you are editing.
- Database access is GORM. Be careful with `default:` tags on non-nullable fields:
  GORM omits a zero-valued field when the column declares a default. This has already
  caused one silent data-corruption bug (see the catalog).
- The gateway reaches workers over HTTP with the payload as the raw body and metadata
  in `X-Chunk-*` headers. No JSON or base64 around binary data.
