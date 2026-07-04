# 48 — dosbox-x(heavy debugger)docker 建置與操作

> 目的:doc 47 §5 留下的未解——`0x627d8` acting 資源表的填表點(表在 BSS,填表位址靠暫存器基底寫入,
> 靜態難尋)——建議下一輪用「dosbox 記憶體 dump 反查來源」(rulebook 64 第三條路)。原版 `DOSBox.exe`
> (0.74)不含 debugger,本篇建置支援 heavy debugger 的 **dosbox-x**,並記錄如何用它 dump 執行期記憶體。

## 1. 為什麼是 dosbox-x 不是原版 DOSBox

原版 DOSBox 官方 build 不含互動式 debugger(需自行編譯 `--enable-debug` 的特殊 build,且專案已停止更新)。
**dosbox-x**(joncampbell123 維護的 active fork)把 `--enable-debug=heavy` 直接做成建置腳本選項,
底層沿用同一顆 ncurses debugger(命令集與原版 DOSBox debugger 相容),適合拿來對 FD2 這種 16-bit
protected-mode(DOS4GW)老遊戲做記憶體 dump。

## 2. 建置

Dockerfile:[`docker/dosbox-x/Dockerfile`](../../docker/dosbox-x/Dockerfile)

```dockerfile
FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends \
        ca-certificates curl \
        automake autoconf libtool pkg-config \
        gcc g++ make nasm \
        libncurses-dev \
        libsdl-net1.2-dev libsdl2-net-dev libsdl2-dev \
        libpcap-dev libslirp-dev \
        fluidsynth libfluidsynth-dev \
        libavdevice-dev libavformat-dev libavcodec-dev libswscale-dev \
        libfreetype-dev libxkbfile-dev libxrandr-dev \
        libpng-dev zlib1g-dev \
        xvfb x11-apps imagemagick tmux xdotool \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /src
RUN curl -sL -o dosbox-x.tar.gz \
        https://github.com/joncampbell123/dosbox-x/archive/0d7b272b690351a92405ee1d672152ee134da35b.tar.gz \
    && tar xzf dosbox-x.tar.gz \
    && mv dosbox-x-0d7b272b690351a92405ee1d672152ee134da35b dosbox-x \
    && rm dosbox-x.tar.gz

WORKDIR /src/dosbox-x
RUN ./build-debug-sdl2 && make install

WORKDIR /game
ENV SDL_VIDEODRIVER=x11 DISPLAY=:70
CMD ["dosbox-x"]
```

```
docker build -t fd2-dosbox-x docker/dosbox-x
```

- 套件清單抄自 dosbox-x 官方 `BUILD.md`「To compile DOSBox-X in Ubuntu」段(逐一在 `debian:bookworm-slim`
  驗證過皆存在),另加 `libsdl2-dev`(讓建置腳本偵測到系統 `sdl2-config`,跳過內建重編 SDL2,加速建置)、
  `xvfb`/`x11-apps`/`imagemagick`/`xdotool`(headless 開圖 + 截圖 + 送鍵,沿用 dq3 專案已驗證的
  `tools/dosbox_run.sh` 模式)、`tmux`(驅動 ncurses debugger TUI,見 §4)。
- 來源固定 commit `0d7b272b690351a92405ee1d672152ee134da35b`(2026-07-04 抓的 master HEAD)而非
  `git clone`,理由:①docker build 內 `git clone` 該 repo 曾在此環境實測逾時(repo 含大量歷史
  二進位測試資料,`.git` 遠大於單一 commit 的 tarball);②commit 釘死可重現,不受上游後續 push 影響。
- **`--enable-debug=heavy`**(由 `build-debug-sdl2` 腳本內建呼叫)= `configure.ac` 定義
  `C_DEBUG` + `C_HEAVY_DEBUG`(需要 curses)。這就是啟用 debugger 的唯一開關,查證依據:
  `dosbox-x/configure.ac` 第 1140-1154 行 `AC_ARG_ENABLE(debug, ...)` 區塊。

## 3. 驗證(本輪已實測,非紙上談兵)

| 項目 | 指令 | 結果 |
|---|---|---|
| 版本 | `docker run --rm fd2-dosbox-x dosbox-x --version` | `DOSBox-X version 2026.07.02 SDL2` |
| debugger 編進去 | `strings /usr/bin/dosbox-x \| grep -E '^MEMDUMPBIN$\|^BPLM$\|^DEBUGBOX$'` | 三個字串皆命中(只在 `#if C_DEBUG` 編譯區塊存在,證明 heavy debug 真的編進去) |
| debugger 真的能互動 | Xvfb + tmux 跑 `dosbox-x` + `xdotool key --window <win> alt+Pause` | tmux pane 截到完整 ncurses TUI(Code Overview / Output 視窗 + `I->` 提示字元),見 §4.1 |
| **MEMDUMPBIN 真的能 dump** | debugger 內 `tmux send-keys` 送 `MEMDUMPBIN F000 D186 20` | pane 印出 `DEBUG: Memory dump binary success.`,容器內確認 `MEMDUMP.BIN` 產生,大小 32 bytes(與要求的 0x20 相符) |
| 能跑 FD2 到畫面 | 掛遊戲目錄跑 `FD2.EXE`,Xvfb + `import -window root` 截圖 | `extracted/dosbox_x_verify/fd2_title.png`(序幕過場其中一幀:紅背景剪影 + 機器人臉,證明遊戲圖像正確渲染) |

啟動 FD2(掛載唯讀遊戲目錄,`-c` 疊加 autoexec;實測用的完整指令):

```bash
docker run --rm -e TERM=xterm \
  -v "$PWD/org_game/炎龍騎士團/FLAME2:/game:ro" \
  -v "$PWD/extracted/dosbox_x_verify:/out" \
  fd2-dosbox-x bash -c '
    Xvfb :70 -screen 0 1024x768x24 -ac >/tmp/xvfb.log 2>&1 &
    sleep 2
    export DISPLAY=:70
    mkdir -p /tmp/run && cp -r /game/* /tmp/run/    # 唯讀掛載改複製到可寫目錄,避免 FD2.TMP/存檔寫入失敗
    cd /tmp/run
    tmux new-session -d -s t -x 200 -y 50 \
      "dosbox-x -c \"MOUNT C /tmp/run\" -c \"C:\" -c \"FD2.EXE\" -c \"EXIT\""
    sleep 8
    import -window root /out/fd2_title.png
  '
```

- 遊戲目錄用 `:ro` 掛載沒問題,但 FD2 執行期會寫 `FD2.TMP` 等暫存檔,直接在唯讀掛載點跑會出錯,
  **實測解法**:掛進容器後先 `cp -r` 到容器內可寫路徑(`/tmp/run`)再從那邊跑 `dosbox-x`。
- 8 秒後截到的是序幕開場動畫的其中一幀而非嚴格的標題選單畫面(該幕約 30 秒,見 doc 46/23);
  要截到真正標題畫面需拉長 `sleep` 或送按鍵跳過(`xdotool key --window ... Return`)略過開場。

## 4. debugger 操作

### 4.1 怎麼進 debugger(本輪實測修正)

- Linux/Mac:**debugger TUI 需要一個真正的 pty**——`tmux new-session` 開的 pane 算數(已實測跑通),
  單純 `dosbox-x &` 丟到 log 檔重導向**不算**(沒有 pty,ncurses 起不來)。
- **實測結果推翻了「預設就會啟動即斷」的預期**:`[log]` 段 `debuggerrun=debugger` 是預設值沒錯,但
  它只決定「debugger 被觸發後的行為模式」,**不代表 dosbox-x 一啟動就自動斷點暫停**——實測 8 秒內
  終端機只印一般 `LOG:` 訊息,直到主動觸發才會切換成 ncurses TUI。
- **實際觸發方式(已驗證)**:熱鍵 `Alt+Pause` 是 SDL 視窗的 mapper shortcut,必須用
  `xdotool key --window <DOSBox 視窗 ID> alt+Pause` 對著 X11 視窗送(不是對 tmux pane 送,tmux 對
  這個熱鍵無效)。送出後,原本印 LOG 的那個 pty 畫面**立刻切換成 ncurses debugger TUI**
  (Code Overview / Data / Output 視窗 + 底部 `I->` 指令列),此時才能用 tmux 對同一個 pane
  send-keys 打 debugger 指令。
- `DEBUGBOX <command> [options]`:DOS shell 內建指令,可放進 `[autoexec]`,啟動指定程式並斷在
  entry point(**本輪未實測**,理論上比等 Alt+Pause 手動觸發更適合「一啟動就要斷」的場景,留待下輪)。

### 4.2 常用指令(節錄自 `README.debugger`)

| 指令 | 用途 |
|---|---|
| `BP <seg> <off>` / `BPM` / `BPLM <offset>` | 中斷點(real mode / protected mode / linear) |
| `BPPM <seg> <off>` | 記憶體變更中斷點(protected mode) |
| `RUN` / `RUNWATCH` | 恢復執行(後者邊跑邊顯示狀態) |
| `MEMDUMP <seg> <off> <bytecount>` | dump 記憶體到 `MEMDUMP.TXT`(文字) |
| **`MEMDUMPBIN <seg> <off> <bytecount>`** | dump 記憶體到 `MEMDUMP.BIN`(binary,我們要的) |
| `C <seg> <off>` / `D <seg> <off>` / `DV <offset>` / `DP <offset>` | 設定 code / data(seg:off / linear / physical)檢視位置 |
| `SR <reg> <value>` | 設定暫存器值 |
| `GDT` / `LDT` / `PAGING` | 傾印 GDT/LDT/分頁表(DOS4GW 保護模式定位必備,見 §5) |
| `F5`(鍵) / `F9`(鍵) / `F10`/`F11`(鍵) | Resume / 設中斷點 / Step over / Step into |

完整指令表見 `README.debugger`(dosbox-x 原始碼根目錄,已抄錄於本文撰寫時的查證過程)。

### 4.3 headless 自動化(本輪已實測跑通,沿用 dq3 專案模式,見 `docs/29-dosbox-oracle.md`)

**雙通道輸入**——這是本輪最重要的釐清,兩者不能混用,且已用真實流程驗證(§3):

1. **X11/xdotool 通道**:遊戲鍵盤輸入(方向鍵、Enter 過場等)**以及**觸發 `Alt+Pause` 進 debugger,
   都是對 Xvfb 上那個 SDL 視窗送事件,用 `xdotool key --window $(xdotool search --name DOSBox) <key>`。
   dq3 專案的 `tools/dosbox_run.sh` 已驗證同一模式,FD2 可直接照搬。
2. **tmux/pty 通道**:**已經進入 debugger TUI 之後**,對著跑 `dosbox-x` 的那個 tmux pane 用
   `tmux send-keys` 打字串指令(如 `MEMDUMPBIN F000 D186 20` + `Enter`)——本輪已實測 `MEMDUMPBIN`
   真的產生檔案(§3)。**進 debugger 前**這個 pane 只是普通 LOG 輸出,tmux 送鍵對遊戲本身無效
   (遊戲鍵盤走 X11,不走 stdin)。

已驗證流程(以本輪實測指令為基礎):

```bash
Xvfb :70 -screen 0 1024x768x24 -ac >/tmp/xvfb.log 2>&1 &
export DISPLAY=:70
tmux new-session -d -s dbg -x 200 -y 50 'dosbox-x -c "MOUNT C /tmp/run" -c "C:" -c "FD2.EXE"'
sleep 4                                            # 等 SDL 視窗建立(有上界)
WIN=$(xdotool search --name DOSBox | head -1)
xdotool key --window "$WIN" alt+Pause              # 觸發:切進 debugger TUI(已驗證)
sleep 2
tmux send-keys -t dbg 'BPLM 0x627d8' Enter          # 中斷點(實際換算見 §5,本輪未對 FD2 實測)
tmux send-keys -t dbg 'RUN' Enter
# ... 等中斷點觸發(有上界;不可無限等,配合 rulebook 35)...
tmux send-keys -t dbg 'MEMDUMPBIN DS 0 0x10000' Enter
tmux capture-pane -t dbg -p > /tmp/debugger_screen.txt   # 佐證目前 TUI 狀態(已驗證會印出結果訊息)
```

`dosbox-x` 若有「設定檔/命令列自動跑一串 debugger 指令」的原生機制(如序列化 debugger script),
**本輪未找到**——`README.debugger` 與 `configure.ac` 都沒有這類選項;`-c` 命令列參數是疊加
`[autoexec]` 的 **DOS shell 指令**(給 `MOUNT`/跑 `.EXE` 用),不會被 debugger 主控台解讀。
所以自動化 debugger 操作目前只能走「xdotool 觸發進 debugger + tmux 送鍵打指令」這條路,
沒有更捷徑的原生批次介面——但這條路本身已完整驗證可行,不是空想。

## 5. 針對 acting 資源表(`0x627d8`)的備註

- FD2 用 **DOS4GW**(Watcom/Rational Systems DOS extender)跑保護模式,`0x627d8` 是**反組譯工具算出來
  的 app 端 linear/flat 位址**(從 EXE 的 LE/LX 影像位移推算),**不是** dosbox-x 模擬器的實體記憶體
  位址,也不等於 `DV`/`DP` 直接可用的位移——兩者中間隔著 DOS4GW 的 GDT/LDT 段描述子與分頁表映射。
  直接對 `0x627d8` 下 `DP`/`MEMDUMPBIN` **極可能对錯位址**。
- **不要用「硬算基底」的方式換算**(如假設某固定 offset)。正確做法:
  1. 中斷點打在**已知會讀寫這張表的 code**(getter `0x4e803`,已在 doc 47 §5 定位),用 `BPLM` 或
     `BP <seg> <off>`(先用 `C`/`GDT`/`LDT` 對照 CS 段基底換算 seg:off)。
  2. 觸發後用暫存器視窗直接讀出**當時的有效位址**(debugger 的 Register/Data 視窗會顯示 CPU 實際
     算出的 seg:off,不必自己反推)。
  3. 以該有效位址為準 `MEMDUMPBIN` 整張表,再用**已知簽章掃描**(FDTXT/FDOTHER/FDICON 等容器已知
     header pattern,或 acting 幀格式的 `[幀數]+每幀{(拍數,N)+N×(單位idx,姿態)}` 結構)反查
     dump 出來的位元組落在哪個資源檔案裡——這比「dump 完再猜位址對應哪個檔案」可靠(rulebook 64
     第三條路:已知輸出反推位置,不要反過來瞎猜)。

## 6. 待續

本輪已完整驗證的是**通用機制**:image 建置、debugger 確實編進去、Alt+Pause 能觸發 TUI、
`MEMDUMPBIN` 真的能產生檔案、FD2 能在此環境內正常跑起來並截圖。**尚未驗證**的是把這條流程
接到 FD2 的實際 code(getter `0x4e803`)上——下一輪待辦:

1. 用 `C`/`GDT`/`LDT` 把 `0x4e803`(file offset)換算成 dosbox-x debugger 認得的 `seg:off`
   (位址基準換算陷阱見 `/home/anr2/dq3/docs/00-re-methodology.md` §2,同一套換算原則適用本專案),
   下 `BP`/`BPLM` 真斷點(§5 步驟 1)。
2. 讓 FD2 跑到序章觸發 acting 播放(§4.3 流程 + 送方向鍵/Enter 跳過姓名輸入等前置畫面,
   參考 dq3 `docs/29-dosbox-oracle.md` 的按鍵序列反組譯手法),等斷點命中。
3. 讀暫存器拿到當時有效位址,`MEMDUMPBIN` dump 整張表,回填 doc 47 §5。
