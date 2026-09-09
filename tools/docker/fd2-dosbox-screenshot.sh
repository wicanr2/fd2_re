#!/usr/bin/env bash
# 輔助診斷用，不是對拍執行器。
#
# FD2 的原版側對拍一律走 dosgolem（`tools/dosgolem_oracle.sh`）；這一支只在
# dosgolem 尚未具備某項 CPU／DOS／顯示／音訊／輸入／時序能力時，用來定位那
# 個缺口。它的擷取必須標成輔助基準，不得換名或登錄成 dosgolem 原版收據，
# 也不能單獨宣稱同狀態或一般玩家路徑。規則見 AGENTS.md「原版側對拍執行器」。
#
# Run the user-owned FD2 DOS build inside Xvfb and execute a small input/
# screenshot timeline. /game is a disposable writable sandbox; /shots is an
# explicit output mount. The original game directory is never mounted here.
set -euo pipefail

timeline="${1:-wait:12;shot:title}"
cycles="${2:-fixed 12000}"
export DISPLAY=:99

mkdir -p /shots /shots/dosbox-captures
Xvfb :99 -screen 0 1024x768x24 -nolisten tcp &
xvfb_pid=$!
for _ in $(seq 1 50); do
    [[ -S /tmp/.X11-unix/X99 ]] && break
    sleep 0.1
done
if [[ ! -S /tmp/.X11-unix/X99 ]]; then
    echo "Xvfb did not create display socket :99" >&2
    exit 1
fi

# socket 出現早於 X server 完成初始化；先以真正 X11 round-trip 驗證就緒。
display_ready=false
for _ in $(seq 1 50); do
    if xdotool getdisplaygeometry >/dev/null 2>&1; then display_ready=true; break; fi
    sleep 0.1
done
if [[ "$display_ready" != true ]]; then
    echo "Xvfb socket exists but display :99 is not ready" >&2
    kill "$xvfb_pid" 2>/dev/null || true
    exit 1
fi

cleanup() {
    kill "${dosbox_pid:-}" 2>/dev/null || true
    kill "$xvfb_pid" 2>/dev/null || true
}
trap cleanup EXIT

cat >/tmp/fd2-dosbox.conf <<EOF
[sdl]
fullscreen=false
fulldouble=false
output=surface
autolock=false
waitonerror=false

[dosbox]
machine=svga_s3
captures=/shots/dosbox-captures
memsize=32

[render]
frameskip=0
aspect=false
scaler=none

[cpu]
core=auto
cputype=auto
cycles=${cycles}

[mixer]
nosound=true
[midi]
mpu401=none
[sblaster]
sbtype=none
[gus]
gus=false
[speaker]
pcspeaker=false
[joystick]
joysticktype=none
[dos]
xms=true
ems=true
umb=true
keyboardlayout=us

[autoexec]
mount c /game
c:
FD2.EXE
EOF

dosbox -conf /tmp/fd2-dosbox.conf > /tmp/fd2-dosbox.log 2>&1 &
dosbox_pid=$!

window=""
for _ in $(seq 1 30); do
    window=$(xdotool search --name DOSBox 2>/dev/null | head -1 || true)
    [[ -n "$window" ]] && break
    sleep 0.5
done
if [[ -z "$window" ]]; then
    cat /tmp/fd2-dosbox.log >&2
    exit 1
fi

xdotool windowfocus "$window"

# Root-window captures include the Xvfb desktop and can place the DOSBox
# client at an arbitrary offset.  Evidence screenshots must contain the
# actual DOSBox client window; keep root captures only for fixed-coordinate
# readiness probes below.
capture_window() {
    local output="$1"
    import -window "$window" "$output"
}

town_ready() {
    import -window root /tmp/fd2-town-probe.png
    [[ "$(convert /tmp/fd2-town-probe.png -format \
        '%[pixel:p{10,10}]|%[pixel:p{160,130}]|%[pixel:p{300,190}]' info:)" == \
        'srgb(56,77,16)|srgb(170,142,101)|srgb(117,138,138)' ]]
}

IFS=';' read -ra steps <<< "$timeline"
for step in "${steps[@]}"; do
    [[ -z "$step" ]] && continue
    kind="${step%%:*}"
    arg="${step#*:}"
    case "$kind" in
        wait) echo "[fd2-shot] wait $arg"; sleep "$arg" ;;
        key) echo "[fd2-shot] key $arg"; xdotool windowfocus "$window"; xdotool key "$arg" ;;
        repeat)
            IFS=',' read -r count key delay_ms <<< "$arg"
            if [[ ! "$count" =~ ^[1-9][0-9]*$ || -z "$key" ||
                  ! "$delay_ms" =~ ^[0-9]+$ ]]; then
                echo "repeat expects count,key,delay_ms: $step" >&2
                exit 2
            fi
            echo "[fd2-shot] repeat $count $key every ${delay_ms}ms"
            xdotool windowfocus "$window"
            xdotool key --repeat "$count" --delay "$delay_ms" "$key"
            ;;
        waittown0)
            IFS=',' read -r key delay max_tries <<< "$arg"
            if [[ -z "$key" || ! "$delay" =~ ^[0-9]+([.][0-9]+)?$ ||
                  ! "$max_tries" =~ ^[1-9][0-9]*$ ]]; then
                echo "waittown0 expects key,delay_seconds,max_tries: $step" >&2
                exit 2
            fi
            echo "[fd2-shot] waittown0 $key every ${delay}s, max $max_tries"
            found=false
            for _ in $(seq 1 "$max_tries"); do
                if town_ready; then
                    found=true
                    break
                fi
                sleep 0.1
                if town_ready; then
                    found=true
                    break
                fi
                xdotool windowfocus "$window"
                xdotool key "$key"
                sleep "$delay"
            done
            if [[ "$found" != true ]]; then
                echo "town background signature not reached: $step" >&2
                exit 3
            fi
            ;;
        waitpixel)
            IFS=',' read -r x y red green blue delay max_tries <<< "$arg"
            if [[ ! "$x" =~ ^[0-9]+$ || ! "$y" =~ ^[0-9]+$ ||
                  ! "$red" =~ ^[0-9]+$ || ! "$green" =~ ^[0-9]+$ ||
                  ! "$blue" =~ ^[0-9]+$ ||
                  ! "$delay" =~ ^[0-9]+([.][0-9]+)?$ ||
                  ! "$max_tries" =~ ^[1-9][0-9]*$ ||
                  "$red" -gt 255 || "$green" -gt 255 || "$blue" -gt 255 ]]; then
                echo "waitpixel expects x,y,r,g,b,delay_seconds,max_tries: $step" >&2
                exit 2
            fi
            expected="srgb(${red},${green},${blue})"
            echo "[fd2-shot] waitpixel ($x,$y)=$expected every ${delay}s, max $max_tries"
            found=false
            for _ in $(seq 1 "$max_tries"); do
                import -window root /tmp/fd2-pixel-probe.png
                actual=$(convert /tmp/fd2-pixel-probe.png -format \
                    "%[pixel:p{$x,$y}]" info:)
                if [[ "$actual" == "$expected" ]]; then
                    found=true
                    break
                fi
                sleep "$delay"
            done
            if [[ "$found" != true ]]; then
                echo "pixel signature not reached: expected=$expected actual=$actual" >&2
                exit 4
            fi
            ;;
        type) echo "[fd2-shot] type $arg"; xdotool windowfocus "$window"; xdotool type --delay 80 "$arg" ;;
        shot) echo "[fd2-shot] shot $arg"; capture_window "/shots/${arg}.png" ;;
        *) echo "unknown timeline step: $step" >&2; exit 2 ;;
    esac
done
