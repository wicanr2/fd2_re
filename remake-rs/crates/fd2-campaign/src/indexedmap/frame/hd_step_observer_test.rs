//! 移植自 Go `indexedmap/hd_step_observer_test.go`。

use super::*;
use crate::indexedmap::Result;
use crate::indexedmap::test_util::bank;
use fd2_formats::fdicon::{NativeSelectorCache, NativeTerrainCell};

#[test]
fn test_native_step_terrain_observer_preserves_both_callers_and_atomicity() {
    let (t, u, f) = (bank(12, 7), bank(12, 8), bank(12, 9));
    let cache = NativeSelectorCache::default();
    let controls = [0u8; 4];
    let lut = vec![0u8; 256];
    let in_ = FrameInput {
        terrain_bank: Some(&t),
        unit_bank: Some(&u),
        foreground_bank: Some(&f),
        selector_cache: Some(&cache),
        map_width: 13,
        camera_y: 1,
        cells: vec![NativeTerrainCell { tile: 0, blit_mode: 0xff }; 13 * 10],
        controls: &controls,
        lut: &lut,
        ..Default::default()
    };
    for up in [false, true] {
        let name = if up { "up-shift" } else { "ordinary" };
        let mut work = vec![99u8; NATIVE_UNIT_PRESENT_WORK_SIZE as usize];
        let mut vga = vec![77u8; NATIVE_MAP_VGA_SIZE as usize];
        let (mut observed_work, mut observed_vga) = (work.clone(), vga.clone());
        let mut calls = 0;
        let mut observer = |stage: FrameStage, w: &mut [u8], v: &mut [u8]| -> Result<()> {
            assert_eq!(stage, FRAME_STAGE_TERRAIN, "{name}: unexpected diagnostic stage");
            calls += 1;
            w[0] = 201;
            v[0] = 202;
            Ok(())
        };
        let (first, second);
        if up {
            first = compose_native_up_step_frame(&mut work, &mut vga, &in_, 2, 3);
            second = compose_native_up_step_frame_observed(
                &mut observed_work,
                &mut observed_vga,
                &in_,
                2,
                3,
                Some(&mut observer),
            );
            let layout = native_up_step_layout(&in_, 2, 3);
            assert!(
                matches!(layout, Ok((_, _, 9, base)) if base == WORK_BASE - 3 * NATIVE_MOTION_PIXELS * WORK_STRIDE),
                "{name}: up-copy geometry changed"
            );
        } else {
            first = compose_native_step_frame(&mut work, &mut vga, &in_);
            second =
                compose_native_step_frame_observed(&mut observed_work, &mut observed_vga, &in_, Some(&mut observer));
        }
        assert!(
            first.is_ok() && second.is_ok() && calls == 1 && work == observed_work && vga == observed_vga,
            "{name}: observer changed original output: {first:?} {second:?}"
        );
        let (before_work, before_vga) = (observed_work.clone(), observed_vga.clone());
        let mut refuse = |_: FrameStage, _: &mut [u8], _: &mut [u8]| -> Result<()> {
            Err(crate::indexedmap::Error::new("reject diagnostic stage"))
        };
        let second = if up {
            compose_native_up_step_frame_observed(&mut observed_work, &mut observed_vga, &in_, 2, 3, Some(&mut refuse))
        } else {
            compose_native_step_frame_observed(&mut observed_work, &mut observed_vga, &in_, Some(&mut refuse))
        };
        assert!(
            second.is_err() && before_work == observed_work && before_vga == observed_vga,
            "{name}: rejected stage published partial pixels"
        );
    }
}
