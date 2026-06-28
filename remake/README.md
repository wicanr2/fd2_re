# 炎龍騎士團2 重製 — Go / Ebiten

一套程式碼跑**桌面 / 網頁(WASM)/ 手機**的 FD2 重製。設計見 [`../docs/knowledge-base/21-go-ebiten-remake-plan.md`](../docs/knowledge-base/21-go-ebiten-remake-plan.md)。

> **狀態**:MVP 垂直切片——載入序章戰場(tileset + 地圖)→ hi-res 渲染 → 方向鍵 / WASD / 觸控移動游標、相機跟隨。
> WASM 已可編譯(10.5MB)。後續見 `21` 里程碑(戰棋核心 → 文字 → 音訊 → ScenarioRunner → 跨平台)。

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

## 結構
```
cmd/fd2/main.go   進入點 + MVP(地圖渲染 + 游標 + 輸入)
web/              WASM harness(index.html + wasm_exec.js)
assets/           遊戲資產(gitignore,自備原版產生)
```

## 操作(MVP)
方向鍵 / WASD / 觸控點格 → 移動游標;相機跟隨。
