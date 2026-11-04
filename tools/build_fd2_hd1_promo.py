#!/usr/bin/env python3
"""依明示剪輯表製作HD1推廣片；字幕準備與編碼可分別使用既有Docker工具映像。"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import subprocess


def digest(path):
    h = hashlib.sha256()
    with Path(path).open('rb') as stream:
        for block in iter(lambda: stream.read(1024*1024), b''):
            h.update(block)
    return h.hexdigest()


def prepare(plan, work, font_path):
    from PIL import Image, ImageDraw, ImageFont
    work.mkdir(parents=True, exist_ok=True)
    font = lambda n: ImageFont.truetype(str(font_path), n, index=3)
    bounds = []
    def line(draw, caption, at, size, fill):
        box = draw.textbbox(at, caption, font=font(size))
        assert 72 <= box[0] and box[2] <= 1848 and 32 <= box[1] and box[3] <= 1048, (caption,box)
        bounds.append({'text': caption, 'box': list(box)})
        draw.text(at, caption, font=font(size), fill=fill)
    for i,scene in enumerate(plan['scenes']):
        if scene['kind'] == 'still':
            canvas = Image.open(scene['source']).convert('RGBA')
            assert canvas.size == (1920,1080)
        elif scene['kind'] == 'card':
            image = Image.open(scene['background']).convert('RGB').resize((1728,1080), Image.Resampling.LANCZOS)
            canvas = Image.new('RGB',(1920,1080),(8,18,30)); canvas.paste(image,(96,0))
            veil = Image.new('RGBA',canvas.size,(5,15,27,195)); canvas = Image.alpha_composite(canvas.convert('RGBA'),veil)
            draw = ImageDraw.Draw(canvas)
            draw.rectangle((0,900,1920,1080),fill=(8,18,30,255))
            draw.rectangle((116,330,124,780), fill=(220,183,102,255))
            if scene.get('lines'):
                for row in scene['lines']:
                    line(draw,row['text'],tuple(row['at']),row['size'],row.get('color','#eee9db'))
            else:
                line(draw,scene['title'],(160,350),100,(247,239,216,255))
                line(draw,scene['subtitle'],(160,510),82,(237,196,112,255))
                line(draw,scene['detail'],(166,670),40,(227,233,237,255))
                line(draw,'Windows　macOS　Linux',(166,750),34,(184,202,217,255))
            line(draw,'github.com/wicanr2/fd2_re',(166,970),30,(184,202,217,255))
        else:
            canvas=Image.new('RGBA',(1920,1080),(0,0,0,0));draw=ImageDraw.Draw(canvas)
            draw.rectangle((0,984,1920,1080),fill=(8,18,30,255))
            draw.rectangle((100,1008,107,1046),fill=(228,187,105,255))
            line(draw,scene['caption'],(130,1000),34,(246,237,215,255))
        canvas.save(work/f'overlay-{i:02}.png')
    (work/'caption-bounds.json').write_text(json.dumps({'canvas':[1920,1080],'safe_bounds':[72,32,1848,1048],'labels':bounds},ensure_ascii=False,indent=2)+'\n')


def encode(plan, work, out):
    assert not out.exists(), '禁止覆寫已交付推廣片目錄'
    out.mkdir(parents=True)
    segments=[]
    for i,scene in enumerate(plan['scenes']):
        segment=work/f'segment-{i:02}.mp4';assert not segment.exists()
        overlay=work/f'overlay-{i:02}.png';assert overlay.is_file()
        base=['ffmpeg','-nostdin','-hide_banner','-loglevel','warning','-threads','2',
              '-filter_threads','1','-filter_complex_threads','1']
        duration=scene['duration']
        if scene['kind'] in ('card','still'):
            command=base+['-loop','1','-framerate','30','-i',str(overlay),'-stream_loop','-1','-ss',str(scene.get('music_start',0)),
                          '-i',plan['card_music']]
            filters=f'[0:v]fps=30,format=yuv420p,fade=t=in:st=0:d=0.3[v];[1:a]atrim=duration={duration},asetpts=PTS-STARTPTS,afade=t=in:d=0.12,afade=t=out:st={duration-.12}:d=0.12[a]'
        else:
            source_duration=scene.get('source_duration',duration)
            rate=source_duration/duration
            assert .25<=rate<=32, ('unsupported editorial playback rate',rate)
            command=base+['-ss',str(scene['start']),'-i',scene['source']]
            if scene.get('audio'):
                command+=['-ss',str(max(0,scene['start']-scene.get('audio_offset',0))),'-i',scene['audio']]
                audio_index,overlay_index=1,2
            else:
                audio_index,overlay_index=0,1
            command+=['-loop','1','-framerate','30','-i',str(overlay)]
            tempos=[]
            remainder=rate
            while remainder>2:
                tempos.append('atempo=2');remainder/=2
            while remainder<.5:
                tempos.append('atempo=0.5');remainder/=.5
            tempos.append(f'atempo={remainder:.12g}')
            tempo=','.join(tempos)
            filters=f'[0:v]trim=duration={source_duration},setpts=(PTS-STARTPTS)/{rate:.12g},fps=30,scale=1536:960:flags=lanczos,pad=1920:1080:192:20:color=0x08121e,setsar=1[game];[game][{overlay_index}:v]overlay=shortest=1,format=yuv420p[v];[{audio_index}:a]atrim=duration={source_duration},asetpts=PTS-STARTPTS,{tempo},afade=t=in:d=0.08,afade=t=out:st={duration-.08}:d=0.08[a]'
        command+=['-filter_complex',filters,'-map','[v]','-map','[a]','-t',str(duration),
                  '-c:v','libx264','-preset','veryfast','-crf','18','-threads','2',
                  '-c:a','aac','-b:a','192k','-ar','48000','-ac','2','-movflags','+faststart',str(segment)]
        subprocess.run(command,check=True,timeout=max(90,int(duration*8)))
        segments.append(segment.resolve())
        print(f'片段 {i+1}/{len(plan["scenes"])} 完成',flush=True)
    listing=work/'concat.txt';listing.write_text(''.join("file '"+str(p)+"'\n" for p in segments))
    movie=out/f'FD2-{plan["version"]}-hd1-promo.mp4'
    subprocess.run(['ffmpeg','-nostdin','-hide_banner','-loglevel','warning','-threads','2',
                    '-f','concat','-safe','0','-i',str(listing),'-c:v','copy',
                    '-af','loudnorm=I=-18:TP=-1.5:LRA=9','-c:a','aac','-b:a','192k','-ar','48000',
                    '-t',str(plan['duration_seconds']),'-metadata','title=炎龍騎士團2：讓經典再現',
                    '-movflags','+faststart',str(movie)],check=True,timeout=180)
    manifest=dict(plan)
    manifest.update(status='BUILT-AWAITING-VALIDATION',movie={'file':movie.name,'bytes':movie.stat().st_size,'sha256':digest(movie)},
                    encoder_script_sha256=digest(__file__),segments=[{'file':p.name,'sha256':digest(p)} for p in segments])
    (out/'EDIT-MANIFEST.json').write_text(json.dumps(manifest,ensure_ascii=False,indent=2)+'\n')
    print(movie)


def main():
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--plan',type=Path,required=True)
    parser.add_argument('--work',type=Path,required=True)
    parser.add_argument('--out',type=Path,required=True)
    parser.add_argument('--font',type=Path)
    group=parser.add_mutually_exclusive_group()
    group.add_argument('--prepare-only',action='store_true')
    group.add_argument('--encode-only',action='store_true')
    args=parser.parse_args()
    assert Path('/.dockerenv').is_file() and os.getuid()!=0
    plan=json.loads(args.plan.read_text())
    assert sum(s['duration'] for s in plan['scenes'])==plan['duration_seconds']
    for row in plan['inputs']:
        assert digest(row['path'])==row['sha256'],row['path']
    if not args.encode_only:
        assert args.font and args.font.is_file()
        prepare(plan,args.work,args.font)
    if not args.prepare_only:
        encode(plan,args.work,args.out)


if __name__=='__main__':
    main()
