# Fault injection against the erasure-coded read path. Requires common.sh, upload.sh.
#
# Damage is applied cumulatively, weakest first, so no restore step is needed:
# intact -> 4 shards destroyed (the RS 8+4 limit) -> a 5th corrupted (one past it).

run_fault_suite() {
  local results="$1"
  local work; work="$(mktemp -d)"
  local passed=0 total=0
  : > "$results"

  local src="$work/fault_source.bin"
  head -c $((40 * 1024 * 1024)) /dev/urandom > "$src"
  local expected; expected=$(sha256sum "$src" | cut -d' ' -f1)

  local blob
  blob="$(r10_upload "$src" "$work")" || { err "upload failed"; rm -rf "$work"; return 1; }
  say "uploaded blob $blob (40MB -> one 8+4 stripe plus an 8MB tail chunk)"

  _check() {  # <label> <ok|refuse>
    local label="$1" expect="$2" code got
    total=$((total+1))
    code=$(curl -s -o "$work/out.bin" -w '%{http_code}' "$GATEWAY/api/v1/files/$blob/download")
    got=$(sha256sum "$work/out.bin" 2>/dev/null | cut -d' ' -f1)
    if [ "$expect" = "ok" ]; then
      if [ "$code" = "200" ] && [ "$got" = "$expected" ]; then
        ok "$label -> rebuilt byte-perfect"; passed=$((passed+1))
        emit_result "$results" "$label" pass "" "$blob"
      else
        err "$label -> HTTP $code, checksum mismatch"
        emit_result "$results" "$label" fail "HTTP $code checksum mismatch" "$blob"
      fi
    else
      if [ "$code" != "200" ]; then
        ok "$label -> correctly refused (HTTP $code) rather than serving bad data"; passed=$((passed+1))
        emit_result "$results" "$label" pass "refused HTTP $code" "$blob"
      else
        err "$label -> served a response it should have refused"
        emit_result "$results" "$label" fail "served corrupt data" "$blob"
      fi
    fi
  }

  head1 "baseline: intact cluster"
  _check "intact" ok

  head1 "4 shards destroyed (the documented RS 8+4 tolerance)"
  local i p
  for i in 0 1 2 3; do
    p="$(shard_path "$blob" $i)"
    [ -n "$p" ] && rm -f "$CLUSTER_ROOT/$p"
  done
  _check "4-shards-lost" ok

  head1 "silent bit-rot in a 5th shard: the checksum must catch it, pushing loss past the limit"
  p="$(shard_path "$blob" 5)"
  printf 'ROT' | dd of="$CLUSTER_ROOT/$p" bs=1 seek=1000 conv=notrunc status=none
  _check "4-lost-plus-1-corrupted" refuse

  rm -rf "$work"
  say ""
  say "fault injection: $passed/$total passed"
  [ "$passed" -eq "$total" ]
}
