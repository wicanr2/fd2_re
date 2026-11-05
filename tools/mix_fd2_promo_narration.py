#!/usr/bin/env python3
"""將具來源收據的原創角色旁白混入已編製影片，保留遊戲音效並壓低旁白期間底音。"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import shutil
import subprocess


def sha(path):
    h = hashlib.sha256()
    with Path(path).open('rb') as stream:
        for block in iter(lambda: stream.read(1024 * 1024), b''):
            h.update(block)
    return h.hexdigest()


def run(command):
    subprocess.run(command, check=True, timeout=240)


def seconds(path):
    value = subprocess.check_output(['ffprobe', '-v', 'error', '-show_entries', 'format=duration',
                                     '-of', 'json', str(path)], text=True, timeout=30)
    return float(json.loads(value)['format']['duration'])


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ['base', 'voice-plan', 'work', 'out']:
        parser.add_argument('--' + name, type=Path, required=True)
    parser.add_argument('--voice-receipt', type=Path, action='append', required=True)
    args = parser.parse_args()
    assert Path('/.dockerenv').is_file() and os.getuid() != 0
    assert not args.work.exists() and not args.out.exists()
    assert args.work.parent.stat().st_uid == os.getuid()
    base = json.loads((args.base / 'EDIT-MANIFEST.json').read_text())
    movie = args.base / base['movie']['file']
    assert sha(movie) == base['movie']['sha256']
    assert args.out.name == 'promo' and args.out.parent.name == base['version']
    plan = json.loads(args.voice_plan.read_text())
    assert plan['kind'] == 'original-yuni-character-performance' and plan['original_actor_imitation'] is False
    assert plan['duration_seconds'] == base['duration_seconds']
    inputs, receipts, generation_sources = {}, [], []
    for path in args.voice_receipt:
        receipt = json.loads(path.read_text())
        assert receipt['kind'] == plan['kind'] and receipt['original_actor_imitation'] is False
        assert receipt['status'] in ['GENERATED-AWAITING-LISTENING', 'GENERATED-PARTIAL-PRESERVED']
        receipts.append(dict(file=str(path), sha256=sha(path)))
        generation_sources.append({key: value for key, value in receipt.items() if key != 'segments'})
        for row in receipt['segments']:
            assert row['id'] not in inputs and Path(row['file']).name == row['file']
            source = path.parent / row['file']
            assert sha(source) == row['sha256']
            inputs[row['id']] = (row, source)
    assert set(inputs) == {row['id'] for row in plan['segments']}
    args.work.mkdir()
    normalized, records = [], []
    common = ['ffmpeg', '-nostdin', '-hide_banner', '-loglevel', 'warning', '-threads', '2']
    previous_end = 0.0
    for index, wanted in enumerate(plan['segments']):
        actual, source = inputs[wanted['id']]
        assert all(actual[key] == value for key, value in wanted.items())
        assert wanted['speaker'] == 'Serena'
        trimmed = args.work / f'trimmed-{index:02}.wav'
        silence = 'silenceremove=start_periods=1:start_silence=0.04:start_threshold=-48dB'
        run(common + ['-i', str(source), '-af', silence + ',areverse,' + silence + ',areverse',
                      '-c:a', 'pcm_s16le', '-ar', '48000', '-ac', '2', str(trimmed)])
        duration = seconds(trimmed)
        speed = max(1.0, duration / wanted['window_seconds'])
        assert 0.3 < duration < 65 and speed <= 1.25, (wanted['id'], duration, wanted['window_seconds'])
        final_seconds = duration / speed
        start = wanted['start_seconds']
        assert start >= previous_end - 0.001 and start + final_seconds <= plan['duration_seconds']
        previous_end = start + final_seconds
        target = args.work / f'voice-{index:02}.wav'
        filters = (f'atempo={speed:.12g},loudnorm=I=-18:TP=-2:LRA=7,'
                   f'afade=t=in:d=0.02,afade=t=out:st={max(0, final_seconds - .04):.9f}:d=0.04')
        run(common + ['-i', str(trimmed), '-af', filters, '-c:a', 'pcm_s16le',
                      '-ar', '48000', '-ac', '2', str(target)])
        normalized.append(target)
        records.append(dict(wanted, input_file=str(source), input_sha256=sha(source),
                            normalized_file=target.name, normalized_sha256=sha(target),
                            trimmed_seconds=duration, playback_rate=speed, final_seconds=seconds(target)))
    duration = plan['duration_seconds']
    command = common + ['-filter_complex_threads', '1']
    filters, labels = [], []
    for index, (path, record) in enumerate(zip(normalized, records)):
        command += ['-i', str(path)]
        delay = round(record['start_seconds'] * 1000)
        filters.append(f'[{index}:a]adelay={delay}:all=1[v{index}]')
        labels.append(f'[v{index}]')
    filters.append(''.join(labels) + f'amix=inputs={len(labels)}:normalize=0,apad,atrim=duration={duration}[voice]')
    narration = args.work / 'narration.wav'
    run(command + ['-filter_complex', ';'.join(filters), '-map', '[voice]', '-c:a', 'pcm_s16le',
                   '-ar', '48000', '-ac', '2', str(narration)])
    args.out.mkdir(parents=True)
    output = args.out / base['movie']['file']
    filters = ('[1:a]asplit=2[side][voice];'
               '[0:a][side]sidechaincompress=threshold=0.04:ratio=4:attack=15:release=220[bed];'
               '[bed][voice]amix=inputs=2:normalize=0,loudnorm=I=-16:TP=-1.5:LRA=9[a]')
    run(common + ['-filter_complex_threads', '1', '-i', str(movie), '-i', str(narration),
                  '-filter_complex', filters, '-map', '0:v:0', '-map', '[a]', '-c:v', 'copy',
                  '-c:a', 'aac', '-b:a', '192k', '-ar', '48000', '-ac', '2', '-t', str(duration),
                  '-metadata', 'title=炎龍騎士團2：讓經典再現｜悠妮原創角色旁白',
                  '-movflags', '+faststart', str(output)])
    report = dict(status='MIXED-AWAITING-VALIDATION', kind=plan['kind'], speaker='Serena',
                  original_actor_imitation=False, voice_plan_sha256=sha(args.voice_plan),
                  receipts=receipts, generation_sources=generation_sources,
                  segments=records, narration_wav_sha256=sha(narration),
                  mixer_sha256=sha(__file__), human_listening_confirmed=False,
                  ducking=dict(threshold=0.04, ratio=4, attack_ms=15, release_ms=220),
                  movie_sha256=sha(output))
    (args.out / 'NARRATION-MANIFEST.json').write_text(json.dumps(report, ensure_ascii=False, indent=2) + '\n')
    base['base_movie'] = dict(base['movie'], path=str(movie))
    base['movie'] = dict(file=output.name, bytes=output.stat().st_size, sha256=sha(output))
    base['status'] = 'BUILT-AWAITING-VALIDATION'
    base['narration'] = dict(file='NARRATION-MANIFEST.json', sha256=sha(args.out / 'NARRATION-MANIFEST.json'))
    (args.out / 'EDIT-MANIFEST.json').write_text(json.dumps(base, ensure_ascii=False, indent=2) + '\n')
    shutil.copyfile(args.voice_plan, args.out / 'NARRATION-TEXT.json')
    print(json.dumps(dict(movie=str(output), segments=len(records), sha256=sha(output))))


if __name__ == '__main__':
    main()
