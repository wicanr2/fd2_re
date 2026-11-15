//! Go 來源：`remake/cmd/fd2/native_confirmation_mouth.go`（81 行）。
//!
//! 骨架：Go 檔的 4 個函式中產生 3 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

impl Game {
    /// Only confirmed YESNO waiting owners enter sub_19953's mouth loop.
    /// Church and arbitrary-key messages retain their independent owners.
    ///
    /// Go：`native_confirmation_mouth.go:12`，`func (g *Game) nativeConfirmationMouthWaitingOwner() any`
    ///
    /// R4 B 組：Go 回傳 `any`，三種擁有者是 `g.nativeSystemEndTurnUI` 指標、`"confirmation:"+節點 id`、
    /// `"prompt:"+節點 id`，以 `==` 比較。Rust 對應成 [`OwnerToken`]：結束回合介面是固定標記
    /// `NATIVE_CONFIRMATION_OWNER_END_TURN`，字串擁有者是字串的 FNV-1a 雜湊（最高位元清零，與固定標記錯開）。
    /// Go 以指標區分前後兩個結束回合介面，Rust 視為同一擁有者；兩個介面之間一定經過沒有擁有者的幀，
    /// 嘴型狀態照樣重來（KNOWN_DIFFERENCES `confirmation-mouth-owner-token`）。
    pub(crate) fn native_confirmation_mouth_waiting_owner(&self) -> Option<OwnerToken> {
        if self.native_class_ui_job.is_some() {
            return None;
        }
        if self.native_system_end_turn_confirm && self.native_system_end_turn_ui.is_some() {
            return Some(NATIVE_CONFIRMATION_OWNER_END_TURN);
        }
        let camp = self.camp.as_ref()?;
        if self.prep_required_missing.is_some() {
            return None;
        }
        let node = camp.node()?;
        if node.r#type != "preparation" {
            return None;
        }
        if self.prep_confirm {
            return Some(confirmation_owner_token(&format!("confirmation:{}", camp.node_id())));
        }
        if self.native_preparation_prompt_active() {
            return Some(confirmation_owner_token(&format!("prompt:{}", camp.node_id())));
        }
        None
    }

    /// Go：`native_confirmation_mouth.go:35`，`func (g *Game) sampleNativeConfirmationMouthRandom() int`
    ///
    /// R4 B 組：Go 以 `rand.Int63()`（全域來源）當種子建立私有亂數；Rust 用固定種子
    /// `CONFIRMATION_MOUTH_RNG_SEED`（KNOWN_DIFFERENCES `cosmetic-mouth-rng-fixed-seed`）。
    ///
    /// R5 階段 2 r5d：`go_global_rand` 有值（畫格擷取打開）時照 Go 以它的 `Int63()` 當種子，與故事對白嘴型
    /// 共用同一個全域序列。
    pub(crate) fn sample_native_confirmation_mouth_random(&mut self) -> i64 {
        if self.native_confirmation_mouth_rng.is_none() {
            // Cosmetic RNG stays independent of the native battle RNG and SAV.
            let seed = match self.go_global_rand.as_mut() {
                Some(global) => global.int63(),
                None => CONFIRMATION_MOUTH_RNG_SEED,
            };
            self.native_confirmation_mouth_rng = Some(battle::GoMathRand::new(seed));
        }
        self.native_confirmation_mouth_rng.as_mut().unwrap().intn(30)
    }

    /// Go：`native_confirmation_mouth.go:43`，`func (g *Game) stepNativeConfirmationMouth(owner any, advanced bool)`
    pub(crate) fn step_native_confirmation_mouth(&mut self, owner: Option<OwnerToken>, advanced: bool) {
        let Some(owner) = owner else {
            self.native_confirmation_mouth_owner = None;
            return;
        };
        if Some(owner) != self.native_confirmation_mouth_owner {
            let sample = self.sample_native_confirmation_mouth_random();
            match dato::ConfirmationMouthState::new(sample) {
                Ok(state) => {
                    (self.native_confirmation_mouth, self.native_confirmation_mouth_owner) = (state, Some(owner));
                }
                Err(err) => {
                    self.load_err = err.to_string();
                    return;
                }
            }
        }
        if !advanced {
            return;
        }
        let mut sample = 0;
        if self.native_confirmation_mouth.open {
            sample = self.sample_native_confirmation_mouth_random();
        }
        match self.native_confirmation_mouth.tick(sample) {
            Ok(state) => self.native_confirmation_mouth = state,
            Err(err) => self.load_err = err.to_string(),
        }
    }
}

/// R4 B 組新增：確認嘴型亂數的固定種子（與故事對白嘴型、遊戲亂數錯開）。
pub(crate) const CONFIRMATION_MOUTH_RNG_SEED: i64 = VIRTUAL_CLOCK_ORIGIN.0 + 2;

/// R4 B 組新增：結束回合確認介面當擁有者時的固定標記（最高位元為 1，字串雜湊不會撞上）。
pub(crate) const NATIVE_CONFIRMATION_OWNER_END_TURN: OwnerToken = OwnerToken(1 << 63);

/// R4 B 組新增：字串擁有者（Go 的 `"confirmation:"+id`、`"prompt:"+id`）的 64 位元 FNV-1a 雜湊，最高位元清零。
fn confirmation_owner_token(owner: &str) -> OwnerToken {
    let mut hash: u64 = 0xcbf2_9ce4_8422_2325;
    for byte in owner.bytes() {
        hash ^= byte as u64;
        hash = hash.wrapping_mul(0x0000_0100_0000_01b3);
    }
    OwnerToken(hash & !(1 << 63))
}

#[cfg(test)]
mod native_confirmation_mouth_test;

// ---- R5 階段 1 骨架（tools/rust_r5_skeleton.sh 產生；本體由 R5 階段 2 移植）----

impl Game {
    /// Go：`native_confirmation_mouth.go:71`，`func (g *Game) composeNativeConfirmationMouth(frame []byte, portraits []dato.Frame) ([]byte, bool)`
    ///
    /// R5 階段 2 r5d：Go 不成立時回傳同一個切片，Rust 回傳複本；`ok` 照 Go。
    pub(crate) fn compose_native_confirmation_mouth(&self, frame: &[u8], portraits: &[dato::Frame]) -> (Vec<u8>, bool) {
        let owner = self.native_confirmation_mouth_waiting_owner();
        if owner.is_none() || owner != self.native_confirmation_mouth_owner || !self.native_confirmation_mouth.open {
            return (frame.to_vec(), frame.len() == 320 * 200);
        }
        if portraits.len() <= 3 {
            return (Vec::new(), false);
        }
        match campaign::compose_native_confirmation_portrait(frame, &portraits[3]) {
            Ok(result) => (result, true),
            Err(_) => (Vec::new(), false),
        }
    }
}
