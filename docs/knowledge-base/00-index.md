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
| 下一批可執行工作 | [`91` worklist](91-worklist.md) 檔首有效佇列 | 檔內歷史輪次、已刪除的舊規劃快照 |
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
`56` FD2 remake SDD（UI／campaign／證據 gate） · `57` UI evidence matrix · `58` FD2.EXE RE／remake 覆蓋矩陣 · `91`worklist · `99`反思

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
