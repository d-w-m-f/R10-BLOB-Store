# E2E Failure Catalog

Historic record of what has actually broken in the R10 Blob Store end to end, and
which check now catches it. This is the first thing to read when e2e fails: this
system has recognisable failure modes, and most new failures rhyme with an old one.

**Add an entry every time e2e finds something.** One entry per real failure, with
the symptom as observed (not the diagnosis), the root cause, and the guard added.
The symptom line matters most -- that is what a future reader will be matching against.

---

## Environment failures

These produce *misleading* results rather than honest ones, which makes them the
most expensive class. `r10 preflight` exists for this section alone.

### stale-process
**Symptom:** Bootstrap reports success, but rows land with empty/wrong columns that
the current code clearly sets. Debugging the model and the SQL finds nothing wrong.
**Cause (2026-08-18):** A leftover `go run` binary from an earlier session held port
8080. The start script (now `r10 up`) started the new gateway, which failed with
`bind: address already in use` and died -- while the *old* binary kept serving.
Every request was answered by pre-fix code.
**Cost:** A full debug cycle chasing a phantom bug in freshly-written code.
**Guard:** `r10 preflight` resolves each listening port to its binary via
`/proc/<pid>/exe` and fails if it is not the one `r10 up` built.

### stale-build
**Symptom:** A fix that is definitely in the source has no effect at runtime.
**Cause:** The running binary predates the source change.
**Guard:** `r10 preflight` compares each binary's mtime against the newest `.go`
file in its module. This fired for real on 2026-08-19, immediately after being written.

---

## Persistence failures

### gorm-zero-value-omission
**Symptom:** Reed-Solomon stripes unreadable; `shard_index` is `-1` for a shard that
was definitely written as `0`.
**Cause (2026-08-19):** `ShardIndex int` was declared `gorm:"not null;default:-1"`.
GORM omits a field from the INSERT when its value is the Go zero value **and** the
column declares a default, so shard 0 fell through to the database default of -1.
**Guard:** The placement assertion in every Case 3 matrix entry compares the full
shard set against `[0..11]`. Any zero-valued field silently replaced by a default
now fails the matrix.
**Watch for:** This applies to *every* `default:` tag on a non-nullable field.
Auditing for it is worthwhile before adding new columns.

### lexicographic-part-ordering
**Symptom:** Files above ~80MB reassemble scrambled; checksum mismatch on download,
smaller files fine.
**Cause:** Upload parts were staged as `part_%s` with the number interpolated as a
string, so `part_10` sorted before `part_2`.
**Guard:** Part indices are zero-padded (`part_%09d`). The `case3-multi-stripe`
entry (68MB, 9 slices) is the cheapest case that would catch a regression;
a >10-slice case would be stronger.

### namespace-collision
**Symptom:** Bootstrap fails partway with a unique-constraint violation on
`infra.discs.serial_number` or `infra.machines.namespace`.
**Cause:** `randomString` re-seeded `math/rand` from `time.Now().UnixNano()` on
every call, so calls inside one clock tick returned identical strings.
**Guard:** `r10 preflight` asserts 4 workers / 38 machines exist, so a partial
topology cannot be mistaken for a healthy one.

---

## Design ambiguities

Not bugs -- places where the code had to decide something the docs left open.
Recorded so the next reader does not "fix" a deliberate choice.

### boundary-32mb
`docs/erasure-coding.md` tabulates `128KB - 32MB` as Case 2 (no erasure coding) and
`> 32MB` as Case 3, which leaves a file of *exactly* 32MB ambiguous.
**The code erasure-codes it:** a full 32MB block is a complete stripe, and treating
it as one whole chunk would give a 32MB file less durability than a 33MB file.
Pinned by the `case3-exact-block` entry (tagged `full`).
If the intent was the other reading, change `storage_service.go` and that case together.

---

## Template

```
### short-kebab-id
**Symptom:** what an observer sees, before knowing the cause
**Cause (YYYY-MM-DD):** the actual mechanism
**Guard:** the check that now catches it, by name
```
