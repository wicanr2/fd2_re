//! 移植自 `battle_entry_step_test.go`。

use crate::fdother::*;

fn put_word(raw: &mut [u8], offset: usize, value: u16) {
    raw[offset..offset + 2].copy_from_slice(&value.to_le_bytes());
}

// Go: TestNativeBattleEntryStepPreservesRawGateAndClampedAdvance
#[test]
fn native_battle_entry_step_preserves_raw_gate_and_clamped_advance() {
    let mut raw = vec![0u8; 0x50];
    raw[6] = 2;
    put_word(&mut raw, 0x40, 30);
    put_word(&mut raw, 0x42, 100);
    assert_eq!(native_battle_entry_step(&mut raw), Ok((50, true, true)));
    put_word(&mut raw, 0x40, 95);
    assert_eq!(native_battle_entry_step(&mut raw), Ok((100, true, true)), "clamp");
}

// Go: TestNativeBattleEntryStepRejectsOtherRawGates
#[test]
fn native_battle_entry_step_rejects_other_raw_gates() {
    let mutations: [fn(&mut [u8]); 5] =
        [|raw| raw[6] = 1, |raw| raw[5] = 1, |raw| raw[5] = 0x80, |raw| raw[0x25] = 1, |raw| raw[0x26] = 1];
    for mutate in mutations {
        let mut raw = vec![0u8; 0x50];
        raw[6] = 2;
        put_word(&mut raw, 0x40, 20);
        put_word(&mut raw, 0x42, 100);
        mutate(&mut raw);
        let before = raw.clone();
        let (_, eligible, _) = native_battle_entry_step(&mut raw).unwrap();
        assert!(!eligible && raw == before, "gate mutation accepted");
    }
}
