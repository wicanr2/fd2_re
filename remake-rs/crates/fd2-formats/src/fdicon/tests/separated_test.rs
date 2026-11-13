//! 移植自 `separated_test.go`。

use super::{TempDir, original_fdicon_path, separated_assets_root};
use crate::fdicon::*;

fn archive_present() -> Option<String> {
    let archive = original_fdicon_path();
    if std::path::Path::new(&archive).exists() {
        Some(archive)
    } else {
        eprintln!("略過：player-provided FDICON.B24 is absent");
        None
    }
}

#[test]
fn separated_bank_matches_player_archive() {
    let Some(archive) = archive_present() else {
        return;
    };
    let want = decode_file(&archive).unwrap();
    let got = load_separated_bank(format!("{}/sprites/fdicon", separated_assets_root())).unwrap();
    assert!(
        got.sprites.len() == want.sprites.len() && got.sprites.len() as i64 == SEPARATED_SPRITE_COUNT,
        "separated sprites={} archive={}",
        got.sprites.len(),
        want.sprites.len()
    );
    for (index, (g, w)) in got.sprites.iter().zip(&want.sprites).enumerate() {
        for (layer, a, b) in
            [("pixels", &g.pixels, &w.pixels), ("mask", &g.mask, &w.mask), ("remap_mask", &g.remap_mask, &w.remap_mask)]
        {
            assert_eq!(a, b, "sprite {index} {layer} differs from archive");
        }
    }
}

#[test]
fn separated_bank_rejects_incomplete_pack() {
    let root = TempDir::new("bank-incomplete");
    let document = r#"{"schema_version":1,"kind":"fdicon_sprite_bank","asset_id":"sprites/fdicon","status":"decoded","evidence":"confirmed","source":{"file":"FDICON.B24","size":624010,"md5":"46f793540209a063ea73a5373ca14bf4","sha256":"7efb4448d05f19c1e17ebd53f3e3afead235f5c008d5167548d834c3686b1e44"},"sprites":[]}"#;
    std::fs::write(root.0.join("bank.json"), document).unwrap();
    assert!(load_separated_bank(root.path()).is_err(), "incomplete separated FDICON bank was accepted");
}

#[test]
fn native_source_groups_match_player_archive_masks() {
    let Some(archive) = archive_present() else {
        return;
    };
    let bank = decode_file(&archive).unwrap();
    assert_eq!(
        bank.sprites.len() as i64,
        NATIVE_SOURCE_GROUP_COUNT * 12,
        "native group bound differs from actual archive"
    );
    let (mut visible_groups, mut visible_frames) = (0, 0);
    for group in 0..NATIVE_SOURCE_GROUP_COUNT {
        let mut visible = false;
        for sprite in &bank.sprites[(group * 12) as usize..((group + 1) * 12) as usize] {
            if sprite.mask.iter().any(|&m| m != 0) {
                visible = true;
                visible_frames += 1;
            }
        }
        assert_eq!(
            visible,
            native_source_group_has_pixels(group),
            "source-write classification differs at group {group}"
        );
        if visible {
            visible_groups += 1;
        }
    }
    assert!(
        visible_groups == 125
            && visible_frames == 1455
            && !native_source_group_has_pixels(-1)
            && !native_source_group_has_pixels(NATIVE_SOURCE_GROUP_COUNT),
        "source inventory groups={visible_groups} frames={visible_frames}"
    );
}
