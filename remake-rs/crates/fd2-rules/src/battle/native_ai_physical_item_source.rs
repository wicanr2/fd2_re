//! Go 來源：`remake/internal/battle/native_ai_physical_item_source.go`（77 行）。

use super::*;

/// NativeAIPhysicalItemSource 保存 0x14237 actor 端已證實的物品來源。
/// Slot／ItemID／RawGeometryByte0B／RawGeometryByte0C 都保留原始欄位角色；
/// 不把兩個幾何 byte 命名成射程、範圍或陣營。
///
/// Go：`native_ai_physical_item_source.go:8`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeAIPhysicalItemSource {
    pub slot: i64,
    pub item_id: u8,
    pub row: Vec<u8>,
    pub raw_geometry_byte0_b: u8,
    pub raw_geometry_byte0_c: u8,
}

/// ResolveNativeAIPhysicalItemSource 重播 0x14288..0x142BE 的無副作用來源鏈：
///
///     0x1B83D(unit, 0) → equipped low-item slot
///     0x1B722(unit, slot) → runtime record +0x0B + slot*2
///     0x4E56C(item) → 0x602AD + item*0x17
///
/// 它只回傳 detached item-row snapshot 與 0x142B2 讀取的兩個 raw byte，
/// 不產生 target geometry、不評分、不移動，也不接 NextAIPlan。缺少完整
/// runtime record 或 item table 時失敗即關閉；沒有符合的 equipped low item
/// 則回傳 found=false，對應原版 0x1B83D 的 -1 路徑。
///
/// Go：`native_ai_physical_item_source.go:26`，`func ResolveNativeAIPhysicalItemSource( record []byte, itemRows []byte, ) (NativeAIPhysicalItemSource, bool, error)`
pub fn resolve_native_ai_physical_item_source(
    record: &[u8],
    item_rows: &[u8],
) -> Result<(NativeAIPhysicalItemSource, bool)> {
    todo!("battle/native_ai_physical_item_source.go:ResolveNativeAIPhysicalItemSource")
}
