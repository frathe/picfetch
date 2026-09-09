#!/bin/bash
# The public Make runner owns host artifacts; direct CI partitions keep their
# existing TEST_CAPTURE contract. Keep this compatible with macOS Bash 3.2.
set -euo pipefail

if [ "${1-}" = --container ]; then
    locale=$2
    capture=$3
    cgroup=$4
    snapshot_memory() {
        local destination="$capture/memory-$1"
        local base="$cgroup"
        local files="memory.current memory.max memory.peak memory.events memory.events.local memory.stat memory.swap.current memory.swap.max"
        mkdir -p "$destination"
        if [ ! -f "$base/memory.events" ]; then
            if [ -d "$base/memory" ]; then base="$base/memory"; fi
            files="memory.usage_in_bytes memory.limit_in_bytes memory.max_usage_in_bytes memory.failcnt memory.oom_control memory.stat memory.memsw.usage_in_bytes memory.memsw.limit_in_bytes"
        fi
        local name
        for name in $files; do
            if [ ! -r "$base/$name" ]; then
                printf '%s unavailable\n' "$name" >> "$destination/unavailable.txt"
            elif ! cat "$base/$name" > "$destination/$name"; then
                printf '%s read failed\n' "$name" >> "$destination/unavailable.txt"
            fi
        done
    }
    finish_container() {
        local result=$?
        trap - EXIT
        set +e
        snapshot_memory after
        printf '%s\n' "$result" > "$capture/container-exit-code.txt"
        chown -R "$HOST_UID:$HOST_GID" "$capture"
        if [ -d internal/ui/testdata/failed ]; then
            chown -R "$HOST_UID:$HOST_GID" internal/ui/testdata/failed
        fi
        exit "$result"
    }
    trap finish_container EXIT
    snapshot_memory before
    apt-get update -qq
    apt-get install -y -qq apt-utils htop make gcc libgl1-mesa-dev xorg-dev libwayland-dev libxkbcommon-dev golang-go ca-certificates locales procps >/dev/null
    locale-gen "$locale" >/dev/null
    make --no-print-directory test-race-direct
    exit 0
fi

root=$1
test_image=$2
memory_gib=$3
label=$4
locale=$5
artifacts_parent=$6
mkdir -p "$artifacts_parent"
capture=$(mktemp -d "$artifacts_parent/$(date -u +%Y%m%dT%H%M%SZ)-XXXXXX")
capture=$(cd "$capture" && pwd)
printf 'Race artifacts: %s\n' "$capture"
runner_pid=
started=$(date +%s)

diagnostic() {
    local file=$1
    shift
    if "$@" > "$capture/$file" 2> "$capture/$file.stderr"; then
        return 0
    else
        printf '%s: %s unavailable (exit %s)\n' "$file" "$*" "$?" >> "$capture/diagnostics-errors.log"
        return 1
    fi
}

finish_host() {
    local result=$?
    trap - EXIT INT TERM
    set +e
    printf '%s\n' "$result" > "$capture/exit-code.txt"
    if [ -s "$capture/container.cid" ]; then
        container=$(cat "$capture/container.cid")
        diagnostic stop.log docker stop --time 10 "$container"
        if [ -n "$runner_pid" ]; then
            wait "$runner_pid"
        fi
        diagnostic container-state.json docker inspect --type=container --format '{{json .State}}' "$container"
        # Docker retains only its last 256 events. An empty query is not proof
        # that no OOM occurred; the live cgroup snapshots are separate evidence.
        diagnostic oom-events.jsonl docker events --since "$started" --until "$(($(date +%s) + 1))" \
            --filter "container=$container" --filter event=oom --format '{{json .}}'
        diagnostic cleanup.log docker rm "$container"
        if [ ! -f "$capture/container-exit-code.txt" ]; then
            printf 'Container exit snapshot unavailable; the process may have ended before its exit handler.\n' >> "$capture/diagnostics-errors.log"
        fi
    fi
    if [ -s "$capture/diagnostics-errors.log" ]; then
        printf 'Race diagnostics incomplete: %s/diagnostics-errors.log\n' "$capture" >&2
    fi
    printf 'Race artifacts retained: %s\n' "$capture"
    exit "$result"
}
trap finish_host EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

printf 'image=%s\nmemory_gib=%s\nlocale=%s\nstarted_unix=%s\n' "$test_image" "$memory_gib" "$locale" "$started" > "$capture/run.txt"
diagnostic docker-memory-bytes.txt docker info --format '{{.MemTotal}}' || true
docker create --platform linux/amd64 --cidfile "$capture/container.cid" \
    --memory "${memory_gib}g" --memory-swap "${memory_gib}g" --label "$label" \
    -v "$root:/work" -w /work -v "$capture:/capture" \
    -v picfetch-go-build-linux-amd64:/root/.cache/go-build \
    -v picfetch-go-mod-linux-amd64:/root/go/pkg/mod \
    -e HOST_UID="$(id -u)" -e HOST_GID="$(id -g)" \
    -e 'TEST_CAPTURE=/capture/$(TEST_PARTITION).json' \
    "$test_image" bash /work/scripts/testshards/docker-race.sh --container "$locale" /capture /sys/fs/cgroup \
    > "$capture/create.log" 2>&1
container=$(cat "$capture/container.cid")
set +e
(set -o pipefail; docker start --attach "$container" 2>&1 | tee "$capture/console.log") &
runner_pid=$!
wait "$runner_pid"
result=$?
runner_pid=
exit "$result"
