use std::fmt::Write as _;
use std::path::PathBuf;

use serde_json::{Value, json};
use sha2::Digest as _;

use super::*;
use crate::test_support::temp_dir;

fn portrait(root: &Path, speaker: i64, side: u32, alpha: u8) -> Value {
    let file = format!("{speaker}-{side}.png");
    let mut bytes = Vec::new();
    let mut encoder = png::Encoder::new(&mut bytes, side, side);
    encoder.set_color(png::ColorType::Rgba);
    encoder.write_header().unwrap().write_image_data(&[73, 11, 25, alpha].repeat((side * side) as usize)).unwrap();
    std::fs::write(root.join(&file), &bytes).unwrap();
    let (role, contract) = if side == 80 {
        ("story_portrait_frame", "native_story_dialogue_rgba_overlay_v1")
    } else {
        ("story_portrait_hd_frame", "native_story_dialogue_rgba_overlay_hd6_v1")
    };
    json!({
        "role":role, "consumer_contract":contract, "status":"runtime_ready", "speaker_id":speaker,
        "frame":0, "mouth_state":"closed", "width":side, "height":side, "file":file, "sha256":format!("{:x}", Sha256::digest(bytes))
    })
}

#[test]
fn story_portraits_require_valid_pairs_and_install_atomically() {
    let root = temp_dir("story-portrait-catalog");
    let catalog = root.join("catalog.json");
    let write = |assets: &[Value]| {
        std::fs::write(
            &catalog,
            json!({"schema_version":1,"theme_id":MODERN_HANDPAINTED_THEME_ID,"assets":assets}).to_string(),
        )
        .unwrap();
    };
    let base = portrait(&root, 0, 80, 255);
    let sibling = portrait(&root, 1, 80, 255);
    let hd = portrait(&root, 0, 480, 255);
    let mut set = ModernStoryPortraitSet::default();
    write(std::slice::from_ref(&base));
    set.load_story_portraits(&catalog, &root).unwrap();
    assert!(!set.has_hd);
    assert!(set.portraits[&0].hd_image.is_none());
    write(&[base.clone(), hd.clone()]);
    set.load_story_portraits(&catalog, &root).unwrap();
    assert!(set.has_hd);
    let before = set.clone();
    assert_eq!(before.portraits[&0].hd_image.as_ref().unwrap().width, 480);

    let transparent = portrait(&root, 2, 80, 128);
    let mut wrong_digest = hd.clone();
    wrong_digest["sha256"] = json!("wrong");
    let mut wrong_geometry = hd.clone();
    (wrong_geometry["file"], wrong_geometry["sha256"]) = (base["file"].clone(), base["sha256"].clone());
    let mut unsafe_name = hd.clone();
    unsafe_name["file"] = json!("../0-480.png");
    for bad in [
        vec![base.clone(), sibling, hd.clone()],
        vec![base.clone(), base.clone()],
        vec![base.clone(), transparent],
        vec![base.clone(), wrong_digest],
        vec![base.clone(), wrong_geometry],
        vec![base, unsafe_name],
        vec![hd],
    ] {
        write(&bad);
        assert!(set.load_story_portraits(&catalog, &root).is_err());
        assert_eq!(set, before);
    }
}

#[test]
#[ignore = "正式私人素材與 Go 黃金由 tools/rust_r6_opening.sh 安排"]
fn real_story_portraits_match_go() {
    let env = |name| PathBuf::from(std::env::var_os(name).unwrap());
    let mut set = ModernStoryPortraitSet::default();
    set.load_story_portraits(&env("FD2_TEST_OPENING_CATALOG"), &env("FD2_TEST_OPENING_PACK")).unwrap();
    assert!(set.has_hd);
    let mut actual = String::new();
    for (&id, p) in &set.portraits {
        let base = p.image.as_ref().unwrap();
        let hd = p.hd_image.as_ref().unwrap();
        writeln!(actual, "{id} {:x} {:x}", Sha256::digest(base.read_pixels()), Sha256::digest(hd.read_pixels()))
            .unwrap();
    }
    let out = env("FD2_R6_OPENING_OUT");
    std::fs::write(out.join("rust-story-portraits.txt"), &actual).unwrap();
    assert_eq!(actual, std::fs::read_to_string(out.join("go-story-portraits.txt")).unwrap());
    println!("{} 對正式故事頭像與 Go 相同，序列 SHA-256 {:x}", set.portraits.len(), Sha256::digest(actual.as_bytes()));
}
