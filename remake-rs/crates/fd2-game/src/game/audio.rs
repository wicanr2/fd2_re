//! Go 來源：`remake/cmd/fd2/audio.go`（347 行）。
//!
//! audio.go — BGM 播放(doc 12):只消費 music_catalog.json 驗證過的分離 OGG。
//! 忠實 play_bgm(0x25977)語意:同曲不重播;換曲=釋放舊曲再依 Miles loop count 播放。
//!
//! 骨架：Go 檔的 21 個函式中產生 15 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。
//!
//! # R4 的切分（播放請求與狀態）
//!
//! `fd2-game` 只記錄「要播什麼」與播放器狀態（[`AudioVoice`]），不解碼、不出聲：
//!
//! - Go 在載入時把 OGG／WAV 解成 44.1 kHz PCM（`vorbis.DecodeWithSampleRate`、`wav.Decode…`）。Rust 的
//!   音效庫與 `sfx*` 欄位保存**編碼後**的位元組（分離素材包的 OGG、`loadWav` 讀到的 WAV），解碼在
//!   `fd2-app`（R5）。Go 只在解碼失敗或解出空 PCM 時拒收；Rust 在載入時只能檢查編碼位元組非空
//!   （KNOWN_DIFFERENCES `audio-decode-deferred`）。
//! - Go `audioCtx.NewPlayer(...)`／`p.Play()` → 建一個 `AudioVoice`（`playing = true`），外殼依
//!   `source`／`encoded`／`looping` 播放，播完把 `playing` 設回 `false`；`Close()` → `closed = true`
//!   後丟棄。無畫面驅動端可以在下一幀直接回報播完。
//! - `audioCtx` 只在 Go 建立的同一批時機（BGM 播放、WAV／音效庫解碼）設定，`playRaw` 等仍以它是否
//!   存在決定要不要發出請求，與 Go 相同。

use super::*;

/// Go：`audio.go:18`
///
/// Go 會在執行期改寫這個套件層變數，Rust 以 `RwLock` 包住。
pub(crate) static AUDIO_CTX: LazyLock<RwLock<Option<AudioContext>>> = LazyLock::new(|| RwLock::new(Default::default()));

/// Go：`audio.go:20`
/// Go 介面（`IsPlaying`、`Close`）只由 `*audio.Player` 實作；Rust 直接用 [`AudioVoice`]。
pub(crate) type SfxVoice = AudioVoice;

/// Go `if audioCtx == nil { audioCtx = audio.NewContext(44100) }`。
fn ensure_audio_ctx() {
    let mut ctx = AUDIO_CTX.write().unwrap();
    if ctx.is_none() {
        *ctx = Some(AudioContext { sample_rate: 44100 });
    }
}

fn audio_ctx_ready() -> bool {
    AUDIO_CTX.read().unwrap().is_some()
}

/// Go `os.Getenv("FD2_MUTE") != ""`。
fn audio_muted() -> bool {
    std::env::var_os("FD2_MUTE").is_some_and(|v| !v.is_empty())
}

/// Go `audio.NewPlayerFromBytes(audioCtx, b)` 後 `Play()`：一個播放中的音效請求。
fn new_pcm_voice(b: &[u8]) -> SfxVoice {
    AudioVoice {
        source: String::from("pcm"),
        playing: true,
        encoded: b.to_vec(),
        serial: next_voice_serial(),
        ..Default::default()
    }
}

impl Game {
    /// musicRenderPath 只從通過完整 catalog 驗證的 FM／MT-32 bundle 解析路徑。
    /// 缺 catalog、錯 hash、少任一 render 或未知 track 時整批失敗即關閉。
    ///
    /// Go：`audio.go:27`，`func (g *Game) musicRenderPath(track string) (string, error)`（`r8`：第一章重播
    /// 沒有走到真正播放；`playBGMCount` 需要它，隨本檔一起移植）。
    fn music_render_path(&mut self, track: &str) -> Result<String> {
        if self.music_catalog.is_none() {
            self.music_catalog = Some(musiccatalog::load(&asset_path("assets"))?);
        }
        Ok(self.music_catalog.as_ref().unwrap().resolve(&self.bgm_source, track)?)
    }

    /// playBGM 播指定曲(如 "FDMUS_008");同曲不重播;檔案缺失/解碼失敗靜默略過。
    /// FD2_MUTE=1 或截圖模式(headless 無音訊裝置)不播。
    ///
    /// Go：`audio.go:43`，`func (g *Game) playBGM(track string)`
    pub(crate) fn play_bgm(&mut self, track: &str) {
        self.play_bgm_count(track, 0);
    }

    /// playBGMCount reproduces 0x25977(track, loopCount). Miles loop count zero is
    /// indefinite; one plays the decoded sequence once. The native same-track
    /// early return ignores a changed loop count, so bgmCur remains the sole gate.
    ///
    /// Go：`audio.go:50`，`func (g *Game) playBGMCount(track string, loopCount int)`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn play_bgm_count(&mut self, track: &str, loop_count: i64) {
        if !self.current_native_system_options().music_enabled() {
            if !track.is_empty() {
                self.native_system_bgm_track = track.to_string();
            }
            return;
        }
        if track.is_empty() || track == self.bgm_cur || audio_muted() || !self.shot_path.is_empty() {
            return;
        }
        if !(0..=1).contains(&loop_count) {
            return;
        }
        let Ok(path) = self.music_render_path(track) else {
            return;
        };
        let Ok(raw) = std::fs::read(&path) else {
            return;
        };
        ensure_audio_ctx();
        // Go 在這裡以 vorbis 解碼，失敗就不播（R5）；Rust 只拒收空檔。
        if raw.is_empty() {
            return;
        }
        if let Some(old) = self.bgm.as_mut() {
            old.closed = true;
        }
        self.bgm = Some(AudioVoice {
            source: path,
            playing: true,
            closed: false,
            encoded: raw,
            looping: loop_count == 0,
            serial: next_voice_serial(),
        });
        self.bgm_cur = track.to_string();
    }

    /// stopBGM is the proven play_bgm(track=-1) operation used by original
    /// cutscene handlers.  Clearing bgmCur allows the following explicit BGM beat
    /// to restart the same track.
    ///
    /// Go：`audio.go:97`，`func (g *Game) stopBGM()`
    pub(crate) fn stop_bgm(&mut self) {
        if let Some(mut bgm) = self.bgm.take() {
            bgm.closed = true;
        }
        self.bgm_cur.clear();
        self.native_system_bgm_track.clear();
    }
}

/// loadSFX 只由分離素材包載入完整FDOTHER#31 OGG bank。
///
/// Go：`audio.go:120`，`func loadSFX() (map[int][]byte, error)`
pub(crate) fn load_sfx() -> Result<BTreeMap<i64, Vec<u8>>> {
    decode_separated_sound_bank(31)
}

/// loadWav 載單一 WAV 為 PCM bytes(戰鬥池等零散樣本用)。失敗回 nil。
///
/// Go：`audio.go:125`，`func loadWav(path string) []byte`
/// Rust 回傳 WAV 檔原始位元組（解碼在 R5，見模組文件）；空檔照 Go 解碼失敗回空。
pub(crate) fn load_wav(path: &str) -> Vec<u8> {
    let Ok(raw) = std::fs::read(asset_path(path)) else {
        return Vec::new();
    };
    ensure_audio_ctx();
    raw
}

/// loadSeparatedCommandSoundBanks 載入已閉合的標題與戰鬥指令音效庫。
/// 它只消費分離素材包；任一音效庫缺漏或 OGG 解碼失敗時整批拒絕。
///
/// Go：`audio.go:146`，`func loadSeparatedCommandSoundBanks() (map[int]map[int][]byte, error)`
pub(crate) fn load_separated_command_sound_banks() -> Result<BTreeMap<i64, BTreeMap<i64, Vec<u8>>>> {
    let resources = [49, 50, 53, 77, 78, 80, 82, 83, 84, 85, 86, 87, 88, 90, 91, 92, 93, 94, 95];
    let mut out = BTreeMap::new();
    for resource in resources {
        out.insert(resource, decode_separated_sound_bank(resource)?);
    }
    Ok(out)
}

/// Go：`audio.go:159`，`func decodeSeparatedSoundBank(resource int) (map[int][]byte, error)`
///
/// Rust 保存每個子資源的 OGG 位元組（見模組文件）。Go 的「decoded empty」錯誤在 Rust 改為檢查
/// 編碼位元組非空，訊息照 Go（`%w` 的內層錯誤是 nil，Go 印成 `%!w(<nil>)`）。
pub(crate) fn decode_separated_sound_bank(resource: i64) -> Result<BTreeMap<i64, Vec<u8>>> {
    let bank = fdother::load_separated_sound_bank(separated_asset_path("sfx"), resource)?;
    ensure_audio_ctx();
    let mut decoded = BTreeMap::new();
    for (subresource, encoded) in bank.encoded {
        if encoded.is_empty() {
            return Err(game_err!("sound bank {resource} subresource {subresource} decoded empty: %!w(<nil>)"));
        }
        decoded.insert(subresource, encoded);
    }
    Ok(decoded)
}

/// Go `banks[r][s]`：缺 map 項目得到 nil 切片。
fn bank_sample(banks: &BTreeMap<i64, BTreeMap<i64, Vec<u8>>>, resource: i64, sample: i64) -> Vec<u8> {
    banks.get(&resource).and_then(|bank| bank.get(&sample)).cloned().unwrap_or_default()
}

impl Game {
    /// Go：`audio.go:182`，`func (g *Game) installSeparatedCommandSounds(banks map[int]map[int][]byte)`
    pub(crate) fn install_separated_command_sounds(&mut self, banks: &BTreeMap<i64, BTreeMap<i64, Vec<u8>>>) {
        let s = |resource, sample| bank_sample(banks, resource, sample);
        self.separated_command_sfx = banks.clone();
        self.sfx_title_move = s(77, 2);
        self.sfx_title_confirm = s(77, 1);
        self.sfx_title_ani1 = s(78, 0);
        self.sfx_command24_actor = s(53, 3);
        self.sfx_command24_target = s(53, 2);
        self.sfx_command_modifier = s(80, 0);
        self.sfx_command9_player_palette = s(80, 0);
        self.sfx_command9_player_initial = s(80, 14);
        self.sfx_command9_player_repeat = s(80, 15);
        self.sfx_command1012_prelude = s(80, 2);
        self.sfx_command1012_main = s(80, 13);
        (self.sfx_command0_actor, self.sfx_command0_target) = (s(82, 0), s(82, 1));
        (self.sfx_command2_actor, self.sfx_command2_mode2) = (s(83, 0), s(83, 1));
        (self.sfx_command2_mode5, self.sfx_command2_mode6) = (s(83, 2), s(83, 3));
        (self.sfx_command3_actor, self.sfx_command3_sub1, self.sfx_command3_sub2) = (s(84, 0), s(84, 1), s(84, 2));
        (self.sfx_command4_actor, self.sfx_command4_target) = (s(85, 0), s(85, 1));
        (self.sfx_command5_actor, self.sfx_command5_target) = (s(86, 0), s(86, 1));
        (self.sfx_command6_actor, self.sfx_command6_target) = (s(87, 0), s(87, 1));
        (self.sfx_command6_front, self.sfx_command6_tail) = (s(87, 2), s(87, 3));
        (self.sfx_command7_actor, self.sfx_command7_target) = (s(88, 0), s(88, 1));
        (self.sfx_death, self.sfx_transition) = (s(88, 0), s(88, 1));
        (self.sfx_command8_actor, self.sfx_command8_sub1, self.sfx_command8_sub2) = (s(90, 0), s(90, 1), s(90, 2));
        self.sfx_spawn_intro = s(95, 0);
    }

    /// playRaw 直接播 PCM bytes(nil 安全)。
    ///
    /// Go：`audio.go:240`，`func (g *Game) playRaw(b []byte)`
    /// Rust 收 `&[u8]`（骨架是 `&mut [u8]`，呼叫端傳 `&mut Vec<u8>` 仍可編譯；簽名變更見 `PORTING.md`）。
    /// Go 的 `b == nil` 以空切片判斷（Go 的 nil 與空切片在這裡都不播：`playPCMVoice` 再擋 `len == 0`）。
    pub(crate) fn play_raw(&mut self, b: &[u8]) {
        if b.is_empty()
            || !audio_ctx_ready()
            || !self.current_native_system_options().sfx_enabled()
            || audio_muted()
            || !self.shot_path.is_empty()
        {
            return;
        }
        self.play_pcm_voice(b);
    }

    /// playSFX 播一個音效(疊播;原版雙 handle 0x26896/0x26945 可同時兩個,這裡不限)。
    ///
    /// Go：`audio.go:248`，`func (g *Game) playSFX(id int)`
    /// Go `g.sfx == nil` 以空 map 判斷（`loadSFX` 失敗時 Go 留 nil，成功時至少 13 項）。
    pub(crate) fn play_sfx(&mut self, id: i64) {
        if self.sfx.is_empty()
            || !self.current_native_system_options().sfx_enabled()
            || audio_muted()
            || !self.shot_path.is_empty()
        {
            return;
        }
        let Some(b) = self.sfx.get(&id).cloned() else {
            return;
        };
        if !audio_ctx_ready() {
            return;
        }
        self.play_pcm_voice(&b);
    }

    /// Go：`audio.go:260`，`func (g *Game) playPCMVoice(b []byte)`（`r8`，`playRaw`／`playSFX` 需要它，
    /// 隨本檔一起移植）。
    fn play_pcm_voice(&mut self, b: &[u8]) {
        if b.is_empty() || !audio_ctx_ready() {
            return;
        }
        self.sfx_voices.push(new_pcm_voice(b));
    }

    /// playNativePhysicalSound 保留25A96的同通道替換契約。
    /// 證據入口：fd2_physical_background_selection_20261004.json的physical_sound_spec。
    ///
    /// Go：`audio.go:270`，`func (g *Game) playNativePhysicalSound(b []byte)`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    /// Rust 收 `&[u8]`（簽名變更見 `PORTING.md`）。
    pub(crate) fn play_native_physical_sound(&mut self, b: &[u8]) {
        if b.is_empty()
            || !audio_ctx_ready()
            || !self.current_native_system_options().sfx_enabled()
            || audio_muted()
            || !self.shot_path.is_empty()
        {
            return;
        }
        self.stop_native_physical_sound();
        self.native_physical_voice = Some(new_pcm_voice(b));
    }

    /// Go：`audio.go:281`，`func (g *Game) stopNativePhysicalSound()`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn stop_native_physical_sound(&mut self) {
        if let Some(mut voice) = self.native_physical_voice.take() {
            voice.closed = true;
        }
    }

    /// startTitleANI1Sound 建立 ANI #1 專用 voice；原版不把 #78 放入一般疊播池，
    /// 而是在第一幀啟動並於動畫自然結束或略過時明確停止。
    ///
    /// Go：`audio.go:290`，`func (g *Game) startTitleANI1Sound()`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn start_title_ani1_sound(&mut self) {
        self.stop_title_ani1_sound();
        if self.sfx_title_ani1.is_empty()
            || !audio_ctx_ready()
            || !self.current_native_system_options().sfx_enabled()
            || audio_muted()
            || !self.shot_path.is_empty()
        {
            return;
        }
        self.title_ani1_voice = Some(new_pcm_voice(&self.sfx_title_ani1));
    }

    /// Go：`audio.go:301`，`func (g *Game) stopTitleANI1Sound()`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn stop_title_ani1_sound(&mut self) {
        if let Some(mut voice) = self.title_ani1_voice.take() {
            voice.closed = true;
        }
    }

    /// stepSFXVoices 每幀回收已自然結束的短音效；仍在播放者保持獨立 voice，
    /// 讓原版不同 handle 的疊播不會被後一個 sample 截斷。
    ///
    /// Go：`audio.go:310`，`func (g *Game) stepSFXVoices()`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn step_sfx_voices(&mut self) {
        if self.sfx_voices.is_empty() {
            return;
        }
        // Go 對播完的 voice 呼叫 Close() 後丟掉；Rust 直接丟棄（外殼依 voice 是否還在佇列判斷）。
        self.sfx_voices.retain(|voice| voice.is_playing());
    }

    /// R5 階段 4 r5o 新增（外殼介面，沒有對應的 Go 函式）：目前存在的全部播放請求。
    ///
    /// Go 的 `*audio.Player` 自己出聲並回報 `IsPlaying()`；Rust 由 `fd2-app` 每幀呼叫這裡，依
    /// [`AudioVoice::serial`] 開始或停止播放，播完把 `playing` 設回 `false`。不在清單裡的序號代表
    /// Go 已 `Close()`（`stopBGM`、換曲、`stepSFXVoices` 回收等），外殼要停掉對應的播放。
    /// 順序固定：BGM、標題 ANI #1、物理演出、疊播音效（依加入順序）。
    pub fn audio_voices_mut(&mut self) -> Vec<(VoiceSlot, &mut AudioVoice)> {
        let mut out = Vec::with_capacity(3 + self.sfx_voices.len());
        if let Some(voice) = self.bgm.as_mut() {
            out.push((VoiceSlot::Bgm, voice));
        }
        if let Some(voice) = self.title_ani1_voice.as_mut() {
            out.push((VoiceSlot::TitleAni1, voice));
        }
        if let Some(voice) = self.native_physical_voice.as_mut() {
            out.push((VoiceSlot::NativePhysical, voice));
        }
        out.extend(self.sfx_voices.iter_mut().map(|voice| (VoiceSlot::Sfx, voice)));
        out
    }
}

#[cfg(test)]
#[path = "audio/audio_test.rs"]
mod audio_test;
