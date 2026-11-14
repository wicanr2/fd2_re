//! Go 來源：`remake/cmd/fd2/save_test.go`（12 個測試，逐條移植，期望值不改）。
//!
//! 經過 A 組 `userDataPath`／`assetPath`／`localeMessage`、C 組 `currentNativeSystemOptions` 等函式的
//! 測試先標 `#[ignore = "等 … 合併"]`。Go 以 `t.Setenv` 設環境變數的測試在子行程執行（`f_child_env`）；
//! Go 的 `userDataDirCached = ""` 在子行程裡本來就是空的。

use super::*;

/// Go 測試 helper `attachOfficialLocale`（`locale_fixture_test.go:11`）。
pub(crate) fn attach_official_locale(g: &mut Game) {
    let locale_id = "zh-Hant";
    let catalog = load_official_locale(locale_id).unwrap_or_else(|e| panic!("載入官方語系 {locale_id}：{e}"));
    let content =
        load_official_locale_content(locale_id).unwrap_or_else(|e| panic!("載入官方全量內容 {locale_id}：{e}"));
    let entities =
        load_official_locale_entities(locale_id).unwrap_or_else(|e| panic!("載入官方實體名稱 {locale_id}：{e}"));
    g.locale_id = locale_id.to_string();
    g.locale_catalog = catalog;
    g.locale_content = content;
    g.locale_entities = entities;
}

fn unit(f: impl FnOnce(&mut battle::Unit)) -> battle::Unit {
    let mut u = battle::Unit::default();
    f(&mut u);
    u
}

fn node(t: &str) -> campaign::Node {
    campaign::Node { r#type: t.to_string(), ..Default::default() }
}

fn graph(start: &str, nodes: Vec<(&str, campaign::Node)>) -> campaign::Campaign {
    campaign::Campaign {
        start: start.to_string(),
        nodes: nodes.into_iter().map(|(k, v)| (k.to_string(), v)).collect(),
        ..Default::default()
    }
}

fn write_save_slot(slot: i64, raw: &[u8]) {
    let path = save_slot_path(slot);
    std::fs::create_dir_all(std::path::Path::new(&path).parent().unwrap()).unwrap();
    std::fs::write(&path, raw).unwrap();
}

/// Go：`save_test.go:17`
#[test]
fn start_native_slot_preserves_container_and_rejects_missing_provenance() {
    let name = "start_native_slot_preserves_container_and_rejects_missing_provenance";
    let Some(dir) = f_child_env(module_path!(), name, |d| vec![("FD2_NATIVE_SAVE", d.join("FD2.SAV").into())]) else {
        return;
    };
    let plain = vec![0x5a; fdsave::FILE_SIZE];
    let stored = fdsave::encode(&plain).unwrap();
    let path = dir.join("FD2.SAV");
    std::fs::write(&path, &stored).unwrap();
    let mut g = Game { handler_chapter: 1, gold: 1000, ..Default::default() };
    if g.save_native_chapter_slot(2).is_ok() || g.native_chapter_slot_baseline.is_some() {
        panic!("沒有 START／LOAD 來源卻發布保存基底");
    }
    g.native_chapter_slot_started = true;
    std::fs::write(&path, &stored[..5]).unwrap();
    if g.save_native_chapter_slot(2).is_ok()
        || g.native_chapter_slot_baseline.is_some()
        || !g.native_chapter_slot_plain.is_empty()
    {
        panic!("壞容器沒有零發布拒收");
    }
    if std::fs::read(&path).unwrap() != stored[..5] {
        panic!("拒收改動了容器");
    }
    std::fs::write(&path, &stored).unwrap();
    let join =
        campaign::load_native_join_constructor_table(&asset_path("assets/data/native_join_constructor.json")).unwrap();
    let items =
        battle::load_native_item_effect_row_prefix(&asset_path("assets/data/native_item_effect_rows.json")).unwrap();
    let u = join.materialize_persistent_unit(0, &unit(|u| u.fig = 0), &items).unwrap();
    g.party_roster = BTreeMap::from([(0, u)]);
    g.party_join_order = vec![0];
    g.save_native_chapter_slot(2).unwrap();
    let raw = std::fs::read(&path).unwrap();
    let result = fdsave::decode(&raw).unwrap();
    let (start, end) = fdsave::slot_bounds(2).unwrap();
    if result[..start] != plain[..start]
        || result[end..fdsave::CHECKSUM_OFF] != plain[end..fdsave::CHECKSUM_OFF]
        || result[start + fdsave::ROSTER_SIZE + 10..end] != plain[start + fdsave::ROSTER_SIZE + 10..end]
    {
        panic!("START 保存改動了其他槽、目前戰況或 metadata 尾段");
    }
    let snapshot = fdsave::inspect_chapter_slot(&result, 2);
    let ok = snapshot.as_ref().is_ok_and(|s| {
        s.verified.chapter == 1
            && s.verified.roster_count == 1
            && s.verified.currency == 1000
            && s.records[0].raw[8] == 0
    });
    if !ok || g.native_chapter_slot_baseline.is_none() || g.native_chapter_slot_started {
        panic!("START 保存發布不完整：{snapshot:?}");
    }
    g.save_native_chapter_slot(2).unwrap();
    if std::fs::read(&path).unwrap() != raw {
        panic!("相同狀態重存不一致");
    }
}

/// Go：`save_test.go:85`
#[test]
fn save_data_round_trips_persistent_party() {
    let want = SaveData {
        node: "story_ch02".to_string(),
        flags: Some(BTreeMap::from([("won_ch01".to_string(), true)])),
        gold: 321,
        party_members: BTreeMap::from([(0, true), (9, true)]),
        party_join_order: vec![0, 9],
        party_deploy: BTreeMap::from([(0, true), (9, false)]),
        party_roster: BTreeMap::from([(
            9,
            unit(|u| {
                u.fig = 9;
                u.name = "悠妮".to_string();
                u.lv = 4;
                u.hp = 23;
                u.max_hp = 37;
                u.mp = 18;
                u.max_mp = 24;
                u.exp = 67.5;
                u.spells = vec![0, 4, 13];
                u.inventory = vec![0xc6, 0x64];
            }),
        )]),
        chapter: 1,
        native_hud_gate_a: Some(0),
        ..Default::default()
    };
    // Go `json.Marshal`（不縮排）再 `json.Unmarshal`。
    let raw = serde_json::to_vec(&want).unwrap();
    let got: SaveData = go_json_unmarshal_zero(&raw).unwrap();
    let yuni = got.party_roster.get(&9);
    if yuni.is_none()
        || got.node != want.node
        || got.chapter != 1
        || got.native_hud_gate_a != Some(0)
        || !got.party_members.get(&0).copied().unwrap_or(false)
        || got.party_join_order.len() != 2
        || !got.party_deploy.get(&0).copied().unwrap_or(false)
        || got.party_deploy.get(&9).copied().unwrap_or(false)
    {
        panic!("campaign progress did not round-trip: {got:?}");
    }
    let yuni = yuni.unwrap();
    if yuni.fig != 9
        || yuni.lv != 4
        || yuni.hp != 23
        || yuni.max_hp != 37
        || yuni.mp != 18
        || yuni.max_mp != 24
        || yuni.exp != 67.5
        || yuni.spells.len() != 3
        || yuni.inventory.len() != 2
        || yuni.inventory[0] != 0xc6
        || yuni.inventory[1] != 0x64
    {
        panic!("party roster did not round-trip: {yuni:?}");
    }
}

/// Go：`save_test.go:114`
#[test]
fn save_slots_are_bounded_and_distinct() {
    let name = "save_slots_are_bounded_and_distinct";
    let envs = |d: &std::path::Path| vec![("XDG_DATA_HOME", d.into()), ("FD2_MUTE", "1".into())];
    if f_child_env(module_path!(), name, envs).is_none() {
        return;
    }
    if save_slot_path(-1) != save_slot_path(0) || save_slot_path(4) != save_slot_path(0) {
        panic!("invalid save slot did not fail closed to slot 0");
    }
    let base = std::path::Path::new(&save_slot_path(3)).file_name().map(|n| n.to_string_lossy().into_owned());
    if save_slot_path(0) == save_slot_path(1) || base.as_deref() != Some("fd2_save_3.json") {
        panic!("save slot paths are not distinct/bounded: {:?} {:?}", save_slot_path(0), save_slot_path(3));
    }
}

/// Go：`save_test.go:125`
#[test]
fn save_rejects_post_battle_handler_without_serializable_runtime_context() {
    let mut post = node("cutscene");
    post.handler_binding = "assets/cutscenes/bindings/ch01_post.json".to_string();
    post.next = "choice".to_string();
    let c = graph("post", vec![("post", post), ("choice", node("choice"))]);
    let catalog = load_official_locale("zh-Hant").unwrap();
    let mut st = battle::State::default();
    st.units = vec![unit(|u| u.fig = 0)];
    let mut g =
        Game { camp: Some(campaign::new_runner(&c)), st: Some(st), locale_catalog: catalog, ..Default::default() };
    g.save_game();
    if g.msg != "戰後演出進行中，請在下一個節點存檔" {
        panic!("unsafe postbattle save was not rejected: {:?}", g.msg);
    }
}

/// Go：`save_test.go:141`
#[test]
fn save_rejects_unbound_postbattle_boundary() {
    let mut post = node("cutscene");
    post.next = "town_ch05".to_string();
    let c = graph("postbattle_ch04_persist", vec![("postbattle_ch04_persist", post), ("town_ch05", node("town"))]);
    let catalog = load_official_locale("zh-Hant").unwrap();
    let mut g = Game { camp: Some(campaign::new_runner(&c)), locale_catalog: catalog, ..Default::default() };
    g.save_game();
    if g.msg != "戰後演出進行中，請在下一個節點存檔" {
        panic!("unbound postbattle save was not rejected: {:?}", g.msg);
    }
}

/// Go：`save_test.go:157`
#[test]
fn write_save_file_replaces_complete_json_atomically() {
    let dir = crate::test_support::temp_dir("write-save-file");
    let path = dir.join("fd2_save.json");
    std::fs::write(&path, br#"{"node":"old"}"#).unwrap();
    let want = br#"{"node":"town_ch04","gold":99}"#;
    write_save_file(&path.to_string_lossy(), want).unwrap();
    let got = std::fs::read(&path).unwrap();
    if got != want {
        panic!("save contents = {:?}, want {:?}", String::from_utf8_lossy(&got), String::from_utf8_lossy(want));
    }
    for entry in std::fs::read_dir(&dir).unwrap() {
        let name = entry.unwrap().file_name().to_string_lossy().into_owned();
        if name.starts_with(".fd2-save-") {
            panic!("temporary save file leaked: {name}");
        }
    }
}

/// Go：`save_test.go:184`
#[test]
#[ignore = "等 E 合併（停在 reset_native_class_ui_pulse；之後可能還要 B、D）"]
fn campaign_save_load_restores_town_boundary_and_party() {
    let name = "campaign_save_load_restores_town_boundary_and_party";
    if f_child_env(module_path!(), name, |d| vec![("XDG_DATA_HOME", d.into())]).is_none() {
        return;
    }
    let mut church = node("church");
    church.next = "town_ch02".to_string();
    let mut c = graph("town_ch02", vec![("town_ch02", node("town")), ("church_ch02", church)]);
    c.flags = BTreeMap::new();
    let u = unit(|u| {
        u.name = "悠妮".to_string();
        u.portrait = 0x34;
        u.class_id = 21;
        u.lv = 20;
        u.hp = 44;
        u.max_hp = 44;
        u.mv = 7;
        u.inventory = vec![0x64];
    });
    let mut g = Game {
        camp: Some(campaign::new_runner(&c)),
        gold: 279,
        items: vec!["sky-key".to_string()],
        handler_chapter: 2,
        party_members: BTreeMap::from([(0, true)]),
        party_join_order: vec![0],
        party_deploy: BTreeMap::from([(0, true)]),
        party_roster: BTreeMap::from([(0, u.clone())]),
        native_map_hud_persistent: battle::NativeMapHUDPersistentState {
            display_gate_a: 0,
            anchor_x: 0xf2,
            has_display_gate_a: true,
            has_anchor_x: true,
        },
        ..Default::default()
    };
    attach_official_locale(&mut g);
    let mut options = g.current_native_system_options();
    options.raw_53af9 = 1;
    options.raw_51e61 = 0;
    options.raw_51e62 = 0;
    g.native_system_options = Some(options);
    let _ = g.save_game_to_slot(2);
    if g.msg != "已存檔（槽位 3：town_ch02）" {
        panic!("save boundary message={:?}", g.msg);
    }
    g.camp.as_mut().unwrap().cur = "church_ch02".to_string();
    g.gold = 1;
    g.items = Vec::new();
    g.party_members = BTreeMap::new();
    g.party_join_order = Vec::new();
    g.party_deploy = BTreeMap::new();
    g.party_roster = BTreeMap::new();
    g.native_map_hud_persistent = battle::initial_native_map_hud_persistent_state();
    g.native_system_options = None;
    g.shop_mode = "sell".to_string();
    g.shop_picking = true;
    g.shop_equip_prompt = true;
    g.shop_recipients = vec![9];
    g.native_shop_ui_job = Some(NativeClassUIJob { frames: vec![vec![1]], ..Default::default() });
    g.native_shop_mode = "transfer_full".to_string();
    g.native_shop_variant = 5;
    g.native_shop_has_pending_unit = true;
    g.native_shop_sell_item_ids = vec![0x64];
    g.native_shop_transfer_source = 9;
    g.native_shop_transfer_items = vec![0];
    g.native_shop_transfer_ids = vec![9];
    g.native_item_panel_base = vec![1];
    g.native_item_panel_record = vec![1];
    g.native_item_effect_rows = vec![1];
    g.item_anim_step = 11;
    g.item_closing = true;
    g.native_player_status = Some(NativePlayerStatusState { steady: vec![1], ..Default::default() });
    let callback_ran = std::sync::Arc::new(std::sync::atomic::AtomicBool::new(false));
    let flag = callback_ran.clone();
    g.native_level_up_dialogue = Some(NativeLevelUpDialogueState {
        then: Some(Box::new(move |_: &mut Game| flag.store(true, std::sync::atomic::Ordering::SeqCst))),
        ..Default::default()
    });
    // Go 的 `unit: &u` 指向測試的區域變數；Rust 以 UnitIdx 指向戰場單位，這裡沒有戰場，留空。
    g.pending_native_level_ups = vec![PendingNativeLevelUp { unit: None, exp: 1, ..Default::default() }];
    g.native_move_plan = Some(battle::NativePlayerMovement::default());
    g.load_game_from_slot(2);
    if g.native_level_up_dialogue.is_some()
        || !g.pending_native_level_ups.is_empty()
        || g.native_move_plan.is_some()
        || callback_ran.load(std::sync::atomic::Ordering::SeqCst)
    {
        panic!("LOAD retained discarded battle owners or ran their continuation");
    }
    let camp = g.camp.as_ref().unwrap();
    if camp.node_id() != "town_ch02" || g.gold != 279 || g.items.len() != 1 || g.items[0] != "sky-key" {
        panic!("campaign boundary did not restore: node={:?} gold={} items={:?}", camp.node_id(), g.gold, g.items);
    }
    let got = g.party_roster.get(&0);
    if !got.is_some_and(|got| got.portrait == 0x34 && got.class_id == 21 && got.mv == 7 && got.hp == 44)
        || g.party_join_order.len() != 1
        || !g.party_deploy.get(&0).copied().unwrap_or(false)
    {
        panic!(
            "persistent party did not restore: roster={:?} join={:?} deploy={:?}",
            g.party_roster, g.party_join_order, g.party_deploy
        );
    }
    let hud = g.native_map_hud_persistent;
    if !hud.has_display_gate_a || hud.display_gate_a != 0 || !hud.has_anchor_x || hud.anchor_x != 1 {
        panic!("save HUD persistence={hud:?}");
    }
    let got_options = g.current_native_system_options();
    if got_options.raw_53af9 != 1
        || got_options.raw_51aab != 0
        || got_options.raw_51e61 != 0
        || got_options.raw_51e62 != 0
    {
        panic!("save system options={got_options:?}");
    }
    if g.st.is_some()
        || g.sel.is_some()
        || g.native_player_status.is_some()
        || !g.shop_mode.is_empty()
        || !g.church_mode.is_empty()
        || g.shop_picking
        || g.shop_equip_prompt
        || !g.shop_recipients.is_empty()
        || g.native_shop_ui_job.is_some()
        || !g.native_shop_mode.is_empty()
        || g.native_shop_variant != 0
        || g.native_shop_has_pending_unit
        || !g.native_shop_sell_item_ids.is_empty()
        || g.native_shop_transfer_source != -1
        || !g.native_shop_transfer_items.is_empty()
        || !g.native_shop_transfer_ids.is_empty()
        || !g.native_item_panel_base.is_empty()
        || !g.native_item_panel_record.is_empty()
        || !g.native_item_effect_rows.is_empty()
        || g.item_anim_step != 0
        || g.item_closing
    {
        panic!(
            "town boundary retained transient scene state: shop={:?} nativeShop={:?} church={:?}",
            g.shop_mode, g.native_shop_mode, g.church_mode
        );
    }
}

/// Go：`save_test.go:268`
#[test]
fn campaign_load_rejects_out_of_range_native_hud_gate_without_mutation() {
    let name = "campaign_load_rejects_out_of_range_native_hud_gate_without_mutation";
    if f_child_env(module_path!(), name, |d| vec![("XDG_DATA_HOME", d.into())]).is_none() {
        return;
    }
    let c = graph("town", vec![("town", node("town"))]);
    let raw = serde_json::to_vec(&SaveData {
        node: "town".to_string(),
        gold: 99,
        native_hud_gate_a: Some(256),
        ..Default::default()
    })
    .unwrap();
    write_save_slot(0, &raw);
    let mut g = Game {
        camp: Some(campaign::new_runner(&c)),
        gold: 7,
        native_map_hud_persistent: battle::initial_native_map_hud_persistent_state(),
        ..Default::default()
    };
    g.load_game_from_slot(0);
    if g.gold != 7
        || g.native_map_hud_persistent.display_gate_a != 1
        || g.msg != "存檔 native HUD gate A 超出原始布林範圍"
    {
        panic!("invalid HUD save mutated game: gold={} HUD={:?} msg={:?}", g.gold, g.native_map_hud_persistent, g.msg);
    }
}

/// Go：`save_test.go:297`
#[test]
fn campaign_load_rejects_out_of_range_native_system_option_without_mutation() {
    let name = "campaign_load_rejects_out_of_range_native_system_option_without_mutation";
    if f_child_env(module_path!(), name, |d| vec![("XDG_DATA_HOME", d.into())]).is_none() {
        return;
    }
    let c = graph("town", vec![("town", node("town"))]);
    let raw = serde_json::to_vec(&SaveData {
        node: "town".to_string(),
        gold: 99,
        native_raw_53af9: Some(2),
        ..Default::default()
    })
    .unwrap();
    write_save_slot(0, &raw);
    let status = NativePlayerStatusState { steady: vec![1], ..Default::default() };
    let movement = battle::NativePlayerMovement::default();
    let mut g = Game {
        camp: Some(campaign::new_runner(&c)),
        gold: 7,
        native_player_status: Some(status.clone()),
        native_level_up_dialogue: Some(NativeLevelUpDialogueState::default()),
        native_move_plan: Some(movement.clone()),
        ..Default::default()
    };
    g.load_game_from_slot(0);
    // Go 以指標相等確認沒有換掉；Rust 以「仍然存在且內容相同」確認。
    if g.native_level_up_dialogue.is_none() || g.native_move_plan.as_ref() != Some(&movement) {
        panic!("rejected LOAD discarded current battle owners");
    }
    if g.gold != 7
        || g.native_system_options.is_some()
        || g.native_player_status.as_ref() != Some(&status)
        || g.msg != "存檔 native 系統設定超出原始布林範圍"
    {
        panic!(
            "invalid system option mutated game: gold={} options={:?} msg={:?}",
            g.gold, g.native_system_options, g.msg
        );
    }
}

/// Go：`save_test.go:329`
#[test]
fn campaign_load_rejects_broken_persistent_party_topology_without_mutation() {
    let name = "campaign_load_rejects_broken_persistent_party_topology_without_mutation";
    if f_child_env(module_path!(), name, |d| vec![("XDG_DATA_HOME", d.into())]).is_none() {
        return;
    }
    let c = graph("town", vec![("town", node("town")), ("preparation", node("preparation"))]);
    let broken = SaveData {
        node: "preparation".to_string(),
        gold: 999,
        party_members: BTreeMap::from([(0, true)]),
        party_join_order: vec![0, 0],
        party_deploy: BTreeMap::from([(0, true)]),
        party_roster: BTreeMap::from([(0, unit(|u| u.name = "索爾".to_string()))]),
        ..Default::default()
    };
    write_save_slot(0, &serde_json::to_vec(&broken).unwrap());
    let want_roster = BTreeMap::from([(
        9,
        unit(|u| {
            u.name = "悠妮".to_string();
            u.hp = 17;
        }),
    )]);
    let mut g = Game {
        camp: Some(campaign::new_runner(&c)),
        gold: 7,
        party_members: BTreeMap::from([(9, true)]),
        party_join_order: vec![9],
        party_deploy: BTreeMap::from([(9, true)]),
        party_roster: want_roster.clone(),
        ..Default::default()
    };
    g.load_game_from_slot(0);
    if g.camp.as_ref().unwrap().node_id() != "town"
        || g.gold != 7
        || g.party_join_order != [9]
        || g.party_roster != want_roster
        || g.msg != "存檔持續隊伍的 JOIN 順序重複角色 0"
    {
        panic!(
            "broken party save mutated state: node={:?} gold={} order={:?} msg={:?}",
            g.camp.as_ref().unwrap().node_id(),
            g.gold,
            g.party_join_order,
            g.msg
        );
    }
}

/// Go：`save_test.go:370`
#[test]
#[ignore = "等 E 合併（停在 reset_native_class_ui_pulse）"]
fn title_json_load_accepts_saved_zero_chapter_and_keeps_bytes() {
    let name = "title_json_load_accepts_saved_zero_chapter_and_keeps_bytes";
    let envs = |d: &std::path::Path| vec![("XDG_DATA_HOME", d.into()), ("FD2_NATIVE_SAVE", "".into())];
    if f_child_env(module_path!(), name, envs).is_none() {
        return;
    }
    let c = graph("saved_boundary", vec![("saved_boundary", node("story")), ("other", node("story"))]);
    let mut writer = Game { camp: Some(campaign::new_runner(&c)), gold: 279, ..Default::default() };
    writer.save_game_to_slot(0).unwrap();
    let before = std::fs::read(save_slot_path(0)).unwrap();
    if String::from_utf8_lossy(&before).contains(r#""chapter""#) {
        panic!("zero chapter save changed format: {}", String::from_utf8_lossy(&before));
    }
    let mut cold = Game {
        camp: Some(campaign::new_runner(&c)),
        title_phase: "loadslots".to_string(),
        gold: 7,
        handler_chapter: 9,
        ..Default::default()
    };
    attach_official_locale(&mut cold);
    cold.camp.as_mut().unwrap().cur = "other".to_string();
    if !cold.confirm_title_load_slot(0)
        || !cold.title_phase.is_empty()
        || cold.camp.as_ref().unwrap().node_id() != "saved_boundary"
        || cold.gold != 279
        || cold.handler_chapter != 0
    {
        panic!(
            "saved zero chapter refused: title={:?} node={:?} gold={} chapter={} error={:?}",
            cold.title_phase,
            cold.camp.as_ref().unwrap().node_id(),
            cold.gold,
            cold.handler_chapter,
            cold.load_err
        );
    }
    let after = std::fs::read(save_slot_path(0)).unwrap();
    if before != after {
        panic!("LOAD rewrote the input save");
    }
}

/// Go：`save_test.go:398`
#[test]
fn title_json_load_rejects_before_leaving_selector() {
    let name = "title_json_load_rejects_before_leaving_selector";
    let envs = |d: &std::path::Path| vec![("XDG_DATA_HOME", d.into()), ("FD2_NATIVE_SAVE", "".into())];
    if f_child_env(module_path!(), name, envs).is_none() {
        return;
    }
    let c = graph("safe", vec![("safe", node("town"))]);
    // Go 以 map 的隨機順序跑子測試；Rust 依名稱排序，子測試彼此獨立。
    for (name, data) in [
        ("negative chapter", SaveData { node: "safe".into(), chapter: -1, gold: 999, ..Default::default() }),
        ("oversized chapter", SaveData { node: "safe".into(), chapter: 256, gold: 999, ..Default::default() }),
        ("unknown node", SaveData { node: "missing".into(), chapter: 1, gold: 999, ..Default::default() }),
    ] {
        write_save_slot(0, &serde_json::to_vec(&data).unwrap());
        let mut g = Game {
            camp: Some(campaign::new_runner(&c)),
            title_phase: "loadslots".to_string(),
            gold: 7,
            handler_chapter: 9,
            ..Default::default()
        };
        if g.confirm_title_load_slot(0)
            || g.title_phase != "loadslots"
            || g.camp.as_ref().unwrap().node_id() != "safe"
            || g.gold != 7
            || g.handler_chapter != 9
        {
            panic!(
                "{name}: invalid JSON escaped selector or mutated state: title={:?} node={:?} gold={} chapter={}",
                g.title_phase,
                g.camp.as_ref().unwrap().node_id(),
                g.gold,
                g.handler_chapter
            );
        }
    }
}

/// F 組補充：Go 寫出的固定存檔原樣讀回、重新寫出相同（`TestSaveDataRoundTripsPersistentParty` 的
/// 位元組層版本；完整的 Go 黃金在 `golden_save_test.rs`）。
#[test]
fn save_data_indent_bytes_match_go_layout() {
    let d = SaveData {
        node: "town".to_string(),
        flags: Some(BTreeMap::new()),
        gold: 99,
        party_members: BTreeMap::from([(2, true), (10, false)]),
        party_join_order: vec![2, 10],
        chapter: 0,
        native_hud_gate_a: Some(1),
        ..Default::default()
    };
    let raw = go_json_marshal_indent(&d).unwrap();
    // Go `json.MarshalIndent(d, "", " ")` 的輸出（整數鍵以字串排序，"10" 在 "2" 前面；空 map 寫 {}；
    // nil 切片寫 null；chapter 是 0 時省略）。
    let want = "{\n \"node\": \"town\",\n \"flags\": {},\n \"gold\": 99,\n \"items\": null,\n \"party_members\": {\n  \"10\": false,\n  \"2\": true\n },\n \"party_join_order\": [\n  2,\n  10\n ],\n \"native_hud_gate_a\": 1\n}";
    assert_eq!(String::from_utf8(raw).unwrap(), want);
}
