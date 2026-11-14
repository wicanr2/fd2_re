//! 移植自 `remake/internal/campaign/native_shop_scene_test.go`。
//!
//! 這個 Go 測試檔也涵蓋 `native_shop_{equipment_recipient,item_list,purchase_message,
//! purchase_success,recipient,service_menu}.go`（那幾個檔沒有自己的測試檔）。

use super::*;
use crate::campaign::church::church_test::unit;
use crate::test_support::{input, remake_asset};

fn put16(b: &mut [u8], off: usize, v: u16) {
    b[off..off + 2].copy_from_slice(&v.to_le_bytes());
}

fn get_i16(b: &[u8], off: usize) -> i64 {
    i16::from_le_bytes([b[off], b[off + 1]]) as i64
}

/// FDOTHER#5 的對話格 1..17（索引 0 留空）。
fn original_dialogue_cells(resource5: &[u8]) -> Vec<fdother::RawCell> {
    let mut dialogue = vec![fdother::RawCell::default(); 18];
    for (index, cell) in dialogue.iter_mut().enumerate().skip(1) {
        *cell = fdother::parse_lmi1_raw_entry(resource5, index as i64)
            .unwrap_or_else(|e| panic!("dialogue cell {index}: {e}"));
    }
    dialogue
}

/// FDOTHER#5 的金額數字 31..40。
fn original_digits(resource5: &[u8]) -> Vec<fdother::Frame> {
    (0..10)
        .map(|digit| {
            fdother::parse_lmi1_frame_entry(resource5, 31 + digit).unwrap_or_else(|e| panic!("digit {digit}: {e}"))
        })
        .collect()
}

/// Go：`TestDecodeNativeShopAssetsUsesOriginalMixedCodecResources`
#[test]
fn decode_native_shop_assets_uses_original_mixed_codec_resources() {
    let Some(dat_path) = input("FD2_FLAME2_DIR", "FDOTHER.DAT") else { return };
    // 0x2e341 selects exactly these three backgrounds by hub variant.  The
    // outer containers differ, but cell #1 is always the 0x4e8af opaque
    // 63x15 decoration placed at VGA+0x76c5.
    for resource_id in [12, 29, 63] {
        let assets = decode_native_shop_assets(&dat_path.to_string_lossy(), resource_id)
            .unwrap_or_else(|e| panic!("FDOTHER#{resource_id}: {e}"));
        assert!(
            assets.resource_id == resource_id
                && assets.background.len() as i64 == NATIVE_SHOP_WIDTH * NATIVE_SHOP_HEIGHT,
            "FDOTHER#{resource_id} background={}",
            assets.background.len()
        );
        assert!(
            assets.raw_entries.len() >= 11,
            "FDOTHER#{resource_id} entries={}, want at least 11",
            assets.raw_entries.len()
        );
        let cell = &assets.decoration;
        assert!(
            cell.width == 63 && cell.height == 15 && cell.pixels.len() == 63 * 15,
            "FDOTHER#{resource_id} cell#1={}x{} pixels={}, want 63x15",
            cell.width,
            cell.height,
            cell.pixels.len()
        );
    }
}

/// Go：`TestComposeNativeShopSceneUsesOriginalStableResources`
#[test]
fn compose_native_shop_scene_uses_original_stable_resources() {
    let Some(fdother_path) = input("FD2_FLAME2_DIR", "FDOTHER.DAT") else { return };
    let Some(effect_rows_path) = remake_asset("data/native_item_effect_rows.json") else { return };
    let base = fdother_path.parent().unwrap().to_path_buf();
    let fdother_s = fdother_path.to_string_lossy().into_owned();
    let assets = decode_native_shop_assets(&fdother_s, 12).unwrap();
    let resource5 = fdother::read_resource(&fdother_path, 5).unwrap();
    let dialogue = original_dialogue_cells(&resource5);
    let digits = original_digits(&resource5);
    let portraits = dato::decode_resource(base.join("DATO.DAT"), 0x81).unwrap();
    let strings = fdtxt::parse(&fdother::read_resource(base.join("FDTXT.DAT"), 0).unwrap()).unwrap();
    let font = fdtxt::parse_font(&fdother::read_resource(&fdother_path, 4).unwrap()).unwrap();
    let (s, f) = (Some(&strings), Some(&font));
    let frame =
        compose_native_shop_scene(Some(&assets), &dialogue, &digits, &portraits[0], 0x81, s, f, 12345678, 0x1b8)
            .unwrap();
    assert_eq!(frame.len() as i64, NATIVE_SHOP_WIDTH * NATIVE_SHOP_HEIGHT, "frame bytes");
    assert!(frame != assets.background, "stable shop overlays did not change the original background");
    let stable_before = frame.clone();
    for step in 0..4 {
        let opening = compose_native_shop_service_opening_frame(&frame, Some(&assets), step)
            .unwrap_or_else(|e| panic!("service opening step {step}: {e}"));
        assert!(
            opening.len() as i64 == NATIVE_SHOP_WIDTH * NATIVE_SHOP_HEIGHT && opening != frame,
            "service opening step {step} did not render"
        );
        let closing = compose_native_shop_service_closing_frame(&frame, Some(&assets), step)
            .unwrap_or_else(|e| panic!("service closing step {step}: {e}"));
        assert!(
            closing.len() as i64 == NATIVE_SHOP_WIDTH * NATIVE_SHOP_HEIGHT && closing != frame,
            "service closing step {step} did not render"
        );
    }
    let normal = compose_native_shop_service_steady_frame(&frame, Some(&assets), 0, 0).unwrap();
    let selected = compose_native_shop_service_steady_frame(&frame, Some(&assets), 0, 2).unwrap();
    assert!(normal != selected, "selected service pulse variant did not change the frame");
    assert!(frame == stable_before, "service compositors mutated the caller-owned stable frame");
    let item_assets =
        battle::load_native_item_panel_data_assets_archive(&fdother_s, &base.join("FDTXT.DAT").to_string_lossy())
            .unwrap();
    let effect_rows = battle::load_native_item_effect_row_prefix(&effect_rows_path).unwrap();
    let purchase = compose_native_shop_item_list_frame(
        &frame,
        Some(&assets),
        &item_assets,
        &[0, 1],
        0,
        1,
        &effect_rows,
        battle::NATIVE_FACILITY_FULL_PRICE,
    )
    .unwrap();
    let sale = compose_native_shop_item_list_frame(
        &frame,
        Some(&assets),
        &item_assets,
        &[0, 1],
        0,
        1,
        &effect_rows,
        battle::NATIVE_FACILITY_THREE_QUARTER_PRICE,
    )
    .unwrap();
    assert!(purchase != sale, "purchase and sale price modes rendered the same child panel");
    assert!(frame == stable_before, "child-panel compositor mutated the stable shop frame");
    let purchase_portraits = dato::decode_resource(base.join("DATO.DAT"), 0x80).unwrap();
    let pp = &purchase_portraits[0];
    let purchase_source =
        compose_native_shop_scene(Some(&assets), &dialogue, &digits, pp, 0x80, s, f, 12345678, 0x1f5).unwrap();
    let choices = fdother::decode_raw_cell_resource(&fdother_path, 2).unwrap();
    for message in
        [NATIVE_SHOP_PURCHASE_QUESTION, NATIVE_SHOP_PURCHASE_NO_ELIGIBLE_RECIPIENT, NATIVE_SHOP_PURCHASE_EQUIP_QUESTION]
    {
        let message_frame =
            compose_native_shop_purchase_message(&purchase_source, &dialogue, pp, 0x80, s, f, message, 1, 0, 50)
                .unwrap_or_else(|e| panic!("purchase message {}: {e}", message.0));
        assert_eq!(
            message_frame.len() as i64,
            NATIVE_SHOP_WIDTH * NATIVE_SHOP_HEIGHT,
            "purchase message {}",
            message.0
        );
    }
    let confirmation = compose_native_shop_purchase_confirmation(
        &purchase_source,
        &dialogue,
        pp,
        0x80,
        s,
        f,
        &choices,
        NATIVE_SHOP_PURCHASE_QUESTION,
        1,
        0,
        50,
        0,
        1,
    )
    .unwrap();
    assert!(confirmation != purchase_source, "purchase confirmation did not change the source frame");
    let question = compose_native_shop_purchase_message(
        &purchase_source,
        &dialogue,
        pp,
        0x80,
        s,
        f,
        NATIVE_SHOP_PURCHASE_QUESTION,
        1,
        0,
        50,
    )
    .unwrap();
    let mut closing = native_class_confirmation_closing_frames(&question, &choices).unwrap();
    assert_eq!(closing.len(), 4, "purchase confirmation closing frames");
    let post_choice_close = closing.last_mut().unwrap();
    let post_choice_close_before = post_choice_close.clone();
    let insufficient = compose_native_shop_purchase_insufficient_gold(post_choice_close, s, f, 1).unwrap();
    assert!(insufficient != post_choice_close_before, "insufficient-gold feedback did not append after choice closing");
    let cut = (157 * NATIVE_SHOP_WIDTH + 12) as usize;
    assert!(
        insufficient[..cut] == post_choice_close_before[..cut],
        "insufficient-gold feedback changed pixels before literal VGA target"
    );
    assert!(
        compose_native_shop_purchase_message(
            &purchase_source,
            &dialogue,
            pp,
            0x80,
            s,
            f,
            NATIVE_SHOP_PURCHASE_INSUFFICIENT_GOLD,
            1,
            0,
            50
        )
        .is_err(),
        "insufficient-gold feedback accepted a fresh dialogue source"
    );
    let fdicons = fdicon::decode_file(base.join("FDICON.B24")).unwrap();
    let rows: Vec<NativeRosterRow> = (0..6)
        .map(|identity| {
            let sprite =
                fdicons.sprite_for(identity, 0, 0).unwrap_or_else(|e| panic!("recipient sprite {identity}: {e}"));
            NativeRosterRow { sprite: sprite.clone(), name_text_index: identity + 1 }
        })
        .collect();
    let recipient =
        compose_native_shop_consumable_recipient_frame(&purchase_source, Some(&assets), &rows, 0, 0x20, s, f).unwrap();
    assert!(recipient != purchase_source, "consumable recipient list did not change the source frame");
    assert!(
        compose_native_shop_consumable_recipient_frame(&purchase_source, Some(&assets), &rows, 0, 0x1f, s, f).is_err(),
        "equipment recipient accepted the consumable roster renderer"
    );
    let full = compose_native_shop_purchase_recipient_full(&purchase_source, &dialogue, pp, 0x80, s, f, 1, 1).unwrap();
    assert!(full != purchase_source, "recipient-full feedback did not change the source frame");
    let mut equipment_rows = Vec::with_capacity(3);
    for (identity, row) in rows.iter().enumerate().take(3) {
        let mut record = vec![0u8; 0x50];
        (record[7], record[8]) = (identity as u8, identity as u8);
        put16(&mut record, 0x37, 40 + identity as u16);
        put16(&mut record, 0x39, 30 + identity as u16);
        put16(&mut record, 0x3e, 20 + identity as u16);
        for (stat, offset) in [0x48, 0x4a, 0x4c, 0x4e].into_iter().enumerate() {
            put16(&mut record, offset, (50 + stat + identity) as u16);
        }
        let candidate = native_shop_equipment_candidate_stats(&record, 0, &effect_rows).unwrap();
        let current = native_shop_equipment_current_stats(&record).unwrap();
        equipment_rows.push(NativeShopEquipmentRecipientRow {
            sprite: row.sprite.clone(),
            name_text_index: identity as i64 + 1,
            current,
            candidate,
        });
    }
    let equipment = compose_native_shop_equipment_recipient_frame(
        &purchase_source,
        Some(&assets),
        &item_assets,
        &equipment_rows,
        0,
    )
    .unwrap();
    assert!(equipment != purchase_source, "equipment recipient comparison did not change the source frame");
    let opening = native_class_list_opening_frames(&purchase_source, &equipment).unwrap();
    assert_eq!(opening.len(), 6, "equipment recipient opening");
    let list_closing = native_class_list_closing_frames(&purchase_source, &equipment).unwrap();
    assert_eq!(list_closing.len(), 5, "equipment recipient closing");
}

/// Go：`TestNativeShopEquipmentCandidateStatsReplacesSameCategoryOnly`
#[test]
fn native_shop_equipment_candidate_stats_replaces_same_category_only() {
    let size = battle::NATIVE_ITEM_EFFECT_ROW_SIZE as usize;
    let mut rows = vec![0u8; 3 * size];
    // New weapon: AP/HIT/DP/EV = 10/20/30/40.
    rows[0] = 0x10;
    for (offset, value) in [(1, 10i16), (3, 20), (5, 30), (7, 40)] {
        put16(&mut rows, offset, value as u16);
    }
    // Equipped weapon must be replaced and excluded.
    rows[size] = 0x11;
    for offset in [1, 3, 5, 7] {
        put16(&mut rows, size + offset, 100);
    }
    // Equipped armor is the opposite category and remains included.
    rows[2 * size] = 0x15;
    for offset in [1, 3, 5, 7] {
        put16(&mut rows, 2 * size + offset, 5);
    }
    let mut record = vec![0u8; 0x50];
    put16(&mut record, 0x37, 1);
    put16(&mut record, 0x39, 2);
    put16(&mut record, 0x3e, 3);
    (record[0x0a], record[0x0b]) = (0x40, 1);
    (record[0x0c], record[0x0d]) = (0x40, 2);
    let got = native_shop_equipment_candidate_stats(&record, 0, &rows).unwrap();
    assert_eq!(got, [16, 37, 28, 48], "candidate");
}

/// Go：`TestPlanNativeShopPurchaseSuccessPreservesThreeShopCases`
#[test]
fn plan_native_shop_purchase_success_preserves_three_shop_cases() {
    let tests = [
        NativeShopPurchaseSuccessPlan {
            hub_variant: 1,
            resource_id: 12,
            frame_count: 5,
            effect_offset: 45 * 320 + 169,
            per_frame_delay_bios_ticks: 2,
            restore_portrait_mode0: true,
            optional_equip_before: true,
            debit_after_presentation: true,
            return_to_product_loop: true,
            ..Default::default()
        },
        NativeShopPurchaseSuccessPlan {
            hub_variant: 3,
            resource_id: 29,
            frame_count: 1,
            effect_offset: 39 * 320 + 148,
            pre_delay_bios_ticks: 1,
            post_delay_bios_ticks: 8,
            restore_portrait_mode0: true,
            optional_equip_before: true,
            debit_after_presentation: true,
            return_to_product_loop: true,
            ..Default::default()
        },
        NativeShopPurchaseSuccessPlan {
            hub_variant: 5,
            resource_id: 63,
            frame_count: 7,
            effect_offset: 28 * 320 + 131,
            per_frame_delay_bios_ticks: 2,
            optional_equip_before: true,
            debit_after_presentation: true,
            return_to_product_loop: true,
            ..Default::default()
        },
    ];
    for want in tests {
        let got = plan_native_shop_purchase_success(want.hub_variant);
        assert!(got.as_ref().is_ok_and(|g| *g == want), "variant{} plan={got:?} want {want:?}", want.hub_variant);
    }
    assert!(plan_native_shop_purchase_success(4).is_err(), "church success variant accepted by shop plan");
}

/// Go：`TestNativeShopPurchaseSuccessUsesOriginalVariantResources`
#[test]
fn native_shop_purchase_success_uses_original_variant_resources() {
    let Some(fdother_path) = input("FD2_FLAME2_DIR", "FDOTHER.DAT") else { return };
    let dato_path = fdother_path.with_file_name("DATO.DAT");
    let resource5 = fdother::read_resource(&fdother_path, 5).unwrap();
    let digits = original_digits(&resource5);
    for (variant, resource_id, portrait_id) in [(1, 12, 0x80), (3, 29, 0x82), (5, 63, 0x84)] {
        let assets = decode_native_shop_assets(&fdother_path.to_string_lossy(), resource_id)
            .unwrap_or_else(|e| panic!("variant{variant} assets: {e}"));
        let portraits =
            dato::decode_resource(&dato_path, portrait_id).unwrap_or_else(|e| panic!("variant{variant} portrait: {e}"));
        let bare = compose_native_shop_bare_scene(Some(&assets), &digits, 1000)
            .unwrap_or_else(|e| panic!("variant{variant} bare scene: {e}"));
        let (animation, last) =
            compose_native_shop_purchase_success_frames(&bare, Some(&assets), &portraits[0], portrait_id, variant)
                .unwrap_or_else(|e| panic!("variant{variant} success: {e}"));
        let changed = animation.iter().any(|frame| *frame != assets.background);
        assert!(
            animation.len() == assets.success_frames.len()
                && last.len() as i64 == NATIVE_SHOP_WIDTH * NATIVE_SHOP_HEIGHT
                && changed,
            "variant{variant} frames={} final={} changed={changed} assets={}x{}",
            animation.len(),
            last.len(),
            assets.success_frames[0].width,
            assets.success_frames[0].height
        );
        if variant == 1 {
            const FIRST: usize = 90 * NATIVE_SHOP_WIDTH as usize + 175;
            assert!(
                bare[FIRST] == 190 && bare[FIRST + 1] == 190,
                "variant1 caller backing pixels=({},{}), want (190,190)",
                bare[FIRST],
                bare[FIRST + 1]
            );
            assert!(
                animation[0][FIRST] == 190 && animation[0][FIRST + 1] == 190,
                "variant1 success started after portrait restore: ({},{})",
                animation[0][FIRST],
                animation[0][FIRST + 1]
            );
            assert!(
                last[FIRST] == 96 && last[FIRST + 1] == 191,
                "variant1 final portrait pixels=({},{}), want (96,191)",
                last[FIRST],
                last[FIRST + 1]
            );
        }
    }
}

/// Go：`TestNativeShopEquipmentRecordRequiresAndPreservesBaseStats`
#[test]
fn native_shop_equipment_record_requires_and_preserves_base_stats() {
    let mut u = unit! {
        battle_fig: 2,
        native_identity: 2,
        has_native_identity: true,
        native_record_byte6: 3,
        has_native_record_byte6: true,
        native_record_race: 1,
        has_native_record_race: true,
        native_record_class: 4,
        has_native_record_class: true,
        lv: 8,
        mv: 5,
        exp: 20.0,
        dx: 17,
        hp: 30,
        max_hp: 35,
        mp: 5,
        max_mp: 9,
        ap: 41,
        dp: 32,
        hit: 70,
        ev: 22,
        inventory_slots: vec![0, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff],
        native_inventory_flags: vec![0x40, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80],
        base_ap: 29,
        base_dp: 25,
    };
    assert!(native_shop_equipment_record_for_unit(Some(&u)).is_err(), "equipment preview accepted unproven base AP/DP");
    u.equipment_base_set = true;
    let record = native_shop_equipment_record_for_unit(Some(&u)).unwrap();
    assert_eq!(get_i16(&record, 0x37), 29, "base AP");
    assert_eq!(get_i16(&record, 0x39), 25, "base DP");
    assert_eq!(get_i16(&record, 0x3e), 17, "base DX");
    assert_eq!(get_i16(&record, 0x48), 41, "current AP");
}

/// Go：`TestNativeShopEquipmentCurrentStatsPreservesSignedWords`
#[test]
fn native_shop_equipment_current_stats_preserves_signed_words() {
    let mut record = vec![0u8; 0x50];
    for (value, offset) in [-1i16, -32768, 32767, -23].into_iter().zip([0x48, 0x4a, 0x4c, 0x4e]) {
        put16(&mut record, offset, value as u16);
    }
    let got = native_shop_equipment_current_stats(&record).unwrap();
    assert_eq!(got, [-1, -32768, 32767, -23], "signed current stats");
}

/// Go：`TestNativeShopPurchaseTextTablesPreserveSixVariants`
#[test]
fn native_shop_purchase_text_tables_preserve_six_variants() {
    let want = [[1, 502, 1, 439, 1, 439], [1, 504, 1, 438, 1, 438], [1, 505, 1, 437, 1, 437], [1, 507, 1, 507, 1, 507]];
    for message in NATIVE_SHOP_PURCHASE_QUESTION.0..=NATIVE_SHOP_PURCHASE_EQUIP_QUESTION.0 {
        for variant in 0..6 {
            let (got, ok) = native_shop_purchase_text_index(NativeShopPurchaseMessage(message), variant);
            assert!(
                ok && got == want[message as usize][variant as usize],
                "message={message} variant={variant}: {got},{ok}"
            );
        }
    }
    let (_, ok) = native_shop_purchase_text_index(NativeShopPurchaseMessage(4), 0);
    assert!(!ok, "unknown purchase message was accepted");
}

/// Go：`TestComposeNativeShopServiceMenuRejectsInvalidState`
#[test]
fn compose_native_shop_service_menu_rejects_invalid_state() {
    let stable = vec![0u8; (NATIVE_SHOP_WIDTH * NATIVE_SHOP_HEIGHT) as usize];
    let assets = NativeShopAssets::default();
    assert!(compose_native_shop_service_opening_frame(&stable, Some(&assets), -1).is_err(), "opening step -1 accepted");
    assert!(
        compose_native_shop_service_steady_frame(&stable, Some(&assets), 4, 0).is_err(),
        "steady selected 4 accepted"
    );
    assert!(compose_native_shop_service_steady_frame(&stable, Some(&assets), 0, 4).is_err(), "steady phase 4 accepted");
}

/// Go：`TestDecodeNativeShopAssetsRejectsUnselectedResource`
#[test]
fn decode_native_shop_assets_rejects_unselected_resource() {
    assert!(decode_native_shop_assets("irrelevant", 13).is_err(), "unselected shop resource was accepted");
}
