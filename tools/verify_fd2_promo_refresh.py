#!/usr/bin/env python3
"""核對重剪影片、已核准旁白與原速配樂；不把剪輯視為實機效能。"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import subprocess
import wave


def sha(path):
    digest = hashlib.sha256()
    with Path(path).open('rb') as stream:
        for block in iter(lambda: stream.read(1024 * 1024), b''):
            digest.update(block)
    return digest.hexdigest()


def load(path):
    return json.loads(Path(path).read_text())


def media_hash(path, kind, duration=None):
    command = ['ffmpeg', '-nostdin', '-v', 'error', '-threads', '2', '-i', str(path)]
    if kind == 'video':
        command += ['-map', '0:v:0', '-c:v', 'copy']
    else:
        command += ['-map', '0:a:0', '-c:a', 'pcm_f32le', '-ar', '48000', '-ac', '2']
    if duration is not None:
        command += ['-t', str(duration)]
    command += ['-f', 'hash', '-hash', 'sha256', '-']
    return subprocess.check_output(command, text=True, timeout=90).strip()


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ['promo', 'video-base', 'mix', 'reference-mix', 'music', 'reference-music', 'audio-plan']:
        parser.add_argument('--' + name, type=Path, required=True)
    parser.add_argument('--reference-promo', type=Path, help='重排時間時明示前版；逐段聲音保持，整軌位置依新計畫')
    args = parser.parse_args()
    assert Path('/.dockerenv').is_file() and os.getuid() != 0
    manifest = load(args.promo / 'EDIT-MANIFEST.json')
    validation = load(args.promo / 'VALIDATION.json')
    assert validation['status'] == 'PASS-TECHNICAL'
    assert validation['movie'] == manifest['movie']
    movie = args.promo / manifest['movie']['file']
    assert sha(movie) == manifest['movie']['sha256']
    base = load(args.video_base / 'EDIT-MANIFEST.json')
    base_movie = args.video_base / base['movie']['file']
    assert sha(base_movie) == base['movie']['sha256']
    assert manifest['scenes'] == base['scenes']
    fresh = [s['id'] for s in manifest['scenes'] if s['refresh'] != 'reused-unchanged-source']
    reused = [s['id'] for s in manifest['scenes'] if s['refresh'] == 'reused-unchanged-source']
    assert len(fresh) == 27 and len(reused) == 16
    for source in manifest['inputs']:
        assert sha(source['path']) == source['sha256'], source['path']
    video_hash = media_hash(movie, 'video')
    assert video_hash == media_hash(base_movie, 'video')
    narration = load(args.promo / 'NARRATION-MANIFEST.json')
    assert len(narration['segments']) == 36
    assert narration['speaker'] == 'zh-TW-HsiaoYuNeural'
    voices = []
    for index, row in enumerate(narration['segments']):
        assert row['generation_rate'] == '+8%' and row['generation_pitch'] == '+0Hz'
        assert row['playback_rate'] == 1
        path = args.mix / row['normalized_file']
        assert sha(path) == row['normalized_sha256']
        reference = args.reference_mix / f'voice-{index:02}.wav'
        actual = media_hash(path, 'audio')
        assert actual == media_hash(reference, 'audio'), row['id']
        voices.append(dict(id=row['id'], pcm_sha256=actual))
    narration_pcm = media_hash(args.mix / 'narration.wav', 'audio')
    if args.reference_promo:
        previous = load(args.reference_promo / 'EDIT-MANIFEST.json')
        old_voice = load(args.reference_promo / 'NARRATION-MANIFEST.json')
        assert all(r['normal_update_hz'] == 60 for r in manifest['pace_correction']['changed_scenes'].values())
        assert len(previous['scenes']) == len(manifest['scenes']) == 43
        unchanged = 0
        for i, (old, new) in enumerate(zip(previous['scenes'], manifest['scenes'])):
            if not new.get('logical_clock'):
                assert old == new
                assert previous['segments'][i]['sha256'] == manifest['segments'][i]['sha256']
                unchanged += 1
            else:
                points = new['logical_clock']
                for point in points:
                    assert abs(point['movie_seconds'] - (point['game_frame']-points[0]['game_frame'])/60) < 1e-7
        assert unchanged == 37
        with wave.open(str(args.mix/'narration.wav'), 'rb') as source:
            assert (source.getframerate(), source.getnchannels(), source.getsampwidth()) == (48000, 2, 2)
            whole = source.readframes(source.getnframes())
        end = 0
        for i, (row, old) in enumerate(zip(narration['segments'], old_voice['segments'])):
            assert row['text'] == old['text'] and row['id'] == old['id']
            with wave.open(str(args.reference_mix/f'voice-{i:02}.wav'), 'rb') as source:
                assert (source.getframerate(), source.getnchannels(), source.getsampwidth()) == (48000, 2, 2)
                samples = source.readframes(source.getnframes())
            start = round(row['start_seconds']*48000)*4
            assert not whole[end:start].strip(b'\x00')
            assert whole[start:start+len(samples)] == samples
            end = start+len(samples)
        assert not whole[end:].strip(b'\x00')
        # 新片只提前片尾淡出；淡出前PCM須與已驗正常配樂的同位置前綴相同。
        music_pcm = media_hash(args.music, 'audio', manifest['duration_seconds']-2)
        assert music_pcm == media_hash(args.reference_music, 'audio', manifest['duration_seconds']-2)
    else:
        assert narration_pcm == media_hash(args.reference_mix / 'narration.wav', 'audio')
        music_pcm = media_hash(args.music, 'audio')
        assert music_pcm == media_hash(args.reference_music, 'audio')
    audio = load(args.audio_plan)
    assert audio['music']['playback_rate'] == 1
    assert all(c['playback_rate'] == 1 for c in audio['cues'])
    assert audio['source_movie_sha256'] == base['movie']['sha256']
    heavy = [c for c in audio['cues'] if c.get('source_kind') == 'hd-original-b']
    assert {c['scene'] for c in heavy} == {'ares', 'gaia-hd'}
    report = dict(status='PASS-CURRENT-VISUALS-APPROVED-VOICE-NORMAL-MUSIC',
        movie=manifest['movie'], producer_sha256=sha(__file__),
        video_packets_identical_to_new_edit=True, video_packet_sha256=video_hash,
        fresh_scenes=fresh, reused_scenes=reused, input_hashes_verified=len(manifest['inputs']),
        voices=voices, narration_pcm_sha256=narration_pcm, music_pcm_sha256=music_pcm,
        reference_mix=str(args.reference_mix), reference_music=str(args.reference_music),
        music_playback_rate=1, sfx_playback_rate=1, heavy_b_scenes=['ares', 'gaia-hd'],
        human_full_listening_confirmed=False, windows_60fps_verified=False,
        limits=['配樂PCM與既有正常速度底音相同，36段旁白與已核准來源相同。',
                '音效依錄影姿勢／命令拍點剪輯對位，未宣稱硬體時序。',
                '畫面經剪輯調速；30fps交片不代表Windows即時幀率。',
                '來源含明示建構存檔；不宣稱全角色由自然招募取得。'])
    if args.reference_promo:
        report.update(pace_correction=manifest['pace_correction'], unchanged_scene_segments=37,
            narration_sample_alignment='PASS-36-REFERENCE-WAV-SLICES-AND-ZERO-GAPS',
            music_comparison='相同位置的正常配樂PCM前綴；最後兩秒另作淡出。')
        report['limits'][0] = '36段旁白PCM保持，整軌位置依新剪輯調整；配樂淡出前PCM前綴保持。'
    output = args.promo / 'REFRESH-PROOF.json'
    assert not output.exists()
    output.write_text(json.dumps(report, ensure_ascii=False, indent=2) + '\n')
    print(json.dumps(dict(status=report['status'], voices=len(voices), fresh_scenes=len(fresh))))


if __name__ == '__main__':
    main()
