//! 移植自 Go `indexedmap/hud_test.go`。

use super::*;
use crate::indexedmap::test_util::{bank, frame, hud_frames, solid};
use crate::test_support;
use fd2_formats::fdicon::NativeSelectorCache;

const STRIDE: usize = NATIVE_MAP_STRIDE as usize;

fn cache0() -> NativeSelectorCache {
    let mut cache = NativeSelectorCache::default();
    cache.slot_for(0).unwrap();
    cache
}

fn layout1() -> fdicon::NativeMapHUDLayout {
    fdicon::native_map_hud_layout_for(1, NATIVE_MAP_STRIDE).unwrap()
}

fn px(dst: &[u8], off: i64) -> u8 {
    dst[off as usize]
}

#[test]
fn test_separated_native_map_hud_frames_match_fixed_archive() {
    let Some(archive) = test_support::input("FD2_FLAME2_DIR", "FDOTHER.DAT") else {
        return;
    };
    let Some(ui) = test_support::input("FD2_ASSET_PACK", "ui") else {
        return;
    };
    let want = decode_native_map_hud_frames(&archive).unwrap();
    let got = load_separated_native_map_hud_frames(&ui).unwrap();
    // 兩邊的內部表示本來就不一樣：原版封存經 ParseSingleFrame 之後帶的是
    // 0x4E63D 的 RLE 位元組（Frame.Pixels），分離包帶的是已展開的
    // Indexed+Mask。可比的是**畫出來的結果**，所以兩邊各自 blit 到同一張
    // 底圖再逐位元組比。
    let render = |label: &str, frame: &Frame| -> Vec<u8> {
        let stride = frame.x + frame.width;
        let mut dst = vec![0xffu8; (stride * (frame.y + frame.height)) as usize]; // 透明哨兵；0x4E63D 的透明分支不會寫入這些位元組
        frame.blit(&mut dst, stride, -1).unwrap_or_else(|e| panic!("{label} blit 失敗：{e}"));
        dst
    };
    let compare = |label: &str, got: &Frame, want: &Frame| {
        assert!(
            got.width == want.width && got.height == want.height && got.x == want.x && got.y == want.y,
            "{label} 版面 ({},{}) {}x{}，封存 ({},{}) {}x{}",
            got.x,
            got.y,
            got.width,
            got.height,
            want.x,
            want.y,
            want.width,
            want.height
        );
        let got_pixels = render(&format!("{label}（分離包）"), got);
        let want_pixels = render(&format!("{label}（封存）"), want);
        for i in 0..got_pixels.len() {
            assert_eq!(
                got_pixels[i],
                want_pixels[i],
                "{label} 第 {i} 個像素不同：分離包 {:#x}、封存 {:#x}（共 {}）",
                got_pixels[i],
                want_pixels[i],
                got_pixels.len()
            );
        }
    };
    compare("panel", &got.panel, &want.panel);
    compare("positive", &got.positive_sign, &want.positive_sign);
    compare("negative", &got.negative_sign, &want.negative_sign);
    compare("equal overflow", &got.hp_equal_overflow, &want.hp_equal_overflow);
    compare("mismatch overflow", &got.hp_mismatch_overflow, &want.hp_mismatch_overflow);
    for index in 0..got.digits.len() {
        compare("digit", &got.digits[index], &want.digits[index]);
        compare("mismatch digit", &got.hp_mismatch_digits[index], &want.hp_mismatch_digits[index]);
    }
}

#[test]
fn test_separated_native_map_hud_frames_fail_closed() {
    let dir = test_support::temp_dir("hud-separated");
    assert!(load_separated_native_map_hud_frames(&dir).is_err(), "missing separated map HUD bank was accepted");
}

#[test]
fn test_advance_native_map_hud_anchor_matches1_ad2_a() {
    for (anchor, raw53abd, raw53ab9, want) in
        [(1, 6, 2, 0xf2), (0xf2, 6, 10, 1), (0xf2, 5, 2, 0xf2), (0xf2, 6, 3, 0xf2), (0xf2, 6, 9, 0xf2), (7, 99, 7, 7)]
    {
        let got = advance_native_map_hud_anchor(anchor, raw53abd, raw53ab9);
        assert_eq!(got, want, "anchor={anchor:#x} raw=({raw53abd},{raw53ab9}): got {got:#x}, want {want:#x}");
    }
}

#[test]
fn test_advance_native_map_hud_state_matches1297_d() {
    for (state, timer_tick, last, want_state, want_last) in
        [(0, 10, 5, 1, 10), (3, 10, 5, 0, 10), (2, 8, 10, 3, 8), (1, 8, 10, 2, 8)]
    {
        let (got_state, got_last) = advance_native_map_hud_state(state, timer_tick, last);
        assert!(
            got_state == want_state && got_last == want_last,
            "raw=(state={state} timer={timer_tick} last={last}): got ({got_state},{got_last}), want ({want_state},{want_last})"
        );
    }
}

#[test]
fn test_advance_native_map_sprite_cycles_matches_complete1297_d() {
    let tests = [
        ("within gate still advances moving", 2, 1, 13, 10, 2, 2, 10),
        ("positive boundary four stays closed", 3, 3, 14, 10, 3, 0, 10),
        ("positive delta five advances both", 3, 2, 15, 10, 0, 3, 15),
        ("negative delta advances idle and moving", 1, 3, 9, 10, 2, 0, 9),
    ];
    for (name, idle, moving, timer_tick, last, want_idle, want_moving, want_last) in tests {
        let got = advance_native_map_sprite_cycles(idle, moving, timer_tick, last);
        assert_eq!(got, (want_idle, want_moving, want_last), "{name}");
    }
}

#[test]
fn test_native_map_hud_optional_unit_eligible_matches1_ae2_a() {
    for (b7, b1f, b6, want) in [(0x79u8, 0u8, 0u8, false), (0, 0x0a, 1, false), (0, 0x0a, 0, true), (0, 9, 1, true)] {
        let got = native_map_hud_optional_unit_eligible(b7, b1f, b6);
        assert_eq!(got, want, "raw=({b7:#x},{b1f:#x},{b6:#x})");
    }
}

#[test]
fn test_blit_native_map_hud_panel_gates_and_origin() {
    let mut dst = vec![0u8; STRIDE * 200];
    blit_native_map_hud_panel(&hud_frames(), &mut dst, true, false, 1).unwrap();
    let layout = layout1();
    assert_eq!(px(&dst, layout.frame), 0, "closed display gate drew panel");
    blit_native_map_hud_panel(&hud_frames(), &mut dst, true, true, 1).unwrap();
    assert_eq!(px(&dst, layout.frame), 0x5a, "panel byte");
}

#[test]
fn test_blit_native_map_hud_panel_rejects_invalid_entry_before_write() {
    let mut frames = hud_frames();
    frames.panel = frame(1, 1, 7);
    let (mut dst, before) = (vec![0u8; STRIDE * 200], vec![0u8; STRIDE * 200]);
    assert!(blit_native_map_hud_panel(&frames, &mut dst, true, true, 1).is_err(), "wrong panel geometry accepted");
    assert_eq!(dst, before, "rejected panel mutated destination");
}

#[test]
fn test_blit_native_map_hud_composes_recovered_subpasses_atomically() {
    let mut terrain = bank(2, 0);
    terrain.sprites[1] = solid(0x66);
    let cache = cache0();
    let mut units = bank(12, 0);
    units.sprites[1] = solid(0x77);
    let mut in_ = NativeMapHUDInput {
        display_gate_a: true,
        display_gate_b: true,
        anchor_x: 1,
        terrain_descriptor: 1,
        terrain_control: 2,
        optional_unit: Some(NativeMapHUDOptionalUnit { selector_slot: 0, raw_state: 3, current: 7, maximum: 8 }),
    };
    let mut dst = vec![0u8; STRIDE * 200];
    let mut terrain_only = dst.clone();
    let mut terrain_input = in_;
    terrain_input.optional_unit = None;
    blit_native_map_hud(&hud_frames(), Some(&terrain), None, None, &mut terrain_only, &terrain_input).unwrap();
    let layout = layout1();
    assert_eq!(px(&terrain_only, layout.terrain), 0x66, "terrain-only HUD byte");
    blit_native_map_hud(&hud_frames(), Some(&terrain), Some(&units), Some(&cache), &mut dst, &in_).unwrap();
    // Native draws the optional unit icon after terrain at the same row-5
    // destination, so the unit is the final byte when both subpasses run.
    assert!(
        layout.terrain == layout.unit
            && px(&dst, layout.frame) == 0x5a
            && px(&dst, layout.terrain) == 0x77
            && px(&dst, layout.ap) == 0x42
            && px(&dst, layout.dp) == 0x31
            && px(&dst, layout.unit) == 0x77
            && px(&dst, layout.hp) == 0x70,
        "HUD composition={:#x}/{:#x}/{:#x}/{:#x}/{:#x}/{:#x}",
        px(&dst, layout.frame),
        px(&dst, layout.terrain),
        px(&dst, layout.ap),
        px(&dst, layout.dp),
        px(&dst, layout.unit),
        px(&dst, layout.hp)
    );
    let before = dst.clone();
    in_.terrain_control = 6;
    assert!(
        blit_native_map_hud(&hud_frames(), Some(&terrain), Some(&units), Some(&cache), &mut dst, &in_).is_err(),
        "invalid terrain control accepted"
    );
    assert_eq!(dst, before, "failed full HUD composition mutated destination");
    in_.display_gate_b = false;
    blit_native_map_hud(&hud_frames(), None, None, None, &mut dst, &in_).unwrap();
    assert_eq!(dst, before, "closed display gates mutated destination");
}

#[test]
fn test_blit_native_map_hud_terrain_icon_uses12_e38_tile_at_panel_plus6() {
    let mut terrain = bank(2, 0);
    terrain.sprites[1] = solid(0x66);
    let mut dst = vec![0u8; STRIDE * 200];
    blit_native_map_hud_terrain_icon(Some(&terrain), &mut dst, 1, 1).unwrap();
    let layout = layout1();
    assert_eq!(px(&dst, layout.terrain), 0x66, "terrain icon");
    let before = dst.clone();
    assert!(
        blit_native_map_hud_terrain_icon(Some(&terrain), &mut dst, 1, 2).is_err(),
        "out-of-bank terrain descriptor accepted"
    );
    assert_eq!(dst, before, "rejected terrain descriptor mutated HUD");
}

#[test]
fn test_blit_native_map_hud_unit_icon_uses_cache_and_aliases_state_three() {
    let cache = cache0();
    let mut units = bank(12, 0);
    units.sprites[1] = solid(0x77); // cache key 0, pose 0, aliased cycle 1
    let mut dst = vec![0u8; STRIDE * 200];
    blit_native_map_hud_unit_icon(Some(&units), Some(&cache), &mut dst, 1, 0, 3).unwrap();
    let layout = layout1();
    assert_eq!(px(&dst, layout.unit), 0x77, "unit icon");
    let before = dst.clone();
    assert!(
        blit_native_map_hud_unit_icon(Some(&units), Some(&cache), &mut dst, 1, 0, 4).is_err(),
        "invalid raw state accepted"
    );
    assert_eq!(dst, before, "rejected unit icon selector mutated HUD");
}

#[test]
fn test_native_map_hud_terrain_apdp_matches1_acf3_tables() {
    for (code, want) in [(0u8, (5, 0)), (1, (0, 0)), (2, (-5, 10)), (3, (-5, 10)), (4, (-5, -5)), (5, (0, 0))] {
        let got = native_map_hud_terrain_apdp(code);
        assert_eq!(got.as_ref().ok(), Some(&want), "code={code} got={got:?}");
    }
    assert!(native_map_hud_terrain_apdp(6).is_err(), "invalid control byte accepted");
}

#[test]
fn test_blit_native_map_hud_terrain_apdp_uses_layout_and_is_atomic() {
    let mut dst = vec![0u8; STRIDE * 200];
    blit_native_map_hud_terrain_apdp(&hud_frames(), &mut dst, 1, 2).unwrap();
    let l = layout1();
    let cells = [
        px(&dst, l.ap),
        px(&dst, l.ap + 8),
        px(&dst, l.ap + 14),
        px(&dst, l.dp),
        px(&dst, l.dp + 8),
        px(&dst, l.dp + 14),
    ];
    assert_eq!(cells, [0x42, 0x50, 0x55, 0x31, 0x51, 0x50], "unexpected AP/DP cells: {cells:#x?}");
    let before = dst.clone();
    assert!(blit_native_map_hud_terrain_apdp(&hud_frames(), &mut dst, 1, 6).is_err(), "invalid control byte accepted");
    assert_eq!(dst, before, "invalid terrain control mutated HUD");
}

#[test]
fn test_blit_native_map_hud_hp_matches1875_d_and187_d6() {
    let layout = layout1();
    for (current, maximum, first, second, third) in [(7u16, 7u16, 0x50u8, 0x50u8, 0x57u8), (7, 8, 0x70, 0x70, 0x77)] {
        let mut dst = vec![0u8; STRIDE * 200];
        blit_native_map_hud_hp(&hud_frames(), &mut dst, 1, current, maximum).unwrap();
        let got = [px(&dst, layout.hp), px(&dst, layout.hp + 6), px(&dst, layout.hp + 12)];
        assert_eq!(got, [first, second, third], "HP {current}/{maximum}");
    }
    for (maximum, want) in [(1000u16, 0x7au8), (1001, 0x7b)] {
        let mut dst = vec![0u8; STRIDE * 200];
        blit_native_map_hud_hp(&hud_frames(), &mut dst, 1, 1000, maximum).unwrap();
        assert_eq!(px(&dst, layout.hp), want, "HP overflow 1000/{maximum}");
    }
    let (mut frames, mut dst) = (hud_frames(), vec![0u8; STRIDE * 200]);
    frames.hp_mismatch_overflow = frame(1, 1, 0);
    let before = dst.clone();
    assert!(blit_native_map_hud_hp(&frames, &mut dst, 1, 1000, 1001).is_err(), "invalid overflow geometry accepted");
    assert_eq!(dst, before, "rejected HP overflow mutated HUD");
}

#[test]
fn test_blit_native_map_hud_signed_number_selects_sign_and_absolute_value() {
    let mut dst = vec![0u8; STRIDE * 30];
    let (mut called_origin, mut called_absolute) = (-1i64, -1i64);
    let origin = NATIVE_MAP_STRIDE + 10;
    {
        let mut draw = |frame: &mut [u8], origin: i64, absolute: i64| -> Result<()> {
            (called_origin, called_absolute) = (origin, absolute);
            frame[origin as usize] = 0x5a;
            Ok(())
        };
        blit_native_map_hud_signed_number(&hud_frames(), &mut dst, origin, 12, Some(&mut draw)).unwrap();
    }
    assert!(
        px(&dst, origin) == 0x31
            && px(&dst, origin + 8) == 0x5a
            && called_origin == origin + 8
            && called_absolute == 12,
        "positive sign/digits mismatch: sign={:#x} origin={called_origin} absolute={called_absolute}",
        px(&dst, origin)
    );
    {
        let mut draw = |frame: &mut [u8], origin: i64, absolute: i64| -> Result<()> {
            (called_origin, called_absolute) = (origin, absolute);
            frame[origin as usize] = 0x5a;
            Ok(())
        };
        blit_native_map_hud_signed_number(&hud_frames(), &mut dst, origin, -9, Some(&mut draw)).unwrap();
    }
    assert!(
        px(&dst, origin) == 0x42 && called_absolute == 9,
        "negative sign/absolute mismatch: sign={:#x} absolute={called_absolute}",
        px(&dst, origin)
    );
}

#[test]
fn test_blit_native_map_hud_signed_number_is_atomic_on_digit_failure() {
    let mut dst = vec![0u8; STRIDE * 20];
    let before = dst.clone();
    let mut fail = |_: &mut [u8], _: i64, _: i64| -> Result<()> { Err(Error::new("digits")) };
    assert!(
        blit_native_map_hud_signed_number(&hud_frames(), &mut dst, 1, 1, Some(&mut fail)).is_err(),
        "digit failure accepted"
    );
    assert_eq!(dst, before, "digit failure partially drew sign");
}

#[test]
fn test_blit_native_map_hud_two_digit_number_matches187_d6_call_slice() {
    let mut dst = vec![0u8; STRIDE * 30];
    let origin = NATIVE_MAP_STRIDE + 10;
    blit_native_map_hud_two_digit_number(&hud_frames(), &mut dst, origin, -12).unwrap();
    let got = [px(&dst, origin), px(&dst, origin + 8), px(&dst, origin + 14)];
    assert_eq!(got, [0x42, 0x51, 0x52], "sign/digits");
    let before = dst.clone();
    assert!(
        blit_native_map_hud_two_digit_number(&hud_frames(), &mut dst, origin, 100).is_err(),
        "three-digit value accepted"
    );
    assert_eq!(dst, before, "rejected value mutated HUD");
}

#[test]
fn test_decode_native_map_hud_frames_uses_four_mode_directory_entries() {
    let Some(dat_path) = test_support::input("FD2_FLAME2_DIR", "FDOTHER.DAT") else {
        return;
    };
    let frames = decode_native_map_hud_frames(&dat_path).unwrap();
    assert!(
        frames.panel.width == 69
            && frames.panel.height == 34
            && frames.positive_sign.width == 6
            && frames.positive_sign.height == 7
            && frames.negative_sign.width == 6
            && frames.negative_sign.height == 5,
        "frames={frames:?}"
    );
    for (i, digit) in frames.digits.iter().enumerate() {
        let want_width = if i == 1 { 5 } else { 6 };
        assert!(digit.width == want_width && digit.height == 8, "digit {i}={}x{}", digit.width, digit.height);
    }
    for (i, digit) in frames.hp_mismatch_digits.iter().enumerate() {
        let want_width = if i == 1 { 5 } else { 6 };
        assert!(digit.width == want_width && digit.height == 8, "mismatch digit {i}={}x{}", digit.width, digit.height);
    }
    assert!(
        frames.hp_equal_overflow.width == 18
            && frames.hp_equal_overflow.height == 8
            && frames.hp_mismatch_overflow.width == 18
            && frames.hp_mismatch_overflow.height == 8,
        "HP overflow={}x{}/{}x{}",
        frames.hp_equal_overflow.width,
        frames.hp_equal_overflow.height,
        frames.hp_mismatch_overflow.width,
        frames.hp_mismatch_overflow.height
    );
    blit_native_map_hud_panel(&frames, &mut vec![0u8; STRIDE * 200], true, true, 1).unwrap();
}
