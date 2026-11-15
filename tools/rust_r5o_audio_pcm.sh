#!/bin/bash
# R5 階段 4 r5o（docs/goal/121）：BGM 與音效的 Go／Rust 逐樣本 PCM 比對。
#
# 用法：tools/rust_r5o_audio_pcm.sh <輸出目錄>
#
# 步驟：
#   1. Go（fd2-go-test-local，--network none）：remake/cmd/fd2-r5-audio-pcm 以 cmd/fd2 的解碼路徑
#      （vorbis／wav.DecodeWithSampleRate(44100)＋io.ReadAll）輸出 <out>/go/go-pcm.jsonl 與 pcm/*.s16；
#      跑兩次，兩次的 jsonl 必須相同。
#   2. Rust（fd2-rust:1.99.1-r3，--network none）：fd2-app 的 fmt、clippy、單元測試；r5o_audio_pcm 以 fd2-app 的
#      解碼路徑解同一批檔案，逐檔與 Go 比對，輸出 <out>/rust/rust-pcm.jsonl、<out>/compare/compare.jsonl、
#      summary.json、summary.txt；r5o_audio_play 在容器的 ALSA null 裝置上以 Bevy 播放腳本化的 AudioVoice，
#      輸出 <out>/play/r5o-audio-play.json（沒有人耳聽辨）。
#
# 輸入（唯讀）：資料根（FD2_DATA_ROOT，預設本儲存庫；在工作樹副本中執行時指向主工作區）的 remake/assets
# （music_catalog.json、music_fm、music_mt32、sfx/*.wav）與 remake/generated-assets/fd2-original-b97caf22/sfx。
#
# 環境變數：
#   FD2_DATA_ROOT    素材所在的儲存庫根
#   FD2_GOCACHE      Go 建置快取（預設 <資料根>/work/gocache）
#   FD2_RUST_BUILD   Rust 建置目錄（預設 <資料根>/work/rust-build）
#   FD2_RUST_TARGET  建置目錄下的 cargo target 目錄名稱（預設 target）
#   FD2_R5O_KEEP_PCM=1  保留 Go 與 Rust 的 .s16 檔（預設比對完刪掉，只留 jsonl 與摘要）
set -euo pipefail
root="$(cd "$(dirname "$0")/.." && pwd -P)"
data="$(cd "${FD2_DATA_ROOT:-$root}" && pwd -P)"
out="$(mkdir -p "${1:?輸出目錄}" && cd "$1" && pwd -P)"
gocache="${FD2_GOCACHE:-$data/work/gocache}"
build="${FD2_RUST_BUILD:-$data/work/rust-build}"
target="${FD2_RUST_TARGET:-target}"
assets="$data/remake/assets"
sfx="$data/remake/generated-assets/fd2-original-b97caf22/sfx"
for d in "$root/remake/cmd/fd2-r5-audio-pcm" "$assets/music_fm" "$assets/music_mt32" "$assets/sfx" "$sfx" "$gocache" "$build/cargo"; do
  test -d "$d" || { echo "找不到目錄 $d" >&2; exit 2; }
done
test -f "$assets/music_catalog.json" || { echo "缺少 $assets/music_catalog.json" >&2; exit 2; }
mkdir -p "$build/$target"

for run in r1 r2; do
  rm -rf "$out/go-$run"
  mkdir -p "$out/go-$run"
  timeout 1800 docker run --rm --name "fd2-r5o-go-$run-$$" --network none --memory 4g --cpus 4 --pids-limit 512 \
    --log-opt max-size=10m --log-opt max-file=3 -u "$(id -u):$(id -g)" \
    -e HOME=/tmp/home -e GOCACHE=/gocache -e GOFLAGS=-mod=mod \
    --mount "type=bind,source=$root/remake,target=/src/remake,readonly" \
    --mount "type=bind,source=$assets,target=/assets,readonly" \
    --mount "type=bind,source=$sfx,target=/sfx,readonly" \
    --mount "type=bind,source=$gocache,target=/gocache" \
    --mount "type=bind,source=$out/go-$run,target=/out" \
    -w /tmp fd2-go-test-local:latest bash -euo pipefail -c '
      cp -a /src/remake /tmp/remake && cd /tmp/remake
      go run ./cmd/fd2-r5-audio-pcm -assets /assets -sfx /sfx -out /out'
done
cmp "$out/go-r1/go-pcm.jsonl" "$out/go-r2/go-pcm.jsonl"
rm -rf "$out/go-r2/pcm" "$out/go"
mv "$out/go-r1" "$out/go"
rm -rf "$out/go-r2"
echo "Go 兩次相同：$(sha256sum "$out/go/go-pcm.jsonl" | cut -c1-64)"

rm -rf "$out/rust" "$out/compare" "$out/play"
mkdir -p "$out/rust" "$out/compare" "$out/play"
timeout 3000 docker run --rm --name "fd2-r5o-rust-$$" --network none --memory 6g --cpus 4 --pids-limit 1024 \
  --log-opt max-size=10m --log-opt max-file=3 -u "$(id -u):$(id -g)" -e HOME=/tmp/h \
  -e CARGO_HOME=/build/cargo -e CARGO_TARGET_DIR=/build/target-x \
  --mount "type=bind,source=$root/remake-rs,target=/src" \
  --mount "type=bind,source=$root/LICENSE,target=/LICENSE,readonly" \
  --mount "type=bind,source=$build/cargo,target=/build/cargo" \
  --mount "type=bind,source=$build/$target,target=/build/target-x" \
  --mount "type=bind,source=$assets,target=/assets,readonly" \
  --mount "type=bind,source=$sfx,target=/sfx,readonly" \
  --mount "type=bind,source=$out,target=/out" -w /src \
  fd2-rust:1.99.1-r3 bash -euo pipefail -c '
    cargo fmt --check -p fd2-app
    cargo clippy --offline --locked -p fd2-app --all-targets -- -D warnings
    cargo test --offline --locked -p fd2-app 2>&1 | tee /out/play/fd2-app-test.log | grep -E "^test result|FAILED|panicked"
    cargo build --offline --locked --release -p fd2-app --bin r5o_audio_pcm --bin r5o_audio_play
    /build/target-x/release/r5o_audio_pcm --assets /assets --sfx /sfx --go /out/go --out /out
    RUST_LOG=warn timeout 300 /build/target-x/release/r5o_audio_play --assets /assets --sfx /sfx \
      --report /out/play/r5o-audio-play.json > /out/play/r5o-audio-play.log 2>&1
    echo "播放驗證：$(grep -m1 "\"status\"" /out/play/r5o-audio-play.json)"'
if [ "${FD2_R5O_KEEP_PCM:-}" != 1 ]; then
  rm -rf "$out/go/pcm" "$out/rust/pcm"
fi
cat "$out/compare/summary.txt"
