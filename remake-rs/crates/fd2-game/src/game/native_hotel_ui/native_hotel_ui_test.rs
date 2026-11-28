//! Go 來源：`remake/cmd/fd2/hotel_trace_test.go`（R4 E 組，期望值不改）。
//!
//! 未移植：`:172`（傳聞 story 節點，主題屬於 A／B 的劇情流程）、`:209`（以環境變數啟用、輸出畫面的同源探針）。
//! Go 以 `t.Setenv` 設 `FD2_NATIVE_SAVE`／`XDG_DATA_HOME` 的酒店讀檔測試在子行程執行（`f_child_env`）。

use super::*;

/// Go：`hotel_trace_test.go:18` `TestCampaignTownHotelRawRouteReturnTrace`
#[test]
fn test_campaign_town_hotel_raw_route_return_trace() {
    let c = campaign::Campaign {
        start: String::from("town_ch02"),
        nodes: BTreeMap::from([
            (
                String::from("town_ch02"),
                campaign::Node {
                    r#type: String::from("town"),
                    options: vec![campaign::ChoiceOption {
                        label: String::from("旅館"),
                        to: String::from("hotel_ch02"),
                        ..Default::default()
                    }],
                    ..Default::default()
                },
            ),
            (
                String::from("hotel_ch02"),
                campaign::Node {
                    r#type: String::from("hotel"),
                    text: String::from("旅館／整備"),
                    next: String::from("town_ch02"),
                    ..Default::default()
                },
            ),
        ]),
        ..Default::default()
    };
    let mut g = Game { camp: Some(campaign::new_runner(&c)), st: Some(battle::State::default()), ..Default::default() };
    g.step_campaign_menu(campaign::MENU_CONFIRM);
    g.camp.as_mut().unwrap().advance("opt0");
    g.enter_node();
    assert!(
        g.camp.as_ref().unwrap().node_id() == "hotel_ch02" && g.st.is_none() && g.hotel_sel == 0 && !g.hotel_has_route,
        "town→hotel boundary={:?} st={} sel={} route={}",
        g.camp.as_ref().unwrap().node_id(),
        g.st.is_some(),
        g.hotel_sel,
        g.hotel_has_route
    );
    for (selector, want) in [(0u8, 0x2ffa5u32), (1, 0x30012), (2, 0x301f4), (3, 0x19953)] {
        let secondary = if selector == 3 { 0x197e5 } else { 0 };
        let expected = fdother::NativeHotelServiceRoute { selector, resource_id: 13, primary: want, secondary };
        assert!(
            g.apply_hotel_service_selection(selector) && g.hotel_has_route && g.hotel_route == expected,
            "selector {selector} route={:?} msg={:?}",
            g.hotel_route,
            g.msg
        );
    }
    let prior = g.hotel_route;
    assert!(
        !g.apply_hotel_service_selection(4) && g.hotel_route == prior,
        "invalid selector unexpectedly changed route: {:?} msg={:?}",
        g.hotel_route,
        g.msg
    );
    g.leave_hotel();
    assert_eq!(g.camp.as_ref().unwrap().node_id(), "town_ch02", "hotel return node");
}

/// Go：`native_chapter_slot_restore_test.go:48` `writeNativeRestoreFixture`，寫到 `path`。
fn write_native_restore_fixture(path: &std::path::Path, raw_chapter: u8) {
    let mut plain = vec![0u8; fdsave::FILE_SIZE];
    for slot in 0..fdsave::SLOT_COUNT {
        let (start, _) = fdsave::slot_bounds(slot).unwrap();
        plain[start + fdsave::ROSTER_SIZE] = 0xff;
    }
    let (start, _) = fdsave::slot_bounds(1).unwrap();
    let record = &mut plain[start..start + fdsave::UNIT_SIZE];
    (record[7], record[8], record[0x20], record[0x21], record[0x3b]) = (0x44, 9, 5, 7, 6);
    record[0x40..0x42].copy_from_slice(&31u16.to_le_bytes());
    record[0x42..0x44].copy_from_slice(&40u16.to_le_bytes());
    let metadata = start + fdsave::ROSTER_SIZE;
    (plain[metadata], plain[metadata + 1]) = (raw_chapter, 1);
    plain[metadata + 2..metadata + 6].copy_from_slice(&789u32.to_le_bytes());
    plain[metadata + 6..metadata + 10].copy_from_slice(&[0, 0, 1, 1]);
    std::fs::write(path, fdsave::encode(&plain).unwrap()).unwrap();
}

/// Go：`hotel_trace_test.go:49` `hotelLoadFixtureGame`。#32 使用正式酒店 input 與既有四槽交易；
/// fixture 只驗 E1，不稱原版玩家收據。
fn hotel_load_fixture_game() -> Game {
    let node = |t: &str, next: &str, options: Vec<campaign::ChoiceOption>| campaign::Node {
        r#type: t.to_string(),
        next: next.to_string(),
        options,
        ..Default::default()
    };
    let to_hotel = vec![campaign::ChoiceOption { to: String::from("hotel_ch02"), ..Default::default() }];
    let graph = campaign::Campaign {
        start: String::from("hotel_current"),
        nodes: BTreeMap::from([
            (String::from("hotel_current"), node("hotel", "town_current", vec![])),
            (String::from("town_current"), node("town", "", vec![])),
            (String::from("town_ch02"), node("town", "", to_hotel)),
            (String::from("hotel_ch02"), node("hotel", "town_ch02", vec![])),
        ]),
        ..Default::default()
    };
    let mut g = Game {
        camp: Some(campaign::new_runner(&graph)),
        gold: 99,
        handler_chapter: 8,
        party_members: BTreeMap::from([(4, true)]),
        native_hotel_ui: Some(NativeHotelUIAssets::default()),
        native_class_ui: Some(NativeClassUIAssets::default()),
        ..Default::default()
    };
    g.enter_node();
    g.hotel_sel = NATIVE_HOTEL_SERVICE_LOAD;
    g
}

fn hotel_input(f: impl FnOnce(&mut NativeHotelInput)) -> NativeHotelInput {
    let mut input = NativeHotelInput::default();
    f(&mut input);
    input
}

fn enter() -> NativeHotelInput {
    hotel_input(|i| i.enter = true)
}

fn down() -> NativeHotelInput {
    hotel_input(|i| i.down = true)
}

fn esc() -> NativeHotelInput {
    hotel_input(|i| i.esc = true)
}

fn node_id(g: &Game) -> String {
    g.camp.as_ref().unwrap().node_id().to_string()
}

fn party(g: &Game, id: i64) -> bool {
    g.party_members.get(&id).copied().unwrap_or(false)
}

fn native_save_env(d: &std::path::Path) -> Vec<(&'static str, std::ffi::OsString)> {
    vec![("FD2_NATIVE_SAVE", d.join("FD2.SAV").into()), ("FD2_MUTE", "1".into())]
}

/// Go：`hotel_trace_test.go:62` `TestNativeHotelLoadPublishesAtHotelThenReturnsThroughAuthoredTown`
#[test]
fn test_native_hotel_load_publishes_at_hotel_then_returns_through_authored_town() {
    let name = "test_native_hotel_load_publishes_at_hotel_then_returns_through_authored_town";
    let Some(dir) = f_child_env(module_path!(), name, native_save_env) else {
        return;
    };
    write_native_restore_fixture(&dir.join("FD2.SAV"), 1);
    let mut g = hotel_load_fixture_game();
    g.handle_native_hotel_input(enter());
    assert!(g.native_hotel_mode == "loadslots" && g.native_hotel_slot_sel == 0, "服務2沒有開四槽");
    g.handle_native_hotel_input(enter());
    assert!(g.native_hotel_mode == "loadslots" && g.gold == 99, "空槽發布交易");
    g.handle_native_hotel_input(down());
    g.handle_native_hotel_input(enter());
    assert!(
        g.native_hotel_mode == "loaded"
            && node_id(&g) == "hotel_ch02"
            && g.hotel_sel == 2
            && g.gold == 789
            && party(&g, 9)
            && !party(&g, 4)
            && g.handler_chapter == 1
            && g.native_chapter_slot_baseline.as_ref().is_some_and(|b| b.slot == 1),
        "loaded={} node={} gold={} party={:?} msg={:?} err={:?}",
        g.native_hotel_mode,
        node_id(&g),
        g.gold,
        g.party_members,
        g.msg,
        g.load_err
    );
    g.handle_native_hotel_input(enter());
    assert!(
        g.native_hotel_mode == "menu"
            && g.native_hotel_prompt_return
            && g.hotel_sel == 2
            && node_id(&g) == "hotel_ch02",
        "確認後沒有保持服務2酒店選單"
    );
    g.handle_native_hotel_input(esc());
    assert!(
        node_id(&g) == "town_ch02" && g.native_hotel_mode.is_empty() && !g.native_hotel_prompt_return,
        "ESC沒有沿已編寫邊返回loaded town"
    );
}

/// Go：`hotel_trace_test.go:89` `TestNativeHotelLoadRejectsEveryPreparationGateBeforeRosterPublication`
#[test]
fn test_native_hotel_load_rejects_every_preparation_gate_before_roster_publication() {
    let name = "test_native_hotel_load_rejects_every_preparation_gate_before_roster_publication";
    let Some(dir) = f_child_env(module_path!(), name, native_save_env) else {
        return;
    };
    for chapter in [22u8, 23, 24, 27, 28, 29] {
        write_native_restore_fixture(&dir.join("FD2.SAV"), chapter);
        let mut g = hotel_load_fixture_game();
        g.handle_native_hotel_input(enter());
        g.handle_native_hotel_input(down());
        g.handle_native_hotel_input(enter());
        assert!(
            g.native_hotel_mode == "load_blocked"
                && g.gold == 99
                && g.handler_chapter == 8
                && party(&g, 4)
                && g.native_chapter_slot_baseline.is_none()
                && node_id(&g) == "hotel_current",
            "chapter {chapter}: gate1槽沒有零交易拒收 mode={} msg={:?}",
            g.native_hotel_mode,
            g.msg
        );
        g.handle_native_hotel_input(enter());
        assert!(
            g.native_hotel_mode == "loadslots" && g.native_hotel_slot_sel == 1,
            "chapter {chapter}: 拒絕提示沒有回原槽列表"
        );
        g.handle_native_hotel_input(esc());
        assert!(
            g.native_hotel_mode == "menu" && g.hotel_sel == 2 && node_id(&g) == "hotel_current",
            "chapter {chapter}: 取消沒有回原酒店"
        );
    }
}

/// Go：`hotel_trace_test.go:112` `TestNativeHotelLoadRejectsTamperAndMissingAuthoredReturn`
#[test]
fn test_native_hotel_load_rejects_tamper_and_missing_authored_return() {
    let name = "test_native_hotel_load_rejects_tamper_and_missing_authored_return";
    let Some(dir) = f_child_env(module_path!(), name, native_save_env) else {
        return;
    };
    let path = dir.join("FD2.SAV");
    for bad in ["checksum", "return-edge"] {
        write_native_restore_fixture(&path, 1);
        let mut g = hotel_load_fixture_game();
        if bad == "checksum" {
            let mut raw = std::fs::read(&path).unwrap();
            raw[0x123] ^= 1;
            std::fs::write(&path, raw).unwrap();
        } else {
            let graph = g.camp.as_mut().unwrap().c.as_mut().unwrap();
            graph.nodes.get_mut("town_ch02").unwrap().options = Vec::new();
        }
        g.handle_native_hotel_input(enter());
        g.handle_native_hotel_input(down());
        g.handle_native_hotel_input(enter());
        assert!(
            g.native_hotel_mode == "loadslots"
                && g.gold == 99
                && g.handler_chapter == 8
                && party(&g, 4)
                && g.native_chapter_slot_baseline.is_none()
                && node_id(&g) == "hotel_current",
            "{bad}: 壞來源沒有原子拒收 mode={} msg={:?}",
            g.native_hotel_mode,
            g.msg
        );
    }
}

/// Go：`hotel_trace_test.go:140` `TestNativeHotelLoadUsesJSONBoundaryReader`
#[test]
fn test_native_hotel_load_uses_json_boundary_reader() {
    let name = "test_native_hotel_load_uses_json_boundary_reader";
    let envs = |d: &std::path::Path| {
        vec![("FD2_NATIVE_SAVE", "".into()), ("XDG_DATA_HOME", d.into()), ("FD2_MUTE", "1".into())]
    };
    if f_child_env(module_path!(), name, envs).is_none() {
        return;
    }
    let mut g = hotel_load_fixture_game();
    g.locale_catalog = load_official_locale("zh-Hant").unwrap();
    let d = SaveData {
        node: String::from("town_ch02"),
        chapter: 1,
        gold: 543,
        flags: Some(BTreeMap::from([(String::from("keep"), true)])),
        ..Default::default()
    };
    let p = save_slot_path(2);
    std::fs::create_dir_all(std::path::Path::new(&p).parent().unwrap()).unwrap();
    std::fs::write(&p, serde_json::to_vec(&d).unwrap()).unwrap();
    g.handle_native_hotel_input(enter());
    g.handle_native_hotel_input(down());
    g.handle_native_hotel_input(down());
    g.handle_native_hotel_input(enter());
    let flags = &g.camp.as_ref().unwrap().flags;
    assert!(
        g.native_hotel_mode == "loaded"
            && node_id(&g) == "hotel_ch02"
            && g.gold == 543
            && flags.get("keep").copied().unwrap_or(false),
        "JSON邊界未回相應酒店 mode={} node={} gold={} flags={flags:?} msg={} loadErr={}",
        g.native_hotel_mode,
        node_id(&g),
        g.gold,
        g.msg,
        g.load_err
    );
}
