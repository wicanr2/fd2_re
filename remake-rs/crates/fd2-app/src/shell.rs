//! Bevy 外殼（R5 階段 4，r5w）：視窗、畫格呈現與輸入。
//!
//! 每個顯示幀照 Ebiten `updateFrameImpl`（`internal/ui/context.go:93`）：讀一次實體輸入，跑 N 次 `Update`
//! （N 由 [`EbitenClock`] 依 `60 × speed` TPS 算出，每次都用同一份輸入），再畫一次 640×400 CPU 畫格（每幀先清空）。
//! 畫格上傳成貼圖，以整數倍最近鄰放大置中（[`crate::video`]）。
//!
//! 設 `FD2_APP_SCRIPT=<腳本>`、`FD2_APP_OUT=<目錄>` 時改由腳本驅動（[`crate::script`]）：每個顯示幀的 `Update` 次數與
//! 按鍵由腳本決定，按鍵與觸控以 Bevy 的 `KeyboardInput`／`TouchInput` 訊息注入，走與實體鍵盤相同的路徑；每個顯示幀的
//! CPU 畫格雜湊寫到 `frames.jsonl`，腳本的 `shot` 截取視窗並逐像素核對（結果寫到 `shots.jsonl`）。截圖期間不前進遊戲、
//! 不重畫，視窗持續顯示同一張畫格。腳本只有按鍵與觸控，所以腳本模式不讀實體滑鼠與手把（視窗焦點當成沒有），
//! 兩個驅動端的輸入相同。
//!
//! R5 階段 4（r5i）：滑鼠右鍵、滾輪、視窗焦點與手把（Bevy 內建的 gilrs 支援）也放進 [`PhysicalSnapshot`]；F12 設定面板
//! 開著時，把 `fd2-game` 畫好的 1920×1200 疊層以線性濾鏡縮成畫格大小，疊在畫格上（Go `drawControlsPanel` 的最後一步）。
//!
//! R5 階段 5（r5g）：`--renderer=gpu` 或 `FD2_RENDERER=gpu` 時畫格改由 GPU 合成（[`crate::gpu_shell`]），預設 CPU。腳本模式
//! 在 GPU 模式下照樣寫 `frames.jsonl`（雜湊取自 GPU 讀回），另寫 `timing.jsonl`（兩種模式都寫）。

use std::collections::{BTreeSet, VecDeque};
use std::io::Write as _;
use std::path::PathBuf;
use std::sync::{Arc, Mutex};
use std::time::Instant;

use bevy::asset::RenderAssetUsages;
use bevy::audio::{AudioPlayer, AudioSink};
use bevy::image::ImageSampler;
use bevy::input::ButtonState;
use bevy::input::gamepad::{AxisSettings, Gamepad, GamepadAxis, GamepadSettings};
use bevy::input::keyboard::{Key as BevyKey, KeyboardInput, NativeKey};
use bevy::input::mouse::{MouseScrollUnit, MouseWheel};
use bevy::input::touch::{TouchInput, TouchPhase};
use bevy::prelude::*;
use bevy::render::render_resource::{Extent3d, TextureDimension, TextureFormat};
use bevy::render::view::screenshot::{Screenshot, ScreenshotCaptured, save_to_disk};
use bevy::window::{MonitorSelection, PresentMode, PrimaryWindow, WindowMode, WindowResolution};
use bevy::winit::{UpdateMode, WinitSettings};
use fd2_game::game::{AudioVoice, Key as GameKey, Picture, RawDevices, VoiceSlot, display_list};
use fd2_game::{PlayerRunner, RunnerStep};
use serde_json::json;
use sha2::{Digest, Sha256};

use crate::audio::{Fd2AudioPlugin, Fd2Sound, GameAudioHost};
use crate::clock::EbitenClock;
use crate::gpu_shell::{Renderer, ShellGpu, requested_renderer};
use crate::input::{Bindings, PhysicalInput, PhysicalSnapshot, gamepad_snapshot};
use crate::script::{Script, frame_record};
use crate::video::{LOGICAL_H, LOGICAL_W, Letterbox, centered_game, expected_window, letterbox, opaque_rgba};

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
    hd_world_image: Handle<Image>,
    hd_world_sprite: Option<Entity>,
    hd_world_source: Option<Arc<Picture>>,
    hd_portrait_image: Handle<Image>,
    hd_portrait_sprite: Option<Entity>,
    hd_portrait_source: Option<(Arc<Picture>, f32)>,
    /// r5i：設定面板疊層的貼圖、精靈與目前上傳的面板狀態。
    panel_image: Handle<Image>,
    panel_sprite: Option<Entity>,
    panel_key: Option<String>,
    poster_sprites: Option<[PosterSprite; 2]>,
    poster_scenery: Handle<Image>,
    poster_scenery_source: Option<Arc<Picture>>,
    script: Option<ScriptRun>,
    /// 上一個顯示幀畫好的 CPU 畫格（截圖核對用）。
    last_frame: Vec<u8>,
    fullscreen: bool,
    /// R5 r5g：要求 GPU 合成時的狀態。
    gpu: Option<ShellGpu>,
    /// 上一個 `drive` 開始的時間（`timing.jsonl` 的幀間隔）。
    last_drive: Option<Instant>,
}

struct PosterSprite {
    picture: Handle<Image>,
    front: Entity,
    background: Entity,
    cached: Option<Arc<Picture>>,
}

impl GameAudioHost for Shell {
    fn audio_voices(&mut self) -> Vec<(VoiceSlot, &mut AudioVoice)> {
        self.runner.audio_voices_mut()
    }
}

/// R5 r5g：GPU 模式下還沒讀回的顯示幀（讀回後照順序寫 `frames.jsonl`）。
struct PendingFrame {
    id: u64,
    index: usize,
    updates: u32,
    game_frame: i64,
    progress: (String, String),
    timing: serde_json::Value,
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
    /// R5 r5g：每個顯示幀的時間（毫秒）。
    timing: std::fs::File,
    pending: VecDeque<PendingFrame>,
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
    let (w, h) = script.as_ref().map_or(((runner.layout().0 as u32) * 2, LOGICAL_H * 2), |(s, _)| s.window);
    let started = Instant::now();
    let script = script.map(|(script, out)| ScriptRun {
        frames: std::fs::File::create(out.join("frames.jsonl")).expect("frames.jsonl"),
        shots: std::fs::File::create(out.join("shots.jsonl")).expect("shots.jsonl"),
        timing: std::fs::File::create(out.join("timing.jsonl")).expect("timing.jsonl"),
        script,
        out,
        next: 0,
        warmup: 0,
        shot: ShotState::Idle,
        injected_keys: BTreeSet::new(),
        injected_touches: BTreeSet::new(),
        shot_failures: 0,
        pending: VecDeque::new(),
    });
    let args: Vec<String> = std::env::args().skip(1).collect();
    let renderer =
        requested_renderer(&args, std::env::var("FD2_RENDERER").ok().as_deref()).unwrap_or_else(|e| panic!("{e}"));
    let shell = Shell {
        runner,
        bindings: Bindings::ebiten(),
        clock: EbitenClock::new(now_ns(started)),
        started,
        image: Handle::default(),
        sprite: None,
        hd_world_image: Handle::default(),
        hd_world_sprite: None,
        hd_world_source: None,
        hd_portrait_image: Handle::default(),
        hd_portrait_sprite: None,
        hd_portrait_source: None,
        panel_image: Handle::default(),
        panel_sprite: None,
        panel_key: None,
        poster_sprites: None,
        poster_scenery: Handle::default(),
        poster_scenery_source: None,
        script,
        last_frame: Vec::new(),
        fullscreen: false,
        gpu: None,
        last_drive: None,
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
    let mut app = App::new();
    app.add_plugins(DefaultPlugins.set(ImagePlugin::default_nearest()).set(WindowPlugin {
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
    .insert_resource(winit);
    // R5 r5g：GPU 模式接到渲染世界（不可用時 ShellGpu 記下原因，退回 CPU）。
    let mut shell = shell;
    if renderer == Renderer::Gpu {
        shell.gpu = Some(ShellGpu::install(&mut app));
    }
    app.add_plugins(Fd2AudioPlugin::<Shell>::default())
        .insert_non_send(shell)
        .add_systems(Startup, setup)
        .add_systems(First, inject_script_input)
        .add_systems(Update, (raw_gamepad_axes, drive).chain())
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
    if let Some(gpu) = &shell.gpu {
        gpu.set_target(shell.image.id());
    }
    shell.sprite = Some(commands.spawn(Sprite::from_image(shell.image.clone())).id());
    let mut world = Image::new_fill(
        Extent3d { width: 1, height: 1, depth_or_array_layers: 1 },
        TextureDimension::D2,
        &[0, 0, 0, 255],
        TextureFormat::Rgba8UnormSrgb,
        RenderAssetUsages::MAIN_WORLD | RenderAssetUsages::RENDER_WORLD,
    );
    world.sampler = ImageSampler::linear();
    shell.hd_portrait_image = images.add(world.clone());
    shell.hd_portrait_sprite =
        Some(commands.spawn((Sprite::from_image(shell.hd_portrait_image.clone()), Visibility::Hidden)).id());
    shell.hd_world_image = images.add(world);
    shell.hd_world_sprite =
        Some(commands.spawn((Sprite::from_image(shell.hd_world_image.clone()), Visibility::Hidden)).id());
    // r5i：設定面板疊層（Go 以 `ebiten.FilterLinear` 縮 1/3 疊到內容區）。
    let mut panel = Image::new_fill(
        Extent3d { width: 1920, height: 1200, depth_or_array_layers: 1 },
        TextureDimension::D2,
        &[0, 0, 0, 0],
        TextureFormat::Rgba8UnormSrgb,
        RenderAssetUsages::MAIN_WORLD | RenderAssetUsages::RENDER_WORLD,
    );
    panel.sampler = ImageSampler::linear();
    shell.panel_image = images.add(panel);
    shell.panel_sprite = Some(
        commands
            .spawn((
                Sprite::from_image(shell.panel_image.clone()),
                Transform::from_xyz(0.0, 0.0, 1.0),
                Visibility::Hidden,
            ))
            .id(),
    );
    let mut poster_image = Image::new_fill(
        Extent3d { width: 1, height: 1, depth_or_array_layers: 1 },
        TextureDimension::D2,
        &[0, 0, 0, 0],
        TextureFormat::Rgba8UnormSrgb,
        RenderAssetUsages::MAIN_WORLD | RenderAssetUsages::RENDER_WORLD,
    );
    poster_image.sampler = ImageSampler::linear();
    shell.poster_scenery = images.add(poster_image.clone());
    shell.poster_sprites = Some(std::array::from_fn(|_| {
        let picture = images.add(poster_image.clone());
        let front = commands.spawn((Sprite::from_image(picture.clone()), Visibility::Hidden)).id();
        let background =
            commands.spawn((Sprite::from_color(Color::srgb_u8(8, 19, 33), Vec2::ONE), Visibility::Hidden)).id();
        PosterSprite { picture, front, background, cached: None }
    }));
}

/// r5i：手把接上時把 Bevy 的搖桿死區與重新映射關掉（Bevy 預設 ±5% 死區、區間外線性放大），讀到的值與 Ebiten 的
/// `StandardGamepadAxisValue` 一樣是原值；防飄移死區由 `fd2-game` 照 Go 的 `stickDirection` 處理。
fn raw_gamepad_axes(mut pads: Query<&mut GamepadSettings, Added<Gamepad>>) {
    for mut settings in &mut pads {
        if let Ok(raw) = AxisSettings::new(-1.0, 0.0, 0.0, 1.0, 0.0) {
            settings.default_axis_settings = raw;
        }
    }
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
    mut sprites: Query<(&mut Sprite, &mut Transform, &mut Visibility)>,
    mut windows: Query<&mut Window, With<PrimaryWindow>>,
    keys: Res<ButtonInput<KeyCode>>,
    mouse: Res<ButtonInput<MouseButton>>,
    touches: Res<Touches>,
    mut key_events: MessageReader<KeyboardInput>,
    mut wheel_events: MessageReader<MouseWheel>,
    gamepads: Query<(Entity, &Gamepad, Option<&Name>)>,
    audio_sinks: Query<&AudioSink, With<AudioPlayer<Fd2Sound>>>,
    mut exit: MessageWriter<AppExit>,
) {
    let Ok(mut window) = windows.single_mut() else {
        return;
    };
    let (w, h, sf) = (window.physical_width(), window.physical_height(), window.scale_factor());
    let shell = &mut *shell;
    let side = shell.runner.poster_side_width();
    let lb = centered_game(w, h, side);

    // r5i：滾輪（Ebiten `Wheel()`：一格是 1，上為正）、焦點與手把。Bevy 的像素單位（觸控板）以 100 像素當一格
    // （KNOWN_DIFFERENCES `wheel-pixel-unit`）。腳本模式不讀（見模組文件）。
    let wheel_y: f64 = wheel_events
        .read()
        .map(|e| match e.unit {
            MouseScrollUnit::Line => e.y as f64,
            MouseScrollUnit::Pixel => e.y as f64 / 100.0,
        })
        .sum();
    let devices = if shell.script.is_some() {
        RawDevices::default()
    } else {
        let mut pads: Vec<_> = gamepads
            .iter()
            .map(|(entity, pad, name)| {
                let stick =
                    (pad.get(GamepadAxis::LeftStickX).unwrap_or(0.0), pad.get(GamepadAxis::LeftStickY).unwrap_or(0.0));
                gamepad_snapshot(entity.to_bits(), name.map_or("", |n| n.as_str()), |b| pad.pressed(b), stick)
            })
            .collect();
        pads.sort_by_key(|p| p.id);
        RawDevices { focused: window.focused, mouse_right: mouse.pressed(MouseButton::Right), wheel_y, pads }
    };

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
        devices,
    };

    // R5 r5g：先收 GPU 讀回的畫格（腳本模式照順序寫紀錄）。
    if let Err(e) = collect_gpu_results(shell) {
        eprintln!("{e}");
        exit.write(AppExit::from_code(4));
        return;
    }

    // 本顯示幀的 Update 次數。
    let use_gpu = shell.gpu.as_mut().is_some_and(ShellGpu::poll_status);
    let updates = match shell.script.as_mut() {
        None => {
            let tps = shell.runner.ticks_per_second();
            shell.clock.update_count(tps, now_ns(shell.started))
        }
        Some(run) => match step_script_state(
            run,
            &mut commands,
            shell.last_frame.as_slice(),
            &mut exit,
            shell.gpu.as_ref().is_none_or(ShellGpu::settled),
            || {
                let renderer =
                    shell.gpu.as_ref().map_or_else(|| json!({"requested": "cpu", "used": "cpu"}), ShellGpu::summary);
                // 測試以實際建立的播放裝置數核對音訊接線。
                (renderer, audio_sinks.iter().count())
            },
        ) {
            Some(updates) => updates,
            None => return,
        },
    };

    let mut raw = shell.bindings.logical_state(&snap, lb);
    if snap.cursor.is_none() {
        raw.cursor = (-1, -1);
    }
    if side != 0 {
        if let Some((x, _)) = snap.cursor {
            raw.cursor.0 = lb.game_x(x, raw.cursor.0);
        }
        for ((_, logical, _), &(_, physical, _)) in raw.touches.iter_mut().zip(&snap.touches) {
            *logical = lb.game_x(physical, *logical);
        }
    }
    let runner = &mut shell.runner;
    let t_frame = Instant::now();
    for i in 0..updates {
        // Ebiten `readInputState` 讀完就把滾輪歸零（`internal/ui/input.go`）：同一顯示幀只有第一次 Update 看得到滾輪。
        if i == 1 {
            raw.devices.wheel_y = 0.0;
        }
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
    let t_update = Instant::now();
    // Ebiten 每幀清空畫面後呼叫 Draw（`IsScreenClearedEveryFrame` 預設 true）。R5 r5g：GPU 模式錄成顯示清單（畫面目標
    // 同樣從全透明開始），交給渲染世界合成，不上傳 CPU 畫格。
    let use_gpu = use_gpu && (shell.script.is_some() || !runner.has_hd_window_layers());
    let mut hd_world = None;
    let (pix, gpu_id) = if use_gpu {
        let list = display_list::record(LOGICAL_W as i64, LOGICAL_H as i64, |screen| runner.draw(screen));
        let id = shell.gpu.as_mut().unwrap().submit(list, shell.script.is_some());
        (None, Some(id))
    } else {
        let mut screen = Picture::new_image(LOGICAL_W as i64, LOGICAL_H as i64);
        if shell.script.is_some() {
            runner.draw(&mut screen);
        } else {
            hd_world = runner.draw_window_layers(&mut screen);
        }
        // HD 世界與介面是兩張貼圖。世界變動時，同一張介面仍可沿用。
        // 進出 HD 的當幀要重傳，因為兩側的 alpha 處理不同。
        let reuse_hd_ui = hd_world.is_some() && shell.hd_world_source.is_some() && screen.pix == shell.last_frame;
        if !reuse_hd_ui && let Some(mut image) = images.get_mut(&shell.image) {
            image.data = Some(if hd_world.is_some() { straight_alpha(&screen.pix) } else { opaque_rgba(&screen.pix) });
        }
        (Some(screen.pix), None)
    };
    let t_draw = Instant::now();
    let progress = runner.progress();
    let game_frame = runner.game_frame();
    let lb = centered_game(w, h, shell.runner.poster_side_width());
    place_sprite(shell, &mut sprites, lb, w, h, sf);
    place_hd_world(shell, hd_world.as_ref(), &mut images, &mut sprites);
    place_hd_portrait(shell, &mut images, &mut sprites);
    place_controls_panel(shell, &mut images, &mut sprites);
    place_posters(shell, &mut images, &mut sprites);
    let interval = shell.last_drive.map(|t| (t_frame - t).as_secs_f64() * 1e3);
    shell.last_drive = Some(t_frame);
    if let Some(run) = shell.script.as_mut() {
        let index = run.next;
        let timing = json!({
            "frame": index, "updates": updates, "renderer": if use_gpu { "gpu" } else { "cpu" },
            "update_ms": (t_update - t_frame).as_secs_f64() * 1e3,
            "draw_ms": (t_draw - t_update).as_secs_f64() * 1e3, "interval_ms": interval,
        });
        let pending = PendingFrame { id: gpu_id.unwrap_or(u64::MAX), index, updates, game_frame, progress, timing };
        match &pix {
            Some(pix) => write_frame(run, pending, pix, None),
            None => run.pending.push_back(pending),
        }
        if let Some(name) = run.script.frames[index].shot.clone() {
            run.shot = ShotState::Settle { name, frames: SHOT_SETTLE_FRAMES };
        }
        run.next += 1;
    }
    if let Some(pix) = pix {
        shell.last_frame = pix;
    }
}

fn place_posters(
    shell: &mut Shell,
    images: &mut Assets<Image>,
    sprites: &mut Query<(&mut Sprite, &mut Transform, &mut Visibility)>,
) {
    let game =
        shell.sprite.and_then(|e| sprites.get(e).ok()).and_then(|(s, t, _)| Some((s.custom_size?, t.translation)));
    let side = shell.runner.poster_side_width();
    let scenery = shell.runner.poster_scenery();
    if let Some((source, _)) = &scenery {
        if shell.poster_scenery_source.as_ref().is_none_or(|last| !Arc::ptr_eq(last, source))
            && let Some(mut image) = images.get_mut(&shell.poster_scenery)
        {
            image.texture_descriptor.size =
                Extent3d { width: source.width as u32, height: source.height as u32, depth_or_array_layers: 1 };
            image.data = Some(opaque_rgba(source.read_pixels()));
            shell.poster_scenery_source = Some(source.clone());
        }
    } else {
        shell.poster_scenery_source = None;
    }
    let layers = shell.runner.poster_sides();
    let Some(sides) = &mut shell.poster_sprites else { return };
    for (index, poster) in sides.iter_mut().enumerate() {
        let (Some((layers, opacity)), Some((game_size, center))) = (layers, game) else {
            poster.cached = None;
            for entity in [poster.front, poster.background] {
                if let Ok((_, _, mut visibility)) = sprites.get_mut(entity)
                    && *visibility != Visibility::Hidden
                {
                    *visibility = Visibility::Hidden;
                }
            }
            continue;
        };
        let source = &layers[index];
        if poster.cached.as_ref().is_none_or(|last| !Arc::ptr_eq(last, source))
            && let Some(mut image) = images.get_mut(&poster.picture)
        {
            image.texture_descriptor.size =
                Extent3d { width: source.width as u32, height: source.height as u32, depth_or_array_layers: 1 };
            image.data = Some(straight_alpha(source.read_pixels()));
            poster.cached = Some(source.clone());
        }
        let size = Vec2::new(game_size.x * side as f32 / LOGICAL_W as f32, game_size.y);
        let distance = (game_size.x + size.x) * 0.5;
        let x = center.x + if index == 0 { -distance } else { distance };
        for (entity, depth) in [(poster.front, -2.0), (poster.background, -3.0)] {
            if let Ok((mut sprite, mut transform, mut visibility)) = sprites.get_mut(entity) {
                if sprite.custom_size != Some(size) {
                    sprite.custom_size = Some(size);
                }
                let position = Vec3::new(x, center.y, depth);
                if transform.translation != position {
                    transform.translation = position;
                }
                if entity == poster.front && sprite.color.alpha() != opacity {
                    sprite.color.set_alpha(opacity);
                }
                if entity == poster.background {
                    let (image, color, crop) = if let Some((_, bounds)) = &scenery {
                        let width = (bounds.max.x - bounds.min.x) * i64::from(side) / i64::from(LOGICAL_W);
                        let x = if index == 0 { bounds.min.x } else { bounds.max.x - width };
                        let crop = Rect::new(x as f32, bounds.min.y as f32, (x + width) as f32, bounds.max.y as f32);
                        (shell.poster_scenery.clone(), Color::WHITE, Some(crop))
                    } else {
                        (Handle::default(), Color::srgb_u8(8, 19, 33), None)
                    };
                    if sprite.image != image || sprite.color != color || sprite.rect != crop {
                        sprite.image = image;
                        sprite.color = color;
                        sprite.rect = crop;
                        sprite.flip_x = crop.is_some();
                    }
                }
                if *visibility != Visibility::Visible {
                    *visibility = Visibility::Visible;
                }
            }
        }
    }
}

/// 寫一個顯示幀的 `frames.jsonl` 與 `timing.jsonl`。
fn write_frame(run: &mut ScriptRun, f: PendingFrame, pix: &[u8], gpu_ms: Option<f64>) {
    writeln!(run.frames, "{}", frame_record(f.index, f.updates, pix, f.game_frame, &f.progress)).expect("frames.jsonl");
    let mut timing = f.timing;
    timing["gpu_ms"] = json!(gpu_ms);
    writeln!(run.timing, "{timing}").expect("timing.jsonl");
}

/// R5 r5g：收 GPU 讀回的畫格，照順序寫紀錄。腳本模式下 GPU 合成失敗回錯誤（驗證不退回 CPU）；一般模式失敗時
/// `ShellGpu` 下一幀起改用 CPU。
fn collect_gpu_results(shell: &mut Shell) -> Result<(), String> {
    let Some(gpu) = shell.gpu.as_ref() else {
        return Ok(());
    };
    for result in gpu.take_results() {
        let Some(run) = shell.script.as_mut() else {
            if let Err(e) = result.pix {
                eprintln!("GPU 合成失敗，改用 CPU 合成：{e}");
            }
            continue;
        };
        let pix = result.pix.map_err(|e| format!("GPU 合成第 {} 份清單：{e}", result.id))?;
        let pending = run.pending.pop_front().ok_or("GPU 讀回沒有對應的顯示幀")?;
        if pending.id != result.id {
            return Err(format!("GPU 讀回順序不對：預期 {}，收到 {}", pending.id, result.id));
        }
        write_frame(run, pending, &pix, Some(result.gpu_ms));
        shell.last_frame = pix;
    }
    Ok(())
}

/// 腳本模式的每幀狀態：回傳本幀要跑的 `Update` 次數；`None` 表示本幀不前進（暖機、截圖中或已結束）。
fn step_script_state(
    run: &mut ScriptRun,
    commands: &mut Commands,
    last_frame: &[u8],
    exit: &mut MessageWriter<AppExit>,
    gpu_settled: bool,
    diagnostics: impl FnOnce() -> (serde_json::Value, usize),
) -> Option<u32> {
    // R5 r5g：GPU 模式等合成器狀態確定（可用或退回 CPU）才開始腳本。
    if run.warmup < SCRIPT_WARMUP_FRAMES || !gpu_settled {
        run.warmup = run.warmup.saturating_add(1);
        return None;
    }
    match std::mem::replace(&mut run.shot, ShotState::Idle) {
        ShotState::Idle => {}
        ShotState::Settle { name, frames } if frames > 0 => {
            run.shot = ShotState::Settle { name, frames: frames - 1 };
            return None;
        }
        // R5 r5g：GPU 模式等這一幀讀回（`last_frame` 更新）才截圖。
        ShotState::Settle { name, frames } if !run.pending.is_empty() => {
            run.shot = ShotState::Settle { name, frames };
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
        if !run.pending.is_empty() {
            return None;
        }
        let (renderer, audio_sinks) = diagnostics();
        let summary = json!({
            "schema": "fd2-app-run/1", "frames": run.next,
            "shots": run.script.frames.iter().filter(|f| f.shot.is_some()).count(), "shot_failures": run.shot_failures,
            "renderer": renderer,
            "audio_sinks": audio_sinks,
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

/// r5i：設定面板開著時，把疊層上傳成貼圖（面板狀態變了才重傳；預乘 alpha 換成 Bevy 精靈用的直接 alpha），擺在畫格
/// 同一個位置與大小、畫在畫格上面；面板關著時隱藏。
fn place_hd_world(
    shell: &mut Shell,
    world: Option<&Arc<Picture>>,
    images: &mut Assets<Image>,
    sprites: &mut Query<(&mut Sprite, &mut Transform, &mut Visibility)>,
) {
    let game = shell.sprite.and_then(|e| sprites.get(e).ok()).map(|(sprite, t, _)| (sprite.custom_size, t.translation));
    let Some((mut sprite, mut transform, mut visibility)) = shell.hd_world_sprite.and_then(|e| sprites.get_mut(e).ok())
    else {
        return;
    };
    let (Some(world), Some((size, translation))) = (world, game) else {
        if *visibility != Visibility::Hidden {
            *visibility = Visibility::Hidden;
        }
        shell.hd_world_source = None;
        return;
    };
    if shell.hd_world_source.as_ref().is_none_or(|source| !Arc::ptr_eq(source, world))
        && let Some(mut image) = images.get_mut(&shell.hd_world_image)
    {
        image.texture_descriptor.size =
            Extent3d { width: world.width as u32, height: world.height as u32, depth_or_array_layers: 1 };
        image.data = Some(opaque_rgba(world.read_pixels()));
        shell.hd_world_source = Some(world.clone());
    }
    if sprite.custom_size != size {
        sprite.custom_size = size;
    }
    let position = Vec3::new(translation.x, translation.y, -1.0);
    if transform.translation != position {
        transform.translation = position;
    }
    if *visibility != Visibility::Visible {
        *visibility = Visibility::Visible;
    }
}

fn place_hd_portrait(
    shell: &mut Shell,
    images: &mut Assets<Image>,
    sprites: &mut Query<(&mut Sprite, &mut Transform, &mut Visibility)>,
) {
    let game = shell.sprite.and_then(|e| sprites.get(e).ok()).map(|(sprite, t, _)| (sprite.custom_size, t.translation));
    let Some((mut sprite, mut transform, mut visibility)) =
        shell.hd_portrait_sprite.and_then(|e| sprites.get_mut(e).ok())
    else {
        return;
    };
    let portrait = shell.script.is_none().then(|| shell.runner.window_portrait()).flatten();
    let (Some(portrait), Some((Some(size), origin))) = (portrait, game) else {
        if *visibility != Visibility::Hidden {
            *visibility = Visibility::Hidden;
        }
        return;
    };
    if shell
        .hd_portrait_source
        .as_ref()
        .is_none_or(|(source, brightness)| !Arc::ptr_eq(source, &portrait.image) || *brightness != portrait.brightness)
        && let Some(mut image) = images.get_mut(&shell.hd_portrait_image)
    {
        image.texture_descriptor.size = Extent3d {
            width: portrait.image.width as u32,
            height: portrait.image.height as u32,
            depth_or_array_layers: 1,
        };
        image.data = Some(straight_alpha(&portrait.rgba()));
        shell.hd_portrait_source = Some((portrait.image.clone(), portrait.brightness));
    }
    let (min, max) = (portrait.rect.min, portrait.rect.max);
    let scale = size / Vec2::new(LOGICAL_W as f32, LOGICAL_H as f32);
    let dimensions = Vec2::new((max.x - min.x) as f32, (max.y - min.y) as f32);
    let x = (min.x + max.x - LOGICAL_W as i64) as f32 * 0.5;
    let y = (LOGICAL_H as i64 - min.y - max.y) as f32 * 0.5;
    sprite.custom_size = Some(dimensions * scale);
    sprite.flip_x = portrait.mirror_x;
    transform.translation = Vec3::new(origin.x + x * scale.x, origin.y + y * scale.y, 0.5);
    *visibility = Visibility::Visible;
}

fn place_controls_panel(
    shell: &mut Shell,
    images: &mut Assets<Image>,
    sprites: &mut Query<(&mut Sprite, &mut Transform, &mut Visibility)>,
) {
    let game = shell.sprite.and_then(|e| sprites.get(e).ok()).map(|(s, t, _)| (s.custom_size, t.translation));
    let Some((mut sprite, mut transform, mut visibility)) = shell.panel_sprite.and_then(|e| sprites.get_mut(e).ok())
    else {
        return;
    };
    let Some((key, surface)) = shell.runner.controls_panel_surface() else {
        if *visibility != Visibility::Hidden {
            *visibility = Visibility::Hidden;
        }
        shell.panel_key = None;
        return;
    };
    if shell.panel_key.as_deref() != Some(key) {
        if let Some(mut image) = images.get_mut(&shell.panel_image) {
            image.data = Some(straight_alpha(&surface.pix));
        }
        shell.panel_key = Some(key.to_string());
    }
    if let Some((size, translation)) = game {
        if sprite.custom_size != size {
            sprite.custom_size = size;
        }
        let t = Vec3::new(translation.x, translation.y, 1.0);
        if transform.translation != t {
            transform.translation = t;
        }
    }
    if *visibility != Visibility::Visible {
        *visibility = Visibility::Visible;
    }
}

/// 預乘 alpha 的 RGBA → 直接 alpha（Bevy 精靈的混色是 `src*a + dst*(1-a)`）。
fn straight_alpha(pix: &[u8]) -> Vec<u8> {
    let mut out = pix.to_vec();
    for px in out.as_chunks_mut::<4>().0 {
        let a = px[3] as u32;
        if a != 0 && a != 255 {
            for c in &mut px[..3] {
                *c = ((*c as u32 * 255 + a / 2) / a).min(255) as u8;
            }
        }
    }
    out
}

/// 依視窗實體大小擺放畫格：倍率 `lb.scale`、左上角在 `(ox, oy)`。Bevy 的 2D 座標以視窗中心為原點、y 向上、
/// 單位是邏輯像素，所以先換算成邏輯像素（除以 `sf`）。
fn place_sprite(
    shell: &Shell,
    sprites: &mut Query<(&mut Sprite, &mut Transform, &mut Visibility)>,
    lb: Letterbox,
    w: u32,
    h: u32,
    sf: f32,
) {
    let Some((mut sprite, mut transform, _)) = shell.sprite.and_then(|e| sprites.get_mut(e).ok()) else {
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
