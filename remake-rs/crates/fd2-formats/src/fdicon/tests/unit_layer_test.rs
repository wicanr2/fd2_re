//! 移植自 `unit_layer_test.go`。

use crate::fdicon::*;

const N: usize = NATIVE_SIZE as usize;

pub(super) fn native_layer_bank() -> Bank {
    let sprites = (0..12)
        .map(|i| {
            let mut pixels = vec![0u8; N * N];
            let mut mask = vec![0u8; N * N];
            pixels[0] = 0x10 + i as u8;
            mask[0] = 1;
            Sprite { pixels, mask, remap_mask: vec![0; N * N] }
        })
        .collect();
    Bank { sprites }
}

fn base() -> usize {
    native_placement_offset(0, 0, 0, 0, 0, 0, 0, false).unwrap() as usize
}

#[test]
fn blit_native_unit_layer_matches_127e0_order_and_branches() {
    let mut cache = NativeSelectorCache::default();
    cache.slot_for(0).unwrap();
    let mut dst = vec![0u8; (NATIVE_MAP_STRIDE * 100) as usize];
    let units = [
        NativeUnitLayerEntry { x: 0, y: 0, slot: 0, pose: 0, flags: 0, ..Default::default() },
        NativeUnitLayerEntry { x: 1, y: 0, slot: 0, pose: 0, flags: 0x80, ..Default::default() },
        NativeUnitLayerEntry { x: 2, y: 0, slot: 0, pose: 0, inactive: true, ..Default::default() },
    ];
    native_layer_bank()
        .blit_native_unit_layer(&mut dst, NATIVE_MAP_STRIDE, &cache, &units, 0, 0, 12, 7, 0, 0, 0)
        .unwrap();
    let base = base();
    assert_eq!(dst[base], 0x10, "raw unit pixel");
    assert_eq!(dst[base + N], 0x18, "band unit pixel");
    assert_eq!(dst[base + 2 * N], 0, "inactive unit wrote");
}

#[test]
fn blit_native_unit_layer_fails_before_partial_frame() {
    let mut cache = NativeSelectorCache::default();
    cache.slot_for(0).unwrap();
    let mut dst = vec![0u8; (NATIVE_MAP_STRIDE * 100) as usize];
    let units = [
        NativeUnitLayerEntry { x: 0, y: 0, slot: 0, pose: 0, ..Default::default() },
        NativeUnitLayerEntry { x: 1, y: 0, slot: 7, pose: 0, ..Default::default() },
    ];
    let result =
        native_layer_bank().blit_native_unit_layer(&mut dst, NATIVE_MAP_STRIDE, &cache, &units, 0, 0, 12, 7, 0, 0, 0);
    assert!(result.is_err(), "unknown slot was accepted");
    assert_eq!(dst[base()], 0, "failed layer partially modified destination");
}

#[test]
fn blit_native_unit_layer_with_base_shift_matches_32999_pointer_lift() {
    let mut cache = NativeSelectorCache::default();
    cache.slot_for(0).unwrap();
    let mut dst = vec![0u8; (NATIVE_MAP_STRIDE * 100) as usize];
    let unit = NativeUnitLayerEntry::default();
    native_layer_bank()
        .blit_native_unit_layer_with_base_shift(
            &mut dst,
            NATIVE_MAP_STRIDE,
            &cache,
            &[unit],
            0,
            0,
            12,
            7,
            0,
            0,
            0,
            -8 * NATIVE_MAP_STRIDE,
        )
        .unwrap();
    let base = base();
    let lifted = base - 8 * NATIVE_MAP_STRIDE as usize;
    assert!(
        dst[base] == 0 && dst[lifted] == 0x10,
        "shifted unit wrote base={:#x} lifted={:#x}",
        dst[base],
        dst[lifted]
    );
}
