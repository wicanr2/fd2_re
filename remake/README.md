# 炎龍騎士團2 重製 — Go / Ebiten

一套以玩家自備原版資產為 oracle 的 Go/Ebiten FD2 重製引擎。設計與證據 gate 見
[`56` SDD](../docs/knowledge-base/56-fd2-remake-sdd.md)，工程待辦見 [`91` worklist](../docs/knowledge-base/91-worklist.md)。

> **狀態（2026-07-27）**：多個可測試垂直切片已完成（地圖／戰棋核心／對話／部分 action overlay、
> shop、persistent save、preparation/church），但**尚非完整 30 章原版等價通關**。未知的 native
> handler、renderer 與 campaign transition 維持 fail-closed；請不要把本 README 的切片列表解讀成 parity 宣稱。

目前進度與差距總覽請先讀根目錄 [`README.md`](../README.md) 的「目前狀態」表；可視化驗證產物在
[`docs/figures`](../docs/figures/)（目前戰場命中對照為
[`battle-impact-compare-20260810.png`](../docs/figures/battle-impact-compare-20260810.png)；
[`battle_restore.gif`](../docs/figures/battle_restore.gif) 僅是歷史分鏡，另有
[`preparation-remake.png`](../docs/figures/preparation-remake.png)）。

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
# 匯出戰鬥音效集合；包含 0x32999 固定使用的 FDOTHER #95
python3 tools/export_sfx.py --battle
```

以上是容器內命令，依根目錄 `AGENTS.md` 規則必須透過專案 Docker 工具鏈執行，
不可在主機 Python 安裝相依套件或直接執行。

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

macOS 走 GitHub Actions（`.github/workflows/build-macos.yml`）；最新
[run 33038019716](https://github.com/wicanr2/fd2_re/actions/runs/33038019716)
已在 `macos-14` 完成 arm64／amd64、`lipo` universal、bundle 自我檢查、DMG／tar.gz
與可攜 SHA-256 manifest artifact。Go CGO 跨編 macOS 需要 Apple SDK，不像 Linux／Windows 能在純 Linux
Docker 內完成；實體 Mac 的視窗、輸入、存檔、音訊與 Gatekeeper 仍須另行驗收。

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
