//! Go 來源：`remake/cmd/fd2/hotel_trace_test.go`（R4 E 組，期望值不改）。
//!
//! 未移植：`:62`、`:89`、`:112`、`:140`（酒店讀檔，經 `confirmNativeHotelLoadSlot`，`r8`）、`:172`（傳聞
//! story 節點，主題屬於 A／B 的劇情流程）、`:209`（以環境變數啟用、輸出畫面的同源探針）。

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
