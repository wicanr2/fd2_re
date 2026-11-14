//! 與凍結 Go 版 `remake/cmd/fd2-golden-present-map` 的黃金差分：fd2-campaign 的 `indexedmap`（R3 present 組）。
//!
//! 由 `tools/rust_golden_present_map.sh` 產生黃金 JSON 並設定 `FD2_GOLDEN_PRESENT_MAP`；真實輸入掛在
//! `FD2_GOLDEN_ASSETS`（受版控的 `remake/assets`）與 `FD2_GOLDEN_PACK`（分離素材包）底下，先逐檔核對黃金檔
//! 記錄的大小與 SHA-256。設定 `FD2_REQUIRE_GOLDEN=1` 時，黃金檔或真實輸入缺少即失敗。
//!
//! 比對方式：每個案例依黃金檔記錄的局面（33 張真實地圖、固定種子產生的鏡頭／單位層／前景層／選擇器快取／
//! HUD 輸入，以及 xorshift32 填滿的初始緩衝）在 Rust 端重做同一串呼叫，比對工作緩衝與 VGA 的 SHA-256、
//! 觀察者各階段的 SHA-256、回傳值與錯誤文字（逐字相同）。

use fd2_campaign::indexedmap::*;
use fd2_formats::fdicon::{
    self, Bank, NativeForegroundLayerEntry, NativeSelectorCache, NativeTerrainCell, NativeUnitLayerEntry,
};
use fd2_formats::fdother::{self, LMI1Entry, NativeChapterAuxSurface, NativeMapParallaxSurface};
use fd2_rules::battle::NativePresentationDigit;
use serde_json::Value;
use sha2::{Digest, Sha256};
use std::path::{Path, PathBuf};

/// 刻意不照做 Go 的行為。
const KNOWN_DIFFERENCES: &[&str] = &[
    // Go 的 RunNativeUnitPresentStripBridge 延遲 callback 只收列號；Rust 另外把目前的 VGA 唯讀交給 callback
    // （Go 閉包可以同時讀呼叫端緩衝，Rust 不能）。黃金案例只比列號序列。
    "逐列延遲 callback 多一個 VGA 參數",
];

/// 比對前的表示法正規化（不影響行為）：
const NORMALIZATIONS: &[&str] = &[
    // 緩衝只比 SHA-256；格子陣列以 (tile 低位元組, tile 高位元組, blit_mode) 序列的 SHA-256 比對。
    "緩衝與格子比雜湊",
];

const WORK: usize = NATIVE_UNIT_PRESENT_WORK_SIZE as usize;
const VGA: usize = NATIVE_MAP_VGA_SIZE as usize;

fn digest(b: &[u8]) -> String {
    Sha256::digest(b).iter().map(|x| format!("{x:02x}")).collect()
}

/// 與 Go `fill` 相同的 xorshift32。
fn fill(n: usize, salt: i64) -> Vec<u8> {
    let mut x = (salt as u32).wrapping_mul(2654435761).wrapping_add(1);
    (0..n)
        .map(|_| {
            x ^= x << 13;
            x ^= x >> 17;
            x ^= x << 5;
            x as u8
        })
        .collect()
}

fn errv<T>(r: &Result<T>) -> Value {
    match r {
        Ok(_) => Value::Null,
        Err(e) => Value::String(e.to_string()),
    }
}

fn i(v: &Value) -> i64 {
    v.as_i64().unwrap_or_else(|| panic!("不是整數：{v}"))
}

fn b(v: &Value) -> bool {
    v.as_bool().unwrap_or_else(|| panic!("不是布林：{v}"))
}

fn cells_digest(cells: &[NativeTerrainCell]) -> String {
    let mut out = Vec::with_capacity(cells.len() * 3);
    for c in cells {
        out.extend_from_slice(&[c.tile as u8, (c.tile >> 8) as u8, c.blit_mode]);
    }
    digest(&out)
}

struct MapAssets {
    width: i64,
    tiles: Vec<i64>,
    modes: Vec<u8>,
    terrain: Bank,
    controls: Vec<u8>,
    chapter_aux: Option<NativeChapterAuxSurface>,
    parallax: Option<NativeMapParallaxSurface>,
}

struct Assets {
    units: Bank,
    range: Bank,
    luts: Vec<Vec<u8>>,
    frames: NativeMapHUDFrames,
    spawn: Vec<LMI1Entry>,
    digits: Vec<LMI1Entry>,
    fdother6: Vec<LMI1Entry>,
    maps: Vec<MapAssets>,
}

fn load_assets(assets: &Path, pack: &Path) -> Assets {
    let p = |rel: &str| pack.join(rel);
    let mut maps = Vec::new();
    for m in 0..fdicon::SEPARATED_FDSHAP_MAP_COUNT {
        let field: Value =
            serde_json::from_slice(&std::fs::read(assets.join(format!("maps/map{m}/map.json"))).unwrap()).unwrap();
        let tiles = field["tiles"].as_array().unwrap().iter().map(i).collect();
        let modes = field["native_tile_blit_modes"].as_array().unwrap().iter().map(|v| i(v) as u8).collect();
        let (terrain, controls) = fdicon::load_separated_fdshap_bank(p("tilesets/fdshap"), m).unwrap();
        let chapter_aux = fdother::native_chapter_aux_surface_for(m)
            .map(|c| fdother::load_separated_chapter_aux_surface_resource(p("surfaces"), &c).unwrap());
        let parallax = (m == 17).then(|| fdother::load_separated_native_map_parallax(p("surfaces")).unwrap());
        maps.push(MapAssets { width: i(&field["w"]), tiles, modes, terrain, controls, chapter_aux, parallax });
    }
    Assets {
        units: fdicon::load_separated_bank(p("sprites/fdicon")).unwrap(),
        range: fdother::load_separated_range_overlay_bank(p("sprites/fdother_001_range_overlay")).unwrap(),
        luts: fdother::load_separated_lut_bank(p("palette")).unwrap(),
        frames: load_separated_native_map_hud_frames(p("ui")).unwrap(),
        spawn: fdother::load_separated_lmi1_bank(
            p("animations/fdother_009_spawn_intro"),
            fdother::NATIVE_SPAWN_INTRO_FRAME_RESOURCE,
        )
        .unwrap(),
        digits: fdother::load_separated_lmi1_bank(
            p("ui/fdother_005_lmi1_opaque"),
            fdother::NATIVE_COMMAND_HEAL_TAIL_DIGIT_RESOURCE,
        )
        .unwrap(),
        fdother6: fdother::load_separated_lmi1_bank(
            p("effects/fdother_006_lmi1_opaque"),
            fdother::NATIVE_COMMAND_HEAL_TAIL_EFFECT_RESOURCE,
        )
        .unwrap(),
        maps,
    }
}

/// 黃金檔記錄的局面，加上從真實素材重建的輸入。
struct Scenario<'a> {
    json: &'a Value,
    map: &'a MapAssets,
    cells: Vec<NativeTerrainCell>,
    cache: NativeSelectorCache,
    units: Vec<NativeUnitLayerEntry>,
    fg: Vec<NativeForegroundLayerEntry>,
    hud: NativeMapHUDInput,
    lut: &'a [u8],
}

impl<'a> Scenario<'a> {
    fn new(a: &'a Assets, json: &'a Value) -> Self {
        let map = &a.maps[i(&json["map"]) as usize];
        let cells = build_native_terrain_cells(&map.tiles, &map.modes).unwrap();
        let keys: Vec<u8> = json["keys"].as_array().unwrap().iter().map(|k| i(k) as u8).collect();
        let cache = NativeSelectorCache::from_keys(&keys).unwrap();
        let units = json["units"]
            .as_array()
            .unwrap()
            .iter()
            .map(|u| NativeUnitLayerEntry {
                x: i(&u["x"]),
                y: i(&u["y"]),
                slot: i(&u["slot"]),
                pose: i(&u["pose"]),
                motion_offset: i(&u["motion"]),
                flags: i(&u["flags"]) as u8,
                force_base: b(&u["force"]),
                inactive: b(&u["inactive"]),
            })
            .collect();
        let fg = json["fg"]
            .as_array()
            .unwrap()
            .iter()
            .map(|u| NativeForegroundLayerEntry {
                x: i(&u["x"]),
                y: i(&u["y"]),
                pose: i(&u["pose"]) as u8,
                motion_offset: i(&u["motion"]),
                inactive: b(&u["inactive"]),
                unit7: i(&u["unit7"]) as u8,
                race: i(&u["race"]) as u8,
                class: i(&u["class"]) as u8,
            })
            .collect();
        let h = &json["hud"];
        let hud = NativeMapHUDInput {
            display_gate_a: b(&h["gate_a"]),
            display_gate_b: b(&h["gate_b"]),
            anchor_x: i(&h["anchor"]),
            terrain_descriptor: i(&h["descriptor"]),
            terrain_control: i(&h["control"]) as u8,
            optional_unit: (!h["unit"].is_null()).then(|| NativeMapHUDOptionalUnit {
                selector_slot: i(&h["unit"]["slot"]),
                raw_state: i(&h["unit"]["state"]),
                current: i(&h["unit"]["current"]) as u16,
                maximum: i(&h["unit"]["maximum"]) as u16,
            }),
        };
        let lut = &a.luts[i(&json["lut"]) as usize];
        Scenario { json, map, cells, cache, units, fg, hud, lut }
    }

    fn frame(&self, a: &'a Assets) -> FrameInput<'_> {
        let j = self.json;
        FrameInput {
            terrain_bank: Some(&self.map.terrain),
            range_bank: Some(&a.range),
            unit_bank: Some(&a.units),
            foreground_bank: Some(&self.map.terrain),
            selector_cache: Some(&self.cache),
            cells: self.cells.clone(),
            controls: &self.map.controls,
            lut: self.lut,
            map_width: self.map.width,
            camera_x: i(&j["camera_x"]),
            camera_y: i(&j["camera_y"]),
            flip: i(&j["flip"]),
            terrain_cycle: i(&j["terrain_cycle"]),
            idle_cycle: i(&j["idle"]),
            moving_cycle: i(&j["moving"]),
            pixel_shift: i(&j["pixel_shift"]),
            range_mode: i(&j["range_mode"]),
            cursor_x: i(&j["cursor_x"]),
            cursor_y: i(&j["cursor_y"]),
            units: self.units.clone(),
            foreground_units: self.fg.clone(),
            parallax: if b(&j["parallax"]) { self.map.parallax.as_ref() } else { None },
            parallax_scroll_x: i(&j["scroll_x"]),
            parallax_scroll_y: i(&j["scroll_y"]),
            chapter_aux: if b(&j["aux"]) { self.map.chapter_aux.as_ref() } else { None },
            chapter_aux_phase: i(&j["aux_phase"]),
            selection_overlay: None,
        }
    }

    fn transition(&self, a: &'a Assets) -> NativeTransitionFrameInput<'_> {
        let f = self.frame(a);
        NativeTransitionFrameInput {
            parallax: f.parallax,
            parallax_scroll_x: f.parallax_scroll_x,
            parallax_scroll_y: f.parallax_scroll_y,
            terrain_bank: f.terrain_bank,
            unit_bank: f.unit_bank,
            foreground_bank: f.foreground_bank,
            selector_cache: f.selector_cache,
            cells: f.cells,
            controls: f.controls,
            terrain_lut: f.lut,
            map_width: f.map_width,
            camera_x: f.camera_x,
            camera_y: f.camera_y,
            flip: f.flip,
            terrain_cycle: f.terrain_cycle,
            idle_cycle: f.idle_cycle,
            moving_cycle: f.moving_cycle,
            pixel_shift: f.pixel_shift,
            units: f.units,
            foreground_units: f.foreground_units,
        }
    }

    fn work_salt(&self) -> i64 {
        i(&self.json["work_salt"])
    }

    fn buffers(&self) -> (Vec<u8>, Vec<u8>) {
        (fill(WORK, self.work_salt()), fill(VGA, i(&self.json["vga_salt"])))
    }
}

fn stage_observer(stages: &mut Vec<Value>) -> impl FnMut(FrameStage, &mut [u8], &mut [u8]) -> Result<()> + '_ {
    move |stage, work, vga| {
        stages.push(serde_json::json!([stage.0, digest(work), digest(vga)]));
        Ok(())
    }
}

fn check(label: &str, field: &str, go: &Value, rust: &Value) {
    let empty = |v: &Value| v.is_null() || v.as_array().is_some_and(Vec::is_empty);
    if empty(go) && empty(rust) {
        return;
    }
    assert_eq!(rust, go, "{label} {field}");
}

fn check_buffers(label: &str, want: &Value, err: Value, work: &[u8], vga: &[u8]) {
    check(label, "error", &want["error"], &err);
    check(label, "work", &want["work"], &Value::String(digest(work)));
    check(label, "vga", &want["vga"], &Value::String(digest(vga)));
}

fn targets_from(v: &Value) -> Vec<NativeCommandHealTailTarget> {
    v.as_array()
        .map(Vec::as_slice)
        .unwrap_or_default()
        .iter()
        .map(|t| NativeCommandHealTailTarget {
            record_index: i(&t[0]),
            x: i(&t[1]),
            y: i(&t[2]),
            selector_slot: i(&t[3]),
        })
        .collect()
}

fn layers_json(layers: &[NativeCommandTailLayer]) -> Value {
    Value::Array(layers.iter().map(|l| serde_json::json!([l.resource, l.entry, l.x, l.y])).collect())
}

fn run_scenario_op(a: &Assets, op: &str, input: &Value, want: &Value, label: &str) {
    let s = Scenario::new(a, &input["scenario"]);
    match op {
        "frame" => {
            let (mut work, mut vga) = s.buffers();
            let mut in_ = NativeFrameInput {
                frame: s.frame(a),
                hud: s.hud,
                frames: a.frames.clone(),
                hud_terrain: Some(&s.map.terrain),
                hud_units: Some(&a.units),
                hud_cache: Some(&s.cache),
            };
            let mut stages = Vec::new();
            let r =
                compose_native_frame_observed(&mut work, &mut vga, &mut in_, Some(&mut stage_observer(&mut stages)));
            check_buffers(label, want, errv(&r), &work, &vga);
            check(label, "cells", &want["cells"], &Value::String(cells_digest(&in_.frame.cells)));
            check(label, "stages", &want["stages"], &Value::Array(stages));
        }
        "step" => {
            let (mut work, mut vga) = s.buffers();
            let mut stages = Vec::new();
            let r = compose_native_step_frame_observed(
                &mut work,
                &mut vga,
                &s.frame(a),
                Some(&mut stage_observer(&mut stages)),
            );
            check_buffers(label, want, errv(&r), &work, &vga);
            check(label, "stages", &want["stages"], &Value::Array(stages));
        }
        "upstep" => {
            let mut in_ = s.frame(a);
            if b(&input["strip_layers"]) {
                (in_.parallax, in_.chapter_aux) = (None, None);
            }
            let (unit_y, motion) = (i(&input["unit_y"]), i(&input["motion"]));
            let (mut work, mut vga) = s.buffers();
            let mut stages = Vec::new();
            let r = compose_native_up_step_frame_observed(
                &mut work,
                &mut vga,
                &in_,
                unit_y,
                motion,
                Some(&mut stage_observer(&mut stages)),
            );
            check_buffers(label, want, errv(&r), &work, &vga);
            check(label, "stages", &want["stages"], &Value::Array(stages));
            let layout = native_up_step_layout(&in_, unit_y, motion);
            check(label, "layout_error", &want["layout_error"], &errv(&layout));
            let (tb, tcy, rows, vb) = layout.unwrap_or_default();
            check(label, "layout", &want["layout"], &serde_json::json!([tb, tcy, rows, vb]));
        }
        "transition" => {
            let p: Vec<i64> = input["pass"].as_array().unwrap().iter().map(i).collect();
            let pass = fdother::build_native_indexed_transition_pass(p[0], p[1], p[2], p[3], p[4]);
            check(
                label,
                "pass_error",
                &want["pass_error"],
                &pass.as_ref().err().map(|e| Value::String(e.to_string())).unwrap_or_default(),
            );
            if let Ok(pass) = pass {
                let (mut work, mut vga) = s.buffers();
                let r = compose_native_transition_frame(
                    &mut work,
                    &mut vga,
                    &s.transition(a),
                    pass,
                    &a.luts[i(&input["lut2"]) as usize],
                );
                check_buffers(label, want, errv(&r), &work, &vga);
            }
        }
        "unit_present" => run_unit_present(a, &s, input, want, label),
        "command10" => {
            let (mut work, mut vga) = s.buffers();
            let r = compose_native_command10_to12_surface(
                &mut work,
                &mut vga,
                &s.transition(a),
                i(&input["x"]),
                i(&input["y"]),
                i(&input["inc"]),
            );
            check_buffers(label, want, errv(&r), &work, &vga);
        }
        "heal_effect" | "heal_mask" | "heal_blend" | "heal_digit" => run_heal(a, &s, op, input, want, label),
        "spawn" => {
            let in_ = NativeSpawnIntroFrameInput { frame: s.transition(a), old_unit_count: i(&input["old"]) };
            let (mut work, mut vga) = s.buffers();
            let mut snapshot = seed_native_spawn_intro_snapshot(&fill(WORK, s.work_salt() + 6000)).unwrap();
            let mut steps = Vec::new();
            for step in fdother::native_spawn_intro_schedule() {
                let entry = &a.spawn[(step.lmi_entry as usize) % a.spawn.len()];
                let r = compose_native_spawn_intro_pass(&mut work, &mut vga, &mut snapshot, &in_, entry, step);
                steps.push(serde_json::json!([step.pass, errv(&r), digest(&work), digest(&vga), digest(&snapshot)]));
            }
            let mut bad = fdother::native_spawn_intro_schedule()[3];
            bad.delay_ticks += 1;
            let r = compose_native_spawn_intro_pass(&mut work, &mut vga, &mut snapshot, &in_, &a.spawn[0], bad);
            steps.push(serde_json::json!([-1, errv(&r), digest(&work), digest(&vga), digest(&snapshot)]));
            check(label, "steps", &want["steps"], &Value::Array(steps));
        }
        "hud" => run_hud(a, &s, want, label),
        other => panic!("未知 op {other}"),
    }
}

fn run_unit_present(a: &Assets, s: &Scenario, input: &Value, want: &Value, label: &str) {
    let in_ = s.transition(a);
    let mut steps = Vec::new();
    let mut step = |name: &str, err: Value, bufs: &[&[u8]]| {
        let mut row = vec![Value::String(name.to_string()), err];
        row.extend(bufs.iter().map(|b| Value::String(digest(b))));
        steps.push(Value::Array(row));
    };
    let (mut work, mut vga) = s.buffers();
    let mut snapshot = fill(WORK, s.work_salt() + 1000);
    let r = compose_native_unit_present_terrain_snapshot(&mut snapshot, &in_);
    step("terrain_snapshot", errv(&r), &[&snapshot]);
    let mut staging = fill(WORK, s.work_salt() + 2000);
    let r = compose_native24_b4_d_staging(&mut staging, &in_);
    step("24b4d", errv(&r), &[&staging]);
    let mut redraw = fill(WORK, s.work_salt() + 3000);
    let r = redraw_native_unit_present_objects(&mut redraw, &in_);
    step("redraw", errv(&r), &[&redraw]);
    let (map_x, map_y) = (i(&input["map_x"]), i(&input["map_y"]));
    let entry = &a.fdother6[i(&input["entry"]) as usize];
    let r = compose_native_unit_present_intro_frame(&mut work, &mut vga, &snapshot, &in_, entry, map_x, map_y);
    step("intro", errv(&r), &[&work, &vga]);
    let mut lut_snap = fill(WORK, s.work_salt() + 4000);
    let r = compose_native_unit_present_lut_snapshot(&mut lut_snap, &snapshot, &in_, &a.fdother6[0x7c], map_x, map_y);
    step("lut_snapshot", errv(&r), &[&lut_snap]);
    let (raw9, raw_d) = (map_x - in_.camera_x, map_y - in_.camera_y);
    let frames = fdother::native_unit_present_lut_frames(raw9, raw_d);
    step("lut_frames", frames.as_ref().err().map(|e| Value::String(e.to_string())).unwrap_or_default(), &[]);
    if let Ok(frames) = &frames {
        let frame = &frames[i(&input["frame"]) as usize];
        let r = compose_native_unit_present_lut_frame(
            &mut work,
            &mut vga,
            &lut_snap,
            &a.luts[i(&input["lut"]) as usize],
            &in_,
            frame,
        );
        step("lut_frame", errv(&r), &[&work, &vga]);
    }
    let mut rows = Vec::new();
    let mut delay = |row: i64, _: &[u8]| -> Result<()> {
        rows.push(Value::from(row));
        Ok(())
    };
    let r = compose_native_unit_present_strip_bridge(
        &mut work,
        &mut vga,
        &lut_snap,
        &a.luts[i(&input["bridge_lut"]) as usize],
        &in_,
        map_x,
        map_y,
        raw9,
        raw_d,
        Some(&mut delay),
    );
    step("bridge", errv(&r), &[&work, &vga]);
    check(label, "steps", &want["steps"], &Value::Array(steps));
    check(label, "rows", &want["rows"], &Value::Array(rows));
    let l = native_unit_present_strip_layout_for(map_x, map_y, in_.camera_x, in_.camera_y);
    check(label, "layout", &want["layout"], &serde_json::json!([l.work_offset, l.vga_offset, l.rows]));
}

fn run_heal(a: &Assets, s: &Scenario, op: &str, input: &Value, want: &Value, label: &str) {
    let targets = targets_from(&input["targets"]);
    let (cx, cy) = (i(&s.json["camera_x"]), i(&s.json["camera_y"]));
    let snapshot = fill(WORK, s.work_salt() + 5000);
    let (mut work, mut vga) = s.buffers();
    match op {
        "heal_effect" => {
            let entry = i(&input["entry"]);
            let r = compose_native_command_heal_effect_frame_with_sources(
                &mut work,
                &mut vga,
                &snapshot,
                &a.fdother6[entry as usize],
                &targets,
                cx,
                cy,
                6,
                entry,
            );
            check_buffers(label, want, errv(&r), &work, &vga);
            check(label, "layers", &want["layers"], &layers_json(r.as_deref().unwrap_or_default()));
        }
        "heal_mask" => {
            let r = compose_native_command_heal_mask_frame(
                &mut work,
                &mut vga,
                &snapshot,
                Some(&a.units),
                Some(&s.cache),
                &targets,
                cx,
                cy,
                i(&input["cycle"]),
                i(&input["index"]) as u8,
            );
            check_buffers(label, want, errv(&r), &work, &vga);
        }
        "heal_blend" => {
            let r = compose_native_ai_item_damage_blend_frame(
                &mut work,
                &mut vga,
                &snapshot,
                Some(&a.units),
                Some(&s.cache),
                &targets,
                cx,
                cy,
                i(&input["idle"]),
                i(&input["blend"]),
                i(&input["base"]) as u8,
            );
            check_buffers(label, want, errv(&r), &work, &vga);
        }
        _ => {
            let queue: Vec<NativePresentationDigit> = input["queue"]
                .as_array()
                .map(Vec::as_slice)
                .unwrap_or_default()
                .iter()
                .map(|q| NativePresentationDigit { position_code: i(&q[0]), target: i(&q[1]), digit: i(&q[2]) })
                .collect();
            let mut vertical = [0i64; 25];
            for (k, v) in input["vertical"].as_array().unwrap().iter().enumerate() {
                vertical[k] = i(v);
            }
            let r = compose_native_command_heal_digit_frame_with_sources(
                &mut work,
                &mut vga,
                &snapshot,
                &a.digits,
                &queue,
                &targets,
                cx,
                cy,
                vertical,
                i(&input["frame"]),
            );
            check_buffers(label, want, errv(&r), &work, &vga);
            check(label, "layers", &want["layers"], &layers_json(r.as_deref().unwrap_or_default()));
        }
    }
}

fn run_hud(a: &Assets, s: &Scenario, want: &Value, label: &str) {
    let stride = fdicon::NATIVE_MAP_STRIDE as usize;
    let mut rows = want["rows"].as_array().unwrap().iter();
    for k in 0..4 {
        let row = rows.next().unwrap();
        let mut hud = s.hud;
        hud.anchor_x = i(&row[0]);
        hud.terrain_control = i(&row[1]) as u8;
        assert!(k > 0 || (hud.anchor_x == s.hud.anchor_x && hud.terrain_control == s.hud.terrain_control), "{label}");
        let mut dst = fill(stride * 200, s.work_salt() + 7000 + k);
        let r = blit_native_map_hud(&a.frames, Some(&s.map.terrain), Some(&a.units), Some(&s.cache), &mut dst, &hud);
        check(label, "hud", &serde_json::json!([row[2], row[3]]), &serde_json::json!([errv(&r), digest(&dst)]));
    }
    for v in [-100i64, -99, -12, -1, 0, 5, 9, 10, 99, 100] {
        let row = rows.next().unwrap();
        assert_eq!(i(&row[1]), v, "{label}");
        let origin = i(&row[2]);
        let mut dst = fill(stride * 30, s.work_salt() + v);
        let r = blit_native_map_hud_two_digit_number(&a.frames, &mut dst, origin, v);
        check(label, "two", &serde_json::json!([row[3], row[4]]), &serde_json::json!([errv(&r), digest(&dst)]));
    }
    for (cur, max) in [(7u16, 7u16), (7, 8), (999, 999), (1000, 1000), (1000, 1001), (0, 1), (65535, 0)] {
        let row = rows.next().unwrap();
        let anchor = i(&row[3]);
        let mut dst = fill(stride * 200, s.work_salt() + cur as i64);
        let r = blit_native_map_hud_hp(&a.frames, &mut dst, anchor, cur, max);
        check(label, "hp", &serde_json::json!([row[4], row[5]]), &serde_json::json!([errv(&r), digest(&dst)]));
    }
    assert!(rows.next().is_none(), "{label} rows 多於預期");
}

fn run_scalars(a: &Assets, want: &Value) {
    for row in want["banner"].as_array().unwrap() {
        let step = i(&row[0]);
        let got = serde_json::json!([
            step,
            phase_banner_block(step),
            phase_banner_dim(step),
            phase_banner_slide_in_offset(step),
            phase_banner_slide_out_offset(step)
        ]);
        assert_eq!(&got, row, "banner {step}");
    }
    let consts = serde_json::json!([
        PHASE_BANNER_STEPS,
        PHASE_BANNER_ENTER_STEPS,
        PHASE_BANNER_EXIT_STEPS,
        PHASE_BANNER_SLIDE_IN_STEPS,
        PHASE_BANNER_SLIDE_OUT_STEPS,
        PHASE_BANNER_SLIDE_ORIGIN_X,
        PHASE_BANNER_DIM_FIRST_INDEX,
        PHASE_BANNER_DIM_LAST_INDEX,
        NATIVE_MAP_VIEWPORT_X0,
        NATIVE_MAP_VIEWPORT_Y0,
        NATIVE_MAP_VIEWPORT_X1,
        NATIVE_MAP_VIEWPORT_Y1,
        NATIVE_MAP_VGA_SIZE,
        NATIVE_UNIT_PRESENT_WORK_SIZE
    ]);
    assert_eq!(consts, want["consts"], "consts");
    assert_eq!(PHASE_BANNER_STEP_MILLIS.to_bits(), want["millis"].as_f64().unwrap().to_bits(), "PhaseBannerStepMillis");
    assert_eq!(
        PHASE_BANNER_STEP_TICK_HZ.to_bits(),
        want["tick_hz"].as_f64().unwrap().to_bits(),
        "PhaseBannerStepTickHz"
    );
    for r in want["anchors"].as_array().unwrap() {
        assert_eq!(advance_native_map_hud_anchor(i(&r[0]), i(&r[1]), i(&r[2])), i(&r[3]), "anchor {r}");
    }
    for r in want["states"].as_array().unwrap() {
        assert_eq!(advance_native_map_hud_state(i(&r[0]), i(&r[1]), i(&r[2])), (i(&r[3]), i(&r[4])), "state {r}");
    }
    for r in want["cycles"].as_array().unwrap() {
        let got = advance_native_map_sprite_cycles(i(&r[0]), i(&r[1]), i(&r[2]), i(&r[3]));
        assert_eq!(got, (i(&r[4]), i(&r[5]), i(&r[6])), "cycles {r}");
    }
    for r in want["eligible"].as_array().unwrap() {
        let got = native_map_hud_optional_unit_eligible(i(&r[0]) as u8, i(&r[1]) as u8, i(&r[2]) as u8);
        assert_eq!(got, b(&r[3]), "eligible {r}");
    }
    for r in want["strips"].as_array().unwrap() {
        let l = native_unit_present_strip_layout_for(i(&r[0]), i(&r[1]), i(&r[2]), i(&r[3]));
        assert_eq!((l.work_offset, l.vga_offset, l.rows), (i(&r[4]), i(&r[5]), i(&r[6])), "strip {r}");
    }
    for r in want["apdp"].as_array().unwrap() {
        let got = native_map_hud_terrain_apdp(i(&r[0]) as u8);
        let (ap, dp) = got.as_ref().map(|v| *v).unwrap_or_default();
        assert_eq!(serde_json::json!([i(&r[0]), ap, dp, errv(&got)]), *r, "apdp");
    }
    let cells = want["cells"].as_array().unwrap();
    for (m, row) in a.maps.iter().zip(cells) {
        let got = build_native_terrain_cells(&m.tiles, &m.modes);
        check("cells", "error", &row[1], &errv(&got));
        check("cells", "digest", &row[2], &Value::String(cells_digest(got.as_deref().unwrap_or_default())));
    }
    for row in &cells[a.maps.len()..] {
        let tiles: Vec<i64> = row[0].as_array().map(Vec::as_slice).unwrap_or_default().iter().map(i).collect();
        let modes: Vec<u8> =
            row[1].as_array().map(Vec::as_slice).unwrap_or_default().iter().map(|v| i(v) as u8).collect();
        let got = build_native_terrain_cells(&tiles, &modes);
        check("cells", "error", &row[2], &errv(&got));
        check("cells", "digest", &row[3], &Value::String(cells_digest(got.as_deref().unwrap_or_default())));
    }
}

fn require_golden() -> bool {
    std::env::var_os("FD2_REQUIRE_GOLDEN").is_some_and(|v| v == "1")
}

fn env_dir(name: &str) -> Option<PathBuf> {
    std::env::var_os(name).map(PathBuf::from).filter(|p| p.exists())
}

#[test]
fn golden_present_map() {
    let _ = (KNOWN_DIFFERENCES, NORMALIZATIONS);
    let (Some(path), Some(assets), Some(pack)) =
        (env_dir("FD2_GOLDEN_PRESENT_MAP"), env_dir("FD2_GOLDEN_ASSETS"), env_dir("FD2_GOLDEN_PACK"))
    else {
        assert!(
            !require_golden(),
            "FD2_REQUIRE_GOLDEN=1 但缺 FD2_GOLDEN_PRESENT_MAP／FD2_GOLDEN_ASSETS／FD2_GOLDEN_PACK"
        );
        eprintln!("略過：沒有黃金檔或真實輸入");
        return;
    };
    let golden: Value = serde_json::from_slice(&std::fs::read(&path).unwrap()).unwrap();
    assert_eq!(golden["schema"], "fd2-golden-present-map-v1");
    for f in golden["files"].as_array().unwrap() {
        let label = f["path"].as_str().unwrap();
        let file = match label.strip_prefix("pack/") {
            Some(rel) => pack.join(rel),
            None => assets.join(label),
        };
        let raw = std::fs::read(&file).unwrap_or_else(|e| panic!("讀不到真實輸入 {label}: {e}"));
        assert_eq!(raw.len() as u64, f["size"].as_u64().unwrap(), "{label} 大小");
        assert_eq!(digest(&raw), f["sha256"].as_str().unwrap(), "{label} SHA-256");
    }
    let a = load_assets(&assets, &pack);
    let mut counts = std::collections::BTreeMap::<String, usize>::new();
    for case in golden["cases"].as_array().unwrap() {
        let (op, input, want) = (case["op"].as_str().unwrap(), &case["input"], &case["result"]);
        let label = format!("#{} {op}", case["index"]);
        match op {
            "mosaic" => {
                let salt = i(&input["salt"]);
                let src = fill(VGA, salt);
                for row in want["rows"].as_array().unwrap() {
                    let mut dst = fill(VGA, salt + 100);
                    let r = mosaic_native_map_viewport(&mut dst, &src, i(&row[0]));
                    check(
                        &label,
                        "mosaic",
                        &serde_json::json!([row[1], row[2]]),
                        &serde_json::json!([errv(&r), digest(&dst)]),
                    );
                }
            }
            "ch23" => {
                let mut work = fill(WORK, 9200);
                let r = seed_native_ch23_staging(&mut work, &fill(312 * 192, 9201));
                check(&label, "error", &want["error"], &errv(&r));
                check(&label, "work", &want["work"], &Value::String(digest(&work)));
            }
            "scalars" => run_scalars(&a, want),
            _ => run_scenario_op(&a, op, input, want, &label),
        }
        *counts.entry(op.to_string()).or_default() += 1;
    }
    let total: usize = counts.values().sum();
    eprintln!("present-map 黃金：比對 {total} 案例 {counts:?}");
    assert_eq!(total, golden["cases"].as_array().unwrap().len());
}
