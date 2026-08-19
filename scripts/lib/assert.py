#!/usr/bin/env python3
"""Compares the facts read back from Postgres against a case's expectations.

Kept in Python because the comparison is structural (sets, lists, counts) and
doing it in bash would be write-only. Called by scripts/lib/cases.sh.

Usage: assert.py <case-json> <facts-json>
Prints one diagnostic line per mismatch; exits non-zero if any.
"""
import json
import sys


def main() -> int:
    case = json.loads(sys.argv[1])
    facts = json.loads(sys.argv[2] or "{}")
    expect = case.get("expect", {})

    problems = []
    for key, want in expect.items():
        if key == "storage_case":
            # Derived, not stored: inline machine = 1, striped = 3, otherwise 2.
            types = set(facts.get("machine_types") or [])
            shards = set(facts.get("shards") or [])
            got = 1 if types == {"inline"} else (3 if shards - {-1} else 2)
        else:
            got = facts.get(key)

        if isinstance(want, list):
            if sorted(got or []) != sorted(want):
                problems.append(f"{key}: expected {sorted(want)}, got {sorted(got or [])}")
        elif got != want:
            problems.append(f"{key}: expected {want}, got {got}")

    for line in problems:
        print(line)
    return 1 if problems else 0


if __name__ == "__main__":
    sys.exit(main())
