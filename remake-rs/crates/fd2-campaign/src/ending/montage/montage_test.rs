//! 移植自 `remake/internal/ending/montage_test.go`。

use super::*;
use crate::test_support::{input, remake_asset};

pub(crate) fn montage() -> Option<Montage> {
    let path = remake_asset("endings/native_2c548.json")?;
    Some(load_montage(&path).unwrap())
}

#[test]
fn test_native2_c548_montage_map_plans_native_party_slot_order() {
    let Some(montage) = montage() else { return };
    let plans = montage.plan_party_cycle(&[4, 0, 96]).unwrap();
    assert!(
        plans.len() == 3
            && plans[0]
                == PartyCyclePlan {
                    loop_index: 2,
                    unit_slot: 2,
                    visual_group: 96,
                    primary_figani: 289,
                    secondary_figani: 288,
                    frames: 20,
                    frame_delay_ticks: 1,
                }
            && plans[1].loop_index == 1
            && plans[1].unit_slot == 0
            && plans[1].primary_figani == 13
            && plans[2].loop_index == 0
            && plans[2].unit_slot == 1
            && plans[2].primary_figani == 1
            && plans[2].secondary_figani == 0,
        "party plans = {plans:?}"
    );
}

#[test]
fn test_native2_c548_portrait_countdown_matches_zero_reset_and_mouth_frames() {
    let mut state = MontagePortraitState::default();
    let frame = state.step(0xff);
    assert!(
        frame.as_ref().is_ok_and(|f| *f == NATIVE_MONTAGE_PORTRAIT_NORMAL_FRAME) && state.countdown == 0x47,
        "reset frame={frame:?} countdown={}",
        state.countdown
    );
    state.countdown = 2;
    let frame = state.step(0);
    assert!(
        frame.as_ref().is_ok_and(|f| *f == NATIVE_MONTAGE_PORTRAIT_MOUTH_FRAME) && state.countdown == 1,
        "mouth1 frame={frame:?} countdown={}",
        state.countdown
    );
    let frame = state.step(0);
    assert!(
        frame.as_ref().is_ok_and(|f| *f == NATIVE_MONTAGE_PORTRAIT_MOUTH_FRAME) && state.countdown == 0,
        "mouth0 frame={frame:?} countdown={}",
        state.countdown
    );
    let frame = state.step(0);
    assert!(
        frame.as_ref().is_ok_and(|f| *f == NATIVE_MONTAGE_PORTRAIT_NORMAL_FRAME) && state.countdown == 0x28,
        "next reset frame={frame:?} countdown={}",
        state.countdown
    );
    state.countdown = 0x48;
    assert!(state.step(0).is_err(), "invalid native countdown accepted");
}

#[test]
fn test_native2_c548_portrait_duration_makes_only_loop_zero_reach_special_text() {
    assert_eq!(native_montage_portrait_iterations(1).unwrap(), 0xdc, "regular iterations");
    assert_eq!(native_montage_portrait_iterations(0).unwrap(), 0x1b8, "final iterations");
    assert!(native_montage_portrait_iterations(-1).is_err(), "negative loop index accepted");
}

#[test]
fn test_compose_native2_c548_portrait_frame_with_player_resources() {
    let Some(fdother_path) = input("FD2_FLAME2_DIR", "FDOTHER.DAT") else { return };
    let Some(fdtxt_path) = input("FD2_FLAME2_DIR", "FDTXT.DAT") else { return };
    let Some(dato_path) = input("FD2_FLAME2_DIR", "DATO.DAT") else { return };
    let Some(montage) = montage() else { return };
    let mut restore = vec![0u8; BYTES as usize];
    render_dialogue_frame_grid_resource(&montage, &fdother_path.to_string_lossy(), &mut restore).unwrap();
    let original_restore = restore.clone();
    let portraits = dato::decode_resource(&dato_path, 37).unwrap();
    let current = fdtxt::parse(&fdother::read_resource(&fdtxt_path, 31).unwrap()).unwrap();
    let permanent = fdtxt::parse(&fdother::read_resource(&fdtxt_path, 0).unwrap()).unwrap();
    let font = fdtxt::parse_font(&fdother::read_resource(&fdother_path, 4).unwrap()).unwrap();
    let mut unit = vec![0u8; 0x21];
    (unit[7], unit[8], unit[0x20]) = (37, 4, 2);
    let normal = compose_montage_portrait_frame(
        &montage,
        &restore,
        &unit,
        0,
        0,
        &portraits,
        Some(&current),
        Some(&permanent),
        Some(&font),
    )
    .unwrap();
    let mouth = compose_montage_portrait_frame(
        &montage,
        &restore,
        &unit,
        0xdc,
        3,
        &portraits,
        Some(&current),
        Some(&permanent),
        Some(&font),
    )
    .unwrap();
    assert_eq!(restore, original_restore, "portrait composition mutated restore snapshot");
    assert!(
        normal.len() as i64 == BYTES && mouth.len() as i64 == BYTES && normal != mouth,
        "original portrait frames were not independently composed"
    );
    for frame in [&normal, &mouth] {
        let visible = frame.iter().filter(|&&p| p == 0xcd || p == 0x4c).count();
        assert!(visible > 0, "native montage text colors are absent");
    }
}

#[test]
fn test_native2_c548_montage_refuses_empty_party() {
    let Some(montage) = montage() else { return };
    let plans = montage.plan_party_cycle(&[4]);
    assert!(plans.is_err(), "plans={plans:?}");
}

#[test]
fn test_native2_c548_portrait_text_uses_edi_for_special_epilogue() {
    let Some(montage) = montage() else { return };
    let mut unit = vec![0u8; 0x21];
    (unit[7], unit[8], unit[0x20]) = (37, 4, 2);
    let plan = montage.plan_portrait_text(&unit, 0xdb).unwrap();
    assert!(
        plan.portrait_id == 37
            && plan.character_name.index == 5
            && plan.class_name.index == 0x98
            && plan.epilogue.index == 0x10
            && plan.epilogue.destination == 0x7d08,
        "portrait plan = {plan:?}"
    );
    let d = &montage.party_cycle.dialogue_frame_layout;
    assert!(
        d.stride == 320
            && d.resource == 5
            && d.arg8 == 5
            && d.arg_c == 7
            && d.arg10 == 5
            && d.arg14 == 5
            && d.destination == "portrait_restore_buffer_C",
        "dialogue frame layout = {d:?}"
    );
    let plan = montage.plan_portrait_text(&unit, 0xdc);
    assert!(plan.as_ref().is_ok_and(|p| p.epilogue.index == 0x2d), "special epilogue plan = {plan:?}");
}

#[test]
fn test_native2_c548_dialogue_frame_grid_transcribes_all_raw_cells() {
    let Some(montage) = montage() else { return };
    let cells = montage.plan_dialogue_frame_grid().unwrap();
    assert!(
        cells.len() == 49
            && cells[0] == DialogueFramePlacement { resource_index: 1, destination_byte: 2245 }
            && cells[1].resource_index == 2
            && cells[1].destination_byte == 2328,
        "dialogue frame cells head={:?} len={}",
        &cells[..2],
        cells.len()
    );
    let last = cells[cells.len() - 1];
    assert!(last.resource_index == 13 && last.destination_byte == 23752, "dialogue frame cells tail={last:?}");
}

#[test]
fn test_render_dialogue_frame_grid_uses_raw_opaque_copies() {
    let Some(montage) = montage() else { return };
    let cells: Vec<fdother::RawCell> =
        (0..18).map(|i| fdother::RawCell { width: 1, height: 1, pixels: vec![i as u8] }).collect();
    let mut dst = vec![0u8; BYTES as usize];
    render_dialogue_frame_grid(&montage, &cells, &mut dst).unwrap();
    assert!(
        dst[2245] == 1 && dst[2248] == 5 && dst[23752] == 13,
        "dialogue frame pixels={}/{}/{}",
        dst[2245],
        dst[2248],
        dst[23752]
    );
}

#[test]
fn test_render_dato_frame_at_keeps_caller_offset_explicit() {
    let mut dst = vec![0u8; BYTES as usize];
    let frame = dato::Frame { width: 2, height: 1, pixels: vec![0, 7] };
    render_dato_frame_at(&mut dst, &frame, 320, 0x0c88).unwrap();
    assert!(dst[0x0c88] == 0 && dst[0x0c89] == 7, "DATO staging paste={}/{}", dst[0x0c88], dst[0x0c89]);
}

#[test]
fn test_render_dialogue_frame_grid_resource_uses_player_fdother_when_present() {
    let Some(dat_path) = input("FD2_FLAME2_DIR", "FDOTHER.DAT") else { return };
    let Some(montage) = montage() else { return };
    let mut dst = vec![0u8; BYTES as usize];
    render_dialogue_frame_grid_resource(&montage, &dat_path.to_string_lossy(), &mut dst).unwrap();
    assert!(dst.iter().any(|&v| v != 0), "FDOTHER#5 dialogue frame grid remained empty");
}

#[test]
fn test_native2_c548_figure_fade_is_nine_non_mirrored_passes() {
    let Some(montage) = montage() else { return };
    let passes = montage.plan_figure_fade(1).unwrap();
    assert!(
        passes.len() == 9
            && passes[0] == FigureFadePass { stage: 8, source_offset: 80, palette_delta: 48 }
            && passes[8] == FigureFadePass { stage: 0, source_offset: 0, palette_delta: 0 },
        "fade passes={passes:?}"
    );
    let mirrored = montage.plan_figure_fade(0);
    assert!(mirrored.is_err(), "mirrored passes={mirrored:?}");
    let m = &montage.party_cycle.mirror_branch;
    assert!(
        m.required_unit_side == 0 && m.primary_source == "staging+0x140-stage*10" && m.secondary_when == "arg4==0",
        "mirror branch = {m:?}"
    );
    let mirror_passes = montage.plan_mirror_figure_fade(0, 0).unwrap();
    assert!(
        mirror_passes.len() == 9
            && mirror_passes[0]
                == MirrorFigureFadePass {
                    stage: 8,
                    primary_source_offset: 0xf0,
                    palette_delta: 48,
                    draw_secondary: true,
                    draw_platform: true,
                }
            && mirror_passes[8].primary_source_offset == 0x140
            && mirror_passes[8].draw_secondary,
        "mirror passes = {mirror_passes:?}"
    );
    let mirror_passes = montage.plan_mirror_figure_fade(0, 1);
    assert!(
        mirror_passes.as_ref().is_ok_and(|p| !p[0].draw_secondary && !p[0].draw_platform),
        "arg4-gated mirror passes = {mirror_passes:?}"
    );
    let wrong = montage.plan_mirror_figure_fade(1, 0);
    assert!(wrong.is_err(), "wrong-side mirror passes = {wrong:?}");
}
