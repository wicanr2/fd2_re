#!/usr/bin/env bash
set -euo pipefail

repo=$(cd "$(dirname "$0")/.." && pwd)
original=${1:-$repo/video/fd2-ch1.mp4}
remake=${2:-$repo/dist/promo/fd2-v0.1.0-live-gameplay.mp4}
out=${3:-$repo/dist/promo/fd2-v0.1.0-ch01-dosbox-vs-remake.mp4}

for path in "$original" "$remake"; do
  test -f "$path" || { echo "缺少動態對拍輸入：$path" >&2; exit 2; }
done

mkdir -p "$(dirname "$out")"
uid=$(id -u)
gid=$(id -g)
original_dir=$(cd "$(dirname "$original")" && pwd)
remake_dir=$(cd "$(dirname "$remake")" && pwd)
out_dir=$(cd "$(dirname "$out")" && pwd)

docker run --rm --network none --memory 2g --cpus 2 --pids-limit 256 \
  -u "$uid:$gid" \
  -v "$original_dir:/original:ro" -v "$remake_dir:/remake:ro" -v "$out_dir:/out:rw" \
  --entrypoint bash game-video:latest -lc '
set -euo pipefail
font=/usr/share/fonts/opentype/noto/NotoSansCJK-Bold.ttc
ffmpeg -hide_banner -loglevel warning -y -filter_threads 1 -filter_complex_threads 1 -threads 2 \
  -ss 300 -t 12 -i "/original/$1" -ss 25 -t 12 -i "/remake/$2" \
  -filter_complex "
    [0:v]scale=600:375:flags=neighbor,pad=600:450:0:37:black[left];
    [1:v]scale=600:375:flags=neighbor,pad=600:450:0:37:black[right];
    [left][right]hstack=inputs=2,pad=1280:720:40:135:0x081018,
      drawtext=fontfile=${font}:text=原版 DOSBox:fontcolor=white:fontsize=30:x=235:y=78,
      drawtext=fontfile=${font}:text=重製版:fontcolor=white:fontsize=30:x=900:y=78,
      drawbox=x=0:y=650:w=1280:h=70:color=black@0.82:t=fill,
      drawtext=fontfile=${font}:text=第一關動態對拍（相近狀態比較）:fontcolor=0xE7C35A:fontsize=31:x=(w-text_w)/2:y=668,
      fps=30,setsar=1,format=yuv420p[v]" \
  -map "[v]" -an -c:v libx264 -preset medium -crf 20 -r 30 -movflags +faststart "/out/$3"
' _ "$(basename "$original")" "$(basename "$remake")" "$(basename "$out")"

echo "$out"
