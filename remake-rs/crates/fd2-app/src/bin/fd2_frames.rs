//! 無畫面驅動端（R5 階段 4，r5w）：不開視窗、不經 Bevy，以 `fd2-game` 的 [`PlayerRunner`] 照腳本跑，每個顯示幀
//! 畫一次 CPU 畫格並記雜湊，格式與 Bevy 外殼的 `frames.jsonl` 相同（`tools/rust_r5_app_frames.sh` 逐行比對）。
//!
//! 用法：`fd2_frames <腳本> <frames.jsonl> [timing.jsonl]`。環境變數同外殼（`FD2_CAMPAIGN` 要明確設定）。
//!
//! R5 階段 5（r5g）：`FD2_RENDERER=gpu` 時每個顯示幀改錄成顯示清單，以無視窗 GPU 合成器合成並讀回再記雜湊
//! （GPU 不可用時結束碼 2，驗證不退回 CPU）。給了第三個參數時寫每幀的更新與畫格時間（毫秒）。

use std::io::Write as _;
use std::time::Instant;

use fd2_app::gpu::GpuCompositor;
use fd2_app::gpu_shell::{Renderer, requested_renderer};
use fd2_app::script::{frame_record, parse};
use fd2_app::video::{LOGICAL_H, LOGICAL_W, letterbox};
use fd2_game::game::{Picture, display_list};
use fd2_game::{PlayerRunner, RunnerStep};

fn main() {
    let args: Vec<String> = std::env::args().collect();
    assert!(args.len() == 3 || args.len() == 4, "用法：fd2_frames <腳本> <frames.jsonl> [timing.jsonl]");
    let script = parse(&std::fs::read_to_string(&args[1]).expect("讀腳本")).expect("腳本");
    assert!(std::env::var_os("FD2_CAMPAIGN").is_some(), "FD2_CAMPAIGN 要明確設定（外殼由 bin/fd2 補預設值）");
    let renderer =
        requested_renderer(&[], std::env::var("FD2_RENDERER").ok().as_deref()).unwrap_or_else(|e| panic!("{e}"));
    let mut gpu = (renderer == Renderer::Gpu).then(|| match GpuCompositor::headless(LOGICAL_W, LOGICAL_H) {
        Ok(c) => {
            eprintln!("GPU 模式：{}", c.adapter);
            c
        }
        Err(e) => {
            eprintln!("GPU 模式不可用：{e}");
            std::process::exit(2);
        }
    });
    let mut out = std::io::BufWriter::new(std::fs::File::create(&args[2]).expect("建立輸出"));
    let mut timing = args.get(3).map(|p| std::io::BufWriter::new(std::fs::File::create(p).expect("建立時間輸出")));
    let lb = letterbox(script.window.0, script.window.1);
    let mut runner = PlayerRunner::load();
    if !runner.load_error().is_empty() {
        eprintln!("loadGame：{}", runner.load_error());
    }
    for (index, frame) in script.frames.iter().enumerate() {
        let raw = frame.raw_state(lb);
        let t0 = Instant::now();
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
        let t1 = Instant::now();
        let (pix, record_ms) = match gpu.as_mut() {
            None => {
                let mut screen = Picture::new_image(LOGICAL_W as i64, LOGICAL_H as i64);
                runner.draw(&mut screen);
                (screen.pix, None)
            }
            Some(comp) => {
                let list = display_list::record(LOGICAL_W as i64, LOGICAL_H as i64, |screen| runner.draw(screen));
                let rec = t1.elapsed().as_secs_f64() * 1e3;
                (comp.render(&list).unwrap_or_else(|e| panic!("第 {index} 幀 GPU 合成：{e}")), Some(rec))
            }
        };
        let t2 = Instant::now();
        let line = frame_record(index, frame.updates, &pix, runner.game_frame(), &runner.progress());
        writeln!(out, "{line}").expect("寫輸出");
        if let Some(t) = timing.as_mut() {
            let row = serde_json::json!({
                "frame": index, "updates": frame.updates,
                "renderer": if gpu.is_some() { "gpu" } else { "cpu" },
                "update_ms": (t1 - t0).as_secs_f64() * 1e3, "draw_ms": (t2 - t1).as_secs_f64() * 1e3,
                "record_ms": record_ms,
            });
            writeln!(t, "{row}").expect("寫時間輸出");
        }
    }
}
