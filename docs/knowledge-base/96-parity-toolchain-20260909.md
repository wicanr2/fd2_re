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

尚未實測的是全螢幕戰鬥演出（FIGANI）那條路徑，它與地圖層是不同的呈現流程；
需要時要另外取一次收據，不可由地圖層可用推定它也可用。

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
  bash -c 'xvfb-run -a -s "-screen 0 1280x800x24 -nolisten tcp" go test ./... -count=1'
```

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
