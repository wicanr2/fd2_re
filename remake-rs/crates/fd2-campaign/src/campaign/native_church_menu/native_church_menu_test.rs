//! 移植自 `remake/internal/campaign/native_church_menu_test.go`。

use super::*;
use crate::test_support::input;

/// Go：`syntheticNativeChurchMenuEntries`
fn synthetic_native_church_menu_entries() -> Vec<fdother::LMI1Entry> {
    let mut entries = vec![fdother::LMI1Entry::default(); 11];
    for option in 0..4 {
        for pulse in 0..2 {
            let index = 2 * option + 3 + pulse;
            entries[index] = fdother::LMI1Entry { width: 1, height: 1, pixels: vec![index as u8] };
        }
    }
    entries
}

/// Go：`TestNativeChurchMenuTransitionUsesRecoveredDivisors`
#[test]
fn native_church_menu_transition_uses_recovered_divisors() {
    let background = vec![0u8; 320 * 200];
    let entries = synthetic_native_church_menu_entries();
    let opening = native_church_menu_transition_frames(&background, &entries, true).unwrap();
    let closing = native_church_menu_transition_frames(&background, &entries, false).unwrap();
    assert!(
        opening.len() == 4 && closing.len() == 4,
        "transition frame counts opening={} closing={}",
        opening.len(),
        closing.len()
    );
    for pass in 0..4 {
        let (open_divisor, close_divisor) = (4 - pass as i64, pass as i64 + 1);
        for (option, &offset) in NATIVE_CHURCH_MENU_OFFSETS.iter().enumerate() {
            let index = (2 * option + 3) as u8;
            let got = opening[pass][(NATIVE_CHURCH_MENU_BASE + offset / open_divisor) as usize];
            assert_eq!(got, index, "opening pass{pass} option{option}");
            let got = closing[pass][(NATIVE_CHURCH_MENU_BASE + offset / close_divisor) as usize];
            assert_eq!(got, index, "closing pass{pass} option{option}");
        }
    }
}

/// Go：`TestComposeNativeChurchMenuFramePulsesOnlySelectedCell`
#[test]
fn compose_native_church_menu_frame_pulses_only_selected_cell() {
    let frame =
        compose_native_church_menu_frame(&vec![0u8; 320 * 200], &synthetic_native_church_menu_entries(), 2, 1).unwrap();
    for (option, &offset) in NATIVE_CHURCH_MENU_OFFSETS.iter().enumerate() {
        let mut want = (2 * option + 3) as u8;
        if option == 2 {
            want += 1;
        }
        assert_eq!(frame[(NATIVE_CHURCH_MENU_BASE + offset) as usize], want, "option{option}");
    }
    assert_eq!(frame[169 * 320 + 201 + 1], 0x4a, "cleared source pixel");
}

/// Go：`nativeChurchOriginalTestEntries`
pub(crate) fn native_church_original_test_entries(resource: &[u8]) -> Vec<fdother::LMI1Entry> {
    let mut entries = vec![fdother::LMI1Entry::default(); 17];
    for index in [1usize, 16] {
        entries[index] = fdother::parse_lmi1_opaque_entry(resource, index as i64).unwrap();
    }
    for (index, entry) in entries.iter_mut().enumerate().take(11).skip(3) {
        let cell = fdother::parse_lmi1_raw_entry(resource, index as i64).unwrap();
        *entry = fdother::LMI1Entry { width: cell.width, height: cell.height, pixels: cell.pixels };
    }
    entries
}

/// Go：`TestNativeChurchMenuOriginalResourceCells`
#[test]
fn native_church_menu_original_resource_cells() {
    let Some(path) = input("FD2_FLAME2_DIR", "FDOTHER.DAT") else { return };
    let resource = fdother::read_resource(&path, 14).unwrap();
    let entries = native_church_original_test_entries(&resource);
    let background_frame = fdother::parse_lmi1_frame_entry(&resource, 0).unwrap();
    let mut background = vec![0u8; 320 * 200];
    background_frame.blit_at(&mut background, 320, 0, -1).unwrap();
    let opening = native_church_menu_transition_frames(&background, &entries, true).unwrap();
    let steady = compose_native_church_menu_frame(&background, &entries, 0, 1).unwrap();
    for (index, entry) in entries.iter().enumerate().take(11).skip(3) {
        assert!(
            entry.width == 24 && entry.height == 20,
            "FDOTHER#14 cell{index}={}x{} want24x20",
            entry.width,
            entry.height
        );
    }
    assert!(
        opening.len() == 4 && steady.len() == 320 * 200,
        "native church menu original-resource compositor is incomplete"
    );
}
