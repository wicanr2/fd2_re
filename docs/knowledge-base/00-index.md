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
