//! Go `remake/internal/campaign/campaign_test.go` 的逐條移植。
//!
//! Go 以 `t.Run` 分的子測試在 Rust 是同一個測試裡的迴圈，失敗訊息帶上子測試名稱；Go 的指標同一性
//! 比較（`&beats[i]`）改比索引。需要 `remake/assets` 的測試經 `crate::test_support`。

use super::*;
use crate::test_support::{docs_file, remake_asset, temp_dir};

const SAMPLE: &str = r#"{
  "title": "test",
  "start": "intro",
  "flags": {"retried": false},
  "nodes": {
    "intro":  {"type":"story","lines":[{"speaker":0,"text":"哈囉"}],"next":"b1"},
	"b1":     {"type":"battle","scenario":"ch01.json","protect":"亞雷斯","on_win":"pick","on_lose":"retreat"},
    "retreat":{"type":"story","lines":[{"speaker":4,"text":"撤退!"}],"set_flags":{"retried":true},"next":"b1"},
    "pick":   {"type":"choice","prompt":"走哪邊?","options":[
                 {"label":"山路","to":"end"},
                 {"label":"祕道","to":"end","if":"retried"}]},
    "end":    {"type":"ending","text":"完"}
  }
}"#;

/// Go `os.WriteFile(filepath.Join(t.TempDir(), name), raw)`：回傳路徑。
fn write_temp(tag: &str, name: &str, raw: &str) -> String {
    let path = temp_dir(tag).join(name);
    std::fs::write(&path, raw).expect("寫入暫存檔");
    path.to_string_lossy().into_owned()
}

/// Go 的 helper `load`。
fn load_sample() -> Campaign {
    let p = write_temp("campaign-sample", "c.json", SAMPLE);
    load(&p).unwrap_or_else(|e| panic!("{e}"))
}

/// Go `Load("../../assets/scenarios/campaign_full.json")`。
fn full_campaign() -> Option<Campaign> {
    let path = remake_asset("scenarios/campaign_full.json")?;
    Some(load(&path).unwrap_or_else(|e| panic!("{e}")))
}

/// Go 以 `../../` 加上 campaign 節點的 `assets/...` 相對路徑讀檔；Rust 換到 `FD2_REMAKE_ASSETS`。
fn asset_path(rel: &str) -> String {
    let rest = rel.strip_prefix("assets/").unwrap_or_else(|| panic!("{rel} 不是 assets/ 相對路徑"));
    remake_asset(rest).unwrap_or_else(|| panic!("缺少 {rel}"))
}

/// Go `CompileHandlerBinding("../../assets/cutscenes/bindings/<name>")`。
fn compile_binding(name: &str) -> Option<(Vec<Beat>, Vec<HandlerCompileIssue>)> {
    let path = remake_asset(&format!("cutscenes/bindings/{name}"))?;
    Some(compile_handler_binding(&path).unwrap_or_else(|e| panic!("{name}: {e}")))
}

fn node<'a>(c: &'a Campaign, id: &str) -> Option<&'a Node> {
    c.nodes.get(id)
}

fn flag(r: &Runner, key: &str) -> bool {
    r.flags.get(key).copied().unwrap_or(false)
}

/// Go：`TestWinPath`
#[test]
fn win_path() {
    let mut r = new_runner(&load_sample());
    assert_eq!(r.node().unwrap().r#type, "story", "start 應為 story,得 {}", r.node().unwrap().r#type);
    r.advance(""); // intro → b1
    r.advance("win"); // b1 → pick
    assert_eq!(r.cur, "pick", "勝利應到 pick,得 {}", r.cur);
    let n = r.visible().len(); // retried=false → 祕道隱藏
    assert_eq!(n, 1, "choice 應只剩 1 選項,得 {n}");
    r.advance("opt0");
    assert_eq!(r.node().unwrap().r#type, "ending", "應到 ending,得 {}", r.node().unwrap().r#type);
    assert_eq!(r.advance(""), "", "ending 後應結束");
}

/// Go：`TestBattleProtectTargetIsEditable`
#[test]
fn battle_protect_target_is_editable() {
    let mut r = new_runner(&load_sample());
    r.advance("");
    let got = &r.node().unwrap().protect;
    assert_eq!(got, "亞雷斯", "battle protect={got:?}, want editable target");
}

/// Go：`TestLoseRouteAndFlags`
#[test]
fn lose_route_and_flags() {
    let mut r = new_runner(&load_sample());
    r.advance(""); // intro → b1
    r.advance("lose"); // b1 → retreat(敗北路線,非 game over)
    assert_eq!(r.cur, "retreat", "敗北應到 retreat,得 {}", r.cur);
    r.advance(""); // retreat → b1(set retried)
    assert!(flag(&r, "retried"), "retreat 應設 retried 旗標");
    r.advance("win"); // b1 → pick
    let n = r.visible().len(); // retried=true → 祕道出現
    assert_eq!(n, 2, "choice 應有 2 選項,得 {n}");
    r.advance("opt1");
    assert_eq!(r.cur, "end", "祕道應到 end,得 {}", r.cur);
}

/// Go：`TestLoadValidation`
#[test]
fn load_validation() {
    let p = write_temp("campaign-bad", "bad.json", r#"{"start":"x","nodes":{"a":{"type":"story","next":"nope"}}}"#);
    assert!(load(&p).is_err(), "start 不存在應報錯");
}

/// Go：`TestPreparationRequiredRosterAcceptsNoncityOwnerAndRejectsInvalidData`
#[test]
fn preparation_required_roster_accepts_noncity_owner_and_rejects_invalid_data() {
    for (name, fields, valid) in [
        (
            "非城鎮",
            r#""type":"preparation","required_party_identities":[24],"preparation_front_identities":[24]"#,
            true,
        ),
        (
            "城鎮",
            r#""type":"preparation","cancel":"end","required_party_identities":[24],"preparation_front_identities":[24]"#,
            true,
        ),
        ("錯節點", r#""type":"story","required_party_identities":[24]"#, false),
        ("身份越界", r#""type":"preparation","required_party_identities":[32]"#, false),
        ("身份重複", r#""type":"preparation","required_party_identities":[24,24]"#, false),
        (
            "前置未必出",
            r#""type":"preparation","required_party_identities":[24],"preparation_front_identities":[21]"#,
            false,
        ),
        (
            "前置重複",
            r#""type":"preparation","required_party_identities":[24],"preparation_front_identities":[24,24]"#,
            false,
        ),
    ] {
        let raw =
            format!(r#"{{"start":"prep","nodes":{{"prep":{{{fields},"next":"end"}},"end":{{"type":"ending"}}}}}}"#);
        let path = write_temp("campaign-prep", "campaign.json", &raw);
        let result = load(&path);
        assert_eq!(result.is_ok(), valid, "{name}: Load err={:?}，預期合法={valid}", result.err());
    }
}

/// Go：`TestNativeEndingPrefixRejectsUnprovenCampaignAdmission`
#[test]
fn native_ending_prefix_rejects_unproven_campaign_admission() {
    for (name, raw) in [
        (
            "wrong node type",
            r#"{"start":"end","nodes":{"end":{"type":"story","native_ending_prefix":{"timeline":"assets/endings/native_2bce5.json","handler":"0x2bce5","chapter":29,"mode":"source_bound_e1_terminal_hold"}}}}"#,
        ),
        (
            "wrong handler",
            r#"{"start":"end","nodes":{"end":{"type":"ending","native_ending_prefix":{"timeline":"assets/endings/native_2bce5.json","handler":"0x2c548","chapter":29,"mode":"source_bound_e1_terminal_hold"}}}}"#,
        ),
        (
            "wrong chapter",
            r#"{"start":"end","nodes":{"end":{"type":"ending","native_ending_prefix":{"timeline":"assets/endings/native_2bce5.json","handler":"0x2bce5","chapter":28,"mode":"source_bound_e1_terminal_hold"}}}}"#,
        ),
        (
            "wrong mode",
            r#"{"start":"end","nodes":{"end":{"type":"ending","native_ending_prefix":{"timeline":"assets/endings/native_2bce5.json","handler":"0x2bce5","chapter":29,"mode":"ready"}}}}"#,
        ),
    ] {
        let path = write_temp("campaign-ending", "invalid-native-ending.json", raw);
        assert!(load(&path).is_err(), "{name}: unproven native ending prefix was accepted");
    }
}

/// Go：`TestNativeMapRuntimeRejectsUnsupportedOrUnanchoredState`
#[test]
fn native_map_runtime_rejects_unsupported_or_unanchored_state() {
    for (name, raw) in [
        (
            "hud only",
            r#"{"start":"b","nodes":{"b":{"type":"battle","native_map_hud":{"display_gate_a":1,"display_gate_b":1,"anchor_x":1}}}}"#,
        ),
        (
            "inherited HUD only",
            r#"{"start":"b","nodes":{"b":{"type":"battle","native_map_hud_inherited":{"display_gate_b":1}}}}"#,
        ),
        (
            "fixed and inherited",
            r#"{"start":"b","nodes":{"b":{"type":"battle","native_map_view":{"camera_x":1,"camera_y":13,"cursor_x":8,"cursor_y":17,"visible_cursor_x":7,"visible_cursor_y":4,"range_mode":1},"native_map_hud":{"display_gate_a":1,"display_gate_b":1,"anchor_x":1},"native_map_hud_inherited":{"display_gate_b":1}}}}"#,
        ),
        (
            "unsupported inherited",
            r#"{"start":"b","nodes":{"b":{"type":"battle","native_map_view":{"camera_x":1,"camera_y":13,"cursor_x":8,"cursor_y":17,"visible_cursor_x":7,"visible_cursor_y":4,"range_mode":1},"native_map_hud_inherited":{"display_gate_b":0}}}}"#,
        ),
        (
            "bad gate",
            r#"{"start":"b","nodes":{"b":{"type":"battle","native_map_view":{"camera_x":1,"camera_y":13,"cursor_x":8,"cursor_y":17,"visible_cursor_x":7,"visible_cursor_y":4,"range_mode":1},"native_map_hud":{"display_gate_a":256,"display_gate_b":1,"anchor_x":1}}}}"#,
        ),
        (
            "missing range mode",
            r#"{"start":"b","nodes":{"b":{"type":"battle","native_map_view":{"camera_x":1,"camera_y":13,"cursor_x":8,"cursor_y":17,"visible_cursor_x":7,"visible_cursor_y":4},"native_map_hud":{"display_gate_a":1,"display_gate_b":1,"anchor_x":1}}}}"#,
        ),
        (
            "runtime selector",
            r#"{"start":"b","nodes":{"b":{"type":"battle","native_map_view":{"camera_x":1,"camera_y":13,"cursor_x":8,"cursor_y":17,"visible_cursor_x":7,"visible_cursor_y":4,"range_mode":11},"native_map_hud":{"display_gate_a":1,"display_gate_b":1,"anchor_x":1}}}}"#,
        ),
    ] {
        let path = write_temp("campaign-map", "invalid-native-map.json", raw);
        assert!(load(&path).is_err(), "{name}: unsupported or unanchored native map state must fail closed");
    }
}

/// Go：`TestNativeMapRuntimeAllowsProvenViewWithoutUnsourcedHUD`
#[test]
fn native_map_runtime_allows_proven_view_without_unsourced_hud() {
    let raw = r#"{"start":"b","nodes":{"b":{"type":"battle","native_map_view":{"camera_x":9,"camera_y":49,"cursor_x":14,"cursor_y":54,"visible_cursor_x":5,"visible_cursor_y":5,"range_mode":0}}}}"#;
    let path = write_temp("campaign-view", "view-only.json", raw);
    let c = load(&path).unwrap_or_else(|e| panic!("{e}"));
    let n = node(&c, "b");
    let ok =
        n.is_some_and(|n| n.native_map_view.is_some_and(|v| v.range_mode == Some(0)) && n.native_map_hud.is_none());
    assert!(ok, "view-only native state={n:?}");
}

/// 第 1／27／29／30 章四個測試共用：檢查固定視圖與繼承 HUD。
fn check_view_and_inherited(id: &str, want: [i64; 6], range_mode: i64) {
    let Some(c) = full_campaign() else {
        return;
    };
    let n = node(&c, id);
    let Some(n) =
        n.filter(|n| n.native_map_view.is_some() && n.native_map_hud.is_none() && n.native_map_hud_inherited.is_some())
    else {
        panic!("{id} native map state={n:?}");
    };
    let view = n.native_map_view.unwrap();
    assert!(
        view.camera_x == want[0]
            && view.camera_y == want[1]
            && view.cursor_x == want[2]
            && view.cursor_y == want[3]
            && view.visible_cursor_x == want[4]
            && view.visible_cursor_y == want[5]
            && view.range_mode == Some(range_mode),
        "{id} native map view={view:?}"
    );
    let inherited = n.native_map_hud_inherited.unwrap();
    assert_eq!(inherited.display_gate_b, 1, "{id} inherited native map HUD={inherited:?}");
}

/// Go：`TestFullCampaignCarriesVerifiedChapterOneNativeMapRuntime`
#[test]
fn full_campaign_carries_verified_chapter_one_native_map_runtime() {
    // 這組值是 START 那條路徑的：ch00 handler 走完之後聚焦 slot 0，
    // 由 TestCh00CompiledHandlerCarriesItsExactRuntimeRosterIntoChapterOne
    // 以完整 ch00→ch01 交接實跑驗證。
    //
    // 另有一組不同的原版觀測：由隨遊戲附帶的 FD2.SAV 走 CONTINUE 進第一關時，
    // dosgolem 在取得操作權那一刻量到 camera (1,13)、cursor (8,17)、
    // visible (7,4)（收據
    // docs/data/ui-traces/fd2-move-confirm-cursor-20260909.json）。兩條入口
    // 本來就可能不同——存檔會帶自己的視圖——目前沒有同時涵蓋兩條的收據，
    // 所以節點常數跟著 START，CONTINUE 那組只作紀錄。見
    // docs/knowledge-base/104-regression-baseline-review-20260909.md。
    check_view_and_inherited("battle_ch01", [0, 13, 7, 14, 7, 1], 1);
}

/// Go：`TestFullCampaignCarriesVerifiedChapter27ViewAndInheritedHUD`
#[test]
fn full_campaign_carries_verified_chapter27_view_and_inherited_hud() {
    check_view_and_inherited("battle_ch27", [9, 49, 14, 54, 5, 5], 0);
}

/// Go：`TestFullCampaignCarriesVerifiedChapter29ViewAndInheritedHUD`
#[test]
fn full_campaign_carries_verified_chapter29_view_and_inherited_hud() {
    // 原版正常LOAD及ACT86完成後的checkpoint86，見第29章主契約。
    check_view_and_inherited("battle_ch29", [9, 55, 14, 60, 5, 5], 1);
}

/// Go：`TestFullCampaignCarriesVerifiedChapter30ViewAndInheritedHUD`
#[test]
fn full_campaign_carries_verified_chapter30_view_and_inherited_hud() {
    check_view_and_inherited("battle_ch30", [16, 17, 22, 23, 6, 6], 1);
}

fn typed(t: &str) -> Node {
    Node { r#type: t.into(), ..Default::default() }
}

/// Go：`TestInventoryGateRequiresBothTargetsAndRoutesWithoutPlayerChoice`
#[test]
fn inventory_gate_requires_both_targets_and_routes_without_player_choice() {
    let mut nodes = BTreeMap::new();
    nodes.insert(
        "gate".to_string(),
        Node {
            r#type: "inventory_gate".into(),
            item_id: Some(100),
            if_present: "continue".into(),
            if_missing: "bad".into(),
            ..Default::default()
        },
    );
    nodes.insert("continue".to_string(), typed("preparation"));
    nodes.insert("bad".to_string(), typed("ending"));
    let c = Campaign { start: "gate".into(), nodes, ..Default::default() };
    for (outcome, want) in [("present", "continue"), ("missing", "bad")] {
        let mut r = new_runner(&c);
        let got = r.advance(outcome);
        assert!(
            got == want && r.cur == want,
            "inventory gate {outcome} = {got:?} / current {:?}, want {want:?}",
            r.cur
        );
    }

    for (name, raw) in [
        (
            "missing item",
            r#"{"start":"gate","nodes":{"gate":{"type":"inventory_gate","if_present":"yes","if_missing":"no"},"yes":{"type":"ending"},"no":{"type":"ending"}}}"#,
        ),
        (
            "missing arm",
            r#"{"start":"gate","nodes":{"gate":{"type":"inventory_gate","item_id":100,"if_present":"yes"},"yes":{"type":"ending"}}}"#,
        ),
    ] {
        let path = write_temp("campaign-gate", "invalid-gate.json", raw);
        assert!(load(&path).is_err(), "{name}: invalid inventory gate must fail closed");
    }
}

/// Go：`TestInventoryRecipeRequiresExactDataAndRoutesWithoutPlayerChoice`
#[test]
fn inventory_recipe_requires_exact_data_and_routes_without_player_choice() {
    let mut nodes = BTreeMap::new();
    nodes.insert(
        "recipe".to_string(),
        Node {
            r#type: "inventory_recipe".into(),
            item_ids: vec![0xd1, 0xd2],
            slot_count: 16,
            required_matches: 2,
            reward_item_id: Some(100),
            if_crafted: "yes".into(),
            if_insufficient: "no".into(),
            ..Default::default()
        },
    );
    nodes.insert("yes".to_string(), typed("cutscene"));
    nodes.insert("no".to_string(), typed("cutscene"));
    let c = Campaign { start: "recipe".into(), nodes, ..Default::default() };
    for (outcome, want) in [("crafted", "yes"), ("insufficient", "no")] {
        let mut r = new_runner(&c);
        let got = r.advance(outcome);
        assert!(
            got == want && r.cur == want,
            "inventory recipe {outcome} = {got:?} / current {:?}, want {want:?}",
            r.cur
        );
    }

    for (name, raw) in [
        (
            "missing items",
            r#"{"start":"recipe","nodes":{"recipe":{"type":"inventory_recipe","slot_count":16,"required_matches":6,"reward_item_id":100,"if_crafted":"yes","if_insufficient":"no"},"yes":{"type":"ending"},"no":{"type":"ending"}}}"#,
        ),
        (
            "missing arm",
            r#"{"start":"recipe","nodes":{"recipe":{"type":"inventory_recipe","item_ids":[209],"slot_count":16,"required_matches":1,"reward_item_id":100,"if_crafted":"yes"},"yes":{"type":"ending"}}}"#,
        ),
    ] {
        let path = write_temp("campaign-recipe", "invalid-recipe.json", raw);
        assert!(load(&path).is_err(), "{name}: invalid inventory recipe must fail closed");
    }
}

/// `n` 是 `type` 的 cutscene、綁定 `binding`、接 `next`。
fn is_bound_cutscene(n: Option<&Node>, binding: &str, next: &str) -> bool {
    n.is_some_and(|n| n.r#type == "cutscene" && n.handler_binding == binding && n.next == next)
}

/// Go：`TestCampaignFullPrologueFollowsOriginalTextGroups`
#[test]
fn campaign_full_prologue_follows_original_text_groups() {
    let Some(c) = full_campaign() else {
        return;
    };
    let prologue = node(&c, "story_ch00_handler");
    assert!(
        c.start == "story_ch00_handler"
            && is_bound_cutscene(prologue, "assets/cutscenes/bindings/ch00_pre.json", "battle_ch01"),
        "campaign must start through the complete editable ch00 handler: start={:?} node={prologue:?}",
        c.start
    );
    let throne = node(&c, "story_ch01_palace_throne");
    assert!(
        throne.is_some_and(|t| t.beats.len() == 5
            && t.beats[2].line == 0
            && t.beats[2].count == 6
            && t.beats[4].line == 6
            && t.beats[4].count == 13),
        "throne beats do not preserve FDTXT #0/#1 groups: {throne:?}"
    );
    let grass = node(&c, "story_ch01_palace_path");
    let Some(grass) = grass.filter(|g| g.actors.len() == 2 && g.actors[1].x == 10 && g.actors[1].y == 47) else {
        panic!("grass initial Ares placement = {grass:?}, want proven (10,47)");
    };
    let (mut first_walk, mut second_walk) = (false, false);
    for beat in &grass.beats {
        first_walk =
            first_walk || beat.op == "walk" && beat.fig == 4 && beat.from_x == 13 && beat.x == 10 && beat.y == 47;
        second_walk = second_walk || beat.op == "walk" && beat.fig == 4 && beat.x == 7 && beat.y == 46;
    }
    assert!(first_walk && second_walk, "grass Ares walks missing: {:?}", grass.beats);
    let ch05 = node(&c, "story_ch05");
    assert!(
        is_bound_cutscene(ch05, "assets/cutscenes/bindings/ch04_pre.json", "battle_ch05"),
        "player chapter 5 must execute the recovered zero-based ch04 pre-handler: {ch05:?}"
    );
    let ch15 = node(&c, "story_ch15");
    assert!(
        is_bound_cutscene(ch15, "assets/cutscenes/bindings/ch14_pre.json", "battle_ch15")
            && ch15.unwrap().beats.is_empty(),
        "chapter 15 must execute the recovered dynamic ch14 pre-handler: {ch15:?}"
    );
    let ch17 = node(&c, "story_ch17");
    assert!(
        is_bound_cutscene(ch17, "assets/cutscenes/bindings/ch16_pre.json", "battle_ch17")
            && ch17.unwrap().beats.is_empty(),
        "chapter 17 must execute the recovered conditional ch16 pre-handler: {ch17:?}"
    );
    let post15 = node(&c, "postbattle_ch15_persist");
    assert!(
        is_bound_cutscene(post15, "assets/cutscenes/bindings/ch14_post.json", "town_ch16")
            && post15.unwrap().beats.is_empty(),
        "chapter 15 must execute zero-based ch14_post before town: {post15:?}"
    );
    let post14 = node(&c, "postbattle_ch14_persist");
    assert!(
        post14
            .is_some_and(|p| p.handler_binding == "assets/cutscenes/bindings/ch13_post.json" && p.next == "town_ch15"),
        "chapter 14 must execute zero-based ch13_post before town: {post14:?}"
    );
    let (battle2, post2) = (node(&c, "battle_ch02"), node(&c, "story_ch02_post"));
    assert!(
        battle2.is_some_and(|b| b.on_win == "story_ch02_post")
            && is_bound_cutscene(post2, "assets/cutscenes/bindings/ch01_post.json", "town_ch03"),
        "chapter2 battle must flow through editable post handler: battle={battle2:?} post={post2:?}"
    );
    let (previous_post, pre2) = (node(&c, "story_ch02"), node(&c, "story_ch02_pre"));
    assert!(
        previous_post
            .is_some_and(|p| p.handler_binding == "assets/cutscenes/bindings/ch00_post.json" && p.next == "town_ch02")
            && pre2.is_some_and(
                |p| p.handler_binding == "assets/cutscenes/bindings/ch01_pre.json" && p.next == "battle_ch02"
            ),
        "chapter2 must preserve post→town/preparation→pre→battle handlers: previous={previous_post:?} pre={pre2:?}"
    );
    let pre3 = node(&c, "story_ch03");
    assert!(
        is_bound_cutscene(pre3, "assets/cutscenes/bindings/ch02_pre.json", "battle_ch03"),
        "chapter3 must enter through editable ch02_pre handler: {pre3:?}"
    );
    let (battle3, post3) = (node(&c, "battle_ch03"), node(&c, "story_ch03_post"));
    assert!(
        battle3.is_some_and(|b| b.on_win == "story_ch03_post")
            && is_bound_cutscene(post3, "assets/cutscenes/bindings/ch02_post.json", "town_ch04"),
        "chapter3 battle must flow through Tino's editable post handler: battle={battle3:?} post={post3:?}"
    );
    for (chapter, town) in [(11, "town_ch12"), (15, "town_ch16")] {
        let battle_id = format!("battle_ch{chapter:02}");
        let post_id = format!("postbattle_ch{chapter:02}_persist");
        let (battle_node, post) = (node(&c, &battle_id), node(&c, &post_id));
        let Some(post) = post
            .filter(|p| battle_node.is_some_and(|b| b.on_win == post_id) && p.r#type == "cutscene" && p.next == town)
        else {
            panic!(
                "chapter{chapter} material acquisition must sync before {town}: battle={battle_node:?} post={post:?}"
            );
        };
        if chapter == 11 || chapter == 15 {
            let want_binding = if chapter == 15 {
                "assets/cutscenes/bindings/ch14_post.json"
            } else {
                "assets/cutscenes/bindings/ch10_post.json"
            };
            assert!(
                post.handler_binding == want_binding && post.beats.is_empty(),
                "chapter{chapter} must preserve dynamic post handler: {post:?}"
            );
            continue;
        }
        assert!(
            post.beats.len() == 2
                && post.beats[0].op == "sync_party"
                && post.beats[1].op == "set_chapter"
                && post.beats[1].chapter == Some(chapter),
            "chapter{chapter} persistent post beats={:?}",
            post.beats
        );
    }
    let battle27 = node(&c, "battle_ch27");
    let intro27 = node(&c, "story_ch27_post_intro");
    let gate = node(&c, "inventory_gate_ch27_sky_key");
    let success = node(&c, "story_ch27_post_sky_key_success");
    let bad_ending = node(&c, "ending_ch27_no_sky_key");
    assert!(
        battle27.is_some_and(|b| b.on_win == "story_ch27_post_intro")
            && intro27.is_some_and(|i| i.handler_binding == "assets/cutscenes/bindings/ch26_post_intro.json"
                && i.next == "inventory_gate_ch27_sky_key")
            && gate.is_some_and(|g| g.r#type == "inventory_gate"
                && g.item_id == Some(0x64)
                && g.if_present == "story_ch27_post_sky_key_success"
                && g.if_missing == "story_ch27_post_sky_key_missing"),
        "chapter27 must preserve original sky-key inventory branch: battle={battle27:?} gate={gate:?}"
    );
    assert!(
        is_bound_cutscene(success, "assets/cutscenes/bindings/ch26_post_sky_key_success.json", "preparation_ch28")
            && success.unwrap().beats.is_empty(),
        "sky-key success must preserve the proven ch26_post success tail without reusing ch27_post: {success:?}"
    );
    let post29 = node(&c, "postbattle_ch29_persist");
    assert!(
        is_bound_cutscene(post29, "assets/cutscenes/bindings/ch28_post.json", "preparation_ch30")
            && post29.unwrap().beats.is_empty(),
        "chapter29 must use the recovered zero-based raw ch28 post handler: {post29:?}"
    );
    let missing = node(&c, "story_ch27_post_sky_key_missing");
    let Some(missing) = missing.filter(|m| {
        m.r#type == "story"
            && m.script == "assets/story/ch27.json"
            && m.scene == "缺少天空之鑰的離別(分支)"
            && m.next == "ending_ch27_no_sky_key"
    }) else {
        panic!("missing sky-key branch must preserve editable farewell scene: {missing:?}");
    };
    let Some(story_path) = remake_asset("story/ch27.json") else {
        return;
    };
    #[derive(serde::Deserialize, Default)]
    #[serde(default)]
    struct StoryScene {
        label: String,
        lines: Vec<serde_json::Value>,
    }
    #[derive(serde::Deserialize, Default)]
    #[serde(default)]
    struct StoryDoc {
        scenes: Vec<StoryScene>,
    }
    let story_doc: StoryDoc = serde_json::from_slice(&std::fs::read(story_path).unwrap()).unwrap();
    let mut found_missing = false;
    for scene in &story_doc.scenes {
        if scene.label == missing.scene {
            found_missing = true;
            assert_eq!(scene.lines.len(), 17, "missing sky-key scene lines={} want 17", scene.lines.len());
        }
    }
    assert!(found_missing, "missing sky-key scene label {:?} absent from ch27 story", missing.scene);
    let ch28 = node(&c, "story_ch28");
    assert!(
        is_bound_cutscene(ch28, "assets/cutscenes/bindings/ch27_pre.json", "battle_ch28"),
        "chapter28 must execute raw-index-27 ch27_pre handler: {ch28:?}"
    );
    let ch29 = node(&c, "story_ch29");
    assert!(
        is_bound_cutscene(ch29, "assets/cutscenes/bindings/ch28_pre.json", "battle_ch29"),
        "chapter29 must execute raw-index-28 ch28_pre handler: {ch29:?}"
    );
    assert!(
        bad_ending.is_some_and(|e| e.r#type == "ending"
            && !e.text.is_empty()
            && e.native_ending_prefix.as_ref().is_some_and(|p| p.timeline == "assets/endings/native_2bce5.json"
                && p.handler == "0x2bce5"
                && p.chapter == 26
                && p.mode == NATIVE_ENDING_PREFIX_SOURCE_BOUND_E1)),
        "missing sky key must reach an editable bad ending: {bad_ending:?}"
    );
    let battle21 = node(&c, "battle_ch21");
    let intro21 = node(&c, "story_ch21_post_sky_key_intro");
    let recipe21 = node(&c, "inventory_recipe_ch21_sky_key");
    let crafted21 = node(&c, "story_ch21_post_sky_key_crafted");
    let insufficient21 = node(&c, "story_ch21_post_sky_key_insufficient");
    let intro_ok = intro21.is_some_and(|i| {
        i.script == "assets/story/ch21.json"
            && i.scene == "浴血決戰,團長真身現形——萊汀舊識瑪爾"
            && i.beats.len() == 2
            && i.beats[0].op == "layout_units"
            && i.beats[0].source == "0x2415b"
            && i.beats[0].layout.as_ref().is_some_and(|l| l.units.len() == 26 && l.cam_x == 336 && l.cam_y == 240)
            && i.beats[1].source == "0x24182"
            && i.beats[1].line == 7
            && i.beats[1].count == 10
            && i.beats[1].native_dialogues.len() == 10
            && i.next == "inventory_recipe_ch21_sky_key"
    });
    assert!(
        battle21.is_some_and(|b| b.on_win == "story_ch21_post_sky_key_intro") && intro_ok,
        "chapter21 must preserve editable pre-recipe FDTXT #5: battle={battle21:?} intro={intro21:?}"
    );
    let units = &intro21.unwrap().beats[0].layout.as_ref().unwrap().units;
    let (first, special) = (units[0], units[25]);
    assert!(
        first.slot == 0
            && first.x == 21
            && first.y == 14
            && first.pose == 2
            && special.slot == 25
            && special.x == 23
            && special.y == 14
            && special.pose == 1,
        "chapter21 recovered layout endpoints drifted: first={first:?} special={special:?}"
    );
    assert!(
        recipe21.is_some_and(|r| r.r#type == "inventory_recipe"
            && r.item_ids == vec![0xd1, 0xd2, 0xd3, 0xd4, 0xd5, 0xd6]
            && r.slot_count == 16
            && r.required_matches == 6
            && r.reward_item_id == Some(0x64)
            && r.if_crafted == "story_ch21_post_sky_key_crafted"
            && r.if_insufficient == "story_ch21_post_sky_key_insufficient"),
        "chapter21 sky-key recipe does not match original nested loops: {recipe21:?}"
    );
    let Some(crafted21) = crafted21 else {
        panic!("chapter21 crafted arm is missing");
    };
    assert_eq!(crafted21.beats.len(), 11, "chapter21 crafted arm beats={}, want 11", crafted21.beats.len());
    let got_crafted_dialog = [&crafted21.beats[0], &crafted21.beats[2], &crafted21.beats[4], &crafted21.beats[6]];
    let want_crafted_dialog = [[0, 1, 1], [1, 3, 3], [4, 2, 2], [6, 10, 10]];
    for (index, beat) in got_crafted_dialog.iter().enumerate() {
        let want = want_crafted_dialog[index];
        assert!(
            beat.op == "dialog"
                && beat.line == want[0]
                && beat.count == want[1]
                && beat.native_dialogues.len() as i64 == want[2],
            "chapter21 crafted dialog group {index}={beat:?}, want line/count/native={want:?}"
        );
    }
    let (act63, act64) = (&crafted21.beats[1], &crafted21.beats[3]);
    assert!(
        crafted21.scene == "希爾法鑄成傳說法器「天空之鑰」"
            && act63.op == "act"
            && act63.source == "0x2425f"
            && act63.acting.len() == 4
            && act64.op == "act"
            && act64.source == "0x2429a"
            && act64.acting.len() == 5
            && crafted21.beats[5].op == "native_ch20_sky_key_sequence"
            && crafted21.beats[5].source == "0x242c9"
            && crafted21.beats[5].native_ch20_sky_key.as_ref().is_some_and(|s| s.is_recovered_contract())
            && crafted21.next == "town_ch22",
        "crafted arm must preserve all editable #7..#10 dialogue and town22: {crafted21:?}"
    );
    let Some(insufficient21) = insufficient21.filter(|i| {
        i.scene == "決議直赴巨塔(未鑄成天空之鑰)"
            && i.beats.len() == 5
            && i.beats[0].op == "dialog"
            && i.beats[0].line == 0
            && i.beats[0].count == 4
            && i.beats[0].native_dialogues.len() == 4
            && i.next == "town_ch22"
    }) else {
        panic!("insufficient arm must preserve all editable #6 dialogue and town22: {insufficient21:?}");
    };
    for (id, node) in [("crafted", crafted21), ("insufficient", insufficient21)] {
        let tail = &node.beats[node.beats.len() - 4..];
        assert!(
            tail[0].op == "join"
                && tail[0].char_id == 24
                && tail[1].op == "join"
                && tail[1].char_id == 23
                && tail[2].op == "sync_party"
                && tail[3].op == "set_chapter"
                && tail[3].chapter == Some(21),
            "chapter21 {id} common JOIN/sync/chapter tail = {tail:?}"
        );
    }
}

/// Go：`TestCampaignFullPostbattleBindingsUseVerifiedRawOwner`
#[test]
fn campaign_full_postbattle_bindings_use_verified_raw_owner() {
    let Some(c) = full_campaign() else {
        return;
    };
    let want = [
        ("postbattle_ch04_persist", "assets/cutscenes/bindings/ch03_post.json"),
        ("postbattle_ch05_persist", "assets/cutscenes/bindings/ch04_post.json"),
        ("postbattle_ch06_persist", "assets/cutscenes/bindings/ch05_post.json"),
        ("postbattle_ch07_persist", "assets/cutscenes/bindings/ch06_post.json"),
        ("postbattle_ch08_persist", "assets/cutscenes/bindings/ch07_post.json"),
        ("postbattle_ch09_persist", "assets/cutscenes/bindings/ch08_post.json"),
        ("postbattle_ch10_persist", "assets/cutscenes/bindings/ch09_post.json"),
        ("postbattle_ch11_persist", "assets/cutscenes/bindings/ch10_post.json"),
        ("postbattle_ch12_persist", "assets/cutscenes/bindings/ch11_post.json"),
        ("postbattle_ch13_persist", "assets/cutscenes/bindings/ch12_post.json"),
        ("postbattle_ch14_persist", "assets/cutscenes/bindings/ch13_post.json"),
        ("postbattle_ch15_persist", "assets/cutscenes/bindings/ch14_post.json"),
        ("postbattle_ch16_persist", "assets/cutscenes/bindings/ch15_post.json"),
        ("postbattle_ch17_persist", "assets/cutscenes/bindings/ch16_post.json"),
        ("postbattle_ch18_persist", "assets/cutscenes/bindings/ch17_post.json"),
        ("postbattle_ch19_persist", "assets/cutscenes/bindings/ch18_post.json"),
        ("postbattle_ch20_persist", "assets/cutscenes/bindings/ch19_post.json"),
        ("postbattle_ch22_persist", "assets/cutscenes/bindings/ch21_post.json"),
        ("postbattle_ch23_persist", "assets/cutscenes/bindings/ch22_post.json"),
        ("postbattle_ch24_persist", "assets/cutscenes/bindings/ch23_post.json"),
        ("postbattle_ch25_persist", "assets/cutscenes/bindings/ch24_post.json"),
        ("postbattle_ch26_persist", "assets/cutscenes/bindings/ch25_post.json"),
        ("postbattle_ch28_persist", "assets/cutscenes/bindings/ch27_post.json"),
        ("postbattle_ch29_persist", "assets/cutscenes/bindings/ch28_post.json"),
    ];
    let mut errors = Vec::new();
    for (node_id, want_binding) in want {
        let Some(n) = node(&c, node_id) else {
            errors.push(format!("missing node {node_id}"));
            continue;
        };
        if n.handler_binding != want_binding {
            errors.push(format!("{node_id} binding={:?}, want {want_binding:?}", n.handler_binding));
        }
        if want_binding.is_empty() && !n.beats.is_empty() {
            errors.push(format!(
                "{node_id} unbound node retained {} inline beats and can bypass the runtime guard",
                n.beats.len()
            ));
        }
    }
    assert!(errors.is_empty(), "{}", errors.join("\n"));
}

/// Go：`TestCampaignFullPostBattleTownContractMatchesOriginalShopChapters`
#[test]
fn campaign_full_post_battle_town_contract_matches_original_shop_chapters() {
    #[derive(serde::Deserialize, Default)]
    #[serde(default)]
    struct ShopRecord {
        chapter: i64,
        town: String,
        kind: String,
        goods: Vec<Good>,
    }
    #[derive(serde::Deserialize, Default)]
    #[serde(default)]
    struct ShopData {
        shops: Vec<ShopRecord>,
    }

    let Some(shops_path) = docs_file("data/shops.json") else {
        return;
    };
    let source: ShopData = serde_json::from_slice(&std::fs::read(shops_path).unwrap()).unwrap();
    let mut town_by_chapter: BTreeMap<i64, String> = BTreeMap::new();
    let mut goods_by_chapter_kind: BTreeMap<String, Vec<Good>> = BTreeMap::new();
    let secret_gate_by_chapter: BTreeMap<i64, [i64; 2]> = [
        (2, [0, 0x54]),
        (3, [1, 0x5f]),
        (4, [2, 0x6a]),
        (5, [3, 0x57]),
        (6, [4, 0x62]),
        (7, [0, 0x6d]),
        (8, [1, 0x5a]),
        (9, [2, 0x65]),
        (10, [3, 0x70]),
        (11, [4, 0x5d]),
        (12, [0, 0x5e]),
        (13, [1, 0x69]),
        (14, [2, 0x56]),
        (15, [3, 0x61]),
        (16, [4, 0x6c]),
        (17, [0, 0x58]),
        (18, [1, 0x64]),
        (19, [2, 0x6f]),
        (20, [3, 0x5c]),
        (21, [4, 0x67]),
        (22, [0, 0x68]),
        (26, [4, 0x58]),
        (27, [0, 0x63]),
    ]
    .into_iter()
    .collect();
    let town_variant_by_chapter: BTreeMap<i64, i64> = [
        (2, 0),
        (3, 2),
        (4, 0),
        (5, 0),
        (6, 2),
        (7, 2),
        (8, 0),
        (9, 0),
        (10, 2),
        (11, 2),
        (12, 1),
        (13, 1),
        (14, 1),
        (15, 1),
        (16, 1),
        (17, 1),
        (18, 1),
        (19, 2),
        (20, 1),
        (21, 1),
        (22, 1),
        (26, 0),
        (27, 0),
    ]
    .into_iter()
    .collect();
    for shop in &source.shops {
        if let Some(previous) = town_by_chapter.get(&shop.chapter) {
            assert_eq!(
                previous, &shop.town,
                "chapter {} shop town names disagree: {previous:?} / {:?}",
                shop.chapter, shop.town
            );
        }
        town_by_chapter.insert(shop.chapter, shop.town.clone());
        goods_by_chapter_kind.insert(format!("{:02}/{}", shop.chapter, shop.kind), shop.goods.clone());
    }
    let got_chapters: Vec<i64> = (1..=30).filter(|c| town_by_chapter.contains_key(c)).collect();
    let want_chapters = vec![2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 26, 27];
    assert_eq!(got_chapters, want_chapters, "shops.json chapter set = {got_chapters:?}, want {want_chapters:?}");

    let Some(campaign) = full_campaign() else {
        return;
    };
    let follow_post_battle_path = |battle_id: &str, target_id: &str| {
        let battle = node(&campaign, battle_id);
        let Some(battle) = battle.filter(|b| b.r#type == "battle") else {
            panic!("missing battle node {battle_id}: {battle:?}");
        };
        let mut current = battle.on_win.clone();
        for _ in 0..campaign.nodes.len() {
            if current == target_id {
                return;
            }
            let Some(n) = node(&campaign, &current) else {
                panic!("{battle_id} on_win path reaches missing node {current:?}");
            };
            assert!(n.r#type != "battle", "{battle_id} reaches next battle {current:?} before {target_id}");
            if n.r#type == "inventory_gate" {
                current = n.if_present.clone(); // 此契約驗證持有天空之鑰的原版正常路徑；missing arm 另有專測。
                continue;
            }
            if n.r#type == "inventory_recipe" {
                current = n.if_crafted.clone();
                continue;
            }
            assert!(
                n.r#type == "story" || n.r#type == "cutscene" || n.r#type == "event",
                "{battle_id} on_win path reaches {} node {current:?} before {target_id}",
                n.r#type
            );
            assert!(!n.next.is_empty(), "{battle_id} on_win path stops at {current:?} before {target_id}");
            current = n.next.clone();
        }
        panic!("{battle_id} on_win path did not reach {target_id}");
    };

    for (&chapter, town_name) in &town_by_chapter {
        let ctx = format!("shop_chapter_{chapter:02}");
        let town_id = format!("town_ch{chapter:02}");
        // shops.json 的 chapter 是下一場戰鬥章：例如 chapter 2 的
        // 羅德鎮位於 battle_ch01 戰後，不是 battle_ch02 戰後。
        follow_post_battle_path(&format!("battle_ch{:02}", chapter - 1), &town_id);

        let town = node(&campaign, &town_id);
        let Some(town) = town.filter(|t| t.r#type == "town" && &t.town == town_name) else {
            panic!("{ctx}: {town_id} = {town:?}, want town {town_name:?}");
        };
        let want_variant = town_variant_by_chapter.get(&chapter).copied().unwrap_or(0);
        assert_eq!(
            town.native_town_variant,
            Some(want_variant),
            "{ctx}: {town_id} native town variant = {:?}, want {want_variant}",
            town.native_town_variant
        );
        let want_gate = secret_gate_by_chapter.get(&chapter).copied().unwrap_or_default();
        assert!(
            town.native_secret_gate.as_ref().is_some_and(|g| g.selection == want_gate[0]
                && g.scan_code == want_gate[1]
                && g.to == format!("shop_ch{chapter:02}_secret")),
            "{ctx}: {town_id} native secret gate = {:?}, want selection={} scan={:#x}",
            town.native_secret_gate,
            want_gate[0],
            want_gate[1]
        );
        let preparation_id = format!("preparation_ch{chapter:02}");
        let next_story = if chapter == 2 { "story_ch02_pre".to_string() } else { format!("story_ch{chapter:02}") };
        let opt = |label: &str, to: String| ChoiceOption { label: label.into(), to, r#if: String::new() };
        let want_options = vec![
            opt("酒店：打聽消息", format!("hotel_ch{chapter:02}")),
            opt("武器店", format!("shop_ch{chapter:02}_weapon")),
            opt("出口：出戰整備", preparation_id.clone()),
            opt("道具店", format!("shop_ch{chapter:02}_item")),
            opt("教會", format!("church_ch{chapter:02}")),
            ChoiceOption {
                label: "神秘商店".into(),
                to: format!("shop_ch{chapter:02}_secret"),
                r#if: format!("found_secret_ch{chapter:02}"),
            },
        ];
        assert_eq!(town.options, want_options, "{ctx}: {town_id} options = {:?}, want {want_options:?}", town.options);
        let mut town_runner = Runner { c: Some(campaign.clone()), cur: town_id.clone(), flags: BTreeMap::new() };
        let visible = town_runner.visible();
        assert_eq!(
            visible.len(),
            5,
            "{ctx}: {town_id} visible facilities before secret unlock = {visible:?}, want five"
        );
        town_runner.flags.insert(format!("found_secret_ch{chapter:02}"), true);
        let visible = town_runner.visible();
        assert!(
            visible.len() == 6 && visible[5].to == format!("shop_ch{chapter:02}_secret"),
            "{ctx}: {town_id} visible facilities after secret unlock = {visible:?}, want hidden secret shop sixth"
        );
        for kind in ["weapon", "item", "secret"] {
            let shop_id = format!("shop_ch{chapter:02}_{kind}");
            let shop = node(&campaign, &shop_id);
            let want_goods = goods_by_chapter_kind.get(&format!("{chapter:02}/{kind}")).cloned().unwrap_or_default();
            assert!(
                shop.is_some_and(|s| s.r#type == "shop" && s.next == town_id && s.goods == want_goods),
                "{ctx}: {shop_id} = {shop:?}, want editable original goods {want_goods:?} and return to {town_id}"
            );
        }
        for return_id in
            [&want_options[0].to, &want_options[1].to, &want_options[3].to, &want_options[4].to, &want_options[5].to]
        {
            let n = node(&campaign, return_id);
            assert!(n.is_some_and(|n| n.next == town_id), "{ctx}: {return_id} must return to {town_id}: {n:?}");
        }
        let preparation = node(&campaign, &preparation_id);
        let Some(preparation) =
            preparation.filter(|p| p.r#type == "preparation" && p.next == next_story && p.cancel == town_id)
        else {
            panic!("{ctx}: {preparation_id} = {preparation:?}, want preparation leading to {next_story}");
        };
        assert_eq!(
            preparation.prompt, "要進入戰場嗎？",
            "{ctx}: {preparation_id} prompt={:?}, want town departure confirmation",
            preparation.prompt
        );
        let story = node(&campaign, &next_story);
        assert!(
            story.is_some_and(|s| s.r#type == "story" || s.r#type == "cutscene"),
            "{ctx}: {town_id} departure target = {story:?}, want next chapter story/cutscene"
        );
    }

    let mut errors = Vec::new();
    for chapter in 1..=30 {
        if town_by_chapter.contains_key(&chapter) {
            continue;
        }
        if campaign.nodes.contains_key(&format!("town_ch{chapter:02}")) {
            errors.push(format!("chapter {chapter} has no shops.json records but defines town_ch{chapter:02}"));
        }
    }
    assert!(errors.is_empty(), "{}", errors.join("\n"));

    for chapter in [23, 24, 25, 28, 29, 30] {
        let ctx = format!("preparation_chapter_{chapter:02}");
        let prep_id = format!("preparation_ch{chapter:02}");
        follow_post_battle_path(&format!("battle_ch{:02}", chapter - 1), &prep_id);
        let prep = node(&campaign, &prep_id);
        let Some(prep) = prep.filter(|p| p.r#type == "preparation") else {
            panic!("{ctx}: {prep_id} = {prep:?}, want non-shop preparation intermission");
        };
        assert!(
            prep.prompt == "要記錄戰況嗎？" && prep.cancel.is_empty(),
            "{ctx}: {prep_id} prompt/cancel = {:?}/{:?}, want preparation-only save prompt",
            prep.prompt,
            prep.cancel
        );
        assert_eq!(
            prep.next,
            format!("story_ch{chapter:02}"),
            "{ctx}: {prep_id} next = {:?}, want departure to chapter story",
            prep.next
        );
        if chapter >= 28 {
            assert_eq!(
                prep.party_limit, 19,
                "{ctx}: {prep_id} party_limit={}, want late-route original cap 19",
                prep.party_limit
            );
        }
        if chapter < 28 {
            assert_eq!(
                prep.party_limit, 0,
                "{ctx}: {prep_id} party_limit={}, want default original cap 15",
                prep.party_limit
            );
        }
    }
    let battle30 = node(&campaign, "battle_ch30");
    assert!(
        battle30.is_some_and(|b| b.on_win == "postbattle_ch30_terminal" && !b.ending_party_snapshot_on_win),
        "battle_ch30 must end campaign: {battle30:?}"
    );
    let ending = node(&campaign, "ending");
    assert!(
        ending.is_some_and(|e| e.native_ending_prefix.as_ref().is_some_and(|p| p.timeline
            == "assets/endings/native_2bce5.json"
            && p.handler == "0x2bce5"
            && p.chapter == 30
            && p.mode == NATIVE_ENDING_PREFIX_SOURCE_BOUND_E1)),
        "terminal ending prefix={ending:?}"
    );
}

/// Go：`TestCampaignFullChapter16BattlePreservesTownShopPreparationBoundary`
#[test]
fn campaign_full_chapter16_battle_preserves_town_shop_preparation_boundary() {
    let Some(c) = full_campaign() else {
        return;
    };
    let mut r = new_runner(&c);
    r.cur = "battle_ch16".into();

    let got = r.advance("win");
    assert_eq!(got, "postbattle_ch16_persist", "battle_ch16 win={got:?}, want postbattle_ch16_persist");
    let post = r.node();
    assert!(
        post.is_some_and(|p| p.r#type == "cutscene"
            && !p.handler_binding.is_empty()
            && p.next == "town_ch17"
            && p.beats.is_empty()),
        "ch16 postbattle={post:?}, want bound cutscene→town with no inline bypass"
    );
    let got = r.advance("");
    assert_eq!(got, "town_ch17", "postbattle_ch16→town={got:?}");
    let town = r.node();
    assert!(
        town.is_some_and(|t| t.r#type == "town") && r.visible().len() == 5,
        "town_ch17 node={town:?} visible={}, want five non-secret entries",
        r.visible().len()
    );

    // The battle must not skip the hub: visit the weapon shop and return to the
    // same town before entering preparation.
    for (outcome, want, what) in [
        ("opt1", "shop_ch17_weapon", "town_ch17 weapon option"),
        ("", "town_ch17", "shop_ch17_weapon→town"),
        ("opt2", "preparation_ch17", "town_ch17 preparation option"),
        ("cancel", "town_ch17", "preparation_ch17 cancel"),
        ("opt2", "preparation_ch17", "town_ch17 preparation retry"),
        ("confirm", "story_ch17", "preparation_ch17 confirm"),
        ("", "battle_ch17", "story_ch17→battle"),
    ] {
        let got = r.advance(outcome);
        assert_eq!(got, want, "{what}={got:?}");
    }
}

/// Go：`TestEveryContinuingBattleSyncsBeforeOriginalIntermission`
#[test]
fn every_continuing_battle_syncs_before_original_intermission() {
    let Some(c) = full_campaign() else {
        return;
    };
    let mut want_intermission: BTreeMap<i64, String> = BTreeMap::new();
    for chapter in 1..=21 {
        want_intermission.insert(chapter, format!("town_ch{:02}", chapter + 1));
    }
    for chapter in [22, 23, 24, 28, 29] {
        want_intermission.insert(chapter, format!("preparation_ch{:02}", chapter + 1));
    }
    want_intermission.insert(25, "town_ch26".into());
    want_intermission.insert(26, "town_ch27".into());
    want_intermission.insert(27, "preparation_ch28".into());

    let count_sync = |node_id: &str, n: &Node| -> i64 {
        let mut beats = n.beats.clone();
        if !n.handler_binding.is_empty() {
            let path = asset_path(&n.handler_binding);
            match compile_handler_binding(&path) {
                Ok((compiled, issues)) if issues.is_empty() => beats = compiled,
                result => panic!("{node_id} handler compile result={result:?}"),
            }
        }
        fn walk(bs: &[Beat]) -> i64 {
            bs.iter().map(|beat| i64::from(beat.op == "sync_party") + walk(&beat.then) + walk(&beat.r#else)).sum()
        }
        walk(&beats)
    };

    'chapters: for chapter in 1..=29 {
        let ctx = format!("chapter_{chapter:02}");
        let want = &want_intermission[&chapter];
        let battle_id = format!("battle_ch{chapter:02}");
        let battle = node(&c, &battle_id);
        let Some(battle) = battle.filter(|b| b.r#type == "battle" && !b.on_win.is_empty()) else {
            panic!("{ctx}: {battle_id} = {battle:?}");
        };
        if let Some(first) = node(&c, &battle.on_win) {
            assert!(
                !(first.r#type == "town" || first.r#type == "preparation" || first.r#type == "ending"),
                "{ctx}: {battle_id} has bare on_win edge to runtime-clearing {} node {}",
                first.r#type,
                battle.on_win
            );
        }

        let (mut current, mut syncs) = (battle.on_win.clone(), 0);
        for _ in 0..c.nodes.len() {
            let Some(n) = node(&c, &current) else {
                panic!("{ctx}: {battle_id} path reaches missing node {current:?}");
            };
            if current.starts_with("postbattle_") && n.handler_binding.is_empty() && n.beats.is_empty() {
                assert_eq!(&n.next, want, "{ctx}: {battle_id} unresolved postbattle next={}, want {want}", n.next);
                continue 'chapters; // runtime guard keeps this fail-closed before the intermission
            }
            syncs += count_sync(&current, n);
            match n.r#type.as_str() {
                "town" | "preparation" => {
                    assert_eq!(&current, want, "{ctx}: {battle_id} first intermission={current}, want {want}");
                    let want_syncs = 1;
                    assert_eq!(
                        syncs, want_syncs,
                        "{ctx}: {battle_id} sync_party count before {current}={syncs}, want {want_syncs}"
                    );
                    continue 'chapters;
                }
                "inventory_gate" => current = n.if_present.clone(), // ch27 normal/hidden-chapter route
                "inventory_recipe" => current = n.if_crafted.clone(), // ch21; insufficient arm has its own contract test
                "story" | "cutscene" | "event" => current = n.next.clone(),
                "battle" | "ending" => {
                    panic!("{ctx}: {battle_id} reaches {} node {current} before original intermission {want}", n.r#type)
                }
                _ => panic!("{ctx}: {battle_id} path has unsupported node {current} type={}", n.r#type),
            }
        }
        panic!("{ctx}: {battle_id} path did not reach {want}");
    }

    let battle30 = node(&c, "battle_ch30");
    assert!(
        battle30.is_some_and(|b| b.on_win == "postbattle_ch30_terminal" && !b.ending_party_snapshot_on_win),
        "terminal battle must retain the full original postbattle edge: {battle30:?}"
    );
    let post = node(&c, "postbattle_ch30_terminal");
    assert!(
        post.is_some_and(|p| p.handler_binding == "assets/cutscenes/bindings/ch29_post.json" && p.next == "ending"),
        "缺完整25757戰後：{post:?}"
    );
}

/// TestFullCampaignGraphReachesEndingThroughAllBattles verifies the editable
/// campaign graph only. It deliberately does not claim that native handlers,
/// original assets, or an unmodified player route have been proven here.
///
/// Go：`TestFullCampaignGraphReachesEndingThroughAllBattles`
#[test]
fn full_campaign_graph_reaches_ending_through_all_battles() {
    let Some(c) = full_campaign() else {
        return;
    };
    let mut r = new_runner(&c);
    let want_battles: Vec<String> = (1..=30).map(|i| format!("battle_ch{i:02}")).collect();
    let mut got_battles: Vec<String> = Vec::new();
    let mut intermissions: Vec<String> = Vec::new();
    let mut last_battle = String::new();
    const MAX_STEPS: usize = 400;

    for steps in 0..MAX_STEPS {
        let Some(n) = r.node().cloned() else {
            panic!("step {steps} reached missing node {:?}", r.cur);
        };
        if n.r#type == "ending" {
            assert_eq!(r.cur, "ending", "terminal node={:?}, want editable ending node", r.cur);
            assert_eq!(got_battles, want_battles, "battle order={got_battles:?}, want {want_battles:?}");
            assert_eq!(
                intermissions.len(),
                29,
                "intermission count={}, want 29 (one after each battle except ch30)",
                intermissions.len()
            );
            return;
        }

        let current = r.cur.clone();
        let mut outcome = String::new();
        match n.r#type.as_str() {
            "battle" => {
                got_battles.push(current.clone());
                last_battle = current.clone();
                outcome = "win".into();
            }
            "town" => {
                for (i, option) in r.visible().iter().enumerate() {
                    if c.nodes.get(&option.to).is_some_and(|t| t.r#type == "preparation") {
                        outcome = format!("opt{i}");
                        break;
                    }
                }
                assert!(!outcome.is_empty(), "town {current:?} has no visible preparation departure");
            }
            "preparation" => outcome = "confirm".into(),
            "inventory_recipe" => outcome = "crafted".into(),
            "inventory_gate" => outcome = "present".into(),
            "choice" => {
                for (i, option) in r.visible().iter().enumerate() {
                    if c.nodes.get(&option.to).is_some_and(|t| t.r#type != "ending") {
                        outcome = format!("opt{i}");
                        break;
                    }
                }
                assert!(!outcome.is_empty(), "choice {current:?} has no visible continuing branch");
            }
            "story" | "cutscene" | "event" | "shop" | "church" => {}
            _ => panic!("step {steps} reached unsupported graph node {current:?} type={:?}", n.r#type),
        }

        let next = r.advance(&outcome);
        if (n.r#type == "town" || n.r#type == "preparation") && !last_battle.is_empty() {
            intermissions.push(current.clone());
            last_battle.clear();
        }
        assert!(!next.is_empty(), "step {steps} ended at {current:?} before terminal ending");
    }
    panic!("campaign graph exceeded {MAX_STEPS} steps at node {:?}", r.cur);
}

/// Go：`TestEndingPartySnapshotRejectsNonTerminalEdges`
#[test]
fn ending_party_snapshot_rejects_non_terminal_edges() {
    for (name, raw) in [
        (
            "non-battle owner",
            r#"{"start":"story","nodes":{"story":{"type":"story","on_win":"ending","ending_party_snapshot_on_win":true},"ending":{"type":"ending"}}}"#,
        ),
        (
            "non-ending target",
            r#"{"start":"battle","nodes":{"battle":{"type":"battle","on_win":"story","ending_party_snapshot_on_win":true},"story":{"type":"story"}}}"#,
        ),
        (
            "missing win edge",
            r#"{"start":"battle","nodes":{"battle":{"type":"battle","ending_party_snapshot_on_win":true}}}"#,
        ),
    ] {
        let path = write_temp("campaign-snapshot", "invalid-approximate-terminal-sync.json", raw);
        assert!(load(&path).is_err(), "{name}: 非終局勝利邊接受了近似隊伍同步");
    }
}

/// Go：`TestRunnerTownUsesVisibleOptionOutcome`
#[test]
fn runner_town_uses_visible_option_outcome() {
    let mut nodes = BTreeMap::new();
    nodes.insert(
        "town".to_string(),
        Node {
            r#type: "town".into(),
            options: vec![
                ChoiceOption { label: "酒店".into(), to: "rumor".into(), ..Default::default() },
                ChoiceOption { label: "出發".into(), to: "road".into(), ..Default::default() },
            ],
            ..Default::default()
        },
    );
    nodes.insert("rumor".to_string(), typed("story"));
    nodes.insert("road".to_string(), typed("story"));
    let c = Campaign { start: "town".into(), nodes, ..Default::default() };
    let mut runner = new_runner(&c);
    let got = runner.advance("opt1");
    assert!(
        got == "road" && runner.cur == "road",
        "town opt1 transition = {got:?} / current {:?}, want road",
        runner.cur
    );
}

/// Go：`TestRunnerNativeTownSecretGateRevealsThenConfirms`
#[test]
fn runner_native_town_secret_gate_reveals_then_confirms() {
    let mut nodes = BTreeMap::new();
    nodes.insert(
        "town".to_string(),
        Node {
            r#type: "town".into(),
            native_secret_gate: Some(NativeTownSecretGate { selection: 0, scan_code: 0x54, to: "secret".into() }),
            ..Default::default()
        },
    );
    nodes.insert("secret".to_string(), Node { r#type: "shop".into(), native_hub_variant: 5, ..Default::default() });
    let c = Campaign { start: "town".into(), nodes, ..Default::default() };
    for mismatch in [[1, 0x54], [0, 0x55]] {
        let r = new_runner(&c);
        assert!(
            !r.match_native_town_secret(mismatch[0], mismatch[1]) && r.cur == "town",
            "mismatch {mismatch:?} revealed or entered hidden shop"
        );
    }
    let mut r = new_runner(&c);
    assert!(r.match_native_town_secret(0, 0x54) && r.cur == "town", "exact native gate did not reveal in place: {r:?}");
    assert!(!r.confirm_native_town_secret(4) && r.cur == "town", "visible selection dispatched hidden shop: {r:?}");
    assert!(
        r.confirm_native_town_secret(5) && r.cur == "secret",
        "revealed native selection did not enter hidden shop: {r:?}"
    );
}

/// Go：`TestCampaignFullNativeTownSecretGatesAreChapterSpecific`
#[test]
fn campaign_full_native_town_secret_gates_are_chapter_specific() {
    let Some(c) = full_campaign() else {
        return;
    };
    let want = [
        ("town_ch02", 0, 0x54),
        ("town_ch03", 1, 0x5f),
        ("town_ch04", 2, 0x6a),
        ("town_ch05", 3, 0x57),
        ("town_ch06", 4, 0x62),
        ("town_ch07", 0, 0x6d),
        ("town_ch08", 1, 0x5a),
        ("town_ch09", 2, 0x65),
        ("town_ch10", 3, 0x70),
        ("town_ch11", 4, 0x5d),
        ("town_ch12", 0, 0x5e),
        ("town_ch13", 1, 0x69),
        ("town_ch14", 2, 0x56),
        ("town_ch15", 3, 0x61),
        ("town_ch16", 4, 0x6c),
        ("town_ch17", 0, 0x58),
        ("town_ch18", 1, 0x64),
        ("town_ch19", 2, 0x6f),
        ("town_ch20", 3, 0x5c),
        ("town_ch21", 4, 0x67),
        ("town_ch22", 0, 0x68),
        ("town_ch26", 4, 0x58),
        ("town_ch27", 0, 0x63),
    ];
    assert_eq!(want.len(), 23, "test gate table has {} entries, want 23", want.len());
    for (town, selection, scan) in want {
        let Some(gate) = node(&c, town).and_then(|n| n.native_secret_gate.clone()) else {
            panic!("{town} lacks editable native_secret_gate");
        };
        assert!(
            gate.selection == selection && gate.scan_code == scan && gate.to == format!("shop_{}_secret", &town[5..]),
            "{town} gate={gate:?} want selection={selection} scan={scan:#x}"
        );
        // A chord only reveals selection 5. Confirming any visible option must
        // remain on the hub; the next Enter is a separate lifecycle step.
        let mut runner = new_runner(&c);
        runner.cur = town.into();
        assert!(
            runner.match_native_town_secret(selection, scan) && runner.cur == town,
            "{town} exact chord did not reveal in place"
        );
        assert!(
            !runner.confirm_native_town_secret(selection) && runner.cur == town,
            "{town} visible selection entered hidden shop"
        );
        assert!(
            runner.confirm_native_town_secret(5) && runner.cur == gate.to,
            "{town} revealed selection did not enter {}",
            gate.to
        );
    }
}

/// Go：`TestCampaignFullBGMUsesVerifiedChapterAndTownTables`
#[test]
fn campaign_full_bgm_uses_verified_chapter_and_town_tables() {
    let Some(c) = full_campaign() else {
        return;
    };
    // Raw 0x51e63 is indexed by the original zero-based chapter.  The
    // campaign IDs are player-facing ch01..ch30, so ch01 consumes table[0].
    let want_battle = [
        "FDMUS_019",
        "FDMUS_019",
        "FDMUS_019",
        "FDMUS_019",
        "FDMUS_003",
        "FDMUS_019",
        "FDMUS_019",
        "FDMUS_019",
        "FDMUS_003",
        "FDMUS_004",
        "FDMUS_019",
        "FDMUS_019",
        "FDMUS_019",
        "FDMUS_019",
        "FDMUS_003",
        "FDMUS_019",
        "FDMUS_004",
        "FDMUS_019",
        "FDMUS_019",
        "FDMUS_003",
        "FDMUS_019",
        "FDMUS_003",
        "FDMUS_004",
        "FDMUS_019",
        "FDMUS_003",
        "FDMUS_019",
        "FDMUS_004",
        "FDMUS_019",
        "FDMUS_019",
        "FDMUS_008",
    ];
    for (chapter, want) in want_battle.iter().enumerate() {
        let id = format!("battle_ch{:02}", chapter + 1);
        let n = node(&c, &id);
        let Some(n) = n.filter(|n| n.r#type == "battle") else {
            panic!("{id} missing battle node: {n:?}");
        };
        assert_eq!(&n.bgm, want, "{id} BGM={:?} want raw 0x51e63 table {want:?}", n.bgm);
    }
    for (id, n) in &c.nodes {
        if n.r#type == "town" {
            assert_eq!(n.bgm, "FDMUS_010", "{id} town BGM={:?} want verified FDMUS_010", n.bgm);
        }
    }
    let ending = node(&c, "ending");
    assert!(
        ending.is_some_and(|e| e.r#type == "ending"
            && !e.text.is_empty()
            && !e.text.contains("campaign_full.json 自動生成")),
        "ending text is not an editable player-facing epilogue: {ending:?}"
    );
}

/// Go：`TestCampaignFullStoryScriptCoverageMatchesAudit`
#[test]
fn campaign_full_story_script_coverage_matches_audit() {
    let Some(c) = full_campaign() else {
        return;
    };
    let story = |n: &Node| n.r#type == "story" || n.r#type == "cutscene";
    let (mut story_nodes, mut scripted) = (0, 0);
    for n in c.nodes.values().filter(|n| story(n)) {
        story_nodes += 1;
        if !n.script.is_empty() {
            scripted += 1;
        }
    }
    let (mut handler_bound, mut fallback) = (0, 0);
    for n in c.nodes.values().filter(|n| story(n)) {
        if !n.handler_binding.is_empty() {
            handler_bound += 1;
        } else if n.script.is_empty() {
            fallback += 1;
        }
    }
    let (mut retreat, mut rumor, mut postbattle, mut generic) = (0, 0, 0, 0);
    for (id, n) in &c.nodes {
        if !story(n) || !n.script.is_empty() || !n.handler_binding.is_empty() {
            continue;
        }
        if id.starts_with("retreat_") {
            retreat += 1;
        } else if id.starts_with("rumor_") {
            rumor += 1;
        } else if id.starts_with("postbattle_") {
            postbattle += 1;
        } else {
            generic += 1;
        }
    }
    // #47／#61／#79／#214以原版pending1返回標題取代retreat_ch12／ch13／ch15／ch26；現況統計入口為58，
    // 可重跑工具 tools/audit_story_script_coverage.py，不把移除的自訂台詞算成證據。
    assert!(
        story_nodes == 119
            && scripted == 9
            && handler_bound == 60
            && fallback == 50
            && retreat == 26
            && rumor == 23
            && postbattle == 0
            && generic == 1,
        "campaign story coverage changed: nodes={story_nodes} scripted={scripted} handler_bound={handler_bound} fallback={fallback} retreat={retreat} rumor={rumor} postbattle={postbattle} generic={generic}; update the audit before changing claims"
    );
}

/// Go：`TestCh24PostBindingResolvesPersistentJoins`
#[test]
fn ch24_post_binding_resolves_persistent_joins() {
    let Some(script_path) = remake_asset("cutscenes/handlers/ch24_post.json") else {
        return;
    };
    let script = load_handler_script(&script_path).unwrap_or_else(|e| panic!("{e}"));
    let unknown = script.diagnostics.get("unknown_ops").copied().unwrap_or(0);
    assert_eq!(unknown, 0, "ch24 unknown operation count={unknown}, want 0");
    let joins: Vec<&HandlerBeat> = script.beats.iter().filter(|b| b.op == "join").collect();
    assert!(
        joins.len() == 2
            && joins[0].char_id == Some(26)
            && joins[0].source.addr == "0x24e6c"
            && joins[1].char_id == Some(29)
            && joins[1].source.addr == "0x237c8",
        "ch24 persistent joins={joins:?}"
    );
    let Some((beats, issues)) = compile_binding("ch24_post.json") else {
        return;
    };
    assert!(
        !beats.is_empty() && issues.is_empty(),
        "ch24 binding must compile its proven prefix and joins: beats={beats:?} issues={issues:?}"
    );
    let mut joined = Vec::new();
    let mut spawn_groups = Vec::new();
    let mut dialog_upper = Vec::new();
    let mut dialog_pages = Vec::new();
    for beat in &beats {
        if beat.op == "join" {
            joined.push(beat.char_id);
        }
        if beat.op == "spawn" {
            assert!(beat.group == 2 && beat.raw_placement_gate == Some(0), "ch24 proven FDFIELD materializer={beat:?}");
            spawn_groups.push(beat.group);
        }
        if beat.op == "dialog" {
            let Some(upper) = beat.upper else {
                panic!("ch24 dialog {beat:?} lost its raw FFEC..FFEF placement");
            };
            let Some(native) = &beat.native_dialogue else {
                panic!("ch24 dialog {beat:?} lost its FFFE/FFFD page projection");
            };
            dialog_upper.push(upper);
            dialog_pages.push(native.pages.len());
        }
    }
    assert_eq!(spawn_groups, vec![2], "ch24 group2 materializer was not retained: {spawn_groups:?}");
    assert_eq!(joined, vec![26, 29], "ch24 compiled join order={joined:?}, want [26 29]");
    let want_upper = vec![
        true, false, true, false, false, true, false, true, false, true, false, true, false, true, false, true, true,
        true,
    ];
    assert_eq!(dialog_upper, want_upper, "ch24 compiled dialog placements={dialog_upper:?}, want {want_upper:?}");
    let want_pages = vec![2, 2, 2, 1, 1, 3, 1, 1, 1, 1, 1, 2, 1, 3, 1, 2, 1, 1];
    assert_eq!(dialog_pages, want_pages, "ch24 compiled native pages={dialog_pages:?}, want {want_pages:?}");
}

/// Go：`TestCh25PostBindingResolvesEventBranchesAndPersistentTail`
#[test]
fn ch25_post_binding_resolves_event_branches_and_persistent_tail() {
    let Some((beats, issues)) = compile_binding("ch25_post.json") else {
        return;
    };
    assert!(issues.is_empty(), "ch25 branch dialogue issues={issues:?}");
    let mut layout: Option<&Beat> = None;
    let mut acts = BTreeSet::new();
    let mut branches: Vec<&Beat> = Vec::new();
    let mut syncs = 0;
    let mut chapter: Option<i64> = None;
    for beat in &beats {
        match beat.op.as_str() {
            "layout_units" => layout = Some(beat),
            "act" => {
                for frame in &beat.acting {
                    for unit in &frame.units {
                        if let Some(slot) = unit.slot {
                            acts.insert(slot);
                        }
                    }
                }
            }
            "if" => branches.push(beat),
            "sync_party" => syncs += 1,
            "set_chapter" => chapter = beat.chapter,
            _ => {}
        }
    }
    assert!(
        layout.is_some_and(|l| l
            .layout
            .as_ref()
            .is_some_and(|l| l.units.len() == 16 && l.cam_x == 216 && l.cam_y == 120))
            && acts.contains(&0)
            && acts.contains(&1)
            && acts.contains(&2),
        "ch25 evidence layout={layout:?} acting slots={acts:?}"
    );
    let event12 = |b: &Beat| b.condition.as_ref().is_some_and(|c| c.event_state_index == Some(12));
    assert!(
        branches.len() == 2 && event12(branches[0]) && event12(branches[1]),
        "ch25 event-state branches={branches:?}"
    );
    assert!(
        branches[0].then.len() == 5
            && branches[0].r#else.len() == 4
            && branches[1].then.len() == 18
            && branches[1].r#else.len() == 4,
        "ch25 branch dialogue sizes={}/{} {}/{}",
        branches[0].then.len(),
        branches[0].r#else.len(),
        branches[1].then.len(),
        branches[1].r#else.len()
    );
    assert!(syncs == 1 && chapter == Some(26), "ch25 persistent tail syncs={syncs} chapter={chapter:?}");
}

/// 第一拍是 `runtime_context` 時回傳它的內容。
fn runtime_context(beats: &[Beat]) -> Option<&HandlerRuntimeContext> {
    beats.first().filter(|b| b.op == "runtime_context").and_then(|b| b.runtime_context.as_ref())
}

fn spawn_group(context: &HandlerRuntimeContext, group: i64) -> i64 {
    context.spawn_groups.get(&group).copied().unwrap_or(0)
}

fn first_slot(beat: &Beat) -> Option<i64> {
    beat.acting.first().and_then(|f| f.units.first()).and_then(|u| u.slot)
}

/// Go：`TestCh13PostBindingMaterializesSpawnLayoutActAndDialogue`
#[test]
fn ch13_post_binding_materializes_spawn_layout_act_and_dialogue() {
    let Some((beats, issues)) = compile_binding("ch13_post.json") else {
        return;
    };
    assert!(issues.is_empty(), "ch13 post compile issues={issues:?}");
    let context = runtime_context(&beats);
    assert!(
        context.is_some_and(|c| c.slot_count == 67 && spawn_group(c, 1) == 1),
        "ch13 runtime context={:?}",
        &beats[..beats.len().min(1)]
    );
    let mut layout: Option<&Beat> = None;
    let mut act: Option<&Beat> = None;
    let mut dialogs: Vec<&Beat> = Vec::new();
    for beat in &beats {
        match beat.op.as_str() {
            "layout_units" => layout = Some(beat),
            "act" => act = Some(beat),
            "dialog" => dialogs.push(beat),
            _ => {}
        }
    }
    let layout_ok = layout.and_then(|b| b.layout.as_ref()).is_some_and(|l| {
        l.units.len() == 16
            && l.units[0].x == 18
            && l.units[0].y == 15
            && l.units[0].pose == 2
            && l.cam_x == 288
            && l.cam_y == 240
    });
    assert!(
        layout_ok
            && act.is_some_and(|a| a.acting.len() == 1 && first_slot(a) == Some(67))
            && dialogs.len() == 17
            && dialogs[0].scene_index == Some(0)
            && dialogs[2].line == 10
            && dialogs[3].scene_index == Some(0)
            && dialogs[16].line == 6,
        "ch13 layout={layout:?} act={act:?} dialogs={dialogs:?}"
    );
}

/// Go：`TestCh11PostBindingMaterializesLayoutActAndDialogue`
#[test]
fn ch11_post_binding_materializes_layout_act_and_dialogue() {
    let Some((beats, issues)) = compile_binding("ch11_post.json") else {
        return;
    };
    assert!(issues.is_empty(), "ch11 post compile issues={issues:?}");
    let context = runtime_context(&beats);
    assert!(
        context.is_some_and(|c| c.slot_counts == vec![45, 60]),
        "ch11 runtime context={:?}",
        &beats[..beats.len().min(1)]
    );
    let mut layout: Option<&Beat> = None;
    let mut act: Option<&Beat> = None;
    let mut dialogs: Vec<&Beat> = Vec::new();
    for beat in &beats {
        match beat.op.as_str() {
            "layout_units" => layout = Some(beat),
            "act" => act = Some(beat),
            "dialog" => dialogs.push(beat),
            _ => {}
        }
    }
    let layout_ok = layout.and_then(|b| b.layout.as_ref()).is_some_and(|l| {
        l.units.len() == 15
            && l.units[2].x == 9
            && l.units[2].y == 4
            && l.units[2].pose == 2
            && l.cam_x == 96
            && l.cam_y == 0
            && l.units[14].slot == 14
            && l.units[14].x == 10
            && l.units[14].y == 2
            && l.units[14].pose == 0
    });
    assert!(
        layout_ok
            && act.is_some_and(|a| a.acting.len() == 2 && a.acting[0].special && first_slot(a) == Some(8))
            && dialogs.len() == 10
            && dialogs[0].scene_index == Some(3)
            && dialogs[2].line == 2
            && dialogs[3].scene_index == Some(3)
            && dialogs[9].line == 9,
        "ch11 layout={layout:?} act={act:?} dialogs={dialogs:?}"
    );
}

/// Go：`TestCh08PostBindingMaterializesSpawnPanActAndDialogue`
#[test]
fn ch08_post_binding_materializes_spawn_pan_act_and_dialogue() {
    let Some((beats, issues)) = compile_binding("ch08_post.json") else {
        return;
    };
    assert!(issues.is_empty(), "ch08 post compile issues={issues:?}");
    // 第九章 r1 收據戰末 47 筆（11＋group 0 的 23＋事件 30 group 1 的 5＋事件 31 group 2／3 的 6／2）；
    // 39／45 是事件 31 跑零次／一次時由群組筆數推出。
    let context = runtime_context(&beats);
    assert!(
        context.is_some_and(|c| c.slot_counts == vec![39, 45, 47] && spawn_group(c, 4) == 1),
        "ch08 runtime context={:?}",
        &beats[..beats.len().min(1)]
    );
    let mut pan: Option<&Beat> = None;
    let mut act: Option<&Beat> = None;
    let mut dialogs: Vec<&Beat> = Vec::new();
    for beat in &beats {
        match beat.op.as_str() {
            "pan" => pan = Some(beat),
            "act" => act = Some(beat),
            "dialog" => dialogs.push(beat),
            _ => {}
        }
    }
    assert!(
        pan.is_some_and(|p| p.x == 144 && p.y == 24 && p.tile_step)
            && act.is_some_and(|a| a.acting.len() == 1 && first_slot(a) == Some(47))
            && dialogs.len() == 5
            && dialogs[0].scene_index == Some(4)
            && dialogs[4].line == 4,
        "ch08 pan={pan:?} act={act:?} dialogs={dialogs:?}"
    );
}

/// Go：`TestCh05PostBindingMaterializesSpawnPanActAndDialogue`
#[test]
fn ch05_post_binding_materializes_spawn_pan_act_and_dialogue() {
    let Some((beats, issues)) = compile_binding("ch05_post.json") else {
        return;
    };
    assert!(issues.is_empty(), "ch05 post compile issues={issues:?}");
    let context = runtime_context(&beats);
    assert!(
        context.is_some_and(|c| c.slot_counts == vec![33, 46] && spawn_group(c, 3) == 1),
        "ch05 runtime context={:?}",
        &beats[..beats.len().min(1)]
    );
    let mut pan: Option<&Beat> = None;
    let mut act: Option<&Beat> = None;
    let mut dialogs: Vec<&Beat> = Vec::new();
    let (mut join, mut spawn, mut sync, mut chapter) = (0, 0, 0, 0);
    let (mut join_at, mut spawn_at, mut pan_at, mut act_at, mut dialog_at, mut sync_at, mut chapter_at) =
        (-1i64, -1i64, -1i64, -1i64, -1i64, -1i64, -1i64);
    for (i, beat) in beats.iter().enumerate() {
        let i = i as i64;
        match beat.op.as_str() {
            "join" => (join, join_at) = (beat.char_id, i),
            "spawn" => (spawn, spawn_at) = (beat.group, i),
            "pan" => {
                pan = Some(beat);
                pan_at = i;
            }
            "act" => {
                act = Some(beat);
                act_at = i;
            }
            "dialog" => {
                if dialog_at < 0 {
                    dialog_at = i;
                }
                dialogs.push(beat);
            }
            "sync_party" => {
                sync += 1;
                sync_at = i;
            }
            "set_chapter" => {
                chapter_at = i;
                if let Some(c) = beat.chapter {
                    chapter = c;
                }
            }
            _ => {}
        }
    }
    assert!(
        pan.is_some_and(|p| p.x == 120 && p.y == 336 && p.tile_step)
            && act.is_some_and(|a| a.acting.len() == 1 && first_slot(a) == Some(34))
            && dialogs.len() == 19
            && dialogs[0].scene_index == Some(6)
            && dialogs[18].line == 18,
        "ch05 pan={pan:?} act={act:?} dialogs={dialogs:?}"
    );
    assert!(
        join == 13
            && spawn == 3
            && sync == 1
            && chapter == 6
            && join_at < spawn_at
            && spawn_at < pan_at
            && pan_at < act_at
            && act_at < dialog_at
            && dialog_at < sync_at
            && sync_at < chapter_at,
        "ch05 ordered tail join={join}@{join_at} spawn={spawn}@{spawn_at} pan@{pan_at} act@{act_at} dialog@{dialog_at} sync={sync}@{sync_at} chapter={chapter}@{chapter_at}"
    );
}

/// Go：`TestCh12PostBindingPreservesCrossSceneIndex9AndTail`
#[test]
fn ch12_post_binding_preserves_cross_scene_index9_and_tail() {
    let Some((beats, issues)) = compile_binding("ch12_post.json") else {
        return;
    };
    assert!(issues.is_empty(), "ch12 post compile issues={issues:?}");
    // Go 以指標比對最後一句對白就是 sync 前一拍；Rust 記索引。
    let mut dialogs: Vec<usize> = Vec::new();
    let (mut sync_at, mut join_at, mut chapter_at) = (-1i64, -1i64, -1i64);
    for (i, beat) in beats.iter().enumerate() {
        match beat.op.as_str() {
            "dialog" => dialogs.push(i),
            "sync_party" if beat.source == "0x238d0" => sync_at = i as i64,
            "join" if beat.source == "0x237c8" && beat.char_id == 3 => join_at = i as i64,
            "set_chapter" if beat.source == "0x231f2" && beat.chapter == Some(13) => chapter_at = i as i64,
            _ => {}
        }
    }
    assert_eq!(dialogs.len(), 12, "ch12 post dialogs={}, want FDTXT_013 index9 twelve utterances", dialogs.len());
    for (i, &at) in dialogs.iter().enumerate() {
        let dialog = &beats[at];
        let (want_scene, want_line) = if i >= 6 { (4, i as i64 - 6) } else { (3, i as i64) };
        assert!(
            dialog.source == "0x238c8"
                && dialog.script == "ch13.json"
                && dialog.scene_index == Some(want_scene)
                && dialog.line == want_line,
            "ch12 post dialog[{i}]={dialog:?}, want source 0x238c8 scene{want_scene} line{want_line}"
        );
    }
    let last_dialog = *dialogs.last().unwrap() as i64;
    assert!(
        sync_at > 0 && last_dialog == sync_at - 1 && sync_at < join_at && join_at < chapter_at,
        "ch12 post tail order dialogs_end={} sync={sync_at} join3={join_at} chapter13={chapter_at}",
        dialogs.len() - 1
    );
}

/// Go：`TestCh04PostBindingMaterializesLayoutAndDialogue`
#[test]
fn ch04_post_binding_materializes_layout_and_dialogue() {
    let Some((beats, issues)) = compile_binding("ch04_post.json") else {
        return;
    };
    assert!(issues.is_empty(), "ch04 post compile issues={issues:?}");
    let context = runtime_context(&beats);
    assert!(
        context.is_some_and(|c| c.slot_counts == vec![42, 48, 52]),
        "ch04 runtime context={:?}",
        &beats[..beats.len().min(1)]
    );
    let mut layout: Option<&Beat> = None;
    let mut dialogs: Vec<&Beat> = Vec::new();
    for beat in &beats {
        match beat.op.as_str() {
            "layout_units" => layout = Some(beat),
            "dialog" => dialogs.push(beat),
            _ => {}
        }
    }
    assert!(
        layout
            .and_then(|b| b.layout.as_ref())
            .is_some_and(|l| l.units.len() == 8 && l.units[7].slot == 41 && l.cam_x == 144 && l.cam_y == 96),
        "ch04 layout={layout:?}"
    );
    assert!(
        dialogs.len() == 17 + 2
            && dialogs[0].scene_index == Some(5)
            && dialogs[16].line == 16
            && dialogs[17].scene_index == Some(6)
            && dialogs[18].line == 1,
        "ch04 dialogs={dialogs:?}"
    );
}

/// Go：`TestCh19PostBindingPreservesRawSlotFrontier`
#[test]
fn ch19_post_binding_preserves_raw_slot_frontier() {
    let Some((beats, issues)) = compile_binding("ch19_post.json") else {
        return;
    };
    assert!(issues.is_empty(), "ch19 post compile issues={issues:?}");
    let context = runtime_context(&beats);
    assert!(context.is_some_and(|c| c.slot_count == 83), "ch19 runtime context={:?}", &beats[..beats.len().min(1)]);
    let mut patch: Option<&Beat> = None;
    let mut branch: Option<&Beat> = None;
    for beat in &beats {
        match beat.op.as_str() {
            "direct_record_patch" => patch = Some(beat),
            "if" if beat.source == "0x23ffe" => branch = Some(beat),
            _ => {}
        }
    }
    assert!(
        patch.is_some_and(|p| p.source == "0x23ec4"
            && p.direct_record_patch
                .as_ref()
                .is_some_and(|d| d.units.len() == 25 && d.view.is_some_and(|v| v.camera_x == 26 && v.camera_y == 31))),
        "ch19 direct record patch={patch:?}"
    );
    let Some(branch) = branch.filter(|b| {
        b.condition.as_ref().is_some_and(|c| c.op == "native_round_gt" && c.native_round == Some(15))
            && b.then.is_empty()
    }) else {
        panic!("ch19 round branch={branch:?}");
    };
    let (mut spawned, mut acted83, mut joined28) = (false, false, false);
    for beat in &branch.r#else {
        if beat.op == "spawn" && beat.group == 1 {
            spawned = true;
        }
        if beat.op == "act" && beat.acting.len() == 1 && first_slot(beat) == Some(83) {
            acted83 = true;
        }
        if beat.op == "join" && beat.char_id == 28 {
            joined28 = true;
        }
    }
    assert!(spawned && acted83 && joined28, "ch19 low-round arm={:?}", branch.r#else);
}
