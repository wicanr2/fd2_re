//! 遊戲狀態機，對應凍結 Go 版 `remake/cmd/fd2`（`package main`）的流程部分（R4 骨架）。
//!
//! 每個 Go 檔 `x.go` 對應子模組 `game/x.rs`，各自寫 `impl Game`；只有含骨架項目的 Go 檔才有子模組。
//! 型別、常數與套件層變數已完整產生；函式只有階段 0 清單（第一章受控流程執行到的
//! {{SELECTED}} 個），本體是 `todo!("<file>.go:<FuncName>")`。同檔其餘函式在 `PORTING.json` 標 `r8`。
//! 骨架由 `tools/rust_r4_skeleton.sh` 產生，對應規則見
//! `docs/goal/120-goal-rust-r4-game-state-split-20261114.md`「階段 1 骨架的對應規則」，實際落地與偏離見
//! crate 根目錄的 `PORTING.md`。
//!
//! # 所有權
//!
//! - `Game` 保留單一型別，Go 的 {{GAME_FIELDS}} 個欄位全部保留（重設類函式幾乎碰到每個欄位）；只替換帶
//!   繪圖資源的欄位型別。
//! - **單位**：`*battle.Unit` 欄位 → `Option<UnitIdx>`，指向 `Game` 持有的 `battle::State::units`。
//!   `Game` 的方法與帶 `*battle.State` 參數的函式，`*battle.Unit` 參數也是 `UnitIdx`；其他函式收
//!   `&battle::Unit`／`&mut battle::Unit`（同 R2、R3）。容器裡的 `*battle.Unit` 是 `UnitIdx`。
//! - **畫面**：`*ebiten.Image`（含容器元素）→ `Option<Arc<Picture>>`；要畫上去的參數收 `&mut Picture`，
//!   只讀的收 `&Arc<Picture>`（可以直接存進狀態）。演出工作保存索引色畫格（`IndexedFrame` 或 `Vec<u8>`）
//!   與調色盤；GPU 貼圖由 `fd2-app` 建立。
//! - **延續閉包**：Go 的 func 欄位捕捉 `g`，Rust 的第一個參數是 `&mut Game`。`then`／`after`／`rollback`
//!   這類工作結束時只呼叫一次的延續是 `Option<Box<dyn FnOnce(&mut Game, ..) + Send>>`，其餘（觀察者、
//!   交易、時鐘注入）是 `FnMut`。呼叫前先 `take()` 出來，避免同時借用 `Game`。函式參數的 func 是
//!   `&mut dyn FnMut(..)`。
//! - **其他指標**：結構欄位的指標是 `Option<T>`（遞迴型別用 `Option<Box<T>>`）；參數的指標是 `&T`／`&mut T`，
//!   Go 會檢查 nil 的包成 `Option`。指標回傳值若指向接收者的狀態，回傳借用；新建的回傳擁有的值。
//!
//! # 其他型別決定
//!
//! - Go `int` → `i64`；會溢位繞回的運算用 `wrapping_*`。具名整數型別 → 單欄 newtype，常數是模組層 `const`。
//! - Go `map` → `BTreeMap`，`map[K]struct{}` → `BTreeSet`；Go 的 map 迭代順序隨機、Rust 有序。
//! - Go `error` → [`GameError`]（`support.rs`）；`(T, error)` → `Result<T>`。錯誤欄位是 `Option<GameError>`。
//! - Go `*rand.Rand` 參數 → `&mut dyn GoRand`，欄位 → `Option<battle::GoMathRand>`。
//! - Go `time.Time`／`time.Duration` → [`GameInstant`]／[`GoDuration`]；收 `now time.Time` 的函式照 Go
//!   保留參數，來源改成虛擬時鐘。
//! - Go `image.Point`／`image.Rectangle`／`image.RGBA` → `fd2_campaign::campaign` 的 `Point`／`Rectangle`／
//!   `Rgba`（R3 已定義）。
//! - 套件層變數：只讀的表格 → `LazyLock<T>`；Go 執行期會改寫的 → `LazyLock<RwLock<T>>`；`errors.New`
//!   哨兵 → `static GameError`。
//! - Go 的 JSON 標籤保留在欄位 doc（`Go 標籤：…`）。骨架不 derive serde，由擁有該檔的組在階段 2 加
//!   （`[]byte` 用 base64、nil 切片的 `null` 等，比照 `fd2_rules::battle::go_bytes`／`null_as_default`）。
//! - HD（`modern_*`、`player_poster*`、`player_toolbar*`）型別保留型別名，欄位只留第一章忠實模式執行到的
//!   函式用到的，其餘 R6。

mod support;

pub use support::*;

pub(crate) use crate::game_err;
pub(crate) use fd2_campaign::campaign::{Point, Rectangle, Rgba};
pub(crate) use fd2_campaign::{battlepresent, campaign, editorcanonical, ending, indexedmap, localization, musiccatalog};
pub(crate) use fd2_formats::save as fdsave;
pub(crate) use fd2_formats::{afm, archive, dato, fdicon, fdother, fdtxt, figani};
pub(crate) use fd2_rules::battle::{self, GoRand, UnitIdx};
pub(crate) use std::collections::{BTreeMap, BTreeSet};
pub(crate) use std::sync::{Arc, LazyLock, RwLock};
