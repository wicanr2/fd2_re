//! Go `player_controls_panel_test.go`（Go v.1.1.15 `7c710b34`）的測試，逐條移植。`TestToolbarControlsButtonOpensPanelAndFits`
//! 測工具列「按鍵」按鈕（R6），沒有移植；`player_controls_panel_gpu_test.go` 的 HD 擷圖由局部黃金
//! （`rust_golden_r5i_test.rs`）取代。
//!
//! Go 把 `userDataDirCached` 換成 `t.TempDir()`；Rust 的測試平行執行，改設本執行緒的設定檔路徑
//! （`CONTROLS_SETTINGS_PATH_OVERRIDE`）。

use super::*;
use crate::go_ebiten_keys as ek;
use crate::test_support;

const LOCALE_IDS: [&str; 4] = ["zh-Hant", "zh-Hans", "ja", "en"];

/// 本執行緒的設定檔改到暫存目錄；回傳目錄（測試結束時刪掉）。
pub(super) fn isolate_settings(tag: &str) -> std::path::PathBuf {
    let dir = test_support::temp_dir(tag);
    let path = dir.join("controls-settings.json").to_string_lossy().into_owned();
    CONTROLS_SETTINGS_PATH_OVERRIDE.with(|p| *p.borrow_mut() = Some(path));
    dir
}

pub(super) fn locale_path(locale: &str) -> Option<String> {
    test_support::remake_asset(&format!("locales/{locale}/controls.json"))
}

/// Go：`player_controls_panel_test.go:16` `controlsPanelFixture`
fn controls_panel_fixture(tag: &str) -> Option<(Game, std::path::PathBuf)> {
    let path = locale_path("zh-Hant")?;
    let dir = isolate_settings(tag);
    let text = load_controls_locale_from(&path, "zh-Hant").unwrap();
    let mut g = Game {
        player_input: Some(PlayerInput::new(default_control_bindings(), ControlsLoadStatus::Default)),
        locale_id: String::from("zh-Hant"),
        ..Default::default()
    };
    g.open_controls_panel();
    g.controls.text = Some(text);
    Some((g, dir))
}

fn panel_step(g: &mut Game, edit: impl FnOnce(&mut ControlsPanelInput)) {
    let mut input = no_controls_panel_input();
    edit(&mut input);
    g.step_controls_panel(input);
}

fn saved_controls() -> ControlsSettingsFile {
    let raw = std::fs::read(controls_settings_path()).unwrap();
    serde_json::from_slice(&raw).unwrap()
}

fn t(g: &Game, key: &str) -> String {
    ControlsLocale::text(g.controls.text.as_ref(), key, &[])
}

fn idx(id: &str) -> usize {
    control_action_index(id) as usize
}

fn bindings(g: &Game) -> &ControlBindings {
    &g.player_input.as_ref().unwrap().bindings
}

fn map(pairs: &[(&str, &str)]) -> Option<BTreeMap<String, String>> {
    Some(pairs.iter().map(|(k, v)| (k.to_string(), v.to_string())).collect())
}

/// Go：`player_controls_panel_test.go:50` `TestControlsLocaleFilesMatchContract`
#[test]
fn test_controls_locale_files_match_contract() {
    let contract = controls_locale_contract();
    for locale in LOCALE_IDS {
        let Some(path) = locale_path(locale) else {
            return;
        };
        let c = load_controls_locale_from(&path, locale).unwrap_or_else(|e| panic!("{locale}: {}", e.message()));
        assert_eq!(c.entries.len(), contract.len(), "{locale} key count");
        let got = ControlsLocale::text(
            Some(&c),
            "controls.msg.conflict",
            &[ControlsArg::S(String::from("Z")), ControlsArg::S(String::from("確認"))],
        );
        assert!(got != "controls.msg.conflict" && !got.is_empty(), "{locale} format {got}");
        assert!(load_controls_locale_from(&path, "xx").is_err(), "locale header mismatch accepted");
        let file: ControlsLocaleFile = serde_json::from_slice(&std::fs::read(&path).unwrap()).unwrap();
        let mutations: [(&str, fn(&mut ControlsLocaleFile)); 4] = [
            ("missing", |f| {
                f.entries.remove("controls.title");
            }),
            ("extra", |f| {
                f.entries.insert(
                    String::from("controls.extra"),
                    ControlsLocaleEntry { text: "x".into(), variables: vec![] },
                );
            }),
            ("variables", |f| {
                f.entries.insert(
                    String::from("controls.pad.connected"),
                    ControlsLocaleEntry { text: "%s".into(), variables: vec!["%s".into()] },
                );
            }),
            ("empty", |f| {
                f.entries.insert(
                    String::from("controls.title"),
                    ControlsLocaleEntry { text: String::new(), variables: vec![] },
                );
            }),
        ];
        for (name, mutate) in mutations {
            let mut copied = file.clone();
            mutate(&mut copied);
            let dir = test_support::temp_dir("r5i-controls-locale");
            let bad = dir.join("controls.json");
            std::fs::write(&bad, serde_json::to_vec(&copied).unwrap()).unwrap();
            assert!(load_controls_locale_from(&bad.to_string_lossy(), locale).is_err(), "{locale} {name} accepted");
            let _ = std::fs::remove_dir_all(&dir);
        }
    }
    assert_eq!(ControlsLocale::text(None, "controls.title", &[]), "controls.title", "missing locale shows the key");
}

/// Go：`player_controls_panel_test.go:102` `TestControlsPanelAssignConflictConfirmAndCancel`
#[test]
fn test_controls_panel_assign_conflict_confirm_and_cancel() {
    let Some((mut g, dir)) = controls_panel_fixture("r5i-panel-assign") else {
        return;
    };
    let (confirm, next) = (idx("confirm"), idx("next_unit"));
    while g.controls.selected != confirm {
        panel_step(&mut g, |i| i.down = true);
    }
    panel_step(&mut g, |i| i.confirm = true);
    assert!(g.controls.capture && g.controls.message == t(&g, "controls.msg.capture"), "Enter starts capture");
    panel_step(&mut g, |i| i.capture_key = ek::KEY_SHIFT);
    assert!(g.controls.capture && g.controls.message == t(&g, "controls.msg.reserved"), "modifier is reserved");
    panel_step(&mut g, |i| i.capture_key = ek::KEY_Z);
    assert!(
        g.controls.conflict.is_some() && bindings(&g).keyboard[confirm] == ek::KEY_ENTER,
        "prompt before replacing"
    );
    let want = ControlsLocale::text(
        g.controls.text.as_ref(),
        "controls.msg.conflict",
        &[ControlsArg::S(String::from("Z")), ControlsArg::S(String::from("換下一位角色"))],
    );
    assert_eq!(g.controls.message, want, "conflict message");
    panel_step(&mut g, |i| i.back = true);
    assert!(
        g.controls.conflict.is_none() && bindings(&g).keyboard[confirm] == ek::KEY_ENTER && g.controls.open,
        "cancel keeps the old binding"
    );
    assert!(std::fs::metadata(controls_settings_path()).is_err(), "cancel must not save");
    panel_step(&mut g, |i| i.confirm = true);
    panel_step(&mut g, |i| i.capture_key = ek::KEY_Z);
    panel_step(&mut g, |i| i.confirm = true);
    assert!(
        bindings(&g).keyboard[confirm] == ek::KEY_Z && bindings(&g).keyboard[next] == KEY_UNBOUND,
        "confirmed replacement"
    );
    assert_eq!(
        g.player_input.as_ref().unwrap().table.produce[ek::KEY_Z as usize],
        ek::KEY_ENTER,
        "mapping table not rebuilt"
    );
    assert_eq!(saved_controls().keyboard, map(&[("confirm", "Z"), ("next_unit", "")]), "saved immediately");
    // 手把：A 指定給空白鍵 → 衝突、點一下確認取代。
    let space = idx("space");
    g.controls.selected = space;
    panel_step(&mut g, |i| i.confirm = true);
    panel_step(&mut g, |i| i.capture_button = PAD_A);
    assert!(g.controls.conflict.is_some_and(|c| c.gamepad), "gamepad conflict");
    panel_step(&mut g, |i| {
        i.click = true;
        i.point = (5, 5);
    });
    assert!(
        bindings(&g).gamepad[space] == PAD_A && bindings(&g).gamepad[confirm] == PAD_NONE,
        "click confirms the replacement"
    );
    panel_step(&mut g, |i| i.confirm = true);
    panel_step(&mut g, |i| i.capture_cancel = true);
    assert!(!g.controls.capture && bindings(&g).gamepad[space] == PAD_A, "B/Esc cancels capture");
    let (loaded, status) = load_controls_settings();
    assert!(status == ControlsLoadStatus::Ok && loaded == *bindings(&g), "reopen keeps the settings");
    let _ = std::fs::remove_dir_all(&dir);
}

/// Go：`player_controls_panel_test.go:166` `TestControlsPanelClearResetSliderAndClose`
#[test]
fn test_controls_panel_clear_reset_slider_and_close() {
    let Some((mut g, dir)) = controls_panel_fixture("r5i-panel-clear") else {
        return;
    };
    let (confirm, space) = (idx("confirm"), idx("space"));
    g.controls.selected = space;
    panel_step(&mut g, |i| i.clear = true);
    assert!(bindings(&g).keyboard[space] == KEY_UNBOUND && bindings(&g).gamepad[space] == PAD_NONE, "Delete clears");
    panel_step(&mut g, |i| i.reset = true);
    assert_eq!(*bindings(&g), default_control_bindings(), "R restores the row");
    // 空白鍵這列改成 Enter，確認那列恢復預設時要先提示。
    let mut b = bindings(&g).clone();
    b.set_keyboard(space, ek::KEY_ENTER);
    g.player_input.as_mut().unwrap().set_bindings(b);
    g.controls.selected = confirm;
    panel_step(&mut g, |i| i.reset = true);
    assert!(
        g.controls.conflict.is_some_and(|c| c.reset) && bindings(&g).keyboard[confirm] == KEY_UNBOUND,
        "reset conflict prompts first"
    );
    let want = ControlsLocale::text(
        g.controls.text.as_ref(),
        "controls.msg.conflict",
        &[ControlsArg::S(String::from("Enter")), ControlsArg::S(g.controls.action_name(space))],
    );
    assert_eq!(g.controls.message, want, "reset conflict message");
    panel_step(&mut g, |i| i.confirm = true);
    assert!(
        bindings(&g).keyboard[confirm] == ek::KEY_ENTER && bindings(&g).keyboard[space] == KEY_UNBOUND,
        "confirmed reset"
    );
    g.controls.selected = CONTROLS_SLIDER_SELECTS - 1;
    panel_step(&mut g, |i| i.down = true);
    assert_eq!(g.controls.selected, CONTROLS_SLIDER_SELECTS, "slider row is after the last action");
    panel_step(&mut g, |i| i.right = true);
    assert!(bindings(&g).deadzone == 25 && saved_controls().stick_deadzone == 25, "slider right");
    for _ in 0..20 {
        panel_step(&mut g, |i| i.left = true);
    }
    assert_eq!(bindings(&g).deadzone, MIN_STICK_DEADZONE, "slider clamps");
    panel_step(&mut g, |i| {
        i.click = true;
        i.point = (CONTROLS_SLIDER_TRACK.max.x, CONTROLS_SLIDER_ROW.min.y + 3);
    });
    assert_eq!(bindings(&g).deadzone, MAX_STICK_DEADZONE, "slider click");
    panel_step(&mut g, |i| {
        i.click = true;
        i.point = (CONTROLS_RESET_ALL_RECT.min.x + 2, CONTROLS_RESET_ALL_RECT.min.y + 2);
    });
    assert!(
        *bindings(&g) == default_control_bindings() && g.controls.message == t(&g, "controls.msg.reset_all"),
        "reset all"
    );
    // 滑鼠：點一列選取，再點同一列開始指定；滾輪捲動。
    g.controls.selected = 0;
    g.controls.scroll = 0;
    let line_y = |g: &Game, action: usize| {
        CONTROLS_LIST_RECT.min.y + (controls_line_of(action) - g.controls.scroll) * CONTROLS_LINE_HEIGHT + 5
    };
    let y = line_y(&g, confirm);
    panel_step(&mut g, |i| {
        i.click = true;
        i.point = (100, y);
    });
    assert!(g.controls.selected == confirm && !g.controls.capture, "click selects");
    panel_step(&mut g, |i| {
        i.click = true;
        i.point = (100, y);
    });
    assert!(g.controls.capture, "second click starts capture");
    panel_step(&mut g, |i| i.back = true);
    panel_step(&mut g, |i| i.wheel = -1);
    panel_step(&mut g, |i| i.wheel = -1);
    assert_eq!(g.controls.scroll, 2, "wheel scroll");
    for _ in 0..40 {
        panel_step(&mut g, |i| i.wheel = -1);
    }
    assert_eq!(g.controls.scroll, controls_lines().len() as i64 - CONTROLS_VISIBLE_LINES, "scroll clamp");
    g.controls.selected = idx("controls");
    g.controls.ensure_visible();
    let line = controls_line_of(g.controls.selected);
    assert!(line >= g.controls.scroll && line < g.controls.scroll + CONTROLS_VISIBLE_LINES, "selected row visible");
    panel_step(&mut g, |i| i.back = true);
    assert!(!g.controls.open, "Esc returns to the game");
    g.open_controls_panel();
    g.controls.capture = true;
    panel_step(&mut g, |i| i.close = true);
    assert!(!g.controls.open, "F12/Start closes from any state");
    let _ = std::fs::remove_dir_all(&dir);
}

/// Go：`player_controls_panel_test.go:264` `TestControlsPanelReportsCorruptSettings`
#[test]
fn test_controls_panel_reports_corrupt_settings() {
    let Some((mut g, dir)) = controls_panel_fixture("r5i-panel-corrupt") else {
        return;
    };
    g.player_input.as_mut().unwrap().status = ControlsLoadStatus::Corrupt;
    let text = g.controls.text.clone();
    g.open_controls_panel();
    g.controls.text = text.clone();
    let want = ControlsLocale::text(text.as_ref(), "controls.msg.corrupt", &[]);
    assert!(g.controls.message == "controls.msg.corrupt" || g.controls.message == want, "corrupt notice");
    let _ = std::fs::remove_dir_all(&dir);
}

/// Rust 新增：Go `readControlsPanelInput` 的換算（實體鍵、滑鼠右鍵、手把按鈕與搖桿連發、指定時第一個剛按下的鍵）。
#[test]
fn read_controls_panel_input_follows_go() {
    let mut g = Game {
        player_input: Some(PlayerInput::new(default_control_bindings(), ControlsLoadStatus::Default)),
        ..Default::default()
    };
    g.controls.open = true;
    let mut state = EbitenInputState::default();
    let mut devices = DeviceInputState::default();
    let mut tick = |g: &mut Game, raw: RawInputState| {
        let physical = state.update(&raw);
        let dev = devices.update(&raw.devices, &physical);
        g.read_controls_panel_input(&physical, &dev)
    };
    let keys = |ks: &[Key], others: &[EbitenKey]| RawInputState {
        keys: ks.iter().copied().collect(),
        other_keys: others.iter().map(|&k| k as u32).collect(),
        ..Default::default()
    };
    let input = tick(&mut g, keys(&[Key::ArrowDown, Key::F12], &[ek::KEY_PAGE_UP, ek::KEY_DELETE, ek::KEY_R]));
    assert!(input.down && input.close && input.page_up && input.clear && input.reset && !input.confirm);
    let input = tick(&mut g, keys(&[Key::ArrowDown, Key::F12], &[ek::KEY_NUMPAD_ENTER]));
    assert!(!input.down && !input.close && input.confirm, "held keys repeat on 1, 12, 17…; NumpadEnter confirms");
    for _ in 0..9 {
        tick(&mut g, keys(&[Key::ArrowDown], &[]));
    }
    assert!(tick(&mut g, keys(&[Key::ArrowDown], &[])).down, "12th tick repeats");
    let right = RawInputState { devices: RawDevices { mouse_right: true, ..Default::default() }, ..Default::default() };
    assert!(tick(&mut g, right.clone()).back && !tick(&mut g, right).back, "right click = back once");
    // 指定中：第一個剛按下的鍵（Ebiten 數值最小）；Esc 是取消；F12 不關面板。
    g.controls.capture = true;
    tick(&mut g, keys(&[], &[]));
    let input = tick(&mut g, keys(&[Key::Z, Key::F12], &[ek::KEY_Q]));
    assert!(input.capture_key == ek::KEY_Q && !input.close, "Q < Z < F12 in Ebiten order");
    let input = tick(&mut g, keys(&[Key::Z, Key::F12, Key::Escape], &[ek::KEY_Q]));
    assert!(input.capture_cancel && input.capture_key == KEY_UNBOUND);
    // 手把：指定中 B 是取消、其他按鈕是要指定的鍵。
    let pad = |buttons: &[ControlPadButton], x: f64| {
        let mut p = GamepadSnapshot { id: 7, stick_x: x, ..Default::default() };
        for &b in buttons {
            p.buttons[b as usize] = true;
        }
        RawInputState {
            devices: RawDevices { focused: true, pads: vec![p], ..Default::default() },
            ..Default::default()
        }
    };
    let input = tick(&mut g, pad(&[PAD_Y], 0.0));
    assert_eq!(input.capture_button, PAD_Y);
    let input = tick(&mut g, pad(&[PAD_Y, PAD_B], 0.0));
    assert!(input.capture_cancel && input.capture_button == PAD_NONE);
    g.controls.capture = false;
    tick(&mut g, pad(&[], 0.0));
    let input = tick(&mut g, pad(&[PAD_A, PAD_START], 0.9));
    assert!(input.confirm && input.close && input.right, "A, Start and the stick");
    let input = tick(&mut g, pad(&[PAD_A, PAD_START], 0.9));
    assert!(!input.confirm && !input.close && !input.right, "stick repeats like the arrows");
    assert_eq!(g.controls.stick_held[3], 2);
}
