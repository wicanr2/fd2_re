//! Go 來源：`remake/cmd/fd2/audio_voice_test.go`、`separated_command_sound_test.go`。
//!
//! Go 的 `fakeSFXVoice` 是 `sfxVoice` 介面的測試替身；Rust 的 voice 就是 [`AudioVoice`] 資料，直接
//! 設定 `playing`。改寫 `FD2_ASSET_PACK` 指向暫存目錄的 fail-closed 測試，以及 `closeAudioPlayers`
//! （`r8`）與物理演出（D 組）的測試，列在 `PORTING.md`「A 組測試」。

use super::*;
use crate::test_support;

fn voice(playing: bool, tag: &str) -> SfxVoice {
    AudioVoice { source: tag.to_string(), playing, ..Default::default() }
}

/// Go：`audio_voice_test.go:32` `TestStepSFXVoicesRetainsPlayingAndClosesFinished`
/// （Go 的 nil 元素：Rust 的 `Vec<SfxVoice>` 沒有 nil，對應不移植）
#[test]
fn test_step_sfx_voices_retains_playing_and_closes_finished() {
    let mut g = Game { sfx_voices: vec![voice(true, "playing"), voice(false, "finished")], ..Default::default() };
    g.step_sfx_voices();
    assert_eq!(g.sfx_voices.len(), 1);
    assert_eq!(g.sfx_voices[0].source, "playing");
    assert!(!g.sfx_voices[0].closed);
}

/// Go：`audio_voice_test.go:47` `TestStepSFXVoicesEventuallyReleasesAllVoices`
#[test]
fn test_step_sfx_voices_eventually_releases_all_voices() {
    let mut g = Game { sfx_voices: vec![voice(true, "v")], ..Default::default() };
    g.step_sfx_voices();
    g.sfx_voices[0].playing = false;
    g.step_sfx_voices();
    assert!(g.sfx_voices.is_empty(), "voices={}, want released", g.sfx_voices.len());
}

/// Go：`audio_voice_test.go:71` `TestPlayRawRetainsRealAudioPlayer`
#[test]
fn test_play_raw_retains_real_audio_player() {
    if test_support::input("FD2_ASSET_PACK", "sfx/FDOTHER_031/resource.json").is_none() {
        return;
    }
    let bank = load_sfx().unwrap();
    let pcm = bank.get(&4).cloned().unwrap_or_default();
    assert!(!pcm.is_empty() && audio_ctx_ready(), "separated SFX fixture did not load into an audio context");
    let mut g = Game::default();
    g.play_raw(&pcm);
    assert_eq!(g.sfx_voices.len(), 1, "retained voices");
    assert_eq!(g.sfx_voices[0].encoded, pcm);
}

/// Go：`audio_voice_test.go:95` `TestFMAndMT32TracksDecodeAndSwitchAtRuntime`
#[test]
fn test_fm_and_mt32_tracks_decode_and_switch_at_runtime() {
    if !file_exists(&asset_path("assets/music_fm/FDMUS_001.ogg")) {
        test_support::skip("assets/music_fm/FDMUS_001.ogg（分離音樂 render）");
        return;
    }
    for profile in ["fm", "mt32"] {
        let mut g = Game { bgm_source: profile.to_string(), ..Default::default() };
        for track in ["FDMUS_004", "FDMUS_018"] {
            let path = g.music_render_path(track).unwrap();
            assert!(file_exists(&path), "{path}");
            g.play_bgm_count(track, 0);
            assert!(g.bgm.is_some() && g.bgm_cur == track, "{profile} {track}");
            assert!(g.bgm.as_ref().unwrap().looping);
            g.stop_bgm();
        }
    }
}

/// Go：`audio_voice_test.go:120` `TestUnknownCatalogTrackDoesNotMutateCurrentBGM`
#[test]
fn test_unknown_catalog_track_does_not_mutate_current_bgm() {
    let mut g = Game { bgm_source: String::from("fm"), bgm_cur: String::from("FDMUS_004"), ..Default::default() };
    g.play_bgm_count("FDMUS_999", 0);
    assert!(g.bgm.is_none() && g.bgm_cur == "FDMUS_004");
}

/// Go：`separated_command_sound_test.go:9` `TestSeparatedCommandSoundBanksDecodeWithoutOriginalArchive`
#[test]
fn test_separated_command_sound_banks_decode_without_original_archive() {
    if test_support::input("FD2_ASSET_PACK", "sfx/FDOTHER_080/resource.json").is_none() {
        return;
    }
    let banks = load_separated_command_sound_banks().unwrap();
    let want = [
        (50, 5),
        (53, 4),
        (77, 4),
        (78, 1),
        (80, 16),
        (82, 2),
        (83, 4),
        (84, 3),
        (85, 2),
        (86, 2),
        (87, 4),
        (88, 2),
        (90, 3),
        (91, 3),
        (92, 2),
        (93, 2),
        (94, 3),
        (95, 1),
    ];
    for (resource, count) in want {
        let bank = &banks[&resource];
        assert_eq!(bank.len(), count, "resource {resource}");
        for sub in 0..count as i64 {
            assert!(!bank[&sub].is_empty(), "resource {resource} subresource {sub} decoded empty");
        }
    }
}

/// Go：`separated_command_sound_test.go:36` `TestInstallSeparatedCommandSoundsUsesTitleBankSelectors`
#[test]
fn test_install_separated_command_sounds_uses_title_bank_selectors() {
    let banks = BTreeMap::from([
        (77, BTreeMap::from([(1, b"confirm".to_vec()), (2, b"move".to_vec())])),
        (78, BTreeMap::from([(0, b"ani1".to_vec())])),
    ]);
    let mut g = Game::default();
    g.install_separated_command_sounds(&banks);
    assert_eq!(g.sfx_title_move, b"move");
    assert_eq!(g.sfx_title_confirm, b"confirm");
    assert_eq!(g.sfx_title_ani1, b"ani1");
}

/// Go：`separated_command_sound_test.go:58` `TestSeparatedUISoundBankDecodesWithoutOriginalArchive`
#[test]
fn test_separated_ui_sound_bank_decodes_without_original_archive() {
    if test_support::input("FD2_ASSET_PACK", "sfx/FDOTHER_031/resource.json").is_none() {
        return;
    }
    let bank = load_sfx().unwrap();
    assert_eq!(bank.len(), 13, "UI sound samples");
    for sub in 0..13 {
        assert!(!bank[&sub].is_empty(), "UI sound subresource {sub} decoded empty");
    }
}

/// `stopTitleANI1Sound`／`stopNativePhysicalSound` 關閉並清空各自的 owner（Go
/// `TestNativePhysicalBodyCueKeepsGeneralVoicesIndependent` 的通道隔離部分）。
#[test]
fn stop_owned_voices_only_clears_their_own_channel() {
    let mut g = Game {
        sfx_voices: vec![voice(true, "ui")],
        title_ani1_voice: Some(voice(true, "title")),
        native_physical_voice: Some(voice(true, "physical")),
        ..Default::default()
    };
    g.stop_native_physical_sound();
    assert!(g.native_physical_voice.is_none());
    assert!(g.title_ani1_voice.is_some() && g.sfx_voices.len() == 1);
    g.stop_title_ani1_sound();
    assert!(g.title_ani1_voice.is_none() && g.sfx_voices.len() == 1);
}
