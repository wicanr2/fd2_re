//! 與凍結 Go 版 `remake/cmd/fd2-golden-fdsave` 的黃金差分。
//!
//! 由 `tools/rust_golden_fdsave.sh` 產生黃金 JSON 並設定 `FD2_GOLDEN_FDSAVE`。
//! 設定 `FD2_REQUIRE_GOLDEN=1` 時缺檔即失敗，避免把「沒跑到」當成通過。

use fd2_formats::save::*;
use serde_json::{Value, json};
use sha2::{Digest, Sha256};

fn digest(b: &[u8]) -> String {
    Sha256::digest(b).iter().map(|x| format!("{x:02x}")).collect()
}

fn hex(b: &[u8]) -> String {
    b.iter().map(|x| format!("{x:02x}")).collect()
}

fn unhex(s: &str) -> Vec<u8> {
    (0..s.len()).step_by(2).map(|i| u8::from_str_radix(&s[i..i + 2], 16).unwrap()).collect()
}

fn err_value<T>(r: &Result<T>) -> Value {
    match r {
        Ok(_) => Value::Null,
        Err(e) => Value::String(e.to_string()),
    }
}

fn meta_json(m: &VerifiedMetadata) -> Value {
    json!({
        "chapter": m.chapter, "roster_count": m.roster_count, "currency": m.currency,
        "hud_gate_a": m.hud_gate_a, "raw_53af9": m.raw_53af9, "raw_51e61": m.raw_51e61, "raw_51e62": m.raw_51e62,
    })
}

fn view_json(r: &PersistentRecord) -> Value {
    let v = r.view();
    let inventory: Vec<[u8; 2]> = v.inventory.iter().map(|c| [c.flags, c.item_id]).collect();
    json!({
        "raw_byte5": v.raw_byte5, "raw_camp": v.raw_camp, "raw_presentation_key": v.raw_presentation_key,
        "raw_identity": v.raw_identity, "inventory": inventory, "command_mask": v.command_mask,
        "race": v.race, "class": v.class, "level": v.level, "transient": v.transient,
        "raw_byte34": v.raw_byte34, "raw_byte35": v.raw_byte35, "raw_byte36": v.raw_byte36,
        "base_ap": v.base_ap, "base_dp": v.base_dp, "movement": v.movement, "experience": v.experience,
        "dx": v.dx, "hp": v.hp, "max_hp": v.max_hp, "mp": v.mp, "max_mp": v.max_mp,
        "ap": v.ap, "dp": v.dp, "hit": v.hit, "ev": v.ev,
    })
}

fn records_digest(records: &[PersistentRecord]) -> String {
    digest(&records.iter().flat_map(|r| r.raw).collect::<Vec<_>>())
}

fn views_json(records: &[PersistentRecord]) -> Value {
    Value::Array(records.iter().map(view_json).collect())
}

fn header_json(h: &CurrentRuntimeHeader) -> Value {
    json!({
        "turn_counter": h.turn_counter, "runtime_count": h.runtime_count, "chapter": h.chapter,
        "camera_x": h.camera_x, "camera_y": h.camera_y, "cursor_x": h.cursor_x, "cursor_y": h.cursor_y,
        "visible_cursor_x": h.visible_cursor_x, "visible_cursor_y": h.visible_cursor_y,
        "persistent_count": h.persistent_count, "currency": h.currency, "raw_53af9": h.raw_53af9,
        "hud_gate_a": h.hud_gate_a, "raw_51e61": h.raw_51e61, "raw_51e62": h.raw_51e62,
    })
}

fn context_from(v: &Value) -> ContinueRuntimeContext {
    let i = |k: &str| v[k].as_i64().unwrap();
    ContinueRuntimeContext {
        chapter: i("chapter"),
        field_width: i("field_width"),
        field_height: i("field_height"),
        selector_group_count: i("selector_group_count"),
        title_timer_tick: i("title_timer_tick"),
        has_title_timer_tick: v["has_title_timer_tick"].as_bool().unwrap(),
    }
}

fn continue_json(snapshot: &CurrentSnapshot, context: ContinueRuntimeContext) -> Value {
    let input = match build_continue_runtime_input(snapshot, context) {
        Ok(input) => input,
        Err(e) => return json!({ "err": e.to_string() }),
    };
    let fc = &input.field_control;
    let mp = input.map_presentation;
    let ts = input.map_timing_seed;
    json!({
        "err": null,
        "selector": input.runtime_records.iter().map(|r| [r.selector_key, r.selector_slot]).collect::<Vec<_>>(),
        "map_presentation": {
            "opening_range_mode": mp.opening_range_mode, "interactive_range_mode": mp.interactive_range_mode,
            "hud_gate_b": mp.hud_gate_b, "hud_anchor_x": mp.hud_anchor_x,
        },
        "timing": [ts.sprite_idle_cycle, ts.sprite_moving_cycle, ts.sprite_last_timer_tick, ts.terrain_phase,
            ts.terrain_phase_last_timer_tick, ts.terrain_phase_override, ts.terrain_flip,
            ts.terrain_flip_last_timer_tick, ts.unit_pixel_shift, ts.unit_pixel_shift_last_timer_tick],
        "field_control": {
            "map_selector": fc.raw_map_selector, "own_deploy": fc.raw_own_deploy_count, "unit_count": fc.raw_unit_count,
            "turn_events": fc.turn_events.iter().map(|e| [e.turn, e.event_id, e.raw_camp]).collect::<Vec<_>>(),
            "field_events": fc.field_events.iter().map(|e| [e.event_id, e.selector]).collect::<Vec<_>>(),
            "chests": fc.chests.iter().map(|e| [e.raw_type as u16, e.value]).collect::<Vec<_>>(),
            "units_len": fc.units.len(),
            "units_sha256": digest(&fc.units.iter().flat_map(|u| u.raw).collect::<Vec<_>>()),
        },
        "ready": input.ready_for_continue(),
        "owners": input.unresolved_owners.iter().map(|o| o.as_str()).collect::<Vec<_>>(),
        "validated": input.validated_for_runtime_bridge(),
    })
}

fn summarize(index: usize, stored: &[u8], context: ContinueRuntimeContext) -> Value {
    let mut out = serde_json::Map::new();
    let decoded = decode(stored);
    out.insert("decode_err".into(), err_value(&decoded));
    if stored.len() != FILE_SIZE {
        return Value::Object(out);
    }
    let plain = decoded.unwrap_or_else(|_| xor_envelope(stored));
    out.insert("plain_sha256".into(), json!(digest(&plain)));
    out.insert("checksum".into(), json!(checksum(&plain).unwrap()));
    out.insert("encode_sha256".into(), json!(digest(&encode(&plain).unwrap())));

    let mut slots = Vec::new();
    for slot in 0..=SLOT_COUNT {
        let mut row = serde_json::Map::new();
        let bounds = slot_bounds(slot);
        row.insert("bounds_err".into(), err_value(&bounds));
        let (start, end) = bounds.unwrap_or((0, 0));
        row.insert("bounds".into(), json!([start, end]));
        match read_verified_metadata(&plain, slot) {
            Ok(m) => row.insert("meta".into(), meta_json(&m)),
            Err(e) => row.insert("meta_err".into(), json!(e.to_string())),
        };
        match inspect_chapter_slot(&plain, slot) {
            Ok(s) => row.insert(
                "inspect".into(),
                json!({
                    "verified": meta_json(&s.verified), "metadata_hex": hex(&s.metadata),
                    "records_sha256": records_digest(&s.records), "source_sha256": s.source_save_sha256,
                    "active_views": views_json(&s.active_records()),
                }),
            ),
            Err(e) => row.insert("inspect_err".into(), json!(e.to_string())),
        };
        slots.push(Value::Object(row));
    }
    out.insert("slots".into(), Value::Array(slots));

    match inspect_current_snapshot(&plain) {
        Err(e) => {
            out.insert("current".into(), json!({ "err": e.to_string() }));
        }
        Ok(mut snapshot) => {
            out.insert(
                "current".into(),
                json!({
                    "err": null,
                    "header": header_json(&snapshot.header),
                    "native_field_control_sha256": digest(&snapshot.native_field_control),
                    "persistent_sha256": records_digest(&snapshot.persistent_records),
                    "runtime_sha256": records_digest(&snapshot.runtime_records),
                    "event_state_hex": hex(&snapshot.native_event_state),
                    "active_persistent_views": views_json(&snapshot.active_persistent_records()),
                    "runtime_views": views_json(&snapshot.runtime_records),
                }),
            );
            out.insert("continue".into(), continue_json(&snapshot, context));
            let written = write_current_snapshot(&plain, &snapshot);
            out.insert("write_current_err".into(), err_value(&written));
            if let Ok(w) = &written {
                out.insert("write_current_sha256".into(), json!(digest(w)));
            }
            snapshot.header.turn_counter = snapshot.header.turn_counter.wrapping_add(1);
            snapshot.header.currency ^= 0x5a5a;
            snapshot.native_field_control[2] = (index % 40) as u8;
            let written = write_current_snapshot(&plain, &snapshot);
            out.insert("write_current_mutated_err".into(), err_value(&written));
            if let Ok(w) = &written {
                out.insert("write_current_mutated_sha256".into(), json!(digest(w)));
            }
        }
    }

    let source = read_slot(&plain, (index + 1) % SLOT_COUNT).unwrap();
    let written = write_slot(&plain, index % SLOT_COUNT, &source);
    out.insert("write_slot_err".into(), err_value(&written));
    if let Ok(w) = &written {
        out.insert("write_slot_sha256".into(), json!(digest(w)));
    }
    Value::Object(out)
}

/// 回傳第一個不同的 JSON 路徑，方便定位。
fn first_diff(path: &str, want: &Value, got: &Value) -> Option<String> {
    match (want, got) {
        (Value::Object(a), Value::Object(b)) => {
            for key in a.keys().chain(b.keys()) {
                let (x, y) = (a.get(key).unwrap_or(&Value::Null), b.get(key).unwrap_or(&Value::Null));
                if a.contains_key(key) != b.contains_key(key) {
                    return Some(format!(
                        "{path}.{key}: present go={} rust={}",
                        a.contains_key(key),
                        b.contains_key(key)
                    ));
                }
                if let Some(d) = first_diff(&format!("{path}.{key}"), x, y) {
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

#[test]
fn matches_frozen_go_golden() {
    let Ok(path) = std::env::var("FD2_GOLDEN_FDSAVE") else {
        assert!(std::env::var("FD2_REQUIRE_GOLDEN").is_err(), "FD2_REQUIRE_GOLDEN=1 但沒有 FD2_GOLDEN_FDSAVE");
        eprintln!("略過：未設定 FD2_GOLDEN_FDSAVE");
        return;
    };
    let doc: Value = serde_json::from_slice(&std::fs::read(&path).expect("讀黃金檔")).unwrap();
    assert_eq!(doc["schema"], "fd2-golden-fdsave/1");
    let cases = doc["cases"].as_array().unwrap();
    assert!(!cases.is_empty());
    let mut failures = Vec::new();
    let mut kinds = std::collections::BTreeMap::<String, usize>::new();
    for case in cases {
        let index = case["index"].as_u64().unwrap() as usize;
        *kinds.entry(case["kind"].as_str().unwrap().to_string()).or_default() += 1;
        let got = summarize(index, &unhex(case["stored_hex"].as_str().unwrap()), context_from(&case["context"]));
        if let Some(d) = first_diff("result", &case["result"], &got) {
            failures.push(format!("case {index} ({}): {d}", case["kind"]));
        }
    }
    eprintln!("黃金案例 {}：{kinds:?}", cases.len());
    assert!(failures.is_empty(), "{} 個案例不符：\n{}", failures.len(), failures.join("\n"));
}
