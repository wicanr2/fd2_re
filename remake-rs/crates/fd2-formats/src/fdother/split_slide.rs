//! 移植自 Go `split_slide.go`：`0x1a30b` 進入戰鬥時 `0x1f42d` 的五次左右滑入。

use super::{FdotherError, LMI1Entry, Result};

/// 從 `0x1a30b` 的進入戰鬥索引編排抵達的一次已驗證 `0x1f42d`。封存圖格與任何玩家看得到的
/// 標籤分開：原生程式碼只證明這些索引放置，不證明是整備選擇視窗的動畫。
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeBattleEntrySplitSlideStep {
    pub offset: i64,
    pub left_x: i64,
    pub left_y: i64,
    pub right_x: i64,
    pub right_y: i64,
}

/// 回傳 `0x1f1cc` 使用、`0x1f30a` 反向使用的五次順序。座標在 456 位元組寬的索引戰場畫面。
pub fn native_battle_entry_split_slide_steps() -> Vec<NativeBattleEntrySplitSlideStep> {
    [100, 75, 50, 25, 0]
        .into_iter()
        .map(|offset| NativeBattleEntrySplitSlideStep {
            offset,
            left_x: 85 - offset,
            left_y: 82,
            right_x: 165 + offset,
            right_y: 81,
        })
        .collect()
}

/// 執行一次原生圖格處理。
pub fn blit_native_battle_entry_split_slide_cell(
    cell: &LMI1Entry,
    step: NativeBattleEntrySplitSlideStep,
    dst: &mut [u8],
    stride: i64,
) -> Result<()> {
    if stride != 456 {
        return Err(FdotherError::fixed("fdother: native split slide requires stride 456"));
    }
    cell.blit_at_clipped(dst, stride, step.left_x, step.left_y, false)?;
    cell.blit_at_clipped(dst, stride, step.right_x, step.right_y, false)
}

/// 執行已驗證的五次編排。呈現與還原仍是回呼，因為它們的原生 VGA／tick 實作歸呼叫端，
/// 尚未一般化。
///
/// Go 的回呼回傳任意 `error` 並原樣傳回；這裡讓呼叫端選錯誤型別 `E`，fdother 自己的錯誤
/// 以 `From` 轉入。Go 版拒收 nil 回呼；Rust 以型別排除 nil，該分支不存在。
pub fn run_native_battle_entry_split_slide<E: From<FdotherError>>(
    cell: &LMI1Entry,
    dst: &mut [u8],
    mut present: impl FnMut() -> std::result::Result<(), E>,
    mut restore: impl FnMut() -> std::result::Result<(), E>,
) -> std::result::Result<(), E> {
    for step in native_battle_entry_split_slide_steps() {
        blit_native_battle_entry_split_slide_cell(cell, step, dst, 456)?;
        present()?;
        restore()?;
    }
    Ok(())
}
