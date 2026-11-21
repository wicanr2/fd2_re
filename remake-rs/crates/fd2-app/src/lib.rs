//! Bevy 外殼（docs/goal/119 的 R5 起逐步填入）。
//!
//! R5 階段 4（r5w）：視窗、畫格呈現（[`video`]）、輸入對應（[`input`]）、Ebiten 的 TPS 時鐘（[`clock`]）、
//! 腳本驅動的驗證（[`script`]）與 Bevy 主迴圈（[`shell`]）。遊戲狀態與 CPU 畫格在 `fd2-game`。
//! R5 階段 4（r5o）：音訊解碼與播放（[`audio`]）。
//! R5 階段 4（r5v）：照 Go 移植的音訊解碼鏈，`rust-port-no-go-stdlib` 的具名例外：[`go_vorbis`]
//! （jfreymuth/vorbis v1.0.2）、[`go_oggvorbis`]（jfreymuth/oggvorbis v1.0.5）、[`go_ebiten_audio`]
//! （Ebiten v2.6.6 的 `audio/vorbis`、`audio/wav`、`audio/internal/convert`），以及它們共用的 Go `io` 契約
//! [`go_io`]。原授權在各模組文件與 `third_party/`。

pub mod audio;
pub mod clock;
pub mod go_ebiten_audio;
pub mod go_io;
pub mod go_oggvorbis;
pub mod go_vorbis;
// R5 階段 5（r5g）：GPU 加速模式（顯示清單的 GPU 合成器與外殼接線）。
pub mod gpu;
pub mod gpu_shell;
pub mod input;
pub mod script;
pub mod shell;
pub mod video;
