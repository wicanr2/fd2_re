//! 移植自 `remake/internal/campaign/native_church_scene_test.go`。

use super::*;
use crate::campaign::native_church_menu::native_church_menu_test::native_church_original_test_entries;
use crate::test_support::input;

/// Go `loadNativeChurchOriginalSceneAssets` 的回傳值：背景、FDOTHER#14 項目、FDOTHER#5 對話格、
/// 金額數字、DATO#131 第 0 幀、FDTXT_000 與字型。
pub(crate) struct NativeChurchOriginalSceneAssets {
    pub background: Vec<u8>,
    pub entries14: Vec<fdother::LMI1Entry>,
    pub dialogue_cells: Vec<fdother::RawCell>,
    pub digit_frames: Vec<fdother::Frame>,
    pub portrait: dato::Frame,
    pub strings: fdtxt::Strings,
    pub font: fdtxt::Font,
}

/// Go：`loadNativeChurchOriginalSceneAssets`。缺原版檔時回 `None`（Go 的 `t.Skip`）。
pub(crate) fn load_native_church_original_scene_assets() -> Option<NativeChurchOriginalSceneAssets> {
    let fdother_path = input("FD2_FLAME2_DIR", "FDOTHER.DAT")?;
    let base = fdother_path.parent().unwrap().to_path_buf();
    let resource14 = fdother::read_resource(&fdother_path, 14).unwrap();
    let entries14 = native_church_original_test_entries(&resource14);
    let background_frame = fdother::parse_lmi1_frame_entry(&resource14, 0).unwrap();
    let mut background = vec![0u8; 320 * 200];
    background_frame.blit_at(&mut background, 320, 0, -1).unwrap();
    let resource5 = fdother::read_resource(&fdother_path, 5).unwrap();
    let mut dialogue_cells = vec![fdother::RawCell::default(); 18];
    for (index, cell) in dialogue_cells.iter_mut().enumerate().skip(1) {
        *cell = fdother::parse_lmi1_raw_entry(&resource5, index as i64)
            .unwrap_or_else(|e| panic!("dialogue cell {index}: {e}"));
    }
    let digit_frames = (0..10)
        .map(|digit| {
            fdother::parse_lmi1_frame_entry(&resource5, 31 + digit).unwrap_or_else(|e| panic!("digit {digit}: {e}"))
        })
        .collect();
    let portraits = dato::decode_resource(base.join("DATO.DAT"), 131).unwrap();
    let text_raw = fdother::read_resource(base.join("FDTXT.DAT"), 0).unwrap();
    let strings = fdtxt::parse(&text_raw).unwrap();
    let font_raw = fdother::read_resource(&fdother_path, 4).unwrap();
    let font = fdtxt::parse_font(&font_raw).unwrap();
    Some(NativeChurchOriginalSceneAssets {
        background,
        entries14,
        dialogue_cells,
        digit_frames,
        portrait: portraits[0].clone(),
        strings,
        font,
    })
}

/// Go：`TestComposeNativeChurchSceneUsesOriginalMixedCodecResources`
#[test]
fn compose_native_church_scene_uses_original_mixed_codec_resources() {
    let Some(a) = load_native_church_original_scene_assets() else { return };
    for text_index in [585, 586] {
        let frame = compose_native_church_scene(
            &a.background,
            &a.entries14[1],
            &a.dialogue_cells,
            &a.digit_frames,
            &a.portrait,
            &a.strings,
            &a.font,
            12345678,
            text_index,
        )
        .unwrap_or_else(|e| panic!("FDTXT{text_index}: {e}"));
        assert_eq!(frame.len(), 320 * 200, "FDTXT{text_index} frame bytes");
        native_church_menu_transition_frames(&frame, &a.entries14, true)
            .unwrap_or_else(|e| panic!("FDTXT{text_index} menu transition: {e}"));
    }
}
