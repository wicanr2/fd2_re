//! 單位索引與 callback 包裝。沒有對應的 Go 檔；設計說明見 `battle` 模組文件。

use std::sync::Arc;

use super::*;

/// 單位參照：`State::units` 裡的位置（自由函式則是該函式 `units` 參數裡的位置）。
///
/// Go 版以 `*Unit` 指標表示；`units` 在戰鬥中只 append 不刪除，所以索引在整場戰鬥中穩定。
pub type UnitIdx = usize;

/// Go func 欄位與可改寫全域 func 變數的包裝。
///
/// Go 的 func 值可以複製，也可以在多個結構間共用，所以用 `Arc`；`Clone` 只複製參照，與 Go 相同。
/// Go 的 func 不能比較相等，含 callback 的結構因此不 derive `PartialEq`。
pub struct Callback<F: ?Sized>(pub Arc<F>);

impl<F: ?Sized> Callback<F> {
    /// 由 `Arc` 包好的閉包建立。
    pub fn new(f: Arc<F>) -> Self {
        Self(f)
    }
}

impl<F: ?Sized> Clone for Callback<F> {
    fn clone(&self) -> Self {
        Self(Arc::clone(&self.0))
    }
}

impl<F: ?Sized> fmt::Debug for Callback<F> {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        f.write_str("Callback(..)")
    }
}

impl<F: ?Sized> std::ops::Deref for Callback<F> {
    type Target = F;

    fn deref(&self) -> &F {
        &self.0
    }
}
