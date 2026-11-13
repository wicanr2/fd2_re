//! 移植自 Go `pending_code1_interlude.go`。

/// 一個觀察到的 `sub_22E5C` 操作。Go 版是具名 `string` 型別加常數；這裡改成列舉，
/// [`as_str`](Self::as_str) 回傳 Go 的字串值。
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum PendingCode1PresentationKind {
    StopBgm,
    PreparePalette,
    ClearScreen,
    DrawFrame,
    FadeIn,
    WaitTick,
    Release,
}

impl PendingCode1PresentationKind {
    /// Go 常數的字串值。
    pub fn as_str(self) -> &'static str {
        match self {
            PendingCode1PresentationKind::StopBgm => "stop_bgm",
            PendingCode1PresentationKind::PreparePalette => "palette_prepare",
            PendingCode1PresentationKind::ClearScreen => "clear_indexed_320x200",
            PendingCode1PresentationKind::DrawFrame => "draw_frame",
            PendingCode1PresentationKind::FadeIn => "palette_fade_in",
            PendingCode1PresentationKind::WaitTick => "wait_tick",
            PendingCode1PresentationKind::Release => "release",
        }
    }
}

/// 讓 raw `sub_22E5C` 呈現順序可編輯，不自創高層場景名稱或章節綁定。
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct PendingCode1PresentationStep {
    pub kind: PendingCode1PresentationKind,
    pub frame: i64,
    pub count: i64,
    pub duration_ms: i64,
}

/// 保留 0x22E5C 固定呈現順序；正式敗北 owner 依 READY 規格接線，其他 pending producer 不自動取得
/// 敗北語意。
pub fn native_pending_code1_presentation_plan() -> Vec<PendingCode1PresentationStep> {
    use PendingCode1PresentationKind::*;
    let step = |kind, frame, count, duration_ms| PendingCode1PresentationStep { kind, frame, count, duration_ms };
    vec![
        step(StopBgm, 0, 0, 0),
        step(WaitTick, 0, 1, 0),
        step(PreparePalette, 0, 0, 0),
        step(ClearScreen, 0, 0, 0),
        step(DrawFrame, 0, 0, 0),
        step(FadeIn, 0, 65, 2),
        step(WaitTick, 0, 9, 0),
        step(DrawFrame, 1, 0, 0),
        step(WaitTick, 0, 36, 0),
        step(Release, 0, 0, 0),
    ]
}
