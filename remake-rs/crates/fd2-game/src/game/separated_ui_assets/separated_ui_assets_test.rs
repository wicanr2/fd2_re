//! Go 來源：`remake/cmd/fd2/separated_ui_assets_test.go`。
//!
//! Go 另外三個測試（`TestSeparatedCommandGridLoadsWithoutOriginalArchive`、
//! `TestSeparatedCommandGridFailsClosedOnMissingCell`、`TestSeparatedPaletteRejectsOutOfRangeDAC`）
//! 以 `t.Setenv("FD2_ASSET_PACK", <暫存素材包>)` 改寫環境變數；Rust 2024 的 `std::env::set_var`
//! 是 `unsafe`，工作區又 `forbid(unsafe_code)`，所以不移植，列在 `PORTING.md`「A 組測試」。

use super::*;
use crate::test_support;

/// Go：`separated_ui_assets_test.go:149` `TestNativeBattlePaletteMatchesFixedArchiveResourceZero`
///
/// Go 以相對路徑設定 `FD2_ASSET_PACK`；Rust 由 `tools/rust_game_test.sh` 設定同一個素材包。
#[test]
fn test_native_battle_palette_matches_fixed_archive_resource_zero() {
    let Some(archive) = test_support::input("FD2_FLAME2_DIR", "FDOTHER.DAT") else {
        return;
    };
    if test_support::input("FD2_ASSET_PACK", "palette/fdother_000.json").is_none() {
        return;
    }
    let (dac, palette) = load_native_battle_palette().unwrap();
    let want = fd2_formats::archive::read_resource(&archive, 0).unwrap();
    assert_eq!(dac, want, "separated FDOTHER #0 DAC differs from fixed archive oracle");
    let want_palette = color_palette_from_nrgba(&fdother::parse_vga_palette(&want).unwrap());
    assert_eq!(palette.len(), want_palette.len());
    for (index, (got, want)) in palette.iter().zip(&want_palette).enumerate() {
        assert_eq!(got, want, "palette color {index} differs from fixed archive oracle");
    }
}

/// 分離素材包的指令環格數與 `NATIVE_ACTION_OVERLAY_CELL_COUNT` 相同，且每格都轉成畫面
/// （Go `TestSeparatedCommandGridLoadsWithoutOriginalArchive` 對真實素材包的對應）。
#[test]
fn separated_command_grid_loads_from_real_pack() {
    if test_support::input("FD2_ASSET_PACK", "ui/action_cells/resource.json").is_none() {
        return;
    }
    let palette = load_native_ui_palette();
    assert_eq!(palette.len(), 256);
    let (images, cells) = load_native_action_cells_with_raw(&palette);
    assert_eq!(images.len() as i64, NATIVE_ACTION_OVERLAY_CELL_COUNT);
    assert_eq!(cells.len(), images.len());
    for (image, cell) in images.iter().zip(&cells) {
        let image = image.as_ref().unwrap();
        assert_eq!((image.width, image.height), (cell.width, cell.height));
    }
    assert!(load_native_action_cells_with_raw(&palette[..255].to_vec()).0.is_empty());
}
