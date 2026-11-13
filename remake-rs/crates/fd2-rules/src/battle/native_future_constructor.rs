//! Go 來源：`remake/internal/battle/native_future_constructor.go`（155 行）。

use super::*;

/// NativeFutureConstructorBase is the table-derived subset written by
/// 0x10d7f..0x10ebd before inventory/equipment recomputation. Values retain
/// the original 16-bit write boundary.
///
/// Go：`native_future_constructor.go:11`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeFutureConstructorBase {
    pub record_race: u8,
    pub record_class: u8,
    pub base_ap: u16,
    pub base_dp: u16,
    pub base_dx: u16,
    pub mobility: u8,
    pub max_hp: u16,
    pub max_mp: u16,
}

/// DecodeNativeFutureConstructorBase transcribes the two b1-selected table
/// branches in 0x10d99..0x10ebd. It deliberately stops before 0x1b750:
/// effective AP/DP/HIT/EV require the separately verified inventory table and
/// are not inferred here.
///
/// Go：`native_future_constructor.go:26`，`func DecodeNativeFutureConstructorBase( table *NativeConstructorTable, level byte, ) (NativeFutureConstructorBase, error)`
pub fn decode_native_future_constructor_base(
    table: Option<&NativeConstructorTable>,
    level: u8,
) -> Result<NativeFutureConstructorBase> {
    todo!("battle/native_future_constructor.go:DecodeNativeFutureConstructorBase")
}

/// MaterializeNativeFutureConstructor applies the typed fields closed by
/// 0x10c50's table/inventory tail and its 0x1b750 call. It is intentionally a
/// Unit projection rather than a claim of byte-identical 0x50-byte storage:
/// fields without a typed consumer remain in their existing raw provenance.
/// Every source and selected item row is validated before unit is replaced.
///
/// Go：`native_future_constructor.go:87`，`func MaterializeNativeFutureConstructor(unit *Unit, itemTable []byte) error`
pub fn materialize_native_future_constructor(unit: Option<&mut Unit>, item_table: &[u8]) -> Result<()> {
    todo!("battle/native_future_constructor.go:MaterializeNativeFutureConstructor")
}

/// Go：`native_future_constructor.go:149`，`func mulWord(value, multiplier byte) uint16`
pub(crate) fn mul_word(value: u8, multiplier: u8) -> u16 {
    todo!("battle/native_future_constructor.go:mulWord")
}

/// Go：`native_future_constructor.go:153`，`func addScaledWord(base uint16, value byte, multiplier int32) uint16`
pub(crate) fn add_scaled_word(base: u16, value: u8, multiplier: i32) -> u16 {
    todo!("battle/native_future_constructor.go:addScaledWord")
}
