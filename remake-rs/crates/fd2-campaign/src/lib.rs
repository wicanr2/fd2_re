//! 《炎龍騎士團 2》重製引擎的戰役與內容層。
//!
//! 對應凍結 Go 版 `remake/internal` 的戰役與內容套件（R3，見
//! `docs/goal/119-goal-rust-bevy-rewrite-20261113.md`）。每個 Go 套件一個模組，同名；不依賴 Bevy。

pub mod battlepresent;
pub mod campaign;
pub mod editorcanonical;
pub mod ending;
mod go_filepath;
pub mod indexedmap;
pub mod localization;
pub mod musiccatalog;

#[cfg(test)]
#[allow(dead_code)] // 各模組共用的測試工具，不是每個模組都用到全部
mod test_support;
