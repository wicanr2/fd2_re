# 對拍控制序列

這裡的 `.jsonl` 是原版側（dosgolem `apps/fd2/cmd/oracle`）的控制序列，由
[`tools/dosgolem_oracle.sh`](../../../tools/dosgolem_oracle.sh) 執行。收進版控是
因為收據要能重生：序列變了，收據就不是同一件事。

每行一個 JSON 物件，欄位見
[`96-parity-toolchain-20260909.md`](../../knowledge-base/96-parity-toolchain-20260909.md)。

| 檔案 | 走到哪 | 重製端對應 |
|---|---|---|
| `ch01-move-attack.jsonl` | 第一關第 1 回合：選 (8,16) 的亞雷斯、移動到 (6,19)、攻擊 (5,19) 的盜賊 | `TestDumpChapterOneMoveAttackFrames`（`FD2_ATTACK_DUMP`） |
| `ch01-phase-banner.jsonl` | 第一關第 1 回合：開系統面板、選 END、確認 YES，進敵方回合橫幅 | `phase_banner_glyph_test.go`（比的是字樣落點，不是整幀） |
| `ch01-clear.jsonl` | 第一關打到敵方全滅（閉環，見下） | 待補 |
| `ch02-town-load.jsonl` | 從存檔載入第二章羅德鎮，走出口進第二關戰場 | 待補 |
| `ch02-clear.jsonl` | 同上，接著打第二關到敵方全滅（閉環）| 待補 |

## 續跑點

第一關從標題重跑要 73 億指令（約 20 分鐘）。打完之後在城鎮的**酒店**存檔，
`FD2.SAV` 會落在 `FD2_ORACLE_STATE` 指的覆蓋層；之後從標題走 **LOAD** 載回同一個
狀態只要 2 億指令。第二關以後的每一次迭代都從那裡起跑。

存檔本身是遊戲自己寫的、由標題的 LOAD 讀回，不是第三方存檔；但產生它的那一輪
開了 `FD2_ORACLE_LOCK_ALLY_HP`，所以**由它延伸的收據一律不是 `PLAYER-E2`**。

存檔檔案不進版控（`FD2.SAV`／`FD2.TMP` 每跑一次就變，不是固定版本基準）。要重新
產生一份：

```bash
state=work/parity-state/ch01-cleared      # 覆蓋層，掛成 oracle 的 -state
mkdir -p "$state"
FD2_ORACLE_STATE=$state FD2_ORACLE_LOCK_ALLY_HP=1 \
  tools/dosgolem_oracle.sh <輸出目錄> docs/data/parity-plans/ch01-clear.jsonl
# 打完進城鎮之後：enter 進 0 號酒店、right×1 切到第二個圖示、enter、
# enter 確認（畫面回「記錄儲存完畢！」），"$state" 就會多出 FD2.SAV。
```

每一輪實驗用它的**複本**，不要直接掛這一份——遊戲會往覆蓋層寫東西，原地跑幾次
之後續跑點就不是原來那個狀態了。

每一關打完要在城鎮再存一次，續跑點才會往前推。序列尾端接 `{"town_save": true}`
就行；它進 0 號酒店按完那四個鍵，再用覆蓋層裡 `FD2.SAV` 的內容雜湊確認真的存到。
不存的話那一輪的進度隨容器一起消失，下一關又要從這一關重打一次。

標題選單三項，開機停在 START：

| 項目 | 讀什麼 | 落點 |
|---|---|---|
| START | — | 從頭開始 |
| LOAD | `FD2.SAV` | 存檔當時的位置；enter 之後還有一層四槽選擇，要再一個 enter |
| CONTINUE | `FD2.TMP` | 關卡開場，不是存檔 |

## 城鎮

城鎮不是走動畫面，是五棟建築排成一圈：`left` 遞增、`right` 遞減、超出 0..4 繞
回去，`up` 與 `down` 無效。目前選到哪一棟看畫面右下角那塊標籤。

| 編號 | 建築 | 作用 |
|---|---|---|
| 0 | 酒店 | 打聽消息、**存檔**（載入後停在這裡）|
| 1 | 武器店 | 買賣 |
| 2 | 出口 | 出戰整備 → 「要進入戰場嗎？」YES → 下一關 |
| 3 | 道具店 | 買賣 |
| 4 | 教會 | 未探 |

順序與 `remake/assets/scenarios/campaign_full.json` 的 `town_ch02` 選項一致；
第六項（`selection 5`）是神秘商店，要在指定建築上按該章專屬的 BIOS 掃描碼才會
出現，一般切換到不了。

## 陣營編碼

單位 record `+6` 有三種值，分派與重製端的 `native_continue_runtime_units.go` 一致：

| 值 | 意義 |
|---|---|
| 0 | 敵方 |
| 1 | 友軍（AI 控制的盟友，第二關有六個）|
| 2 | 我方（玩家操作）|

**camp 1 是友軍不是敵人。** 它們會自己打，敵方 HP 因此有時在我方沒出手的回合也會
掉。「敵方全滅」只算 camp 0；把友軍算進去會永遠打不完。序列檔裡 `ally_alive`
這個變數指的是我方（camp 2），不是 camp 1。

## 開環與閉環

前兩份是**開環**：每一格寫死送哪一個鍵，座標藏在方向鍵次數裡。它只在「起點固定」
時成立，所以只能寫單一回合的短路徑。

`ch01-clear.jsonl` 與 `ch02-clear.jsonl` 是**閉環**：`sweep_battle` 每一步回讀
`current.json`，挑還沒行動的我方單位、算落腳格、送鍵之後再確認單位座標真的變了。
「還沒行動」看兩件事——record `+5` bit7，加上驅動端自己記的 identity 清單；只靠
bit7 不夠，它在換手、升級對白與「移動了但還沒結束行動」時都不成立。第二回合以後每個單位都在上一回合走到的位置，開環寫不出來——不是
麻煩而是做不到。移動成本受地形影響、可達範圍在狀態層看不到，所以可達與否一律
由「送 enter 之後座標變了沒」裁決，猜不到就試，試不成就換下一格。

決策靠的是 oracle 快照裡的 `input_chain`——等鍵盤時堆疊上這一條輸入路徑的返回
位址，也就是「現在是哪一個介面在收鍵」。方向鍵在五種介面下的意義完全不同，而它們
在 eip、`view` 與 `units` 上分不出來（`overlay_selector` 恆為 1）：

| 介面 | 特徵位址 | 方向鍵 |
|---|---|---|
| 地圖游標自由移動 | `0x117F8` | 移動游標 |
| 選取後的移動格／目標選擇 | `0x117AE`（選取瞬間是 `0x18978`）| 移動游標 |
| 指令環 | `0x18EEF` | 選 ↑攻擊／←法術／→物品／↓待機；`esc` 是**取消行動**，不是關選單 |
| 系統選單 | `0x16FAE` | 選單項；`esc` 退得掉 |
| 城鎮建築選擇 | `0x2CE08` | `left` 遞增／`right` 遞減，五格循環；上下無效 |
| 對白等待 | `0x16039` 或 `0x1E400..0x1E5FF` | 無效，要 `enter` |

沒有它就只能猜，猜錯會把方向鍵送進選單，然後游標「莫名其妙不動」。細節與三個實測
陷阱見 [`96`](../../knowledge-base/96-parity-toolchain-20260909.md)。

閉環的決策邏輯在 [`tools/dosgolem_oracle_drive.py`](../../../tools/dosgolem_oracle_drive.py)，
純函式部分由 [`tools/test_dosgolem_oracle_drive.py`](../../../tools/test_dosgolem_oracle_drive.py)
逐條驗兩個方向：那些函式錯了不會噴錯，只會安靜地把按鍵送到錯的地方。

兩側的輸入層不同——原版送 BIOS 按鍵，重製端直接驅動同一條狀態機——但走過的節點
與座標相同，逐幀畫面因此可以對照。座標寫死在兩邊：原版在序列的方向鍵次數裡，
重製端在測試檔頂端的常數裡，改一邊就要改另一邊。

`ch01-phase-banner.jsonl` 這一組比的不是整幀：兩側的地圖底圖本來就不同，能對照
的是原版畫面上的字樣像素與落點。原版側用
[`tools/match_lmi1_glyph.py`](../../../tools/match_lmi1_glyph.py) 做模板匹配，
重製端側由 `remake/cmd/fd2/phase_banner_glyph_test.go` 綁住同一份字模與同一組
座標。
