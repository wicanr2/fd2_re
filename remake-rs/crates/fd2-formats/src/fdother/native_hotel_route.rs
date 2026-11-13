//! 移植自 Go `native_hotel_route.go`。

/// 保存 `0x2fc85` 的 raw selector／資源／callee 計畫。callee 保持位址層級，不命名成服務。
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeHotelServiceRoute {
    pub selector: u8,
    pub resource_id: u8,
    pub primary: u32,
    pub secondary: u32,
}

/// 對應 `0x2fc85` 已確認的四個分支：selector 0、1、2 呼叫 `0x2ffa5`、`0x30012`、`0x301f4`；
/// selector 3 先經 `0x19953` 讀整備輸入，再用 `0x197e5`。資源 13 在 selector 迴圈之前由
/// 同一家族載入。只回傳資料。
///
/// Go 版回傳 `(route, ok)`，`ok` 為 false 時 route 是零值；Rust 以 `None` 表示。
pub fn resolve_native_hotel_service_route(selector: u8) -> Option<NativeHotelServiceRoute> {
    let mut route = NativeHotelServiceRoute { selector, resource_id: 13, ..Default::default() };
    match selector {
        0 => route.primary = 0x2ffa5,
        1 => route.primary = 0x30012,
        2 => route.primary = 0x301f4,
        3 => (route.primary, route.secondary) = (0x19953, 0x197e5),
        _ => return None,
    }
    Some(route)
}

#[cfg(test)]
mod tests;
