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

每個控制邊界輸出一組 `checkpoint-NNNN.png`（320×200 索引畫面）與
`checkpoint-NNNN.json`。JSON 帶 `runner`、`input_kind`、`state_injections`、
指令步數、`eip`、暫存器，以及：

- `view`：`camera_x`／`camera_y`／`cursor_x`／`cursor_y`／`visible_x`／
  `visible_y`／`round`，取自 `0x53AA9..0x53ABD` 與 `0x53BEF`。
- `units`：`0x53A45` 單位陣列、`0x53BEB` 單位數；每筆保留完整 80 byte
  `raw_hex`，其餘欄位（x／y／pose／camp／hp…）只作導覽，不是證據層。

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
