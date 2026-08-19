# The e2e matrix runner. Requires common.sh and upload.sh.

CASES_FILE="${CASES_FILE:-$ROOT/scripts/e2e_cases.json}"

# run_cases <tag> <results_jsonl_path>
run_cases() {
  local tag="$1" results="$2"
  local work; work="$(mktemp -d)"
  local total=0 passed=0
  : > "$results"

  local count; count=$(python3 -c "
import json,sys
d=json.load(open('$CASES_FILE'))
print(sum(1 for c in d['cases'] if '$tag' in c.get('tags',[])))")

  say "running $count case(s) tagged '$tag'"

  local i
  for ((i=0; i<count; i++)); do
    local case_json name src gen source_path
    case_json=$(python3 -c "
import json
d=json.load(open('$CASES_FILE'))
sel=[c for c in d['cases'] if '$tag' in c.get('tags',[])]
print(json.dumps(sel[$i]))")
    name=$(python3 -c "import json,sys;print(json.loads(sys.argv[1])['name'])" "$case_json")
    src=$(python3 -c "import json,sys;print(json.loads(sys.argv[1]).get('source',''))" "$case_json")
    gen=$(python3 -c "import json,sys;print(json.loads(sys.argv[1]).get('generate',''))" "$case_json")

    total=$((total+1))
    head1 "$name"
    python3 -c "import json,sys;print('  '+json.loads(sys.argv[1]).get('why',''))" "$case_json"

    if [ -n "$src" ]; then
      source_path="$ROOT/$src"
      if [ ! -f "$source_path" ]; then
        err "source file missing: $src"
        emit_result "$results" "$name" fail "source file missing: $src"
        continue
      fi
    else
      source_path="$work/$name.bin"
      head -c "$gen" /dev/urandom > "$source_path"
    fi

    run_one_case "$name" "$case_json" "$source_path" "$work" "$results" && passed=$((passed+1))
  done

  rm -rf "$work"
  say ""
  say "matrix: $passed/$total passed"
  [ "$passed" -eq "$total" ]
}

# run_one_case <name> <case_json> <source_path> <work> <results>
run_one_case() {
  local name="$1" case_json="$2" source_path="$3" work="$4" results="$5"
  local size blob facts problems code

  size=$(stat -c%s "$source_path")

  blob="$(r10_upload "$source_path" "$work" 2>"$work/upload.err")" || {
    err "upload failed: $(head -c 300 "$work/upload.err")"
    emit_result "$results" "$name" fail "upload failed"
    return 1
  }
  ok "uploaded $size bytes as $blob"

  # 1. Placement: did it land where the design says it should?
  facts="$(blob_facts "$blob")"
  problems="$(python3 "$ROOT/scripts/lib/assert.py" "$case_json" "$facts")"
  if [ -n "$problems" ]; then
    while IFS= read -r line; do err "placement -- $line"; done <<< "$problems"
    say "        facts: $facts"
    emit_result "$results" "$name" fail "placement mismatch" "$blob" "$facts"
    return 1
  fi
  ok "placement matches: $facts"

  # 2. Round trip: the assertion that actually matters.
  code=$(curl -s -o "$work/$name.out" -w '%{http_code}' "$GATEWAY/api/v1/files/$blob/download")
  if [ "$code" != "200" ]; then
    err "download returned HTTP $code -- $(head -c 200 "$work/$name.out")"
    emit_result "$results" "$name" fail "download HTTP $code" "$blob" "$facts"
    return 1
  fi
  if ! cmp -s "$source_path" "$work/$name.out"; then
    err "downloaded bytes DIFFER from the original"
    emit_result "$results" "$name" fail "content mismatch" "$blob" "$facts"
    return 1
  fi
  ok "round trip byte-identical"

  emit_result "$results" "$name" pass "" "$blob" "$facts"
  return 0
}

# emit_result <file> <name> <status> [detail] [blob] [facts]
emit_result() {
  python3 -c "
import json,sys
row = {'case': sys.argv[2], 'status': sys.argv[3]}
if sys.argv[4]: row['detail'] = sys.argv[4]
if len(sys.argv) > 5 and sys.argv[5]: row['blob'] = sys.argv[5]
if len(sys.argv) > 6 and sys.argv[6]:
    try: row['facts'] = json.loads(sys.argv[6])
    except Exception: pass
open(sys.argv[1], 'a').write(json.dumps(row) + '\n')
" "$1" "$2" "$3" "${4:-}" "${5:-}" "${6:-}"
}
