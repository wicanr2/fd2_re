//! 音訊：解碼 BGM 與音效，依 `fd2-game` 的 [`AudioVoice`] 狀態以 Bevy 播放（R5 階段 4，代號 r5o）。
//!
//! Go 對照：`remake/cmd/fd2/audio.go`。Go 在 `audio.NewContext(44100)` 下把 OGG（`vorbis.DecodeWithSampleRate`）與
//! WAV（`wav.DecodeWithSampleRate`）解成 44.1 kHz、16 位元、雙聲道交錯的 PCM，BGM 以 `audio.NewInfiniteLoop`
//! 循環，音效以 `audio.NewPlayerFromBytes` 一次性播放。`fd2-game` 只記錄播放請求（見其 `audio.rs` 模組文件），
//! 本模組負責：
//!
//! - **解碼**（[`open_stream`]、[`decode_pcm`]）：輸出格式與 Go 相同（44.1 kHz、雙聲道、`i16`）。OGG 以 `lewton`
//!   解碼，WAV 以本模組的 RIFF 解析器（只收 Go `wav` 套件接受的線性 PCM 8／16 位元、單／雙聲道）。取樣率不是
//!   44.1 kHz 時以 `rodio` 的 `SampleRateConverter` 轉換。浮點轉 16 位元照 Go（Ebiten `convert`）：夾到 ±1 後乘
//!   32767、向零取整；單聲道複製成雙聲道。與 Go 的逐樣本差異與原因見 `fd2-game/PORTING.md`「R5 階段 4：r5o」，
//!   接受與否由使用者定案，本模組的重取樣與取整順序在定案前不照 Go 改。
//! - **播放**（[`VoiceSync`]、[`Fd2AudioPlugin`]）：每幀讀 `Game::audio_voices_mut`，以 [`AudioVoice::serial`]
//!   分辨播放器。新序號開始播放（BGM `looping` 用 `PlaybackSettings::LOOP`，其餘 `ONCE`）；不在清單、已
//!   `closed` 或 `playing == false` 的序號停止；一次性播放的 sink 播完時把 `playing` 設回 `false`（Go `IsPlaying()`），
//!   `fd2-game` 的 `stepSFXVoices` 再回收。音量照 Go 一律 1.0（Go 沒有呼叫 `SetVolume`）；`FD2_MUTE`、截圖模式與
//!   系統設定的音樂／音效開關由 `fd2-game` 在建立請求時判斷，與 Go 相同。
//! - 解碼失敗：Go 的 BGM 解不開就不換曲，音效庫解不開整批拒收；Rust 在 `fd2-game` 只檢查編碼位元組非空
//!   （KNOWN_DIFFERENCES `audio-decode-deferred`），外殼解不開時不播放並把 `playing` 設成 `false`。

use std::collections::{BTreeMap, BTreeSet};
use std::io::Cursor;
use std::num::NonZero;
use std::sync::Arc;
use std::time::Duration;

use bevy::audio::{AddAudioSource, AudioPlayer, AudioSink, AudioSinkPlayback, Decodable, PlaybackSettings, Source};
use bevy::prelude::*;
use bevy::transform::TransformSystems;
use fd2_game::game::{AudioVoice, VoiceSlot};
use lewton::inside_ogg::OggStreamReader;
use lewton::samples::InterleavedSamples;

/// Go `audio.NewContext(44100)`。
pub const OUTPUT_RATE: u32 = 44100;
/// Go 的播放格式固定雙聲道。
pub const OUTPUT_CHANNELS: u16 = 2;

/// 解碼失敗的原因。
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum DecodeError {
    /// 不是 `OggS` 也不是 `RIFF`。
    UnknownFormat,
    /// `lewton` 回報的錯誤。
    Ogg(String),
    /// WAV 標頭不合 Go `wav` 套件的要求（訊息照 Go）。
    Wav(&'static str),
    /// Go `vorbis: number of channels must be 1 or 2 but was %d`。
    Channels(u16),
    /// 取樣率為 0。
    SampleRate,
}

impl std::fmt::Display for DecodeError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            DecodeError::UnknownFormat => write!(f, "audio: unknown format"),
            DecodeError::Ogg(e) => write!(f, "vorbis: {e}"),
            DecodeError::Wav(e) => write!(f, "wav: {e}"),
            DecodeError::Channels(n) => write!(f, "number of channels must be 1 or 2 but was {n}"),
            DecodeError::SampleRate => write!(f, "sample rate is zero"),
        }
    }
}

impl std::error::Error for DecodeError {}

/// 編碼格式（依開頭四個位元組）。
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum Format {
    Ogg,
    Wav,
}

/// 依開頭位元組判斷格式。
pub fn sniff(bytes: &[u8]) -> Option<Format> {
    match bytes.get(..4) {
        Some(b"OggS") => Some(Format::Ogg),
        Some(b"RIFF") => Some(Format::Wav),
        _ => None,
    }
}

/// 原始取樣率與聲道數下的樣本（重取樣之前）。
pub struct NativeStream {
    pub channels: u16,
    pub sample_rate: u32,
    /// 交錯樣本；OGG 是 `lewton` 的浮點輸出，WAV 是整數樣本除以 32767。
    pub samples: Box<dyn Iterator<Item = f32> + Send>,
    /// WAV 的整數樣本（不經浮點）；OGG 為 `None`。
    pub integer: Option<Vec<i16>>,
}

/// 以 `lewton` 逐封包解碼的浮點樣本串流。
struct OggSamples {
    reader: OggStreamReader<Cursor<Arc<[u8]>>>,
    packet: Vec<f32>,
    next: usize,
    done: bool,
}

impl Iterator for OggSamples {
    type Item = f32;

    fn next(&mut self) -> Option<f32> {
        while self.next >= self.packet.len() {
            if self.done {
                return None;
            }
            match self.reader.read_dec_packet_generic::<InterleavedSamples<f32>>() {
                Ok(Some(p)) => {
                    self.packet = p.samples;
                    self.next = 0;
                }
                // 解到一半出錯：Go 的 `io.ReadAll` 會回傳錯誤；這裡在錯誤處結束串流（只可能出現在損壞檔）。
                Ok(None) | Err(_) => self.done = true,
            }
        }
        let s = self.packet[self.next];
        self.next += 1;
        Some(s)
    }
}

/// 開啟原始取樣率的樣本串流（不重取樣、不轉 16 位元）。
pub fn open_native(bytes: Arc<[u8]>) -> Result<NativeStream, DecodeError> {
    match sniff(&bytes) {
        Some(Format::Ogg) => {
            let reader = OggStreamReader::new(Cursor::new(bytes)).map_err(|e| DecodeError::Ogg(e.to_string()))?;
            let channels = u16::from(reader.ident_hdr.audio_channels);
            let sample_rate = reader.ident_hdr.audio_sample_rate;
            if channels != 1 && channels != 2 {
                return Err(DecodeError::Channels(channels));
            }
            if sample_rate == 0 {
                return Err(DecodeError::SampleRate);
            }
            let samples = OggSamples { reader, packet: Vec::new(), next: 0, done: false };
            Ok(NativeStream { channels, sample_rate, samples: Box::new(samples), integer: None })
        }
        Some(Format::Wav) => {
            let wav = parse_wav(&bytes)?;
            let floats: Vec<f32> = wav.samples.iter().map(|&s| f32::from(s) / 32767.0).collect();
            Ok(NativeStream {
                channels: wav.channels,
                sample_rate: wav.sample_rate,
                samples: Box::new(floats.into_iter()),
                integer: Some(wav.samples),
            })
        }
        None => Err(DecodeError::UnknownFormat),
    }
}

/// Ebiten `convert` 的浮點轉 16 位元：夾到 ±1、乘 32767、向零取整（Go `int16(f * (1<<15 - 1))`）。
pub fn quantize(f: f32) -> i16 {
    (f.clamp(-1.0, 1.0) * 32767.0) as i16
}

/// 44.1 kHz、雙聲道交錯 `i16` 的串流。
pub struct PcmStream {
    inner: Box<dyn Iterator<Item = i16> + Send>,
}

impl Iterator for PcmStream {
    type Item = i16;

    fn next(&mut self) -> Option<i16> {
        self.inner.next()
    }
}

/// 以 fd2-app 的解碼路徑開啟串流：原始樣本 →（取樣率不同時）`rodio` 重取樣 → 轉 16 位元 → 單聲道複製成雙聲道。
/// 取樣率相同的 WAV 直接用整數樣本（與 Go 一樣不經浮點）。
pub fn open_stream(bytes: Arc<[u8]>) -> Result<PcmStream, DecodeError> {
    let native = open_native(bytes)?;
    let channels = native.channels;
    let samples: Box<dyn Iterator<Item = i16> + Send> = match (native.integer, native.sample_rate == OUTPUT_RATE) {
        (Some(integer), true) => Box::new(integer.into_iter()),
        (_, true) => Box::new(native.samples.map(quantize)),
        (_, false) => {
            let from = NonZero::new(native.sample_rate).ok_or(DecodeError::SampleRate)?;
            let to = NonZero::new(OUTPUT_RATE).unwrap();
            let ch = NonZero::new(channels).unwrap();
            Box::new(rodio::conversions::SampleRateConverter::new(native.samples, from, to, ch).map(quantize))
        }
    };
    let inner: Box<dyn Iterator<Item = i16> + Send> =
        if channels == 1 { Box::new(samples.flat_map(|s| [s, s])) } else { samples };
    Ok(PcmStream { inner })
}

/// 整檔解碼（[`open_stream`] 收集成 `Vec`）。
pub fn decode_pcm(bytes: &[u8]) -> Result<Vec<i16>, DecodeError> {
    Ok(open_stream(Arc::from(bytes))?.collect())
}

/// WAV 解析結果：整數樣本（8 位元已照 Go 轉成 16 位元）。
pub struct Wav {
    pub channels: u16,
    pub sample_rate: u32,
    pub samples: Vec<i16>,
}

/// 照 Ebiten `audio/wav` 的 `decode`：RIFF／WAVE 標頭、`fmt ` 必須是線性 PCM（格式 1）、1 或 2 聲道、8 或 16 位元，
/// 遇到 `data` 區塊就停。8 位元無號樣本照 Ebiten `convert.Stereo16` 轉成 `int16(int(b)*0x101 - 32768)`。
/// `data` 宣告長度超過檔案時只讀到檔尾（Go 的 `stream.Read` 讀到 EOF 為止）。
pub fn parse_wav(bytes: &[u8]) -> Result<Wav, DecodeError> {
    let header = bytes.get(..12).ok_or(DecodeError::Wav("invalid header"))?;
    if &header[0..4] != b"RIFF" {
        return Err(DecodeError::Wav("invalid header: 'RIFF' not found"));
    }
    if &header[8..12] != b"WAVE" {
        return Err(DecodeError::Wav("invalid header: 'WAVE' not found"));
    }
    let mut pos = 12usize;
    let (mut channels, mut bits, mut rate) = (0u16, 0u16, 0u32);
    let le16 = |b: &[u8], at: usize| u16::from_le_bytes([b[at], b[at + 1]]);
    let le32 = |b: &[u8], at: usize| u32::from_le_bytes([b[at], b[at + 1], b[at + 2], b[at + 3]]);
    let data = loop {
        let chunk = bytes.get(pos..pos + 8).ok_or(DecodeError::Wav("invalid header"))?;
        let size = le32(chunk, 4) as usize;
        pos += 8;
        match &chunk[0..4] {
            b"fmt " => {
                if size < 16 {
                    return Err(DecodeError::Wav("invalid header: maybe non-PCM file?"));
                }
                let fmt = bytes.get(pos..pos + size).ok_or(DecodeError::Wav("invalid header"))?;
                if le16(fmt, 0) != 1 {
                    return Err(DecodeError::Wav("format must be linear PCM"));
                }
                channels = le16(fmt, 2);
                if channels != 1 && channels != 2 {
                    return Err(DecodeError::Channels(channels));
                }
                bits = le16(fmt, 14);
                if bits != 8 && bits != 16 {
                    return Err(DecodeError::Wav("bits per sample must be 8 or 16"));
                }
                rate = le32(fmt, 4);
                pos += size;
            }
            b"data" => break &bytes[pos..bytes.len().min(pos + size)],
            _ => {
                bytes.get(pos..pos + size).ok_or(DecodeError::Wav("invalid header"))?;
                pos += size;
            }
        }
    };
    if rate == 0 {
        return Err(DecodeError::SampleRate);
    }
    let samples = if bits == 8 {
        data.iter().map(|&b| (i32::from(b) * 0x101 - 32768) as i16).collect()
    } else {
        data.as_chunks::<2>().0.iter().map(|c| i16::from_le_bytes(*c)).collect()
    };
    Ok(Wav { channels, sample_rate: rate, samples })
}

// ---------------- Bevy 播放 ----------------

/// 一個要播放的音檔（編碼後位元組，與 `AudioVoice::encoded` 相同）。
#[derive(Asset, TypePath, Clone)]
pub struct Fd2Sound {
    pub encoded: Arc<[u8]>,
}

/// 交給 Bevy／rodio 的樣本來源：[`open_stream`] 的 `i16` 除以 32768 成為 `rodio::Sample`（`f32`，2 的冪次，
/// 不損失精度；乘回 32768 就是解碼結果）。
pub struct PcmSource {
    stream: Option<PcmStream>,
}

impl Iterator for PcmSource {
    type Item = f32;

    fn next(&mut self) -> Option<f32> {
        self.stream.as_mut()?.next().map(|s| f32::from(s) / 32768.0)
    }
}

impl Source for PcmSource {
    fn current_span_len(&self) -> Option<usize> {
        None
    }

    fn channels(&self) -> bevy::audio::ChannelCount {
        NonZero::new(OUTPUT_CHANNELS).unwrap()
    }

    fn sample_rate(&self) -> bevy::audio::SampleRate {
        NonZero::new(OUTPUT_RATE).unwrap()
    }

    fn total_duration(&self) -> Option<Duration> {
        None
    }
}

impl Decodable for Fd2Sound {
    type Decoder = PcmSource;

    /// [`VoiceSync`] 只對 [`open_stream`] 成功的位元組建立 `Fd2Sound`；這裡再失敗時給空來源（不出聲）。
    fn decoder(&self) -> PcmSource {
        PcmSource { stream: open_stream(self.encoded.clone()).ok() }
    }
}

/// 播放後端：[`VoiceSync`] 只透過它開始、停止與查詢播放器，Bevy 與測試各自實作。
pub trait AudioBackend {
    type Handle: Clone;
    /// 開始播放；`looping` 對應 Go `audio.NewInfiniteLoop`。
    fn start(&mut self, sound: Fd2Sound, looping: bool) -> Self::Handle;
    /// 停止並釋放（Go `Close()`）。
    fn stop(&mut self, handle: &Self::Handle);
    /// 一次性播放已自然結束（Go `IsPlaying()` 由 true 轉 false）。
    fn finished(&self, handle: &Self::Handle) -> bool;
}

/// 外殼正在播放的一個請求。
#[derive(Debug, Clone)]
pub struct ActiveVoice<H> {
    pub slot: VoiceSlot,
    pub handle: H,
    pub looping: bool,
}

/// 一次同步做了什麼（給驗證與除錯）。
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct SyncReport {
    pub started: Vec<(VoiceSlot, u64)>,
    pub stopped: Vec<u64>,
    pub finished: Vec<u64>,
    pub rejected: Vec<(u64, String)>,
}

/// `AudioVoice` 與播放器的對應，以序號為鍵。
pub struct VoiceSync<H> {
    pub active: BTreeMap<u64, ActiveVoice<H>>,
    rejected: BTreeSet<u64>,
    /// 同一份編碼位元組只驗證一次（音效每次播放都是新的 `AudioVoice`，內容重複）。
    checked: BTreeMap<Arc<[u8]>, Result<(), DecodeError>>,
}

impl<H> Default for VoiceSync<H> {
    fn default() -> Self {
        Self { active: BTreeMap::new(), rejected: BTreeSet::new(), checked: BTreeMap::new() }
    }
}

impl<H: Clone> VoiceSync<H> {
    /// 每幀呼叫一次（`fd2-game` 的 `Update` 之後）。
    pub fn sync<B: AudioBackend<Handle = H>>(
        &mut self,
        voices: Vec<(VoiceSlot, &mut AudioVoice)>,
        backend: &mut B,
    ) -> SyncReport {
        let mut report = SyncReport::default();
        let live: BTreeSet<u64> = voices.iter().filter(|(_, v)| v.is_playing()).map(|(_, v)| v.serial).collect();
        let gone: Vec<u64> = self.active.keys().copied().filter(|serial| !live.contains(serial)).collect();
        for serial in gone {
            let active = self.active.remove(&serial).unwrap();
            backend.stop(&active.handle);
            report.stopped.push(serial);
        }
        for (slot, voice) in voices {
            if !voice.is_playing() {
                continue;
            }
            let serial = voice.serial;
            if let Some(active) = self.active.get(&serial) {
                if !active.looping && backend.finished(&active.handle) {
                    backend.stop(&active.handle);
                    self.active.remove(&serial);
                    voice.playing = false;
                    report.finished.push(serial);
                }
                continue;
            }
            if self.rejected.contains(&serial) {
                voice.playing = false;
                continue;
            }
            let encoded: Arc<[u8]> = Arc::from(voice.encoded.as_slice());
            let verdict =
                self.checked.entry(encoded.clone()).or_insert_with(|| open_stream(encoded.clone()).map(|_| ())).clone();
            match verdict {
                Ok(()) => {
                    let handle = backend.start(Fd2Sound { encoded }, voice.looping);
                    self.active.insert(serial, ActiveVoice { slot, handle, looping: voice.looping });
                    report.started.push((slot, serial));
                }
                Err(e) => {
                    warn!("音訊解碼失敗，不播放（{slot:?}，{}）：{e}", voice.source);
                    self.rejected.insert(serial);
                    voice.playing = false;
                    report.rejected.push((serial, e.to_string()));
                }
            }
        }
        report
    }
}

/// 提供播放請求的 non-send 資源（`fd2_game::Game` 或測試替身）。`Game` 不是 `Send`／`Sync`（欄位有
/// `Box<dyn FnMut + Send>`、借用 `&dyn Fn` 的 `FrameInput`、`mpsc::Receiver` 等），所以以
/// `App::insert_non_send` 放進 Bevy，系統用 `NonSendMut` 取得（在主執行緒執行）。
pub trait GameAudioHost: 'static {
    fn audio_voices(&mut self) -> Vec<(VoiceSlot, &mut AudioVoice)>;
}

/// [`VoiceSync`] 的 Bevy 資源。
#[derive(Resource, Default)]
pub struct Fd2AudioState {
    pub sync: VoiceSync<Entity>,
    /// 最近一次同步的結果。
    pub last: SyncReport,
}

/// Bevy 後端：每個播放請求一個帶 `AudioPlayer<Fd2Sound>` 的實體。
struct BevyBackend<'a, 'w, 's, 'qw, 'qs> {
    commands: &'a mut Commands<'w, 's>,
    sounds: &'a mut Assets<Fd2Sound>,
    sinks: &'a Query<'qw, 'qs, &'static AudioSink>,
}

impl AudioBackend for BevyBackend<'_, '_, '_, '_, '_> {
    type Handle = Entity;

    fn start(&mut self, sound: Fd2Sound, looping: bool) -> Entity {
        let settings = if looping { PlaybackSettings::LOOP } else { PlaybackSettings::ONCE };
        self.commands.spawn((AudioPlayer(self.sounds.add(sound)), settings)).id()
    }

    fn stop(&mut self, handle: &Entity) {
        if let Ok(sink) = self.sinks.get(*handle) {
            sink.stop();
        }
        if let Ok(mut entity) = self.commands.get_entity(*handle) {
            entity.despawn();
        }
    }

    /// sink 在 `PostUpdate` 的 `AudioPlaybackSystems` 才建立；還沒建立時視為仍在播放。
    fn finished(&self, handle: &Entity) -> bool {
        self.sinks.get(*handle).is_ok_and(|sink| sink.empty())
    }
}

/// 同步系統：讀 `H` 的播放請求，開始或停止 Bevy 的播放器。
pub fn sync_audio<H: GameAudioHost>(
    mut host: NonSendMut<H>,
    mut state: ResMut<Fd2AudioState>,
    mut commands: Commands,
    mut sounds: ResMut<Assets<Fd2Sound>>,
    sinks: Query<&'static AudioSink>,
) {
    let mut backend = BevyBackend { commands: &mut commands, sounds: &mut sounds, sinks: &sinks };
    let voices = host.audio_voices();
    let state = &mut *state;
    state.last = state.sync.sync(voices, &mut backend);
}

/// 註冊 [`Fd2Sound`] 與同步系統。需要先加 Bevy 的 `AssetPlugin` 與 `AudioPlugin`（`DefaultPlugins` 都有），
/// 並以 `App::insert_non_send` 放入 `H`。
/// 同步在 `PostUpdate` 的 `TransformSystems::Propagate` 之前執行；Bevy 的播放系統（私有的 `AudioPlaybackSystems`）
/// 排在 `Propagate` 之後，所以同一幀就建立 sink。`fd2-game` 的 `Update` 要排在它之前（`Update`／`FixedUpdate` 都在
/// `PostUpdate` 之前）。
pub struct Fd2AudioPlugin<H>(std::marker::PhantomData<fn() -> H>);

impl<H> Default for Fd2AudioPlugin<H> {
    fn default() -> Self {
        Self(std::marker::PhantomData)
    }
}

impl<H: GameAudioHost> Plugin for Fd2AudioPlugin<H> {
    fn build(&self, app: &mut App) {
        app.add_audio_source::<Fd2Sound>()
            .init_resource::<Fd2AudioState>()
            .add_systems(PostUpdate, sync_audio::<H>.before(TransformSystems::Propagate));
    }
}

impl GameAudioHost for fd2_game::Game {
    fn audio_voices(&mut self) -> Vec<(VoiceSlot, &mut AudioVoice)> {
        self.audio_voices_mut()
    }
}

#[cfg(test)]
#[path = "audio/audio_test.rs"]
mod audio_test;
