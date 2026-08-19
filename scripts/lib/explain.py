#!/usr/bin/env python3
"""Renders one blob's physical layout from the gateway's catalog response.

Reads the GET /api/v1/files/:id payload on stdin. Kept as a file rather than
inline in the CLI so the formatting is readable and quoting-safe.
"""
import json
import sys


def main() -> int:
    doc = json.load(sys.stdin)
    if "error" in doc:
        print("error:", doc["error"])
        return 1

    checksum = doc.get("blob_checksum", "")
    print("{}  {} bytes  {}:{}...".format(
        doc.get("blob_filename"), doc.get("blob_size"),
        doc.get("blob_checksum_alg"), checksum[:16]))

    chunks = sorted(doc.get("blob_chunks") or [],
                    key=lambda c: (c["block_index"], c["shard_index"]))
    print("{} chunk(s):".format(len(chunks)))
    print("  {:>5} {:>5} {:>10} {:>10}  {}".format("block", "shard", "size", "offset", "kind / path"))
    for c in chunks:
        idx = c["shard_index"]
        kind = "whole" if idx < 0 else ("data" if idx < 8 else "parity")
        print("  {:>5} {:>5} {:>10} {:>10}  {:<6} {}".format(
            c["block_index"], idx, c["blob_size"], c["physical_offset"], kind, c["physical_path"]))
    return 0


if __name__ == "__main__":
    sys.exit(main())
