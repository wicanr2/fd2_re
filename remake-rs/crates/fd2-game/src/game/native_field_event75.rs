//! Go 來源：`remake/cmd/fd2/native_field_event75.go`（92 行）。
//!
//! 骨架：Go 檔的 2 個函式中產生 1 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

impl Game {
    /// beginNativeFieldEvent75 只由成功動作 owner 消費 map28 selector1。
    /// FDTXT_029 index1 必須先完成，之後才可一起提交 live rows 與 0x20-byte
    /// event-state table。
    ///
    /// Go：`native_field_event75.go:48`，`func (g *Game) beginNativeFieldEvent75(actor *battle.Unit, after func()) bool`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    /// Go 的 func 參數 `after` 會被存起來（工作結構的延續），Rust 收擁有的 `Box`。
    pub(crate) fn begin_native_field_event75(
        &mut self,
        actor: Option<UnitIdx>,
        after: Option<Box<dyn FnOnce(&mut Game) + Send>>,
    ) -> bool {
        // R4 B 組：要播對白時呼叫 `event75DialogueActions`（`r8`，第 29 章），停在 `todo!`。
        let (Some(st), Some(actor)) = (self.st.as_ref(), actor) else {
            return false;
        };
        let (x, y) = (st.units[actor].x, st.units[actor].y);
        let (event_id, bound) = battle::native_field_event_id_at(Some(st), x, y, 1);
        if !bound || event_id != 75 {
            return false;
        }
        let plan = match battle::plan_native_field_event75(Some(st), Some(actor), x, y) {
            Ok(plan) => plan,
            Err(err) => {
                self.load_err = err.to_string();
                self.msg = String::from("事件 75 的原版證據不完整，已停止事件寫入");
                if let Some(after) = after {
                    after(self);
                }
                return true;
            }
        };
        if plan.noop {
            if let Some(after) = after {
                after(self);
            }
            return true;
        }
        let Some(mut actions) = event75_dialogue_actions(self, plan.text_index, Some(actor)) else {
            if self.load_err.is_empty() {
                self.load_err = String::from("event75: FDTXT_029 editable dialogue unavailable");
            }
            if let Some(after) = after {
                after(self);
            }
            return true;
        };
        let then: Option<Box<dyn FnOnce(&mut Game) + Send>> = if plan.activate {
            Some(Box::new(move |g: &mut Game| {
                if let Err(err) = battle::commit_native_field_event75(g.st.as_mut(), &plan) {
                    g.load_err = err.to_string();
                }
                if let Some(after) = after {
                    after(g);
                }
            }))
        } else {
            after
        };
        self.start_battle_event(&mut actions, then);
        true
    }
}

/// FDTXT_029 index0（1 句）／index1（5 句）轉成可編輯對白；index0 的說話者取觸發 record 的 BattleFig
/// （sub_35C79 把 raw +7 交給 0x1956B）。
///
/// Go：`native_field_event75.go:7`，`func event75DialogueActions(g *Game, textIndex int, trigger *battle.Unit) ([]battle.Action, bool)`
fn event75_dialogue_actions(g: &mut Game, text_index: i64, trigger: Option<UnitIdx>) -> Option<Vec<battle::Action>> {
    let (line, count) = match text_index {
        0 => (0usize, 1usize),
        1 => (1, 5),
        _ => return None,
    };
    let mut actions = ch29_story_dialogue_actions(g, 0, line, count, "event75")?;
    if text_index == 0 {
        let unit = g.st.as_ref().zip(trigger).map(|(st, index)| &st.units[index])?;
        if !unit.has_battle_fig {
            return None;
        }
        actions[0].speaker = unit.battle_fig;
    }
    Some(actions)
}

/// 全軍移動預檢用（Go 以 BattleFig 0 的探針單位呼叫 `event75DialogueActions`）：探針一定帶 BattleFig，
/// 結果只取決於對白能否轉成動作。
pub(crate) fn event75_dialogue_preflight(g: &mut Game, text_index: i64) -> bool {
    let (line, count) = match text_index {
        0 => (0usize, 1usize),
        1 => (1, 5),
        _ => return false,
    };
    ch29_story_dialogue_actions(g, 0, line, count, "event75").is_some()
}

/// `assets/story/ch29.json` 第 `scene_index` 幕的 `line..line+count` 句轉成對白動作；有 speaker_slot 的句子或
/// 越界時回 `None`（失敗即關閉）。Go 的 event75／event76DialogueActions 共用這段迴圈。
pub(crate) fn ch29_story_dialogue_actions(
    g: &mut Game,
    scene_index: i64,
    line: usize,
    count: usize,
    tag: &str,
) -> Option<Vec<battle::Action>> {
    let lines = load_story_script_with_identity_at("assets/story/ch29.json", "", Some(&scene_index)).ok()?;
    if count == 0 || line + count > lines.len() {
        return None;
    }
    let mut actions = Vec::with_capacity(count);
    for source in &lines[line..line + count] {
        if source.speaker_slot.is_some() {
            return None;
        }
        let text = match g.localized_story_text(source) {
            Ok(text) => text,
            Err(err) => {
                g.load_err = format!("{tag} locale: {err}");
                return None;
            }
        };
        actions.push(battle::Action {
            r#type: String::from("dialogue"),
            speaker: source.speaker,
            text,
            ..Default::default()
        });
    }
    Some(actions)
}
