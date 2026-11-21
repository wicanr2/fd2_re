use super::*;
use crate::test_support::temp_dir;

fn bank(root: &Path, source: Source, frames: &[([u8; 4], bool)]) -> SkyKeyBank {
    let offset = source.offset();
    let mut rows = Vec::new();
    let row = |index, blank| SourceRow {
        index,
        blank,
        rect: Descriptor::default(),
        canvas: String::new(),
        canvas_ff: String::new(),
        pixel_sha256: String::new(),
        dac_sha256: String::new(),
        source_colours: vec![0],
    };
    if offset == 1 {
        rows.push(row(0, false));
    }
    let mut files = Vec::new();
    for (i, (rgba, blank)) in frames.iter().enumerate() {
        let raw = {
            let mut raw = Vec::new();
            let mut encoder = png::Encoder::new(&mut raw, 2, 1);
            encoder.set_color(png::ColorType::Rgba);
            let mut data = rgba.to_vec();
            data.extend_from_slice(&[0, 0, 0, if source == Source::Fd34 { 0 } else { 255 }]);
            encoder.write_header().unwrap().write_image_data(&data).unwrap();
            raw
        };
        let path = root.join(format!("{i}.png"));
        std::fs::write(&path, &raw).unwrap();
        let sha = format!("{:x}", Sha256::digest(&raw));
        files.push(FrameFile { file: CheckedFile { path, sha }, size: (2, 1) });
        rows.push(row(i + offset, *blank));
    }
    SkyKeyBank {
        source,
        rows,
        files,
        plan_file: CheckedFile { path: root.join("plan.json"), sha: String::new() },
        recent: VecDeque::new(),
        decoded_sha: BTreeMap::new(),
        rejected: false,
        notice: None,
    }
}

#[test]
fn recent_frames_share_by_content_but_recheck_every_path() {
    let dir = temp_dir("sky-key-recent");
    let white = [255; 4];
    let mut bank = bank(
        &dir,
        Source::Fd34,
        &[(white, false), (white, false), ([12, 0, 0, 255], false), ([13, 0, 0, 255], false), ([14, 0, 0, 255], false)],
    );
    assert!(bank.picture(0).is_err());
    let mut first = bank.picture(1).unwrap().unwrap();
    let original = Arc::downgrade(&first);
    assert!(Arc::ptr_eq(&first, &bank.picture(2).unwrap().unwrap()));
    Arc::make_mut(&mut first).clear();
    assert_ne!(first.read_pixels(), bank.picture(1).unwrap().unwrap().read_pixels());
    std::fs::remove_file(dir.join("1.png")).unwrap();
    assert!(bank.picture(2).is_err());
    assert!(!bank.rejected);
    for i in 3..=5 {
        assert!(bank.picture(i).unwrap().is_some());
    }
    assert_eq!(bank.recent.len(), 3);
    assert!(original.upgrade().is_none());
    assert_eq!(&bank.picture(1).unwrap().unwrap().read_pixels()[..4], &white);
    std::fs::remove_dir_all(dir).unwrap();
}

#[test]
fn repeated_asset_checks_each_native_black_frame_and_rejects_once() {
    let dir = temp_dir("sky-key-black");
    let black = [0, 0, 0, 255];
    let mut ani = bank(&dir, Source::Ani0, &[(black, true), (black, false)]);
    assert!(ani.picture(0).unwrap().is_some());
    assert!(ani.picture(1).unwrap().is_none());
    assert!(ani.recent.is_empty());
    assert!(ani.picture(0).unwrap().is_none());
    assert!(ani.take_notice().unwrap().contains("frame 1"));
    assert!(ani.take_notice().is_none());
    let mut fd34 = bank(&dir, Source::Fd34, &[([0, 0, 0, 128], false)]);
    assert!(fd34.picture(1).unwrap().is_none());
    assert!(fd34.take_notice().is_some());
    std::fs::remove_dir_all(dir).unwrap();
}

#[test]
#[ignore = "正式私人素材與 Go 黃金由 tools/rust_r6_opening.sh 安排"]
fn real_sky_key_matches_go() {
    use crate::game::modern_opening::{ModernOpening, OpeningArchive, OpeningSource};
    use std::fmt::Write as _;

    let env_path = |var| PathBuf::from(std::env::var_os(var).unwrap());
    let original = env_path("FD2_ASSET_PACK");
    let mut opening =
        ModernOpening::load(&env_path("FD2_TEST_OPENING_CATALOG"), &env_path("FD2_TEST_OPENING_PACK"), &original)
            .unwrap()
            .unwrap();
    let mut frames =
        fdother::load_separated_ch20_sky_key_frames(original.join("animations/fdother_034_ch20_sky_key")).unwrap();
    let mut clip = afm::load_separated_resource(original.join("animations"), 0).unwrap();
    assert!(opening.sky_key.preflight(&frames, &clip).unwrap());
    frames[100].width += 1;
    assert!(opening.sky_key.fd34.as_ref().unwrap().rows[100].validate_fdother(&frames[100]).is_err());
    clip.palettes[95][0] ^= 1;
    assert!(
        opening.sky_key.ani0.as_ref().unwrap().rows[95]
            .validate_ani(&clip.indexed_frames[95], &clip.palettes[95])
            .is_err()
    );
    clip.palettes[95][0] ^= 1;
    clip.indexed_frames[95][0] ^= 1;
    assert!(
        opening.sky_key.ani0.as_ref().unwrap().rows[95]
            .validate_ani(&clip.indexed_frames[95], &clip.palettes[95])
            .is_err()
    );
    let ani0 = OpeningSource { source_archive: OpeningArchive::Ani, source_resource: 0 };
    let mut report = String::new();
    for i in 0..96 {
        let picture = opening.frame(ani0, i).unwrap().unwrap();
        if i == 0 {
            assert!(Arc::ptr_eq(&picture, &opening.sky_key.ani0.as_mut().unwrap().picture(i).unwrap().unwrap()));
        }
        writeln!(report, "ANI.DAT 0 {i} {:x}", Sha256::digest(picture.read_pixels())).unwrap();
    }
    let fd34 = opening.sky_key.fd34.as_mut().unwrap();
    for i in 1..=100 {
        let picture = fd34.picture(i).unwrap().unwrap();
        writeln!(report, "FDOTHER.DAT 34 {i} {:x}", Sha256::digest(picture.read_pixels())).unwrap();
    }
    assert!(opening.take_notices().is_empty());
    assert_eq!(report.lines().count(), 196);
    let output = env_path("FD2_R6_OPENING_OUT");
    std::fs::write(output.join("rust-sky-key.txt"), &report).unwrap();
    assert_eq!(report, std::fs::read_to_string(output.join("go-sky-key.txt")).unwrap());
    println!("共用圖庫 196 格與 Go 相同，序列 SHA-256 {:x}", Sha256::digest(report.as_bytes()));
}
