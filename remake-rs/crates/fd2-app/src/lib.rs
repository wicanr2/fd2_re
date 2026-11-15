//! Bevy 外殼（docs/goal/119 的 R5 起逐步填入）。
//!
//! R5 階段 4（r5w）：視窗、畫格呈現（[`video`]）、輸入對應（[`input`]）、Ebiten 的 TPS 時鐘（[`clock`]）、
//! 腳本驅動的驗證（[`script`]）與 Bevy 主迴圈（[`shell`]）。遊戲狀態與 CPU 畫格在 `fd2-game`。
//! R5 階段 4（r5o）：音訊解碼與播放（[`audio`]）。

pub mod audio;
pub mod clock;
pub mod input;
pub mod script;
pub mod shell;
pub mod video;
