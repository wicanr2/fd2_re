//! Go `remake/internal/campaign/handler_compile_test.go` 的逐條移植。
//!
//! Go 的 `any` 值以 `serde_json::Value` 表達：Go 寫 `float64(10)` 用 `json!(10.0)`，寫 `int` 字面值
//! 用 `json!(10)`。Go 以指標同一性比較 `*bool`（例如 `beats[5].Upper != &upper`）的地方改比值。

use super::*;
use serde_json::{Value, json};
use std::collections::{BTreeMap, BTreeSet};
use std::sync::Arc;

/// Go `intPtr`。
fn ip(v: i64) -> Option<i64> {
    Some(v)
}

fn src(addr: &str, target: &str) -> HandlerSource {
    HandlerSource { addr: addr.into(), target: target.into() }
}

fn addr(a: &str) -> HandlerSource {
    src(a, "")
}

fn script(beats: Vec<HandlerBeat>) -> HandlerScript {
    HandlerScript { beats, ..Default::default() }
}

fn compile(beats: Vec<HandlerBeat>, bindings: &HandlerBindings) -> (Vec<Beat>, Vec<HandlerCompileIssue>) {
    compile_handler_script(Some(&script(beats)), bindings)
}

fn none() -> HandlerBindings {
    HandlerBindings::default()
}

fn ctx(slot_count: i64) -> HandlerBindings {
    HandlerBindings {
        runtime_context: Some(HandlerRuntimeContext { slot_count, ..Default::default() }),
        ..Default::default()
    }
}

/// `Some(Callback::new(Arc::new(閉包)))`，對應 Go 直接把 func 放進 `HandlerBindings`。
macro_rules! cb {
    ($f:expr) => {
        Some(Callback::new(Arc::new($f)))
    };
}

fn native_call(target: &str, semantic: &str, raw_args: Vec<Value>, source: HandlerSource) -> HandlerBeat {
    HandlerBeat {
        op: "native_call".into(),
        native_target: target.into(),
        native_semantic: semantic.into(),
        native_confidence: "已證實".into(),
        native_evidence: vec!["evidence".into()],
        raw_args,
        source,
        ..Default::default()
    }
}

fn unknown(target: &str, raw_args: Vec<Value>, source: HandlerSource) -> HandlerBeat {
    HandlerBeat { op: "unknown".into(), native_target: target.into(), raw_args, source, ..Default::default() }
}

fn op(o: &str) -> HandlerBeat {
    HandlerBeat { op: o.into(), ..Default::default() }
}

fn op_at(o: &str, source: HandlerSource) -> HandlerBeat {
    HandlerBeat { op: o.into(), source, ..Default::default() }
}

fn load_ch_state(chapter: i64, map: &str, roster: &str, slot_count: i64, script: &str) -> LoadCHState {
    LoadCHState {
        chapter,
        map: map.into(),
        roster: roster.into(),
        slot_count,
        script: script.into(),
        ..Default::default()
    }
}

/// Go `filepath.Join("../../assets", rel)`；缺檔時依 `FD2_REQUIRE_ORIGINAL` 略過或失敗。
fn asset(rel: &str) -> Option<String> {
    crate::test_support::remake_asset(rel)
}

/// Go：`TestCompileCh28PostUnitPresentPreservesDynamicLastSlotABI`
#[test]
fn compile_ch28_post_unit_present_preserves_dynamic_last_slot_abi() {
    let mut input = HandlerBeat {
        op: "unresolved_native_call".into(),
        native_target: "0x22253".into(),
        native_semantic: "native_indexed_presentation_schedule".into(),
        native_confidence: "已證實".into(),
        native_evidence: vec!["docs/data/ida/fd2_ch28_post_ida.txt".into()],
        raw_args: vec![json!(10.0), json!(15.0), json!(10.0), json!(15.0), json!("ebx")],
        source: src("0x25535", "0x22253"),
        ..Default::default()
    };
    let (beats, issues) = compile(vec![input.clone()], &none());
    assert!(
        issues.is_empty()
            && beats.len() == 1
            && beats[0].op == "native_unit_present"
            && beats[0].native_unit_present.is_some(),
        "beats={beats:?} issues={issues:?}"
    );
    let got = beats[0].native_unit_present.unwrap();
    assert!(
        got.last_runtime_slot
            && got.slot == 0
            && got.new_x == 15
            && got.new_y == 10
            && got.visual_x == 15
            && got.visual_y == 10,
        "payload={got:?}"
    );
    input.source.addr = "0x25534".into();
    let (beats, issues) = compile(vec![input], &none());
    assert!(beats.is_empty() && issues.len() == 1, "mismatched source compiled: beats={beats:?} issues={issues:?}");
}

/// Go：`TestCompileCh28Post35BBARequiresExactPreludeAndPresenterContract`
#[test]
fn compile_ch28_post35_bba_requires_exact_prelude_and_presenter_contract() {
    let mut input = HandlerBeat {
        op: "native_call".into(),
        native_target: "0x35bba".into(),
        native_semantic: "native_clear_record40_from".into(),
        native_confidence: "已證實".into(),
        native_evidence: vec!["docs/data/ida/fd2_ch28_post_ida.txt".into()],
        raw_args: vec![json!(20.0)],
        source: src("0x254c0", "0x35bba"),
        ..Default::default()
    };
    let (beats, issues) = compile(vec![input.clone()], &none());
    assert!(
        issues.is_empty()
            && beats.len() == 1
            && beats[0].op == "native_ch28_post_present"
            && beats[0].native_ch28_post_present.is_some_and(|p| p.is_recovered_contract()),
        "beats={beats:?} issues={issues:?}"
    );
    input.raw_args = vec![json!(19.0)];
    let (beats, issues) = compile(vec![input], &none());
    assert!(beats.is_empty() && issues.len() == 1, "wrong start slot compiled: beats={beats:?} issues={issues:?}");
}

/// Go：`TestCompileHandlerScriptUsesOnlyExplicitBindings`
#[test]
fn compile_handler_script_uses_only_explicit_bindings() {
    let upper = true;
    let beats_in = vec![
        HandlerBeat { op: "loadch".into(), chapter: ip(32), source: addr("0x3231e"), ..Default::default() },
        HandlerBeat { op: "delay".into(), ms: ip(200), ..Default::default() },
        HandlerBeat { op: "bgm".into(), track: ip(11), ..Default::default() },
        HandlerBeat { op: "bgm".into(), track: ip(-1), ..Default::default() },
        HandlerBeat { op: "pan".into(), grid_x: ip(3), grid_y: ip(34), source: addr("0x32339"), ..Default::default() },
        HandlerBeat {
            op: "dialog".into(),
            text_table: "FDTXT_033".into(),
            text_index: json!(0.0),
            source: addr("0x32382"),
            ..Default::default()
        },
        HandlerBeat { op: "act".into(), acting_id: ip(99), source: addr("0x32343"), ..Default::default() },
        HandlerBeat { op: "spawn".into(), group: ip(3), raw_placement_gate: ip(0), ..Default::default() },
        HandlerBeat { op: "join".into(), char_id: ip(12), ..Default::default() },
        op_at("unknown", addr("0xdead")),
    ];
    let bindings = HandlerBindings {
        load_ch: cb!(|input: HandlerBeat| {
            if input.source.addr == "0x3231e" {
                return (
                    load_ch_state(
                        32,
                        "assets/maps/map32",
                        "assets/maps/map32/map32_units.json",
                        30,
                        "assets/story/ch00_palace.json",
                    ),
                    true,
                );
            }
            (LoadCHState::default(), false)
        }),
        pan: cb!(|input: HandlerBeat| {
            if input.source.addr == "0x32339" && input.grid_x == Some(3) && input.grid_y == Some(34) {
                return (HandlerPoint { x: 72, y: 816, frames: 60, ..Default::default() }, true);
            }
            (HandlerPoint::default(), false)
        }),
        dialog: cb!(move |input: HandlerBeat| {
            if input.source.addr == "0x32382" && input.text_table == "FDTXT_033" && input.text_index == json!(0.0) {
                return (HandlerDialog { line: 4, count: 2, upper: Some(upper), ..Default::default() }, true);
            }
            (HandlerDialog::default(), false)
        }),
        acting: cb!(|input: HandlerBeat| {
            if input.source.addr == "0x32343" && input.acting_id == Some(99) {
                return (
                    vec![ActingFrame {
                        beats: 1,
                        units: vec![ActingUnit { fig: 0, pose: 3, ..Default::default() }],
                        ..Default::default()
                    }],
                    true,
                );
            }
            (Vec::new(), false)
        }),
        ..Default::default()
    };
    let (beats, issues) = compile(beats_in, &bindings);
    assert!(
        issues.len() == 1 && issues[0].source.addr == "0xdead",
        "issues = {issues:?}, want only unknown left explicit"
    );
    assert_eq!(beats.len(), 9, "compiled beats = {}, want 9", beats.len());
    assert!(
        beats[0].op == "loadch"
            && beats[0].load_ch.as_ref().is_some_and(|s| s.roster == "assets/maps/map32/map32_units.json"),
        "loadch lowering = {:?}",
        beats[0]
    );
    assert!(beats[1].op == "delay" && beats[1].ms == 200, "delay lowering = {:?}", beats[1]);
    assert!(
        beats[4].source == "0x32339"
            && beats[5].source == "0x32382"
            && beats[6].source == "0x32343"
            && beats[7].op == "spawn"
            && beats[7].group == 3
            && beats[7].raw_placement_gate == Some(0)
            && beats[8].op == "join"
            && beats[8].char_id == 12,
        "compiled source chain lost: {:?}",
        &beats[4..]
    );
    assert!(beats[2].track == "FDMUS_011" && beats[3].op == "bgm_stop", "BGM lowerings = {:?}", &beats[2..4]);
    assert!(beats[4].x == 72 && beats[4].y == 816 && beats[4].frames == 60, "pan lowering = {:?}", beats[4]);
    // Go 比的是指標同一性（`beats[5].Upper != &upper`）；Rust 改比值。
    assert!(
        beats[5].line == 4 && beats[5].count == 2 && beats[5].upper == Some(upper),
        "dialog lowering = {:?}",
        beats[5]
    );
    assert!(beats[6].acting.len() == 1 && beats[6].acting[0].units[0].fig == 0, "act lowering = {:?}", beats[6]);
}

/// Go：`TestCompileNativeFocusLowersToTileStepPan`
#[test]
fn compile_native_focus_lowers_to_tile_step_pan() {
    let (beats, issues) = compile(vec![unknown("0x12cea", vec![json!(23), json!(22)], addr("0xfocus"))], &none());
    assert!(issues.is_empty() && beats.len() == 1, "focus lowering beats={beats:?} issues={issues:?}");
    assert!(
        beats[0].op == "pan" && beats[0].x == 528 && beats[0].y == 552 && beats[0].frames == 30 && beats[0].tile_step,
        "focus lowering={:?}",
        beats[0]
    );
}

/// Go：`TestCompileClassifiedNativeCallsRequireEvidenceAndKeepUnresolvedClosed`
#[test]
fn compile_classified_native_calls_require_evidence_and_keep_unresolved_closed() {
    let classified = HandlerBeat {
        op: "native_call".into(),
        native_target: "0x12cea".into(),
        native_semantic: "native_camera_focus".into(),
        native_confidence: "已證實".into(),
        native_evidence: vec!["docs/data/ida/fd2_ch28_post_ida.txt".into()],
        raw_args: vec![json!(23), json!(22)],
        source: src("0x2582e", "0x12cea"),
        ..Default::default()
    };
    let (beats, issues) = compile(vec![classified.clone()], &none());
    assert!(
        issues.is_empty() && beats.len() == 1 && beats[0].op == "pan",
        "classified native call beats={beats:?} issues={issues:?}"
    );

    let mut without_evidence = classified;
    without_evidence.native_evidence = Vec::new();
    let (beats, issues) = compile(vec![without_evidence], &none());
    assert!(
        beats.is_empty()
            && issues.len() == 1
            && issues[0].reason == "native call requires semantic/confidence/evidence metadata",
        "metadata gate beats={beats:?} issues={issues:?}"
    );

    let unresolved = HandlerBeat {
        op: "unresolved_native_call".into(),
        native_target: "0x22253".into(),
        native_semantic: "native_indexed_presentation_schedule".into(),
        native_confidence: "強推論".into(),
        native_evidence: vec!["docs/data/ida/fd2_ch29_terminal_callers_ida.txt".into()],
        raw_args: vec![json!("edx"), json!("eax"), json!(255), json!(255), json!(1)],
        source: src("0x25406", "0x22253"),
        ..Default::default()
    };
    let (beats, issues) = compile(vec![unresolved], &none());
    assert!(
        beats.is_empty() && issues.len() == 1 && issues[0].op == "unresolved_native_call",
        "unresolved native call must fail closed: beats={beats:?} issues={issues:?}"
    );
}

/// Go：`TestCompileNativeStagingHelperPreservesSourcePushOrder`
#[test]
fn compile_native_staging_helper_preserves_source_push_order() {
    // 0x33d95 pushes group=6, y=16, x=0 before calling 0x35822.  Keep this
    // tuple asymmetric so a group/x reversal cannot hide behind equal values.
    let (beats, issues) =
        compile(vec![unknown("0x35822", vec![json!(6), json!(16), json!(0)], addr("0x33d95"))], &none());
    assert!(issues.is_empty() && beats.len() == 7, "staging helper beats={beats:?} issues={issues:?}");
    assert!(
        beats[0].op == "pan" && beats[0].x == 0 && beats[0].y == 384 && beats[0].frames == 30 && beats[0].tile_step,
        "staging pan={:?}",
        beats[0]
    );
    assert!(beats[1].op == "spawn" && beats[1].group == 6, "staging spawn={:?}", beats[1]);
    assert!(
        beats[2].op == "delay"
            && beats[2].ms == 300
            && beats[3].op == "palette_update"
            && beats[4].ms == 200
            && beats[5].op == "palette_update"
            && beats[6].op == "redraw",
        "staging choreography={:?}",
        &beats[2..]
    );
    assert!(
        beats[3].palette_start == 0
            && beats[3].palette_end == 255
            && beats[3].palette_delta == 255
            && beats[5].palette_start == 0
            && beats[5].palette_end == 255
            && beats[5].palette_delta == 0,
        "staging palette flash/restore={:?}/{:?}",
        beats[3],
        beats[5]
    );
}

/// Go：`TestCompilePersistentRosterCleanupIsEditable`
#[test]
fn compile_persistent_roster_cleanup_is_editable() {
    let (beats, issues) = compile(vec![unknown("0x25089", Vec::new(), addr("0x25089"))], &none());
    assert!(
        issues.is_empty() && beats.len() == 1 && beats[0].op == "reset_persistent_roster_state",
        "persistent cleanup={beats:?} issues={issues:?}"
    );
}

/// Go：`TestCompileNativeTickWaitLowersToDelay`
#[test]
fn compile_native_tick_wait_lowers_to_delay() {
    let (beats, issues) = compile(vec![unknown("0x17aa9", vec![json!(1)], HandlerSource::default())], &none());
    assert!(
        issues.is_empty() && beats.len() == 1 && beats[0].op == "delay" && beats[0].frames == 3,
        "tick wait={beats:?} issues={issues:?}"
    );
}

/// Go：`TestCompileNativeChapterLoaderRequiresCompleteLoadCHBinding`
#[test]
fn compile_native_chapter_loader_requires_complete_load_ch_binding() {
    let input = vec![HandlerBeat {
        op: "loadch".into(),
        chapter: ip(30),
        source: src("0x25870", "0x1088d"),
        ..Default::default()
    }];
    let mut state =
        load_ch_state(30, "assets/maps/map29", "assets/maps/map29/map29_units.json", 70, "assets/story/ch30.json");
    state.party_scenario = "assets/scenarios/ch30.json".into();
    let bound = state.clone();
    let bindings = HandlerBindings {
        load_ch: cb!(move |input: HandlerBeat| {
            if input.source.addr == "0x25870" {
                return (bound.clone(), true);
            }
            (LoadCHState::default(), false)
        }),
        ..Default::default()
    };
    let (beats, issues) = compile(input.clone(), &bindings);
    assert!(
        issues.is_empty()
            && beats.len() == 1
            && beats[0].op == "loadch"
            && beats[0].load_ch.as_ref().is_some_and(|s| s.chapter == state.chapter
                && s.map == state.map
                && s.roster == state.roster
                && s.slot_count == state.slot_count
                && s.script == state.script
                && s.party_scenario == state.party_scenario),
        "chapter loader={beats:?} issues={issues:?}"
    );

    let (beats, issues) = compile(input, &none());
    assert!(
        beats.is_empty() && issues.len() == 1 && issues[0].source.addr == "0x25870",
        "unbound chapter loader must stay unresolved: beats={beats:?} issues={issues:?}"
    );
}

/// Go：`TestCompileRejectsObsoleteLoadChapterTextName`
#[test]
fn compile_rejects_obsolete_load_chapter_text_name() {
    let bindings = HandlerBindings {
        load_ch: cb!(|_: HandlerBeat| {
            (
                load_ch_state(
                    30,
                    "assets/maps/map29",
                    "assets/maps/map29/map29_units.json",
                    70,
                    "assets/story/ch30.json",
                ),
                true,
            )
        }),
        ..Default::default()
    };
    let (beats, issues) = compile(vec![op_at("load_ch_text", src("0x25870", "0x1088d"))], &bindings);
    assert!(
        beats.is_empty()
            && issues.len() == 1
            && issues[0].op == "load_ch_text"
            && issues[0].reason == "operation has no proven runtime lowering",
        "obsolete load_ch_text lowering={beats:?} issues={issues:?}"
    );
}

/// Go：`TestCompileChapterAuxGraphicsOnlyLowersRecoveredCh22Callsite`
#[test]
fn compile_chapter_aux_graphics_only_lowers_recovered_ch22_callsite() {
    let (beats, issues) = compile(vec![op_at("prepare_chapter_aux_graphics", src("0x24a9a", "0x10652"))], &none());
    assert!(
        issues.is_empty() && beats.len() == 1 && beats[0].op == "native_ch22_prepare_aux",
        "chapter auxiliary graphics lowering={beats:?} issues={issues:?}"
    );
    let (bad_beats, bad_issues) =
        compile(vec![op_at("prepare_chapter_aux_graphics", src("0x108a6", "0x10652"))], &none());
    assert!(
        bad_beats.is_empty()
            && bad_issues.len() == 1
            && bad_issues[0].reason == "prepare_chapter_aux_graphics requires recovered ch22 call-site 0x24a9a",
        "unproven auxiliary caller lowering={bad_beats:?} issues={bad_issues:?}"
    );
}

/// Go：`TestCompileChapter22ReloadPreservesArchiveOwnerAndGridReset`
#[test]
fn compile_chapter22_reload_preserves_archive_owner_and_grid_reset() {
    let input = vec![
        op_at("load_res", src("0x24a4b", "0x111ba")),
        op_at("load_res", src("0x24a65", "0x111ba")),
        op_at("load_res", src("0x24a7f", "0x111ba")),
        HandlerBeat {
            op: "native_call".into(),
            source: src("0x24a92", "0x4dbfc"),
            native_target: "0x4dbfc".into(),
            native_semantic: "native_mask_raw_cells".into(),
            native_confidence: "已證實".into(),
            native_evidence: vec!["docs/data/ida/fd2_ch22_post_ida.txt".into()],
            raw_args: vec![json!("dword ptr [0x3a51]")],
            ..Default::default()
        },
    ];
    let resource = |id: i64, archive: &str, owner: &str| HandlerResource {
        resource_id: id,
        archive: archive.into(),
        owner: owner.into(),
        sfx_index: None,
    };
    let resources: BTreeMap<String, HandlerResource> = [
        ("0x24a4b".to_string(), resource(69, "FDFIELD.DAT", "0x53a51")),
        ("0x24a65".to_string(), resource(46, "FDSHAP.DAT", "0x53a5d")),
        ("0x24a7f".to_string(), resource(47, "FDSHAP.DAT", "0x53a69")),
    ]
    .into_iter()
    .collect();
    let bound = resources.clone();
    let bindings = HandlerBindings {
        resource: cb!(move |input: HandlerBeat| match bound.get(&input.source.addr) {
            Some(r) => (r.clone(), true),
            None => (HandlerResource::default(), false),
        }),
        ..Default::default()
    };
    let (beats, issues) = compile(input, &bindings);
    assert!(
        issues.is_empty() && beats.len() == 4 && beats[3].op == "native_ch22_reset_grid",
        "compiled reload={beats:?} issues={issues:?}"
    );
    for (index, source) in ["0x24a4b", "0x24a65", "0x24a7f"].iter().enumerate() {
        let want = &resources[*source];
        assert!(
            beats[index].resource_id == Some(want.resource_id)
                && beats[index].resource_archive == want.archive
                && beats[index].resource_owner == want.owner,
            "resource beat{index}={:?} want={want:?}",
            beats[index]
        );
    }
}

/// Go：`TestCompileDynamicPaletteLoopMaterializesDescendingRange`
#[test]
fn compile_dynamic_palette_loop_materializes_descending_range() {
    let (beats, issues) = compile(
        vec![
            unknown("0x11df2", vec![json!("ebx"), json!(255), json!(0)], src("0x258cd", "0x11df2")),
            HandlerBeat { op: "delay".into(), ms: ip(4), source: src("0x258d7", "0x375b2"), ..Default::default() },
        ],
        &none(),
    );
    assert!(issues.is_empty() && beats.len() == 126, "dynamic palette beats={} issues={issues:?}", beats.len());
    assert!(
        beats[0].op == "palette_update"
            && beats[0].palette_start == 0x3e
            && beats[1].op == "delay"
            && beats[1].ms == 4
            && beats[124].palette_start == 0
            && beats[125].op == "delay",
        "dynamic palette sequence head/tail={:?} ... {:?}",
        &beats[..2],
        &beats[124..]
    );
}

/// Go：`TestCompileDynamicPaletteLoopsPreserveCallerDirectionAndConsumeRawDelay`
#[test]
fn compile_dynamic_palette_loops_preserve_caller_direction_and_consume_raw_delay() {
    let tests = [
        ("ch22 even ascending", "0x24a24", "0x24a2e", 32usize, 0i64, 0x3ei64),
        ("ch28 ascending", "0x256eb", "0x256f5", 64, 0, 0x3f),
        ("ch28 descending", "0x25733", "0x2573d", 63, 0x3e, 0),
    ];
    for (name, palette_source, delay_source, want_count, want_first, want_last) in tests {
        let mut palette = native_call(
            "0x11df2",
            "native_palette_update",
            vec![json!("ebx"), json!(255), json!(0)],
            src(palette_source, "0x11df2"),
        );
        palette.native_evidence = vec!["evidence".into()];
        let (beats, issues) = compile(
            vec![
                palette,
                HandlerBeat {
                    op: "delay".into(),
                    ms: ip(4),
                    source: src(delay_source, "0x375b2"),
                    ..Default::default()
                },
            ],
            &none(),
        );
        assert!(issues.is_empty() && beats.len() == want_count * 2, "{name}: beats={} issues={issues:?}", beats.len());
        let n = beats.len();
        assert!(
            beats[0].palette_start == want_first
                && beats[n - 2].palette_start == want_last
                && beats[n - 1].source == delay_source,
            "{name}: palette endpoints/delay = first{} last{} tail={:?}",
            beats[0].palette_start,
            beats[n - 2].palette_start,
            beats[n - 1]
        );
    }

    let (_, issues) = compile(
        vec![
            unknown("0x11df2", vec![json!("ebx"), json!(255), json!(0)], src("0xdead", "0x11df2")),
            HandlerBeat { op: "delay".into(), ms: ip(4), source: src("0xbeef", "0x375b2"), ..Default::default() },
        ],
        &none(),
    );
    assert_eq!(issues.len(), 1, "unknown register-bound caller must fail closed: {issues:?}");
}

/// 已證實的 0x24618 參數組（多個測試共用）。
fn recovered_transition() -> HandlerIndexedTransition {
    HandlerIndexedTransition {
        tile_x: 9,
        tile_y: 8,
        radial_radius: 10,
        radial_radius_step: 8,
        start_y: 0,
        end_y: 0xc0,
        clip_width: 0x138,
        clip_height: 0xc0,
        frames: 9,
        frame_delay_ms: 5,
        tail_delay_ms: 500,
        palette_range_start: 0,
        palette_range_end: 255,
        palette_delta_start: 0,
        palette_delta_end: 62,
        palette_delta_step: 2,
        palette_delay_ms: 4,
        ..Default::default()
    }
}

/// Go：`TestCompileIndexedTransitionRequiresRecoveredBinding`
#[test]
fn compile_indexed_transition_requires_recovered_binding() {
    let transition = recovered_transition();
    let bindings =
        HandlerBindings { transition: cb!(move |_: HandlerBeat| (transition.clone(), true)), ..Default::default() };
    let (beats, issues) = compile(
        vec![unknown(
            "0x24618",
            vec![json!("global_x"), json!("global_y"), json!(10), json!(8)],
            HandlerSource::default(),
        )],
        &bindings,
    );
    assert!(
        issues.is_empty()
            && beats.len() == 1
            && beats[0].op == "indexed_transition"
            && beats[0].indexed_transition.is_some(),
        "indexed transition={beats:?} issues={issues:?}"
    );
}

/// Go：`TestCompileUnitPresentRejectsIncompleteSixFrameSchema`
#[test]
fn compile_unit_present_rejects_incomplete_six_frame_schema() {
    let input = HandlerBeat {
        op: "unit_present".into(),
        unit_present: Some(HandlerUnitPresent { slot: 2, x: 4, y: 5, frames: 6, frame_delay_ms: 10, tail_ticks: 2 }),
        ..Default::default()
    };
    let (beats, issues) = compile(vec![input], &ctx(4));
    assert!(
        beats.is_empty() && issues.len() == 1 && issues[0].op == "unit_present",
        "unit_present={beats:?} issues={issues:?}"
    );
}

/// Go：`TestCompileHandlerJoinRejectsScenePortrait`
#[test]
fn compile_handler_join_rejects_scene_portrait() {
    let (beats, issues) = compile(
        vec![HandlerBeat { op: "join".into(), char_id: ip(75), source: addr("0x123"), ..Default::default() }],
        &none(),
    );
    assert!(
        beats.is_empty()
            && issues.len() == 1
            && issues[0].reason == "join char_id 75 is outside the original 0..31 player roster",
        "scene portrait must not compile as join: beats={beats:?} issues={issues:?}"
    );
}

/// Go：`TestCompileHandlerPaletteFadePreservesNative65StepSchedule`
#[test]
fn compile_handler_palette_fade_preserves_native65_step_schedule() {
    let (beats, issues) = compile(vec![op_at("palette_fade", src("0x236f6", "0x1f525"))], &none());
    assert!(
        issues.is_empty()
            && beats.len() == 1
            && beats[0].op == "native_palette_fade_in"
            && beats[0].native_palette_fade_in.is_some_and(|f| f.start == 64 && f.end == 0 && f.delay_ms == 2)
            && beats[0].source == "0x236f6",
        "palette fade lowering = {beats:?} issues={issues:?}"
    );
}

/// Go：`TestCompileHandlerChapterZeroPaletteApproximationIsSourceBound`
#[test]
fn compile_handler_chapter_zero_palette_approximation_is_source_bound() {
    let (beats, issues) = compile(vec![op_at("palette_fade", src("0x3241f", "0x1f525"))], &none());
    assert!(
        issues.is_empty() && beats.len() == 1 && beats[0].op == "fade" && !beats[0].out && beats[0].source == "0x3241f",
        "chapter-zero bounded approximation = {beats:?} issues={issues:?}"
    );
}

/// Go：`TestCompilePostBattlePrimitives`
#[test]
fn compile_post_battle_primitives() {
    let (beats, issues) = compile(
        vec![
            op_at("sync_party", src("0x22f27", "0x11506")),
            HandlerBeat { op: "set_chapter".into(), chapter: ip(1), source: addr("0x22f2c"), ..Default::default() },
        ],
        &none(),
    );
    assert!(issues.is_empty() && beats.len() == 2, "post primitives beats={beats:?} issues={issues:?}");
    assert!(beats[0].op == "sync_party" && beats[0].source == "0x22f27", "sync_party lowering = {:?}", beats[0]);
    assert!(beats[1].op == "set_chapter" && beats[1].chapter == Some(1), "set_chapter lowering = {:?}", beats[1]);
}

/// Go：`TestCompileSetChapterRejectsMissingImmediate`
#[test]
fn compile_set_chapter_rejects_missing_immediate() {
    let (beats, issues) = compile(vec![op("set_chapter")], &none());
    assert!(
        beats.is_empty()
            && issues.len() == 1
            && issues[0].reason == "set_chapter requires a non-negative immediate chapter",
        "missing set_chapter immediate beats={beats:?} issues={issues:?}"
    );
}

/// Go：`TestCompileGrantItemPrimitive`
#[test]
fn compile_grant_item_primitive() {
    let (beats, issues) = compile(
        vec![HandlerBeat {
            op: "grant_item".into(),
            item_id: ip(0xc6),
            source: src("0x22f9f", "0x1c220"),
            ..Default::default()
        }],
        &none(),
    );
    assert!(
        issues.is_empty() && beats.len() == 1 && beats[0].op == "grant_item" && beats[0].item_id == Some(0xc6),
        "grant_item lowering = {beats:?} issues={issues:?}"
    );
    let (beats, issues) =
        compile(vec![HandlerBeat { op: "grant_item".into(), item_id: ip(0x100), ..Default::default() }], &none());
    assert!(
        beats.is_empty() && issues.len() == 1 && issues[0].reason == "grant_item requires an unsigned byte item_id",
        "invalid grant_item = {beats:?} issues={issues:?}"
    );
}

/// Go：`TestCompileAnyUnitInactiveBranchRequiresBothArms`
#[test]
fn compile_any_unit_inactive_branch_requires_both_arms() {
    let input = vec![HandlerBeat {
        op: "if".into(),
        source: src("0x22f71", "0x22fa9"),
        condition: Some(HandlerCondition {
            op: "any_unit_inactive".into(),
            unit_slots: vec![5, 6, 7, 8, 9, 10],
            ..Default::default()
        }),
        then: vec![HandlerBeat {
            op: "dialog".into(),
            text_index: json!(7),
            source: addr("0x22fc8"),
            ..Default::default()
        }],
        r#else: vec![
            HandlerBeat { op: "dialog".into(), text_index: json!(6), source: addr("0x22f92"), ..Default::default() },
            HandlerBeat { op: "grant_item".into(), item_id: ip(0xc6), source: addr("0x22f9f"), ..Default::default() },
        ],
        ..Default::default()
    }];
    let bindings = HandlerBindings {
        dialog: cb!(|input: HandlerBeat| match input.source.addr.as_str() {
            "0x22fc8" => (HandlerDialog { line: 7, ..Default::default() }, true),
            "0x22f92" => (HandlerDialog { line: 6, ..Default::default() }, true),
            _ => (HandlerDialog::default(), false),
        }),
        ..Default::default()
    };
    let (beats, issues) = compile(input.clone(), &bindings);
    assert!(
        issues.is_empty() && beats.len() == 1 && beats[0].op == "if",
        "structured if lowering = {beats:?} issues={issues:?}"
    );
    let branch = &beats[0];
    assert!(
        branch.condition.as_ref().is_some_and(|c| c.op == "any_unit_inactive" && c.unit_slots.len() == 6),
        "condition = {:?}",
        branch.condition
    );
    assert!(
        branch.then.len() == 1 && branch.then[0].op == "dialog" && branch.then[0].line == 7,
        "then arm = {:?}",
        branch.then
    );
    assert!(
        branch.r#else.len() == 2 && branch.r#else[0].line == 6 && branch.r#else[1].item_id == Some(0xc6),
        "else arm = {:?}",
        branch.r#else
    );

    let mut missing = bindings.clone();
    missing.dialog =
        cb!(|input: HandlerBeat| (HandlerDialog { line: 7, ..Default::default() }, input.source.addr == "0x22fc8"));
    let (beats, issues) = compile(input, &missing);
    assert!(
        beats.is_empty()
            && issues.len() == 1
            && issues[0].reason == "if else: no remake line mapping for original FDTXT lookup",
        "unresolved arm must fail closed: beats={beats:?} issues={issues:?}"
    );
}

/// Go：`TestCompileAnyUnitInactiveRejectsInvalidCondition`
#[test]
fn compile_any_unit_inactive_rejects_invalid_condition() {
    let tests = [
        HandlerCondition { op: "unknown".into(), unit_slots: vec![5], ..Default::default() },
        HandlerCondition { op: "any_unit_inactive".into(), ..Default::default() },
        HandlerCondition { op: "any_unit_inactive".into(), unit_slots: vec![5, -1], ..Default::default() },
        HandlerCondition { op: "any_unit_inactive".into(), unit_slots: vec![5, 5], ..Default::default() },
    ];
    for condition in tests {
        let (beats, issues) = compile(
            vec![HandlerBeat { op: "if".into(), condition: Some(condition.clone()), ..Default::default() }],
            &none(),
        );
        assert!(
            beats.is_empty() && issues.len() == 1,
            "invalid condition {condition:?}: beats={beats:?} issues={issues:?}"
        );
    }
    let valid = HandlerCondition { op: "any_unit_inactive".into(), unit_slots: vec![5], ..Default::default() };
    let (beats, issues) = compile(
        vec![HandlerBeat {
            op: "if".into(),
            condition: Some(valid),
            then: vec![HandlerBeat { op: "act".into(), acting_id: ip(1), ..Default::default() }],
            ..Default::default()
        }],
        &none(),
    );
    assert!(
        beats.is_empty()
            && issues.len() == 1
            && issues[0].reason
                == "if arms cannot use active-slot operations before branch compiler context is modeled",
        "active-slot branch must fail closed: beats={beats:?} issues={issues:?}"
    );
}

/// Go：`TestCompileNativeInactiveCountGreaterThanPreservesRawThreshold`
#[test]
fn compile_native_inactive_count_greater_than_preserves_raw_threshold() {
    let condition = HandlerCondition {
        op: "native_inactive_count_gt".into(),
        unit_slots: vec![66, 67, 68, 69, 70, 71, 72, 73],
        threshold: ip(4),
        ..Default::default()
    };
    let (beats, issues) =
        compile(vec![HandlerBeat { op: "if".into(), condition: Some(condition), ..Default::default() }], &ctx(74));
    assert!(
        issues.is_empty()
            && beats.len() == 1
            && beats[0]
                .condition
                .as_ref()
                .is_some_and(|c| c.op == "native_inactive_count_gt" && c.threshold == Some(4)),
        "raw inactive-count lowering={beats:?} issues={issues:?}"
    );
    let (_, issues) = compile(
        vec![HandlerBeat {
            op: "if".into(),
            condition: Some(HandlerCondition {
                op: "native_inactive_count_gt".into(),
                unit_slots: vec![66],
                threshold: ip(-1),
                ..Default::default()
            }),
            ..Default::default()
        }],
        &none(),
    );
    assert_eq!(issues.len(), 1, "negative threshold must fail closed: {issues:?}");
}

/// Go：`TestCompileNativeRawCh15ComparisonsPreserveProvenance`
#[test]
fn compile_native_raw_ch15_comparisons_preserve_provenance() {
    let word = |offset: i64| HandlerCondition {
        op: "native_record_word_gte".into(),
        unit_slot: ip(0),
        native_record_word_offset: ip(offset),
        native_record_word_value: ip(0x140),
        ..Default::default()
    };
    let input = vec![
        HandlerBeat {
            op: "if".into(),
            condition: Some(HandlerCondition {
                op: "native_round_gt".into(),
                native_round: ip(18),
                ..Default::default()
            }),
            ..Default::default()
        },
        HandlerBeat { op: "if".into(), condition: Some(word(0x42)), ..Default::default() },
    ];
    let (beats, issues) = compile(input, &none());
    assert!(
        issues.is_empty()
            && beats.len() == 2
            && beats[0].condition.as_ref().unwrap().native_round == Some(18)
            && beats[1].condition.as_ref().unwrap().native_record_word_offset == Some(0x42),
        "raw ch15 comparisons were not preserved: beats={beats:?} issues={issues:?}"
    );
    let (_, issues) =
        compile(vec![HandlerBeat { op: "if".into(), condition: Some(word(0x40)), ..Default::default() }], &none());
    assert!(!issues.is_empty(), "non-proven raw record offset did not fail closed");
}

/// Go：`TestCompileNativeAnyOfRestrictsChildrenToProvenRawOps`
#[test]
fn compile_native_any_of_restricts_children_to_proven_raw_ops() {
    let condition = HandlerCondition {
        op: "native_any_of".into(),
        any: vec![
            HandlerCondition { op: "native_round_gt".into(), native_round: ip(18), ..Default::default() },
            HandlerCondition {
                op: "native_inactive_count_gt".into(),
                unit_slots: vec![66, 67],
                threshold: ip(4),
                ..Default::default()
            },
        ],
        ..Default::default()
    };
    let (beats, issues) =
        compile(vec![HandlerBeat { op: "if".into(), condition: Some(condition), ..Default::default() }], &none());
    assert!(
        issues.is_empty() && beats.len() == 1 && beats[0].condition.as_ref().unwrap().any.len() == 2,
        "native_any_of was not preserved: beats={beats:?} issues={issues:?}"
    );
    let bad = HandlerCondition {
        op: "native_any_of".into(),
        any: vec![HandlerCondition { op: "roster_has".into(), char_id: ip(1), ..Default::default() }],
        ..Default::default()
    };
    let (_, issues) =
        compile(vec![HandlerBeat { op: "if".into(), condition: Some(bad), ..Default::default() }], &none());
    assert!(!issues.is_empty(), "unsupported native_any_of child did not fail closed");
}

/// Go：`TestCompileChapter1PostResolvesBothDialogueBranchArms`
#[test]
fn compile_chapter1_post_resolves_both_dialogue_branch_arms() {
    let Some(path) = asset("cutscenes/bindings/generated/ch01_post.json") else {
        return;
    };
    let (beats, issues) = compile_handler_binding(&path).unwrap();
    assert_eq!(
        issues.len(),
        6,
        "ch01 post issues = {}, want 6 remaining pan/spawn/act bindings: {issues:?} (beats={beats:?})",
        issues.len()
    );
    for issue in &issues {
        assert!(
            !(issue.source.addr == "0x22fc8" || issue.source.addr == "0x22f92" || issue.op == "dialog"),
            "resolved ch01 post dialog still reported as issue: {issue:?}"
        );
    }
    let mut found_branch = false;
    for beat in &beats {
        if beat.op != "if" {
            continue;
        }
        found_branch = beat.then.len() == 1
            && beat.r#else.len() == 2
            && beat.then[0].op == "dialog"
            && beat.r#else[0].op == "dialog"
            && beat.r#else[1].op == "grant_item";
    }
    assert!(found_branch, "compiled ch01 post lost resolved structured branch: {beats:?}");
}

/// Go：`TestCompileHandlerScriptDoesNotGuessMissingMappings`
#[test]
fn compile_handler_script_does_not_guess_missing_mappings() {
    let (beats, issues) = compile(
        vec![
            HandlerBeat { op: "loadch".into(), chapter: ip(5), ..Default::default() },
            HandlerBeat { op: "pan".into(), grid_x: ip(2), grid_y: ip(4), ..Default::default() },
            HandlerBeat { op: "dialog".into(), text_index: json!(3.0), ..Default::default() },
            HandlerBeat { op: "act".into(), acting_id: ip(1), ..Default::default() },
        ],
        &none(),
    );
    assert!(
        beats.is_empty() && issues.len() == 4,
        "beats={beats:?} issues={issues:?}, want no guessed beats and four issues"
    );
}

/// Go：`TestCompileHandlerSpawnRequiresLoadedRoster`
#[test]
fn compile_handler_spawn_requires_loaded_roster() {
    let (beats, issues) = compile(
        vec![HandlerBeat {
            op: "spawn".into(),
            group: ip(1),
            raw_placement_gate: ip(0),
            source: addr("0x100"),
            ..Default::default()
        }],
        &none(),
    );
    assert!(
        beats.is_empty()
            && issues.len() == 1
            && issues[0].op == "spawn"
            && issues[0].reason == "spawn requires a preceding complete loadch roster",
        "spawn without loadch must fail closed: beats={beats:?} issues={issues:?}"
    );
}

/// Go：`TestCompileHandlerSpawnRequiresExplicitRawPlacementGate`
#[test]
fn compile_handler_spawn_requires_explicit_raw_placement_gate() {
    let bindings = HandlerBindings {
        load_ch: cb!(|_: HandlerBeat| {
            (
                load_ch_state(1, "assets/maps/map1", "assets/maps/map1/map1_units.json", 20, "assets/story/ch01.json"),
                true,
            )
        }),
        ..Default::default()
    };
    let (beats, issues) = compile(
        vec![
            HandlerBeat { op: "loadch".into(), chapter: ip(1), source: addr("0x90"), ..Default::default() },
            HandlerBeat { op: "spawn".into(), group: ip(2), source: addr("0x100"), ..Default::default() },
        ],
        &bindings,
    );
    assert!(
        beats.len() == 1
            && issues.len() == 1
            && issues[0].reason == "spawn requires an explicit raw_placement_gate byte",
        "missing raw gate must fail closed: beats={beats:?} issues={issues:?}"
    );
}

/// Go：`TestHandlerBindingUsesSourceAddress`
#[test]
fn handler_binding_uses_source_address() {
    let mut overrides = BTreeMap::new();
    overrides.insert(
        "0x32339".to_string(),
        HandlerBindingOverride {
            pan: Some(HandlerPoint { x: 72, y: 816, frames: 60, ..Default::default() }),
            ..Default::default()
        },
    );
    overrides.insert(
        "0x32382".to_string(),
        HandlerBindingOverride {
            dialog: Some(HandlerDialog {
                lines: vec![HandlerDialogLine { line: 0, ..Default::default() }],
                ..Default::default()
            }),
            ..Default::default()
        },
    );
    let binding = HandlerBinding {
        schema_version: 1,
        handler_script: "handlers/ch00_pre.json".into(),
        overrides,
        ..Default::default()
    };
    let input = vec![
        HandlerBeat { op: "pan".into(), grid_x: ip(3), grid_y: ip(34), source: addr("0x32339"), ..Default::default() },
        // Same raw coordinates but a different original call site must not use
        // the earlier scene's camera interpretation.
        HandlerBeat { op: "pan".into(), grid_x: ip(3), grid_y: ip(34), source: addr("0x99999"), ..Default::default() },
        HandlerBeat { op: "dialog".into(), text_index: json!(0.0), source: addr("0x32382"), ..Default::default() },
    ];
    let (beats, issues) = compile(input, &binding.compiler_bindings());
    assert!(beats.len() == 2 && beats[0].x == 72 && beats[1].line == 0, "bound beats = {beats:?}");
    assert!(issues.len() == 1 && issues[0].source.addr == "0x99999", "issues = {issues:?}, want only unmatched source");
}

/// Go：`TestCompileHandlerDialogExpandsOriginalTextGroup`
#[test]
fn compile_handler_dialog_expands_original_text_group() {
    let upper = true;
    let bindings = HandlerBindings {
        dialog: cb!(move |_: HandlerBeat| {
            (
                HandlerDialog {
                    lines: vec![
                        HandlerDialogLine { line: 3, upper: Some(upper), ..Default::default() },
                        HandlerDialogLine { line: 4, ..Default::default() },
                        HandlerDialogLine { line: 5, count: 2, upper: Some(upper), ..Default::default() },
                    ],
                    ..Default::default()
                },
                true,
            )
        }),
        ..Default::default()
    };
    let (beats, issues) = compile(vec![op_at("dialog", addr("0x40000"))], &bindings);
    assert!(issues.is_empty() && beats.len() == 3, "beats={beats:?} issues={issues:?}");
    // Go 比的是指標同一性（`beats[0].Upper != &upper`）；Rust 改比值。
    assert!(
        beats[0].line == 3 && beats[0].upper == Some(upper) && beats[1].line == 4 && beats[2].count == 2,
        "expanded dialog beats = {beats:?}"
    );
}

/// Go：`TestCompileHandlerActingPreservesOriginalRosterSlot`
#[test]
fn compile_handler_acting_preserves_original_roster_slot() {
    let bindings = HandlerBindings {
        acting: cb!(|_: HandlerBeat| {
            (
                vec![ActingFrame {
                    beats: 8,
                    special: true,
                    units: vec![ActingUnit { slot: Some(17), pose: 2, ..Default::default() }],
                }],
                true,
            )
        }),
        ..Default::default()
    };
    let (beats, issues) = compile(
        vec![HandlerBeat { op: "act".into(), acting_id: ip(0x66), source: addr("0x32466"), ..Default::default() }],
        &bindings,
    );
    assert!(
        issues.is_empty() && beats.len() == 1 && beats[0].acting.len() == 1,
        "slot acting compilation beats={beats:?} issues={issues:?}"
    );
    let unit = beats[0].acting[0].units[0];
    assert!(unit.slot == Some(17) && unit.fig == 0 && unit.pose == 2, "acting target lost original slot: {unit:?}");
}

/// Go：`TestCompileCompleteChapter0Binding`
#[test]
fn compile_complete_chapter0_binding() {
    let (Some(binding_path), Some(script_path)) =
        (asset("cutscenes/bindings/ch00_pre.json"), asset("cutscenes/handlers/ch00_pre.json"))
    else {
        return;
    };
    let binding = load_handler_binding(&binding_path).unwrap();
    let overrides = binding.dialogue_overrides.clone().unwrap_or_default();
    let d = overrides.get("0x32382#0").cloned().unwrap_or_default();
    assert!(
        d.lines.len() == 6 && d.scene_index == Some(0),
        "throne FDTXT #0 binding = {d:?}, want six contextual lines"
    );
    let d = overrides.get("0x3244d#2").cloned().unwrap_or_default();
    assert!(
        d.lines.len() == 5 && d.scene_index == Some(1),
        "grass FDTXT #2 binding = {d:?}, want five contextual lines"
    );
    let script = load_handler_script(&script_path).unwrap();
    let (beats, issues) = compile_handler_script(Some(&script), &binding.compiler_bindings());
    assert!(issues.is_empty(), "ch00 must compile without unresolved handler beats: {issues:?}");
    let (mut pan, mut dialog, mut slot_act, mut direct_slot_act, mut act99, mut act100) =
        (false, false, false, false, false, false);
    // 一格六幀：0x13185 的計數器由 1 起、`cmp [esp], 7 ; jge` 離開、遞增在
    // 迴圈尾端，主體跑 1..6（見 handler_compile.go 的 nativeGridStepFrames）。
    let mut scroll_steps: BTreeMap<&str, (i64, i64, i64)> =
        [("0x32351", (2, 15, 90)), ("0x3239a", (2, 13, 78))].into_iter().collect();
    let mut focus_slots: BTreeMap<&str, i64> = [("0x32961", 0)].into_iter().collect();
    let mut map31_spawns: BTreeMap<&str, i64> = [("0x32555", 1), ("0x32610", 3), ("0x3269c", 5)].into_iter().collect();
    let mut map32_acts: BTreeMap<&str, i64> = [
        ("0x32343", 99),
        ("0x323f5", 100),
        ("0x32426", 101),
        ("0x32461", 102),
        ("0x3249c", 103),
        ("0x324d7", 104),
        ("0x3251c", 105),
    ]
    .into_iter()
    .collect();
    let mut map31_acts: BTreeMap<&str, i64> = [
        ("0x3255f", 90),
        ("0x3259a", 91),
        ("0x325d5", 92),
        ("0x32657", 93),
        ("0x326d7", 94),
        ("0x32712", 95),
        ("0x3274d", 96),
        ("0x32788", 97),
        ("0x327d9", 98),
    ]
    .into_iter()
    .collect();
    let mut map0_acts: BTreeMap<&str, i64> =
        [("0x3283a", 0), ("0x328a5", 1), ("0x328c5", 2), ("0x3290d", 5)].into_iter().collect();
    let mut spawn_intros: BTreeMap<&str, i64> = [("0x3289b", 1), ("0x328bb", 2)].into_iter().collect();
    let mut activate_slots: BTreeMap<&str, i64> = [("0x32692", 2), ("0x32917", 9)].into_iter().collect();
    let mut pan_targets: BTreeMap<&str, [i64; 2]> = [
        ("0x3254b", [120, 1008]),
        ("0x3261c", [96, 984]),
        ("0x32830", [96, 288]),
        ("0x32891", [0, 0]),
        ("0x328b1", [0, 360]),
    ]
    .into_iter()
    .collect();
    let (mut reset_pose, mut redraw) = (false, false);
    let mut loadchs: BTreeMap<i64, (&str, &str, i64)> = [
        (32, ("assets/maps/map32", "assets/cutscenes/rosters/map32_runtime.json", 21)),
        (31, ("assets/maps/map31", "assets/maps/map31/map31_units.json", 30)),
        (0, ("assets/maps/map0", "assets/maps/map0/map0_units.json", 30)),
    ]
    .into_iter()
    .collect();
    let mut dialog_counts: BTreeMap<String, i64> = BTreeMap::new();
    let mut native_dialogues = 0;
    for beat in &beats {
        let source = beat.source.as_str();
        pan = pan || beat.op == "pan" && beat.x == 72 && beat.y == 816;
        if let Some(want) = pan_targets.get(source).copied()
            && beat.op == "pan"
            && beat.x == want[0]
            && beat.y == want[1]
        {
            pan_targets.remove(source);
        }
        dialog = dialog || beat.op == "dialog" && beat.line == 0;
        if beat.op == "dialog" {
            *dialog_counts.entry(beat.source.clone()).or_default() += 1;
            assert!(
                beat.native_dialogue.is_some(),
                "compiled ch00 dialog {} line{} lacks raw native layout",
                beat.source,
                beat.line
            );
            native_dialogues += 1;
        }
        if beat.op == "act" && source == "0x32461" && beat.acting.len() == 3 {
            let u = beat.acting[0].units[0];
            slot_act = u.slot == Some(4) && u.fig == 0 && !beat.acting[0].special;
        }
        if beat.op == "act" && source == "0x324d7" && beat.acting.len() == 1 {
            let u = beat.acting[0].units[0];
            direct_slot_act = u.slot == Some(3) && beat.acting[0].special && beat.acting[0].beats == 2;
        }
        if beat.op == "act" && source == "0x32343" && beat.acting.len() == 1 {
            let u = beat.acting[0].units[0];
            act99 = !beat.acting[0].special && beat.acting[0].beats == 6 && u.slot == Some(2) && u.pose == 2;
        }
        if beat.op == "act" && source == "0x323f5" && beat.acting.len() == 1 {
            let u = beat.acting[0].units[0];
            act100 = !beat.acting[0].special && beat.acting[0].beats == 10 && u.slot == Some(2) && u.pose == 0;
        }
        if let Some((slot, steps, frames)) = scroll_steps.get(source).copied()
            && beat.op == "scroll_step"
        {
            assert!(
                beat.slot == Some(slot) && beat.steps == steps && beat.frames == frames && beat.follow,
                "scroll_step {source} = {beat:?}, want slot={slot} steps={steps} frames={frames} follow"
            );
            scroll_steps.remove(source);
        }
        if let Some(want) = focus_slots.get(source).copied()
            && beat.op == "focus_unit"
        {
            assert!(beat.slot == Some(want), "focus_unit {source} = {beat:?}, want slot={want}");
            focus_slots.remove(source);
        }
        if let Some(id) = map32_acts.get(source).copied()
            && beat.op == "act"
            && !beat.acting.is_empty()
        {
            map32_acts.remove(source);
            assert!(
                beat.acting[0].units[0].slot.is_some(),
                "map32 ACT({id}) did not preserve source roster slot: {beat:?}"
            );
        }
        if let Some(id) = map31_acts.get(source).copied()
            && beat.op == "act"
            && !beat.acting.is_empty()
        {
            map31_acts.remove(source);
            assert!(
                !beat.acting[0].units.is_empty() && beat.acting[0].units[0].slot.is_some(),
                "map31 ACT({id}) did not preserve source roster slot: {beat:?}"
            );
        }
        if let Some(id) = map0_acts.get(source).copied()
            && beat.op == "act"
            && !beat.acting.is_empty()
        {
            map0_acts.remove(source);
            assert!(beat.acting[0].units[0].slot.is_some(), "map0 ACT({id}) did not preserve runtime slot: {beat:?}");
        }
        if let Some(group) = map31_spawns.get(source).copied()
            && beat.op == "spawn"
            && beat.group == group
        {
            map31_spawns.remove(source);
        }
        if let Some(group) = spawn_intros.get(source).copied()
            && beat.op == "spawn_intro"
            && beat.group == group
            && beat.frames == 0
        {
            spawn_intros.remove(source);
        }
        if let Some(slot) = activate_slots.get(source).copied()
            && beat.op == "deactivate_unit"
            && beat.slot == Some(slot)
        {
            activate_slots.remove(source);
        }
        reset_pose = reset_pose || source == "0x3295a" && beat.op == "reset_pose" && beat.ms == 20;
        redraw = redraw || source == "0x32921" && beat.op == "redraw" && beat.frames == 1;
        if beat.op == "loadch"
            && let Some(state) = &beat.load_ch
        {
            assert!(
                !(state.chapter == 0
                    && (state.party_scenario != "assets/scenarios/ch01.json"
                        || state.party_order.len() != 4
                        || state.party_order[1] != 9)),
                "map0 LOADCH lacks persistent party deployment/order: {state:?}"
            );
            if let Some((map, roster, slots)) = loadchs.get(&state.chapter).copied()
                && state.map == map
                && state.roster == roster
                && state.slot_count == slots
            {
                loadchs.remove(&state.chapter);
            }
        }
    }
    assert!(
        pan && dialog
            && slot_act
            && direct_slot_act
            && act99
            && act100
            && reset_pose
            && redraw
            && scroll_steps.is_empty()
            && focus_slots.is_empty()
            && pan_targets.is_empty()
            && map32_acts.is_empty()
            && map31_acts.is_empty()
            && map0_acts.is_empty()
            && map31_spawns.is_empty()
            && spawn_intros.is_empty()
            && activate_slots.is_empty()
            && loadchs.is_empty(),
        "loaded binding did not lower its proven pan/dialog/slot-acting overrides: {beats:?}"
    );
    for (source, want) in [
        ("0x32382", 6),
        ("0x323cb", 13),
        ("0x3244d", 5),
        ("0x32488", 4),
        ("0x324c3", 1),
        ("0x324fe", 12),
        ("0x32586", 5),
        ("0x325c1", 1),
        ("0x325fc", 1),
        ("0x32643", 2),
        ("0x3267e", 2),
        ("0x326c3", 3),
        ("0x326fe", 6),
        ("0x32739", 2),
        ("0x32774", 8),
        ("0x327af", 7),
        ("0x3286e", 5),
        ("0x328ec", 2),
        ("0x32952", 12),
    ] {
        let got = dialog_counts.get(source).copied().unwrap_or(0);
        assert_eq!(got, want, "compiled dialog {source} emitted {got} editable lines, want {want}");
    }
    assert_eq!(native_dialogues, 97, "compiled ch00 native dialogues={native_dialogues}, want 97");
}

// ---- 讀真實綁定檔的共用輔助 ----

/// Go `CompileHandlerBinding("../../assets/cutscenes/bindings/<name>")`，並要求沒有錯誤與問題。
fn bind_ok(name: &str) -> Option<Vec<Beat>> {
    let path = asset(&format!("cutscenes/bindings/{name}"))?;
    match compile_handler_binding(&path) {
        Ok((beats, issues)) => {
            assert!(issues.is_empty(), "{name} issues={issues:?}");
            Some(beats)
        }
        Err(err) => panic!("{name} err={err}"),
    }
}

/// Go 測試常見的 `seen[beat.Source] = beat`（後者覆蓋前者）與 dialog 收集。
fn collect(beats: &[Beat]) -> (BTreeMap<String, Beat>, Vec<Beat>) {
    let mut seen = BTreeMap::new();
    let mut dialogs = Vec::new();
    for beat in beats {
        seen.insert(beat.source.clone(), beat.clone());
        if beat.op == "dialog" {
            dialogs.push(beat.clone());
        }
    }
    (seen, dialogs)
}

/// Go `seen[source]`：缺鍵時是零值。
fn at(seen: &BTreeMap<String, Beat>, source: &str) -> Beat {
    seen.get(source).cloned().unwrap_or_default()
}

/// LOADCH 斷言：chapter／map／script／party_scenario，slot_count 有給才比。
fn check_load(name: &str, load: &Beat, chapter: i64, map: &str, slots: Option<i64>, script: &str, party: &str) {
    let ok = load.load_ch.as_ref().is_some_and(|s| {
        s.chapter == chapter
            && s.map == map
            && slots.is_none_or(|n| s.slot_count == n)
            && s.script == script
            && s.party_scenario == party
    });
    assert!(ok, "{name} LOADCH = {:?}", load.load_ch);
}

fn check_pan(name: &str, pan: &Beat, x: i64, y: i64) {
    assert!(pan.x == x && pan.y == y && pan.tile_step, "{name} PAN = {pan:?}");
}

fn check_act_units(name: &str, act: &Beat) {
    assert!(!act.acting.is_empty() && !act.acting[0].units.is_empty(), "{name} acting = {:?}", act.acting);
}

/// Go：`TestCompileCompleteChapter0PostBinding`
#[test]
fn compile_complete_chapter0_post_binding() {
    let Some(beats) = bind_ok("ch00_post.json") else {
        return;
    };
    assert!(
        beats.len() == 16
            && beats[0].op == "runtime_context"
            && beats[0]
                .runtime_context
                .as_ref()
                .is_some_and(|c| c.story_viewport && c.slot_counts == vec![12, 14, 18, 23, 27]),
        "ch00_post compiled beats/context = {}/{:?}",
        beats.len(),
        beats[0]
    );
    for i in 0..13 {
        let beat = &beats[i + 1];
        assert!(
            beat.op == "dialog" && beat.scene_index == Some(7) && beat.native_dialogue.is_some(),
            "post dialog beat {i} = {beat:?}"
        );
    }
    assert!(
        beats[14].op == "sync_party" && beats[15].op == "set_chapter" && beats[15].chapter == Some(1),
        "post tail = {:?}",
        &beats[14..]
    );
}

/// Go：`TestCompileCompleteChapter1PostBinding`
#[test]
fn compile_complete_chapter1_post_binding() {
    let Some(beats) = bind_ok("ch01_post.json") else {
        return;
    };
    assert!(
        beats.len() == 39 && beats[0].op == "runtime_context" && beats[0].runtime_context.is_some(),
        "ch01_post compiled beats/context = {}/{:?}",
        beats.len(),
        beats[0]
    );
    let context = beats[0].runtime_context.as_ref().unwrap();
    assert!(
        context.slot_count == 27 && context.spawn_groups.get(&4).copied().unwrap_or(0) == 1 && context.story_viewport,
        "ch01_post runtime context = {context:?}"
    );
    let mut seen = BTreeSet::new();
    for beat in &beats {
        if !beat.source.is_empty() {
            seen.insert(beat.source.clone());
        }
        if beat.source == "0x22fd4" {
            assert!(beat.op == "pan" && beat.x == 336 && beat.y == 48, "first map1 pan = {beat:?}");
        }
        if beat.source == "0x23084" {
            assert!(beat.op == "pan" && beat.x == 336 && beat.y == 24, "second map1 pan = {beat:?}");
        }
    }
    for source in ["0x22fd4", "0x22fde", "0x22ff2", "0x2303a", "0x23084", "0x2309b"] {
        assert!(seen.contains(source), "ch01_post missing lowered source {source}");
    }
}

/// Go：`TestCompileRuntimeContextSpawnExpandsActingSlotFrontier`
#[test]
fn compile_runtime_context_spawn_expands_acting_slot_frontier() {
    let input = vec![
        HandlerBeat { op: "spawn".into(), group: ip(4), raw_placement_gate: ip(0), ..Default::default() },
        HandlerBeat { op: "act".into(), acting_id: ip(14), ..Default::default() },
    ];
    let mut bindings = HandlerBindings {
        runtime_context: Some(HandlerRuntimeContext {
            slot_count: 27,
            spawn_groups: [(4, 1)].into_iter().collect(),
            ..Default::default()
        }),
        acting: cb!(|_: HandlerBeat| {
            (
                vec![ActingFrame {
                    beats: 1,
                    units: vec![ActingUnit { slot: Some(27), ..Default::default() }],
                    ..Default::default()
                }],
                true,
            )
        }),
        ..Default::default()
    };
    let (beats, issues) = compile(input.clone(), &bindings);
    assert!(issues.is_empty() && beats.len() == 2, "spawn frontier beats={beats:?} issues={issues:?}");
    bindings.runtime_context.as_mut().unwrap().spawn_groups = BTreeMap::new();
    let (_, issues) = compile(input, &bindings);
    assert!(!issues.is_empty() && issues[0].op == "spawn", "missing group cardinality did not fail closed: {issues:?}");
}

/// Go：`TestCompileCompleteChapter2PostBindingPreservesTinoBranches`
#[test]
fn compile_complete_chapter2_post_binding_preserves_tino_branches() {
    let Some(beats) = bind_ok("ch02_post.json") else {
        return;
    };
    assert!(
        beats.len() == 4
            && beats[0].op == "runtime_context"
            && beats[1].op == "sync_party"
            && beats[2].op == "if"
            && beats[3].op == "set_chapter",
        "ch02_post top-level beats = {beats:?}"
    );
    let context = &beats[0].runtime_context;
    assert!(
        context.as_ref().is_some_and(|c| c.slot_counts == vec![15, 27] && c.story_viewport),
        "ch02_post runtime frontiers = {context:?}"
    );
    let branch = &beats[2];
    assert!(
        branch.condition.as_ref().is_some_and(|c| c.op == "any_unit_inactive" && c.unit_slots == vec![6]),
        "ch02_post Tino condition = {:?}",
        branch.condition
    );
    assert_eq!(
        branch.then.len(),
        5,
        "inactive #6 arm beats={}, want five mourning lines: {:?}",
        branch.then.len(),
        branch.then
    );
    for beat in &branch.then {
        assert!(beat.op == "dialog" && beat.scene_index == Some(1), "inactive arm line = {beat:?}");
    }
    let e = &branch.r#else;
    assert!(
        e.len() == 15
            && e[0].op == "layout_units"
            && e[1].op == "redraw"
            && e[2].op == "fade"
            && e[3].op == "delay"
            && e[14].op == "join"
            && e[14].char_id == 2,
        "active layout/dialog/JOIN arm = {e:?}"
    );
    let layout = &e[0].layout;
    assert!(
        layout.as_ref().is_some_and(|l| l.units.len() == 7
            && l.cam_x == 48
            && l.cam_y == 0
            && l.units[6] == HandlerUnitLayout { slot: 6, x: 8, y: 1, pose: 0 }),
        "native 0x233c6 layout = {layout:?}"
    );
    assert!(beats[3].chapter == Some(3) && beats[3].source == "0x2328a", "shared chapter tail = {:?}", beats[3]);
}

/// Go：`TestRuntimeContextAlternativeSlotCounts`
#[test]
fn runtime_context_alternative_slot_counts() {
    let context = HandlerRuntimeContext { slot_counts: vec![15, 27], ..Default::default() };
    assert!(
        context.minimum_slot_count() == 15
            && context.maximum_slot_count() == 27
            && context.accepts_slot_count(15)
            && context.accepts_slot_count(27)
            && !context.accepts_slot_count(16),
        "alternative runtime context = {context:?}"
    );
}

/// Go：`TestCompileCompleteChapter1PreUsesChapter2ContextAndSharedTail`
#[test]
fn compile_complete_chapter1_pre_uses_chapter2_context_and_shared_tail() {
    let Some(beats) = bind_ok("ch01_pre.json") else {
        return;
    };
    assert_eq!(
        beats.len(),
        38,
        "ch01_pre compiled beats={}, want 22 source beats with 4 dialogs expanded 4→20",
        beats.len()
    );
    let (seen, dialogs) = collect(&beats);
    assert_eq!(dialogs.len(), 20, "FDTXT_002 #0..3 dialogs={}, want 1+3+6+10", dialogs.len());
    for (i, dialog) in dialogs.iter().enumerate() {
        assert!(dialog.native_dialogue.is_some(), "FDTXT_002 dialog {i} lacks caller-bound native layout: {dialog:?}");
    }
    for (i, start) in [0usize, 1, 4, 10].into_iter().enumerate() {
        let d = &dialogs[start];
        assert!(
            d.line == start as i64 && d.script == "ch02.json" && d.scene_index == Some(0),
            "dialog group {i} start = {d:?}"
        );
    }
    assert!(
        dialogs[0].source == "0x32d66"
            && dialogs[1].source == "0x32dbb"
            && dialogs[4].source == "0x32e24"
            && dialogs[10].source == "0x3320c",
        "chapter2 dialog source groups drifted: {dialogs:?}"
    );
    let load = at(&seen, "0x32d22");
    // Go 以 `fmt.Sprint(PartyOrder) == "[0 9 4 30 1]"` 比對；Rust 直接比切片。
    assert!(
        load.load_ch.as_ref().is_some_and(|s| s.chapter == 1
            && s.map == "assets/maps/map1"
            && s.script == "assets/story/ch02.json"
            && s.party_scenario == "assets/scenarios/ch02.json"
            && s.party_order == vec![0, 9, 4, 30, 1]),
        "ch01_pre LOADCH = {:?}",
        load.load_ch
    );
    let (first, second) = (at(&seen, "0x32d2b"), at(&seen, "0x32e3f"));
    assert!(
        first.tile_step && first.x == 312 && first.y == 264 && second.tile_step && second.x == 144 && second.y == 288,
        "ch01_pre PAN mappings = {first:?} / {second:?}"
    );
    let focus = at(&seen, "0x33142");
    assert!(focus.op == "focus_unit" && focus.slot == Some(0), "shared-tail focus missing: {focus:?}");
}

/// Go：`TestCompileCompleteChapter2PreUsesRecoveredChapter3Text`
#[test]
fn compile_complete_chapter2_pre_uses_recovered_chapter3_text() {
    let Some(beats) = bind_ok("ch02_pre.json") else {
        return;
    };
    assert_eq!(
        beats.len(),
        26,
        "ch02_pre compiled beats={}, want 16 source beats with dialogs expanded 4→14",
        beats.len()
    );
    let (seen, dialogs) = collect(&beats);
    assert_eq!(dialogs.len(), 14, "FDTXT_003 #0..3 dialogs={}, want 2+1+4+7", dialogs.len());
    for (i, start) in [0usize, 2, 3, 7].into_iter().enumerate() {
        let d = &dialogs[start];
        assert!(
            d.line == start as i64 && d.script == "ch03.json" && d.scene_index == Some(0),
            "dialog group {i} start = {d:?}"
        );
    }
    assert!(
        dialogs[0].source == "0x32ed3"
            && dialogs[2].source == "0x32f3b"
            && dialogs[3].source == "0x32f76"
            && dialogs[7].source == "0x33133",
        "chapter3 dialog source groups drifted: {dialogs:?}"
    );
    let load = at(&seen, "0x32e96");
    assert!(
        load.load_ch.as_ref().is_some_and(|s| s.chapter == 2
            && s.map == "assets/maps/map2"
            && s.script == "assets/story/ch03.json"
            && s.party_scenario == "assets/scenarios/ch03.json"
            && s.party_order == vec![0, 9, 4, 30, 1, 8]),
        "ch02_pre LOADCH = {:?}",
        load.load_ch
    );
    for (source, want) in [("0x32e9f", [72, 408]), ("0x32efd", [72, 144]), ("0x32f8c", [72, 408])] {
        let pan = at(&seen, source);
        assert!(pan.tile_step && pan.x == want[0] && pan.y == want[1], "ch02_pre PAN {source} = {pan:?}");
    }
    let (act18, act17, act19) = (at(&seen, "0x32ee7"), at(&seen, "0x32f14"), at(&seen, "0x32f4f"));
    assert!(
        act18.acting.len() == 1
            && act18.acting[0].units.len() == 6
            && act17.acting.len() == 5
            && act17.acting[0].units[0].slot.unwrap() == 6
            && act19.acting.len() == 3
            && act19.acting[0].units.len() == 8
            && act19.acting[0].units[0].slot.unwrap() == 7,
        "ch02_pre acting resources drifted: 18={:?} 17={:?} 19={:?}",
        act18.acting,
        act17.acting,
        act19.acting
    );
    let focus = at(&seen, "0x33142");
    assert!(focus.op == "focus_unit" && focus.slot == Some(0), "ch02_pre shared-tail focus missing: {focus:?}");
}

/// Go：`TestCompileChapter3PreUsesRecoveredChapter4TextGroups`
#[test]
fn compile_chapter3_pre_uses_recovered_chapter4_text_groups() {
    let Some(beats) = bind_ok("ch03_pre.json") else {
        return;
    };
    let (seen, dialogs) = collect(&beats);
    assert_eq!(dialogs.len(), 9, "FDTXT_004 #0/#1 dialogs={}, want 4+5", dialogs.len());
    assert!(
        dialogs[0].script == "ch04.json" && dialogs[0].scene_index == Some(0) && dialogs[0].line == 0,
        "ch03_pre first dialogue = {:?}",
        dialogs[0]
    );
    assert!(
        dialogs[4].script == "ch04.json" && dialogs[4].scene_index == Some(1) && dialogs[4].line == 0,
        "ch03_pre second dialogue = {:?}",
        dialogs[4]
    );
    check_load(
        "ch03_pre",
        &at(&seen, "0x32fbc"),
        3,
        "assets/maps/map3",
        None,
        "assets/story/ch04.json",
        "assets/scenarios/ch04.json",
    );
    check_pan("ch03_pre initial", &at(&seen, "0x32fc5"), 96, 264);
    let act = at(&seen, "0x32fcf");
    assert!(act.acting.len() == 4 && !act.acting[0].units.is_empty(), "ch03_pre acting = {:?}", act.acting);
}

/// Go：`TestCompileChapter4PreUsesRecoveredChapter5TextGroups`
#[test]
fn compile_chapter4_pre_uses_recovered_chapter5_text_groups() {
    let Some(beats) = bind_ok("ch04_pre.json") else {
        return;
    };
    let (seen, dialogs) = collect(&beats);
    assert_eq!(dialogs.len(), 15, "FDTXT_005 #0/#1/#2 dialogs={}, want 3+3+9", dialogs.len());
    for (i, (index, line, scene)) in [(0usize, 0i64, 0i64), (3, 3, 0), (6, 0, 1)].into_iter().enumerate() {
        let got = &dialogs[index];
        assert!(
            got.script == "ch05.json" && got.scene_index == Some(scene) && got.line == line,
            "ch04_pre dialogue group {i} = {got:?}"
        );
    }
    check_load(
        "ch04_pre",
        &at(&seen, "0x33053"),
        4,
        "assets/maps/map4",
        None,
        "assets/story/ch05.json",
        "assets/scenarios/ch05.json",
    );
    check_pan("ch04_pre initial", &at(&seen, "0x3308d"), 72, 72);
    check_pan("ch04_pre second", &at(&seen, "0x33102"), 192, 336);
    // Go 比 `Units == nil`；Rust 的 Vec 不分 nil 與空，改比是否為空。
    let act22 = at(&seen, "0x330c5");
    assert!(act22.acting.len() == 2 && !act22.acting[0].units.is_empty(), "ch04_pre acting22 = {:?}", act22.acting);
    let act21 = at(&seen, "0x3310c");
    assert!(act21.acting.len() == 3 && !act21.acting[0].units.is_empty(), "ch04_pre acting21 = {:?}", act21.acting);
}

/// Go：`TestCompileLoadCHBindingExposesSharedTailDialogueGap`
#[test]
fn compile_load_ch_binding_exposes_shared_tail_dialogue_gap() {
    let Some(beats) = bind_ok("ch05_pre.json") else {
        return;
    };
    let n = beats.len();
    assert!(
        n == 20
            && beats[0].op == "loadch"
            && beats[0].source == "0x33155"
            && beats[0].load_ch.is_some()
            && beats[n - 1].op == "focus_unit"
            && beats[n - 1].source == "0x33142",
        "ch05 loadch beat = {beats:?}"
    );
    let (_, dialogs) = collect(&beats);
    assert_eq!(dialogs.len(), 18, "FDTXT_006 #0 cross-scene dialogs={}, want 18", dialogs.len());
    let wants = [(0, 0), (1, 0), (1, 2), (2, 0), (2, 4), (3, 0), (3, 8)];
    let indices = [0usize, 1, 3, 4, 8, 9, 17];
    for (i, (scene, line)) in wants.into_iter().enumerate() {
        let got = &dialogs[indices[i]];
        assert!(
            got.script == "ch06.json" && got.scene_index == Some(scene) && got.line == line,
            "cross-scene dialogue boundary {i} = {got:?}"
        );
    }
    let state = beats[0].load_ch.as_ref().unwrap();
    // Handler filenames are jump-table indices. Index 5 leaves the global
    // chapter at 5, so LOADCH selects map5 and FDTXT_006/ch06, not map4/ch05.
    assert!(
        state.chapter == 5
            && state.map == "assets/maps/map5"
            && state.roster == "assets/maps/map5/map5_units.json"
            && state.slot_count == 40
            && state.script == "assets/story/ch06.json",
        "ch05 loadch state = {state:?}"
    );
}

/// Go：`TestCompileChapter6PreUsesRecoveredChapter7TextGroups`
#[test]
fn compile_chapter6_pre_uses_recovered_chapter7_text_groups() {
    let Some(beats) = bind_ok("ch06_pre.json") else {
        return;
    };
    let (seen, dialogs) = collect(&beats);
    assert_eq!(dialogs.len(), 8, "FDTXT_007 #0/#1 dialogs={}, want 2+6", dialogs.len());
    check_load(
        "ch06_pre",
        &at(&seen, "0x33173"),
        6,
        "assets/maps/map6",
        None,
        "assets/story/ch07.json",
        "assets/scenarios/ch07.json",
    );
    check_pan("ch06_pre first", &at(&seen, "0x331c5"), 192, 24);
    check_pan("ch06_pre second", &at(&seen, "0x331db"), 192, 0);
    check_act_units("ch06_pre acting28", &at(&seen, "0x331cf"));
    check_act_units("ch06_pre acting29", &at(&seen, "0x331e5"));
}

/// Go：`TestCompileChapter7PreUsesRecoveredChapter8TextGroups`
#[test]
fn compile_chapter7_pre_uses_recovered_chapter8_text_groups() {
    let Some(beats) = bind_ok("ch07_pre.json") else {
        return;
    };
    let (seen, dialogs) = collect(&beats);
    assert_eq!(dialogs.len(), 17, "FDTXT_008 #0/#1 dialogs={}, want 15+2", dialogs.len());
    check_load(
        "ch07_pre",
        &at(&seen, "0x33223"),
        7,
        "assets/maps/map7",
        None,
        "assets/story/ch08.json",
        "assets/scenarios/ch08.json",
    );
    check_pan("ch07_pre first", &at(&seen, "0x3322c"), 168, 768);
    check_pan("ch07_pre second", &at(&seen, "0x33269"), 168, 552);
    check_act_units("ch07_pre acting31", &at(&seen, "0x33236"));
}

/// Go：`TestCompileChapter8PreUsesRecoveredChapter9TextGroups`
#[test]
fn compile_chapter8_pre_uses_recovered_chapter9_text_groups() {
    let Some(beats) = bind_ok("ch08_pre.json") else {
        return;
    };
    let (seen, dialogs) = collect(&beats);
    assert_eq!(dialogs.len(), 7, "FDTXT_009 #0/#1 dialogs={}, want 2+5", dialogs.len());
    check_load(
        "ch08_pre",
        &at(&seen, "0x33288"),
        8,
        "assets/maps/map8",
        None,
        "assets/story/ch09.json",
        "assets/scenarios/ch09.json",
    );
    check_pan("ch08_pre", &at(&seen, "0x332b0"), 144, 0);
    check_act_units("ch08_pre acting35", &at(&seen, "0x332eb"));
}

/// Go：`TestCompileChapter9PreUsesRecoveredChapter10TextGroups`
#[test]
fn compile_chapter9_pre_uses_recovered_chapter10_text_groups() {
    let Some(beats) = bind_ok("ch09_pre.json") else {
        return;
    };
    let (seen, dialogs) = collect(&beats);
    assert_eq!(dialogs.len(), 12, "FDTXT_010 #0 cross-scene dialogs={}, want 6+6", dialogs.len());
    assert!(
        dialogs[0].scene_index == Some(0)
            && dialogs[0].line == 0
            && dialogs[5].scene_index == Some(0)
            && dialogs[5].line == 5
            && dialogs[6].scene_index == Some(1)
            && dialogs[6].line == 0
            && dialogs[11].line == 5,
        "ch09_pre cross-scene boundaries = {dialogs:?}"
    );
    check_load(
        "ch09_pre",
        &at(&seen, "0x33335"),
        9,
        "assets/maps/map9",
        None,
        "assets/story/ch10.json",
        "assets/scenarios/ch10.json",
    );
    check_pan("ch09_pre", &at(&seen, "0x3333e"), 240, 0);
}

/// Go：`TestCompileChapter10PreUsesRecoveredChapter11TextGroups`
#[test]
fn compile_chapter10_pre_uses_recovered_chapter11_text_groups() {
    let Some(beats) = bind_ok("ch10_pre.json") else {
        return;
    };
    let (seen, dialogs) = collect(&beats);
    assert_eq!(dialogs.len(), 25, "FDTXT_011 #0/#1/#2 dialogs={}, want 12+1+12", dialogs.len());
    assert!(
        dialogs[0].scene_index == Some(0)
            && dialogs[4].scene_index == Some(1)
            && dialogs[10].scene_index == Some(2)
            && dialogs[12].scene_index == Some(2)
            && dialogs[12].line == 2
            && dialogs[13].line == 3,
        "ch10_pre cross-scene boundaries = {dialogs:?}"
    );
    check_load(
        "ch10_pre",
        &at(&seen, "0x33371"),
        10,
        "assets/maps/map10",
        None,
        "assets/story/ch11.json",
        "assets/scenarios/ch11.json",
    );
    check_pan("ch10_pre", &at(&seen, "0x333ab"), 240, 168);
    check_act_units("ch10_pre acting38", &at(&seen, "0x333bf"));
}

/// Go：`TestCompileChapter11PreUsesRecoveredChapter12TextGroups`
#[test]
fn compile_chapter11_pre_uses_recovered_chapter12_text_groups() {
    let Some(beats) = bind_ok("ch11_pre.json") else {
        return;
    };
    let (seen, dialogs) = collect(&beats);
    assert_eq!(dialogs.len(), 11, "FDTXT_012 #0 cross-scene dialogs={}, want 2+9", dialogs.len());
    assert!(
        dialogs[0].scene_index == Some(0)
            && dialogs[1].scene_index == Some(0)
            && dialogs[2].scene_index == Some(1)
            && dialogs[10].line == 8,
        "ch11_pre cross-scene boundaries = {dialogs:?}"
    );
    check_load(
        "ch11_pre",
        &at(&seen, "0x333ff"),
        11,
        "assets/maps/map11",
        None,
        "assets/story/ch12.json",
        "assets/scenarios/ch12.json",
    );
    check_pan("ch11_pre first", &at(&seen, "0x33408"), 96, 96);
    check_pan("ch11_pre second", &at(&seen, "0x33436"), 264, 960);
    check_act_units("ch11_pre acting40", &at(&seen, "0x3342a"));
}

/// Go：`TestCompileChapter12PreUsesRecoveredChapter13TextGroups`
#[test]
fn compile_chapter12_pre_uses_recovered_chapter13_text_groups() {
    let Some(beats) = bind_ok("ch12_pre.json") else {
        return;
    };
    let (seen, dialogs) = collect(&beats);
    assert_eq!(dialogs.len(), 6, "FDTXT_013 #0 dialogs={}, want 6", dialogs.len());
    assert!(dialogs[0].scene_index == Some(0) && dialogs[5].line == 5, "ch12_pre dialogue mapping = {dialogs:?}");
    check_load(
        "ch12_pre",
        &at(&seen, "0x33475"),
        12,
        "assets/maps/map12",
        Some(70),
        "assets/story/ch13.json",
        "assets/scenarios/ch13.json",
    );
}

/// Go：`TestCompileChapter13PreUsesRecoveredChapter14TextGroups`
#[test]
fn compile_chapter13_pre_uses_recovered_chapter14_text_groups() {
    let Some(beats) = bind_ok("ch13_pre.json") else {
        return;
    };
    let (seen, dialogs) = collect(&beats);
    assert_eq!(dialogs.len(), 4, "FDTXT_014 #0 dialogs={}, want 4", dialogs.len());
    assert!(dialogs[0].scene_index == Some(0) && dialogs[3].line == 3, "ch13_pre dialogue mapping = {dialogs:?}");
    check_load(
        "ch13_pre",
        &at(&seen, "0x33486"),
        13,
        "assets/maps/map13",
        Some(70),
        "assets/story/ch14.json",
        "assets/scenarios/ch14.json",
    );
    check_pan("ch13_pre", &at(&seen, "0x3348f"), 480, 480);
}

/// Go：`TestCompileChapter14PreLowersRosterHasDialogueVariants`
#[test]
fn compile_chapter14_pre_lowers_roster_has_dialogue_variants() {
    let Some(beats) = bind_ok("ch14_pre.json") else {
        return;
    };
    assert!(
        beats.len() == 5
            && beats[0].op == "loadch"
            && beats[1].op == "if"
            && beats[2].op == "act"
            && beats[3].op == "if"
            && beats[4].op == "focus_unit",
        "ch14_pre top-level beats={beats:?}"
    );
    let load = &beats[0].load_ch;
    assert!(
        load.as_ref().is_some_and(|s| s.chapter == 14
            && s.map == "assets/maps/map14"
            && s.slot_count == 80
            && s.script == "assets/story/ch15.json"
            && s.party_scenario == "assets/scenarios/ch15.json"),
        "ch14_pre LOADCH={load:?}"
    );
    let roster12 = |b: &Beat| b.condition.as_ref().is_some_and(|c| c.op == "roster_has" && c.char_id == Some(12));
    for (arm, want_scene) in [0i64, 1].into_iter().enumerate() {
        let mut branch = beats[1].clone();
        if arm == 1 {
            branch.then = branch.r#else.clone();
        }
        let t = &branch.then;
        assert!(
            roster12(&branch)
                && t.len() == 5
                && t[0].op == "dialog"
                && t[0].scene_index == Some(want_scene)
                && t[2].op == "dialog"
                && t[2].line == 2
                && t[3].op == "pan"
                && t[3].x == 576
                && t[3].y == 408
                && t[3].tile_step
                && t[4].op == "dialog"
                && t[4].line == 3,
            "ch14_pre variant {arm}={branch:?}"
        );
    }
    assert!(
        !beats[2].acting.is_empty() && beats[2].acting[0].units[0].slot == Some(64),
        "ch14_pre acting={:?}",
        beats[2]
    );
    // FDTXT_015 string indexes 2/5 are count-aligned continuations after
    // strings 0/1 and 3/4, so their editable story lines begin at 4, not 0.
    for (arm, (scene, first_line, count)) in [(0i64, 4i64, 9usize), (1, 4, 5)].into_iter().enumerate() {
        let mut branch = beats[3].clone();
        if arm == 1 {
            branch.then = branch.r#else.clone();
        }
        let t = &branch.then;
        assert!(
            roster12(&branch)
                && t.len() == count
                && t[0].op == "dialog"
                && t[0].scene_index == Some(scene)
                && t[0].line == first_line
                && t[t.len() - 1].line == first_line + count as i64 - 1,
            "ch14_pre final variant {arm}={branch:?}"
        );
    }
}

/// Go：`TestCompileChapter14PostLowersRosterHasDialogueVariants`
#[test]
fn compile_chapter14_post_lowers_roster_has_dialogue_variants() {
    let Some(beats) = bind_ok("ch14_post.json") else {
        return;
    };
    assert!(
        beats.len() == 4
            && beats[0].op == "if"
            && beats[1].op == "sync_party"
            && beats[2].op == "join"
            && beats[2].char_id == 15
            && beats[3].op == "set_chapter"
            && beats[3].chapter == Some(15),
        "ch14_post beats={beats:?}"
    );
    let branch = &beats[0];
    let (t, e) = (&branch.then, &branch.r#else);
    assert!(
        branch.condition.as_ref().is_some_and(|c| c.op == "roster_has" && c.char_id == Some(12))
            && t.len() == 12
            && e.len() == 12
            && t[0].op == "dialog"
            && t[0].line == 0
            && t[11].line == 11
            && e[0].op == "dialog"
            && e[0].line == 0
            && e[11].line == 11
            && t[0].scene_index == Some(4)
            && e[0].scene_index == Some(5),
        "ch14_post dialogue variants={branch:?}"
    );
}

/// Go：`TestCompileChapter15PreUsesRecoveredChapter16TextGroups`
#[test]
fn compile_chapter15_pre_uses_recovered_chapter16_text_groups() {
    let Some(beats) = bind_ok("ch15_pre.json") else {
        return;
    };
    let (seen, dialogs) = collect(&beats);
    assert_eq!(dialogs.len(), 16, "FDTXT_016 #0 dialogs={}, want 16", dialogs.len());
    assert!(dialogs[0].scene_index == Some(0) && dialogs[15].line == 15, "ch15_pre dialogue mapping = {dialogs:?}");
    check_load(
        "ch15_pre",
        &at(&seen, "0x33475"),
        15,
        "assets/maps/map15",
        Some(60),
        "assets/story/ch16.json",
        "assets/scenarios/ch16.json",
    );
}

/// Go：`TestCompileChapter16PreLowersAbsentRosterSpawn`
#[test]
fn compile_chapter16_pre_lowers_absent_roster_spawn() {
    let Some(beats) = bind_ok("ch16_pre.json") else {
        return;
    };
    assert!(
        beats.len() >= 4
            && beats[0].op == "loadch"
            && beats[0]
                .load_ch
                .as_ref()
                .is_some_and(|s| s.chapter == 16 && s.map == "assets/maps/map16" && s.slot_count == 60)
            && beats[1].op == "if"
            && beats[1].condition.as_ref().is_some_and(|c| c.op == "roster_has" && c.char_id == Some(18))
            && beats[1].then.is_empty()
            && beats[1].r#else.len() == 1
            && beats[1].r#else[0].op == "spawn"
            && beats[1].r#else[0].group == 1,
        "ch16_pre roster branch={beats:?}"
    );
}

/// Go：`TestCompileChapter17PreUsesRecoveredChapter18TextGroups`
#[test]
fn compile_chapter17_pre_uses_recovered_chapter18_text_groups() {
    let Some(beats) = bind_ok("ch17_pre.json") else {
        return;
    };
    let (seen, dialogs) = collect(&beats);
    assert_eq!(dialogs.len(), 24, "FDTXT_018 #0/#1/#2 dialogs={}, want 7+4+13", dialogs.len());
    assert!(
        dialogs[0].scene_index == Some(0)
            && dialogs[6].line == 6
            && dialogs[7].scene_index == Some(0)
            && dialogs[11].scene_index == Some(1),
        "ch17_pre dialogue mapping = {dialogs:?}"
    );
    check_load(
        "ch17_pre",
        &at(&seen, "0x335e4"),
        17,
        "assets/maps/map17",
        Some(70),
        "assets/story/ch18.json",
        "assets/scenarios/ch18.json",
    );
    check_pan("ch17_pre first", &at(&seen, "0x3361e"), 384, 96);
    check_act_units("ch17_pre acting54", &at(&seen, "0x33628"));
}

/// Go：`TestCompileChapter18PreUsesRecoveredChapter19TextGroups`
#[test]
fn compile_chapter18_pre_uses_recovered_chapter19_text_groups() {
    let Some(beats) = bind_ok("ch18_pre.json") else {
        return;
    };
    let (seen, dialogs) = collect(&beats);
    assert_eq!(dialogs.len(), 8, "FDTXT_019 #0 dialogs={}, want 8", dialogs.len());
    assert!(dialogs[0].scene_index == Some(0) && dialogs[7].line == 7, "ch18_pre dialogue mapping = {dialogs:?}");
    check_load(
        "ch18_pre",
        &at(&seen, "0x33475"),
        18,
        "assets/maps/map18",
        Some(70),
        "assets/story/ch19.json",
        "assets/scenarios/ch19.json",
    );
}

/// Go：`TestCompileChapter19PreUsesRecoveredChapter20TextGroups`
#[test]
fn compile_chapter19_pre_uses_recovered_chapter20_text_groups() {
    let Some(beats) = bind_ok("ch19_pre.json") else {
        return;
    };
    let (seen, dialogs) = collect(&beats);
    assert_eq!(dialogs.len(), 17, "FDTXT_020 #0 dialogs={}, want 17", dialogs.len());
    assert!(dialogs[0].scene_index == Some(0) && dialogs[16].line == 16, "ch19_pre dialogue mapping = {dialogs:?}");
    check_load(
        "ch19_pre",
        &at(&seen, "0x33475"),
        19,
        "assets/maps/map19",
        Some(70),
        "assets/story/ch20.json",
        "assets/scenarios/ch20.json",
    );
}

/// Go：`TestCompileChapter20PreUsesRecoveredChapter21TextGroups`
#[test]
fn compile_chapter20_pre_uses_recovered_chapter21_text_groups() {
    let Some(beats) = bind_ok("ch20_pre.json") else {
        return;
    };
    let (seen, dialogs) = collect(&beats);
    assert_eq!(dialogs.len(), 17, "FDTXT_021 #0 dialogs={}, want 17", dialogs.len());
    assert!(dialogs[0].scene_index == Some(0) && dialogs[16].line == 16, "ch20_pre dialogue mapping = {dialogs:?}");
    check_load(
        "ch20_pre",
        &at(&seen, "0x33475"),
        20,
        "assets/maps/map20",
        Some(80),
        "assets/story/ch21.json",
        "assets/scenarios/ch21.json",
    );
}

/// Go：`TestCompileChapter21PreUsesRecoveredChapter22TextGroups`
#[test]
fn compile_chapter21_pre_uses_recovered_chapter22_text_groups() {
    let Some(beats) = bind_ok("ch21_pre.json") else {
        return;
    };
    let (seen, dialogs) = collect(&beats);
    assert_eq!(dialogs.len(), 11, "FDTXT_022 #0 dialogs={}, want 11", dialogs.len());
    assert!(dialogs[0].scene_index == Some(0) && dialogs[10].line == 10, "ch21_pre dialogue mapping = {dialogs:?}");
    check_load(
        "ch21_pre",
        &at(&seen, "0x33688"),
        21,
        "assets/maps/map21",
        Some(70),
        "assets/story/ch22.json",
        "assets/scenarios/ch22.json",
    );
    check_pan("ch21_pre", &at(&seen, "0x33691"), 384, 672);
    check_act_units("ch21_pre acting67", &at(&seen, "0x33440"));
}

/// Go：`TestCompileHandlerScriptRejectsActingOutsideActiveLoadCHSlots`
#[test]
fn compile_handler_script_rejects_acting_outside_active_load_ch_slots() {
    let bindings = HandlerBindings {
        load_ch: cb!(|_: HandlerBeat| {
            (
                load_ch_state(0, "assets/maps/map0", "assets/maps/map0/map0_units.json", 30, "assets/story/ch01.json"),
                true,
            )
        }),
        acting: cb!(|_: HandlerBeat| {
            (
                vec![ActingFrame {
                    beats: 1,
                    units: vec![ActingUnit { slot: Some(30), pose: 3, ..Default::default() }],
                    ..Default::default()
                }],
                true,
            )
        }),
        ..Default::default()
    };
    let (beats, issues) = compile(
        vec![
            op_at("loadch", addr("0x100")),
            HandlerBeat { op: "act".into(), acting_id: ip(9), source: addr("0x101"), ..Default::default() },
        ],
        &bindings,
    );
    assert!(
        beats.len() == 1 && beats[0].op == "loadch" && issues.len() == 1 && issues[0].op == "act",
        "out-of-range acting must fail closed: beats={beats:?} issues={issues:?}"
    );
}

/// Go：`TestLoadActingResourceSetAndCh00References`
#[test]
fn load_acting_resource_set_and_ch00_references() {
    let (Some(acting_path), Some(binding_path)) =
        (asset("cutscenes/acting/map32.json"), asset("cutscenes/bindings/ch00_pre.json"))
    else {
        return;
    };
    let resources = load_acting_resource_set(&acting_path);
    let count = resources.as_ref().map(|r| r.len()).unwrap_or(0);
    assert!(resources.is_ok() && count == 106, "acting resources err={:?} count={count}", resources.as_ref().err());
    let resources = resources.unwrap();
    let get = |id: i64| resources.get(&id).cloned().unwrap_or_default();
    let frames = get(102);
    assert!(frames.len() == 3 && !frames[0].special && frames[0].units[0].slot == Some(4), "resource 102 = {frames:?}");
    let frames = get(0);
    assert!(
        frames.len() == 5
            && !frames[0].special
            && frames[0].beats == 6
            && frames[0].units.len() == 4
            && frames[0].units[1].slot.unwrap() == 1
            && frames[1].special,
        "resource 0 = {frames:?}"
    );
    let frames = get(2);
    assert!(
        frames.len() == 4 && frames[3].beats == 4 && frames[3].special && frames[3].units.len() == 5,
        "resource 2 = {frames:?}"
    );
    let frames = get(5);
    assert!(
        frames.len() == 1
            && frames[0].beats == 4
            && frames[0].units[0].slot.unwrap() == 9
            && frames[0].units[0].pose == 0,
        "resource 5 = {frames:?}"
    );
    let binding = load_handler_binding(&binding_path).unwrap();
    let acting = binding.compiler_bindings().acting.unwrap();
    let call = |id: i64, source: &str| {
        (acting.0)(HandlerBeat { acting_id: ip(id), source: addr(source), ..Default::default() })
    };
    let (frames, ok) = call(104, "0x324d7");
    assert!(
        ok && frames.len() == 1 && frames[0].special && frames[0].units[0].slot == Some(3),
        "ch00 resource acting resolve={frames:?} ok={ok}"
    );
    let (_, ok) = call(103, "0x324d7");
    assert!(!ok, "acting resource reference accepted mismatched original resource id");
    let (map0, ok) = call(0, "0x3283a");
    assert!(ok && map0.len() == 5 && map0[0].units[0].slot == Some(0), "map0 resource acting resolve={map0:?} ok={ok}");
    let (map31, ok) = call(90, "0x3255f");
    assert!(
        ok && map31.len() == 5
            && map31[0].beats == 1
            && !map31[0].special
            && map31[0].units.len() == 1
            && map31[0].units[0].slot == Some(0),
        "map31 resource acting resolve={map31:?} ok={ok}"
    );
}

/// Go：`TestCompileGeneratedHandlerBindingsCompletionFrontier`
#[test]
fn compile_generated_handler_bindings_completion_frontier() {
    let Some(dir) = asset("cutscenes/bindings/generated") else {
        return;
    };
    // Go `filepath.Glob(".../generated/ch??_*.json")`：結果依名稱排序。
    let mut paths: Vec<String> = std::fs::read_dir(&dir)
        .unwrap()
        .filter_map(|entry| {
            let name = entry.ok()?.file_name().into_string().ok()?;
            let b = name.as_bytes();
            let matches = name.starts_with("ch")
                && b.len() > 5
                && b[4] == b'_'
                && name.ends_with(".json")
                && name[2..4].chars().count() == 2;
            matches.then(|| format!("{dir}/{name}"))
        })
        .collect();
    paths.sort();
    assert_eq!(paths.len(), 60, "generated bindings={}", paths.len());
    let complete: BTreeSet<&str> = [
        "ch00_post.json",
        "ch03_post.json",
        "ch10_post.json",
        "ch18_post.json",
        "ch27_post.json",
        "ch25_post.json",
        "ch14_post.json",
    ]
    .into_iter()
    .collect();
    let mut failures = Vec::new();
    for path in &paths {
        let (_, issues) =
            compile_handler_binding(path).unwrap_or_else(|err| panic!("CompileHandlerBinding({path:?}): {err}"));
        let base = std::path::Path::new(path).file_name().unwrap().to_string_lossy().into_owned();
        // Go `filepath.Join(filepath.Dir(path), "../../handlers", base)`。
        let script_path = format!("{dir}/../../handlers/{base}");
        let script = load_handler_script(&script_path).unwrap();
        let want_complete = complete.contains(base.as_str());
        if !script.beats.is_empty() && issues.is_empty() != want_complete {
            failures.push(format!(
                "{path} completion={} issues={issues:?}, want completion={want_complete}",
                issues.is_empty()
            ));
        }
    }
    assert!(failures.is_empty(), "{}", failures.join("\n"));
}

/// Go：`TestCompileFixedRepeatDeactivateRange`
#[test]
fn compile_fixed_repeat_deactivate_range() {
    let limit = 16;
    let input = HandlerBeat {
        op: "deactivate_unit".into(),
        source: addr("0x336b5"),
        unit_slot_expr: json!("ebx"),
        repeat_hint: Some(HandlerRepeatHint { loop_back_to: "0x336b4".into(), limit }),
        ..Default::default()
    };
    let (beats, issues) = compile(vec![input], &ctx(70));
    assert!(issues.is_empty(), "fixed repeat issues={issues:?}");
    assert_eq!(beats.len() as i64, limit, "fixed repeat beats={} want {limit}", beats.len());
    for (i, beat) in beats.iter().enumerate() {
        assert!(beat.op == "deactivate_unit" && beat.slot == Some(i as i64), "fixed repeat beat[{i}]={beat:?}");
    }
}

/// Go：`TestCompilePaletteUpdateFromNativeCall`
#[test]
fn compile_palette_update_from_native_call() {
    let (beats, issues) = compile(
        vec![unknown("0x11df2", vec![json!(0.0), json!(255.0), json!(0.0)], src("0x3372d", "0x11df2"))],
        &none(),
    );
    assert!(issues.is_empty() && beats.len() == 1, "palette update lowering beats={beats:?} issues={issues:?}");
    assert!(
        beats[0].op == "palette_update"
            && beats[0].palette_start == 0
            && beats[0].palette_end == 255
            && beats[0].palette_delta == 0,
        "palette update beat={:?}",
        beats[0]
    );
}

/// Go：`TestCompileTransitionRevealFromNativeCall`
#[test]
fn compile_transition_reveal_from_native_call() {
    let (beats, issues) = compile(vec![unknown("0x24b4d", vec![json!(60.0)], src("0x33a70", "0x24b4d"))], &none());
    assert!(issues.is_empty() && beats.len() == 1, "transition lowering beats={beats:?} issues={issues:?}");
    assert!(
        beats[0].op == "transition_reveal" && beats[0].reveal_frames == 60 && beats[0].reveal_delay_ms == 20,
        "transition beat={:?}",
        beats[0]
    );
}

/// Go：`TestCompilePaletteRamp25052PreservesInclusiveDescendingDeltas`
#[test]
fn compile_palette_ramp25052_preserves_inclusive_descending_deltas() {
    let (beats, issues) =
        compile(vec![unknown("0x25052", vec![json!(2), json!(80)], src("0x252a9", "0x25052"))], &none());
    assert!(issues.is_empty() && beats.len() == 6, "0x25052 beats={beats:?} issues={issues:?}");
    for (i, delta) in [2, 1, 0].into_iter().enumerate() {
        let (palette, delay) = (&beats[i * 2], &beats[i * 2 + 1]);
        assert!(
            palette.op == "palette_update"
                && palette.source == "0x252a9"
                && palette.palette_start == 0
                && palette.palette_end == 255
                && palette.palette_delta == delta
                && delay.op == "delay"
                && delay.ms == 80,
            "ramp step {i} = {palette:?} / {delay:?}"
        );
    }
    let (_, issues) = compile(vec![unknown("0x25052", vec![json!(64), json!(1)], HandlerSource::default())], &none());
    assert_eq!(issues.len(), 1, "invalid 0x25052 ramp issues={issues:?}");
}

/// Go：`TestCompileNativePaletteFadeOut1882PreservesExactDACSchedule`
#[test]
fn compile_native_palette_fade_out1882_preserves_exact_dac_schedule() {
    let (beats, issues) = compile(vec![unknown("0x1f882", Vec::new(), src("0x2531a", "0x1f882"))], &none());
    assert!(
        issues.is_empty()
            && beats.len() == 1
            && beats[0].op == "native_palette_fade_out"
            && beats[0].native_palette_fade.is_some(),
        "0x1f882 beats={beats:?} issues={issues:?}"
    );
    let got = beats[0].native_palette_fade.unwrap();
    assert!(got.start == 0 && got.end == 63 && got.delay_ms == 2, "0x1f882 payload={got:?}");
    let (_, issues) = compile(vec![unknown("0x1f882", vec![json!(1)], HandlerSource::default())], &none());
    assert_eq!(issues.len(), 1, "argument-bearing 0x1f882 issues={issues:?}");
}

/// Go：`TestCompileNativePalettePulse35E5APreservesExactDACSchedule`
#[test]
fn compile_native_palette_pulse35_e5_a_preserves_exact_dac_schedule() {
    let (beats, issues) = compile(vec![unknown("0x35e5a", Vec::new(), src("0x33efd", "0x35e5a"))], &none());
    assert!(
        issues.is_empty()
            && beats.len() == 1
            && beats[0].op == "native_palette_pulse"
            && beats[0].native_palette_pulse.is_some(),
        "0x35e5a beats={beats:?} issues={issues:?}"
    );
    let got = beats[0].native_palette_pulse.unwrap();
    assert!(
        got.rise_start == 0
            && got.rise_end == 63
            && got.rise_delay_ms == 8
            && got.hold_ms == 400
            && got.fall_start == 62
            && got.fall_end == 0
            && got.fall_delay_ms == 8,
        "0x35e5a payload={got:?}"
    );
    let (_, issues) = compile(vec![unknown("0x35e5a", vec![json!(1)], HandlerSource::default())], &none());
    assert_eq!(issues.len(), 1, "argument-bearing 0x35e5a issues={issues:?}");
}

/// Go：`TestCompileNativeRecordBit7ClearAndPaletteRegisterSnapshot`
#[test]
fn compile_native_record_bit7_clear_and_palette_register_snapshot() {
    let (beats, issues) = compile(
        vec![
            unknown("0x1f882", vec![json!("ebx"), json!("esi"), json!("edi")], src("0x23623", "0x1f882")),
            unknown("0x13536", Vec::new(), src("0x23628", "0x13536")),
        ],
        &none(),
    );
    assert!(
        issues.is_empty()
            && beats.len() == 2
            && beats[0].op == "native_palette_fade_out"
            && beats[1].op == "clear_native_record_bit7",
        "raw handler lowerings beats={beats:?} issues={issues:?}"
    );
    let (_, issues) = compile(vec![unknown("0x1f882", vec![json!(1)], HandlerSource::default())], &none());
    assert_eq!(issues.len(), 1, "immediate argument-bearing palette fade should remain rejected: {issues:?}");
}

/// Go：`TestCompileNativeStagingPresent33F78PreservesWrapperABI`
#[test]
fn compile_native_staging_present33_f78_preserves_wrapper_abi() {
    let (beats, issues) =
        compile(vec![unknown("0x33f78", vec![json!(5), json!(23), json!(22)], src("0x33eb2", "0x33f78"))], &none());
    assert!(
        issues.is_empty()
            && beats.len() == 1
            && beats[0].op == "native_staging_present"
            && beats[0].native_staging_present.is_some(),
        "0x33f78 beats={beats:?} issues={issues:?}"
    );
    let got = beats[0].native_staging_present.unwrap();
    assert!(got.slot == 22 && got.x == 23 && got.y == 5, "0x33f78 payload={got:?}");
    let (_, issues) = compile(vec![unknown("0x33f78", vec![json!(5), json!(23)], HandlerSource::default())], &none());
    assert_eq!(issues.len(), 1, "malformed 0x33f78 issues={issues:?}");
}

/// Go：`TestCompileChapter29PreLowersEveryNativeStagingPresent`
#[test]
fn compile_chapter29_pre_lowers_every_native_staging_present() {
    let Some(beats) = bind_ok("ch29_pre.json") else {
        return;
    };
    let mut want: BTreeMap<&str, bool> = ["0x33ea4", "0x33eb2", "0x33ec0", "0x33ece", "0x33f45", "0x33f53", "0x33f61"]
        .into_iter()
        .map(|s| (s, false))
        .collect();
    let mut dialog_counts: BTreeMap<String, i64> = BTreeMap::new();
    let mut load: Option<LoadCHState> = None;
    for beat in &beats {
        if beat.op == "loadch" {
            load = beat.load_ch.clone();
        }
        if beat.op == "dialog" {
            *dialog_counts.entry(beat.source.clone()).or_default() += beat.count.max(1);
        }
        if let Some(seen) = want.get_mut(beat.source.as_str()) {
            assert!(
                beat.op == "native_staging_present" && beat.native_staging_present.is_some(),
                "ch29 source {}={beat:?}",
                beat.source
            );
            *seen = true;
        }
    }
    let missing: Vec<_> = want.iter().filter(|(_, seen)| !**seen).map(|(s, _)| *s).collect();
    assert!(missing.is_empty(), "ch29 staging sources {missing:?} were not lowered");
    assert!(
        load.as_ref().is_some_and(|s| s.chapter == 29
            && s.map == "assets/maps/map29"
            && s.slot_count == 70
            && s.script == "assets/story/ch30.json"),
        "ch29 pre loadch={load:?}"
    );
    let count = |s: &str| dialog_counts.get(s).copied().unwrap_or(0);
    assert!(
        count("0x33e80") == 10 && count("0x33ef5") == 5 && count("0x33f21") == 6,
        "ch29 pre dialogue counts={dialog_counts:?}"
    );
}

/// Go：`TestCompileChapterHandlersLowerEveryNativePalettePulse`
#[test]
fn compile_chapter_handlers_lower_every_native_palette_pulse() {
    let (Some(ch28), Some(ch29)) =
        (asset("cutscenes/handlers/ch28_post.json"), asset("cutscenes/handlers/ch29_pre.json"))
    else {
        return;
    };
    let mut want_sources: BTreeMap<&str, bool> =
        ["0x25682", "0x25694", "0x256a6", "0x33efd"].into_iter().map(|s| (s, false)).collect();
    for path in [ch28, ch29] {
        let script = load_handler_script(&path).unwrap();
        let (beats, issues) = compile_handler_script(Some(&script), &none());
        for issue in &issues {
            assert!(issue.source.target != "0x35e5a", "{path} left pulse unresolved: {issue:?}");
        }
        for beat in &beats {
            let Some(seen) = want_sources.get_mut(beat.source.as_str()) else {
                continue;
            };
            assert!(
                beat.op == "native_palette_pulse" && beat.native_palette_pulse.is_some(),
                "{path} source {}={beat:?}",
                beat.source
            );
            *seen = true;
        }
    }
    let missing: Vec<_> = want_sources.iter().filter(|(_, seen)| !**seen).map(|(s, _)| *s).collect();
    assert!(missing.is_empty(), "native palette pulse sources {missing:?} were not lowered");
}

/// Go：`TestCompileChapter26PostLowersEveryNativePaletteRamp`
#[test]
fn compile_chapter26_post_lowers_every_native_palette_ramp() {
    let Some(path) = asset("cutscenes/handlers/ch26_post.json") else {
        return;
    };
    let script = load_handler_script(&path).unwrap();
    let (beats, issues) = compile_handler_script(Some(&script), &none());
    for issue in &issues {
        assert!(
            issue.source.target != "0x25052" && issue.source.target != "0x1f882",
            "native palette operation remained unresolved: {issue:?}"
        );
    }
    let want_starts: BTreeMap<&str, i64> =
        [("0x25244", 5), ("0x25277", 4), ("0x25290", 3), ("0x252a9", 2), ("0x252bf", 2), ("0x252d5", 2)]
            .into_iter()
            .collect();
    let mut seen: BTreeMap<&str, Vec<Beat>> = BTreeMap::new();
    for beat in &beats {
        if let Some((key, _)) = want_starts.get_key_value(beat.source.as_str()) {
            seen.entry(key).or_default().push(beat.clone());
        }
    }
    for (source, &start) in &want_starts {
        let sequence = seen.get(source).cloned().unwrap_or_default();
        assert_eq!(
            sequence.len() as i64,
            2 * (start + 1),
            "{source} lowered beat count={} want {}: {sequence:?}",
            sequence.len(),
            2 * (start + 1)
        );
        for (i, delta) in (0..=start).rev().enumerate() {
            let (palette, delay) = (&sequence[i * 2], &sequence[i * 2 + 1]);
            assert!(
                palette.op == "palette_update"
                    && palette.palette_start == 0
                    && palette.palette_end == 255
                    && palette.palette_delta == delta
                    && delay.op == "delay"
                    && delay.ms == 80,
                "{source} ramp[{i}]={palette:?}/{delay:?}"
            );
        }
    }
    let fades: Vec<&Beat> = beats.iter().filter(|b| b.source == "0x2531a").collect();
    assert!(
        fades.len() == 1
            && fades[0].op == "native_palette_fade_out"
            && fades[0].native_palette_fade.is_some_and(|f| f.start == 0 && f.end == 63 && f.delay_ms == 2),
        "ch26 native palette fade={fades:?}"
    );
}

/// Go：`TestChapter27OpeningUsesOnlyOriginalDialogueCalls`
#[test]
fn chapter27_opening_uses_only_original_dialogue_calls() {
    let Some(beats) = bind_ok("ch26_pre.json") else {
        return;
    };
    assert!(
        !beats.is_empty() && beats[0].load_ch.as_ref().is_some_and(|s| s.chapter == 26 && s.slot_count == 80),
        "第27章LOADCH未保持原始80筆來源"
    );
    let want: BTreeMap<i64, i64> = [(0, 2), (3, 1), (4, 2), (5, 1), (6, 4), (7, 9)].into_iter().collect();
    let mut got: BTreeMap<i64, i64> = BTreeMap::new();
    fn visit(sequence: &[Beat], got: &mut BTreeMap<i64, i64>) {
        for beat in sequence {
            visit(&beat.then, got);
            visit(&beat.r#else, got);
            if beat.op != "dialog" {
                continue;
            }
            let layout = beat.native_dialogue.as_ref();
            assert!(layout.is_some_and(|l| l.source_dat == "FDTXT_027"), "揭幕對白缺原始控制碼：{beat:?}");
            *got.entry(layout.unwrap().string_index).or_default() += 1;
        }
    }
    visit(&beats, &mut got);
    assert_eq!(got, want, "揭幕對白混入死亡事件或漏句：got={got:?} want={want:?}");
}

/// Go：`TestChapter27PostBindingsPreserveOriginalDialogueAndTail`
#[test]
fn chapter27_post_bindings_preserve_original_dialogue_and_tail() {
    let want: BTreeMap<i64, i64> = [(8, 2), (9, 17), (10, 2), (11, 1), (12, 1)].into_iter().collect();
    let mut got: BTreeMap<i64, i64> = BTreeMap::new();
    let mut acts: BTreeSet<String> = BTreeSet::new();
    let mut tail: Vec<String> = Vec::new();
    for name in ["ch26_post_intro", "ch26_post_sky_key_success"] {
        let Some(beats) = bind_ok(&format!("{name}.json")) else {
            return;
        };
        for beat in &beats {
            if beat.op == "dialog" {
                let layout = beat.native_dialogue.as_ref();
                assert!(layout.is_some_and(|l| l.source_dat == "FDTXT_027"), "第27章戰後對白缺原始控制碼");
                *got.entry(layout.unwrap().string_index).or_default() += 1;
            }
            if beat.op == "act" {
                acts.insert(beat.source.clone());
            }
            if beat.op == "sync_party" || beat.op == "set_chapter" || beat.op == "reset_persistent_roster_state" {
                tail.push(beat.op.clone());
            }
        }
    }
    let want_acts: BTreeSet<String> = ["0x2517c", "0x251ca", "0x25211"].into_iter().map(String::from).collect();
    assert!(
        got == want && acts == want_acts && tail == vec!["sync_party", "set_chapter", "reset_persistent_roster_state"],
        "第27章戰後漏句、漏演出或尾端順序錯誤：dialog={got:?} acts={acts:?} tail={tail:?}"
    );
}

/// Go：`TestCompileChapter23PreUsesRecoveredChapter24TextGroups`
#[test]
fn compile_chapter23_pre_uses_recovered_chapter24_text_groups() {
    let Some(beats) = bind_ok("ch23_pre.json") else {
        return;
    };
    let (_, dialogs) = collect(&beats);
    assert_eq!(dialogs.len(), 14, "ch23_pre dialog beats={} want 14", dialogs.len());
    assert!(
        beats[0].op == "loadch" && beats[0].load_ch.as_ref().is_some_and(|s| s.chapter == 23 && s.slot_count == 70),
        "ch23_pre loadch={:?}",
        beats[0]
    );
    assert!(
        dialogs[0].script == "ch24.json" && dialogs[0].line == 0 && dialogs[0].scene_index == Some(0),
        "ch23_pre first dialog context={:?}",
        dialogs[0]
    );
    assert!(
        dialogs[5].line == 5 && dialogs[13].line == 13,
        "ch23_pre second text group boundaries={:?}/{:?}",
        dialogs[5],
        dialogs[13]
    );
}

/// Go：`TestCompileChapter24PreUsesTransitionAndFDOther88SFX`
#[test]
fn compile_chapter24_pre_uses_transition_and_fd_other88_sfx() {
    let Some(beats) = bind_ok("ch24_pre.json") else {
        return;
    };
    let transitions: Vec<&Beat> = beats.iter().filter(|b| b.op == "transition_reveal").collect();
    let sfx: Vec<&Beat> = beats.iter().filter(|b| b.op == "play_sfx").collect();
    assert!(
        transitions.len() == 4 && transitions[0].reveal_frames == 20 && transitions[3].reveal_frames == 60,
        "ch24_pre transitions={transitions:?}"
    );
    assert!(sfx.len() == 5 && sfx[0].resource_id == Some(88) && sfx[0].sfx_index == Some(1), "ch24_pre sfx={sfx:?}");
    assert!(sfx[4].sfx_index == Some(-1), "ch24_pre stop sfx={:?}", sfx[4]);
}

/// Go：`TestCompileChapter25PreUsesDirectFDTXT026StringZeroAlignment`
#[test]
fn compile_chapter25_pre_uses_direct_fdtxt026_string_zero_alignment() {
    let Some(beats) = bind_ok("ch25_pre.json") else {
        return;
    };
    let mut dialog = Beat::default();
    let mut act = Beat::default();
    for beat in &beats {
        if beat.op == "dialog" {
            dialog = beat.clone();
        }
        if beat.op == "act" {
            act = beat.clone();
        }
    }
    assert!(
        dialog.script == "ch26.json" && dialog.scene_index == Some(0) && dialog.line == 0 && dialog.count == 12,
        "ch25_pre dialog={dialog:?}"
    );
    assert!(!act.acting.is_empty(), "ch25_pre acting={act:?}");
    assert!(
        beats[0].load_ch.as_ref().is_some_and(|s| s.chapter == 25 && s.slot_count == 70),
        "ch25_pre loadch={:?}",
        beats[0]
    );
}

/// Go：`TestCompileChapter27PostMapsFDTXT028StringSeven`
#[test]
fn compile_chapter27_post_maps_fdtxt028_string_seven() {
    let Some(beats) = bind_ok("ch27_post.json") else {
        return;
    };
    let (_, dialogs) = collect(&beats);
    // FDTXT_028 的第 7 句在 ch28.json 對到 5 行，而 binding 的
    // dialogue_overrides 為每一行各自帶了 native_dialogue（utterance 0..4、
    // 不同的 control operand 與分頁）。因此編譯結果是五個 dialog 拍，不是一個
    // count=5 的群組——折成群組會丟掉逐行的原生版面。
    assert_eq!(dialogs.len(), 5, "ch27_post dialogs={}, want 5：{dialogs:?}", dialogs.len());
    for (i, beat) in dialogs.iter().enumerate() {
        let i = i as i64;
        assert!(
            beat.script == "ch28.json" && beat.scene_index == Some(1) && beat.line == 11 + i && beat.count == 0,
            "ch27_post dialog[{i}]={beat:?}，want line={} scene1 count=0",
            11 + i
        );
        assert!(
            beat.native_dialogue
                .as_ref()
                .is_some_and(|l| l.source_dat == "FDTXT_028" && l.string_index == 7 && l.utterance == i),
            "ch27_post dialog[{i}] 原生版面={:?}，want FDTXT_028#7 utterance={i}",
            beat.native_dialogue
        );
    }
}

/// Go：`TestCompileChapter27PreUsesNativeStagingPushOrder`
#[test]
fn compile_chapter27_pre_uses_native_staging_push_order() {
    let Some(path) = asset("cutscenes/handlers/ch27_pre.json") else {
        return;
    };
    let script = load_handler_script(&path).unwrap();
    let (beats, issues) = compile_handler_script(Some(&script), &none());
    for issue in &issues {
        assert!(
            issue.source.addr != "0x33d95" && issue.source.addr != "0x33da3",
            "ch27_pre staging remained unresolved: {issue:?}"
        );
    }
    for (source, (x, y, group)) in [("0x33d95", (0, 384, 6)), ("0x33da3", (168, 384, 7))] {
        let staging: Vec<&Beat> = beats.iter().filter(|b| b.source == source).collect();
        assert!(
            staging.len() == 7
                && staging[0].op == "pan"
                && staging[0].x == x
                && staging[0].y == y
                && staging[1].op == "spawn"
                && staging[1].group == group,
            "ch27_pre {source} staging={staging:?}"
        );
    }
}

/// Go：`TestCompileChapter27PreBindingClosesExactLateGameOwner`
#[test]
fn compile_chapter27_pre_binding_closes_exact_late_game_owner() {
    let Some(beats) = bind_ok("ch27_pre.json") else {
        return;
    };
    assert!(
        !beats.is_empty()
            && beats[0].op == "runtime_context"
            && beats[0].runtime_context.as_ref().is_some_and(|c| c.slot_count == 60 && c.story_viewport),
        "ch27_pre runtime context={beats:?}"
    );
    let mut deactivations = 0;
    let mut transition = Beat::default();
    let mut reactivation = Beat::default();
    for beat in &beats {
        match beat.op.as_str() {
            "deactivate_unit" => deactivations += 1,
            "indexed_transition" => transition = beat.clone(),
            "reactivate_nonzero_hp" => reactivation = beat.clone(),
            _ => {}
        }
    }
    assert!(
        deactivations == 20 && reactivation.source == "0x33cea" && reactivation.count == 20,
        "ch27_pre deactivate/reactivate={deactivations}/{reactivation:?}"
    );
    assert!(
        transition.source == "0x33ce2"
            && transition.indexed_transition.as_ref().is_some_and(|t| t.cursor_source == "native_relative_cursor"
                && t.cursor_x_offset == 6
                && t.cursor_y_offset == 5),
        "ch27_pre indexed transition={transition:?}"
    );
}

/// Go：`TestCompileChapter23PrePreservesCountedReactivationBeforeRedraw`
#[test]
fn compile_chapter23_pre_preserves_counted_reactivation_before_redraw() {
    let Some(beats) = bind_ok("ch22_pre.json") else {
        return;
    };
    for (i, beat) in beats.iter().enumerate() {
        if beat.op == "reactivate_nonzero_hp" {
            assert!(
                beat.source == "0x336ed"
                    && beat.count == 16
                    && beat.dir == Some(2)
                    && i + 1 < beats.len()
                    && beats[i + 1].source == "0x3371c",
                "非零HP復位契約或位置失真：{beat:?}"
            );
            return;
        }
    }
    panic!("binding遺漏非零HP復位");
}

/// Go：`TestCompileChapter28PreLowersStagingHelper`
#[test]
fn compile_chapter28_pre_lowers_staging_helper() {
    let Some(beats) = bind_ok("ch28_pre.json") else {
        return;
    };
    assert!(
        !beats.is_empty()
            && beats[0].op == "runtime_context"
            && beats[0].runtime_context.as_ref().is_some_and(|c| c.slot_count == 76 && c.story_viewport),
        "ch28_pre runtime context={beats:?}"
    );
    let load = beats.iter().find(|b| b.op == "loadch").cloned().unwrap_or_default();
    assert!(
        load.load_ch.as_ref().is_some_and(|s| s.chapter == 28
            && s.map == "assets/maps/map28"
            && s.roster == "assets/maps/map28/map28_units.json"
            && s.slot_count == 76
            && s.party_scenario == "assets/scenarios/ch29.json"),
        "ch28_pre loadch owner={:?}",
        load.load_ch
    );
    let staging: Vec<&Beat> = beats.iter().filter(|b| b.source == "0x33e16").collect();
    assert_eq!(
        staging.len(),
        7,
        "ch28_pre staging beats={} want pan/spawn/delay/palette/delay/palette/redraw",
        staging.len()
    );
    // Source PUSH order at 0x33e16 is group=8,y=19,x=9.
    assert!(
        staging[0].op == "pan"
            && staging[0].x == 216
            && staging[0].y == 456
            && staging[1].op == "spawn"
            && staging[1].group == 8,
        "ch28_pre staging front={:?}",
        &staging[..2]
    );
    assert!(
        staging[2].op == "delay"
            && staging[2].ms == 300
            && staging[3].op == "palette_update"
            && staging[4].op == "delay"
            && staging[4].ms == 200
            && staging[5].op == "palette_update"
            && staging[6].op == "redraw",
        "ch28_pre staging timing={staging:?}"
    );
    assert!(
        staging[3].palette_start == 0
            && staging[3].palette_end == 255
            && staging[3].palette_delta == 255
            && staging[5].palette_start == 0
            && staging[5].palette_end == 255
            && staging[5].palette_delta == 0,
        "ch28_pre staging palette={:?}/{:?}",
        staging[3],
        staging[5]
    );
}

/// Go：`TestCompileChapter29PostPreservesDialogueAcrossChapterTextSwitch`
#[test]
fn compile_chapter29_post_preserves_dialogue_across_chapter_text_switch() {
    let Some(path) = asset("cutscenes/bindings/ch29_post.json") else {
        return;
    };
    let (beats, issues) = compile_handler_binding(&path).unwrap();
    assert!(
        issues.is_empty() && !beats.is_empty() && beats[beats.len() - 1].op == "ending_handoff",
        "ch29_post issues={issues:?}，完整戰後必須接已證實的ending owner"
    );
    let first = |source: &str| beats.iter().find(|b| b.source == source).cloned().unwrap_or_default();
    let layout = first("0x257b4");
    assert!(
        layout.op == "layout_units"
            && layout.layout.as_ref().is_some_and(|l| l.units.len() == 20
                && l.units[0] == HandlerUnitLayout { slot: 0, x: 22, y: 23, pose: 2 }
                && l.units[1] == HandlerUnitLayout { slot: 1, x: 22, y: 19, pose: 0 }
                && l.units[19] == HandlerUnitLayout { slot: 19, x: 24, y: 25, pose: 2 }
                && l.cam_x == 384
                && l.cam_y == 432),
        "ch29_post native layout={layout:?}"
    );
    let loader = first("0x25870");
    assert!(
        loader.op == "loadch"
            && loader.load_ch.as_ref().is_some_and(|s| s.chapter == 30
                && s.map == "assets/maps/map30"
                && s.party_scenario == "assets/scenarios/ch30.json"
                && s.all_persistent_party
                && s.slot_count == 31
                && s.source_slot_count == Some(1)),
        "ch29_post full chapter loader={loader:?}"
    );
    let transition = first("0x25848");
    assert!(
        transition.op == "indexed_transition"
            && transition
                .indexed_transition
                .as_ref()
                .is_some_and(|t| t.tile_x == 6 && t.tile_y == 6 && t.radial_radius == 10 && t.radial_radius_step == 8),
        "ch29_post indexed transition={transition:?}"
    );
    let groups = [
        ("ch30.json", 1, 15, 2),
        ("ch30.json", 2, 0, 6),
        ("ch30.json", 3, 0, 18),
        ("ch31.json", 0, 0, 13),
        ("ch31.json", 0, 13, 3),
    ];
    let mut want: Vec<(&str, i64, i64, i64)> = Vec::new();
    for (script, scene, line, count) in groups {
        for l in line..line + count {
            want.push((script, scene, l, 0));
        }
    }
    let (_, dialogs) = collect(&beats);
    assert_eq!(dialogs.len(), want.len(), "ch29_post逐句數={}，預期{}", dialogs.len(), want.len());
    for (i, w) in want.iter().enumerate() {
        let d = &dialogs[i];
        assert!(
            d.script == w.0 && d.scene_index == Some(w.1) && d.line == w.2 && d.count == w.3,
            "ch29_post dialog[{i}]={d:?} want {w:?}"
        );
        assert!(d.native_dialogue.is_some(), "第{i}句缺原始controls／glyphs來源");
    }
    let focus = first("0x2582e");
    assert!(
        focus.op == "pan" && focus.x == 22 * 24 && focus.y == 23 * 24 && focus.tile_step,
        "ch29 focus lower={focus:?}"
    );
    let final_pan = first("0x25937");
    assert!(
        final_pan.op == "pan" && final_pan.x == 11 * 24 && final_pan.y == 12 * 24 && final_pan.tile_step,
        "ch29 final pan lower={final_pan:?}"
    );
}

/// Go：`TestCompileChapter21PostCandidateFailsClosedBelowMinimumRuntimeFrontier`
#[test]
fn compile_chapter21_post_candidate_fails_closed_below_minimum_runtime_frontier() {
    let Some(path) = asset("cutscenes/bindings/ch21_post_candidate.json") else {
        return;
    };
    let (beats, issues) = compile_handler_binding(&path).unwrap();
    assert!(
        issues.len() == 1 && issues[0].op == "layout_units" && issues[0].source.addr == "0x24512",
        "ch21_post candidate must reject slot72 at minimum frontier 66: issues={issues:?}"
    );
    for beat in &beats {
        assert!(
            beat.op != "runtime_context" && beat.op != "layout_units",
            "ch21_post unresolved candidate emitted runnable context/layout: beat={beat:?}"
        );
    }
    let (mut act65, mut act66, mut transition) = (Beat::default(), Beat::default(), Beat::default());
    for beat in &beats {
        match beat.source.as_str() {
            "0x24543" => act65 = beat.clone(),
            "0x24580" => act66 = beat.clone(),
            "0x245ce" => transition = beat.clone(),
            _ => {}
        }
    }
    assert!(
        act65.op == "act"
            && act65.acting.len() == 2
            && act65.acting[0].units.len() == 1
            && act65.acting[0].units[0].slot == Some(1),
        "ch21_post acting65={act65:?}"
    );
    assert!(act66.op == "act" && act66.acting.len() == 50, "ch21_post acting66 frame count={}", act66.acting.len());
    let found_slot72 = act66.acting.iter().flat_map(|f| &f.units).any(|u| u.slot == Some(72));
    assert!(found_slot72, "ch21_post acting66 does not preserve special slot72");
    assert!(
        transition.op == "indexed_transition"
            && transition.indexed_transition.as_ref().is_some_and(|t| t.cursor_source == "native_relative_cursor"
                && t.cursor_y_offset == 3
                && t.frames == 9
                && t.frame_delay_ms == 5
                && t.tail_delay_ms == 500
                && t.palette_delta_end == 62
                && t.palette_delay_ms == 4),
        "ch21_post indexed transition={transition:?}"
    );
    let found_fade = beats.iter().any(|b| b.op == "native_palette_fade_out" && b.source == "0x245fa");
    assert!(found_fade, "ch21_post 0x1f882 palette fade was not lowered");
}

/// Go：`TestCompileChapter23PostProductionPreservesRawLoopSchedule`
#[test]
fn compile_chapter23_post_production_preserves_raw_loop_schedule() {
    let Some(beats) = bind_ok("ch23_post.json") else {
        return;
    };
    let loops: Vec<&Beat> = beats.iter().filter(|b| b.op == "native_ch23_loop").collect();
    assert!(
        loops.len() == 2 && loops[0].source == "0x24c61" && loops[1].source == "0x24cc1",
        "ch23_post raw loops={loops:?}"
    );
    let initial = loops[0].native_ch23_loop.as_ref();
    assert!(
        initial.is_some_and(|l| l.phase == "initial"
            && l.repeat == 30
            && native_stage_values(&l.stage_values, 2, 9)
            && l.palette.is_none()
            && native_call_matches(Some(&l.draw), "0x24c61", "0x11cac", &[json!(1)])
            && native_call_matches(Some(&l.tick), "0x24c6b", "0x17aa9", &[json!(1)])),
        "ch23_post initial loop={initial:?}"
    );
    let palette = loops[1].native_ch23_loop.as_ref();
    assert!(
        palette.is_some_and(|l| l.phase == "palette"
            && l.repeat == 12
            && native_stage_values(&l.stage_values, 10, 14)
            && l.palette_table_source == "0x60003"
            && native_call_matches(l.palette.as_ref(), "0x24cc1", "0x11d40", &[json!("esi"), json!(255), json!(0)])
            && native_call_matches(Some(&l.draw), "0x24cd3", "0x11cac", &[json!(0)])
            && native_call_matches(Some(&l.tick), "0x24cdd", "0x17aa9", &[json!(1)])),
        "ch23_post palette loop={palette:?}"
    );
    let initial = initial.unwrap();
    assert_eq!(initial.stage.source.addr, "0x24c81", "ch23 initial stage source={}", initial.stage.source.addr);
    let got = &palette.unwrap().stage.source.addr;
    assert_eq!(got, "0x24cf2", "ch23 palette stage source={got}");
    let mut bad = initial.clone();
    bad.stage_values[0] += 1;
    let (_, bad_issues) = compile(
        vec![HandlerBeat {
            op: "native_ch23_loop".into(),
            source: src("0x24c61", "0x11cac"),
            native_ch23_loop: Some(bad),
            ..Default::default()
        }],
        &none(),
    );
    assert_eq!(bad_issues.len(), 1, "mutated ch23 loop must fail closed: {bad_issues:?}");
}

/// Go：`TestLoadChapter22PostPreservesNative2189ALoops`
#[test]
fn load_chapter22_post_preserves_native2189_a_loops() {
    let Some(path) = asset("cutscenes/handlers/ch22_post.json") else {
        return;
    };
    let script = load_handler_script(&path).unwrap();
    let loops: Vec<&HandlerBeat> = script.beats.iter().filter(|b| b.op == "native_2189a_loop").collect();
    assert_eq!(loops.len(), 3, "ch22_post native 0x2189a loops={}, want 3", loops.len());
    for (i, beat) in loops.iter().enumerate() {
        let l = beat.native2189_a_loop.as_ref();
        assert!(
            beat.native_target == "0x2189a"
                && l.is_some_and(|l| l.repeat == 10
                    && l.work_offset == 0x8088
                    && l.work_stride == 456
                    && l.map_rows == 13
                    && l.map_columns == 8
                    && l.clip_width == 312
                    && l.clip_height == 192
                    && l.present_stride == 320),
            "loop[{i}] payload={l:?}"
        );
        let l = l.unwrap();
        assert!(
            l.step_source == "caller_arg9"
                && l.map_draw.source.addr == "0x21914"
                && l.composite.source.addr == "0x21955"
                && l.present.source.addr == "0x21986",
            "loop[{i}] raw call chain={l:?}"
        );
    }
    assert!(
        loops[0].source.addr == "0x24978" && loops[1].source.addr == "0x249c4" && loops[2].source.addr == "0x24a10",
        "ch22_post loop sources={loops:?}"
    );
    assert!(
        script.beats.len() >= 3
            && script.beats[1].op == "if"
            && script.beats[1]
                .condition
                .as_ref()
                .is_some_and(|c| c.op == "native_inventory_item_present" && c.native_inventory_item_id == Some(100)),
        "ch22_post inventory branch={:?}",
        script.beats[1]
    );
    assert!(
        script.beats[2].op == "if"
            && script.beats[2].condition.as_ref().is_some_and(
                |c| c.op == "native_persistent_identity_present" && c.native_persistent_identity == Some(18)
            ),
        "ch22_post persistent branch={:?}",
        script.beats[2]
    );
    let nested = &script.beats[2].r#else;
    assert!(
        nested.len() == 1
            && nested[0].op == "if"
            && nested[0].condition.as_ref().is_some_and(|c| c.op == "native_round_lt" && c.native_round == Some(15)),
        "ch22_post round branch={nested:?}"
    );
}

/// Go：`TestCompileCompleteChapter22PostBindingPreservesLayoutResourcesAndPaletteLoop`
#[test]
fn compile_complete_chapter22_post_binding_preserves_layout_resources_and_palette_loop() {
    let Some(beats) = bind_ok("ch22_post.json") else {
        return;
    };
    assert!(
        !beats.is_empty()
            && beats[0].op == "runtime_context"
            && beats[0]
                .runtime_context
                .as_ref()
                .is_some_and(|c| c.slot_count == 0 && c.slot_counts == vec![42, 62, 86] && c.story_viewport),
        "ch22_post runtime context={beats:?}"
    );
    let context = beats[0].runtime_context.as_ref().unwrap();
    for count in [0, 18, 41, 43, 54, 61, 63, 85, 87, 96] {
        assert!(!context.accepts_slot_count(count), "ch22_post accepted unverified runtime frontier {count}");
    }
    #[derive(Default)]
    struct Visit {
        layout: Option<HandlerLayout>,
        palette_starts: Vec<i64>,
        palette_delays: i64,
        resources: BTreeMap<String, Beat>,
        acts: BTreeMap<String, Beat>,
    }
    fn visit_beats(compiled: &[Beat], v: &mut Visit) {
        for beat in compiled {
            if beat.op == "layout_units" {
                v.layout = beat.layout.clone();
            } else if beat.op == "palette_update" && beat.source == "0x24a24" {
                v.palette_starts.push(beat.palette_start);
            } else if beat.op == "delay" && beat.source == "0x24a2e" {
                v.palette_delays += 1;
            } else if beat.op == "load_res" {
                v.resources.insert(beat.source.clone(), beat.clone());
            } else if beat.op == "act" {
                v.acts.insert(beat.source.clone(), beat.clone());
            }
            visit_beats(&beat.then, v);
            visit_beats(&beat.r#else, v);
        }
    }
    let mut v = Visit::default();
    visit_beats(&beats, &mut v);
    assert!(
        v.layout.as_ref().is_some_and(|l| l.units.len() == 18
            && l.cam_x == 336
            && l.cam_y == 336
            && l.units[16] == HandlerUnitLayout { slot: 16, x: 19, y: 21, pose: 0 }
            && l.units[17] == HandlerUnitLayout { slot: 17, x: 21, y: 21, pose: 2 }),
        "ch22_post layout={:?}",
        v.layout
    );
    assert!(
        v.palette_starts.len() == 32
            && v.palette_delays == 32
            && v.palette_starts[0] == 0
            && v.palette_starts[31] == 0x3e,
        "ch22_post palette starts={:?} delays={}",
        v.palette_starts,
        v.palette_delays
    );
    for (source, (id, archive, owner)) in [
        ("0x24a4b", (69, "FDFIELD.DAT", "0x53a51")),
        ("0x24a65", (46, "FDSHAP.DAT", "0x53a5d")),
        ("0x24a7f", (47, "FDSHAP.DAT", "0x53a69")),
    ] {
        let got = v.resources.get(source);
        assert!(
            got.is_some_and(|b| b.resource_id == Some(id)
                && b.resource_archive == archive
                && b.resource_owner == owner),
            "resource {source}={got:?} want={id}/{archive}/{owner}"
        );
    }
    for source in ["0x2482e", "0x24877", "0x248f1", "0x24abe", "0x24ad4", "0x24ade"] {
        assert!(v.acts.contains_key(source), "missing compiled acting source {source}");
    }
}

/// Go 測試的 0x2189a 參數組。
fn native2189a_loop() -> Native2189ALoop {
    let call = |a: &str, t: &str, raw_args: Vec<Value>| HandlerNativeCall { source: src(a, t), raw_args };
    Native2189ALoop {
        repeat: 10,
        step_source: "caller_arg9".into(),
        work_offset: 0x8088,
        work_stride: 456,
        map_rows: 13,
        map_columns: 8,
        clip_width: 312,
        clip_height: 192,
        present_stride: 320,
        map_draw: call(
            "0x21914",
            "0x11eee",
            vec![json!("[0x53aad]"), json!("[0x53aa9]"), json!(8), json!(13), json!(456), json!("work+0x8088")],
        ),
        composite: call(
            "0x21955",
            "0x219ad",
            vec![json!("edi"), json!(192), json!(0), json!(12), json!("esi"), json!("[esp+0x18]"), json!("[esp+0x18]")],
        ),
        stage: call("0x2195d", "0x127a9", Vec::new()),
        present: call(
            "0x21986",
            "0x11eb0",
            vec![json!(192), json!(456), json!(312), json!("work+0x8088"), json!(320), json!(656644)],
        ),
        tail: call("0x219a3", "0x11cac", vec![json!(0)]),
        ..Default::default()
    }
}

/// Go：`TestCompileNative2189ALoopPreservesRawCallShapeAndFailsClosed`
#[test]
fn compile_native2189_a_loop_preserves_raw_call_shape_and_fails_closed() {
    let r#loop = native2189a_loop();
    let input = |l: Native2189ALoop| HandlerBeat {
        op: "native_2189a_loop".into(),
        native_target: "0x2189a".into(),
        raw_args: vec![json!(10), json!(15), json!(1)],
        source: src("0x24978", "0x2189a"),
        native2189_a_loop: Some(l),
        ..Default::default()
    };
    let (beats, issues) = compile(vec![input(r#loop.clone())], &none());
    assert!(
        issues.is_empty()
            && beats.len() == 1
            && beats[0].op == "native_2189a_loop"
            && beats[0].native2189_a_loop.is_some(),
        "compiled loop={beats:?} issues={issues:?}"
    );
    let got = beats[0].native2189_a_loop.as_ref().unwrap();
    assert_eq!(got.present.raw_args[5], json!(656644), "present raw target={:?}, want 656644", got.present.raw_args[5]);
    assert!(
        got.slot == 10 && got.initial_radius == 15 && got.radius_step == 1,
        "compiled outer ABI={got:?}, want slot10/radius15/step1"
    );
    let mut bad = r#loop;
    bad.repeat = 9;
    let (_, bad_issues) = compile(vec![input(bad)], &none());
    assert_eq!(bad_issues.len(), 1, "mutated 0x2189a loop must fail closed: {bad_issues:?}");
}

/// Go：`TestCompileChapter22PostRawBranchConditions`
#[test]
fn compile_chapter22_post_raw_branch_conditions() {
    let input = vec![
        HandlerBeat {
            op: "if".into(),
            source: addr("0x247c6"),
            native_target: "0x24b14".into(),
            raw_args: vec![json!(100)],
            condition: Some(HandlerCondition {
                op: "native_inventory_item_present".into(),
                native_inventory_item_id: ip(100),
                ..Default::default()
            }),
            ..Default::default()
        },
        HandlerBeat {
            op: "if".into(),
            source: addr("0x24840"),
            native_target: "0x24bde".into(),
            raw_args: vec![json!(18)],
            condition: Some(HandlerCondition {
                op: "native_persistent_identity_present".into(),
                native_persistent_identity: ip(18),
                ..Default::default()
            }),
            ..Default::default()
        },
        HandlerBeat {
            op: "if".into(),
            source: src("0x248b5", "0x53bef"),
            raw_args: vec![json!(15)],
            condition: Some(HandlerCondition {
                op: "native_round_lt".into(),
                native_round: ip(15),
                ..Default::default()
            }),
            ..Default::default()
        },
    ];
    let (beats, issues) = compile(input, &none());
    assert!(issues.is_empty() && beats.len() == 3, "compiled ch22 conditions={beats:?} issues={issues:?}");
    let c = |i: usize| beats[i].condition.clone();
    assert!(
        c(0).is_some_and(|c| c.op == "native_inventory_item_present" && c.native_inventory_item_id == Some(100)),
        "inventory condition={:?}",
        c(0)
    );
    assert!(
        c(1).is_some_and(|c| c.op == "native_persistent_identity_present" && c.native_persistent_identity == Some(18)),
        "persistent condition={:?}",
        c(1)
    );
    assert!(
        c(2).is_some_and(|c| c.op == "native_round_lt" && c.native_round == Some(15)),
        "round condition={:?}",
        c(2)
    );
    for (name, condition) in [
        (
            "item out of range",
            HandlerCondition {
                op: "native_inventory_item_present".into(),
                native_inventory_item_id: ip(256),
                ..Default::default()
            },
        ),
        (
            "identity out of range",
            HandlerCondition {
                op: "native_persistent_identity_present".into(),
                native_persistent_identity: ip(-1),
                ..Default::default()
            },
        ),
        ("round missing threshold", HandlerCondition { op: "native_round_lt".into(), ..Default::default() }),
    ] {
        let (_, bad_issues) =
            compile(vec![HandlerBeat { op: "if".into(), condition: Some(condition), ..Default::default() }], &none());
        assert_eq!(bad_issues.len(), 1, "{name}: condition must fail closed: {bad_issues:?}");
    }
}

/// Go：`TestCompileUnitPresentRejectsFormerlyAcceptedBinding`
#[test]
fn compile_unit_present_rejects_formerly_accepted_binding() {
    let p = HandlerUnitPresent { slot: 18, x: 22, y: 24, frames: 6, frame_delay_ms: 10, tail_ticks: 2 };
    let (beats, issues) = compile(
        vec![HandlerBeat {
            op: "unit_present".into(),
            source: src("0x33ea4", "0x22253"),
            unit_present: Some(p),
            ..Default::default()
        }],
        &ctx(19),
    );
    assert!(
        beats.is_empty() && issues.len() == 1 && issues[0].op == "unit_present",
        "unit_present lowering={beats:?} issues={issues:?}"
    );
}

/// Go：`TestCompileUnitPresentFailsClosedOutsideProvenShape`
#[test]
fn compile_unit_present_fails_closed_outside_proven_shape() {
    let base_present = HandlerUnitPresent { slot: 0, x: 1, y: 1, frames: 6, frame_delay_ms: 10, tail_ticks: 2 };
    let base = HandlerBeat { op: "unit_present".into(), unit_present: Some(base_present), ..Default::default() };
    let mut wrong_timing = base.clone();
    wrong_timing.unit_present = Some(HandlerUnitPresent { frames: 5, ..base_present });
    let mut outside = base.clone();
    outside.unit_present = Some(HandlerUnitPresent { slot: 2, ..base_present });
    for (name, beat) in
        [("no runtime context", base), ("wrong timing", wrong_timing), ("slot outside context", outside)]
    {
        let bindings = if name != "no runtime context" { ctx(2) } else { none() };
        let (beats, issues) = compile(vec![beat], &bindings);
        assert!(
            beats.is_empty() && issues.len() == 1 && issues[0].op == "unit_present",
            "{name}: beats={beats:?} issues={issues:?}"
        );
    }
}

/// Go：`TestCompileChapter7PostUsesEvent25RefinedSlotFrontier`
#[test]
fn compile_chapter7_post_uses_event25_refined_slot_frontier() {
    let Some(beats) = bind_ok("ch06_post.json") else {
        return;
    };
    assert!(
        beats.len() == 4
            && beats[0].op == "runtime_context"
            && beats[0].runtime_context.as_ref().is_some_and(|c| c.slot_counts == vec![34, 44])
            && beats[1].op == "sync_party"
            && beats[2].op == "if"
            && beats[3].op == "set_chapter",
        "ch06_post beats={beats:?}"
    );
    let outer = &beats[2];
    assert!(
        outer.condition.as_ref().is_some_and(|c| c.op == "native_event_state_eq"
            && c.event_state_index == Some(17)
            && c.event_state_value == Some(1)
            && c.required_slot_count == Some(44))
            && outer.then.len() == 1
            && outer.r#else.len() == 4,
        "ch06_post outer branch={outer:?}"
    );
    let inner = &outer.then[0];
    let ie = &inner.r#else;
    assert!(
        inner.op == "if"
            && inner.condition.as_ref().is_some_and(|c| c.op == "any_unit_inactive" && c.unit_slots == vec![43])
            && inner.then.len() == 4
            && inner.then[0].op == "dialog"
            && ie.len() == 13
            && ie[0].op == "layout_units"
            && ie[0].layout.as_ref().is_some_and(
                |l| l.units.len() == 10 && l.units[9] == HandlerUnitLayout { slot: 43, x: 12, y: 7, pose: 2 }
            )
            && ie[ie.len() - 1].op == "join"
            && ie[ie.len() - 1].char_id == 12
            && outer.r#else.len() == 4
            && outer.r#else[0].op == "dialog",
        "ch06_post inner branch={inner:?}"
    );
}

/// Go：`TestCompileChapter8PostBindingPreservesRawCh07Sequence`
#[test]
fn compile_chapter8_post_binding_preserves_raw_ch07_sequence() {
    let Some(beats) = bind_ok("ch07_post.json") else {
        return;
    };
    assert!(
        beats.len() == 19
            && beats[0].op == "runtime_context"
            && beats[0]
                .runtime_context
                .as_ref()
                .is_some_and(|c| c.slot_counts == vec![29, 31, 33, 35, 37, 39, 41] && c.story_viewport),
        "ch07_post beats/context={beats:?}"
    );
    let mut by_source: BTreeMap<String, Vec<Beat>> = BTreeMap::new();
    for beat in &beats {
        by_source.entry(beat.source.clone()).or_default().push(beat.clone());
    }
    let get = |s: &str| by_source.get(s).cloned().unwrap_or_default();
    let layout = get("0x234fe").into_iter().rfind(|b| b.op == "layout_units").unwrap_or_default();
    assert!(
        layout.layout.as_ref().is_some_and(|l| l.units.len() == 11
            && l.cam_x == 192
            && l.cam_y == 336
            && l.units[10] == HandlerUnitLayout { slot: 28, x: 14, y: 16, pose: 0 }),
        "ch07_post layout={:?}",
        get("0x234fe")
    );
    let acting = get("0x23539");
    assert!(
        acting.len() == 1
            && acting[0].op == "act"
            && acting[0].acting.len() == 4
            && acting[0].acting[0].units[1].slot == Some(28),
        "ch07_post ACT33={acting:?}"
    );
    let acting = get("0x2357e");
    assert!(acting.len() == 1 && acting[0].op == "act" && acting[0].acting.len() == 1, "ch07_post ACT34={acting:?}");
    let blackout = get("0x23599");
    assert!(
        blackout.len() == 1
            && blackout[0].op == "native_palette_blackout"
            && blackout[0]
                .native_palette_blackout
                .is_some_and(|b| b.start == 0 && b.end == 255 && b.delta == 64 && b.clear_bytes == 0xFA00),
        "ch07_post blackout={blackout:?}"
    );
    let tail = &beats[beats.len() - 3..];
    assert!(
        tail[0].op == "join"
            && tail[0].char_id == 5
            && tail[1].op == "sync_party"
            && tail[2].op == "set_chapter"
            && tail[2].chapter == Some(8),
        "ch07_post tail={tail:?}"
    );
}

/// Go：`TestCompileCh07PostBlackoutRequiresExactCallSiteAndArguments`
#[test]
fn compile_ch07_post_blackout_requires_exact_call_site_and_arguments() {
    let exact = unknown("0x11d40", vec![json!(64.0), json!(255.0), json!(0.0)], src("0x23599", "0x11d40"));
    let (beats, issues) = compile(vec![exact.clone()], &none());
    assert!(
        issues.is_empty() && beats.len() == 1 && beats[0].op == "native_palette_blackout",
        "exact blackout beats={beats:?} issues={issues:?}"
    );
    let mutations: [(&str, fn(&mut HandlerBeat)); 4] = [
        ("different source", |beat| beat.source.addr = "0x23598".into()),
        ("different target", |beat| beat.source.target = "0x11df2".into()),
        ("different delta", |beat| beat.raw_args[0] = json!(63.0)),
        ("partial range", |beat| beat.raw_args[1] = json!(254.0)),
    ];
    for (name, mutate) in mutations {
        let mut candidate = exact.clone();
        mutate(&mut candidate);
        let (beats, issues) = compile(vec![candidate], &none());
        assert!(beats.is_empty() && issues.len() == 1, "{name}: beats={beats:?} issues={issues:?}");
    }
}

/// Go：`TestCompileCh20SkyKeySequenceRequiresExactCallSite`
#[test]
fn compile_ch20_sky_key_sequence_requires_exact_call_site() {
    let exact = HandlerBeat {
        op: "native_call".into(),
        native_target: "0x24336".into(),
        native_semantic: "ch20 天空之鑰固定合成演出序列".into(),
        native_confidence: "已證實".into(),
        native_evidence: vec!["docs/data/ida/fd2_ch20_sky_key_sequence_ida.txt".into()],
        source: src("0x242c9", "0x24336"),
        ..Default::default()
    };
    let (beats, issues) = compile(vec![exact.clone()], &none());
    assert!(
        issues.is_empty()
            && beats.len() == 1
            && beats[0].op == "native_ch20_sky_key_sequence"
            && beats[0].source == "0x242c9"
            && beats[0].native_ch20_sky_key.is_some_and(|s| s.is_recovered_contract()),
        "exact sky-key sequence beats={beats:?} issues={issues:?}"
    );
    let mutations: [(&str, fn(&mut HandlerBeat)); 3] = [
        ("different source", |beat| beat.source.addr = "0x242ca".into()),
        ("different target", |beat| beat.source.target = "0x24337".into()),
        ("unexpected arg", |beat| beat.raw_args = vec![json!(0.0)]),
    ];
    for (name, mutate) in mutations {
        let mut candidate = exact.clone();
        mutate(&mut candidate);
        let (beats, issues) = compile(vec![candidate], &none());
        assert!(beats.is_empty() && issues.len() == 1, "{name}: beats={beats:?} issues={issues:?}");
    }
}

/// Go：`TestNativeEventStateEqualsRejectsFrontierAbsentFromBinding`
#[test]
fn native_event_state_equals_rejects_frontier_absent_from_binding() {
    let input = HandlerBeat {
        op: "if".into(),
        condition: Some(HandlerCondition {
            op: "native_event_state_eq".into(),
            event_state_index: ip(17),
            event_state_value: ip(1),
            required_slot_count: ip(44),
            ..Default::default()
        }),
        ..Default::default()
    };
    let bindings = HandlerBindings {
        runtime_context: Some(HandlerRuntimeContext { slot_counts: vec![34], ..Default::default() }),
        ..Default::default()
    };
    let (_, issues) = compile(vec![input], &bindings);
    assert_eq!(issues.len(), 1, "unlisted refined frontier issues={issues:?}");
}

/// Go：`TestCompileChapter22PreCandidatePreservesMap22TransitionAndDialogueContext`
#[test]
fn compile_chapter22_pre_candidate_preserves_map22_transition_and_dialogue_context() {
    let Some(beats) = bind_ok("ch22_pre_candidate.json") else {
        return;
    };
    assert!(
        !beats.is_empty()
            && beats[0].op == "runtime_context"
            && beats[0].runtime_context.as_ref().is_some_and(|c| c.slot_counts == vec![70] && c.story_viewport),
        "ch22_pre runtime context={beats:?}"
    );
    let mut transition = Beat::default();
    let mut acts = 0;
    let mut dialog_sources = BTreeSet::new();
    for beat in &beats {
        match beat.op.as_str() {
            "indexed_transition" => transition = beat.clone(),
            "act" => acts += 1,
            "dialog" => {
                dialog_sources.insert(beat.source.clone());
            }
            _ => {}
        }
    }
    assert!(
        acts == 3
            && dialog_sources.len() == 5
            && transition.indexed_transition.as_ref().is_some_and(|t| t.cursor_source == "native_relative_cursor"
                && t.cursor_y_offset == 5
                && t.radial_radius == 10
                && t.radial_radius_step == 8
                && t.frames == 9
                && t.tail_delay_ms == 500),
        "ch22_pre transition/consumers acts={acts} dialog_sources={} transition={transition:?}",
        dialog_sources.len()
    );
}
