//! Go `remake/cmd/fd2/chapter_parity_start_test.go`：正常 START 入口（`startFromNormalTitle`、
//! `verifyParityStartSource`）與酒店保存的冷讀（`verifyChapterSlotSave`）。

use super::*;
use std::path::Path;

/// 原版 `FD2.EXE` 的 SHA-256（`docs/data/fd2-reference-files.json`）；Go 測試逐字寫在多處。
pub(crate) const PARITY_EXE_SHA256: &str = "222b7d067ad4450eb9c5f6e6bce1797d54bb050417ba39ced6067f8039f28c4f";

/// 逐行讀 JSONL（空行略過）；讀不到檔回 `Err`。
pub(crate) fn read_jsonl(path: &Path) -> std::io::Result<Vec<serde_json::Value>> {
    let raw = std::fs::read_to_string(path)?;
    Ok(raw
        .lines()
        .filter(|l| !l.trim().is_empty())
        .map(|l| serde_json::from_str(l).unwrap_or_else(|e| panic!("{}：{e}：{l}", path.display())))
        .collect())
}

fn json_str_list(v: &serde_json::Value) -> Vec<String> {
    v.as_array().map(|a| a.iter().filter_map(|s| s.as_str().map(String::from)).collect()).unwrap_or_default()
}

/// Go `chapter_parity_start_test.go:83`：`verifyParityStartSource`，首章 READY 契約
/// （`fd2_ch01_flow_20261017.json` entry_contract）。
pub(crate) fn verify_parity_start_source(run: &Path) -> std::result::Result<(), String> {
    let boot: Option<serde_json::Value> =
        std::fs::read_to_string(run.join("checkpoint-0000.json")).ok().and_then(|raw| serde_json::from_str(&raw).ok());
    let fixed = boot.as_ref().is_some_and(|boot| {
        boot["exe_sha256"] == PARITY_EXE_SHA256
            && boot["kbd_reads"] == 0
            && boot["kbd_pending"] == 0
            && boot["state_injections"].as_array().is_some_and(|a| a.is_empty())
    });
    if !fixed {
        return Err("START缺少固定原版未注入的標題初態".into());
    }
    if !json_str_list(&boot.unwrap()["input_chain"]).iter().any(|a| a == "0x1FE60") {
        return Err("START缺少1FE60標題owner".into());
    }
    let control = read_jsonl(&run.join("control-history.jsonl")).map_err(|e| e.to_string())?;
    let first = control.first();
    if first.is_none_or(|f| f["seq"] != 1 || f["key"] != "enter") {
        return Err("START首鍵不是正常Enter；LOAD／CONTINUE不得冒稱START".into());
    }
    let trace = read_jsonl(&run.join("eip-trace.jsonl")).map_err(|e| e.to_string())?;
    for entry in &trace {
        if entry["eip"] == "0x3231B" && json_str_list(&entry["stack"]).first().is_some_and(|s| s == "0x25F17") {
            return Ok(());
        }
    }
    Err("START缺少25F17→3231B正常序章caller".into())
}

impl ParityReplay {
    /// Go `chapter_parity_start_test.go:17`：`verifyChapterSlotSave`。正常 START（或明示啟用）的章重播，酒店保存
    /// 須寫出原版格式並經新 `Game` 正式 LOAD 冷讀；只驗已知玩家狀態。
    pub(crate) fn verify_chapter_slot_save(&mut self, slot: i64) {
        if std::env::var("FD2_PARITY_ENTRY").unwrap_or_default() != "start"
            && std::env::var("FD2_PARITY_VERIFY_COLD_LOAD").unwrap_or_default() != "1"
        {
            return;
        }
        let g = &self.d.g;
        if g.native_chapter_slot_baseline.is_none() || g.native_chapter_slot_save_err.is_some() {
            panic!("酒店未建立正式保存基底");
        }
        let stored = std::fs::read(native_current_save_path()).expect("讀 FD2.SAV");
        let plain = fdsave::decode(&stored).expect("FD2.SAV 解碼");
        let snapshot = fdsave::inspect_chapter_slot(&plain, slot as usize).expect("章槽");
        if i64::from(snapshot.verified.chapter) != g.handler_chapter
            || i64::from(snapshot.verified.roster_count) != g.party_roster.len() as i64
            || i64::from(snapshot.verified.currency) != g.gold
        {
            panic!("章正式保存 metadata 不同");
        }
        let options = g.current_native_system_options();
        // 用新 Game 經正式標題 LOAD 冷讀正式保存。只驗目前已知玩家狀態。
        let mut cold = load_game().expect("loadGame");
        cold.title_phase = "loadslots".to_string();
        let loaded = cold.confirm_title_load_slot(slot);
        if !loaded
            || !cold.load_err.is_empty()
            || node_id(&cold) != self.current_town()
            || cold.gold != g.gold
            || cold.handler_chapter != g.handler_chapter
            || cold.party_join_order != g.party_join_order
        {
            panic!("章保存正常 LOAD 冷讀失敗：node={} msg={} err={}", node_id(&cold), cold.msg, cold.load_err);
        }
        for (id, expected) in &g.party_roster {
            let ok = cold.party_roster.get(id).is_some_and(|actual| {
                actual.hp == expected.hp
                    && actual.max_hp == expected.max_hp
                    && actual.mp == expected.mp
                    && actual.max_mp == expected.max_mp
                    && actual.lv == expected.lv
                    && actual.exp == expected.exp
                    && actual.ap == expected.ap
                    && actual.dp == expected.dp
                    && actual.dx == expected.dx
                    && actual.hit == expected.hit
                    && actual.ev == expected.ev
                    && actual.mv == expected.mv
                    && actual.inventory_slots == expected.inventory_slots
                    && actual.native_inventory_flags == expected.native_inventory_flags
            });
            if !ok {
                panic!("章保存正常 LOAD 角色 {id} 玩家狀態不同");
            }
        }
        if cold.current_native_system_options() != options {
            panic!("章保存正常 LOAD 系統選項不同");
        }
        let raw = std::fs::read(save_slot_path(slot)).expect("讀重製端存檔");
        std::fs::write(self.out.join("hotel-save.json"), raw).expect("寫 hotel-save.json");
        eprintln!(
            "正式保存與 LOAD 冷讀通過：chapter={} roster={} gold={}",
            snapshot.verified.chapter, snapshot.verified.roster_count, snapshot.verified.currency
        );
    }

    /// Go `chapter_parity_start_test.go:142`：`startFromNormalTitle`。
    pub(crate) fn start_from_normal_title(&mut self) {
        if let Err(err) = verify_parity_start_source(&self.run) {
            panic!("{err}");
        }
        if self.d.g.title_phase != "cutscene" {
            panic!("正常START未從開場動畫開始：{:?}", self.d.g.title_phase);
        }
        let mut trace = new_journey_trace();
        trace.seed = "4".to_string();
        let mut frame = 0;
        while self.d.g.title_phase != "menu" {
            if frame >= JOURNEY_TITLE_FRAMES || self.d.g.title_phase.is_empty() {
                panic!("正常開場動畫未進標題選單");
            }
            trace.observe(&self.d.g);
            ack_presents(&mut self.d.g);
            if let Err(err) = parity_update_game(&mut self.d) {
                panic!("{err}");
            }
            frame += 1;
        }
        if self.d.g.title_sel != 0 || !self.d.g.apply_title_menu_event(TITLE_MENU_CONFIRM) {
            panic!("正常START未由標題owner消費");
        }
        trace.press("title");
        self.start_entry_trace = Some(trace);
        if !self.d.pump(600, |g| g.title_phase.is_empty())
            || self.d.g.camp.is_none()
            || node_id(&self.d.g) != JOURNEY_OPENING_NODE
        {
            panic!("START未正常進入序章");
        }
    }
}
