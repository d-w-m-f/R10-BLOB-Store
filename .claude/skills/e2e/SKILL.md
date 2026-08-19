---
name: e2e
description: Run end-to-end verification of the R10 Blob Store — upload files through the gateway, confirm they are stored as the design says, download them back byte-identical, and inject faults against the erasure-coded read path. Use when asked to test e2e, verify the store works, check a storage change end to end, or diagnose an e2e failure.
---

# R10 end-to-end verification

You are the e2e harness. The scripts do the mechanics; your job is to **decide whether
the system is actually correct** and to **diagnose failures**. Do not re-derive the
architecture from source — everything you need to start is below.

## Run it

```bash
./scripts/r10 preflight    # ALWAYS first — never skip, never assume
./scripts/r10 test         # the matrix (~1 min)
./scripts/r10 fault        # fault injection against the 8+4 read path (~1 min)
```

`./scripts/r10 all` chains all three. Use `test --full` before a merge to main; it adds
the slow boundary cases (exactly 32MB, multi-stripe). Results land as JSON lines in
`/tmp/r10_logs/{matrix,fault}.jsonl`.

If preflight fails, **fix the environment and re-run it before doing anything else.**
A failing preflight makes every downstream result meaningless — the most expensive bug
in this project's history was a stale process producing confidently wrong output.

Not up yet: `./scripts/r10 up`, then `./scripts/r10 reset` if preflight reports the
topology is missing (4 workers / 38 machines expected). `reset` destroys stored blobs.

## What is actually being asserted

Each matrix case checks two independent things:

1. **Placement** — did the blob land where `docs/erasure-coding.md` says it should?
   Read back from Postgres: chunk count, shard set, distinct machines, machine type.
2. **Round trip** — are the downloaded bytes identical to what went in? This is the
   assertion that matters; everything else is diagnosis.

Fault injection damages the cluster underneath a stored blob, cumulatively:
intact → 4 shards destroyed (the RS 8+4 tolerance, must still rebuild) → a 5th
corrupted (one past the limit, must refuse rather than serve bad bytes).

Cases are **data** in `scripts/e2e_cases.json`. To cover a new size or shape, add an
entry there — do not write new test logic. Each entry declares `expect` and, for
generated files, a byte count.

## When something fails

1. **Read `docs/failure-catalog.md` first.** Failures here rhyme with past ones, and
   the catalog maps symptom → cause → guard. This is usually faster than fresh debugging.
2. `./scripts/r10 explain <blob-id>` — the blob's full physical layout, shard by shard,
   plus which chunk files are actually present on disk. Blob ids are in the JSONL results.
3. Logs are in `/tmp/r10_logs/{gateway,wkr10_N}.log`.
4. Direct SQL is fine and expected: `docker exec r10_postgres psql -U r10_user -d r10_db -c '...'`.
   Tables: `storage.blobs`, `storage.blob_chunks`, `infra.{workers,machines,discs}`,
   `control_plane.jobs`.

Distinguish carefully between **the harness is wrong**, **the environment is dirty**,
and **the system under test is broken**. Only the third is a real finding. State which
one you concluded and what evidence separates it from the other two.

## After the run

Report honestly: what passed, what failed, and what you could not determine. If a test
was skipped or a suite did not run, say so — do not imply coverage you did not get.

If you found a genuine new failure, **add an entry to `docs/failure-catalog.md`** using
the template at the bottom of that file, and prefer adding a matrix case that would have
caught it over adding bespoke test code.

## Scope note

E2E is not part of per-commit CI — too slow and heavy. It runs locally on demand, and is
intended for stage 2 of a future 3-stage pipeline, as the only job in that stage, right
before a merge to main. So it is fine for a run to take minutes and to be thorough;
optimise for catching real problems, not for speed.
