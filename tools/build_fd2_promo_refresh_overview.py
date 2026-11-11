#!/usr/bin/env python3
"""將新推廣片的實際畫面拼為README總覽；保留來源畫幅，不重繪內容。"""
import argparse
import hashlib
import json
import os
from pathlib import Path
from PIL import Image, ImageDraw, ImageFont


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def main():
    parser=argparse.ArgumentParser(description=__doc__)
    for name in ['opening','actions','key-frame','cards','font','out','receipt']:
        parser.add_argument('--'+name,type=Path,required=True)
    parser.add_argument('--deployment-frame',type=Path,help='從出場淡入完成後錄影抽出的完整畫面。')
    args=parser.parse_args()
    assert Path('/.dockerenv').is_file() and os.getuid()!=0
    assert not args.out.exists() and not args.receipt.exists()
    assert args.out.parent.stat().st_uid==args.receipt.parent.stat().st_uid==os.getuid()
    opening=json.loads((args.opening/'opening-promo-gui.json').read_text())
    assert opening['status']=='PASS-NORMAL-START-ROYAL-DEPLOYMENT'
    cards=json.loads((args.cards/'character-cards.json').read_text())
    assert all(c['portrait_style']=='HD' for p in cards['pages'] for c in p['characters'])
    frames=json.loads((args.actions/'frames.json').read_text())
    action=lambda name,index:args.actions/next(r['file'] for r in frames if r['name']==name and r['file']==f'{name}-{index}.png')
    panels=[('開始選單',args.opening/'start-menu.png'),
            ('皇城・原版',args.opening/'royal-original-original.png'),
            ('皇城・HD',args.opening/'royal-hd-hd.png'),
            ('第一關・伙伴就位',args.deployment_frame or args.opening/'first-battle-deployed.png'),
            ('亞雷斯・普通攻擊',action('ares',2)),
            ('蓋亞・重拳出擊',action('gaia',1)),
            ('悠妮・魔法',action('yuni',1)),
            ('蜜蒂・音速刃',action('middy',1)),
            ('鐵諾・熾炎刀',action('tino',3)),
            ('羅德曼・淒煌斬',action('rodman',2)),
            ('天空之鑰・動畫展示',args.key_frame),
            ('伙伴簡介・HD頭像',args.cards/'companions-04.png')]
    canvas=Image.new('RGB',(1920,1080),'#08121e');draw=ImageDraw.Draw(canvas)
    title=ImageFont.truetype(str(args.font),32,index=3)
    label=ImageFont.truetype(str(args.font),23,index=3)
    footer=ImageFont.truetype(str(args.font),18,index=3)
    draw.text((24,10),'讓經典再現　HD1 遊玩畫面',font=title,fill='#f6dfaa')
    inputs=[]
    for i,(text,path) in enumerate(panels):
        assert path.is_file()
        picture=Image.open(path).convert('RGB');original=picture.size
        picture.thumbnail((476,296),Image.Resampling.LANCZOS)
        x,y=(i%4)*480,56+(i//4)*328
        draw.rectangle((x+1,y+29,x+478,y+326),outline='#7e6b44',width=1)
        draw.text((x+12,y+1),text,font=label,fill='#e5cf9c')
        canvas.paste(picture,(x+(480-picture.width)//2,y+29+(298-picture.height)//2))
        inputs.append(dict(label=text,file=str(path),sha256=sha(path),source_size=list(original),full_frame_retained=True))
    draw.text((24,1048),'重製引擎畫面拼排；影片為剪輯。天空之鑰及人物卡依既有來源另列。',font=footer,fill='#bac7d4')
    canvas.save(args.out)
    args.receipt.write_text(json.dumps(dict(status='COMPILED-REVIEW-PENDING',file=str(args.out),
        sha256=sha(args.out),size=[1920,1080],inputs=inputs,font_sha256=sha(args.font),
        compiler_sha256=sha(Path(__file__))),ensure_ascii=False,indent=2)+'\n')
    print(args.out)


if __name__=='__main__':main()
