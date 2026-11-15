//! 腳本輸入（R5 階段 4，r5w）：外殼驗證用。同一份腳本交給 Bevy 外殼（按鍵與觸控以 Bevy 輸入訊息注入）與
//! 無畫面驅動端（`fd2_frames`，直接組 [`RawInputState`]），兩邊每個顯示幀的 CPU 畫格雜湊要相同。
//!
//! 格式（JSON）：
//!
//! ```text
//! {"schema": "fd2-app-script/1", "window": [寬, 高],
//!  "steps": [{"frames": N, "updates": U, "keys": ["Enter", ...], "touch": [x, y], "shot": "名稱"}, ...]}
//! ```
//!
//! - 每個 step 是 `frames` 個顯示幀（預設 1），每個顯示幀跑 `updates` 次 `Update`（預設 1，對應 Ebiten 一幀補跑多次
//!   `Update`），之後畫一次。`keys` 是這些幀按住的邏輯鍵（[`Key`] 的名稱），`touch` 是按住的觸控點（視窗實體像素，ID 1）。
//! - `shot`：這個 step 最後一幀畫完後截取視窗，核對等於 CPU 畫格的整數倍最近鄰放大。
//! - 鍵在相鄰 step 之間持續按住就不會有新的「剛按下」；要連按兩次，中間放一個空 step。
//! - `{"repeat": N, "steps": [...]}` 把內層 step 重複 N 次（可巢狀）。

use std::collections::BTreeSet;

use fd2_game::game::{Key, RawInputState};
use serde_json::Value;
use sha2::{Digest, Sha256};

use crate::video::Letterbox;

/// 一個顯示幀的腳本輸入。
#[derive(Debug, Clone, PartialEq)]
pub struct ScriptFrame {
    pub updates: u32,
    pub keys: BTreeSet<Key>,
    /// 按住的觸控點 `(id, x, y)`，視窗實體像素。
    pub touches: Vec<(u64, f32, f32)>,
    pub shot: Option<String>,
}

/// 展開後的腳本。
#[derive(Debug, Clone, PartialEq)]
pub struct Script {
    pub window: (u32, u32),
    pub frames: Vec<ScriptFrame>,
}

/// [`Key`] 的名稱（Debug 輸出）→ 鍵。
pub fn key_by_name(name: &str) -> Option<Key> {
    Key::ALL.into_iter().find(|k| format!("{k:?}") == name)
}

/// 讀腳本。格式不對就回錯誤（不猜測）。
pub fn parse(raw: &str) -> Result<Script, String> {
    let v: Value = serde_json::from_str(raw).map_err(|e| format!("腳本 JSON：{e}"))?;
    if v["schema"] != "fd2-app-script/1" {
        return Err(format!("腳本 schema 不是 fd2-app-script/1：{}", v["schema"]));
    }
    let size = |i: usize| v["window"][i].as_u64().filter(|&n| n > 0 && n <= 8192).map(|n| n as u32);
    let window = (size(0).ok_or("window[0]")?, size(1).ok_or("window[1]")?);
    let mut frames = Vec::new();
    expand(v["steps"].as_array().ok_or("steps 不是陣列")?, "steps", &mut frames)?;
    if frames.len() > 1_000_000 {
        return Err(format!("腳本展開後 {} 幀，超過上限", frames.len()));
    }
    Ok(Script { window, frames })
}

/// 展開一串 step（`repeat` 會遞迴展開）。
fn expand(steps: &[Value], at: &str, frames: &mut Vec<ScriptFrame>) -> Result<(), String> {
    for (i, step) in steps.iter().enumerate() {
        let at = format!("{at}[{i}]");
        let obj = step.as_object().ok_or(format!("{at} 不是物件"))?;
        if let Some(n) = step.get("repeat") {
            for key in obj.keys() {
                if !matches!(key.as_str(), "repeat" | "steps" | "note") {
                    return Err(format!("{at} 有不認得的欄位 {key}"));
                }
            }
            let n = n.as_u64().filter(|&n| n > 0).ok_or(format!("{at}.repeat"))?;
            let inner = step["steps"].as_array().ok_or(format!("{at}.steps 不是陣列"))?;
            for _ in 0..n {
                expand(inner, &format!("{at}.steps"), frames)?;
                if frames.len() > 1_000_000 {
                    return Err(format!("{at} 展開後超過上限"));
                }
            }
            continue;
        }
        for key in obj.keys() {
            if !matches!(key.as_str(), "frames" | "updates" | "keys" | "touch" | "shot" | "note") {
                return Err(format!("{at} 有不認得的欄位 {key}"));
            }
        }
        let n = step.get("frames").map_or(Some(1), Value::as_u64).filter(|&n| n > 0).ok_or(format!("{at}.frames"))?;
        let updates = step.get("updates").map_or(Some(1), Value::as_u64).ok_or(format!("{at}.updates"))? as u32;
        let mut keys = BTreeSet::new();
        for name in step.get("keys").and_then(Value::as_array).into_iter().flatten() {
            let name = name.as_str().ok_or(format!("{at}.keys"))?;
            keys.insert(key_by_name(name).ok_or(format!("{at} 不認得的鍵 {name}"))?);
        }
        let touches = match step.get("touch") {
            None => Vec::new(),
            Some(t) => {
                let c = |j: usize| t[j].as_f64().map(|f| f as f32).ok_or(format!("{at}.touch"));
                vec![(1, c(0)?, c(1)?)]
            }
        };
        let shot = step.get("shot").map(|s| s.as_str().map(str::to_string).ok_or(format!("{at}.shot"))).transpose()?;
        for f in 0..n {
            let last = f + 1 == n;
            frames.push(ScriptFrame {
                updates,
                keys: keys.clone(),
                touches: touches.clone(),
                shot: if last { shot.clone() } else { None },
            });
        }
    }
    Ok(())
}

impl ScriptFrame {
    /// 無畫面驅動端的邏輯狀態：鍵直接是邏輯鍵，觸控依腳本視窗大小的畫格位置換算。
    pub fn raw_state(&self, lb: Letterbox) -> RawInputState {
        RawInputState {
            keys: self.keys.clone(),
            touches: self
                .touches
                .iter()
                .map(|&(id, x, y)| {
                    let (lx, ly) = lb.to_logical(x, y);
                    (id, lx, ly)
                })
                .collect(),
            ..Default::default()
        }
    }
}

/// 一個顯示幀的紀錄（兩個驅動端寫同一格式，逐行比對）。
pub fn frame_record(index: usize, updates: u32, pix: &[u8], game_frame: i64, progress: &(String, String)) -> String {
    let sha: String = Sha256::digest(pix).iter().map(|b| format!("{b:02x}")).collect();
    serde_json::json!({
        "frame": index, "updates": updates, "rgba_sha256": sha, "game_frame": game_frame,
        "title_phase": progress.0, "node": progress.1,
    })
    .to_string()
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn parse_expands_steps() {
        let s = parse(
            r#"{"schema":"fd2-app-script/1","window":[1280,800],"steps":[
              {"frames":2},{"keys":["Enter","AltLeft"],"updates":3,"shot":"a"},{"frames":2,"touch":[10,20.5],"shot":"b"}]}"#,
        )
        .unwrap();
        assert_eq!(s.window, (1280, 800));
        assert_eq!(s.frames.len(), 5);
        assert_eq!(s.frames[2].updates, 3);
        assert_eq!(s.frames[2].keys, [Key::Enter, Key::AltLeft].into());
        assert_eq!(s.frames[2].shot.as_deref(), Some("a"));
        assert_eq!(s.frames[3].shot, None);
        assert_eq!(s.frames[4].touches, vec![(1, 10.0, 20.5)]);
        assert_eq!(s.frames[4].shot.as_deref(), Some("b"));
        let r = parse(
            r#"{"schema":"fd2-app-script/1","window":[640,400],"steps":[
              {"repeat":3,"steps":[{"keys":["Enter"]},{"frames":2}]}]}"#,
        )
        .unwrap();
        assert_eq!(r.frames.len(), 9);
        assert!(r.frames[3].keys.contains(&Key::Enter) && r.frames[4].keys.is_empty());
    }

    #[test]
    fn parse_rejects_unknown_keys_and_fields() {
        let base = |step: &str| format!(r#"{{"schema":"fd2-app-script/1","window":[640,400],"steps":[{step}]}}"#);
        assert!(parse(&base(r#"{"keys":["Q"]}"#)).is_err());
        assert!(parse(&base(r#"{"hold":1}"#)).is_err());
        assert!(parse(&base(r#"{"frames":0}"#)).is_err());
        assert!(parse(&base(r#"{"repeat":0,"steps":[]}"#)).is_err());
        assert!(parse(&base(r#"{"repeat":2,"steps":[],"keys":[]}"#)).is_err());
        assert!(parse(r#"{"schema":"x","window":[1,1],"steps":[]}"#).is_err());
    }
}
