#!/bin/sh
# 在具 ImageMagick 的 FD2 Docker 映像內，從一段重製戰鬥逐幀 PNG 中找出
# 與 320x200 原版 oracle 最接近的相位。這只量測 RGB 差異，不替畫格推論語意。
set -eu

if [ "$#" -ne 3 ]; then
	printf '用法：%s ORACLE_320x200.png REMAKE_FRAMES_DIR OUTPUT.tsv\n' "$0" >&2
	exit 2
fi

oracle=$1
frames=$2
output=$3
test -f "$oracle"
test -d "$frames"
: >"$output"

found=0
for frame in "$frames"/frame_*.png; do
	test -f "$frame" || continue
	found=1
	normalized="${TMPDIR:-/tmp}/fd2-battle-series-normalized-$$.png"
	trap 'rm -f "$normalized"' EXIT INT TERM
	convert "$frame" -filter point -resize 320x200\! "$normalized"
	ae=$(compare -metric AE "$oracle" "$normalized" null: 2>&1 || true)
	case "$ae" in
		''|*[!0-9]*)
			printf '無法解析 %s 的 AE：%s\n' "$frame" "$ae" >&2
			exit 1
			;;
	esac
	printf '%s\t%s\n' "$ae" "$frame" >>"$output"
	rm -f "$normalized"
	trap - EXIT INT TERM
done

if [ "$found" -ne 1 ]; then
	printf '找不到逐幀 PNG：%s/frame_*.png\n' "$frames" >&2
	exit 1
fi

sort -n "$output" -o "$output"
head -n 1 "$output"
