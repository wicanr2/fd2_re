//! Go 來源：`remake/cmd/fd2/assets.go`（202 行）。
//!
//! assets.go — 資產路徑解析層(打包前置,doc38 §6.5)。
//!
//! 唯讀資產(assets/...)五層查找,先中先贏,不混層:
//!  1. XDG 使用者覆蓋層 $XDG_DATA_HOME/fd2_re/assets/...
//!     (doc38「使用者資料優先」原則:玩家編輯過的版本 / 版權衍生素材如 sprites·music·portraits·
//!     maps 皆放這裡,AppImage 只打包 scenarios/story 等原創內容,故這些目錄實務上只在此層出現)
//!  2. AppImage 唯讀基底 $APPDIR/assets/...(有設 APPDIR 才查;AppImage 執行期自動設)
//!  3. macOS app bundle 的 Contents/Resources 相對
//!  4. 執行檔所在目錄 相對(playfix #2:雙擊 / 從別的 cwd 用絕對路徑啟動 ./fd2-linux 時,
//!     cwd 不一定是 remake/,資產解不到會靜默跳過開場動畫直接進戰場;開發模式 cwd=remake/
//!     時這層與第4層同值,行為不變)
//!  5. cwd 與祖先目錄(開發模式；供 `go test` 在 cmd/fd2 package cwd 回到 remake/)
//!
//! 可寫檔(存檔/設定)一律走 $XDG_DATA_HOME/fd2_re/,不再用 cwd(唯讀 mount 內無法寫入)。
//!
//! 骨架：Go 檔的 7 個函式中產生 6 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// Go：`assets.go:29`
///
/// Go 會在執行期改寫這個套件層變數，Rust 以 `RwLock` 包住。
pub(crate) static EXE_DIR_CACHED: LazyLock<RwLock<String>> = LazyLock::new(|| RwLock::new(Default::default()));

/// Go：`assets.go:30`
///
/// Go 會在執行期改寫這個套件層變數，Rust 以 `RwLock` 包住。
pub(crate) static EXE_DIR_LOOKED: LazyLock<RwLock<bool>> = LazyLock::new(|| RwLock::new(Default::default()));

/// exeDir 回傳執行檔所在目錄(符號連結已解),取不到則回傳 ""。
///
/// Go：`assets.go:33`，`func exeDir() string`
pub(crate) fn exe_dir() -> String {
    todo!("assets.go:exeDir")
}

/// Go：`assets.go:49`
///
/// Go 會在執行期改寫這個套件層變數，Rust 以 `RwLock` 包住。
pub(crate) static USER_DATA_DIR_CACHED: LazyLock<RwLock<String>> = LazyLock::new(|| RwLock::new(Default::default()));

/// userDataDir 回傳(並確保存在)可寫使用者資料夾:$XDG_DATA_HOME/fd2_re/,預設 ~/.local/share/fd2_re/。
///
/// Go：`assets.go:52`，`func userDataDir() string`
pub(crate) fn user_data_dir() -> String {
    todo!("assets.go:userDataDir")
}

/// userDataPath 回傳可寫檔的完整路徑(存檔/設定/編輯器輸出)。
///
/// Go：`assets.go:71`，`func userDataPath(name string) string`
pub(crate) fn user_data_path(name: &str) -> String {
    todo!("assets.go:userDataPath")
}

/// Go：`assets.go:75`，`func fileExists(p string) bool`
pub(crate) fn file_exists(p: &str) -> bool {
    todo!("assets.go:fileExists")
}

/// macBundleResourceDir 回傳標準 .app bundle 的 Resources 目錄候選。
/// 只辨識 Contents/MacOS 這個結構；實際資產仍須由呼叫端以存在性檢查承認。
///
/// Go：`assets.go:82`，`func macBundleResourceDir() string`
pub(crate) fn mac_bundle_resource_dir() -> String {
    todo!("assets.go:macBundleResourceDir")
}

/// assetPath 解析一個唯讀資產相對路徑(如 "assets/map.json" 或 "assets/maps/map3" 目錄)。
/// 絕對路徑(系統字型等)原樣回傳。五層都沒有 → 回傳未改寫的 cwd 相對路徑,
/// 呼叫端的 os.ReadFile 自然得到「檔案不存在」錯誤,行為與改動前一致。
///
/// Go：`assets.go:97`，`func assetPath(rel string) string`
pub(crate) fn asset_path(rel: &str) -> String {
    todo!("assets.go:assetPath")
}
