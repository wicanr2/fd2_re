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
