//! Go 來源：`remake/cmd/fd2/native_chapter_result.go`（82 行）。
//!
//! 骨架：Go 檔的 3 個函式中產生 0 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// nativeChapterResultJob 保留原版同步處理器在非同步介面裡的擁有權：
/// 當前列對白完全收框之後，才讀取下一列的raw條件。
///
/// Go：`native_chapter_result.go:7`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub(crate) struct NativeChapterResultJob {
    pub(crate) next_rule: i64,
    pub(crate) code: i64,
    pub(crate) matched: Vec<String>,
}

impl Game {
    /// Go：`native_chapter_result.go:13`，`func (g *Game) beginNativeChapterResult()`
    pub(crate) fn begin_native_chapter_result(&mut self) {
        if self.native_chapter_result.is_some() || self.battle_event.is_some() || !self.load_err.is_empty() {
            return;
        }
        let code = match battle::native_initial_result_code(self.st.as_ref(), self.sc.as_ref()) {
            Ok(code) => code,
            Err(err) => {
                self.fail_native_chapter_result(&err.to_string());
                return;
            }
        };
        self.native_chapter_result = Some(NativeChapterResultJob { code, ..Default::default() });
        self.advance_native_chapter_result();
    }

    /// Go：`native_chapter_result.go:26`，`func (g *Game) failNativeChapterResult(message string)`
    pub(crate) fn fail_native_chapter_result(&mut self, message: &str) {
        self.load_err = format!("原生章節結果：{message}");
        self.startup_blocked = true;
    }

    /// 當前列對白完全收框之後才讀下一列的 raw 條件；全部條件與對白完成後才把 pending 交給外層
    /// 結果／敗北擁有者。
    ///
    /// Go：`native_chapter_result.go:31`，`func (g *Game) advanceNativeChapterResult()`
    pub(crate) fn advance_native_chapter_result(&mut self) {
        if self.native_chapter_result.is_none() || self.sc.is_none() || self.st.is_none() || !self.load_err.is_empty() {
            return;
        }
        loop {
            let next = self.native_chapter_result.as_ref().unwrap().next_rule;
            let Some(rule) = self.sc.as_ref().unwrap().native_result_rules.get(next as usize).cloned() else {
                break;
            };
            self.native_chapter_result.as_mut().unwrap().next_rule += 1;
            let matched = match rule.r#match(self.st.as_ref()) {
                Ok(matched) => matched,
                Err(err) => {
                    self.fail_native_chapter_result(&format!("{}：{err}", rule.id));
                    return;
                }
            };
            if !matched {
                continue;
            }
            if !rule.actions.is_empty() && !self.native_battle_dialogue_available() {
                self.fail_native_chapter_result(&format!("{}：缺少原生戰場對白狀態", rule.id));
                return;
            }
            let job = self.native_chapter_result.as_mut().unwrap();
            job.code = rule.code;
            job.matched.push(rule.id.clone());
            // sub_20822 的 code1 覆寫沒有對白；不要製造事件擁有者或素材要求。
            if rule.actions.is_empty() {
                continue;
            }
            let mut actions = rule.actions.clone();
            self.start_battle_event(&mut actions, Some(Box::new(|g: &mut Game| g.advance_native_chapter_result())));
            return;
        }
        let job = self.native_chapter_result.take().unwrap();
        self.native_result_matched_rules = job.matched.clone();
        match job.code {
            0 => self.result = String::new(),
            1 => self.result = String::from("lose"),
            2 => self.result = String::from("win"),
            _ => {}
        }
        self.begin_native_defeat_if_configured();
        // main25E02→25E23 在 code2 之後直接進 post；第 29 章不增加玩家確認。
        if self.result == "win" && self.sc.as_ref().is_some_and(|sc| sc.native_result_auto_advance) {
            let edge_ok = self
                .camp
                .as_ref()
                .and_then(|camp| camp.node())
                .is_some_and(|node| node.r#type == "battle" && !node.on_win.is_empty());
            if !edge_ok {
                self.fail_native_chapter_result("automatic victory requires a recovered battle edge");
                return;
            }
            if !self.confirm_battle_result() {
                self.fail_native_chapter_result("automatic victory handoff failed");
            }
        }
    }
}

#[cfg(test)]
mod native_chapter_result_test;
