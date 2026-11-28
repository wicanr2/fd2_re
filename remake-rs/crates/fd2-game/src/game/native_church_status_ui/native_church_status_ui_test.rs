//! Go 來源：`remake/cmd/fd2/native_church_status_ui_test.go` 的 `TestNativeChurchPanelLifecycleMatches17AEDSchedules`。
//! 其餘三項經 `beginNativeChurchStatus`（`r8`）；Rust 教會狀態檢視走 `native_church_panels.rs` 的既有實作。

use super::*;

/// Go：`native_church_status_ui_test.go:11` `TestNativeChurchPanelLifecycleMatches17AEDSchedules`
#[test]
fn test_native_church_panel_lifecycle_matches_17aed_schedules() {
    let source = vec![0u8; 320 * 200];
    let mut panel = vec![0u8; 320 * 200];
    for y in 7..196 {
        for x in 5..315 {
            panel[y * 320 + x] = 9;
        }
    }
    let opening = native_church_panel_frames(&source, &panel, true);
    assert!(opening.as_ref().is_ok_and(|f| f.len() == 12), "opening frames err={:?}", opening.as_ref().err());
    let closing = native_church_panel_frames(&source, &panel, false);
    assert!(closing.as_ref().is_ok_and(|f| f.len() == 12), "closing frames err={:?}", closing.as_ref().err());
    let opening = opening.unwrap();
    assert!(
        opening[0][7 * 320 + 5] == 9 && opening[11][7 * 320 + 5] == 9,
        "left panel was not composed in opening schedule"
    );
    let transition = native_church_bottom_panel_frame(&source, &panel, 6).unwrap();
    assert!(transition[190 * 320 + 5] == 9 && transition[189 * 320 + 5] == 0, "bottom step6 did not land at y190");
}
