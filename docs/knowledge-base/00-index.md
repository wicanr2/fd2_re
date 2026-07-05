# 炎龍騎士團2 逆向工程知識庫 — 索引(問題導向路由)

> 《炎龍騎士團2》(Flame Dragon Knight 2),漢堂國際 1995,DOS / DOS4GW 保護模式。
> 由逆向工程逐輪累積。**每輪 RE 發現與反思都寫進這裡**;後輪推翻前輪時回去修正/刪除,不堆矛盾。
>
> **用法**:先看下面「§A 問題 → 查哪份」路由表定位;§B 完整文件清單;§C 機器可讀資料(別忘了用);§D 還原 chNN.json 工作流。
>
> **🎯 目標 + [HARD] 鐵則**:用 Go/Ebiten + RE 原版 DOS 做**一模一樣**的 remake。
> **禁止用推測/外推寫 code**——每個進 code 的值(座標/幀數/鏡頭/時機/射程/回合)必須有 RE 來源
> (反組譯 doc47/50、dosbox doc48、青衫、影片、FDFIELD 直讀)。不知道就先 RE 拿真值,拿不到就誠實停,不准猜。
> (BeatRunner 外推 pan 值→越改越偏的教訓;驗收對 reference 實測非「測試綠」,規則 65。)

---

## §A 問題 → 查哪份文件(路由表)

### 過場 / 開場動畫 / 劇情演出(近期主線)
| 我想知道… | 查 |
|---|---|
| 第一章**開場**逐幕時間軸(王座廳→草地→密林→行軍→海島)、remake 差異 | **`46`**(影片 ground truth,最準的視覺時間軸) |
| 開場 handler `0x3231b` **完整指令序列**(每 beat 的 call+參數+語意) | **`47`**(§3/§7 逐 beat 全轉錄) |
| START→開場→第一關第一回合**逐項 RE 來源對照**(禁推測驗收表:哪些✅可寫/⚠須換/❓待RE) | **`53`** |
| 過場**原語**(pan/走位/對白/演出/spawn/入隊/step家族)怎麼運作、位址 | **`50`**(過場機制唯一主檔;原始逐beat轉錄見`47`) |
| 「兩套腳本系統」——開場 cutscene vs 戰鬥中事件對話,界線在哪 | **`52`** §0(**先讀這個再碰過場**) |
| 某關**事件骨架**(第幾回合/增援/加入/勝敗)——還原 chNN.json 的 ground truth | **青衫攻略 `references/text/fd2-walkthrough-index.md`**(30 關分關卡索引)+ 全文 `references/text/fd2.md` |
| 某句對白**第幾回合/什麼事件**觸發(哈諾/海盜頭目/海防隊) | 青衫索引(時機)+ **`ch01.json` events**(Fable5 RE 範本)+ `26` + `battle_events.json` + `52` §1.2 |
| 草地幕走位逐幀量測(原始數據) | `55`(機制見 **`50` §1.1**) |
| 索爾四人**怎麼進戰場**(進場動畫/站位) | `52` §1.1 + `46` §4(⚠ 進場動畫細節待 dosbox 定稿) |
| acting 機制(=原地姿態/幀動畫,**不搬格子**) | **`50` §1.2**(`54` 僅存 dosbox 實測原始記錄) |
| 「走位」機制(step家族4方向+路徑走位0x13488)、單位欄位 +0/+1/+3/+4、面向規則 | **`50` §1.1** |
| remake 過場**引擎**(BeatRunner / cutscene 節點 / beats DSL)怎麼設計 | **`50`**(§2 DSL,§3 全33關管線) |
| 全 33 關過場 beats(機器可讀) | `docs/data/chapter_beats/chNN_{pre,post}.json` |
| 開機/標題/主選單/劇情自動過場流程(反組譯) | `23` + `39`(ANI.DAT AFM 開場) |

### 角色 / 單位 / 數值
| 我想知道… | 查 |
|---|---|
| portrait/char id → **角色名** | **`49`** + `docs/data/portrait_names.json`(證據分級) |
| 說話者 id 兩種定址(-17/-18 全域 vs -19/-20 場景) | `40` |
| 職業名顯示錯位(海盜→劍士 bug) | `45` |
| **武器/攻擊範圍/物品數值**(靜態表,不需 debugger) | `32` + `02` + `03`(青衫+反組譯) |
| **法術**(id→特效、效果、面板) | `37` + `02` + `13`(Get_EasyMagic) |
| 戰鬥公式(命中/暴擊/傷害/**成長**) | `02` §4 + `27` + `internal/battle/growth.go` |
| 敵/NPC **AI** 決策 | `11` |
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
| 重製架構 / 三平台 / 工作拆解 | `21` + `30` + `22`(技術驗證) |
| 字型現代化(UTF-8/TTF) | `18` |
| 劇本/事件系統設計(節點圖/可擴展 DSL) | `19` + `29` |
| **試玩落差清單**(結束回合/武器射程/法術/狀態欄/對話框) | **`51`**(最新)+ `42` + `44` |
| 打包(AppImage/Win/macOS) | `41` |
| 編輯器設計 | `38` |
| 可行性 / 第一性原理 | `20` |

### 工具 / 方法
| 我想做… | 查 |
|---|---|
| **dosbox-x debugger**(建置/BP trace/dump/BPLM 判死) | **`48`** |
| Call-graph 反組譯方法紀錄 | `24` |
| 當年開發工具考證 | `04` |
| 「1995 年怎麼做這遊戲」總覽 | `15` |

### 專案管理
| | 查 |
|---|---|
| 這輪做什麼 / 待辦 | `91`(worklist) |
| 逐輪反思 / 踩雷 | `99`(reflections) |
| 計畫 | `90` |

---

## §B 完整文件清單(依編號)

`01`容器/資產 · `02`遊戲數值(青衫) · `03`EXE表/結構 · `04`開發工具考證 · `05`圖像RLE · `06`動畫AFM ·
`07`XMIDI · `08`文本/字型 · `09`劇情/對話 · `10`sprite著色/狀態 · `11`AI · `12`音樂播放/場景 ·
`13`戰場選單 · `14`文本控制碼 · `15`1995怎麼做(總覽) · `16`音色合成 · `17`擴充可行性 · `18`字型現代化 ·
`19`劇本系統設計 · `20`第一性原理可行性 · `21`Go/Ebiten架構 · `22`技術驗證 · `23`開機/標題/過場流程 ·
`24`callgraph紀錄 · `25`戰場事件系統 · `26`逐關事件handler · `27`戰鬥規則+驗證清單 · `28`全30關目標 ·
`29`可擴展事件系統 · `30`工作拆解WBS · `31`FDICON地圖sprite · `32`物品/戰鬥數值 · `35`戰鬥演出繪圖 ·
`36`SFX · `37`法術特效對映 · `38`編輯器設計 · `39`ANI.DAT AFM · `40`說話者→頭像查表 · `41`打包 ·
`42`RE-vs-remake稽核 · `44`第一章對照 · `45`職業名錯位 · `46`第一章開場時間軸 · `47`序章handler全轉錄 ·
`48`dosbox-x debugger · `49`char id→角色名 · `50`**過場機制總表(唯一主檔)** · `51`試玩落差R2 · `52`戰場分鏡+兩套系統 · `53`START→ch1回合1 RE來源表 · `54`acting實測原始記錄(機制見`50`) · `55`草地走位量測 ·
`90`計畫 · `91`worklist · `99`反思

(缺號 33/34/43 = 曾用後併入他篇或未建。)

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
- **`dosbox_dump/acting_decoded/acting_decoded_throne.txt`** — **74 筆演出(acting)完整解碼**(每 id 的幀/拍數/單位/pose)。
  ⚠ **這份已解出但 remake 尚未接用**(BeatRunner `act` 目前只是方向切換近似)——接上它才能忠實重現原版演出。
- `dosbox_dump/out/*.bin` — 單位陣列槽 dump、acting 資源原始 bytes、鏡頭/單位數快照(`47`/`48` 實測證據)
- `extracted/maps/` `extracted/images/` `extracted/story/` 等 — 解出的地圖/圖/劇情文本(玩家自備原版跑 tools 解)

---

## §D 還原 chNN.json 的工作流(核心目標)

remake 每關的劇本檔 `remake/assets/scenarios/chNN.json` = **事件骨架 + 對白文字**兩者合成:

1. **事件骨架**(何時發生什麼)← **青衫攻略**(`references/text/fd2-walkthrough-index.md`,每關的回合時機/增援/
   加入/勝敗,ground truth)+ `battle_events.json`(反組譯的 handler 條件,交叉驗證)。
2. **對白文字**← FDTXT 轉錄(`extracted/story/`,全 1533 句)。
3. **範本**:**`ch01.json` 是 Fable 5 RE 建立的黃金範本**——其餘 ch02~30 照它的結構(events: trigger/when/do,
   dialogue speaker+text)仿製。系統 A(開場過場)進 cutscene 節點;系統 B(戰鬥中事件)進 scenario events(doc52)。

## 標註慣例
- **[已驗證]** 原版實檔/反組譯/dosbox 交叉確認 · **[假設]** 待後輪確認/推翻 · **[攻略]** 青衫玩家觀測(實作以反組譯為準)

## 原始素材(不入 git,不散布)
- 遊戲本體 `org_game/炎龍騎士團/FLAME2/` · 攻略鏡像 `references/` · 原版錄影 `video/`(ground truth)
