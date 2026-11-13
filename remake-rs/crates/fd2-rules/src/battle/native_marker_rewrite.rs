//! Go 來源：`remake/internal/battle/native_marker_rewrite.go`（24 行）。

use super::*;

/// RewriteNativeMarker1F reproduces the raw loop at 0x24e80: beginning at
/// runtime slot 0x10 and stopping before caller-supplied count, records whose
/// byte +0x07 equals 0x1f receive bytes +0=0x10 and +1=0x06. These bytes remain
/// unnamed; the helper deliberately exposes only the proven mutation.
///
/// Go：`native_marker_rewrite.go:9`，`func RewriteNativeMarker1F(records []byte, count int) error`
pub fn rewrite_native_marker1_f(records: &mut [u8], count: i64) -> Result<()> {
    if count < 0 || count > records.len() as i64 / NATIVE_RECORD_SIZE {
        return Err(battle_err!("native marker rewrite count {count} is outside record buffer"));
    }
    if count > 0x10 && (records.len() as i64) < count * NATIVE_RECORD_SIZE {
        return Err(battle_err!(
            "native marker rewrite needs {} bytes, got {}",
            count * NATIVE_RECORD_SIZE,
            records.len()
        ));
    }
    for index in 0x10..count {
        let base = (index * NATIVE_RECORD_SIZE) as usize;
        if records[base + 0x07] == 0x1f {
            records[base] = 0x10;
            records[base + 1] = 0x06;
        }
    }
    Ok(())
}

#[cfg(test)]
mod native_marker_rewrite_test;
