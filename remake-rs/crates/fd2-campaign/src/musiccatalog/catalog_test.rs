//! 移植自 Go `musiccatalog/catalog_test.go`。

use super::*;
use crate::test_support;
use serde_json::Value;

/// Go `repositoryAssets`：受版控的 `remake/assets`（`FD2_REMAKE_ASSETS`）。
fn repository_assets() -> Option<String> {
    test_support::remake_asset("music_catalog.json")?;
    std::env::var("FD2_REMAKE_ASSETS").ok()
}

/// Go `requireRenderBundle`：分離音樂 render 沒裝時 Go 略過；`FD2_REQUIRE_ORIGINAL=1` 下 Rust 失敗。
fn require_render_bundle() -> Option<String> {
    test_support::remake_asset("music_fm/FDMUS_001.ogg")?;
    repository_assets()
}

fn mutated_catalog(mutate: impl FnOnce(&mut Value)) -> Option<(String, std::path::PathBuf)> {
    let assets = repository_assets()?;
    let raw = std::fs::read(std::path::Path::new(&assets).join("music_catalog.json")).unwrap();
    let mut document: Value = serde_json::from_slice(&raw).unwrap();
    mutate(&mut document);
    let mutated = serde_json::to_vec(&document).unwrap();
    let root = test_support::temp_dir("music-mutated");
    std::fs::write(root.join("music_catalog.json"), mutated).unwrap();
    Some((assets, root))
}

#[test]
fn test_load_validates_all_thirty_separated_renders() {
    let Some(assets) = require_render_bundle() else {
        return;
    };
    let catalog = load(&assets).unwrap();
    for profile in ["fm", "mt32"] {
        for track in ["FDMUS_004", "FDMUS_018"] {
            let path = catalog.resolve(profile, track).unwrap();
            let size = std::fs::metadata(&path).map(|m| m.len());
            assert!(matches!(size, Ok(n) if n > 0), "{profile}/{track} path={path:?} size={size:?}");
        }
    }
}

#[test]
fn test_load_fails_closed_on_catalog_identity_mutation() {
    let Some((_, root)) = mutated_catalog(|document| {
        document["source"]["sha256"] = Value::from("0000000000000000000000000000000000000000000000000000000000000000");
    }) else {
        return;
    };
    assert!(load(&root.to_string_lossy()).is_err(), "mutated FDMUS identity was accepted");
}

#[test]
fn test_load_fails_closed_on_render_hash_mutation() {
    if require_render_bundle().is_none() {
        return;
    }
    let Some((assets, root)) = mutated_catalog(|document| {
        document["tracks"][0]["renders"]["fm"]["sha256"] =
            Value::from("0000000000000000000000000000000000000000000000000000000000000000");
    }) else {
        return;
    };
    for directory in ["music_fm", "music_mt32"] {
        std::os::unix::fs::symlink(std::path::Path::new(&assets).join(directory), root.join(directory)).unwrap();
    }
    assert!(load(&root.to_string_lossy()).is_err(), "mutated OGG hash was accepted");
}

#[test]
fn test_resolve_rejects_unknown_profile_and_track() {
    let Some(assets) = require_render_bundle() else {
        return;
    };
    let catalog = load(&assets).unwrap();
    assert!(catalog.resolve("legacy", "FDMUS_004").is_err(), "legacy profile was accepted");
    assert!(catalog.resolve("fm", "FDMUS_999").is_err(), "unknown track was accepted");
}
