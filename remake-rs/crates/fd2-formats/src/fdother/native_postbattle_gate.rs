//! 移植自 Go `native_postbattle_gate.go`。

use super::native_flow_error::at;

/// 保存原版戰後 hub 閘門選出的位址層級分支。`callee` 刻意是位址，不是正規化的
/// 城鎮／商店／教會標籤。欄位公開，呼叫端可自行組出任意 route 交給
/// [`resolve_native_postbattle_outcome`] 驗證。
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativePostbattleRoute {
    pub selector: u8,
    pub callee: u32,
    pub preparation: bool,
    pub direct_preparation: bool,
}

/// 保存 `0x2cad7` 是在內部迴圈還是把 raw 值回傳給呼叫端。`return_value` 只有在 `repeat`
/// 為 false 時才有意義；這層不把 raw 0/1 命名成場景或戰役結果。
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativePostbattleOutcome {
    pub repeat: bool,
    pub return_value: i64,
}

/// 對應已證實的 `0x2cad7/0x2d093` 邊界。`0x526b9` 項目非零時，在讀 hub 選項之前就選
/// 整備路徑。可選擇的 hub 中，選項 0、1/3、4 派發到還原出的 raw 呼叫端；選項 2 是
/// 存檔／確認→`0x318ad` 的整備路徑。未知索引／選項一律拒收，不呼叫任何 callee。
///
/// Go 版回傳 `(route, ok)`，`ok` 為 false 時 route 是零值；Rust 以 `None` 表示。
pub fn resolve_native_postbattle_route(
    chapter_index: i64,
    gate_table: &[u8],
    selector: u8,
) -> Option<NativePostbattleRoute> {
    if chapter_index < 0 || chapter_index >= gate_table.len() as i64 {
        return None;
    }
    if gate_table[at(chapter_index)] != 0 {
        return Some(NativePostbattleRoute { selector, callee: 0x318ad, preparation: true, direct_preparation: true });
    }
    let route =
        |callee, preparation| NativePostbattleRoute { selector, callee, preparation, direct_preparation: false };
    match selector {
        0 => Some(route(0x2fc85, false)),
        1 | 3 => Some(route(0x2e341, false)),
        2 => Some(route(0x318ad, true)),
        4 => Some(route(0x3072f, false)),
        _ => None,
    }
}

/// 對應選定的 raw callee 回傳之後的 `0x2cccc..0x2ccff` 與 `0x2cf3d..0x2cf6a`。
///
/// callee 回傳 0 時重複目前的直接整備提示或城鎮 hub。直接整備或城鎮 selector 2 回傳非零時
/// 回傳 raw 0；其他城鎮 selector 回傳非零時回傳 raw 1。這些 raw 值的意義屬於外層
/// `0x25de5` 呼叫端。
pub fn resolve_native_postbattle_outcome(
    route: NativePostbattleRoute,
    callee_result: i64,
) -> Option<NativePostbattleOutcome> {
    if route.callee == 0 {
        return None;
    }
    if route.direct_preparation {
        if route.callee != 0x318ad || !route.preparation {
            return None;
        }
    } else {
        let valid = match route.selector {
            0 => route.callee == 0x2fc85 && !route.preparation,
            1 | 3 => route.callee == 0x2e341 && !route.preparation,
            2 => route.callee == 0x318ad && route.preparation,
            4 => route.callee == 0x3072f && !route.preparation,
            _ => false,
        };
        if !valid {
            return None;
        }
    }
    if callee_result == 0 {
        return Some(NativePostbattleOutcome { repeat: true, return_value: 0 });
    }
    if route.direct_preparation || route.selector == 2 {
        return Some(NativePostbattleOutcome { repeat: false, return_value: 0 });
    }
    Some(NativePostbattleOutcome { repeat: false, return_value: 1 })
}

#[cfg(test)]
mod tests;
