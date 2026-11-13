//! 移植自 `remake/internal/battle/native_command_palette_flash_test.go`。

use super::*;
use crate::test_support::remake_asset;

/// Go：`TestLoadNativeCommandPaletteFlashTablePreservesIDABytes`
#[test]
fn load_native_command_palette_flash_table_preserves_ida_bytes() {
    let Some(path) = remake_asset("data/native_command_palette_flash.json") else { return };
    let table = load_native_command_palette_flash_table(&path).unwrap();
    for id in [17, 18, 19] {
        let phases = table.native_command_palette_flash_phases(id).unwrap();
        assert_eq!(phases.len(), 8, "command {id} phases");
        for (i, rgb) in phases.iter().enumerate() {
            let want = if i % 2 == 0 { [0x32, 0x32, 0x32] } else { [0; 3] };
            assert_eq!(*rgb, want, "command {id} phase {i}");
        }
    }
}

/// Go：`TestNativeCommandPaletteFlashPhasesRejectsUnknownID`
#[test]
fn native_command_palette_flash_phases_rejects_unknown_id() {
    let table = NativeCommandPaletteFlashTable {
        cycles: 4,
        ticks_per_phase: 1,
        entries: vec![[0; 3]; 36],
        ..Default::default()
    };
    assert!(table.native_command_palette_flash_phases(-1).is_err(), "negative command accepted");
    assert!(table.native_command_palette_flash_phases(36).is_err(), "out-of-range command accepted");
}
