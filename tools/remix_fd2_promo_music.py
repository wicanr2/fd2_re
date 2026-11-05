#!/usr/bin/env python3
"""以正常速度配樂及具出處的音效重建推廣片底音，逐封包保留既有視訊。"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import subprocess


def sha(path):
    h = hashlib.sha256()
    with Path(path).open('rb') as stream:
        for block in iter(lambda: stream.read(1024 * 1024), b''):
            h.update(block)
    return h.hexdigest()


def run(command):
    return subprocess.run(command, check=True, capture_output=True, text=True, timeout=180)


def packet_hash(movie):
    return run(['ffmpeg', '-nostdin', '-v', 'error', '-threads', '2', '-i', str(movie),
                '-map', '0:v:0', '-c', 'copy', '-f', 'hash', '-hash', 'sha256', '-']).stdout.strip()


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ['source-promo', 'plan', 'work', 'out']:
        parser.add_argument('--' + name, type=Path, required=True)
    args = parser.parse_args()
    assert Path('/.dockerenv').exists() and os.getuid() != 0
    assert not args.work.exists() and not args.out.exists()
    assert args.work.parent.stat().st_uid == args.out.parent.stat().st_uid == os.getuid()
    manifest = json.loads((args.source_promo / 'EDIT-MANIFEST.json').read_text())
    plan = json.loads(args.plan.read_text())
    movie = args.source_promo / manifest['movie']['file']
    assert sha(movie) == manifest['movie']['sha256'] == plan['source_movie_sha256']
    assert plan['kind'] == 'normal-speed-music-separate-native-sfx'
    assert plan['music']['playback_rate'] == 1 and plan['duration_seconds'] == manifest['duration_seconds']
    for source in [plan['music'], *plan['cues']]:
        assert sha(source['file']) == source['sha256']
        assert source['playback_rate'] == 1
    assert all(0 <= x['at_seconds'] < plan['duration_seconds'] for x in plan['cues'])
    args.work.mkdir(); args.out.mkdir()
    common = ['ffmpeg', '-nostdin', '-hide_banner', '-loglevel', 'warning', '-threads', '2',
              '-filter_threads', '1', '-filter_complex_threads', '1']
    duration = plan['duration_seconds']
    music = args.work / 'music-normal-speed.wav'
    # 配樂只循環及調整音量，不消費任何鏡頭的source_duration／duration。
    music_filter = (f'aresample=48000,asetpts=N/SR/TB,atrim=duration={duration},'
                    f'volume={plan["music"]["gain"]},afade=t=in:d=0.4,'
                    f'afade=t=out:st={duration-2}:d=2')
    run(common + ['-stream_loop', '-1', '-i', plan['music']['file'], '-af', music_filter,
                  '-t', str(duration), '-c:a', 'pcm_f32le', '-ar', '48000', '-ac', '2', str(music)])
    effects = args.work / 'native-effects.wav'
    command, filters, labels = common[:], [], []
    for i, cue in enumerate(plan['cues']):
        command += ['-i', cue['file']]
        delay = round(cue['at_seconds'] * 1000)
        filters.append(f'[{i}:a]aresample=48000,asetpts=N/SR/TB,volume={cue["gain"]},'
                       f'afade=t=in:d=0.003,adelay={delay}:all=1[fx{i}]')
        labels.append(f'[fx{i}]')
    assert labels
    filters.append(''.join(labels) + f'amix=inputs={len(labels)}:normalize=0,apad,'
                   f'atrim=duration={duration},asetpts=N/SR/TB[fx]')
    effects_filter = ';'.join(filters)
    assert 'atempo' not in music_filter + effects_filter and 'asetrate' not in music_filter + effects_filter
    run(command + ['-filter_complex', effects_filter, '-map', '[fx]', '-t', str(duration),
                   '-c:a', 'pcm_f32le', '-ar', '48000', '-ac', '2', str(effects)])
    bed = args.work / 'music-and-effects.wav'
    run(common + ['-i', str(music), '-i', str(effects), '-filter_complex',
                  '[0:a][1:a]amix=inputs=2:normalize=0,alimiter=limit=0.95:level=false[a]',
                  '-map', '[a]', '-t', str(duration), '-c:a', 'pcm_f32le', '-ar', '48000', '-ac', '2', str(bed)])
    output = args.out / manifest['movie']['file']
    run(common + ['-i', str(movie), '-i', str(bed), '-map', '0:v:0', '-map', '1:a:0',
                  '-c:v', 'copy', '-c:a', 'aac', '-b:a', '192k', '-ar', '48000', '-ac', '2',
                  '-t', str(duration), '-movflags', '+faststart', str(output)])
    old_video, new_video = packet_hash(movie), packet_hash(output)
    assert old_video == new_video
    report = dict(status='PASS-NORMAL-SPEED-BED-VIDEO-UNCHANGED', plan_file=str(args.plan),
        plan_sha256=sha(args.plan), producer_sha256=sha(__file__), previous_movie=manifest['movie'],
        music_playback_rate=1, sfx_playback_rate=1, sfx_cues=len(plan['cues']),
        video_packet_hash=old_video, video_packets_unchanged=True,
        music_filter=music_filter, effects_filter=effects_filter,
        stems=[dict(file=str(p), sha256=sha(p)) for p in [music, effects, bed]],
        human_listening_confirmed=False, limits=plan['limits'])
    (args.out / 'AUDIO-REPAIR.json').write_text(json.dumps(report, ensure_ascii=False, indent=2)+'\n')
    manifest['previous_audio_movie'] = manifest['movie']
    manifest.pop('narration', None)
    manifest['movie'] = dict(file=output.name, bytes=output.stat().st_size, sha256=sha(output))
    manifest['status'] = 'AUDIO-REPAIRED-BASE-AWAITING-NARRATION'
    manifest['audio_repair'] = dict(file='AUDIO-REPAIR.json', sha256=sha(args.out/'AUDIO-REPAIR.json'))
    (args.out / 'EDIT-MANIFEST.json').write_text(json.dumps(manifest, ensure_ascii=False, indent=2)+'\n')
    print(json.dumps(dict(status=report['status'], movie=str(output), cues=len(plan['cues']))))


if __name__ == '__main__':
    main()
