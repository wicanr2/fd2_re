#!/usr/bin/env bash
set -euo pipefail

repo=$(cd "$(dirname "$0")/.." && pwd)
out=${1:-$repo/dist/promo/fd2-opening-ch01-promo-20260827.mp4}
out_dir=$(dirname "$out")
out_name=$(basename "$out")

required=(
  extracted/title_re/dragonfx_compare/dos_original_intro_20s.mp4
  extracted/title_re/dragonfx_compare/remake_intro_8s.mp4
  docs/figures/title-original-dosbox.png
  docs/figures/title-remake-runtime.png
  docs/figures/ch01-dialogue-original-dosbox.png
  docs/figures/dialogue-remake-runtime.png
  docs/figures/battle-field-ch01-scoped-compare-20260810.png
  docs/figures/native-continue-current-command-compare-e1.png
)
for path in "${required[@]}"; do
  test -f "$repo/$path" || { echo "缺少素材：$path" >&2; exit 2; }
done
mkdir -p "$out_dir"

uid=$(id -u)
gid=$(id -g)
docker run --rm --network none --memory 2g --cpus 2 --pids-limit 256 \
  -u "$uid:$gid" \
  -v "$repo:/work:ro" -v "$out_dir:/out" -w /work \
  --entrypoint bash game-video:latest -c '
set -euo pipefail
font=/usr/share/fonts/opentype/noto/NotoSansCJK-Bold.ttc
ffmpeg -hide_banner -loglevel warning -y -filter_threads 1 -filter_complex_threads 1 -threads 2 \
  -f lavfi -t 4 -i color=c=0x081018:s=1280x720:r=30 \
  -ss 0 -t 8 -i extracted/title_re/dragonfx_compare/dos_original_intro_20s.mp4 \
  -ss 0 -t 8 -i extracted/title_re/dragonfx_compare/remake_intro_8s.mp4 \
  -loop 1 -t 6 -i docs/figures/title-original-dosbox.png \
  -loop 1 -t 6 -i docs/figures/title-remake-runtime.png \
  -loop 1 -t 6 -i docs/figures/ch01-dialogue-original-dosbox.png \
  -loop 1 -t 6 -i docs/figures/dialogue-remake-runtime.png \
  -loop 1 -t 7 -i docs/figures/battle-field-ch01-scoped-compare-20260810.png \
  -loop 1 -t 7 -i docs/figures/native-continue-current-command-compare-e1.png \
  -f lavfi -t 15 -i color=c=0x081018:s=1280x720:r=30 \
  -f lavfi -t 5 -i color=c=0x081018:s=1280x720:r=30 \
  -f lavfi -t 66 -i sine=frequency=110:sample_rate=48000 \
  -f lavfi -t 66 -i sine=frequency=165:sample_rate=48000 \
  -filter_complex "
    [0:v]drawtext=fontfile=${font}:text=《炎龍騎士團 2》:fontcolor=white:fontsize=58:x=(w-text_w)/2:y=220,
         drawtext=fontfile=${font}:text=1995 DOS 原版 × 潔淨室重製:fontcolor=0xE7C35A:fontsize=34:x=(w-text_w)/2:y=315,
         drawtext=fontfile=${font}:text=開場到第一關成果比較:fontcolor=white:fontsize=30:x=(w-text_w)/2:y=375[v0];
    [1:v]scale=1280:720:force_original_aspect_ratio=decrease:flags=neighbor,pad=1280:720:(ow-iw)/2:(oh-ih)/2:black,
         drawbox=x=28:y=28:w=220:h=54:color=black@0.75:t=fill,
         drawtext=fontfile=${font}:text=原版 DOS 錄影:fontcolor=white:fontsize=28:x=45:y=40[v1];
    [2:v]scale=1280:720:force_original_aspect_ratio=decrease:flags=neighbor,pad=1280:720:(ow-iw)/2:(oh-ih)/2:black,
         drawbox=x=28:y=28:w=220:h=54:color=black@0.75:t=fill,
         drawtext=fontfile=${font}:text=重製執行期:fontcolor=white:fontsize=28:x=52:y=40[v2];
    [3:v]scale=600:375:flags=neighbor,pad=600:450:0:37:black[ta];
    [4:v]scale=600:375:flags=neighbor,pad=600:450:0:37:black[tb];
    [ta][tb]hstack=inputs=2,pad=1280:720:40:135:0x081018,
         drawtext=fontfile=${font}:text=原版 DOS:fontcolor=white:fontsize=30:x=250:y=78,
         drawtext=fontfile=${font}:text=重製版:fontcolor=white:fontsize=30:x=900:y=78,
         drawtext=fontfile=${font}:text=標題與主選單:fontcolor=0xE7C35A:fontsize=34:x=(w-text_w)/2:y=625[v3];
    [5:v]scale=600:375:flags=neighbor,pad=600:450:0:37:black[da];
    [6:v]scale=600:375:flags=neighbor,pad=600:450:0:37:black[db];
    [da][db]hstack=inputs=2,pad=1280:720:40:135:0x081018,
         drawtext=fontfile=${font}:text=原版 DOS:fontcolor=white:fontsize=30:x=250:y=78,
         drawtext=fontfile=${font}:text=重製版:fontcolor=white:fontsize=30:x=900:y=78,
         drawtext=fontfile=${font}:text=繁中對話與原版版面:fontcolor=0xE7C35A:fontsize=34:x=(w-text_w)/2:y=625[v4];
    [7:v]scale=1280:720:force_original_aspect_ratio=decrease:flags=neighbor,pad=1280:720:(ow-iw)/2:(oh-ih)/2:0x081018,
         drawbox=x=0:y=654:w=1280:h=66:color=black@0.78:t=fill,
         drawtext=fontfile=${font}:text=第一關同狀態範圍比較　原版／重製／差異:fontcolor=white:fontsize=30:x=(w-text_w)/2:y=670[v5];
    [8:v]split=3[cmd0][cmd1][cmd2];
    [cmd0]crop=640:400:0:0,scale=400:250:flags=neighbor[c0];
    [cmd1]crop=640:400:0:400,scale=400:250:flags=neighbor[c1];
    [cmd2]crop=640:400:0:800,scale=400:250:flags=neighbor[c2];
    [c0][c1][c2]hstack=inputs=3,pad=1280:720:40:210:0x081018,
         drawtext=fontfile=${font}:text=原版 DOS:fontcolor=white:fontsize=28:x=175:y=155,
         drawtext=fontfile=${font}:text=重製版:fontcolor=white:fontsize=28:x=585:y=155,
         drawtext=fontfile=${font}:text=差異遮罩:fontcolor=white:fontsize=28:x=970:y=155,
         drawbox=x=0:y=654:w=1280:h=66:color=black@0.78:t=fill,
         drawtext=fontfile=${font}:text=普通鍵盤命令格比較　數值區仍持續校正:fontcolor=white:fontsize=30:x=(w-text_w)/2:y=670[v6];
    [9:v]drawtext=fontfile=${font}:text=我們完成了什麼？:fontcolor=0xE7C35A:fontsize=46:x=(w-text_w)/2:y=80,
         drawtext=fontfile=${font}:text=可編輯流程:fontcolor=white:fontsize=38:x=100:y=205:enable=between(t\,0\,5),
         drawtext=fontfile=${font}:text=對話、戰後、城鎮、商店與終局不再寫死在位址中:fontcolor=white:fontsize=28:x=100:y=280:enable=between(t\,0\,5),
         drawtext=fontfile=${font}:text=60份 handler　93筆已分類呼叫　0筆未知位置:fontcolor=0x9ED7FF:fontsize=28:x=100:y=345:enable=between(t\,0\,5),
         drawtext=fontfile=${font}:text=保存1995年的開發技術:fontcolor=white:fontsize=38:x=100:y=205:enable=between(t\,5\,10),
         drawtext=fontfile=${font}:text=AFM 1.00／Lo Yuan Tsung／DOS中文字模／DAT與RLE格式:fontcolor=white:fontsize=28:x=100:y=280:enable=between(t\,5\,10),
         drawtext=fontfile=${font}:text=Watcom＋DOS／4GW＋Miles AIL的可重現考證:fontcolor=0x9ED7FF:fontsize=28:x=100:y=345:enable=between(t\,5\,10),
         drawtext=fontfile=${font}:text=潔淨室跨平台重製:fontcolor=white:fontsize=38:x=100:y=205:enable=between(t\,10\,15),
         drawtext=fontfile=${font}:text=Go／Ebiten引擎　戰鬥、AI、持續隊伍、存讀檔與終局:fontcolor=white:fontsize=28:x=100:y=280:enable=between(t\,10\,15),
         drawtext=fontfile=${font}:text=可建置 Linux／Windows／macOS　並為新戰役保留擴充邊界:fontcolor=0x9ED7FF:fontsize=28:x=100:y=345:enable=between(t\,10\,15)[v7];
    [10:v]drawtext=fontfile=${font}:text=重現經典，也保存它如何被創造:fontcolor=white:fontsize=42:x=(w-text_w)/2:y=220,
          drawtext=fontfile=${font}:text=目前已可由正式標題進入第一關遊玩:fontcolor=0xE7C35A:fontsize=32:x=(w-text_w)/2:y=320,
          drawtext=fontfile=${font}:text=第一輪抽樣驗收 14／60　持續改進中:fontcolor=white:fontsize=28:x=(w-text_w)/2:y=390[v8];
    [v0][v1][v2][v3][v4][v5][v6][v7][v8]concat=n=9:v=1:a=0,format=yuv420p[v];
    [11:a][12:a]amix=inputs=2:normalize=0,volume=0.25,afade=t=in:st=0:d=2,afade=t=out:st=63:d=3[a]
  " \
  -map "[v]" -map "[a]" -c:v libx264 -preset medium -crf 20 -r 30 \
  -c:a aac -b:a 128k -ar 48000 -ac 2 -movflags +faststart -shortest "/out/$1"
' _ "$out_name"

echo "$out"
