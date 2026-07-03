# 炎龍騎士團2 重製 — Go / Ebiten

一套程式碼跑**桌面 / 網頁(WASM)/ 手機**的 FD2 重製。設計見 [`../docs/knowledge-base/21-go-ebiten-remake-plan.md`](../docs/knowledge-base/21-go-ebiten-remake-plan.md)。

> **狀態**:MVP 垂直切片——載入序章戰場(tileset + 地圖)→ hi-res 渲染 → 方向鍵 / WASD / 觸控移動游標、相機跟隨。
> **本機桌面執行檔已建成**(Linux ELF 10.8MB);WASM 也可編譯(10.5MB)。技術驗證見 [`22`](../docs/knowledge-base/22-remake-tech-validation.md)。
> 後續里程碑見 [`91-worklist`](../docs/knowledge-base/91-worklist.md) 重製區(M1 戰棋核心 → 文字 → 音訊 → 腳本 → 內容 → 跨平台打包)。

## 資產(玩家自備原版)

引擎**不含原版資產**(著作權)。放入合法原版後產生:
```bash
# 先解包(專案根目錄)
python3 tools/unpack_dat.py --all "org_game/炎龍騎士團/FLAME2" extracted/raw
# 產生引擎資產(序章戰場為例):tileset.png + map.json → remake/assets/
python3 tools/export_engine_assets.py \
    extracted/raw/FDFIELD/FDFIELD_000.bin \
    extracted/raw/FDSHAP/FDSHAP_000.bin \
    extracted/raw/FDOTHER/FDOTHER_000.bin \
    remake/assets 16
```

## 建置(Docker first,無需本機 Go)

### Web(WASM)
```bash
cd remake
docker run --rm -v "$PWD":/src -w /src golang:1.22-bookworm bash -c '
  cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" web/wasm_exec.js
  GOOS=js GOARCH=wasm go build -o web/fd2.wasm ./cmd/fd2'
cp -r assets web/assets        # 資產
cd web && python3 -m http.server 8770   # 開 http://localhost:8770
```

### 桌面(Linux,CGO)
```bash
docker run --rm -v "$PWD":/src -w /src golang:1.22-bookworm bash -c '
  apt-get update -qq && apt-get install -y -qq libgl1-mesa-dev libx11-dev libxrandr-dev \
    libxcursor-dev libxinerama-dev libxi-dev libxxf86vm-dev libasound2-dev pkg-config
  CGO_ENABLED=1 go build -o /src/fd2 ./cmd/fd2'
./fd2
```

### 手機(Android)
`ebitenmobile bind` → `.aar` → Gradle 打 APK(觸控已支援;見 `21`)。

## 打包散布版本

詳細設計/版權資產分離規則見 [`41-packaging`](../docs/knowledge-base/41-packaging.md)。

```bash
cd remake
./packaging/build-appimage.sh   # Linux AppImage(已驗證可 headless 執行)
./packaging/build-windows.sh    # Windows exe(已編譯,執行未在真機驗證)
```

macOS 走 GitHub Actions(`.github/workflows/build-macos.yml`,草稿未跑過)——Go CGO 跨編 macOS
需要 Apple SDK,不像 Linux/Windows 能純 docker 跨編,借用 `macos-14` runner 原生編譯。

打包內容只含**已入庫的原創資產**(`assets/scenarios/`、`assets/story/`、`assets/spells.json`);
`maps`/`sprites`/`music`/`portraits` 等 ROM 衍生素材是版權物,不隨散布物打包。玩家需自備原版跑
`tools/export_engine_assets.py` 等工具,把產出放到 `$XDG_DATA_HOME/fd2_re/assets/`
(預設 `~/.local/share/fd2_re/assets/`;Windows 走 exe 旁的 `assets/` 資料夾,見 `41` §3)。

## 結構
```
cmd/fd2/main.go     進入點 + MVP(地圖渲染 + 游標 + 輸入)
cmd/fd2/assets.go   資產路徑解析層(唯讀 XDG/APPDIR/cwd 三層 + 可寫 XDG 存檔/設定)
web/                WASM harness(index.html + wasm_exec.js)
assets/             遊戲資產(gitignore,自備原版產生;scenarios/story/spells.json 例外入庫)
packaging/          AppImage/Windows 打包腳本 + 素材(見 `41-packaging`)
```

## 操作(MVP)
方向鍵 / WASD / 觸控點格 → 移動游標;相機跟隨。
