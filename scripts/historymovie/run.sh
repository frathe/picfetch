#!/usr/bin/env bash
# Capture Git on the host; the renderer receives only exported history and output.
set -euo pipefail
script_dir="$(cd -- "$(dirname -- "$0")" && pwd)"
repo_dir="$(git -C "$script_dir" rev-parse --show-toplevel)"
mode="${1:-render}"
movie_seconds="${MOVIE_SECONDS:-180}"
movie_parent="${MOVIE_DIR:-.scratch/history-movies}"
if [[ "$mode" != render && "$mode" != test ]]; then
  printf 'Usage: bash scripts/historymovie/run.sh [render|test]\n' >&2
  exit 2
fi
if ! [[ "$movie_seconds" =~ ^[0-9]+([.][0-9]+)?$ ]] || ! awk -v seconds="$movie_seconds" 'BEGIN { exit !(seconds >= 30 && seconds <= 900) }'; then
  printf 'MOVIE_SECONDS must be between 30 and 900.\n' >&2
  exit 2
fi
command -v docker >/dev/null || { printf 'Docker is required. Start Docker and retry.\n' >&2; exit 1; }
docker info >/dev/null
if [[ "$(git -C "$repo_dir" rev-parse --is-shallow-repository)" == true ]]; then
  printf 'Full Git history is required. Run git fetch --unshallow, then retry.\n' >&2
  exit 1
fi
head_commit="$(git -C "$repo_dir" rev-parse --verify HEAD)"
# Recipe-specific tags avoid reusing an image from an older tool definition.
image_key="$(cksum < "$script_dir/Dockerfile" | awk '{print $1}')"
image="picfetch-history-render:recipe-$image_key"
if ! docker image inspect "$image" >/dev/null 2>&1; then
  printf 'Building the movie tools in Docker (first run only)...\n'
  docker build -t "$image" "$script_dir"
fi
if [[ "$movie_parent" != /* ]]; then movie_parent="$repo_dir/$movie_parent"; fi
mkdir -p "$movie_parent"
movie_parent="$(cd -- "$movie_parent" && pwd)"
run_dir="$(mktemp -d "$movie_parent/$(date +%Y%m%d-%H%M%S)-${head_commit:0:7}-XXXXXX")"
# mktemp uses 0700; the non-root container user must be able to write the mount.
chmod 755 "$run_dir"
cid_file="$run_dir/container.cid"
cleanup() {
  if [[ -s "$cid_file" ]]; then
    docker rm -f "$(cat "$cid_file")" >/dev/null 2>&1 || true
  fi
}
trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM
printf 'Movie artifacts: %s\n' "$run_dir"
printf '%s\n' "$head_commit" > "$run_dir/source-head.txt"
docker image inspect "$image" --format '{{.Id}}' > "$run_dir/render-image.txt"
if [[ "$mode" == render ]]; then
  printf 'Exporting committed history through %s...\n' "${head_commit:0:7}"
  git -C "$repo_dir" -c core.quotePath=false log --reverse --topo-order --root \
    --no-renames --no-ext-diff --no-textconv --name-status -z \
    --format='%x00PICFETCH-COMMIT%x00%H%x00%ct%x00%cI%x00%aN%x00%s' \
    "$head_commit" > "$run_dir/history.git"
  git -C "$repo_dir" ls-tree -rz --name-only "$head_commit" > "$run_dir/expected-tree.paths"
  mkdir "$run_dir/merge-trees"
  git -C "$repo_dir" rev-list --min-parents=2 "$head_commit" > "$run_dir/merge-commits.txt"
  while IFS= read -r merge_commit; do
    git -C "$repo_dir" ls-tree -rz --name-only "$merge_commit" > "$run_dir/merge-trees/$merge_commit.paths"
  done < "$run_dir/merge-commits.txt"
fi
container_command=(python3 /scripts/movie.py --seconds "$movie_seconds")
if [[ "$mode" == test ]]; then
  container_command=(python3 -m unittest discover -s /scripts -p 'test_*.py' -v)
fi
container_uid="$(id -u)"
container_gid="$(id -g)"
if [[ "$container_uid" == 0 ]]; then
  container_uid=1000
  container_gid=1000
  chown -R 1000:1000 "$run_dir"
fi
docker run --rm --init --cidfile "$cid_file" --network none \
  --cap-drop ALL --security-opt no-new-privileges --read-only \
  --tmpfs /tmp:rw,nosuid,size=512m --cpus 6 --memory 4g \
  --user "$container_uid:$container_gid" \
  -v "$script_dir:/scripts:ro" -v "$run_dir:/work" \
  "$image" "${container_command[@]}"
if [[ "$mode" == render ]]; then
  printf '\nMovie ready:\n%s/PicFetch-history.mp4\n' "$run_dir"
fi
