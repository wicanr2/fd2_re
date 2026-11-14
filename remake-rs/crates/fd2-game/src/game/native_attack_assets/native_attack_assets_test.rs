//! Go `native_attack_assets_test.go`。

use super::*;

/// Go：`native_attack_assets_test.go:13`。Go 另設 `FD2_ORIGINAL_FIGANI` 指向不存在的檔，證明不讀 FIGANI.DAT；
/// Rust 的 `ensureNativeAttackPresentation` 只讀分離素材包，沒有那條後備路徑。
#[test]
fn test_ensure_native_attack_presentation_loads_separated_resources_without_archive() {
    for name in ["FIGANI_168", "FIGANI_379"] {
        if crate::test_support::input("FD2_ASSET_PACK", &format!("animations/{name}/animation.json")).is_none() {
            return;
        }
    }
    let mut g = Game { native_ui_palette: load_native_ui_palette(), ..Default::default() };
    g.ensure_native_attack_presentation(126, 56).unwrap();
    assert!(g.native_attack_presentation_available(126, 56), "late FIGANI pair is not available after exact raw load");
    for id in [379, 168] {
        let frames = g.figani.get(&id).map_or(0, Vec::len);
        assert!(
            frames != 0
                && g.figani_delays.get(&id).map_or(0, Vec::len) == frames
                && g.fig_meta.get(&id).map_or(0, Vec::len) == frames,
            "resource {id} was not published as a complete frame/delay/position set"
        );
    }
}

/// Go：`native_attack_assets_test.go:101`。Go 以 `t.Setenv("FD2_ASSET_PACK", t.TempDir())` 換成空素材包；Rust 在
/// 子行程設定（F 組的 `f_child_env`）。
#[test]
fn test_ensure_native_attack_presentation_rejects_without_partial_publish() {
    let name = "test_ensure_native_attack_presentation_rejects_without_partial_publish";
    if f_child_env(module_path!(), name, |dir| vec![("FD2_ASSET_PACK", dir.into())]).is_none() {
        return;
    }
    let palette = (0..256).map(|i| ColorRgba { r: i as u8, g: i as u8, b: i as u8, a: 0xff }).collect();
    let mut g = Game { native_ui_palette: palette, ..Default::default() };
    assert!(g.ensure_native_attack_presentation(126, 56).is_err(), "missing separated FIGANI resources were accepted");
    assert!(
        g.figani.is_empty() && g.figani_delays.is_empty() && g.fig_meta.is_empty(),
        "failed preload partially published presentation maps"
    );
}

/// D 組補充：調色盤不足 256 色時在載入任何資源前拒收（Go `native FIGANI palette unavailable`）。
#[test]
fn ensure_native_attack_presentation_requires_full_palette() {
    let mut g = Game::default();
    assert_eq!(
        g.ensure_native_attack_presentation(126, 56).unwrap_err().message(),
        "native FIGANI palette unavailable"
    );
}
