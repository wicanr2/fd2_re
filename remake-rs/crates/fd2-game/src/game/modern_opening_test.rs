use super::*;
use crate::test_support::temp_dir;
use serde_json::{Value, json};

const FD100: OpeningSource = OpeningSource { source_archive: OpeningArchive::Fdother, source_resource: 100 };

fn png(width: u32, height: u32, depth: png::BitDepth, pixels: &[u8]) -> Vec<u8> {
    let mut bytes = Vec::new();
    let mut encoder = png::Encoder::new(&mut bytes, width, height);
    encoder.set_color(png::ColorType::Rgba);
    encoder.set_depth(depth);
    encoder.write_header().unwrap().write_image_data(pixels).unwrap();
    bytes
}

fn digest(bytes: &[u8]) -> String {
    format!("{:x}", Sha256::digest(bytes))
}

#[test]
fn opacity_uses_the_complete_sixteen_bit_alpha() {
    let opaque = png(1, 1, png::BitDepth::Sixteen, &[8, 1, 16, 2, 32, 3, 255, 255]);
    let partial = png(1, 1, png::BitDepth::Sixteen, &[8, 1, 16, 2, 32, 3, 255, 0]);
    assert_eq!(decode_opaque_png_picture(&opaque).unwrap().read_pixels(), &[8, 16, 32, 255]);
    assert!(decode_opaque_png_picture(&partial).is_none());
    assert_eq!(decode_png_picture(&partial).unwrap().read_pixels(), &[8, 16, 32, 255]);
    assert_eq!(decode_precise_png_picture(&partial).unwrap().read_pixels(), &[7, 15, 31, 255]);
}

#[test]
fn frame_keeps_only_the_current_picture_and_rejects_a_bank_once() {
    let bank = |colors: &[[u8; 4]]| OpeningBank {
        shape: (1, 1),
        frames: colors
            .iter()
            .map(|color| {
                let data = png(1, 1, png::BitDepth::Eight, color);
                EncodedFrame { sha256: digest(&data), bytes: data.into() }
            })
            .collect(),
        verified: BTreeSet::new(),
        rejected: false,
    };
    let fd74 = OpeningSource { source_resource: 74, ..FD100 };
    let mut opening = ModernOpening::default();
    opening.banks.insert(FD100, bank(&[[10, 20, 30, 255], [30, 20, 10, 255], [0, 0, 0, 0]]));
    opening.banks.insert(fd74, bank(&[[10, 20, 30, 255]]));
    let first = opening.frame(FD100, 0).unwrap().unwrap();
    assert!(Arc::ptr_eq(&first, &opening.frame(FD100, 0).unwrap().unwrap()));
    let weak = Arc::downgrade(&first);
    drop(first);
    assert_eq!(opening.frame(FD100, 1).unwrap().unwrap().read_pixels(), &[30, 20, 10, 255]);
    assert!(weak.upgrade().is_none());
    assert!(opening.frame(FD100, 2).unwrap().is_none());
    assert!(opening.frame(FD100, 0).unwrap().is_none());
    assert!(opening.frame(fd74, 0).unwrap().is_some());
    assert!(opening.frame(fd74, 1).is_err());
    assert_eq!(opening.take_notices().len(), 1);
    assert!(opening.take_notices().is_empty());
}

#[test]
fn opening_bank_checks_headers_before_decoding_pixels() {
    let dir = temp_dir("r6-opening");
    let surface = "surfaces/FDOTHER_100/resource.json";
    let palette = "palette/fdother_099.json";
    for file in [surface, palette] {
        let path = dir.join(file);
        std::fs::create_dir_all(path.parent().unwrap()).unwrap();
        std::fs::write(path, b"{}").unwrap();
    }
    let source_digest = FD100.metadata_digest(&dir).unwrap();
    let plan = json!({
        "schema_version": 1, "kind": "opening_hd_art_derivation",
        "source_archive": "FDOTHER.DAT", "source_resource": 100,
        "source_metadata_sha256": source_digest, "frames": [{}]
    })
    .to_string();
    std::fs::write(dir.join("plan.json"), &plan).unwrap();
    let pixels = vec![0; 1920 * 1200 * 4];
    let transparent = png(1920, 1200, png::BitDepth::Eight, &pixels);
    let frame_file = dir.join("frame.png");
    std::fs::write(&frame_file, &transparent).unwrap();
    let asset = json!({
        "role": "opening_presentation_hd_set",
        "status": "runtime_ready", "consumer_contract": "opening_native_schedule_rgba_hd6_v1",
        "layout_policy": "full_native_canvas_hd6", "alpha_contract": "opaque",
        "source_archive": "FDOTHER.DAT", "source_resource": 100,
        "width": 1920, "height": 1200, "frame_count": 1,
        "files":["frame.png"], "frame_sha256":[digest(&transparent)],
        "master_file":"plan.json", "source_files":["plan.json"], "source_sha256":[digest(plan.as_bytes())],
        "source_metadata_sha256": source_digest
    });
    let admit = |asset: Value| serde_json::from_value::<OpeningAsset>(asset).unwrap().admit(&dir, &dir);
    let mut opening = ModernOpening::default();
    opening.banks.insert(FD100, admit(asset.clone()).unwrap());
    assert!(opening.banks[&FD100].verified.is_empty());
    assert!(opening.frame(FD100, 0).unwrap().is_none());
    for (key, value, message) in [
        ("width", json!(1919), "invalid bank contract"),
        ("source_metadata_sha256", json!("wrong"), "original source metadata"),
        ("frame_sha256", json!(["wrong"]), "SHA-256 differs"),
        ("files", json!(["../frame.png"]), "unsafe or duplicate frame path"),
        ("master_file", json!("absent.png"), "master or unique plan absent"),
    ] {
        let mut invalid = asset.clone();
        invalid[key] = value;
        assert!(admit(invalid).unwrap_err().message().contains(message));
    }
    let catalog = dir.join("catalog.json");
    let load = |assets: Value| {
        let raw = json!({"schema_version": 1, "theme_id": MODERN_HANDPAINTED_THEME_ID, "assets": assets});
        std::fs::write(&catalog, raw.to_string()).unwrap();
        ModernOpening::load(&catalog, &dir, &dir)
    };
    assert!(load(json!([asset.clone()])).unwrap_err().message().contains("scene group incomplete"));
    assert!(load(json!([asset.clone(), asset.clone()])).unwrap_err().message().contains("duplicate source"));
    std::fs::remove_file(&frame_file).unwrap();
    assert!(admit(asset.clone()).is_err());
    // PNG 的第一個 IDAT 標頭可讀，入場時尚未解碼被截斷的內容。
    let idat = transparent.windows(4).position(|w| w == b"IDAT").unwrap();
    let corrupt = &transparent[..idat + 5];
    std::fs::write(&frame_file, corrupt).unwrap();
    let mut truncated = asset;
    truncated["frame_sha256"][0] = json!(digest(corrupt));
    let bank = admit(truncated).unwrap();
    opening.banks.insert(FD100, bank);
    assert!(opening.frame(FD100, 0).unwrap().is_none());
    std::fs::remove_dir_all(dir).unwrap();
}

#[test]
#[ignore = "正式私人素材與 Go 黃金由 tools/rust_r6_opening.sh 安排"]
fn real_opening_matches_go() {
    let env_path = |var| std::path::PathBuf::from(std::env::var_os(var).unwrap());
    let mut opening = ModernOpening::load(
        &env_path("FD2_TEST_OPENING_CATALOG"),
        &env_path("FD2_TEST_OPENING_PACK"),
        &env_path("FD2_ASSET_PACK"),
    )
    .unwrap()
    .unwrap();
    assert_eq!(opening.banks.len(), 12);
    assert!(opening.current.is_none());
    let sources: Vec<_> = opening.banks.keys().copied().collect();
    let mut report = String::new();
    for source in sources {
        let bank = &opening.banks[&source];
        assert!(bank.verified.is_empty());
        for frame in 0..bank.frames.len() {
            let picture = opening.frame(source, frame).unwrap().unwrap();
            use std::fmt::Write as _;
            writeln!(
                report,
                "{} {} {} {}",
                source.source_archive.name(),
                source.source_resource,
                frame,
                digest(picture.read_pixels())
            )
            .unwrap();
        }
    }
    assert!(opening.take_notices().is_empty());
    assert_eq!(report.lines().count(), 173);
    let output = env_path("FD2_R6_OPENING_OUT");
    std::fs::write(output.join("rust-opening.txt"), &report).unwrap();
    assert_eq!(report, std::fs::read_to_string(output.join("go-opening.txt")).unwrap());
    println!("開場 173 格與 Go 逐格像素雜湊相同，序列 SHA-256 {}", digest(report.as_bytes()));
}
