//! 移植自 `fdicon_test.go`。

use super::original_fdicon_path;
use crate::fdicon::*;

const N: usize = NATIVE_SIZE as usize;

#[test]
fn parse_and_blit_preserve_transparent_dither_and_palette_band() {
    // 24 列：第一列是 run／dither／透明；其餘列全透明。
    let mut body = vec![0x00, 7, 0x40, 9, 0xd4];
    body.extend(std::iter::repeat_n(0xd7, N - 1));
    let mut raw = vec![0u8; 10 + body.len()];
    raw[0] = 24;
    raw[2] = 24;
    raw[4] = 1;
    raw[6] = 10;
    raw[10..].copy_from_slice(&body);
    let b = parse(&raw).unwrap();
    let mut dst = vec![1u8; 32 * 24];
    b.sprites[0].blit_at(&mut dst, 32, 0, 0).unwrap();
    let got = &dst[..6];
    assert!(got[0] == 7 && got[1] == 1 && got[2] == 9 && got[3] == 1 && got[4] == 1, "pixels={got:?}");
    b.sprites[0].blit_palette_band(&mut dst, 32, 0, 0).unwrap();
    let got = &dst[..6];
    assert!(got[0] == 0x1f && got[1] == 1 && got[2] == 0x19 && got[3] == 1 && got[4] == 1, "band pixels={got:?}");
    dst.fill(1);
    b.sprites[0].blit_for_native_flags(&mut dst, 32, 0, 0, 0x80).unwrap();
    assert_eq!(dst[0], 0x1f, "acted flag used raw path: {}", dst[0]);
    assert_eq!(b.sprites[0].remap_mask[4], 1, "native mode-3 span was not retained");
    let lut: Vec<u8> = (0..256).map(|i| (i + 0x20) as u8).collect();
    dst.fill(1);
    b.sprites[0].blit_lut(&mut dst, 32, 0, 0, &lut).unwrap();
    let got = &dst[..6];
    assert!(got[0] == 0x27 && got[1] == 1 && got[2] == 0x29 && got[3] == 0x21 && got[4] == 0x21, "LUT pixels={got:?}");
    assert!(b.sprites[0].blit_lut(&mut dst, 32, 0, 0, &lut[..255]).is_err(), "short LUT accepted");
    dst.fill(1);
    b.sprites[0].blit_lut_transparent(&mut dst, 32, 0, 0, &lut).unwrap();
    let got = &dst[..5];
    assert!(got[0] == 0x27 && got[2] == 0x29 && got[3] == 1 && got[4] == 1, "transparent LUT pixels={got:?}");
}

#[test]
fn decode_original_fdicon() {
    let path = original_fdicon_path();
    if !std::path::Path::new(&path).exists() {
        eprintln!("略過：player-provided FDICON.B24 is absent");
        return;
    }
    let b = decode_file(&path).unwrap();
    assert!(b.sprites.len() == 1680 && b.sprites[0].pixels.len() == N * N, "bank sprites={}", b.sprites.len());
    let got = b.sprite_for(4, 2, 1).expect("native group×12+pose×3+frame selector");
    assert_eq!(got.pixels.len(), N * N);
    assert!(b.sprite_for(4, 4, 0).is_err(), "out-of-range native pose was accepted");
}

#[test]
fn native_frame_index_matches_127e0() {
    for (motion, force, idle, moving, want) in
        [(0, false, 2, 0, 2), (0, false, 3, 2, 1), (1, false, 2, 3, 1), (7, false, 0, 2, 2), (1, true, 2, 2, 0)]
    {
        let got = native_frame_index(motion, force, idle, moving);
        assert_eq!(got, Ok(want), "({motion},{force},{idle},{moving})");
    }
    assert!(native_frame_index(0, false, 4, 0).is_err(), "invalid cycle accepted");
}

#[test]
fn native_selector_cache_matches_11019_first_seen_slots() {
    let mut cache = NativeSelectorCache::default();
    for (key, want) in [(2, 0), (0, 1), (2, 0), (1, 2), (0, 1)] {
        assert_eq!(cache.slot_for(key), Ok(want), "key={key}");
    }
    assert_eq!(cache.key_for_slot(1), Ok(0), "slot 1 key");
    assert!(cache.key_for_slot(3).is_err(), "unknown slot accepted");
    assert!(cache.slot_for(-1).is_err(), "negative raw key accepted");
    assert!(cache.slot_for(0x100).is_err(), "wide raw key accepted");
}

#[test]
fn native_selector_cache_clone_does_not_publish_preflight_slots() {
    let mut cache = NativeSelectorCache::default();
    cache.slot_for(2).unwrap();
    let mut clone = cache.clone();
    assert_eq!(clone.slot_for(7), Ok(1), "clone slot");
    assert!(cache.key_for_slot(1).is_err(), "clone allocation leaked into source cache");
    assert_eq!(clone.key_for_slot(0), Ok(2), "clone lost existing slot");
}

#[test]
fn sprite_for_native_slot_resolves_cache_key_before_b24_selector() {
    let mut bank = Bank { sprites: vec![Sprite::default(); 3 * 12] };
    bank.sprites[2 * 12 + 3] = Sprite { pixels: vec![0; N * N], mask: vec![0; N * N], remap_mask: Vec::new() };
    let mut cache = NativeSelectorCache::default();
    cache.slot_for(2).unwrap();
    let got = bank.sprite_for_native_slot(&cache, 0, 1, 0).expect("cached native selector sprite");
    assert_eq!(got.pixels.len(), N * N);
    assert!(bank.sprite_for_native_slot(&cache, 1, 0, 0).is_err(), "unknown native slot accepted");
}

#[test]
fn native_placement_offset_matches_127e0() {
    let base = NATIVE_UNIT_ORIGIN_BYTES + 2 * NATIVE_SIZE * NATIVE_MAP_STRIDE + 3 * NATIVE_SIZE;
    // 方向位移用已閉合的原始數值釘死，不要再寫成實作那條算式的複述：
    // handoff 2026-07-26 的 byte equation 是
    // 0x75d8 + (Y-cameraY)*24*0x1c8 + (X-cameraX)*24 + unit[+4]*d，
    // d 依 pose 下／左／上／右為 +0x720/-4/-0x720/+4（doc47 的方向表同值）。
    // 0x720 是 456-byte stride 的四列，也就是每拍四像素、六拍剛好一格。
    for (pose, motion, shift, force, want) in [
        (0, 2, 0, false, base + 2 * 0x720),
        (1, 2, 0, false, base - 8),
        (2, 2, 0, false, base - 2 * 0x720),
        (3, 2, 0, false, base + 8),
        (3, 1, 1, true, base + 5),
    ] {
        let got = native_placement_offset(8, 6, 5, 4, pose, motion, shift, force);
        assert_eq!(got, Ok(want), "pose={pose} motion={motion} shift={shift} force={force}");
    }
    assert!(native_placement_offset(0, 0, 0, 0, 4, 0, 0, false).is_err(), "invalid pose accepted");
    assert!(native_placement_offset(0, 0, 0, 0, 0, 0, 2, true).is_err(), "invalid native pixel shift accepted");
}

#[test]
fn blit_constant_mask_at_uses_writes_not_pixel_values() {
    let mut sprite = Sprite { pixels: vec![0; N * N], mask: vec![0; N * N], remap_mask: Vec::new() };
    sprite.mask[0] = 1;
    sprite.mask[N + 2] = 1;
    let mut dst = vec![7u8; 32 * 32];
    sprite.blit_constant_mask_at(&mut dst, 32, 3, 4, 0xc0).unwrap();
    assert!(
        dst[4 * 32 + 3] == 0xc0 && dst[5 * 32 + 5] == 0xc0,
        "masked pixels were not filled: {:x} {:x}",
        dst[4 * 32 + 3],
        dst[5 * 32 + 5]
    );
    assert_eq!(dst[4 * 32 + 4], 7, "mode-3/unwritten pixel changed");
    let before = dst.clone();
    assert!(sprite.blit_constant_mask_at(&mut dst, 32, -1, 0, 0xc0).is_err(), "out-of-bounds destination accepted");
    assert_eq!(dst, before, "rejected constant-mask blit mutated destination");
}
