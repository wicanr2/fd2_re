# 炎龍騎士團2 逆向工程知識庫 — 索引

> 《炎龍騎士團2》(Flame Dragon Knight 2)，漢堂國際 1995，DOS / DOS4GW 保護模式。
> 本知識庫由逆向工程逐輪累積。**每一輪的 RE 發現與反思都必須寫進這裡**；
> 後輪推翻前輪結論時，回去修正或刪除舊敘述，不堆積矛盾。

## 文件清單

| 檔案 | 內容 | 主要來源 |
|---|---|---|
| `01-container-and-asset-formats.md` | `.DAT` 容器格式、圖像/調色盤/文本/地形資產格式 | 第 1 輪 RE(實檔驗證) |
| `02-game-data-reference.md` | 裝備/法術/人物/屬性/公式一覽 | 青衫攻略萃取 |
| `03-exe-and-data-structures.md` | `FD2.EXE` 內資料表 offset、單位/物品/法術/地圖結構 | 青衫攻略 + 實檔驗證 |
| `04-original-toolchain.md` | **當年開發工具考證**(Watcom/DOS4GW/Miles AIL/AFM) | 第 2 輪 binary 取證 |
| `05-image-compression-format.md` | **圖像 RLE 壓縮格式**完整規格 | 第 2 輪 RE(視覺驗證) |
| `06-animation-format.md` | **動畫機制(AFM)**容器與幀結構 | 第 2–3 輪 RE |
| `07-music-xmidi-format.md` | **音樂 XMIDI 格式**與轉換 | 第 2 輪 RE(結構驗證) |
| `08-text-and-font-format.md` | **文本格式 + 自製 16×16 字型**(可還原中文) | 第 3 輪 RE(視覺驗證) |
| `09-story-and-dialogue.md` | 劇情/對話結構(說話者+控制碼)與抽取方法 | 第 3 輪 RE |
| `10-sprite-rendering-camp-and-state.md` | 敵/我方與狀態的動畫機制(解碼器變體/陣營著色/面向) | 第 3 輪 RE(反組譯) |
| `11-enemy-ai.md` | 戰場 AI:敵人/NPC 行動決策(目標評分/移動/地形) | 第 3 輪 RE(反組譯) |
| `12-music-playback-and-scene.md` | 音樂播放(Miles AIL/XMIDI 序列)與場景切換換曲 | 第 3 輪 RE(反組譯) |
| `13-battle-menu-system.md` | 戰場選單與行動系統(行動狀態機/選單游標/Get_EasyMagic) | 第 3 輪 RE(反組譯) |
| `14-text-control-codes.md` | 文本控制碼與對話框機制(開框/頭像/換行/翻頁) | 第 5 輪 RE(反組譯) |
| `16-audio-synthesis-soundfont-mt32.md` | 音色合成:SoundFont/MT-32/版本切換(含 MDI 驅動說明) | 評估+RE |
| `17-scenario-expansion-evaluation.md` | 擴充劇本/玩法(戰場/對話/商店/機制)可行性評估 | 評估 |
| `18-font-modernization-utf8-ttf-plan.md` | 字型現代化規劃:UTF-8 + TTF 渲染(重製) | 規劃 |
| `19-scenario-script-system-design.md` | 劇本/關卡腳本系統設計(可分支節點圖/敗北路線/擴充) | 設計 |
| `20-first-principles-feasibility.md` | 第一性原理:重製可行性再確認 | 確認 |
| `21-go-ebiten-remake-plan.md` | Go/Ebiten 重製架構規劃(桌面/Web/手機) | 規劃 |
| `15-how-fd2-was-made-1995.md` | **總覽:1995 年怎麼做出炎龍騎士團2**(綜合全紀錄) | 綜合 |
| `90-re-plan.md` | 分階段逆向與重製計畫 | 規劃 |
| `91-worklist.md` | 逐輪 worklist(依序執行) | 逐輪更新 |
| `99-reflections-log.md` | 逐輪反思日誌(lessons learned) | 逐輪更新 |

> `04`–`08` 同時是「1995 年台灣怎麼做遊戲」的技術保存紀錄。對應工具在 `tools/`。

## 標註慣例

- **[已驗證]** — 已在原版實檔 / 反組譯交叉確認。
- **[假設]** — 合理推論但尚未驗證，後輪需確認或推翻。
- **[攻略]** — 來自青衫攻略的玩家觀測值，實作時以反組譯為準。

## 原始素材位置(不納入 git，不散布)

- 遊戲本體：`org_game/炎龍騎士團/FLAME2/`(`FD2.EXE` + `*.DAT` 資產 + Miles 音效驅動)
- 攻略備份：`references/`(青衫圖文攻略離線鏡像)

## 工具

- `tools/unpack_dat.py` — 通用 `.DAT` 容器解包器(第 1 輪)。
