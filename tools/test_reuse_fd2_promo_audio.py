"""檢查等長視訊修訂不會讓已核准的旁白時窗錯位。"""
import copy
import sys
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
from reuse_fd2_promo_audio import validate_reuse


class AudioReuse(unittest.TestCase):
    def setUp(self):
        self.source = dict(movie=dict(sha256='approved'), duration_seconds=293.9,
                           scenes=[dict(id='intro', duration=8, caption='讓經典再現'),
                                   dict(id='body', duration=285.9)])
        self.base = copy.deepcopy(self.source)
        self.base['movie'] = dict(sha256='new-video')
        self.voice = dict(movie_sha256='approved', segments=[{} for _ in range(36)])
        self.proof = dict(status='PASS-CURRENT-VISUALS-APPROVED-VOICE-NORMAL-MUSIC',
                          movie=dict(sha256='approved'), music_playback_rate=1,
                          sfx_playback_rate=1, voices=[{} for _ in range(36)],
                          video_packets_identical_to_new_edit=True)

    def test_fractional_duration_preserves_audio_clock(self):
        self.assertEqual(validate_reuse(self.source, self.base, self.proof, self.voice), 293.9)

    def test_equal_total_with_shifted_scene_boundary_is_rejected(self):
        self.base['scenes'][0]['duration'] += 1
        self.base['scenes'][1]['duration'] -= 1
        with self.assertRaises(AssertionError):
            validate_reuse(self.source, self.base, self.proof, self.voice)

    def test_unbound_voice_or_accelerated_music_is_rejected(self):
        for key, value in [('movie', dict(sha256='other')), ('music_playback_rate', 1.08)]:
            proof = copy.deepcopy(self.proof)
            proof[key] = value
            with self.assertRaises(AssertionError):
                validate_reuse(self.source, self.base, proof, self.voice)

    def test_scene_reordering_is_rejected(self):
        self.base['scenes'].reverse()
        with self.assertRaises(AssertionError):
            validate_reuse(self.source, self.base, self.proof, self.voice)


if __name__ == '__main__':
    unittest.main()
