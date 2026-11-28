//! Go 來源：`remake/cmd/fd2/settings_test.go` 與 `font_scale_test.go` 裡直接呼叫 F 組函式的測試。
//!
//! `font_scale_test.go` 其餘三個測 A 組的 `Font` 方法、R8 的 `cycleFontScale` 與真字型量寬（R5），
//! 不在這裡（見 PORTING.md 的 F 組段落）。

use super::*;

/// Go：`settings_test.go:11`，`TestOfficialLocalesCoverSystemAndSaveMessages`
#[test]
fn official_locales_cover_system_and_save_messages() {
    let wants: [(&str, [&str; 4]); 4] = [
        (
            "zh-Hant",
            [
                "語言：日本語",
                "已存檔（槽位 2：town_ch02）",
                "索爾 施放 火球：命中 2（造成 18）、未命中 1",
                "物品 2Ah：原始移位效果完成",
            ],
        ),
        (
            "zh-Hans",
            [
                "语言：日本語",
                "已存档（槽位 2：town_ch02）",
                "索尔 施放 火球：命中 2（造成 18）、未命中 1",
                "物品 2Ah：原始移位效果完成",
            ],
        ),
        (
            "ja",
            [
                "言語：日本語",
                "スロット 2 にセーブしました：town_ch02",
                "ソルはファイアを唱えた：2体に命中（18ダメージ）、1体にミス",
                "アイテム 2Ah：移動効果が完了しました",
            ],
        ),
        (
            "en",
            [
                "Language: 日本語",
                "Saved to slot 2: town_ch02",
                "Sol casts Fire: 2 hit (18 damage), 1 missed",
                "Item 2Ah: relocation complete",
            ],
        ),
    ];
    for (locale_id, want) in wants {
        let catalog = load_official_locale(locale_id).unwrap_or_else(|e| panic!("{locale_id}: {e}")).unwrap();
        let changed = catalog.format("system.locale.changed", &["日本語".into()]);
        assert_eq!(changed.as_deref().ok(), Some(want[0]), "{locale_id} locale message={changed:?}");
        let saved = catalog.format("save.saved", &[2i64.into(), "town_ch02".into()]);
        assert_eq!(saved.as_deref().ok(), Some(want[1]), "{locale_id} save message={saved:?}");
        let names = match locale_id {
            "zh-Hans" => ["索尔", "火球"],
            "ja" => ["ソル", "ファイア"],
            "en" => ["Sol", "Fire"],
            _ => ["索爾", "火球"],
        };
        let mut game = Game { locale_catalog: Some(catalog), ..Default::default() };
        let (result, ok) = game.localized_spell_result(names[0], names[1], 2, 18, false, 1);
        assert!(ok && result == want[2], "{locale_id} spell result={result:?} ok={ok}");
        let catalog = game.locale_catalog.as_ref().unwrap();
        let item_result = catalog.format("battle.item.relocation_complete", &[0x2ai64.into()]);
        assert_eq!(item_result.as_deref().ok(), Some(want[3]), "{locale_id} item result={item_result:?}");
        let item_title = catalog.format("battle.item.title", &[names[0].into()]);
        assert!(item_title.as_ref().is_ok_and(|t| !t.is_empty()), "{locale_id} item title={item_title:?}");
        for key in [
            "system.audio.changed",
            "save.unsupported",
            "save.postbattle_blocked",
            "save.none",
            "save.node_missing",
            "save.loaded",
            "battle.mp.insufficient",
            "battle.command.unavailable",
            "battle.attack.choose_target",
            "battle.command.native_unavailable",
            "battle.spell.sealed",
            "battle.spell.none",
            "battle.item.choose_slot",
            "battle.item.empty_slot",
            "battle.item.data_incomplete",
            "battle.item.unusable",
            "battle.item.effect_complete",
            "battle.item.choose_target",
            "battle.item.effect_unverified",
            "battle.item.choose_destination",
            "battle.item.damage_presenting",
            "battle.item.recovery_complete",
            "battle.item.relocation_presenting",
            "battle.item.relocation_complete",
            "battle.item.title",
            "battle.item.empty_label",
            "battle.spell.target_prompt",
            "battle.spell.blocked",
            "battle.unit.paralyzed",
            "battle.spell.result_damage",
            "battle.spell.result_heal",
            "battle.spell.miss_suffix",
            "battle.treasure.gold",
            "battle.treasure.item",
            "battle.treasure.inventory_full",
            "battle.reward.item",
            "battle.reward.item_full",
            "battle.reward.gold",
            "battle.spell.menu_title",
            "church.transfer.empty",
            "church.transfer.success",
            "church.revive.success",
            "church.class_change.success",
            "church.class_change.confirm_title",
            "church.class_change.empty",
            "church.class_change.target",
            "shop.purchase.success",
            "shop.sell.success",
            "shop.purchase.equip_prompt",
            "shop.recipient.none",
            "shop.purchase.equip_prompt.title",
            "shop.purchase.equip_prompt.controls",
            "shop.sell.item_title",
            "shop.item.equipped_label",
            "shop.sell.roster_title",
            "shop.sell.inventory_count",
            "shop.purchase.recipient_title",
            "shop.purchase.recipient_inventory",
            "shop.panel.title_controls",
            "hotel.title.fallback",
            "hotel.controls",
            "preparation.title",
            "preparation.save_prompt",
            "preparation.deploy.controls",
            "preparation.unknown_character",
            "preparation.enter_confirm",
            "preparation.save_hint",
            "battle.result.win",
            "battle.result.lose",
            "battle.result.continue",
            "ending.assets_missing",
            "ending.assets_missing_montage",
            "ending.continue_editable",
            "ending.terminal_missing",
            "ending.montage_requirements",
            "postbattle.preparation.title",
            "title.load.controls",
            "title.load.slot",
            "save.slot.empty",
            "save.slot.present",
            "church.service.status",
            "church.service.transfer",
            "church.service.revive",
            "church.service.class_change",
            "church.controls",
            "church.transfer.source",
            "church.transfer.unequipped_item",
            "church.transfer.destination",
            "church.empty_selection",
            "church.transfer.item_label",
            "church.selection_controls",
            "church.execute_controls",
            "ending.title",
            "common.yes",
            "common.no",
        ] {
            let entry = catalog.entries.get(key);
            assert!(entry.is_some_and(|e| !e.text.trim().is_empty()), "{locale_id} missing {key}");
        }
    }
}

/// Go：`settings_test.go:99`
#[test]
fn settings_round_trip_keeps_audio_and_locale_outside_save() {
    let name = "settings_round_trip_keeps_audio_and_locale_outside_save";
    let Some(root) = f_child_env(module_path!(), name, |d| vec![("XDG_DATA_HOME", d.into())]) else {
        return;
    };
    save_settings(&Settings { bgm_source: "mt32".to_string(), locale_id: "ja".to_string(), ..Default::default() });
    let got = load_settings();
    if got.bgm_source != "mt32" || got.locale_id != "ja" {
        panic!("settings={got:?}");
    }
    let raw = std::fs::read(root.join("fd2_re").join("fd2_settings.json")).unwrap();
    let document: serde_json::Map<String, serde_json::Value> = serde_json::from_slice(&raw).unwrap();
    if !document.contains_key("locale_id") {
        panic!("settings lost locale_id");
    }
    // Go `json.Marshal(saveData{})`。
    let save_raw = serde_json::to_vec(&SaveData::default()).unwrap();
    let save_document: serde_json::Map<String, serde_json::Value> = serde_json::from_slice(&save_raw).unwrap();
    if save_document.contains_key("locale_id") {
        panic!("campaign save contains locale_id");
    }
}

/// Go：`settings_test.go:146`
#[test]
fn settings_reject_unsupported_locale() {
    let name = "settings_reject_unsupported_locale";
    let Some(root) = f_child_env(module_path!(), name, |d| vec![("XDG_DATA_HOME", d.into())]) else {
        return;
    };
    let dir = root.join("fd2_re");
    std::fs::create_dir_all(&dir).unwrap();
    std::fs::write(dir.join("fd2_settings.json"), br#"{"bgm_source":"fm","locale_id":"xx"}"#).unwrap();
    let got = load_settings();
    if got.locale_id != "zh-Hant" {
        panic!("unsupported locale accepted: {got:?}");
    }
}

/// Go：`font_scale_test.go:10`
#[test]
fn normalize_font_scale_snaps_to_offered() {
    for (input, want) in [
        (0.0, DEFAULT_FONT_SCALE),  // 舊設定檔沒有這個欄位
        (-1.0, DEFAULT_FONT_SCALE), // 手改壞的值
        (1.0, 1.0),
        (0.83, 0.8),
        (1.19, 1.25),
        (5.0, 1.25), // 超出範圍取最接近的一格
    ] {
        let got = normalize_font_scale(input);
        assert_eq!(got, want, "normalizeFontScale({input:.2}) = {got:.2}，應為 {want:.2}");
    }
    for &scale in FONT_SCALES.iter() {
        let got = normalize_font_scale(scale);
        assert_eq!(got, scale, "提供的 {scale:.2} 被改成 {got:.2}");
    }
}

/// F 組補充：`loadSettings` 起點上的解碼（Go `json.Unmarshal` 只改寫到的欄位；完整的 Go 黃金在
/// `save/golden_save_test.rs`）。
#[test]
fn decode_settings_keeps_base_for_missing_and_null_fields() {
    let base = settings_load_base();
    let got = decode_settings_onto(&base, br#"{"locale_id":"ja","font_scale":null,"poster":null}"#).unwrap();
    assert_eq!(got.bgm_source, "fm");
    assert_eq!(got.locale_id, "ja");
    assert_eq!(got.font_scale, DEFAULT_FONT_SCALE);
    assert!(got.poster.is_none());
    assert_eq!(decode_settings_onto(&base, b"null"), Some(base.clone()));
    assert_eq!(decode_settings_onto(&base, b"[]"), None);
    assert_eq!(decode_settings_onto(&base, br#"{"locale_id":"ja","locale_id":"en"}"#).unwrap().locale_id, "en");
}
