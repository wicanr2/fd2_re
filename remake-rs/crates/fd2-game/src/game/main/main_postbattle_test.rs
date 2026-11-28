//! `main.go` 名下 R8 小切片的測試：近似戰後前進（Go `postbattle_guard_test.go`）、戰鬥事件錯誤擋住原生章節結果、
//! 第 28 章戰後對話繼承戰鬥視圖（Rust 補充）。

use super::*;

/// Go：`postbattle_guard_test.go:29` `TestApproximatePostbattlePreservesAuthoredIntermissionBoundary`
#[test]
fn test_approximate_postbattle_preserves_authored_intermission_boundary() {
    for (next, next_type) in [("town_after", "town"), ("preparation_after", "preparation")] {
        let c = campaign::Campaign {
            start: String::from("postbattle_missing_handler"),
            nodes: BTreeMap::from([
                (
                    String::from("postbattle_missing_handler"),
                    campaign::Node { r#type: String::from("cutscene"), next: next.to_string(), ..Default::default() },
                ),
                (next.to_string(), campaign::Node { r#type: next_type.to_string(), ..Default::default() }),
            ]),
            ..Default::default()
        };
        let mut unit = battle::Unit::default();
        (unit.fig, unit.camp, unit.on_field) = (0, battle::OWN, true);
        (unit.hp, unit.max_hp, unit.mp, unit.max_mp) = (4, 10, 1, 3);
        let mut st = battle::State::default();
        st.units.push(unit.clone());
        let mut g = Game {
            camp: Some(campaign::new_runner(&c)),
            approximate_mode: true,
            st: Some(st),
            party_members: BTreeMap::from([(0, true)]),
            party_roster: BTreeMap::from([(0, unit)]),
            ..Default::default()
        };
        g.enter_node();
        assert!(
            g.load_err.is_empty() && g.approximate_postbattle,
            "{next_type}: approximate postbattle did not stop at prompt: err={:?}",
            g.load_err
        );
        assert_eq!(g.camp.as_ref().unwrap().node_id(), "postbattle_missing_handler", "advanced before confirmation");
        assert_eq!(g.party_roster[&0].hp, 10, "battle snapshot was not synchronized before intermission");
        assert!(g.continue_approximate_postbattle(), "approximate postbattle confirmation was rejected");
        assert_eq!(g.camp.as_ref().unwrap().node_id(), next, "confirmed approximate postbattle landing");
        assert!(g.st.is_none(), "intermission boundary retained battle state for {next_type}");
        assert!(!g.continue_approximate_postbattle(), "second confirmation must be rejected");
    }
}

/// Rust 補充：Go `finishBattleEventWithError` 在原生章節結果進行中時，另以 `failNativeChapterResult` 擋住啟動。
#[test]
fn test_battle_event_error_blocks_native_chapter_result() {
    let mut g = Game { native_chapter_result: Some(Default::default()), ..Default::default() };
    g.finish_battle_event_with_error("boom");
    assert!(g.startup_blocked && g.load_err == "原生章節結果：boom", "err={:?}", g.load_err);
    let mut g = Game::default();
    g.finish_battle_event_with_error("boom");
    assert!(!g.startup_blocked && g.load_err == "battle event: boom", "err={:?}", g.load_err);
}

/// Rust 補充：Go `materializeInheritedNativeDialogueView` 只接受圖塊對齊、非負的戰鬥鏡頭。
#[test]
fn test_materialize_inherited_native_dialogue_view_requires_tile_aligned_camera() {
    let mut st = battle::State::default();
    (st.w, st.h) = (40, 40);
    let mut g = Game {
        st: Some(st),
        m: Some(MapData { w: 40, h: 40, tile_w: 24, tile_h: 24, ..Default::default() }),
        cam_x: 48.0,
        cam_y: 72.0,
        cur_x: 7,
        cur_y: 9,
        ..Default::default()
    };
    g.materialize_inherited_native_dialogue_view().expect("aligned camera rejected");
    let st = g.st.as_ref().unwrap();
    let view = st.native_map_view_state;
    assert!(st.has_native_map_view_state);
    assert_eq!(
        (view.camera_x, view.camera_y, view.cursor_x, view.cursor_y, view.visible_cursor_x, view.visible_cursor_y),
        (2, 3, 7, 9, 5, 6)
    );
    g.cam_x = 50.0;
    let err = g.materialize_inherited_native_dialogue_view().unwrap_err();
    assert!(err.message().contains("not tile-aligned"), "{}", err.message());
    g.cam_x = -24.0;
    let err = g.materialize_inherited_native_dialogue_view().unwrap_err();
    assert!(err.message().contains("negative"), "{}", err.message());
    g.m = None;
    assert!(g.materialize_inherited_native_dialogue_view().is_err(), "missing map accepted");
}

/// Rust 補充：`drawCampaignUI` 的 TTF 後備版面（流程結束的 GAME OVER、choice 選單、ending 結語）以真字型畫出，
/// 不經原生呈現。
#[test]
fn test_campaign_ui_ttf_fallbacks_draw_game_over_choice_and_ending() {
    if !crate::test_support::cjk_font_available()
        || crate::test_support::remake_asset("locales/zh-Hant/pack.json").is_none()
    {
        return;
    }
    let option = |label: &str| campaign::ChoiceOption {
        label: label.to_string(),
        to: String::from("end"),
        ..Default::default()
    };
    let choice = campaign::Node {
        r#type: String::from("choice"),
        prompt: String::from("要往哪裡？"),
        options: vec![option("北"), option("南")],
        ..Default::default()
    };
    let ending = campaign::Node { r#type: String::from("ending"), text: String::from("完"), ..Default::default() };
    let nodes = BTreeMap::from([(String::from("pick"), choice), (String::from("end"), ending)]);
    let runner = |start: &str| {
        let c = campaign::Campaign { start: start.to_string(), nodes: nodes.clone(), ..Default::default() };
        campaign::new_runner(&c)
    };
    let mut g = Game { font: Some(load_font().unwrap()), ..Default::default() };
    crate::game::save::save_test::attach_official_locale(&mut g);
    let mut painted = |start: &str| {
        g.camp = Some(runner(start));
        let mut screen = Picture::new_image(LOGICAL_W, LOGICAL_H);
        g.draw_campaign_ui(&mut screen);
        assert!(g.load_err.is_empty(), "{start}: {}", g.load_err);
        screen.read_pixels().chunks(4).filter(|p| p[3] != 0 && p[..3] != [0, 0, 0]).count()
    };
    assert!(painted("pick") > 0, "choice fallback drew nothing");
    assert!(painted("end") > 0, "ending fallback drew nothing");
    // 起點不在節點表裡：`node()` 是 `None`，即流程結束。
    assert!(painted("gone") > 0, "GAME OVER fallback drew nothing");
}
