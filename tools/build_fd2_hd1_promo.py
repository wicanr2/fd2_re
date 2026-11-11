#!/usr/bin/env python3
"""依明示剪輯表製作HD1推廣片；字幕準備與編碼可分別使用既有Docker工具映像。"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import shutil
import subprocess


def digest(path):
    h = hashlib.sha256()
    with Path(path).open('rb') as stream:
        for block in iter(lambda: stream.read(1024*1024), b''):
            h.update(block)
    return h.hexdigest()


def validate_audio_clock(scene):
    """混有配樂的擷取音軌不能隨影片一起調速。"""
    if scene['kind'] != 'game':
        return
    rate = scene.get('source_duration', scene['duration']) / scene['duration']
    if rate != 1 and not scene.get('audio_cues') and scene.get('source_audio_kind') != 'sfx_only':
        raise ValueError('retimed mixed audio prohibited: use normal-speed music and separate native SFX')


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
        elif scene['kind'] == 'concept':
            picture = Image.open(scene['background']).convert('RGBA')
            picture.thumbnail((1760,860), Image.Resampling.LANCZOS)
            canvas = Image.new('RGBA',(1920,1080),(8,18,30,255))
            canvas.alpha_composite(picture,((1920-picture.width)//2,(900-picture.height)//2))
            draw = ImageDraw.Draw(canvas)
            draw.rectangle((100,930,107,1040),fill=(228,187,105,255))
            line(draw,scene['title'],(130,916),50,(246,237,215,255))
            line(draw,scene['detail'],(130,995),30,(184,202,217,255))
        elif scene['kind'] == 'card':
            image = Image.open(scene['background']).convert('RGB').resize((1728,1080), Image.Resampling.LANCZOS)
            canvas = Image.new('RGB',(1920,1080),(8,18,30)); canvas.paste(image,(96,0))
            veil = Image.new('RGBA',canvas.size,(5,15,27,195)); canvas = Image.alpha_composite(canvas.convert('RGBA'),veil)
            draw = ImageDraw.Draw(canvas)
            draw.rectangle((0,860,1920,1080),fill=(8,18,30,255))
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
            canvas=Image.new('RGBA',(1920,1080),(0,0,0,0))
            if scene['kind'] == 'capture':
                canvas=Image.new('RGBA',(1920,1080),(8,18,30,255))
                picture=Image.open(scene['source']).convert('RGBA')
                picture.thumbnail((1536,960),Image.Resampling.LANCZOS)
                canvas.alpha_composite(picture,((1920-picture.width)//2,20+(960-picture.height)//2))
            draw=ImageDraw.Draw(canvas)
            draw.rectangle((0,984,1920,1080),fill=(8,18,30,255))
            draw.rectangle((100,1008,107,1046),fill=(228,187,105,255))
            line(draw,scene['caption'],(130,1000),34,(246,237,215,255))
        canvas.save(work/f'overlay-{i:02}.png')
    (work/'caption-bounds.json').write_text(json.dumps({'canvas':[1920,1080],'safe_bounds':[72,32,1848,1048],'labels':bounds},ensure_ascii=False,indent=2)+'\n')


def cue_audio(plan, scene, work, index):
    """Restore a missing recording tail from the admitted native sample cues."""
    duration=scene.get('source_duration',scene['duration'])
    target=work/f'cue-audio-{index:02}.wav'
    assert not target.exists()
    command=['ffmpeg','-nostdin','-hide_banner','-loglevel','warning','-threads','2',
             '-filter_complex_threads','1','-f','lavfi','-i','anullsrc=r=48000:cl=stereo']
    filters=[f'[0:a]atrim=duration={duration},asetpts=N/SR/TB[quiet]']
    labels=['[quiet]']
    for number,cue in enumerate(scene['audio_cues'],1):
        assert 0<=cue['seconds']<duration
        command+=['-i',cue['path']]
        delay=round(cue['seconds']*1000)
        filters.append(f'[{number}:a]aresample=48000,asetpts=N/SR/TB,adelay={delay}:all=1[cue{number}]')
        labels.append(f'[cue{number}]')
    filters.append(''.join(labels)+f'amix=inputs={len(labels)}:normalize=0,alimiter=limit=0.95,atrim=duration={duration},asetpts=N/SR/TB[a]')
    command+=['-filter_complex',';'.join(filters),'-map','[a]','-t',str(duration),
              '-c:a','pcm_s16le','-ar','48000','-ac','2',str(target)]
    subprocess.run(command,check=True,timeout=90)
    return target


def encode(plan, work, out, reuse_manifest=None, reuse_work=None):
    for scene in plan['scenes']:
        validate_audio_clock(scene)
    assert not out.exists(), '禁止覆寫已交付推廣片目錄'
    out.mkdir(parents=True)
    segments=[]
    previous = json.loads(reuse_manifest.read_text()) if reuse_manifest else None
    assert (reuse_manifest is None) == (reuse_work is None)
    for i,scene in enumerate(plan['scenes']):
        segment=work/f'segment-{i:02}.mp4';assert not segment.exists()
        if previous and i < len(previous['scenes']) and scene == previous['scenes'][i]:
            old = reuse_work / previous['segments'][i]['file']
            assert digest(old) == previous['segments'][i]['sha256'], old
            shutil.copyfile(old, segment)
            segments.append(segment.resolve())
            print(f'片段 {i+1}/{len(plan["scenes"])} 雜湊核對後沿用',flush=True)
            continue
        overlay=work/f'overlay-{i:02}.png';assert overlay.is_file()
        base=['ffmpeg','-nostdin','-hide_banner','-loglevel','warning','-threads','2',
              '-filter_threads','1','-filter_complex_threads','1']
        duration=scene['duration']
        if scene['kind'] in ('card','still','concept','capture'):
            command=base+['-loop','1','-framerate','30','-i',str(overlay),'-stream_loop','-1','-ss',str(scene.get('music_start',0)),
                          '-i',plan['card_music']]
            filters=f'[0:v]fps=30,format=yuv420p,fade=t=in:st=0:d=0.3[v];[1:a]atrim=duration={duration},asetpts=PTS-STARTPTS,afade=t=in:d=0.12,afade=t=out:st={duration-.12}:d=0.12[a]'
        else:
            source_duration=scene.get('source_duration',duration)
            rate=source_duration/duration
            assert .25<=rate<=32, ('unsupported editorial playback rate',rate)
            command=base+['-ss',str(scene['start']),'-i',scene['source']]
            if scene.get('audio_cues'):
                command+=['-i',str(cue_audio(plan,scene,work,i))]
                audio_index,overlay_index=1,2
            elif scene.get('audio'):
                command+=['-ss',str(max(0,scene['start']-scene.get('audio_offset',0))),'-i',scene['audio']]
                audio_index,overlay_index=1,2
            else:
                audio_index,overlay_index=0,1
            command+=['-loop','1','-framerate','30','-i',str(overlay)]
            bed_index = overlay_index + 1
            bed_gain = scene.get('audio_bed', 0.35 if scene.get('audio_cues') else None)
            if bed_gain:
                command+=['-stream_loop','-1','-i',plan['card_music']]
            tempos=[]
            remainder=rate
            while remainder>2:
                tempos.append('atempo=2');remainder/=2
            while remainder<.5:
                tempos.append('atempo=0.5');remainder/=.5
            tempos.append(f'atempo={remainder:.12g}')
            tempo=','.join(tempos)
            filters=f'[0:v]trim=duration={source_duration},setpts=(PTS-STARTPTS)/{rate:.12g},fps=30,scale=1536:960:flags=lanczos,pad=1920:1080:192:20:color=0x08121e,setsar=1[game];[game][{overlay_index}:v]overlay=shortest=1,format=yuv420p[v];[{audio_index}:a]atrim=duration={source_duration},asetpts=PTS-STARTPTS,{tempo},afade=t=in:d=0.08,afade=t=out:st={duration-.08}:d=0.08[a]'
            if bed_gain:
                filters=filters[:-3]+'[native];'+f'[{bed_index}:a]atrim=duration={duration},aresample=48000,asetpts=N/SR/TB,volume={bed_gain}[bed];[native][bed]amix=inputs=2:normalize=0,alimiter=limit=0.95,atrim=duration={duration},asetpts=N/SR/TB[a]'
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


def encode_video_only(plan, work, out, reuse_manifest=None, reuse_work=None):
    """只剪畫面；另由已登錄的原速配樂與旁白工具混音。"""
    assert not out.exists(), '禁止覆寫已交付推廣片目錄'
    out.mkdir(parents=True)
    segments=[]
    previous=json.loads(reuse_manifest.read_text()) if reuse_manifest else None
    assert (reuse_manifest is None)==(reuse_work is None)
    if previous:
        assert previous['status']=='VIDEO-ONLY-AWAITING-SEPARATE-AUDIO'
    for index,scene in enumerate(plan['scenes']):
        segment=work/f'segment-{index:02}.mp4'
        overlay=work/f'overlay-{index:02}.png'
        assert not segment.exists() and overlay.is_file()
        duration=scene['duration']
        frames=round(duration*30)
        assert frames>0 and abs(frames/30-duration)<1e-8
        if previous and index<len(previous['scenes']) and scene==previous['scenes'][index]:
            old=reuse_work/previous['segments'][index]['file']
            assert digest(old)==previous['segments'][index]['sha256']
            streams=json.loads(subprocess.check_output(['ffprobe','-v','error','-show_entries',
                'stream=codec_type','-of','json',str(old)],text=True,timeout=30))['streams']
            assert [s['codec_type'] for s in streams]==['video'], '純畫面沿用不可帶入舊音軌'
            shutil.copyfile(old,segment);segments.append(segment.resolve())
            print(f'純畫面片段 {index+1}/{len(plan["scenes"])} SHA核對後沿用',flush=True)
            continue
        command=['ffmpeg','-nostdin','-hide_banner','-loglevel','warning','-threads','2',
                 '-filter_threads','1','-filter_complex_threads','1']
        if scene['kind'] in ('card','still','concept','capture'):
            command+=['-loop','1','-framerate','30','-i',str(overlay)]
            filters='[0:v]fps=30,format=yuv420p,fade=t=in:st=0:d=0.3[v]'
        else:
            source_duration=scene.get('source_duration',duration)
            rate=max(scene.get('minimum_playback_rate',0),source_duration/duration)
            assert .25<=rate<=32, ('unsupported editorial playback rate',rate)
            command+=['-ss',str(scene['start']),'-i',scene['source'],
                      '-loop','1','-framerate','30','-i',str(overlay)]
            crop=''
            if 'window_rect' in scene:
                x,y,width,height=scene['window_rect']
                assert all(isinstance(v,int) for v in (x,y,width,height))
                assert x>=0 and y>=0 and width>0 and height>0
                # A native-art window may occupy only part of the X11 capture.
                probe=json.loads(subprocess.check_output(['ffprobe','-v','error',
                    '-select_streams','v:0','-show_entries','stream=width,height',
                    '-of','json',scene['source']],text=True,timeout=30))['streams'][0]
                assert x+width<=probe['width'] and y+height<=probe['height']
                assert width*5==height*8, '遊戲視窗比例必須與原始8:5一致'
                crop=f'crop={width}:{height}:{x}:{y},'
            hold=max(0,duration-source_duration/rate)
            pad=f'tpad=stop_mode=clone:stop_duration={hold:.12g},' if hold else ''
            filters=(f'[0:v]trim=duration={source_duration},setpts=(PTS-STARTPTS)/{rate:.12g},'
                     f'fps=30,{pad}{crop}scale=1536:960:flags=lanczos,'
                     'pad=1920:1080:192:20:color=0x08121e,setsar=1[game];'
                     '[game][1:v]overlay=shortest=1,format=yuv420p[v]')
        command+=['-filter_complex',filters,'-map','[v]','-an','-frames:v',str(frames),
                  '-c:v','libx264','-preset','veryfast','-crf','18','-threads','2',str(segment)]
        subprocess.run(command,check=True,timeout=max(90,int(duration*8)))
        segments.append(segment.resolve())
        print(f'純畫面片段 {index+1}/{len(plan["scenes"])} 完成',flush=True)
    listing=work/'concat.txt'
    listing.write_text(''.join("file '"+str(p)+"'\n" for p in segments))
    movie=out/f'FD2-{plan["version"]}-hd1-promo.mp4'
    subprocess.run(['ffmpeg','-nostdin','-hide_banner','-loglevel','warning','-threads','2',
                    '-f','concat','-safe','0','-i',str(listing),'-an','-c:v','copy',
                    '-t',str(plan['duration_seconds']),'-movflags','+faststart',str(movie)],
                   check=True,timeout=180)
    manifest=dict(plan)
    manifest.update(status='VIDEO-ONLY-AWAITING-SEPARATE-AUDIO',
        audio_policy='No captured mixed audio is consumed or retimed; rebuild normal-speed music, source SFX and selected narration separately.',
        movie={'file':movie.name,'bytes':movie.stat().st_size,'sha256':digest(movie)},
        encoder_script_sha256=digest(__file__),
        segments=[{'file':p.name,'sha256':digest(p)} for p in segments])
    (out/'EDIT-MANIFEST.json').write_text(json.dumps(manifest,ensure_ascii=False,indent=2)+'\n')
    print(movie)


def main():
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--plan',type=Path,required=True)
    parser.add_argument('--work',type=Path,required=True)
    parser.add_argument('--out',type=Path,required=True)
    parser.add_argument('--font',type=Path)
    parser.add_argument('--reuse-manifest',type=Path)
    parser.add_argument('--reuse-work',type=Path)
    parser.add_argument('--video-only',action='store_true',help='只剪視訊，背景音樂、音效與旁白另混；不讀取遊戲錄影的混合音軌。')
    group=parser.add_mutually_exclusive_group()
    group.add_argument('--prepare-only',action='store_true')
    group.add_argument('--encode-only',action='store_true')
    args=parser.parse_args()
    assert Path('/.dockerenv').is_file() and os.getuid()!=0
    plan=json.loads(args.plan.read_text())
    plan=plan.get('render_plan',plan)
    assert sum(s['duration'] for s in plan['scenes'])==plan['duration_seconds']
    for row in plan['inputs']:
        assert digest(row['path'])==row['sha256'],row['path']
    if not args.encode_only:
        assert args.font and args.font.is_file()
        prepare(plan,args.work,args.font)
    if not args.prepare_only:
        if args.video_only:
            encode_video_only(plan,args.work,args.out,args.reuse_manifest,args.reuse_work)
        else:
            encode(plan,args.work,args.out,args.reuse_manifest,args.reuse_work)


if __name__=='__main__':
    main()
