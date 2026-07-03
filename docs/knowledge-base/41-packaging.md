# 41 — 三平台打包(AppImage / Windows / macOS)

對應 `remake/`(Go/Ebiten)的散布版本。前置是「資產路徑解析層」——沒有這層,唯讀掛載
(AppImage squashfs)內既讀不到 `assets/`、也寫不了存檔,詳見 `38-editor-design.md` §6.5 的設計討論;
本篇是那份設計的**實作紀錄 + 驗證證據**。

## 1. 資產路徑解析層(`remake/cmd/fd2/assets.go`)

### 1.1 三層查找,不混層

唯讀資產(`assets/...` 相對路徑)依序:

1. **XDG 使用者覆蓋層** `$XDG_DATA_HOME/fd2_re/assets/...`(預設 `~/.local/share/fd2_re/assets/`)——
   玩家自備原版跑 `tools/export_engine_assets.py` 等工具產出的版權衍生素材(maps/sprites/music/portraits/
   figani/bg/tai/ui/sfx/fonts/title/tileset.png/spells.json 以外的其他非入庫檔/ANI.DAT)放這裡。
2. **AppImage 唯讀基底** `$APPDIR/assets/...`(僅在 `APPDIR` 環境變數有設時查;AppImage runtime
   執行時會自動設好)——只含**已入庫的原創內容**:`assets/scenarios/`、`assets/story/`、`assets/spells.json`
   (與 `remake/.gitignore` 的例外清單完全一致,見下方「版權資產分離」)。
3. **cwd 相對**(開發模式既有行為,`APPDIR` 未設、XDG 也無覆蓋時)——`go run`/`go build` 後直接在
   `remake/` 目錄執行,行為與改動前完全一致。

三層都沒有 → 回傳未改寫的 cwd 相對路徑,呼叫端的 `os.ReadFile` 自然得到「檔案不存在」,行為與改動前一致。

萬用字元批次載入(sprite/portrait/figani 逐檔 glob)用 `assetGlob`,同樣三層查找,但**第一層有命中
就整層採用,不同層的檔案不混拼**(避免玩家覆蓋一半、AppImage 基底補另一半這種不一致狀態)。

可寫檔(存檔 `fd2_save.json`、設定 `fd2_settings.json`)一律走 `userDataPath()` → `$XDG_DATA_HOME/fd2_re/`,
不再用 cwd(唯讀 mount 內無法寫入;這條規則不分 AppImage/開發模式,全平台統一)。

### 1.2 為什麼 macOS/Windows 也能沿用同一套邏輯

`userDataDir()` 用 `os.Getenv("XDG_DATA_HOME")` 找不到時 fallback 到 `$HOME/.local/share/`,而
`os.UserHomeDir()` 是 Go 標準庫的跨平台實作(Windows 對到 `%USERPROFILE%`,macOS 對到 `/Users/<user>`)。
沒有刻意做「macOS 用 `~/Library/Application Support`、Windows 用 `%APPDATA%`」這種平台慣例路徑——
先用最小可行的統一實作,三平台都能找到並寫入同一種目錄結構,以後有需要再依平台慣例拆。

### 1.3 驗證(非「應該可以」,實測)

用 dev binary 直接模擬「AppImage 唯讀基底 + XDG 覆蓋層」的資產分割(空 cwd、`APPDIR` 只放
`scenarios/story/spells.json`、其餘全部搬進 `~/.local/share/fd2_re/assets/`),跑 headless
screenshot(`FD2_SHOT`)比對:

```
$ cmp verify_dev.png verify_appdir_xdg_split.png
IDENTICAL
```

三層查找、可寫檔分離、玩家自備 ANI.DAT 走 XDG——全部路徑都在這次驗證裡被實際跑過一次,不是紙上設計。

## 2. Linux AppImage(`remake/packaging/build-appimage.sh`)

全程 docker(`fd2-build` image:`golang:1.22-bookworm` + ebiten X11/ALSA headers),不污染系統;
`linuxdeploy`/`appimagetool` 用 `--appimage-extract-and-run` 執行,不需要 host 有 FUSE。

流程:編譯 Linux amd64 binary → 組 AppDir(`AppRun` + `.desktop` + 圖示 + 只放已入庫資產的
`assets/`)→ `linuxdeploy` 掃 ELF 依賴補齊 `libXau`/`libXdmcp`/`libbsd`/`libmd` 等動態庫
(`libX11`/`libasound`/`libc`/`libm`/`libxcb` 是系統黑名單庫,不打包,任何目標發行版都有)→
`appimagetool` 封裝成 `FD2-x86_64.AppImage`。

### 2.1 AppRun

```bash
HERE="$(dirname "$(readlink -f "${0}")")"
export APPDIR="${APPDIR:-$HERE}"
exec "${APPDIR}/usr/bin/fd2" "$@"
```

AppImage runtime 正常執行時已經設好 `APPDIR`;這行是給「直接執行解壓後的 `AppDir/AppRun`」這種
繞過 squashfs 的手動測試場景補上,兩種執行方式行為一致。

### 2.2 圖示

`packaging/gen_icon.py` 用 PIL 畫的原創幾何圖形(深藍底+金邊+紅劍),不是從遊戲截圖裁的——
遊戲原始素材(title/sprite 等)是受著作權保護的抽取物,`remake/.gitignore` 本來就不讓它們入庫,
桌面圖示這種會被打包進散布物、也會被 commit 進 repo 的檔案更不能用抽取素材頂替。

### 2.3 驗證(三個情境都實測,非推論)

**A. 全新玩家(尚未跑 export 工具,XDG 空)**——headless 跑 `FD2-x86_64.AppImage`:不崩潰、
乾淨顯示「缺 assets/(tileset.png + map.json),用 tools/export_engine_assets.py 產生」提示並在
第 20 幀正常存下截圖證明。

> 這條路徑原本測不出來——`Draw()` 的 `g.m == nil` fallback 分支舊版沒呼叫 `saveShot`,補了這行
> (`cmd/fd2/main.go`,+3 行)截圖鉤子才對這個狀態也生效。順手修的既有缺口,不是本次任務的核心改動。

**B. 玩家已跑 export 工具(XDG 填滿完整資產)**——直接執行 `FD2-x86_64.AppImage --appimage-extract-and-run`
(headless Xvfb,空 cwd):

```
$ cmp verify_dev.png verify_appimage_direct.png
IDENTICAL
```

真正打包出來的 AppImage 檔案,從空目錄雙擊等效執行,渲染結果與原本 dev 模式(`go run` 在
`remake/` 目錄下)逐位元組相同——資產解析三層邏輯在真實打包產物上驗證通過。

### 2.4 已知限制

- 未做簽章/AppStream metadata(`appimagetool` 有警告,不影響執行)。
- `libXau`/`libXdmcp`/`libbsd`/`libmd` 是 X11 認證鏈的傳遞依賴,目標機器理論上都有,但沒有在
  非 Debian 系發行版(如 Arch/Fedora)上實測執行。

## 3. Windows(`remake/packaging/build-windows.sh`)

CGO 跨編:`packaging/Dockerfile.mingw` 建一個 `golang:1.22-bookworm` + `gcc-mingw-w64-x86-64` 的
image(`fd2-build-mingw`),`CGO_ENABLED=1 GOOS=windows GOARCH=amd64 CC=x86_64-w64-mingw32-gcc`。
`-ldflags="-H=windowsgui"` 讓正式版雙擊不彈 cmd 黑窗。

產物 `fd2-windows-x86_64.zip`:`fd2.exe` + 已入庫資產(scenarios/story/spells.json)。Windows 沒有
XDG 慣例,桌面版走 `assetPath()` 三層查找的**第 3 層(cwd 相對)**——玩家自備原版產出的資產放在
`fd2.exe` 旁的 `assets/` 資料夾即可,不強制走 `%USERPROFILE%\.local\share`(該路徑仍是存檔/設定
的落點,兩者不衝突)。

### 3.1 驗證(誠實揭露:編譯過,執行未在真機驗證)

- `file fd2.exe` 確認 `PE32+ executable (GUI) x86-64, for MS Windows`——交叉編譯產物格式正確。
- 嘗試用 Wine(headless Xvfb)smoke test:`wine fd2.exe` 掛起逾時(60–90 秒無回應,無錯誤輸出)。
  Wine + Xvfb + OpenGL(Ebiten Windows 後端走 win32+wgl)是已知脆弱組合,掛起不代表 exe 本身有問題,
  但也**不構成「跑得起來」的證據**——沒有 Windows 實機或穩定的 Wine GL 環境,無法排除兩者。
- 依 worklist 慣例如實標記:**已編譯,未實機測試**。下一步需要 Windows 實機(或 GitHub Actions
  `windows-latest` runner)跑一次 headless 截圖驗證,補齊這塊證據缺口。

## 4. macOS(評估,未實作)

### 4.1 為什麼不能純 docker 跨編

Go 的 CGO 需要目標平台的 C 工具鏈與 SDK headers;Ebiten 的 macOS 後端吃 Cocoa/OpenGL framework,
這些 header 只存在於 macOS SDK 內。跨平台編譯 macOS binary 理論上可行(`osxcross`),但:

- 需要合法取得並打包一份 macOS SDK(Apple SDK EULA 限制只能在 Apple 硬體上編譯 macOS 目標),
  法務風險不是「重」的問題,是「不該做」的問題。
- `osxcross` 建置本身也重(要抽 SDK、編一輪 cctools/clang wrapper),投入產出比很差。

### 4.2 建議路徑:GitHub Actions `macos-14` runner

不跨編,借用 GitHub 提供的 Apple Silicon runner(`macos-14`,免費額度每月 6 小時,足夠這種輕量
build)直接原生編譯。草稿見 `.github/workflows/build-macos.yml`:

- `GOARCH=arm64` 原生編一份、`GOARCH=amd64` + `CGO_CFLAGS="-arch x86_64"` 用系統 clang 的
  universal 工具鏈跨架構編另一份,`lipo -create` 合併成 universal binary(單一 `fd2.exe` 同時
  支援 Apple Silicon 與 Intel Mac,不用像 Windows/Linux 那樣拆兩個產物)。
- 因為 Ebiten macOS 後端只吃系統 framework、不依賴第三方 `.dylib`,不需要 SDL2/C++ 老遊戲那套
  `dylibbundler` 打包工序(對照 `mac-app-cross-pack` skill 的 SDL 案例複雜度低很多)。
- 產 `.dmg`(`hdiutil`,CI 上就是真 macOS,不必走「WSL mkisofs -hfs 土砲」那條路)+ `.tar.gz` 雙保險。
- 版權資產一樣不 ship,`.app` 內只放 `assets/scenarios`、`assets/story`、`assets/spells.json`;
  玩家資產一樣走 XDG fallback(見 §1.2)。

### 4.3 未驗證項目(誠實揭露)

這份 workflow **從未在真的 GitHub Actions 上跑過**(此輪工作沒有觸發 CI 的授權範圍),下列都是
待確認,不是既成結論:

- `CGO_CFLAGS="-arch x86_64"` 這種跨架構 CGO 編譯,ebiten 依賴的 `purego`/`oto`(音訊)是否真的
  乾淨過關——只是沿用 Go 官方文件記載的已知模式,沒有實測。
- `fd2.icns` 目前是用 PNG 佔位(workflow 裡直接複製 `fd2.png`),正式版要用 `iconutil` 轉真正的
  `.icns`(多解析度),`gen_icon.py` 產的原始 PNG 已備好,轉檔步驟待補。
- 未簽署 app 會被 Gatekeeper 擋,需要玩家自己 `xattr -dr com.apple.quarantine FD2.app`,
  README 待補這段安裝說明。

## 5. 版權資產分離(三平台一致)

| 內容 | 入庫(git) | 打包進散布物 | 玩家自備位置 |
|---|---|---|---|
| `assets/scenarios/*.json`(節點圖劇本) | ✅ | ✅ | — |
| `assets/story/*.json`(對白文本,版權已過期) | ✅ | ✅ | — |
| `assets/spells.json`(EXE dump 數值) | ✅ | ✅ | — |
| `assets/maps/`、`sprites/`、`music*/`、`portraits/`、`figani/`、`bg/`、`tai/`、`ui/`、`fonts/`、`title/`、`tileset.png`、`map.json`、`map0_units.json` | ❌ | ❌ | `$XDG_DATA_HOME/fd2_re/assets/`(玩家跑 `tools/export_engine_assets.py` 等工具產出) |
| `ANI.DAT`(開場過場,玩家原版檔案本身) | ❌ | ❌ | 同上,`assets/ANI.DAT` |

判準:**已入庫清單 = `remake/.gitignore` 的例外規則**,三個打包腳本(AppImage/Windows/macOS)
都從同一份清單複製,沒有各自維護一份「該打包什麼」的影子清單。

## 6. 檔案清單

```
remake/cmd/fd2/assets.go        資產路徑解析層(assetPath/assetGlob/userDataDir/userDataPath)
remake/packaging/
  AppRun                        AppImage 進入點
  fd2.desktop                   桌面項目
  gen_icon.py / fd2.png         原創圖示(PIL 產生,非抽取素材)
  build-appimage.sh             Linux AppImage 建置腳本(docker fd2-build)
  Dockerfile.mingw              Windows 跨編 docker image 定義
  build-windows.sh              Windows exe 建置腳本(docker fd2-build-mingw)
  dist/                         建置產物(gitignore,可重跑腳本重建)
.github/workflows/build-macos.yml   macOS universal binary CI 草稿(未跑過,見 §4.3)
```
