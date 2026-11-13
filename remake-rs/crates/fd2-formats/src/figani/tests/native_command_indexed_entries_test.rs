//! Go `native_command_indexed_entries_test.go`。

use super::*;

const TABLE: &str = "native_command_indexed_entries.json";

// Go: TestLoadNativeCommandIndexedEntryTablePreservesCorrectedLinearEntries
#[test]
fn load_native_command_indexed_entry_table_preserves_corrected_linear_entries() {
    let Some(path) = remake_asset(TABLE) else { return };
    let table = load_native_command_indexed_entry_table(&path).unwrap();
    for command_id in 1..=9 {
        let want = native_command_indexed_entry_address(command_id).unwrap();
        let entry = table.schedule(command_id).unwrap();
        assert_eq!(entry.entry, want, "command {command_id}");
    }
    let ret = |e: &NativeCommandIndexedEntrySchedule, k: &str| e.mode_returns.get(k).copied().unwrap_or(0);
    let entry7 = table.schedule(7).unwrap();
    assert!(
        entry7.init_channels == 4
            && entry7.draw_channels == 3
            && ret(&entry7, "3") == 32
            && entry7.sample_markers.len() == 6,
        "command7 raw channel contract={entry7:?}"
    );
    let entry3 = table.schedule(3).unwrap();
    assert!(
        entry3.raw_side_zero_x_shift == 20
            && !entry3.uses_rng
            && entry3.state_ranges == ["0x53F81..0x53FE3"]
            && entry3.sample_markers.len() == 6,
        "command3 direct-instruction correction={entry3:?}"
    );
    let entry8 = table.schedule(8).unwrap();
    assert!(
        entry8.entry != "0x174B0"
            && ret(&entry8, "3") == 34
            && entry8.state_ranges == ["0x540BA..0x540F9"]
            && entry8.sample_markers.len() == 4,
        "command8 address-base correction={entry8:?}"
    );
    let entry9 = table.schedule(9).unwrap();
    assert!(
        entry9.entry == "0x275D6"
            && ret(&entry9, "0") == 20
            && ret(&entry9, "3") == 60
            && ret(&entry9, "6") == 20
            && entry9.state_ranges == ["0x540FA..0x540FB"]
            && entry9.sample_markers.len() == 2,
        "command9 direct-instruction contract={entry9:?}"
    );
    let entry2 = table.schedule(2).unwrap();
    assert!(
        entry2.offset_tables.is_empty() && entry2.state_ranges.len() == 1 && entry2.sample_markers.len() == 3,
        "command2 corrected ownership={entry2:?}"
    );
    let entry6 = table.schedule(6).unwrap();
    assert!(
        ret(&entry6, "0") == 7
            && ret(&entry6, "3") == 12
            && ret(&entry6, "6") == 7
            && entry6.init_channels == 5
            && entry6.draw_channels == 5
            && entry6.sample_markers.len() == 3,
        "command6 direct-instruction contract={entry6:?}"
    );
    let entry5 = table.schedule(5).unwrap();
    assert!(
        ret(&entry5, "0") == 1
            && ret(&entry5, "3") == 12
            && ret(&entry5, "6") == 8
            && entry5.init_channels == 6
            && entry5.draw_channels == 6
            && entry5.uses_rng
            && entry5.sample_markers.len() == 6,
        "command5 direct-instruction contract={entry5:?}"
    );
}

// Go: TestLoadNativeCommandIndexedEntryTableRejectsRawUnrelocatedAddress
#[test]
fn load_native_command_indexed_entry_table_rejects_raw_unrelocated_address() {
    let Some(path) = remake_asset(TABLE) else { return };
    let mut raw = std::fs::read(path).unwrap();
    if let Some(index) = raw.windows(7).position(|w| w == b"0x274B0") {
        raw[index..index + 7].copy_from_slice(b"0x174B0");
    }
    let dir = TempDir::new("indexed-entries");
    let path = dir.0.join("entries.json");
    std::fs::write(&path, raw).unwrap();
    assert!(
        load_native_command_indexed_entry_table(&path.to_string_lossy()).is_err(),
        "raw unrelocated entry address was accepted"
    );
}
