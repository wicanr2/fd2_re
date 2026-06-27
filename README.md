# 炎龍騎士團2 逆向工程與重製 · fd2_re

> 把 1995 年漢堂國際的經典戰棋 RPG《炎龍騎士團2》(Flame Dragon Knight 2) 徹底逆向，
> 用第一性原理還原規則與素材，並以兩套現代技術重製成可在**網頁與手機**上重新遊玩的版本。

這是一個**乾淨重寫**的逆向工程專案：以原版 DOS 程式作為「行為真值 oracle」抽取演算法、
破解原版資料格式，再手寫可公開、可維護、易中文化的引擎。原版程式與素材受著作權保護，
**不包含在本倉庫中**，玩家須自備合法原版。

## 為什麼這個專案值得做

《炎龍騎士團2》是 1990 年代華文單機 SRPG 的代表作之一，但只有 DOS 版、且**沒有 DOSBox debugger 級別的逆向資料**留存。
本專案從零開始，把它的封裝、資產、數值與規則一塊塊還原成公開知識，並重建成跨平台可玩版本。

## 第 1 輪成果：`.DAT` 容器格式全破

漢堂把幾乎所有資產打包成同一種極簡歸檔容器。第 1 輪即破解並驗證此格式，
寫成一支**通吃全部 12 個 `.DAT`** 的解包器，解出約 1000 個資源：

![.DAT 容器格式](docs/figures/container-format.png)

關鍵的已驗證發現：

| 項目 | 發現 |
|---|---|
| 容器 | `LLLLLL` magic + uint32 LE offset 目錄，`N = (offsets[0]-6)/4` |
| 圖像 | uint16 寬高開頭 — 標題 320×200、戰鬥背景 320×100、圖塊 24×24(VGA mode 13h) |
| 調色盤 | `FDOTHER` 第 0 資源 = 768B = 256 色 ×RGB(6-bit) |
| 文本 | `FDTXT` 兩層結構，資源內含 uint16 字串次目錄(中文化核心) |
| 地形 | `FDSHAP@0x2422E` 300 格 ×4B，與青衫攻略 modify2 **交叉吻合** ✓ |

```bash
# 解包任一 .DAT(需自備原版)
python3 tools/unpack_dat.py --list  FLAME2/TITLE.DAT
python3 tools/unpack_dat.py --all   FLAME2/  extracted/
```

## 第 2 輪成果：圖像 / 音樂 / 數值 / 工具考證

**圖像壓縮全破** — 還原出遊戲標題畫面與所有戰鬥背景：

![還原的標題畫面](docs/figures/title.png)

![還原的戰鬥背景](docs/figures/backgrounds.png)

- **RLE 壓縮**破解(`c≥0x80` literal / `c<0x80` run)+ VGA 256 色調色盤 → 約 125 張全幅圖可解。詳見 [`05-image-compression-format.md`](docs/knowledge-base/05-image-compression-format.md)。
- **音樂**確認為 Miles AIL 的 **XMIDI**，寫 `tools/xmi2mid.py` 轉出 15 首標準 MIDI(音符平衡、tempo 保留)。詳見 [`07-music-xmidi-format.md`](docs/knowledge-base/07-music-xmidi-format.md)。
- **EXE 數值表**全部 dump 並對攻略自驗通過(物品 215 / 法術 36 / 敵我單位 68 / 升級成長…)，連攻略原本缺的法術數值編號都還原了。見 [`docs/data/exe_tables/`](docs/data/exe_tables/)。

### 為台灣留一份技術紀念

逆向過程中，在動畫資料裡找到當年漢堂程式設計師自製工具的署名：

> **AFM — Animation File Manager Version 1.00　Copyright (C) 1993 Lo Yuan Tsung**

我們把破解出的每一項技術都整理成保存品質的文件，記錄 1995 年台灣團隊怎麼做一款 DOS 遊戲：
[開發工具考證](docs/knowledge-base/04-original-toolchain.md)、[圖像壓縮](docs/knowledge-base/05-image-compression-format.md)、[動畫機制](docs/knowledge-base/06-animation-format.md)、[音樂格式](docs/knowledge-base/07-music-xmidi-format.md)。

## 知識庫

逆向發現逐輪累積在 [`docs/knowledge-base/`](docs/knowledge-base/)，每輪同步更新、錯誤知識即時修正：

- [`00-index.md`](docs/knowledge-base/00-index.md) — 索引與標註慣例
- [`01-container-and-asset-formats.md`](docs/knowledge-base/01-container-and-asset-formats.md) — 容器與資產格式(第 1 輪)
- [`02-game-data-reference.md`](docs/knowledge-base/02-game-data-reference.md) — 裝備/法術/人物/公式(攻略萃取)
- [`03-exe-and-data-structures.md`](docs/knowledge-base/03-exe-and-data-structures.md) — EXE 資料表 offset 與單位/物品/法術/地圖結構
- [`90-re-plan.md`](docs/knowledge-base/90-re-plan.md) — 分階段逆向與重製計畫
- [`99-reflections-log.md`](docs/knowledge-base/99-reflections-log.md) — 逐輪反思日誌

## 重製目標(規劃中)

| 技術棧 | 目標平台 | 參考專案 |
|---|---|---|
| **SDL2 + C++** | 桌面(Linux/Windows/Mac) | 精訊《勇者鬥惡龍三》重製 |
| **Go / Ebiten** | Web(WASM) / Android | 《魔法大帝》重製 |

兩者共用同一份從原版還原的資料與規則。詳見 [`90-re-plan.md`](docs/knowledge-base/90-re-plan.md)。

## 倉庫結構

```
docs/knowledge-base/   逆向知識庫(逐輪累積)
docs/figures/          圖解(SVG + PNG)
tools/                 逆向工具(unpack_dat.py …)
references/README.md   青衫攻略致謝與連結(原文不轉載)
org_game/              原版本體與素材(.gitignore,不散布)
```

## 致謝與版權

- 遊戲《炎龍騎士團2》著作權屬**漢堂國際**。本專案僅供研究、保存與技術重製，原版資產不散布。
- 攻略知識庫取材自圖文攻略作者**青衫**：<https://chiuinan.github.io/game/game/intro/ch/c31/fd2/>。
  本倉庫不轉載其原文與圖片，僅做結構化數值整理並標註出處。
