//! 與凍結 Go 版 `remake/cmd/fd2-golden-fdother-native` 的黃金差分（fdother B 組：原版流程邏輯）。
//!
//! 由 `tools/rust_golden_fdother_native.sh` 產生黃金 JSON 並設定 `FD2_GOLDEN_FDOTHER_NATIVE`。設定
//! `FD2_REQUIRE_GOLDEN=1` 時沒有黃金檔即失敗，避免把「沒跑到」當成通過。本組不讀原版檔。
//!
//! 全部案例逐欄比對，錯誤文字必須逐字相同。Go 會因 64 位元繞回騙過範圍檢查而 panic 的輸入，
//! 這裡以 `catch_unwind` 對應，比對 panicked 旗標與 panic 之前已寫入的緩衝區、callback 呼叫紀錄。
//! `phase_execute` 的 callback 由黃金檔中的腳本驅動：每個 callback 依序取用腳本列，用完回到
//! 預設值，規則與 Go 匯出工具的 `scriptCallbacks` 相同。

use fd2_formats::fdother::*;
use serde_json::{Value, json};
use std::cell::{Cell, RefCell};
use std::collections::{BTreeMap, BTreeSet};

/// 本組沒有刻意不照做的 Go 行為；保留清單讓差異政策與其他黃金測試一致。
const KNOWN_DIFFERENCES: &[(&str, &str)] = &[];

fn hex(b: &[u8]) -> String {
    b.iter().map(|x| format!("{x:02x}")).collect()
}

fn unhex(s: &str) -> Vec<u8> {
    (0..s.len()).step_by(2).map(|i| u8::from_str_radix(&s[i..i + 2], 16).unwrap()).collect()
}

fn digest(b: &[u8]) -> String {
    use sha2::{Digest, Sha256};
    hex(&Sha256::digest(b))
}

fn i(v: &Value, key: &str) -> i64 {
    v[key].as_i64().unwrap_or_else(|| panic!("欄位 {key} 不是整數：{}", v[key]))
}

fn u8_(v: &Value, key: &str) -> u8 {
    u8::try_from(i(v, key)).unwrap()
}

fn b(v: &Value, key: &str) -> bool {
    v[key].as_bool().unwrap_or_else(|| panic!("欄位 {key} 不是布林：{}", v[key]))
}

fn bytes_of(v: &Value, key: &str) -> Vec<u8> {
    match &v[key] {
        Value::Null => Vec::new(),
        s => unhex(s.as_str().unwrap()),
    }
}

fn err_value<T>(r: &NativeFlowResult<T>) -> Value {
    match r {
        Ok(_) => Value::Null,
        Err(e) => Value::String(e.to_string()),
    }
}

/// 對應 Go `protect`：panic 時回傳 `None`（Go 的 err 此時是 nil）。
fn catch<R>(f: impl FnOnce() -> R) -> Option<R> {
    let previous = std::panic::take_hook();
    std::panic::set_hook(Box::new(|_| {}));
    let r = std::panic::catch_unwind(std::panic::AssertUnwindSafe(f));
    std::panic::set_hook(previous);
    r.ok()
}

/// 把可能 panic 的呼叫結果整理成 `{err, panicked, ...}`；`ok` 只在成功時呼叫，回傳要加上的欄位。
fn protected<T>(
    r: Option<NativeFlowResult<T>>,
    ok: impl FnOnce(&T) -> Vec<(&'static str, Value)>,
) -> serde_json::Map<String, Value> {
    let mut out = serde_json::Map::new();
    match r {
        None => {
            out.insert("err".into(), Value::Null);
            out.insert("panicked".into(), json!(true));
        }
        Some(r) => {
            out.insert("err".into(), err_value(&r));
            out.insert("panicked".into(), json!(false));
            if let Ok(v) = &r {
                for (key, value) in ok(v) {
                    out.insert(key.into(), value);
                }
            }
        }
    }
    out
}

fn ok_or_err<T>(r: &NativeFlowResult<T>, key: &str, ok: impl FnOnce(&T) -> Value) -> Value {
    let mut out = serde_json::Map::new();
    out.insert("err".into(), err_value(r));
    if let Ok(v) = r {
        out.insert(key.into(), ok(v));
    }
    Value::Object(out)
}

fn candidates(c: &[NativePhaseDispatchCandidate]) -> Value {
    c.iter().map(|x| json!([x.unit_index, x.selector_arg])).collect()
}

fn script_err(v: &Value) -> NativeFlowResult<()> {
    match v {
        Value::Null => Ok(()),
        s => Err(NativeFlowError::Callback(s.as_str().unwrap().to_string())),
    }
}

fn rows<'a>(script: &'a Value, key: &str) -> &'a [Value] {
    script[key].as_array().map(Vec::as_slice).unwrap_or(&[])
}

/// 依腳本執行 `execute_native_phase_unit_scans`；對應 Go 匯出工具的 `phaseExecute`。
fn phase_execute(input: &Value) -> Value {
    let mut raw = bytes_of(input, "records");
    let count = i(input, "count");
    let script = &input["script"];
    let calls = RefCell::new(Vec::<String>::new());
    let result = {
        let records = Cell::from_mut(raw.as_mut_slice()).as_slice_of_cells();
        let (pre, act, glob, chap, chev, pend) = (
            rows(script, "preselect"),
            rows(script, "action"),
            rows(script, "global"),
            rows(script, "chapter"),
            rows(script, "chapter_event"),
            rows(script, "pending"),
        );
        let next = |n: &Cell<usize>, len: usize| {
            let k = n.get();
            if k >= len {
                None
            } else {
                n.set(k + 1);
                Some(k)
            }
        };
        let (ip, ia, ig, ic, ie, iq) =
            (Cell::new(0), Cell::new(0), Cell::new(0), Cell::new(0), Cell::new(0), Cell::new(0));
        let mut cb = NativePhaseScanCallbacks {
            preselect: Some(Box::new(|unit| {
                calls.borrow_mut().push(format!("preselect:{unit}"));
                let Some(k) = next(&ip, pre.len()) else { return Ok(NativePhasePreselectionScores::default()) };
                let row = &pre[k];
                script_err(&row[2])?;
                Ok(NativePhasePreselectionScores {
                    spell_candidate_score_53c23: row[0].as_i64().unwrap() as i32,
                    item_command_score_53c33: row[1].as_i64().unwrap() as i32,
                })
            })),
            run_unit_action: Some(Box::new(|unit, selector| {
                calls.borrow_mut().push(format!("action:{unit}:{selector}"));
                let Some(k) = next(&ia, act.len()) else { return Ok(0xff) };
                let row = &act[k];
                let offset = row[2].as_i64().unwrap();
                if offset >= 0 {
                    records[offset as usize].set(row[3].as_i64().unwrap() as u8);
                }
                script_err(&row[1])?;
                Ok(row[0].as_i64().unwrap())
            })),
            run_global_event: Some(Box::new(|event_id, unit| {
                calls.borrow_mut().push(format!("global:{event_id}:{unit}"));
                let Some(k) = next(&ig, glob.len()) else { return Ok(()) };
                script_err(&glob[k])
            })),
            chapter_index: Some(Box::new(|| {
                calls.borrow_mut().push("chapter-index".to_string());
                let Some(k) = next(&ic, chap.len()) else { return Ok(3) };
                script_err(&chap[k][1])?;
                Ok(chap[k][0].as_i64().unwrap())
            })),
            run_chapter_event: Some(Box::new(|chapter, unit| {
                calls.borrow_mut().push(format!("chapter:{chapter}:{unit}"));
                let Some(k) = next(&ie, chev.len()) else { return Ok(()) };
                script_err(&chev[k])
            })),
            pending_code: Some(Box::new(|| {
                calls.borrow_mut().push("pending".to_string());
                let Some(k) = next(&iq, pend.len()) else { return Ok(0) };
                script_err(&pend[k][1])?;
                Ok(pend[k][0].as_i64().unwrap())
            })),
        };
        let nil: Vec<bool> = script["nil"].as_array().unwrap().iter().map(|x| x.as_bool().unwrap()).collect();
        if nil[0] {
            cb.preselect = None;
        }
        if nil[1] {
            cb.run_unit_action = None;
        }
        if nil[2] {
            cb.run_global_event = None;
        }
        if nil[3] {
            cb.chapter_index = None;
        }
        if nil[4] {
            cb.run_chapter_event = None;
        }
        if nil[5] {
            cb.pending_code = None;
        }
        catch(|| execute_native_phase_unit_scans(records, count, &mut cb))
    };
    let mut out = protected(result, |r| vec![("result", json!([r.stopped, r.pass.0, r.unit_index, r.pending_code]))]);
    out.insert("calls".into(), json!(calls.into_inner()));
    out.insert("records".into(), json!(hex(&raw)));
    Value::Object(out)
}

fn run(op: &str, v: &Value) -> Value {
    match op {
        "battle_entry_step" => {
            let mut raw = bytes_of(v, "raw");
            let r = native_battle_entry_step(&mut raw);
            let (next, eligible, changed) = r.clone().unwrap_or((0, false, false));
            json!({"err": err_value(&r), "next": next, "eligible": eligible, "changed": changed, "raw": hex(&raw)})
        }
        "copy_buffer" => {
            let (mut dst, src) = (bytes_of(v, "dst"), bytes_of(v, "src"));
            let r = copy_native_transition_buffer(&mut dst, &src);
            json!({"err": err_value(&r), "dst": hex(&dst)})
        }
        "copy_viewport" => {
            let (mut dst, src) = (bytes_of(v, "dst"), bytes_of(v, "src"));
            let r = catch(|| {
                copy_native_transition_viewport(
                    &mut dst,
                    i(v, "dst_stride"),
                    &src,
                    i(v, "src_offset"),
                    i(v, "src_stride"),
                    i(v, "width"),
                    i(v, "height"),
                )
            });
            let mut out = protected(r, |_| Vec::new());
            out.insert("dst".into(), json!(hex(&dst)));
            Value::Object(out)
        }
        "transition_lut" => {
            let bank: Vec<Vec<u8>> = v["bank"].as_array().unwrap().iter().map(|x| unhex(x.as_str().unwrap())).collect();
            ok_or_err(&native_indexed_transition_lut(&bank, i(v, "descriptor")), "lut", |l| json!(hex(l)))
        }
        "transition_schedule" => {
            let r = build_native_indexed_transition_schedule(i(v, "start"), i(v, "step"));
            let mut out = serde_json::Map::new();
            out.insert("err".into(), err_value(&r));
            if let Ok(s) = &r {
                let frames: Vec<Value> =
                    s.frames.iter().map(|f| json!([f.pass, f.lut_index, f.radius, f.frame_delay_ms])).collect();
                out.insert("frames".into(), json!(frames));
                out.insert("tail_delay_ms".into(), json!(s.tail_delay_ms));
                out.insert("palette_deltas".into(), json!(s.palette_deltas));
                out.insert("palette_delay_ms".into(), json!(s.palette_delay_ms));
            }
            Value::Object(out)
        }
        "map_index" => {
            let r = map_index_from_asset_path(v["path"].as_str().unwrap());
            json!({"err": err_value(&r), "index": r.clone().unwrap_or(0)})
        }
        "ai_item_damage" => ok_or_err(&build_native_ai_item_damage_tail_schedule(i(v, "id")), "schedule", |s| {
            json!({"effect_resource": s.effect_resource, "effect_start": s.effect_start,
                "effect_frames": s.effect_frames, "effect_sample": s.effect_sample,
                "effect_frame_delay_ticks": s.effect_frame_delay_ticks, "blend_frames": s.blend_frames,
                "raw_base": s.raw_base, "blend_delay_ticks": s.blend_delay_ticks, "blend": s.blend,
                "digit_resource": s.digit_resource, "digit_bias": s.digit_bias, "digit_frames": s.digit_frames,
                "digit_frame_delay_ticks": s.digit_frame_delay_ticks, "digit_hold_ms": s.digit_hold_ms,
                "digit_vertical": s.digit_vertical})
        }),
        "command10_12" => ok_or_err(&build_native_command10_to12_schedule(i(v, "id")), "schedule", |s| {
            let p = s.prelude;
            json!({"command_id": s.command_id, "sound_resource": s.sound_resource,
                "prelude": [p.enabled, p.initial_radius, p.radius_step, p.repeat, p.sample],
                "x_origins": s.x_origins, "y_origins": s.y_origins, "sampling_increments": s.sampling_increments,
                "surface_cycle": s.surface_cycle, "frames": s.frames, "delay_ms": s.delay_ms,
                "main_sample": s.main_sample, "main_sample_frames": s.main_sample_frames,
                "result_resource": s.result_resource, "result_digit_bias": s.result_digit_bias,
                "result_miss_descriptors": s.result_miss_descriptors, "result_frames": s.result_frames,
                "result_hold_ms": s.result_hold_ms, "result_vertical": s.result_vertical})
        }),
        "heal_presentation" => {
            ok_or_err(&build_native_command_heal_presentation_schedule(i(v, "id")), "schedule", |s| {
                let frames: Vec<Value> =
                    s.frames.iter().map(|f| json!([f.lut_index, f.radius, f.frame_delay_ms])).collect();
                json!({"frames": frames, "mid_frame": s.mid_frame, "mid_delay_ms": s.mid_delay_ms,
                    "tail_delay_ms": s.tail_delay_ms, "sample_index": s.sample_index})
            })
        }
        "heal_tail" => ok_or_err(&build_native_command_heal_tail_schedule(i(v, "id")), "schedule", |s| {
            json!({"effect_resource": s.effect_resource, "effect_start": s.effect_start,
                "effect_frames": s.effect_frames, "effect_sample": s.effect_sample,
                "effect_frame_delay_ticks": s.effect_frame_delay_ticks, "mask_sample": s.mask_sample,
                "mask_index": s.mask_index, "mask_pairs": s.mask_pairs, "digit_resource": s.digit_resource,
                "digit_bias": s.digit_bias, "digit_frames": s.digit_frames,
                "digit_frame_delay_ms": s.digit_frame_delay_ms, "digit_hold_ms": s.digit_hold_ms,
                "digit_vertical": s.digit_vertical})
        }),
        "command34" => ok_or_err(&build_native_command34_tail_schedule(), "rows", |rows| {
            rows.iter()
                .map(|r| {
                    json!({"command_id": r.command_id, "effect_start": r.effect_start,
                        "effect_frames": r.effect_frames, "effect_sample": r.effect_sample,
                        "extra_sample_frame_indices": r.extra_sample_frame_indices, "mask_index": r.mask_index,
                        "mask_pairs": r.mask_pairs, "digit_bias": r.digit_bias, "digit_frames": r.digit_frames,
                        "digit_hold_milliseconds": r.digit_hold_milliseconds, "digit_vertical": r.digit_vertical})
                })
                .collect()
        }),
        "compound_dac" => {
            let raw: [u8; 3] = bytes_of(v, "raw").try_into().unwrap();
            let r = interpolate_native_compound_dac(
                &bytes_of(v, "baseline"),
                i(v, "start"),
                i(v, "end"),
                i(v, "delta"),
                raw,
            );
            ok_or_err(&r, "out", |o| json!(hex(o)))
        }
        "hotel_route" => match resolve_native_hotel_service_route(u8_(v, "selector")) {
            Some(r) => json!({"ok": true, "route": [r.selector, r.resource_id, r.primary, r.secondary]}),
            None => json!({"ok": false}),
        },
        "shop_route" => {
            let variant = u8_(v, "hub_variant");
            let routes: Vec<Value> = (0..=255u8)
                .map(|s| match resolve_native_shop_service_route(variant, s) {
                    Some(r) => json!([r.hub_variant, r.resource_id, r.selector, r.callee, r.kind.as_str()]),
                    None => Value::Null,
                })
                .collect();
            json!({"routes": routes})
        }
        "postbattle_route" => {
            let table = bytes_of(v, "table");
            let chapter = i(v, "chapter");
            let routes: Vec<Value> = (0..=255u8)
                .map(|s| match resolve_native_postbattle_route(chapter, &table, s) {
                    Some(r) => json!([r.selector, r.callee, r.preparation, r.direct_preparation]),
                    None => Value::Null,
                })
                .collect();
            json!({"routes": routes})
        }
        "postbattle_outcome" => {
            let route = NativePostbattleRoute {
                selector: u8_(v, "selector"),
                callee: v["callee"].as_u64().unwrap() as u32,
                preparation: b(v, "preparation"),
                direct_preparation: b(v, "direct_preparation"),
            };
            match resolve_native_postbattle_outcome(route, i(v, "callee_result")) {
                Some(o) => json!({"ok": true, "outcome": [o.repeat, o.return_value]}),
                None => json!({"ok": false}),
            }
        }
        "party_limit" => json!({"limit": native_preparation_party_limit(i(v, "index"))}),
        "preparation_key" => {
            let d = u8_(v, "raw53a8d");
            let keys: Vec<u8> = (0..=255u8).map(|k| normalize_native_preparation_key(d, k)).collect();
            json!({"keys": hex(&keys)})
        }
        "preparation_input" => {
            let scan = u8_(v, "scan");
            let results: Vec<Value> = v["states"]
                .as_array()
                .unwrap()
                .iter()
                .map(|s| match s.as_i64() {
                    None => json!([apply_native_preparation_input(scan, None).0, null]),
                    Some(mut state) => {
                        let got = apply_native_preparation_input(scan, Some(&mut state));
                        json!([got.0, state])
                    }
                })
                .collect();
            json!({"results": results})
        }
        "preparation_dispatch" => {
            let r = find_native_preparation_dispatch(&bytes_of(v, "table"), u8_(v, "gate3"), u8_(v, "selector"));
            ok_or_err(&r, "matches", |m| m.iter().map(|x| json!([x.slot, x.entry_start, x.function_index])).collect())
        }
        "preparation_record" => {
            let r = parse_native_preparation_record(&bytes_of(v, "raw"));
            let mut out = serde_json::Map::new();
            out.insert("err".into(), err_value(&r));
            if let Ok(rec) = r {
                let (adjusted, divisor) = native_preparation_adjusted_word40(rec);
                out.insert("record".into(), json!([rec.raw25, rec.raw5, rec.raw6, rec.raw7, rec.word40, rec.word42]));
                out.insert("eligible".into(), json!(native_preparation_eligible(rec, i(v, "selector"))));
                out.insert("adjusted".into(), json!(adjusted));
                out.insert("divisor".into(), json!(divisor));
            }
            Value::Object(out)
        }
        "preparation_timer" => {
            let mut records = bytes_of(v, "records");
            let r = catch(|| tick_native_preparation_timers(&mut records, i(v, "count"), i(v, "selector")));
            let mut out = protected(r, |x| {
                vec![(
                    "expired",
                    x.iter()
                        .map(|e| json!([e.record_index, e.counter_index, e.counter_offset, e.visual_source_offset]))
                        .collect(),
                )]
            });
            out.insert("records".into(), json!(hex(&records)));
            Value::Object(out)
        }
        "post_resolution" => {
            let raw = bytes_of(v, "raw");
            let r = catch(|| parse_native_post_resolution_commands(&raw, i(v, "count")));
            Value::Object(protected(r, |c| vec![("commands", c.iter().map(|x| json!([x.kind, x.payload])).collect())]))
        }
        "unit_mode" => {
            let r = plan_native_unit_mode(&bytes_of(v, "record"), i(v, "unit_index"), u8_(v, "caller_arg"));
            let plan = match &r {
                Ok(Some(p)) => {
                    json!([p.unit_index, p.caller_arg, p.mode, p.raw_arg35, p.raw_arg36, p.raw_counter])
                }
                _ => Value::Null,
            };
            json!({"err": err_value(&r), "plan": plan})
        }
        "unit_mode_branch" => {
            let a = v["args"].as_array().unwrap();
            let f = |k: usize| a[k].as_bool().unwrap();
            let n = |k: usize| a[k].as_i64().unwrap();
            let actions = match i(v, "mode") {
                0 => plan_native_unit_mode0(f(0), f(1), f(2)),
                1 => plan_native_unit_mode1(f(0), f(1)),
                2 => plan_native_unit_mode2(f(0)),
                3 => plan_native_unit_mode3(f(0), n(1), f(2), f(3), f(4)),
                4 => plan_native_unit_mode4(f(0)),
                5 => plan_native_unit_mode5(NativeUnitMode5Inputs {
                    candidate_dispatch_succeeded: f(0),
                    event_state_byte: n(1) as u8,
                    event_lookup_succeeded: f(2),
                    movement_succeeded: f(3),
                    arrived_at_event_coordinate: f(4),
                    event_record_type: n(5) as u8,
                    blocked_search_succeeded: f(6),
                    nearest_coordinate_search_succeeded: f(7),
                }),
                7 => plan_native_unit_mode7(f(0), f(1)),
                9 => plan_native_unit_mode9(n(0), f(1), f(2), f(3), f(4)),
                10 => plan_native_unit_mode10(f(0), f(1)),
                11 => plan_native_unit_mode11(n(0) as i32, n(1) as i32, f(2)),
                other => panic!("未知的 mode {other}"),
            };
            let actions: Vec<u8> = actions.iter().map(|x| *x as u8).collect();
            json!({"actions": actions})
        }
        "phase_find" => {
            let records = bytes_of(v, "records");
            let r = catch(|| find_native_phase_dispatch_candidates(&records, i(v, "count")));
            Value::Object(protected(r, |c| vec![("candidates", candidates(c))]))
        }
        "phase_plan" => {
            let records = bytes_of(v, "records");
            let scores: Vec<NativePhasePreselectionScores> = v["scores"]
                .as_array()
                .unwrap()
                .iter()
                .map(|s| NativePhasePreselectionScores {
                    spell_candidate_score_53c23: s[0].as_i64().unwrap() as i32,
                    item_command_score_53c33: s[1].as_i64().unwrap() as i32,
                })
                .collect();
            let r = catch(|| plan_native_phase_unit_scans(&records, i(v, "count"), &scores));
            Value::Object(protected(r, |p| {
                let pre: Value = p
                    .zero_preselection
                    .iter()
                    .map(|x| json!([x.unit_index, x.selector_arg, x.run_unit_action]))
                    .collect();
                vec![
                    ("selector_one", candidates(&p.selector_one)),
                    ("zero_preselection", pre),
                    ("zero_second_pass", candidates(&p.zero_second_pass)),
                ]
            }))
        }
        "phase_execute" => phase_execute(v),
        "sysinfo_regions" => {
            let r = native_system_info_regions(v["phase"].as_str().unwrap(), i(v, "frame"));
            ok_or_err(&r, "regions", |regions| {
                regions
                    .iter()
                    .map(|r| {
                        let s = r.source;
                        json!([s.min_x, s.min_y, s.max_x, s.max_y, r.destination.0, r.destination.1])
                    })
                    .collect()
            })
        }
        "sysinfo_frames" | "sysinfo_steady" => {
            let surface = |key: &str| match &v[key] {
                Value::Null => vec![0u8; i(v, &format!("{key}_len")) as usize],
                s => unhex(s.as_str().unwrap()),
            };
            let (baseline, information) = (surface("baseline"), surface("information"));
            if op == "sysinfo_steady" {
                ok_or_err(&native_system_info_steady_frame(&baseline, &information), "frame", |f| json!(digest(f)))
            } else {
                let r = native_system_info_transition_frames(&baseline, &information);
                let mut out = serde_json::Map::new();
                out.insert("err".into(), err_value(&r));
                if let Ok((opening, closing)) = &r {
                    out.insert("opening".into(), opening.iter().map(|f| json!(digest(f))).collect());
                    out.insert("closing".into(), closing.iter().map(|f| json!(digest(f))).collect());
                }
                Value::Object(out)
            }
        }
        other => panic!("未知的 op {other}"),
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

/// 覆蓋分類：錯誤文字（數字換成 #）、panic、接受，另依 op 加上分支細節。
fn classify(op: &str, input: &Value, result: &Value) -> String {
    if result["panicked"] == json!(true) {
        return "panicked".to_string();
    }
    if let Some(e) = result["err"].as_str() {
        let mut s = String::new();
        for ch in e.chars() {
            if ch.is_ascii_digit() || ch == '-' {
                if !s.ends_with('#') {
                    s.push('#');
                }
            } else {
                s.push(ch);
            }
        }
        return format!("err: {s}");
    }
    match op {
        "battle_entry_step" => format!("ok eligible={} changed={}", result["eligible"], result["changed"]),
        "hotel_route" | "postbattle_outcome" => match &result["outcome"] {
            Value::Array(o) => format!("ok repeat={} return={}", o[0], o[1]),
            _ => format!("ok={}", result["ok"]),
        },
        "shop_route" | "postbattle_route" => {
            let routes = result["routes"].as_array().unwrap();
            let callees: BTreeSet<String> = routes
                .iter()
                .filter(|r| !r.is_null())
                .map(|r| format!("{}:{}", r[1], r[r.as_array().unwrap().len() - 1]))
                .collect();
            format!("accepted={} {:?}", routes.iter().filter(|r| !r.is_null()).count(), callees)
        }
        "unit_mode" => format!("plan={}", if result["plan"].is_null() { "gated" } else { "ok" }),
        "unit_mode_branch" => format!("mode {} actions={}", input["mode"], result["actions"]),
        "phase_execute" => match &result["result"] {
            Value::Array(r) if r[0] == json!(true) => format!("stopped pass={}", r[1]),
            _ => "completed".to_string(),
        },
        "preparation_record" => format!("eligible={} clamp={}", result["eligible"], result["adjusted"] == json!(0)),
        "preparation_timer" => {
            format!("expired={}", result["expired"].as_array().map(|x| !x.is_empty()).unwrap_or(false))
        }
        "preparation_dispatch" => format!("matches={}", result["matches"].as_array().unwrap().len().min(2)),
        "phase_plan" => format!(
            "zero_pre={} run={}",
            result["zero_preselection"].as_array().unwrap().len().min(1),
            result["zero_preselection"].as_array().unwrap().iter().any(|x| x[2] == json!(true))
        ),
        _ => "ok".to_string(),
    }
}

#[test]
fn matches_frozen_go_golden() {
    let required = std::env::var("FD2_REQUIRE_GOLDEN").is_ok();
    let Ok(path) = std::env::var("FD2_GOLDEN_FDOTHER_NATIVE") else {
        assert!(!required, "FD2_REQUIRE_GOLDEN=1 但沒有 FD2_GOLDEN_FDOTHER_NATIVE");
        eprintln!("略過：未設定 FD2_GOLDEN_FDOTHER_NATIVE");
        return;
    };
    let doc: Value = serde_json::from_slice(&std::fs::read(&path).expect("讀黃金檔")).unwrap();
    assert_eq!(doc["schema"], "fd2-golden-fdother-native/1");
    let cases = doc["cases"].as_array().unwrap();
    assert!(!cases.is_empty());

    let mut failures = Vec::new();
    let mut known = Vec::new();
    let mut kinds = BTreeMap::<String, usize>::new();
    let mut coverage = BTreeMap::<String, BTreeMap<String, usize>>::new();
    for case in cases {
        let index = case["index"].as_u64().unwrap();
        let (op, kind) = (case["op"].as_str().unwrap(), case["kind"].as_str().unwrap());
        *kinds.entry(format!("{op}/{kind}")).or_default() += 1;
        let want = &case["result"];
        let got = run(op, &case["input"]);
        *coverage.entry(op.to_string()).or_default().entry(classify(op, &case["input"], want)).or_default() += 1;
        if let Some(d) = first_diff("result", want, &got) {
            if let Some((_, reason)) = KNOWN_DIFFERENCES.iter().find(|(k, _)| *k == kind) {
                known.push(format!("case {index} ({kind} {op})：刻意不照做，{reason}；{d}"));
            } else {
                failures.push(format!("case {index} ({kind} {op}): {d}"));
            }
        }
    }
    eprintln!("黃金案例 {}：", cases.len());
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
    assert!(failures.is_empty(), "{} 個案例不符：\n{}", failures.len(), failures.join("\n"));
}
