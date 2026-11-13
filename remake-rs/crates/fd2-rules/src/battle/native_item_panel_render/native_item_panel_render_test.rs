//! 移植自 `remake/internal/battle/native_item_panel_render_test.go`。

use super::*;
use crate::test_support::{input, temp_dir};

const PANEL: usize = NATIVE_ITEM_PANEL_BYTES as usize;

fn cells_with(pixel: impl Fn(usize) -> u8) -> Vec<fdother::RawCell> {
    (0..18).map(|i| fdother::RawCell { width: 1, height: 1, pixels: vec![pixel(i)] }).collect()
}

/// Go：`TestRenderNativeItemPanelBasePreservesNativeOrderAndOpaqueZero`
#[test]
fn render_native_item_panel_base_preserves_native_order_and_opaque_zero() {
    let cells = cells_with(|i| i as u8);
    // Grid entry 13 first writes at portrait origin 0x0c88. Portrait must
    // overwrite it, including its zero pixel.
    let portrait = dato::Frame { width: 2, height: 1, pixels: vec![0, 0x77] };
    let upper = fdother::LMI1Entry { width: 2, height: 1, pixels: vec![0, 0x55] };
    let bottom = fdother::LMI1Entry { width: 1, height: 1, pixels: vec![0x66] };
    let mut dst = vec![0x99u8; PANEL];
    render_native_item_panel_base(&cells, &portrait, &upper, &bottom, &mut dst).unwrap();
    assert!(
        dst[2245] == 1 && dst[3208] == 0 && dst[3209] == 0x77,
        "grid/portrait={:#x}/{:#x}/{:#x}",
        dst[2245],
        dst[3208],
        dst[3209]
    );
    let at = 7 * 320 + 92;
    assert!(dst[at] == 0 && dst[at + 1] == 0x55, "upper={:#x}/{:#x}", dst[at], dst[at + 1]);
    assert_eq!(dst[94 * 320 + 5], 0x66, "bottom");
}

/// Go：`TestRenderNativeItemPanelBaseIsAtomicOnInvalidSource`
#[test]
fn render_native_item_panel_base_is_atomic_on_invalid_source() {
    let cells = cells_with(|_| 1);
    let mut dst = vec![0x44u8; PANEL];
    let before = dst.clone();
    let bad = fdother::LMI1Entry { width: 2, height: 1, pixels: vec![1] };
    let portrait = dato::Frame { width: 1, height: 1, pixels: vec![2] };
    assert!(
        render_native_item_panel_base(&cells, &portrait, &bad, &bad, &mut dst).is_err(),
        "invalid source was accepted"
    );
    assert!(dst == before, "failed item panel render mutated destination");
}

/// Go：`TestRenderNativeItemPanelBaseWithPlayerAssets`
#[test]
fn render_native_item_panel_base_with_player_assets() {
    let (Some(fdother_path), Some(_)) =
        (input("FD2_FLAME2_DIR", "FDOTHER.DAT"), input("FD2_ASSET_PACK", "portraits/DATO_000_m0.png"))
    else {
        return;
    };
    let fdother_path = fdother_path.to_string_lossy().into_owned();
    let asset_pack_root = std::env::var("FD2_ASSET_PACK").unwrap();
    let portrait_root = format!("{asset_pack_root}/portraits");
    let mut dst = vec![0u8; PANEL];
    render_native_item_panel_base_resources(&asset_pack_root, &portrait_root, 0, &mut dst).unwrap();
    let mut oracle = vec![0u8; PANEL];
    render_native_item_panel_base_resources_archive(&fdother_path, &portrait_root, 0, &mut oracle).unwrap();
    assert!(dst == oracle, "separated native item panel base differs from fixed archive");
    let nonzero = dst.iter().filter(|&&p| p != 0).count();
    assert!(nonzero >= 1000, "player item panel nonzero pixels={nonzero}");
}

/// Go：`TestRenderNativeItemPanelBaseResourcesFailsClosedWithoutSeparatedPack`
#[test]
fn render_native_item_panel_base_resources_fails_closed_without_separated_pack() {
    let mut dst = vec![0u8; PANEL];
    let before = dst.clone();
    let (a, b) = (temp_dir("panel-pack"), temp_dir("panel-portraits"));
    assert!(
        render_native_item_panel_base_resources(&a.to_string_lossy(), &b.to_string_lossy(), 0, &mut dst).is_err(),
        "missing separated item panel base was accepted"
    );
    assert!(dst == before, "missing separated item panel base mutated destination");
}
