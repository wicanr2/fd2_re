//! `player_toolbar.go` 的觸控座標換算（R5 階段 4 r5w）。Go 測試裡與工具列、海報本體有關的斷言在 R6。

use super::*;

#[test]
fn chapter_guide_hides_event_details_until_requested() {
    let graph = campaign::Campaign {
        start: "battle".into(),
        nodes: [("battle".into(), campaign::Node { r#type: "battle".into(), ..Default::default() })].into(),
        ..Default::default()
    };
    let mut scenario = battle::Scenario::default();
    scenario.events.push(
        serde_json::from_value(serde_json::json!({
            "id": "reinforcement", "trigger": "on_turn_end", "once": true, "when": {"turn": 4},
            "do": [{"type": "spawn_group", "groups": [2]}]
        }))
        .unwrap(),
    );
    let mut g = Game {
        camp: Some(campaign::new_runner(&graph)),
        sc: Some(scenario),
        st: Some(battle::State::default()),
        ..Default::default()
    };
    g.activate_toolbar(ToolbarAction::Chapter);
    assert!(g.chapter_guide_open);
    assert_eq!(g.chapter_guide_lines(), ["待觸發　增援登場"]);
    g.update_chapter_guide(&TickInput { cursor: (470, 55), mouse_left_just_pressed: true, ..Default::default() }, true);
    assert_eq!(g.chapter_guide_lines(), ["待觸發　增援登場", "　第 4 回合"]);
    g.update_chapter_guide(&TickInput { just_pressed_touches: vec![(500, 330)], ..Default::default() }, false);
    assert!(!g.chapter_guide_open);
}

/// Go `player_toolbar_floating_test.go:23` `TestFloatingToolbarGeometryAndOwnership` 末段的觸控表（`:54`..`:68`）。
#[test]
fn test_floating_toolbar_touch_rows() {
    for (top, block, y, want, ok) in [
        (0, false, 12, 12, true),
        (0, true, 12, 0, false),
        (0, true, 24, 24, true),
        (24, false, 12, 0, false),
        (24, false, 24, 0, true),
    ] {
        let g = Game { player_viewport_top: top, player_toolbar_touch_blocked: block, ..Default::default() };
        let (x, gy, got) = g.player_touch_coordinates(300, y);
        assert!(got == ok && (!ok || (x == 300 && gy == want)), "touch {top},{block},{y} => {x},{gy},{got}");
    }
}

/// Go `player_poster_test.go:164` `TestPlayerPosterGeometryInputAndBlocking` 的 `playerTouchCoordinates` 斷言（`:176`..`:191`）：
/// 海報讓畫面左移 132、固定工具列讓畫面下移 24 時，觸控仍落在同一個遊戲點；海報區與選單區不進遊戲。
#[test]
fn test_player_poster_touch_geometry_and_blocking() {
    // Go `playerPosterMenu = image.Rect(362, 26, 634, 200)`（`player_poster_controls.go:14`）。
    let poster_menu = Rectangle { min: Point { x: 362, y: 26 }, max: Point { x: 634, y: 200 } };
    for toolbar_offset in [0, PLAYER_TOOLBAR_HEIGHT] {
        let mut g = Game { player_viewport_left: 132, player_viewport_top: toolbar_offset, ..Default::default() };
        for (px, py) in [(0, 0), (17, 72), (639, 399)] {
            let (x, y, ok) = g.player_touch_coordinates(px + 132, py + toolbar_offset);
            assert!(ok && x == px && y == py, "poster changed the selected game point {px},{py} => {x},{y},{ok}");
        }
        for x in [0, 131, 772, 903] {
            assert!(!g.player_touch_coordinates(x, 200).2, "poster touch entered the game");
        }
        g.player_overlay_block = poster_menu;
        assert!(!g.player_touch_coordinates(132 + 400, 120).2, "menu touch entered the game");
    }
}
