//! Go 來源：`remake/internal/battle/native_item_panel_record.go`（90 行）。

use super::*;

/// NativeItemPanelRecordForUnit materializes only the 80-byte fields consumed
/// by 0x17eef/0x17fc0/0x184c0. Raw provenance is mandatory for selectors,
/// constructor bytes and inventory cells; normalized fields are used only
/// where the native record meaning is independently established.
///
/// Go：`native_item_panel_record.go:13`，`func NativeItemPanelRecordForUnit(unit *Unit) ([]byte, error)`
pub fn native_item_panel_record_for_unit(unit: Option<&Unit>) -> Result<Vec<u8>> {
    todo!("battle/native_item_panel_record.go:NativeItemPanelRecordForUnit")
}

/// Go：`native_item_panel_record.go:72`，`func nativeRecordByte8ForUnit(unit *Unit) (byte, bool)`
pub(crate) fn native_record_byte8_for_unit(unit: Option<&Unit>) -> (u8, bool) {
    todo!("battle/native_item_panel_record.go:nativeRecordByte8ForUnit")
}

/// Go：`native_item_panel_record.go:88`，`func putNativeItemPanelWord(record []byte, offset, value int)`
pub(crate) fn put_native_item_panel_word(record: &mut [u8], offset: i64, value: i64) {
    todo!("battle/native_item_panel_record.go:putNativeItemPanelWord")
}
