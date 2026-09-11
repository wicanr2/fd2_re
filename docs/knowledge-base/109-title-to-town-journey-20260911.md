# 109 — 從標題 START 走到羅德鎮：重製端整段對原版證據（2026-09-11）

重製端從開機動畫開始，在標題選 START、看完序章、打完第一關、看完戰後過場，一路
走進羅德鎮。全程只透過正式輸入 owner 送鍵，固定亂數種子跑兩次，兩次的紀錄逐位元
相同。每一段都拿既有的原版證據對照，七項全部相符。

對照途中找到兩個重製端缺口並修掉：地圖資料少了死亡獎勵（進城金幣 0，原版 1000），
以及原生戰場單位升級時不長數值。

原版證據一律綁定固定版本 `FD2.EXE`：357074 位元組，
MD5 `b97caf2239a27a896069d03549d96e1e`，
SHA-256 `222b7d067ad4450eb9c5f6e6bce1797d54bb050417ba39ced6067f8039f28c4f`。
收據在 [title-to-town-ch01-e1.json](../data/ui-traces/title-to-town-ch01-e1.json)，
每一項預期值都在裡面標了原版出處。

## 1. 怎麼跑、證據等級

`remake/cmd/fd2/native_title_to_town_test.go` 的
`TestPlayTitleStartThroughChapterOneToTown`：

- 標題走 `applyTitleMenuEvent`、故事走 `handleNativeStoryInput`、戰場事件對白走
  `handleBattleEventDialogueInput`、勝敗畫面走 `confirmBattleResult`。只有 owner
  正在等鍵的那一幀才送，相當於原版的 `kbd_empty` 閘門，不會把鍵堆在緩衝區。
- 第一關的打法沿用 [doc108 §4](108-terrain-cost-and-move-confirm-20260911.md) 的
  `playChapterOneRounds`。
- 種子固定 `FD2_SEED=20260911`。同一個種子跑兩次，`journey.json` 除了幀數
  以外的全部欄位，雜湊必須相同，否則測試失敗。實際兩次連幀數都相同（24737 幀）。

**證據等級 `RUNTIME-E1`。** 驅動的是重製端的正式狀態機；戰術是測試自己的啟發式，
離屏演出的「已呈現」由測試承認。這不是一般玩家的鍵盤路徑，不得冒稱 `PLAYER-E2`。
原版側這一輪沒有新跑 oracle，比的是收據裡列出的既有證據。

## 2. 對照結果

| 項目 | 原版證據 | 重製端 | 判定 |
|---|---|---|---|
| 序章對白 | `ch00_pre` 處理器位元組碼的 19 次 `0x15F84` 呼叫（`LOADCH 0x20/0x1f/0` 切換 FDTXT_033/032/001），合計 97 句 | 同順序 19 次、97 句 | 相符 |
| 第一回合開局 | 原版 `FD2.SAV` 的目前戰況快照：第 1 回合、12 筆 runtime record，其中 (4,23) 那筆 `+5` bit0 = 不在場 | 在場 11 筆的座標、HP 與我方身分逐筆對上；我方 4、敵方 7、友軍 0 | 相符 |
| 戰場事件對白 | 各呼叫點前的 `push imm`：第 3 回合字串 11、3，第 4／5／6 回合字串 4／5／6（`0x34242`／`0x342A0`／`0x3430A`／`0x34373`／`0x343DD`） | 同樣的（回合, 字串）序列 | 相符 |
| 戰後過場 | `ch00_post`：FDTXT_001 字串 9，13 句 | 13 句，全部是字串 9 | 相符 |
| 進城 | 原版第一關後在羅德鎮酒店的存檔（章節槽 0）：`town_ch02` 羅德鎮、金幣 1000、隊伍順序 索爾 0／悠妮 9／亞雷斯 4／蓋亞 30／哈諾 1；沒升級的人 MaxHP 索爾 42、悠妮 28、哈諾 36 | 全部相同 | 相符（第 3 節修正後） |
| 取得操作權前的按鍵 | 開場長度收據：HUD 出現時原版讀鍵 118..122 次 | 118 次 Enter | 在範圍內 |
| 開局資訊畫面 | 見下方呼叫鏈：原版不會自動顯示 | 序章最後一句之後直接交出操作權 | 相符 |

按鍵數只能比範圍：原版那一臂在開場動畫期間也在送鍵，兩邊的送鍵協定不同。

**第一關開局沒有自動的資訊畫面或行軍確認。** 影片裡看到的「MAP·01 TURN·001」資訊
畫面與「決定要行軍嗎?」問句，都是玩家在空格按確認叫出的系統選單：

- `0x1B1E7`（資訊畫面）唯一的呼叫端是 `0x19F03`，位於巢狀系統選單 `0x19DF7`；
- `0x19DF7` 唯一的呼叫端是 `0x16FED`，位於空游標系統選單 `0x16F55`；
- `0x16F55` 唯一的呼叫端是 `0x118C1`，條件是 `0x118B3` 的 `0x12C0D`（游標下的單位）
  回 `-1`，也就是游標停在空格上。
- 「決定要行軍嗎?」是 `0x16F55` 外層 selector 1（全軍行軍）的問句 FDTXT `0x1A1`
  （[fd2_system_exit_and_group_march_ida.txt](../data/ida/fd2_system_exit_and_group_march_ida.txt)）。

重製端兩條都已接在同一個空游標選單上（資訊畫面 `fdother.NativeSystemInfoTransitionFrames`、
全軍行軍 `beginNativeSystemGroupMarch`），不需要另加開局畫面。

## 3. 修正一：地圖資料少了死亡獎勵

原版的 FDFIELD 名冊 b22..b24 由 `0x10fa8..0x10fb2` 抄進 runtime `+0x31..+0x33`。
型態 0 是物品、型態 1 是金幣（b23..b24 的 u16）、型態 2 走特殊 handler 表：id 39
（`0x34F74`）交給同一個獎勵 dispatcher 的 `00 D3 00`，id 41（`0x34FF0`）是
`00 D5 00`。

`tools/export_units.py` 會產生 `death_effect` 與可執行的 `death_reward`，但目前的
地圖單位檔是由 `tools/sync_native_selector_fields.py` 維護的，那支工具沒有這兩欄。
重製端的獎勵 dispatcher 因此拿不到資料：第一關 group 4 盜賊的 `[1, 0xE8, 0x03]`
就是金幣 1000，打倒它也不給錢，進城金幣是 0。

修正：降階規則抽成 `export_units.native_death_reward`，同步工具共用它，寫回 24 張
地圖。第一關 map0 的內容：

| 單位 | 原始效果 | 可執行獎勵 |
|---|---|---|
| group 1 兩名（(1,3)、(6,0)） | `[0, 0xC0]` | 物品 0xC0 |
| group 4 (20,22) | `[1, 1000]` | 金幣 1000 |
| group 4 (23,21) | `[0, 0xC0]` | 物品 0xC0 |
| group 3 哈諾 | `[2, 4]` | 無（id 4 的 handler 未閉合） |
| group 5 海盜頭目（頭像 97） | `[3, 8]` | 無（型態 3 未閉合） |

未閉合的型態保持不可執行，runtime 不猜。原版進城金幣正好是 1000，與只算盜賊那一筆
相符，所以頭目的 `[3, 8]` 至少不給金幣；會不會給物品仍未知。

`tools/test_export_units.py` 的 `NativeDeathRewardTest` 釘住降階規則，
`sync_native_selector_fields.py --check` 釘住地圖檔與原始名冊一致。

同一次檢查也發現反方向的遺失：map28／31／32 在 2026-08-12 由 `export_units.py` 重生過
（`9cade588`），同步工具寫的 `native_record_race`／`native_record_class` 整批不見，共 134
名。map31／32 是序章的故事地圖，缺這兩欄時 `composeNativeStoryMapFrame` 走的是保守的
近似前景重畫，原本只該用在 map32 前兩名不在建構表範圍內的劇情角色。以
`--write` 補回，值與 `08efeb3b` 逐筆相同，`--check` 歸零。
`MapAssetInvariantTest` 另外釘住「有 `native_constructor` 就必須有種族與職業」，
這一條不需要 `extracted/raw` 也跑得動。

## 4. 修正二：原生戰場單位升級不長數值

`0x1E292` 的成長列由記錄 `+7` 決定，不是角色名：

```text
0x01E2F8  movzx eax, byte ptr [esi+7]
0x01E2FC  push  eax
0x01E2FD  call  0x4E4D1          ; 回傳 0x620A1 + 11 × selector
```

`+7` 在初始職業時就是身分（0..31），轉職後改寫成 32..67。重製端舊的 `GainExp`
拿角色名查成長表，而原生戰場的單位不帶名字，一律查不到，升級只加等級。

修正：`battle.State.NativeGrowthRows` 從 `class_change_growth.json`（與指令學習
selector 同一份表）載入，`NativeGrowthRowFor` 以 `BattleFig`（即 `+7`）選列；沒有
`+7` 的單位才退回舊的名字表。表內每欄是 [下限, 上限)，轉成閉區間時上限減一，
上下限相同就是固定值；這個區間語意早先已用 doc02 §7.2 的 63 列逐列核對過（見
`growth.go` 的 `growthTable` 註解）。`TestNativeGrowthRowFollowsRecordPlus7` 核對第 4 列
與舊名字表的「亞雷斯／騎士」逐欄相同，並確認不帶名字的單位升級時 HP 成長落在 8..10。

## 5. 沒有比到的部分

| 項目 | 為什麼沒比 | 要怎樣才能比 |
|---|---|---|
| 戰鬥過程、等級、經驗、HP、道具 | 戰術是測試的啟發式，擊殺順序和存活者不會等於某一次原版遊玩；城鎮因此只比沒升級的人的 MaxHP | 讓重製端重播原版的鍵盤序列，或在原版側重播重製端的操作 |
| 死亡效果型態 3 與型態 2 的其他 id | handler 未閉合（第一關有 `[3, 8]`、`[2, 4]`） | 追到 handler 的 writer／consumer 後資料化 |
| 升級上限 | `0x1E2E0..0x1E2F2`：`+7` 為 0x1E／0x1F 時比 99，其餘比 40，相等就跳離升級處理；重製端還沒實作 | 實作並加測試；第一關等級到不了上限，不影響本篇結果 |
| 幀數與畫面 | 本篇只比狀態與順序；逐像素、節奏要另走 dosgolem 同狀態收據 | 依 `57` 的 UI 項目逐一取收據 |

## 6. 重跑

```sh
FD2_TITLE_TO_TOWN=<輸出目錄> go test ./cmd/fd2 \
    -run TestPlayTitleStartThroughChapterOneToTown -count=1 -timeout 50m
```

在 `fd2-go-test-local` 容器內、以 `with-xvfb` 執行，並以 `FD2_ASSET_PACK` 指向完整
分離素材根。輸出目錄會有 `run1/`、`run2/` 兩份 `journey.json` 與 `rounds.jsonl`。
