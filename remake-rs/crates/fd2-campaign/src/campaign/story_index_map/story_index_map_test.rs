//! Go `remake/internal/campaign/story_index_map_test.go` 的逐條移植。

use super::*;

fn scene_is(target: &StoryIndexTarget, want: &str) -> bool {
    target.scene.as_deref() == Some(want)
}

/// Go：`TestLoadCountAlignedStoryIndexMap`
#[test]
fn load_count_aligned_story_index_map() {
    let Some(path) = crate::test_support::remake_asset("cutscenes/dialogue-index/count-aligned.json") else {
        return;
    };
    let index = load_story_index_map(&path).unwrap();
    assert!(
        index.mapping_kind == "count_aligned_only" && index.diagnostics.len() == 4,
        "manifest kind={} diagnostics={}",
        index.mapping_kind,
        index.diagnostics.len()
    );

    let (palace_first, ok) = index.lookup("FDTXT_033", "ch00_palace.json", 0);
    assert!(
        ok && palace_first.len() == 1 && scene_is(&palace_first[0], "王座廳,傳位"),
        "palace FDTXT #0 = {palace_first:?}, want throne lines"
    );
    let got = &palace_first[0].lines;
    assert!(got.len() == 6 && got[0] == 0 && got[5] == 5, "palace FDTXT #0 lines = {got:?}, want 0..5");

    let (palace_last, ok) = index.lookup("FDTXT_033", "ch00_palace.json", 5);
    assert!(
        ok && palace_last.len() == 1 && scene_is(&palace_last[0], "王城一隅,亞雷斯撞見"),
        "palace FDTXT #5 = {palace_last:?}, want grass tail"
    );
    let got = &palace_last[0].lines;
    assert!(got.len() == 12 && got[0] == 10 && got[11] == 21, "palace FDTXT #5 lines = {got:?}, want 10..21");

    let (_, ok) = index.lookup("FDTXT_033", "ch33.json", 0);
    assert!(!ok, "unmapped ch33 context unexpectedly resolved");
    let (ch02_reward, ok) = index.lookup("FDTXT_002", "ch02.json", 6);
    assert!(
        ok && ch02_reward.len() == 1 && scene_is(&ch02_reward[0], "戰鬥中,強盜兵分兩路"),
        "ch02 reward FDTXT #6 = {ch02_reward:?}"
    );
    let got = &ch02_reward[0].lines;
    assert!(got.len() == 1 && got[0] == 10, "ch02 reward #6 lines = {got:?}, want scene1 line10");
    let (ch02_post, ok) = index.lookup("FDTXT_002", "ch02.json", 8);
    assert!(
        ok && ch02_post.len() == 1 && scene_is(&ch02_post[0], "希莉亞登場") && ch02_post[0].lines.len() == 15,
        "ch02 post FDTXT #8 = {ch02_post:?}"
    );
    let (ch03_reinforce, ok) = index.lookup("FDTXT_003", "ch03.json", 4);
    assert!(
        ok && ch03_reinforce.len() == 1 && ch03_reinforce[0].scene.is_none() && ch03_reinforce[0].scene_index == 0,
        "ch03 turn3 FDTXT #4 = {ch03_reinforce:?}"
    );
    let got = &ch03_reinforce[0].lines;
    assert!(got.len() == 7 && got[0] == 14 && got[6] == 20, "ch03 turn3 #4 lines = {got:?}, want scene0 lines14..20");

    let (ch26_missing_item, ok) = index.lookup("FDTXT_026", "ch26.json", 2);
    assert!(
        ok && ch26_missing_item.len() == 1
            && ch26_missing_item[0].scene_index == 0
            && ch26_missing_item[0].lines.len() == 1
            && ch26_missing_item[0].lines[0] == 10,
        "ch26 event61 missing-item text #2 = {ch26_missing_item:?}"
    );
    let (ch26_insert_item, ok) = index.lookup("FDTXT_026", "ch26.json", 3);
    assert!(
        ok && ch26_insert_item.len() == 1
            && ch26_insert_item[0].scene_index == 0
            && ch26_insert_item[0].lines.len() == 1
            && ch26_insert_item[0].lines[0] == 11,
        "ch26 event61 insertion text #3 = {ch26_insert_item:?}"
    );
    let (ch26_recruit, ok) = index.lookup("FDTXT_026", "ch26.json", 4);
    assert!(
        ok && ch26_recruit.len() == 1
            && ch26_recruit[0].scene_index == 1
            && ch26_recruit[0].lines.len() == 10
            && ch26_recruit[0].lines[0] == 0
            && ch26_recruit[0].lines[9] == 9,
        "ch26 event61 recruitment text #4 = {ch26_recruit:?}"
    );
}

/// Go：`TestStoryIndexMapRequiresExactCounts`
#[test]
fn story_index_map_requires_exact_counts() {
    let resource = StoryIndexResource {
        source_dat: "FDTXT_001".into(),
        raw_string_count: 1,
        raw_utterance_count: 2,
        ..Default::default()
    };
    let mapping = StoryIndexScriptMapping {
        script: "ch01.json".into(),
        source_dat: "FDTXT_001".into(),
        status: "count_aligned".into(),
        raw_utterance_count: 2,
        story_line_count: 1,
        mappings: vec![StoryIndexStringMap {
            string_index: 0,
            utterance_count: 2,
            targets: vec![StoryIndexTarget { scene: None, scene_index: 0, lines: vec![0, 1] }],
        }],
    };
    assert!(validate_story_index_mapping(&resource, &mapping).is_err(), "mismatched story count was accepted");
}
