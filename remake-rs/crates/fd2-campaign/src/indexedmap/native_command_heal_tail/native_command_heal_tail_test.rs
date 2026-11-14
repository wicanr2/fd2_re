//! 移植自 Go `indexedmap/native_command_heal_tail_test.go`。

use super::*;
use fd2_formats::fdicon::{NATIVE_SIZE, Sprite};

const WORK_SIZE: usize = NATIVE_UNIT_PRESENT_WORK_SIZE as usize;
const VGA_SIZE: usize = NATIVE_MAP_VGA_SIZE as usize;
const ORIGIN: usize = (WORK_BASE - 6 * WORK_STRIDE) as usize;

fn cache0() -> NativeSelectorCache {
    let mut cache = NativeSelectorCache::default();
    cache.slot_for(0).unwrap();
    cache
}

#[test]
fn test_compose_native_command_heal_effect_and_mask_use_recovered_origin() {
    let snapshot = vec![0u8; WORK_SIZE];
    let mut work = vec![0u8; WORK_SIZE];
    let mut vga = vec![0u8; VGA_SIZE];
    let target = NativeCommandHealTailTarget { record_index: 3, x: 0, y: 0, selector_slot: 0 };
    let entry = LMI1Entry { width: 1, height: 1, pixels: vec![0x66] };
    compose_native_command_heal_effect_frame(&mut work, &mut vga, &snapshot, &entry, &[target], 0, 0).unwrap();
    assert_eq!(work[ORIGIN], 0x66, "effect origin");

    let n = (NATIVE_SIZE * NATIVE_SIZE) as usize;
    let mut mask = vec![0u8; n];
    (mask[0], mask[2]) = (1, 1);
    let mut bank = Bank { sprites: vec![Sprite::default(); 12] };
    bank.sprites[0] = Sprite { pixels: vec![0; n], mask, remap_mask: vec![0; n] };
    let cache = cache0();
    compose_native_command_heal_mask_frame(
        &mut work,
        &mut vga,
        &snapshot,
        Some(&bank),
        Some(&cache),
        &[target],
        0,
        0,
        0,
        0xc0,
    )
    .unwrap();
    assert!(
        work[ORIGIN] == 0xc0 && work[ORIGIN + 2] == 0xc0 && work[ORIGIN + 1] == 0,
        "mask pixels={:x} {:x} {:x}",
        work[ORIGIN],
        work[ORIGIN + 1],
        work[ORIGIN + 2]
    );
}

#[test]
fn test_compose_native_ai_item_damage_blend_uses_raw_table_base_and_cycle_remap() {
    let snapshot = vec![0u8; WORK_SIZE];
    let mut work = vec![0u8; WORK_SIZE];
    let mut vga = vec![0u8; VGA_SIZE];
    let n = (NATIVE_SIZE * NATIVE_SIZE) as usize;
    let (mut pixels, mut mask) = (vec![0u8; n], vec![0u8; n]);
    (pixels[0], pixels[1], mask[0]) = (6, 7, 1);
    let mut bank = Bank { sprites: vec![Sprite::default(); 12] };
    bank.sprites[2] = Sprite { pixels, mask, remap_mask: vec![0; n] };
    let cache = cache0();
    let target = NativeCommandHealTailTarget { record_index: 9, x: 0, y: 0, selector_slot: 0 };
    compose_native_ai_item_damage_blend_frame(
        &mut work,
        &mut vga,
        &snapshot,
        Some(&bank),
        Some(&cache),
        &[target],
        0,
        0,
        3,
        7,
        0x20,
    )
    .unwrap();
    assert!(work[ORIGIN] == 0x25 && work[ORIGIN + 1] == 0, "blend pixels={:#x} {:#x}", work[ORIGIN], work[ORIGIN + 1]);
}

#[test]
fn test_compose_native_ai_item_damage_blend_rejects_atomically() {
    let snapshot = vec![0u8; WORK_SIZE];
    let mut work = vec![0u8; WORK_SIZE];
    work[0] = 0x77;
    let mut vga = vec![0u8; VGA_SIZE];
    let before = work.clone();
    let (bank, cache) = (Bank::default(), NativeSelectorCache::default());
    assert!(
        compose_native_ai_item_damage_blend_frame(
            &mut work,
            &mut vga,
            &snapshot,
            Some(&bank),
            Some(&cache),
            &[],
            0,
            0,
            4,
            0,
            0x20
        )
        .is_err(),
        "invalid idle cycle accepted"
    );
    assert_eq!(work, before, "rejected blend mutated work");
}

#[test]
fn test_compose_native_command_heal_digit_frame_uses_queue_phase_and_target() {
    let snapshot = vec![0u8; WORK_SIZE];
    let mut work = vec![0u8; WORK_SIZE];
    let mut vga = vec![0u8; VGA_SIZE];
    let mut digits = vec![LMI1Entry::default(); 0x6a];
    digits[0x69] = LMI1Entry { width: 1, height: 1, pixels: vec![0x77] };
    let target = NativeCommandHealTailTarget { record_index: 4, x: 0, y: 0, ..Default::default() };
    let mut queue = vec![NativePresentationDigit { position_code: 2, target: 4, digit: 0x69 }];
    let vertical = [15, 15, 15, 15, 7, 3, 1, 0, 0, 1, 3, 7, 15, 15, 11, 9, 8, 8, 9, 11, 15, 15, 15, 15, 15];
    compose_native_command_heal_digit_frame(
        &mut work,
        &mut vga,
        &snapshot,
        &digits,
        &queue,
        &[target],
        0,
        0,
        vertical,
        0,
    )
    .unwrap();
    let origin = (WORK_BASE + 2 + (vertical[0] - 3) * WORK_STRIDE) as usize;
    assert_eq!(work[origin], 0x77, "digit origin");
    let before = work.clone();
    queue[0].target = 9;
    assert!(
        compose_native_command_heal_digit_frame(
            &mut work,
            &mut vga,
            &snapshot,
            &digits,
            &queue,
            &[target],
            0,
            0,
            vertical,
            0
        )
        .is_err(),
        "unknown queue target accepted"
    );
    assert_eq!(work, before, "rejected digit frame mutated work");
}
