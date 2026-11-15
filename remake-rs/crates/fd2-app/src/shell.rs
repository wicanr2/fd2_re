//! Bevy 外殼（R5 階段 4，r5w）：視窗、畫格呈現與輸入。
//!
//! 每個顯示幀照 Ebiten `updateFrameImpl`（`internal/ui/context.go:93`）：讀一次實體輸入，跑 N 次 `Update`
//! （N 由 [`EbitenClock`] 依 `60 × speed` TPS 算出，每次都用同一份輸入），再畫一次 640×400 CPU 畫格（每幀先清空）。
//! 畫格上傳成貼圖，以整數倍最近鄰放大置中（[`crate::video`]）。
//!
//! 設 `FD2_APP_SCRIPT=<腳本>`、`FD2_APP_OUT=<目錄>` 時改由腳本驅動（[`crate::script`]）：每個顯示幀的 `Update` 次數與
//! 按鍵由腳本決定，按鍵與觸控以 Bevy 的 `KeyboardInput`／`TouchInput` 訊息注入，走與實體鍵盤相同的路徑；每個顯示幀的
//! CPU 畫格雜湊寫到 `frames.jsonl`，腳本的 `shot` 截取視窗並逐像素核對（結果寫到 `shots.jsonl`）。截圖期間不前進遊戲、
//! 不重畫，視窗持續顯示同一張畫格。

use std::collections::BTreeSet;
use std::io::Write as _;
use std::path::PathBuf;
use std::sync::{Arc, Mutex};
use std::time::Instant;

use bevy::asset::RenderAssetUsages;
use bevy::input::ButtonState;
use bevy::input::keyboard::{Key as BevyKey, KeyboardInput, NativeKey};
use bevy::input::touch::{TouchInput, TouchPhase};
use bevy::prelude::*;
use bevy::render::render_resource::{Extent3d, TextureDimension, TextureFormat};
use bevy::render::view::screenshot::{Screenshot, ScreenshotCaptured, save_to_disk};
use bevy::window::{MonitorSelection, PresentMode, PrimaryWindow, WindowMode, WindowResolution};
use bevy::winit::{UpdateMode, WinitSettings};
use fd2_game::game::{Key as GameKey, Picture};
use fd2_game::{PlayerRunner, RunnerStep};
use serde_json::json;
use sha2::{Digest, Sha256};

use crate::clock::EbitenClock;
use crate::input::{Bindings, PhysicalInput, PhysicalSnapshot};
use crate::script::{Script, frame_record};
use crate::video::{LOGICAL_H, LOGICAL_W, Letterbox, expected_window, letterbox, opaque_rgba};

/// 截圖前等幾個顯示幀（讓貼圖上傳與管線化繪圖跑完）。
const SHOT_SETTLE_FRAMES: u32 = 3;
/// 腳本開始前等幾個顯示幀（視窗建立、大小穩定）。
const SCRIPT_WARMUP_FRAMES: u32 = 10;

/// Bevy 的 non-send 資源：`Game` 有不是 `Sync` 的 callback（`indexedmap` 的 `FrameInput`），所有系統都在主執行緒存取。
struct Shell {
    runner: PlayerRunner,
    bindings: Bindings,
    clock: EbitenClock,
    started: Instant,
    image: Handle<Image>,
    sprite: Option<Entity>,
    script: Option<ScriptRun>,
    /// 上一個顯示幀畫好的 CPU 畫格（截圖核對用）。
    last_frame: Vec<u8>,
    fullscreen: bool,
}

struct ScriptRun {
    script: Script,
    out: PathBuf,
    next: usize,
    warmup: u32,
    shot: ShotState,
    injected_keys: BTreeSet<GameKey>,
    injected_touches: BTreeSet<u64>,
    frames: std::fs::File,
    shots: std::fs::File,
    shot_failures: usize,
}

enum ShotState {
    Idle,
    Settle { name: String, frames: u32 },
    Pending { name: String, slot: Arc<Mutex<Option<Image>>> },
}

fn now_ns(started: Instant) -> i64 {
    started.elapsed().as_nanos() as i64
}

/// 外殼入口。`applyPackagedPlayerDefaults` 由 `bin/fd2.rs` 先處理。
pub fn run() -> AppExit {
    let env = |key: &str| std::env::var(key).ok().filter(|v| !v.is_empty());
    let script = match (env("FD2_APP_SCRIPT"), env("FD2_APP_OUT")) {
        (Some(path), Some(out)) => {
            let raw = std::fs::read_to_string(&path).unwrap_or_else(|e| panic!("讀腳本 {path}：{e}"));
            let script = crate::script::parse(&raw).unwrap_or_else(|e| panic!("{path}：{e}"));
            let out = PathBuf::from(out);
            std::fs::create_dir_all(out.join("shots")).expect("建立輸出目錄");
            Some((script, out))
        }
        (None, None) => None,
        _ => panic!("FD2_APP_SCRIPT 與 FD2_APP_OUT 要一起設定"),
    };
    let runner = PlayerRunner::load();
    if !runner.load_error().is_empty() {
        eprintln!("loadGame：{}", runner.load_error());
    }
    let (w, h) = script.as_ref().map_or((LOGICAL_W * 2, LOGICAL_H * 2), |(s, _)| s.window);
    let started = Instant::now();
    let script = script.map(|(script, out)| ScriptRun {
        frames: std::fs::File::create(out.join("frames.jsonl")).expect("frames.jsonl"),
        shots: std::fs::File::create(out.join("shots.jsonl")).expect("shots.jsonl"),
        script,
        out,
        next: 0,
        warmup: 0,
        shot: ShotState::Idle,
        injected_keys: BTreeSet::new(),
        injected_touches: BTreeSet::new(),
        shot_failures: 0,
    });
    let shell = Shell {
        runner,
        bindings: Bindings::ebiten(),
        clock: EbitenClock::new(now_ns(started)),
        started,
        image: Handle::default(),
        sprite: None,
        script,
        last_frame: Vec::new(),
        fullscreen: false,
    };
    // 腳本模式不等垂直同步、視窗沒有焦點也連續更新：顯示幀與牆鐘無關（每幀的 Update 次數由腳本決定），
    // 只影響跑完腳本要多久。一般模式照 Bevy 預設（垂直同步，同 Ebiten 預設）。
    let scripted = shell.script.is_some();
    let present_mode = if scripted { PresentMode::AutoNoVsync } else { PresentMode::default() };
    let winit = if scripted {
        WinitSettings { focused_mode: UpdateMode::Continuous, unfocused_mode: UpdateMode::Continuous }
    } else {
        WinitSettings::default()
    };
    App::new()
        .add_plugins(DefaultPlugins.set(ImagePlugin::default_nearest()).set(WindowPlugin {
            primary_window: Some(Window {
                // Go `main.go:13482`：`炎龍騎士團2 重製 (fd2_re <版本>)`。
                title: format!("炎龍騎士團2 重製 (fd2_re rust {})", env!("CARGO_PKG_VERSION")),
                // Go `main.go:13481`：`SetWindowSize(logicalW*2, height*2)`，可調整大小。
                resolution: WindowResolution::new(w, h),
                resizable: true,
                present_mode,
                ..default()
            }),
            ..default()
        }))
        .insert_resource(ClearColor(Color::BLACK))
        .insert_resource(winit)
        .insert_non_send(shell)
        .add_systems(Startup, setup)
        .add_systems(First, inject_script_input)
        .add_systems(Update, drive)
        .run()
}

fn setup(mut commands: Commands, mut images: ResMut<Assets<Image>>, mut shell: NonSendMut<Shell>) {
    commands.spawn(Camera2d);
    let image = Image::new_fill(
        Extent3d { width: LOGICAL_W, height: LOGICAL_H, depth_or_array_layers: 1 },
        TextureDimension::D2,
        &[0, 0, 0, 255],
        TextureFormat::Rgba8UnormSrgb,
        RenderAssetUsages::MAIN_WORLD | RenderAssetUsages::RENDER_WORLD,
    );
    shell.image = images.add(image);
    shell.sprite = Some(commands.spawn(Sprite::from_image(shell.image.clone())).id());
}

/// 腳本模式：把本顯示幀要按住的鍵與觸控寫成 Bevy 輸入訊息（`PreUpdate` 的輸入系統會讀）。按住的鍵每幀都重送
/// `Pressed`（已按住時不會產生新的「剛按下」），失去焦點清掉按鍵時也能恢復。
fn inject_script_input(
    mut shell: NonSendMut<Shell>,
    windows: Query<Entity, With<PrimaryWindow>>,
    mut keys: MessageWriter<KeyboardInput>,
    mut touches: MessageWriter<TouchInput>,
) {
    let Ok(window) = windows.single() else {
        return;
    };
    let shell = &mut *shell;
    let Some(run) = shell.script.as_mut() else {
        return;
    };
    if run.warmup < SCRIPT_WARMUP_FRAMES {
        return;
    }
    let Some(frame) = run.script.frames.get(run.next) else {
        return;
    };
    let frame = frame.clone();
    let send = |keys: &mut MessageWriter<KeyboardInput>, key: GameKey, state: ButtonState| {
        let code =
            shell.bindings.keyboard_code_for(key).expect("腳本的鍵都有鍵盤對應（Alt 由 AltLeft／AltRight 組成）");
        keys.write(KeyboardInput {
            key_code: code,
            logical_key: BevyKey::Unidentified(NativeKey::Unidentified),
            state,
            text: None,
            repeat: false,
            window,
        });
    };
    for &key in &frame.keys {
        send(&mut keys, key, ButtonState::Pressed);
    }
    for &key in run.injected_keys.difference(&frame.keys) {
        send(&mut keys, key, ButtonState::Released);
    }
    run.injected_keys = frame.keys.clone();
    let mut active = BTreeSet::new();
    for &(id, x, y) in &frame.touches {
        let phase = if run.injected_touches.contains(&id) { TouchPhase::Moved } else { TouchPhase::Started };
        touches.write(TouchInput { phase, position: Vec2::new(x, y), window, force: None, id });
        active.insert(id);
    }
    for &id in run.injected_touches.difference(&active) {
        touches.write(TouchInput { phase: TouchPhase::Ended, position: Vec2::ZERO, window, force: None, id });
    }
    run.injected_touches = active;
}

// Bevy 系統的參數即依賴注入；拆開只會讓每幀的順序更難讀。
#[allow(clippy::too_many_arguments)]
fn drive(
    mut commands: Commands,
    mut shell: NonSendMut<Shell>,
    mut images: ResMut<Assets<Image>>,
    mut sprites: Query<(&mut Sprite, &mut Transform)>,
    mut windows: Query<&mut Window, With<PrimaryWindow>>,
    keys: Res<ButtonInput<KeyCode>>,
    mouse: Res<ButtonInput<MouseButton>>,
    touches: Res<Touches>,
    mut key_events: MessageReader<KeyboardInput>,
    mut exit: MessageWriter<AppExit>,
) {
    let Ok(mut window) = windows.single_mut() else {
        return;
    };
    let (w, h, sf) = (window.physical_width(), window.physical_height(), window.scale_factor());
    let lb = letterbox(w, h);
    let shell = &mut *shell;
    place_sprite(shell, &mut sprites, lb, w, h, sf);

    // 實體輸入（座標換成實體像素）。
    let snap = PhysicalSnapshot {
        held: keys.get_pressed().map(|&c| PhysicalInput::Keyboard(c)).collect(),
        mouse_left: mouse.pressed(MouseButton::Left),
        cursor: window.cursor_position().map(|p| (p.x * sf, p.y * sf)),
        touches: touches.iter().map(|t| (t.id(), t.position().x * sf, t.position().y * sf)).collect(),
        chars: key_events
            .read()
            .filter(|e| e.state == ButtonState::Pressed)
            .filter_map(|e| e.text.as_ref())
            .flat_map(|t| t.chars().filter(|c| !c.is_control()).collect::<Vec<_>>())
            .collect(),
    };

    // 本顯示幀的 Update 次數。
    let updates = match shell.script.as_mut() {
        None => {
            let tps = shell.runner.ticks_per_second();
            shell.clock.update_count(tps, now_ns(shell.started))
        }
        Some(run) => match step_script_state(run, &mut commands, shell.last_frame.as_slice(), &mut exit) {
            Some(updates) => updates,
            None => return,
        },
    };

    let raw = shell.bindings.logical_state(&snap, lb);
    let runner = &mut shell.runner;
    for _ in 0..updates {
        match runner.update(&raw) {
            Ok(RunnerStep::Continue) => {}
            Ok(RunnerStep::Terminated) => {
                exit.write(AppExit::Success);
                return;
            }
            Err(err) => {
                eprintln!("Update：{err}");
                exit.write(AppExit::from_code(1));
                return;
            }
        }
    }
    if runner.take_fullscreen_request() {
        shell.fullscreen = !shell.fullscreen;
        window.mode = if shell.fullscreen {
            WindowMode::BorderlessFullscreen(MonitorSelection::Current)
        } else {
            WindowMode::Windowed
        };
    }
    // Ebiten 每幀清空畫面後呼叫 Draw（`IsScreenClearedEveryFrame` 預設 true）。
    let mut screen = Picture::new_image(LOGICAL_W as i64, LOGICAL_H as i64);
    runner.draw(&mut screen);
    let progress = runner.progress();
    let game_frame = runner.game_frame();
    if let Some(mut image) = images.get_mut(&shell.image) {
        image.data = Some(opaque_rgba(&screen.pix));
    }
    if let Some(run) = shell.script.as_mut() {
        let index = run.next;
        writeln!(run.frames, "{}", frame_record(index, updates, &screen.pix, game_frame, &progress))
            .expect("frames.jsonl");
        if let Some(name) = run.script.frames[index].shot.clone() {
            run.shot = ShotState::Settle { name, frames: SHOT_SETTLE_FRAMES };
        }
        run.next += 1;
    }
    shell.last_frame = screen.pix;
}

/// 腳本模式的每幀狀態：回傳本幀要跑的 `Update` 次數；`None` 表示本幀不前進（暖機、截圖中或已結束）。
fn step_script_state(
    run: &mut ScriptRun,
    commands: &mut Commands,
    last_frame: &[u8],
    exit: &mut MessageWriter<AppExit>,
) -> Option<u32> {
    if run.warmup < SCRIPT_WARMUP_FRAMES {
        run.warmup += 1;
        return None;
    }
    match std::mem::replace(&mut run.shot, ShotState::Idle) {
        ShotState::Idle => {}
        ShotState::Settle { name, frames } if frames > 0 => {
            run.shot = ShotState::Settle { name, frames: frames - 1 };
            return None;
        }
        ShotState::Settle { name, .. } => {
            let slot = Arc::new(Mutex::new(None));
            let sink = slot.clone();
            commands
                .spawn(Screenshot::primary_window())
                .observe(save_to_disk(run.out.join("shots").join(format!("{name}.png"))))
                .observe(move |captured: On<ScreenshotCaptured>| {
                    *sink.lock().unwrap() = Some(captured.image.clone());
                });
            run.shot = ShotState::Pending { name, slot };
            return None;
        }
        ShotState::Pending { name, slot } => {
            let captured = slot.lock().unwrap().take();
            match captured {
                None => {
                    run.shot = ShotState::Pending { name, slot };
                    return None;
                }
                Some(image) => record_shot(run, &name, &image, last_frame),
            }
            return None;
        }
    }
    if run.next >= run.script.frames.len() {
        let summary = json!({
            "schema": "fd2-app-run/1", "frames": run.next,
            "shots": run.script.frames.iter().filter(|f| f.shot.is_some()).count(), "shot_failures": run.shot_failures,
        });
        std::fs::write(run.out.join("summary.json"), serde_json::to_vec_pretty(&summary).unwrap())
            .expect("summary.json");
        exit.write(if run.shot_failures == 0 { AppExit::Success } else { AppExit::from_code(3) });
        return None;
    }
    Some(run.script.frames[run.next].updates)
}

/// 截圖核對：視窗畫面要等於上一張 CPU 畫格的整數倍最近鄰放大（畫格外黑色）。另把 CPU 畫格存成
/// `shots/<名稱>.rgba`（640×400、alpha 255）給外部工具獨立核對。
fn record_shot(run: &mut ScriptRun, name: &str, image: &Image, last_frame: &[u8]) {
    let (w, h) = (image.width(), image.height());
    let format = image.texture_descriptor.format;
    let data = image.data.clone().unwrap_or_default();
    let bgra = matches!(format, TextureFormat::Bgra8UnormSrgb | TextureFormat::Bgra8Unorm);
    let rgba_ok = bgra || matches!(format, TextureFormat::Rgba8UnormSrgb | TextureFormat::Rgba8Unorm);
    let mut shot = data.clone();
    if bgra {
        for px in shot.as_chunks_mut::<4>().0 {
            px.swap(0, 2);
        }
    }
    let frame = opaque_rgba(last_frame);
    std::fs::write(run.out.join("shots").join(format!("{name}.rgba")), &frame).expect("寫 CPU 畫格");
    let expected = expected_window(&frame, w, h);
    let sha = |b: &[u8]| -> String { Sha256::digest(b).iter().map(|x| format!("{x:02x}")).collect() };
    let (mut diff, mut first) = (0u64, None);
    if rgba_ok && shot.len() == expected.len() {
        for (i, (a, b)) in shot.as_chunks::<4>().0.iter().zip(expected.as_chunks::<4>().0).enumerate() {
            if a != b {
                diff += 1;
                first.get_or_insert(((i as u32) % w, (i as u32) / w));
            }
        }
    }
    let equal = rgba_ok && shot.len() == expected.len() && diff == 0;
    if !equal {
        run.shot_failures += 1;
    }
    let lb: Letterbox = letterbox(w, h);
    let line = json!({
        "name": name, "frame": run.next - 1, "window": [w, h], "scale": lb.scale, "offset": [lb.ox, lb.oy],
        "format": format!("{format:?}"), "bytes": data.len(), "equal": equal, "diff_pixels": diff, "first_diff": first,
        "cpu_rgba_sha256": sha(last_frame), "expected_sha256": sha(&expected), "shot_rgba_sha256": sha(&shot),
    });
    writeln!(run.shots, "{line}").expect("shots.jsonl");
}

/// 依視窗實體大小擺放畫格：倍率 `lb.scale`、左上角在 `(ox, oy)`。Bevy 的 2D 座標以視窗中心為原點、y 向上、
/// 單位是邏輯像素，所以先換算成邏輯像素（除以 `sf`）。
fn place_sprite(
    shell: &Shell,
    sprites: &mut Query<(&mut Sprite, &mut Transform)>,
    lb: Letterbox,
    w: u32,
    h: u32,
    sf: f32,
) {
    let Some((mut sprite, mut transform)) = shell.sprite.and_then(|e| sprites.get_mut(e).ok()) else {
        return;
    };
    let (sw, sh) = ((LOGICAL_W * lb.scale) as f32, (LOGICAL_H * lb.scale) as f32);
    let size = Vec2::new(sw / sf, sh / sf);
    if sprite.custom_size != Some(size) {
        sprite.custom_size = Some(size);
    }
    let cx = lb.ox as f32 + sw / 2.0 - w as f32 / 2.0;
    let cy = lb.oy as f32 + sh / 2.0 - h as f32 / 2.0;
    let t = Vec3::new(cx / sf, -cy / sf, 0.0);
    if transform.translation != t {
        transform.translation = t;
    }
}
