//! 移植自 `remake/internal/campaign/native_class_list_test.go`。

use super::*;
use crate::campaign::native_church_menu::native_church_menu_test::native_church_original_test_entries;
use crate::test_support::input;

/// Go：`nativeClassListStrings`。659 條合成字串；B 組其他測試也用它。
pub(crate) fn native_class_list_strings() -> fdtxt::Strings {
    const COUNT: usize = 659;
    let offsets_bytes = COUNT * 2;
    let mut raw = vec![0u8; offsets_bytes];
    let mut payload: Vec<u8> = Vec::with_capacity(COUNT * 4);
    for i in 0..COUNT {
        raw[i * 2..i * 2 + 2].copy_from_slice(&((offsets_bytes + payload.len()) as u16).to_le_bytes());
        let words: &[u16] = match i {
            1 => &[1],
            150 => &[2],
            151 => &[3],
            0x19f => &[1, 2],
            0x19a => &[1, 2, 3],
            0x19b => &[1, 2],
            0x1a0 => &[1, 2, 3],
            0x1a1 => &[1, 2, 3, 4],
            0x1a2 => &[1, 2],
            0x1a6 | 0x1ad => &[3, 0xfffc, 4],
            0x175 => &[1, 2],
            0x19c | 0x1a3 | 0x1a4 | 513 => &[4],
            593 => &[4],
            594 => &[0xfffc, 4],
            658 => &[4],
            _ => &[0],
        };
        for &word in words {
            payload.extend_from_slice(&word.to_le_bytes());
        }
        payload.extend_from_slice(&fdtxt::STRING_END.to_le_bytes());
    }
    raw.extend_from_slice(&payload);
    fdtxt::parse(&raw).unwrap()
}

/// Go：`nativeClassListFont`。字形 1..4 每列只有最左一點。
pub(crate) fn native_class_list_font() -> fdtxt::Font {
    let mut raw = vec![0u8; 5 * fdtxt::GLYPH_BYTES];
    for glyph in 1..=4 {
        for y in 0..fdtxt::GLYPH_HEIGHT {
            let at = glyph * fdtxt::GLYPH_BYTES + y * 2;
            raw[at..at + 2].copy_from_slice(&0x8000u16.to_be_bytes());
        }
    }
    fdtxt::parse_font(&raw).unwrap()
}

/// 24×24 的空白 FDICON 圖格；`first` 不為零時左上角放這個色號。
pub(crate) fn native_test_sprite(first: u8) -> fdicon::Sprite {
    let n = (fdicon::NATIVE_SIZE * fdicon::NATIVE_SIZE) as usize;
    let mut sprite = fdicon::Sprite { pixels: vec![0; n], mask: vec![0; n], ..Default::default() };
    if first != 0 {
        (sprite.pixels[0], sprite.mask[0]) = (first, 1);
    }
    sprite
}

/// Go：`TestRenderNativeClassListRowsMatches31019CoordinatesAndColours`
#[test]
fn render_native_class_list_rows_matches_31019_coordinates_and_colours() {
    let sprite = native_test_sprite(99);
    let row = NativeClassListRow { sprite, name_text_index: 1, current_class_text_id: 0, target_class_text_id: 1 };
    let rows = vec![row.clone(), row];
    let mut dst = vec![0u8; (NATIVE_CLASS_LIST_STRIDE * NATIVE_CLASS_LIST_HEIGHT) as usize];
    let (strings, font) = (native_class_list_strings(), native_class_list_font());
    render_native_class_list_rows(&mut dst, &rows, 1, Some(&strings), Some(&font)).unwrap();
    assert_eq!(dst[117 * 320 + 14], 99, "row0 sprite");
    assert_eq!(dst[(117 + 26) * 320 + 14], 99, "row1 sprite");
    assert_eq!(dst[121 * 320 + 40], 205, "unselected foreground");
    assert_eq!(dst[(121 + 26) * 320 + 40], 201, "selected foreground");
    assert_eq!(dst[122 * 320 + 39], 76, "native shadow");
    assert_eq!(dst[121 * 320 + 39], 0, "same-row shadow");
    assert_eq!(dst[121 * 320 + 175], 205, "FDTXT593 foreground");
    assert_eq!(dst[121 * 320 + 239], 205, "target class foreground");
}

/// Go：`TestRenderNativeClassListRowsFailsAtomically`
#[test]
fn render_native_class_list_rows_fails_atomically() {
    let mut dst = vec![7u8; (NATIVE_CLASS_LIST_STRIDE * NATIVE_CLASS_LIST_HEIGHT) as usize];
    let before = dst.clone();
    let row = NativeClassListRow {
        sprite: fdicon::Sprite { pixels: vec![0; 1], mask: vec![0; 1], ..Default::default() },
        name_text_index: 1,
        current_class_text_id: 0,
        target_class_text_id: 1,
    };
    let (strings, font) = (native_class_list_strings(), native_class_list_font());
    assert!(
        render_native_class_list_rows(&mut dst, &[row], 0, Some(&strings), Some(&font)).is_err(),
        "malformed sprite unexpectedly rendered"
    );
    assert!(dst == before, "destination mutated");
}

/// Go：`TestComposeNativeClassListFrameUsesFDOTHER14Entry16Geometry`
#[test]
fn compose_native_class_list_frame_uses_fdother14_entry16_geometry() {
    let panel = fdother::LMI1Entry {
        width: NATIVE_CLASS_PANEL_W,
        height: NATIVE_CLASS_PANEL_H,
        pixels: vec![33; (NATIVE_CLASS_PANEL_W * NATIVE_CLASS_PANEL_H) as usize],
    };
    let rows = [NativeClassListRow {
        sprite: native_test_sprite(0),
        name_text_index: 1,
        current_class_text_id: 0,
        target_class_text_id: 1,
    }];
    let (strings, font) = (native_class_list_strings(), native_class_list_font());
    let frame =
        compose_native_class_list_frame(&vec![0u8; 320 * 200], &panel, &rows, 0, Some(&strings), Some(&font)).unwrap();
    assert_eq!(frame[(NATIVE_CLASS_PANEL_Y * 320 + NATIVE_CLASS_PANEL_X) as usize], 33, "panel origin");
    let wrong = fdother::LMI1Entry { width: 309, height: 86, pixels: vec![0; 309 * 86] };
    assert!(
        compose_native_class_list_frame(&vec![0u8; 320 * 200], &wrong, &rows, 0, Some(&strings), Some(&font)).is_err(),
        "wrong panel geometry unexpectedly accepted"
    );
}

/// 兩個裁切排程測試共用：合成畫面 y=112..197、x=5..314 填 `y-111`。
fn crop_schedule_composed() -> Vec<u8> {
    let mut composed = vec![0u8; 320 * 200];
    for y in 112..198 {
        for x in 5..315 {
            composed[y * 320 + x] = (y - 111) as u8;
        }
    }
    composed
}

/// Go：`TestNativeClassListOpeningFramesMatch1974CCropSchedule`
#[test]
fn native_class_list_opening_frames_match_1974c_crop_schedule() {
    let frames = native_class_list_opening_frames(&vec![0u8; 320 * 200], &crop_schedule_composed()).unwrap();
    assert_eq!(frames.len(), 6, "frames");
    for (i, y) in [177usize, 164, 151, 138, 125, 112].into_iter().enumerate() {
        assert_eq!(frames[i][y * 320 + 5], 1, "frame{i} origin want source row1");
        assert!(!(y > 0 && frames[i][(y - 1) * 320 + 5] != 0), "frame{i} wrote before destination y={y}");
    }
    assert_eq!(frames[0][199 * 320 + 5], 23, "first clipped tail");
    assert_eq!(frames[5][197 * 320 + 5], 86, "final full tail");
}

/// Go：`TestNativeClassListClosingFramesMatch2D31BCropSchedule`
#[test]
fn native_class_list_closing_frames_match_2d31b_crop_schedule() {
    let frames = native_class_list_closing_frames(&vec![0u8; 320 * 200], &crop_schedule_composed()).unwrap();
    assert_eq!(frames.len(), 5, "frames");
    for (i, y) in [125usize, 138, 151, 164, 177].into_iter().enumerate() {
        assert_eq!(frames[i][y * 320 + 5], 1, "frame{i} origin want source row1");
        assert!(!(y > 0 && frames[i][(y - 1) * 320 + 5] != 0), "frame{i} wrote before destination y={y}");
    }
    assert_eq!(frames[0][197 * 320 + 5], 73, "first clipped tail");
    assert_eq!(frames[4][199 * 320 + 5], 23, "last clipped tail");
}

/// Go：`TestNativeClassListComposesPlayerOriginalResources`
#[test]
fn native_class_list_composes_player_original_resources() {
    let Some(fdother_path) = input("FD2_FLAME2_DIR", "FDOTHER.DAT") else { return };
    let base = fdother_path.parent().unwrap().to_path_buf();
    let resource14 = fdother::read_resource(&fdother_path, 14).unwrap();
    let entries = native_church_original_test_entries(&resource14);
    let background_frame = fdother::parse_lmi1_frame_entry(&resource14, 0).unwrap();
    assert!(
        entries.len() > 16
            && background_frame.width == 320
            && background_frame.height == 200
            && entries[16].width == 310
            && entries[16].height == 86,
        "FDOTHER#14 shape count={} entry0={}x{} entry16={}x{}",
        entries.len(),
        background_frame.width,
        background_frame.height,
        entries[16].width,
        entries[16].height
    );
    let mut background = vec![0u8; 320 * 200];
    background_frame.blit_at(&mut background, 320, 0, -1).unwrap();
    let strings = fdtxt::parse(&fdother::read_resource(base.join("FDTXT.DAT"), 0).unwrap()).unwrap();
    let font = fdtxt::parse_font(&fdother::read_resource(&fdother_path, 4).unwrap()).unwrap();
    let units = fdicon::decode_file(base.join("FDICON.B24")).unwrap();
    let sprite = units.sprite_for(9, 0, 0).unwrap().clone();
    let rows = [NativeClassListRow { sprite, name_text_index: 10, current_class_text_id: 5, target_class_text_id: 21 }];
    let frame =
        compose_native_class_list_frame(&background, &entries[16], &rows, 0, Some(&strings), Some(&font)).unwrap();
    let frames = native_class_list_opening_frames(&background, &frame).unwrap();
    assert!(
        frames.len() == 6 && frame[117 * 320 + 14] != background[117 * 320 + 14],
        "player original class-list composition produced no visible native row"
    );
    let cells = fdother::decode_raw_cell_resource(&fdother_path, 2).unwrap();
    let question =
        compose_native_class_confirmation_frame(&background, &cells, Some(&strings), Some(&font), 10, 0, 0).unwrap();
    let open_confirm = native_class_confirmation_opening_frames(&question, &cells).unwrap();
    let close_confirm = native_class_confirmation_closing_frames(&question, &cells).unwrap();
    let question_changed = (119..136).any(|y| (12..220).any(|x| question[y * 320 + x] != background[y * 320 + x]));
    assert!(
        open_confirm.len() == 4 && close_confirm.len() == 4 && question_changed,
        "player original confirmation composition produced no visible native question"
    );
}
