//! `editorcanonical` 用到的本 crate `campaign` 項目清單。沒有對應的 Go 檔。
//!
//! Go 正式程式不呼叫 `campaign`；測試以 `campaign.Decode`／`campaign.Load` 驗證編譯結果，
//! 這裡在 `cfg(test)` 下確認對應的 Rust 函式存在（2026-11-14 以 `go/types` 列出）。

#[cfg(test)]
#[rustfmt::skip]
#[allow(unused_imports)]
mod campaign_test_items {
    use crate::campaign::{
        decode, // func campaign.Decode
        load, // func campaign.Load
    };
}

// 以上是 Go 測試檔另外用到的外部項目（正式程式已用到的不重列）；`cfg(test)` 下編譯檢查。

// 方法（掃描 Rust 原始碼時確認存在；`use` 無法引入方法）：

// 找不到 Rust 對應（缺口，見 PORTING.md）：
