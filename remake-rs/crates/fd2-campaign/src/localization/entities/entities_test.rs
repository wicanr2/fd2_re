//! 移植自 Go `localization/entities_test.go`。

use super::*;
use crate::test_support;
use std::collections::BTreeSet;

#[test]
fn test_official_entity_catalogs_share_item_identity() {
    let Some(root) = test_support::remake_asset("locales") else {
        return;
    };
    let mut reference: Option<BTreeSet<i64>> = None;
    for locale in ["zh-Hant", "zh-Hans", "ja", "en"] {
        let catalog = load_official_entities(&root, locale).unwrap_or_else(|e| panic!("{locale}: {e}"));
        assert_eq!(catalog.items.len(), 200, "{locale} items, want 200");
        assert_eq!(catalog.characters.len(), 32, "{locale} characters, want 32");
        assert_eq!(catalog.battle_names.len(), 94, "{locale} battle names, want 94");
        assert_eq!(catalog.command_names.len(), 35, "{locale} command names, want 35");
        assert_eq!(catalog.class_names.len(), 29, "{locale} class names, want 29");
        let name = catalog.character_name(9);
        assert!(
            name.is_ok() && (locale != "zh-Hant" || name.as_deref().ok() == Some("悠妮")),
            "{locale} character 9={name:?}"
        );
        match &reference {
            None => reference = Some(catalog.items.keys().copied().collect()),
            Some(reference) => {
                for id in reference {
                    assert!(catalog.items.contains_key(id), "{locale} misses item {id}");
                }
            }
        }
    }
}

#[test]
fn test_entity_catalog_class_lookup_preserves_placeholders_and_fails_closed() {
    let Some(root) = test_support::remake_asset("locales") else {
        return;
    };
    let catalog = load_official_entities(&root, "en").unwrap();
    let got = catalog.class_name(15);
    assert_eq!(got.as_deref().ok(), Some("Dragonblade"), "class 15={got:?}");
    for id in [26, 27, 28] {
        catalog.class_name(id).unwrap_or_else(|e| panic!("original placeholder class {id} was lost: {e}"));
    }
    for id in [-1, 29] {
        assert!(catalog.class_name(id).is_err(), "unknown class {id} was accepted");
    }
}

#[test]
fn test_entity_catalog_character_lookup_fails_closed() {
    let Some(root) = test_support::remake_asset("locales") else {
        return;
    };
    let catalog = load_official_entities(&root, "en").unwrap();
    assert!(catalog.character_name(32).is_err(), "missing character name was accepted");
}

#[test]
fn test_entity_catalog_rejects_confirmed_empty_item_ids() {
    let Some(root) = test_support::remake_asset("locales") else {
        return;
    };
    let catalog = load_official_entities(&root, "zh-Hant").unwrap();
    for id in 108..=122 {
        assert!(catalog.item_name(id).is_err(), "confirmed-empty item {id} was accepted");
    }
}

#[test]
fn test_entity_catalog_rejects_confirmed_empty_battle_name_ids() {
    let Some(root) = test_support::remake_asset("locales") else {
        return;
    };
    let catalog = load_official_entities(&root, "zh-Hant").unwrap();
    for id in [32, 67, 75, 128, 132, 136, 138] {
        assert!(catalog.battle_name(id).is_err(), "confirmed-empty battle name {id} was accepted");
    }
}

#[test]
fn test_entity_catalog_rejects_confirmed_empty_command_id() {
    let Some(root) = test_support::remake_asset("locales") else {
        return;
    };
    let catalog = load_official_entities(&root, "zh-Hant").unwrap();
    assert!(catalog.command_name(31).is_err(), "confirmed-empty command 31 was accepted");
}

#[test]
fn test_entity_catalog_rejects_future_fields() {
    let Some(path) = test_support::remake_asset("locales/en/entities.json") else {
        return;
    };
    let raw = std::fs::read_to_string(path).unwrap();
    let bad = raw.replacen(r#""schema_version": 1,"#, r#""schema_version": 1, "future": true,"#, 1);
    let directory = test_support::temp_dir("entities-future");
    std::fs::create_dir(directory.join("en")).unwrap();
    std::fs::write(directory.join("en").join("entities.json"), bad).unwrap();
    assert!(load_official_entities(&directory.to_string_lossy(), "en").is_err(), "future entity field was accepted");
}
