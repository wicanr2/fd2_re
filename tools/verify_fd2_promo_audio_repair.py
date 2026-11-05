#!/usr/bin/env python3
"""驗證配樂維持1倍速、原旁白相同、原視訊封包相同；不冒稱人耳試聽。"""
import argparse
from array import array
import hashlib
import json
import math
import os
from pathlib import Path
import subprocess
from build_fd2_hd1_promo import validate_audio_clock


def sha(path):
    return hashlib.sha256(Path(path).read_bytes()).hexdigest()


decoded_pcm = {}


def pcm(path, start, duration, filters=None):
    # 完整解碼後用樣本索引裁切，排除OGG封包與命令列時間裁切的長度差異。
    cache_key = (str(path), filters)
    if cache_key not in decoded_pcm:
        command = ['ffmpeg', '-nostdin', '-v', 'error', '-threads', '2', '-i', str(path), '-vn']
        if filters:
            command += ['-af', filters]
        command += ['-ar', '48000', '-ac', '2', '-f', 'f32le', '-']
        values = array('f')
        values.frombytes(subprocess.check_output(command, timeout=60))
        decoded_pcm[cache_key] = values
    first, count = round(start*48000)*2, round(duration*48000)*2
    result = decoded_pcm[cache_key][first:first+count]
    assert len(result) == count
    return result


def similarity(reference, actual):
    assert len(reference) == len(actual) and len(actual) > 1000
    aa = sum(x*x for x in reference); bb = sum(x*x for x in actual)
    ab = sum(x*y for x, y in zip(reference, actual))
    assert aa > 1e-10 and bb > 1e-10
    return dict(correlation=ab/math.sqrt(aa*bb), gain=ab/aa, samples=len(actual))


def video_hash(path):
    return subprocess.check_output(['ffmpeg', '-nostdin', '-v', 'error', '-threads', '2', '-i', str(path),
        '-map', '0:v:0', '-c', 'copy', '-f', 'hash', '-hash', 'sha256', '-'], text=True, timeout=90).strip()


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ['source-promo', 'candidate', 'stems', 'plan', 'out']:
        parser.add_argument('--'+name, type=Path, required=True)
    a = parser.parse_args()
    assert Path('/.dockerenv').exists() and os.getuid() != 0
    assert not a.out.exists() and a.out.parent.stat().st_uid == os.getuid()
    before = json.loads((a.source_promo/'EDIT-MANIFEST.json').read_text())
    after = json.loads((a.candidate/'EDIT-MANIFEST.json').read_text())
    old_voice = json.loads((a.source_promo/'NARRATION-MANIFEST.json').read_text())
    new_voice = json.loads((a.candidate/'NARRATION-MANIFEST.json').read_text())
    plan = json.loads(a.plan.read_text())
    assert len(old_voice['segments']) == len(new_voice['segments']) == 36
    voice_fields = ['id', 'text', 'speaker', 'start_seconds', 'playback_rate', 'final_seconds',
                    'input_sha256', 'normalized_sha256']
    for old, new in zip(old_voice['segments'], new_voice['segments']):
        assert all(old[k] == new[k] for k in voice_fields), old['id']
    assert old_voice['narration_wav_sha256'] == new_voice['narration_wav_sha256']
    source_movie = a.source_promo/before['movie']['file']; movie = a.candidate/after['movie']['file']
    assert sha(source_movie) == before['movie']['sha256'] == plan['source_movie_sha256']
    assert sha(movie) == after['movie']['sha256'] == new_voice['movie_sha256']
    original_video = video_hash(source_movie)
    assert original_video == video_hash(movie)
    music_checks = []
    for t in [5, 15, 30, 50]:
        value = similarity(pcm(plan['music']['file'], t, .5), pcm(a.stems/'music-normal-speed.wav', t, .5))
        assert value['correlation'] > .999 and abs(value['gain']-plan['music']['gain']) < .002, (t, value)
        music_checks.append(dict(seconds=t, **value))
    # 一段真正加速過的波形必須被本檢查拒絕，避免只檢查計畫欄位。
    wrong = similarity(pcm(plan['music']['file'], 15, .5), pcm(plan['music']['file'], 15, .5, 'atempo=2'))
    assert wrong['correlation'] < .95, wrong
    rejected = False
    try:
        validate_audio_clock(dict(kind='game', duration=27, source_duration=242, audio='mixed-capture.wav'))
    except ValueError:
        rejected = True
    assert rejected
    validate_audio_clock(dict(kind='game', duration=27, source_duration=242, source_audio_kind='sfx_only'))
    effect_checks = []
    for scene in ['gaia-hd', 'yuni']:
        cue = next(x for x in plan['cues'] if x['scene']==scene)
        value = similarity(pcm(cue['file'], .02, .03), pcm(a.stems/'native-effects.wav', round(cue['at_seconds'],3)+.02, .03))
        assert value['correlation'] > .99 and abs(value['gain']-cue['gain']) < .02, (scene, value)
        effect_checks.append(dict(scene=scene, **value))
    report = dict(status='PASS-NORMAL-MUSIC-SAME-VOICE-VIDEO', source_movie=before['movie'], movie=after['movie'],
        video_packet_hash=original_video, video_packets_unchanged=True, voice_segments_unchanged=36,
        narration_wav_sha256=new_voice['narration_wav_sha256'], music_playback_rate=1,
        music_waveform_checks=music_checks, accelerated_audio_negative=wrong,
        mixed_capture_retiming_rejected=True,
        sfx_sample_waveform_checks=effect_checks, sfx_playback_rate=1,
        plan_sha256=sha(a.plan), verifier_sha256=sha(__file__), human_listening_confirmed=False,
        scope='Original-rate decoded music and dry SFX samples; identical voice WAV and video packets. Editorial cue timing remains approximate.')
    a.out.write_text(json.dumps(report, ensure_ascii=False, indent=2)+'\n')
    print(json.dumps(dict(status=report['status'], voice_segments=36, music_samples=len(music_checks), sfx_samples=len(effect_checks))))


if __name__ == '__main__':
    main()
