//! 《炎龍騎士團 2》重製引擎的決定性規則核心。
//!
//! 對應凍結 Go 版 `remake/internal/battle`（R2，見 `docs/goal/119-goal-rust-bevy-rewrite-20261113.md`）。
//! 目前是階段 1 骨架：型別、常數與表格已完整移植，函式本體多數仍是 `todo!()`。

// 階段 1 骨架：函式本體是 `todo!()`，參數與多數項目暫時沒有使用者。階段 2 每組移植完後
// 移除這組 allow，讓未使用與無法到達的警告重新生效。
#![allow(unused, dead_code, unreachable_code, clippy::todo)]
// 函式 doc 逐字保留 Go 註解（含條列與縮排），不改寫成 Markdown 慣例。
#![allow(clippy::doc_lazy_continuation, clippy::doc_overindented_list_items)]
// 簽名逐一對應 Go 版：參數個數與 callback 型別照 Go 保留。
#![allow(clippy::too_many_arguments, clippy::type_complexity)]

pub mod battle;
