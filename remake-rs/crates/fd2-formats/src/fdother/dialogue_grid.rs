//! 移植自 Go `dialogue_grid.go`：`0x168b6` 對白框格線的原始圖格放置計畫。

use super::{FdotherError, Result};

/// 一次 `0x168b6` -> `0x1685c` 的原始圖格複製。
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeDialogueGridPlacement {
    pub resource_index: i64,
    pub destination_byte: i64,
}

/// 依呼叫端明確給的引數轉寫 `0x168b6`。目的端偏移同時含 x 與 y 原點。
///
/// 呼叫端給的 y、rows 沒有上限；Go 的 int 乘加在極端值會繞回，這裡照做。
pub fn plan_native_dialogue_frame_grid(
    stride: i64,
    x: i64,
    y: i64,
    columns: i64,
    rows: i64,
) -> Result<Vec<NativeDialogueGridPlacement>> {
    if stride <= 0 || x < 0 || y < 0 || columns < 2 || rows < 2 || x >= stride || columns > (stride - x) / 16 {
        return Err(FdotherError::message(format!(
            "fdother: native dialogue grid {stride}/{x},{y}/{columns}x{rows} is invalid"
        )));
    }
    let (add, mul) = (i64::wrapping_add, i64::wrapping_mul);
    let base = add(mul(y, stride), x);
    let cell_row_stride = mul(16, stride);
    // Go 先 make(0, 12+2*(columns-2)+2*(rows-2)+columns*rows)；容量繞回成負值時 Go 執行期
    // panic（makeslice: cap out of range），這裡同樣 panic。正的容量只影響配置，不影響結果。
    let capacity = add(add(12 + 2 * (columns - 2), mul(2, rows - 2)), mul(columns, rows));
    assert!(capacity >= 0, "makeslice: cap out of range");
    let mut placements = Vec::with_capacity((capacity as usize).min(1 << 16));
    let mut put = |resource_index: i64, destination_byte: i64| {
        placements.push(NativeDialogueGridPlacement { resource_index, destination_byte });
    };
    let s3 = mul(3, stride);
    let bottom = add(add(base, s3), mul(rows, cell_row_stride));
    put(1, base);
    put(2, add(base, 3 + 16 * columns));
    put(3, bottom);
    put(4, add(bottom, 16 * columns + 3));
    put(5, add(base, 3));
    put(6, add(base, 19 + 16 * (columns - 2)));
    put(7, add(bottom, 3));
    put(8, add(bottom, 19 + 16 * (columns - 2)));
    put(14, add(base, s3));
    let right_middle = add(add(base, s3), 35 + 16 * (columns - 2));
    put(15, right_middle);
    let lower_middle = add(add(mul(rows - 2 + 1, cell_row_stride), base), s3);
    put(16, lower_middle);
    put(17, add(mul(rows - 2 + 1, cell_row_stride), right_middle));
    for column in 0..columns - 2 {
        put(9, add(base, 19 + 16 * column));
        put(12, add(bottom, 19 + 16 * column));
    }
    for row in 0..rows - 2 {
        let left = add(add(base, s3), mul(row + 1, cell_row_stride));
        put(10, left);
        put(11, add(left, 16 * columns + 3));
    }
    for row in 0..rows {
        for column in 0..columns {
            put(13, add(add(add(base, s3), 3), add(mul(row, cell_row_stride), 16 * column)));
        }
    }
    Ok(placements)
}
