//! 移植自 `remake/internal/battle/native_map_walk_view_test.go`。

use super::*;

fn view(camera_x: i64, camera_y: i64, cursor_x: i64, cursor_y: i64, vis_x: i64, vis_y: i64) -> NativeMapViewState {
    NativeMapViewState { camera_x, camera_y, cursor_x, cursor_y, visible_cursor_x: vis_x, visible_cursor_y: vis_y }
}

/// Go：`TestFocusNativeMapCursorStepsThroughKeyboardHandlers`
///
/// 釘住 0x12CEA 的契約：游標先 X 後 Y 逐格移到目標，每一格都走鍵盤處理器，
/// 所以可見游標與鏡頭跟著動。
///
/// 玩家確認移動時走的就是這條路（0x18960 存游標 → 0x18A26 呼叫 0x12CEA），
/// 收據 fd2-move-confirm-cursor-20260909 的 cp0071 量到游標 (8,14)→(8,16) 的
/// 同時可見游標 1→2——第三格是接著開始的走行步進先扣掉的。這一條同時是反例：
/// 若確認是「只寫絕對游標的瞬跳」，可見游標會每動一次就累積一格偏差。
#[test]
fn focus_native_map_cursor_steps_through_keyboard_handlers() {
    let mut st = State { w: 24, h: 24, ..Default::default() };
    st.materialize_native_map_view_state(view(1, 13, 8, 14, 7, 1)).unwrap();
    assert!(st.focus_native_map_cursor(8, 16), "focus 被拒絕");
    let got = st.native_map_view_state;
    let want = view(1, 13, 8, 16, 7, 3);
    assert!(got == want, "focus 後視圖={got:?}，預期 {want:?}");
    // 三組全域成對搬動：visible 與 cursor-camera 的差在整條路上不變。
    assert!(got.visible_cursor_y == got.cursor_y - got.camera_y, "focus 之後可見游標與 cursor-camera 脫節：{got:?}");
    assert!(!(st.focus_native_map_cursor(-1, 16) || st.focus_native_map_cursor(8, 24)), "接受了場外的目標");
}

/// Go：`TestFocusNativeMapCursorScrollsCameraAtSafeBand`
///
/// 同一條路的捲動分支：目標在視窗外時，鏡頭跟著捲，可見游標停在安全帶邊界。
#[test]
fn focus_native_map_cursor_scrolls_camera_at_safe_band() {
    let mut st = State { w: 24, h: 24, ..Default::default() };
    st.materialize_native_map_view_state(view(0, 0, 2, 2, 2, 2)).unwrap();
    assert!(st.focus_native_map_cursor(20, 20), "focus 被拒絕");
    let got = st.native_map_view_state;
    assert!(got.cursor_x == 20 && got.cursor_y == 20, "游標={},{}，預期 20,20", got.cursor_x, got.cursor_y);
    assert!(
        got.cursor_in_field(st.w, st.h) && got.visible_cursor_in_viewport(),
        "focus 之後游標或可見游標出界：{got:?}"
    );
    assert!(
        got.visible_cursor_x == got.cursor_x - got.camera_x && got.visible_cursor_y == got.cursor_y - got.camera_y,
        "可見游標與 cursor-camera 脫節：{got:?}"
    );
}

/// Go：`TestAdvanceNativeMapWalkStepViewUsesUnitRelativeBand`
///
/// 釘住走行步進的判準是單位自己的相對列／行（0x131DE 的 unitY-[0x53AAD]），
/// 不是已存的可見游標。
#[test]
fn advance_native_map_walk_step_view_uses_unit_relative_band() {
    struct Case {
        name: &'static str,
        start: NativeMapViewState,
        unit_x: i64,
        unit_y: i64,
        dx: i64,
        dy: i64,
        want: NativeMapViewState,
    }
    let cases = [
        Case {
            name: "上：單位在安全帶內只動可見游標",
            start: view(1, 13, 8, 16, 7, 3),
            unit_x: 8,
            unit_y: 16,
            dx: 0,
            dy: -1,
            want: view(1, 13, 8, 15, 7, 2),
        },
        Case {
            name: "上：單位貼近上緣且鏡頭還能捲時改捲鏡頭",
            start: view(1, 13, 8, 14, 7, 1),
            unit_x: 8,
            unit_y: 14,
            dx: 0,
            dy: -1,
            want: view(1, 12, 8, 13, 7, 1),
        },
        Case {
            name: "上：鏡頭已到頂就只能動可見游標",
            start: view(1, 0, 8, 1, 7, 1),
            unit_x: 8,
            unit_y: 1,
            dx: 0,
            dy: -1,
            want: view(1, 0, 8, 0, 7, 0),
        },
        Case {
            name: "下：單位越過第 5 列且鏡頭還能捲時改捲鏡頭",
            start: view(1, 5, 8, 11, 7, 6),
            unit_x: 8,
            unit_y: 11,
            dx: 0,
            dy: 1,
            want: view(1, 6, 8, 12, 7, 6),
        },
        Case {
            name: "右：單位越過第 10 行且鏡頭還能捲時改捲鏡頭",
            start: view(1, 5, 13, 9, 12, 4),
            unit_x: 13,
            unit_y: 9,
            dx: 1,
            dy: 0,
            want: view(2, 5, 14, 9, 12, 4),
        },
        Case {
            name: "左：單位在安全帶內只動可見游標",
            start: view(1, 5, 8, 9, 7, 4),
            unit_x: 8,
            unit_y: 9,
            dx: -1,
            dy: 0,
            want: view(1, 5, 7, 9, 6, 4),
        },
        // 收據 cp0071 的那一幀：走行步進先扣可見游標（0x13205），游標要到
        // 函式結尾才扣（0x1330A），所以中途取樣會看到 visible 比
        // cursor-camera 少一格。
        Case {
            name: "步進中途取樣時仍以單位相對列判斷",
            start: view(1, 13, 8, 16, 7, 2),
            unit_x: 8,
            unit_y: 16,
            dx: 0,
            dy: -1,
            want: view(1, 13, 8, 15, 7, 1),
        },
    ];
    for c in &cases {
        let mut st = State { w: 24, h: 24, ..Default::default() };
        st.materialize_native_map_view_state(c.start).unwrap_or_else(|err| panic!("{}: {err}", c.name));
        assert!(
            st.advance_native_map_walk_step_view(c.unit_x, c.unit_y, c.dx, c.dy),
            "{}: 走行步進視圖更新被拒絕",
            c.name
        );
        let got = st.native_map_view_state;
        assert!(got == c.want, "{}: 視圖={got:?}，預期 {:?}", c.name, c.want);
    }
}

/// Go：`TestAdvanceNativeMapWalkStepViewLeavesViewport`
///
/// 釘住原版真的走得到的出界狀態。dosgolem 收據
/// docs/data/ui-traces/fd2-story-pan-cursor-20260909.json 的 frames idx=75：
/// 序章走行捲動 15 格之後 camera_y 34→20、cursor_y 34→19、visible_y 0→-1。
/// 單位還在安全帶內（unitY - camY >= 2）的那一步走的是 `0x13205 dec [0x53ABD]`，
/// 可見游標因此掉到視窗外；界線是消費端的事，這一步本身合法。
#[test]
fn advance_native_map_walk_step_view_leaves_viewport() {
    let mut st = State { w: 24, h: 24, ..Default::default() };
    st.materialize_native_map_view_state(view(1, 13, 8, 15, 7, 0)).unwrap();
    assert!(st.advance_native_map_walk_step_view(8, 15, 0, -1), "拒絕了原版走得到的出界可見游標");
    let want = view(1, 13, 8, 14, 7, -1);
    let got = st.native_map_view_state;
    assert!(got == want, "視圖={got:?}，預期 {want:?}");
    assert!(!st.native_map_view_state.visible_cursor_in_viewport(), "出界的可見游標被判成視窗內");
}

/// Go：`TestAdvanceNativeMapWalkStepViewMatchesCh00PreScroll`
///
/// 用原版收據的端點釘住整段走行捲動。dosgolem 收據
/// docs/data/ui-traces/fd2-story-pan-cursor-20260909.json 的抓幀邊界在 0x11CAC，
/// 而走行重繪不走那條路徑，所以整段捲動一次都沒被抓到；取得的是端點：序章 pan 到
/// (3,34) 之後 scroll_step 15 格，走完是 camera_y 34→20、cursor_y 34→19、visible_y 0→-1。
///
/// 逐格套用 0x13185 的規則會走出同一組端點：主角在 (8,36)，第一格
/// `unitY - camY = 2` 走可見游標，之後 `unitY - camY = 1` 每一格都捲鏡頭，
/// 合計 1 次可見 + 14 次鏡頭，而絕對游標 15 格都跟著單位。
#[test]
fn advance_native_map_walk_step_view_matches_ch00_pre_scroll() {
    const MAP_W: i64 = 18; // map32
    const MAP_H: i64 = 51;
    let mut v = view(3, 34, 3, 34, 0, 0);
    let (unit_x, mut unit_y) = (8, 36);
    for step in 0..15 {
        let (next, ok) = advance_native_map_walk_step_view_state(v, MAP_W, MAP_H, unit_x, unit_y, 0, -1);
        assert!(ok, "第 {step} 格被拒絕：view={v:?} unit=({unit_x},{unit_y})");
        (v, unit_y) = (next, unit_y - 1);
    }
    let want = view(3, 20, 3, 19, 0, -1);
    assert!(v == want, "走完 15 格 view={v:?}，原版收據 {want:?}");
}
