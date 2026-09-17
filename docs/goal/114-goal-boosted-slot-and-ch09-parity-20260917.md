# 114 — 強化建構槽（AP／DP／DX 政策值）＋第九章工作單元（騎士的抉擇，map 8）（2026-09-17）

> 這是 [`111`](111-goal-original-parity-campaign-20260915.md) 的下一個工作單元，接在
> [`113`](113-goal-ch08-parity-20260917.md)（第八章已 passed，提交 `de83d928`）之後。
> 接手時先讀 `CLAUDE.md`，再讀 `111`，再讀本檔。本檔只定義這一輪的順序、門檻與交付物；
> 證據分級與位址規則仍以 `56`／`57`／`58` 為準，結果寫回那三份、`91-worklist-history.md`
> 與台帳，本檔不承載結果。

## 為什麼要強化

第八章以建構槽的隊伍正常打撐不久：r1 第 3 回合起陸續陣亡、第 10 回合只剩 4 人，只能提早
`force_enemy_clear`，死亡程式 29 與第 15 回合 event 28 都沒抽到。章內週期性鎖 HP 也不能用
（寫回會落在施法演出中途，原版在 `0x4E6BD` 解碼失控）。之後的章節敵人只會更強，而且第九章
的主線演出（騎士倒下後倒戈）掛在**擊倒 Boss** 的死亡程式上，打不倒就抽不到。

## 使用者定案（2026-09-17）

| 項目 | 定案 |
|---|---|
| 強化位置 | 建槽時寫進存檔：原版與重製端 LOAD 同一份 bytes，章內不注入 |
| 強化幅度 | 固定政策值，所有章相同；先用第八章校準一次，定案後寫進本檔與 manifest。DP 不能拉太高，閃避可以拉高 |
| 證據等級 | 視同 111 例外的 `PLAYER-E2`：強化是建構槽政策的一部分，和每章升級政策同等 |
| 範圍 | 機制＋第八章校準回補＋第九章 |

## 0. 規則同步（先做，不改碼）

- `CLAUDE.md`「實作與驗證」的 111 例外、`111`「起點存檔與證據等級」、`56` §2 各補一句：建構槽
  可以依受版控工具的明示政策值提高我方基底 AP／DP／DX，收據視同 `PLAYER-E2`；政策值與「不得用來談
  傷害、存活、敵方選目標」的限制寫進 manifest assumption 與收據 limitations。
- 台帳 `parity-campaign-progress.json` 每章加 `slot_policy` 欄位（升級、seed、event-states、強化值）；
  `tools/fd2_parity_progress.py verify` 檢查 `passed` 的章都有這一欄。

## 1. 強化機制（`remake/cmd/fd2-chapter-slot`）

- 新旗標 `-boost-base-ap N`、`-boost-base-dp N`、`-boost-base-dx N`（包裝器 `--boost-base-ap`／
  `--boost-base-dp`／`--boost-base-dx`）：對名冊每筆記錄把 `+0x37`（基底 AP）、`+0x39`（基底 DP）、
  `+0x3E`（DX）加上 N，**再跑工具既有的重算**讓 `+0x48`／`+0x4A`／`+0x4C`／`+0x4E` 與裝備一致。
- **閃避沒有獨立的基底欄位（已證實）。** `0x1145A`／`0x1B750` 的重算以 `+0x3E` DX 同時當 HIT 與 EV
  的基底，再加裝備列 `+3`／`+7`（`remake/internal/battle/native_equipment.go nativeEquipmentTotals`）。
  要拉高閃避就是拉高 DX，命中會一起變高；不要直接改 `+0x4E`，下一次重算就蓋回去。不直接改 `+0x48`／`+0x4A`：下一次換裝或升級重算就會蓋回去，兩側
  會在不同時點分岔。word 上限 `0x7FFF` 失敗即關閉。
- 強化在所有章節套完、升級政策之後做一次；manifest 的 `assumptions` 記一筆 `boost`（值、套用筆數、
  前後 AP／DP）。
- 測試：同一組參數不帶強化與帶強化建槽，逐欄比只有 `+0x37`／`+0x39`／`+0x3E`／`+0x48`／`+0x4A`／
  `+0x4C`／`+0x4E`（與檢查碼）不同；重算恆等檢查仍通過；第七章槽不帶強化重建 sha 仍是 `85b080cb…`。
- **HP 不在這一輪強化**：先看 AP／DP／DX 夠不夠。第八章校準兩輪後仍有陣亡再議。

### 選值的限制（校準時要守）

- **DP 不能高到敵人不攻擊。** `0x14237` 物理候選在 `actor +0x48 − target +0x4A <= 2` 時直接略過
  （11-enemy-ai.md §物理攻擊候選）；DP 拉太高，敵人全部改走移動與 `0x13FD4` 回復，敵方攻擊、
  反擊、升級這些戰鬥節拍就抽不到了。校準要確認每回合仍有敵方攻擊命中我方。
- **閃避（DX）可以拉高。** AI 物理評分只看 `+0x48`／`+0x4A`，不看 EV，所以敵人照樣出手，只是比較常
  被閃掉；攻擊、未命中與反擊的節拍仍抽得到。命中與閃避的擲骰公式要先查 `58` 確認讀的是 `+0x4C`／
  `+0x4E`，再決定加多少；每回合至少要有一次敵方命中，否則「受傷、升級對話底圖、我方回復」這些
  節拍會消失。
- **AP 要讓 Boss 在幾回合內倒得下。** 第八章騎士 slot 10 是 300 HP，第九章 Boss（map 8 unit 0，
  Lv18）要在第 6 回合前後倒下，才有時間抽事件 30 之後的回合。
- 施法傷害走指令記錄與另一條公式，AP 強化不一定影響；敵方施法者仍可能打死低 HP 的隊員，
  計畫上照舊讓後排留守。

## 2. 第八章校準與回補

- 建槽：`tools/fd2_chapter_slot.py build --base work/parity-state/ch02-cleared/FD2.SAV --target 7
  --levels-per-chapter 6 --seed 4 --event-states 7:17=1 --boost-base-ap A --boost-base-dp D
  --boost-base-dx X --out-dir work/parity-slot-ch08-boost/`，合法性檢查沿用 `ch08-slot-load.jsonl`。
- 計畫另存 `docs/data/parity-plans/ch08-boost-sample.jsonl`：第 1～N 回合北上接戰，**擊倒騎士
  slot 10**（死亡程式 29：text 2、slots 10..27 `+0x34 &= 0x80`），推到第 15 回合抽 event 28，
  之後 `force_enemy_clear` → 戰後 → 城鎮收尾（同 r1）。開跑前報時長。
- 校準最多三輪，每輪只改 A、D、X 其中一個，記下「第幾回合騎士倒、有無陣亡、每回合敵方出手與命中
  次數」。定案的 A、D、X 寫回本檔下面的表與 56，之後各章固定用這組值。建議順序：先加 A 與 X，
  D 維持小值。
- 收據：通過後 `parity-ch08.json` 換成強化版（死亡程式 29 與 event 28 補上原版收據），舊的
  sample-r1 收據移到台帳的 `prior_receipts`；57／58 的第八章段落改寫成現況，limitations 拿掉
  「沒有原版收據」那一條。

| 政策值 | 定案 | 校準輪 | 依據 |
|---|---|---|---|
| 基底 AP 加值 A | （待校準） | | |
| 基底 DP 加值 D | （待校準，維持小值） | | |
| DX 加值 X（HIT 與 EV 共用基底） | （待校準） | | |

## 3. 第九章：這一章有什麼（開工前已知，動手時用收據核）

| 項目 | 內容 | 出處 |
|---|---|---|
| 節點 | 起點 `town_ch09`（祕密商店 selection 2、scan 101＝Ctrl+F8）→ `preparation_ch09` → `story_ch09`（raw `ch08_pre`）→ `battle_ch09`（map 8，25×36，目標敵全滅）→ `postbattle_ch09_persist`（raw `ch08_post`）→ `town_ch10`「洞窟中的激戰」（variant 2、祕密商店 selection 3、scan 112＝Alt+F9） | `campaign_full.json` |
| 隊伍 | 11 人（第八章 JOIN5 洛娜入隊），佈陣 11 格 y 33..35 | `ch09.json` |
| 開場群組 | `ch09.json` 寫 `initial_groups [0,2,3,4]`；但戰後 handler 會登場 group 4、事件 30 會登場 group 1。**先照第八章的做法用 `0x1088D` 核對開場只登場哪些群組**，不對就改劇本 | `ch09.json`、`ch08_post.json` |
| 戰前 handler | `0x3327D`：LOADCH → pan (6,0) → text 0 → ACTING 35 → text 1 → 聚焦記錄 0 → `0x134E4` | `cutscenes/handlers/ch08_pre.json` |
| 回合事件 | 沒有靜態回合事件；控制列 slot 0／1 是休眠的 `(255, 31, 0)`（FDFIELD_025 控制段），由事件 30 啟用 | `tools/parse_field.py native_turn_event_controls` |
| 死亡程式 `2:30` | 觸發單位 map 8 unit 0（Boss，Lv18，(12,15)，raw `+0x34=0x82`）。`0x34A7A`：slots 12..33 `+0x34` 整個寫 0 → 控制列 slot 0 回合＝目前回合＋1、slot 1＝＋2（事件 31、selector 0）→ 記錄 11 寫 `+5=0`、`+6=1`、`+7=6`、`+8=6`、`+0x31=0xFF`、`+0x34=0x80`、HP=1 → text 2 → 登場 group 1 → text 3 → 本次行動不給經驗 → state16=2 | `native_death_events.json` |
| 事件 31 | **還沒轉寫**（全域事件表 `0x51B91[31]`）；第九章的主線在事件 30／31 | — |
| 戰後 handler | `0x235BC`：pan (6,1) → 登場 group 4 → ACTING 36 → text 4 → `0x11506` → chapter=9；沒有 JOIN | `cutscenes/handlers/ch08_post.json` |
| 掉落 | unit 5、7、19、22 物品（202、206、193、200），unit 18 金錢 1000 | `map8_units.json` |

要先確認的語意：

- 記錄 11 的索引：隊伍 11 人時 runtime slot 11 正好是 group 0 第一筆（Boss 自己），事件 30 就是
  「Boss 以 1 HP 倒戈成友軍、身分改成 6」。用收據核對，不從名字推。
- 倒戈之後有 `+6==1` 的友軍，`0x1A30B` 的友軍 AI（`0x1D80B`）會開始跑；第七章凱麗之後這是第二個
  友軍，照 56 的順序規則核對。
- 勝負判定：第九章用 default `0x205b4`，友軍與已變友軍的 Boss 不算敵方。

## 4. 第九章工作單元（順序固定，做完才提交）

1. **轉寫**：事件 31 進 `tools/extract_native_death_events.py`（每條非序言指令都要被認領），事件 30
   已有；`sync_native_turn_events.py` 要能處理「由 `control_turn` 動態啟用的休眠列」——目前工具只讀
   `turn_events.json` 的靜態列，要先查 `0x1A813` 怎麼掃控制列，再決定劇本怎麼表示「第 k+1、k+2
   回合的事件 31」（k 是 Boss 倒下的回合）。不能先接 runtime 再補文件。
2. **建槽**：`--target 8 --levels-per-chapter 6 --seed 4 --event-states 7:17=1 --boost-base-ap A
   --boost-base-dp D --boost-base-dx X`（第八章戰後沒有讀戰場狀態的分支就不用加新的 event-states，先看 `ch07_post`
   確認）。正對照：與強化版第八章重製端寫出的酒店存檔逐欄比。合法性檢查
   `ch09-slot-load.jsonl`。
3. **計畫** `ch09-sample.jsonl`：LOAD → 出口 → 戰前 → 北上擊倒 Boss（事件 30：倒戈、登場 group 1、
   兩句對白）→ 再推兩回合抽事件 31 → 接戰幾回合抽友軍 AI 與敵方回合 → `force_enemy_clear` → 戰後
   （group 4 登場、ACTING 36、text 4）→ `town_ch10` 五棟、一買一賣、酒店存檔、Alt+F9 祕密商店。
   擊倒 Boss 那一擊要用我方攻擊（事件 30 有 `exp_cancel`，收據要能看到這次沒經驗）。
4. **原版側**：`FD2_ORACLE_STATE=<覆蓋層> FD2_ORACLE_FORCE_ENEMY_CLEAR=1
   FD2_ORACLE_EIP_TRACE=0x13A9F,0x1E54A,0x4E893 tools/dosgolem_oracle.sh
   work/parity-slot-ch09/sample-rN docs/data/parity-plans/ch09-sample.jsonl`。不用 `FD2_ORACLE_LOCK_ALLY_HP`。
   dosgolem 缺口進 `~/cht/dosgolem-fd2-oracle` `feat/fd2-oracle-input-chain` 並提交，runner.json 不得 dirty。
5. **重製側**：`tools/chapter_parity.sh 9 …`。`battle_ch09` 補 `native_map_view`（抄 battle_start）；
   `ch08_post` 的 `runtime_context.slot_counts` 用收據核（事件 30 登場 group 1 之後的筆數）；
   `ai_order` 分岔 0。
6. **抽樣截圖**：`tools/parity_sample_sheet.py`，指定點至少包含 Boss 倒下那一擊、事件 30 兩句對白、
   事件 31、戰後 ACTING 36。
7. **收尾**：任一 gate 失敗開 缺陷／RE待解 issue 修掉重跑；全過後台帳第九章 `passed`，用同一份重製端
   重跑第四～八章確認仍過（第八章用強化版收據），56 加第九章段、57／58 各一段、91-history 一則、
   教訓寫成規則，`#33` 留言記第九章轉寫，`fd2_worklist_issues.py pull` → `fd2_worklist.py verify` →
   commit，push 前問一次。

## 已知規則（沿用第八章，新增的放最後）

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

## 紀律

- 改過 Go 檔最後再跑 `tools/rebind_string_review.sh` → 複製 inv-new.json 成
  `docs/data/fd2-string-inventory.json`（gitignore）→ Docker 內 `go run . -repo ../../.. -summary
  -output /src/docs/data/fd2-string-inventory-summary.json`（輸出要給容器內絕對路徑）→
  `tools/migrate_full_locale_content.py` → 三個語言包驗證。
- gofmt／go vet 在 Docker 跑；全套 `tools/remake_go_test.sh <素材根> <log> ./...`；oracle 跑的時候
  不要同時起全套回歸。
- 提交前分開看每項檢查再 commit；push 前問一次；每批 Docker 後 `docker ps -a` 檢查 FD2 容器，禁止
  任何 prune／rmi；不改 #15／#9／#6／#7／#8／#4／#10／#11。
