# Drives the chunked upload protocol (init -> parts -> complete), exactly as the
# Angular client does. Sourced by the r10 CLI; requires lib/common.sh first.

SLICE_SIZE=$((8 * 1024 * 1024))   # 8MB slices, matching the Angular client

# r10_upload <path> [scratch_dir] -> prints the blob uuid on stdout
r10_upload() {
  local path="$1" scratch="${2:-$(mktemp -d)}"
  local name size mime init upload_id total i start part complete
  name="$(basename "$path")"
  size="$(stat -c%s "$path")"
  mime="$(file -b --mime-type "$path")"

  init=$(curl -s -X POST "$GATEWAY/api/v1/uploads/init" \
    -H 'Content-Type: application/json' \
    -d "{\"filename\":\"$name\",\"total_size\":$size,\"content_type\":\"$mime\"}")
  upload_id=$(echo "$init" | python3 -c 'import sys,json;print(json.load(sys.stdin).get("upload_id",""))' 2>/dev/null)
  if [ -z "$upload_id" ]; then echo "init failed: $init" >&2; return 1; fi

  total=$(( (size + SLICE_SIZE - 1) / SLICE_SIZE ))
  for ((i=0; i<total; i++)); do
    start=$((i * SLICE_SIZE))
    dd if="$path" bs=1 skip=$start count=$SLICE_SIZE of="$scratch/slice" \
       iflag=skip_bytes,count_bytes status=none
    part=$(curl -s -X PUT "$GATEWAY/api/v1/uploads/$upload_id/parts/$((i+1))" \
      -H 'Content-Type: application/octet-stream' --data-binary "@$scratch/slice")
    if ! echo "$part" | grep -q bytes_written; then
      echo "part $((i+1)) failed: $part" >&2; return 1
    fi
  done

  complete=$(curl -s -X POST "$GATEWAY/api/v1/uploads/$upload_id/complete" \
    -H 'Content-Type: application/json' -d '{}')
  echo "$complete" | python3 -c 'import sys,json;print(json.load(sys.stdin)["blob"]["blob_uuid"])' 2>/dev/null \
    || { echo "complete failed: $complete" >&2; return 1; }
}
