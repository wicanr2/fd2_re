//! Rust 補充（R4 E 組）：第一章重播執行到的城鎮 → 出口提示 → ESC 回城鎮路徑。Go 沒有只測這段的單元測試
//! （`preparation_test.go` 的五個測試都經 `r8` 的 `acceptTownDeparturePrompt`／`restartPreparationSelection`），
//! 這裡以真實分離素材與 `tick_input` 驅動 `camp_input`，`drawn = true` 模擬呈現回報。

use super::*;
use crate::test_support;

fn town_preparation_campaign() -> campaign::Campaign {
    let option = |to: &str| campaign::ChoiceOption { to: to.to_string(), ..Default::default() };
    campaign::Campaign {
        start: String::from("town"),
        nodes: BTreeMap::from([
            (
                String::from("town"),
                campaign::Node {
                    r#type: String::from("town"),
                    native_town_variant: Some(0),
                    options: vec![option("town"), option("town"), option("prep"), option("town"), option("town")],
                    ..Default::default()
                },
            ),
            (
                String::from("prep"),
                campaign::Node {
                    r#type: String::from("preparation"),
                    cancel: String::from("town"),
                    next: String::from("battle"),
                    ..Default::default()
                },
            ),
        ]),
        ..Default::default()
    }
}

fn press(g: &mut Game, key: Key) {
    g.tick_input = TickInput { pressed: BTreeSet::from([key]), ..Default::default() };
    assert!(g.camp_input(), "campInput 必須攔截 {key:?}");
    g.tick_input = TickInput::default();
}

/// 每次呈現回報推進一格，直到工作結束；回傳回報次數。
fn present_until_done(g: &mut Game, limit: usize) -> usize {
    for n in 0..limit {
        let Some(job) = g.native_class_ui_job.as_mut() else {
            return n;
        };
        job.drawn = true;
        g.step_native_class_ui_lifecycle(GameInstant(1));
    }
    panic!("整備對話框工作在 {limit} 次呈現內沒有結束");
}

#[test]
fn town_exit_prompt_escape_returns_to_same_town_selector() {
    if test_support::input("FD2_ASSET_PACK", "ui/fdother_010_town_label/resource.json").is_none() {
        return;
    }
    let class_ui = load_native_class_ui_assets().expect("loadNativeClassUIAssets").unwrap();
    let mut g = Game {
        camp: Some(campaign::new_runner(&town_preparation_campaign())),
        native_preparation_ui: load_native_preparation_ui_assets().expect("loadNativePreparationUIAssets"),
        native_town_ui: load_native_town_ui_assets().expect("loadNativeTownUIAssets"),
        native_class_ui: Some(class_ui),
        ..Default::default()
    };
    g.enter_node();
    // 0x2cf01..0x2cf65：右鍵遞減、左鍵遞增；出口是第 2 格。
    press(&mut g, Key::ArrowLeft);
    press(&mut g, Key::ArrowLeft);
    assert_eq!(g.camp_sel, 2);
    press(&mut g, Key::Enter);
    assert_eq!(g.camp.as_ref().unwrap().node_id(), "prep");
    assert_eq!(g.prep_prompt_source.len(), 320 * 200, "城鎮畫面要留給出口提示當底圖");
    assert!(g.prep_town_source.as_ref().is_some_and(|d| d.matches(&g.prep_prompt_source)));
    let job = g.native_class_ui_job.as_ref().expect("出口提示的開框工作");
    assert_eq!(job.frames.len(), 10);
    assert!(job.preparation_text.as_ref().is_some_and(|t| t.kind == NATIVE_PREPARATION_DEPARTURE_QUESTION));
    // 開框期間輸入被擋下。
    press(&mut g, Key::Escape);
    assert_eq!(g.camp.as_ref().unwrap().node_id(), "prep");
    assert_eq!(present_until_done(&mut g, 20), 10);
    assert!(g.native_preparation_prompt_active());
    press(&mut g, Key::Escape);
    let job = g.native_class_ui_job.as_ref().expect("出口提示的收框工作");
    assert!(job.frames.len() == 9 && job.restore == g.prep_prompt_source);
    // 九格收框加一格還原畫面之後才回城鎮。
    assert_eq!(present_until_done(&mut g, 20), 10);
    let camp = g.camp.as_ref().unwrap();
    assert_eq!(camp.node_id(), "town");
    assert!(g.prep_prompt_source.is_empty());
    assert_eq!(g.camp_sel, 2, "從出口提示回城鎮保留選單位置（0x2CAD7）");
}

/// Go：`preparation_test.go:95` `TestPreparationPromptSourcePreservesTownAndClearsStandalone`（部分移植）
///
/// 中段的 `restartPreparationSelection`（`r8`，選人開始時保留城鎮畫面）不移植；前後兩段只經 `setup_preparation`。
#[test]
fn test_preparation_prompt_source_preserves_town_and_clears_standalone() {
    let mut g = Game { prep_prompt_source: vec![7u8; 320 * 200], ..Default::default() };
    let town_backed = campaign::Node {
        r#type: String::from("preparation"),
        cancel: String::from("town"),
        party_limit: 15,
        ..Default::default()
    };
    g.setup_preparation(Some(&town_backed));
    assert!(
        g.prep_prompt_source.len() == 320 * 200 && g.prep_prompt_source[0] == 7,
        "town-backed preparation discarded its source frame"
    );
    let standalone = campaign::Node { r#type: String::from("preparation"), party_limit: 15, ..Default::default() };
    g.setup_preparation(Some(&standalone));
    assert!(
        g.prep_prompt_source.len() == 320 * 200 && g.prep_prompt_source[0] == 0,
        "standalone preparation did not use 0x2cc04 black source"
    );
}

/// Rust 補充：`setup_preparation` 的選人順序、名額與每次進入的清除（Go `main.go:4628`）。
#[test]
fn setup_preparation_orders_join_then_roster_and_resets_selection() {
    let mut g = Game {
        party_join_order: vec![0, 9, 4],
        party_roster: BTreeMap::from([
            (0, battle::Unit::default()),
            (4, battle::Unit::default()),
            (9, battle::Unit::default()),
            (30, battle::Unit::default()),
        ]),
        party_deploy: BTreeMap::from([(9, true)]),
        prep_selecting: true,
        prep_confirm: true,
        prep_required_missing: Some(4),
        camp: Some(campaign::Runner { cur: String::from("prep28"), ..Default::default() }),
        ..Default::default()
    };
    g.setup_preparation(None);
    // 第 0 筆固定在隊首，不佔名額；JOIN 順序其後，再接不在 JOIN 裡的名冊成員。
    assert_eq!(g.prep_ids, vec![9, 4, 30]);
    assert!(g.party_deploy.is_empty(), "0x318c7 每次進入都清空出擊旗標");
    assert!(!g.prep_selecting && !g.prep_confirm && g.prep_required_missing.is_none());
    assert_eq!(g.prep_limit, 19, "第 28 章以後的名額 0x13");
    assert!(g.prep_prompt_source.len() == 320 * 200 && g.prep_town_source.is_none());
    g.camp.as_mut().unwrap().cur = String::from("prep27");
    let limited = campaign::Node { r#type: String::from("preparation"), party_limit: 7, ..Default::default() };
    g.setup_preparation(Some(&limited));
    assert_eq!(g.prep_limit, 7);
    g.setup_preparation(None);
    assert_eq!(g.prep_limit, 15);
}
