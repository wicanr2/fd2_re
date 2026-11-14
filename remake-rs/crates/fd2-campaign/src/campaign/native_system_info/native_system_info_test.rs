//! 移植自 `remake/internal/campaign/native_system_info_test.go`。

use super::*;
use crate::campaign::church::church_test::unit;

/// Go：`nativeSystemInfoFixture`
fn native_system_info_fixture() -> NativeSystemInfoAssets {
    let mut assets = NativeSystemInfoAssets {
        numbers: battle::NativeItemPanelDataAssets { frames: Some(BTreeMap::new()), ..Default::default() },
        ..Default::default()
    };
    let geometry = [[102, 17], [170, 117], [170, 16], [63, 15]];
    for (index, [w, h]) in geometry.into_iter().enumerate() {
        let n = (w * h) as usize;
        assets.panels[index] = fdother::Frame::separated(0, 0, w, h, vec![(0x80 + index) as u8; n], vec![255; n]);
    }
    let frames = assets.numbers.frames.as_mut().unwrap();
    for index in 31..=51 {
        frames.insert(index, fdother::Frame::raw(0, 0, 1, 1, vec![1, 0, 1, 0, 0, index as u8]));
    }
    let mut font_raw = vec![0u8; fdtxt::GLYPH_BYTES];
    font_raw[0] = 0x80;
    assets.font = Some(fdtxt::parse_font(&font_raw).unwrap());
    assets
}

/// Go：`TestComposeNativeSystemInfoSurfaceUses1B41DPanelAndNumberAnchors`
#[test]
fn compose_native_system_info_surface_uses_1b41d_panel_and_number_anchors() {
    let input = NativeSystemInfoInput {
        raw_chapter: 1,
        raw_round: 23,
        currency: 456,
        raw_camp_counts: [7, 8, 9],
        lines: [vec![0], vec![0]],
    };
    let frame = compose_native_system_info_surface(&native_system_info_fixture(), &input).unwrap();
    for (x, y, want) in [
        (109, 19, 0x80u8),
        (75, 37, 0x81),
        (75, 155, 0x82),
        (129, 172, 0x83),
        (143, 24, 42),
        (188, 24, 42),
        (140, 176, 31),
        (120, 159, 42),
        (182, 159, 42),
        (228, 159, 42),
        (80, 61, 0xcd),
        (80, 116, 0xcd),
    ] {
        assert_eq!(frame[y * 320 + x], want, "pixel ({x},{y})");
    }
}

/// Go：`TestNativeSystemInfoCampCountsUsesRaw1B41DPredicate`
#[test]
fn native_system_info_camp_counts_uses_raw_1b41d_predicate() {
    let mk = |camp: u8, byte5: u8, fig: u8, race: u8| {
        unit! {
            native_record_byte5: byte5,
            has_native_record_byte5: true,
            native_record_byte6: camp,
            has_native_record_byte6: true,
            battle_fig: fig as i64,
            has_battle_fig: true,
            native_record_race: race,
            has_native_record_race: true,
        }
    };
    let units = [
        mk(0, 0, 1, 1),
        mk(1, 0, 2, 2),
        mk(2, 0, 3, 3),
        mk(0, 1, 4, 4),
        mk(1, 0, 0x79, 5),
        mk(2, 0, 6, 10),
        mk(3, 0, 7, 7),
    ];
    let counts = native_system_info_camp_counts(&units);
    assert!(counts.as_ref().is_ok_and(|c| *c == [1, 1, 1]), "counts={counts:?}");
    let mut missing = mk(0, 0, 1, 1);
    missing.has_native_record_race = false;
    // Go 失敗時回傳零值 counts；Rust 回 `Err`。
    let got = native_system_info_camp_counts(&[missing]);
    assert!(got.is_err(), "missing provenance counts={got:?}");
}

/// Go：`TestComposeNativeSystemInfoSurfaceRejectsInvalidPanelBeforePublication`
#[test]
fn compose_native_system_info_surface_rejects_invalid_panel_before_publication() {
    let mut assets = native_system_info_fixture();
    assets.panels[2].width -= 1;
    let frame = compose_native_system_info_surface(&assets, &NativeSystemInfoInput::default());
    assert!(frame.is_err(), "invalid panel published frame={}", frame.is_ok());
}
