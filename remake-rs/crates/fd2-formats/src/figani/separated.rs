//! 分離素材包（`animations/FIGANI_NNN/`、`FDOTHER_NNN/`）的動畫讀取。
//!
//! 移植自 Go `separated.go`。figani 自己的契約檢查與錯誤文字和 Go 版逐字相同；JSON、PNG 與檔案系統的
//! 下層錯誤刻意不照做 Go 標準函式庫的文字（見 `doc`、`image` 模組說明），黃金資料只比接受／拒收。

use super::doc::{self, Object};
use super::image::{DecodedImage, decode_png};
use super::{Animation, Cause, FiganiError, Frame, Result, msg};
use std::path::Path;

fn wrapped(prefix: String, cause: Cause) -> FiganiError {
    FiganiError::Wrapped { prefix, cause }
}

fn read(path: &Path) -> std::result::Result<Vec<u8>, Cause> {
    std::fs::read(path).map_err(|source| Cause::Io { path: path.display().to_string(), source })
}

/// Go `filepath.Base(p) == p`（Unix）：非空、不含 `/`，或剛好是 `/`。這是 figani 的幀路徑契約，
/// 結果與 Go 相同。
fn is_base_name(p: &str) -> bool {
    p == "/" || (!p.is_empty() && !p.contains('/'))
}

struct FrameDoc {
    frame_id: String,
    path: String,
    mask_path: String,
    delay: i64,
    x: i64,
    y: i64,
    width: i64,
    height: i64,
    raw_byte4: u8,
    raw_byte5: u8,
    raw_byte7: u8,
}

fn frame_doc(o: &Object) -> std::result::Result<FrameDoc, String> {
    Ok(FrameDoc {
        frame_id: doc::string(o, "frame_id")?,
        path: doc::string(o, "path")?,
        mask_path: doc::string(o, "mask_path")?,
        delay: doc::int(o, "delay_native")?,
        x: doc::int(o, "x")?,
        y: doc::int(o, "y")?,
        width: doc::int(o, "width")?,
        height: doc::int(o, "height")?,
        raw_byte4: doc::byte(o, "raw_byte_4")?,
        raw_byte5: doc::byte(o, "raw_byte_5")?,
        raw_byte7: doc::byte(o, "raw_byte_7")?,
    })
}

struct AnimationDoc {
    schema_version: i64,
    kind: String,
    animation_id: String,
    source_file: String,
    source_resource: i64,
    header: [u8; 3],
    frames: Vec<FrameDoc>,
}

fn animation_doc(raw: &[u8]) -> std::result::Result<AnimationDoc, String> {
    let o = doc::parse_object(raw)?;
    let source = doc::object(&o, "source")?;
    let header = doc::object(&o, "native_header")?;
    Ok(AnimationDoc {
        schema_version: doc::int(&o, "schema_version")?,
        kind: doc::string(&o, "kind")?,
        animation_id: doc::string(&o, "animation_id")?,
        source_file: doc::string(source, "file")?,
        source_resource: doc::int(source, "resource")?,
        header: [doc::byte(header, "byte_1")?, doc::byte(header, "byte_2")?, doc::byte(header, "byte_4")?],
        frames: doc::objects(&o, "frames")?.into_iter().map(frame_doc).collect::<std::result::Result<_, _>>()?,
    })
}

/// 重建 FIGANI 的原始索引像素、透明遮罩與 descriptor 後設資料。正式 runtime 不可在此失敗後
/// 回退讀 FIGANI.DAT。
pub fn load_separated_resource(animation_root: &str, resource: i64) -> Result<Animation> {
    load_separated_archive_resource(animation_root, "FIGANI.DAT", resource)
}

/// 載入已證實使用 FIGANI 幀契約的容器資源，同時保留原始來源家族，不把 FDOTHER 內嵌動畫冒充
/// FIGANI.DAT。不支援的來源名稱以 Rust 的 `{:?}` 引號表示；可列印字元與 Go `%q` 相同，控制字元的
/// 跳脫寫法可能不同。
pub fn load_separated_archive_resource(animation_root: &str, source_file: &str, resource: i64) -> Result<Animation> {
    if animation_root.is_empty() || !(0..=999).contains(&resource) {
        return Err(FiganiError::Static("figani: invalid separated animation request"));
    }
    let (prefix, id_prefix) = match source_file {
        "FIGANI.DAT" => ("FIGANI", "figani"),
        "FDOTHER.DAT" => ("FDOTHER", "fdother"),
        _ => return Err(msg(format!("figani: unsupported separated animation source {source_file:?}"))),
    };
    let directory = Path::new(animation_root).join(format!("{prefix}_{resource:03}"));
    let context = || format!("figani: separated metadata {resource}: ");
    let raw = read(&directory.join("animation.json")).map_err(|e| wrapped(context(), e))?;
    let document = animation_doc(&raw).map_err(|e| wrapped(context(), Cause::Json(e)))?;
    let want_id = format!("animation/{id_prefix}_{resource:03}");
    if document.schema_version != 1
        || document.kind != "animation"
        || document.animation_id != want_id
        || document.source_file != source_file
        || document.source_resource != resource
        || document.frames.is_empty()
    {
        return Err(msg(format!("figani: separated metadata {resource} violates the animation contract")));
    }
    let mut animation = Animation {
        header_byte1: document.header[0],
        header_byte2: document.header[1],
        header_byte4: document.header[2],
        frames: Vec::with_capacity(document.frames.len()),
    };
    for (index, item) in document.frames.iter().enumerate() {
        if item.frame_id != format!("frame/{index:03}")
            || !is_base_name(&item.path)
            || !is_base_name(&item.mask_path)
            || item.width <= 0
            || item.height <= 0
            || !(0..=255).contains(&item.delay)
        {
            return Err(msg(format!("figani: separated frame {index} contract is invalid")));
        }
        let load = |name: &str, what: &str| {
            let context = format!("figani: separated frame {index} {what}: ");
            let data = read(&directory.join(name)).map_err(|e| wrapped(context.clone(), e))?;
            // Go 的格式嗅探要求開頭是 PNG 簽章，否則回報 image: unknown format。
            if !data.starts_with(b"\x89PNG\r\n\x1a\n") {
                return Err(wrapped(context, Cause::UnknownFormat));
            }
            decode_png(&data).map_err(|e| wrapped(context, Cause::Png(e)))
        };
        let pixels = load(&item.path, "pixels")?;
        let mask = load(&item.mask_path, "mask")?;
        let (width, height) = (item.width as usize, item.height as usize);
        let pixels = match pixels {
            DecodedImage::Paletted { width: w, height: h, pix } if w == width && h == height => pix,
            _ => return Err(msg(format!("figani: separated frame {index} is not a packed indexed PNG"))),
        };
        let mask = match mask {
            DecodedImage::Gray { width: w, height: h, pix } if w == width && h == height => pix,
            _ => return Err(msg(format!("figani: separated frame {index} mask is not a packed grayscale PNG"))),
        };
        animation.frames.push(Frame {
            x: item.x,
            y: item.y,
            width: item.width,
            height: item.height,
            pixels,
            mask: mask.iter().map(|&v| (v != 0) as u8).collect(),
            delay: item.delay,
            raw_byte4: item.raw_byte4,
            raw_byte5: item.raw_byte5,
            raw_byte7: item.raw_byte7,
        });
    }
    Ok(animation)
}

struct StatusDoc {
    schema_version: i64,
    kind: String,
    resource_id: String,
    source_file: String,
    source_resource: i64,
    raw_size: i64,
    header_word_le: Option<i64>,
    status: String,
    reason_code: String,
    evidence: String,
}

fn status_doc(raw: &[u8]) -> std::result::Result<StatusDoc, String> {
    let o = doc::parse_object(raw)?;
    let source = doc::object(&o, "source")?;
    Ok(StatusDoc {
        schema_version: doc::int(&o, "schema_version")?,
        kind: doc::string(&o, "kind")?,
        resource_id: doc::string(&o, "resource_id")?,
        source_file: doc::string(source, "file")?,
        source_resource: doc::int(source, "resource")?,
        raw_size: doc::int(&o, "raw_size")?,
        header_word_le: doc::opt_int(&o, "header_word_le")?,
        status: doc::string(&o, "status")?,
        reason_code: doc::string(&o, "reason_code")?,
        evidence: doc::string(&o, "evidence")?,
    })
}

/// Go `separatedResourceHasZeroHeader`（未公開）：讀 `resource.json` 判斷是否為已證實的零標頭。
pub(crate) fn separated_resource_has_zero_header(animation_root: &str, resource: i64) -> Result<bool> {
    if animation_root.is_empty() || !(0..=999).contains(&resource) {
        return Err(FiganiError::Static("figani: invalid separated resource status request"));
    }
    let directory = Path::new(animation_root).join(format!("FIGANI_{resource:03}"));
    let context = || format!("figani: separated resource status {resource}: ");
    let raw = read(&directory.join("resource.json")).map_err(|e| wrapped(context(), e))?;
    let document = status_doc(&raw).map_err(|e| wrapped(context(), Cause::Json(e)))?;
    let want_id = format!("animation-resource/figani_{resource:03}");
    if document.schema_version != 1
        || document.kind != "animation_resource_status"
        || document.resource_id != want_id
        || document.source_file != "FIGANI.DAT"
        || document.source_resource != resource
        || document.raw_size < 2
    {
        return Err(msg(format!("figani: separated resource status {resource} violates the contract")));
    }
    Ok(document.status == "empty_header_zero"
        && document.reason_code == "zero_header_word"
        && document.evidence == "confirmed"
        && document.header_word_le == Some(0))
}

/// 保存原版 caller 已證實的「selector*3+2 首 word 為零時退到前一資源」規則。缺少或矛盾的狀態
/// 文件不可被誤當成零標頭，亦不可回退讀 FIGANI.DAT。
pub fn load_separated_resource_with_zero_header_fallback(animation_root: &str, resource: i64) -> Result<Animation> {
    load_separated_resource_with_zero_header_fallback_source(animation_root, resource).map(|(animation, _)| animation)
}

/// 同時回傳實際採用的資源序號。繪製端必須把替換美術綁到這個來源（包含已證實的零標頭回退），
/// 而非請求的 selector。Go 版錯誤時回傳的序號固定是 -1，這裡不另外保留。
pub fn load_separated_resource_with_zero_header_fallback_source(
    animation_root: &str,
    resource: i64,
) -> Result<(Animation, i64)> {
    let primary = match load_separated_resource(animation_root, resource) {
        Ok(animation) => return Ok((animation, resource)),
        Err(e) => e,
    };
    let zero = match separated_resource_has_zero_header(animation_root, resource) {
        Ok(zero) => zero,
        Err(status) => {
            return Err(FiganiError::FallbackUnavailable {
                resource,
                primary: Box::new(primary),
                status: Box::new(status),
            });
        }
    };
    if !zero || resource == 0 {
        return Err(primary);
    }
    let animation = load_separated_resource(animation_root, resource - 1)?;
    Ok((animation, resource - 1))
}
