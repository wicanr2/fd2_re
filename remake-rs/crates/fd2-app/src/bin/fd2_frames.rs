//! 無畫面驅動端（R5 階段 4，r5w）：不開視窗、不經 Bevy，以 `fd2-game` 的 [`PlayerRunner`] 照腳本跑，每個顯示幀
//! 畫一次 CPU 畫格並記雜湊，格式與 Bevy 外殼的 `frames.jsonl` 相同（`tools/rust_r5_app_frames.sh` 逐行比對）。
//!
//! 用法：`fd2_frames <腳本> <frames.jsonl>`。環境變數同外殼（`FD2_CAMPAIGN` 要明確設定）。

use std::io::Write as _;

use fd2_app::script::{frame_record, parse};
use fd2_app::video::{LOGICAL_H, LOGICAL_W, letterbox};
use fd2_game::game::Picture;
use fd2_game::{PlayerRunner, RunnerStep};

fn main() {
    let args: Vec<String> = std::env::args().collect();
    assert_eq!(args.len(), 3, "用法：fd2_frames <腳本> <frames.jsonl>");
    let script = parse(&std::fs::read_to_string(&args[1]).expect("讀腳本")).expect("腳本");
    assert!(std::env::var_os("FD2_CAMPAIGN").is_some(), "FD2_CAMPAIGN 要明確設定（外殼由 bin/fd2 補預設值）");
    let mut out = std::io::BufWriter::new(std::fs::File::create(&args[2]).expect("建立輸出"));
    let lb = letterbox(script.window.0, script.window.1);
    let mut runner = PlayerRunner::load();
    if !runner.load_error().is_empty() {
        eprintln!("loadGame：{}", runner.load_error());
    }
    for (index, frame) in script.frames.iter().enumerate() {
        let raw = frame.raw_state(lb);
        for _ in 0..frame.updates {
            match runner.update(&raw) {
                Ok(RunnerStep::Continue) => {}
                Ok(RunnerStep::Terminated) => {
                    eprintln!("第 {index} 幀：遊戲結束");
                    return;
                }
                Err(err) => panic!("第 {index} 幀 Update：{err}"),
            }
        }
        let _ = runner.take_fullscreen_request();
        let mut screen = Picture::new_image(LOGICAL_W as i64, LOGICAL_H as i64);
        runner.draw(&mut screen);
        let line = frame_record(index, frame.updates, &screen.pix, runner.game_frame(), &runner.progress());
        writeln!(out, "{line}").expect("寫輸出");
    }
}
