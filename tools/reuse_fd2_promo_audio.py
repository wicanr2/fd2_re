#!/usr/bin/env python3
"""將已驗證的配音音軌無損接到修正後畫面，分別核對兩條串流。"""
import argparse
import json
import os
from pathlib import Path
import re
import shutil
import subprocess

from remix_fd2_promo_music import sha


def stream_hash(movie, stream):
    return subprocess.check_output([
        'ffmpeg', '-nostdin', '-v', 'error', '-threads', '2', '-i', str(movie),
        '-map', '0:'+stream+':0', '-c', 'copy', '-f', 'hash', '-hash', 'sha256', '-'
    ], text=True, timeout=60).strip()


def read(path):
    return json.loads(path.read_text())


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ['source-promo', 'video-base', 'audio-proof', 'out']:
        parser.add_argument('--'+name, type=Path, required=True)
    a = parser.parse_args()
    assert Path('/.dockerenv').exists() and os.getuid() != 0
    assert not a.out.exists() and a.out.parent.stat().st_uid == os.getuid()
    source = read(a.source_promo/'EDIT-MANIFEST.json')
    base = read(a.video_base/'EDIT-MANIFEST.json')
    proof = read(a.audio_proof)
    voice = read(a.source_promo/'NARRATION-MANIFEST.json')
    source_movie = a.source_promo/source['movie']['file']
    video = a.video_base/base['movie']['file']
    assert source['movie']['sha256'] == sha(source_movie) == proof['movie']['sha256']
    assert base['movie']['sha256'] == sha(video)
    assert voice['movie_sha256'] == source['movie']['sha256']
    assert proof['status'] == 'PASS-NORMAL-MUSIC-SAME-VOICE-VIDEO'
    assert proof['voice_segments_unchanged'] == len(voice['segments']) == 36
    assert proof['music_playback_rate'] == proof['sfx_playback_rate'] == 1
    assert proof['mixed_capture_retiming_rejected'] is True
    assert source['duration_seconds'] == base['duration_seconds'] == 313
    assert re.fullmatch(r'v\.[0-9]+\.[0-9]+\.[0-9]+-[0-9]{8}', base['version'])
    assert a.out.name == 'promo' and a.out.parent.name == base['version']
    a.out.mkdir()
    movie = a.out/base['movie']['file']
    subprocess.run([
        'ffmpeg', '-nostdin', '-v', 'warning', '-threads', '2',
        '-i', str(video), '-i', str(source_movie), '-map', '0:v:0', '-map', '1:a:0',
        '-c', 'copy', '-t', '313', '-metadata', 'title=炎龍騎士團2：讓經典再現｜悠妮原創角色旁白',
        '-movflags', '+faststart', str(movie)
    ], check=True, timeout=90)
    audio_hash = stream_hash(source_movie, 'a')
    video_hash = stream_hash(video, 'v')
    assert stream_hash(movie, 'a') == audio_hash
    assert stream_hash(movie, 'v') == video_hash
    result = dict(status='PASS-AUDIO-AND-VIDEO-STREAM-COPY',
                  source_movie=dict(source['movie'], path=str(source_movie)),
                  source_video=dict(base['movie'], path=str(video)),
                  audio_proof_sha256=sha(a.audio_proof),
                  audio_packet_hash=audio_hash, audio_packets_unchanged=True,
                  video_packet_hash=video_hash, video_packets_unchanged=True,
                  producer_sha256=sha(__file__))
    receipt = a.out/'AUDIO-REUSE.json'
    receipt.write_text(json.dumps(result, ensure_ascii=False, indent=2)+'\n')
    voice['source_voiced_movie'] = dict(source['movie'], path=str(source_movie))
    voice['source_narration_manifest_sha256'] = sha(a.source_promo/'NARRATION-MANIFEST.json')
    voice['movie_sha256'] = sha(movie)
    narration = a.out/'NARRATION-MANIFEST.json'
    narration.write_text(json.dumps(voice, ensure_ascii=False, indent=2)+'\n')
    shutil.copyfile(a.source_promo/'NARRATION-TEXT.json', a.out/'NARRATION-TEXT.json')
    base['base_movie'] = dict(base['movie'], path=str(video))
    base['movie'] = dict(file=movie.name, bytes=movie.stat().st_size, sha256=sha(movie))
    base['status'] = 'BUILT-AWAITING-VALIDATION'
    base['delivery_kind'] = 'promo-only-gaia-scene-repair'
    base['audio_repair'] = dict(file=receipt.name, sha256=sha(receipt))
    base['narration'] = dict(file=narration.name, sha256=sha(narration))
    (a.out/'EDIT-MANIFEST.json').write_text(json.dumps(base, ensure_ascii=False, indent=2)+'\n')
    print(json.dumps(result, ensure_ascii=False))


if __name__ == '__main__':
    main()
