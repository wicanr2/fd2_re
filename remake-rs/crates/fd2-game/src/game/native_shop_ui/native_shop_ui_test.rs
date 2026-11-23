//! Go 來源：`remake/cmd/fd2/native_shop_ui_test.go`（R4 E 組，期望值不改）。
//!
//! 未移植：`:21`（`TestNativeShopProductionOwnerDrawsOriginalMenuAndPurchaseList`，以 `Draw` 推進並走購買、
//! 販賣、裝備與轉移的 `r8` 函式）與四個截圖狀態測試（`:785`、`:862`、`:940`、`:1027`，`r8`）。

use super::*;
use crate::test_support;

/// Go：`native_shop_ui_test.go:697` `TestNativeShopAssetsDoNotRequireFDOTHERArchive`
///
/// Go 另把 `FD2_ORIGINAL_FDOTHER` 換成不存在的檔，證明商店載入器不開封存檔；Rust 的載入器只讀分離素材包。
#[test]
fn test_native_shop_assets_do_not_require_fdother_archive() {
    if test_support::input("FD2_ASSET_PACK", "shop/FDOTHER_012/resource.json").is_none() {
        return;
    }
    let shared = load_native_class_ui_assets().expect("loadNativeClassUIAssets").unwrap();
    let shops = load_native_shop_ui_assets(Some(&shared)).expect("loadNativeShopUIAssets").unwrap();
    for variant in [1, 3, 5] {
        assert!(shops.shops.contains_key(&variant), "variant {variant} was not preflighted");
    }
}

/// Go：`native_shop_ui_test.go:1100` `TestNativeShopProductionOwnerFailsClosedForCustomVariant`
#[test]
fn test_native_shop_production_owner_fails_closed_for_custom_variant() {
    let c = campaign::Campaign {
        start: String::from("shop"),
        nodes: BTreeMap::from([(
            String::from("shop"),
            campaign::Node { r#type: String::from("shop"), native_hub_variant: 0, ..Default::default() },
        )]),
        ..Default::default()
    };
    let mut g = Game { camp: Some(campaign::new_runner(&c)), ..Default::default() };
    assert!(
        !g.setup_native_shop() && g.native_shop_mode.is_empty(),
        "custom shop node was incorrectly claimed by original shop owner"
    );
}

#[test]
fn chapter_two_secret_shop_purchase_and_save() {
    let Ok(path) = std::env::var("FD2_SHOP_PROBE_SAVE") else { return };
    let name = "chapter_two_secret_shop_purchase_and_save";
    if f_child_env(module_path!(), name, |dir| {
        vec![
            ("XDG_DATA_HOME", dir.into()),
            ("FD2_TITLE", "1".into()),
            ("FD2_MUTE", "1".into()),
            ("FD2_CAMPAIGN", "canonical".into()),
            ("FD2_NATIVE_SAVE", "".into()),
        ]
    })
    .is_none()
    {
        return;
    }

    let mut game = load_game().unwrap();
    assert!(game.load_err.is_empty(), "{}", game.load_err);
    game.load_native_game_from_slot(&path, 0).unwrap();
    game.title_phase.clear();
    assert_eq!(game.camp.as_ref().unwrap().node_id(), "town_ch02");
    assert!(game.camp.as_mut().unwrap().confirm_native_town_secret(5));
    game.enter_node();
    let mut screen = Picture::new_image(640, 400);
    let mut clock = 1_000_000_000;
    let drain = |game: &mut Game, screen: &mut Picture, clock: &mut i64| {
        for _ in 0..700 {
            if game.native_shop_ui_job.is_none() {
                return;
            }
            assert!(game.draw_native_shop(screen));
            *clock += 16_666_667;
            game.step_native_shop_ui_lifecycle(GameInstant(*clock));
        }
        panic!("shop did not finish presenting");
    };
    drain(&mut game, &mut screen, &mut clock);
    let confirm = TickInput::default();
    let key = |key| TickInput { pressed: [key].into(), ..Default::default() };
    let initial_gold = game.gold;
    let original = game.party_roster.clone();

    game.handle_native_shop_input(&confirm, true);
    game.handle_native_shop_input(&key(Key::ArrowRight), false);
    game.handle_native_shop_input(&confirm, true);
    assert_eq!(game.native_shop_mode, "confirm");
    game.handle_native_shop_input(&confirm, true);
    assert_eq!(game.native_shop_mode, "insufficient");
    assert!(game.draw_native_shop(&mut screen));
    game.handle_native_shop_input(&confirm, true);
    assert_eq!(game.gold, initial_gold);
    assert_eq!(game.party_roster, original);

    game.handle_native_shop_input(&key(Key::ArrowDown), false);
    game.handle_native_shop_input(&confirm, true);
    game.handle_native_shop_input(&key(Key::Escape), false);
    assert_eq!(game.native_shop_mode, "purchase");
    assert_eq!(game.party_roster, original);
    game.handle_native_shop_input(&confirm, true);
    game.handle_native_shop_input(&confirm, true);
    assert_eq!(game.native_shop_mode, "recipient_consumable");
    let recipient = game.shop_recipients[0];
    let full = game.party_roster.get_mut(&recipient).unwrap();
    while full.inventory.len() < 8 {
        assert!(full.add_inventory_item(192, false));
    }
    let full = full.clone();
    game.handle_native_shop_input(&confirm, true);
    assert_eq!(game.native_shop_mode, "recipient_full");
    assert_eq!(game.party_roster[&recipient], full);
    assert_eq!(game.gold, initial_gold);
    game.handle_native_shop_input(&confirm, true);
    game.party_roster.insert(recipient, original[&recipient].clone());
    game.handle_native_shop_input(&confirm, true);
    assert_eq!(game.native_shop_mode, "success", "{}", game.msg);
    assert_eq!(game.gold, initial_gold);
    assert_eq!(game.party_roster, original);
    drain(&mut game, &mut screen, &mut clock);
    assert_eq!(game.native_shop_mode, "purchase");
    assert_eq!(game.gold, initial_gold - 10);
    assert_eq!(game.party_roster[&recipient].inventory.len(), original[&recipient].inventory.len() + 1);
    assert_eq!(game.party_roster[&recipient].inventory.last(), Some(&192));

    game.handle_native_shop_input(&key(Key::ArrowUp), false);
    game.handle_native_shop_input(&key(Key::ArrowLeft), false);
    game.handle_native_shop_input(&confirm, true);
    game.handle_native_shop_input(&confirm, true);
    assert_eq!(game.native_shop_mode, "recipient_equipment");
    let recipient = game.shop_recipients[0];
    game.handle_native_shop_input(&confirm, true);
    assert_eq!(game.native_shop_mode, "equip_confirm");
    game.handle_native_shop_input(&key(Key::ArrowRight), false);
    game.handle_native_shop_input(&confirm, true);
    assert_eq!(game.native_shop_mode, "success", "{}", game.msg);
    drain(&mut game, &mut screen, &mut clock);
    assert_eq!(game.gold, initial_gold - 260);
    assert_eq!(game.party_roster[&recipient].inventory.last(), Some(&1));
    assert_eq!(game.party_roster[&recipient].equipped.last(), Some(&false));

    game.handle_native_shop_input(&confirm, true);
    game.handle_native_shop_input(&confirm, true);
    game.handle_native_shop_input(&confirm, true);
    assert_eq!(game.native_shop_mode, "equip_confirm");
    game.handle_native_shop_input(&confirm, true);
    assert_eq!(game.native_shop_mode, "success", "{}", game.msg);
    drain(&mut game, &mut screen, &mut clock);
    assert_eq!(game.gold, initial_gold - 510);
    assert_eq!(game.party_roster[&recipient].inventory.last(), Some(&1));
    assert_eq!(game.party_roster[&recipient].equipped.last(), Some(&true));

    let before_sale = game.party_roster.clone();
    let funds = game.gold;
    let roster_position = game.party_join_order.iter().position(|id| *id == recipient).unwrap();
    game.handle_native_shop_input(&key(Key::Escape), false);
    drain(&mut game, &mut screen, &mut clock);
    game.handle_native_shop_input(&key(Key::ArrowRight), false);
    game.handle_native_shop_input(&confirm, true);
    assert_eq!(game.native_shop_mode, "sell_roster", "{}", game.msg);
    drain(&mut game, &mut screen, &mut clock);
    let first = game.party_join_order[0];
    let empty = game.party_roster.get_mut(&first).unwrap();
    while !empty.inventory.is_empty() {
        battle::remove_native_compact_inventory(Some(empty), 0).unwrap();
    }
    game.handle_native_shop_input(&confirm, true);
    assert_eq!(game.native_shop_mode, "sell_empty", "{}", game.msg);
    drain(&mut game, &mut screen, &mut clock);
    game.handle_native_shop_input(&confirm, true);
    drain(&mut game, &mut screen, &mut clock);
    assert_eq!(game.native_shop_mode, "sell_roster");
    assert_eq!(game.gold, funds);
    game.party_roster.insert(first, before_sale[&first].clone());
    for _ in 0..roster_position {
        game.handle_native_shop_input(&key(Key::ArrowRight), false);
    }
    let select_last = |game: &mut Game, screen: &mut Picture, clock: &mut i64| {
        game.handle_native_shop_input(&confirm, true);
        assert_eq!(game.native_shop_mode, "sell_items", "{}", game.msg);
        drain(game, screen, clock);
        for _ in 1..game.native_shop_sell_item_ids.len() {
            game.handle_native_shop_input(&key(Key::ArrowRight), false);
        }
        game.handle_native_shop_input(&confirm, true);
        assert_eq!(game.native_shop_mode, "sell_confirm", "{}", game.msg);
        drain(game, screen, clock);
    };
    select_last(&mut game, &mut screen, &mut clock);
    game.handle_native_shop_input(&key(Key::ArrowRight), false);
    game.handle_native_shop_input(&confirm, true);
    drain(&mut game, &mut screen, &mut clock);
    assert_eq!(game.native_shop_mode, "sell_roster");
    assert_eq!(game.shop_sell_unit_sel, roster_position as i64);
    assert_eq!(game.party_roster, before_sale);
    assert_eq!(game.gold, funds);

    select_last(&mut game, &mut screen, &mut clock);
    game.handle_native_shop_input(&confirm, true);
    assert_eq!(game.native_shop_mode, "sell_success", "{}", game.msg);
    assert_eq!(game.gold, funds);
    assert_eq!(game.party_roster, before_sale);
    assert!(game.save_game_to_slot(0).is_err());
    for _ in 0..700 {
        assert!(game.draw_native_shop(&mut screen));
        clock += 16_666_667;
        game.step_native_shop_ui_lifecycle(GameInstant(clock));
        if game.gold != funds {
            break;
        }
    }
    assert_eq!(game.gold, funds + 187);
    assert_eq!(game.party_roster, before_sale);
    assert!(game.save_game_to_slot(0).is_err());
    drain(&mut game, &mut screen, &mut clock);
    assert_eq!(game.native_shop_mode, "sell_roster");
    let sold = &game.party_roster[&recipient];
    let before = &before_sale[&recipient];
    assert_eq!(&sold.inventory[..], &before.inventory[..before.inventory.len() - 1]);
    assert_eq!(sold.equipped.last(), Some(&false));
    assert_ne!(sold.ap, before.ap);
    let free = sold.inventory.len();
    assert_eq!(sold.native_inventory_flags[free] & 0x80, 0x80);
    game.handle_native_shop_input(&key(Key::Escape), false);
    drain(&mut game, &mut screen, &mut clock);

    game.handle_native_shop_input(&key(Key::ArrowRight), false);
    game.handle_native_shop_input(&confirm, true);
    assert_eq!(game.native_shop_mode, "equip_roster", "{}", game.msg);
    drain(&mut game, &mut screen, &mut clock);
    for _ in 0..roster_position {
        game.handle_native_shop_input(&key(Key::ArrowRight), false);
    }
    game.handle_native_shop_input(&confirm, true);
    assert_eq!(game.native_shop_mode, "equip_panel", "{}", game.msg);
    drain(&mut game, &mut screen, &mut clock);
    game.handle_native_shop_input(&key(Key::ArrowUp), false);
    game.handle_native_shop_input(&confirm, true);
    assert_eq!(game.party_roster[&recipient].equipped.last(), Some(&true), "{}", game.msg);
    assert_eq!(game.party_roster[&recipient].ap, before.ap);
    assert_eq!(game.gold, funds + 187);
    game.handle_native_shop_input(&key(Key::Escape), false);
    drain(&mut game, &mut screen, &mut clock);
    assert_eq!(game.native_shop_mode, "equip_roster");
    game.handle_native_shop_input(&key(Key::Escape), false);
    drain(&mut game, &mut screen, &mut clock);

    game.handle_native_shop_input(&key(Key::ArrowRight), false);
    game.handle_native_shop_input(&confirm, true);
    assert_eq!(game.native_shop_mode, "transfer_intro", "{}", game.msg);
    drain(&mut game, &mut screen, &mut clock);
    game.handle_native_shop_input(&confirm, true);
    assert_eq!(game.native_shop_mode, "transfer_source", "{}", game.msg);
    drain(&mut game, &mut screen, &mut clock);
    game.handle_native_shop_input(&confirm, true);
    assert_eq!(game.native_shop_mode, "transfer_items", "{}", game.msg);
    drain(&mut game, &mut screen, &mut clock);
    let source = game.party_join_order[0];
    let destination = game.party_join_order[1];
    let source_before = game.party_roster[&source].inventory.clone();
    let destination_before = game.party_roster[&destination].inventory.len();
    let compact = game.native_shop_transfer_items[0] as usize;
    game.handle_native_shop_input(&confirm, true);
    assert_eq!(game.native_shop_mode, "transfer_dest_prompt", "{}", game.msg);
    drain(&mut game, &mut screen, &mut clock);
    game.handle_native_shop_input(&confirm, true);
    assert_eq!(game.native_shop_mode, "transfer_dest", "{}", game.msg);
    drain(&mut game, &mut screen, &mut clock);
    game.handle_native_shop_input(&key(Key::ArrowRight), false);
    game.handle_native_shop_input(&confirm, true);
    assert_eq!(game.native_shop_mode, "transfer_intro", "{}", game.msg);
    assert_eq!(game.party_roster[&source].inventory.len(), source_before.len() - 1);
    assert_eq!(game.party_roster[&destination].inventory.len(), destination_before + 1);
    assert_eq!(game.party_roster[&destination].inventory.last(), Some(&source_before[compact]));
    assert_eq!(game.party_roster[&destination].equipped.last(), Some(&false));
    assert_eq!(game.gold, funds + 187);
    drain(&mut game, &mut screen, &mut clock);
    game.handle_native_shop_input(&confirm, true);
    drain(&mut game, &mut screen, &mut clock);
    game.handle_native_shop_input(&key(Key::Escape), false);
    drain(&mut game, &mut screen, &mut clock);
    game.handle_native_shop_input(&key(Key::Escape), false);
    game.save_game_to_slot(0).unwrap();
    let save = game.build_save_data();
    assert_eq!(save.node, "town_ch02");
    assert!(game.load_json_game_from_slot(0, false));
    assert_eq!(game.build_save_data().gold, save.gold);
    assert_eq!(game.build_save_data().party_roster, save.party_roster);

    for variant in [1, 3] {
        let prior_gold = game.gold;
        game.camp_sel = variant;
        game.tick_input = key(Key::Enter);
        assert!(game.camp_input());
        assert_eq!(game.native_shop_variant, variant);
        drain(&mut game, &mut screen, &mut clock);
        assert!(game.draw_native_shop(&mut screen));
        game.tick_input = key(Key::ArrowRight);
        assert!(game.camp_input());
        game.tick_input = key(Key::Enter);
        assert!(game.camp_input());
        assert_eq!(game.native_shop_mode, "sell_roster", "{}", game.msg);
        drain(&mut game, &mut screen, &mut clock);
        game.camp_input();
        assert_eq!(game.native_shop_mode, "sell_items", "{}", game.msg);
        drain(&mut game, &mut screen, &mut clock);
        game.camp_input();
        assert_eq!(game.native_shop_mode, "sell_confirm", "{}", game.msg);
        drain(&mut game, &mut screen, &mut clock);
        game.camp_input();
        assert_eq!(game.native_shop_mode, "sell_success", "{}", game.msg);
        drain(&mut game, &mut screen, &mut clock);
        assert!(game.gold > prior_gold);
        game.tick_input = key(Key::Escape);
        game.camp_input();
        drain(&mut game, &mut screen, &mut clock);
        game.camp_input();
        assert_eq!(game.camp.as_ref().unwrap().node_id(), "town_ch02");
    }
}
