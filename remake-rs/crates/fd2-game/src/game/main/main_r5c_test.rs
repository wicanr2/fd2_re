//! R5 階段 2 r5c 組（城鎮與設施）的測試。
//!
//! Go 測試移植：`native_confirmation_mouth_test.go:15` `TestNativePreparationConfirmationMouthWaitShowsBothFrames`
//! （r5d 留給本組；經 `composeNativePreparationPromptFrame`）。同檔另兩項經 `beginNativeSystemEndTurn`（crate 裡還沒有），
//! 其餘呼叫本組函式的 Go 測試（`native_shop_ui_test.go`、`native_shop_recipient_path_test.go`、
//! `native_church_class_path_test.go`、`native_church_revive_path_test.go`、`native_preparation_ui_test.go:124`）經商店購買
//! 清單、教會轉職清單、復活與整備名冊的合成（`r8`），`hotel_trace_test.go:209` 是以環境變數啟用、輸出畫面的同源探針，
//! 都不移植。其餘是「Rust 新增」：期望值照 Go 程式碼與 `presentNativeClassFrame` 的 2 倍最近鄰貼上；整段的逐擷取點
//! 比對見 `PORTING.md`「R5 階段 2：r5c」。

use super::*;
use crate::test_support;

fn pixel(p: &Picture, x: i64, y: i64) -> [u8; 4] {
    let at = ((y * p.width + x) * 4) as usize;
    [p.pix[at], p.pix[at + 1], p.pix[at + 2], p.pix[at + 3]]
}

/// 索引 i 的顏色是 (i, 255-i, i/2, 255)。
fn palette() -> ColorPalette {
    (0..256).map(|i| ColorRgba { r: i as u8, g: 255 - i as u8, b: (i / 2) as u8, a: 0xff }).collect()
}

fn indexed(seed: u8) -> Vec<u8> {
    (0..320 * 200).map(|i| (i as u8).wrapping_mul(7).wrapping_add(seed)).collect()
}

fn black_screen() -> Picture {
    let mut s = Picture::new_image(640, 400);
    s.fill(ColorRgba { r: 0, g: 0, b: 0, a: 0xff });
    s
}

/// `presentNativeClassFrame`：320×200 索引畫格以 2 倍最近鄰蓋滿 640×400。
fn assert_presented(screen: &Picture, frame: &[u8], what: &str) {
    let pal = palette();
    for (x, y) in [(0, 0), (1, 1), (2, 0), (319, 199), (320, 200), (639, 399), (123, 77)] {
        let c = pal[frame[((y / 2) * 320 + x / 2) as usize] as usize];
        assert_eq!(pixel(screen, x, y), [c.r, c.g, c.b, c.a], "{what} ({x},{y})");
    }
}

fn class_ui_game() -> Game {
    Game {
        native_class_ui: Some(NativeClassUIAssets { palette: palette(), ..Default::default() }),
        ..Default::default()
    }
}

/// Rust 新增：`drawNativeClassUIJob` 依 frame 畫開關框幀、畫完後畫還原畫面，設 `drawn`；沒有工作、frame 為負或
/// 兩者都沒有時回 false 且不畫。
#[test]
fn draw_native_class_ui_job_presents_frames_then_restore() {
    let mut g = class_ui_game();
    let mut screen = black_screen();
    assert!(!g.draw_native_class_ui_job(&mut screen), "沒有工作");
    let (a, b, restore) = (indexed(1), indexed(2), indexed(3));
    g.native_class_ui_job =
        Some(NativeClassUIJob { frames: vec![a.clone(), b.clone()], restore: restore.clone(), ..Default::default() });
    g.native_map_presented = true;
    g.native_interface_hud_visible = true;
    assert!(g.draw_native_class_ui_job(&mut screen));
    assert_presented(&screen, &a, "第 0 幀");
    assert!(g.native_class_ui_job.as_ref().unwrap().drawn);
    assert!(!g.native_map_presented && !g.native_interface_hud_visible, "presentNativeClassFrame 清本幀旗標");
    g.native_class_ui_job.as_mut().unwrap().frame = 1;
    assert!(g.draw_native_class_ui_job(&mut screen));
    assert_presented(&screen, &b, "第 1 幀");
    g.native_class_ui_job.as_mut().unwrap().frame = 2;
    assert!(g.draw_native_class_ui_job(&mut screen));
    assert_presented(&screen, &restore, "還原畫面");
    g.native_class_ui_job.as_mut().unwrap().restore.clear();
    g.native_class_ui_job.as_mut().unwrap().drawn = false;
    let mut untouched = black_screen();
    assert!(!g.draw_native_class_ui_job(&mut untouched), "畫完且沒有還原畫面");
    assert!(!g.native_class_ui_job.as_ref().unwrap().drawn);
    assert_eq!(pixel(&untouched, 10, 10), [0, 0, 0, 0xff]);
    g.native_class_ui_job.as_mut().unwrap().frame = -1;
    assert!(!g.draw_native_class_ui_job(&mut untouched), "frame 為負");
}

/// Rust 新增：教會與商店的開關框工作與 `drawNativeClassUIJob` 同一套選幀；`drawNativeShop` 在模式為空時不畫，
/// 有工作時交給 `drawNativeShopUIJob`。
#[test]
fn draw_native_church_and_shop_ui_jobs_present_frames() {
    let mut g = class_ui_game();
    let mut screen = black_screen();
    assert!(!g.draw_native_church_ui_job(&mut screen));
    let (a, restore) = (indexed(9), indexed(4));
    g.native_church_ui_job = Some(NativeChurchUIJob {
        frames: vec![a.clone()],
        restore: restore.clone(),
        frame: 0,
        drawn: false,
        service_menu: None,
        after: None,
    });
    assert!(g.draw_native_church_ui_job(&mut screen));
    assert_presented(&screen, &a, "教會第 0 幀");
    assert!(g.native_church_ui_job.as_ref().unwrap().drawn);
    g.native_church_ui_job.as_mut().unwrap().frame = 1;
    assert!(g.draw_native_church_ui_job(&mut screen));
    assert_presented(&screen, &restore, "教會還原畫面");

    let shop = indexed(21);
    g.native_shop_ui_job = Some(NativeClassUIJob { frames: vec![shop.clone()], ..Default::default() });
    assert!(!g.draw_native_shop(&mut screen), "native_shop_mode 為空時不畫");
    g.native_shop_mode = String::from("menu");
    let mut screen = black_screen();
    assert!(g.draw_native_shop(&mut screen));
    assert_presented(&screen, &shop, "商店開框");
    assert!(g.native_shop_ui_job.as_ref().unwrap().drawn);
    g.native_shop_ui_job.as_mut().unwrap().frame = 1;
    assert!(!g.draw_native_shop_ui_job(&mut screen), "畫完且沒有還原畫面");
}

/// Rust 新增：`drawCampaignUI` 的守門與分派。沒有 campaign 或字型時不畫；戰場等沒有分支的節點不畫；教會節點的
/// 開關框工作優先，接著 `drawNativeClassUIJob`；酒店模式為空、商店模式為空時走 TTF 後備（`todo!("r8:…")`）。
#[test]
fn draw_campaign_ui_dispatches_native_facility_presentations() {
    let nodes = BTreeMap::from([
        ("battle".to_string(), campaign::Node { r#type: "battle".into(), ..Default::default() }),
        ("church".to_string(), campaign::Node { r#type: "church".into(), ..Default::default() }),
    ]);
    let c = campaign::Campaign { start: "battle".into(), nodes, ..Default::default() };
    let mut g = class_ui_game();
    let job = indexed(5);
    g.native_class_ui_job = Some(NativeClassUIJob { frames: vec![job.clone()], ..Default::default() });
    let mut screen = black_screen();
    g.draw_campaign_ui(&mut screen);
    assert_eq!(pixel(&screen, 5, 5), [0, 0, 0, 0xff], "沒有 campaign");
    g.camp = Some(campaign::new_runner(&c));
    g.draw_campaign_ui(&mut screen);
    assert_eq!(pixel(&screen, 5, 5), [0, 0, 0, 0xff], "沒有字型");
    g.font = Some(Font::default());
    g.draw_campaign_ui(&mut screen);
    assert_eq!(pixel(&screen, 5, 5), [0, 0, 0, 0xff], "戰場節點沒有分支");
    assert!(!g.native_class_ui_job.as_ref().unwrap().drawn);
    g.camp.as_mut().unwrap().cur = "church".into();
    g.draw_campaign_ui(&mut screen);
    assert_presented(&screen, &job, "教會節點的 drawNativeClassUIJob");
    let church = indexed(6);
    g.native_church_ui_job = Some(NativeChurchUIJob {
        frames: vec![church.clone()],
        restore: Vec::new(),
        frame: 0,
        drawn: false,
        service_menu: None,
        after: None,
    });
    g.draw_campaign_ui(&mut screen);
    assert_presented(&screen, &church, "教會開關框工作優先");
}

/// `FD2_ASSET_PACK` 存在才回傳 `true`；要求原版輸入時缺檔即失敗。
fn asset_pack_ready(rel: &str) -> bool {
    test_support::input("FD2_ASSET_PACK", rel).is_some()
}

/// Rust 新增：酒店的穩態、存檔列表與「記錄儲存完畢」三種畫格，與直接呼叫 campaign 合成器的結果逐位元組相同；
/// `drawNativeHotel` 把 `menu` 畫格 2 倍貼上，模式為空時不畫。存檔列表讀 `FD2_NATIVE_SAVE`，在子行程裡設成空字串並
/// 把使用者資料目錄指到暫存目錄（同 Go 酒店測試的 `t.Setenv`）。
#[test]
fn compose_native_hotel_frame_matches_campaign_compositors() {
    if !asset_pack_ready("ui/fdother_010_town_label/resource.json") {
        return;
    }
    let Some(_dir) = f_child_env(module_path!(), "compose_native_hotel_frame_matches_campaign_compositors", |dir| {
        vec![("FD2_NATIVE_SAVE", "".into()), ("XDG_DATA_HOME", dir.as_os_str().to_owned())]
    }) else {
        return;
    };
    let shared = load_native_class_ui_assets().expect("loadNativeClassUIAssets").unwrap();
    let hotel = load_native_hotel_ui_assets(Some(&shared)).expect("loadNativeHotelUIAssets").unwrap();
    let mut g = Game { native_class_ui: Some(shared), native_hotel_ui: Some(hotel), ..Default::default() };
    let (shared, hotel) = (g.native_class_ui.clone().unwrap(), g.native_hotel_ui.clone().unwrap());
    let assets = hotel.assets.as_ref().unwrap();
    let (strings, font) = (shared.strings.as_ref().unwrap(), shared.font.as_ref().unwrap());
    let overlay = |portrait: usize| {
        campaign::compose_native_church_dialogue_overlay_at(
            &assets.background,
            &shared.dialogue,
            &hotel.portraits[portrait],
            campaign::native_facility_portrait_offset(NATIVE_HOTEL_PORTRAIT_ID),
        )
        .unwrap()
    };
    let mut stable = overlay(0);
    let stable = campaign::compose_native_church_text_at(
        &mut stable,
        strings,
        font,
        NATIVE_HOTEL_PROMPT_TEXT,
        campaign::NATIVE_SHOP_TEXT_OFFSET,
    )
    .unwrap();
    assert_eq!(g.compose_native_hotel_stable(0), (stable.clone(), true));
    assert!(!g.compose_native_hotel_stable(-1).1 && !g.compose_native_hotel_stable(hotel.portraits.len() as i64).1);
    assert!(!g.compose_native_hotel_frame(0).1, "模式為空");

    (g.native_hotel_mode, g.hotel_sel, g.native_shop_ui_pulse) = ("menu".into(), 1, 3);
    let menu = campaign::compose_native_shop_service_steady_frame(&stable, Some(assets), 1, 3).unwrap();
    assert_eq!(g.compose_native_hotel_frame(0), (menu.clone(), true));
    let mut screen = black_screen();
    assert!(g.draw_native_hotel(&mut screen));
    let pal = &shared.palette;
    for (x, y) in [(0, 0), (101, 251), (639, 399)] {
        let c = pal[menu[((y / 2) * 320 + x / 2) as usize] as usize];
        assert_eq!(pixel(&screen, x, y), [c.r, c.g, c.b, c.a], "menu ({x},{y})");
    }

    (g.native_hotel_mode, g.native_hotel_slot_sel) = ("slots".into(), 2);
    let (slots, ok) = native_load_slot_metadata();
    assert!(ok);
    let list =
        campaign::compose_native_load_slots_frame(&stable, &hotel.slots_box, Some(strings), Some(font), slots, 2)
            .unwrap();
    assert_eq!(g.compose_native_hotel_frame(0), (list, true));

    for (mode, text) in [("saved", NATIVE_HOTEL_SAVED_TEXT), ("loaded", 0x1de), ("load_blocked", 0x1df)] {
        g.native_hotel_mode = mode.into();
        let mut frame = overlay(0);
        let want =
            campaign::compose_native_church_text_at(&mut frame, strings, font, text, campaign::NATIVE_SHOP_TEXT_OFFSET)
                .unwrap();
        assert_eq!(g.compose_native_hotel_frame(0), (want, true), "{mode}");
    }
    g.native_hotel_mode.clear();
    assert!(!g.draw_native_hotel(&mut black_screen()), "模式為空時不畫");
}

/// Go 測試 helper `attachOfficialLocale`（`locale_fixture_test.go:11`；`save_test.rs` 有同一份，模組私有）。
fn attach_official_locale(g: &mut Game) {
    let locale_id = "zh-Hant";
    g.locale_id = locale_id.to_string();
    g.locale_catalog = load_official_locale(locale_id).expect("官方語系");
    g.locale_content = load_official_locale_content(locale_id).expect("官方全量內容");
    g.locale_entities = load_official_locale_entities(locale_id).expect("官方實體名稱");
}

fn unit_with_fig(fig: i64) -> battle::Unit {
    let mut u = battle::Unit::default();
    u.fig = fig;
    u
}

/// Go `native_preparation_ui_test.go:15` `newPreparationRecordTestGame`＋`finishPreparationRecordTransition`。
fn new_preparation_record_test_game() -> Game {
    let ui = load_native_preparation_ui_assets().expect("loadNativePreparationUIAssets").unwrap();
    let (strings, font) = (ui.status.strings.clone(), ui.status.font.clone());
    let nodes = BTreeMap::from([(
        "prep".to_string(),
        campaign::Node { r#type: "preparation".into(), party_limit: 15, ..Default::default() },
    )]);
    let c = campaign::Campaign { start: "prep".into(), nodes, ..Default::default() };
    let mut g = Game {
        camp: Some(campaign::new_runner(&c)),
        native_preparation_ui: Some(ui),
        native_class_ui: Some(NativeClassUIAssets { strings, font, ..Default::default() }),
        party_join_order: vec![0, 1],
        party_roster: BTreeMap::from([(0, unit_with_fig(0)), (1, unit_with_fig(1))]),
        party_members: BTreeMap::from([(0, true), (1, true)]),
        ..Default::default()
    };
    attach_official_locale(&mut g);
    g.handler_chapter = 22;
    let node = g.camp.as_ref().unwrap().node().cloned();
    g.setup_preparation(node.as_ref());
    for _ in 0..32 {
        let Some(job) = g.native_class_ui_job.as_mut() else {
            break;
        };
        job.drawn = true;
        g.step_native_class_ui_lifecycle(GameInstant(0));
    }
    assert!(g.native_class_ui_job.is_none(), "整備提示動畫沒有完成");
    g
}

/// Go：`native_confirmation_mouth_test.go:15` `TestNativePreparationConfirmationMouthWaitShowsBothFrames`
#[test]
fn test_native_preparation_confirmation_mouth_wait_shows_both_frames() {
    if !asset_pack_ready("ui/fdother_010_town_label/resource.json") {
        return;
    }
    let Some(_dir) =
        f_child_env(module_path!(), "test_native_preparation_confirmation_mouth_wait_shows_both_frames", |dir| {
            vec![("FD2_NATIVE_SAVE", "".into()), ("XDG_DATA_HOME", dir.as_os_str().to_owned())]
        })
    else {
        return;
    };
    let mut g = new_preparation_record_test_game();
    g.native_confirmation_mouth_rng = Some(battle::GoMathRand::new(177));
    g.native_confirmation_mouth_owner = None;
    let portraits = load_native_separated_portrait(0x4b).expect("confirmation DATO0/3 unavailable");
    assert!(portraits.len() >= 4, "confirmation DATO0/3 unavailable");
    let mut seen = BTreeSet::new();
    let start = GameInstant(100 * GoDuration::SECOND.0);
    for tick in 0..80i64 {
        g.step_native_class_ui_lifecycle(start + GoDuration(tick * 2 * NATIVE_BIOS_TICK_PERIOD.0));
        let (got, ok) = g.compose_native_preparation_prompt_frame();
        assert!(ok, "waiting frame unavailable");
        let ui = g.native_preparation_ui.as_ref().unwrap();
        let mut matched = false;
        for index in [0usize, 3] {
            let question = campaign::compose_native_preparation_record_question(
                &g.prep_prompt_source,
                &ui.dialogue,
                &portraits[index],
                ui.status.strings.as_ref(),
                ui.status.font.as_ref(),
            )
            .expect("record question");
            let want = campaign::compose_native_confirmation_choices(
                &question,
                &ui.choices,
                g.prep_confirm_sel,
                g.native_class_ui_pulse / 2,
            )
            .expect("confirmation choices");
            if got == want {
                seen.insert(index);
                matched = true;
            }
        }
        assert!(matched, "waiting frame is neither complete DATO0 nor DATO3 {tick}");
    }
    assert!(
        seen.contains(&0) && seen.contains(&3),
        "formal confirmation wait does not show both mouth frames {seen:?}"
    );
}

/// 跑完目前的原版開合工作（每格都先標成已畫）。
fn finish_native_class_ui_job(g: &mut Game) {
    for _ in 0..64 {
        let Some(job) = g.native_class_ui_job.as_mut() else {
            return;
        };
        job.drawn = true;
        g.step_native_class_ui_lifecycle(GameInstant(0));
    }
    panic!("原版開合工作沒有結束");
}

/// Go：`native_confirmation_mouth_test.go:59` `TestNativeConfirmationMouthENDWaitAndCallerPortrait` 的流程部分：
/// 系統游標選單 Down→END 收合四格後建立確認框；取消時收合、回覆、停留延遲、收框都走完，不結束回合。
/// Go 測試後半的嘴型共用（另一個呼叫端頭像）由 `test_native_preparation_confirmation_mouth_wait_shows_both_frames`
/// 涵蓋同一個合成器。
#[test]
fn test_native_system_end_turn_confirmation_cancel_keeps_turn() {
    if !asset_pack_ready("ui/fdother_010_town_label/resource.json") {
        return;
    }
    let Some(_dir) = f_child_env(module_path!(), "test_native_system_end_turn_confirmation_cancel_keeps_turn", |dir| {
        vec![("FD2_NATIVE_SAVE", "".into()), ("XDG_DATA_HOME", dir.as_os_str().to_owned())]
    }) else {
        return;
    };
    let mut g = new_preparation_record_test_game();
    g.camp = None;
    let mut st = battle::State::default();
    (st.w, st.h) = (24, 24);
    g.st = Some(st);
    (g.ring, g.ring_sel, g.native_system_cursor_overlay) = (true, 3, true);
    g.native_map_vga = vec![0; 320 * 200];
    g.native_confirmation_mouth_rng = Some(battle::GoMathRand::new(177));
    assert!(g.begin_native_system_end_turn(), "normal END refused");
    assert!(g.native_system_end_turn_ui.is_none(), "END state published before the overlay closed");
    for _ in 0..4 {
        g.mark_action_overlay_drawn();
        g.step_action_overlay_lifecycle();
    }
    assert!(!g.native_system_cursor_overlay && g.native_system_end_turn_confirm, "overlay did not hand over to END");
    assert!(g.native_system_end_turn_ui.as_ref().is_some_and(|state| state.portraits.len() >= 4));
    finish_native_class_ui_job(&mut g);
    assert!(g.native_system_end_turn_confirm, "opening frames consumed the confirmation");
    g.cancel_native_system_end_turn();
    assert!(!g.native_system_end_turn_confirm, "cancel kept the confirmation open");
    finish_native_class_ui_job(&mut g);
    finish_native_class_ui_job(&mut g);
    assert_eq!(g.native_system_end_turn_delay, NATIVE_SYSTEM_END_TURN_DELAY_FRAMES);
    for _ in 0..NATIVE_SYSTEM_END_TURN_DELAY_FRAMES {
        g.step_native_system_end_turn();
    }
    finish_native_class_ui_job(&mut g);
    let state = g.st.as_ref().unwrap();
    assert!(g.native_system_end_turn_ui.is_none() && state.turn == 0 && !g.ai_busy, "cancel committed the turn");
}

/// Go：`native_confirmation_mouth_test.go:122` `TestNativeConfirmationMouthMissingFrameFailsBeforeENDMutation`
#[test]
fn test_native_confirmation_mouth_missing_frame_fails_before_end_mutation() {
    if !asset_pack_ready("ui/fdother_010_town_label/resource.json") {
        return;
    }
    let Some(_dir) =
        f_child_env(module_path!(), "test_native_confirmation_mouth_missing_frame_fails_before_end_mutation", |dir| {
            vec![("FD2_NATIVE_SAVE", "".into()), ("XDG_DATA_HOME", dir.as_os_str().to_owned())]
        })
    else {
        return;
    };
    let mut g = new_preparation_record_test_game();
    g.camp = None;
    let mut st = battle::State::default();
    (st.w, st.h) = (24, 24);
    g.st = Some(st);
    (g.ring, g.ring_sel, g.native_system_cursor_overlay) = (true, 3, true);
    g.native_map_vga = vec![0; 320 * 200];
    g.native_preparation_ui.as_mut().unwrap().portraits.truncate(1);
    assert!(
        !g.begin_native_system_end_turn()
            && g.ring
            && g.native_system_cursor_overlay
            && g.native_system_end_turn_ui.is_none()
            && g.action_overlay_phase.is_empty(),
        "missing DATO3 changed END owner"
    );
}
