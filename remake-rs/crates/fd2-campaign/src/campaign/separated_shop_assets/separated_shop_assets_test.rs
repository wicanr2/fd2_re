//! 移植自 `remake/internal/campaign/separated_shop_assets_test.go`。

use super::*;
use crate::test_support::{input, temp_dir};

/// Go：`TestSeparatedNativeShopAssetsMatchFixedArchive`
#[test]
fn separated_native_shop_assets_match_fixed_archive() {
    let Some(archive) = input("FD2_FLAME2_DIR", "FDOTHER.DAT") else { return };
    let Some(pack) = input("FD2_ASSET_PACK", ".") else { return };
    let (archive, pack) = (archive.to_string_lossy().into_owned(), pack.to_string_lossy().into_owned());
    for resource in [12, 29, 63] {
        let want = decode_native_shop_assets(&archive, resource).unwrap();
        let got =
            load_separated_native_shop_assets(&pack, resource).unwrap_or_else(|e| panic!("resource {resource}: {e}"));
        assert!(
            got.background == want.background
                && got.decoration.width == want.decoration.width
                && got.decoration.height == want.decoration.height
                && got.decoration.pixels == want.decoration.pixels
                && got.gold_roll_strip.pixels == want.gold_roll_strip.pixels,
            "resource {resource} stable assets differ"
        );
        for option in 0..4 {
            for variant in 0..2 {
                assert!(
                    got.service_cells[option][variant].pixels == want.service_cells[option][variant].pixels,
                    "resource {resource} service {option}/{variant} differs"
                );
            }
        }
        assert!(
            got.price_cell.pixels == want.price_cell.pixels && got.panel.pixels == want.panel.pixels,
            "resource {resource} panel assets differ"
        );
        for i in 0..got.compare_cells.len() {
            assert!(
                got.compare_cells[i].pixels == want.compare_cells[i].pixels,
                "resource {resource} compare {i} differs"
            );
        }
        assert_eq!(got.success_frames.len(), want.success_frames.len(), "resource {resource} success frames");
        for i in 0..got.success_frames.len() {
            let (indexed, mask) = want.success_frames[i].indexed_layers().unwrap();
            let g = &got.success_frames[i];
            assert!(
                g.width == want.success_frames[i].width
                    && g.height == want.success_frames[i].height
                    && g.indexed == indexed
                    && g.mask == mask,
                "resource {resource} success frame {i} differs"
            );
        }
    }
}

/// Go：`TestSeparatedNativeShopAssetsFailClosedWithoutCompletePack`
#[test]
fn separated_native_shop_assets_fail_closed_without_complete_pack() {
    let root = temp_dir("shop-pack");
    let root_s = root.to_string_lossy().into_owned();
    assert!(load_separated_native_shop_assets(&root_s, 12).is_err(), "missing shop pack was accepted");
    let dir = root.join("shop").join("FDOTHER_012");
    std::fs::create_dir_all(&dir).unwrap();
    std::fs::write(dir.join("resource.json"), br#"{"schema_version":1}"#).unwrap();
    assert!(load_separated_native_shop_assets(&root_s, 12).is_err(), "incomplete shop metadata was accepted");
}
