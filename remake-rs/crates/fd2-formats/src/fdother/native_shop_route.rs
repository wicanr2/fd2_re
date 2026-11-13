//! 移植自 Go `native_shop_route.go`。

/// 商店服務種類。Go 版是字串具名型別；Rust 用列舉，[`NativeShopServiceKind::as_str`] 回傳
/// Go 常數的字串值。
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
pub enum NativeShopServiceKind {
    Purchase,
    Sell,
    Equip,
    Transfer,
}

impl NativeShopServiceKind {
    /// Go 常數 `NativeShopPurchase` 等的字串值。
    pub fn as_str(self) -> &'static str {
        match self {
            NativeShopServiceKind::Purchase => "purchase",
            NativeShopServiceKind::Sell => "sell",
            NativeShopServiceKind::Equip => "equip",
            NativeShopServiceKind::Transfer => "inventory_transfer",
        }
    }
}

/// 保存 `0x2e341` 選出的資源、callee 與獨立證實的資料異動語意。
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct NativeShopServiceRoute {
    pub hub_variant: u8,
    pub resource_id: u8,
    pub selector: u8,
    pub callee: u32,
    pub kind: NativeShopServiceKind,
}

/// 對應 `0x2e341` 已證實的分支邊界：hub variant 3 與 5 選資源 29 與 63，其餘 variant 選資源
/// 12。已確認的四項 selector 對應四個 raw callee。只回傳資料，不觸發場景也不改戰役狀態。
///
/// Go 版回傳 `(route, ok)`，`ok` 為 false 時 route 是零值；Rust 以 `None` 表示。
pub fn resolve_native_shop_service_route(hub_variant: u8, selector: u8) -> Option<NativeShopServiceRoute> {
    let resource = match hub_variant {
        3 => 29,
        5 => 63,
        _ => 12,
    };
    const CALLEE_BY_SELECTOR: [u32; 4] = [0x2f0b0, 0x2f642, 0x2f883, 0x2f8ea];
    const KIND_BY_SELECTOR: [NativeShopServiceKind; 4] = [
        NativeShopServiceKind::Purchase,
        NativeShopServiceKind::Sell,
        NativeShopServiceKind::Equip,
        NativeShopServiceKind::Transfer,
    ];
    let index = selector as usize;
    if index >= CALLEE_BY_SELECTOR.len() {
        return None;
    }
    Some(NativeShopServiceRoute {
        hub_variant,
        resource_id: resource,
        selector,
        callee: CALLEE_BY_SELECTOR[index],
        kind: KIND_BY_SELECTOR[index],
    })
}

#[cfg(test)]
mod tests;
