//! Go `remake/internal/campaign/native_story_portrait_rect_test.go` 的逐條移植。

use super::*;

/// Go：`TestNativeStoryDialoguePortraitRectPreservesAsymmetricAnchors`
#[test]
fn native_story_dialogue_portrait_rect_preserves_asymmetric_anchors() {
    let tests = [
        ("FFEF", go_rect(232, 5, 312, 85)),
        ("FFED", go_rect(232, 5, 312, 85)),
        ("FFEE", go_rect(8, 115, 88, 195)),
        ("FFEC", go_rect(8, 115, 88, 195)),
    ];
    for (control, want) in tests {
        let got = native_story_dialogue_portrait_rect(control);
        assert!(matches!(&got, Ok(r) if *r == want), "{control} rect={got:?}, want {want:?}");
    }
    assert!(native_story_dialogue_portrait_rect("FFFF").is_err(), "unsupported control accepted");
}
