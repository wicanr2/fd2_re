# 30 — remake 工作拆解(WBS:模組 / 資料管線 / 衝刺)

> 把里程碑(worklist M0–M6)+ 可擴展事件系統(doc 29)拆成**可指派、可驗收的工作包(WP)**。
> 每個 WP 標:輸入(依賴哪些 doc/資料)、產出(Go 檔/資料檔)、驗收、可否平行。
> 原則(承 doc 21/22):**本機桌面優先 → 能完整玩 → 才跨平台打包**;每個 WP 是「能編譯、能驗收」的切片;**遊戲差異全在資料**(引擎不為任一關寫死)。Docker first。

## 1. 引擎模組結構(Go packages,`remake/internal/`)

```
cmd/fd2/main.go        進入點 + Game(ebiten.Game)+ 場景切換
internal/
  assets/    載入 png/json/ogg(嵌入或檔案);資產目錄抽象          [無依賴]
  gfx/       hi-res 畫布、tile/sprite 繪製、相機、調色盤            → assets
  input/     鍵盤/WASD/觸控 → 抽象動作(已在 MVP)                 [無依賴]
  battle/    Unit/Terrain/BattleState、flood-fill 移動、攻擊結算、勝敗 → assets
  ai/        敵方決策(評分,對齊 doc 11)                          → battle
  text/      TTF 渲染、glyph、對話框、{{控制碼}}解析               → assets,gfx
  audio/     OGG 串流、場景→曲、SFX                                → assets
  event/     EventSystem:ConditionRegistry/ActionRegistry(doc 29) → battle
  scenario/  ScenarioRunner:campaign 節點/轉場/旗標(doc 19)      → event,battle,text,audio
  scene/     title/menu/battle/world/shop 場景 + 切換               → 全部
  save/      自有存檔(旗標/隊伍/進度,doc 27)                     → scenario
```
依賴方向單向向上(assets/input 最底層,scene 最上層)→ 好測試、好平行。

## 2. 資料管線(tools/ → remake/assets/;玩家自備原版產生,不入庫)

| 資產 | 來源 | 工具 | 狀態 |
|---|---|---|---|
| `tileset.png` + `map.json`(每戰場) | FDFIELD×FDSHAP | `export_engine_assets.py` | ✅ 有(序章) |
| `units.json`(各關 roster+數值) | FDFIELD roster + EXE 表(doc 03) | **待建** `export_units.py` | ⬜ |
| `script.json`(35 章對話,UTF-8) | FDTXT + glyph_map | **待建** `decode_story_text.py --script-json` | ⬜ |
| `portraits/`(DATO 頭像) | DATO.DAT | `decode_dato.py` | ✅ 有 |
| `music/*.ogg`(15 首 MT-32) | FDMUS→munt | `export_music_ogg.sh` | ✅ 有 |
| `campaign.json`(30 關流程+目標+事件) | battle_events(26)+目標(28)+章節跳表 | **待建** `gen_campaign.py` | ⬜ |
| `items/spells.json`(數值) | EXE 表 | `dump_exe_tables.py`(已 dump,轉 json) | 🟡 半 |

→ 「待建」三支工具(units/script/campaign)是資料層關鍵路徑。

## 3. 工作拆解(WBS)

### M1 — 戰棋核心(讓它「能玩一場序章戰鬥」)
| WP | 內容 | 輸入 | 產出 | 驗收 | 平行 |
|---|---|---|---|---|---|
| M1-1 | 資料模型 Unit/Terrain/BattleState | doc 03/27 | `battle/model.go` | 單元測試:建場、放單位 | — |
| M1-2 | `export_units.py` + units.json | doc 03,roster | `tools/`,`units.json` | 序章我方+敵方數值正確 | ∥ M1-1 |
| M1-3 | flood-fill 移動範圍 + 路徑 | doc 11 地形成本 | `battle/move.go` | 高亮可達格,扣地形成本 | → M1-1 |
| M1-4 | 戰場選單狀態機(移/攻/休/道具/結束) | doc 13 | `ui/battlemenu.go` | 游標/Enter/ESC,對齊原版 | → M1-1 |
| M1-5 | 攻擊結算(青衫公式) | doc 02 §4,27 | `battle/combat.go` | 物理/劍技/法術/命中/暴擊/經驗 | → M1-1 |
| M1-6 | 敵方 AI 回合 | doc 11(0x15140) | `ai/decide.go` | 評分選目標(擊殺×2) | → M1-3,5 |
| M1-7 | 回合推進(**無上限**)+ 勝敗 | doc 27§1,28 | `battle/turn.go` | 我方全動+敵AI→回合+1;殲滅/索爾死判定 | → M1-6 |
| M1-8 | headless 確定性回歸 | — | `battle/*_test.go` | 固定種子打一場結果可重現 | → M1-7 |

> **M1 = 第一個可玩衝刺**(見 §5)。完成 = 序章戰場能部署→移動→攻擊→敵 AI→分勝負。

### M2 — 文字 / 對話層
| WP | 內容 | 輸入 | 產出 | 驗收 |
|---|---|---|---|---|
| M2-1 | `decode_story_text.py --script-json` | doc 09/14,glyph_map | script.json | 35 章 UTF-8 + 控制碼結構 |
| M2-2 | TTF 文字渲染 | doc 18 | `text/render.go` | CJK 清晰(hi-res 拉畫布) |
| M2-3 | 對話框 UI(開框/翻頁/換行/頭像) | doc 14,01§7 | `text/dialogue.go` | 對齊原版控制碼語意 |
| M2-4 | `{{事件控制碼}}` 解析器 | doc 29§5 | `text/eventcode.go` | 對話中觸發事件動作 |

### M3 — 音訊層
| WP | 內容 | 輸入 | 產出 | 驗收 |
|---|---|---|---|---|
| M3-1 | OGG 串流播放 | music_ogg | `audio/player.go` | 15 首可播放/循環 |
| M3-2 | 場景→曲號 + 切換 | doc 12,曲表 0x51e63 | `audio/scene.go` | 標題/戰鬥/劇情正確切曲 |
| M3-3 | (選配)MT-32/SoundFont 切換 | doc 16 | — | 設定開關 |

### M4 — 腳本系統 / 事件 / 流程(doc 19+29 落地)
| WP | 內容 | 輸入 | 產出 | 驗收 |
|---|---|---|---|---|
| M4-1 | EventSystem(Condition/Action Registry) | doc 29 | `event/*.go` | 註冊 unit_state/roster_has/turn + spawn/win/flag… |
| M4-2 | `gen_campaign.py` → campaign.json | doc 26/28 | `tools/`,campaign.json | 原版 30 關目標+事件資料化 |
| M4-3 | ScenarioRunner 狀態機 | doc 19 | `scenario/runner.go` | 節點/轉場/旗標;序章→商店→下一關 |
| M4-4 | 商店節點 | doc 02(賣價75折) | `scene/shop.go` | 買賣/裝備 |
| M4-5 | 分支 + 敗北路線 | doc 19/29 | scenario | on_lose→敗北關,非 game over |
| M4-6 | 存檔/讀檔(自有格式) | doc 27 | `save/*.go` | 存旗標/隊伍/進度 |

### M5 — 內容完整化(原版可破關)
| WP | 內容 | 驗收 |
|---|---|---|
| M5-1 | 全 30 關資產(units/map/script/campaign) | 逐關可載入 |
| M5-2 | 全劇情接入 | 35 章對話正確 |
| M5-3 | 完整性盤點(對照原版,缺漏列冊) | doc(`83` 完整性>投報) |
| M5-4 | 正常玩法可達性驗證 | 無 debug hook 可從序章破到結局 |

### M6 — 跨平台打包
| WP | 內容 | 驗收 |
|---|---|---|
| M6-1 | 桌面打包(Win `.exe`/macOS `.app`/Linux AppImage) | 三平台可執行 |
| M6-2 | WASM 上網頁(資產載入完整化) | 瀏覽器可玩 |
| M6-3 | Android(`ebitenmobile`→aar→APK) | 手機觸控可玩 |
| M6-4 | 玩家向 README(圖文)+ 工程文件分離 | — |

## 4. 依賴與關鍵路徑

```
M0✅ ─ M1(戰棋核心)─┬─ M2(文字)─┐
                     ├─ M3(音訊)─┤
                     └─ M4(事件/腳本)┴─ M5(內容)─ M6(打包)
資料管線:export_units(M1-2) → script(M2-1) → gen_campaign(M4-2) 為資料關鍵路徑
```
- **關鍵路徑**:M1 → M4 → M5 → M6(玩法骨架 → 流程 → 內容 → 出貨)。
- **可平行**:M2、M3 與 M4 可同時做(各自獨立模組);資料工具(units/script/campaign)可提早並行建。

## 5. 第一個衝刺:M1「序章可玩戰鬥」(建議先做這個)

最小可玩切片,task 順序:
1. M1-2 `export_units.py` → 序章 units.json(我方 4 + 敵方 roster)
2. M1-1 Unit/BattleState + 載入 units.json 放到 map(接 MVP 的 map 渲染)
3. M1-4 戰場選單(移動/攻擊/待機)+ M1-3 flood-fill 移動
4. M1-5 攻擊結算(先物理,青衫公式)
5. M1-6 敵方 AI(簡版:接近+攻擊)→ M1-7 回合+勝敗
6. M1-8 headless 測試:固定種子打完序章

**衝刺驗收**:本機 `./fd2-linux` 開序章戰場,玩家部署→移動→攻擊→敵方 AI 動→分出勝負,回合無上限。

## 6. 風險與緩解

| 風險 | 緩解 |
|---|---|
| 資料工具(units/campaign)落後拖累引擎 | 工具與引擎並行;引擎先用手寫小 fixture 開發,工具好了再換真資料 |
| 戰鬥數值與原版有出入 | 青衫公式(doc 02)+ EXE 表(doc 03)雙來源;headless 回歸鎖定 |
| 事件系統過度設計 | M4 先做原版 30 關用得到的條件/動作,擴充項(doc 29)按需加 |
| 正常玩法不可達(soft-lock) | M5-4 無 debug hook 全程驗證(retro skill 踩雷) |
| 跨平台延後致 WASM/手機踩雷晚發現 | M0 已驗三平台可建(doc 22);M6 前每里程碑保持 WASM 可編譯 |

> 相關:doc 21(架構)· 22(技術驗證)· 19/29(腳本/事件)· 26/28(事件/關卡資料)· 02/03(數值)· worklist M0–M6。
