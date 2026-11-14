//! 結局演出（終章對話、蒙太奇、片尾與合成器），對應凍結 Go 版 `remake/internal/ending`（R3 骨架）。
//!
//! 每個 Go 檔 `x.go` 對應子模組 `ending/x.rs`；型別、常數、表格已完整移植，函式只有簽名與 Go
//! 註解，本體是 `todo!("ending/<file>.go:<FuncName>")`。移植狀態與檔案相依見 crate 根目錄的
//! `PORTING.json`，分組與順序見 `PORTING.md`。
//!
//! 型別對應與 `campaign` 相同（見 `crate::campaign` 模組文件）：錯誤型別共用
//! [`crate::campaign::CampaignError`]，Go `image.Point`／`image.Rectangle`／`image.RGBA` 用
//! `crate::campaign::go_image`。`ending` 不持有任何戰場單位，只呼叫
//! `fd2_rules::battle::apply_native_runtime_equipment_recalc` 與 `campaign` 的兩個職業清單轉場函式。
//!
//! # 合成器的共享
//!
//! Go 的 `*IndexedCompositor` 同時被 `Player`、`MontageCycle`、`MontageTailPlayer`、`Phase0Player`
//! 持有並改寫：`cmd/fd2/ending_preview.go` 把 `p.player.Compositor` 傳給 `NewMontageCycle` 與
//! `NewMontageTailPlayer`，之後 `Player` 仍繼續用同一個合成器。Rust 以 [`SharedCompositor`]
//! （`Arc<Mutex<IndexedCompositor>>`）保存這個共享關係：
//!
//! - 結構欄位 `Compositor *IndexedCompositor` → `Option<SharedCompositor>`；
//! - 把合成器存進結構的建構函式（`NewPlayer`、`NewMontageCycle`、`NewMontageTailPlayer`、
//!   `NewPhase0Player`、`NewPhase0PlayerFromAssets`）收 `Option<SharedCompositor>`，呼叫端 `clone()`
//!   這個 `Arc`；
//! - 只在呼叫期間使用合成器的函式與 `IndexedCompositor` 自己的方法照一般規則收
//!   `&IndexedCompositor`／`&mut IndexedCompositor`，呼叫端先 `lock()`。
//!
//! 其他 Go 指標欄位（`*afm.Clip`、`*fdtxt.Font`、`*fdtxt.Strings`、`*figani.Animation` 等）只讀，
//! Rust 由結構自己擁有一份（`Option<T>`），和 `fd2_rules::battle` 的「Go 指標欄位 → `Option<T>`」相同。

// 函式 doc 逐字保留 Go 註解（含條列與縮排），不改寫成 Markdown 慣例。
#![allow(clippy::doc_lazy_continuation, clippy::doc_overindented_list_items)]
// 簽名逐一對應 Go 版：參數個數與 callback 型別照 Go 保留（同 fd2-rules 的 crate 層設定）。
#![allow(clippy::too_many_arguments, clippy::type_complexity)]

mod ext_deps;

pub(crate) use crate::campaign::{self, CampaignError, Result, campaign_err, go_image};
pub(crate) use fd2_formats::{afm, dato, fdicon, fdother, fdtxt, figani, save};
pub(crate) use fd2_rules::battle;
pub(crate) use std::collections::{BTreeMap, BTreeSet};
pub(crate) use std::fmt;

/// 多個播放器共用、可改寫的合成器（見模組文件「合成器的共享」）。
pub type SharedCompositor = std::sync::Arc<std::sync::Mutex<IndexedCompositor>>;

// ---- 每個 Go 檔一個子模組（自動產生；新增 Go 對應檔時依字母順序插入）----
mod compositor;
mod dato_frame;
mod dialogue_frame;
mod figure_fade;
mod finale;
mod montage;
mod montage_cycle;
mod montage_portrait;
mod montage_tail;
mod montage_tail_player;
mod native_dialogue;
mod player;
mod separated_fdfield;
mod timeline;

pub use compositor::*;
pub use dato_frame::*;
pub use dialogue_frame::*;
pub use figure_fade::*;
pub use finale::*;
pub use montage::*;
pub use montage_cycle::*;
pub use montage_portrait::*;
pub use montage_tail::*;
pub use montage_tail_player::*;
pub use native_dialogue::*;
pub use player::*;
pub(crate) use separated_fdfield::*;
pub use timeline::*;
