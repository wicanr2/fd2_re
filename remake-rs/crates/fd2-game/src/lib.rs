//! 《炎龍騎士團 2》重製引擎的遊戲狀態機（R4，見 `docs/goal/120-goal-rust-r4-game-state-split-20261114.md`）。
//!
//! 對應凍結 Go 版 `remake/cmd/fd2` 的流程部分：給定狀態、輸入與亂數就有固定結果，不依賴 Bevy。
//! 畫面、音訊與平台在 `fd2-app`（R5）。模組結構與移植狀態見 `PORTING.md`、`PORTING.json`。

// 函式 doc 逐字保留 Go 註解（含條列與縮排），不改寫成 Markdown 慣例。
#![allow(clippy::doc_lazy_continuation, clippy::doc_overindented_list_items)]
// 簽名逐一對應 Go 版：參數個數與 callback 型別照 Go 保留（同 fd2-rules 的 crate 層設定）。
#![allow(clippy::too_many_arguments, clippy::type_complexity)]
// 骨架專用（R4 階段 1）：函式本體是 todo!()，參數、欄位與匯入尚未使用。階段 2 全部移植後移除。
#![allow(unused, dead_code, unreachable_code, clippy::todo)]

pub mod game;

pub use game::Game;
