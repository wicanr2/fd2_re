#!/usr/bin/env python3
"""依已證實的斧擊／拳擊cue製作原創HD音色試聽，不接正式播放器。"""

import argparse
from array import array
import hashlib
import json
import math
import os
from pathlib import Path
import random
import subprocess
import sys
import wave


RATE = 48000
PEAK = 0.70
ROOT = Path(__file__).resolve().parents[1]


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def decode(path):
    raw = subprocess.check_output(['ffmpeg', '-v', 'error', '-i', str(path),
                                   '-ar', str(RATE), '-ac', '1', '-f', 'f32le', '-'])
    result = array('f')
    result.frombytes(raw)
    if sys.byteorder != 'little':
        result.byteswap()
    assert result and all(math.isfinite(x) for x in result)
    return result


def normalized(samples):
    peak = max(abs(x) for x in samples)
    assert peak > 0
    return array('f', (x*PEAK/peak for x in samples))


def stats(samples):
    peak = max(abs(x) for x in samples)
    rms = math.sqrt(sum(x*x for x in samples)/len(samples))
    return {'duration_seconds': len(samples)/RATE,
            'peak_dbfs': 20*math.log10(peak), 'rms_dbfs': 20*math.log10(rms),
            'clipped_samples': sum(abs(x) >= 1 for x in samples)}


def impact(count, heavy, mechanical):
    """只用固定種子噪聲、衰減振盪與濾波，不取樣或變速原版素材。"""
    rng = random.Random(20261106+10*heavy+mechanical)
    data = array('f')
    low = previous = 0.0
    for i in range(count):
        t = i/RATE
        noise = rng.uniform(-1, 1)
        low += (noise-low)*0.13
        high = noise-previous
        previous = noise
        body = math.sin(2*math.pi*((135 if mechanical else 175)*t-55*t*t))
        weight = (0.85 if heavy else 0.44)*body*math.exp(-t*(15 if heavy else 23))
        transient = 0.32*high*math.exp(-t*95)
        rough = (0.68*low+0.10*noise)*math.exp(-t*(22 if heavy else 33))
        metal = 0.0
        for n, frequency in enumerate([510, 830, 1490, 2470] if mechanical else [680, 1180, 1930, 3290]):
            metal += math.sin(2*math.pi*frequency*t)*(0.11/(n+1))*math.exp(-t*(14+8*n))
        # A short fade protects both boundaries. The main strike remains at the cue.
        fade_in = min(1, i/24)
        fade_out = min(1, (count-1-i)/max(1, int(RATE*.012)))
        data.append((weight+transient+rough+metal)*fade_in*fade_out)
    return normalized(data)


def write_wav(path, samples):
    pcm = array('h', (round(max(-1, min(1, x))*32767) for x in samples))
    if sys.byteorder != 'little':
        pcm.byteswap()
    with wave.open(str(path), 'wb') as stream:
        stream.setnchannels(1)
        stream.setsampwidth(2)
        stream.setframerate(RATE)
        stream.writeframes(pcm.tobytes())


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument('--original', type=Path, required=True)
    p.add_argument('--out', type=Path, required=True)
    args = p.parse_args()
    assert not args.out.exists() and args.out.parent.stat().st_uid == os.getuid()
    args.out.mkdir()
    rows, comparison = [], array('f')
    # nativePhysicalSoundBank uses 48+header4-1; raw_byte_5 is the cue.
    for instrument, resource, cue in [('axe', 49, 4), ('gaia-punch', 53, 2)]:
        source = args.original/'sfx'/f'FDOTHER_{resource:03d}'/f'sample_{cue:03d}.ogg'
        source_samples = decode(source)
        reference = normalized(source_samples)
        count = len(reference)
        assert RATE*.02 <= count <= RATE*5
        group = [('original-peak-matched', reference),
                 ('hd-crisp', impact(count, False, instrument == 'gaia-punch')),
                 ('hd-heavy', impact(count, True, instrument == 'gaia-punch'))]
        for style, samples in group:
            wav = args.out/f'{instrument}-{style}.wav'
            ogg = wav.with_suffix('.ogg')
            write_wav(wav, samples)
            subprocess.run(['ffmpeg', '-v', 'error', '-nostdin', '-i', str(wav),
                            '-c:a', 'libvorbis', '-q:a', '6', str(ogg)], check=True)
            decoded = decode(ogg)
            measured = stats(decoded)
            assert abs(len(decoded)-count)/RATE < .05
            assert measured['clipped_samples'] == 0 and measured['peak_dbfs'] < -.1
            rows.append({'instrument': instrument, 'style': style,
                         'source_resource': resource, 'source_cue': cue,
                         'source_file': str(source), 'source_sha256': sha(source),
                         'synthesis_uses_original_samples': style.startswith('original'),
                         'wav': str(wav.relative_to(ROOT)), 'wav_sha256': sha(wav),
                         'ogg': str(ogg.relative_to(ROOT)), 'ogg_sha256': sha(ogg),
                         'measurement': measured})
            # Each style sounds twice; the intentional gaps distinguish comparisons.
            for repeat in range(2):
                comparison.extend(samples)
                comparison.extend([0.0]*round(RATE*(.28 if repeat == 0 else .85)))
        comparison.extend([0.0]*round(RATE*.45))
    preview = args.out/'impact-comparison.wav'
    write_wav(preview, comparison)
    manifest = {'schema': 'fd2-hd1-impact-preview-v1', 'status': 'PASS-TECHNICAL-PREVIEW',
                'scope': '離線音色比較；原版參考只對齊峰值，HD音色為原創合成。不是正式接線或聽感通過。',
                'order': '斧擊：原版、A俐落、B厚重，各兩下；接蓋亞拳擊，同順序。',
                'sample_rate': RATE, 'channels': 1, 'vorbis_quality': 6,
                'target_peak': PEAK, 'seed': 20261106,
                'runtime_registered': False, 'human_listening_confirmed': False,
                'tool_sha256': sha(Path(__file__)), 'outputs': rows,
                'comparison': str(preview.relative_to(ROOT)), 'comparison_sha256': sha(preview)}
    (args.out/'preview-manifest.json').write_text(json.dumps(manifest, ensure_ascii=False, indent=2)+'\n')
    print(json.dumps({'status': manifest['status'], 'samples': len(rows),
                      'preview': str(preview), 'duration_seconds': len(comparison)/RATE}, ensure_ascii=False))


if __name__ == '__main__':
    main()
