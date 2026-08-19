# Shared environment and helpers for the r10 CLI. Sourced, never executed.

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
GATEWAY="${GATEWAY:-http://localhost:8080}"
CLUSTER_ROOT="${CLUSTER_ROOT_DIR:-/tmp/r10_cluster}"
LOG_DIR="${LOG_DIR:-/tmp/r10_logs}"
BIN_DIR="$LOG_DIR"
PG_CONTAINER="${PG_CONTAINER:-r10_postgres}"

# The local topology. Must match services.ClusterTopology in the gateway.
WORKER_PORTS=(8081 8082 8083 8084)
EXPECTED_WORKERS=4
EXPECTED_MACHINES=38

say()  { printf '%s\n' "$*"; }
head1() { printf '\n== %s ==\n' "$*"; }
ok()   { printf '  ok    %s\n' "$*"; }
warn() { printf '  warn  %s\n' "$*"; }
err()  { printf '  FAIL  %s\n' "$*"; }

psql_q() { docker exec "$PG_CONTAINER" psql -U r10_user -d r10_db -tAc "$1" 2>/dev/null; }

# port_pid <port> -> pid of the listening process, empty if none
port_pid() { ss -ltnp 2>/dev/null | grep -E "[:.]$1[[:space:]]" | grep -oP 'pid=\K[0-9]+' | head -1; }

# port_exe <port> -> absolute path of the listening binary
port_exe() {
  local pid; pid="$(port_pid "$1")"
  [ -n "$pid" ] && readlink -f "/proc/$pid/exe" 2>/dev/null
}

# blob_facts <blob_id> -> one JSON object describing how the blob was stored
blob_facts() {
  psql_q "select coalesce(json_build_object(
            'chunks', count(*),
            'shards', json_agg(distinct c.shard_index),
            'distinct_machines', count(distinct d.machine_id),
            'machine_types', json_agg(distinct m.type),
            'blocks', count(distinct c.block_index)
          )::text, '{}')
          from storage.blob_chunks c
          join infra.discs d on d.id = c.disc_id
          join infra.machines m on m.id = d.machine_id
          where c.blob_id = '$1';"
}

# shard_path <blob_id> <shard_index> -> path relative to the cluster root
shard_path() {
  psql_q "select m.name || '/' || c.physical_path
          from storage.blob_chunks c
          join infra.discs d on d.id = c.disc_id
          join infra.machines m on m.id = d.machine_id
          where c.blob_id = '$1' and c.shard_index = $2;"
}
