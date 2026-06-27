# 炎龍騎士團2 攻略（離線備份）

來源：<https://chiuinan.github.io/game/game/intro/ch/c31/fd2/>（圖文攻略作者：青衫）
擷取日期：2026-06-27

原站採用 frameset（左側選單 `menu.htm` + 右側內容）。本備份已完整抓下所有頁面與圖片，並另外轉成方便閱讀的 Markdown。

## 目錄結構

- `html/` — 原始 HTML 離線鏡像（含 36 張攻略圖 `fd2-*.jpg`）
  - `index.html` — 本地 frameset 入口，瀏覽器直接開啟即可瀏覽
  - `menu.htm` — 左側選單
  - 各內容頁見下表
- `text/` — 由 HTML 轉出的 Markdown（純文字快速閱讀，圖片以相對路徑連回 `html/`）

## 頁面對照

| 主題 | HTML | Markdown |
|---|---|---|
| 新手提示（操作、屬性說明） | `html/notes.htm` | `text/notes.md` |
| 遊戲攻略（逐章圖文，含 36 圖） | `html/fd2.htm` | `text/fd2.md` |
| 裝備列表 | `html/item.htm` | `text/item.md` |
| 法術列表 | `html/spell.htm` | `text/spell.md` |
| 人物屬性 | `html/list.htm` | `text/list.md` |
| 記憶體修改 | `html/memory.htm` | `text/memory.md` |
| 程式修改 | `html/modify1.htm` | `text/modify1.md` |
| 資訊修改 | `html/modify2.htm` | `text/modify2.md` |

## 備註

- 所有 HTML 為 UTF-8（含 BOM），中文可直接顯示。
- 圖片僅 `fd2.htm`（遊戲攻略）使用，共 36 張，已全數下載。
- Markdown 中的 `![圖](fd2-N.jpg)` 路徑相對於 `html/`，若要在 `text/` 內直接看圖，請改開 `html/fd2.htm`。
