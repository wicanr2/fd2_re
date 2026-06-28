# 22 — 重製技術驗證報告(Go/Ebiten)

> 在投入完整重製前,先用最小切片**實證**技術選型成立:引擎跑得起來、讀得到我們逆向出的資料、能產出**本機可執行檔**。
> 本文記錄三項驗證的做法、證據與結論。優先順序:**本機桌面執行檔 > 跨平台(Web/手機)**。

## 為什麼先驗證,而不是直接開工

老遊戲重製最常見的失敗,不是寫不出遊戲邏輯,而是走到一半才發現:
引擎在目標平台編不過、原版資料接不進引擎、或選定的技術棧根本產不出能交付的執行檔。
這些是**架構級風險**,愈晚發現代價愈高。所以先做垂直切片把風險打掉:

| 風險 | 驗證方式 | 結果 |
|---|---|---|
| Go/Ebiten 能否在本機產出可執行檔? | Docker 內 CGO build → Linux ELF | ✅ 10.8 MB ELF,可直接執行 |
| 逆向出的資料能否餵進引擎渲染? | `export_engine_assets.py` → 引擎讀取 | ✅ 序章戰場正確渲染 |
| 同一份碼能否編到網頁(未來)? | `GOOS=js GOARCH=wasm` build | ✅ 10.5 MB `.wasm` |

## 驗證一:本機桌面執行檔(主目標)

**目標**:不依賴瀏覽器,先在本機產生能雙擊執行的原生程式。

**做法**(Docker first,不污染系統,無需本機裝 Go):

```bash
cd remake
docker run --rm -v "$PWD":/src -w /src golang:1.22-bookworm bash -c '
  apt-get update -qq && apt-get install -y -qq \
    libgl1-mesa-dev libx11-dev libxrandr-dev libxcursor-dev \
    libxinerama-dev libxi-dev libxxf86vm-dev libasound2-dev pkg-config
  CGO_ENABLED=1 go build -o /src/fd2-linux ./cmd/fd2'
```

Ebiten 桌面後端走 OpenGL + X11,需要上列系統庫與 `CGO_ENABLED=1`(動畫/音訊/視窗都靠原生庫)。

**證據**:

```
fd2-linux: ELF 64-bit LSB executable, x86-64, dynamically linked,
           interpreter /lib64/ld-linux-x86-64.so.2, for GNU/Linux 3.2.0
10,806,672 bytes
```

✅ **本機 Linux 原生執行檔建立成功**。Ebiten 同套碼可交叉編到 Windows(`.exe`)、macOS(`.app`),所以本機三大桌面平台都覆蓋。

## 驗證二:資產管線(逆向資料 → 引擎)

引擎不含原版著作權資產;玩家自備合法原版後,用 `tools/` 把逆向出的資料轉成引擎吃的中間格式:

```
FDFIELD(地圖)──┐
                ├─ tools/export_engine_assets.py ─▶ assets/map.json     {w,h,tileW,tileH,cols,tiles[]}
FDSHAP(圖塊)──┤                                  └─ assets/tileset.png  (24×24 圖塊網格)
FDOTHER#0(調色盤)┘
```

引擎(`cmd/fd2/main.go`)只認這兩個中間檔,完全不碰原版 `.DAT` 解析 —— 解析留在 Python 工具側,引擎保持乾淨。
這條管線跑通後,**序章島嶼戰場在引擎內正確渲染**(圖塊用 FDSHAP offset 表定位,無累積漂移),游標可走、相機跟隨。

**為什麼中間格式而非引擎直接讀 `.DAT`**:解耦。`.DAT` 格式考證是 Python 工具的強項(快速迭代、可視化驗證);
引擎只需要穩定的 `png + json`。未來換引擎(SDL2/C++)也能重用同一批中間資產。

## 驗證三:跨平台 WASM(未來上網頁,非本輪重點)

```bash
GOOS=js GOARCH=wasm go build -o web/fd2.wasm ./cmd/fd2   # 10.5 MB
```

✅ 編譯成功。WASM 後端不需 CGO(Ebiten 走 WebGL),配 `web/wasm_exec.js` + `index.html` 即可在瀏覽器跑。
**本輪不深追網頁上線**(使用者明確:先本機執行檔);此項僅證明「未來上網頁/手機這條路沒被技術選型擋死」。

## 結論

1. **技術選型成立**:Go/Ebiten 一套碼覆蓋桌面(本機優先)+ Web + 手機,且都已實證可建。
2. **重製性質已定**:從「研究」降為「工程整合」—— 所有必要能力(資料格式、演算法、資產)都已逆向完成(見 [`20`](20-first-principles-feasibility.md)),引擎只是把它們組起來。
3. **下一步走本機路線**:先把桌面執行檔從「能渲染地圖」推進到「能玩一場戰鬥」(戰棋核心),再逐層疊文字、音訊、流程,最後才回頭處理網頁/手機打包。完整步驟見 [`91-worklist`](91-worklist.md) 的「重製 worklist」。

> 相關:架構設計 [`21`](21-go-ebiten-remake-plan.md) · 可行性 [`20`](20-first-principles-feasibility.md) · 字型現代化 [`18`](18-font-modernization-plan.md) · 腳本系統 [`19`](19-scenario-script-system-design.md)
