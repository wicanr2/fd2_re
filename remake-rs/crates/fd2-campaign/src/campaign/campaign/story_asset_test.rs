//! Go `remake/internal/campaign/story_asset_test.go` 的逐條移植。

use crate::test_support::remake_asset;

/// Go：`TestChapterThreePostBattleSpeakerControlCodes`
#[test]
fn chapter_three_post_battle_speaker_control_codes() {
    #[derive(serde::Deserialize, Default)]
    #[serde(default)]
    struct StoryLine {
        speaker: i64,
        speaker_name: String,
    }
    #[derive(serde::Deserialize, Default)]
    #[serde(default)]
    struct StoryScene {
        lines: Vec<StoryLine>,
    }
    #[derive(serde::Deserialize, Default)]
    #[serde(default)]
    struct Story {
        scenes: Vec<StoryScene>,
    }

    let Some(path) = remake_asset("story/ch03.json") else {
        return;
    };
    let data = std::fs::read(path).unwrap();
    let story: Story = serde_json::from_slice(&data).unwrap();
    assert!(story.scenes.len() >= 2, "ch03 story scenes = {}, want at least 2", story.scenes.len());
    assert!(
        story.scenes[0].lines.len() >= 21,
        "ch03 pre/turn3 lines = {}, want at least 21",
        story.scenes[0].lines.len()
    );
    assert!(
        story.scenes[1].lines.len() >= 18,
        "ch03 post-battle lines = {}, want at least 18",
        story.scenes[1].lines.len()
    );
    let mut errors = Vec::new();
    for (line, speaker) in [(14, 77), (15, 2), (16, 77), (17, 8), (18, 2), (19, 8), (20, 77)] {
        let got = story.scenes[0].lines[line].speaker;
        if got != speaker {
            errors.push(format!("ch03 turn3 #4 line{line} speaker={got}, want raw portrait {speaker}"));
        }
    }

    for (line, speaker, speaker_name) in [
        // 說話者 77 是刺客隊長。舊期待值「約」是把 speaker id 當成字模索引讀出來
        // 的產物——unicode_to_glyph 裡「約」的索引剛好也是 77。同一場景另外四句
        // 都寫刺客隊長，句子內容（敗給小鬼、呼喊葛雷大人）也對得上。
        (0, 77, "刺客隊長"),
        (17, 6, "萊汀"),
    ] {
        let got = &story.scenes[1].lines[line];
        if got.speaker != speaker || got.speaker_name != speaker_name {
            errors.push(format!(
                "ch03 scene1 line{line} speaker = ({}, {:?}), want ({speaker}, {speaker_name:?})",
                got.speaker, got.speaker_name
            ));
        }
    }
    assert!(errors.is_empty(), "{}", errors.join("\n"));
}
