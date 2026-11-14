//! Go `remake/internal/campaign/handler_script_test.go` 的逐條移植。

use super::*;

/// Go `filepath.Glob("../../assets/cutscenes/handlers/ch??_*.json")`：依檔名排序（Glob 的結果已排序）。
fn handler_paths() -> Option<Vec<String>> {
    let dir = crate::test_support::remake_asset("cutscenes/handlers")?;
    let mut paths: Vec<String> = std::fs::read_dir(&dir)
        .expect("列出 handlers 目錄")
        .filter_map(|entry| entry.ok())
        .filter_map(|entry| {
            let name = entry.file_name().to_string_lossy().into_owned();
            let chars: Vec<char> = name.chars().collect();
            let matches = name.starts_with("ch")
                && chars.len() >= 10
                && chars[4] == '_'
                && name.ends_with(".json")
                && chars.len() >= 5 + ".json".len();
            matches.then(|| format!("{dir}/{name}"))
        })
        .collect();
    paths.sort();
    Some(paths)
}

/// Go：`TestLoadExportedHandlerScripts`
#[test]
fn load_exported_handler_scripts() {
    let Some(paths) = handler_paths() else {
        return;
    };
    assert_eq!(paths.len(), 62, "exported handler scripts = {}, want 62", paths.len());
    let mut errors = Vec::new();
    for path in &paths {
        let script = load_handler_script(path).unwrap_or_else(|err| panic!("LoadHandlerScript({path:?}): {err}"));
        fn check_sources(path: &str, location: &str, beats: &[HandlerBeat], errors: &mut Vec<String>) {
            for (i, beat) in beats.iter().enumerate() {
                let at = format!("{location}[{i}]");
                if beat.source.addr.is_empty() {
                    errors.push(format!("{path} {at} lacks source address"));
                }
                check_sources(path, &format!("{at}.then"), &beat.then, errors);
                check_sources(path, &format!("{at}.else"), &beat.r#else, errors);
            }
        }
        check_sources(path, "beats", &script.beats, &mut errors);
    }
    assert!(errors.is_empty(), "{}", errors.join("\n"));
}

/// Go：`TestExportedHandlerNativeCallsUseThreeEvidenceStates`
#[test]
fn exported_handler_native_calls_use_three_evidence_states() {
    let Some(paths) = handler_paths() else {
        return;
    };
    let mut counts: BTreeMap<String, i64> = BTreeMap::new();
    let mut targets: BTreeMap<String, BTreeMap<String, i64>> = BTreeMap::new();
    fn visit(
        beats: &[HandlerBeat],
        counts: &mut BTreeMap<String, i64>,
        targets: &mut BTreeMap<String, BTreeMap<String, i64>>,
    ) {
        for beat in beats {
            if matches!(beat.op.as_str(), "native_call" | "unresolved_native_call" | "unknown") {
                *counts.entry(beat.op.clone()).or_default() += 1;
                *targets.entry(beat.op.clone()).or_default().entry(beat.native_target.clone()).or_default() += 1;
            }
            visit(&beat.then, counts, targets);
            visit(&beat.r#else, counts, targets);
        }
    }
    for path in &paths {
        let script = load_handler_script(path).unwrap_or_else(|err| panic!("LoadHandlerScript({path:?}): {err}"));
        visit(&script.beats, &mut counts, &mut targets);
    }
    let count = |op: &str| counts.get(op).copied().unwrap_or(0);
    assert!(
        count("native_call") == 88 && count("unresolved_native_call") == 3 && count("unknown") == 0,
        "handler native-call states = {counts:?}, want native=88 unresolved=3 unknown=0"
    );
    let empty = BTreeMap::new();
    let got = targets.get("unresolved_native_call").unwrap_or(&empty);
    assert!(
        got.len() == 2 && got.get("0x22253") == Some(&1) && got.get("0x2bce5") == Some(&2),
        "unresolved native targets = {got:?}"
    );
    let got = targets.get("unknown").unwrap_or(&empty);
    assert!(got.is_empty(), "true unknown native targets = {got:?}, want none");
    let got = targets.get("native_call").unwrap_or(&empty);
    assert_eq!(got.get("0x24336"), Some(&1), "classified native targets = {got:?}, want one 0x24336 callsite");
}

/// Go：`TestChapter1PostPreservesInactiveDiamond`
#[test]
fn chapter1_post_preserves_inactive_diamond() {
    let Some(path) = crate::test_support::remake_asset("cutscenes/handlers/ch01_post.json") else {
        return;
    };
    let script = load_handler_script(&path).unwrap();
    assert!(
        !script.beats.is_empty() && script.beats[0].op == "if",
        "first beat = {:?}, want structured if",
        script.beats
    );
    let branch = &script.beats[0];
    assert!(
        branch.source.addr == "0x22f71" && branch.condition.as_ref().is_some_and(|c| c.op == "any_unit_inactive"),
        "branch identity = {branch:?}"
    );
    let condition = branch.condition.as_ref().unwrap();
    assert_eq!(condition.unit_slots, vec![5, 6, 7, 8, 9, 10], "inactive slots = {:?}", condition.unit_slots);
    // Go 比 `TextIndex != float64(7)`：JSON 解碼的 `any` 數字是 float64；Rust 比數值。
    let text_index = |beat: &HandlerBeat| beat.text_index.as_f64();
    assert!(branch.then.len() == 1 && text_index(&branch.then[0]) == Some(7.0), "inactive arm = {:?}", branch.then);
    assert!(
        branch.r#else.len() == 2
            && text_index(&branch.r#else[0]) == Some(6.0)
            && branch.r#else[1].item_id == Some(0xc6),
        "all-active arm = {:?}",
        branch.r#else
    );
    assert!(script.beats.len() >= 2 && script.beats[1].op == "pan", "common continuation = {:?}", script.beats);
}

/// Go：`TestChapter0PreHandlerPreservesReclassifiedNativeOperations`
#[test]
fn chapter0_pre_handler_preserves_reclassified_native_operations() {
    let Some(path) = crate::test_support::remake_asset("cutscenes/handlers/ch00_pre.json") else {
        return;
    };
    let script = load_handler_script(&path).unwrap();
    assert!(
        script.handler == "0x3231b" && script.phase == "pre" && script.chapter == 0,
        "chapter 0 pre identity = {script:?}"
    );
    let got = script.diagnostics.get("unknown_ops").copied().unwrap_or(0);
    assert_eq!(got, 0, "chapter 0 unknown operations = {got}, want 0 after native body classification");
    let want = ["loadch", "pan", "act", "scroll_step", "dialog", "scroll_step", "dialog", "bgm"];
    assert!(script.beats.len() >= want.len(), "chapter 0 has only {} beats", script.beats.len());
    let mut errors = Vec::new();
    for (i, op) in want.iter().enumerate() {
        if script.beats[i].op != *op {
            errors.push(format!("beat {i} op = {:?}, want {op:?}", script.beats[i].op));
        }
    }
    if script.beats[0].chapter != Some(32) {
        errors.push(format!("first loadch chapter = {:?}, want 32", script.beats[0].chapter));
    }
    if script.beats[2].acting_id != Some(99) {
        errors.push(format!("first act id = {:?}, want 99", script.beats[2].acting_id));
    }
    if script.beats[3].unit_slot != Some(2) || script.beats[3].repeat != Some(15) {
        errors.push(format!(
            "first scroll step = slot {:?} repeat {:?}, want slot 2 repeat 15",
            script.beats[3].unit_slot, script.beats[3].repeat
        ));
    }
    let want_tail = ["deactivate_unit", "redraw", "delay", "dialog", "reset_pose", "focus_unit"];
    for (i, op) in want_tail.iter().enumerate() {
        let beat = &script.beats[script.beats.len() - want_tail.len() + i];
        if beat.op != *op {
            errors.push(format!("tail beat {i} op={:?}, want {op:?}", beat.op));
        }
    }
    assert!(errors.is_empty(), "{}", errors.join("\n"));
}
