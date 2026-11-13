//! 移植自 Go `system_info_transition.go`。

use super::native_flow_error::{NativeFlowResult, at, fixed};

pub const NATIVE_SYSTEM_INFO_WIDTH: i64 = 320;
pub const NATIVE_SYSTEM_INFO_HEIGHT: i64 = 200;
pub const NATIVE_SYSTEM_INFO_BYTES: usize = (NATIVE_SYSTEM_INFO_WIDTH * NATIVE_SYSTEM_INFO_HEIGHT) as usize;
pub const NATIVE_SYSTEM_INFO_STEPS: i64 = 12;

#[derive(Debug, Clone, Copy)]
struct Rect {
    src_x: i64,
    src_y: i64,
    dst_x: i64,
    dst_y: i64,
    width: i64,
    height: i64,
}

const fn rect(src_x: i64, src_y: i64, dst_x: i64, dst_y: i64, width: i64, height: i64) -> Rect {
    Rect { src_x, src_y, dst_x, dst_y, width, height }
}

/// Go `image.Rectangle` 的最小對應：`[min_x,max_x)×[min_y,max_y)`。只用來公開複製計畫，
/// 不移植 `image` 套件。這裡產生的矩形一定寬高為正，Go `image.Rect` 的端點交換不會發生。
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeSystemInfoRectangle {
    pub min_x: i64,
    pub min_y: i64,
    pub max_x: i64,
    pub max_y: i64,
}

impl NativeSystemInfoRectangle {
    /// Go `Rectangle.Dx()`。
    pub fn dx(&self) -> i64 {
        self.max_x - self.min_x
    }

    /// Go `Rectangle.Dy()`。
    pub fn dy(&self) -> i64 {
        self.max_y - self.min_y
    }
}

/// 公開既有的複製計畫，不改變順序、裁切或獨立的收合 pass。`destination` 是 Go 的
/// `image.Point`，以 `(x, y)` 表示。
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeSystemInfoRegion {
    pub source: NativeSystemInfoRectangle,
    pub destination: (i64, i64),
}

/// `phase` 是 Go 的字串參數：`"steady"`、`"opening"`、`"closing"`，其餘拒收。
pub fn native_system_info_regions(phase: &str, frame: i64) -> NativeFlowResult<Vec<NativeSystemInfoRegion>> {
    let raw = match phase {
        "steady" => {
            if frame != 0 {
                return fixed("fdother: invalid steady information frame");
            }
            open_rects(12)
        }
        "opening" | "closing" => {
            if !(0..NATIVE_SYSTEM_INFO_STEPS).contains(&frame) {
                return fixed("fdother: invalid information transition frame");
            }
            if phase == "opening" { open_rects(frame) } else { close_rects(11 - frame) }
        }
        _ => return fixed("fdother: invalid information phase"),
    };
    Ok(raw
        .into_iter()
        .filter(|r| r.width > 0 && r.height > 0)
        .map(|r| NativeSystemInfoRegion {
            source: NativeSystemInfoRectangle {
                min_x: r.src_x,
                min_y: r.src_y,
                max_x: r.src_x + r.width,
                max_y: r.src_y + r.height,
            },
            destination: (r.dst_x, r.dst_y),
        })
        .collect())
}

fn blit_rect(dst: &mut [u8], src: &[u8], r: Rect) -> NativeFlowResult<()> {
    if r.width <= 0 || r.height <= 0 {
        return Ok(());
    }
    if r.src_x < 0
        || r.src_y < 0
        || r.dst_x < 0
        || r.dst_y < 0
        || r.src_x + r.width > NATIVE_SYSTEM_INFO_WIDTH
        || r.dst_x + r.width > NATIVE_SYSTEM_INFO_WIDTH
        || r.src_y + r.height > NATIVE_SYSTEM_INFO_HEIGHT
        || r.dst_y + r.height > NATIVE_SYSTEM_INFO_HEIGHT
    {
        return fixed("fdother: native system info transition rectangle is invalid");
    }
    for row in 0..r.height {
        let src_start = at((r.src_y + row) * NATIVE_SYSTEM_INFO_WIDTH + r.src_x);
        let dst_start = at((r.dst_y + row) * NATIVE_SYSTEM_INFO_WIDTH + r.dst_x);
        let width = at(r.width);
        dst[dst_start..dst_start + width].copy_from_slice(&src[src_start..src_start + width]);
    }
    Ok(())
}

fn open_rects(step: i64) -> Vec<Rect> {
    let mut rects = Vec::with_capacity(4);
    let (mut dst_x, mut width) = (75, 170);
    if step <= 4 {
        dst_x += (4 - step) * 50;
        if dst_x + width > NATIVE_SYSTEM_INFO_WIDTH {
            width = NATIVE_SYSTEM_INFO_WIDTH - dst_x;
        }
    }
    rects.push(rect(75, 37, dst_x, 37, width, 117));
    if step >= 3 {
        let mut dst_y = 19;
        if step <= 7 {
            dst_y += (7 - step) * 6;
        }
        rects.push(rect(109, 19, 109, dst_y, 102, 17));
    }
    if step >= 6 {
        let mut dst_y = 155;
        if step <= 9 {
            dst_y += (9 - step) * 9;
        }
        rects.push(rect(75, 155, 75, dst_y, 170, 16));
    }
    if step >= 9 {
        let mut dst_y = 172;
        if step <= 12 {
            dst_y += (12 - step) * 4;
        }
        rects.push(rect(129, 172, 129, dst_y, 63, 15));
    }
    rects
}

fn close_rects(step: i64) -> Vec<Rect> {
    let mut rects = open_rects(step);
    if step >= 5 {
        return rects;
    }
    let shift = (4 - step) * 50;
    let (mut src_x, mut dst_x, mut width) = (75, 75 - shift, 170);
    if dst_x < 0 {
        src_x -= dst_x;
        width += dst_x;
        dst_x = 0;
    }
    rects[0] = rect(src_x, 37, dst_x, 37, width, 117);
    rects
}

fn compose(baseline: &[u8], information: &[u8], rects: Vec<Rect>) -> NativeFlowResult<Vec<u8>> {
    let mut frame = baseline.to_vec();
    for r in rects {
        blit_rect(&mut frame, information, r)?;
    }
    Ok(frame)
}

/// 重現 `sub_1B1E7` 的十二次展開呈現與獨立的 11 到 0 收合 pass。每幀都從呼叫端未改動的
/// 戰場底圖開始，再經 `sub_1AF1E/sub_1AF99/sub_1B019/sub_1B0AD/sub_1B14B` 還原的四塊直接
/// 裁切區域複製資訊表面。回傳 `(opening, closing)`。
#[allow(clippy::type_complexity)]
pub fn native_system_info_transition_frames(
    baseline: &[u8],
    information: &[u8],
) -> NativeFlowResult<(Vec<Vec<u8>>, Vec<Vec<u8>>)> {
    if baseline.len() != NATIVE_SYSTEM_INFO_BYTES || information.len() != NATIVE_SYSTEM_INFO_BYTES {
        return fixed("fdother: native system info surfaces must be 320x200");
    }
    let mut opening = Vec::with_capacity(NATIVE_SYSTEM_INFO_STEPS as usize);
    for step in 0..NATIVE_SYSTEM_INFO_STEPS {
        opening.push(compose(baseline, information, open_rects(step))?);
    }
    let mut closing = Vec::with_capacity(NATIVE_SYSTEM_INFO_STEPS as usize);
    for step in (0..NATIVE_SYSTEM_INFO_STEPS).rev() {
        closing.push(compose(baseline, information, close_rects(step))?);
    }
    Ok((opening, closing))
}

/// 保留 `sub_1B1E7` 的戰場備份，只覆蓋四塊資訊面板。
pub fn native_system_info_steady_frame(baseline: &[u8], information: &[u8]) -> NativeFlowResult<Vec<u8>> {
    if baseline.len() != NATIVE_SYSTEM_INFO_BYTES || information.len() != NATIVE_SYSTEM_INFO_BYTES {
        return fixed("fdother: native system info steady surfaces must be 320x200");
    }
    compose(baseline, information, open_rects(12))
}

#[cfg(test)]
mod tests;
