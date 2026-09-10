# 96 — 對拍工具鏈與回歸基線（2026-09-09）

## 三個環節各由誰負責

| 環節 | 工具 | 位置 |
|---|---|---|
| 原版側執行與擷取 | dosgolem `apps/fd2/cmd/oracle` | 由 [`tools/dosgolem_oracle.sh`](../../tools/dosgolem_oracle.sh) 驅動 |
| 影像比較與報告 | dosgolem `apps/fd2/cmd/parity` | 報告含雜湊與 `original_runner` |
| 重製側擷取 | `tools/capture_fd2_*.sh`（Xvfb ＋ AppImage） | 本儲存庫 |
| 相位搜尋（輔助） | `tools/docker/fd2-battle-series-compare.sh`、dosgolem `cmd/framecompare` | 只量 RGB 差異 |
| 能力缺口診斷（輔助） | `tools/docker/fd2-dosbox-screenshot.sh` | 標輔助基準，不得登錄為對拍收據 |

## 為什麼把原版側執行器收編

在 2026-09-08 那一輪，能跑到第一關城鎮的原版執行器是一支容器內腳本
（`prepare_oracle_interactive.py`）動態改寫 `apps/fd2/cmd/bootprobe/main.go`
產生的，只存在於 `work/ch01-town-parity-20260908/`。它跑得出畫面，但下一個
工作階段既讀不到它，也無法宣告某份收據由哪一份程式產生。

**對拍執行器不受版控，等於沒有執行器。** 現在它是 dosgolem 的
`apps/fd2/cmd/oracle`：不給 `-run-dir` 時與 `bootprobe` 行為相同；給了之後
進入互動模式，由該目錄的 `control.json` 逐步推進。需要新能力時改本體並提交，
不要再產生第二份分身。

## 怎麼跑

```sh
# 只跑到第一個 BIOS 等待邊界，輸出 checkpoint-0000 供檢視
tools/dosgolem_oracle.sh /tmp/fd2-oracle-run

# 依宣告式控制序列逐步推進
printf '%s\n' \
  '{"key":"right","steps":3000000}' \
  '{"key":"down","steps":3000000}' \
  '{"key":"enter","steps":3000000}' > plan.jsonl
tools/dosgolem_oracle.sh /tmp/fd2-oracle-run plan.jsonl
```

`key` 為 `up`／`down`／`left`／`right`／`enter`／`esc`，或空字串表示只前進不送鍵。
原版目錄唯讀掛載，容器無網路，只有輸出目錄可寫。

三個選用欄位讓「一直按到某件事發生」不必展開成幾百行計畫：

| 欄位 | 作用 |
|---|---|
| `repeat` | 本行重複 N 次 |
| `gate: kbd_empty` | 該格若還有未被遊戲取走的鍵就改送空鍵 |
| `until: units_present` | 單位陣列一有內容就結束整份計畫 |

`gate` 不是方便而是必要：BIOS 環形緩衝只有 15 格，固定速率送鍵在遊戲消化
不及時會讓 `Enqueue` 回「緩衝已滿」而中斷整輪。設了閘門等於「玩家看到上一
次按鍵被吃掉才再按」，也就是玩家能達到的最快速率。

`until: units_present` 要小心：單位陣列在開場就已經有內容，它判斷不了
「已經進戰場」。

## 開環寫不出多回合：閉環控制

`key`／`repeat`／`gate` 這組欄位是**開環**——每一格寫死送哪一個鍵，座標藏在方向鍵
次數裡。它只在起點固定時成立。第二回合以後每個單位都站在上一回合走到的位置，
方向鍵次數當場失效；移動成本還受地形影響，可達範圍在狀態層看不到。所以整場戰鬥
不是「開環寫起來麻煩」，是**寫不出來**。

驅動端每一步都讀得到 `current.json` 的 `view`（游標座標、回合數）與 `units`
（每個單位的 x／y／hp／camp／80 byte raw），所以決策可以回讀狀態再送鍵。
[`tools/dosgolem_oracle_drive.py`](../../tools/dosgolem_oracle_drive.py) 因此多了
四個閉環命令：

| 命令 | 作用 |
|---|---|
| `{"goto": [x, y]}` | 把地圖游標移到該格；每送一鍵就回讀 `cursor_x`／`cursor_y` 確認有動，連續不動視為撞邊界並回報失敗 |
| `{"await": "round>=2"}` | 只前進不送鍵直到條件成立；變數有 `round`／`enemy_alive`／`ally_alive`／`cursor_x`／`cursor_y`／`steps` |
| `{"engage": [x, y]}` | 選取該格的我方單位，推進到貼著敵人的格，射程內有敵人就攻擊，否則待機 |
| `{"sweep_battle": true}` | 每回合掃完我方所有未行動單位，再等回合數推進，一路到敵方全滅 |

三件事讓它不會安靜走錯：

- **可達與否由結果裁決，不由計算。** 送 `enter` 之後看單位座標變了沒；沒變就換
  下一個候選落腳格。試探成本約兩千萬指令，所以貼敵格離單位比移動力還遠就先剔掉。
- **「已行動」讀 record `+5` bit7**（doc 31 §6 已證實）。只移動不待機的單位這個位元
  不會設，掃描下一輪又會選到它，回合永遠推不掉——所以射程內沒敵人時要送
  `down`＋`enter` 走指令環的待機（↑0 攻擊／←1 法術／→2 物品／↓3 待機）。
- **失敗一律非零離開。** `goto` 卡住、候選格全到不了、`await` 逾時都回報並中止；
  盲目繼續會產生「看起來跑完了但走錯路」的收據。

### 戰鬥中途的過場會換掉單位陣列基底

第一關第 3 回合會觸發哈諾與哈瓦特加入的過場。那段期間 `unit_base` 從戰場的值換成
另一個位址，`units` 讀出來是垃圾：camp 會出現 63／34／54 這種值，x／y 超出地圖，
`hp` 是 16191 這種數量級。**狀態欄位本身不會告訴你它失效了**——它照樣是合法 JSON、
照樣有 12 筆單位。

第一次實測就踩到：驅動器在垃圾裡挑到一個 camp 為 2 的「我方單位」在 (2,0)，`goto`
一路往左送鍵。防護有效（連送三次游標不動就中止並回報），所以沒有產生一份走錯路
的收據——但那一輪就停在第 3 回合。

修法是把戰場的 `unit_base` 記下來當閘門：`sweep_battle` 開始時（那時確定在戰場）
記值，之後每次要用 `units` 之前先比對。不相等就是不在戰場，改送 `enter` 把過場推完
再回來。基底每次執行不一定相同（實測 `0x1765F0` 與 `0x175D30`），所以必須動態記錄，
不能寫死常數。

基底相等還不夠。實測第二輪：過場期間 `units` 的 camp 與座標**剛好都落在合法範圍**
（ally 3、enemy 18），只看內容會誤判成「還在戰場」，然後把游標往 (8,42) 這種地方送。
加上第三個判準才擋得住：**回合數不會倒退**。過場期間 `view` 讀的也是別的記憶體，
`round` 會掉回 1，而戰場的回合只會遞增。

第三輪又發現**狀態在過場期間會抖動**：同一段過場裡連續取樣，`round` 在 1 和 3 之間跳、
`units` 在合理與垃圾之間跳，oracle 讀的那幾個位址被複用。單次取樣判不準，要連續
三次都在戰場才採信。

### 介面模式：驅動端唯一分得出「現在誰在收鍵」的訊號

上面那些補救全部是同一個病：**驅動端在猜遊戲的 UI 狀態**。地圖游標自由移動、
選取後的移動／目標選擇、指令環、系統選單、對白等待——方向鍵在這五種狀態下的
意義完全不同，而它們在既有快照欄位上**分不出來**：eip 都停在 BIOS 等待點、
`overlay_selector` 恆為 1、`units` 也一樣。猜錯就把方向鍵送進選單，然後游標
「莫名其妙不動」。

架構修正在 oracle 那側：快照多帶一個 `input_chain`——等鍵盤時掃 ESP 往上 512
byte，只收前面五個 byte 是 `E8`（call rel32）的值，也就是這一條輸入路徑的返回
位址（dosgolem `feat/fd2-oracle-input-chain`）。兩個試過不行的做法：直接收所有
落在程式碼範圍的 dword 會把堆疊殘值一起帶進來，同一個介面每次取樣都不一樣；
走 EBP 連結串只拿得到一層，這份執行檔省略了 frame pointer。

實測（第一關）五種介面各有穩定特徵，彼此互斥：

| 介面 | 特徵 | 方向鍵的意義 |
|---|---|---|
| 地圖游標自由移動 | `0x117F8` | 移動游標 |
| 選取後的移動格／目標選擇 | `0x117AE`；選取那一瞬間是 `0x18978` | 移動游標 |
| 指令環 | `0x18EEF` | 選 ↑攻擊／←法術／→物品／↓待機 |
| 系統選單 | `0x16FAE` | 選單項；`esc` 退得掉 |
| 對白等待 | `0x16039`，或 `0x1E400..0x1E5FF` | 無效，要 `enter` 才往下走 |

`0x16FAE` 落在已知的系統選單 handler `0x16F55` 那支函式裡、`0x18EEF` 落在 action
chooser `0x18D8C` 同一段——與既有反組譯結論一致。三個細節是實測換來的：

- **選取那一瞬間的鏈不一樣**（`0x18BF4, 0x18978`）。那個迴圈要收到一個鍵才會轉成
  `0x117AE`，而「等它進 target」送的是空鍵——漏了這一條就會永遠等不到。
- **對白不只一支 handler**（`0x1E44E`／`0x1E5A8`／`0x1E464`），要用範圍比對。單一
  位址會讓同一種畫面有一部分掉進 `unknown`，而 `unknown` 的處置是等，對白等不出
  結果。
- **`unknown` 多半是敵方回合**：AI 在走位與演出，遊戲沒在等鍵盤，堆疊上自然沒有
  輸入路徑。那只能等，而且要等得夠久；預算給小了會在換手當下判成「退不回地圖
  游標」。

有了它，判斷就從推測變成觀察：移動成不成功看「介面有沒有進指令環」（被拒絕時會
留在 target），比對照座標可靠——座標比對分不出「沒走成」與「走到別處」。接到
`sweep` 之後，第一關第 2 回合四個單位全部接戰成功，`unknown` 佔比從 104/253 降到
25/237。

### 盲送 enter 會穿過任何選單

過場播動畫時 `kbd_reads` 根本不動——送進去的鍵停在 BIOS 緩衝區沒人取。第一次實作
給了 200 格盲推，我方全滅之後那些 enter 在動畫結束的瞬間一次全部灌進去，一路推過
戰敗畫面、標題選單、開場動畫，**重新開了一局**。log 最後看起來很正常：
「round=1、ally=4、enemy=0、游標在 (8,16)」——那正是第一關的開局。要盯著
`checkpoint-*.png` 才看得出來跑錯了。

所以推過場以等待為主：每三格才送一次 enter，緩衝區還有鍵就不送，而且預算給 30 格
不給 200。推不回來就中止並回報，讓人看 checkpoint 判斷是勝利演出、戰敗，還是過場
沒推完——不要替它猜。

純函式部分（誰還沒行動、往哪一格走、條件成立了沒）由
[`tools/test_dosgolem_oracle_drive.py`](../../tools/test_dosgolem_oracle_drive.py)
逐條驗兩個方向。改這支之後要先用既有的開環序列做重現對照：2026-09-10 以
`ch01-move-attack.jsonl` 跑，53 個檢查點的 `steps`／`eip`／`view`／單位 raw 位元組
雜湊全同，才敢用它產生新收據。

實測值（第一關第 1 回合）：進場後四個單位各接戰一次、全員行動完原版**自己換手**
到第 2 回合，不必送 END；整段約 8 億指令、78 秒。

## 逐幀擷取

從狀態層判斷畫面不準：座標、姿態、動作三個欄位可以連續且正確，畫面上仍然
多畫或少畫東西。逐幀擷取讓「多畫了什麼」直接看得到。

mode13 沒有換頁這個動作——程式直接畫進 `0xA0000`——所以一幀的邊界得自己定。
原版**開機之後完全不再讀 `0x3DA`**（讀取數從第一張到最後一張都是 519），
沒有垂直回掃可以掛。實測三種取樣方式：

| 方式 | 設定 | 結果 |
|---|---|---|
| 只看內容變化 | `stride 2000`、`settle 0` | 每 2 千指令就變一次，寫滿上限；多半是畫到一半的半成品 |
| 遊戲自己的繪圖進入點 | `frame-eip 0x11CAC` | 間隔中位數 129998 指令，極穩定；但**移動動畫期間完全不進入這個位址** |
| 取樣加穩定閘門 | `stride 2000`、`settle 4` | 移動視窗取到 15 張，間隔同樣約 13 萬指令，正好對上每格六個 motion 加抵達 |

所以**原版一幀約 13 萬指令**（虛擬 130 毫秒），而且不必知道繪圖進入點：
`settle` 要求內容連續相同幾次才寫出，就能把半成品濾掉、把幀節奏取回來。
收據見
[fd2-frame-boundary-20260909.json](../data/ui-traces/fd2-frame-boundary-20260909.json)。

```sh
FD2_ORACLE_FRAMES=1 FD2_ORACLE_FRAME_STRIDE=2000 FD2_ORACLE_FRAME_SETTLE=4 FD2_ORACLE_FRAME_FROM=189400000 FD2_ORACLE_FRAME_TO=191600000 FD2_ORACLE_FRAME_MAX=600   tools/dosgolem_oracle.sh /tmp/fd2-frames plan.jsonl
```

輸出是 `<輸出目錄>/frames/frame-NNNNNN.png` 加一份 `frames.jsonl`，每列帶指令
數、EIP、畫面內容的 sha256、視圖全域（含 `0x51A83` 的 overlay selector）、單位
陣列基底與數量，以及 `0x3DA` 讀取數與調色盤寫入數。`FD2_ORACLE_FRAME_EIP` 可
改用遊戲自己的繪圖進入點；`FROM`／`TO`／`MAX` 把輸出限在要看的那一段。

重製端的對應工具是 `remake/cmd/fd2` 的 `TestDumpChapterOneMoveFrames`，用生產
端的 `composeNativeMapFrame` 落地同格式的畫面：

```sh
docker run ... -e FD2_FRAME_DUMP=/frames -v <輸出目錄>:/frames:rw ... \
  go test ./cmd/fd2 -run TestDumpChapterOneMoveFrames -count=1 -v
```

沒有 `FD2_FRAME_DUMP` 就略過，不影響一般回歸。

### 找「這一段是誰畫的」

`FD2_ORACLE_EIP_WATCH` 收逗號分隔的十六進位位址（最多 16 個），每一幀記錄各自
的累計進入次數。逐幀畫面回答「有沒有畫」，這個計數器回答「誰被呼叫了」。

```sh
FD2_ORACLE_FRAMES=1 FD2_ORACLE_FRAME_STRIDE=2000 FD2_ORACLE_FRAME_SETTLE=4 \
FD2_ORACLE_EIP_WATCH=0x11cac,0x122dc,0x11eee,0x127a9,0x127e0,0x1ad72 \
  tools/dosgolem_oracle.sh /tmp/fd2-frames plan.jsonl
```

一次就問出走行重繪的層集合：地形與前景各一次、單位繪製公式十一次，而整幀
排程、範圍圖示與 HUD 皆為零。細節見
[99](99-move-confirm-cursor-20260909.md)。

### 這套方法可以取到哪些既有缺口的收據

[57](57-ui-evidence-matrix.md) 的缺口欄有一整批寫成「同狀態逐幀差分」「DOSBox
E2」「pixel diff」——那些在只有狀態收據的時候取不到，現在取得到。逐幀畫面回答
「畫了什麼」，`-eip-watch` 回答「誰畫的」，兩者都不需要另外架 DOSBox。

| 缺口 | 這套方法能給什麼 |
|---|---|
| UI-01 完整 boot 畫面差分 | 逐幀畫面 |
| UI-02 ch00／ch01 event1/2 同 camera/roster/pass 的逐幀比較 | 逐幀畫面 |
| UI-03／UI-04 command 同狀態逐幀 | 逐幀畫面；音訊仍不涵蓋 |
| UI-04 不可用目標灰化 | 逐幀畫面 |
| UI-05 對白動畫相位與 DAC 狀態 | 逐幀畫面＋收據裡的調色盤寫入數 |
| UI-06 ch27 同 roster/event state 的 pixel diff | 逐幀畫面 |
| UI-07 每章是否進 town/shop/rest/preparation | `-eip-watch` 掛各 handler 進入點 |
| UI-08／09／10 未修改一般玩家路徑 | `gate: kbd_empty` 的長路徑驅動＋逐幀畫面 |

**取得到收據不等於缺口已關**：每一項仍要照原本的規則寫進 `58` 與 `57`，並標
證據等級。這裡只是說「以前缺工具，現在不缺」。

音訊、靜態語意（predicate、順序、高階名稱）不在這套方法的範圍內。

### 為什麼非看畫面不可

第一次用它比對就抓到一個狀態層看不出來的差異：移動動畫期間原版不畫游標白框
也不畫左下 HUD 面板，重製端兩者都還畫著——而兩側的 overlay selector **都是
1**。原版是走另一條不呼叫 `0x122DC` 的繪圖路徑，所以 selector 的值再怎麼對，
白框也不會出現。只看狀態會判定兩側一致。細節見
[99](99-move-confirm-cursor-20260909.md)。

每個控制邊界輸出一組 `checkpoint-NNNN.png`（320×200 索引畫面）與
`checkpoint-NNNN.json`。JSON 帶 `runner`、`input_kind`、`state_injections`、
指令步數、`eip`、暫存器，以及：

- `view`：`camera_x`／`camera_y`／`cursor_x`／`cursor_y`／`visible_x`／
  `visible_y`／`round`，取自 `0x53AA9..0x53ABD` 與 `0x53BEF`。
- `units`：`0x53A45` 單位陣列、`0x53BEB` 單位數；每筆保留完整 80 byte
  `raw_hex`，其餘欄位（x／y／pose／camp／hp…）只作導覽，不是證據層。
- `kbd_pending`：BIOS 環形緩衝 `0x41A`／`0x41C` 的頭尾差，尚未被遊戲取走的
  鍵數。
- `kbd_reads`：遊戲實際取走的鍵數，也就是有效推進次數。它和送出的鍵數不是
  同一件事——問「原版走完這段要按幾次」時要看這個。

有界性有三重：`-steps` 的總上限、每段 `control.steps` 的 1..1e8、
以及 `-wait-timeout` 的牆鐘上限。

## 目前能力

**戰場地形與單位 sprite 都會繪製。** 2026-09-09 以現行 oracle 從標題經
四個宣告式輸入進入第一關，取七個存活單位所在的 24×24 格量測：唯一色數
30–46、平均亮度 55–71；同幀空地格為 12 色。人物外觀、面向與移動的原版收據
因此可以直接由 oracle 產生。

原版地圖視窗的幾何也已由收據量到：**312×192 置於 320×200 的 (4,4)**，
四邊為純黑，逐幀差異的 bbox 恆為 `312x192+4+4`。

全螢幕戰鬥演出（FIGANI）那條路徑也取得到。2026-09-10 以 `-frame-eip 0x29164`
掃出第一關第 2、3 回合各三次演出，並對第一段逐幀擷取：畫面、版面與呼叫順序
（`sub_29164` 一次、`sub_2A289`／`sub_18C6D` 各二到四次、每取樣幀 600～800 次
`sub_373C4` 組幀）全部取得到，收據見
[fd2-figani-fullscreen-20260910.json](../data/ui-traces/fd2-figani-fullscreen-20260910.json)。

**但它的時序量不到**：每段演出 `sub_375B2`（`delay(ms)`）累計 57～64 次，而
`sub_17AA9`（等 BIOS tick）只有 2 次——節奏幾乎全由 `delay` 構成，那在 dosgolem
上不按毫秒消耗時間（見下面「`delay(ms)` 量不到」）。要重建這一段的節奏只能讀
呼叫點常數。

> 2026-09-10 勘誤：本段原本寫「尚未實測的是全螢幕戰鬥演出（FIGANI）」。那在
> 寫下的當天稍晚就已經不成立——同一天的
> [fd2-physical-attack-presentation-20260909.json](../data/ui-traces/fd2-physical-attack-presentation-20260909.json)
> 已經用 oracle 取到三段演出，`-eip-watch` 也記到 `0x29164` 三次、`0x2A289`
> 與 `0x18C6D` 各十二次。斷言沒有跟著收據更新，就這樣活了一天。

## 效能與成本

2026-09-09 以 `-cpuprofile` 對 150M 指令的第一關執行取剖析：**繪圖不是瓶頸**，
前 22 個熱點裡沒有任何 VGA／video 函式。兩處實際成本已處理（dosgolem 側的
`perf/vga-draw-speed`）：

| 項目 | 修改前 | 修改後 |
|---|---|---|
| 整體吞吐（150M 指令，user CPU 最小值） | 12.96 秒 | 11.70 秒（約快 9.7%） |
| 細粒度追蹤每一步（10 萬指令／步） | 151 ms | 39 ms（3.9 倍） |

第一項是 selector 解析：保護模式每次記憶體存取都要查
`map[uint16]Descriptor`，剖析裡 `mapaccess2` 累計 23.84%。改成 16 組直接對映
快取後這一項消失，但 `segmentLinear` 也因此超出 inline 預算，淨得約 9.7%。

第二項是互動控制迴圈：oracle 與驅動兩側原本都固定 100 ms 輪詢，而細粒度追蹤
每格只跑 10 萬指令（約 8 ms），等待因此支配整段時間。改成 200 µs 起、上限
20 ms 的退避；`current` 與剛寫好的 `checkpoint-NNNN` 必然同一時點，不再重複
編一次 PNG，只寫狀態 JSON。

兩項都以「150M 指令的固定 FD2 收據逐位元相同」與 dosgolem 13 個套件全通過驗收。

估算對拍成本時用這兩個數字：粗粒度（每步 300 萬指令）約 0.25 秒／步，
細粒度（每步 10 萬指令）約 0.04 秒／步。

### 判斷能力時的取樣紀律

`work/` 底下保留的舊 run 有些是刻意留著的失敗樣本。`oracle-live-r11` 的檔案
模式表受污染，單位格只有 5–13 色、平均亮度 18；拿它取樣會得出「單位 sprite
未繪製」的假結論。判斷能力一律用目前程式重生的收據。

## 重製端回歸的可重現命令

分離素材與語言包不在公開庫內，測試必須指向完整素材根：

```sh
docker run --rm --network none --memory 8g --cpus 4 --pids-limit 512 \
  -u "$(id -u):$(id -g)" \
  -e HOME=/tmp/home -e GOCACHE=/gocache -e GOFLAGS=-mod=mod \
  -e FD2_ASSET_PACK=/pack \
  -v /home/anr2/cht/fd2:/src -v <完整素材根>:/pack:ro -v <快取>:/gocache \
  -w /src/remake fd2-go-test-local:latest \
  with-xvfb go test ./... -count=1
```

平常直接用受版控的驅動，它會順便跟基線做差異比對：

```sh
tools/remake_go_test.sh <完整素材根> [輸出 log] [套件…]
```

**不要用 `xvfb-run`。** 它與 Xvfb 之間是 SIGUSR1 交握（`trap : USR1` 之後
`wait`）；Xvfb 太早就緒時訊號會落在 `wait` 之前，`wait` 就再也不會回來，
容器裡只剩 `xvfb-run` 與 `Xvfb` 兩個行程、CPU 0%，命令一行都沒跑——外面看
起來像測試跑很久。反向也踩過：命令結束了 wrapper 卻沒收掉 Xvfb，留下無界
背景行程。映像現在內建
[`with-xvfb`](../../tools/docker/with-xvfb.sh)：明確 PID ＋ trap 擁有 Xvfb，
等 X11 socket 出現才執行命令，命令結束就收掉。

素材根缺件會讓失敗數大幅膨脹，且失敗訊息看起來像功能缺陷。判讀前先確認
`ui/action_cells`、`ui/fdother_014_church`、`locales/`、`palette/` 都在。

`remake/generated-assets/fd2-original-b97caf22/` **沒有 `locales/`**，直接拿它
當素材根會多出三項失敗，錯誤字串全是
`read locale entities ".../locales/zh-Hant/entities.json"`。完整素材根要把
儲存庫的 `remake/assets/locales` 疊上去。巢狀 bind mount 掛不進唯讀掛載點，
可行的作法是在暫存目錄組一個符號連結根：

```sh
root=$(mktemp -d)
for e in remake/generated-assets/fd2-original-b97caf22/*; do
  ln -s "/src/${e}" "$root/$(basename "$e")"
done
ln -s /src/remake/assets/locales "$root/locales"
# 之後 -v /home/anr2/cht/fd2:/src -v "$root":/pack:ro -e FD2_ASSET_PACK=/pack
```

符號連結的目標寫成容器內路徑，因為儲存庫本身也掛在 `/src`。

## 改過 `main.go` 之後的字串盤點重新綁定

`TestReviewedGoCandidatesMatchCurrentInventory` 會在改動 Go 原始碼之後失敗，
訊息長這樣：

```
review binds sha=<舊>, want sha=<新>
```

原因是 `docs/data/fd2-string-review.json` 的 `string_id` 是**行號座標**
（`legacy.go.remake.cmd.fd2.main.l10682-c30`），在 `main.go` 插入任何一行，
後面所有 ID 都會位移。這不是新增待審字串，處置內容也沒有變。

重新綁定的作法是拿改動前後兩份盤點，用「檔案 ＋ 文字」配對：

```sh
git worktree add --detach <暫存路徑> HEAD          # 改動前的一份
# 兩邊各跑一次（容器內）：
#   go run ./cmd/fd2-string-inventory -repo <repo 根> -output <輸出.json>
# 以 (source.file, text) 把舊 string_id 對到新 string_id，改寫
# dispositions 的 string_ids，再把 inventory_sha256 換成新盤點的 sha256。
git worktree remove <暫存路徑>
```

不要按排序直接配對：同一段文字可能出現多次，只有「出現順序 ＋ 檔案 ＋ 文字」
一起用才對得準。配對前先確認兩邊 `go_review` 的條目數相同，數量不同就是真的
新增或刪掉了待審字串，那要逐筆判斷處置，不是重新綁定。

## 對拍基底：dosgolem `main`

2026-09-10 起，原版側對拍的基底是 dosgolem 的 `main` 分支。FD2 那批工作
（`372698a` 平台能力、`293e15c` oracle 命令、`3179a8d` 按鍵緩衝、`6d35693`
效能、`09aca58` 逐幀、`61a0f95` overlay selector、`f627cd1` `-eip-watch`）都已
經由 `bbcdfe3` 合進 main，`git merge-base --is-ancestor` 七項全數確認。

換基底之後跑過兩次重現：同一份控制序列在 `main` 的 `d351681` 與 `c8aa69a`
（A20／HMA 位址遮罩改動之後）上，控制邊界的指令數逐格相同，290 幀的 step 與
`indexed_sha256` 序列與 `f627cd1` 那輪完全相同。所以 `f627cd1` 之後進 main 的
時鐘、machine 與 CPU 定址改動都沒有動到 FD2 這條路徑的指令流，既有收據仍然
有效。**換基底就重跑一次這份對照**，比事後解釋數字為什麼變了便宜得多。

### 時鐘：LE 路徑一道指令一微秒

FD2 走的是 LE（DOS/4GW 保護模式）那條路，它的 BIOS 時鐘由
`InstallLEBIOSClock` 掛在 `CPU.StepHook`，每道指令推進一次、一次代表一微秒；
PIT 分頻 65536 時要 **54,926 道指令**才讓 `0000:046C` 加一（dosgolem 的
`TestBIOSClockPeriodMaskAndRollover` 釘住這個數字）。收據裡的指令步距要換算成
毫秒就用它。

`Machine.IRQ0Every`／`CycleClock`／`DefaultCPUHz` 是 real-mode 那條路的時鐘，
與 LE 這條路是兩套，不要互相套用——2026-09-10 就因為套錯而把一段正確的敘述
改成錯的，繞了兩圈才由程式碼糾正回來。

`cmd/probe` 跑 FD2 也跑得動，但它不帶 `apps/fd2` 的平台設定（`0000:046C` 全程
是 0），不能拿它的環境代表 oracle。

### `delay(ms)` 量不到，只能讀常數

Watcom 的 `delay(ms)`（`0x375B2` → `0x3DCCD`）不看時鐘：它把毫秒乘上開機校準值
`[0x541B0]`，換算成要呼叫幾次 `int 21h AH=2Ch`，再用那些呼叫消耗時間。校準值在
模擬器上測出來的數字與真機無關，所以**收據看不到 `delay` 的長度**。

2026-09-10 實測：回合橫幅中段，敵方走 `delay(20ms)`、玩家走 `delay(150ms)`，
兩者的「進場最後一步→退場第一步」步距都是 348,000 指令，兩段橫幅全長只差
52,000 指令（不到一個 tick 的 54,926），而原版設計差 130,000。

所以動畫節奏要分兩種來源看：**等 BIOS tick 的（`sub_17AA9`）在收據裡量得到，
走 `delay(ms)` 的只能讀呼叫點的常數**。全庫 185 個 `delay` 呼叫點與毫秒參數見
[`fd2_delay_call_sites.txt`](../data/fd2_delay_call_sites.txt)。

### 玩家操作的對拍：兩側走同一組座標

原版的完整操作流程（2026-09-10 實測確認）：

1. 方向鍵把游標移到我方單位 → **Enter：選中並進入移動模式**（可移動範圍出現）
2. 方向鍵把游標移到目標格 → **Enter：確認移動**（單位走過去，有走行動畫）
3. 走完自動出現**四向指令環**：↑0 攻擊／←1 法術／→2 物品／↓3 待機
   （由 `0x18D8C` 的 switch 釘死；起始選擇由 `sub_173E7` 從方向 0 找第一個可用的，
   不可用的方向按了完全沒反應——`sub_177FC` 的閘門）
4. **Enter：確認攻擊** → 關環，游標自動跳到射程內的敵人
5. **Enter：確認目標** → 全螢幕戰鬥演出 → 雙方 HP 結算

序列收在 [`docs/data/parity-plans/ch01-move-attack.jsonl`](../data/parity-plans/ch01-move-attack.jsonl)。

重製端那側由 `TestDumpChapterOneMoveAttackFrames`（`FD2_ATTACK_DUMP`）走**同一組
座標**：直接驅動 `Game` 的同一條狀態機，不經過鍵盤層。兩側輸入層不同，但走過的
節點與座標相同，逐幀畫面因此可以對照。

```sh
tools/dosgolem_oracle.sh <輸出> docs/data/parity-plans/ch01-move-attack.jsonl
FD2_ATTACK_DUMP=<輸出> go test ./cmd/fd2 -run TestDumpChapterOneMoveAttackFrames
```

第一次對拍就抓到差異：原版攻方 HP 48→31（**受到反擊**）、守方 28→8；重製端攻方
48→48（沒受傷）、守方 28→6。收據見
[fd2-move-attack-parity-20260910.json](../data/ui-traces/fd2-move-attack-parity-20260910.json)。

**玩家完全不操作跑不到第五回合**：實測每回合直接 END 的話，第 3 回合之後我方
全滅戰敗，畫面回到王座廳第 0 章對白。要走到後段回合，控制序列就得真的會打。

### 找特定演出在哪一段：`-frame-eip` 加 `stride 0`

要在幾億道指令裡定位某個演出，不必大範圍逐幀掃。把 `FD2_ORACLE_FRAME_EIP` 設成
那段演出的入口、`FD2_ORACLE_FRAME_STRIDE` 設 0，就只在進入該位址時取幀。
2026-09-10 用 `0x1F1CC` 兩幀就標出兩次回合橫幅的起點（step 245,502,547 與
318,822,150），再針對第二次設 `FRAME_FROM`／`FRAME_TO` 精取，省掉一次大範圍掃描。

### 收據自己帶出處

掛進容器的是 dosgolem 的**工作區**，所以真正決定結果的是那個目錄當下 checkout
的內容，不是誰記得自己切在哪一個分支。[`tools/dosgolem_oracle.sh`](../../tools/dosgolem_oracle.sh)
現在每一輪都會寫一份 `runner.json` 到輸出目錄：

```json
{
  "dosgolem_commit": "…", "dosgolem_branch": "main",
  "dosgolem_tracked_dirty_files": 0, "dosgolem_untracked_files": 2
}
```

已追蹤檔案有改動時它會在 stderr 警告——那一輪的收據無法由 commit 重現，不可
登錄成正式對拍。未追蹤檔案（別的工作留下的產物）不影響建置，分開記。

**dosgolem 是共用儲存庫，隨時可能有另一個工作階段在改它。** 2026-09-10 就遇到
一次：驗證跑完幾分鐘後，`internal/cpu` 與 `internal/machine` 出現九個未提交的
改動。跑對拍前先看 `runner.json` 的 dirty 計數，不要拿別人改到一半的樹當基底；
也不要為了「乾淨」去 stash 或 checkout 別人的工作區。
