#!/bin/sh
# with-xvfb 在容器內以明確的 PID 與 trap 擁有一個 Xvfb，等 X11 socket 出現才執行
# 後面的命令，命令結束就收掉 Xvfb。
#
# 不要改回 xvfb-run。它與 Xvfb 之間是 SIGUSR1 交握（`trap : USR1` 後 `wait`），
# Xvfb 太早就緒時訊號會落在 wait 之前，wait 就再也不會回來：容器裡只剩
# xvfb-run 與 Xvfb 兩個行程、CPU 0%，命令一行都沒跑，外面看起來像測試跑很久。
# 反過來也踩過：命令結束了 xvfb-run 卻沒收掉 Xvfb，留下無界背景行程。
set -eu

DISPLAY_NUM="${WITH_XVFB_DISPLAY:-99}"
SCREEN="${WITH_XVFB_SCREEN:-1280x800x24}"

Xvfb ":${DISPLAY_NUM}" -screen 0 "${SCREEN}" -nolisten tcp >/tmp/xvfb.log 2>&1 &
xvfb_pid=$!
cleanup() {
    kill "${xvfb_pid}" 2>/dev/null || true
    wait "${xvfb_pid}" 2>/dev/null || true
}
trap cleanup EXIT INT TERM

i=0
while [ "${i}" -lt 200 ]; do
    if [ -S "/tmp/.X11-unix/X${DISPLAY_NUM}" ]; then
        break
    fi
    if ! kill -0 "${xvfb_pid}" 2>/dev/null; then
        echo "with-xvfb: Xvfb 提早結束" >&2
        cat /tmp/xvfb.log >&2 || true
        exit 1
    fi
    i=$((i + 1))
    sleep 0.05
done
if [ ! -S "/tmp/.X11-unix/X${DISPLAY_NUM}" ]; then
    echo "with-xvfb: 等不到 X11 socket /tmp/.X11-unix/X${DISPLAY_NUM}" >&2
    cat /tmp/xvfb.log >&2 || true
    exit 1
fi

DISPLAY=":${DISPLAY_NUM}"
export DISPLAY
"$@"
