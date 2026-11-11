#!/usr/bin/env python3
"""核對重剪影片、已核准旁白與原速配樂；不把剪輯視為實機效能。"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import subprocess


def sha(path):
    digest = hashlib.sha256()
    with Path(path).open('rb') as stream:
        for block in iter(lambda: stream.read(1024 * 1024), b''):
            digest.update(block)
    return digest.hexdigest()


def load(path):
    return json.loads(Path(path).read_text())


def media_hash(path, kind):
    command = ['ffmpeg', '-nostdin', '-v', 'error', '-threads', '2', '-i', str(path)]
    if kind == 'video':
        command += ['-map', '0:v:0', '-c:v', 'copy']
    else:
        command += ['-map', '0:a:0', '-c:a', 'pcm_f32le', '-ar', '48000', '-ac', '2']
    command += ['-f', 'hash', '-hash', 'sha256', '-']
    return subprocess.check_output(command, text=True, timeout=90).strip()


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ['promo', 'video-base', 'mix', 'reference-mix', 'music', 'reference-music', 'audio-plan']:
        parser.add_argument('--' + name, type=Path, required=True)
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
    output = args.promo / 'REFRESH-PROOF.json'
    assert not output.exists()
    output.write_text(json.dumps(report, ensure_ascii=False, indent=2) + '\n')
    print(json.dumps(dict(status=report['status'], voices=len(voices), fresh_scenes=len(fresh))))


if __name__ == '__main__':
    main()
