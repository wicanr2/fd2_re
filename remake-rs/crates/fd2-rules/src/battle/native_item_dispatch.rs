//! Go 來源：`remake/internal/battle/native_item_dispatch.go`（62 行）。

use super::*;

/// NativeItemEffectRoute is the raw call-topology result of 0x20c6f.  The
/// fields intentionally retain addresses and integer arguments instead of
/// naming an item type as potion, status, damage, or equipment behavior.
///
/// Go：`native_item_dispatch.go:6`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeItemEffectRoute {
    pub item_type: u8,
    /// item row +0x0e, passed to callees that consume it
    pub word: u16,
    pub primary: u32,
    pub secondary: u32,
    pub arg2: i64,
    pub arg5: i64,
    pub arg7: i64,
}

/// NativeItemEffectRouteForType reproduces the verified dispatch table in
/// 0x20c6f without executing any callee or mutating runtime state.  A false
/// result means the type is not covered by the recovered table.
///
/// Go：`native_item_dispatch.go:19`，`func NativeItemEffectRouteForType(itemType byte, word uint16) (NativeItemEffectRoute, bool)`
pub fn native_item_effect_route_for_type(item_type: u8, word: u16) -> (NativeItemEffectRoute, bool) {
    let mut route = NativeItemEffectRoute { item_type, word, ..Default::default() };
    match item_type {
        5 | 13 => route.primary = 0x211a4,
        6 => (route.primary, route.arg2, route.arg5) = (0x22af6, 20, 37),
        7 => (route.primary, route.arg2, route.arg5) = (0x22af6, 21, 38),
        8 => (route.primary, route.arg2, route.arg7) = (0x21082, 55, 17),
        9 => (route.primary, route.arg2, route.arg7) = (0x21082, 57, 18),
        10 => (route.primary, route.arg2, route.arg7) = (0x21082, 62, 19),
        11 => (route.primary, route.secondary, route.arg2) = (0x1c4cc, 0x1c2da, 13),
        12 => route.primary = 0x22997,
        14 => (route.primary, route.arg2, route.arg5) = (0x22d1b, 27, 38),
        15 => route.primary = 0x22866,
        16 => route.primary = 0x22721,
        17 => (route.primary, route.arg2, route.arg7) = (0x21082, 66, 13),
        18 => (route.primary, route.arg2, route.arg7) = (0x21082, 70, 13),
        19 => (route.primary, route.arg2, route.arg7) = (0x21082, 59, 19),
        20 | 24 => (route.primary, route.secondary) = (0x1c4cc, 0x1cd17),
        21 => route.primary = 0x2111a,
        22 => (route.primary, route.arg2, route.arg5) = (0x22d1b, 22, 39),
        23 => route.primary = 0x2218a,
        _ => return (NativeItemEffectRoute::default(), false),
    }
    (route, true)
}

/// Go `fmt` 的 `%#v` 對一個結構欄位值的輸出（C 組道具路由結構只用到這幾種欄位型別）。
///
/// Go 的 `%#v` 把無號整數寫成 `0x..`、有號整數寫成十進位、字串（含具名字串型別）以
/// `strconv.Quote` 加引號、布林寫成 `true`／`false`。
pub(crate) enum GoV<'a> {
    /// 無號整數（`byte`、`uint16`、`uint32`）。
    U(u64),
    /// 有號整數（`int`）。
    I(i64),
    /// 布林。
    B(bool),
    /// 字串或具名字串型別。
    S(&'a str),
}

/// Go `fmt.Sprintf("%#v", v)` 對 `battle` 套件結構的輸出：`battle.<型別>{<欄位>:<值>, ...}`。
/// 欄位依 Go 宣告順序傳入。路由驗證失敗的錯誤訊息要與 Go 逐字相同，所以用它組字串。
pub(crate) fn go_sharp_v(type_name: &str, fields: &[(&str, GoV<'_>)]) -> String {
    let body: Vec<String> = fields
        .iter()
        .map(|(name, value)| {
            let v = match value {
                GoV::U(v) => format!("{v:#x}"),
                GoV::I(v) => v.to_string(),
                GoV::B(v) => v.to_string(),
                GoV::S(v) => go_quote(v),
            };
            format!("{name}:{v}")
        })
        .collect();
    format!("battle.{type_name}{{{}}}", body.join(", "))
}

#[cfg(test)]
mod native_item_dispatch_test;
