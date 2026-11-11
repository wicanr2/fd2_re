#!/usr/bin/env python3
"""核對曉雨旁白來源、逐段實際音訊、時間位置、底音與視訊封包。"""
import argparse
from array import array
import hashlib
import json
import math
import os
from pathlib import Path
import wave

from verify_fd2_promo_audio_repair import pcm, similarity, video_hash


def sha(path):
    return hashlib.sha256(Path(path).read_bytes()).hexdigest()


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ['promo', 'previous', 'mix-work', 'stems', 'bed-receipt', 'out']:
        parser.add_argument('--'+name, type=Path, required=True)
    a = parser.parse_args()
    assert Path('/.dockerenv').exists() and os.getuid() != 0 and not a.out.exists()
    before = json.loads((a.previous/'EDIT-MANIFEST.json').read_text())
    after = json.loads((a.promo/'EDIT-MANIFEST.json').read_text())
    old_text = json.loads((a.previous/'NARRATION-TEXT.json').read_text())
    text = json.loads((a.promo/'NARRATION-TEXT.json').read_text())
    voice_path = a.promo/'NARRATION-MANIFEST.json'
    assert sha(voice_path) == after['narration']['sha256']
    voice = json.loads(voice_path.read_text())
    assert voice['speaker'] == 'zh-TW-HsiaoYuNeural' and voice['original_actor_imitation'] is False
    assert voice['voice_plan_sha256'] == sha(a.promo/'NARRATION-TEXT.json')
    assert len(text['segments']) == len(voice['segments']) == len(old_text['segments']) == 36
    for old, wanted, actual in zip(old_text['segments'], text['segments'], voice['segments']):
        assert all(actual[k] == v for k, v in wanted.items())
        assert all(old[k] == wanted[k] for k in ['id', 'scene_id', 'start_seconds', 'window_seconds'])
        if old['id'] == 'companions-intro':
            assert wanted['text'] == '還記得嗎？再一次，並肩作戰。'
        else:
            assert old['text'] == wanted['text']
        assert wanted['speaker'] == voice['speaker'] and actual['playback_rate'] == 1
        assert actual['generation_rate'] == '+8%' and actual['generation_pitch'] == '+0Hz'
    old_movie = a.previous/before['movie']['file']
    movie = a.promo/after['movie']['file']
    assert sha(old_movie) == before['movie']['sha256']
    assert sha(movie) == after['movie']['sha256'] == voice['movie_sha256']
    packet_hash = video_hash(old_movie)
    assert packet_hash == video_hash(movie)
    assert after['duration_seconds'] == before['duration_seconds'] == 313
    bed_receipt = json.loads(a.bed_receipt.read_text())
    assert bed_receipt['music_playback_rate'] == bed_receipt['sfx_playback_rate'] == 1
    for s in bed_receipt['stems']:
        assert sha(a.stems/Path(s['file']).name) == s['sha256']
    bed = a.stems/'music-and-effects.wav'
    assert sha(bed) == voice['independent_bed']['sha256']
    assert voice['independent_bed']['old_narration_used'] is False
    assert voice['independent_bed']['playback_rate'] == 1
    narration = a.mix_work/'narration.wav'
    assert sha(narration) == voice['narration_wav_sha256']
    checks = []
    with wave.open(str(narration)) as timeline:
        assert timeline.getframerate() == 48000 and timeline.getnchannels() == 2 and timeline.getsampwidth() == 2
        assert timeline.getnframes() == 313*48000
        end = 0
        for row in voice['segments']:
            assert sha(row['input_file']) == row['input_sha256']
            normalized = a.mix_work/row['normalized_file']
            assert sha(normalized) == row['normalized_sha256']
            with wave.open(str(normalized)) as w:
                assert w.getframerate() == 48000 and w.getnchannels() == 2 and w.getsampwidth() == 2
                samples = w.readframes(w.getnframes())
                duration = w.getnframes()/48000
            values = array('h', samples)
            peak = max(abs(x) for x in values)/32768
            rms = math.sqrt(sum(x*x for x in values)/len(values))/32768
            assert .001 < rms and .01 < peak < 1
            assert duration <= row['window_seconds'] and abs(duration-row['final_seconds']) < .000001
            start = round(row['start_seconds']*48000)
            assert start >= end
            timeline.setpos(end)
            assert not any(timeline.readframes(start-end))
            assert timeline.readframes(len(samples)//4) == samples, row['id']
            end = start+len(samples)//4
            checks.append(dict(id=row['id'], seconds=duration, peak_dbfs=20*math.log10(peak),
                               rms_dbfs=20*math.log10(rms), source_sha256=row['input_sha256'],
                               placement='PCM-EXACT', clipping=False))
        timeline.setpos(end)
        assert not any(timeline.readframes(timeline.getnframes()-end))
    music = []
    for t in [30, 65, 149, 282]:
        assert not any(r['start_seconds'] < t+.5 and r['start_seconds']+r['final_seconds'] > t
                       for r in voice['segments'])
        value = similarity(pcm(bed, t, .5), pcm(movie, t, .5))
        assert value['correlation'] > .98, (t, value)
        music.append(dict(seconds=t, **value))
    negative = similarity(pcm(bed, 30, .5), pcm(bed, 30, .5, 'atempo=1.08'))
    assert negative['correlation'] < .95
    report = dict(status='PASS-SOURCE-VOICE-TIMING-MUSIC-VIDEO', version=after['version'],
        movie=after['movie'], video_packets_unchanged=True, video_packet_hash=packet_hash,
        narration_segments=checks, text_segments_unchanged=35, approved_text_revision='companions-intro',
        all_placements_pcm_exact=True, generated_rate='+8%', additional_tempo_change=False,
        previous_bed_receipt_sha256=sha(a.bed_receipt), bed_sha256=sha(bed),
        music_playback_rate=1, sfx_playback_rate=1, final_music_waveform_checks=music,
        accelerated_music_negative=negative, verifier_sha256=sha(__file__),
        human_full_listening_confirmed=False)
    a.out.write_text(json.dumps(report, ensure_ascii=False, indent=2)+'\n')
    print(json.dumps(dict(status=report['status'], voice_segments=len(checks), video_packets_unchanged=True,
                         music_correlations=[x['correlation'] for x in music])))


if __name__ == '__main__':
    main()
