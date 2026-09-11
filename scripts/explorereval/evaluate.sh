#!/bin/bash
set -euo pipefail

binary=${1:?binary}
assets=${2:?assets}
library=${3:?library}
evidence=${4:?evidence}
trial=${5:?trial}
provider=${6:?provider}
native=${7:-}
extra=()
if [[ $trial == library ]]; then extra=(-native "$native"); fi
mkdir -p "$evidence"
run_dir=$(mktemp -d "$evidence/$trial-XXXXXX")
set +e
"$binary" -assets "$assets" -library "$library" -out "$run_dir/result" -trial "$trial" -provider "$provider" "${extra[@]}" 2>&1 | tee "$run_dir/console.log"
status=$?
set -e
printf '%s\n' "$status" > "$run_dir/exit-status.txt"
if [[ $status -ne 0 ]]; then
    printf 'Trial failed (%s); evidence retained at %s\n' "$status" "$run_dir" >&2
    exit "$status"
fi
if [[ $trial == library ]]; then
    test -s "$run_dir/result/runner.json"
    test -s "$run_dir/result/session/session.json"
    printf '\nNative collection: %s/result (qualification pending)\n' "$run_dir"
    exit 0
fi
if [[ $trial == throughput ]]; then
    test -s "$run_dir/result/profile.json"
    test -s "$run_dir/result/events.jsonl"
    printf '\nProduction throughput profile: %s/result/profile.json\n' "$run_dir"
    exit 0
fi
for output in result.json initial.json manifest.json review.html pipeline-evaluation.md; do
    test -s "$run_dir/result/$output"
done
printf '# Latest technically completed pipeline run\n\nSemantic verdict pending.\n\n[Measurements](%s/result/pipeline-evaluation.md) · [Local visual report](%s/result/review.html)\n' "$(basename "$run_dir")" "$(basename "$run_dir")" > "$evidence/pipeline-evaluation.md.part"
mv "$evidence/pipeline-evaluation.md.part" "$evidence/pipeline-evaluation.md"
printf '\nLocal cohort review: %s/result/review.html\n' "$run_dir"
