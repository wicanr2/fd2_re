//! 移植自 `remake/internal/campaign/native_town_departure_test.go`。

use super::*;

/// Go：`TestNativeTownDepartureGeometryPreservesSourceSampling`
#[test]
fn native_town_departure_geometry_preserves_source_sampling() {
    let source: Vec<u8> = (0..64000usize).map(|i| ((i * 17 + i / 320 * 31) % 251 + 1) as u8).collect();
    let before = source.clone();
    // The exit coordinates are pinned by the canonical 18-byte EXE tables.
    for (variant, (ex, ey)) in [(59i64, 163i64), (53, 163), (59, 163)].into_iter().enumerate() {
        let variant = variant as i64;
        for step in 1..=10 {
            let got = compose_native_town_departure_frame(&source, variant, 2, step).unwrap();
            let spacing = 128 - 9 * step;
            let cx = 128 * (160 + (ex - 150) * step / 10);
            let cy = 128 * (100 + (ey - 100) * step / 10);
            let geometry = native_town_departure_geometry_for(variant, 2, step);
            assert!(
                geometry.as_ref().is_ok_and(|g| g.spacing == spacing && g.center.x == cx && g.center.y == cy),
                "changed fixed-point geometry {variant} {step} {geometry:?}"
            );
            for y in 0..200i64 {
                for x in 0..320i64 {
                    let (sx, sy) = (cx + (x - 160) * spacing, cy + (y - 100) * spacing);
                    let mut want = 0u8;
                    if (0..320 * 128).contains(&sx) && (0..200 * 128).contains(&sy) {
                        want = source[(sy / 128 * 320 + sx / 128) as usize];
                    }
                    assert_eq!(
                        got[(y * 320 + x) as usize],
                        want,
                        "source sample changed v{variant} step{step} ({x},{y})"
                    );
                }
            }
        }
    }
    assert!(source == before, "source changed");
    for (v, s, st) in [(-1, 2, 1), (3, 2, 1), (0, -1, 1), (0, 6, 1), (0, 2, 0), (0, 2, 11)] {
        assert!(native_town_departure_geometry_for(v, s, st).is_err(), "invalid geometry accepted {:?}", (v, s, st));
    }
}
