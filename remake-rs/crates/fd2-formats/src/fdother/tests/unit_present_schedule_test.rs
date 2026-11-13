//! 移植自 `unit_present_schedule_test.go`。

use crate::fdother::unit_present_schedule::{NATIVE_UNIT_PRESENT_BASE, NATIVE_UNIT_PRESENT_STRIDE};
use crate::fdother::*;

const STRIDE: usize = NATIVE_UNIT_PRESENT_STRIDE as usize;
const WORK: usize = NATIVE_UNIT_PRESENT_WORK_BYTES as usize;

// Go: TestNativeUnitPresentSchedulePreservesAllTwentySevenPresents
#[test]
fn native_unit_present_schedule_preserves_all_twenty_seven_presents() {
    let steps = native_unit_present_schedule();
    validate_native_unit_present_schedule(&steps).unwrap();
    assert_eq!(steps.len(), 27);
    let g = &steps[0];
    assert!(g.phase == "lmi_intro" && g.resource == 6 && g.entry == 0x72 && g.delay_ticks == 1, "first={g:?}");
    let g = &steps[10];
    assert!(g.entry == 0x7c && g.delay_ticks == 1, "last LMI={g:?}");
    let g = &steps[11];
    assert!(g.phase == "lut_contract" && g.resource == 3 && g.entry == 5 && g.delay_ms == 10, "first contract={g:?}");
    let g = &steps[16];
    assert!(g.entry == 0 && g.delay_ms == 10 && g.delay_ticks == 2, "last contract={g:?}");
    let g = &steps[17];
    assert!(g.phase == "lut_release" && g.entry == 0 && g.delay_ticks == 1, "first release={g:?}");
    let g = &steps[26];
    assert!(g.entry == 9 && g.delay_ticks == 1, "last release={g:?}");
}

// Go: TestPlanNativeUnitPresentCallPreservesMutationAndVisualPairs
#[test]
fn plan_native_unit_present_call_preserves_mutation_and_visual_pairs() {
    let d = plan_native_unit_present_call(1, 0xff, 0xff, 15, 10).unwrap();
    assert!(
        d.unit_slot == 1 && d.new_x == 0xff && d.new_y == 0xff && d.visual_x == 15 && d.visual_y == 10,
        "disappear call={d:?}"
    );
    let a = plan_native_unit_present_call(25, 21, 18, 21, 18).unwrap();
    assert!(a.new_x == a.visual_x && a.new_y == a.visual_y, "appear call={a:?}");
    assert!(plan_native_unit_present_call(0x100, 0, 0, 0, 0).is_err(), "accepted unit slot outside byte range");
}

// Go: TestValidateNativeUnitPresentScheduleRejectsShortcuts
#[test]
fn validate_native_unit_present_schedule_rejects_shortcuts() {
    let steps = native_unit_present_schedule();
    assert!(validate_native_unit_present_schedule(&steps[..6]).is_err(), "former six-frame approximation");
    let mut steps = native_unit_present_schedule();
    steps[16].delay_ticks = 0;
    assert!(validate_native_unit_present_schedule(&steps).is_err(), "middle tail ticks at the wrong timing");
}

// Go: TestNativeUnitPresentByteOriginMatches22470AddressExpression
#[test]
fn native_unit_present_byte_origin_matches_22470_address_expression() {
    let got = native_unit_present_byte_origin(22, 23, 16, 19);
    assert_eq!(got, 0x8088 + 0x18 * (22 - 16) + 0x18 * 0x1c8 * (23 - 19) + 0x1c8);
    // 保留 raw 有號算術：裁切是繪圖器／呼叫端的決定，不可悄悄併入還原出的原生位址式。
    assert_eq!(native_unit_present_byte_origin(0, 0, 1, 1), 0x8088 - 0x18 - 0x18 * 0x1c8 + 0x1c8, "offscreen origin");
}

// Go: TestNativeUnitPresentContractStartYMatches22547
#[test]
fn native_unit_present_contract_start_y_matches_22547() {
    for (raw, lut, want) in [(0, 5, 15), (1, 5, 35), (3, 2, 34), (3, 0, 0)] {
        assert_eq!(native_unit_present_contract_start_y(raw, lut).unwrap(), want, "raw={raw} lut={lut}");
    }
    assert!(native_unit_present_contract_start_y(0, 6).is_err(), "invalid LUT index was accepted");
}

// Go: TestNativeUnitPresentLUTPassUsesFixedRadiusAndSplitRows
#[test]
fn native_unit_present_lut_pass_uses_fixed_radius_and_split_rows() {
    let pass = native_unit_present_lut_pass(84, 63, 35).unwrap();
    let (f, s, r) = (pass.first_radial, pass.second_radial, pass.final_rect);
    assert!(
        f.center_x == 84
            && f.center_y == 63
            && f.radius == 11
            && f.scale == 16
            && f.start_y == 35
            && f.end_y == 192
            && s.start_y == 63
            && s.end_y == 192
            && r.horizontal_radius == 17
            && r.start_y == 35
            && r.end_y == 63,
        "unit-present pass={pass:?}"
    );
    assert!(native_unit_present_lut_pass(400, 63, 35).is_err(), "offscreen center unexpectedly accepted");
}

// Go: TestNativeUnitPresentLUTFramesPreserveSixPlusTenGeometry
#[test]
fn native_unit_present_lut_frames_preserve_six_plus_ten_geometry() {
    let frames = native_unit_present_lut_frames(3, 2).unwrap();
    assert!(
        frames.len() == 16
            && frames[0].phase == "lut_contract"
            && frames[0].lut_index == 5
            && frames[0].pass.first_radial.start_y == 60
            && frames[5].lut_index == 0
            && frames[5].pass.first_radial.start_y == 0
            && frames[5].delay_ticks == 2
            && frames[6].phase == "lut_release"
            && frames[6].lut_index == 0
            && frames[15].lut_index == 9
            && frames[15].pass.first_radial.start_y == 0,
        "unit-present LUT frames={frames:?}"
    );
    for frame in &frames {
        let f = frame.pass.first_radial;
        assert!(f.center_x == 84 && f.center_y == 63 && f.radius == 11, "unit-present frame geometry={frame:?}");
    }
    assert!(native_unit_present_lut_frames(20, 2).is_err(), "offscreen native center unexpectedly accepted");
}

// Go: TestRunNativeUnitPresentLUTFrameRestoresSnapshotAndKeepsRedrawBetweenPasses
#[test]
fn run_native_unit_present_lut_frame_restores_snapshot_and_keeps_redraw_between_passes() {
    let frames = native_unit_present_lut_frames(3, 2).unwrap();
    let mut work = vec![0xeeu8; WORK];
    let snapshot = vec![1u8; WORK];
    let lut: Vec<u8> = (0..256).map(|i| (i + 1) as u8).collect();
    let (mut redraws, mut presents) = (0, 0);
    let mut redraw = |full: &mut [u8]| {
        redraws += 1;
        full[NATIVE_UNIT_PRESENT_BASE as usize + 63 * STRIDE + 84] = 9;
        Ok(())
    };
    let mut present = |full: &mut [u8], frame: &UnitPresentLUTFrame| {
        presents += 1;
        assert!(frame.lut_index == 0 && full[0] == 1, "present frame={frame:?} prefix={}", full[0]);
        Ok(())
    };
    run_native_unit_present_lut_frame(&mut work, &snapshot, &lut, &frames[5], Some(&mut redraw), Some(&mut present))
        .unwrap();
    assert!(redraws == 1 && presents == 1, "redraws={redraws} presents={presents}");
    assert_eq!(work[0], 1, "full snapshot was not restored");
    let mut noop_redraw = |_: &mut [u8]| Ok(());
    let mut noop_present = |_: &mut [u8], _: &UnitPresentLUTFrame| Ok(());
    let r = run_native_unit_present_lut_frame(
        &mut work[..10],
        &snapshot,
        &lut,
        &frames[0],
        Some(&mut noop_redraw),
        Some(&mut noop_present),
    );
    assert!(r.is_err(), "short work buffer unexpectedly accepted");
}

// Go: TestBlitNativeUnitPresentLMIUses22470OriginAndTransparency
#[test]
fn blit_native_unit_present_lmi_uses_22470_origin_and_transparency() {
    let mut dst = vec![0xeeu8; STRIDE * 200];
    let entry = LMI1Entry { width: 2, height: 1, pixels: vec![0, 0x44] };
    blit_native_unit_present_lmi(&entry, &mut dst, 22, 23, 16, 19).unwrap();
    let origin = native_unit_present_byte_origin(22, 23, 16, 19) as usize;
    assert!(dst[origin] == 0xee && dst[origin + 1] == 0x44, "LMI blit={:#x},{:#x}", dst[origin], dst[origin + 1]);
    assert!(blit_native_unit_present_lmi(&entry, &mut dst, 0, 0, 100, 100).is_err(), "offscreen origin");
}

// Go: TestRunNativeUnitPresentLMIIntroPreservesElevenRedrawPresentTicks
#[test]
fn run_native_unit_present_lmi_intro_preserves_eleven_redraw_present_ticks() {
    let mut entries = vec![LMI1Entry::default(); 0x7d];
    for (i, e) in entries.iter_mut().enumerate().skip(0x72) {
        *e = LMI1Entry { width: 1, height: 1, pixels: vec![i as u8] };
    }
    let mut dst = vec![0u8; STRIDE * 200];
    let mut presents = 0;
    let mut tick = |_: &mut [u8]| {
        presents += 1;
        Ok(())
    };
    run_native_unit_present_lmi_intro(&entries, &mut dst, 22, 23, 16, 19, Some(&mut tick)).unwrap();
    assert_eq!(presents, 11);
    assert_eq!(dst[native_unit_present_byte_origin(22, 23, 16, 19) as usize], 0x7c, "last entry pixel");
    let mut noop = |_: &mut [u8]| Ok(());
    assert!(
        run_native_unit_present_lmi_intro(&entries[..0x7c], &mut dst, 22, 23, 16, 19, Some(&mut noop)).is_err(),
        "short entry table was accepted"
    );
}
