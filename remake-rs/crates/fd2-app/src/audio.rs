//! 音訊：解碼 BGM 與音效，依 `fd2-game` 的 [`AudioVoice`] 狀態以 Bevy 播放（R5 階段 4，代號 r5o；解碼改寫 r5v）。
//!
//! Go 對照：`remake/cmd/fd2/audio.go`。Go 在 `audio.NewContext(44100)` 下把 OGG（`vorbis.DecodeWithSampleRate`）與
//! WAV（`wav.DecodeWithSampleRate`）解成 44.1 kHz、16 位元、雙聲道交錯的 PCM，BGM 以 `audio.NewInfiniteLoop`
//! 循環，音效以 `io.ReadAll` 後 `audio.NewPlayerFromBytes` 一次性播放。`fd2-game` 只記錄播放請求（見其
//! `audio.rs` 模組文件），本模組負責：
//!
//! - **解碼**（[`open_stream`]、[`decode_pcm`]）：照 Go 的解碼鏈逐行移植（使用者 2026-11-16 定案 C 案），與 Go
//!   逐樣本相同。OGG 走 [`crate::go_ebiten_audio::vorbis`]（Ebiten v2.6.6）→ [`crate::go_oggvorbis`]
//!   （jfreymuth/oggvorbis v1.0.5）→ [`crate::go_vorbis`]（jfreymuth/vorbis v1.0.2），WAV 走
//!   [`crate::go_ebiten_audio::wav`]；取樣率不是 44.1 kHz 時經 [`crate::go_ebiten_audio::convert::Resampling`]
//!   （Ebiten 的加窗 sinc）。BGM 結尾補零到最後 granule 的靜音照 Go 保留。採串流解碼，BGM 不必整首先解完。
//! - **播放**（[`VoiceSync`]、[`Fd2AudioPlugin`]）：每幀讀 `Game::audio_voices_mut`，以 [`AudioVoice::serial`]
//!   分辨播放器。新序號開始播放（BGM `looping` 用 `PlaybackSettings::LOOP`，其餘 `ONCE`）；不在清單、已
//!   `closed` 或 `playing == false` 的序號停止；一次性播放的 sink 播完時把 `playing` 設回 `false`（Go `IsPlaying()`），
//!   `fd2-game` 的 `stepSFXVoices` 再回收。音量照 Go 一律 1.0（Go 沒有呼叫 `SetVolume`）；`FD2_MUTE`、截圖模式與
//!   系統設定的音樂／音效開關由 `fd2-game` 在建立請求時判斷，與 Go 相同。BGM 的 Go `InfiniteLoop` 每圈與第一圈
//!   逐位元組相同（見 [`crate::go_ebiten_audio`] 模組文件），`LOOP` 重播同一串樣本即可。
//! - 解碼失敗：Go 的 BGM 解不開就不換曲，音效庫解不開整批拒收；Rust 在 `fd2-game` 只檢查編碼位元組非空
//!   （KNOWN_DIFFERENCES `audio-decode-deferred`），外殼解不開時不播放並把 `playing` 設成 `false`。

use std::collections::{BTreeMap, BTreeSet};
use std::num::NonZero;
use std::sync::Arc;
use std::time::Duration;

use bevy::audio::{AddAudioSource, AudioPlayer, AudioSink, AudioSinkPlayback, Decodable, PlaybackSettings, Source};
use bevy::prelude::*;
use bevy::transform::TransformSystems;
use fd2_game::game::{AudioVoice, VoiceSlot};

use crate::go_ebiten_audio::{self, Stream};
use crate::go_io::{self, IoError, Read};

/// Go `audio.NewContext(44100)`。
pub const OUTPUT_RATE: u32 = 44100;
/// Go 的播放格式固定雙聲道。
pub const OUTPUT_CHANNELS: u16 = 2;

/// 解碼失敗的原因。
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum DecodeError {
    /// 不是 `OggS` 也不是 `RIFF`（Go 依呼叫端選解碼器，Rust 的外殼依開頭位元組選）。
    UnknownFormat,
    /// Go 解碼鏈的錯誤，文字照 Go（例如 `vorbis: decoding error`、`wav: invalid header`、`unexpected EOF`）。
    Go(String),
}

impl From<IoError> for DecodeError {
    fn from(e: IoError) -> DecodeError {
        DecodeError::Go(e.to_string())
    }
}

impl std::fmt::Display for DecodeError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            DecodeError::UnknownFormat => write!(f, "audio: unknown format"),
            DecodeError::Go(e) => f.write_str(e),
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

/// Ebiten 的解碼串流：`resample == true` 是 `DecodeWithSampleRate(44100, …)`（播放路徑），`false` 是
/// `DecodeWithoutResampling`（原始取樣率、已轉 16 位元雙聲道；比對工具用來分辨差異來自解碼還是重取樣）。
pub fn open_go_stream(bytes: Arc<[u8]>, resample: bool) -> Result<Stream, DecodeError> {
    let rate = i64::from(OUTPUT_RATE);
    let stream = match (sniff(&bytes).ok_or(DecodeError::UnknownFormat)?, resample) {
        (Format::Ogg, true) => go_ebiten_audio::vorbis::decode_with_sample_rate(rate, bytes)?,
        (Format::Ogg, false) => go_ebiten_audio::vorbis::decode_without_resampling(bytes)?,
        (Format::Wav, true) => go_ebiten_audio::wav::decode_with_sample_rate(rate, bytes)?,
        (Format::Wav, false) => go_ebiten_audio::wav::decode_without_resampling(bytes)?,
    };
    Ok(stream)
}

/// 每次向解碼鏈要的位元組數（不影響輸出，見 [`crate::go_ebiten_audio`] 模組文件）。
const PCM_CHUNK: usize = 16384;

/// 44.1 kHz、雙聲道交錯 `i16` 的串流（讀 Ebiten 串流的小端位元組）。解碼中途出錯時結束，錯誤留在 [`Self::error`]。
pub struct PcmStream {
    inner: Stream,
    buf: Vec<u8>,
    start: usize,
    end: usize,
    done: bool,
    error: Option<IoError>,
}

impl PcmStream {
    fn new(inner: Stream) -> PcmStream {
        PcmStream { inner, buf: vec![0; PCM_CHUNK], start: 0, end: 0, done: false, error: None }
    }

    /// Ebiten 串流宣告的總位元組數（Go `Length()`）。
    pub fn length_bytes(&self) -> i64 {
        self.inner.length()
    }

    /// 解碼中途的錯誤（Go `io.ReadAll` 回傳的錯誤）。
    pub fn error(&self) -> Option<&IoError> {
        self.error.as_ref()
    }
}

impl Iterator for PcmStream {
    type Item = i16;

    fn next(&mut self) -> Option<i16> {
        loop {
            if self.end - self.start >= 2 {
                let s = i16::from_le_bytes([self.buf[self.start], self.buf[self.start + 1]]);
                self.start += 2;
                return Some(s);
            }
            if self.done {
                return None;
            }
            let left = self.end - self.start;
            self.buf.copy_within(self.start..self.end, 0);
            self.start = 0;
            self.end = left;
            let (n, err) = self.inner.read(&mut self.buf[left..]);
            self.end += n;
            match err {
                None => {}
                Some(IoError::Eof) => self.done = true,
                Some(e) => {
                    self.error = Some(e);
                    self.done = true;
                }
            }
        }
    }
}

/// 以 Go 的播放路徑開啟串流（`DecodeWithSampleRate(44100, …)`）。
pub fn open_stream(bytes: Arc<[u8]>) -> Result<PcmStream, DecodeError> {
    Ok(PcmStream::new(open_go_stream(bytes, true)?))
}

/// 小端位元組轉 `i16`（多出的單一位元組捨去；Ebiten 串流的長度都是 4 的倍數）。
pub fn bytes_to_i16(bytes: &[u8]) -> Vec<i16> {
    bytes.as_chunks::<2>().0.iter().map(|c| i16::from_le_bytes(*c)).collect()
}

/// 整檔解碼：Go `io.ReadAll(DecodeWithSampleRate(44100, …))`。解碼中途出錯時與 Go 一樣回報錯誤（Go 的音效庫
/// 因此整批拒收）。
pub fn decode_pcm(bytes: &[u8]) -> Result<Vec<i16>, DecodeError> {
    let mut stream = open_go_stream(Arc::from(bytes), true)?;
    let (pcm, err) = go_io::read_all(&mut stream);
    if let Some(e) = err {
        return Err(e.into());
    }
    Ok(bytes_to_i16(&pcm))
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
