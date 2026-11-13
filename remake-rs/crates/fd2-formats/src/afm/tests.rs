//! 逐條移植自 Go `remake/internal/afm/afm_test.go`，期望值不改。
//!
//! Go 版以相對路徑找玩家自備的 ANI.DAT（`org_game`，gitignore），缺檔則 skip。Rust 版另外
//! 接受環境變數 `FD2_ANI_DAT`，讓 Docker 內的測試能掛載同一份原版檔。

use super::*;

/// 找玩家自備的 ANI.DAT；找不到回 `None`，測試視同 Go 的 `t.Skip`。
pub(crate) fn ani_path() -> Option<String> {
    if let Ok(p) = std::env::var("FD2_ANI_DAT") {
        return std::path::Path::new(&p).is_file().then_some(p);
    }
    let manifest = env!("CARGO_MANIFEST_DIR");
    ["../../../org_game/炎龍騎士團/FLAME2/ANI.DAT", "../../../../org_game/炎龍騎士團/FLAME2/ANI.DAT"]
        .iter()
        .map(|rel| format!("{manifest}/{rel}"))
        .find(|p| std::path::Path::new(p).is_file())
}

/// 各資源預期幀數（doc39 反組譯＋視覺驗證）。Go 版是 map，迭代順序不影響斷言。
const WANT_FRAMES: [(i64, usize); 9] = [
    (0, 96), // 金鎖（橢圓→正圓）
    (1, 51), // FLAME DRAGON 標題「2」logo 成形
    (2, 26), // 結局合成器
    (3, 28),
    (4, 12),
    (5, 35), // 拔劍屠龍
    (6, 12),
    (7, 17),
    (8, 12),
];

#[test]
fn decode_ani() {
    let Some(p) = ani_path() else {
        eprintln!("ANI.DAT 不存在（玩家自備素材），跳過");
        return;
    };
    for (res, want) in WANT_FRAMES {
        let clip = decode_resource(&p, res).unwrap_or_else(|e| panic!("資源 {res} 解碼失敗: {e}"));
        assert_eq!(clip.frames.len(), want, "資源 {res} 幀數");
        assert!(
            clip.indexed_frames.len() == want
                && clip.palettes.len() == want
                && clip.indexed_frames[0].len() == FRAME_BYTES
                && clip.palettes[0].len() == 768,
            "資源 {res} indexed snapshots不完整: frames={} indexed={} palettes={}",
            clip.frames.len(),
            clip.indexed_frames.len(),
            clip.palettes.len()
        );
        let b = &clip.frames[0];
        assert!(b.width == SCR_W && b.height == SCR_H, "資源 {res} 幀尺寸 = {}x{},預期 320x200", b.width, b.height);
    }
}

#[test]
fn container_reject() {
    assert!(container_entries(b"not a container").is_err(), "非容器資料應被拒絕");
}

#[test]
fn decode_ani_rejects_truncated_final_frame() {
    let Some(p) = ani_path() else {
        eprintln!("ANI.DAT 不存在");
        return;
    };
    let raw = read_archive_resource(&p, 8).unwrap();
    assert!(decode_afm(&raw[..raw.len() - 1]).is_err(), "截斷的最後一幀不可被當成部分成功 clip");
}

#[test]
fn decode_ani_dump() {
    // 除錯輔助：設 AFM_DUMP=<dir> 時解資源 5 供人眼對照（非 CI）；實際 dump 由外部工具負責。
    if std::env::var("AFM_DUMP").map(|v| v.is_empty()).unwrap_or(true) {
        eprintln!("未設 AFM_DUMP");
        return;
    }
    let Some(p) = ani_path() else {
        eprintln!("ANI.DAT 不存在");
        return;
    };
    let clip = decode_resource(&p, 5).unwrap();
    eprintln!("資源 5:{} 幀,title={:?}", clip.frames.len(), String::from_utf8_lossy(&clip.title));
}
