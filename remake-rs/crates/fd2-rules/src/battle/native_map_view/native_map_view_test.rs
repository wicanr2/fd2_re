//! 移植自 `remake/internal/battle/native_map_view_test.go`。

use super::*;

fn view(camera_x: i64, camera_y: i64, cursor_x: i64, cursor_y: i64, vis_x: i64, vis_y: i64) -> NativeMapViewState {
    NativeMapViewState { camera_x, camera_y, cursor_x, cursor_y, visible_cursor_x: vis_x, visible_cursor_y: vis_y }
}

/// Go：`TestNativeMapViewMaterializesViewportBounds`
///
/// 釘住可見游標的界線契約。
///
/// `[0x53AB9]`／`[0x53ABD]` 的寫入端只有四個鍵盤游標處理器、四個走行步進，
/// 加上初始化與章節歸零（全段掃描，見
/// docs/data/ida/fd2_visible_cursor_writers_ida.txt）。劇情捲動 `0x135DD`
/// 不碰它，但它同時搬游標與鏡頭，所以差值不變。走行步進在函式頭尾分兩處寫
/// （`0x13205` 可見游標、`0x1330A` 游標），中途取樣會看到兩者差一格。
///
/// 原版也沒有把它夾在 13×8 內：走行步進在鏡頭已到邊界時照樣 `dec [0x53ABD]`，
/// dosgolem 收據 docs/data/ui-traces/fd2-story-pan-cursor-20260909.json
/// （frames idx=75）量到 15 格之後 visible_y = -1。13×8 是消費端
/// （0x1741C）的界線，由 VisibleCursorInViewport 與 fdother.ActionOverlayOrigin
/// 負責；物化只擋「偏離超過場地」這種結構性壞值。
#[test]
fn native_map_view_materializes_viewport_bounds() {
    let mut st = State { w: 24, h: 24, ..Default::default() };
    let view = view(1, 13, 8, 17, 7, 4);
    st.materialize_native_map_view_state(view).unwrap();
    assert!(view.visible_cursor_in_viewport(), "視窗內的可見游標被判成出界");
    let mut stale = view;
    stale.visible_cursor_y -= 1; // 確認跳格之後的舊值，原版走得到
    if let Err(err) = st.materialize_native_map_view_state(stale) {
        panic!("拒絕了原版走得到的舊可見游標：{err}");
    }
    let mut outside = view;
    outside.visible_cursor_y = -1; // 走行捲動在鏡頭到頂時走得到
    if let Err(err) = st.materialize_native_map_view_state(outside) {
        panic!("拒絕了原版走得到的出界可見游標：{err}");
    }
    assert!(!outside.visible_cursor_in_viewport(), "出界的可見游標被判成視窗內");
    let mut outside = view;
    outside.visible_cursor_y = NATIVE_MAP_VIEW_HEIGHT;
    if let Err(err) = st.materialize_native_map_view_state(outside) {
        panic!("拒絕了視窗外的可見游標：{err}");
    }
    assert!(!outside.visible_cursor_in_viewport(), "視窗外的可見游標被判成視窗內");
    // 原版不夾這個全域，重製端也不夾：八個寫入端沒有一個檢查範圍。界線在
    // 消費端，不在狀態層。
    let mut drifted = view;
    (drifted.visible_cursor_x, drifted.visible_cursor_y) = (-st.w, st.h * 4);
    if let Err(err) = st.materialize_native_map_view_state(drifted) {
        panic!("狀態層夾了可見游標：{err}");
    }
    assert!(!drifted.visible_cursor_in_viewport(), "偏離很遠的可見游標被判成視窗內");
    // 鏡頭則相反：每個寫入端都在分支條件裡夾它（`0x11B64 cmp [0x53AAD],0`、
    // `0x11BC4` 的 `[0x53AC5]-8`），pan 的終止條件也是鏡頭，所以這裡照樣擋。
    let mut bad_camera = view;
    bad_camera.camera_y = st.h - NATIVE_MAP_VIEW_HEIGHT + 1;
    assert!(st.materialize_native_map_view_state(bad_camera).is_err(), "接受了越界的鏡頭");
    // 絕對游標留著場內檢查，理由是重製端的自我檢查而不是原版會夾它：原版的寫入端
    // 成對搬動三組全域，鏡頭被夾在 0..W-13／0..H-8、可見游標被安全帶留在視窗內，
    // 相加就落在場內。場外代表某條路徑打破了那個配對。
    let mut bad_cursor = view;
    bad_cursor.cursor_x = st.w;
    assert!(st.materialize_native_map_view_state(bad_cursor).is_err(), "接受了場外的絕對游標");
    assert!(!bad_cursor.cursor_in_field(st.w, st.h), "場外的絕對游標被判成場內");
    assert!(view.cursor_in_field(st.w, st.h), "場內的絕對游標被判成場外");
}

/// Go：`TestNativeMapViewKeepsVisibleCursorPairedWithCamera`
///
/// 上一條的正對照：三組全域的寫入端成對搬動，所以整條互動路上
/// `visible == cursor - camera` 都成立。早期把移動確認實作成「只寫絕對游標的瞬跳」時，
/// 這個差每走一步就多累積一格，幾回合後鏡頭再也捲不到游標所在的列。
#[test]
fn native_map_view_keeps_visible_cursor_paired_with_camera() {
    let mut st = State { w: 24, h: 24, ..Default::default() };
    st.materialize_native_map_view_state(view(0, 0, 3, 3, 3, 3)).unwrap();
    let paired = |st: &State, tag: &str| {
        let v = st.native_map_view_state;
        assert!(
            v.visible_cursor_x == v.cursor_x - v.camera_x && v.visible_cursor_y == v.cursor_y - v.camera_y,
            "{tag} 之後脫節：{v:?}"
        );
    };
    for _ in 0..20 {
        // 鍵盤推到右下角，途中會捲鏡頭
        st.move_native_map_cursor(1, 0);
        st.move_native_map_cursor(0, 1);
        paired(&st, "鍵盤");
    }
    let (cx, cy) = (st.native_map_view_state.cursor_x, st.native_map_view_state.cursor_y);
    assert!(st.advance_native_map_walk_step_view(cx, cy, -1, 0), "走行步進被拒絕");
    paired(&st, "走行步進");
    assert!(st.focus_native_map_cursor(4, 5), "focus 被拒絕");
    paired(&st, "focus");
}

/// Go：`TestNativeMapCursorMovesCameraAtRecoveredThresholds`
#[test]
fn native_map_cursor_moves_camera_at_recovered_thresholds() {
    let mut st = State { w: 30, h: 30, ..Default::default() };
    st.materialize_native_map_view_state(view(4, 5, 15, 11, 11, 6)).unwrap();
    assert!(st.move_native_map_cursor(1, 0) == (true, true), "right move rejected");
    let got = st.native_map_view_state;
    assert!(got.camera_x == 5 && got.cursor_x == 16 && got.visible_cursor_x == 11, "right camera-follow={got:?}");
    assert!(st.move_native_map_cursor(0, 1) == (true, true), "down move rejected");
    let got = st.native_map_view_state;
    assert!(got.camera_y == 6 && got.cursor_y == 12 && got.visible_cursor_y == 6, "down camera-follow={got:?}");
    st.native_map_view_state = view(4, 5, 5, 6, 1, 1);
    assert!(st.move_native_map_cursor(-1, 0) == (true, true), "left move rejected");
    let got = st.native_map_view_state;
    assert!(got.camera_x == 3 && got.cursor_x == 4 && got.visible_cursor_x == 1, "left camera-follow={got:?}");
    assert!(st.move_native_map_cursor(0, -1) == (true, true), "up move rejected");
    let got = st.native_map_view_state;
    assert!(got.camera_y == 4 && got.cursor_y == 5 && got.visible_cursor_y == 1, "up camera-follow={got:?}");
}

/// Go：`TestNativeMapCursorFieldEdgeIsValidNoMove`
#[test]
fn native_map_cursor_field_edge_is_valid_no_move() {
    let mut st = State { w: 13, h: 8, ..Default::default() };
    st.materialize_native_map_view_state(NativeMapViewState::default()).unwrap();
    let (moved, ok) = st.move_native_map_cursor(-1, 0);
    assert!(!moved && ok, "edge move={moved} ok={ok}");
    let (_, ok) = st.move_native_map_cursor(1, 1);
    assert!(!ok, "accepted diagonal move");
}

/// Go：`TestNativeMapRangeModePreservesFullRawSelectorBounds`
#[test]
fn native_map_range_mode_preserves_full_raw_selector_bounds() {
    let mut st = State::default();
    assert!(
        st.materialize_native_map_range_mode(0) && st.has_native_map_range_mode_state && st.native_map_range_mode == 0,
        "raw bootstrap range mode rejected"
    );
    for mode in [6, 7, 9, 11, 0x101] {
        assert!(
            st.materialize_native_map_range_mode(mode) && st.native_map_range_mode == mode,
            "verified raw selector {mode} rejected"
        );
    }
    let got = native_map_overlay_selector_from_record_byte(9);
    assert!(got == 11, "record byte 9 selector={got}, want 11");
    assert!(
        !st.materialize_native_map_range_mode(0x102) && st.native_map_range_mode == 0x101,
        "out-of-range mode changed materialized state"
    );
}
