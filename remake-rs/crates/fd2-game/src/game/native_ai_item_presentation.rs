//! Go 來源：`remake/cmd/fd2/native_ai_item_presentation.go`（646 行）。
//!
//! 骨架：Go 檔的 8 個函式中產生 2 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// Go：`native_ai_item_presentation.go:14`
///
/// 不 derive `Debug`／`Clone`：欄位含 callback（`Box<dyn FnOnce>`）、檔案或亂數產生器等做不到的型別。
#[derive(Default)]
pub(crate) struct NativeAIItemPresentationJob {
    pub(crate) transaction: Option<NativeAIItemTransaction>,
    pub(crate) frames: Vec<NativeCompoundPresentedFrame>,
    pub(crate) frame: i64,
    pub(crate) repeat: i64,
    pub(crate) publish_at: i64,
    pub(crate) published: bool,
    pub(crate) drawn: bool,
    pub(crate) holding: bool,
    pub(crate) hold: i64,
    pub(crate) baseline_work: Vec<u8>,
    pub(crate) baseline_vga: Vec<u8>,
    pub(crate) post_transaction_work: Vec<u8>,
    pub(crate) post_vga: Vec<u8>,
    pub(crate) then: Option<Box<dyn FnOnce(&mut Game) + Send>>,
}

impl Game {
    /// Go：`native_ai_item_presentation.go:531`，`func (g *Game) cancelNativeAIItemPresentation()`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn cancel_native_ai_item_presentation(&mut self) {
        let Some(j) = self.native_ai_item_presentation.take() else {
            return;
        };
        if j.published
            && let Some(tx) = j.transaction.as_ref()
        {
            match self.st.as_mut() {
                Some(st) if tx.before.len() == st.units.len() * 80 => {
                    // Go 逐一檢查 `unit == nil`；Rust 的單位不會是 nil，該分支不移植。
                    for (index, unit) in st.units.iter_mut().enumerate() {
                        sync_native_item_runtime_record(unit, &tx.before[index * 80..(index + 1) * 80]);
                    }
                    self.native_rng_state = tx.rng_before;
                }
                _ => {
                    if self.load_err.is_empty() {
                        self.load_err = String::from("native AI item rollback roster changed");
                    }
                }
            }
        }
        self.native_map_work = j.baseline_work.clone();
        self.native_map_vga = j.baseline_vga.clone();
        self.native_ai_item_presentation = None;
    }

    /// Go：`native_ai_item_presentation.go:568`，`func (g *Game) failNativeAIItemPresentation(err error)`（`r8`，隨
    /// `stepNativeAIItemPresentation` 補移植；Go `Draw` 也呼叫它）。
    pub(crate) fn fail_native_ai_item_presentation(&mut self, err: &GameError) {
        self.cancel_native_ai_item_presentation();
        self.load_err = format!("native AI item presentation: {}", err.message());
    }

    /// Go：`native_ai_item_presentation.go:573`，`func (g *Game) stepNativeAIItemPresentation()`
    pub(crate) fn step_native_ai_item_presentation(&mut self) {
        let Some(j) = self.native_ai_item_presentation.as_mut() else {
            return;
        };
        if j.holding {
            if j.hold > 0 {
                j.hold -= 1;
                return;
            }
            let then = j.then.take();
            self.native_ai_item_presentation = None;
            if let Some(then) = then {
                then(self);
            }
            return;
        }
        if !j.drawn || j.frame < 0 || j.frame >= j.frames.len() as i64 {
            return;
        }
        j.drawn = false;
        let frame = &mut j.frames[j.frame as usize];
        if !frame.sound_played && !frame.sound.is_empty() {
            let sound = frame.sound.clone();
            frame.sound_played = true;
            self.play_raw(&sound);
        }
        let j = self.native_ai_item_presentation.as_mut().unwrap();
        j.repeat += 1;
        if j.repeat < j.frames[j.frame as usize].delay {
            return;
        }
        j.repeat = 0;
        let next = j.frame + 1;
        if next == j.publish_at && !j.published {
            let tx = j.transaction.clone();
            if let Err(err) = self.commit_native_ai_item_transaction(tx.as_ref()) {
                self.fail_native_ai_item_presentation(&err);
                return;
            }
            let j = self.native_ai_item_presentation.as_mut().unwrap();
            self.native_map_work = j.post_transaction_work.clone();
            self.native_map_vga = j.post_vga.clone();
            j.published = true;
        }
        let j = self.native_ai_item_presentation.as_mut().unwrap();
        j.frame = next;
        if j.frame >= j.frames.len() as i64 {
            if !j.published {
                self.fail_native_ai_item_presentation(&game_err!(
                    "required item publication boundary was not presented"
                ));
                return;
            }
            j.frame = j.frames.len() as i64 - 1;
            (j.holding, j.hold) = (true, native_delay_ticks(500));
        }
    }
}
