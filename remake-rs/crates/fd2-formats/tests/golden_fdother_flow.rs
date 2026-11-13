//! 與凍結 Go 版 `remake/cmd/fd2-golden-fdother-flow` 的黃金差分（fdother 第三組：依賴核心與原版流程邏輯
//! 的 action_overlay、ch28_post_present、indexed_transition*、native_*_tail、native_command9、native_ch23、
//! pending_code1_interlude、preparation_roster、spawn_intro、unit_present_schedule）。
//!
//! 由 `tools/rust_golden_fdother_flow.sh` 產生黃金 JSON 並設定 `FD2_GOLDEN_FDOTHER_FLOW`。真實原版檔與分離
//! 素材包不在黃金檔裡：`FD2_FLAME2_DIR` 指向原版 FLAME2 目錄，`FD2_ASSET_PACK` 指向分離素材包根目錄，兩者都
//! 先核對黃金檔記錄的 SHA-256。設定 `FD2_REQUIRE_GOLDEN=1` 時，黃金檔或真實檔目錄缺少即失敗，避免把
//! 「沒跑到」當成通過。
//!
//! 大緩衝不嵌入 JSON：內容由 `pat(n, k)` 公式產生（與 Go 匯出工具相同），結果只比 SHA-256。
//!
//! 每個案例的 `compare` 欄決定比對方式：
//! - `exact`：結果逐欄相同，錯誤文字逐字相同。
//! - `err_prefix`：Go 錯誤包了標準函式庫錯誤；Rust 必須同樣拒收，`FdotherError::go_prefix()` 必須等於
//!   黃金檔的 `prefix`，內層文字不比，其餘欄位逐欄比。
//!
//! Go 會因 64 位元繞回騙過範圍檢查而 panic 的輸入，這裡以 `catch_unwind` 對應，比對 panic 旗標與 panic
//! 之前已寫入的緩衝區摘要。連接受與否都和 Go 不同的案例種類列在 [`KNOWN_DIFFERENCES`]。

use fd2_formats::fdicon::{Bank, FdiconError, NATIVE_SIZE, Sprite};
use fd2_formats::fdother::*;
use serde_json::{Map, Value, json};
use sha2::{Digest, Sha256};
use std::cell::{Cell, RefCell};
use std::collections::BTreeMap;
use std::path::{Path, PathBuf};

/// 刻意不照做的 Go 行為：(kind 結尾, 原因)。本組目前沒有；保留清單讓差異政策與其他黃金測試一致。
const KNOWN_DIFFERENCES: &[(&str, &str)] = &[];

fn digest(b: &[u8]) -> String {
    Sha256::digest(b).iter().map(|x| format!("{x:02x}")).collect()
}

fn hex(b: &[u8]) -> String {
    b.iter().map(|x| format!("{x:02x}")).collect()
}

fn unhex(s: &str) -> Vec<u8> {
    (0..s.len()).step_by(2).map(|i| u8::from_str_radix(&s[i..i + 2], 16).unwrap()).collect()
}

/// Go 匯出工具的 `pat`。
fn pat(n: i64, k: i64) -> Vec<u8> {
    let k = k as u64;
    (0..n as u64).map(|i| (i.wrapping_mul(k).wrapping_add((i >> 8) * 3).wrapping_add(k)) as u8).collect()
}

fn i(v: &Value, key: &str) -> i64 {
    v[key].as_i64().unwrap_or_else(|| panic!("欄位 {key} 不是整數：{}", v[key]))
}

fn b(v: &Value, key: &str) -> bool {
    v[key].as_bool().unwrap_or_else(|| panic!("欄位 {key} 不是布林：{}", v[key]))
}

fn s<'a>(v: &'a Value, key: &str) -> &'a str {
    v[key].as_str().unwrap_or_else(|| panic!("欄位 {key} 不是字串：{}", v[key]))
}

fn ints(v: &Value) -> Vec<i64> {
    v.as_array().unwrap().iter().map(|x| x.as_i64().unwrap()).collect()
}

fn arr4(v: &Value) -> [i64; 4] {
    ints(v).try_into().unwrap()
}

fn errv<T>(r: &Result<T>) -> Value {
    match r {
        Ok(_) => Value::Null,
        Err(e) => Value::String(e.to_string()),
    }
}

/// 對應 Go `protect`：panic 時回傳 `None`。
fn catch<R>(f: impl FnOnce() -> R) -> Option<R> {
    let previous = std::panic::take_hook();
    std::panic::set_hook(Box::new(|_| {}));
    let r = std::panic::catch_unwind(std::panic::AssertUnwindSafe(f));
    std::panic::set_hook(previous);
    r.ok()
}

/// 執行結果：輸出欄位與（如有）Rust 錯誤，供 err_prefix 比對前綴。
struct Got {
    v: Map<String, Value>,
    err: Option<FdotherError>,
}

impl Got {
    fn new<T>(r: Result<T>, fields: impl FnOnce(&T) -> Vec<(&'static str, Value)>) -> Got {
        let mut v = Map::new();
        let err = match r {
            Ok(x) => {
                for (k, val) in fields(&x) {
                    v.insert(k.into(), val);
                }
                None
            }
            Err(e) => Some(e),
        };
        v.insert("err".into(), err.as_ref().map(|e| Value::String(e.to_string())).unwrap_or(Value::Null));
        Got { v, err }
    }

    fn plain(fields: Vec<(&'static str, Value)>) -> Got {
        let mut v = Map::new();
        for (k, val) in fields {
            v.insert(k.into(), val);
        }
        v.entry("err").or_insert(Value::Null);
        Got { v, err: None }
    }

    /// 可能 panic 的呼叫：`{err, panic, ...}`。`extra` 不論成功與否都加上（例如 panic 前寫入的緩衝）。
    fn protected<T>(
        r: Option<Result<T>>,
        ok: impl FnOnce(&T) -> Vec<(&'static str, Value)>,
        extra: Vec<(&'static str, Value)>,
    ) -> Got {
        let mut got = match r {
            None => Got::plain(vec![("panic", json!(true))]),
            Some(r) => {
                let mut g = Got::new(r, ok);
                g.v.insert("panic".into(), json!(false));
                g
            }
        };
        for (k, val) in extra {
            got.v.insert(k.into(), val);
        }
        got
    }

    fn with(mut self, key: &'static str, val: Value) -> Got {
        self.v.insert(key.into(), val);
        self
    }
}

// ---- 輸入重建 ----

fn lmi(spec: &Value) -> LMI1Entry {
    let a = ints(spec);
    LMI1Entry { width: a[0], height: a[1], pixels: pat(a[2], a[3]) }
}

/// Go `lmiBank`。
fn lmi_bank(spec: &Value) -> Vec<LMI1Entry> {
    let n = i(spec, "n") as usize;
    let mut out: Vec<LMI1Entry> = (0..n).map(|k| LMI1Entry { width: 1, height: 1, pixels: vec![k as u8] }).collect();
    for bad in spec["bad"].as_array().unwrap() {
        let a = ints(bad);
        if (a[0] as usize) < n {
            out[a[0] as usize] = LMI1Entry { width: a[1], height: a[2], pixels: vec![0; a[3] as usize] };
        }
    }
    out
}

fn state(v: &Value) -> ActionOverlayState {
    ActionOverlayState { availability: arr4(&v["avail"]), direction_state: arr4(&v["dir"]) }
}

fn state_out(s: &ActionOverlayState) -> Value {
    json!(s.availability.iter().chain(&s.direction_state).collect::<Vec<_>>())
}

fn radial(v: &Value) -> RadialLUTRemap {
    let a = ints(v);
    RadialLUTRemap {
        center_x: a[0],
        center_y: a[1],
        radius: a[2],
        scale: a[3],
        start_y: a[4],
        end_y: a[5],
        clip_width: a[6],
    }
}

fn rect(v: &Value) -> CenteredRectLUTRemap {
    let a = ints(v);
    CenteredRectLUTRemap { center_x: a[0], horizontal_radius: a[1], start_y: a[2], end_y: a[3], clip_width: a[4] }
}

fn pass_in(v: &Value) -> IndexedTransitionPass {
    IndexedTransitionPass { first_radial: radial(&v[0]), second_radial: radial(&v[1]), final_rect: rect(&v[2]) }
}

fn radial_out(r: &RadialLUTRemap) -> Value {
    json!([r.center_x, r.center_y, r.radius, r.scale, r.start_y, r.end_y, r.clip_width])
}

fn pass_out(p: &IndexedTransitionPass) -> Value {
    let r = p.final_rect;
    json!([
        radial_out(&p.first_radial),
        radial_out(&p.second_radial),
        [r.center_x, r.horizontal_radius, r.start_y, r.end_y, r.clip_width]
    ])
}

fn frame_spec(v: &Value) -> Frame {
    let (x, y, w, h) = (i(v, "x"), i(v, "y"), i(v, "w"), i(v, "h"));
    let sep_k = v["sep_k"].as_i64().unwrap_or(0);
    if sep_k != 0 {
        let n = w.wrapping_mul(h).max(0);
        let mask = pat(n, sep_k + 1).into_iter().map(|m| if m & 1 == 0 { 0 } else { 255 }).collect();
        return Frame::separated(x, y, w, h, pat(n, sep_k), mask);
    }
    Frame::raw(x, y, w, h, v["raw"].as_str().map(unhex).unwrap_or_default())
}

fn frame_digest(f: &Frame) -> Value {
    json!([f.x, f.y, f.width, f.height, digest(&f.pixels), digest(&f.indexed), digest(&f.mask)])
}

fn lmi_digest(l: &LMI1Entry) -> Value {
    json!([l.width, l.height, digest(&l.pixels)])
}

/// Go `bankSpec.bank`。
fn bank_spec(v: &Value) -> Bank {
    let count = i(v, "n") as usize;
    let mut bank = Bank { sprites: vec![Sprite::default(); count] };
    let n = NATIVE_SIZE * NATIVE_SIZE;
    for sp in v["sprites"].as_array().map(Vec::as_slice).unwrap_or(&[]) {
        let (index, p) = (i(sp, "i") as usize, i(sp, "p"));
        if index >= count {
            continue;
        }
        bank.sprites[index] = match s(sp, "kind") {
            "solid" => {
                Sprite { pixels: vec![p as u8; n as usize], mask: vec![1; n as usize], remap_mask: vec![0; n as usize] }
            }
            "pattern" => Sprite {
                pixels: pat(n, p),
                mask: pat(n, p + 3).into_iter().map(|m| m & 1).collect(),
                remap_mask: vec![0; n as usize],
            },
            "nomask" => Sprite { pixels: pat(n, p), ..Default::default() },
            _ => Sprite::default(),
        };
    }
    bank
}

fn bank_digest(bank: &Bank) -> Value {
    let mut h = Sha256::new();
    for sp in &bank.sprites {
        for part in [&sp.pixels, &sp.mask, &sp.remap_mask] {
            h.update(format!("{}:", part.len()).as_bytes());
            h.update(part);
        }
    }
    json!([bank.sprites.len(), hex(&h.finalize())])
}

fn assets_digest(a: &NativePreparationAssets) -> Value {
    json!({
        "upper_right": lmi_digest(&a.upper_right),
        "lower": lmi_digest(&a.lower),
        "upper_left": frame_digest(&a.upper_left),
        "quota": a.quota_digits.iter().map(frame_digest).collect::<Vec<_>>(),
        "remaining": a.remaining_digits.iter().map(frame_digest).collect::<Vec<_>>(),
        "cursor": bank_digest(&a.cursor),
        "units": bank_digest(&a.units),
    })
}

fn prep_assets_spec(v: &Value) -> NativePreparationAssets {
    let mut a = NativePreparationAssets {
        upper_right: lmi(&v["upper_right"]),
        lower: lmi(&v["lower"]),
        upper_left: frame_spec(&v["upper_left"]),
        cursor: bank_spec(&v["cursor"]),
        units: bank_spec(&v["units"]),
        ..Default::default()
    };
    for d in 0..10 {
        a.quota_digits[d] = frame_spec(&v["quota"][d]);
        a.remaining_digits[d] = frame_spec(&v["remaining"][d]);
    }
    a
}

/// Go `compose`。
fn compose(a: &NativePreparationAssets, c: &Value) -> Result<Vec<u8>> {
    let bg = pat(i(c, "bg_n"), i(c, "bg_k"));
    let keys = ints(&c["keys"]);
    let selected: Vec<bool> = c["selected"].as_array().unwrap().iter().map(|x| x.as_bool().unwrap()).collect();
    let (cursor, cycle, limit) = (i(c, "cursor"), i(c, "cycle"), i(c, "limit"));
    match s(c, "fn") {
        "compose" => compose_native_preparation_frame(a, &keys, &selected, cursor, cycle, limit),
        "background" => compose_native_preparation_frame_on_background(&bg, a, &keys, &selected, cursor, cycle, limit),
        "without_units" => {
            compose_native_preparation_frame_without_roster_units(&bg, a, &keys, &selected, cursor, cycle, limit)
        }
        _ => compose_native_preparation_frame_without_cursor(
            &bg,
            a,
            &keys,
            &selected,
            cursor,
            cycle,
            limit,
            b(c, "draw_units"),
        ),
    }
}

fn spawn_steps(v: &Value) -> Vec<NativeSpawnIntroStep> {
    v.as_array()
        .unwrap()
        .iter()
        .map(|r| NativeSpawnIntroStep {
            pass: r[0].as_i64().unwrap(),
            lmi_entry: r[1].as_i64().unwrap(),
            sound_resource: r[2].as_i64().unwrap(),
            delay_ticks: r[3].as_i64().unwrap(),
            snapshot_mode: NativeSpawnIntroSnapshotMode(r[4].as_u64().unwrap() as u8),
            redraw_terrain: r[5].as_bool().unwrap(),
            new_unit_pixel_lift: r[6].as_i64().unwrap(),
        })
        .collect()
}

fn spawn_out(steps: &[NativeSpawnIntroStep]) -> Value {
    steps
        .iter()
        .map(|s| {
            json!([
                s.pass,
                s.lmi_entry,
                s.sound_resource,
                s.delay_ticks,
                s.snapshot_mode.0,
                s.redraw_terrain,
                s.new_unit_pixel_lift
            ])
        })
        .collect()
}

fn unit_steps(v: &Value) -> Vec<UnitPresentStep> {
    v.as_array()
        .unwrap()
        .iter()
        .map(|r| UnitPresentStep {
            phase: r[0].as_str().unwrap().to_string(),
            resource: r[1].as_i64().unwrap(),
            entry: r[2].as_i64().unwrap(),
            delay_ms: r[3].as_i64().unwrap(),
            delay_ticks: r[4].as_i64().unwrap(),
        })
        .collect()
}

fn unit_steps_out(steps: &[UnitPresentStep]) -> Value {
    steps.iter().map(|s| json!([s.phase, s.resource, s.entry, s.delay_ms, s.delay_ticks])).collect()
}

fn lut_frame_out(f: &UnitPresentLUTFrame) -> Value {
    json!([f.phase, f.lut_index, pass_out(&f.pass), f.delay_ms, f.delay_ticks])
}

fn lut_frame_in(v: &Value) -> UnitPresentLUTFrame {
    UnitPresentLUTFrame {
        phase: v[0].as_str().unwrap().to_string(),
        lut_index: v[1].as_i64().unwrap(),
        pass: pass_in(&v[2]),
        delay_ms: v[3].as_i64().unwrap(),
        delay_ticks: v[4].as_i64().unwrap(),
    }
}

fn synthetic(message: &str) -> FdotherError {
    FdotherError::Contract(message.to_string())
}

struct Env {
    flame2: Option<PathBuf>,
    assets: Option<PathBuf>,
    tmp: PathBuf,
    real_cache: BTreeMap<String, Vec<LMI1Entry>>,
    prep: BTreeMap<String, NativePreparationAssets>,
}

impl Env {
    fn fdother(&self) -> Option<PathBuf> {
        self.flame2.as_ref().map(|d| d.join("FDOTHER.DAT"))
    }

    fn real_bank(&mut self, resource: i64) -> Option<Vec<LMI1Entry>> {
        let key = format!("fdother#{resource}");
        if !self.real_cache.contains_key(&key) {
            let entries = decode_lmi1_resource(self.fdother()?, resource).unwrap();
            self.real_cache.insert(key.clone(), entries);
        }
        self.real_cache.get(&key).cloned()
    }

    fn prep_assets(&mut self, source: &str) -> Option<&NativePreparationAssets> {
        if !self.prep.contains_key(source) {
            let flame2 = self.flame2.clone()?;
            let assets = match source {
                "archive" => {
                    decode_native_preparation_assets_archive(flame2.join("FDOTHER.DAT"), flame2.join("FDICON.B24"))
                }
                "archive_default" => decode_native_preparation_assets_archive(flame2.join("FDOTHER.DAT"), ""),
                _ => load_native_preparation_assets(self.assets.as_ref()?),
            };
            self.prep.insert(source.to_string(), assets.unwrap());
        }
        self.prep.get(source)
    }

    fn scrub(&self, text: &str) -> String {
        let mut t = text.replace(self.tmp.to_str().unwrap(), "$TMP");
        if let Some(a) = &self.assets {
            t = t.replace(a.to_str().unwrap(), "$REAL");
        }
        if let Some(f) = &self.flame2 {
            t = t.replace(f.to_str().unwrap(), "$FLAME2");
        }
        t
    }
}

fn cell_pair(r: Result<i64>) -> Value {
    match r {
        Ok(v) => json!([v, null]),
        Err(e) => json!([0, e.to_string()]),
    }
}

type RedrawBox<'a> = Box<dyn FnMut(&mut [u8]) -> Result<()> + 'a>;

fn redraw_script<'a>(script: &'a Value, calls: &'a Cell<i64>) -> Option<RedrawBox<'a>> {
    let (pos, val) = (i(script, "pos"), i(script, "val"));
    match s(script, "mode") {
        "nil" => None,
        "ok" => Some(Box::new(move |_| {
            calls.set(calls.get() + 1);
            Ok(())
        })),
        "err" => Some(Box::new(move |_| {
            calls.set(calls.get() + 1);
            Err(synthetic("synthetic redraw failure"))
        })),
        _ => Some(Box::new(move |buf: &mut [u8]| {
            calls.set(calls.get() + 1);
            if (pos as usize) < buf.len() {
                buf[pos as usize] = val as u8;
            }
            Ok(())
        })),
    }
}

fn unit_intro(entries: &[LMI1Entry], v: &Value) -> Got {
    let mut dst = pat(i(v, "n"), 21);
    let a = ints(&v["a"]);
    let script = &v["tick"];
    let (mode, at, pos, val) = (s(script, "mode"), i(script, "at"), i(script, "pos"), i(script, "val"));
    let calls = Cell::new(0i64);
    let mut tick = |buf: &mut [u8]| {
        calls.set(calls.get() + 1);
        match mode {
            "err" if calls.get() - 1 == at => Err(synthetic("synthetic present failure")),
            "mutate" => {
                if (pos as usize) < buf.len() {
                    buf[pos as usize] = (val + calls.get()) as u8;
                }
                Ok(())
            }
            _ => Ok(()),
        }
    };
    let hook: Option<NativeUnitPresentTick> = if mode == "nil" { None } else { Some(&mut tick) };
    let r = catch(|| run_native_unit_present_lmi_intro(entries, &mut dst, a[0], a[1], a[2], a[3], hook));
    let calls = calls.get();
    Got::protected(r, |_| Vec::new(), vec![("calls", json!(calls)), ("dst", json!(digest(&dst)))])
}

fn unit_lut_frame(v: &Value) -> Got {
    let frame = lut_frame_in(&v["frame"]);
    let mut work = pat(i(v, "work_n"), i(v, "work_k"));
    let snap = pat(i(v, "snap_n"), i(v, "snap_k"));
    let lut = pat(i(v, "lut_n"), i(v, "lut_k"));
    let calls = RefCell::new(Vec::<String>::new());
    let hook = |script: &Value, name: &'static str| {
        let (mode, pos, val) = (s(script, "mode").to_string(), i(script, "pos"), i(script, "val"));
        let calls = &calls;
        let mut n = 0i64;
        move |buf: &mut [u8]| -> Result<()> {
            n += 1;
            calls.borrow_mut().push(format!("{name}:{}:{}", buf.len(), buf[0]));
            if mode == "err" {
                return Err(synthetic(&format!("synthetic {name} failure")));
            }
            if mode == "mutate" && (pos as usize) < buf.len() {
                buf[pos as usize] = (val + n) as u8;
            }
            Ok(())
        }
    };
    let mut redraw = hook(&v["redraw"], "redraw");
    let mut present_inner = hook(&v["present"], "present");
    let mut present = |buf: &mut [u8], fr: &UnitPresentLUTFrame| {
        calls.borrow_mut().push(format!("frame:{}:{}", fr.phase, fr.lut_index));
        present_inner(buf)
    };
    let redraw_hook: Option<NativeUnitPresentRedraw> =
        if s(&v["redraw"], "mode") == "nil" { None } else { Some(&mut redraw) };
    let present_hook: Option<NativeUnitPresentPresent> =
        if s(&v["present"], "mode") == "nil" { None } else { Some(&mut present) };
    let r = catch(|| run_native_unit_present_lut_frame(&mut work, &snap, &lut, &frame, redraw_hook, present_hook));
    let calls = calls.into_inner();
    Got::protected(r, |_| Vec::new(), vec![("calls", json!(calls)), ("work", json!(digest(&work)))])
}

fn ch23_loop(v: &Value) -> Got {
    let spec = NativeCh23LoopSpec {
        phase: s(v, "phase").to_string(),
        repeat: i(v, "repeat"),
        stage_values: ints(&v["stages"]),
        palette: b(v, "palette"),
    };
    let mut staging = pat(i(v, "n"), 17);
    let mut dac = pat(i(v, "dac_n"), 19);
    let hooks_in = &v["hooks"];
    let nil: Vec<bool> = hooks_in["nil"].as_array().unwrap().iter().map(|x| x.as_bool().unwrap()).collect();
    let (fail_hook, fail_at, write) = (i(hooks_in, "fail_hook"), i(hooks_in, "fail_at"), b(hooks_in, "write"));
    let calls = RefCell::new(Vec::<String>::new());
    let count = RefCell::new([0i64; 4]);
    let r = {
        let st = Cell::from_mut(staging.as_mut_slice()).as_slice_of_cells();
        let dc = Cell::from_mut(dac.as_mut_slice()).as_slice_of_cells();
        let fail = |hook: usize| -> Result<()> {
            let mut c = count.borrow_mut();
            c[hook] += 1;
            if fail_hook == hook as i64 && c[hook] - 1 == fail_at {
                return Err(synthetic(&format!("synthetic hook {hook} failure")));
            }
            Ok(())
        };
        let mut hooks = NativeCh23LoopHooks {
            latch: Some(Box::new(|stage| {
                calls.borrow_mut().push(format!("L{stage}"));
                if write && stage < st.len() as i64 {
                    st[stage as usize].set((stage + 100) as u8);
                    if dc.len() as i64 > stage {
                        dc[stage as usize].set((stage + 50) as u8);
                    }
                }
                fail(0)
            })),
            palette: Some(Box::new(|esi| {
                calls.borrow_mut().push(format!("P{esi}"));
                fail(1)
            })),
            draw: Some(Box::new(|| {
                calls.borrow_mut().push("D".to_string());
                fail(2)
            })),
            tick: Some(Box::new(|| {
                calls.borrow_mut().push("T".to_string());
                fail(3)
            })),
        };
        if nil[0] {
            hooks.latch = None;
        }
        if nil[1] {
            hooks.palette = None;
        }
        if nil[2] {
            hooks.draw = None;
        }
        if nil[3] {
            hooks.tick = None;
        }
        run_native_ch23_loop(&spec, st, dc, &mut hooks)
    };
    let calls = calls.into_inner();
    Got::new(r, |_| Vec::new())
        .with("recovered", json!(spec.is_recovered_contract()))
        .with("calls_n", json!(calls.len()))
        .with("calls", json!(digest(calls.join(",").as_bytes())))
        .with("head", json!(calls[..calls.len().min(12)]))
        .with("staging", json!(digest(&staging)))
        .with("dac", json!(digest(&dac)))
}

fn type21_out(s: &NativeAIItemType21TailSchedule) -> Vec<(&'static str, Value)> {
    vec![
        (
            "s",
            json!([
                s.command_id,
                s.effect_resource,
                s.effect_start,
                s.effect_frames,
                s.sound_resource,
                s.effect_sample,
                s.effect_delay_ticks,
                s.toggle_a,
                s.toggle_b,
                s.toggle_pairs,
                s.toggle_delay_ms,
                s.digit_resource,
                s.digit_bias,
                s.digit_frames,
                s.digit_delay_ms,
                s.digit_hold_ms
            ]),
        ),
        ("v", json!(s.digit_vertical)),
    ]
}

fn command32_out(s: &NativeCommand32TailSchedule) -> Vec<(&'static str, Value)> {
    vec![
        (
            "s",
            json!([
                s.effect_resource,
                s.effect_start,
                s.effect_frames,
                s.sound_resource,
                s.effect_sample,
                s.effect_delay_ticks,
                s.toggle_a,
                s.toggle_b,
                s.toggle_pairs,
                s.toggle_delay_ms,
                s.digit_resource,
                s.digit_bias,
                s.digit_frames,
                s.digit_delay_ms,
                s.digit_hold_ms
            ]),
        ),
        ("v", json!(s.digit_vertical)),
    ]
}

fn command9_out(s: &NativeCommand9PlayerSchedule) -> Vec<(&'static str, Value)> {
    vec![
        (
            "s",
            json!([
                s.effect_resource,
                s.effect_start,
                s.effect_frames,
                s.sound_resource,
                s.initial_sample,
                s.repeat_sample,
                s.repeat_sample_frames[0],
                s.repeat_sample_frames[1],
                s.effect_frame_delay_ticks,
                s.result_resource,
                s.result_digit_bias,
                s.result_miss_descriptors[0],
                s.result_miss_descriptors[1],
                s.result_miss_descriptors[2],
                s.result_miss_descriptors[3],
                s.result_frames,
                s.result_delay_ms,
                s.result_hold_ms
            ]),
        ),
        ("v", json!(s.result_vertical)),
    ]
}

/// Go 在錯誤時回傳零值結構，欄位照樣輸出；這裡在錯誤時以 `Default` 補上。
fn zero_on_err<T: Default>(r: Result<T>, out: impl Fn(&T) -> Vec<(&'static str, Value)>) -> Got {
    let fields = match &r {
        Ok(x) => out(x),
        Err(_) => out(&T::default()),
    };
    Got::new(r, |_| Vec::new()).extend(fields)
}

impl Got {
    fn extend(mut self, fields: Vec<(&'static str, Value)>) -> Got {
        for (k, v) in fields {
            self.v.insert(k.into(), v);
        }
        self
    }
}

fn run(env: &mut Env, kind: &str, op: &str, v: &Value) -> Option<Got> {
    let got = match op {
        // ---- action_overlay ----
        "overlay_state" => {
            let st = state(v);
            let cells: Vec<Value> = (-1..=4).map(|d| cell_pair(st.cell_index(d))).collect();
            let mut selected = Vec::new();
            for d in -1..=4 {
                for phase in -1..=2 {
                    selected.push(cell_pair(st.selected_cell_index(d, phase)));
                }
            }
            Got::plain(vec![("cells", json!(cells)), ("selected", json!(selected))])
        }
        "overlay_ctor" => {
            let st = match s(v, "ctor") {
                "battle" => battle_action_overlay_state(arr4(&v["avail"])),
                "nested" => native_nested_system_action_overlay_state(b(v, "gate"), b(v, "absent")),
                _ => native_continue_action_overlay_state(),
            };
            Got::plain(vec![("state", state_out(&st))])
        }
        "overlay_initial" => {
            let avail = arr4(&v["avail"]);
            let accepts: Vec<bool> = (-2..=5).map(|d| action_overlay_accepts_direction(avail, d)).collect();
            Got::plain(vec![("initial", json!(action_overlay_initial_direction(avail))), ("accepts", json!(accepts))])
        }
        "system_options" => {
            let raw = ints(&v["raw"]);
            let o = NativeSystemOptions {
                raw_53af9: raw[0] as u8,
                raw_51aab: raw[1] as u8,
                raw_51e61: raw[2] as u8,
                raw_51e62: raw[3] as u8,
            };
            let st = match o.action_overlay_state() {
                Ok(st) => json!([state_out(&st), null]),
                Err(e) => json!([state_out(&ActionOverlayState::default()), e.to_string()]),
            };
            let toggles: Vec<Value> = (-1..=4)
                .map(|sel| {
                    let (t, err) = match o.toggle(sel) {
                        Ok(t) => (t, Value::Null),
                        Err(e) => (o, json!(e.to_string())),
                    };
                    json!([[t.raw_53af9, t.raw_51aab, t.raw_51e61, t.raw_51e62], err])
                })
                .collect();
            Got::plain(vec![
                ("validate", errv(&o.validate())),
                ("flags", json!([o.music_enabled(), o.sfx_enabled(), o.full_presentation_enabled(), o.hud_enabled()])),
                ("state", st),
                ("toggles", json!(toggles)),
            ])
        }
        "system_default" => {
            let d = default_native_system_options();
            Got::plain(vec![("raw", json!([d.raw_53af9, d.raw_51aab, d.raw_51e61, d.raw_51e62]))])
        }
        "overlay_offsets" => {
            zero_on_err(action_overlay_frame_offsets(i(v, "frame"), b(v, "closing")), |o| vec![("offsets", json!(o))])
        }
        "overlay_origin" => {
            let (col, row) = (i(v, "col"), i(v, "row"));
            Got::plain(vec![
                ("origin", cell_pair(action_overlay_origin(col, row))),
                ("snapshot", cell_pair(action_overlay_snapshot_origin(col, row))),
            ])
        }
        "snapshot_capture" => {
            let src = pat(i(v, "n"), i(v, "k"));
            let r = catch(|| capture_action_overlay_snapshot(&src, i(v, "stride"), i(v, "x"), i(v, "y")));
            Got::protected(r, |snap| vec![("snapshot", json!(digest(snap)))], Vec::new())
        }
        "snapshot_restore" => {
            let mut dst = pat(i(v, "n"), i(v, "k"));
            let snap = pat(i(v, "sn"), i(v, "sk"));
            let r = catch(|| restore_action_overlay_snapshot(&mut dst, &snap, i(v, "stride"), i(v, "x"), i(v, "y")));
            Got::protected(r, |_| Vec::new(), vec![("dst", json!(digest(&dst)))])
        }
        "overlay_blit" => {
            let cells: Vec<RawCell> = v["cells"]
                .as_array()
                .unwrap()
                .iter()
                .map(|c| {
                    let a = ints(c);
                    RawCell { width: a[0], height: a[1], pixels: pat(a[2], a[3]) }
                })
                .collect();
            let mut dst = pat(i(v, "n"), i(v, "k"));
            let r = catch(|| {
                blit_action_overlay_frame(
                    &cells,
                    state(v),
                    &mut dst,
                    i(v, "stride"),
                    i(v, "origin"),
                    i(v, "frame"),
                    b(v, "closing"),
                )
            });
            Got::protected(r, |_| Vec::new(), vec![("dst", json!(digest(&dst)))])
        }
        "blink" => {
            let start = ints(&v["start"]);
            let mut blink = ActionOverlaySelectionBlink { phase: start[0], tick: start[1] };
            let states: Vec<Value> = ints(&v["ticks"])
                .into_iter()
                .map(|t| {
                    blink.advance(t);
                    json!([blink.phase, blink.tick])
                })
                .collect();
            Got::plain(vec![("states", json!(states))])
        }
        // ---- ch28_post_present ----
        "ch28_frames" => {
            let frames: Vec<Value> =
                native_ch28_post_overlay_frames().iter().map(|f| json!([f.phase, f.entry])).collect();
            Got::plain(vec![("frames", json!(frames))])
        }
        "ch28_origin" => {
            zero_on_err(native_ch28_post_overlay_origin(i(v, "x"), i(v, "y"), i(v, "cx"), i(v, "cy")), |o| {
                vec![("origin", json!(o))]
            })
        }
        "ch28_blit" => {
            let mut work = pat(i(v, "n"), 5);
            let entry = lmi(&v["entry"]);
            let r = catch(|| blit_native_ch28_post_overlay(&entry, &mut work, i(v, "origin")));
            Got::protected(r, |_| Vec::new(), vec![("work", json!(digest(&work)))])
        }
        "ch28_validate" => {
            let entries: Vec<LMI1Entry> = v["entries"].as_array().unwrap().iter().map(lmi).collect();
            Got::new(validate_native_ch28_post_lmi(&entries), |_| Vec::new())
        }
        // ---- indexed_transition ----
        "transition_pass" => {
            let mut dst = pat(i(v, "n"), i(v, "k"));
            let lut = pat(i(v, "lut_n"), i(v, "lut_k"));
            let calls = Cell::new(0i64);
            let mut redraw = redraw_script(&v["redraw"], &calls);
            let r = catch(|| {
                apply_indexed_transition_pass(
                    &mut dst,
                    i(v, "stride"),
                    &lut,
                    pass_in(&v["pass"]),
                    redraw.as_mut().map(|f| f.as_mut() as IndexedTransitionRedraw),
                )
            });
            drop(redraw);
            Got::protected(r, |_| Vec::new(), vec![("dst", json!(hex(&dst))), ("calls", json!(calls.get()))])
        }
        "transition_build" => {
            let a = ints(&v["args"]);
            Got::new(build_native_indexed_transition_pass(a[0], a[1], a[2], a[3], a[4]), |p| {
                vec![("pass", pass_out(p))]
            })
        }
        // ---- 呈現排程 builder ----
        "type21" => zero_on_err(
            build_native_ai_item_type21_tail_schedule(i(v, "id"), &lmi_bank(&v["effect"]), &lmi_bank(&v["digits"])),
            type21_out,
        ),
        "command32" => zero_on_err(
            build_native_command32_tail_schedule(&lmi_bank(&v["effect"]), &lmi_bank(&v["digits"])),
            command32_out,
        ),
        "command9" => zero_on_err(
            build_native_command9_player_schedule(&lmi_bank(&v["effect"]), &lmi_bank(&v["result"])),
            command9_out,
        ),
        "command2022" => Got::new(build_native_command2022_tail_schedule(), |rows| {
            let out: Vec<Value> = rows
                .iter()
                .map(|r| {
                    json!([
                        r.command_id,
                        r.effect_start,
                        r.effect_frames,
                        r.effect_sample,
                        r.extra_sample_frame_indices,
                        r.mask_index,
                        r.mask_pairs,
                        r.digit_bias,
                        r.digit_frames,
                        r.digit_hold_milliseconds,
                        r.digit_vertical
                    ])
                })
                .collect();
            vec![("rows", json!(out))]
        }),
        "command33" => Got::new(build_native_command33_tail_schedule(), |h| {
            vec![
                (
                    "s",
                    json!([
                        h.effect_resource,
                        h.effect_start,
                        h.effect_frames,
                        h.effect_sample,
                        h.effect_frame_delay_ticks,
                        h.mask_sample,
                        h.mask_index,
                        h.mask_pairs,
                        h.digit_resource,
                        h.digit_bias,
                        h.digit_frames,
                        h.digit_frame_delay_ms,
                        h.digit_hold_ms
                    ]),
                ),
                ("v", json!(h.digit_vertical)),
            ]
        }),
        "command35" => Got::new(build_native_command35_tail_schedule(), |rows| {
            let out: Vec<Value> = rows
                .iter()
                .map(|r| {
                    json!([
                        r.command_id,
                        r.effect_start,
                        r.effect_frames,
                        r.effect_sample,
                        r.extra_sample_frame_indices,
                        r.extra_sample,
                        r.mask_index,
                        r.mask_pairs,
                        r.digit_bias,
                        r.digit_frames,
                        r.digit_hold_milliseconds,
                        r.digit_vertical
                    ])
                })
                .collect();
            vec![("rows", json!(out))]
        }),
        "pending_code1" => {
            let plan: Vec<Value> = native_pending_code1_presentation_plan()
                .iter()
                .map(|s| json!([s.kind.as_str(), s.frame, s.count, s.duration_ms]))
                .collect();
            Got::plain(vec![("plan", json!(plan))])
        }
        // ---- spawn_intro ----
        "spawn_schedule" => Got::plain(vec![("steps", spawn_out(&native_spawn_intro_schedule()))]),
        "spawn_validate" => Got::new(validate_native_spawn_intro_schedule(&spawn_steps(&v["steps"])), |_| Vec::new()),
        "spawn_visible" => {
            let a = ints(&v["a"]);
            Got::plain(vec![("v", json!(native_spawn_intro_visible(a[0], a[1], a[2], a[3], a[4], a[5])))])
        }
        "spawn_origin" => {
            let a = ints(&v["a"]);
            Got::plain(vec![("o", json!(native_spawn_intro_lmi_origin(a[0], a[1], a[2], a[3])))])
        }
        "spawn_blit" => {
            let a = ints(&v["a"]);
            let mut work = pat(i(v, "n"), 9);
            let entry = lmi(&v["entry"]);
            let r = catch(|| blit_native_spawn_intro_lmi(&entry, &mut work, a[0], a[1], a[2], a[3]));
            Got::protected(r, |_| Vec::new(), vec![("work", json!(digest(&work)))])
        }
        // ---- unit_present_schedule ----
        "unit_schedule" => Got::plain(vec![("steps", unit_steps_out(&native_unit_present_schedule()))]),
        "unit_validate" => Got::new(validate_native_unit_present_schedule(&unit_steps(&v["steps"])), |_| Vec::new()),
        "unit_origin" => {
            let a = ints(&v["a"]);
            Got::plain(vec![("o", json!(native_unit_present_byte_origin(a[0], a[1], a[2], a[3])))])
        }
        "unit_start_y" => {
            zero_on_err(native_unit_present_contract_start_y(i(v, "raw"), i(v, "lut")), |x| vec![("v", json!(x))])
        }
        "unit_lut_pass" => {
            let a = ints(&v["a"]);
            Got::new(native_unit_present_lut_pass(a[0], a[1], a[2]), |p| vec![("pass", pass_out(p))])
        }
        "unit_lut_frames" => {
            let r = native_unit_present_lut_frames(i(v, "a"), i(v, "b"));
            let frames: Vec<Value> = r.as_ref().map(|f| f.iter().map(lut_frame_out).collect()).unwrap_or_default();
            Got::new(r, |_| Vec::new()).with("frames", json!(frames))
        }
        "unit_plan" => {
            let a = ints(&v["a"]);
            zero_on_err(plan_native_unit_present_call(a[0], a[1], a[2], a[3], a[4]), |c| {
                vec![("call", json!([c.unit_slot, c.new_x, c.new_y, c.visual_x, c.visual_y]))]
            })
        }
        "unit_blit" => {
            let a = ints(&v["a"]);
            let mut dst = pat(i(v, "n"), 13);
            let entry = lmi(&v["entry"]);
            let r = catch(|| blit_native_unit_present_lmi(&entry, &mut dst, a[0], a[1], a[2], a[3]));
            Got::protected(r, |_| Vec::new(), vec![("dst", json!(digest(&dst)))])
        }
        "unit_lmi_intro" => {
            if v.get("real").is_some() {
                let entries = env.real_bank(6)?;
                unit_intro(&entries, v)
            } else {
                let entries: Vec<LMI1Entry> = v["entries"].as_array().unwrap().iter().map(lmi).collect();
                unit_intro(&entries, v)
            }
        }
        "unit_lut_frame" => unit_lut_frame(v),
        // ---- native_ch23 ----
        "ch23_rotate" => {
            let mut buf = pat(i(v, "n"), 23);
            let r = rotate_native_ch23_rows(&mut buf, i(v, "latch"));
            Got::new(r, |_| Vec::new()).with("buf", json!(digest(&buf)))
        }
        "ch23_dac_apply" => {
            let mut dac = pat(i(v, "n"), 29);
            let r = apply_native_dac_palette_cycle_e0ef(&mut dac, i(v, "phase"));
            Got::new(r, |_| Vec::new()).with("dac", json!(digest(&dac)))
        }
        "ch23_dac_advance" => {
            let mut dac = pat(i(v, "n"), 31);
            let (phase, last) = (i(v, "phase"), i(v, "last"));
            let r = advance_native_dac_palette_cycle_e0ef(&mut dac, phase, last, i(v, "now"));
            let (p, t, adv) = *r.as_ref().unwrap_or(&(phase, last, false));
            Got::new(r, |_| Vec::new())
                .with("phase", json!(p))
                .with("tick", json!(t))
                .with("advanced", json!(adv))
                .with("dac", json!(digest(&dac)))
        }
        "ch23_loop" => ch23_loop(v),
        "ch23_blit" => {
            let frame = frame_spec(&v["frame"]);
            let mut staging = pat(i(v, "n"), i(v, "k"));
            let r = catch(|| blit_native_ch23_stage(&frame, &mut staging));
            Got::protected(r, |_| Vec::new(), vec![("staging", json!(digest(&staging)))])
        }
        "ch23_decode_real" => {
            let path = env.fdother()?;
            Got::new(decode_native_ch23_stage(path), |frame| {
                let mut staging = pat(312 * 192, 3);
                let blit = blit_native_ch23_stage(frame, &mut staging);
                let layers = match frame.indexed_layers() {
                    Ok((a, m)) => json!([null, digest(&a), digest(&m)]),
                    Err(e) => json!([e.to_string(), digest(&[]), digest(&[])]),
                };
                vec![
                    ("frame", frame_digest(frame)),
                    ("blit", json!([errv(&blit), digest(&staging)])),
                    ("layers", layers),
                ]
            })
        }
        "ch23_load_sep" => {
            let root = if s(v, "tree") == "real" {
                env.assets.as_ref()?.join("surfaces")
            } else {
                let root = env.tmp.join(s(v, "tree"));
                let dir = root.join("FDOTHER_042");
                std::fs::create_dir_all(&dir).unwrap();
                for (name, data) in v["files"].as_object().unwrap() {
                    std::fs::write(dir.join(name), unhex(data.as_str().unwrap())).unwrap();
                }
                root
            };
            Got::new(load_separated_native_ch23_stage(&root), |f| vec![("frame", frame_digest(f))])
        }
        "spawn_decode_real" => {
            let path = env.fdother()?;
            let r = decode_native_spawn_intro_frames(path);
            let entries: Vec<Value> = r.as_ref().map(|e| e.iter().map(lmi_digest).collect()).unwrap_or_default();
            Got::new(r, |_| Vec::new()).with("entries", json!(entries))
        }
        "type21_real" => {
            let (r5, r6) = (env.real_bank(5)?, env.real_bank(6)?);
            zero_on_err(build_native_ai_item_type21_tail_schedule(i(v, "id"), &r6, &r5), |s| {
                vec![
                    ("start", json!(s.effect_start)),
                    ("frames", json!(s.effect_frames)),
                    ("sample", json!(s.effect_sample)),
                ]
            })
        }
        "command32_real" => {
            let (r5, r6) = (env.real_bank(5)?, env.real_bank(6)?);
            zero_on_err(build_native_command32_tail_schedule(&r6, &r5), |s| {
                vec![("start", json!(s.effect_start)), ("bias", json!(s.digit_bias))]
            })
        }
        "command9_real" => {
            let (r5, r6) = (env.real_bank(5)?, env.real_bank(6)?);
            zero_on_err(build_native_command9_player_schedule(&r6, &r5), |s| {
                vec![("start", json!(s.effect_start)), ("bias", json!(s.result_digit_bias))]
            })
        }
        "ch28_validate_real" => Got::new(validate_native_ch28_post_lmi(&env.real_bank(5)?), |_| Vec::new()),
        "ch28_blit_real" => {
            let r5 = env.real_bank(5)?;
            let a = ints(&v["a"]);
            let mut work = pat(NATIVE_CH28_POST_WORK_SIZE, 37);
            let origin = native_ch28_post_overlay_origin(a[0], a[1], a[2], a[3]).unwrap();
            let errs: Vec<Value> = native_ch28_post_overlay_frames()
                .iter()
                .map(|f| errv(&blit_native_ch28_post_overlay(&r5[f.entry as usize], &mut work, origin)))
                .collect();
            Got::plain(vec![("errs", json!(errs)), ("work", json!(digest(&work)))])
        }
        "spawn_blit_real" => {
            let entries = decode_native_spawn_intro_frames(env.fdother()?).unwrap();
            let a = ints(&v["a"]);
            let mut work = pat(0x25680, 43);
            let errs: Vec<Value> = native_spawn_intro_schedule()
                .iter()
                .map(|step| {
                    errv(&blit_native_spawn_intro_lmi(
                        &entries[step.lmi_entry as usize],
                        &mut work,
                        a[0],
                        a[1],
                        a[2],
                        a[3],
                    ))
                })
                .collect();
            Got::plain(vec![("errs", json!(errs)), ("work", json!(digest(&work)))])
        }
        "prep_assets_real" => {
            let a = env.prep_assets(s(v, "source"))?;
            let Value::Object(fields) = assets_digest(a) else { unreachable!() };
            let mut got = Got::plain(Vec::new());
            got.v.extend(fields);
            got
        }
        "prep_compose_real" => {
            let a = env.prep_assets(s(v, "source"))?;
            Got::new(compose(a, &v["args"]), |f| vec![("frame", json!(digest(f)))])
        }
        "prep_load_missing" => Got::new(load_native_preparation_assets(env.tmp.join(s(v, "tree"))), |_| Vec::new()),
        // ---- preparation_roster ----
        "prep_move" => {
            let (count, scan) = (i(v, "count"), i(v, "scan") as u8);
            let cursors: Vec<i64> = match v.get("cursor") {
                Some(c) => vec![c.as_i64().unwrap()],
                None => (-1..=31).collect(),
            };
            let out: Vec<Value> =
                cursors.into_iter().map(|c| cell_pair(move_native_preparation_roster_cursor(c, count, scan))).collect();
            Got::plain(vec![("out", json!(out))])
        }
        "prep_position" => {
            let index = i(v, "index");
            let flags: Vec<(bool, bool)> = if kind == "edge-extreme" {
                vec![(true, true)]
            } else {
                vec![(false, false), (false, true), (true, false), (true, true)]
            };
            let out: Vec<Value> = flags
                .into_iter()
                .map(|(sel, cur)| match native_preparation_roster_position(index, sel, cur) {
                    Ok((x, y)) => json!([x, y, null]),
                    Err(e) => json!([0, 0, e.to_string()]),
                })
                .collect();
            Got::plain(vec![("out", json!(out))])
        }
        "prep_roster" => {
            let (units, cursor_bank) = (bank_spec(&v["units"]), bank_spec(&v["cursor_bank"]));
            let keys = ints(&v["keys"]);
            let selected: Vec<bool> = v["selected"].as_array().unwrap().iter().map(|x| x.as_bool().unwrap()).collect();
            let stride = i(v, "stride");
            let mut dst = pat(stride * i(v, "rows"), 41);
            let r = catch(|| {
                blit_native_preparation_roster(
                    &mut dst,
                    stride,
                    &units,
                    &cursor_bank,
                    &keys,
                    &selected,
                    i(v, "cursor"),
                    i(v, "cycle"),
                )
            });
            Got::protected(r, |_| Vec::new(), vec![("dst", json!(digest(&dst)))])
        }
        "prep_compose" => {
            let assets = prep_assets_spec(&v["assets"]);
            let r = catch(|| compose(&assets, &v["args"]));
            Got::protected(r, |f| vec![("frame", json!(digest(f)))], Vec::new())
        }
        other => panic!("未知的 op {other}"),
    };
    Some(got)
}

/// Rust 錯誤中屬於 Go 自己文字的前綴。`FdotherError::go_prefix` 對原樣轉出的 fdicon 錯誤回傳全文；
/// fdicon 包了讀檔或 JSON 錯誤的變體（Go `%w`）在這裡拆出前綴，內層是 Rust 函式庫的文字，不比。
fn rust_prefix(e: &FdotherError) -> String {
    match e {
        FdotherError::Fdicon(FdiconError::BankMetadata(detail) | FdiconError::FdshapMetadata(detail)) => {
            e.to_string().strip_suffix(detail.as_str()).unwrap().to_string()
        }
        FdotherError::Fdicon(FdiconError::Io(_)) => String::new(),
        _ => e.go_prefix(),
    }
}

/// 回傳第一個不同的 JSON 路徑，方便定位。
fn first_diff(path: &str, want: &Value, got: &Value) -> Option<String> {
    match (want, got) {
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

/// 覆蓋統計用：錯誤訊息的數字換成 N；panic 與接受另列，部分 op 再加分支細節。
fn classify(op: &str, result: &Value) -> String {
    if result["panic"] == json!(true) {
        return "panic".to_string();
    }
    if let Some(e) = result["err"].as_str() {
        // 數字與 0x 開頭的十六進位都換成 N（負號併入），同一種錯誤不因數值不同而分列。
        let chars: Vec<char> = e.chars().collect();
        let mut out = String::new();
        let mut k = 0;
        while k < chars.len() {
            let mut j = k;
            if chars[j] == '-' && chars.get(j + 1).is_some_and(char::is_ascii_digit) {
                j += 1;
            }
            if !chars[j].is_ascii_digit() {
                out.push(chars[k]);
                k += 1;
                continue;
            }
            if chars[j] == '0' && chars.get(j + 1) == Some(&'x') {
                j += 2;
                while chars.get(j).is_some_and(char::is_ascii_hexdigit) {
                    j += 1;
                }
            } else {
                while chars.get(j).is_some_and(char::is_ascii_digit) {
                    j += 1;
                }
            }
            out.push('N');
            k = j;
        }
        return out.chars().take(100).collect();
    }
    match op {
        "ch23_dac_advance" => format!("ok advanced={}", result["advanced"]),
        "ch23_loop" => format!("ok recovered={}", result["recovered"]),
        "spawn_visible" => format!("v={}", result["v"]),
        "overlay_origin" | "overlay_state" | "system_options" | "prep_move" | "prep_position" => {
            let text = result.to_string();
            format!("ok errors={}", text.matches("fdother:").count().min(1))
        }
        _ => "ok".to_string(),
    }
}

fn tree_digest(root: &Path) -> (String, usize) {
    fn walk(root: &Path, dir: &Path, lines: &mut Vec<String>) {
        for entry in std::fs::read_dir(dir).unwrap() {
            let entry = entry.unwrap();
            let ft = entry.file_type().unwrap();
            if ft.is_dir() {
                walk(root, &entry.path(), lines);
            } else if ft.is_file() {
                let rel = entry.path().strip_prefix(root).unwrap().to_str().unwrap().to_string();
                lines.push(format!("{rel}\0{}\n", digest(&std::fs::read(entry.path()).unwrap())));
            }
        }
    }
    let mut lines = Vec::new();
    walk(root, root, &mut lines);
    lines.sort();
    (digest(lines.concat().as_bytes()), lines.len())
}

#[test]
fn matches_frozen_go_golden() {
    let required = std::env::var("FD2_REQUIRE_GOLDEN").is_ok_and(|v| v == "1");
    let Ok(path) = std::env::var("FD2_GOLDEN_FDOTHER_FLOW") else {
        assert!(!required, "FD2_REQUIRE_GOLDEN=1 但沒有 FD2_GOLDEN_FDOTHER_FLOW");
        eprintln!("略過：未設定 FD2_GOLDEN_FDOTHER_FLOW");
        return;
    };
    let doc: Value = serde_json::from_slice(&std::fs::read(&path).expect("讀黃金檔")).unwrap();
    assert_eq!(doc["schema"], "fd2-golden-fdother-flow/1");
    let flame2 = std::env::var_os("FD2_FLAME2_DIR").map(PathBuf::from).filter(|p| p.is_dir());
    let assets = std::env::var_os("FD2_ASSET_PACK").map(PathBuf::from).filter(|p| p.is_dir());
    if required {
        assert!(flame2.is_some(), "FD2_REQUIRE_GOLDEN=1 但沒有 FD2_FLAME2_DIR");
        assert!(assets.is_some(), "FD2_REQUIRE_GOLDEN=1 但沒有 FD2_ASSET_PACK");
    }
    if let Some(dir) = &flame2 {
        for name in ["FDOTHER.DAT", "FDICON.B24"] {
            let rec = &doc["real"][name];
            let data = std::fs::read(dir.join(name)).unwrap();
            assert_eq!(data.len() as u64, rec["size"].as_u64().unwrap(), "{name} 大小不同");
            assert_eq!(digest(&data), rec["sha256"].as_str().unwrap(), "{name} 與黃金檔記錄的雜湊不同");
        }
    }
    if let Some(dir) = &assets {
        for (rel, rec) in doc["real"]["pack"].as_object().unwrap() {
            let (sum, files) = tree_digest(&dir.join(rel));
            assert_eq!(files as u64, rec["files"].as_u64().unwrap(), "{rel} 檔案數不同");
            assert_eq!(sum, rec["tree_sha256"].as_str().unwrap(), "{rel} 檔案清單摘要不同");
        }
    }
    let tmp = std::env::temp_dir().join(format!("fd2-golden-fdother-flow-{}", std::process::id()));
    std::fs::create_dir_all(&tmp).unwrap();
    let mut env = Env { flame2, assets, tmp: tmp.clone(), real_cache: BTreeMap::new(), prep: BTreeMap::new() };

    let cases = doc["cases"].as_array().unwrap();
    assert!(!cases.is_empty());
    let mut failures = Vec::new();
    let mut known = Vec::new();
    let mut skipped = 0;
    let mut compares = BTreeMap::<String, usize>::new();
    let mut kinds = BTreeMap::<String, usize>::new();
    let mut coverage = BTreeMap::<String, BTreeMap<String, usize>>::new();
    for case in cases {
        let index = case["index"].as_u64().unwrap();
        let (kind, op, compare) = (s(case, "kind"), s(case, "op"), s(case, "compare"));
        let Some(got) = run(&mut env, kind, op, &case["input"]) else {
            skipped += 1;
            continue;
        };
        *compares.entry(compare.to_string()).or_default() += 1;
        *kinds.entry(format!("{op}/{kind}")).or_default() += 1;
        let want = &case["result"];
        *coverage.entry(op.to_string()).or_default().entry(classify(op, want)).or_default() += 1;
        let mut got_value = Value::Object(got.v);
        if let Some(e) = got_value["err"].as_str() {
            got_value["err"] = json!(env.scrub(e));
        }
        let mismatch = match compare {
            "err_prefix" => {
                let want_prefix = case["prefix"].as_str().unwrap();
                match &got.err {
                    None => Some(format!("result.err: 接受與否不同 go={} rust=null", want["err"])),
                    Some(e) => {
                        let got_prefix = env.scrub(&rust_prefix(e));
                        if want_prefix != got_prefix {
                            Some(format!("prefix: go={want_prefix:?} rust={got_prefix:?}（rust 錯誤 {e}）"))
                        } else {
                            let (mut a, mut b) = (want.clone(), got_value.clone());
                            a["err"] = Value::Null;
                            b["err"] = Value::Null;
                            first_diff("result", &a, &b)
                        }
                    }
                }
            }
            _ => first_diff("result", want, &got_value),
        };
        if let Some(d) = mismatch {
            if let Some((_, reason)) = KNOWN_DIFFERENCES.iter().find(|(k, _)| kind.ends_with(k)) {
                known.push(format!("case {index} ({kind} {op})：刻意不照做，{reason}；{d}"));
            } else {
                failures.push(format!("case {index} ({kind} {op} {compare}): {d}"));
            }
        }
    }
    let _ = std::fs::remove_dir_all(&tmp);
    eprintln!("黃金案例 {}（略過 {skipped}）：{compares:?}", cases.len());
    for (k, n) in &kinds {
        eprintln!("  {n:5}  {k}");
    }
    eprintln!("已知差異 {} 個", known.len());
    for k in &known {
        eprintln!("  {k}");
    }
    for (op, paths) in &coverage {
        eprintln!("覆蓋 {op}（{} 種結果）:", paths.len());
        for (path, n) in paths {
            eprintln!("  {n:5}  {path}");
        }
    }
    assert!(!required || skipped == 0, "FD2_REQUIRE_GOLDEN=1 但有 {skipped} 個真實檔案例未執行");
    let shown: Vec<&String> = failures.iter().take(60).collect();
    assert!(
        failures.is_empty(),
        "{} 個案例不符：\n{}",
        failures.len(),
        shown.iter().map(|s| s.as_str()).collect::<Vec<_>>().join("\n")
    );
}
