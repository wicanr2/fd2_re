# 41 — 三平台打包(AppImage / Windows / macOS)

對應 `remake/`(Go/Ebiten)的散布版本。前置是「資產路徑解析層」——沒有這層,唯讀掛載
(AppImage squashfs)內既讀不到 `assets/`、也寫不了存檔,詳見 `38-editor-design.md` §6.5 的設計討論;
本篇是那份設計的**實作紀錄 + 驗證證據**。

> **2026-08-27 最新發行候選：**提交`a5bbaf3f`已由現有鎖定映像重新產出兩個
> 無原版資產封包。Linux AppImage為5,306,872 bytes，SHA-256
> `f085dc58ba021a118114ec33b1972e2500e61c51141b02f35d7cfbb720acadfb`；從空工作目錄
> 以唯讀XDG玩家資產實際啟動至`town_ch02`，見
> [`release-appimage-town-ch02.png`](../figures/release-appimage-town-ch02.png)及
> [狀態紀錄](../data/ui-traces/release-appimage-town-ch02.json)。Windows ZIP為
> 4,895,851 bytes，SHA-256
> `931397d8b60f85af976c68017d17f238a77ab904d4b00d8081e0267ceb0eb286`；內含GUI PE與
> 受版控scenario／story／spells共72項。兩包均逐名確認未夾帶原版EXE、DAT或存檔。
> Windows真機與macOS實體機操作仍是外部平台驗收，不由Linux交叉建置冒稱完成。
>
> 同一AppImage另在既有Ubuntu 24.04 Noble映像以相同空工作目錄／唯讀資產啟動，
> `town_ch02`畫面與Debian系工具映像達`AE=0/256000`。Windows ZIP則在Wine 9.0／
> Xvfb實際接收城鎮方向鍵＋確認並離開`town_ch02`，另一次F5於Windows路徑寫出
> 1,632-byte、node=`town_ch02`的可解析JSON存檔；見
> [`release-windows-wine-smoke.json`](../data/ui-traces/release-windows-wine-smoke.json)及
> [商店畫面](../figures/release-windows-wine-shop-ch02.png)。X11自動注入的方向對映不作
> 原版方向忠實度證據；Wine通過也不取代Windows真機。
>
> **macOS真實CI：**為避免建立正式發行標籤，以一次性測試分支觸發相同workflow。
> [run 32997009964](https://github.com/wicanr2/fd2_re/actions/runs/32997009964)於
> `macos-14`在2026-08-26 17:57:28Z至17:58:43Z完成，job `build`結論為success；
> arm64／amd64編譯、`lipo` universal、`.app`、DMG／tar.gz與artifact upload均由
> workflow成功閘門涵蓋。官方API記錄artifact `fd2-macos-universal` ID 9616984049、
> 19,299,516 bytes、未過期。無憑證下載端點回應401，故本輪不能在本機解包核對；
> 更不能以CI綠燈冒稱實體Mac啟動、輸入、存檔、音訊或Gatekeeper驗收。
>
> **macOS bundle 資產勘誤與執行驗證：**上述首次CI成功後發現 `.app` 把資料放在
> `Contents/Resources/assets`，舊 runtime 卻只會從 `Contents/MacOS/assets` 後備查找；因此
> 首次run不能證明戰役資料可讀。提交`54b4f0a3`依本篇§1契約加入bundle層與失敗即關閉的
> `FD2_PACKAGE_SELF_CHECK=1`。真實[run 32998426963](https://github.com/wicanr2/fd2_re/actions/runs/32998426963)
> 的job `98273613960`已在`macos-14`從空白cwd實際執行universal bundle binary，並成功解析
> `campaign_full.json`、`spells.json`、`ch01.json`與`ch30.json`；其後DMG／tar.gz及artifact
> 上傳亦成功。這關閉「封包內建資料無法解析」的實際缺陷，但仍不等於視窗、輸入、存檔、
> 音訊、簽章或實體Mac驗收。
>
> **具型別完整性加強：**run 32998426963只抽四份JSON語法；後續提交`b9ebf7ca`改以正式
> campaign／spell loader驗證全部轉場、36個唯一法術ID與所有非空`Node.Script`引用。
> 真實[run 32999934526](https://github.com/wicanr2/fd2_re/actions/runs/32999934526)、job
> `98278796187`再次從空白cwd成功執行bundle binary，之後的DMG／tar.gz與artifact上傳亦通過。
> 缺任一中段劇情檔、懸空轉場、法術筆數／ID錯誤都會在發行封包階段失敗，不再只靠首末章抽樣。

> **2026-08-25 現況勘誤：**本篇第2節的 AppImage 結果是歷史封包證據，不代表
> 每個新提交都已重新封裝。提交 `8e7683b1` 已以現行 `fd2-go-test-local` 在
> Docker／Xvfb 實際建置並啟動 Linux x86_64 執行檔至 `town_ch02`，擷取有效
> 320×200 城鎮畫面；執行檔 SHA-256 為
> `8d29122a162dec8f5873cfd183d8352f3b56d0891da714e3ac70c22f07f432ab`。
> 同一提交也以可重建的 `remake/packaging/Dockerfile.mingw` 產生 Windows x86_64
> GUI PE，SHA-256 為
> `665b2cd49823fb74a2531188b6867592a2ae53b178af0b350308a9d6b85a2a8e`；這只證明
> 交叉編譯與格式，不證明 Windows 能啟動。macOS 工作流程目前執行次數為零，
> 本輪因既有 GitHub 權杖失效而無法派送；這是驗證授權受阻，不是編譯失敗。
>
> **2026-08-26 Windows 封包勘誤：** `build-windows.sh` 已移除主機端
> `rm`／`cp`／`zip`、root-owned 輸出與事後遞迴 `chown`。目前由單一
> `docker run --rm --network none`、UID/GID `1000:1000`、3 GiB／2 CPU／384 pids
> 的容器完成清理、CGO 交叉編譯、資產組裝、PE 檢查、ZIP 與 SHA-256。實際產物
> `fd2-windows-x86_64.zip` 為 4,887,398 bytes，SHA-256
> `a8601c8e3e88b71054e4de1624f421f98ae539d8417c816f7722877294182d5f`；其中
> `fd2.exe` 為 `PE32+ executable (GUI) x86-64`。這關閉可重現 Windows 封包，
> 仍不等於 Windows 真機啟動／輸入／存檔／音訊驗收。macOS workflow 同日移除
> PNG 假 `.icns`，改由 `sips` 與 `iconutil` 產生原生圖示；尚待真實 CI 執行。
>
> **2026-08-26 Linux AppImage 現況：** 新增
> `packaging/Dockerfile.appimage`，將 `linuxdeploy`、`appimagetool`及Type-2 runtime
> 三個原本會在封包階段下載的檔案，改在image build階段取得並分別核對受版控
> SHA-256。`build-appimage.sh`目前只啟動單一無網路、指定UID/GID、3 GiB／2 CPU／
> 384 pids的一次性容器；主機不再清檔、複製、下載、root寫入或事後`chown`。
> 實際`FD2-x86_64.AppImage`為5,298,680 bytes，SHA-256
> `0a619a3a431c37ba73f790bf8817a9915cb08d8ff9b99ebd893142307a6c4e63`。
> 以唯讀玩家資產作XDG覆蓋層，從空工作目錄經`--appimage-extract-and-run`抵達
> `town_ch02`並產生203,943-byte PNG，產物雜湊複驗通過。這是Linux封包與啟動
> E1，不外推成其他Linux發行版、音訊或實機長程驗收。

## 1. 資產路徑解析層(`remake/cmd/fd2/assets.go`)

### 1.1 五層查找,不混層

唯讀資產(`assets/...` 相對路徑)依序:

1. **XDG 使用者覆蓋層** `$XDG_DATA_HOME/fd2_re/assets/...`(預設 `~/.local/share/fd2_re/assets/`)——
   玩家自備原版跑 `tools/export_engine_assets.py` 等工具產出的版權衍生素材(maps/sprites/music/portraits/
   figani/bg/tai/ui/sfx/fonts/title/tileset.png/spells.json 以外的其他非入庫檔/ANI.DAT)放這裡。
2. **AppImage 唯讀基底** `$APPDIR/assets/...`(僅在 `APPDIR` 環境變數有設時查;AppImage runtime
   執行時會自動設好)——只含**已入庫的原創內容**:`assets/scenarios/`、`assets/story/`、`assets/spells.json`
   (與 `remake/.gitignore` 的例外清單完全一致,見下方「版權資產分離」)。
3. **macOS app bundle 唯讀基底**：執行檔若位於 `FD2.app/Contents/MacOS/`，查找同一 bundle 的
   `Contents/Resources/assets/...`。只有候選實際存在時才採用；不可因路徑外觀像 `.app` 就吞掉
   後續開發模式。這一層位於 AppImage 後、執行檔同目錄前，且同樣禁止跨層拼接。
4. **執行檔所在目錄相對**：支援從任意 cwd 直接啟動可攜式 binary。
5. **cwd 與其祖先相對**(開發模式既有行為)——`go run`/`go build` 後直接在
   `remake/` 目錄執行,行為與改動前完全一致。

五層都沒有 → 回傳未改寫的 cwd 相對路徑,呼叫端的 `os.ReadFile` 自然得到「檔案不存在」,行為與改動前一致。

萬用字元批次載入(sprite/portrait/figani 逐檔 glob)用 `assetGlob`,同樣五層查找,但**第一層有命中
就整層採用,不同層的檔案不混拼**(避免玩家覆蓋一半、AppImage 基底補另一半這種不一致狀態)。

macOS 封裝必須在空白 cwd 實際執行 bundle 內 binary 的 `FD2_PACKAGE_SELF_CHECK=1` 模式。
檢查必須以正式具型別載入器驗證`campaign_full.json`的起點與全部轉場、確認`spells.json`
恰有ID 0..35各一筆，並逐一解析campaign所有非空`Node.Script`引用；因此不是只抽首／末章或
只呼叫`json.Valid`。這個檢查只驗證可散布資料的封裝與解析，不要求玩家自備的原版衍生素材；
任一檔缺失、重複ID、筆數錯誤、懸空轉場或JSON無效即以非零狀態失敗。

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

全程使用`fd2-build-appimage` image（`golang:1.22-bookworm`、Ebiten X11／ALSA
headers及雜湊鎖定AppImage工具），不污染主機；正式封包階段關閉網路。
`linuxdeploy`／`appimagetool`用`--appimage-extract-and-run`執行，不需要host FUSE。

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

**A. 全新玩家(尚未跑 export 工具,XDG 空)**——本段舊版曾宣稱可在第20幀存下
缺資產提示截圖；2026-08-26以現行封包重跑時，程式確實啟動並回報缺`map.json`，
但證據截圖鉤子依現行失敗即關閉契約拒絕`loadErr`畫面並正常結束。因此舊截圖宣稱
已撤回；它只證明缺資產路徑沒有崩潰，不再當作目前畫面證據。

**B. 玩家已跑 export 工具(XDG 填滿完整資產)**——2026-08-26由空工作目錄直接
執行實際`FD2-x86_64.AppImage --appimage-extract-and-run`，唯讀XDG覆蓋層載入
玩家資產，Xvfb抵達正式`town_ch02`並成功輸出203,943-byte PNG；AppImage
SHA-256由同一無網路容器再次驗證。

較早封包曾留下`cmp verify_dev.png verify_appimage_direct.png → IDENTICAL`的歷史
結果；本次沒有重跑該兩張舊檔比較，因此只宣稱目前封包能由空工作目錄消費XDG
覆蓋層並產生有效城鎮畫面，不把舊比較外推成目前提交的逐位元組證據。

### 2.4 已知限制

- 未做簽章/AppStream metadata(`appimagetool` 有警告,不影響執行)。
- `libXau`/`libXdmcp`/`libbsd`/`libmd` 是 X11 認證鏈的傳遞依賴,目標機器理論上都有,但沒有在
  非 Debian 系發行版(如 Arch/Fedora)上實測執行。

### 2.5 Linux 原生持續整合封包契約

Linux 發行物必須由 `ubuntu-latest` 的正式工作流程重建，不可只引用開發主機上
既有的 `packaging/dist`。工作流程先從受版控的 `Dockerfile.appimage` 建立鎖定工具
映像，再呼叫 `build-appimage.sh`；正式組包容器維持無網路、一次性及目前 runner
UID/GID 寫入。產物完成後，必須從空白工作目錄執行實際
`FD2-x86_64.AppImage --appimage-extract-and-run`。Ebiten 的 Linux 後端會在進入
`main` 前初始化 X11，因此此步必須由有界的 Xvfb 擁有程序，不可把缺少 `DISPLAY`
造成的初始化失敗誤判成封包資料缺陷；其後再以
`FD2_PACKAGE_SELF_CHECK=1` 具型別驗證全部戰役轉場、36 個唯一法術 ID 及所有劇情
引用。只有自我檢查退出碼為零，才可上傳 AppImage 與 SHA-256。
SHA-256 清單只記錄 `FD2-x86_64.AppImage` 相對檔名，不得洩漏建置容器內的
`/src/packaging/dist` 絕對路徑，並須能在下載後的同一目錄直接以 `sha256sum -c`
驗證。

這項契約證明的是 Linux x86_64 原生程序、AppImage 唯讀資產層與可散布資料完整性；
非互動 runner 不證明實體桌面的視窗、鍵盤、音訊、顯示伺服器相容性或長時間遊玩。
這些仍須與 Debian／Ubuntu 既有畫面證據及後續玩家回報分開記錄。

## 3. Windows(`remake/packaging/build-windows.sh`)

### 3.1 Windows 原生持續整合契約

Linux／MinGW容器仍是可重現的正式交叉封包來源，但Wine只能提供相容層煙霧測試。
GitHub Actions的`windows-latest`流程必須另以原生Go／MinGW工具鏈建置同一GUI程式，
只封入`assets/scenarios`、`assets/story`與`assets/spells.json`。組裝後須切換到空白工作目錄，
直接執行ZIP目錄內的`fd2.exe`並設定`FD2_PACKAGE_SELF_CHECK=1`；其驗收條件與macOS相同：
完整campaign轉場、36個唯一法術ID與全部story引用均通過。任何資料缺漏、程式非零退出、
封包失敗或artifact缺失都必須讓workflow失敗。此項只證明Windows原生程序與公開資料封包，
不取代實體玩家桌面的視窗、鍵盤、存檔、音訊或防毒軟體驗收。

CGO 跨編:`packaging/Dockerfile.mingw` 建一個 `golang:1.22-bookworm` + `gcc-mingw-w64-x86-64` 的
image(`fd2-build-mingw`),並預抓 Go modules、內建`file`／`zip`；正式封包容器關閉網路。
`CGO_ENABLED=1 GOOS=windows GOARCH=amd64 CC=x86_64-w64-mingw32-gcc`。
`-ldflags="-H=windowsgui"` 讓正式版雙擊不彈 cmd 黑窗。

產物 `fd2-windows-x86_64.zip`:`fd2.exe` + 已入庫資產(scenarios/story/spells.json)。Windows 沒有
XDG 慣例,桌面版走 `assetPath()` 三層查找的**第 3 層(cwd 相對)**——玩家自備原版產出的資產放在
`fd2.exe` 旁的 `assets/` 資料夾即可,不強制走 `%USERPROFILE%\.local\share`(該路徑仍是存檔/設定
的落點,兩者不衝突)。

### 3.2 驗證

- `file fd2.exe` 確認 `PE32+ executable (GUI) x86-64, for MS Windows`——交叉編譯產物格式正確。
- ZIP 由同一容器組裝並立即以 `sha256sum -c` 驗證；輸出檔與目錄抽查均為目前
  使用者 UID/GID，不再需要 root 或事後 `chown`。
- Wine 9.0／Xvfb後續已實際接收城鎮輸入、進入商店並寫出可解析F5存檔；這仍是相容層證據。
- 真實[Windows run 33000820996](https://github.com/wicanr2/fd2_re/actions/runs/33000820996)、
  job `98281879552`在`windows-latest`成功完成原生CGO建置、公開資產組裝，並從空白cwd
  以`Start-Process -Wait`執行GUI子系統`fd2.exe`的具型別自我檢查；ZIP、SHA-256及artifact
  上傳亦全部成功。首次run `33000572634`的建置已成功，但PowerShell直接呼叫GUI程式無法
  可靠取得退出狀態；改用明確process owner後通過，故該次失敗分類為驗證腳本問題。
- 尚未證實的是實體玩家桌面的視窗繪製、鍵盤、存檔、音訊、防毒軟體與長時間執行；
  Windows runner的非互動自我檢查不冒稱這些項目完成。

## 4. macOS（工作流程與 bundle 自我檢查已在真實 runner 成功）

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

### 4.3 已驗證與仍未驗證項目

真實run `32998426963`已證實雙架構CGO、`lipo`、原生`.icns`、bundle、空白cwd資料解析、
DMG／tar.gz及artifact上傳。仍未驗證的是實體Mac上的視窗、鍵盤、存檔與音訊，以及簽章／公證。
未簽署app可能被Gatekeeper阻擋；測試者可依README使用
`xattr -dr com.apple.quarantine FD2.app`，但這不是正式發行方案。

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
  Dockerfile.appimage           雜湊鎖定工具與相依項目的Linux封包映像
  build-appimage.sh             Linux AppImage單一無網路容器建置腳本
  Dockerfile.mingw              Windows 跨編 docker image 定義
  build-windows.sh              Windows exe 建置腳本(docker fd2-build-mingw)
  dist/                         建置產物(gitignore,可重跑腳本重建)
.github/workflows/build-macos.yml   macOS universal binary、bundle自我檢查與封包流程
```
