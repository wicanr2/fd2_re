//! 移植自 `remake/internal/campaign/separated_town_assets_test.go`（PORTING.md 的測試檔歸屬：歸
//! `native_town_scene.go`）。

use super::native_town_scene_test::load_native_town_text_assets;
use super::*;
use crate::test_support::{input, temp_dir};

/// Go：`TestSeparatedNativeTownAssetsMatchFixedArchive`
#[test]
fn separated_native_town_assets_match_fixed_archive() {
    let Some(fdother_path) = input("FD2_FLAME2_DIR", "FDOTHER.DAT") else { return };
    let Some(pack) = input("FD2_ASSET_PACK", ".") else { return };
    let base = fdother_path.parent().unwrap();
    let want =
        decode_native_town_assets_archive(&fdother_path.to_string_lossy(), &base.join("FDICON.B24").to_string_lossy())
            .unwrap();
    let got = load_native_town_assets(&pack.to_string_lossy()).unwrap();
    for variant in 0..got.backgrounds.len() {
        assert!(got.backgrounds[variant] == want.backgrounds[variant], "variant {variant} background differs");
    }
    assert!(
        got.label.width == want.label.width
            && got.label.height == want.label.height
            && got.label.pixels == want.label.pixels,
        "town label differs"
    );
    let (strings, font) = load_native_town_text_assets(base);
    for variant in 0..3 {
        for selection in 0..6 {
            for pulse in 0..4 {
                let a = compose_native_town_frame(Some(&want), Some(&strings), Some(&font), variant, selection, pulse)
                    .unwrap();
                let b = compose_native_town_frame(Some(&got), Some(&strings), Some(&font), variant, selection, pulse)
                    .unwrap();
                assert!(a == b, "composite differs variant={variant} selection={selection} pulse={pulse}");
            }
        }
    }
}

/// Go：`TestSeparatedNativeTownAssetsFailClosedWithoutCompletePack`
#[test]
fn separated_native_town_assets_fail_closed_without_complete_pack() {
    let dir = temp_dir("town-pack");
    assert!(load_native_town_assets(&dir.to_string_lossy()).is_err(), "incomplete town pack was accepted");
}
