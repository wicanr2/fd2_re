//! 移植自 `system_info_transition_test.go`。

use crate::fdother::*;

// Go: TestNativeSystemInfoRegionsUseAllOriginalCopies
#[test]
fn native_system_info_regions_use_all_original_copies() {
    let base: Vec<u8> = (0..64000).map(|i| (i % 251) as u8).collect();
    let surface: Vec<u8> = (0..64000).map(|i| ((i * 7 + 31) % 253) as u8).collect();
    let (opening, closing) = native_system_info_transition_frames(&base, &surface).unwrap();
    for phase in ["opening", "closing", "steady"] {
        let count = if phase == "steady" { 1 } else { 12 };
        for frame in 0..count {
            let regions = native_system_info_regions(phase, frame).unwrap();
            let mut out = base.clone();
            for region in &regions {
                let r = region.source;
                let (dx, dy) = region.destination;
                for y in 0..r.dy() {
                    let d = ((dy + y) * 320 + dx) as usize;
                    let s = ((r.min_y + y) * 320 + r.min_x) as usize;
                    let w = r.dx() as usize;
                    out[d..d + w].copy_from_slice(&surface[s..s + w]);
                }
            }
            let want = match phase {
                "opening" => opening[frame as usize].clone(),
                "closing" => closing[frame as usize].clone(),
                _ => native_system_info_steady_frame(&base, &surface).unwrap(),
            };
            assert!(out == want, "{phase} {frame}");
        }
    }
    for phase in ["", "other", "steady", "opening", "closing"] {
        for frame in [-1, 12] {
            assert!(native_system_info_regions(phase, frame).is_err(), "accepted invalid region {phase} {frame}");
        }
    }
}

// Go: TestNativeSystemInfoSteadyPreservesBattlefieldOutsidePanels
#[test]
fn native_system_info_steady_preserves_battlefield_outside_panels() {
    let baseline = vec![17u8; NATIVE_SYSTEM_INFO_BYTES];
    let information = vec![31u8; NATIVE_SYSTEM_INFO_BYTES];
    let frame = native_system_info_steady_frame(&baseline, &information).unwrap();
    for (x, y) in [(0, 0), (319, 199), (74, 100), (245, 100), (108, 25)] {
        assert_eq!(frame[y * 320 + x], 17, "battlefield changed at {x},{y}");
    }
    for (x, y) in [(109, 19), (75, 37), (75, 155), (129, 172)] {
        assert_eq!(frame[y * 320 + x], 31, "panel missing at {x},{y}");
    }
    assert_eq!(baseline[37 * 320 + 75], 17, "baseline was mutated");
}

// Go: TestNativeSystemInfoTransitionMatches1B1E7TwelvePassOrder
#[test]
fn native_system_info_transition_matches_1b1e7_twelve_pass_order() {
    let baseline = vec![0u8; NATIVE_SYSTEM_INFO_BYTES];
    let information = vec![0x7du8; NATIVE_SYSTEM_INFO_BYTES];
    let (opening, closing) = native_system_info_transition_frames(&baseline, &information).unwrap();
    assert!(opening.len() == 12 && closing.len() == 12, "transition counts");
    let pixel = |frame: &[u8], x: usize, y: usize| frame[y * 320 + x];
    assert!(
        pixel(&opening[0], 275, 37) == 0x7d && pixel(&opening[0], 274, 37) == 0 && pixel(&opening[4], 75, 37) == 0x7d,
        "horizontal opening boundary does not match 0x1AF99"
    );
    assert!(
        pixel(&opening[2], 109, 43) == 0 && pixel(&opening[3], 109, 43) == 0x7d && pixel(&opening[7], 109, 19) == 0x7d,
        "upper opening boundary does not match 0x1B019"
    );
    assert!(
        pixel(&opening[5], 75, 182) == 0 && pixel(&opening[6], 75, 182) == 0x7d && pixel(&opening[9], 75, 155) == 0x7d,
        "lower opening boundary does not match 0x1B0AD"
    );
    assert!(
        pixel(&opening[8], 129, 184) == 0
            && pixel(&opening[9], 129, 184) == 0x7d
            && pixel(&opening[11], 129, 176) == 0x7d
            && pixel(&opening[11], 129, 172) == 0,
        "footer opening boundary does not match 0x1B14B"
    );
    assert!(
        pixel(&closing[0], 75, 37) == 0x7d
            && pixel(&closing[7], 75, 37) == 0x7d
            && pixel(&closing[11], 0, 37) == 0x7d
            && pixel(&closing[11], 45, 37) == 0,
        "closing pass does not follow 11..0 0x1AF1E order"
    );
}

// Go: TestNativeSystemInfoTransitionRejectsPartialSurfacesAtomically
#[test]
fn native_system_info_transition_rejects_partial_surfaces_atomically() {
    let got = native_system_info_transition_frames(
        &vec![0u8; NATIVE_SYSTEM_INFO_BYTES - 1],
        &vec![0u8; NATIVE_SYSTEM_INFO_BYTES],
    );
    assert!(got.is_err(), "partial surfaces were accepted");
}
