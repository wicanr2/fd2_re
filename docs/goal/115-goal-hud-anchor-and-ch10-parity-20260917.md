# 115 — 戰場 HUD anchor 跟著重繪評估（#40）＋第十章工作單元（洞窟中的激戰，map 9）（2026-09-17）

> 這是 [`111`](111-goal-original-parity-campaign-20260915.md) 的下一個工作單元，接在
> [`114`](114-goal-boosted-slot-and-ch09-parity-20260917.md)（第八章強化收據、第九章已 passed，
> 提交 `fd380cc6`）之後。接手時先讀 `CLAUDE.md`，再讀 `111`，再讀本檔。本檔只定義這一輪的
> 順序、門檻與交付物；證據分級與位址規則仍以 `56`／`57`／`58` 為準，結果寫回那三份、
> `91-worklist-history.md` 與台帳，本檔不承載結果。

## 使用者定案（2026-09-17）

| 項目 | 定案 |
|---|---|
| 先做什麼 | 先修 issue [#40](https://github.com/wicanr2/fd2_re/issues/40)：戰場 HUD 小窗 anchor 要跟著每次 `0x11CAC` 重繪評估 |
| 回歸範圍 | **已通過的第四～九章不重跑**；#40 的驗收直接用第十章的章收據 |
| 建構槽政策 | 沿用 114：`--levels-per-chapter 6 --seed 4 --event-states 7:17=1 --boost-base-ap 200 --boost-base-dp 0 --boost-base-dx 60` |

「不重跑」的意思是這一輪不用第四～九章的舊原版輪當回歸門檻；它們的收據與台帳狀態維持
`passed`，不因為 #40 改了重繪時機而改寫。若第十章以外的章之後重跑出差異，當成新發現處理，
不回頭推翻這一輪的結論。

## 0. 先修 #40：HUD anchor 跟著重繪走

現況（fd380cc6）：

- 原版每次 `0x11CAC` 重繪都經 `0x1ACF3` 檢查閘 A `[0x51AAB]`／閘 B `[0x51AAC]`，兩個都開才由
  `0x1AD2A` 依可見游標決定小窗左右（可見游標 Y>5 時 X<3 翻右、X>9 翻左）。
- 重製端沒有「一次重繪」的單一入口，anchor 只在幾個呼叫點評估：鍵盤游標步
  `nativeCursorStepHUD`（`NativeMapCursorStepRedraws`）、AI 聚焦 `FocusNativeMapCursorSteps`、戰場節點
  進場 `materializeNativeMapRuntime`、回合開頭聚焦 `stepNativePlayerFocus`（每步無條件評估）與
  `restoreNativeDisplayGateB`。
- 已知沒有評估的重繪：`0x13FD4` 原地回復的 `focusUnitJob`（`stepFocusUnit`）、死亡程式與掉落訊息
  開框前的整幀重組（`composeNativeMapFrame*`）、攻擊演出收尾與升級對話底圖。
- 第八章 c2 seq 1667 與第九章 r1 seq 814 都是這一類差異（各約 4600 px），都是逐個呼叫點補掉的。

做法（順序固定）：

1. **盤點原版重繪點（RE，先寫文件）。** 用 IDA 列出 `0x11CAC` 的所有直接呼叫端，按「戰場回合內會走到」
   分類（游標鍵處理器、`0x12CEA` 家族、走行步進是否經過、攻擊演出前後、`0x1AA1D` 訊息、升級、
   死亡程式、回合換手 `0x1A30B`）。每一項記：呼叫端位址、當時閘 A／閘 B／`[0x51A83]` 的值從哪裡來。
   結果寫進 `docs/data/ida/` 一份匯出與 `58` 一列，再動程式。已閉合的位址（`58` 不要重做索引）直接引用。
2. **規格。** 在 `56` 寫出重製端「重繪入口」的契約：哪些重製端工作對應哪個原版重繪點、anchor
   什麼時候評估、閘門與 `[0x51A83]` 怎麼判。對不到原版重繪點的重製端路徑要列出來，不可為了
   收斂差異而自己加評估點。
3. **實作。** 把 anchor 評估收成跟著重繪走的一個入口，拿掉散落的逐點評估；`stepFocusUnit` 與
   死亡程式／掉落訊息重組也經過同一個入口。`battle.NativeFocusTraceHook` 註解裡「走 focusUnitJob，
   不經過這裡」那句是 #40 的 verify 訊號，改完要一起更新。
4. **單元測試。** 至少：原地回復聚焦途中可見游標進翻邊區時照閘門翻或不翻；回合開頭聚焦時閘 B 為 0
   不翻、寫回後評估一次。
5. **驗收在第十章。** 不重跑第四～九章；#40 等第十章四個 gate 全過後，在 issue 留言寫提交與第十章
   收據，再用 `fd2_worklist_issues.py close` 關閉。

## 1. 第十章：這一章有什麼（開工前已知，動手時用收據核）

| 項目 | 內容 | 出處 |
|---|---|---|
| 節點 | 起點 `town_ch10`「洞窟中的激戰」（variant 2、祕密商店 selection 3、scan 112＝Alt+F9）→ `preparation_ch10` → `story_ch10`（raw `ch09_pre`）→ `battle_ch10`（map 9，31×45）→ `postbattle_ch10_persist`（raw `ch09_post`）→ `town_ch11`「幻之森林」（variant 2、祕密商店 selection 4、scan 93） | `campaign_full.json` |
| 目標 | 敵全滅；務必保護「索菲亞、卡納恩三世」存活 | `story_ch10`、`28-chapter-objectives-and-recruits.md` |
| 勝負 handler | 章索引 9（`[0x53C03]`）是**非 default** 的 `0x20707`，條件看記錄 50、51，結果碼 1。重製端還沒有這支的實作（`remake/` 找不到 `0x20707`） | `docs/data/battle_events.json`、`26-per-chapter-event-handlers.md` |
| 隊伍 | 名冊 11 人（第九章戰後沒有 JOIN）；`ch10.json` 的 party 寫 12 人含萊汀，要用建槽結果核對 | `ch10.json`、建槽 manifest |
| 開場群組 | `initial_groups [0]`：map 9 group 0 共 42 筆——敵 39（unit 0 是 Boss，(15,4)）、友軍 2（unit 39 (24,8)、unit 40 (6,8)，raw `+6=1`）、我方 FDFIELD 記錄 1（unit 41 (15,40)，raw `+6=2`，fig 6） | `map9_units.json` |
| 記錄索引（強推論） | 名冊 11 人時 group 0 從記錄 11 起，unit 39／40 正好是記錄 50／51，與 `0x20707` 的條件一致，應是索菲亞與卡納恩三世；unit 41 是記錄 52（萊汀）。用收據核對，不從名字推 | 推算 |
| 戰前 handler | `ch09_pre`：LOADCH → pan → text 0 → 聚焦 | `cutscenes/handlers/ch09_pre.json` |
| 回合事件 | 控制列：回合 5 → 事件 32（raw camp 1，selector 1）、回合 20 → 事件 33（selector 0）。**兩個都還沒轉寫**：`ch10.json` 的 `reinforce_ch10_e32_t5` 是 gen_campaign 的舊版本（登場 group 1、8 筆友軍）。handler 在全域事件表：32＝`0x34BE2`（`0x34BEE` 呼叫 `0x10B4E` group 1 gate 0）、33＝`0x34C1E` | `native_turn_event_controls.json`、`event_id_groups.json` |
| 死亡程式 `2:34` | 觸發單位 map 9 unit 0（Boss）；`0x34C6C` 跳到共用的 `0x34906` 對白，text 3 共 5 句 | `native_death_events.json`、`ch10.json` |
| 戰後 handler | `ch09_post`：`0x1F882` 暗化 → `0x13536` → 直接改我方記錄座標 → 重繪 → 淡入 → text 4 → ACTING 37 → text 5 → 同步隊伍 → JOIN 11（索菲亞）、JOIN 6（萊汀）→ chapter=10；binding `runtime_context.slot_counts [60, 61]` 還沒用收據核 | `cutscenes/handlers/ch09_post.json`、binding |
| 寶箱 | 9 格：物品 4、194、192、202、196、193，金錢 10000 三格 | `map9_units.json chests` |

要先確認的語意：

- `0x20707` 的完整規則：記錄 50／51 是「任一倒下就輸」還是別的，結果碼 1 在外層代表什麼；同一支
  handler 是否也負責勝利判定。先 RE 寫進 `26`／`58`，再接 runtime；未證實前失敗即關閉。
- 被保護的兩個友軍一開場就在北邊敵陣裡（y=8），我方從 y=39..44 出發。強化槽能不能在他們倒下前
  清到附近，第一輪原版側就要記「幾回合時兩人各剩多少 HP」。若倒下就是敗北，照 §2 的處置。
- 事件 32 在 selector 1（我方回復之後、友軍 AI 之前）登場 8 筆友軍，之後友軍 AI 的筆數變多，
  照 56 的 `0x1A30B` 順序規則核對。

## 2. 第十章工作單元（順序固定，做完才提交）

1. **轉寫**：事件 32 `0x34BE2`、事件 33 `0x34C1E` 進 `tools/extract_native_death_events.py`（每條非序言
   指令都要被認領）；`sync_native_turn_events.py --chapters …,10` 把 `reinforce_ch10_e32_t5` 換成轉寫版；
   死亡程式 `2:34` 核對 5 句與尾段。`0x20707` 的 RE 與規格一起寫。不能先接 runtime 再補文件。
2. **建槽**：`tools/fd2_chapter_slot.py build --base work/parity-state/ch02-cleared/FD2.SAV --target 9
   --levels-per-chapter 6 --seed 4 --event-states 7:17=1 --boost-base-ap 200 --boost-base-dp 0
   --boost-base-dx 60 --out-dir work/parity-slot-ch10/`（先看 `ch08_post` 有沒有讀戰場狀態的分支）。
   正對照：與第九章 remake-r2 寫出的酒店存檔逐欄比。合法性檢查 `ch10-slot-load.jsonl`（LOAD 進洞窟中的
   激戰城鎮、五棟建築），manifest 複製到 `docs/data/parity-slots/ch10-manifest.json`。
3. **計畫** `ch10-sample.jsonl`：LOAD → 出口（選項順序照 ch09 計畫） → 戰前 → 全隊北上接戰、
   瑪琳與貝克威留守（skip indices 用建槽名冊核對）→ 第 5 回合結束抽事件 32 → 推到擊倒 Boss（死亡程式 34，
   五句對白）→ `force_enemy_clear` → 戰後（ACTING 37、JOIN 11／6）→ `town_ch11` 五棟、一買一賣、酒店存檔、
   祕密商店（scan 93）。開跑前報時長。事件 33 在第 20 回合，不刻意去抽，limitations 記下。
   - **保護目標倒下的處置**：原版側若在 Boss 倒下前讓記錄 50／51 倒下而敗北，不反覆刷關；先把
     `force_enemy_clear` 提前到倒下前一回合、Boss 另找回合擊倒，仍不行就停下來問使用者（例如改用
     另一組政策值或接受不抽死亡程式 34）。
4. **原版側**：`FD2_ORACLE_STATE=<覆蓋層> FD2_ORACLE_FORCE_ENEMY_CLEAR=1
   FD2_ORACLE_EIP_TRACE=0x13A9F,0x1E54A,0x4E893,0x34BE2,0x34C1E,0x34C6C,0x20707,0x12CEA
   tools/dosgolem_oracle.sh work/parity-slot-ch10/sample-rN docs/data/parity-plans/ch10-sample.jsonl`。不用
   `FD2_ORACLE_LOCK_ALLY_HP`。dosgolem 缺口進 `~/cht/dosgolem-fd2-oracle` `feat/fd2-oracle-input-chain` 並
   提交，runner.json 不得 dirty。
5. **重製側**：`tools/chapter_parity.sh 10 …`。`battle_ch10` 補 `native_map_view`（抄原版
   battle_start）與 `native_map_hud_inherited`，改完重跑 `tools/export_editor_canonical.py --output
   remake/assets/editor-canonical --without-animations`（執行期讀 canonical bundle）；`ch09_post` 的
   `runtime_context.slot_counts` 用收據核；`ai_order` 分岔 0。鏡頭或可見游標差一格時，先用
   `FD2_FOCUS_TRACE_OUT` 與原版 `0x12CEA` eip-trace 逐筆對照。
6. **抽樣截圖**：`tools/parity_sample_sheet.py`，指定點至少包含 Boss 倒下那一擊、死亡程式 34 對白、
   事件 32 之後的第一個比較點、戰後城鎮。
7. **收尾**：任一 gate 失敗開 缺陷／RE待解 issue 修掉重跑；全過後台帳第十章 `passed`（`slot_policy`
   照 114），**不重跑第四～九章**；#40 留言並關閉；56 加第十章段（含 #40 的重繪入口規格與結果）、
   57／58 各一段、91-history 一則，教訓寫成規則，`#33` 留言記事件 32／33 轉寫，
   `fd2_worklist_issues.py pull` → `fd2_worklist.py verify` → 全套 Go 測試 → commit，push 前問一次。

## 已知規則（沿用第九章，新增的放最後）

- 0x1A30B 順序：我方回復 → selector 1 事件 → `sub_1A866(1)` → 友軍 AI → 橫幅 → 0x13536 →
  selector 0 事件 → `sub_1A866(0)` → 敵軍兩遍；selector 2 事件與 `sub_1A866(2)` 在 PLAYER PHASE
  橫幅後、輸入前。
- 0x205b4 勝負只看已登場記錄；0x11506 照抄 +2/+3；存檔差 byte 用 fd2save.py 對回欄位。
- oracle_mid_end_turn 不比；checkpoint 落在 0x373C4／0x11EB0／0x4E809 內 dosgolem 已延後 PNG；
  敵方回合鏡頭是 0x12CEA 游標協定。
- 第七章：格子事件每步記、行動收尾才分派；擊倒掉落是同步對話；指令 13～16 回復量走 `0x4E893`；
  `0x134E4` 每個行動收尾全員 `+3` 歸零；升級同步 `+0x42`／`+0x46`；JOIN 記錄殘值疊底。
- 第八章：`0x1598A` 施法落點用成本列 0；mode 0／1 移動後備沒走成經 `0x13C06` 呼叫 `0x13FD4` 回復
  MaxHP/5；射程只從 ID < 0x80 的已裝備武器取，FDFIELD 直接登場的我方記錄也要套；重播端 mark 先推
  到該回合操作權、select 前重走方向鍵（起點一致才走）；建槽戰後分支的戰場狀態要用
  `--event-states` 明示、seed 沿用 4；`FD2_ORACLE_LOCK_ALLY_HP` 在有施法的章節不可用。
- 114／第九章：死亡程式對白疊在移除屍體、行動者尚未設 `+5` bit7 的重繪底圖上；原生對白照 FDTXT
  原始字模畫（`glyph_ids`，改過字模表跑 `backfill_native_dialogue_glyph_ids.py --check`）；`0x1548E`
  聚焦自己在 `0x14B78` 移動之前，走完只聚焦目標；重播端 END 前重走推游標的方向鍵（副本上走到 `at`
  才重走）、列舉相位變體時凍結 BIOS 取樣；`0x1A30B` 的回合開頭聚焦時閘 B 仍是 0，返回後才寫回；
  事件 31 類「休眠控制列」由 `control_turn` 啟用，劇本以 `native_turn_events.actions` 表示；戰後
  `runtime_context.slot_counts` 用收據核，推出的值要寫明是推出的。

## 紀律

- 改過 Go 檔最後再跑 `tools/rebind_string_review.sh`（新增的診斷字串要手動歸類再寫回）→ 複製
  inv-new.json 成 `docs/data/fd2-string-inventory.json`（gitignore）→ Docker 內 `go run . -repo ../../..
  -summary -output /src/docs/data/fd2-string-inventory-summary.json` → `tools/migrate_full_locale_content.py`
  → 全套 Go 測試。
- gofmt／go vet 在 Docker 跑；全套 `tools/remake_go_test.sh <素材根> <log> ./...`；oracle 跑的時候
  不要同時起全套回歸。
- 掛單一檔案進容器前先 `test -f`（原版在儲存庫內 `org_game/炎龍騎士團/FLAME2/`），來源不存在時
  dockerd 會以 root 建空目錄。
- 提交前分開看每項檢查再 commit；push 前問一次；每批 Docker 後 `docker ps -a` 檢查 FD2 容器，禁止
  任何 prune／rmi；不改 #15／#9／#6／#7／#8／#4／#10／#11。
