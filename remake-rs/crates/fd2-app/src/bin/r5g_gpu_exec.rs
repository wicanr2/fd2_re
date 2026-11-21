//! GPU 執行器（R5 階段 5，`r5g`）：從 stdin 讀 `fd2-game` 的顯示清單，以無視窗 GPU 合成器執行，把讀回的預乘 RGBA8
//! 寫到 stdout。協定見 `fd2_game::game::display_list::ExternalExecutor`。給 `fd2-game` 的畫格擷取測試在 GPU 模式下用
//! （`tools/rust_r5g_gpu_frames.sh`）；GPU 不可用時回報錯誤並結束，驗證不退回 CPU。

use std::io::Write;

use fd2_app::gpu::GpuCompositor;
use fd2_game::game::display_list::DisplayList;

fn reply(out: &mut impl Write, magic: &[u8; 8], status: u32, body: &[u8]) -> std::io::Result<()> {
    out.write_all(magic)?;
    out.write_all(&status.to_le_bytes())?;
    out.write_all(&(body.len() as u32).to_le_bytes())?;
    out.write_all(body)?;
    out.flush()
}

fn main() {
    let stdin = std::io::stdin();
    let mut input = std::io::BufReader::with_capacity(1 << 20, stdin.lock());
    let stdout = std::io::stdout();
    let mut out = std::io::BufWriter::with_capacity(1 << 20, stdout.lock());
    // 合成器先以 640×400 建立；清單大小不同時 `render` 重建輸出緩衝區。
    let probe = GpuCompositor::headless(640, 400);
    let mut comp = match probe {
        Ok(c) => {
            reply(&mut out, b"FD2GPU01", 0, c.adapter.as_bytes()).expect("寫 stdout");
            c
        }
        Err(e) => {
            eprintln!("r5g_gpu_exec：{e}");
            reply(&mut out, b"FD2GPU01", 1, e.as_bytes()).expect("寫 stdout");
            std::process::exit(2);
        }
    };
    let mut frames = 0u64;
    loop {
        let dl = match DisplayList::decode(&mut input) {
            Ok(Some(dl)) => dl,
            Ok(None) => break,
            Err(e) => {
                reply(&mut out, b"FD2PX001", 1, e.as_bytes()).expect("寫 stdout");
                std::process::exit(3);
            }
        };
        match comp.render(&dl) {
            Ok(pix) => reply(&mut out, b"FD2PX001", 0, &pix).expect("寫 stdout"),
            Err(e) => {
                reply(&mut out, b"FD2PX001", 1, e.as_bytes()).expect("寫 stdout");
                std::process::exit(4);
            }
        }
        frames += 1;
    }
    eprintln!("r5g_gpu_exec：{frames} 份清單");
}
