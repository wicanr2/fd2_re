//! Go 來源：`remake/cmd/fd2/audio.go`（347 行）。
//!
//! audio.go — BGM 播放(doc 12):只消費 music_catalog.json 驗證過的分離 OGG。
//! 忠實 play_bgm(0x25977)語意:同曲不重播;換曲=釋放舊曲再依 Miles loop count 播放。
//!
//! 骨架：Go 檔的 21 個函式中產生 15 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// Go：`audio.go:18`
///
/// Go 會在執行期改寫這個套件層變數，Rust 以 `RwLock` 包住。
pub(crate) static AUDIO_CTX: LazyLock<RwLock<Option<AudioContext>>> = LazyLock::new(|| RwLock::new(Default::default()));

/// Go：`audio.go:20`
/// Go 介面（`IsPlaying`、`Close`）只由 `*audio.Player` 實作；Rust 直接用 [`AudioVoice`]。
pub(crate) type SfxVoice = AudioVoice;

impl Game {
    /// playBGM 播指定曲(如 "FDMUS_008");同曲不重播;檔案缺失/解碼失敗靜默略過。
    /// FD2_MUTE=1 或截圖模式(headless 無音訊裝置)不播。
    ///
    /// Go：`audio.go:43`，`func (g *Game) playBGM(track string)`
    pub(crate) fn play_bgm(&mut self, track: &str) {
        todo!("audio.go:playBGM")
    }

    /// playBGMCount reproduces 0x25977(track, loopCount). Miles loop count zero is
    /// indefinite; one plays the decoded sequence once. The native same-track
    /// early return ignores a changed loop count, so bgmCur remains the sole gate.
    ///
    /// Go：`audio.go:50`，`func (g *Game) playBGMCount(track string, loopCount int)`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn play_bgm_count(&mut self, track: &str, loop_count: i64) {
        todo!("audio.go:playBGMCount")
    }

    /// stopBGM is the proven play_bgm(track=-1) operation used by original
    /// cutscene handlers.  Clearing bgmCur allows the following explicit BGM beat
    /// to restart the same track.
    ///
    /// Go：`audio.go:97`，`func (g *Game) stopBGM()`
    pub(crate) fn stop_bgm(&mut self) {
        todo!("audio.go:stopBGM")
    }
}

/// loadSFX 只由分離素材包載入完整FDOTHER#31 OGG bank。
///
/// Go：`audio.go:120`，`func loadSFX() (map[int][]byte, error)`
pub(crate) fn load_sfx() -> Result<BTreeMap<i64, Vec<u8>>> {
    todo!("audio.go:loadSFX")
}

/// loadWav 載單一 WAV 為 PCM bytes(戰鬥池等零散樣本用)。失敗回 nil。
///
/// Go：`audio.go:125`，`func loadWav(path string) []byte`
pub(crate) fn load_wav(path: &str) -> Vec<u8> {
    todo!("audio.go:loadWav")
}

/// loadSeparatedCommandSoundBanks 載入已閉合的標題與戰鬥指令音效庫。
/// 它只消費分離素材包；任一音效庫缺漏或 OGG 解碼失敗時整批拒絕。
///
/// Go：`audio.go:146`，`func loadSeparatedCommandSoundBanks() (map[int]map[int][]byte, error)`
pub(crate) fn load_separated_command_sound_banks() -> Result<BTreeMap<i64, BTreeMap<i64, Vec<u8>>>> {
    todo!("audio.go:loadSeparatedCommandSoundBanks")
}

/// Go：`audio.go:159`，`func decodeSeparatedSoundBank(resource int) (map[int][]byte, error)`
pub(crate) fn decode_separated_sound_bank(resource: i64) -> Result<BTreeMap<i64, Vec<u8>>> {
    todo!("audio.go:decodeSeparatedSoundBank")
}

impl Game {
    /// Go：`audio.go:182`，`func (g *Game) installSeparatedCommandSounds(banks map[int]map[int][]byte)`
    pub(crate) fn install_separated_command_sounds(&mut self, banks: &BTreeMap<i64, BTreeMap<i64, Vec<u8>>>) {
        todo!("audio.go:installSeparatedCommandSounds")
    }

    /// playRaw 直接播 PCM bytes(nil 安全)。
    ///
    /// Go：`audio.go:240`，`func (g *Game) playRaw(b []byte)`
    pub(crate) fn play_raw(&mut self, b: &mut [u8]) {
        todo!("audio.go:playRaw")
    }

    /// playSFX 播一個音效(疊播;原版雙 handle 0x26896/0x26945 可同時兩個,這裡不限)。
    ///
    /// Go：`audio.go:248`，`func (g *Game) playSFX(id int)`
    pub(crate) fn play_sfx(&mut self, id: i64) {
        todo!("audio.go:playSFX")
    }

    /// playNativePhysicalSound 保留25A96的同通道替換契約。
    /// 證據入口：fd2_physical_background_selection_20261004.json的physical_sound_spec。
    ///
    /// Go：`audio.go:270`，`func (g *Game) playNativePhysicalSound(b []byte)`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn play_native_physical_sound(&mut self, b: &mut [u8]) {
        todo!("audio.go:playNativePhysicalSound")
    }

    /// Go：`audio.go:281`，`func (g *Game) stopNativePhysicalSound()`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn stop_native_physical_sound(&mut self) {
        todo!("audio.go:stopNativePhysicalSound")
    }

    /// startTitleANI1Sound 建立 ANI #1 專用 voice；原版不把 #78 放入一般疊播池，
    /// 而是在第一幀啟動並於動畫自然結束或略過時明確停止。
    ///
    /// Go：`audio.go:290`，`func (g *Game) startTitleANI1Sound()`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn start_title_ani1_sound(&mut self) {
        todo!("audio.go:startTitleANI1Sound")
    }

    /// Go：`audio.go:301`，`func (g *Game) stopTitleANI1Sound()`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn stop_title_ani1_sound(&mut self) {
        todo!("audio.go:stopTitleANI1Sound")
    }

    /// stepSFXVoices 每幀回收已自然結束的短音效；仍在播放者保持獨立 voice，
    /// 讓原版不同 handle 的疊播不會被後一個 sample 截斷。
    ///
    /// Go：`audio.go:310`，`func (g *Game) stepSFXVoices()`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn step_sfx_voices(&mut self) {
        todo!("audio.go:stepSFXVoices")
    }
}
