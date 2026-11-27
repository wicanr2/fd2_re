//! Go 來源：`remake/cmd/fd2/battle_snapshot.go`（215 行）。戰鬥中 F5 的 JSON 戰場快照
//! （56「2026-11-13 #303 戰場JSON快照」）。
//!
//! 讀檔先走正常的戰鬥節點進場，讓所有由資產與劇本綁定的表與一般路徑相同；再以快照覆寫可變狀態，
//! 清掉開場劇本產生的暫態。快照不是原版 FD2.SAV ABI，戰鬥中也不寫原版章節槽。型別 `BattleSnapshot`
//! 在 `save.rs`。

use sha2::{Digest as _, Sha256};

use super::*;

/// Go：`battle_snapshot.go:18`
pub(crate) const BATTLE_SNAPSHOT_VERSION: i64 = 1;

impl Game {
    /// 不能存快照的原因；空字串代表我方可操作的地圖待機。
    ///
    /// Go：`battle_snapshot.go:39`，`func (g *Game) battleSnapshotBlocker() string`
    pub(crate) fn battle_snapshot_blocker(&self) -> &'static str {
        self.battle_snapshot_blocker_for(false)
    }

    /// `from_system_menu` 只放行戰鬥系統選單本身（游標空格開啟的 0x16F55／巢狀 0x19DF7）。選單不屬於戰場
    /// 狀態，還原後回到地圖待機。
    ///
    /// Go：`battle_snapshot.go:43`，`func (g *Game) battleSnapshotBlockerFor(fromSystemMenu bool) string`
    pub(crate) fn battle_snapshot_blocker_for(&self, from_system_menu: bool) -> &'static str {
        let system_menu = from_system_menu && self.native_system_cursor_overlay && self.ring && self.sel.is_none();
        let (Some(camp), Some(st)) = (self.camp.as_ref(), self.st.as_ref()) else {
            return "no battle";
        };
        if camp.node().is_none_or(|node| node.r#type != "battle") {
            return "not a battle node";
        }
        if !self.result.is_empty() {
            return "battle result pending";
        }
        if self.ai_busy || self.native_ai_idle_recovery.is_some() {
            return "enemy phase";
        }
        if self.battle_event.is_some()
            || self.battle_event_delay != 0
            || self.native_turn_staging.is_some()
            || self.native_field_event61.is_some()
            || !self.beats.is_empty()
        {
            return "battle event";
        }
        if !self.dialog.is_empty()
            || self.native_level_up_dialogue.is_some()
            || !self.pending_native_level_ups.is_empty()
        {
            return "dialogue";
        }
        if self.walk.is_some()
            || self.atk.is_some()
            || self.native_move_plan.is_some()
            || self.sel.is_some()
            || self.moved
        {
            return "unit action";
        }
        if !system_menu && (self.ring || self.native_system_cursor_overlay) {
            return "menu open";
        }
        if self.native_command_open
            || self.item_open
            || self.native_item_targeting
            || self.native_command0_targeting
            || self.native_system_end_turn_ui.is_some()
            || self.native_class_ui_job.is_some()
            || self.transient_ui
            || self.native_player_status.is_some()
        {
            return "menu open";
        }
        if self.spawn_intro_transition.is_some()
            || self.indexed_transition.is_some()
            || self.transition_reveal.is_some()
            || self.native_unit_present.is_some()
            || self.act_job.is_some()
            || self.cam_pan.is_some()
            || self.focus_job.is_some()
            || self.native_player_focus.is_some()
            || self.fade.is_some()
            || self.banner_t > 0
            || self.native_ch22_reload.is_some()
            || self.native_ch23_loop.is_some()
        {
            return "presentation";
        }
        if self.native_end_turn_recovery.is_some() {
            return "END recovery";
        }
        if self.sc.as_ref().is_some_and(|sc| sc.pending_join_count() != 0) {
            return "pending JOIN";
        }
        if st.snapshot_ready().is_err() {
            return "battle state not snapshot-ready";
        }
        ""
    }

    /// Go：`battle_snapshot.go:82`，`func (g *Game) captureBattleSnapshot(fromSystemMenu bool) (*battleSnapshot, error)`
    pub(crate) fn capture_battle_snapshot(&self, from_system_menu: bool) -> Result<BattleSnapshot> {
        let reason = self.battle_snapshot_blocker_for(from_system_menu);
        if !reason.is_empty() {
            return Err(game_err!("battle snapshot: {reason}"));
        }
        let st = self.st.as_ref().unwrap();
        let state = st.snapshot()?;
        let camp = self.camp.as_ref().unwrap();
        let node = camp.node().unwrap();
        let (tick, ok) = self.native_map_clock.current();
        Ok(BattleSnapshot {
            native_map_clock_tick: ok.then_some(tick),
            version: BATTLE_SNAPSHOT_VERSION,
            node: camp.cur.clone(),
            units: node.units.clone(),
            scenario: node.scenario.clone(),
            map_sha256: self.battle_snapshot_map_digest(),
            state,
            fired_events: self.sc.as_ref().map(|sc| sc.fired_events()).unwrap_or_default(),
            native_rng_state: self.native_rng_state,
            cursor_x: self.cur_x,
            cursor_y: self.cur_y,
        })
    }

    /// 地圖寬高與每格圖塊的 SHA-256（little-endian uint32）。
    ///
    /// Go：`battle_snapshot.go:107`，`func (g *Game) battleSnapshotMapDigest() string`
    pub(crate) fn battle_snapshot_map_digest(&self) -> String {
        let Some(m) = self.m.as_ref() else {
            return String::new();
        };
        let mut h = Sha256::new();
        h.update((m.w as u32).to_le_bytes());
        h.update((m.h as u32).to_le_bytes());
        for &tile in &m.tiles {
            h.update((tile as u32).to_le_bytes());
        }
        format!("{:x}", h.finalize())
    }

    /// 在動到任何 Game 狀態前檢查快照與戰役圖相符。
    ///
    /// Go：`battle_snapshot.go:124`，`func (g *Game) validateBattleSnapshotNode(snap *battleSnapshot, node string) error`
    pub(crate) fn validate_battle_snapshot_node(&self, snap: &BattleSnapshot, node: &str) -> Result<()> {
        if snap.version != BATTLE_SNAPSHOT_VERSION {
            return Err(game_err!("戰場快照版本 {} 不支援", snap.version));
        }
        let graph = self.camp.as_ref().and_then(|camp| camp.c.as_ref());
        let n = graph.and_then(|graph| graph.nodes.get(node));
        let Some(n) = n.filter(|n| snap.node == node && n.r#type == "battle") else {
            return Err(game_err!("戰場快照的節點不是存檔節點的戰鬥"));
        };
        if n.units != snap.units || n.scenario != snap.scenario {
            return Err(game_err!("戰場快照的單位或劇本資產與戰役圖不符"));
        }
        Ok(())
    }

    /// 在 `enter_node` 已正常建好同一戰鬥節點後覆寫可變狀態。
    ///
    /// Go：`battle_snapshot.go:140`，`func (g *Game) applyBattleSnapshot(snap *battleSnapshot) error`
    ///
    /// Go 在整份 `Game` 複本上做完再換上。Rust 的 `Game` 不能複製：能預先驗的（地圖、尺寸、狀態與事件還原、
    /// 計時範圍、快照就緒）都在複本上先做；寫回後的視角同步或可操作檢查失敗時，重新進入同一個戰鬥節點，
    /// 結果與 Go 保留剛進場的 `g` 相同。
    pub(crate) fn apply_battle_snapshot(&mut self, snap: &BattleSnapshot) -> Result<()> {
        let (Some(st), Some(camp)) = (self.st.as_ref(), self.camp.as_ref()) else {
            return Err(game_err!("戰場快照：戰鬥節點沒有建立"));
        };
        if camp.cur != snap.node {
            return Err(game_err!("戰場快照：戰鬥節點沒有建立"));
        }
        if self.battle_snapshot_map_digest() != snap.map_sha256 {
            return Err(game_err!("戰場快照：地圖與存檔時不同"));
        }
        let m = self.m.as_ref().unwrap();
        if snap.state.w != m.w || snap.state.h != m.h {
            return Err(game_err!("戰場快照：戰場尺寸與地圖不符"));
        }
        let mut state = st.clone();
        state.restore_snapshot(&snap.state).map_err(|err| game_err!("戰場快照：{err}"))?;
        let mut scenario = self.sc.clone();
        if let Some(sc) = scenario.as_mut() {
            sc.restore_fired_events(&snap.fired_events).map_err(|err| game_err!("戰場快照：{err}"))?;
        } else if !snap.fired_events.is_empty() {
            return Err(game_err!("戰場快照：本戰沒有劇本事件"));
        }
        let mut clock = NativeBIOSClock::default();
        if let Some(tick) = snap.native_map_clock_tick
            && !clock.restore_tick(tick)
        {
            return Err(game_err!("戰場快照：地圖計時超出範圍"));
        }
        if let Err(err) = state.snapshot_ready() {
            return Err(game_err!("戰場快照：還原後不是可操作狀態（{err}）"));
        }

        self.st = Some(state);
        self.sc = scenario;
        self.bind_native_death_programs();
        self.clear_battle_snapshot_transients();
        self.native_rng_state = snap.native_rng_state;
        (self.cur_x, self.cur_y) = (snap.cursor_x, snap.cursor_y);
        self.native_map_clock = clock;
        let view_ok = !self.st.as_ref().unwrap().has_native_map_view_state || self.sync_native_map_view();
        (self.prev_cur_x, self.prev_cur_y) = (self.cur_x, self.cur_y);
        let failure = if !view_ok {
            Some(String::from("戰場快照：地圖視角無法發布"))
        } else {
            let reason = self.battle_snapshot_blocker();
            (!reason.is_empty()).then(|| format!("戰場快照：還原後不是可操作狀態（{reason}）"))
        };
        if let Some(message) = failure {
            self.st = None;
            self.enter_node();
            return Err(game_err!("{message}"));
        }
        Ok(())
    }

    /// 丟棄進場劇本與開場演出產生的暫態；沿 CONTINUE 發布清單（`publishNativeContinueBattle`）。
    ///
    /// Go：`battle_snapshot.go:192`，`func (g *Game) clearBattleSnapshotTransients()`
    pub(crate) fn clear_battle_snapshot_transients(&mut self) {
        self.dialog = Vec::new();
        (self.dlg_shown, self.dlg_phase, self.dlg_t) = (DLG_NONE, 0, 0);
        self.dlg_upper = None;
        (self.dlg_page, self.dlg_scroll_t, self.dlg_scroll_from) = (0, 0, 0);
        (self.beats, self.beat_idx, self.beat_delay) = (Vec::new(), -1, 0);
        (self.cam_pan, self.focus_job, self.act_job) = (None, None, None);
        (self.fade, self.transition_reveal) = (None, None);
        (self.indexed_transition, self.spawn_intro_transition) = (None, None);
        self.native_turn_staging = None;
        self.native_field_event61 = None;
        self.native_ai_idle_recovery = None;
        self.native_unit_present = None;
        (self.battle_event, self.battle_event_delay) = (None, 0);
        (self.atk, self.walk) = (None, None);
        (self.sel, self.reach) = (None, BTreeMap::new());
        (self.moved, self.result) = (false, String::new());
        self.ai_busy = false;
        (self.banner, self.banner_t) = (String::new(), 0);
        self.native_level_up_dialogue = None;
        self.pending_native_level_ups = Vec::new();
        (self.native_player_focus, self.native_player_focus_started) = (None, false);
        (self.native_map_work, self.native_map_vga) = (Vec::new(), Vec::new());
        self.reset_action_overlay_lifecycle();
    }
}

#[cfg(test)]
#[path = "battle_snapshot_test.rs"]
mod tests;
