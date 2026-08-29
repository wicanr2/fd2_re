#!/usr/bin/env bash
set -euo pipefail

repo=$(cd "$(dirname "$0")/.." && pwd)
live=${1:-$repo/dist/promo/fd2-v0.1.0-live-gameplay.mp4}
out=${2:-$repo/dist/promo/fd2-v0.1.0-promo.mp4}
original=$repo/extracted/title_re/dragonfx_compare/dos_original_intro_20s.mp4
comparison=$repo/docs/figures/battle-field-ch01-scoped-compare-20260810.png
dynamic_compare=$repo/dist/promo/fd2-v0.1.0-ch01-dosbox-vs-remake.mp4

for path in "$live" "$original" "$comparison" "$dynamic_compare"; do
  test -f "$path" || { echo "缺少推廣片輸入：$path" >&2; exit 2; }
done

mkdir -p "$(dirname "$out")"
uid=$(id -u)
gid=$(id -g)
out_dir=$(cd "$(dirname "$out")" && pwd)
out_name=$(basename "$out")

docker run --rm --network none --memory 2g --cpus 2 --pids-limit 256 \
  -u "$uid:$gid" -v "$repo:/work:ro" -v "$out_dir:/out:rw" -w /work \
  --entrypoint bash game-video:latest -lc '
set -euo pipefail
font=/usr/share/fonts/opentype/noto/NotoSansCJK-Bold.ttc
ffmpeg -hide_banner -loglevel warning -y -filter_threads 1 \
  -filter_complex_threads 1 -threads 2 \
  -f lavfi -t 4 -i color=c=0x081018:s=1280x720:r=30 \
  -ss 0 -t 8 -i extracted/title_re/dragonfx_compare/dos_original_intro_20s.mp4 \
  -ss 23.5 -t 30.5 -i dist/promo/fd2-v0.1.0-live-gameplay.mp4 \
  -t 12 -i dist/promo/fd2-v0.1.0-ch01-dosbox-vs-remake.mp4 \
  -loop 1 -t 8 -i docs/figures/battle-field-ch01-scoped-compare-20260810.png \
  -f lavfi -t 15 -i color=c=0x081018:s=1280x720:r=30 \
  -f lavfi -t 6 -i color=c=0x081018:s=1280x720:r=30 \
  -f lavfi -t 83.5 -i sine=frequency=110:sample_rate=48000 \
  -f lavfi -t 83.5 -i sine=frequency=165:sample_rate=48000 \
  -filter_complex "
    [0:v]drawtext=fontfile=${font}:text=《炎龍騎士團 2》重製版:fontcolor=white:fontsize=52:x=(w-text_w)/2:y=205,
         drawtext=fontfile=${font}:text=v0.1.0　第一個公開里程碑:fontcolor=0xE7C35A:fontsize=34:x=(w-text_w)/2:y=305,
         drawtext=fontfile=${font}:text=重現經典，也保存它如何被創造:fontcolor=white:fontsize=28:x=(w-text_w)/2:y=375[v0];
    [1:v]scale=1280:720:force_original_aspect_ratio=decrease:flags=neighbor,pad=1280:720:(ow-iw)/2:(oh-ih)/2:black,
         drawbox=x=28:y=28:w=255:h=54:color=black@0.78:t=fill,
         drawtext=fontfile=${font}:text=1995 原版 DOS:fontcolor=white:fontsize=28:x=47:y=40[v1];
    [2:v]scale=1152:720:flags=neighbor,pad=1280:720:64:0:black,
         drawbox=x=20:y=20:w=330:h=52:color=black@0.78:t=fill,
         drawtext=fontfile=${font}:text=目前版本實際遊玩:fontcolor=white:fontsize=27:x=38:y=31,
         drawbox=x=770:y=20:w=485:h=52:color=black@0.78:t=fill,
         drawtext=fontfile=${font}:text=F4：繁中／簡中／日文／英文:fontcolor=0xE7C35A:fontsize=25:x=790:y=32[v2];
    [3:v]setpts=PTS-STARTPTS,setsar=1[v3];
    [4:v]scale=1280:720:force_original_aspect_ratio=decrease:flags=neighbor,pad=1280:720:(ow-iw)/2:(oh-ih)/2:0x081018,
         drawbox=x=0:y=654:w=1280:h=66:color=black@0.8:t=fill,
         drawtext=fontfile=${font}:text=第一關同狀態範圍比較　原版／重製／差異:fontcolor=white:fontsize=30:x=(w-text_w)/2:y=670[v4];
    [5:v]drawtext=fontfile=${font}:text=我們完成了什麼？:fontcolor=0xE7C35A:fontsize=45:x=(w-text_w)/2:y=75,
         drawtext=fontfile=${font}:text=可編輯的戰役與演出:fontcolor=white:fontsize=36:x=95:y=195:enable=between(t\,0\,5),
         drawtext=fontfile=${font}:text=對話、戰後、城鎮、商店、戰鬥與終局不再寫死在位址中:fontcolor=white:fontsize=27:x=95:y=270:enable=between(t\,0\,5),
         drawtext=fontfile=${font}:text=保存 1995 年台灣遊戲技術:fontcolor=white:fontsize=36:x=95:y=195:enable=between(t\,5\,10),
         drawtext=fontfile=${font}:text=AFM／中文字模／DOS 資料格式／Watcom／Miles AIL:fontcolor=white:fontsize=27:x=95:y=270:enable=between(t\,5\,10),
         drawtext=fontfile=${font}:text=潔淨室跨平台引擎:fontcolor=white:fontsize=36:x=95:y=195:enable=between(t\,10\,15),
         drawtext=fontfile=${font}:text=Linux／Windows／macOS　四語基礎與外部社群語言包架構:fontcolor=white:fontsize=27:x=95:y=270:enable=between(t\,10\,15)[v5];
    [6:v]drawtext=fontfile=${font}:text=第一個公開版本現已釋出:fontcolor=white:fontsize=43:x=(w-text_w)/2:y=195,
         drawtext=fontfile=${font}:text=玩家需自備合法的《炎龍騎士團 2》原版資料:fontcolor=0xE7C35A:fontsize=30:x=(w-text_w)/2:y=300,
         drawtext=fontfile=${font}:text=非官方文化保存與潔淨室重製專案:fontcolor=white:fontsize=27:x=(w-text_w)/2:y=370[v6];
    [v0][v1][v2][v3][v4][v5][v6]concat=n=7:v=1:a=0,format=yuv420p[v];
    [7:a][8:a]amix=inputs=2:normalize=0,volume=0.22,
         afade=t=in:st=0:d=2,afade=t=out:st=80.5:d=3[a]
  " -map "[v]" -map "[a]" -c:v libx264 -preset medium -crf 20 -r 30 \
  -c:a aac -b:a 128k -ar 48000 -ac 2 -movflags +faststart -shortest "/out/$1"
' _ "$out_name"

echo "$out"
