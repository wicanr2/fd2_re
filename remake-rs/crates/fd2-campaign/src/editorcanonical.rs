//! 編輯器 canonical bundle 的驗證與編譯，對應凍結 Go 版 `remake/internal/editorcanonical`（R3 骨架）。
//!
//! 每個 Go 檔 `x.go` 對應子模組 `editorcanonical/x.rs`；型別已完整移植，函式只有簽名與 Go 註解，
//! 本體是 `todo!("editorcanonical/<file>.go:<FuncName>")`。移植狀態見 crate 根目錄的 `PORTING.json`。
//!
//! 錯誤型別共用 [`crate::campaign::CampaignError`]。Go 正式程式沒有呼叫 `campaign` 的任何函式
//! （只有測試以 `campaign.Decode` 驗證編譯結果），所以和 `campaign` 只有測試期相依。
//!
//! # `json.RawMessage`
//!
//! Go `json.RawMessage` 保存原始 JSON 位元組：解碼時原樣保留，編碼時只做 `Compact`，物件鍵順序與
//! 數字字面值都不變。本 crate 的 `serde_json` 沒有開 `raw_value`（`RawValue`）與 `preserve_order`
//! 功能，`serde_json::Value` 會把物件鍵排序、把數字正規化，和 Go 的輸出位元組不同。骨架先以
//! [`GoRawMessage`] 別名指向 `serde_json::Value`，簽名都經由這個別名；階段 2 若要逐位元組對照
//! `CompileCampaignJSON` 的輸出，需由主代理決定是否在 `Cargo.toml` 開 `raw_value`（不改 Cargo.lock
//! 的套件版本），再把別名換成 `Box<serde_json::value::RawValue>`。見 `PORTING.md`「Go 寫法難對應處」。

// 骨架專用：函式本體是 todo!()，參數與匯入尚未使用。階段 2 全部移植後移除。
#![allow(unused, dead_code, unreachable_code, clippy::todo)]

mod ext_deps;

pub(crate) use crate::campaign::{CampaignError, Result, campaign_err};
pub(crate) use std::collections::{BTreeMap, BTreeSet};
pub(crate) use std::fmt;

/// Go `encoding/json.RawMessage` 的骨架對應（見模組文件）。
pub type GoRawMessage = serde_json::Value;

// ---- 每個 Go 檔一個子模組（自動產生；新增 Go 對應檔時依字母順序插入）----
mod bundle;
mod compile;

pub use bundle::*;
pub use compile::*;
