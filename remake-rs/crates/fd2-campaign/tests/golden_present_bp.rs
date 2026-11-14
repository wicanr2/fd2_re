//! 與凍結 Go 版 `remake/cmd/fd2-golden-present-bp` 的黃金差分：fd2-campaign `battlepresent`（戰鬥演出畫格合成）。
//!
//! 由 `tools/rust_golden_present_bp.sh` 產生黃金 JSON 並設定 `FD2_GOLDEN_PRESENT_BP`；真實輸入掛在
//! `FD2_GOLDEN_PACK`（分離素材包）底下，先逐檔核對黃金檔記錄的大小與 SHA-256。設定
//! `FD2_REQUIRE_GOLDEN=1` 時，黃金檔或真實輸入缺少即失敗。
//!
//! 比對方式：每個案例依 `op` 在 Rust 端重做同一串呼叫（動畫依 JSON 記下的 FIGANI／FDOTHER 編號從
//! 分離素材包載入；底圖、工作緩衝、合成 RLE 幀與 LMI1 項目由 JSON 記下的種子經同一個 LCG 重建），
//! 結果與 Go 逐欄比對：每張畫格比 SHA-256，序列比每一步的來源層、HP 段、音效旗標與下一個 idle 狀態，
//! 錯誤比 Go `err.Error()` 原文。表示法正規化列在 [`NORMALIZATIONS`]，刻意不照做 Go 的行為列在
//! [`KNOWN_DIFFERENCES`]。

use fd2_campaign::battlepresent::*;
use fd2_campaign::indexedmap::{NATIVE_MAP_VGA_SIZE, NATIVE_UNIT_PRESENT_WORK_SIZE, NativeCommandHealTailTarget};
use fd2_formats::{fdother, figani};
use fd2_rules::battle::{NativePhysicalRollResult, NativePhysicalStrike};
use serde_json::{Value, json};
use sha2::{Digest, Sha256};
use std::collections::BTreeMap;
use std::path::{Path, PathBuf};

/// 刻意不照做 Go 的行為。黃金案例不依賴這些差異。
const KNOWN_DIFFERENCES: &[&str] = &[
    // Go 的 `*figani.Animation`／`*fdother.Frame` 可為 nil；Rust 以 `Option` 保留，黃金只走非 nil 的真實素材。
    "nil 指標參數只以 Option 表示",
    // Go 的 ResolveTarget 可回傳 error；黃金案例的確定性擲骰不回傳錯誤。
    "ResolveTarget 錯誤路徑不在黃金案例內",
];

/// 比對前的表示法正規化（不影響行為）：
const NORMALIZATIONS: &[&str] = &[
    // Go nil 切片寫成 null、空切片寫成 []；Rust 的 Vec 不分 nil，[] 與 null 視為相同。
    "null 與空陣列相同",
];

// ---- 兩端共用的確定性產生器（與 Go 匯出工具逐行相同）----

struct Lcg(u64);

impl Lcg {
    fn next(&mut self) -> u32 {
        self.0 = self.0.wrapping_mul(6364136223846793005).wrapping_add(1442695040888963407);
        (self.0 >> 32) as u32
    }
}

fn noise(seed: u64, n: usize) -> Vec<u8> {
    let mut g = Lcg(seed);
    (0..n).map(|_| (g.next() >> 24) as u8).collect()
}

fn rle_frame(seed: u64, w: i64, h: i64) -> fdother::Frame {
    let mut g = Lcg(seed);
    let mut raw = Vec::new();
    raw.extend_from_slice(&(w as u16).to_le_bytes());
    raw.extend_from_slice(&(h as u16).to_le_bytes());
    for _ in 0..h {
        let mut remaining = w;
        while remaining > 0 {
            let count = (1 + (g.next() % 64) as i64).min(remaining);
            raw.push((count - 1) as u8);
            raw.push(g.next() as u8);
            remaining -= count;
        }
    }
    fdother::Frame::raw(0, 0, w, h, raw)
}

fn lmi_list(seed: u64, count: usize) -> Vec<fdother::LMI1Entry> {
    let mut g = Lcg(seed);
    (0..count)
        .map(|_| {
            let (w, h) = (1 + (g.next() % 3) as i64, 1 + (g.next() % 3) as i64);
            let pixels = (0..w * h).map(|_| g.next() as u8).collect();
            fdother::LMI1Entry { width: w, height: h, pixels }
        })
        .collect()
}

fn hex(b: &[u8]) -> String {
    b.iter().map(|x| format!("{x:02x}")).collect()
}

fn digest(b: &[u8]) -> String {
    hex(&Sha256::digest(b))
}

// ---- JSON 讀取 ----

fn i(v: &Value, key: &str) -> i64 {
    v[key].as_i64().unwrap_or_else(|| panic!("欄位 {key} 不是整數：{}", v[key]))
}

fn u(v: &Value, key: &str) -> u64 {
    v[key].as_u64().unwrap_or_else(|| panic!("欄位 {key} 不是無號整數：{}", v[key]))
}

fn b(v: &Value, key: &str) -> bool {
    v[key].as_bool().unwrap_or_else(|| panic!("欄位 {key} 不是布林：{}", v[key]))
}

fn arr<'a>(v: &'a Value, key: &str) -> &'a Vec<Value> {
    v[key].as_array().unwrap_or_else(|| panic!("欄位 {key} 不是陣列：{}", v[key]))
}

fn seeds(v: &Value) -> Vec<u64> {
    v.as_array().unwrap().iter().map(|x| x.as_u64().unwrap()).collect()
}

/// 依 [`NORMALIZATIONS`] 比對，回傳第一個不同的位置。
fn diff(path: &str, go: &Value, rust: &Value) -> Option<String> {
    let empty = |v: &Value| v.is_null() || v.as_array().is_some_and(Vec::is_empty);
    match (go, rust) {
        (Value::Array(a), Value::Array(b)) => {
            if a.len() != b.len() {
                return Some(format!("{path}: 長度 go={} rust={}", a.len(), b.len()));
            }
            a.iter().zip(b).enumerate().find_map(|(k, (x, y))| diff(&format!("{path}[{k}]"), x, y))
        }
        (Value::Object(a), Value::Object(b)) => {
            for (k, x) in a {
                match b.get(k) {
                    Some(y) => {
                        if let Some(d) = diff(&format!("{path}.{k}"), x, y) {
                            return Some(d);
                        }
                    }
                    None => return Some(format!("{path}.{k}: Rust 沒有這個欄位")),
                }
            }
            b.keys().find(|k| !a.contains_key(*k)).map(|k| format!("{path}.{k}: Go 沒有這個欄位"))
        }
        _ if empty(go) && empty(rust) => None,
        _ => (go != rust).then(|| format!("{path}: go={go} rust={rust}")),
    }
}

struct Ctx {
    pack: PathBuf,
    anims: BTreeMap<(String, i64), figani::Animation>,
}

impl Ctx {
    fn anim(&mut self, r: &Value) -> figani::Animation {
        let (src, id) = (r["src"].as_str().unwrap().to_string(), i(r, "id"));
        let root = self.pack.join("animations");
        let root = root.to_str().unwrap();
        self.anims
            .entry((src.clone(), id))
            .or_insert_with(|| {
                if src == "FIGANI" {
                    figani::load_separated_resource(root, id)
                } else {
                    figani::load_separated_archive_resource(root, &format!("{src}.DAT"), id)
                }
                .unwrap_or_else(|e| panic!("{src}:{id}: {e}"))
            })
            .clone()
    }
}

fn err_v(e: impl std::fmt::Display) -> Value {
    json!({ "error": e.to_string() })
}

fn effect_layers(layers: &[NativeCommandEffectSource]) -> Value {
    Value::Array(layers.iter().map(|l| json!([l.frame, l.x, l.y])).collect())
}

fn src_v(s: &NativeCommandSources) -> Value {
    json!({
        "base": digest(&s.base), "actor": s.actor_frame, "ax": s.actor_offset_x, "ti": s.target_index,
        "tf": s.target_frame, "tx": s.target_offset_x, "ty": s.target_offset_y, "before": effect_layers(&s.before),
        "after": effect_layers(&s.after), "phase": s.phase, "tbe": s.target_before_effects, "omit": s.omit_target,
        "shader": s.packed_target_shader,
    })
}

fn shas(frames: &[Vec<u8>]) -> Value {
    Value::Array(frames.iter().map(|f| json!(digest(f))).collect())
}

fn srcs_v(s: &[NativeCommandSources]) -> Value {
    Value::Array(s.iter().map(src_v).collect())
}

fn rendered(px: &[u8], hp: i64, p1: bool, p2: bool, src: &NativeCommandSources) -> Value {
    json!({ "px": digest(px), "hp": hp, "p1": p1, "p2": p2, "src": src_v(src) })
}

fn list<T>(items: &[T], f: impl Fn(&T) -> Value) -> Value {
    Value::Array(items.iter().map(f).collect())
}

fn nested<T>(items: &[Vec<T>], f: impl Fn(&T) -> Value + Copy) -> Value {
    Value::Array(items.iter().map(|it| list(it, f)).collect())
}

fn c1src(s: &NativeCommand1FrameSources) -> Value {
    let f = |l: &NativeCommand1EffectLayer| json!([l.frame, l.x, l.y]);
    json!({
        "base": digest(&s.base), "actor": s.actor_frame, "tf": s.target_frame, "tx": s.target_offset_x,
        "next": s.next_target, "before": list(&s.before_target, f), "after": list(&s.after_target, f),
    })
}

/// cmd3／5／6／7／8／9 共用的序列輸入（Go `seqInput`）。
struct Seq {
    side: u8,
    effect: figani::Animation,
    actor: figani::Animation,
    idles: Vec<figani::Animation>,
    front: Vec<u8>,
    tail: Vec<u8>,
    target_bases: Vec<Vec<Vec<u8>>>,
    transition_bases: Vec<Vec<u8>>,
}

impl Seq {
    fn new(ctx: &mut Ctx, v: &Value) -> Seq {
        Seq {
            side: i(v, "side") as u8,
            effect: ctx.anim(&v["effect"]),
            actor: ctx.anim(&v["actor"]),
            idles: arr(v, "idles").iter().map(|r| ctx.anim(r)).collect(),
            front: noise(u(v, "front"), 64000),
            tail: noise(u(v, "tail"), 64000),
            target_bases: arr(v, "target_bases")
                .iter()
                .map(|t| seeds(t).into_iter().map(|s| noise(s, 64000)).collect())
                .collect(),
            transition_bases: seeds(&v["transition_bases"]).into_iter().map(|s| noise(s, 64000)).collect(),
        }
    }

    fn target_refs(&self) -> Vec<Vec<&[u8]>> {
        self.target_bases.iter().map(|t| t.iter().map(Vec::as_slice).collect()).collect()
    }

    fn transition_refs(&self) -> Vec<&[u8]> {
        self.transition_bases.iter().map(Vec::as_slice).collect()
    }

    fn idle_refs(&self) -> Vec<Option<&figani::Animation>> {
        self.idles.iter().map(Some).collect()
    }
}

fn run_case(ctx: &mut Ctx, op: &str, v: &Value) -> Value {
    match op {
        "cmd0_target" => {
            let side = i(v, "side") as u8;
            let effect = ctx.anim(&v["effect"]);
            let target = ctx.anim(&v["target"]);
            let schedule = match figani::build_native_command0_presentation_schedule(side, Some(&effect)) {
                Ok(s) => s,
                Err(e) => return err_v(e),
            };
            match compose_native_command0_target_frame(
                &noise(u(v, "base"), 64000),
                &target.frames[i(v, "frame") as usize],
                Some(&effect),
                schedule,
                i(v, "step"),
            ) {
                Ok(px) => json!({ "px": digest(&px) }),
                Err(e) => err_v(e),
            }
        }
        "cmd0_actor" => {
            let mut actor = ctx.anim(&v["actor"]);
            let marker = i(v, "marker");
            if marker >= 0 {
                actor.frames[marker as usize].raw_byte4 = 1;
            }
            let target = ctx.anim(&v["target"]);
            let (before, after, lut) =
                (noise(u(v, "before"), 64000), noise(u(v, "after"), 64000), noise(u(v, "lut"), 256));
            let r = build_native_command0_actor_frames(&NativeCommand0ActorInput {
                base_before: &before,
                base_after: &after,
                actor_effect: Some(&actor),
                first_target_idle: Some(&target),
                raw_side: i(v, "side") as u8,
                background: rle_frame(u(v, "bg"), 320, 100),
                platform: rle_frame(u(v, "platform"), 40, 20),
                lut: &lut,
            });
            match r {
                Ok(frames) => json!({ "frames": list(&frames, |f| json!({
                    "px": digest(&f.pixels), "ef": f.effect_frame, "er": f.effect_repeat, "tf": f.target_idle_frame,
                    "mp": f.publish_mp, "pulse": f.pulse,
                })) }),
                Err(e) => err_v(e),
            }
        }
        "cmd1_seq" => {
            let side = i(v, "side") as u8;
            let effect = ctx.anim(&v["effect"]);
            let target = ctx.anim(&v["target"]);
            let bases: Vec<Vec<u8>> = seeds(&v["bases"]).into_iter().map(|s| noise(s, 64000)).collect();
            let refs: Vec<&[u8]> = bases.iter().map(Vec::as_slice).collect();
            let schedule = match figani::build_native_command1_presentation_schedule(side, Some(&effect)) {
                Ok(s) => s,
                Err(e) => return err_v(e),
            };
            match build_native_command1_target_sequence(&refs, Some(&target), Some(&effect), schedule) {
                Ok(seq) => json!({
                    "frames": shas(&seq.frames), "src": list(&seq.sources, c1src), "hp": seq.hp_stages,
                    "samples": seq.sample_markers, "next": [seq.next_idle_frame, seq.next_idle_repeat],
                }),
                Err(e) => err_v(e),
            }
        }
        "cmd1_trans" => {
            let (a, c, n) = (ctx.anim(&v["actor"]), ctx.anim(&v["current"]), ctx.anim(&v["next"]));
            match build_native_command1_transition_sequence(
                &noise(u(v, "base"), 64000),
                Some(&a),
                Some(&c),
                Some(&n),
                i(v, "side") as u8,
            ) {
                Ok(seq) => json!({ "frames": shas(&seq.frames), "src": list(&seq.sources, c1src) }),
                Err(e) => err_v(e),
            }
        }
        "cmd2_target" => {
            let (side, sched_side) = (i(v, "side") as u8, i(v, "sched_side") as u8);
            let effect = ctx.anim(&v["effect"]);
            let sched_ref = json!({ "src": "FDOTHER", "id": if sched_side == 0 { 27 } else { 26 } });
            let sched_effect = ctx.anim(&sched_ref);
            let target = ctx.anim(&v["target"]);
            let schedule = match figani::build_native_command2_presentation_schedule(sched_side, Some(&sched_effect)) {
                Ok(s) => s,
                Err(e) => return err_v(e),
            };
            match compose_native_command2_target_frame_with_sources(
                &noise(u(v, "base"), 64000),
                &target.frames[i(v, "frame") as usize],
                Some(&effect),
                schedule,
                side,
                i(v, "state"),
            ) {
                Ok((px, src)) => json!({ "px": digest(&px), "src": src_v(&src) }),
                Err(e) => err_v(e),
            }
        }
        "cmd2_orbit" => {
            let side = i(v, "side") as u8;
            let (effect, actor, target) = (ctx.anim(&v["effect"]), ctx.anim(&v["actor"]), ctx.anim(&v["target"]));
            let helper = figani::NativeCommand2HelperFrame { effect_frame: i(v, "helper"), ..Default::default() };
            let base = noise(u(v, "base"), 64000);
            let r = if b(v, "transition") {
                compose_native_command2_transition_frame_with_sources(
                    &base,
                    Some(&actor),
                    Some(&target),
                    Some(&effect),
                    helper,
                    side,
                    i(v, "offset"),
                )
            } else {
                compose_native_command2_orbit_frame_with_sources(
                    &base,
                    Some(&actor),
                    Some(&target),
                    Some(&effect),
                    helper,
                    side,
                )
            };
            match r {
                Ok((px, src)) => json!({ "px": digest(&px), "src": src_v(&src) }),
                Err(e) => err_v(e),
            }
        }
        "cmd24_bg" => {
            let ls = seeds(&v["layers"]);
            let mut layers = [rle_frame(ls[0], 320, 100), rle_frame(ls[1], 320, 100), rle_frame(ls[2], 320, 100)];
            if b(v, "broken") {
                layers[2] = rle_frame(ls[2], 320, 99);
            }
            let (source, target) = (noise(u(v, "source"), 64000), noise(u(v, "target"), 64000));
            let idle = ctx.anim(&v["idle"]);
            let in_ = NativeCommand24BackgroundInputs {
                layers,
                source: &source,
                target: &target,
                target_idle: idle.frames[i(v, "frame") as usize].clone(),
            };
            let r = if b(v, "right") {
                build_native_physical_right_background_presents(&in_)
            } else {
                build_native_command24_background_presents(&in_)
            };
            match r {
                Ok(presents) => json!({ "presents": list(&presents, |p| json!({
                    "px": digest(&p.pixels), "in": p.incoming, "cx": p.content_x, "bgx": p.bgx, "vx": p.viewport_x,
                    "bg": p.bg_layers,
                })) }),
                Err(e) => err_v(e),
            }
        }
        "prelude" => {
            let (actor, target) = (ctx.anim(&v["actor"]), ctx.anim(&v["target"]));
            let base = noise(u(v, "base"), 64000);
            let dac: Vec<u8> = noise(u(v, "dac"), 768).into_iter().map(|x| x & 63).collect();
            let platform = rle_frame(u(v, "platform"), 40, 20);
            let r = build_native_command_prelude_frames(&NativeCommandPreludeInput {
                base: &base,
                actor_idle: actor.frames[i(v, "aframe") as usize].clone(),
                first_target_idle: target.frames[i(v, "tframe") as usize].clone(),
                platform: b(v, "with_platform").then_some(&platform),
                raw_side: i(v, "side") as u8,
                mode: i(v, "mode") as u8,
                baseline_dac: &dac,
            });
            match r {
                Ok(frames) => json!({ "frames": list(&frames, |f| json!({
                    "stage": f.stage, "px": digest(&f.pixels), "dac": digest(&f.dac),
                })) }),
                Err(e) => err_v(e),
            }
        }
        "cmd3_seq" => {
            let s = Seq::new(ctx, v);
            let schedule = match figani::build_native_command3_presentation_schedule(s.side, Some(&s.effect)) {
                Ok(x) => x,
                Err(e) => return err_v(e),
            };
            let r = build_native_command3_effect_sequence(&NativeCommand3EffectInput {
                front_base: &s.front,
                tail_base: &s.tail,
                target_bases: s.target_refs(),
                transition_bases: s.transition_refs(),
                actor_effect: Some(&s.actor),
                target_idle: s.idle_refs(),
                effect: Some(&s.effect),
                schedule,
                raw_side: s.side,
            });
            let f =
                |r: &NativeCommand3RenderedFrame| rendered(&r.pixels, r.hp_stage, r.play_sub1, r.play_sub2, &r.sources);
            match r {
                Ok(seq) => json!({
                    "front": list(&seq.front, f), "targets": nested(&seq.targets, f),
                    "transitions": nested(&seq.transitions, f), "tail": list(&seq.tail, f),
                    "next": [seq.next_idle_frame, seq.next_idle_repeat],
                }),
                Err(e) => err_v(e),
            }
        }
        "cmd5_seq" => {
            let s = Seq::new(ctx, v);
            let schedule = if i(v, "command") == 4 {
                figani::build_native_command4_presentation_schedule(s.side, Some(&s.effect))
            } else {
                figani::build_native_command5_presentation_schedule(s.side, Some(&s.effect))
            };
            let schedule = match schedule {
                Ok(x) => x,
                Err(e) => return err_v(e),
            };
            // 與 Go 匯出工具相同的確定性擲骰：next = rng*75+74+index（u16 繞回），hit = bit3 為 0。
            let mut resolve = |index: i64, rng: u16| -> Result<(u16, bool)> {
                Ok((rng.wrapping_mul(75).wrapping_add(74).wrapping_add(index as u16), (rng >> 3) & 1 == 0))
            };
            let r = build_native_command5_effect_sequence(NativeCommand5EffectInput {
                front_base: &s.front,
                tail_base: &s.tail,
                target_bases: s.target_refs(),
                transition_bases: s.transition_refs(),
                actor_effect: Some(&s.actor),
                target_idle: s.idle_refs(),
                effect: Some(&s.effect),
                schedule,
                raw_side: s.side,
                visual_rng: i(v, "visual_rng") as u16,
                resolve_target: if b(v, "resolve") { Some(&mut resolve) } else { None },
            });
            let f = |r: &NativeCommand5RenderedFrame| {
                let src = &r.sources;
                json!({
                    "px": digest(&r.pixels), "hp": r.hp_stage, "p1": r.play_primary, "p2": r.play_secondary,
                    "src": {
                        "base": digest(&src.base), "actor": src.actor_frame, "ti": src.target_index, "tf": src.target_frame,
                        "tx": src.target_offset_x, "layers": list(&src.layers, |l| json!([l.channel, l.frame, l.x])),
                        "phase": src.phase,
                    },
                })
            };
            match r {
                Ok(seq) => json!({
                    "front": list(&seq.front, f), "targets": nested(&seq.targets, f),
                    "transitions": nested(&seq.transitions, f), "tail": list(&seq.tail, f), "rng": seq.visual_rng,
                    "next": [seq.next_idle_frame, seq.next_idle_repeat],
                }),
                Err(e) => err_v(e),
            }
        }
        "cmd6_seq" => {
            let s = Seq::new(ctx, v);
            let hits: Vec<bool> = arr(v, "hits").iter().map(|x| x.as_bool().unwrap()).collect();
            let numeric: Vec<u16> = arr(v, "numeric_rng").iter().map(|x| x.as_u64().unwrap() as u16).collect();
            let schedule = match figani::build_native_command6_presentation_schedule(s.side, Some(&s.effect)) {
                Ok(x) => x,
                Err(e) => return err_v(e),
            };
            let r = build_native_command6_effect_sequence(&NativeCommand6EffectInput {
                target_hits: hits,
                target_numeric_rng: numeric,
                front_base: &s.front,
                tail_base: &s.tail,
                target_bases: s.target_refs(),
                transition_bases: s.transition_refs(),
                actor_effect: Some(&s.actor),
                target_idle: s.idle_refs(),
                effect: Some(&s.effect),
                schedule,
                raw_side: s.side,
            });
            match r {
                Ok(seq) => json!({
                    "rng": seq.rng_after, "front": shas(&seq.front), "front_src": srcs_v(&seq.front_sources),
                    "targets": list(&seq.targets, |t| json!({
                        "frames": shas(&t.frames), "hp": t.hp_stages, "src": srcs_v(&t.sources),
                    })),
                    "transitions": list(&seq.transitions, |t| shas(t)),
                    "transition_src": list(&seq.transition_sources, |t| srcs_v(t)),
                    "tail": shas(&seq.tail), "tail_src": srcs_v(&seq.tail_sources),
                }),
                Err(e) => err_v(e),
            }
        }
        "cmd7_seq" => {
            let s = Seq::new(ctx, v);
            let schedule = match figani::build_native_command7_presentation_schedule(s.side, Some(&s.effect)) {
                Ok(x) => x,
                Err(e) => return err_v(e),
            };
            let r = build_native_command7_effect_sequence(&NativeCommand7EffectInput {
                front_base: &s.front,
                tail_base: &s.tail,
                target_bases: s.target_refs(),
                transition_bases: s.transition_refs(),
                actor_effect: Some(&s.actor),
                target_idle: s.idle_refs(),
                effect: Some(&s.effect),
                schedule,
                raw_side: s.side,
            });
            let f = |r: &NativeCommand7RenderedFrame| {
                rendered(&r.pixels, r.hp_stage, r.play_handle_a, r.play_handle_b, &r.sources)
            };
            match r {
                Ok(seq) => json!({
                    "front": list(&seq.front, f), "targets": nested(&seq.targets, f),
                    "transitions": nested(&seq.transitions, f), "tail": list(&seq.tail, f),
                    "next": [seq.next_idle_frame, seq.next_idle_repeat],
                }),
                Err(e) => err_v(e),
            }
        }
        "cmd8_seq" => {
            let s = Seq::new(ctx, v);
            let schedule = match figani::build_native_command8_presentation_schedule(s.side, Some(&s.effect)) {
                Ok(x) => x,
                Err(e) => return err_v(e),
            };
            let r = build_native_command8_effect_sequence(&NativeCommand8EffectInput {
                front_base: &s.front,
                tail_base: &s.tail,
                target_bases: s.target_refs(),
                transition_bases: s.transition_refs(),
                actor_effect: Some(&s.actor),
                target_idle: s.idle_refs(),
                effect: Some(&s.effect),
                schedule,
                raw_side: s.side,
            });
            let f =
                |r: &NativeCommand8RenderedFrame| rendered(&r.pixels, r.hp_stage, r.play_sub1, r.play_sub2, &r.sources);
            match r {
                Ok(seq) => json!({
                    "front": list(&seq.front, f), "targets": nested(&seq.targets, f),
                    "transitions": nested(&seq.transitions, f), "tail": list(&seq.tail, f),
                    "next": [seq.next_idle_frame, seq.next_idle_repeat],
                }),
                Err(e) => err_v(e),
            }
        }
        "cmd9_seq" => {
            let s = Seq::new(ctx, v);
            let target = ctx.anim(&v["target"]);
            let schedule = match figani::build_native_command9_ai_schedule(s.side, Some(&s.effect)) {
                Ok(x) => x,
                Err(e) => return err_v(e),
            };
            let r = build_native_command9_ai_sequence(&NativeCommand9AIInput {
                before_base: &s.front,
                after_base: &s.tail,
                target_bases: s.target_bases[0].iter().map(Vec::as_slice).collect(),
                actor: Some(&s.actor),
                target: Some(&target),
                effect: Some(&s.effect),
                schedule,
            });
            let f =
                |r: &NativeCommand9RenderedFrame| rendered(&r.pixels, r.hp_stage, r.play_sub1, r.play_sub2, &r.sources);
            match r {
                Ok(seq) => json!({
                    "slide_in": list(&seq.slide_in, f), "front": list(&seq.front, f), "target": list(&seq.target, f),
                    "tail": list(&seq.tail, f), "slide_out": list(&seq.slide_out, f),
                    "next": [seq.next_idle_frame, seq.next_idle_repeat],
                }),
                Err(e) => err_v(e),
            }
        }
        "compound_common" => {
            let effect = ctx.anim(&v["effect"]);
            let actor = ctx.anim(&v["actor"]);
            let schedule = match figani::build_native_compound_presentation_schedule(i(v, "command"), Some(&effect)) {
                Ok(x) => x,
                Err(e) => return err_v(e),
            };
            match build_native_compound_common_frames(
                &noise(u(v, "base"), 64000),
                &actor.frames[i(v, "frame") as usize],
                Some(&effect),
                schedule,
            ) {
                Ok(f) => json!({
                    "actor": shas(&f.actor_slide), "effect": shas(&f.effect_slide), "main": shas(&f.main), "tail": shas(&f.tail),
                }),
                Err(e) => err_v(e),
            }
        }
        "compound_actor" => {
            let (actor, target) = (ctx.anim(&v["actor"]), ctx.anim(&v["target"]));
            match build_native_compound_actor_presents(
                &noise(u(v, "base"), 64000),
                Some(&actor),
                Some(&target),
                i(v, "side") as u8,
            ) {
                Ok(p) => json!({ "presents": list(&p, |p| json!({
                    "px": digest(&p.pixels), "ai": p.actor_index, "ti": p.target_index,
                })) }),
                Err(e) => err_v(e),
            }
        }
        "tail" => {
            let effect = lmi_list(u(v, "effect"), i(v, "effect_count") as usize);
            let digits = lmi_list(u(v, "digits"), i(v, "digit_count") as usize);
            let (ws, vs) = (NATIVE_UNIT_PRESENT_WORK_SIZE as usize, NATIVE_MAP_VGA_SIZE as usize);
            let (w, vga) = (noise(u(v, "work"), ws), noise(u(v, "vga"), vs));
            let (pw, pv) = (noise(u(v, "post_work"), ws), noise(u(v, "post_vga"), vs));
            let camera: Vec<i64> = arr(v, "camera").iter().map(|x| x.as_i64().unwrap()).collect();
            let targets: Vec<NativeCommandHealTailTarget> = arr(v, "targets")
                .iter()
                .map(|t| {
                    let t: Vec<i64> = t.as_array().unwrap().iter().map(|x| x.as_i64().unwrap()).collect();
                    NativeCommandHealTailTarget { record_index: t[0], x: t[1], y: t[2], selector_slot: t[3] }
                })
                .collect();
            let results: Vec<NativeCommand32TailResult> = arr(v, "results")
                .iter()
                .map(|r| NativeCommand32TailResult {
                    target_record: i(r, "record"),
                    hit: b(r, "hit"),
                    damage: i(r, "damage"),
                })
                .collect();
            let r = if b(v, "ai") {
                fdother::build_native_ai_item_type21_tail_schedule(i(v, "command"), &effect, &digits)
                    .map_err(|e| e.to_string())
                    .and_then(|s| {
                        build_native_ai_item_type21_tail_frames(
                            &w, &vga, &pw, &pv, &effect, &digits, &targets, &results, camera[0], camera[1], &s,
                        )
                        .map_err(|e| e.to_string())
                    })
            } else {
                fdother::build_native_command32_tail_schedule(&effect, &digits).map_err(|e| e.to_string()).and_then(
                    |s| {
                        build_native_command32_tail_frames(
                            &w, &vga, &pw, &pv, &effect, &digits, &targets, &results, camera[0], camera[1], &s,
                        )
                        .map_err(|e| e.to_string())
                    },
                )
            };
            let layers = |groups: &Vec<Vec<fd2_campaign::indexedmap::NativeCommandTailLayer>>| {
                list(groups, |g| list(g, |l| json!([l.resource, l.entry, l.x, l.y])))
            };
            match r {
                Ok(f) => json!({
                    "effect": shas(&f.effect), "toggle": shas(&f.toggle), "result": shas(&f.result),
                    "queue": list(&f.queue, |d| json!([d.position_code, d.target, d.digit])),
                    "effect_src": layers(&f.effect_sources), "toggle_src": layers(&f.toggle_sources),
                    "result_src": layers(&f.result_sources),
                }),
                Err(e) => err_v(e),
            }
        }
        "phys_plan" => {
            let (attack, idle) = (ctx.anim(&v["attack"]), ctx.anim(&v["idle"]));
            let strikes: Vec<NativePhysicalStrike> = arr(v, "strikes")
                .iter()
                .map(|s| NativePhysicalStrike {
                    roll: NativePhysicalRollResult {
                        missed: b(s, "missed"),
                        crit: b(s, "crit"),
                        status: b(s, "status"),
                        damage: i(s, "damage"),
                        ..Default::default()
                    },
                    defender_hp: i(s, "defender_hp"),
                    ..Default::default()
                })
                .collect();
            match build_native_physical_body_plan(Some(&attack), Some(&idle), i(v, "side") as u8, i(v, "hp"), &strikes)
            {
                Ok(plan) => json!({ "plan": list(&plan, |f| json!({
                    "strike": f.strike, "frame": f.frame, "hp": f.hp, "panel": f.update_panel, "cue": f.cue,
                    "presents": list(&f.presents, |p| json!([
                        p.idle_frame, p.dx, p.dy, p.opaque_fill, p.status_pulse, p.crit_pulse,
                    ])),
                })) }),
                Err(e) => err_v(e),
            }
        }
        "phys_present" => {
            let (attack, idle) = (ctx.anim(&v["attack"]), ctx.anim(&v["idle"]));
            match compose_native_physical_body_present(
                &noise(u(v, "base"), 64000),
                &attack.frames[i(v, "aframe") as usize],
                &idle.frames[i(v, "iframe") as usize],
                i(v, "side") as u8,
                NativePhysicalBodyPresent {
                    dx: i(v, "dx"),
                    dy: i(v, "dy"),
                    opaque_fill: i(v, "fill"),
                    ..Default::default()
                },
            ) {
                Ok(px) => json!({ "px": digest(&px) }),
                Err(e) => err_v(e),
            }
        }
        other => panic!("未知 op {other}"),
    }
}

fn require_golden() -> bool {
    std::env::var_os("FD2_REQUIRE_GOLDEN").is_some_and(|v| v == "1")
}

#[test]
fn golden_present_bp() {
    assert!(!KNOWN_DIFFERENCES.is_empty() && !NORMALIZATIONS.is_empty());
    let Some(path) = std::env::var_os("FD2_GOLDEN_PRESENT_BP").map(PathBuf::from).filter(|p| p.exists()) else {
        assert!(!require_golden(), "FD2_REQUIRE_GOLDEN=1 但缺少黃金檔 FD2_GOLDEN_PRESENT_BP");
        eprintln!("略過：沒有黃金檔 FD2_GOLDEN_PRESENT_BP");
        return;
    };
    let Some(pack) = std::env::var_os("FD2_GOLDEN_PACK").map(PathBuf::from).filter(|p| p.is_dir()) else {
        assert!(!require_golden(), "FD2_REQUIRE_GOLDEN=1 但缺少分離素材包 FD2_GOLDEN_PACK");
        eprintln!("略過：沒有分離素材包 FD2_GOLDEN_PACK");
        return;
    };
    let doc: Value = serde_json::from_slice(&std::fs::read(&path).unwrap()).unwrap();
    assert_eq!(doc["schema"], "fd2-golden-present-bp");
    for f in doc["files"].as_array().unwrap() {
        let label = f["path"].as_str().unwrap();
        let rel = label.strip_prefix("pack/").unwrap_or_else(|| panic!("未知檔案標籤 {label}"));
        let raw = std::fs::read(Path::new(&pack).join(rel)).unwrap_or_else(|e| panic!("{label}: {e}"));
        assert_eq!(raw.len() as u64, u(f, "size"), "{label} 大小不符");
        assert_eq!(digest(&raw), f["sha256"].as_str().unwrap(), "{label} SHA-256 不符");
    }
    let mut ctx = Ctx { pack, anims: BTreeMap::new() };
    let cases = doc["cases"].as_array().unwrap();
    let mut failures = Vec::new();
    let mut by_op: BTreeMap<String, usize> = BTreeMap::new();
    for case in cases {
        let op = case["op"].as_str().unwrap();
        *by_op.entry(op.to_string()).or_default() += 1;
        let got = run_case(&mut ctx, op, &case["input"]);
        if let Some(d) = diff(&format!("case{}({op})", case["index"]), &case["result"], &got) {
            failures.push(d);
        }
    }
    eprintln!(
        "present-bp 黃金：{} 個案例、{} 個檔案，各 op：{by_op:?}",
        cases.len(),
        doc["files"].as_array().unwrap().len()
    );
    assert!(failures.is_empty(), "{} 個案例不同：\n{}", failures.len(), failures.join("\n"));
}
