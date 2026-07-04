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

## 6. §5 懸案已解(下一輪實測,`0x627d8` 的執行期真位址 = `0x207718`)

沿用 §4.3 流程(標題 START → Enter 推進王座廳對白 → Alt+Pause)實測,找到**比「算 GDT/LDT」
更直接的方法**:直接讀取 debugger 對 getter 函式(`0x4e7f8`)的 Code Overview 反組譯結果——

```
0x04e801  mov  eax,[eax+0x27d8]  ; ->0x627d8   ← debugger 顯示的 disassembly
```

這一行的 disp32 立即值,在**檔案裡**是 `D8 27 00 00`(=0x27d8,經 doc47 的 build_fixups 換算成
「絕對值」0x627d8 只是連結期 tool-linear,不是執行期位址);但在 debugger 的 **Data view 直接
dump 這條指令所在位址的原始 bytes**,看到的是 `D8 77 20 00`(=0x00207718)——**loader 在載入時已
經把這個 fixup 修正成執行期的真實 flat linear 位址,debugger 直接讀出來就是答案,不需要另外算
偏移量**。用這個方法反查,`0x4e7f8` 本身(getter 函式開頭)在執行期位於 `0x2047f8`(offset 0x1B6000)。

**關鍵發現(id 有 base 偏移,回答 doc47 §5「或 id 有 base 偏移」的猜測)**:表在 `table_base
+ id*4` 只有 **id 0x50(80)~0x99(153)** 這 74 項是本章節(序章)acting 容器填入的合理指標(遞增、
落在 0x207d40~0x2084c0 這段連續範圍內);id < 0x50 或 > 0x99 讀到的是不相關的殘留值(不在合理
位址範圍,且密林幕/王座廳兩次 dump 這段完全一致,代表**不是本次執行填入,是更早的殘留/其他用途
資料**)。海島幕(id 0~8,依 doc47 應為另一批資源)這次沒觀察到有效值,合理推測其容器在序章當下
尚未載入,base 偏移可能因容器切換而不同。

**交叉驗證(王座廳 vs 密林幕,悠妮甦醒對話中)**:`0x207718` 起 0x400 bytes(id 0~255)兩次 dump
只有 id 0x24/0x28 兩處數值不同(且都不在 0x50~0x99 有效範圍內,數值本身也不像合理位址,判斷是
別的系統用途覆寫,與 acting 表無關);**id 0x50~0x99 這 74 項在兩個時間點完全一致**——證實序章
的 acting 資源表是**一次性靜態填好**,不會隨場景切換重新寫入,與 doc47 §5「填表點難尋」的困難
成因吻合(填表發生在更早的一次性初始化,不是每次呼叫或每次換場景才觸發)。

**格式驗證**:74 筆全部用 doc47 §2 的格式(`[帧数]+每帧{(拍数,N)+N×(单位idx,姿态)}`)成功解碼,
無一筆溢出/失敗,frame/beat/N/unit_idx/pose 數值全部落在合理小範圍——確認格式正確。解碼腳本與
完整輸出見 `extracted/dosbox_dump/acting_decoded/`(本機保留,不入 git)。

**下一輪待辦**:
1. 把 74 筆解碼結果(尤其 `#0x63` 索爾走入、`#0x5e~0x61` 蓋亞阻擋/悠妮甦醒)對照 doc47 §3 逐 beat
   轉錄,標注每個 acting id 實際涉及哪些 unit slot(目前只知道 unit_idx 數字,尚未對回「索爾/亞雷斯/
   悠妮/蓋亞」等具體角色——需要交叉 `[0x53a45]+idx*80` 單位陣列同時 dump 才能對上名字)。
2. 找出「id 0x50 base 偏移」的填表點(getter 只讀表,不負責填表;填表 code 仍未定位,但至少確認
   了「填一次不再變」,可以用 `BPPM`(記憶體變更中斷點)打在 `table_base+0x50*4` 上,配合序章一開始
   (王城幕載入前)重啟遊戲,應該只會觸發一次,能抓到填表的呼叫點與呼叫堆疊)。
3. 海島幕(id 0~8)的容器載入時機另外驗證,確認是否共用同一張表、base 偏移是否不同。

## 7. BPLM 量化判死 + 對白時刻快照(task_f,2026-07-04)

- **cycles/core 量測**:normal core + cycles=fixed 80000 只比 dynamic 慢 17%(42s vs 36s 到標題),基礎速度可用。
- **BPLM 病態行為量化證實**:同設定下設 3 個 BPLM(0x1366a/0x13185/0x137e6 執行期位址),三者皆有觸發
  (證實王座廳序幕確實呼叫此三函式);但觸發後 RUN 退化——20 次 RUN/4 秒僅推進 1 cycle;刪斷點後同
  session RUN 2 秒 cc 恢復 32 億級。**結論:dosbox-x heavy-debug 下任何 BPLM 存在即讓 RUN 近似單步,
  非 cycles/core 配置問題,此路判死**。副作用:命中時 CS:EIP 卡 real-mode callback(F000:xxxx),
  讀不到命中瞬間暫存器 → unit_idx→槽映射公式、0x13185 精確語意退回純靜態反組譯(低優先)。
- **對白時刻 21 槽快照**:「兒臣索爾」顯示中,cam=(3,20)、索爾(槽2)=(8,21)——鏡頭頂端下一格,與
  畫面「索爾在紅毯上段、王座正下方」吻合;為 task_e 遞減序列(31→27→…)的自然延續,walk 終點非 (8,8)
  而是 ~(8,21) 時對白已開(對白與走位重疊的證據)。
- **slot3=索爾(4,46) 之謎已解**:map32 FDFIELD roster 本就有兩筆索爾(slot2=(8,42) 走入起點、
  slot3=(4,46) path 幕站位),非「多場景混放」異常——roster 順序=槽序再次驗證。
