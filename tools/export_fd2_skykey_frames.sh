#!/usr/bin/env bash
# Container-internal, bounded by its Docker owner. Go test selects package cwd.
set -euo pipefail
test -f /.dockerenv
test "$(id -u)" != 0
repo=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
out=${1:?explicit frame output directory}
mode=${2:-fresh}
test "$mode" = fresh || test "$mode" = resume
test -f "${FD2_TEST_UNIT_PRESENT_CATALOG:?explicit catalog}"
test -d "${FD2_TEST_UNIT_PRESENT_PACK:?explicit modern pack}"
for archive in FDOTHER.DAT FDTXT.DAT BG.DAT TAI.DAT; do
  test -f "$repo/org_game/炎龍騎士團/FLAME2/$archive"
done
export FD2_HD_SKY_KEY_GPU=1 FD2_SKY_KEY_EXPORT_FRAMES=1
export FD2_TEST_UNIT_PRESENT_OUT="$out"
if test "$mode" = resume; then
  test -d "$out"
  test "$(stat -c %u "$out")" = "$(id -u)"
  export FD2_SKY_KEY_RESUME_FRAMES=1
else
  test ! -e "$out"
  unset FD2_SKY_KEY_RESUME_FRAMES
fi
cd "$repo/remake"
go test ./cmd/fd2 -run '^TestModernHDSkyKeyGameGPU$' -timeout 25m -count=1 -v
python3 - "$out" <<'PY'
import json, pathlib, sys
p=pathlib.Path(sys.argv[1])
d=json.loads((p/'receipt.json').read_text())
assert d['status']=='PASS-SKY-KEY-HD-GAME' and len(d['rows'])==202
assert len(list(p.glob('movie-*.png')))==202
assert all((p/f'movie-{i:04d}.png').stat().st_size>100 for i in range(202))
print('PASS: 202 actual GPU frames and complete native-control receipt')
PY
