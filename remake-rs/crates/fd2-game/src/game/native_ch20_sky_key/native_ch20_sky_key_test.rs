//! Go 來源：`remake/cmd/fd2/native_ch20_sky_key_test.go`（期望值不改）。
//!
//! Go 以 `t.Setenv` 設的環境變數由父測試以子行程傳入（`f_child_env`）。Go 另把 `FD2_ANI`、`FD2_FDOTHER` 與
//! `FD2_ORIGINAL_FDOTHER` 指到不存在的檔，證明演出只讀分離素材包；Rust 的載入器本來就只讀分離素材包，不設。
//! `FD2_SKY_KEY_EVIDENCE_OUT` 的證據圖輸出是 Go 的手動工具，不移植。

use super::super::main::main_b_test::new_beat_test_game;
use super::*;
use crate::test_support;

/// Go：`native_ch20_sky_key_test.go:21` `recoveredCh20SkyKeySpec`
fn recovered_ch20_sky_key_spec() -> campaign::NativeCh20SkyKeySequence {
    let script = campaign::HandlerScript {
        beats: vec![campaign::HandlerBeat {
            op: String::from("native_call"),
            native_target: String::from("0x24336"),
            native_semantic: String::from("ch20 天空之鑰固定合成演出序列"),
            native_confidence: String::from("已證實"),
            native_evidence: vec![String::from("docs/data/ida/fd2_ch20_sky_key_sequence_ida.txt")],
            source: campaign::HandlerSource { addr: String::from("0x242c9"), target: String::from("0x24336") },
            ..Default::default()
        }],
        ..Default::default()
    };
    let (beats, issues) = campaign::compile_handler_script(Some(&script), &campaign::HandlerBindings::default());
    assert!(
        issues.is_empty() && beats.len() == 1 && beats[0].native_ch20_sky_key.is_some(),
        "compile recovered 0x24336 spec beats={} issues={issues:?}",
        beats.len()
    );
    beats[0].native_ch20_sky_key.unwrap()
}

/// Go：`native_ch20_sky_key_test.go:35` `TestNativeCh20SkyKeyPlayerAssetsMatchRecoveredContract`
#[test]
fn test_native_ch20_sky_key_player_assets_match_recovered_contract() {
    let Some(bank) = test_support::input("FD2_ASSET_PACK", "animations/fdother_034_ch20_sky_key") else {
        return;
    };
    let spec = recovered_ch20_sky_key_spec();
    let frames = fdother::load_separated_ch20_sky_key_frames(bank).expect("LoadSeparatedCh20SkyKeyFrames");
    let clip = load_native_ch20_sky_key_ani(&spec).expect("loadNativeCh20SkyKeyANI");
    validate_native_ch20_sky_key_assets(&spec, &frames, Some(&clip)).expect("validateNativeCh20SkyKeyAssets");
}

/// Go：`native_ch20_sky_key_test.go:64` `TestNativeCh20SkyKeyAssetValidationRejectsIncompleteANI`
#[test]
fn test_native_ch20_sky_key_asset_validation_rejects_incomplete_ani() {
    let spec = recovered_ch20_sky_key_spec();
    let mut frame = fdother::Frame::default();
    (frame.width, frame.height, frame.pixels) = (1, 1, vec![1, 0, 1, 0, 0, 1]);
    let frames = vec![frame; spec.fdother_frame_count as usize];
    assert!(
        validate_native_ch20_sky_key_assets(
            &spec,
            &frames,
            Some(&afm::Clip {
                title: Vec::new(),
                header_frames: 0,
                frames: Vec::new(),
                indexed_frames: Vec::new(),
                palettes: Vec::new()
            })
        )
        .is_err(),
        "incomplete ANI snapshots were accepted"
    );
}

/// Go：`native_ch20_sky_key_test.go:75` `TestNativeCh20SkyKeyBeatRejectsEditedPayloadBeforeMutation`
#[test]
fn test_native_ch20_sky_key_beat_rejects_edited_payload_before_mutation() {
    let mut spec = recovered_ch20_sky_key_spec();
    spec.tail_frame_end = 99;
    let mut g = new_beat_test_game(vec![campaign::Beat {
        op: String::from("native_ch20_sky_key_sequence"),
        source: String::from("0x242c9"),
        native_ch20_sky_key: Some(spec),
        ..Default::default()
    }]);
    g.beat_advance();
    assert!(
        !g.load_err.is_empty() && g.native_ch20_sky_key.is_none() && g.beat_idx == 0,
        "edited payload err={:?} job={} beat={}",
        g.load_err,
        g.native_ch20_sky_key.is_some(),
        g.beat_idx
    );
}

/// Go：`native_ch20_sky_key_test.go:87` `TestNativeCh20SkyKeyFailureRestoresPublishedMapTransaction`
#[test]
fn test_native_ch20_sky_key_failure_restores_published_map_transaction() {
    let mut state = battle::State::default();
    (state.w, state.h) = (41, 40);
    state
        .materialize_native_map_view_state(battle::NativeMapViewState {
            camera_x: 2,
            camera_y: 3,
            cursor_x: 7,
            cursor_y: 9,
            visible_cursor_x: 5,
            visible_cursor_y: 6,
        })
        .unwrap();
    let mut g = Game {
        st: Some(state),
        cur_x: 7,
        cur_y: 9,
        cam_x: 48.0,
        cam_y: 72.0,
        native_map_work: (0..32).map(|i| (i + 1) as u8).collect(),
        native_map_vga: (0..64).map(|i| (i + 2) as u8).collect(),
        native_map_dac: (0..256 * 3).map(|i| (i % 64) as u8).collect(),
        native_fdother_palette_phase: 7,
        ..Default::default()
    };
    let want_state = g.st.clone().unwrap();
    let (want_work, want_vga, want_dac) =
        (g.native_map_work.clone(), g.native_map_vga.clone(), g.native_map_dac.clone());
    let want_clock = g.native_map_clock;
    let want = (g.cur_x, g.cur_y, g.cam_x, g.cam_y);

    let rollback = snapshot_native_ch20_sky_key_state(&g);
    {
        let view = &mut g.st.as_mut().unwrap().native_map_view_state;
        (view.camera_x, view.cursor_x) = (14, 19);
    }
    (g.cur_x, g.cur_y, g.cam_x, g.cam_y) = (19, 18, 336.0, 192.0);
    (g.native_map_work[0], g.native_map_vga[0], g.native_map_dac[0]) = (0xff, 0xfe, 0x3f);
    g.native_fdother_palette_phase = 12;
    g.native_map_clock.elapsed_ticks = 123;
    g.native_ch20_sky_key = Some(NativeCh20SkyKeyJob { rollback: Some(rollback), ..Default::default() });
    g.fail_native_ch20_sky_key("synthetic renderer failure");

    assert!(
        g.native_ch20_sky_key.is_none()
            && !g.load_err.is_empty()
            && format!("{:?}", g.st.as_ref()) == format!("{:?}", Some(&want_state)),
        "failed 0x24336 did not restore state: err={:?}",
        g.load_err
    );
    assert!(
        (g.cur_x, g.cur_y, g.cam_x, g.cam_y) == want
            && g.native_map_work == want_work
            && g.native_map_vga == want_vga
            && g.native_map_dac == want_dac
            && g.native_map_clock == want_clock
            && g.native_fdother_palette_phase == 7,
        "failed 0x24336 leaked presentation state: cursor=({},{}) camera=({},{}) phase={}",
        g.cur_x,
        g.cur_y,
        g.cam_x,
        g.cam_y,
        g.native_fdother_palette_phase
    );
}

/// Go：`native_ch20_sky_key_test.go:136` `rosterHasItem`
fn roster_has_item(roster: &BTreeMap<i64, battle::Unit>, item_id: i64) -> bool {
    roster.values().any(|unit| unit.inventory.contains(&item_id))
}

/// 兩個第 21 戰測試共用的子行程環境。
fn chapter21_child(name: &str) -> bool {
    let need = ["animations/ANI_000/animation.json", "animations/fdother_034_ch20_sky_key/bank.json"];
    if need.iter().any(|rel| test_support::input("FD2_ASSET_PACK", rel).is_none()) {
        return false;
    }
    f_child_env(module_path!(), name, |dir| {
        vec![("FD2_MUTE", "1".into()), ("XDG_DATA_HOME", dir.as_os_str().to_os_string())]
    })
    .is_some()
}

/// 兩個第 21 戰測試共用的開頭：以 `ch21.json` 的實際出戰順序建持續隊伍投影、載 map20、建 16 人戰場，
/// 清掉前 16 格的六素材（0xD1..0xD6），再把原生視圖歸零並組出第一張地圖幀。
///
/// Go 的兩個測試各自寫同一段；這只提供既有 JOIN／部署的資料形狀，不證明上一個整備節點或一般玩家路徑已執行。
fn chapter21_battle() -> Game {
    let scenario = battle::load_scenario(&asset_path("assets/scenarios/ch21.json")).expect("LoadScenario ch21");
    let order: Vec<i64> = scenario.party.iter().map(|unit| unit.fig).collect();
    assert!(order.len() >= 16, "第21戰永久名冊={}，無法建立原版16名出戰前沿", order.len());
    let mut g = Game {
        party_members: order.iter().map(|&id| (id, true)).collect(),
        party_join_order: order.clone(),
        party_deploy: order[1..16].iter().map(|&id| (id, true)).collect(),
        ..Default::default()
    };
    crate::game::save::save_test::attach_official_locale(&mut g);
    // 戰前未出戰的永久成員也必須留在名冊；先用同一份可編輯 scenario 的 constructor 建立持續紀錄，
    // 再由 party_deploy 投影出本戰 16 人。
    g.seed_persistent_party_from_load_ch(&order, &scenario.party_units(&[])).expect("seedPersistentPartyFromLoadCH");
    g.load_map("assets/maps/map20").expect("loadMap map20");
    g.reset_battle("assets/maps/map20/map20_units.json", "assets/scenarios/ch21.json");
    assert!(
        g.load_err.is_empty() && g.st.as_ref().is_some_and(|st| st.units.len() == 75),
        "第21戰初始化 err={:?} slots={}",
        g.load_err,
        g.handler_unit_count()
    );
    for slot in 0..16 {
        let unit = &mut g.st.as_mut().unwrap().units[slot];
        for index in (0..unit.inventory.len()).rev() {
            if (0xd1..=0xd6).contains(&unit.inventory[index]) {
                unit.remove_inventory_index(index as i64);
            }
        }
    }
    (g.cur_x, g.cur_y) = (0, 0);
    let st = g.st.as_mut().unwrap();
    let view_ok = st.materialize_native_map_view_state(battle::NativeMapViewState::default());
    assert!(
        view_ok.is_ok() && st.materialize_native_map_hud_state(1, 1, 1) && st.materialize_native_map_range_mode(1),
        "第21戰原生視圖初始化失敗: {view_ok:?}"
    );
    g
}

fn enter_post_battle(g: &mut Game) {
    let mut full = campaign::load(&asset_path("assets/scenarios/campaign_full.json")).expect("campaign.Load");
    full.start = String::from("battle_ch21");
    g.camp = Some(campaign::new_runner(&full));
    for id in [24, 23] {
        assert!(
            !g.party_roster.contains_key(&id) && !g.party_members.get(&id).copied().unwrap_or(false),
            "戰後新成員{id}不應預先存在持續名冊"
        );
    }
    g.result = String::from("win");
}

fn node_id(g: &Game) -> String {
    g.camp.as_ref().map(|camp| camp.node_id().to_string()).unwrap_or_default()
}

/// 戰後對話的正式輸入：停在等鍵就按 Enter，再跑一次 Update。
fn answer_post_battle_dialogue(g: &mut Game, clock: &mut VirtualFrameClock) {
    if g.native_story_dialogue_at_input_wait() {
        let mut node = g.camp.as_ref().and_then(|camp| camp.node().cloned());
        assert!(
            g.handle_native_story_input(node.as_mut(), NativeStoryInput { enter: true, escape: false }),
            "第21戰戰後正式故事輸入遭拒"
        );
    }
    g.advance_virtual_frame(clock);
    if let Err(err) = g.update() {
        panic!("Update：{}", err.message());
    }
}

fn assert_join_records(g: &Game) {
    for id in [24, 23] {
        let unit = g.party_roster.get(&id);
        assert!(
            unit.is_some_and(|u| u.has_native_identity
                && u.native_identity == id
                && u.has_map_selector_key
                && u.inventory_slots.len() == 8
                && u.native_inventory_flags.len() == 8),
            "JOIN{id}未建立可供正常商店與存檔消費的持續記錄"
        );
    }
}

fn assert_join_chronology(g: &Game) {
    let order = &g.party_join_order;
    let member = |id: i64| g.party_members.get(&id).copied().unwrap_or(false);
    assert!(
        member(24) && member(23) && order.len() >= 2 && order[order.len() - 2] == 24 && order[order.len() - 1] == 23,
        "天空之鑰戰後 JOIN chronology={order:?}"
    );
}

/// Go：`native_ch20_sky_key_test.go:148` `TestChapterTwentyOneSkyKeyBattleResultReachesTownAndSaveBoundary`
#[test]
fn test_chapter_twenty_one_sky_key_battle_result_reaches_town_and_save_boundary() {
    if !chapter21_child("test_chapter_twenty_one_sky_key_battle_result_reaches_town_and_save_boundary") {
        return;
    }
    let mut g = chapter21_battle();
    assert!(g.sc.as_ref().is_some_and(|sc| sc.runtime_append_groups), "第21戰 runtime 群組契約 runtime=false");
    {
        let pending = &g.st.as_ref().unwrap().pending_groups;
        assert!(
            pending.len() == 4 && (1..=4).all(|group| pending.get(&group).copied().unwrap_or(false)),
            "第21戰 runtime 群組契約 pending={pending:?}"
        );
    }
    for (slot, &id) in g.party_join_order.clone()[..16].iter().enumerate() {
        let unit = &g.st.as_ref().unwrap().units[slot];
        assert!(
            unit.has_native_identity && unit.native_identity == id,
            "第21戰出戰 slot{slot} identity={}，want {id}",
            unit.native_identity
        );
    }
    // 只改變勝利當下的物品欄，代表玩家在前面關卡已蒐集六件材料（原版「恰好六組配對」）。
    for offset in 0..6 {
        let unit = &mut g.st.as_mut().unwrap().units[offset];
        assert!(unit.add_inventory_item(0xd1 + offset as i64, false), "第21戰 slot{offset} 無法放入材料");
    }
    assert!(g.native_map_hud_input().1, "第21戰原生索引地圖輸入不完整");
    // Go 的原生地圖時鐘讀 time.Now；Rust 讀虛擬時鐘，給一個非零時刻。
    let mut clock = VirtualFrameClock::default();
    g.advance_virtual_frame(&mut clock);
    g.compose_native_map_frame().expect("composeNativeMapFrame");
    enter_post_battle(&mut g);
    g.native_fdother_palette_phase = 5; // 驗證相對 68 次循環，不宣稱原版程序初值固定。
    assert!(
        g.confirm_battle_result() && g.result.is_empty() && node_id(&g) == "story_ch21_post_sky_key_intro",
        "第21戰勝利邊界 node={:?} result={:?} err={:?}",
        node_id(&g),
        g.result,
        g.load_err
    );
    {
        let units = &g.st.as_ref().unwrap().units;
        let (a, b) = (&units[0], &units[25]);
        assert!(
            a.x == 21
                && a.y == 14
                && a.dir == 2
                && a.has_native_map_presentation
                && a.native_map_presentation.pose == 2,
            "第21戰戰後 layout slot0=({},{},pose{})，want (21,14,2)",
            a.x,
            a.y,
            a.dir
        );
        assert!(
            b.x == 23
                && b.y == 14
                && b.dir == 1
                && b.has_native_map_presentation
                && b.native_map_presentation.pose == 1
                && g.cam_x == 336.0
                && g.cam_y == 240.0,
            "第21戰戰後 layout slot25=({},{},pose{}) camera=({},{})",
            b.x,
            b.y,
            b.dir,
            g.cam_x,
            g.cam_y
        );
    }

    let mut seen = std::collections::BTreeSet::new();
    let (mut seen_act63, mut seen_act64) = (false, false);
    let (mut sky_palette_start, mut sky_first_frame_cycles) = (-1i64, 0i64);
    let mut seen_native_dialogue = std::collections::BTreeSet::new();
    let mut screen = Picture::new_image(LOGICAL_W, LOGICAL_H);
    for _ in 0..30000 {
        if node_id(&g) == "town_ch22" {
            break;
        }
        if let Some(job) = g.act_job.as_ref() {
            seen_act63 = seen_act63 || job.acting.len() == 4;
            seen_act64 = seen_act64 || job.acting.len() == 5;
        }
        if let Some(job) = g.native_ch20_sky_key.as_ref() {
            // 0x4DFCC 只數 FirstFrames 推進的次數；pan 期間的地圖重繪也會推進它，但次數隨時序而變。
            if job.phase == NATIVE_CH20_SKY_KEY_FIRST_FRAMES {
                if sky_palette_start < 0 {
                    // beginFirstFrames 自己那一次發生在第一次觀測之前。
                    (sky_palette_start, sky_first_frame_cycles) = (g.native_fdother_palette_phase, 1);
                } else {
                    sky_first_frame_cycles += (g.native_fdother_palette_phase - sky_palette_start) & 15;
                    sky_palette_start = g.native_fdother_palette_phase;
                }
            }
            let (phase, frame) = (job.phase, job.frame);
            seen.insert(phase.0);
            assert!(g.draw_native_ch20_sky_key(Some(&mut screen)), "天空之鑰 phase={} frame={frame} 無法呈現", phase.0);
            // 測試仍經過正式呈現確認與 stepper，只壓縮 DOS／主機等待時間。
            g.native_ch20_sky_key.as_mut().unwrap().ticks = 0;
            g.step_native_ch20_sky_key();
        }
        if let Some(current) = g.dialog.last() {
            let native = current.native_dialogue.as_ref();
            assert!(
                native.is_some_and(|n| n.source_dat == "FDTXT_021")
                    && current.upper.is_some()
                    && g.native_dialogue_opening.len() == 5
                    && g.native_dialogue_closing.len() >= 5,
                "第21戰戰後對話遺失原生生命週期"
            );
            let native = native.unwrap();
            seen_native_dialogue.insert((native.string_index, native.utterance));
            answer_post_battle_dialogue(&mut g, &mut clock);
            continue;
        }
        g.tick(1);
        assert!(
            g.load_err.is_empty(),
            "第21戰戰後流程在 node={:?} beat={}/{} 停止: {}",
            node_id(&g),
            g.beat_idx,
            g.beats.len(),
            g.load_err
        );
    }
    assert!(
        node_id(&g) == "town_ch22" && g.st.is_none() && g.handler_chapter == 21,
        "天空之鑰流程邊界 node={:?} battle={} chapter={}",
        node_id(&g),
        g.st.is_some(),
        g.handler_chapter
    );
    assert!(seen_act63 && seen_act64, "天空之鑰相鄰 ACTING 未完整消費: act63={seen_act63} act64={seen_act64}");
    assert_eq!(seen_native_dialogue.len(), 26, "第21戰天空之鑰成功臂原生對話數");
    for phase in NATIVE_CH20_SKY_KEY_PAN.0..=NATIVE_CH20_SKY_KEY_TAIL_FRAMES.0 {
        assert!(seen.contains(&phase), "天空之鑰演出未經過 phase={phase}");
    }
    // `beginFirstFrames` 先推一次，之後 frame 1→68 再推 67 次，合計 68。逐相位對齊仍是另一項 dynamic-E2。
    assert!(
        sky_palette_start >= 0 && sky_first_frame_cycles == 68,
        "0x4DFCC FirstFrames 推進 {sky_first_frame_cycles} 次，want 68（取樣到 start={sky_palette_start}）"
    );
    assert_join_chronology(&g);
    assert!(roster_has_item(&g.party_roster, 0x64), "天空之鑰沒有經 sync_party 進入持續隊伍");
    assert_join_records(&g);

    // 只在 town_ch22 這個節點邊界存檔，證實動畫、JOIN 與配方獎勵沒有被直接接往下一場戰鬥而遺失。
    g.save_game_to_slot(0).expect("saveGameToSlot");
    assert!(std::path::Path::new(&save_slot_path(0)).exists(), "town_ch22 存檔未寫入");
    let (want_order, want_members, want_roster) =
        (g.party_join_order.clone(), g.party_members.clone(), g.party_roster.clone());
    (g.party_members, g.party_join_order, g.party_roster) = Default::default();
    g.handler_chapter = 0;
    g.load_game_from_slot(0);
    assert!(
        g.load_err.is_empty()
            && node_id(&g) == "town_ch22"
            && g.st.is_none()
            && g.handler_chapter == 21
            && g.party_members == want_members
            && g.party_join_order == want_order
            && g.party_roster == want_roster
            && roster_has_item(&g.party_roster, 0x64),
        "town_ch22 存讀檔 node={:?} chapter={} order={:?} err={:?}",
        node_id(&g),
        g.handler_chapter,
        g.party_join_order,
        g.load_err
    );
}

/// Go：`native_ch20_sky_key_test.go:436` `TestChapterTwentyOneSkyKeyInsufficientBranchUsesNativeDialogueAndTownSave`
#[test]
fn test_chapter_twenty_one_sky_key_insufficient_branch_uses_native_dialogue_and_town_save() {
    if !chapter21_child("test_chapter_twenty_one_sky_key_insufficient_branch_uses_native_dialogue_and_town_save") {
        return;
    }
    // 清除所有六素材（`chapter21_battle` 已做），證明這條路徑由正式配方判定進入不足臂。
    let mut g = chapter21_battle();
    let mut clock = VirtualFrameClock::default();
    g.advance_virtual_frame(&mut clock);
    g.compose_native_map_frame().expect("composeNativeMapFrame");
    enter_post_battle(&mut g);
    assert!(
        g.confirm_battle_result() && node_id(&g) == "story_ch21_post_sky_key_intro",
        "第21戰材料不足勝利邊界 node={:?} err={:?}",
        node_id(&g),
        g.load_err
    );
    let mut seen = std::collections::BTreeSet::new();
    let (mut seen_act, mut seen_sky_key) = (false, false);
    for _ in 0..20000 {
        if node_id(&g) == "town_ch22" {
            break;
        }
        seen_act = seen_act || g.act_job.is_some();
        seen_sky_key = seen_sky_key || g.native_ch20_sky_key.is_some();
        if let Some(current) = g.dialog.last() {
            let native = current.native_dialogue.as_ref();
            assert!(
                native.is_some_and(|n| n.source_dat == "FDTXT_021" && (n.string_index == 5 || n.string_index == 6))
                    && current.upper.is_some()
                    && g.native_dialogue_opening.len() == 5
                    && g.native_dialogue_closing.len() >= 5,
                "第21戰材料不足對話生命週期漂移"
            );
            let native = native.unwrap();
            seen.insert((native.string_index, native.utterance));
            answer_post_battle_dialogue(&mut g, &mut clock);
            continue;
        }
        g.tick(1);
        assert!(
            g.load_err.is_empty(),
            "第21戰材料不足流程 node={:?} beat={}/{}: {}",
            node_id(&g),
            g.beat_idx,
            g.beats.len(),
            g.load_err
        );
    }
    assert!(
        node_id(&g) == "town_ch22" && g.st.is_none() && g.handler_chapter == 21,
        "第21戰材料不足邊界 node={:?} battle={} chapter={}",
        node_id(&g),
        g.st.is_some(),
        g.handler_chapter
    );
    assert!(
        seen.len() == 14 && !seen_act && !seen_sky_key,
        "第21戰材料不足對話={} act={seen_act} sky-key={seen_sky_key}，want 14/false/false",
        seen.len()
    );
    assert!(!roster_has_item(&g.party_roster, 0x64), "材料不足分支錯誤授予天空之鑰");
    assert_join_chronology(&g);
    assert_join_records(&g);

    g.save_game_to_slot(0).expect("saveGameToSlot");
    let want_order = g.party_join_order.clone();
    (g.party_members, g.party_join_order, g.party_roster) = Default::default();
    g.handler_chapter = 0;
    g.load_game_from_slot(0);
    assert!(
        g.load_err.is_empty()
            && node_id(&g) == "town_ch22"
            && g.handler_chapter == 21
            && g.party_join_order == want_order
            && !roster_has_item(&g.party_roster, 0x64),
        "材料不足town_ch22存讀檔 node={:?} chapter={} order={:?} err={:?}",
        node_id(&g),
        g.handler_chapter,
        g.party_join_order,
        g.load_err
    );
}
