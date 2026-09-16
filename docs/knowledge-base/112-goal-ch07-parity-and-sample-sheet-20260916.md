# 112 — 第七章工作單元提示詞：原版一致對拍（往王城途中，map 6）＋抽樣截圖證據（2026-09-16）

> 這是 [`111`](111-goal-original-parity-campaign-20260915.md) 的第七章工作單元開工指令。
> 接手時先讀 `CLAUDE.md`，再讀 `111`，再讀本檔；台帳
> [`parity-campaign-progress.json`](../data/parity-campaign-progress.json) 的第七章仍是 `todo`
> 才動工。本檔只定義這一章的順序、門檻與交付物；證據分級與位址規則仍以 `56`／`57`／`58`
> 為準，做完把結果寫回那三份與 `91-worklist-history.md`，本檔不承載結果。

第四、五、六章已 passed（parity-ch04.json remake-r60／原版側 r13；parity-ch05.json remake-r7／
原版側 r5；parity-ch06.json remake-r11／原版側 r4；dosgolem `50a3b47`）。本階段只做第七章，
順序固定，做完才提交，不跨章。這一章多一個交付物：抽樣截圖對照表，讓人不用跑工具就看得到
「原版／重製同一個 checkpoint 長什麼樣、差幾個像素」。

## 0. 回顧（半小時內，不改碼）

- 看 #34、#35、#37 現況；把第六章收據 `frames.points` 的 58 點加進 56 §111 五章回顧的分類表
  （`diff_pixels>0`／非 ok 的形狀）。有新形狀就開 issue，沒有就一句話記在 56。
- #37（城鎮進戰場淡出與 `sub_1A866` 三個呼叫點的順序）的取樣併進步驟 4 的原版側跑。

## 1. 先轉寫第七章的事件處理器

- map 6 只有一筆回合事件：第 10 回合 event 25 `0x34924`——守 state16==1 → `0x10B4E(2)` 登場
  group 2（10 筆，slots 34..43，最後一筆是凱麗）→ pan `(16,10)` → ACTING resource 30 →
  FDTXT_007 text 2 → state17=1。
- 它的前置 state16 由格子事件 event 26 `0x3499B` 寫：觸發單位 raw `+6!=0` 才過 →
  `0x3419C(9,27,0)` 把記錄 9..27 的 `+0x34` 低四位清 0 → state16=1；只掛在
  `(9,13)(10,14)(11,14)(12,15)(13,15)(14,15)` 六格。證據
  [`fd2_ch06_post_event25_ida.txt`](../data/ida/fd2_ch06_post_event25_ida.txt)。
- 25 與 26 轉寫進 `tools/extract_native_death_events.py` EVENTS（op：`guard_state`／
  `spawn_group`／`pan`／`acting`／`dialogue`／`state_set`／`ai_mode_range`；每條非序言指令都要被
  某個 op 認領），重生 `remake/assets/data/native_death_events.json`，
  `tools/sync_native_turn_events.py --write --chapters 4,5,6,7`。格子事件的消費端
  （`NativeFieldEventRules`／`0x13A44` 路徑）要確認 26 真的由走格觸發、而且只在「向左踏入」
  那一拍（`0x13488→0x1300D→0x13A44` path byte1）——先當強推論，原版側收據證實後再升級。
- 死亡程式 2:24（記錄 24 陣亡對白 `0x34906` text 3）已在 `ch07.json`，核對消費端還在。

## 2. 建槽

`tools/fd2_chapter_slot.py build --base work/parity-state/ch02-cleared/FD2.SAV --target 6
--levels-per-chapter 6 --out-dir work/parity-slot-ch07/`。
正對照：與第六章 remake-r11 寫出的酒店存檔（`work/parity-slot-ch06/remake-r11/FD2.SAV`，與原版
逐 byte 相同，9 人含貝克威 char 13）逐 byte 比；差異只能是升級政策、擊殺經驗、出售與瑪琳的
已知建構偏差（Lv／物品／存活），其餘任何一個 byte 用 `tools/fd2save.py` 對回欄位再決定。
manifest 到 `docs/data/parity-slots/ch07-manifest.json`，合法性檢查
`docs/data/parity-plans/ch07-slot-load.jsonl`（LOAD → 往王城的途中 → 五棟建築）。

## 3. 寫 `docs/data/parity-plans/ch07-sample.jsonl`

LOAD → 出口 YES → 戰前對白（`ch06_pre`）→ 佈陣 9 格（y 28..32，南邊）→ 前幾回合讓一個我方
單位往北走到六格之一，最後一步向左踏入（例如從 `(15,15)` 左踏 `(14,15)`）→ 其餘單位
`typical_move 1` 守位（第六章教訓：主角衝上去第 8 回合就 game over）→ 第 10 回合 event 25
（登場、pan、ACTING 30、四句對白）→ 第 10～11 回合接戰（抽得到記錄 24 陣亡對白更好）→
`force_enemy_clear`（只能在抽完戰鬥節拍之後）→ 戰後 `ch06_post`（`0x233C6` 佈局 override、
text 4、JOIN12 凱麗；slot 43 要活著、state17==1，否則走 text 5 分支，那是另一份收據，
不算這章 passed）→ `town_ch08` 五棟 → 一買一賣 → 酒店存檔 → 祕密商店 chord
（看 `town_ch08.native_secret_gate`）。

- 驅動端目前只有 `goto`（游標）與 `engage`（貼敵），沒有「選這個單位走到那一格」。補一個
  受版控指令 `{"move_unit": {"from": [x,y], "to": [x,y]}}`：選取、把游標推到目的格、確認、
  指令環選待機，並用 checkpoint 的 units 確認那筆座標真的變了。
- 法術／物品／買入驅動端沒有就 limitations 明寫。
- 事前算張數：變體只乘真的獨立變動的維度（56 §五章回顧）。

## 4. 原版側

```
FD2_ORACLE_STATE=<覆蓋層> FD2_ORACLE_FORCE_ENEMY_CLEAR=1 \
FD2_ORACLE_EIP_TRACE=0x13A9F,0x1E54A,0x4E893 \
tools/dosgolem_oracle.sh work/parity-slot-ch07/sample-rN docs/data/parity-plans/ch07-sample.jsonl
```

- `0x4E893` 追蹤從這章起是標配：每個指令施放都要能在 eip-trace 裡對出呼叫端序列
  （指令 0／4 已閉合；敵方或凱麗若放 1／2／3／6／7／8，先拿序列再改重製端，不猜）。
- #37 併在同一跑：出口 YES 之後到 battle_start 之間用 `FD2_ORACLE_FRAMES`＋`FRAME_EIP`／
  stride 抽幀，定出淡出是哪一支函式、在 LOADCH／佈陣前後；結果進 56 的 0x1A30B 段落，
  #37 依證據關掉或改標題。
- dosgolem 缺口進 `~/cht/dosgolem-fd2-oracle` `feat/fd2-oracle-input-chain` 並提交，
  runner.json 不得 dirty；撞到第二個 opcode 就 capstone 掃整族一次補齊。
- 原版側跑時不要同時起 Go 全套件回歸（單一 replay 測試可以）。

## 5. 重製側

`tools/chapter_parity.sh 7 work/parity-slot-ch07 work/parity-slot-ch07/sample-rN
work/parity-slot-ch07/remake-rN`。`battle_ch07` 補 `native_map_view`（抄 battle_start 那張
checkpoint，不是戰中的）與 HUD；`ch06_post` 的 `runtime_context.slot_counts` 改實跑 frontier
（`[34, 44]` 要用收據核，group 255 五筆不算）；`ai_order` 分岔必須 0。

## 6. 抽樣截圖證據（新交付物，受版控工具）

- 寫 `tools/parity_sample_sheet.py <receipt> <oracle run> <remake run>
  --out docs/figures/parity-ch07-samples.png --index docs/data/ui-traces/parity-ch07-samples.json`。
  從 `frames.points` 抽樣：每種 kind 至少一點、`diff_pixels>0` 的全部、第 10 回合 event 25
  那幾點、戰後 JOIN。每點一列三格「原版 checkpoint PNG｜重製 remake-NNNN-pK.png｜差異遮罩」，
  格上標 seq、kind、diff_pixels、兩側 sha256 前 8 碼；index JSON 記每一列對到收據哪一點
  （seq、兩側檔名、sha），讓人能從圖回查收據。原生 320×200 不縮放，整張控制在 1.5 MB 內。
- #37 的淡出抽幀另出一張，標「輔助基準、非 gate」。
- 57 的第七章段落貼圖與 index 連結；56 §第七章記工具用法。
- 順手用同一支工具回補 ch04／ch05／ch06（run 都還在 `work/`），三張一起進 `docs/figures/`，
  57 各補一行——回顧性證據，不改收據。

## 7. 收尾

任一 gate 失敗：開 缺陷／RE待解 issue（`tools/fd2_worklist_issues.py new`），修掉重跑；
修不掉標 blocked 寫原因。全過才台帳第七章 passed（`tools/fd2_parity_progress.py verify`），
56 加第七章段、57／58 各一段、91-history 一則、教訓寫成規則（`docs/data/fd2-lessons.json`），
`fd2_worklist_issues.py pull` → `fd2_worklist.py verify` → render → commit。視時間 #32、#29。

## 已知規則（沿用＋第六章新增）

- 0x1A30B 順序：我方回復 → selector 1 事件 → 友軍 AI → 橫幅 → 0x13536 清 bit7 →
  selector 0 事件 → 敵軍兩遍；selector 2 事件在 PLAYER PHASE 橫幅後、輸入前。
- 0x205b4 勝負只看已登場記錄；0x11506 照抄 +2/+3；存檔差 byte 用 fd2save.py 對回欄位；
  一章收據裡剛好相等的量不是規則。
- oracle_mid_end_turn 不比；checkpoint 落在 0x373C4／0x11EB0／0x4E809 內 dosgolem 已延後 PNG；
  idle 變體被 0x1297D 推進就重設再畫；敵方回合鏡頭是 0x12CEA 游標協定。
- MAP_LIMIT 63、stop_on_auto_end、清場後 await_ui；force_enemy_clear 只能在抽完戰鬥節拍之後。
- 第六章新增：佈陣格跟槽位不跟角色；HUD anchor 只在 0x1AD2A 真的畫時前進（閘 A／B、游標
  重繪條件）；ACTING 未登場但在 96 筆內是 no-op；待機前先重播游標鍵；battle_start 的
  native_map_view 抄 battle_start；sub_1A866 三個 selector 的暫時狀態掃描每回合都跑
  （+0x22 歸零強化才會再放）；指令演出擲骰與命中／傷害同一條 0x4E893 序列、未命中不抖動；
  重播在每個 0x13A9F 入口對齊亂數，ai_order 分岔要 0。

## 紀律

- 改過 Go 檔最後再跑 `tools/rebind_string_review.sh` → 複製 inv-new.json 成
  `docs/data/fd2-string-inventory.json`（gitignore，不進版控）→ 在 Docker 內
  `go run . -repo ../../.. -summary -output docs/data/fd2-string-inventory-summary.json` →
  `tools/migrate_full_locale_content.py`（新字串 `--new-translations`，machine_draft）→
  `tools/validate_full_locale_content.py`、`tools/validate_locale_packs.py
  remake/assets/locales/*/pack.json`、`tools/test_locale_review_blockers.py`。
- gofmt／go vet 在 Docker 跑；全套件 `tools/remake_go_test.sh <素材根> <log> ./...`；
  `tools/test_dosgolem_oracle_drive.py` 有 3 個既有 `/out` 權限錯誤（HEAD 也一樣）不算新紅。
- 提交前分開看每項檢查再 commit；push 前問一次；每批 Docker 後 `docker ps -a` 檢查 FD2 容器，
  禁止任何 prune／rmi；素材包有新增就同步 `~/cht/fd2-assets-private` 並跑
  `validate_separated_asset_pack.py`；不改 #15／#9／#6／#7／#8／#4／#10／#11。
