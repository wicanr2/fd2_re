# 116 — README 各章對拍進度與自我評分（#42）＋第十一章工作單元（幻之森林，map 10）（2026-09-18）

> 這是 [`111`](111-goal-original-parity-campaign-20260915.md) 的下一個工作單元，接在
> [`115`](115-goal-hud-anchor-and-ch10-parity-20260917.md)（#40 HUD anchor 入口、第十章已 passed，
> 提交 `9fe60752`）之後。接手時先讀 `CLAUDE.md`，再讀 `111`，再讀本檔。本檔只定義這一輪的
> 順序、門檻與交付物；證據分級與位址規則仍以 `56`／`57`／`58` 為準，結果寫回那三份、
> `91-worklist-history.md` 與台帳，本檔不承載結果。

## 使用者定案（2026-09-18）

| 項目 | 定案 |
|---|---|
| 這一輪做兩件事 | 先做 issue [#42](https://github.com/wicanr2/fd2_re/issues/42)：`README.md` 補各章對拍進度、`docs/REMAKE-STATUS.md` 自我評分更新到現況；再做第十一章工作單元 |
| 建構槽政策 | 沿用 114：`--levels-per-chapter 6 --seed 4 --event-states 7:17=1 --boost-base-ap 200 --boost-base-dp 0 --boost-base-dx 60`，target 改 10 |
| 回歸範圍 | 沿用 115 的定案：已 passed 的章不因為新修正而重跑；第十一章的收據就是這一輪的驗收 |

## 0. 先做 #42：README 各章對拍進度＋自我評分

現況：`README.md` 的里程碑段只有第一輪 60／60 抽樣那組敘述，看不出 111 逐章對拍已經做到第十章；
`docs/REMAKE-STATUS.md` 的自評表停在 2026-09-08，「本次完整流程對拍」那一列還寫「當時暫停交人工」。

做法（順序固定）：

1. **數字從台帳來，不手抄。** 進度來源是 [`docs/data/parity-campaign-progress.json`](../data/parity-campaign-progress.json)
   與各章收據 `docs/data/ui-traces/parity-chNN.json`。要在 README 放表就寫一支小工具（例如
   `tools/render_parity_progress.py`）由台帳產生那一段，像 worklist 一樣標出產生區塊；不要在 README
   留會漂移的手抄統計。
2. **README 加一段各章對拍進度**：每章狀態、收據連結、四個 gate 是否通過、抽樣圖入口，以及一句
   限制（強化槽政策值不是原版證據、`force_enemy_clear` 是修改路徑、`PLAYER-E2` 例外依 111／114）。
   文化保存、玩家敘事與技術保存段落不動（`README-RESTORE.md` 的規則照舊）。
3. **`docs/REMAKE-STATUS.md` 自評更新到當日**：對拍那一列換成「第四～十一章已 passed，收據與畫面點
   統計」；其他列若證據沒重做就標日期與「未重測」，不要為了好看調分。計分限制寫清楚：分數不能
   相加或平均成完成率。
4. 驗收：#42 的 verify 是 `docs/REMAKE-STATUS.md` 裡「當時暫停交人工」那句消失；改完留言寫提交與
   數字來源，再 `fd2_worklist_issues.py close`。

## 1. 第十一章：這一章有什麼（開工前已知，動手時用收據核）

| 項目 | 內容 | 出處 |
|---|---|---|
| 節點 | 起點 `town_ch11`「幻之森林」（variant 2、祕密商店 selection 4、scan 93＝Shift+F10）→ `preparation_ch11` → `story_ch11`（raw `ch10_pre`）→ `battle_ch11`（map 10，35×45，BGM `FDMUS_019`）→ `postbattle_ch11_persist`（raw `ch10_post`）→ `town_ch12`「北山道」（variant 1、祕密商店 selection 0、scan 94） | `campaign_full.json` |
| 目標 | 敵全滅；本章招募「珊」 | `28-chapter-objectives-and-recruits.md` |
| 勝負 handler | 章索引 10 走 default `0x205B4`（掃 `+6==0 && (+5&1)==0`），不是第十章那種專用 handler | `26-per-chapter-event-handlers.md` |
| 隊伍 | 第十章戰後 JOIN 11／6 之後名冊 13 人（`ch11.json` party 13、`deploy_cells` 13）；用建槽 manifest 核對 | `ch11.json`、第十章收據 |
| 開場群組 | `initial_groups [0, 1]`：group 0 是 25 筆敵方，group 1 只有 1 筆友軍（fig 14、HP 20、(17,5)、`cls 255`）。另有 14 筆 `group 255` 的佔位列，不登場 | `map10_units.json` |
| 回合事件 | `turn_events.json` 的 map 10 是空陣列——這一章沒有回合事件控制列。若原版收據出現登場或演出，當成新發現先 RE 再接 | `docs/data/turn_events.json` |
| 死亡程式 | `native_death_events.json` 沒有任何 `battle_units` 指向 map 10 → 這一章沒有死亡程式。同樣以收據反證為準 | `native_death_events.json` |
| 戰前 handler | `ch10_pre`：LOADCH → text → pan → spawn → ACTING → text → ACTING → text → 重設姿態 → 聚焦。比第十章的 `ch09_pre` 多兩段 ACTING 與兩段對白，要逐段核 | `cutscenes/handlers/ch10_pre.json` |
| 戰後 handler | `ch10_post`：text → 同步隊伍 → JOIN（珊）→ chapter=11；binding 沒有 `runtime_context`，要看戰末記錄筆數要不要補 | `cutscenes/handlers/ch10_post.json`、binding |
| 寶箱 | 12 格，內容以 `map10_units.json` 的 `chests` 為準；第十章已閉合金錢與物品兩條分支（`0x190AC`） | `map10_units.json` |
| 輔助底面 | raw chapter 10 **不在** `0x10652` 的 9／24／25／28／29 名單，這一章沒有 FDOTHER 底面；畫面差異不要再往那邊猜 | `fd2_chapter_aux_graphics_10652_ida.txt` |

要先確認的語意：

- group 1 那一筆友軍（fig 14、HP 20）是誰、開場就在敵陣北側 (17,5)：它倒下會不會影響勝負或招募。
  `0x205B4` 只看 `+6==0`，所以理論上不影響勝負，但招募（珊）在戰後 handler，要用收據確認。
- `ch10_pre` 的兩段 ACTING 與 spawn 的來源位址要對到 handler 腳本的 `source.addr`；binding 的
  `overrides` 目前是空的，若 LOADCH／pan 參數需要覆寫，先看原版 eip-trace 再補。

## 2. 第十一章工作單元（順序固定，做完才提交）

1. **轉寫與 RE（先寫文件）**：這一章沒有回合事件與死亡程式，重點在 `ch10_pre` 的 spawn／ACTING 段與
   `ch10_post` 的 JOIN。動手前先確認 `sync_native_turn_events.py --chapters 11` 的輸出是空的（與
   `turn_events.json` 一致），不要憑舊 gen_campaign 的殘留資料開工。
2. **建槽**：`tools/fd2_chapter_slot.py build --base work/parity-state/ch02-cleared/FD2.SAV --target 10
   --levels-per-chapter 6 --seed 4 --event-states 7:17=1 --boost-base-ap 200 --boost-base-dp 0
   --boost-base-dx 60 --out-dir work/parity-slot-ch11/`。正對照：與第十章 remake-r6 寫出的酒店存檔逐欄比
   （第十章那份 sha `9f36bbee…` 與原版相同，是現成的正對照）。合法性檢查 `ch11-slot-load.jsonl`
   （LOAD 進幻之森林城鎮、五棟建築），manifest 複製到 `docs/data/parity-slots/ch11-manifest.json`。
3. **計畫** `ch11-sample.jsonl`：LOAD → 出口 → 戰前（`ch10_pre` 全段）→ 北上接戰、留守單位用建槽名冊
   核對 → 三個回合標記 → 清場 → 戰後（JOIN 珊）→ `town_ch12` 五棟、一買一賣、酒店存檔、祕密商店
   （selection 0、scan 94）。**回合數與指令預算一起估**：後段章一回合約 1.5e9 道指令，抽樣回合夠用就
   提前 `force_enemy_clear`，需要更長時設 `FD2_ORACLE_STEPS`（第十章用 2.5e10 跑到第 10 回合清場）。
   開跑前報時長。
4. **原版側**：`FD2_ORACLE_STATE=<覆蓋層> FD2_ORACLE_STEPS=<預算> FD2_ORACLE_FORCE_ENEMY_CLEAR=1
   FD2_ORACLE_EIP_TRACE=0x13A9F,0x1E54A,0x4E893,0x12CEA tools/dosgolem_oracle.sh
   work/parity-slot-ch11/sample-rN docs/data/parity-plans/ch11-sample.jsonl`（這一章沒有事件 handler 要
   追，EIP 清單比第十章短；出現新演出再加）。dosgolem 缺口進 `~/cht/dosgolem-fd2-oracle`
   `feat/fd2-oracle-input-chain` 並提交，runner.json 不得 dirty。
5. **重製側**：`tools/chapter_parity.sh 11 …`。`battle_ch11` 目前沒有 `native_map_view`／
   `native_map_hud_inherited`，要照原版 battle_start 補上，改完重跑
   `tools/export_editor_canonical.py --output remake/assets/editor-canonical --without-animations`。
   鏡頭或可見游標差一格時先用 `FD2_FOCUS_TRACE_OUT` 與原版 `0x12CEA` eip-trace 逐筆對照。
6. **抽樣截圖**：`tools/parity_sample_sheet.py`，指定點至少包含戰前 ACTING 兩段之後的第一個比較點、
   group 1 友軍附近的一戰、清場後戰後城鎮、祕密商店。
7. **收尾**：任一 gate 失敗就開 缺陷／RE待解 issue 修掉重跑；全過後台帳第十一章 `passed`
   （`slot_policy` 照 114）；`56`／`57`／`58` 各一段、`91-worklist-history.md` 一則，教訓寫成規則；
   `fd2_worklist_issues.py pull` → `fd2_worklist.py verify` → `labels --apply` → 全套 Go 測試 → commit，
   push 前問一次。

## 3. 帶過來的未決項（這一輪要記得，不一定要做）

- **[#41](https://github.com/wicanr2/fd2_re/issues/41)**：攻擊演出收尾（`0x1D3FF`、`0x15510`／`0x1563B`／
  `0x1565C`）、`0x1DB65` 的 `0x1DEAE`、訊息關框 `0x19742` 三類重繪還沒經過 HUD anchor 入口。第十一章若
  出現對應差異，就在這一輪把它接掉並關 #41；沒出現就維持未接，不自行加評估點。
- **第四～十章重跑**：115 這一輪改了三處會影響舊章重播的行為（分派器停留聚焦、`0x15055` 道具原地
  使用、`0x112A5` JOIN 旗標與暫態清零）。台帳與收據維持 `passed`，但下次重跑那些章時要以新收據確認；
  要不要在這一輪順帶重跑第九／十章當抽查，動手前問使用者（跑一章原版側約兩小時）。
- **事件 33／34（第十章）** 沒有原版收據；`0x15055` 以外的 item route 與玩家法術／物品仍沒有驅動端指令。
- **私人素材庫**：第十章的 `surfaces/FDOTHER_015` 已同步（`e3486da0`）；新抽素材一律照這個流程。

## 已知規則（沿用第十章，新增的放最後）

- `0x1A30B` 順序：我方回復 → selector 1 事件 → `sub_1A866(1)` → 友軍 AI → 橫幅 → `0x13536` →
  selector 0 事件 → `sub_1A866(0)` → 敵軍兩遍；selector 2 事件與 `sub_1A866(2)` 在 PLAYER PHASE
  橫幅後、輸入前。
- `0x205B4` 勝負只看已登場記錄；`0x11506` 照抄 `+2`／`+3`；存檔差 byte 用 `fd2save.py` 對回欄位。
- `oracle_mid_end_turn` 不比；checkpoint 落在 `0x373C4`／`0x11EB0`／`0x4E809` 內 dosgolem 已延後 PNG；
  敵方回合鏡頭是 `0x12CEA` 游標協定。
- 第七～九章：格子事件每步記、行動收尾才分派；升級同步 `+0x42`／`+0x46`；`0x1598A` 施法落點用成本列 0；
  射程只從 ID < 0x80 的已裝備武器取；死亡程式對白疊在移除屍體、行動者尚未設 `+5` bit7 的重繪底圖上；
  原生對白照 FDTXT 原始字模畫（`glyph_ids`）；`0x1548E` 聚焦自己在 `0x14B78` 之前，走完只聚焦目標；
  `0x1A30B` 的回合開頭聚焦時閘 B 仍是 0，返回後才寫回。
- 第十章新增：
  - 分派器 mode 0／3／4／7／9／10 在 `0x14B78` 之前**無條件** `0x12D7B` 聚焦自己，`0x14B78` 回 0 才
    進 `0x13FD4`（其內 HP 滿或 `+0x25`／`+0x26` 非零直接返回）。
  - `0x15055` 道具路由不移動：`[0x53C37/0x53C3B]` 是 `0x1567E` 在原地指令目標場選出的效果格。
  - `0x112A5` JOIN：有物品的格旗標寫 0、空格 0x80，`+0x22..+0x27` 清零；`+0x17`／`+0x19` 仍是殘值（#23）。
  - `0x1A866` 扣血：每筆聚焦 → `0x1956B` → FDTXT `0x1E7` → `0x1E5C0(10)` → `0x196CB`，全部扣完才
    `0x1DB65` 與到期倒數；到期提示也先聚焦。
  - 輔助底面只在 raw chapter 9／24／25（FDOTHER #15）與 28／29（#55）；相位由 oracle 視圖 `aux_phase`
    推回，實測 `-1` 或 `-2`，重播端兩種各出一組（強推論）。
  - 重播端：`replayDirectionKeys` 的前提檢查用第一個方向鍵前一格的檢查點（上一個動作可能在
    `0x1A30B` 換手中途取樣）；指令環穩態另出 `0x1741C` 最後一張展開幀的位置；寶物提問停在 `0x19953`
    讀鍵時取 `wait` 檢查點並把提問框疊在當下整幀上。

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
