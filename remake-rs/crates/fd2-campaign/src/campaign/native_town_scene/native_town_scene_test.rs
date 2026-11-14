//! 移植自 `remake/internal/campaign/native_town_scene_test.go`。

use super::*;
use crate::test_support::input;
use std::path::Path;

/// Go：`loadNativeTownTextAssets`
pub(crate) fn load_native_town_text_assets(base: &Path) -> (fdtxt::Strings, fdtxt::Font) {
    let strings = fdtxt::parse(&fdother::read_resource(base.join("FDTXT.DAT"), 0).unwrap()).unwrap();
    let font = fdtxt::parse_font(&fdother::read_resource(base.join("FDOTHER.DAT"), 4).unwrap()).unwrap();
    (strings, font)
}

/// Go：`TestDecodeAndComposeNativeTownUsesOriginalResources`
#[test]
fn decode_and_compose_native_town_uses_original_resources() {
    let Some(fdother_path) = input("FD2_FLAME2_DIR", "FDOTHER.DAT") else { return };
    let base = fdother_path.parent().unwrap();
    let assets =
        decode_native_town_assets_archive(&fdother_path.to_string_lossy(), &base.join("FDICON.B24").to_string_lossy())
            .unwrap();
    for (variant, background) in assets.backgrounds.iter().enumerate() {
        assert_eq!(background.len() as i64, NATIVE_TOWN_WIDTH * NATIVE_TOWN_HEIGHT, "variant {variant} background");
    }
    assert!(
        assets.label.width == 62 && assets.label.height == 26,
        "label={}x{}",
        assets.label.width,
        assets.label.height
    );
    let (strings, font) = load_native_town_text_assets(base);
    let frames: Vec<Vec<u8>> = (0..4)
        .map(|pulse| compose_native_town_frame(Some(&assets), Some(&strings), Some(&font), 0, 0, pulse).unwrap())
        .collect();
    for (pulse, frame) in frames.iter().enumerate() {
        assert_eq!(frame.len() as i64, NATIVE_TOWN_WIDTH * NATIVE_TOWN_HEIGHT, "pulse {pulse} frame");
    }
    assert!(frames[0] != frames[1] && frames[1] != frames[2], "town pulse sprites did not change the indexed frame");
    assert!(frames[1] == frames[3], "native pulse sequence did not map cycle 3 back to frame 1");
    for y in 0..NATIVE_TOWN_HEIGHT {
        for x in 0..NATIVE_TOWN_WIDTH {
            if (4..316).contains(&x) && (4..196).contains(&y) {
                continue;
            }
            assert_eq!(
                frames[0][(y * NATIVE_TOWN_WIDTH + x) as usize],
                0,
                "native viewport border ({x},{y}) is not index zero"
            );
        }
    }
}

/// Go：`TestComposeNativeTownRejectsInvalidState`
#[test]
fn compose_native_town_rejects_invalid_state() {
    assert!(compose_native_town_frame(None, None, None, 0, 0, 0).is_err(), "nil native town assets were accepted");
}

/// Go `image.Rectangle.In`（非空矩形）。
fn rect_in(r: go_image::Rectangle, s: go_image::Rectangle) -> bool {
    s.min.x <= r.min.x && r.max.x <= s.max.x && s.min.y <= r.min.y && r.max.y <= s.max.y
}

/// Go `image.Rectangle.Overlaps`（兩個都非空）。
fn rect_overlaps(r: go_image::Rectangle, s: go_image::Rectangle) -> bool {
    r.min.x < s.max.x && s.min.x < r.max.x && r.min.y < s.max.y && s.min.y < r.max.y
}

/// Go：`TestNativeTownCursorLayerReconstructsAllNativeStates`
///
/// Go 以 `os.Getenv("FD2_ASSET_PACK")` 找分離素材包，缺少時直接失敗；Rust 走 `test_support::input`。
#[test]
fn native_town_cursor_layer_reconstructs_all_native_states() {
    let Some(pack) = input("FD2_ASSET_PACK", ".") else { return };
    let Some(fdother_path) = input("FD2_FLAME2_DIR", "FDOTHER.DAT") else { return };
    let assets = load_native_town_assets(&pack.to_string_lossy()).unwrap();
    let (strings, font) = load_native_town_text_assets(fdother_path.parent().unwrap());
    // Go 以 `json.Marshal` 前後比對確認合成不改素材；Rust 以複本比對。
    let before = assets.clone();
    let label =
        go_image::Rectangle { min: go_image::Point { x: 248, y: 166 }, max: go_image::Point { x: 310, y: 192 } };
    for variant in 0..3 {
        for selection in 0..6 {
            for pulse in 0..4 {
                let original =
                    compose_native_town_frame(Some(&assets), Some(&strings), Some(&font), variant, selection, pulse)
                        .unwrap();
                let mut underlay = compose_native_town_frame_without_cursor(
                    Some(&assets),
                    Some(&strings),
                    Some(&font),
                    variant,
                    selection,
                    pulse,
                )
                .unwrap();
                let geometry = native_town_cursor_geometry_for(variant, selection, pulse).unwrap();
                let p = geometry.position;
                let rect = go_image::Rectangle { min: p, max: go_image::Point { x: p.x + 24, y: p.y + 24 } };
                assert!(
                    rect_in(rect, geometry.viewport) && !rect_overlaps(rect, label),
                    "cursor clips the viewport or label: {rect:?}"
                );
                assets.pulse[geometry.frame as usize].blit_at(&mut underlay, 320, p.x, p.y).unwrap();
                assert!(
                    original == underlay,
                    "native reconstruction differs: variant={variant} selection={selection} pulse={pulse}"
                );
            }
        }
    }
    assert!(assets == before, "composition changed source assets");
    for (v, s, p) in [(-1, 0, 0), (3, 0, 0), (0, 6, 0), (0, 0, 4)] {
        assert!(native_town_cursor_geometry_for(v, s, p).is_err(), "accepted invalid geometry {:?}", (v, s, p));
    }
}
