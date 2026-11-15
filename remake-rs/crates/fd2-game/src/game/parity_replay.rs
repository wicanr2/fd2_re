//! R4 階段 3：章重播的無畫面驅動端（只在測試編譯）。
//!
//! 對應凍結 Go 版 `remake/cmd/fd2` 的章重播測試：`chapter_parity_replay_test.go`（`TestChapterParityReplay`
//! 與 `parityReplay`）、`chapter_parity_start_test.go`（START 入口與酒店保存冷讀）、
//! `chapter_parity_hp_lock_test.go`（`parityUpdateGame`）、`chapter_parity_legacy_end_test.go`、
//! `native_chapter_one_play_test.go`（`pump`、`closeRing`、`ackPresents`）與 `native_title_to_town_test.go`
//! （`journeyTrace`、`driveStory`）。每個 Go 測試檔一個子模組，函式逐一對照。
//!
//! 只移植第一章（`FD2_PARITY_CHAPTER=1`、`FD2_PARITY_ENTRY=start`）38 個動作會走到的分支；其他章與其他
//! 動作種類停在 `todo!("R8: …")`。
//!
//! 和 Go 的差異：
//! - Go 每次 `g.Update()` 讀牆鐘；這裡每次 `update` 之前以 [`VirtualFrameClock`] 推進一個邏輯幀
//!   （A 組「虛擬時鐘」）。
//! - Go 呼叫 `g.Draw(screen)` 的地方改呼叫 [`Game::acknowledge_present`]（`present.rs`），不畫像素。
//! - R5 階段 1：`rust_golden_frames_test.rs` 是 Go `rust_golden_frames_test.go` chapter 模式的擷取點，設
//!   `FD2_GOLDEN_FRAMES_MODE=chapter`、`FD2_GOLDEN_FRAMES_OUT` 才打開，預設關閉。
//! - R5 階段 2：`rust_golden_frames_opening_test.rs` 是同一個 Go 檔的 opening 模式（`goldenFramesOpening`），
//!   `#[ignore]`，由 `tools/rust_r5_opening_frames.sh` 執行。
//! - R5 階段 2 r5m：`rust_golden_map_layers_test.rs` 是 Go `rust_golden_map_layers_test.go` 的同一個掛點，設
//!   `FD2_GOLDEN_MAP_LAYERS_OUT` 才打開，預設關閉。
//! - R5 階段 2 r5d：`rust_golden_dialogue_layers_test.rs` 是 Go `rust_golden_dialogue_layers_test.go` 的同一個掛點，設
//!   `FD2_GOLDEN_DIALOGUE_LAYERS_OUT` 才打開，預設關閉；任一擷取打開時以 `pin_global_rand` 接上 Go 的全域亂數。
//! - R5 階段 2 r5p：`rust_golden_r5p_layers_test.rs` 是 Go `rust_golden_r5p_layers_test.go`（chapter 模式同一個掛點，
//!   設 `FD2_GOLDEN_R5P_LAYERS_*` 才打開；opening 模式 `#[ignore]`，由 `tools/rust_r5p_layers.sh` 執行）。
//! - R5 階段 2 r5c：`rust_golden_r5c_layers_test.rs` 是 Go `rust_golden_r5c_layers_test.go`（pump 掛點加上戰後迴圈、
//!   城鎮游標與酒店輸入的 gap 擷取點；設 `FD2_GOLDEN_R5C_LAYERS_OUT` 才打開，由 `tools/rust_r5c_layers.sh` 執行）。
//! - R5 階段 2 r5f：`rust_golden_ttf_layers_test.rs` 是 Go `rust_golden_ttf_layers_test.go`（chapter 模式同一個掛點，
//!   設 `FD2_GOLDEN_TTF_LAYERS_OUT` 才打開；度量黃金 `#[ignore]`，由 `tools/rust_r5f_ttf_layers.sh` 執行）。驅動端在
//!   `load_game` 之後以 `require_golden_font` 要求 Go 黃金同一份字型。
//! - R5 階段 2 r5b：`rust_golden_r5b_layers_test.rs` 是 Go `rust_golden_r5b_layers_test.go`（chapter 模式同一個掛點，
//!   設 `FD2_GOLDEN_R5B_LAYERS_OUT` 才打開，由 `tools/rust_r5b_layers.sh` 執行）。
//! - Go 以 `t.Setenv` 設的環境變數由父測試以子行程傳入（F 組 `f_child_env` 的做法），工作區禁用 `unsafe`。

use super::*;

mod chapter_parity_hp_lock_test;
mod chapter_parity_legacy_end_test;
mod chapter_parity_replay_test;
mod chapter_parity_start_test;
mod native_chapter_one_play_test;
mod native_title_to_town_test;
mod rust_golden_dialogue_layers_test;
mod rust_golden_frames_opening_test;
mod rust_golden_frames_test;
mod rust_golden_map_layers_test;
mod rust_golden_r5b_layers_test;
mod rust_golden_r5c_layers_test;
mod rust_golden_r5p_layers_test;
mod rust_golden_ttf_layers_test;

pub(crate) use chapter_parity_hp_lock_test::*;
pub(crate) use chapter_parity_legacy_end_test::*;
pub(crate) use chapter_parity_replay_test::*;
pub(crate) use chapter_parity_start_test::*;
pub(crate) use native_chapter_one_play_test::*;
pub(crate) use native_title_to_town_test::*;
pub(crate) use rust_golden_dialogue_layers_test::*;
pub(crate) use rust_golden_frames_test::*;
pub(crate) use rust_golden_map_layers_test::*;
pub(crate) use rust_golden_r5b_layers_test::*;
pub(crate) use rust_golden_r5c_layers_test::*;
pub(crate) use rust_golden_r5p_layers_test::*;
pub(crate) use rust_golden_ttf_layers_test::*;
