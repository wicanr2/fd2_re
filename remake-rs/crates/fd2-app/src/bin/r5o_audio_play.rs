//! R5 階段 4 r5o：Bevy 播放驗證（docs/goal/121）。容器內沒有喇叭（ALSA null 裝置），只驗證：
//!
//! 1. 依 `AudioVoice` 狀態建立、切換、停止 `AudioSink`（BGM 循環、切曲、停止、停止後重播同一首；音效一次性播放）；
//! 2. 一次性音效的 sink 播完後 `playing` 被設回 `false`，腳本照 `fd2-game` 的 `stepSFXVoices` 回收；
//! 3. 交給 sink 的樣本（`Fd2Sound::decoder()` 的輸出乘回 32768）與 `decode_pcm` 的結果逐樣本相同。
//!
//! 人耳聽辨不在範圍內。用法：`r5o_audio_play --assets <remake/assets> --sfx <pack/sfx> --report <json>`。

use std::path::PathBuf;
use std::sync::Arc;
use std::time::Duration;

use bevy::app::ScheduleRunnerPlugin;
use bevy::audio::{AudioPlayer, AudioPlugin, AudioSink, AudioSinkPlayback, Decodable};
use bevy::prelude::*;
use fd2_app::audio::{self, Fd2AudioPlugin, Fd2AudioState, Fd2Sound, GameAudioHost};
use fd2_game::game::{AudioVoice, VoiceSlot};
use serde_json::{Value, json};
use sha2::{Digest, Sha256};

const GIVE_UP_FRAME: u32 = 1800;

#[derive(Default)]
struct Script {
    bgm: Option<AudioVoice>,
    sfx: Vec<AudioVoice>,
    frame: u32,
    phase: u32,
    phase_frame: u32,
    sounds: Vec<(String, Vec<u8>)>,
    serial: u64,
    log: Vec<Value>,
    report: PathBuf,
    sample_checks: Vec<Value>,
}

impl GameAudioHost for Script {
    fn audio_voices(&mut self) -> Vec<(VoiceSlot, &mut AudioVoice)> {
        let mut v: Vec<(VoiceSlot, &mut AudioVoice)> = self.bgm.iter_mut().map(|v| (VoiceSlot::Bgm, v)).collect();
        v.extend(self.sfx.iter_mut().map(|v| (VoiceSlot::Sfx, v)));
        v
    }
}

impl Script {
    fn voice(&mut self, index: usize, looping: bool) -> AudioVoice {
        self.serial += 1;
        let (source, encoded) = self.sounds[index].clone();
        AudioVoice { source, playing: true, encoded, looping, serial: self.serial, ..Default::default() }
    }
}

fn sha_i16(samples: impl Iterator<Item = i16>) -> (usize, String) {
    let mut h = Sha256::new();
    let mut n = 0;
    for s in samples {
        h.update(s.to_le_bytes());
        n += 1;
    }
    (n, h.finalize().iter().map(|b| format!("{b:02x}")).collect())
}

fn main() -> AppExit {
    let args: Vec<String> = std::env::args().collect();
    let arg = |key: &str| PathBuf::from(&args[args.iter().position(|a| a == key).expect(key) + 1]);
    let (assets, sfx, report) = (arg("--assets"), arg("--sfx"), arg("--report"));
    let files = [
        ("bgm/fm/FDMUS_018", assets.join("music_fm/FDMUS_018.ogg")),
        ("bgm/fm/FDMUS_004", assets.join("music_fm/FDMUS_004.ogg")),
        ("sfx/FDOTHER_077/sample_002", sfx.join("FDOTHER_077/sample_002.ogg")),
        ("sfx/FDOTHER_077/sample_001", sfx.join("FDOTHER_077/sample_001.ogg")),
        ("wav/battle_48_00", assets.join("sfx/battle_48_00.wav")),
    ];
    let mut script = Script { report, ..Default::default() };
    for (id, path) in files {
        let bytes = std::fs::read(&path).unwrap_or_else(|e| panic!("讀 {path:?}: {e}"));
        // 交給 sink 的樣本與整檔解碼逐樣本相同。
        let (want_n, want) = sha_i16(audio::decode_pcm(&bytes).expect("decode").into_iter());
        let source = Fd2Sound { encoded: Arc::from(bytes.as_slice()) }.decoder();
        let mut exact = true;
        let (got_n, got) = sha_i16(source.map(|f| {
            let s = f * 32768.0;
            exact &= s.fract() == 0.0 && (-32768.0..=32767.0).contains(&s);
            s as i16
        }));
        script.sample_checks.push(json!({
            "id": id, "decoded_samples": want_n, "decoded_sha256": want, "sink_samples": got_n,
            "sink_sha256": got, "exact_i16": exact, "equal": exact && want == got && want_n == got_n,
        }));
        script.sounds.push((id.to_string(), bytes));
    }
    App::new()
        .add_plugins((
            MinimalPlugins.set(ScheduleRunnerPlugin::run_loop(Duration::from_secs_f64(1.0 / 60.0))),
            AssetPlugin::default(),
            AudioPlugin::default(),
            Fd2AudioPlugin::<Script>::default(),
        ))
        .insert_non_send(script)
        .add_systems(Update, drive)
        .add_systems(Last, observe)
        .run()
}

/// 腳本：每個階段改一次播放請求，條件成立才進下一階段。
fn drive(mut script: NonSendMut<Script>, players: Query<(), With<AudioPlayer<Fd2Sound>>>, sinks: Query<&AudioSink>) {
    script.frame += 1;
    script.phase_frame += 1;
    // fd2-game 的 stepSFXVoices。
    script.sfx.retain(|v| v.is_playing());
    let sinks_now = sinks.iter().count();
    let advance = match script.phase {
        0 => {
            let v = script.voice(0, true);
            script.bgm = Some(v);
            true
        }
        // 等 BGM 的 sink 建立並開始消耗樣本。
        1 => sinks_now == 1 && script.phase_frame >= 20,
        2 => {
            let v = script.voice(1, true);
            script.bgm = Some(v);
            true
        }
        3 => sinks_now == 1 && script.phase_frame >= 20,
        4 => {
            for i in [2, 3, 4] {
                let v = script.voice(i, false);
                script.sfx.push(v);
            }
            true
        }
        // 等三個音效都回報播完並被回收。
        5 => script.sfx.is_empty() && script.phase_frame >= 2,
        6 => {
            script.bgm = None;
            true
        }
        7 => players.iter().count() == 0 && script.phase_frame >= 5,
        8 => {
            let v = script.voice(1, true);
            script.bgm = Some(v);
            true
        }
        9 => sinks_now == 1 && script.phase_frame >= 20,
        10 => {
            script.bgm = None;
            true
        }
        11 => players.iter().count() == 0 && script.phase_frame >= 5,
        _ => false,
    };
    if advance {
        script.phase += 1;
        script.phase_frame = 0;
    }
}

fn observe(
    mut script: NonSendMut<Script>,
    state: Res<Fd2AudioState>,
    players: Query<(Entity, &AudioPlayer<Fd2Sound>)>,
    sinks: Query<&AudioSink>,
    mut exit: MessageWriter<AppExit>,
) {
    let last = &state.last;
    let changed = !last.started.is_empty() || !last.stopped.is_empty() || !last.finished.is_empty();
    if changed || script.frame.is_multiple_of(60) {
        let entities: Vec<Value> = players
            .iter()
            .map(|(e, _)| {
                let sink = sinks.get(e).ok();
                json!({
                    "entity": e.to_bits(), "sink": sink.is_some(),
                    "position_ms": sink.map(|s| s.position().as_millis() as u64),
                    "empty": sink.map(|s| s.empty()),
                })
            })
            .collect();
        let entry = json!({
            "frame": script.frame, "phase": script.phase,
            "started": last.started.iter().map(|(slot, s)| json!([format!("{slot:?}"), s])).collect::<Vec<_>>(),
            "stopped": last.stopped, "finished": last.finished, "rejected": last.rejected.len(),
            "bgm_serial": script.bgm.as_ref().map(|v| v.serial),
            "sfx_serials": script.sfx.iter().map(|v| v.serial).collect::<Vec<_>>(),
            "players": entities,
        });
        script.log.push(entry);
    }
    let done = script.phase >= 12;
    if !done && script.frame < GIVE_UP_FRAME {
        return;
    }
    let samples_ok = script.sample_checks.iter().all(|c| c["equal"] == true);
    let status = if done && samples_ok { "passed" } else { "failed" };
    let report = json!({
        "schema": "fd2-r5o-audio-play/1", "status": status, "frames": script.frame, "phase": script.phase,
        "sample_checks": script.sample_checks, "log": script.log,
    });
    std::fs::write(&script.report, serde_json::to_vec_pretty(&report).unwrap()).expect("寫報告");
    exit.write(if status == "passed" { AppExit::Success } else { AppExit::from_code(1) });
}
