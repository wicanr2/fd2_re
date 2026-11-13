//! R0 可行性探針（docs/goal/119，工單 #306）。
//!
//! 只驗證 Bevy 外殼的四件事，結果寫成 JSON 收據：
//! 1. 視窗能在 Docker／Xvfb 的軟體 GPU 開啟並截圖；
//! 2. 640×400 邏輯畫面以整數倍、最近鄰取樣放大；
//! 3. 中文字型可以直接從 TTC 字型集合載入並排版；
//! 4. 原版轉出的 OGG 可解碼，並能建立播放。
//!
//! 環境變數：`FD2_PROBE_FONT`（TTC／TTF 路徑）、`FD2_PROBE_FAMILY`（家族名稱）、
//! `FD2_PROBE_OGG`（以 `:` 分隔）、`FD2_PROBE_SHOT`（PNG 輸出）、`FD2_PROBE_REPORT`（JSON 輸出）。

use bevy::asset::RenderAssetUsages;
use bevy::audio::{AudioSink, Decodable, Source};
use bevy::prelude::*;
use bevy::render::render_resource::{Extent3d, TextureDimension, TextureFormat};
use bevy::render::renderer::RenderAdapterInfo;
use bevy::render::view::screenshot::{Screenshot, ScreenshotCaptured, save_to_disk};
use bevy::text::{FontSource, TextLayoutInfo};
use bevy::window::WindowResolution;
use serde_json::{Value, json};
use std::path::PathBuf;
use std::sync::Arc;

const LOGICAL_W: u32 = 640;
const LOGICAL_H: u32 = 400;
const SCALE: u32 = 2;
const SHOT_FRAME: u32 = 45;
const GIVE_UP_FRAME: u32 = 600;
const PROBE_TEXT: &str = "炎龍騎士團 2　重製\n亞雷斯・悠妮・索爾";

#[derive(Resource)]
struct Probe {
    font_path: Option<PathBuf>,
    family: String,
    shot: PathBuf,
    report: PathBuf,
    audio: Vec<Value>,
    frame: u32,
    captured: bool,
    font_handle: Option<Handle<Font>>,
}

#[derive(Component)]
struct ProbeText(&'static str);

/// 不經 Bevy 播放器，直接用 Bevy 的解碼器完整解一次，記下聲道、取樣率、長度與峰值。
fn decode_ogg(path: &PathBuf) -> (Value, Option<AudioSource>) {
    let bytes = match std::fs::read(path) {
        Ok(bytes) => bytes,
        Err(e) => return (json!({ "path": path, "error": e.to_string() }), None),
    };
    let source = AudioSource { bytes: Arc::from(bytes.into_boxed_slice()) };
    let decoded = std::panic::catch_unwind(|| {
        let decoder = source.decoder();
        let (channels, rate) = (decoder.channels().get(), decoder.sample_rate().get());
        let (mut samples, mut peak) = (0u64, 0f32);
        for sample in decoder {
            samples += 1;
            peak = peak.max(sample.abs());
        }
        (channels, rate, samples, peak)
    });
    match decoded {
        Ok((channels, rate, samples, peak)) => (
            json!({
                "path": path, "bytes": source.bytes.len(), "channels": channels, "sample_rate": rate,
                "samples": samples, "seconds": samples as f64 / (channels as f64 * rate as f64), "peak": peak,
            }),
            Some(source),
        ),
        Err(_) => (json!({ "path": path, "error": "decoder panicked" }), None),
    }
}

fn main() -> AppExit {
    let env = |key: &str| std::env::var(key).ok().filter(|v| !v.is_empty());
    let mut audio = Vec::new();
    let mut sources = Vec::new();
    for path in env("FD2_PROBE_OGG").unwrap_or_default().split(':').filter(|p| !p.is_empty()) {
        let (report, source) = decode_ogg(&PathBuf::from(path));
        audio.push(report);
        sources.extend(source);
    }
    let probe = Probe {
        font_path: env("FD2_PROBE_FONT").map(PathBuf::from),
        family: env("FD2_PROBE_FAMILY").unwrap_or_else(|| "Noto Sans CJK TC".into()),
        shot: env("FD2_PROBE_SHOT").unwrap_or_else(|| "r0-probe.png".into()).into(),
        report: env("FD2_PROBE_REPORT").unwrap_or_else(|| "r0-probe.json".into()).into(),
        audio,
        frame: 0,
        captured: false,
        font_handle: None,
    };
    App::new()
        .add_plugins(DefaultPlugins.set(ImagePlugin::default_nearest()).set(WindowPlugin {
            primary_window: Some(Window {
                title: "炎龍騎士團2 重製 R0 探針".into(),
                resolution: WindowResolution::new(LOGICAL_W * SCALE, LOGICAL_H * SCALE),
                ..default()
            }),
            ..default()
        }))
        .insert_resource(probe)
        .insert_resource(PendingAudio(sources))
        .add_systems(Startup, setup)
        .add_systems(Update, drive)
        .run()
}

#[derive(Resource)]
struct PendingAudio(Vec<AudioSource>);

/// 640×400 測試圖：8 像素棋盤加 16 色條，放大後每個邏輯像素應是 2×2 的實心方塊。
fn test_pattern() -> Image {
    let mut image = Image::new_fill(
        Extent3d { width: LOGICAL_W, height: LOGICAL_H, depth_or_array_layers: 1 },
        TextureDimension::D2,
        &[0, 0, 0, 255],
        TextureFormat::Rgba8UnormSrgb,
        RenderAssetUsages::MAIN_WORLD | RenderAssetUsages::RENDER_WORLD,
    );
    for y in 0..LOGICAL_H {
        for x in 0..LOGICAL_W {
            let rgba = if y < 40 {
                let i = (x * 16 / LOGICAL_W) as u8;
                [i * 17, 255 - i * 17, (i % 4) * 85, 255]
            } else if (x / 8 + y / 8) % 2 == 0 {
                [32, 40, 96, 255]
            } else {
                [16, 20, 48, 255]
            };
            image.pixel_bytes_mut(UVec3::new(x, y, 0)).unwrap().copy_from_slice(&rgba);
        }
    }
    image
}

fn setup(
    mut commands: Commands,
    mut images: ResMut<Assets<Image>>,
    mut fonts: ResMut<Assets<Font>>,
    mut audio_assets: ResMut<Assets<AudioSource>>,
    mut pending: ResMut<PendingAudio>,
    mut probe: ResMut<Probe>,
) {
    commands.spawn(Camera2d);
    commands.spawn((Sprite::from_image(images.add(test_pattern())), Transform::from_scale(Vec3::splat(SCALE as f32))));

    if let Some(path) = probe.font_path.clone() {
        match std::fs::read(&path) {
            Ok(bytes) => probe.font_handle = Some(fonts.add(Font::from_bytes(bytes))),
            Err(e) => error!("讀字型失敗 {path:?}: {e}"),
        }
    }
    let family = probe.family.clone();
    for (label, font, y) in [
        ("family", FontSource::family(family), 120.0),
        ("handle", probe.font_handle.clone().map(FontSource::from).unwrap_or_default(), -40.0),
    ] {
        commands.spawn((
            Text2d::new(PROBE_TEXT),
            TextFont { font, font_size: 36.0.into(), ..default() },
            TextColor(Color::WHITE),
            Transform::from_xyz(0.0, y, 1.0),
            ProbeText(label),
        ));
    }
    for source in pending.0.drain(..) {
        commands.spawn((AudioPlayer::new(audio_assets.add(source)), PlaybackSettings::ONCE));
    }
}

// Bevy 系統的參數即依賴注入；拆成多個系統只會讓探針流程更難讀。
#[allow(clippy::too_many_arguments)]
fn drive(
    mut commands: Commands,
    mut probe: ResMut<Probe>,
    texts: Query<(&ProbeText, &TextLayoutInfo)>,
    sinks: Query<&AudioSink>,
    players: Query<(), With<AudioPlayer>>,
    adapter: Option<Res<RenderAdapterInfo>>,
    windows: Query<&Window>,
    mut exit: MessageWriter<AppExit>,
) {
    probe.frame += 1;
    if probe.frame == SHOT_FRAME {
        let path = probe.shot.clone();
        commands.spawn(Screenshot::primary_window()).observe(save_to_disk(path)).observe(
            |_: On<ScreenshotCaptured>, mut probe: ResMut<Probe>| {
                probe.captured = true;
            },
        );
    }
    let timed_out = probe.frame >= GIVE_UP_FRAME;
    if !probe.captured && !timed_out {
        return;
    }
    let text: Vec<Value> = texts
        .iter()
        .map(|(label, layout)| {
            json!({
                "source": label.0, "glyphs": layout.glyphs.len(),
                "chars": PROBE_TEXT.chars().filter(|c| !c.is_whitespace()).count(),
                "lines": layout.glyphs.iter().map(|g| g.line_index).max().map_or(0, |m| m + 1),
                "size": [layout.size.x, layout.size.y],
            })
        })
        .collect();
    let window = windows.iter().next().map(|w| json!([w.physical_width(), w.physical_height(), w.scale_factor()]));
    let adapter = adapter.map(|a| {
        json!({ "name": a.name, "backend": format!("{:?}", a.backend), "driver": a.driver, "driver_info": a.driver_info })
    });
    let report = json!({
        "schema": "fd2-rust-r0-probe/1",
        "status": if probe.captured { "captured" } else { "timeout" },
        "frames": probe.frame,
        "bevy": "0.20.1",
        "adapter": adapter,
        "window": window,
        "logical": [LOGICAL_W, LOGICAL_H], "scale": SCALE,
        "font": { "path": probe.font_path, "family": probe.family, "loaded": probe.font_handle.is_some() },
        "text": text,
        "audio": { "decoded": probe.audio, "players": players.iter().count(), "sinks": sinks.iter().count() },
        "shot": probe.shot,
    });
    std::fs::write(&probe.report, serde_json::to_vec_pretty(&report).unwrap()).expect("寫收據");
    exit.write(if probe.captured { AppExit::Success } else { AppExit::from_code(2) });
}
