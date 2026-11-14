//! Go 來源：`remake/cmd/fd2/native_story_input_test.go`（R4 B 組，期望值不改）。

use super::*;

/// Go：`native_story_input_test.go:15` `TestNativeStoryAdvanceAcceptsEscapeLikeEnter`
#[test]
#[ignore = "等 D 合併（cancel_native_command32_presentation）"]
fn test_native_story_advance_accepts_escape_like_enter() {
    let build = || {
        let line = |text: &str| campaign::Line { speaker: 0, text: text.to_string(), ..Default::default() };
        let c = campaign::Campaign {
            start: String::from("prologue"),
            nodes: BTreeMap::from([
                (
                    String::from("prologue"),
                    campaign::Node {
                        r#type: String::from("story"),
                        next: String::from("after"),
                        lines: vec![line("第一句"), line("第二句")],
                        ..Default::default()
                    },
                ),
                (
                    String::from("after"),
                    campaign::Node { r#type: String::from("ending"), text: String::from("done"), ..Default::default() },
                ),
            ]),
            ..Default::default()
        };
        let mut g =
            Game { locale_id: String::from("zh-Hant"), camp: Some(campaign::new_runner(&c)), ..Default::default() };
        g.enter_node();
        assert!(
            g.load_err.is_empty() && g.dialog.len() == 2,
            "序幕節點未備妥兩句對白：err={:?} dialog={}",
            g.load_err,
            g.dialog.len()
        );
        g
    };
    let run = |input: NativeStoryInput| {
        let mut g = build();
        let mut seen = Vec::new();
        for i in 0..2 {
            let mut node = g.camp.as_ref().unwrap().node().cloned();
            assert!(g.handle_native_story_input(node.as_mut(), input), "story 節點必須擁有輸入（第 {i} 次）");
            g.tick(STORY_FADE_FRAMES);
            seen.push((g.camp.as_ref().unwrap().node_id(), g.dialog.len()));
        }
        seen
    };
    let enter = run(NativeStoryInput { enter: true, escape: false });
    let escape = run(NativeStoryInput { enter: false, escape: true });
    let silent = run(NativeStoryInput::default());
    let want = [(String::from("prologue"), 1), (String::from("after"), 0)];
    for (name, got) in [("enter", &enter), ("escape", &escape)] {
        for (i, w) in want.iter().enumerate() {
            assert_eq!(&got[i], w, "{name} 第 {i} 次推進");
        }
    }
    for (i, got) in silent.iter().enumerate() {
        assert_eq!(got, &(String::from("prologue"), 2), "不送鍵時第 {i} 次，原版對應收據是畫面停住");
    }
}
