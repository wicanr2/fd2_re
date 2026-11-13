//! 《炎龍騎士團 2》重製引擎的決定性規則核心。
//!
//! 對應凍結 Go 版 `remake/internal/battle`（R2，見 `docs/goal/119-goal-rust-bevy-rewrite-20261113.md`）。
//! 函式與測試逐一移植自 Go 版，對照表見 crate 根目錄的 `PORTING.json`。

// 函式 doc 逐字保留 Go 註解（含條列與縮排），不改寫成 Markdown 慣例。
#![allow(clippy::doc_lazy_continuation, clippy::doc_overindented_list_items)]
// 簽名逐一對應 Go 版：參數個數與 callback 型別照 Go 保留。
#![allow(clippy::too_many_arguments, clippy::type_complexity)]

pub mod battle;

#[cfg(test)]
mod test_support;
