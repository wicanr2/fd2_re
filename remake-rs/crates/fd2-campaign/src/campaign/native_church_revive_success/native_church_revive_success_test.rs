//! 移植自 `remake/internal/campaign/native_church_revive_success_test.go`。

use super::*;
use crate::test_support::input;

/// Go：`TestPlanNativeChurchReviveSuccessPreservesCase4Schedule`
#[test]
fn plan_native_church_revive_success_preserves_case4_schedule() {
    let plan = plan_native_church_revive_success();
    assert!(
        plan.start_music_track == 17
            && plan.return_music_track == 11
            && plan.music_loop_count == 1
            && plan.animation_delay_bios_ticks == 2
            && plan.palette_delay_ms == 4
            && plan.rise_latch_bios_ticks == 10
            && plan.fall_latch_bios_ticks == 5
            && plan.rise_palette_deltas.len() == 32
            && plan.fall_palette_deltas.len() == 32
            && plan.rise_palette_deltas[0] == 0
            && plan.rise_palette_deltas[31] == 62
            && plan.fall_palette_deltas[0] == 62
            && plan.fall_palette_deltas[31] == 0,
        "unexpected revive success plan: {plan:?}"
    );
}

/// Go：`TestNativeChurchReviveSuccessUsesPlayerOriginalEntries23Through31`
#[test]
fn native_church_revive_success_uses_player_original_entries_23_through_31() {
    let Some(fdother_path) = input("FD2_FLAME2_DIR", "FDOTHER.DAT") else { return };
    let dato_path = fdother_path.with_file_name("DATO.DAT");
    let resource = fdother::read_resource(&fdother_path, 14).unwrap();
    let frames: Vec<fdother::Frame> = (0..9)
        .map(|i| fdother::parse_lmi1_frame_entry(&resource, 23 + i).unwrap_or_else(|e| panic!("entry {}: {e}", 23 + i)))
        .collect();
    let portraits = dato::decode_resource(&dato_path, 131).unwrap();
    let background = vec![0u8; 320 * 200];
    let (animation, last) = compose_native_church_revive_success_frames(&background, &frames, &portraits[0]).unwrap();
    let changed: usize = animation.iter().map(|f| f.iter().filter(|&&p| p != 0).count()).sum();
    assert!(changed != 0 && last.len() == 320 * 200, "original revive effect changed={changed} final={}", last.len());
}

/// Go：`TestComposeNativeChurchReviveSuccessFramesAccumulatesAndRestoresPortrait`
#[test]
fn compose_native_church_revive_success_frames_accumulates_and_restores_portrait() {
    let frames: Vec<fdother::Frame> =
        (0..9).map(|i| fdother::Frame::raw(0, 0, 1, 1, vec![1, 0, 1, 0, 0, (i + 1) as u8])).collect();
    let portrait = dato::Frame { width: 1, height: 1, pixels: vec![77] };
    let (animation, last) =
        compose_native_church_revive_success_frames(&vec![0u8; 320 * 200], &frames, &portrait).unwrap();
    const EFFECT_OFFSET: usize = 32 * 320 + 147;
    assert!(
        animation.len() == 9 && animation[0][EFFECT_OFFSET] == 1 && animation[8][EFFECT_OFFSET] == 9,
        "animation accumulation is wrong: frames={} first={} last={}",
        animation.len(),
        animation[0][EFFECT_OFFSET],
        animation[8][EFFECT_OFFSET]
    );
    assert!(
        last[4 * 320 + 118] == 77 && animation[8][4 * 320 + 118] == 0,
        "portrait restore final={} prior={}",
        last[4 * 320 + 118],
        animation[8][4 * 320 + 118]
    );
}
