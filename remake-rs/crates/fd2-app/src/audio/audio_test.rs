//! `audio.rs` 的單元測試（Rust 新增；Go 的 `audio_voice_test.go` 測的是 `fd2-game` 那一層，已在 R4 移植）。

use super::*;

/// 8 位元單聲道、取樣率 `rate` 的最小 WAV。
fn wav8(rate: u32, data: &[u8]) -> Vec<u8> {
    let mut out = Vec::new();
    out.extend_from_slice(b"RIFF");
    out.extend_from_slice(&(36 + data.len() as u32).to_le_bytes());
    out.extend_from_slice(b"WAVEfmt ");
    out.extend_from_slice(&16u32.to_le_bytes());
    out.extend_from_slice(&1u16.to_le_bytes());
    out.extend_from_slice(&1u16.to_le_bytes());
    out.extend_from_slice(&rate.to_le_bytes());
    out.extend_from_slice(&rate.to_le_bytes());
    out.extend_from_slice(&1u16.to_le_bytes());
    out.extend_from_slice(&8u16.to_le_bytes());
    out.extend_from_slice(b"data");
    out.extend_from_slice(&(data.len() as u32).to_le_bytes());
    out.extend_from_slice(data);
    out
}

#[test]
fn sniff_by_magic() {
    assert_eq!(sniff(b"OggS...."), Some(Format::Ogg));
    assert_eq!(sniff(b"RIFF...."), Some(Format::Wav));
    assert_eq!(sniff(b"ID3"), None);
    assert_eq!(decode_pcm(b"junk").unwrap_err(), DecodeError::UnknownFormat);
}

/// Ebiten `convert`：`int16(f * (1<<15 - 1))`，Go 的浮點轉整數向零取整；oggvorbis 先夾到 ±1。
#[test]
fn quantize_matches_go_truncation() {
    for (f, want) in [(1.0, 32767), (-1.0, -32767), (2.0, 32767), (-3.0, -32767), (0.5, 16383), (-0.5, -16383)] {
        assert_eq!(quantize(f), want, "{f}");
    }
    assert_eq!(quantize(1.0 / 32767.0 * 0.999), 0);
}

/// 8 位元無號樣本照 `convert.Stereo16` 轉成 `int16(int(b)*0x101 - 32768)`，單聲道複製成雙聲道；44.1 kHz 不重取樣。
#[test]
fn wav8_mono_44k_is_stereo16_without_resampling() {
    let pcm = decode_pcm(&wav8(44100, &[0, 128, 255])).unwrap();
    assert_eq!(pcm, [-32768, -32768, 128, 128, 32767, 32767]);
}

#[test]
fn wav_header_errors_follow_go_messages() {
    let mut bad = wav8(44100, &[1]);
    bad[8..12].copy_from_slice(b"WAVX");
    assert_eq!(parse_wav(&bad).err(), Some(DecodeError::Wav("invalid header: 'WAVE' not found")));
    let mut pcm16 = wav8(44100, &[1]);
    pcm16[20] = 3; // 格式 3（IEEE float）
    assert_eq!(parse_wav(&pcm16).err(), Some(DecodeError::Wav("format must be linear PCM")));
    assert_eq!(parse_wav(&wav8(44100, &[])[..30]).err(), Some(DecodeError::Wav("invalid header")));
}

/// 11025 Hz 的 WAV 經 `rodio` 重取樣成 44.1 kHz 雙聲道；長度約為 4 倍，常數訊號維持常數。
#[test]
fn wav8_11k_resamples_to_44k_stereo() {
    let pcm = decode_pcm(&wav8(11025, &[200; 100])).unwrap();
    let frames = pcm.len() / 2;
    assert!((396..=400).contains(&frames), "frames={frames}");
    let want = quantize(f32::from((200i32 * 0x101 - 32768) as i16) / 32767.0);
    assert!(pcm.iter().all(|&s| s == want), "{:?}", &pcm[..8]);
}

/// 交給 rodio 的樣本乘回 32768 就是解碼結果。
#[test]
fn pcm_source_samples_equal_decoded_pcm() {
    let bytes = wav8(11025, &(0..=255u8).collect::<Vec<_>>());
    let want = decode_pcm(&bytes).unwrap();
    let source = Fd2Sound { encoded: Arc::from(bytes.as_slice()) }.decoder();
    assert_eq!((source.channels().get(), source.sample_rate().get()), (2, 44100));
    let got: Vec<i16> = source.map(|f| (f * 32768.0) as i16).collect();
    assert_eq!(got, want);
}

/// 測試用後端：記錄開始與停止，`done` 裡的 handle 視為播完。
#[derive(Default)]
struct FakeBackend {
    next: u32,
    started: Vec<(u32, bool)>,
    stopped: Vec<u32>,
    done: BTreeSet<u32>,
}

impl AudioBackend for FakeBackend {
    type Handle = u32;
    fn start(&mut self, _: Fd2Sound, looping: bool) -> u32 {
        self.next += 1;
        self.started.push((self.next, looping));
        self.next
    }
    fn stop(&mut self, handle: &u32) {
        self.stopped.push(*handle);
    }
    fn finished(&self, handle: &u32) -> bool {
        self.done.contains(handle)
    }
}

fn collect<'a>(bgm: &'a mut Option<AudioVoice>, sfx: &'a mut [AudioVoice]) -> Vec<(VoiceSlot, &'a mut AudioVoice)> {
    let mut v: Vec<(VoiceSlot, &mut AudioVoice)> = bgm.iter_mut().map(|v| (VoiceSlot::Bgm, v)).collect();
    v.extend(sfx.iter_mut().map(|v| (VoiceSlot::Sfx, v)));
    v
}

fn voice(serial: u64, encoded: &[u8], looping: bool) -> AudioVoice {
    AudioVoice {
        source: format!("v{serial}"),
        playing: true,
        encoded: encoded.to_vec(),
        looping,
        serial,
        ..Default::default()
    }
}

#[test]
fn voice_sync_starts_finishes_stops_and_restarts_by_serial() {
    let sound = wav8(11025, &[128; 16]);
    let mut sync = VoiceSync::default();
    let mut backend = FakeBackend::default();
    let mut bgm = Some(voice(1, &sound, true));
    let mut sfx = vec![voice(2, &sound, false)];
    let r = sync.sync(collect(&mut bgm, &mut sfx), &mut backend);
    assert_eq!(r.started, [(VoiceSlot::Bgm, 1), (VoiceSlot::Sfx, 2)]);
    assert_eq!(backend.started, [(1, true), (2, false)]);

    // 同一幀再同步：不重播。
    assert_eq!(sync.sync(collect(&mut bgm, &mut sfx), &mut backend), SyncReport::default());

    // 音效播完：回報 playing=false；循環的 BGM 不會因 sink 空而結束。
    backend.done.extend([1, 2]);
    let r = sync.sync(collect(&mut bgm, &mut sfx), &mut backend);
    assert_eq!(r.finished, [2]);
    assert!(!sfx[0].playing && bgm.as_ref().unwrap().playing);
    sfx.retain(|v| v.is_playing()); // fd2-game 的 stepSFXVoices

    // Go stopBGM 後重播同一首：內容相同但序號不同，要停掉舊的再從頭播。
    bgm = Some(voice(3, &sound, true));
    let r = sync.sync(collect(&mut bgm, &mut sfx), &mut backend);
    assert_eq!((r.stopped, r.started), (vec![1], vec![(VoiceSlot::Bgm, 3)]));

    // stopBGM：請求消失就停止。
    bgm = None;
    let r = sync.sync(collect(&mut bgm, &mut sfx), &mut backend);
    assert_eq!(r.stopped, [3]);
    assert!(sync.active.is_empty());
}

#[test]
fn voice_sync_rejects_undecodable_and_marks_not_playing() {
    let mut sync = VoiceSync::default();
    let mut backend = FakeBackend::default();
    let mut v = voice(7, b"not audio", false);
    let r = sync.sync(vec![(VoiceSlot::Sfx, &mut v)], &mut backend);
    assert_eq!(r.rejected.len(), 1);
    assert!(!v.playing && backend.started.is_empty());
}

#[test]
fn game_is_an_audio_host() {
    fn host<T: GameAudioHost>() {}
    host::<fd2_game::Game>();
    let mut g = fd2_game::Game::default();
    assert!(g.audio_voices().is_empty());
}
