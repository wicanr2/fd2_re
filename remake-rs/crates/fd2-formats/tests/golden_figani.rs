//! 與凍結 Go 版 `remake/cmd/fd2-golden-figani` 的黃金差分。
//!
//! 由 `tools/rust_golden_figani.sh` 產生黃金 JSON 並設定 `FD2_GOLDEN_FIGANI`；原版檔案與分離素材包
//! 以 `FD2_FLAME2_DIR`、`FD2_TEST_ORIGINAL_PACK`、`FD2_REMAKE_ASSETS` 指定，先核對 SHA-256 再使用。
//! 設定 `FD2_REQUIRE_GOLDEN=1` 時缺檔即失敗，避免把「沒跑到」當成通過。
//!
//! 比對規則：figani 自己產生的結果與錯誤文字逐欄、逐字相同。錯誤包了 JSON、PNG 或檔案系統的下層
//! 錯誤時（`Cause`），刻意不照做 Go 標準函式庫的下層文字，只要求兩邊都拒收，且 Go 的錯誤文字依序
//! 含有 figani 自己的各段前綴（見 `err_value`）。

use fd2_formats::figani::*;
use serde_json::{Value, json};
use sha2::{Digest, Sha256};
use std::path::{Path, PathBuf};

/// 已知差異：刻意不照做。格式為（案例 kind，一行理由）。這些 kind 的案例只要兩邊都拒收即視為相符。
const KNOWN_DIFFERENCES: &[(&str, &str)] = &[(
    "real_separated_source",
    "刻意不照做：不支援的來源名稱以 Rust `{:?}` 引號，控制字元（如 NUL）跳脫寫法與 Go `%q` 不同",
)];

const FOREIGN: &str = "\u{1}fragments:";

fn digest(b: &[u8]) -> String {
    Sha256::digest(b).iter().map(|x| format!("{x:02x}")).collect()
}

fn hex(b: &[u8]) -> String {
    b.iter().map(|x| format!("{x:02x}")).collect()
}

fn unhex(s: &str) -> Vec<u8> {
    (0..s.len()).step_by(2).map(|i| u8::from_str_radix(&s[i..i + 2], 16).unwrap()).collect()
}

struct Ctx {
    figani: String,
    fdother: String,
    pack: String,
    work: PathBuf,
}

impl Ctx {
    fn text(&self, s: String) -> String {
        s.replace(&self.work.to_string_lossy().into_owned(), "$ROOT").replace(&self.pack, "$PACK")
    }

    fn expand(&self, root: &str) -> String {
        root.replace("$PACK", &self.pack).replace("$ROOT", &self.work.to_string_lossy())
    }

    /// figani 自己的錯誤逐字比對；含下層錯誤時改成「片段」比對（見檔頭說明）。
    fn err_value(&self, e: &FiganiError) -> Value {
        let mut fragments = Vec::new();
        if fragments_of(e, &mut fragments) {
            Value::String(self.text(e.to_string()))
        } else {
            let fragments: Vec<String> = fragments.into_iter().map(|f| self.text(f)).collect();
            Value::String(format!("{FOREIGN}{}", serde_json::to_string(&fragments).unwrap()))
        }
    }

    fn err_obj(&self, e: &FiganiError) -> Value {
        json!({ "err": self.err_value(e) })
    }
}

/// 收集 figani 自己的文字片段；回傳 true 代表整段都是 figani 自己的文字（可逐字比對）。
fn fragments_of(e: &FiganiError, out: &mut Vec<String>) -> bool {
    match e {
        FiganiError::Wrapped { prefix, .. } => {
            out.push(prefix.clone());
            false
        }
        FiganiError::Cause(_) => false,
        FiganiError::FallbackUnavailable { resource, primary, status } => {
            out.push(format!("figani: resource {resource} unavailable ("));
            let a = fragments_of(primary, out);
            out.push("); status: ".to_string());
            let b = fragments_of(status, out);
            a && b
        }
        other => {
            out.push(other.to_string());
            true
        }
    }
}

fn foreign_matches(fragments: &str, go: &str) -> bool {
    let fragments: Vec<String> = serde_json::from_str(fragments).unwrap();
    let mut at = 0;
    for (i, f) in fragments.iter().enumerate() {
        match go[at..].find(f.as_str()) {
            Some(0) => at += f.len(),
            Some(pos) if i > 0 => at += pos + f.len(),
            _ => return false,
        }
    }
    true
}

fn first_diff(path: &str, want: &Value, got: &Value) -> Option<String> {
    match (want, got) {
        (Value::String(go), Value::String(rust)) if rust.starts_with(FOREIGN) => {
            if foreign_matches(&rust[FOREIGN.len()..], go) {
                None
            } else {
                Some(format!("{path}: go={go:?} rust 片段={}", &rust[FOREIGN.len()..]))
            }
        }
        (Value::Object(a), Value::Object(b)) => {
            for key in a.keys().chain(b.keys()) {
                if a.contains_key(key) != b.contains_key(key) {
                    return Some(format!(
                        "{path}.{key}: present go={} rust={}",
                        a.contains_key(key),
                        b.contains_key(key)
                    ));
                }
                if let Some(d) = first_diff(&format!("{path}.{key}"), &a[key], &b[key]) {
                    return Some(d);
                }
            }
            None
        }
        (Value::Array(a), Value::Array(b)) if a.len() == b.len() => {
            a.iter().zip(b).enumerate().find_map(|(i, (x, y))| first_diff(&format!("{path}[{i}]"), x, y))
        }
        _ if want == got => None,
        _ => Some(format!("{path}: go={want} rust={got}")),
    }
}

/// 只比接受／拒收：兩邊結果裡的錯誤位置（含 `err` 鍵的物件）一致即可。
fn shape(v: &Value) -> Value {
    match v {
        Value::Object(map) if map.contains_key("err") => json!("<err>"),
        Value::Object(map) => Value::Object(map.iter().map(|(k, v)| (k.clone(), shape(v))).collect()),
        Value::Array(items) => Value::Array(items.iter().map(shape).collect()),
        other => other.clone(),
    }
}

// ---- 輸入轉換 ----

fn i(v: &Value) -> i64 {
    v.as_i64().unwrap_or_else(|| panic!("not int: {v}"))
}

fn b(v: &Value) -> bool {
    v.as_bool().unwrap()
}

fn ia<const N: usize>(v: &Value) -> [i64; N] {
    let list: Vec<i64> = v.as_array().unwrap().iter().map(i).collect();
    list.try_into().unwrap()
}

fn ba<const N: usize>(v: &Value) -> [u8; N] {
    let list: Vec<u8> = v.as_array().unwrap().iter().map(|x| i(x) as u8).collect();
    list.try_into().unwrap()
}

fn bank(count: usize, delay: i64) -> Animation {
    let frames = (0..count)
        .map(|k| Frame { width: 1, height: 1, pixels: vec![k as u8], mask: vec![1], delay, ..Frame::default() })
        .collect();
    Animation { frames, header_byte2: count as u8, ..Animation::default() }
}

// ---- 摘要（形狀與 Go 版 summarize.go 相同） ----

fn frame_json(f: &Frame) -> Value {
    json!({
        "x": f.x, "y": f.y, "w": f.width, "h": f.height, "delay": f.delay,
        "r4": f.raw_byte4, "r5": f.raw_byte5, "r7": f.raw_byte7,
        "pix": digest(&f.pixels), "mask": digest(&f.mask), "npix": f.pixels.len(), "nmask": f.mask.len(),
    })
}

fn anim_json(a: &Animation) -> Value {
    json!({ "h1": a.header_byte1, "h2": a.header_byte2, "h4": a.header_byte4,
        "frames": a.frames.iter().map(frame_json).collect::<Vec<_>>() })
}

fn c0_json(s: &NativeCommand0PresentationSchedule) -> Value {
    json!({ "effect": s.effect_resource, "sound": s.sound_resource, "sound_index": s.sound_index, "frames": s.frames,
        "stages": s.damage_stages, "shift": s.side_x_shift, "flags": s.layer_flags, "x": s.x_offsets, "y": s.y_offsets })
}

fn c1_json(s: &NativeCommand1PresentationSchedule) -> Value {
    json!({ "effect": s.effect_resource, "shift": s.side_x_shift, "anchor": s.anchor_x, "x": s.x_offsets, "y": s.y_offsets })
}

fn c2_json(s: &NativeCommand2PresentationSchedule) -> Value {
    json!({ "effect": s.effect_resource, "sound": s.sound_resource, "indices": s.sound_indices })
}

fn c2_state(s: &NativeCommand2HelperState) -> Value {
    json!([s.frame, s.repeat])
}

fn c2_frames(frames: &[NativeCommand2HelperFrame]) -> Value {
    frames
        .iter()
        .map(|f| {
            json!([f.effect_frame, f.next.frame, f.next.repeat, f.sample1, f.sample2, f.numeric_marker, f.hp_stage])
        })
        .collect()
}

fn c3_sched_json(s: &NativeCommand3PresentationSchedule) -> Value {
    json!({ "effect": s.effect_resource, "sound": s.sound_resource, "x": s.x_offsets, "y": s.y_offsets, "bases": s.frame_bases })
}

fn c3_sched(v: &Value) -> NativeCommand3PresentationSchedule {
    NativeCommand3PresentationSchedule {
        effect_resource: i(&v["effect"]),
        sound_resource: i(&v["sound"]),
        x_offsets: ia(&v["x"]),
        y_offsets: ia(&v["y"]),
        frame_bases: ia(&v["bases"]),
    }
}

fn c3_state_json(s: &NativeCommand3State) -> Value {
    json!({ "c": s.counters, "p": s.positions, "np": s.next_position, "stop": s.stop, "t": s.toggle })
}

fn c3_state(v: &Value) -> NativeCommand3State {
    NativeCommand3State {
        counters: ia(&v["c"]),
        positions: ia(&v["p"]),
        next_position: i(&v["np"]),
        stop: b(&v["stop"]),
        toggle: i(&v["t"]),
    }
}

fn c3_frame(f: &NativeCommand3Frame) -> Value {
    json!({ "layers": f.layers.iter().map(|l| json!([l.channel, l.frame, l.x, l.y])).collect::<Vec<_>>(),
        "num": f.numeric_marker, "s1": f.play_sub1, "s2": f.play_sub2, "hp": f.hp_stage, "next": c3_state_json(&f.next) })
}

fn c5_sched_json(s: &NativeCommand5PresentationSchedule) -> Value {
    json!({ "cmd": s.command_id, "effect": s.effect_resource, "sound": s.sound_resource, "indices": s.sound_indices,
        "x": s.x_offsets, "ef": s.effect_frames, "ps": s.phase_stride, "ac": s.active_counters, "ff": s.front_frames,
        "tf": s.target_frames, "tl": s.tail_frames, "mc": s.marker_counter, "rc": s.reset_counter, "every": s.sample_every_channel })
}

fn c5_sched(v: &Value) -> NativeCommand5PresentationSchedule {
    NativeCommand5PresentationSchedule {
        command_id: i(&v["cmd"]),
        effect_resource: i(&v["effect"]),
        sound_resource: i(&v["sound"]),
        sound_indices: ia(&v["indices"]),
        x_offsets: ia(&v["x"]),
        effect_frames: i(&v["ef"]),
        phase_stride: i(&v["ps"]),
        active_counters: i(&v["ac"]),
        front_frames: i(&v["ff"]),
        target_frames: i(&v["tf"]),
        tail_frames: i(&v["tl"]),
        marker_counter: i(&v["mc"]),
        reset_counter: i(&v["rc"]),
        sample_every_channel: b(&v["every"]),
    }
}

fn c5_state_json(s: &NativeCommand5State) -> Value {
    json!({ "c": s.counters, "p": s.positions, "ph": s.phases, "np": s.next_position, "stop": s.stop, "rng": s.rng })
}

fn c5_state(v: &Value) -> NativeCommand5State {
    NativeCommand5State {
        counters: ia(&v["c"]),
        positions: ia(&v["p"]),
        phases: ia(&v["ph"]),
        next_position: i(&v["np"]),
        stop: b(&v["stop"]),
        rng: i(&v["rng"]) as u16,
    }
}

fn c5_frame(f: &NativeCommand5Frame) -> Value {
    json!({ "layers": f.layers.iter().map(|l| json!([l.channel, l.frame, l.x])).collect::<Vec<_>>(),
        "next": c5_state_json(&f.next), "num": f.numeric_marker, "hp": f.hp_stage, "pri": f.play_primary, "sec": f.play_secondary })
}

fn c6_sched_json(s: &NativeCommand6PresentationSchedule) -> Value {
    json!({ "effect": s.effect_resource, "sound": s.sound_resource, "indices": s.sound_indices, "base": s.base_byte,
        "dw": s.dword_table, "bt": s.byte_table })
}

fn c6_sched(v: &Value) -> NativeCommand6PresentationSchedule {
    NativeCommand6PresentationSchedule {
        effect_resource: i(&v["effect"]),
        sound_resource: i(&v["sound"]),
        sound_indices: ia(&v["indices"]),
        base_byte: i(&v["base"]) as u8,
        dword_table: ia(&v["dw"]),
        byte_table: ba(&v["bt"]),
    }
}

fn c6_points_json(points: &[NativeCommand6Point; 5]) -> Value {
    points.iter().map(|p| json!([p.x, p.y])).collect()
}

fn c6_points(v: &Value) -> [NativeCommand6Point; 5] {
    let list: Vec<NativeCommand6Point> =
        v.as_array().unwrap().iter().map(|p| NativeCommand6Point { x: i(&p[0]), y: i(&p[1]) }).collect();
    list.try_into().unwrap()
}

fn c6_state_json(s: &NativeCommand6TargetState) -> Value {
    json!({ "c": s.counters, "s": s.secondary })
}

fn c6_layers(layers: &[NativeCommand6Layer]) -> Value {
    layers.iter().map(|l| json!([l.mode, l.channel, l.frame, l.x, l.y, l.secondary])).collect()
}

fn c6_frame(f: &NativeCommand6TargetFrame) -> Value {
    json!({ "m4": c6_layers(&f.mode4), "m5": c6_layers(&f.mode5), "next": c6_state_json(&f.next),
        "num": f.numeric_marker, "hp": f.hp_stage })
}

fn c6_orbit(f: &NativeCommand6OrbitFrame) -> Value {
    json!({ "first": c6_layers(&f.first), "second": c6_layers(&f.second), "nr": f.next_radius, "dt": f.draw_target })
}

fn c7_sched_json(s: &NativeCommand7PresentationSchedule) -> Value {
    json!({ "effect": s.effect_resource, "sound": s.sound_resource, "indices": s.sound_indices, "x": s.x_offsets })
}

fn c7_sched(v: &Value) -> NativeCommand7PresentationSchedule {
    NativeCommand7PresentationSchedule {
        effect_resource: i(&v["effect"]),
        sound_resource: i(&v["sound"]),
        sound_indices: ia(&v["indices"]),
        x_offsets: ia(&v["x"]),
    }
}

fn c7_state_json(s: &NativeCommand7State) -> Value {
    json!({ "c": s.counters, "p": s.positions, "np": s.next_position, "stop": s.stop, "t": s.toggle })
}

fn c7_state(v: &Value) -> NativeCommand7State {
    NativeCommand7State {
        counters: ia(&v["c"]),
        positions: ia(&v["p"]),
        next_position: i(&v["np"]),
        stop: b(&v["stop"]),
        toggle: i(&v["t"]) as u8,
    }
}

fn c7_frame(f: &NativeCommand7Frame) -> Value {
    json!({ "layers": f.layers.iter().map(|l| json!([l.channel, l.frame, l.x])).collect::<Vec<_>>(),
        "next": c7_state_json(&f.next), "num": f.numeric_marker, "hp": f.hp_stage, "a": f.play_handle_a, "b": f.play_handle_b })
}

fn c8_sched_json(s: &NativeCommand8PresentationSchedule) -> Value {
    json!({ "effect": s.effect_resource, "sound": s.sound_resource, "bases": s.frame_bases })
}

fn c8_frame(f: &NativeCommand8Frame) -> Value {
    json!({ "layers": f.layers.iter().map(|l| json!([l.channel, l.frame])).collect::<Vec<_>>(),
        "num": f.numeric_marker, "s1": f.play_sub1, "s2": f.play_sub2, "hp": f.hp_stage, "next": f.next.counters })
}

fn c9_sched_json(s: &NativeCommand9AISchedule) -> Value {
    json!({ "effect": s.effect_resource, "sound": s.sound_resource, "ff": s.front_frames, "tf": s.target_frames,
        "tl": s.tail_frames, "ds": s.damage_stages, "in": s.actor_slide_in, "out": s.actor_slide_out, "hold": s.slide_in_hold_ms })
}

fn c9_sched(v: &Value) -> NativeCommand9AISchedule {
    NativeCommand9AISchedule {
        effect_resource: i(&v["effect"]),
        sound_resource: i(&v["sound"]),
        front_frames: i(&v["ff"]),
        target_frames: i(&v["tf"]),
        tail_frames: i(&v["tl"]),
        damage_stages: i(&v["ds"]),
        actor_slide_in: i(&v["in"]),
        actor_slide_out: i(&v["out"]),
        slide_in_hold_ms: i(&v["hold"]),
    }
}

fn c9_frame(f: &NativeCommand9AIFrame) -> Value {
    json!({ "ef": f.effect_frames, "num": f.numeric_marker, "hp": f.hp_stage, "s1": f.play_sub1, "s2": f.play_sub2,
        "next": [f.next.counter, f.next.toggle] })
}

fn compound_json(s: &NativeCompoundPresentationSchedule) -> Value {
    json!({ "cmd": s.command_id, "effect": s.effect_resource, "sound": s.sound_resource, "asf": s.actor_slide_frames,
        "asx": s.actor_slide_step_x, "esf": s.effect_slide_frames, "esx": s.effect_slide_step_x, "phold": s.prelude_hold_ticks,
        "mdel": s.main_delay_ticks, "ovl": s.overlay_first_frame, "mfc": s.main_frame_count, "ms2": s.mixer_sample2_frame,
        "s1": s.sample1_frame, "tail": s.tail_enabled, "tf": s.tail_frames, "rgb": s.palette_rgb })
}

fn cues(c: &[NativeCommandFrameCue]) -> Value {
    c.iter().map(|c| json!([c.frame, c.sample])).collect()
}

fn strike_json(s: &NativeCommandDerivedStrikeSchedule) -> Value {
    json!({ "cmd": s.command_id, "ts": s.target_start, "aif": s.actor_impact_frame, "tif": s.target_impact_frame,
        "tifs": s.target_impact_frames, "den": s.damage_denominator, "audio": s.audio_resource, "as": s.actor_sample,
        "acues": cues(&s.actor_cues), "tcues": cues(&s.target_cues), "tsamp": s.target_sample, "delays": s.frame_delays,
        "utb": s.uses_target_base, "pm": s.prelude_mode, "ubg": s.uses_bg_transition, "shake": s.shake_offsets })
}

fn c24_json(s: &NativeCommand24Schedule) -> Value {
    json!({ "ts": s.target_start, "aif": s.actor_impact_frame, "tif": s.target_impact_frame, "audio": s.audio_resource,
        "as": s.actor_sample, "tsamp": s.target_sample, "shake": s.shake_offsets })
}

impl Ctx {
    fn res<T>(&self, r: Result<T>, f: impl FnOnce(&T) -> Value) -> Value {
        match r {
            Ok(v) => f(&v),
            Err(e) => self.err_obj(&e),
        }
    }

    fn seq<T, S>(
        &self,
        r: std::result::Result<(T, S), StateError<S>>,
        f: impl FnOnce(&T) -> Value,
        g: impl Fn(&S) -> Value,
    ) -> Value {
        match r {
            Ok((frames, state)) => json!({ "frames": f(&frames), "state": g(&state) }),
            Err(e) => json!({ "err": self.err_value(&e.error), "state": g(&e.state) }),
        }
    }

    fn builders(&self, a: Option<&Animation>) -> Value {
        let sides = [0u8, 1, 2];
        let pick = |f: &dyn Fn(u8) -> Value| -> Value { sides.iter().map(|&s| f(s)).collect() };
        let strike: Vec<Value> = [23, 24, 27, 28, 29, 30, 31, 32]
            .iter()
            .map(|&id| self.res(build_native_command_derived_strike_schedule(a, id), strike_json))
            .collect();
        let compound: Vec<Value> =
            (31..=36).map(|id| self.res(build_native_compound_presentation_schedule(id, a), compound_json)).collect();
        json!({
            "c0": pick(&|s| self.res(build_native_command0_presentation_schedule(s, a), c0_json)),
            "c1": pick(&|s| self.res(build_native_command1_presentation_schedule(s, a), c1_json)),
            "c2": pick(&|s| self.res(build_native_command2_presentation_schedule(s, a), c2_json)),
            "c2front": pick(&|s| match build_native_command2_front_sequence(s, a) {
                Err(e) => self.err_obj(&e),
                Ok((frames, end)) => json!({ "frames": c2_frames(&frames), "state": c2_state(&end),
                    "tail": self.res(build_native_command2_tail_sequence(s, end.repeat, a), |t| c2_frames(t)) }),
            }),
            "c3": pick(&|s| self.res(build_native_command3_presentation_schedule(s, a), c3_sched_json)),
            "c4": pick(&|s| self.res(build_native_command4_presentation_schedule(s, a), c5_sched_json)),
            "c5": pick(&|s| self.res(build_native_command5_presentation_schedule(s, a), c5_sched_json)),
            "c6": pick(&|s| self.res(build_native_command6_presentation_schedule(s, a), c6_sched_json)),
            "c7": pick(&|s| self.res(build_native_command7_presentation_schedule(s, a), c7_sched_json)),
            "c8": pick(&|s| self.res(build_native_command8_presentation_schedule(s, a), c8_sched_json)),
            "c9": pick(&|s| self.res(build_native_command9_ai_schedule(s, a), c9_sched_json)),
            "c24": self.res(build_native_command24_schedule(a), c24_json),
            "strike": strike,
            "compound": compound,
        })
    }

    fn display(&self, delays: &[i64], ticks: i64, limit: usize) -> Value {
        let mut s = match DisplayScheduler::new(delays, ticks) {
            Ok(s) => s,
            Err(e) => return self.err_obj(&e),
        };
        let mut trace = Vec::new();
        for _ in 0..limit {
            match s.step() {
                Err(e) => {
                    trace.push(json!([self.err_value(&e)]));
                    break;
                }
                Ok((frame, presented, done)) => {
                    trace.push(json!([frame, presented, done]));
                    if done && !presented {
                        break;
                    }
                }
            }
        }
        let starts: Vec<Value> = (-1..=delays.len() as i64)
            .map(|f| match s.frame_start(f) {
                Some(start) => json!([start, true]),
                None => json!([0, false]),
            })
            .collect();
        json!({ "body": s.body_ticks(), "current": s.current_frame(), "trace": trace, "starts": starts })
    }

    fn schedulers(&self, a: &Animation, ticks: i64, steps: usize) -> Value {
        let mut native = NativeScheduler::default();
        let mut trace = Vec::new();
        for k in 0..steps {
            match native.step(Some(a), k != 0) {
                Err(e) => {
                    trace.push(json!([self.err_value(&e)]));
                    break;
                }
                Ok((index, rendered)) => trace.push(json!([index, rendered, native.frame_index, native.subframe])),
            }
        }
        let delays: Vec<i64> = a.frames.iter().map(|f| f.delay).collect();
        json!({ "native": trace, "display": self.display(&delays, ticks, 400) })
    }

    fn frame_blits(&self, f: &Frame) -> Value {
        let blit = |size: usize, draw: &dyn Fn(&mut [u8]) -> Result<()>| -> Value {
            let mut dst = vec![9u8; size];
            match draw(&mut dst) {
                Err(e) => json!({ "err": self.err_value(&e), "kept": dst.iter().all(|&x| x == 9) }),
                Ok(()) => json!({ "sha": digest(&dst), "changed": dst.iter().any(|&x| x != 9) }),
            }
        };
        json!([
            blit(320 * 200, &|d| f.blit_at(d, 320)),
            blit(640 * 200, &|d| f.blit_at_base(d, 640, 320)),
            blit(400 * 240, &|d| f.blit_translated(d, 400, -40, 20, None)),
            blit(400 * 240, &|d| f.blit_translated(d, 400, -40, 20, Some(33))),
        ])
    }

    fn animation_result(&self, a: &Animation) -> Value {
        json!({
            "anim": anim_json(a), "builders": self.builders(Some(a)),
            "sched": self.schedulers(a, 1 + (a.frames.len() % 3) as i64, 60),
            "blits": a.frames.iter().map(|f| self.frame_blits(f)).collect::<Vec<_>>(),
        })
    }

    fn walk(&self, input: &Value, run: &dyn Fn(u16, Resolver<'_>) -> std::result::Result<u16, RngWalkError>) -> Value {
        let spec: Vec<[i64; 3]> = input["spec"].as_array().unwrap().iter().map(ia).collect();
        let rng = i(&input["rng"]) as u16;
        let mut calls = Vec::new();
        let mut resolve = |index: i64, rng: u16| -> std::result::Result<(u16, bool), ResolveError> {
            calls.push(json!([index, rng]));
            let s = spec[index as usize % spec.len()];
            if s[2] != 0 {
                return Err("resolver failed".into());
            }
            let next = (0..s[0]).fold(rng, |r, _| fd2_formats::native_rng::native_rng_step(r));
            Ok((next, s[1] != 0))
        };
        let resolver: Resolver<'_> = if b(&input["nil"]) { None } else { Some(&mut resolve) };
        let (last, err) = match run(rng, resolver) {
            Ok(r) => (r, Value::Null),
            Err(e) => (e.state, self.err_value(&e.error)),
        };
        json!({ "rng": last, "err": err, "calls": calls })
    }

    fn table(&self, path: &str) -> Value {
        let table = match load_native_command_indexed_entry_table(path) {
            Ok(t) => t,
            Err(e) => return self.err_obj(&e),
        };
        let schedules: Vec<Value> = (-1..=10)
            .map(|id| match table.schedule(id) {
                Err(e) => self.err_obj(&e),
                Ok(e) => json!({
                    "id": e.command_id, "entry": e.entry, "modes": e.mode_returns, "init": e.init_channels,
                    "draw": e.draw_channels, "draw_modes": e.draw_modes, "shift": e.raw_side_zero_x_shift,
                    "offsets": e.offset_tables, "ranges": e.state_ranges, "rng": e.uses_rng,
                    "markers": e.sample_markers.iter().map(|m| json!([m.mode, m.channel, m.counter, m.counter_phase,
                        m.frame_base_nonzero, m.callee])).collect::<Vec<_>>(),
                }),
            })
            .collect();
        let s = &table.source;
        json!({ "schema": table.schema_version,
            "source": [s.file, s.size, s.md5, s.sha256, s.ida_version, s.address_space, s.entry_table, s.consumer],
            "entries": table.entries.len(), "schedules": schedules })
    }

    fn sep_load(&self, root: &str, source: &str, resource: i64) -> Value {
        match load_separated_archive_resource(&self.expand(root), source, resource) {
            Ok(a) => json!({ "anim": anim_json(&a) }),
            Err(e) => self.err_obj(&e),
        }
    }

    fn sep_fallback_source(&self, root: &str, resource: i64) -> Value {
        match load_separated_resource_with_zero_header_fallback_source(&self.expand(root), resource) {
            Ok((a, source)) => json!({ "anim": anim_json(&a), "source": source }),
            Err(e) => json!({ "err": self.err_value(&e), "source": -1 }),
        }
    }

    fn reset_work(&self) {
        let _ = std::fs::remove_dir_all(&self.work);
        std::fs::create_dir_all(&self.work).unwrap();
    }
}

fn c24_base(ctx: &Ctx) -> Animation {
    decode_resource(&ctx.figani, 98).unwrap()
}

fn compute(ctx: &Ctx, op: &str, input: &Value) -> Value {
    match op {
        "decode_resource" => {
            let path = if input["file"] == "figani" { &ctx.figani } else { &ctx.fdother };
            match decode_resource(path, i(&input["resource"])) {
                Ok(a) => ctx.animation_result(&a),
                Err(e) => ctx.err_obj(&e),
            }
        }
        "sep_load" => {
            ctx.sep_load(input["root"].as_str().unwrap(), input["source"].as_str().unwrap(), i(&input["resource"]))
        }
        "sep_fallback" => {
            let (root, resource) = (input["root"].as_str().unwrap(), i(&input["resource"]));
            let plain = match load_separated_resource_with_zero_header_fallback(&ctx.expand(root), resource) {
                Ok(_) => Value::Null,
                Err(e) => ctx.err_value(&e),
            };
            json!({ "source_result": ctx.sep_fallback_source(root, resource), "plain_err": plain })
        }
        "parse" => match parse(&unhex(input["hex"].as_str().unwrap())) {
            Ok(a) => ctx.animation_result(&a),
            Err(e) => ctx.err_obj(&e),
        },
        "native_sched" => {
            let delays: Vec<i64> = input["delays"].as_array().unwrap().iter().map(i).collect();
            let a = Animation {
                frames: delays.iter().map(|&delay| Frame { delay, ..Frame::default() }).collect(),
                ..Animation::default()
            };
            let animation = if b(&input["nil"]) { None } else { Some(&a) };
            let state = ia::<2>(&input["state"]);
            let mut s = NativeScheduler { frame_index: state[0], subframe: state[1] };
            let trace: Vec<Value> = input["advance"]
                .as_array()
                .unwrap()
                .iter()
                .map(|adv| match s.step(animation, b(adv)) {
                    Err(e) => json!([ctx.err_value(&e)]),
                    Ok((index, rendered)) => json!([index, rendered, s.frame_index, s.subframe]),
                })
                .collect();
            Value::Array(trace)
        }
        "display_sched" => {
            let delays: Vec<i64> = input["delays"].as_array().unwrap().iter().map(i).collect();
            ctx.display(&delays, i(&input["ticks"]), 300)
        }
        "blit" => {
            let f = Frame {
                x: i(&input["x"]),
                y: i(&input["y"]),
                width: i(&input["w"]),
                height: i(&input["h"]),
                pixels: unhex(input["pix"].as_str().unwrap()),
                mask: unhex(input["mask"].as_str().unwrap()),
                ..Frame::default()
            };
            let (len, stride, base) = (i(&input["len"]) as usize, i(&input["stride"]), i(&input["base"]));
            let (dx, dy) = (i(&input["dx"]), i(&input["dy"]));
            let fill = input["fill"].as_i64().map(|v| v as u8);
            let run = |draw: &dyn Fn(&mut [u8]) -> Result<()>| -> Value {
                let mut dst = vec![9u8; len];
                match draw(&mut dst) {
                    Err(e) => json!({ "err": ctx.err_value(&e), "kept": dst.iter().all(|&x| x == 9) }),
                    Ok(()) => json!({ "dst": hex(&dst) }),
                }
            };
            let mut out = json!({
                "at": run(&|d| f.blit_at(d, stride)),
                "base": run(&|d| f.blit_at_base(d, stride, base)),
            });
            if f.width < 1 << 20 && f.height < 1 << 20 {
                out["translated"] = run(&|d| f.blit_translated(d, stride, dx, dy, fill));
            }
            out
        }
        "builders_nil" => ctx.builders(None),
        "c0_grid" => {
            let mut grid = Vec::new();
            for step in -2..=29 {
                for layer in -2..=8 {
                    let (frame, ok) = native_command0_effect_frame(step, layer);
                    grid.push(json!([frame, ok, native_command0_impact(step, layer)]));
                }
            }
            let markers: Vec<bool> = (-2..=29).map(native_command0_marker_step).collect();
            json!({ "grid": grid, "markers": markers })
        }
        "c0_walk" => {
            let schedule = NativeCommand0PresentationSchedule { frames: i(&input["frames"]), ..Default::default() };
            let targets = i(&input["targets"]);
            ctx.walk(input, &|rng, r| walk_native_command0_rng(rng, schedule, targets, r))
        }
        "c1_grid" => {
            let counters: Vec<Value> = (-2..=32).map(|s| ctx.res(native_command1_counters(s), |c| json!(c))).collect();
            let markers: Vec<Value> =
                (-2..=32).map(|s| ctx.res(native_command1_target_markers(s), |m| json!([m.0, m.1]))).collect();
            let mut modes = Vec::new();
            for mode in 2..=6 {
                for slot in -2..=9 {
                    for counter in -2..=16 {
                        let (f, o, ok) = native_command1_mode_frame(mode, slot, counter);
                        modes.push(json!([f, o, ok]));
                    }
                }
            }
            json!({ "counters": counters, "markers": markers, "modes": modes })
        }
        "c2_target" => ctx.seq(
            build_native_command2_target_sequence()
                .map_err(|e| StateError { state: NativeCommand2HelperState::default(), error: e }),
            |f| c2_frames(f),
            c2_state,
        ),
        "c2_seq" => {
            let bank_in = &input["bank"];
            let delays: Vec<i64> = bank_in["delays"].as_array().unwrap().iter().map(i).collect();
            let mut a = bank(delays.len(), 0);
            for (frame, d) in a.frames.iter_mut().zip(&delays) {
                frame.delay = *d;
            }
            a.header_byte1 = i(&bank_in["h1"]) as u8;
            a.header_byte2 = i(&bank_in["h2"]) as u8;
            a.header_byte4 = i(&bank_in["h4"]) as u8;
            let a = if b(&input["nil"]) { None } else { Some(&a) };
            let (side, repeat) = (i(&input["side"]) as u8, i(&input["repeat"]) as u8);
            let st = ia::<2>(&input["state"]);
            let state = NativeCommand2HelperState { frame: st[0] as u8, repeat: st[1] as u8 };
            let front = ctx.seq(
                build_native_command2_front_sequence(side, a)
                    .map_err(|e| StateError { state: NativeCommand2HelperState::default(), error: e }),
                |f| c2_frames(f),
                c2_state,
            );
            json!({
                "front": front,
                "tail": ctx.res(build_native_command2_tail_sequence(side, repeat, a), |t| c2_frames(t)),
                "schedule": ctx.res(build_native_command2_presentation_schedule(side, a), c2_json),
                "transition": c2_frames(&build_native_command2_transition_sequence(state)),
            })
        }
        "c2_walk" => {
            let targets = i(&input["targets"]);
            ctx.walk(input, &|rng, r| walk_native_command2_rng(rng, targets, r))
        }
        "c3_plan" => {
            let (s, state, side) = (c3_sched(&input["schedule"]), c3_state(&input["state"]), i(&input["side"]) as u8);
            let frames = |l: &Vec<NativeCommand3Frame>| -> Value { l.iter().map(c3_frame).collect() };
            json!({
                "plan": ctx.res(plan_native_command3_draw_frame(state, s, side), c3_frame),
                "target": ctx.seq(build_native_command3_target_sequence(state, s, side), frames, c3_state_json),
                "transition": ctx.seq(build_native_command3_transition_sequence(state, s, side), |l| l.iter().map(|f| {
                    let mut j = c3_frame(&f.frame);
                    j["use_next"] = json!(f.use_next_target);
                    j["off"] = json!(f.target_offset_x);
                    j
                }).collect(), c3_state_json),
                "tail": ctx.seq(build_native_command3_tail_sequence(state, s, side), frames, c3_state_json),
            })
        }
        "c5_plan" => {
            let (s, state, side) = (c5_sched(&input["schedule"]), c5_state(&input["state"]), i(&input["side"]) as u8);
            let rng = i(&input["rng"]) as u16;
            let frames = |l: &Vec<NativeCommand5Frame>| -> Value { l.iter().map(c5_frame).collect() };
            let plan = plan_native_command5_draw_frame(state, s, side);
            let shake = match &plan {
                Ok(f) => c5_frame(&native_command5_marker_shake(f.clone())),
                Err(_) => Value::Null,
            };
            json!({
                "new_for_schedule": c5_state_json(&new_native_command5_state_for_schedule(rng, s)),
                "new": c5_state_json(&new_native_command5_state(rng)),
                "plan": ctx.res(plan, c5_frame), "shake": shake,
                "hit": ctx.seq(build_native_command5_target_sequence_hit(state, s, side, true), frames, c5_state_json),
                "miss": ctx.seq(build_native_command5_target_sequence_hit(state, s, side, false), frames, c5_state_json),
                "target": ctx.seq(build_native_command5_target_sequence(state, s, side), frames, c5_state_json),
                "transition": ctx.seq(build_native_command5_transition_sequence(state, s, side), |l| l.iter().map(|f| {
                    let mut j = c5_frame(&f.frame);
                    j["use_next"] = json!(f.use_next_target);
                    j["off"] = json!(f.target_offset_x);
                    j
                }).collect(), c5_state_json),
                "tail": ctx.seq(build_native_command5_tail_sequence(state, s, side), frames, c5_state_json),
            })
        }
        "c5_walk" => {
            let (s, side, targets) = (c5_sched(&input["schedule"]), i(&input["side"]) as u8, i(&input["targets"]));
            ctx.walk(input, &|rng, r| walk_native_command5_rng(rng, s, side, targets, r))
        }
        "c6_coords" => {
            let mut radii = vec![1i64 << 40, -(1i64 << 40), 1 << 53, 1 << 62, i64::MAX, i64::MIN, i64::MAX / 2];
            radii.extend(-60..=130);
            let mut out = Vec::new();
            for radius in radii {
                for base in [0u8, 30, 90, 255] {
                    out.push(json!([radius, base, c6_points_json(&native_command6_coordinates(radius, base))]));
                }
            }
            Value::Array(out)
        }
        "c6_target_coords" => {
            c6_points_json(&native_command6_target_coordinates(c6_points(&input["points"]), i(&input["next"])))
        }
        "c6_orbit" => {
            let (s, radius, side) = (c6_sched(&input["schedule"]), i(&input["radius"]), i(&input["side"]) as u8);
            json!({
                "prelude": ctx.res(plan_native_command6_prelude_frame(radius, side, s), c6_orbit),
                "tail": ctx.res(plan_native_command6_tail_frame(radius, side, s), c6_orbit),
            })
        }
        "c6_plan" => {
            let (s, points, side) =
                (c6_sched(&input["schedule"]), c6_points(&input["points"]), i(&input["side"]) as u8);
            let state =
                NativeCommand6TargetState { counters: ia(&input["state"]["c"]), secondary: ba(&input["state"]["s"]) };
            json!({
                "plan": ctx.res(plan_native_command6_target_frame(state, s, points, side), c6_frame),
                "sequence": ctx.res(build_native_command6_target_sequence_from_state(state, s, points, side),
                    |l| l.iter().map(c6_frame).collect()),
                "transition": ctx.res(build_native_command6_transition_sequence(state, s, points, side), |l| l.iter().map(|f| json!({
                    "m4": c6_layers(&f.mode4), "m5": c6_layers(&f.mode5), "use_next": f.use_next_target,
                    "off": f.target_offset_x, "next": c6_state_json(&f.next) })).collect()),
            })
        }
        "c6_walk" => {
            let (s, side, targets) = (c6_sched(&input["schedule"]), i(&input["side"]) as u8, i(&input["targets"]));
            ctx.walk(input, &|rng, r| walk_native_command6_rng(rng, s, side, targets, r))
        }
        "c6_display" => {
            let side = i(&input["side"]) as u8;
            let s = build_native_command6_presentation_schedule(side, Some(&bank(10, 0))).unwrap();
            let planned = build_native_command6_target_sequence(s, c6_points(&input["points"]), side).unwrap();
            let st = ia::<3>(&input["state"]);
            let state = NativeCommand6DisplayState { shade: st[0], pose: st[1], jitter: st[2] };
            let keep = i(&input["keep"]) as usize;
            ctx.res(
                build_native_command6_target_display_frames(
                    state,
                    &planned[..keep],
                    b(&input["hit"]),
                    i(&input["rng"]) as u16,
                ),
                |l| {
                    l.iter()
                        .map(|f| {
                            json!([
                                f.shader,
                                f.offset_x,
                                f.offset_y,
                                f.rng_after,
                                f.next.shade,
                                f.next.pose,
                                f.next.jitter
                            ])
                        })
                        .collect()
                },
            )
        }
        "c7_plan" => {
            let (s, state, side) = (c7_sched(&input["schedule"]), c7_state(&input["state"]), i(&input["side"]) as u8);
            let frames = |l: &Vec<NativeCommand7Frame>| -> Value { l.iter().map(c7_frame).collect() };
            json!({
                "plan": ctx.res(plan_native_command7_draw_frame(state, s, side), c7_frame),
                "target": ctx.seq(build_native_command7_target_sequence(state, s, side), frames, c7_state_json),
                "transition": ctx.seq(build_native_command7_transition_sequence(state, s, side), |l| l.iter().map(|f| {
                    let mut j = c7_frame(&f.frame);
                    j["use_next"] = json!(f.use_next_target);
                    j["off"] = json!(f.target_offset_x);
                    j
                }).collect(), c7_state_json),
                "tail": ctx.seq(build_native_command7_tail_sequence(state, s, side), frames, c7_state_json),
            })
        }
        "c7_walk" => {
            let (s, side, targets) = (c7_sched(&input["schedule"]), i(&input["side"]) as u8, i(&input["targets"]));
            ctx.walk(input, &|rng, r| walk_native_command7_rng(rng, s, side, targets, r))
        }
        "c8_plan" => {
            let v = &input["schedule"];
            let s = NativeCommand8PresentationSchedule {
                effect_resource: i(&v["effect"]),
                sound_resource: i(&v["sound"]),
                frame_bases: ia(&v["bases"]),
            };
            let state = NativeCommand8State { counters: ia(&input["state"]) };
            let side = i(&input["side"]) as u8;
            let frames = |l: &Vec<NativeCommand8Frame>| -> Value { l.iter().map(c8_frame).collect() };
            let counters = |s: &NativeCommand8State| json!(s.counters);
            json!({
                "plan": ctx.res(plan_native_command8_draw_frame(state, s), c8_frame),
                "target": ctx.seq(build_native_command8_target_sequence(state, s), frames, counters),
                "transition": ctx.seq(build_native_command8_transition_sequence(state, s, side), |l| l.iter().map(|f| {
                    let mut j = c8_frame(&f.frame);
                    j["use_next"] = json!(f.use_next_target);
                    j["off"] = json!(f.target_offset_x);
                    j
                }).collect(), counters),
                "tail": ctx.seq(build_native_command8_tail_sequence(state, s), frames, counters),
            })
        }
        "c9_plan" => {
            let built = ctx.res(
                build_native_command9_ai_schedule(i(&input["side"]) as u8, Some(&bank(i(&input["count"]) as usize, 0))),
                c9_sched_json,
            );
            let s = c9_sched(&input["schedule"]);
            let state = NativeCommand9AIState { counter: i(&input["state"][0]), toggle: b(&input["state"][1]) };
            json!({
                "built": built,
                "orbit": ctx.res(plan_native_command9_ai_orbit_frame(state, s), c9_frame),
                "plan": ctx.res(plan_native_command9_ai_target_frame(state, s), c9_frame),
                "sequence": ctx.seq(build_native_command9_ai_target_sequence(state, s),
                    |l| l.iter().map(c9_frame).collect(), |st| json!([st.counter, st.toggle])),
            })
        }
        "compound_grid" => {
            let mut grid = Vec::new();
            for id in 30..=37 {
                for count in [-1i64, 0, 1, 2, 8] {
                    let a = Animation { frames: vec![Frame::default(); count.max(0) as usize], ..Animation::default() };
                    let a = if count >= 0 { Some(&a) } else { None };
                    grid.push(ctx.res(build_native_compound_presentation_schedule(id, a), compound_json));
                }
            }
            Value::Array(grid)
        }
        "strike" => {
            let frames = input["frames"]
                .as_array()
                .unwrap()
                .iter()
                .map(|f| Frame {
                    delay: i(&f[0]),
                    raw_byte4: i(&f[1]) as u8,
                    raw_byte5: i(&f[2]) as u8,
                    ..Frame::default()
                })
                .collect();
            let a = Animation {
                frames,
                header_byte1: i(&input["h1"]) as u8,
                header_byte2: i(&input["h2"]) as u8,
                header_byte4: i(&input["h4"]) as u8,
            };
            let built = ctx.builders(Some(&a));
            json!({ "strike": built["strike"], "c24": built["c24"] })
        }
        "c24_mutated" => {
            let mut a = c24_base(ctx);
            let (frame, field) = (i(&input["frame"]), input["field"].as_str().unwrap());
            if frame >= 0 {
                let f = &mut a.frames[frame as usize];
                match field {
                    "raw4" => f.raw_byte4 = f.raw_byte4.wrapping_add(1),
                    "raw5" => f.raw_byte5 = f.raw_byte5.wrapping_add(1),
                    "delay" => f.delay += 1,
                    "raw7" => f.raw_byte7 = f.raw_byte7.wrapping_add(1),
                    "delay0" => f.delay = 0,
                    "delay256" => f.delay = 256,
                    other => panic!("unknown field {other}"),
                }
            } else {
                match field {
                    "none" => {}
                    "h1" => a.header_byte1 = a.header_byte1.wrapping_add(1),
                    "h2" => a.header_byte2 = a.header_byte2.wrapping_add(1),
                    "h4" => a.header_byte4 = a.header_byte4.wrapping_add(1),
                    "drop" => {
                        a.frames.pop();
                    }
                    other => panic!("unknown field {other}"),
                }
            }
            ctx.res(build_native_command24_schedule(Some(&a)), c24_json)
        }
        "table" => {
            ctx.reset_work();
            let path = ctx.work.join("entries.json");
            std::fs::write(&path, input["text"].as_str().unwrap()).unwrap();
            ctx.table(&path.to_string_lossy())
        }
        "table_path" => ctx.table(&ctx.expand(input["path"].as_str().unwrap())),
        "sep_tree" => {
            ctx.reset_work();
            for (name, content) in input["files"].as_object().unwrap() {
                let path = ctx.work.join(name);
                std::fs::create_dir_all(path.parent().unwrap()).unwrap();
                match content.as_str() {
                    Some(h) => std::fs::write(&path, unhex(h)).unwrap(),
                    None => std::fs::create_dir_all(&path).unwrap(),
                }
            }
            let results: Vec<Value> = input["calls"]
                .as_array()
                .unwrap()
                .iter()
                .map(|c| {
                    let (root, resource) = (c["root"].as_str().unwrap(), i(&c["resource"]));
                    match c["fn"].as_str().unwrap() {
                        "load" => ctx.sep_load(root, c["source"].as_str().unwrap(), resource),
                        _ => ctx.sep_fallback_source(root, resource),
                    }
                })
                .collect();
            Value::Array(results)
        }
        other => panic!("未知 op {other}"),
    }
}

fn tree_manifest(root: &Path) -> (usize, String) {
    fn walk(dir: &Path, root: &Path, out: &mut Vec<String>) {
        for entry in std::fs::read_dir(dir).unwrap() {
            let path = entry.unwrap().path();
            if path.is_dir() {
                walk(&path, root, out);
            } else {
                let rel = path.strip_prefix(root).unwrap().to_string_lossy().into_owned();
                out.push(format!("{rel} {}", digest(&std::fs::read(&path).unwrap())));
            }
        }
    }
    let mut lines = Vec::new();
    walk(root, root, &mut lines);
    lines.sort();
    (lines.len(), digest(lines.join("\n").as_bytes()))
}

fn env_path(var: &str) -> String {
    std::env::var(var).unwrap_or_else(|_| panic!("黃金差分需要 {var}"))
}

#[test]
fn matches_frozen_go_golden() {
    let Ok(path) = std::env::var("FD2_GOLDEN_FIGANI") else {
        assert!(std::env::var("FD2_REQUIRE_GOLDEN").is_err(), "FD2_REQUIRE_GOLDEN=1 但沒有 FD2_GOLDEN_FIGANI");
        eprintln!("略過：未設定 FD2_GOLDEN_FIGANI");
        return;
    };
    let doc: Value = serde_json::from_slice(&std::fs::read(&path).expect("讀黃金檔")).unwrap();
    assert_eq!(doc["schema"], "fd2-golden-figani/1");
    let flame2 = env_path("FD2_FLAME2_DIR");
    let pack = format!("{}/animations", env_path("FD2_TEST_ORIGINAL_PACK"));
    let entries = format!("{}/data/native_command_indexed_entries.json", env_path("FD2_REMAKE_ASSETS"));
    let ctx = Ctx {
        figani: format!("{flame2}/FIGANI.DAT"),
        fdother: format!("{flame2}/FDOTHER.DAT"),
        pack: pack.clone(),
        work: std::env::temp_dir().join(format!("fd2-golden-figani-{}", std::process::id())),
    };
    // 真實檔案先核對雜湊，確認 Rust 讀到的是 Go 產生黃金資料時的同一份輸入。
    let inputs = &doc["inputs"];
    for (key, file) in [("figani_dat", &ctx.figani), ("fdother_dat", &ctx.fdother), ("indexed_entries", &entries)] {
        assert_eq!(digest(&std::fs::read(file).unwrap()), inputs[key]["sha256"], "{key} 雜湊不符：{file}");
    }
    let (files, manifest) = tree_manifest(Path::new(&pack));
    assert_eq!(
        (files as u64, manifest.as_str()),
        (
            inputs["pack_animations"]["files"].as_u64().unwrap(),
            inputs["pack_animations"]["manifest_sha256"].as_str().unwrap(),
        ),
        "分離素材包內容不符"
    );

    let cases = doc["cases"].as_array().unwrap();
    assert!(!cases.is_empty());
    let mut failures = Vec::new();
    let mut tolerated = std::collections::BTreeMap::<String, usize>::new();
    let mut kinds = std::collections::BTreeMap::<String, usize>::new();
    for case in cases {
        let (index, kind, op) =
            (case["index"].as_u64().unwrap(), case["kind"].as_str().unwrap(), case["op"].as_str().unwrap());
        *kinds.entry(kind.to_string()).or_default() += 1;
        let got = compute(&ctx, op, &case["input"]);
        let Some(diff) = first_diff("result", &case["result"], &got) else { continue };
        let known = KNOWN_DIFFERENCES.iter().any(|(k, _)| *k == kind);
        if known && first_diff("result", &shape(&case["result"]), &shape(&got)).is_none() {
            *tolerated.entry(kind.to_string()).or_default() += 1;
            continue;
        }
        failures.push(format!("case {index} ({kind}/{op}): {diff}"));
    }
    let _ = std::fs::remove_dir_all(&ctx.work);
    eprintln!("黃金案例 {}：{kinds:?}", cases.len());
    eprintln!("已知差異（刻意不照做，只比接受／拒收）：{tolerated:?}");
    let shown: Vec<_> = failures.iter().take(40).cloned().collect();
    assert!(failures.is_empty(), "{} 個案例不符：\n{}", failures.len(), shown.join("\n"));
}
