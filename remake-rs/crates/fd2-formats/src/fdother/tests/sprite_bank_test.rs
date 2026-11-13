//! 移植自 `sprite_bank_test.go`。

use super::{original, pack};
use crate::fdicon::{SEPARATED_FDSHAP_MAP_COUNT, load_separated_fdshap_bank};
use crate::fdother::*;

#[test]
fn decode_fdshap000_sprite_bank_from_player_archive() {
    let Some(path) = original("FDSHAP.DAT") else { return };
    assert_eq!(decode_sprite_bank_resource(&path, 0).unwrap().sprites.len(), 288, "FDSHAP#0 sprite count");
}

#[test]
fn separated_fdshap_banks_match_all_player_archive_resources() {
    let Some(path) = original("FDSHAP.DAT") else { return };
    let Some(root) = pack("tilesets/fdshap", "map_00/bank.json") else { return };
    for map_index in 0..SEPARATED_FDSHAP_MAP_COUNT {
        let (want_bank, want_controls) = decode_map_terrain_resources(&path, map_index).unwrap();
        let (got_bank, got_controls) = load_separated_fdshap_bank(&root, map_index).unwrap();
        assert!(
            got_controls == want_controls && got_bank.sprites.len() == want_bank.sprites.len(),
            "map{map_index} sprites={}/{}",
            got_bank.sprites.len(),
            want_bank.sprites.len()
        );
        for (index, (g, w)) in got_bank.sprites.iter().zip(&want_bank.sprites).enumerate() {
            assert_eq!(g, w, "map{map_index} tile{index} separated layers differ");
        }
    }
}

#[test]
fn decode_map_terrain_resources_pairs_fdshap_map_zero() {
    let Some(path) = original("FDSHAP.DAT") else { return };
    let (bank, controls) = decode_map_terrain_resources(&path, 0).unwrap();
    assert!(
        bank.sprites.len() == 288 && controls.len() == 1200,
        "map0 image/control={}/{}",
        bank.sprites.len(),
        controls.len()
    );
    assert!(decode_map_terrain_resources(&path, -1).is_err(), "negative map index accepted");
}

#[test]
fn decode_map_terrain_resources_allows_trailing_renderer_frames() {
    let Some(path) = original("FDSHAP.DAT") else { return };
    let (bank, controls) = decode_map_terrain_resources(&path, 17).unwrap();
    assert!(
        bank.sprites.len() == 384 && controls.len() == 330 * 4,
        "map17 sprites={} controls={}",
        bank.sprites.len(),
        controls.len()
    );
}
