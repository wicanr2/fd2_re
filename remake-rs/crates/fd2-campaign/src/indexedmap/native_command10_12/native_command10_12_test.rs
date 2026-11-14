//! 移植自 Go `indexedmap/native_command10_12_test.go`。

use super::*;
use fd2_formats::fdicon::{Bank, NativeSelectorCache, NativeTerrainCell, Sprite};

struct SamplingAssets {
    terrain: Bank,
    empty: Bank,
    cache: NativeSelectorCache,
    controls: Vec<u8>,
    lut: Vec<u8>,
}

/// Go `nativeCommandSamplingTestInput` 的素材部分；Rust 的輸入結構借用素材，所以分兩步。
fn sampling_assets() -> SamplingAssets {
    let n = (NATIVE_SIZE * NATIVE_SIZE) as usize;
    let sprites = (0..12)
        .map(|_| {
            let pixels: Vec<u8> =
                (0..n).map(|p| ((p / NATIVE_SIZE as usize) * 24 + p % NATIVE_SIZE as usize) as u8).collect();
            Sprite { mask: vec![0; pixels.len()], remap_mask: vec![0; pixels.len()], pixels }
        })
        .collect();
    SamplingAssets {
        terrain: Bank { sprites },
        empty: Bank::default(),
        cache: NativeSelectorCache::default(),
        controls: vec![0; 4],
        lut: vec![0; 256],
    }
}

fn native_command_sampling_test_input(a: &SamplingAssets) -> NativeTransitionFrameInput<'_> {
    NativeTransitionFrameInput {
        terrain_bank: Some(&a.terrain),
        unit_bank: Some(&a.empty),
        foreground_bank: Some(&a.empty),
        selector_cache: Some(&a.cache),
        cells: vec![NativeTerrainCell::default(); 13 * 8],
        controls: &a.controls,
        terrain_lut: &a.lut,
        map_width: 13,
        ..Default::default()
    }
}

#[test]
fn test_compose_native_command10_to12_surface_identity_sampling() {
    let assets = sampling_assets();
    let in_ = native_command_sampling_test_input(&assets);
    let (mut work, mut vga) =
        (vec![0u8; NATIVE_UNIT_PRESENT_WORK_SIZE as usize], vec![0u8; NATIVE_MAP_VGA_SIZE as usize]);
    compose_native_command10_to12_surface(&mut work, &mut vga, &in_, 0, 0, 128).unwrap();
    assert_eq!(vga[0], 0, "first pixel");
    assert_eq!(vga[23], 23, "tile edge");
    assert_eq!(vga[VIEW_WIDTH as usize + 24], 24, "next tile row");
}

#[test]
fn test_compose_native_command10_to12_surface_rejects_without_mutation() {
    let (mut work, mut vga) =
        (vec![0u8; NATIVE_UNIT_PRESENT_WORK_SIZE as usize], vec![0u8; NATIVE_MAP_VGA_SIZE as usize]);
    (work[0], vga[0]) = (7, 9);
    let r =
        compose_native_command10_to12_surface(&mut work, &mut vga, &NativeTransitionFrameInput::default(), 0, 0, 128);
    assert!(r.is_err() && work[0] == 7 && vga[0] == 9, "malformed sampling input mutated buffers");
}
