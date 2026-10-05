# 炎龍騎士團2 逆向工程知識庫 — 索引(問題導向路由)

> 《炎龍騎士團2》(Flame Dragon Knight 2),漢堂國際 1995,DOS / DOS4GW 保護模式。
> 由逆向工程逐輪累積。本檔只負責把問題路由到主證據；現況與逐輪反思分別寫入
> `58` 覆蓋矩陣、`91` 工作佇列與 `99` 歷史反思，不再把同一結論複製到每份文件。
>
> **用法**:先看下面「§A 問題 → 查哪份」路由表定位;§B 完整文件清單;§C 機器可讀資料(別忘了用);§D 還原 chNN.json 工作流。
>
> **🎯 目標 + [HARD] 鐵則**:用 Go/Ebiten + RE 原版 DOS，逐步收斂到可驗證的 remake parity；
> 「目標是一模一樣」不是目前完成宣稱，所有未閉合部分都必須保持 fail-closed。
> **禁止用推測/外推寫 code**——每個進 code 的值(座標/幀數/鏡頭/時機/射程/回合)必須有 RE 來源
> (反組譯 doc47/50、dosbox doc48、青衫、影片、FDFIELD 直讀)。不知道就先 RE 拿真值,拿不到就誠實停,不准猜。
> (BeatRunner 外推 pan 值→越改越偏的教訓;驗收對 reference 實測非「測試綠」,規則 65。)

> **進度入口**：整體覆蓋與下一個缺欄位先看
> [`58-fd2-exe-re-coverage.md`](58-fd2-exe-re-coverage.md)，再依問題讀 `56` SDD、
> `57` UI evidence matrix 與 `91` worklist。根目錄 [`README.md`](../../README.md)
> 是對外摘要；`SESSION-HANDOFF-*` 是歷史 provenance，不是目前工作指令。
> 本索引與其他專題文件是證據路由，不是「已完成全遊戲」清單。

> **文件裁決順序（2026-08-27）**：整體 RE／資料／執行期／E2 分層狀態看 `58`；
> raw ABI 與系統契約看 `56`；UI 證據看 `57`；有效下一步看 `91`。沒有獨有證據、
> 已被這些入口完全取代的早期規劃快照已刪除，原文仍可由 Git 歷史回查。`51`、
> `99` 與 `SESSION-HANDOFF-*` 因保有玩家實測或錯誤形成脈絡而保留；其中的
> 「下一輪」「全章可玩」「已完成」等歷史字樣不覆蓋現況。

第12章後段地形追蹤使用 [六回合有界輸入](../data/parity-plans/ch12-terrain-late-probe.jsonl)。它逐字保留 ch12-sample 前50行，停止於第六回合抽樣；原版使用[官方容器入口](../../tools/dosgolem_oracle_container.sh)。追蹤歸入 #52 與 [mode3 主證據](../data/ida/fd2_terrain_mode3_review_20261001.json)，尚未證實的寫入來源不接正式執行期。

### 現況文件與歷史文件

| 需求 | 唯一入口 | 不可取代它的文件 |
|---|---|---|
| 判斷是否還要 RE、缺實作還是缺 E2 | [`58` 覆蓋矩陣](58-fd2-exe-re-coverage.md) | handoff、raw exporter 的 `unknown` 統計 |
| 系統架構、ABI 與證據 gate | [`56` SDD](56-fd2-remake-sdd.md) | 舊 WBS、聊天摘要 |
| 玩家可見 UI 與畫面差距 | [`57` UI 矩陣](57-ui-evidence-matrix.md) | README 圖片、單一 screenshot |
| 下一批可執行工作 | [`fd2-worklist.json`](../data/fd2-worklist.json)，渲染在 [`91`](91-worklist.md) | [`91-worklist-history.md`](91-worklist-history.md) 的舊勾選與規劃快照 |
| 全戰役原版一致目標：門檻、章工作單元、開工順序 | [`111` 目標提示詞](../goal/111-goal-original-parity-campaign-20260915.md)，進度台帳 `parity-campaign-progress.json` | 逐章收據自己的散文摘要 |
| 目前這一章的工作單元（順序、門檻、交付物） | [`117` 寶箱／HUD 修正（#44／#43）＋第十二章（北山道）提示詞](../goal/117-goal-chest-hud-fixes-and-ch12-parity-20260918.md)（做完就換下一章的；目標文件都在 [`docs/goal/`](../goal/README.md)） | 聊天裡貼的舊版提示詞 |
| 位址／位元組主證據 | `docs/data/ida/`、`docs/data/fd2_*` | 自訂名稱、handoff 重述、generated binding |
| 歷史錯誤形成與勘誤 | [`SESSION-HANDOFF`](SESSION-HANDOFF-2026-07-06.md)、[`99`](99-reflections-log.md) | 現況矩陣 |

---

## §A 問題 → 查哪份文件(路由表)

### 過場 / 開場動畫 / 劇情演出(近期主線)
| 我想知道… | 查 |
|---|---|
| 第一章**開場**逐幕時間軸(王座廳→草地→密林→行軍→海島)、remake 差異 | **`46`**(影片觀測時間軸；畫面／順序 oracle，不取代 handler E0) |
| 開場 handler `0x3231b` **完整指令序列**(每 beat 的 call+參數+語意) | **`47`**(§3/§7 逐 beat 全轉錄) |
| START→開場→第一關第一回合**逐項 RE 來源對照**(禁推測驗收表:哪些✅可寫/⚠須換/❓待RE) | **`53`** |
| 過場**原語**(pan/走位/對白/演出/spawn/入隊/step家族)怎麼運作、位址 | **`50`**(過場機制唯一主檔;原始逐beat轉錄見`47`) |
| 「兩套腳本系統」——開場 cutscene vs 戰鬥中事件對話,界線在哪 | **`52`** §0(**先讀這個再碰過場**) |
| 某關**玩家可見事件候選**(第幾回合/增援/加入/勝敗) | **青衫攻略 `references/text/fd2-walkthrough-index.md`**(E3 authored reference)+ 全文 `references/text/fd2.md`；忠實 chNN 仍須 handler/FDFIELD/DOSBox 證據 |
| 某句對白**第幾回合/什麼事件**觸發(哈諾/海盜頭目/海防隊) | 青衫索引(時機)+ **`ch01.json` events**(Fable5 RE 範本)+ `26` + `battle_events.json` + `52` §1.2 |
| 草地幕走位逐幀量測(原始數據) | `55`(機制見 **`50` §1.1**) |
| 單位倒下時的死亡效果（掉落、金錢、全域事件、死亡台詞）、狀態致死無分派，以及升級上限、30 級歸零 | **`110`**（轉寫 `remake/assets/data/native_death_events.json`；狀態 phase 主證據 `fd2_status_death_ida.txt`） |
| 重製端從標題 START 走到第一關後的城鎮,逐項對原版證據(序章對白、第 1 回合單位、戰場事件、戰後、進城) | **`109`**(收據 `docs/data/ui-traces/title-to-town-ch01-e1.json`) |
| 索爾四人**怎麼進戰場**(進場動畫/站位) | `52` §1.1 + `46` §4(⚠ 進場動畫細節待 dosbox 定稿) |
| acting 機制(正常 frame 逐格移動；特殊 frame 原地姿態) | **`50` §1.2**(`54` 僅存 dosbox 實測原始記錄) |
| 「走位」機制(step家族4方向+路徑走位0x13488)、單位欄位 +0/+1/+3/+4、面向規則 | **`50` §1.1** |
| remake 過場**引擎**(BeatRunner / cutscene 節點 / beats DSL)怎麼設計 | **`50`**(§2 DSL,§3 全33關管線) |
| **某一幕的原始資料×解讀**(handler beat 反組譯 + acting hex+解碼 + roster + campaign 對映 + 可疑點,供人工覆核) | `scene-decode/ch1-throne.md`(皇宮)、`scene-decode/ch1-meadow.md`(草地);每幕一份 |
| 全 33 關過場 beats(機器可讀) | `docs/data/chapter_beats/chNN_{pre,post}.json` |
| 開機/標題/主選單/劇情自動過場流程(反組譯) | `23` + `39`(ANI.DAT AFM 開場)；`sub_1F894` 捲動／AFM 交錯的 canonical IDA 主證據見 [`fd2_title_scroll_schedule_ida.txt`](../data/ida/fd2_title_scroll_schedule_ida.txt) |

### 角色 / 單位 / 數值
| 我想知道… | 查 |
|---|---|
| portrait/char id → **角色名** | **`49`** + `docs/data/portrait_names.json`(證據分級) |
| 說話者 id 兩種定址(-17/-18 全域 vs -19/-20 場景) | `40` |
| 職業名顯示錯位(海盜→劍士 bug) | `45` |
| **武器/攻擊範圍/物品數值**(靜態表,不需 debugger) | `32` + `02` + `03`(青衫+反組譯) |
| **法術**(id→特效、效果、面板) | `37` + `02` + `13`(Get_EasyMagic) |
| 戰鬥公式(命中/暴擊/傷害/**成長**) | `02` §4 + `27` + `internal/battle/growth.go` |
| 敵/NPC **AI** 決策 | `11`（`0x13A9F/0x14EF0/0x15B77` raw boundaries；完整 runtime 仍 fail-closed） |
| 全 30 關**目標/勝敗/加入條件** | `28` + `docs/data/battle_events.json` |
| 逐關戰鬥事件 handler 細節 | `25`(機制)+ `26`(逐關)+ `battle_events.json` |
| 地圖單位 sprite(FDICON Q版小人/待機動畫) | `31` |

### 資產格式(RE 完成度高)
| 我想知道… | 查 |
|---|---|
| `.DAT` 容器 / 圖像 / 調色盤 / 地形格式 | `01` |
| 圖像 RLE 壓縮 | `05` |
| 動畫(FIGANI/AFM)格式 | `06` + `39`(ANI.DAT) |
| 全螢幕戰鬥演出繪圖 | `35` |
| 文本 / 自製字型 / 控制碼 | `08` + `09` + `14` |
| 音樂 XMIDI / 播放換曲 / 音色(SoundFont/MT-32) | `07` + `12` + `16` |
| 音效 SFX 資料 | `36` + `docs/data/battle_sfx_map.json` |
| EXE 資料表 offset / 核心結構 | `03` |

### remake(Go/Ebiten)
| 我想知道… | 查 |
|---|---|
| 重製架構／建置／驗證入口 | [`docs/ENGINEERING.md`](../ENGINEERING.md) + `56` + `41` |
| 字型現代化(UTF-8/TTF) | `18` |
| 劇本/事件系統設計(節點圖/可擴展 DSL) | `19` + `29` |
| **試玩落差清單**（結束回合／武器射程／法術／狀態欄／對話框） | **`51`**（玩家實測快照）+ `44` + `57`（現況） |
| 打包(AppImage/Win/macOS) | `41` |
| 編輯器設計 | `38` |
| 目前可交付範圍與限制 | [`REMAKE-STATUS.md`](../REMAKE-STATUS.md) + `58` |

### 工具 / 方法
| 我想做… | 查 |
|---|---|
| **dosbox-x debugger**(建置/BP trace/dump/BPLM 判死) | **`48`** |
| 原版 dosgolem 對拍、直接 Docker 入口與出處收據 | [96 對拍工具鏈](96-parity-toolchain-20260909.md)，入口 [dosgolem_oracle_container.sh](../../tools/dosgolem_oracle_container.sh) |
| Call-graph 反組譯方法紀錄 | `24` |
| Watcom `push N; call helper` stack check／probe/runtime 辨識 | [`59`](59-watcom-stack-runtime-patterns.md) |
| 原作 compiler／linker／DOS extender／Miles AIL／AFM 工具鏈指紋 | [`04`](04-original-toolchain.md)、[`0x3EEDA` AIL 證據](../data/ida/fd2_ail_background_3eeda_ida.txt) |
| 當年開發工具考證 | `04` |
| 「1995 年怎麼做這遊戲」總覽 | `15` |

### 專案管理
| | 查 |
|---|---|
| 整體 RE／資料／執行期／E2 覆蓋與「是否重做」 | **`58`**（唯一現況矩陣） |
| 這輪做什麼 / 待辦 | `91`(worklist) |
| 逐輪反思 / 踩雷 | `99`(reflections) |
| 工程入口／有效計畫 | [`docs/ENGINEERING.md`](../ENGINEERING.md) + `91` 檔首 |

---

## §B 完整文件清單(依編號)

`01`容器/資產 · `02`遊戲數值(青衫) · `03`EXE表/結構 · `04`開發工具考證 · `05`圖像RLE · `06`動畫AFM ·
`07`XMIDI · `08`文本/字型 · `09`劇情/對話 · `10`sprite著色/狀態 · `11`AI · `12`音樂播放/場景 ·
`13`戰場選單 · `14`文本控制碼 · `15`1995怎麼做(總覽) · `16`音色合成 · `17`擴充可行性 · `18`字型現代化 ·
`19`劇本系統設計 · `23`開機/標題/過場流程 ·
`24`callgraph紀錄 · `25`戰場事件系統 · `26`逐關事件handler · `27`戰鬥規則+驗證清單 · `28`全30關目標 ·
`29`可擴展事件系統 · `31`FDICON地圖sprite · `32`物品/戰鬥數值 · `35`戰鬥演出繪圖 ·
`36`SFX · `37`法術特效對映 · `38`編輯器設計 · `39`ANI.DAT AFM · `40`說話者→頭像查表 · `41`打包 ·
`44`第一章對照 · `45`職業名錯位 · `46`第一章開場時間軸 · `47`序章handler全轉錄 ·
`48`dosbox-x debugger · `49`char id→角色名 · `50`**過場機制總表(唯一主檔)** · `51`試玩落差R2 · `52`戰場分鏡+兩套系統 · `53`START→ch1回合1 RE來源表 · `54`acting實測原始記錄(機制見`50`) · `55`草地走位量測 ·
`56` FD2 remake SDD（UI／campaign／證據 gate） · `57` UI evidence matrix · `58` FD2.EXE RE／remake 覆蓋矩陣 · `91`worklist（現況；歷史在 `91-worklist-history`） · `99`反思

`100`對話視窗 · `101`回合橫幅（呈現） · `102`戰果訊息 · `103`戰鬥台座 z 序 · `104`回歸基線審查 · `105`回合橫幅（節奏、字樣與版面） · `106`物理攻擊：結算鏈、反擊條件與傷害公式 · `107`六個暫時狀態的語意 · `108`地形移動成本表與移動確認游標 · `109`標題 START 到羅德鎮的整段對照 · `110`死亡效果與升級上限

（缺號 20／21／22／30／42／90 是 2026-08-27 刪除的早期規劃／落差快照；
其內容已被現行入口取代且沒有獨有原始證據，原文仍保留於 Git 歷史。缺號
33／34／43 則是曾用後併入他篇或未建。）

---

## §C 機器可讀資料 + 本機 dump(別忘了用!)

**入庫(`docs/data/`,可公開整理)**:
- `chapter_beats/chNN_{pre,post}.json`(+`_stats.json`)— 全 33 關過場 beats(系統 A),`50` 產出
- `battle_events.json` — 全 30 關戰鬥事件(系統 B),`26` 產出
- `portrait_names.json` — char id→角色名(證據分級),`49`
- `turn_events.json` / `event_id_groups.json` / `shops.json` / `battle_sfx_map.json` — 事件/商店/音效
- `glyph_map.json` / `unicode_to_glyph.json` — 字型對照
- `exe_tables/` — EXE dump 出的數值表
- `campaign_sample.json` — 節點圖範例

**本機 dump(`extracted/`,gitignore,版權物,不上 GitHub)**:
- **舊 `acting_decoded_throne.txt`** — 其 `0x207718`／高 ID 74 筆／id−48 結論已確認為錯 context dump，
  僅保留考古用途，不能供 remake 使用。正確來源是 EXE 106-entry direct-ID bank，可由
  `tools/export_acting_resources.py` 決定性重建（詳見 `47`、`48`、`50`）。
- `dosbox_dump/out/*.bin` — 單位陣列槽 dump、acting 資源原始 bytes、鏡頭/單位數快照(`47`/`48` 實測證據)
- `extracted/maps/` `extracted/images/` `extracted/story/` 等 — 解出的地圖/圖/劇情文本(玩家自備原版跑 tools 解)

---

## §D 還原 chNN.json 的工作流(核心目標)

remake 每關的劇本檔 `remake/assets/scenarios/chNN.json` = **事件骨架 + 對白文字**兩者合成:

1. **事件候選骨架**(玩家觀測到何時發生什麼)← **青衫攻略**
   (`references/text/fd2-walkthrough-index.md`,E3 authored reference)+
   `battle_events.json`(只保存部分勝敗 handler metadata，不含完整動作／postbattle)；
   兩者都不能單獨解除逐章 evidence gate。
2. **對白文字**← FDTXT 轉錄(`extracted/story/`,全 1533 句)。
3. **資料結構示例**:`ch01.json` 可示範現有 events/trigger/when/do 與 dialogue
   schema，但不是其餘章的語意 oracle；ch02~30 必須各自轉錄 pre/battle/post
   handler、FDFIELD、town/preparation 與 persistence 邊界。系統 A(開場過場)
   進 cutscene 節點；系統 B(戰鬥中事件)進 scenario events(doc52)。

## 標註慣例
- **[已驗證]** 原版實檔/反組譯/dosbox 交叉確認 · **[假設]** 待後輪確認/推翻 · **[攻略]** 青衫玩家觀測(實作以反組譯為準)

## 原始素材(不入 git,不散布)
- 遊戲本體 `org_game/炎龍騎士團/FLAME2/` · 攻略鏡像 `references/`(E3 authored reference) · 原版錄影 `video/`(E2/E3 visual oracle，依捕捉 provenance 分級)


## 2026-10-01 JOIN 戰後尾格有限證據入口

[58 分層現況](58-fd2-exe-re-coverage.md) →
[IDA／FDFIELD寫入與複製證據](../data/ida/fd2_join_copyback_20261001.json)、
[九項驗收收據](../data/fd2_join_copyback_verification_20261001.json)及
[原版受版控控制計畫](../data/parity-plans/ch02-join-copyback.jsonl)。
工具、規格、正式與建槽回歸入口集中58；既有JOIN殘值歷史收據保留。


#53 取寶提示底圖：[58蒐證入口](58-fd2-exe-re-coverage.md) →
[開框VGA與行動caller](../data/ida/fd2_treasure_background_20261001.json)；
原版控制計畫沿用 ch12-sample.jsonl，規格狀態與限制由56／58承載。


#52 地圖tile27殘差定位勘誤：[58入口](58-fd2-exe-re-coverage.md) →
[17點座標與既有raw／LUT指令契約](../data/ida/fd2_terrain_mode3_review_20261001.json)。
[正常操作前綴唯讀探針](../data/parity-plans/ch12-terrain-mode3-probe.jsonl)。
[正常 LOAD 的工作緩衝交接診斷](../../remake/cmd/fd2/native_terrain_work_handoff_test.go)。
原版底色寫入來源仍未知，不重開已閉合的解碼器。

第十三章正常LOAD：[58入口](58-fd2-exe-re-coverage.md) → [有限收據](../data/parity-slots/ch13-load-validation.json)；
manifest、控制計畫與slot-ready現況統一由[戰役台帳](../data/parity-campaign-progress.json)連結。

#58 第十三章原生交接與#59完整函式chunks：[58入口](58-fd2-exe-re-coverage.md) →
[IDA與59筆原版起手對照](../data/ida/fd2_ch13_handoff_20261001.json)。

第十三章診斷收據：[58目前阻塞](58-fd2-exe-re-coverage.md) →
[有界預檢，整章未通過](../data/ui-traces/parity-ch13-preflight.json)；#60／#61分別追畫面與結果判定。

#60 色盤表與對拍相位：[58入口](58-fd2-exe-re-coverage.md) →
[IDA原始93-byte表及consumer](../data/ida/fd2_palette_cycle_table_20261001.json)；規格見56末節。


#61 第十三章結果條件與 #62 匯出器勘誤：[58入口](58-fd2-exe-re-coverage.md) →
[IDA直接分支、完整chunks與原版raw狀態](../data/ida/fd2_ch13_result_conditions_20261001.json)。
正式接線與敗北／戰後對拍狀態仍由58及#61承載。

#62來源工具回歸：`tools/test_event_handler_dump.py` 與
`tools/test_extract_event_id_groups.py`；固定原版與主證據、執行命令見56／58。

#61有限敗北重生：[58現況](58-fd2-exe-re-coverage.md) →
[第十三章正常輸入控制計畫](../data/parity-plans/ch13-defeat.jsonl)。

#61逐條原生結果：[56 READY契約](56-fd2-remake-sdd.md) →
`remake/internal/battle/native_result.go`、`remake/cmd/fd2/native_chapter_result.go`；
對應測試native_result_test.go／native_chapter_result_test.go，有限敗北收據仍由58索引。
- 第十三章有限敗北同槽收據：[ch13-defeat-return-title.json](../data/ui-traces/ch13-defeat-return-title.json)，依原版固定呼叫錨點比較兩張提示，保存亂數控制與存檔限制；不代表整章通關。
- 第十三章保護友軍與第九回合抽樣計畫：[ch13-protect-sample.jsonl](../data/parity-plans/ch13-protect-sample.jsonl)，保持114建構槽、seed4與正常章內操作；完整驗收狀態見58與#61。
- 第十三章東側護援計畫：[ch13-east-guard-sample.jsonl](../data/parity-plans/ch13-east-guard-sample.jsonl)，以正常鍵盤提早派 slot3／14 向東；同槽、亂數與驗收限制維持 #61，結果由58記錄。
- 第十三章戰後秘密商店 #67：[合法酒店存檔重載計畫](../data/parity-plans/ch13-secret-shop-reload.jsonl)，只驗證第十四章城鎮 selection2／Shift+F3；結果與完整章限制由58及第十三章結果條件證據保存。
- 第十三章整章 r3 [四項診斷](../data/ui-traces/parity-ch13-r3.json)：行為、交易與258張畫面通過，秘密商店錯誤按鍵使節點驗收失敗；[秘密商店有界重載收據](../data/ui-traces/ch13-secret-shop-reload.json)驗證修正後的輸入，兩份收據不拼接成整章完成。
- 第十三章修正後完整計畫的[整章收據](../data/ui-traces/parity-ch13.json)由同一建構槽與完整原版／重製重播產生；驗收狀態及111／114限制統一見58與戰役台帳。
- 原版掃描末名自動換手缺陷 #65：[有界回合計畫](../data/parity-plans/ch13-auto-end-boundary.jsonl)與[驗證收據](../data/fd2_oracle_sweep_auto_end_20261002.json)，只驗證新回合不被多送 END，完整章現況見58。
- 原版麻痺單位選取缺陷 #66：[原始選取判準與工具回歸](../data/fd2_oracle_paralyzed_selection_20261002.json)，保存第八回合角色狀態面板與 raw 閘門；完整章仍由 #61 驗收。
- 戰後測試入口勘誤 #64：[八條測試與分支驗證](../data/fd2_post_fixture_verification_20261001.json)，保存正確素材、前置狀態與實際非略過結果；證據範圍見58。

- 第十四章 #69：[建構槽清冊](../data/parity-slots/ch14-manifest.json)與[正常LOAD有界預檢計畫](../data/parity-plans/ch14-preflight.jsonl)；固定政策依111／114，整章現況由58與戰役台帳承載。
- 第十四章有界預檢[診斷收據](../data/ui-traces/parity-ch14-preflight.json)：原版已進場，重製起手HUD缺少輸入；整章尚未通過，後續修正與驗收由#69追蹤。
- 第十四章起手缺陷 #70：[IDA章別入口與同槽原版狀態](../data/ida/fd2_ch14_startup_20261002.json)，共用constructor／HUD沿用已閉合證據；規格與分層現況見56／58。
- 第十四章 #69 的[整章抽樣計畫](../data/parity-plans/ch14-sample.jsonl)：三回合正常抽樣後依111清敵，戰後完整節點驗收；正式收據尚未產生，現況見58。
- 第十四章 #70 [同槽原生起手收據](../data/ui-traces/ch14-native-startup.json)：三張完整畫面0px，僅有限RUNTIME-E1，整章仍由#69驗收；原始失敗預檢收據保留。
- 第十四章整章[首次失敗診斷](../data/ui-traces/parity-ch14-r1.json)：原版完整退出，重製在事件10停止並有AI順序差異；正式完成範圍仍由58及#69承載。
- 第十四章事件10 #71：[IDA完整owner／範圍consumer與原版live模式轉移](../data/ida/fd2_ch14_event10_20261002.json)；超過live前沿的處理仍待證據，規格DRAFT，AI較早分岔由#72獨立處理。

- 第十四章AI兩遍差異 #72：[mode8跳過成功收尾的IDA與原版trace](../data/ida/fd2_ch14_mode8_dispatch_20261002.json)；有限READY規格與重開原因見56／58。

- 第十四章戰後排列缺陷 #75：[既有 IDA 主證據追加16筆原表](../data/ida/fd2_ch12_post_persistence_20261001.json)，原始定位與writer保留；正式規格、重開原因與驗收由56／58承載。

- 第十四章修正後[完整四項收據](../data/ui-traces/parity-ch14.json)，保持同槽、固定亂數與111／114限制；完整現況統一見58。

- 第十四章[原版／重製／差異固定抽樣索引](../data/ui-traces/parity-ch14-samples.json)與[總覽一](../figures/parity-ch14-samples-p1.png)、[總覽二](../figures/parity-ch14-samples-p2.png)，按每種動作首點與所有非零差異抽樣，不另提升驗證範圍。

- 第十五章 #76：[建構槽清冊](../data/parity-slots/ch15-manifest.json)與[正常LOAD有界預檢計畫](../data/parity-plans/ch15-preflight.jsonl)，僅起手核對，不宣稱整章完成；現況仍由58與台帳承載。

- 第十五章 #77：[起手原版證據與READY規格](../data/ida/fd2_ch15_startup_20261002.json)，只核對74筆名冊、16格部署與視圖／HUD，整章由 #76 驗收。

- 第十五章 #77：[正常LOAD起手比較收據](../data/ui-traces/ch15-native-startup.json)，三張完整RGB影像差異均為0；整章12張與戰後／存檔門檻仍保留。

- 第十五章 #78：[回合事件13／18／38原始指令與CONFORMED規格](../data/ida/fd2_ch15_turn_events_20261002.json)，沿用全域事件表與既有record_bytes，不以spawn-only代表完整handler。

- 第十五章 #76：[原首次完整計畫（歷史敗北樣本）](../data/parity-plans/ch15-sample.jsonl)，正常鍵盤涵蓋第4／7／9回合後round10清敵，戰後town_ch16買賣／酒店／Alt+F5秘密商店。

- [第十五章 record64 無對白敗北（#79 有限 CONFORMED）](../data/ida/fd2_ch15_result_conditions_20261002.json)：固定雜湊、完整20822原始指令、結果writer／consumer、同槽正常回放至完整標題選單、存檔不變與重生命令；整章成功路徑依正式parity-ch15收據驗收。
- [第十五章正常護援控制計畫（#76／#78，完整四項通過）](../data/parity-plans/ch15-guard.jsonl)：相同槽與亂數政策，記錄2／10先推進護援、保留第4／7／9回合、round10才清敵；原失敗計畫與收據保留。
- [原版預算重跑前綴核對工具](../../tools/verify_oracle_prefix.py)與[拒收測試](../../tools/test_verify_oracle_prefix.py)：比較完整CPU／原始記錄、亂數及控制序列，只排除PNG背景輸出排程；配合章驗證器確認提高預算未變更既有結果。

- [第十五章完整首次失敗診斷（#76／#81）](../data/ui-traces/parity-ch15-r1.json)：原版第六回合敗北、AI首個分岔及61張嚴格整幀比較，四項皆未通過。

- [第十五章離屏狀態到期前置修正（#80）](../data/fd2_ch15_transient_replay_20261002.json)：原失敗checkpoint、正式Draw來源、到期提示測試與19套件回歸；整章仍failed。

- [第十五章 AI 首個分岔診斷計畫](../data/parity-plans/ch15-ai-prefix.jsonl)：GitHub #81，沿整章相同正常輸入只到第五回合；不作整章收據。

- [第十五章敵方承接共享經驗與成長槽](../data/ida/fd2_ch15_ai_growth_20261002.json)：#81，RE-CLOSED／READY；[256槽資料](../data/exe_tables/native_growth_slots.json)與[重生工具](../../tools/extract_native_growth_slots.py)。68列後是相鄰讀取，不是作者表。

- [第十五章成長修正後嚴格診斷（#81／#82）](../data/ui-traces/parity-ch15-r2.json)：297筆AI無分岔、66個行為點通過；seq849色盤與完整章仍未通過，分層現況見58。

- [第十五章色盤寫入中間狀態（#82）](../data/ida/fd2_ch15_palette_writer_20261002.json)：原版EIP／暫存器直接證實phase5→6的四槽進度，有限對拍工具規格READY；現況見58。

- [第十五章AI與嚴格畫面修正後診斷（#81／#82）](../data/ui-traces/parity-ch15-r3.json)：61張畫面與66個行為點通過，完整章節點／交易仍未完成；最新分層由58承載。

- [第十五章完整對拍正式收據（#76／#78）](../data/ui-traces/parity-ch15.json)：第4／7／9回合、416筆AI、戰後城鎮、交易與酒店SAV；依111／114例外列PLAYER-E2，舊失敗與140億預算前綴均可回查。

- [第十五章固定抽樣對照總覽索引](../data/ui-traces/parity-ch15-samples.json)：每種畫面第一點、所有非零差異與回合事件後指定點，完整原版／重製／差異三欄，可回查正式收據與PNG雜湊。

- 第十六章 #83：[建構槽清冊](../data/parity-slots/ch16-manifest.json)與[正常LOAD有界起手預檢](../data/parity-plans/ch16-preflight.jsonl)，只驗名冊／部署／視圖；完整章四項與戰後SAV見正式parity-ch16收據。

- 第十六章 #84：[17人名冊正常選人預檢計畫](../data/parity-plans/ch16-preflight-select.jsonl)，15位可選隊員與固定隊長；原首次計畫停在選人，舊輸入與收據保留，不列產品缺陷。

- [章對拍選人工具規格（#84）](../data/fd2_parity_preparation_selection_20261002.json)：既有IDA固定隊長契約、17人槽正常按鍵與有界起手收據；有限CONFORMED／RUNTIME-E1，完整章已由#83正式收據驗收。

- [第十六章原生起手規格（#85）](../data/ida/fd2_ch16_startup_20261002.json)：335A0跳躍尾段、既有constructor與76筆同槽起手，有限CONFORMED／RUNTIME-E1。
- [城鎮整備caller背景（#86）](../data/ida/fd2_town_preparation_background_20261002.json)：完整原始保存／還原與確認consumer，有限CONFORMED／RUNTIME-E1；[原失敗診斷](../data/ui-traces/parity-ch16-preflight.json)保留。

- [第十六章有限原生起手收據（#84／#85／#86）](../data/ui-traces/ch16-native-startup.json)：五個完整RGB點0px、16人原序與76筆起手；僅RUNTIME-E1，完整章12幀及戰後SAV由#83正式收據另驗。

- [第十六章有限起手總覽索引](../data/ui-traces/ch16-native-startup-samples.json)與[原版／重製／差異總覽](../figures/ch16-native-startup-samples.png)：固定抽樣五個kind首點，僅有限起手RUNTIME-E1，完整章另見#83正式收據。

- [第十六章完整抽樣計畫（#83）](../data/parity-plans/ch16-sample.jsonl)：正常選人、三回合抽樣、第4回合清敵、戰後交易／酒店SAV及城鎮17秘密商店。計畫本身不代表通過，驗收結果見正式parity-ch16收據。

- [第十六章完整首次診斷（#83）](../data/ui-traces/parity-ch16-r1.json)：行為／節點／76張完整RGB通過，酒店SAV全檔不同；整章保持未完成。

- [第十六章未出戰持續槽（#87）](../data/ida/fd2_ch16_unselected_persistence_20261002.json)：沿已閉合11506的raw身分匹配，五欄缺呈現出處的consumer重開及有限CONFORMED規格；原失敗收據保留。

- [第十六章完整對拍收據（#83／#87）](../data/ui-traces/parity-ch16.json)：同槽三回合、戰後18人、買賣／酒店SAV／秘密商店四項通過；依111／114例外列PLAYER-E2，完整Go已通過。

- [第十六章固定抽樣總覽索引](../data/ui-traces/parity-ch16-samples.json)、[總覽一](../figures/parity-ch16-samples-p1.png)與[總覽二](../figures/parity-ch16-samples-p2.png)：每種畫面首點及所有非零差異，完整原版／重製／差異三欄。

- 第十七章 #88：[建構槽清冊](../data/parity-slots/ch17-manifest.json)與[正常選人起手預檢計畫](../data/parity-plans/ch17-preflight-select.jsonl)。目前為待驗輸入，不代表完整章對拍通過。

- [第十七章有限原生起手規格（#89）](../data/ida/fd2_ch17_startup_20261002.json)：sub_335AA、shared text／focus原始指令及53筆同槽起手；完整章仍由#88驗收。

- [第十七章有限起手收據（#89）](../data/ui-traces/ch17-native-startup.json)：五點完整RGB0px、53筆起手；僅RUNTIME-E1，完整章仍#88。另保留[缺HUD首次診斷](../data/ui-traces/parity-ch17-preflight.json)。

- [第十七章有限起手總覽索引](../data/ui-traces/ch17-native-startup-samples.json)與[完整三欄總覽](../figures/ch17-native-startup-samples.png)：五點RUNTIME-E1，無完整章或存檔宣稱。

- [第十七章完整抽樣計畫（#88）](../data/parity-plans/ch17-sample.jsonl)：正常選人、四回合至round5、turn4增援後清敵、戰後交易／酒店SAV與Ctrl+F7。計畫本身不代表整章通過。

- [第十七章原版CPU缺口診斷與修復（#90）](../data/ida/fd2_ch17_oracle_d1_20261002.json)：IDA LE 15C59 bytes d1642408、原版D1退出保留；CPU有限CONFORMED，升級後118點起手及592點失敗前綴全同。原版正常敗北，完整章由#88護援續驗。
- [第十七章正常護援計畫（#88）](../data/parity-plans/ch17-guard.jsonl)：NPC52後撤，南側記錄0／2正常移動掩護；四回合與turn4事件保留，round5才依111清敵，完整章仍待驗。
- [第十七章FIGANI畫布界線consumer（#92）](../data/ida/fd2_ch17_figani_bounds_20261002.json)：FANI51 frame0原始header與640-stride右半畫布最後span；有限CONFORMED，修正合法末列誤拒收，不重新解原版prelude或handler。

- [第十七章調色盤下一筆寫入前停點](../data/ida/fd2_ch17_palette_writer_20261002.json)：#93，有限CONFORMED，只擴充已證實完整triplet窗口的對拍候選。

- [第十七章完整同槽章收據](../data/ui-traces/parity-ch17.json)：#88，98張RGB／酒店SAV四項通過，依111／114例外列PLAYER-E2。
- 第十七章原始診斷：[FIGANI拒收r1](../data/ui-traces/parity-ch17-figani-r1.json)、[writer未承接r3](../data/ui-traces/parity-ch17-writer-r3.json)，保留失敗形成原因；由正式收據取代。
- [第十七章固定抽樣總覽索引](../data/ui-traces/parity-ch17-samples.json)：每種畫面首點及所有非零差異，包含seq1287原始writer停點；[總覽一](../figures/parity-ch17-samples-p1.png)與[總覽二](../figures/parity-ch17-samples-p2.png)。

- 第十八章 #94：[建構槽清冊](../data/parity-slots/ch18-manifest.json)與[正常選人起手預檢](../data/parity-plans/ch18-preflight-select.jsonl)。已由[完整章收據](../data/ui-traces/parity-ch18.json)驗收，現況依58。

- [第十八章有限原生起手規格（#95）](../data/ida/fd2_ch18_startup_20261002.json)：335DA與既有shared tail、53筆原版起手；有限CONFORMED，整章例外驗收另見完整收據。

- [第十八章山景底面規格](../data/ida/fd2_ch18_parallax_backdrop_20261002.json)：#96，FDOTHER16/17 writer／consumer與有限CONFORMED契約；同狀態起手已驗收。

- [第十八章有限起手收據](../data/ui-traces/ch18-native-startup.json)與[五點抽樣來源](../data/ui-traces/ch18-native-startup-samples.json)：#95／#96，正常起手五點完整RGB0px、Go19與第17章98張四項回歸；只列有限RUNTIME-E1，整章仍#94。

- [第十八章正常護援計畫](../data/parity-plans/ch18-guard.jsonl)：#94，八回合含event43／42，round9才清敵；早期計畫的原始拒收保留，現行計畫見r4。

- [第十八章 event43 規格](../data/ida/fd2_ch18_event43_20261002.json)：#97，原始35091與34F37尾段將單位16模式低四位設3；CONFORMED，完整T3與戰後收據已通過。

- [第十八章模式9分派規格](../data/ida/fd2_ch18_mode9_dispatch_20261002.json)：#98，重用13A9F直接branch；有效raw目標先跟隨，不進14EF0；有限CONFORMED，首輪同槽已驗收。

- [第十八章護援r2計畫](../data/parity-plans/ch18-guard-r2.jsonl)：#94，固定同槽seed4與前四輪，只移除後四輪對戰死record2的選取；舊r1收據保留。

- [第十八章四輪有限對拍收據](../data/ui-traces/ch18-turns1-4.json)：#97／#98的正式事件與模式9順序修正；原版該計畫於round5停止；整章已由r4收據驗收。

- [第十八章event42完整規格](../data/ida/fd2_ch18_event42_20261002.json)：#99，3505F先追加group1再播text6，原始shared tail與CONFORMED；正常T8及戰後SAV已驗收。

- [指令傷害的原生職業抗性索引](../data/ida/fd2_command_damage_raw_class_20261002.json)：#100，sub_1C75E 的 target record+0x20 讀取證據與有限CONFORMED修正规格。

- [指令2目標間的演出亂數](../data/ida/fd2_command2_target_rng_20261002.json)：#101，既有六段HP marker的原始consumer與有限CONFORMED規格。

- [第十八章六輪有限收據](../data/ui-traces/ch18-turns1-6.json)：#100／#101，同一原版r2的213筆AI順序與153張RGB；該r2完整章仍拒收，現行r4見完整章收據。

- [第十八章有界後撤計畫r3](../data/parity-plans/ch18-guard-r3.jsonl)：#94，只將我方record0的正常移動目標改為8,10，原版第7輪STOSB停止，未解限制見#102；不修改NPC或HP。

- [工具收尾有限驗收](../data/ui-traces/tooling-closeout-20261003.json)：#91，四項過時 fixture／快照修正及既有 Capstone、Pillow 映像分流，63項通過；不改遊戲規則或證據等級。

- [第十八章原版素材解碼停止與唯讀追蹤](../data/ida/fd2_ch18_oracle_stosb_20261003.json)：#102，固定原版r3的STOSB停止、IDA sub_4E63D與有限重製前綴；目前oracle有限生命週期根因閉合，原生配置器重用未知；同r3仍停止，#102保持開啟。

- [第十八章第7／8輪正常後撤計畫r4](../data/parity-plans/ch18-guard-r4.jsonl)：#94，前六輪保持r3，record8正常鍵盤後撤；不改slot、seed、NPC或原版記憶體契約，完整章已驗收。

- [第十八章增援後戰後槽數契約](../data/ida/fd2_ch18_postbattle_slots_20261003.json)：#103，正常T8的53→75及11506 reader，舊55保留建構E1來源；CONFORMED，完整章通過。

- [第十八章戰後裝備重算契約](../data/ida/fd2_ch18_postbattle_equipment_20261003.json)：#104，115AC→1145A與酒店SAV單欄差異；CONFORMED，完整章及共用同步回歸通過。

- [戰後同步工具回歸與字串位置對照](../data/ui-traces/postbattle-tooling-regression-20261003.json)：#105，六個舊章測試補正式覆蓋；97筆人工處置保持，清冊位置轉移逐項可查。

- [第十八章整章四項收據](../data/ui-traces/parity-ch18.json)：#94／#97／#99／#103／#104，受版控r4正常章r8與Go r3；全檔SAV及正式回歸通過，依111／114列本章PLAYER-E2。

- [第十八章整章總覽來源索引](../data/ui-traces/parity-ch18-samples.json)：完整RGB抽樣及全部非零差異來源；圖面留本機，不新增公開原版素材。

- [第十九章建構槽清冊](../data/parity-slots/ch19-manifest.json)與[正常啟動計畫](../data/parity-plans/ch19-startup.jsonl)：固定111／114政策；前15人缺凱拉斯，拒收前綴保留。
- [第十九章初始整章抽樣計畫](../data/parity-plans/ch19-sample.jsonl)：缺identity16的早期拒收診斷，沒有完成戰鬥。
- [第十九章錯選identity7的計畫r2](../data/parity-plans/ch19-sample-r2.jsonl)：誤讀名字的拒收診斷，正式必出者是identity16凱拉斯。
- [第十九章正確選人計畫r3](../data/parity-plans/ch19-sample-r3.jsonl)：納入identity16，第二回合record2自然陣亡，後續計畫停止。
- [第十九章正式整章計畫r4](../data/parity-plans/ch19-sample-r4.jsonl)：保留前兩輪，後三輪正常END；T6先清敵一次，再以END執行事件46。舊第二行註解失效，實際順序見正式收據。
- [第十九章T6事件46完整契約](../data/ida/fd2_ch19_turn_event46_20261003.json)：#107，有限CONFORMED，文字1與永久JOIN27。
- [第十九章凱拉斯必出限制](../data/ida/fd2_ch19_required_character_20261003.json)：#108，identity16、原生訊息與返回城鎮，有限CONFORMED。
- [第十九章原生起手](../data/ida/fd2_ch19_startup_20261003.json)：#110，64筆、16格部署與HUD，有限CONFORMED。
- [第十九章事件44／45與整備名冊排列](../data/ida/fd2_ch19_modes_and_selection_pack_20261003.json)：#111，T4與持續排列通過；T10限局部E1。
- [第十九章完整四項對拍](../data/ui-traces/parity-ch19.json)、[全部非零差異抽樣索引](../data/ui-traces/parity-ch19-samples.json)與[凱拉斯必出拒收前綴](../data/ui-traces/ch19-required-party-rejection.json)：原版圖面與SAV留本機，分級與限制依58。

- [第二十章正常啟動計畫](../data/parity-plans/ch20-startup.jsonl)與[前章酒店存檔接續清冊](../data/parity-slots/ch20-manifest.json)：[#112](https://github.com/wicanr2/fd2_re/issues/112)，沿用第十九章SAV；準備期清冊，正常選人與整章結果見正式章收據。

- [第二十章原生起手視圖 READY](../data/ida/fd2_ch20_startup_20261003.json)：[#113](https://github.com/wicanr2/fd2_re/issues/113)，原版83筆前沿與HUD；record35位置差異另由#114追查。

- 第二十章初始配置的 writer／consumer 與 READY 規格：[fd2_ch20_initial_placement_20261003.json](../data/ida/fd2_ch20_initial_placement_20261003.json)；Issue #114，沿用已閉合的 LOADCH／0x10C50 證據。

- 第二十章章內／戰後正常鍵盤計畫：[ch20-sample-r1.jsonl](../data/parity-plans/ch20-sample-r1.jsonl)，Issue #112，尚待四項驗收。

- [第二十章有限起手收據](../data/ui-traces/ch20-native-startup.json)：83筆前沿與五點完整RGB，RUNTIME-E1；整章權威仍依58。

- [第二十章短路徑計畫](../data/parity-plans/ch20-sample-r2.jsonl)：沿用r1首回合與固定SAV，一次敵方階段後清敵；r1保留失敗来源，#112。

- [章比較器獨立原版介面契約](../data/fd2-chapter-node-comparison-contract.json)：#117，原版輸入鏈分類、跨時序限制與假通過的拒收條件；入口為 `tools/verify_chapter_parity.py`，在 `fd2-assets-local:20260829-sfx` 執行。 #158 的固定指紋／巢狀頭像返回配對補正亦存於此契約。

- [章比較器原版來源回歸](../data/ui-traces/parity-node-source-regression-20261003.json)：#117，第二十章兩個介面差異拒收、第十九章四項重新通過，以及舊節點欄位的證據限制。

- [原生物理攻擊目標確認 CONFORMED](../data/ida/fd2_player_physical_target_confirmation_20261003.json)：#115，sub_18D8C／sub_115B6保留原始名稱與bytes，補正式確認consumer與重播按鍵邊界。
- [第二十章商店方向鍵訂正計畫r3](../data/parity-plans/ch20-sample-r3.jsonl)：#112／#116，保留r2首回合與同源SAV，酒店選擇0以right到4，再Ctrl+F10；新原版r3及正式章收據已通過。

- [第二十章物理確認有限收據](../data/ui-traces/ch20-physical-target-confirmation.json)：#115，seq593／612完整RGB0px、原版正常取消／待機鍵序、完整Go19套件及第十九章完整回歸。整章未在此驗收。

- [第二十章正式四項對拍](../data/ui-traces/parity-ch20.json)與[全部非零差異抽樣索引](../data/ui-traces/parity-ch20-samples.json)：#112／#115／#116，31張完整RGB、酒店全檔SAV、正常商店鍵序；PLAYER-E2限111／114建構槽例外，現況與未驗範圍見58。

- [第二十一章酒店存檔接續清冊](../data/parity-slots/ch21-manifest.json)與[正常啟動計畫](../data/parity-plans/ch21-startup.jsonl)：[#118](https://github.com/wicanr2/fd2_re/issues/118)，SAV源自第二十章正式收據，不追加強化或治療；有限起手已驗收，整章仍待四項及酒店SAV。

- [第二十一章約拿必出及record1排序CONFORMED規格](../data/ida/fd2_ch21_required_character_20261003.json)：#119，raw chapter20／31C93 push21，沿用既有IDA writer與checker；提示初讀「索爾」已由固定名字索引22訂正。

- [第二十一章合法選入約拿計畫r2](../data/parity-plans/ch21-startup-r2.jsonl)：#118／#119，三次right僅改正常選人，SAV與政策不變；原版合法進場，末點PNG限制由r3補擷取。

- [第二十一章相同合法前綴與擷取等待r3](../data/parity-plans/ch21-startup-r3.jsonl)：#118／#120，原r2最後PNG未落檔；只在battle_start mark之後增加有界空白等待，固定同一SAV與鍵序。

- [第二十一章原生起手主證據與有限CONFORMED](../data/ida/fd2_ch21_startup_20261003.json)：#120，shared runtime0 focus、正常75筆及視圖／HUD；原版raw gate B精確值保留限制，五點完整RGB皆0px。

- [第二十一章有限起手及必出拒收收據](../data/ui-traces/ch21-native-startup.json)：#119／#120，75筆前沿、72筆active快照、五點完整RGB0px；缺約拿返回城鎮的五點畫面另列，拒收前綴節點判準仍有缺項，不提升整章。

- [第二十一章章內／天空之鑰與城鎮抽樣計畫](../data/parity-plans/ch21-sample-r1.jsonl)：#118，同源SAV與合法前綴，正常一個敵方階段後清敵一次；戰後使用實際道具分支，正式四項與全檔SAV通過。

- [第二十一章戰後JOIN資料來源有限CONFORMED](../data/ida/fd2_ch21_post_join_materialization_20261003.json)：#121，已閉合24312／2431C來源補到兩臂；修正缺少持續record的販售拒收，同收據四項／SAV通過。

- [第二十一章戰後固定配置表有限CONFORMED訂正](../data/ida/fd2_ch21_post_layout_tables_20261003.json)：#122，IDA與固定EXE75-byte三表一致，X起點21；保留舊證據並說明取代範圍，整檔SAV通過。

- [第二十一章正式四項對拍](../data/ui-traces/parity-ch21.json)與[全部非零抽樣索引](../data/ui-traces/parity-ch21-samples.json)：#118／#121／#122，30張完整RGB、酒店全檔SAV及原版不足臂；PLAYER-E2限111／114例外，現況與限制見58。

- [第二十二章接續清冊](../data/parity-slots/ch22-manifest.json)與[首15候選拒收計畫](../data/parity-plans/ch22-startup-r1.jsonl)：#123，同源第二十一章正式SAV，不追加強化或治療；原r1拒收保留，合法起手另見r2有限收據，後續記錄邊界已由正式第二十二章收據驗收。

- [第二十二章必出希爾法與持續record排序有限CONFORMED](../data/ida/fd2_ch22_required_character_20261003.json)：#124，raw21兩個caller均push24，固定名字索引25；沿用已閉合checker／321C8，合法起手五點完整RGB0px。

- [第二十二章合法選入希爾法起手計畫](../data/parity-plans/ch22-startup-r2.jsonl)：#124，鍵序依同源SAV解碼身份定位，原版及重製有限起手五點完整RGB0px。

- [第二十二章原生起手有限CONFORMED](../data/ida/fd2_ch22_startup_20261003.json)：#126，3367E新caller與shared runtime0 focus，66筆／camera16,27；raw HUD B精確值保留限制。

- [第二十二章正常起手與必出拒收有限收據](../data/ui-traces/ch22-native-startup.json)：#124／#126，66筆前沿、63筆active快照；合法與拒收各五點完整RGB0px，此起手最高RUNTIME-E1，整章另見正式收據；#125保存局部E1已由四槽收據驗收。

- [非城鎮記錄提示探查清冊](../data/parity-slots/preparation-record-probe-manifest.json)與[正常鍵盤計畫](../data/parity-plans/preparation-record-probe-r1.jsonl)：#125，獨立建構槽只探保存UI與DOS writer，最高局部E1，不代替第二十二章同源整章路徑。

- [非城鎮記錄提示保存後Escape計畫](../data/parity-plans/preparation-record-probe-r2.jsonl)：#125，原版已證成功保存後留四槽列表；r2只改最後退出鍵，歷史待驗狀態由r3正常選人收據取代。

整備記錄四槽保存主證據與 READY 規格：[fd2_preparation_record_save_20261003.json](../data/ida/fd2_preparation_record_save_20261003.json)。保留原始名稱、IDA LE 位址與bytes；建構raw22槽probe只算E1。

#125／#127 可重跑保存計畫：[preparation-record-save-r3.jsonl](../data/parity-plans/preparation-record-save-r3.jsonl)。同一建構槽，正式driver正常YES／保存／ESC；局部四項、四張完整RGB0px及全檔SAV已驗收，最高RUNTIME-E1。

### 2026-10-03 非城鎮整備四槽保存（#125／#127）

[主規格](../data/ida/fd2_preparation_record_save_20261003.json)與[有限收據](../data/ui-traces/preparation-record-save.json)限局部RUNTIME-E1列CONFORMED。2CC76問題YES→2CCBB／3009C四槽；Enter寫完整SAV後保留列表，ESC才到31A2E零勾選選人。重製已補上正式owner，NO不保存；素材／來源或寫入失敗即停止。

四個完整RGB皆0px，22987 bytes SAV兩側SHA-256相同。原版成功寫入22528+459 bytes，內容相同也屬合法覆寫。driver、replay及比較器新增preparation_save；缺planned save、錯owner、無成功寫入均拒收，沒有跨時序畫面豁免。相關三套件與第二十一章30畫面／全檔SAV回歸通過。建構raw22槽不代表第二十二章戰後可達；#123仍開啟，整章數以正式台帳為準。

- [第二十二章整章抽樣計畫r1](../data/parity-plans/ch22-sample-r1.jsonl)：#123，同源SAV、正常城鎮交易／酒店保存、T3／5／7增援與T7清敵一次，非城鎮記錄保存；r1為第五回合敗北診斷，未到第七回合；正式通過來源為r2。

- [第二十二章第五回合短路徑計畫r2](../data/parity-plans/ch22-sample-r2.jsonl)：#123，同源SAV及seed4；原r1第五回合敗北，r2在已抽樣T3／T5的玩家游標清敵一次，不刷關、不追加強化或治療，T7不列本收據範圍。

- [第二十二章T5事件50完整有限CONFORMED契約](../data/ida/fd2_ch22_turn_event50_20261003.json)：#129，group2、PAN16,42、8 BIOS tick、JOIN20及文字2九句；正式原版視圖與全檔保存已在章收據驗收。

- [第二十二章原版record欄位有限CONFORMED契約](../data/ida/fd2_ch22_record_fields_20261003.json)：#131，24512原始16-byte布局表與LOADCH 10AB1死亡效果禁用writer；不在SAV或重播猜補。

- [第二十二章正式四項章收據](../data/ui-traces/parity-ch22.json)：#123／#128至132，PLAYER-E2限111／114例外；原975點、42動作、212筆AI零分岔，34張完整RGB、全檔SAV相同。T7與延後PNG限制如實保存。

- [第二十三章接續清冊](../data/parity-slots/ch23-manifest.json)：#133，沿用第22章正式SAV3c7298cd，不追加強化或治療。
- [第二十三章raw22必出／前置有限CONFORMED](../data/ida/fd2_ch23_required_character_20261003.json)：#134，重用已有31CA0／31D12原始條件與checker／writer，未外推整章。
- [第二十三章缺希爾法探查](../data/parity-plans/ch23-startup-r1.jsonl)與[合法首15起手探查](../data/parity-plans/ch23-startup-r2.jsonl)：正常非城鎮NO→選人；r1保留誤標診斷，r2有限驗收通過。

- [第二十三章拒收返回選人計畫r3](../data/parity-plans/ch23-startup-r3.jsonl)：#134／#135，同源r1鍵序，僅訂正最後來源標籤；保留r1誤標診斷。

- [第二十三章原生起手契約（#136）](../data/ida/fd2_ch23_startup_20261003.json)：原始caller、typed view／HUD及有限驗收。

- [第二十三章第二回合診斷計畫r1](../data/parity-plans/ch23-sample-r1.jsonl)：#133，同源SAV、正常攻擊／移動、敵方回合與一次清敵，已完成T2有限診斷。不涵蓋晚期事件52，不作整章驗收。

- [第二十三章有限原生起手收據](../data/ui-traces/ch23-native-startup.json)：#134／#135／#136／#137，合法四點與拒收返回五張RGB；整章及事件52另驗。

- [第二十三章42筆戰後前沿READY](../data/ida/fd2_ch23_post_frontier_20261003.json)：#139，正常T2診斷反證exact86；沿現有slot_counts格式，不外推晚期前沿。

- [第23章戰後視圖重設與共用載體 CONFORMED，Issue #140](../data/ida/fd2_ch23_post_view_20261003.json)：沿用233C6六全域writer；T2有限RGB／SAV與完整Go通過，不提升整章覆蓋。

- [第23章T2戰後有限收據](../data/ui-traces/ch23-post-frontier.json)：42筆正常戰後、14張全RGB及完整SAV；不作整章驗收。

- [第23章晚期事件抽樣計畫r2](../data/parity-plans/ch23-sample-r2.jsonl)：#133／#138，正常END推進與事件前後T13／15／18／22；r2於T18主角死亡，T22與保存未執行，不算整章驗收。

- [第23章晚期抽樣r3](../data/parity-plans/ch23-sample-r3.jsonl)：r2索爾T18死亡後，以相同seed／SAV、正常移動record0避開敵軍；不改HP／DP，未通過不作驗收。

- [第23章事件52 CONFORMED契約](../data/ida/fd2_ch23_event52_20261003.json)：#138，IDA完整byte算式、空群組consumer及T13／15／18原版stack；T22動態未抽樣；整章正式驗收範圍另見本章收據。

- [第23章三次事件後保存計畫r4](../data/parity-plans/ch23-sample-r4.jsonl)：#133／#138，保留r3相同前綴，到T19玩家游標清敵一次；T22不列本次抽樣，敗北不刪除。

- [第23章正式四項對拍](../data/ui-traces/parity-ch23.json)及[全部非零RGB抽樣](../data/ui-traces/parity-ch23-samples.json)：#133／#138／#141；T13／15／18正常事件、T19一次清敵、62筆正常戰後與全檔SAV，24張完整RGB。PLAYER-E2限111／114例外，T22未抽樣；回歸已通過。

- [第24章接續槽清冊](../data/parity-slots/ch24-manifest.json)與[正常起手有限計畫](../data/parity-plans/ch24-startup-r1.jsonl)：#142，沿用第23章完整SAV、seed4，零額外改寫；尚未驗收。

- [第24章四次增援抽樣計畫](../data/parity-plans/ch24-sample-r1.jsonl)：#142，正常移動與END至T11後清敵一次、正常戰後保存；未通過四項門檻不作驗收。

- [第24章正常起手前沿有限CONFORMED契約](../data/ida/fd2_ch24_startup_frontier_20261003.json)：#143，0x338CE／0x338FC正常16＋4筆，完整清冊97筆重建失敗反證；先規格再資料契約。

- [第24章存活隊員正常選人計畫r2](../data/parity-plans/ch24-sample-r2.jsonl)：#142，保留r1 T4敗北；相同SAV與seed4，只用正常方向鍵排除HP0隊員，不改HP。

- [第24章舞台執行期有限CONFORMED契約](../data/ida/fd2_ch24_stage_runtime_20261003.json)：#144／#150，沿既有#42／0x10652／0x11EEE case23；補正常compositor並接續戰後owner，完整0x11D3B copy可重綁來源。抽樣相位已驗，晚期RGB／AI差異與未抽樣仍未知。

- [第24章T3戰後保存有限計畫r3](../data/parity-plans/ch24-sample-r3.jsonl)：#142／#144，r1與r2均T4敗北，不再刷長局；只抽樣事件54@T2，再一次清敵及正常戰後保存。本有限計畫未抽樣T4／7／10；目前整章現況見[58](58-fd2-exe-re-coverage.md)。

- [第24章28筆正常戰後有限CONFORMED](../data/ida/fd2_ch24_post_frontier_20261003.json)：#145，24C4C／24CAD原版caller與28筆count；86只留歷史fixture，其他形狀仍拒收。

- [第24章既定政策全員存活建構槽](../data/parity-slots/ch24-fresh-policy-manifest.json)與[有界正常LOAD探查](../data/parity-plans/ch24-fresh-startup-r1.jsonl)：#142，沿111／114固定AP+200／DP+0／DX+60與seed4，不額外改政策。

- [第24章有限同狀態收據](../data/ui-traces/ch24-finite-parity.json)：#143／#144／#145限RUNTIME-E1驗收，兩種正常選人分支、T2增援與28筆戰後保存；#142整章仍需T4／7／10。

- [第24章全員存活槽正常接戰至T11計畫](../data/parity-plans/ch24-fresh-sample-r1.jsonl)：#142，沿已驗正常LOAD前綴，正常接戰後END，T11才清敵一次；晚期與戰後尚未驗收。

- [第24章原版執行器D8 /1停止與READY契約](../data/ida/fd2_ch24_oracle_d8_20261004.json)：工單#146，CPU切片與整章#142分開。

- [第24章原版執行器FCOS缺口與三角函數READY](../data/ida/fd2_ch24_oracle_fcos_20261004.json)：#147，FSIN目前只有靜態原bytes證據。

- [第24章原版短JP／JNP缺口與READY](../data/ida/fd2_ch24_oracle_parity_branch_20261004.json)：#148，原x87狀態消費端的PF分支。

- [第24章原版FILD m16int缺口與READY](../data/ida/fd2_ch24_oracle_fild16_20261004.json)：#149，已知乘加鏈的有界指令盤點。

- [第24章正常避敵計畫r2](../data/parity-plans/ch24-fresh-sample-r2.jsonl)：#142同槽／seed鍵盤輸入；現況與限制見[58](58-fd2-exe-re-coverage.md)。

- [第24章指令6工作緩衝與目標演出契約](../data/ida/fd2_ch24_command6_work_bounds_20261004.json)：#151 的0x2A300-byte／640-byte列距配置；#156 首次mode3座標、分層與跨目標state有限CONFORMED；#154負列仍拒收；#157正常命中／落空亂數交錯及#162的__CHP向零取整達有限CONFORMED；#161已證caller組圖接入正式路徑，11張正式component全圖indexed／RGB相同，第7張按#154拒收；12張一致仍屬隔離原型，完整正式施法未完成。

- [指令7跨目標亂數交錯](../data/ida/fd2_command7_target_rng_20261004.json)：#152，有界原版trace及規格入口。

- [指令6亂數有界正常計畫](../data/parity-plans/ch24-command6-rng-r1.jsonl)：#157，同fresh-r1至第5回合入口，命中／miss逐次trace；主證據與狀態在[指令6契約](../data/ida/fd2_ch24_command6_work_bounds_20261004.json)。

- [指令6非零側正常施法計畫](../data/parity-plans/ch24-command6-side1-r1.jsonl)：#154，同固定槽第4回合記錄13，原版寫入／viewport消費端追蹤；主證據仍為指令6工作契約。

- [#154 非零側正常指令6輸入修正版](../data/parity-plans/ch24-command6-side1-r2.jsonl)：同槽同政策；指令環方向鍵後確認，再等待法術清單。主證據見fd2_ch24_command6_work_bounds_20261004.json的normal_probe。

- [#154 玩家槽原地指令6短路徑](../data/parity-plans/ch30-command6-side1-player-r1.jsonl)：既有固定第三方槽普通CONTINUE、前回合移動待機，下回合原地施法；不增加章E2。來源與實驗收據見指令6主證據normal_probe。

- [#154 玩家槽原地範圍指令6探針](../data/parity-plans/ch30-command6-side1-player-r2.jsonl)：同源正常CONTINUE、不經過敵軍回合；檢驗既有selection／effect與空格確認，非章E2。

- [#160 玩家指令6範圍中心 RE 與草案](../data/ida/fd2_player_command6_cursor_center_20261004.json)：固定IDA caller／bytes、正常原地範圍確認收據；修正直接enemy候選假設及指令6白名單漏接；READY與有限RUNTIME-E1、同源CONTINUE探針、#154阻擋及剩餘驗收在同一主證據。

- [酒店讀檔與傳聞返回契約](../data/ida/fd2_hotel_load_20261004.json)：#32，服務2的整備槽拒絕／成功提示及酒店返回直接bytes；[正常酒店讀檔計畫](../data/parity-plans/hotel-load-ch04-r1.jsonl)只使用已登記第4章固定槽。

  #32 計畫多送鍵的勘誤及[修訂計畫r2](../data/parity-plans/hotel-load-ch04-r2.jsonl)由同一酒店主契約承載，r1不覆寫。

  [酒店 LOAD 完整影格比較工具](../../tools/verify_hotel_load_parity.py)只驗固定停點的合法相位等價；用法與收據見酒店主契約。

- [END回復演出主契約](../data/ida/fd2_end_turn_recovery_20261004.json)：#29，原版兩輪work發布／一次sample4／HP與bit7 writer；[正常END探針](../data/parity-plans/end-recovery-ch04-r1.jsonl)固定第4章建構槽，無章內注入。

  #29 [END有候選探針r2](../data/parity-plans/end-recovery-ch04-r2.jsonl)完整沿既有第4章輸入，第一／二輪無候選與後續正例均由同一主契約承載；#163為另已登記的AI consumer缺口。

- [END 正常探針 r3](../data/parity-plans/end-recovery-ch04-r3.jsonl)：Issue #29，固定槽五次正常 END，不移動與不注入。

- [END 回復正式演出](../../remake/cmd/fd2/native_end_turn_recovery.go)：Issue #29，依上述 READY 規格執行兩輪發布。
- [END 回復演出驗證](../../remake/cmd/fd2/native_end_turn_recovery_test.go)：發布、交易拒絕、遮罩與畫面外候選。
- [END 同源探針](../../remake/cmd/fd2/native_end_turn_recovery_probe_test.go)：固定正常原版收據的 E1 狀態夾具與正式 END／YES。
- [END 完整畫面比較工具](../../tools/verify_end_recovery_parity.py)與[收據](../data/ui-traces/end-recovery-20261004.json)：固定三張原版影格、單次 sample4、完整索引與 RGB。

- #163 [AI原地回復caller規格](../data/ida/fd2_ai_idle_recovery_20261004.json)與[正常ch04探針](../data/parity-plans/ai-idle-recovery-ch04-r1.jsonl)：重用13FD4／1DA16／4DDD7證據，補三個indexed邊界；已CONFORMED／RUNTIME-E1；正常原版接受分支三張完整畫面已閉合，範圍限此caller。

- #163 [正常ch08探針](../data/parity-plans/ai-idle-recovery-ch08-r3.jsonl)改選既有mode2守軍章，原政策不變；結果回填同一caller規格。

- #163 [AI同源E1探針](../../remake/cmd/fd2/native_ai_idle_recovery_probe_test.go)：正常LOAD／出戰後匯入原版停點raw狀態，經正式AI呈現consumer輸出三張完整影格，原版像素只供比較。

- #163 [正常洛娜原地攻擊探針](../data/parity-plans/ai-idle-recovery-ch08-r4.jsonl)：依已裝item22的range[1,2]選取正常鍵盤路徑，回填同一caller規格。

- #163 [ACTING末端起點的正常洛娜路徑](../data/parity-plans/ai-idle-recovery-ch08-r5.jsonl)：修正r4使用演出中途座標的輸入計畫，完整來源與限制見同一caller規格。

- #163 [索爾正常接戰探針](../data/parity-plans/ai-idle-recovery-ch08-r6.jsonl)：依raw AP／DP與已證mode2分支取得受傷敵方停點，不修改正式隊伍政策，回填同一caller規格。

- [AI 原地回復 r6 輸入的精確停點重播](../data/parity-plans/ai-idle-recovery-ch08-r7.jsonl)：#163 的三次顯示入點與 HP 尾端；不改鍵盤輸入或亂數。
- [AI 原地回復完整索引與 RGB 核對工具](../../tools/verify_ai_idle_recovery_parity.py)：在 Docker 內核對 #163 的 r7 三停點、HP 尾端及合法相位；不遮罩像素。

- [AI 原地回復三次顯示與 HP 尾端收據](../data/ui-traces/ai-idle-recovery-20261004.json)：#163，正常原版重生、同源 raw E1、完整索引／RGB 0 差異；不提升章台帳。

- [第七章目前重播的節點配對回歸收據](../data/ui-traces/parity-ch07-node-regression-20261004.json)：#164，唯一語意來源與實際對白 EIP 判準；歷史原版 sample-r6、目前 Game 四 gate 通過，不新增章 PLAYER-E2。規格沿用[節點主契約 extension164](../data/fd2-chapter-node-comparison-contract.json)。

- [原版近堆替代政策與透明底色勘誤](../data/ida/fd2_terrain_mode3_review_20261001.json)：#167，oracle_heap_policy_correction；檢查方式見[對拍工具鏈](96-parity-toolchain-20260909.md)，入口[Docker收據](../../tools/dosgolem_oracle_container.sh)與[來源反例](../../tools/test_dosgolem_oracle_drive.py)。原版配置器未知，#52最後1px保留，不改章門檻。

- [物理攻擊背景與台座選擇契約](../data/ida/fd2_physical_background_selection_20261004.json)：#166，保留28A6C與12E38的原始位址、bytes、推論等級及正常第八章dosgolem收據。選擇規則有限CONFORMED，正式BG／TAI快照列RUNTIME-E1；剩餘整場景與工作緩衝驗收見[58](58-fd2-exe-re-coverage.md)。
- #166 程式與測試：[具型別選擇](../../remake/internal/battle/native_physical_scene.go)、[選擇反例](../../remake/internal/battle/native_physical_scene_test.go)、[正式素材consumer](../../remake/cmd/fd2/native_physical_scene.go)、[Game來源及原子性測試](../../remake/cmd/fd2/native_physical_scene_test.go)。

- #166 [正常敵方場景輸入計畫](../data/parity-plans/physical-background-ch08-enemy-r1.jsonl)：完整沿用第八章r7前綴；此版end_turn未受支援，失敗紀錄保留於上列主契約。
- [第八章敵方物理場景正常 END 修正版計畫](../data/parity-plans/physical-background-ch08-enemy-r2.jsonl)：同一前段輸入，以實際按鍵送 END；證據同上。

- #166 [第十二章雙敵方回合場景計畫](../data/parity-plans/physical-background-ch12-enemy-r1.jsonl)：沿用既有抽樣前35行，實測見[場景主契約](../data/ida/fd2_physical_background_selection_20261004.json)。

- #166 [非零旗標首次轉場固定輸入](../data/parity-plans/physical-scroll-ch12-fixed-r1.jsonl)與[IDA9.4非破壞匯出工具](../../tools/ida_probe_physical_presentation.py)：首次departure與29C90／29DED的caller、bytes及分級合併至[場景主契約](../data/ida/fd2_physical_background_selection_20261004.json)。
- [物理首次捲動比較圖](../figures/physical-scroll-scoped-compare.png)：原版、同輸入合成與完整未遮差異；不外推完整攻擊或章PLAYER-E2。


- #166 [逐揮物理演出計畫](../../remake/internal/battlepresent/native_physical_body.go)與[規則反例](../../remake/internal/battlepresent/native_physical_body_test.go)、[正式Game owner](../../remake/cmd/fd2/native_physical_body.go)與[完整合成／GPU及原子性測試](../../remake/cmd/fd2/native_physical_body_test.go)：READY及有限CONFORMED範圍見[場景主契約](../data/ida/fd2_physical_background_selection_20261004.json)。
- [單次未命中物理尾段比較圖](../figures/physical-tail-scoped-compare.png)：原版、同輸入合成與完整未遮差異，不外推連擊、counter或章PLAYER-E2。
- #166 分離音效[可重跑匯出工具](../../tools/export_sfx.py)與[strict銀行consumer](../../remake/internal/fdother/separated_sound.go)：來源、硬體規格近似、私人清冊同步及目前覆蓋只見[主契約](../data/ida/fd2_physical_background_selection_20261004.json)與[58](58-fd2-exe-re-coverage.md)。


- #166 正常零旗標主攻／counter的完整對拍：[既有測試檔的TestNativePhysicalBodyCounterNormalOracle](../../remake/cmd/fd2/native_physical_body_test.go)，由正式resolver解碼raw記錄並結算一次。固定copy／DAC取樣契約、IDA11EED返回證據及同輸入限制見[主契約physical_counter_validation](../data/ida/fd2_physical_background_selection_20261004.json)，唯一結果見[58](58-fd2-exe-re-coverage.md)。


- #168 [正式finishAttackPresentation](../../remake/cmd/fd2/main.go)與[原生VGA收尾回歸](../../remake/cmd/fd2/native_physical_body_test.go)：沿用已閉合290AC..290BD直接bytes，證據與READY契約見[physical_map_return_spec](../data/ida/fd2_physical_background_selection_20261004.json)，目前驗證只見[58](58-fd2-exe-re-coverage.md)。map work初值限制仍在#166／#167。

- #166 [第八章攻方非零側別的固定四輪探針](../data/parity-plans/physical-own-nonzero-ch08-r1.jsonl)：正常LOAD前74步沿用r7，首次射擊與窗口政策記錄於[physical_opposite_side_probe](../data/ida/fd2_physical_background_selection_20261004.json)。

- #166 [第八章攻方非零側別探針 r2](../data/parity-plans/physical-own-nonzero-ch08-r2.jsonl)：排除實測第28筆普通我方隊員，只修正常輸入名單；r1失敗與同槽限制保留於上述physical_opposite_side_probe。

- #166 [第八章弓手首擊的固定控制窗口](../data/parity-plans/physical-own-nonzero-ch08-fixed-r1.jsonl)：r2首次record5→11射擊的controls1..201，不挑結果；READY驗收見上述physical_tail_spec.opposite_side_acceptance_extension。

- #166 [弓手首擊完整畫面測試](../../remake/cmd/fd2/native_physical_body_test.go)：TestNativePhysicalBodyOwnNonzeroNormalOracle與既有GPU入口共用正式resolver；caller分界及有限CONFORMED見[physical_opposite_side_validation](../data/ida/fd2_physical_background_selection_20261004.json)，唯一現況見[58](58-fd2-exe-re-coverage.md)。


- #169／#170 [正常物理尾端等待與聲音wrapper契約](../data/ida/fd2_physical_background_selection_20261004.json)：physical_sound_evidence／physical_return_wait_spec；IDA窄匯出由[既有工具](../../tools/ida_probe_physical_presentation.py)的FD2_IDA_PHYSICAL_SOUND=1重生，正常17AA9(6)與25A96停止保留原始定位；目前分層與限制只見[58](58-fd2-exe-re-coverage.md)。


- #170 [正常物理同通道PCM與返回順序的正式實作](../../remake/cmd/fd2/audio.go)、[回歸與原子拒收](../../remake/cmd/fd2/audio_voice_test.go)：physical_sound_spec／validation的有限CONFORMED見[主契約](../data/ida/fd2_physical_background_selection_20261004.json)，唯一現況與原版限制見[58](58-fd2-exe-re-coverage.md)。事件61的追加敵兵回歸見[既有測試](../../remake/cmd/fd2/native_field_event61_test.go)。

- #166 [零旗標敵方主攻與友軍命中反擊的完整演出驗收](../data/ida/fd2_physical_background_selection_20261004.json)：physical_counter_hit_probe／validation與counter_hit_acceptance_extension；[受版控測試](../../remake/cmd/fd2/native_physical_body_test.go)沿用106與固定輸入，保留raw camp 1為Ally、同控制移動座標及原始caller界線。唯一現況見[58](58-fd2-exe-re-coverage.md)。

- #171 [逐格同時點單位原始記錄的已驗工具契約](../data/ida/fd2_physical_background_selection_20261004.json)：physical_frame_unit_records_spec／validation；[既有oracle包裝入口](../../tools/dosgolem_oracle_container.sh)傳遞可選旗標，完整來源與範圍見[58](58-fd2-exe-re-coverage.md)。

- #172 [正常物理返回11CAC(1)色盤閘門證據與限定CONFORMED](../data/ida/fd2_physical_background_selection_20261004.json)：physical_return_palette_evidence／spec／validation重用已閉合caller及palette gate原始bytes；正式來源見[58](58-fd2-exe-re-coverage.md)。

- #173 [完整地圖同時點週期／HUD／色盤來源缺口](../data/ida/fd2_physical_background_selection_20261004.json)：physical_map_runtime_spec／tool_validation已限定工具CONFORMED；原缺口歷史保持，限定consumer結果見後文，現況見[58](58-fd2-exe-re-coverage.md)。

#173的[physical_map_runtime_evidence／spec](../data/ida/fd2_physical_background_selection_20261004.json)已複核既有IDA欄位與HUD dword指令，為RE-CLOSED／READY。只授權唯一oracle可選唯讀觀測；完整map與PLAYER-E2未驗收。

- #173 [16個原始地圖全域與獨立色盤的已驗工具](../data/ida/fd2_physical_background_selection_20261004.json)：physical_map_runtime_tool_validation；[包裝入口](../../tools/dosgolem_oracle.sh)以FD2_ORACLE_MAP_STATE=1啟用，支援既有有界trace／接受frame，來源與限制見[58](58-fd2-exe-re-coverage.md)。

#166的[physical_map_consumer_probe](../data/ida/fd2_physical_background_selection_20261004.json)保留DRAFT診斷歷史；正式限定驗收見後文。

#166的[physical_map_return_acceptance_spec／validation](../data/ida/fd2_physical_background_selection_20261004.json)已限定CONFORMED／RUNTIME-E1；同源完整返回畫布與色盤通過，時鐘近似及一般work生命週期限制保持。正式測試入口為[物理演出測試](../../remake/cmd/fd2/native_physical_body_test.go)，目前數字見[58](58-fd2-exe-re-coverage.md)，不升PLAYER-E2。

#166的[physical_own_map_return_probe](../data/ida/fd2_physical_background_selection_20261004.json)為DRAFT；固定第八章弓手首擊後返回地圖，沿原槽／控制／窗口補同時點來源，尚未提高驗收層級。

#166的[physical_own_map_return_acceptance_spec](../data/ida/fd2_physical_background_selection_20261004.json)已READY：固定原版輸入、caller與同時點來源已核對，授權共用完整畫布驗收；重製結果仍待實跑。

#174的[physical_map_hud_cycle_evidence／spec](../data/ida/fd2_physical_background_selection_20261004.json)已RE-CLOSED／READY；optional HUD讀取更新前idle，先建立產品反例再修正候選交易來源。#166其餘索引差異仍待分類。

#175的[physical_own_map_selector_probe](../data/ida/fd2_physical_background_selection_20261004.json)為DRAFT；固定首次返回窄追原始cycle與sprite pointer，不用候選像素挑幀。

#175的[physical_own_map_raw_restore_spec](../data/ida/fd2_physical_background_selection_20261004.json)已READY：正式oracle確認slot5／pose2／cycle2與指標，308差異來自驗收constructor重設raw姿勢。修正限測試輸入，未證明正式blitter有缺陷。

目前#174 HUD來源與#175驗收raw匯入勘誤已限定CONFORMED；入口為[physical_map_hud_cycle_validation／physical_own_map_return_acceptance_validation](../data/ida/fd2_physical_background_selection_20261004.json)及[58現況表](58-fd2-exe-re-coverage.md)。先前READY與拒收段落保留形成過程，不再作現況待辦。

#166反擊MISS的[physical_counter_miss_map_return_acceptance_spec／validation](../data/ida/fd2_physical_background_selection_20261004.json)已限定CONFORMED／RUNTIME-E1；完整返回畫布與色盤通過，原版caller、固定來源與限制見[58現況表](58-fd2-exe-re-coverage.md)。一般保留work生命週期尚未驗收。

#166的[physical_owner_map_handoff_probe](../data/ida/fd2_physical_background_selection_20261004.json)保存DRAFT反例；正式正常Game保留work交接與DAC caller已限定CONFORMED，見physical_owner_return_dac_validation，未宣稱PLAYER-E2。

- [正常物理owner的DAC返回交接](../data/ida/fd2_physical_background_selection_20261004.json)：#176的physical_owner_return_dac_evidence／physical_owner_return_dac_spec，IDA9.4與正常caller漸暗／漸亮已閉合；正常完整Game保留work與GPU已限定CONFORMED，驗收見physical_owner_return_dac_validation。

[完整正常物理owner驗收測試](../../remake/cmd/fd2/native_physical_owner_test.go)核對固定前置map、完整兩段DAC／GPU、保留work與最後caller返回；主契約physical_owner_return_dac_spec／validation限定RUNTIME-E1。

- [城鎮出發與LOADCH過場主證據](../data/ida/fd2_town_departure_20261005.json)：#39 的限定CONFORMED／RUNTIME-E1；[可重跑原版鍵盤序列](../data/parity-plans/ch07-town-departure-probe.jsonl)。正式現況見[58](58-fd2-exe-re-coverage.md)。

#39 的正式接線入口為 [城鎮出發擁有者](../../remake/cmd/fd2/native_town_departure.go)及[定點採樣器](../../remake/internal/campaign/native_town_departure.go)；READY 範圍與驗收仍依主證據。

[城鎮過場完整來源／正常owner驗收](../../remake/cmd/fd2/native_town_departure_test.go)使用#39的固定槽與原版兩個取樣窗口；沒有收據時明示略過。

[城鎮過場總覽產生器](../../tools/fd2_town_departure_sheet.py)由原版indexed及同時點DAC重建左欄、正式GPU擷取作中欄，僅輸出壓平總覽；命令與範圍見主契約。

- [確認框共享相位證據](../data/ida/fd2_confirmation_pulse_20261005.json)：#35的原始writer／consumer；共享相位規格已限定CONFORMED，現行原版及GPU通過。[有界38格正常鍵盤探針](../data/parity-plans/ch04-confirmation-pulse-probe.jsonl)使用既有第四章槽。

[#35四點有限驗收](../../remake/cmd/fd2/native_confirmation_pulse_oracle_test.go)由正常LOAD及出口／ESC取得兩個合法相位；戰後兩點只比較排版，不取代章收據。

[#35四點收據產生器](../../tools/fd2_confirmation_pulse_receipts.py)只補驗已知確認框並保存歷史四點，不重跑或重寫整章來源。

[#38確認嘴型與循環色候選契約](../data/ida/fd2_parity_mouth_cycle_20261005.json)沿既有19953／4DFCC原始證據；正式嘴型owner另見[#177](https://github.com/wicanr2/fd2_re/issues/177)。

[#38整備嘴型候選與狀態還原測試](../../remake/cmd/fd2/chapter_parity_preparation_mouth_test.go)及[三點收據補驗工具](../../tools/fd2_mouth_cycle_receipts.py)只閉合工具層，不代替#177正式等待owner。
