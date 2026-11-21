//! R5 階段 4 r5i 的局部黃金（Rust 端）。Go 端是 `remake/cmd/fd2/rust_golden_r5i_controls_test.go`，驅動腳本
//! `tools/rust_r5i_controls.sh`。讀 Go 輸出的五份 JSON，以同一組輸入重跑 Rust 的移植並逐項比對：
//!
//! | 檔 | 比對 |
//! |---|---|
//! | `keys.json` | Ebiten 鍵的數值與 `String()`、`UnmarshalText` 的結果、面板的鍵名 |
//! | `settings.json` | 隨機綁定寫出的 `controls-settings.json` 位元組、讀回的綁定與狀態；手寫檔案的讀檔結果 |
//! | `input.json` | 每一步 `step` 之後的指向、109 個鍵的 `duration`、`appendJustPressed`、手把數與鎖定 |
//! | `pointer.json` | 地圖格、指令環方向、地圖跟隨、各選單的矩形與指向後的選取列（Rust 依同一組迴圈產生，逐筆比） |
//! | `panel.json` | 面板操作序列每一步之後的狀態與設定檔內容；擷取點的矩形層雜湊與 `text.Draw` 清單逐項相同；整張疊層的 TTF 像素差異只統計（KNOWN_DIFFERENCES `ttf-raster-coverage`），且必須落在文字外框內 |
//!
//! 報告寫到 `$FD2_R5I_GOLDEN_OUT/report.json`；任何一項不同就失敗。

use super::*;
use crate::go_ebiten_keys as ek;
use serde_json::{Value, json};
use sha2::{Digest, Sha256};
use std::path::PathBuf;

fn sha(b: &[u8]) -> String {
    Sha256::digest(b).iter().map(|x| format!("{x:02x}")).collect()
}

fn read_json(dir: &std::path::Path, name: &str) -> Value {
    let raw = std::fs::read(dir.join(name)).unwrap_or_else(|e| panic!("{name}: {e}"));
    serde_json::from_slice(&raw).unwrap_or_else(|e| panic!("{name}: {e}"))
}

fn int(v: &Value) -> i64 {
    v.as_i64().unwrap_or_else(|| panic!("不是整數：{v}"))
}

fn ints(v: &Value) -> Vec<i64> {
    v.as_array().map(|a| a.iter().map(int).collect()).unwrap_or_default()
}

/// 比對結果：每節的筆數與前幾筆不同。
#[derive(Default)]
struct Section {
    checked: u64,
    diffs: Vec<String>,
    count: u64,
}

impl Section {
    fn check(&mut self, ok: bool, what: impl FnOnce() -> String) {
        self.checked += 1;
        if !ok {
            self.count += 1;
            if self.diffs.len() < 20 {
                self.diffs.push(what());
            }
        }
    }

    fn json(&self) -> Value {
        json!({"checked": self.checked, "different": self.count, "first": self.diffs})
    }
}

fn bindings_of(v: &Value) -> ControlBindings {
    ControlBindings {
        keyboard: ints(&v["keyboard"]).into_iter().map(|k| k as EbitenKey).collect(),
        gamepad: ints(&v["gamepad"]).into_iter().map(|b| b as ControlPadButton).collect(),
        deadzone: int(&v["deadzone"]),
    }
}

fn status_code(s: ControlsLoadStatus) -> i64 {
    match s {
        ControlsLoadStatus::Default => 0,
        ControlsLoadStatus::Ok => 1,
        ControlsLoadStatus::Partial => 2,
        ControlsLoadStatus::Corrupt => 3,
    }
}

fn keys(dir: &std::path::Path) -> Section {
    let v = read_json(dir, "keys.json");
    let mut s = Section::default();
    for k in v["keys"].as_array().unwrap() {
        let (code, name) = (int(&k["code"]) as EbitenKey, k["name"].as_str().unwrap());
        s.check(ek::key_string(code) == name, || format!("String({code})：Rust {}", ek::key_string(code)));
    }
    for p in v["parsed"].as_array().unwrap() {
        let name = p["name"].as_str().unwrap();
        let got = ek::key_from_name(name).map_or(-2, |k| k as i64);
        s.check(got == int(&p["code"]), || format!("UnmarshalText({name:?})：Go {} Rust {got}", p["code"]));
    }
    for (code, label) in v["labels"].as_object().unwrap() {
        let k: EbitenKey = code.parse().unwrap();
        let got = control_key_label(k);
        s.check(got == label.as_str().unwrap(), || format!("controlKeyLabel({k})：Go {label} Rust {got:?}"));
    }
    s
}

fn settings(dir: &std::path::Path) -> Section {
    let v = read_json(dir, "settings.json");
    let mut s = Section::default();
    let tmp = crate::test_support::temp_dir("r5i-golden-settings");
    let path = tmp.join("controls-settings.json").to_string_lossy().into_owned();
    for (i, case) in v["saved"].as_array().unwrap().iter().enumerate() {
        let b = bindings_of(&case["bindings"]);
        save_controls_settings_to(&path, &b).unwrap();
        let raw = std::fs::read_to_string(&path).unwrap();
        let want = case["file"].as_str().unwrap();
        s.check(raw == want, || format!("saved[{i}] 位元組：Go {want:?} Rust {raw:?}"));
        std::fs::write(&path, want).unwrap();
        let (loaded, status) = load_controls_settings_from(&path);
        s.check(loaded == bindings_of(&case["loaded"]) && status_code(status) == int(&case["status"]), || {
            format!("saved[{i}] 讀回：Rust {loaded:?} {status:?}")
        });
    }
    for case in v["loads"].as_array().unwrap() {
        let file = case["file"].as_str().unwrap();
        std::fs::write(&path, file).unwrap();
        let (loaded, status) = load_controls_settings_from(&path);
        s.check(loaded == bindings_of(&case["loaded"]) && status_code(status) == int(&case["status"]), || {
            format!("讀檔 {file:?}：Go status {} Rust {loaded:?} {status:?}", case["status"])
        });
    }
    let _ = std::fs::remove_dir_all(&tmp);
    s
}

fn accept_below_toolbar(x: i64, y: i64) -> (i64, i64, bool) {
    if y < 24 || !(0..LOGICAL_W).contains(&x) || y >= LOGICAL_H {
        return (0, 0, false);
    }
    (x, y, true)
}

fn snapshot_of(v: &Value) -> InputDeviceSnapshot {
    let b = |k: &str| v[k].as_bool().unwrap();
    InputDeviceSnapshot {
        focused: b("focused"),
        cursor_x: int(&v["cursor_x"]),
        cursor_y: int(&v["cursor_y"]),
        left: b("left"),
        right: b("right"),
        left_just: b("left_just"),
        right_just: b("right_just"),
        wheel_y: v["wheel_y"].as_f64().unwrap(),
        pads: v["pads"]
            .as_array()
            .into_iter()
            .flatten()
            .map(|p| {
                let mut pad = GamepadSnapshot {
                    name: p["name"].as_str().unwrap().to_string(),
                    stick_x: p["stick_x"].as_f64().unwrap(),
                    stick_y: p["stick_y"].as_f64().unwrap(),
                    ..Default::default()
                };
                for (i, pressed) in p["buttons"].as_array().unwrap().iter().enumerate() {
                    pad.buttons[i] = pressed.as_bool().unwrap();
                }
                pad
            })
            .collect(),
    }
}

fn input(dir: &std::path::Path) -> Section {
    let v = read_json(dir, "input.json");
    let mut s = Section::default();
    let none = TickInput::default();
    let accept: &dyn Fn(i64, i64) -> (i64, i64, bool) = &accept_below_toolbar;
    for seq in v["sequences"].as_array().unwrap() {
        let seed = int(&seq["seed"]);
        let mut input = PlayerInput::new(bindings_of(&seq["bindings"]), ControlsLoadStatus::Ok);
        for (i, step) in seq["steps"].as_array().unwrap().iter().enumerate() {
            if let Some(b) = step.get("set_bindings").filter(|b| !b.is_null()) {
                input.set_bindings(bindings_of(b));
            }
            let ptr = input.step(&snapshot_of(&step["snapshot"]), step["game_input"].as_bool().unwrap(), Some(accept));
            for k in ints(&step["suppress"]) {
                input.suppress_virtual(k as EbitenKey);
            }
            let got_ptr = vec![ptr.active as i64, ptr.clicked as i64, ptr.x, ptr.y];
            s.check(got_ptr == ints(&step["pointer"]), || format!("seed {seed} 第 {i} 步指向：Rust {got_ptr:?}"));
            let want: BTreeMap<i64, i64> =
                step["durations"].as_object().unwrap().iter().map(|(k, d)| (k.parse().unwrap(), int(d))).collect();
            let got: BTreeMap<i64, i64> = (0..=ek::KEY_MAX)
                .filter_map(|k| Some((k as i64, input.duration(&none, k))).filter(|(_, d)| *d != 0))
                .collect();
            s.check(got == want, || format!("seed {seed} 第 {i} 步 duration：Go {want:?} Rust {got:?}"));
            let just: Vec<i64> = input.append_just_pressed(&none, Vec::new()).into_iter().map(|k| k as i64).collect();
            s.check(just == ints(&step["just_pressed"]), || {
                format!("seed {seed} 第 {i} 步 appendJustPressed：Rust {just:?}")
            });
            s.check(
                input.pad_count == int(&step["pad_count"])
                    && input.pad_latched == step["pad_latched"].as_bool().unwrap(),
                || format!("seed {seed} 第 {i} 步手把：Rust {} {}", input.pad_count, input.pad_latched),
            );
        }
    }
    s
}

fn native_pointer_game(camera_x: i64, camera_y: i64, visible: (i64, i64)) -> Game {
    let mut st = battle::State::default();
    (st.w, st.h, st.has_native_map_view_state) = (40, 30, true);
    st.native_map_view_state = battle::NativeMapViewState {
        camera_x,
        camera_y,
        cursor_x: camera_x + 6,
        cursor_y: camera_y + 4,
        visible_cursor_x: visible.0,
        visible_cursor_y: visible.1,
    };
    Game {
        st: Some(st),
        m: Some(MapData { w: 40, h: 30, tile_w: 48, tile_h: 48, ..Default::default() }),
        ..Default::default()
    }
}

/// Go `r5iGoldenPointer` 的同一組迴圈（順序相同），逐筆比。
fn pointer(dir: &std::path::Path) -> Section {
    let v = read_json(dir, "pointer.json");
    let mut s = Section::default();
    let mut cells = Vec::new();
    for camera in [(0, 0), (3, 5), (27, 22)] {
        for visible in [(6, 4), (0, 0), (12, 7)] {
            let mut g = native_pointer_game(camera.0, camera.1, visible);
            let mut x = -20;
            while x < 660 {
                let mut y = -20;
                while y < 420 {
                    g.pointer = PointerTick { active: true, clicked: false, x, y };
                    let (cx, cy, ok) = g.pointer_map_cell();
                    let (dir, dok) = g.pointer_overlay_direction();
                    cells.push((x, y, [cx * 1000 + cy, ok as i64, dir * 10 + dok as i64]));
                    y += 7;
                }
                x += 7;
            }
        }
    }
    for (cam_x, cam_y, tile_w, tile_h) in [(0.0, 0.0, 48, 48), (96.7, 48.2, 48, 48), (-30.5, 10.0, 24, 32)] {
        let mut g = Game {
            m: Some(MapData { w: 20, h: 20, tile_w, tile_h, ..Default::default() }),
            cam_x,
            cam_y,
            ..Default::default()
        };
        let mut x = -20;
        while x < 660 {
            let mut y = -20;
            while y < 420 {
                g.pointer = PointerTick { active: true, clicked: false, x, y };
                let (cx, cy, ok) = g.pointer_map_cell();
                (g.cur_x, g.cur_y) = (3, 4);
                g.follow_pointer_on_map();
                cells.push((x, y, [cx * 1000 + cy, ok as i64, g.cur_x * 1000 + g.cur_y]));
                y += 11;
            }
            x += 11;
        }
    }
    let want = v["cells"].as_array().unwrap();
    s.check(want.len() == cells.len(), || format!("格子筆數：Go {} Rust {}", want.len(), cells.len()));
    for (w, (x, y, out)) in want.iter().zip(&cells) {
        s.check(int(&w["x"]) == *x && int(&w["y"]) == *y && ints(&w["out"]) == out.to_vec(), || {
            format!("{} ({x},{y})：Go {} Rust {out:?}", w["case"], w["out"])
        });
    }
    let to_ints =
        |rs: &[Rectangle]| -> Vec<Vec<i64>> { rs.iter().map(|r| vec![r.min.x, r.min.y, r.max.x, r.max.y]).collect() };
    let mut rects: Vec<Vec<Vec<i64>>> = Vec::new();
    let mut menus = Vec::new();
    for w in [40, 57, 64, 101] {
        let mut assets = TitleAssets::default();
        for (i, item) in assets.items.iter_mut().enumerate() {
            item[0] = Some(Arc::new(Picture::new_image(w + i as i64 * 3, 8)));
        }
        let mut g = Game { title_phase: String::from("menu"), title_assets: Some(assets), ..Default::default() };
        rects.push(to_ints(&g.title_menu_rects()));
        for start in 0..3 {
            for x in (150..500).step_by(13) {
                for y in (320..400).step_by(3) {
                    (g.title_sel, g.title_flash) = (start, 0);
                    g.pointer = PointerTick { active: true, clicked: false, x, y };
                    g.apply_pointer_to_title_menu();
                    menus.push((start, x, y, g.title_sel));
                }
            }
        }
    }
    rects.push(to_ints(&title_slot_rects()));
    for variant in 0..3 {
        rects.push(to_ints(&native_town_selection_rects(variant)));
        for start in 0..5 {
            for x in (0..640).step_by(9) {
                for y in (0..400).step_by(9) {
                    let mut g = Game { camp_sel: start, ..Default::default() };
                    g.pointer = PointerTick { active: true, clicked: false, x, y };
                    g.apply_pointer_to_native_town(variant);
                    menus.push((start, x, y, g.camp_sel));
                }
            }
        }
    }
    for count in 0..=6 {
        rects.push(to_ints(&choice_menu_rects(count)));
    }
    let want = v["rects"].as_array().unwrap();
    s.check(want.len() == rects.len(), || format!("矩形組數：Go {} Rust {}", want.len(), rects.len()));
    for (w, got) in want.iter().zip(&rects) {
        let w_rects: Vec<Vec<i64>> = w["rects"].as_array().unwrap().iter().map(ints).collect();
        s.check(w_rects == *got, || format!("{}：Go {w_rects:?} Rust {got:?}", w["case"]));
    }
    let want = v["menus"].as_array().unwrap();
    s.check(want.len() == menus.len(), || format!("選單筆數：Go {} Rust {}", want.len(), menus.len()));
    for (w, &(start, x, y, result)) in want.iter().zip(&menus) {
        s.check(
            int(&w["start"]) == start && int(&w["x"]) == x && int(&w["y"]) == y && int(&w["result"]) == result,
            || format!("{} start={start} ({x},{y})：Go {} Rust {result}", w["case"], w["result"]),
        );
    }
    s
}

fn panel_input_of(v: &Value) -> ControlsPanelInput {
    let b = |k: &str| v[k].as_bool().unwrap();
    let point = ints(&v["Point"]);
    ControlsPanelInput {
        up: b("Up"),
        down: b("Down"),
        left: b("Left"),
        right: b("Right"),
        page_up: b("PageUp"),
        page_down: b("PageDown"),
        confirm: b("Confirm"),
        back: b("Back"),
        clear: b("Clear"),
        reset: b("Reset"),
        close: b("Close"),
        capture_key: int(&v["CaptureKey"]) as EbitenKey,
        capture_button: int(&v["CaptureButton"]) as ControlPadButton,
        capture_cancel: b("CaptureCancel"),
        click: b("Click"),
        drag: b("Drag"),
        point: (point[0], point[1]),
        wheel: int(&v["Wheel"]),
    }
}

/// 面板節的統計：狀態、矩形層、文字清單逐項；整張疊層的 TTF 差異另計。
#[derive(Default)]
struct PanelStats {
    shots: u64,
    full_equal: u64,
    stored: u64,
    diff_pixels: u64,
    total_pixels: u64,
    max_channel: u8,
    outside_text: u64,
}

fn panel(dir: &std::path::Path, stats: &mut PanelStats) -> Section {
    let v = read_json(dir, "panel.json");
    let mut s = Section::default();
    let shots = v["shots"].as_array().unwrap();
    let base = load_font().expect("需要 Noto Sans CJK（tools/rust_ttf_font.sh）");
    for run in v["runs"].as_array().unwrap() {
        let locale = run["locale"].as_str().unwrap();
        let tmp = crate::test_support::temp_dir("r5i-golden-panel");
        let settings_path = tmp.join("controls-settings.json").to_string_lossy().into_owned();
        CONTROLS_SETTINGS_PATH_OVERRIDE.with(|p| *p.borrow_mut() = Some(settings_path.clone()));
        let path =
            crate::test_support::remake_asset(&format!("locales/{locale}/controls.json")).expect("controls.json");
        let text = load_controls_locale_from(&path, locale).unwrap();
        let mut g = Game {
            player_input: Some(PlayerInput::new(default_control_bindings(), ControlsLoadStatus::Default)),
            locale_id: locale.to_string(),
            font: Some(base.clone()),
            ..Default::default()
        };
        g.open_controls_panel();
        g.controls.text = Some(text.clone());
        let steps = run["steps"].as_array().unwrap();
        let states = run["states"].as_array().unwrap();
        s.check(steps.len() == states.len(), || format!("{locale} 步數"));
        for (i, (step, want)) in steps.iter().zip(states).enumerate() {
            if !g.controls.open {
                g.open_controls_panel();
                g.controls.text = Some(text.clone());
            }
            g.step_controls_panel(panel_input_of(step));
            let c = &g.controls;
            let input = g.player_input.as_ref().unwrap();
            let conflict = c.conflict.map(|x| {
                vec![x.action as i64, x.other as i64, x.reset as i64, x.gamepad as i64, x.key as i64, x.button as i64]
            });
            let saved = std::fs::read_to_string(&settings_path).unwrap_or_default();
            let ok = want["open"].as_bool().unwrap() == c.open
                && int(&want["selected"]) == c.selected as i64
                && int(&want["scroll"]) == c.scroll
                && want["capture"].as_bool().unwrap() == c.capture
                && (if want["conflict"].is_null() {
                    conflict.is_none()
                } else {
                    conflict == Some(ints(&want["conflict"]))
                })
                && want["message"].as_str().unwrap() == c.message
                && want["dragging"].as_bool().unwrap() == c.dragging
                && bindings_of(&want["bindings"]) == input.bindings
                && int(&want["status"]) == status_code(input.status)
                && want["saved"].as_str().unwrap() == saved;
            s.check(ok, || {
                format!(
                    "{locale} 第 {i} 步狀態：Go {want} Rust open={} sel={} scroll={} capture={} conflict={conflict:?} msg={:?} drag={} {:?} saved={saved:?}",
                    c.open, c.selected, c.scroll, c.capture, c.message, c.dragging, input.bindings
                )
            });
            let shot = int(&want["shot"]);
            if shot < 0 {
                continue;
            }
            let w = &shots[shot as usize];
            g.player_input.as_mut().unwrap().pad_count = int(&w["pad_count"]);
            let mut rect_draws = Vec::new();
            let mut full_draws = Vec::new();
            let mut surface = Picture::new_image(CONTROLS_SURFACE_W, CONTROLS_SURFACE_H);
            g.draw_controls_panel_surface(&mut surface, Some(&mut rect_draws), false);
            let rect_sha = sha(&surface.pix);
            g.draw_controls_panel_surface(&mut surface, Some(&mut full_draws), true);
            g.player_input.as_mut().unwrap().pad_count = 0;
            stats.shots += 1;
            s.check(rect_sha == w["rect_sha256"].as_str().unwrap(), || {
                format!("擷取 {shot}（{locale} 第 {i} 步）矩形層")
            });
            s.check(rect_draws == full_draws, || format!("擷取 {shot} 兩次繪製的文字清單不同"));
            let want_draws: Vec<ControlsTextDraw> = w["text_draws"]
                .as_array()
                .unwrap()
                .iter()
                .map(|d| ControlsTextDraw {
                    text: d["text"].as_str().unwrap().to_string(),
                    x: int(&d["x"]),
                    y: int(&d["y"]),
                    size: int(&d["size"]),
                    color: {
                        let c = ints(&d["color"]);
                        [c[0] as u8, c[1] as u8, c[2] as u8, c[3] as u8]
                    },
                    bound: {
                        let b = ints(&d["bound"]);
                        [b[0], b[1], b[2], b[3]]
                    },
                })
                .collect();
            s.check(want_draws == full_draws, || {
                let first = want_draws.iter().zip(&full_draws).position(|(a, b)| a != b);
                format!(
                    "擷取 {shot} 文字清單（Go {} 筆、Rust {} 筆）第一個不同 {first:?}：Go {:?} Rust {:?}",
                    want_draws.len(),
                    full_draws.len(),
                    first.map(|j| &want_draws[j]),
                    first.map(|j| &full_draws[j])
                )
            });
            if sha(&surface.pix) == w["full_sha256"].as_str().unwrap() {
                stats.full_equal += 1;
            }
            let file = w["file"].as_str().unwrap();
            if file.is_empty() {
                continue;
            }
            stats.stored += 1;
            let go = std::fs::read(dir.join(file)).unwrap();
            s.check(go.len() == surface.pix.len(), || format!("{file} 大小"));
            // 文字外框（平移到原點，四周放寬 2 像素）以外不能有差異。
            let boxes: Vec<[i64; 4]> = full_draws
                .iter()
                .map(|d| [d.x + d.bound[0] - 2, d.y + d.bound[1] - 2, d.x + d.bound[2] + 2, d.y + d.bound[3] + 2])
                .collect();
            for (j, (a, b)) in go.as_chunks::<4>().0.iter().zip(surface.pix.as_chunks::<4>().0).enumerate() {
                stats.total_pixels += 1;
                if a == b {
                    continue;
                }
                stats.diff_pixels += 1;
                let m = a.iter().zip(b).map(|(x, y)| x.abs_diff(*y)).max().unwrap();
                stats.max_channel = stats.max_channel.max(m);
                let (x, y) = ((j as i64) % CONTROLS_SURFACE_W, (j as i64) / CONTROLS_SURFACE_W);
                if !boxes.iter().any(|bx| bx[0] <= x && x < bx[2] && bx[1] <= y && y < bx[3]) {
                    stats.outside_text += 1;
                }
            }
        }
        CONTROLS_SETTINGS_PATH_OVERRIDE.with(|p| *p.borrow_mut() = None);
        let _ = std::fs::remove_dir_all(&tmp);
    }
    s.check(stats.outside_text == 0, || format!("文字外框以外有 {} 個不同像素", stats.outside_text));
    s
}

/// `tools/rust_r5i_controls.sh` 執行：`FD2_R5I_GOLDEN_IN` 是 Go 輸出目錄，報告寫到 `FD2_R5I_GOLDEN_OUT`。
#[test]
#[ignore]
fn test_rust_golden_r5i_controls() {
    let dir = PathBuf::from(std::env::var("FD2_R5I_GOLDEN_IN").expect("需要 FD2_R5I_GOLDEN_IN（Go 輸出目錄）"));
    let out = PathBuf::from(std::env::var("FD2_R5I_GOLDEN_OUT").expect("需要 FD2_R5I_GOLDEN_OUT"));
    assert!(crate::test_support::cjk_font_available(), "需要 Go 黃金用的 Noto Sans CJK");
    let mut stats = PanelStats::default();
    let sections = [
        ("keys", keys(&dir)),
        ("settings", settings(&dir)),
        ("input", input(&dir)),
        ("pointer", pointer(&dir)),
        ("panel", panel(&dir, &mut stats)),
    ];
    let mut report = serde_json::Map::new();
    let mut failed = 0;
    for (name, section) in &sections {
        report.insert(name.to_string(), section.json());
        failed += section.count;
    }
    report.insert(
        String::from("panel_pixels"),
        json!({
            "shots": stats.shots, "full_surface_equal": stats.full_equal, "stored_surfaces": stats.stored,
            "pixels": stats.total_pixels, "different": stats.diff_pixels, "max_channel_diff": stats.max_channel,
            "outside_text_bounds": stats.outside_text,
        }),
    );
    report.insert(String::from("manifest_sha256"), json!(sha(&std::fs::read(dir.join("manifest.txt")).unwrap())));
    report.insert(String::from("ok"), json!(failed == 0));
    let raw = serde_json::to_vec_pretty(&Value::Object(report)).unwrap();
    std::fs::write(out.join("report.json"), &raw).unwrap();
    println!("{}", String::from_utf8_lossy(&raw));
    assert_eq!(failed, 0, "r5i 局部黃金有不同，見 report.json");
}
