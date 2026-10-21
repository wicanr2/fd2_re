#!/usr/bin/env python3
from pathlib import Path
import argparse, json, os, shutil, sys, tempfile
from PIL import Image
sys.path.insert(0, str(Path(__file__).resolve().parent))
from verify_fd2_compound_caster_gpu import make_native_occlusion_test
from build_fd2_gaia_battle_hd import digest

parser=argparse.ArgumentParser(description='核固定FDOTHER32原生遮擋及65現代特效alpha，不調低人物RGB門檻。')
for name in ['original-pack','source','out']: parser.add_argument('--'+name,type=Path,required=True)
args=parser.parse_args()
r=Path(__file__).resolve().parents[1]; out=args.out
assert Path('/.dockerenv').exists() and not out.exists() and out.parent.stat().st_uid==os.getuid()
proof=json.loads((r/'docs/data/fd2-battle-hd-source-inventory-20261019.json').read_text())
original=args.original_pack; layer=dict(archive='FDOTHER.DAT',resource=32,frame=4,x=193,y=38,width=33,height=33,fill=-1,clip=None)
mask_path=original/'animations/FDOTHER_032/FDOTHER_032_f04_mask.png'
with Image.open(mask_path) as im: mask=im.convert('L')
opaque=next((x,y) for y in range(33) for x in range(33) if mask.getpixel((x,y))!=0)
clear=next((x,y) for y in range(33) for x in range(33) if mask.getpixel((x,y))==0)
pos=lambda p: (layer['x']*6+p[0]*6+2,layer['y']*6+p[1]*6+2)
occluded=make_native_occlusion_test({},Path('/unused'),original,proof)
assert occluded(*pos(opaque),[layer])
assert not occluded(*pos(clear),[layer])
assert not occluded(*pos(opaque),[{**layer,'clip':{'Min':{'X':0,'Y':0},'Max':{'X':1,'Y':1}}}])
checks=['native-opaque-mask','native-transparent-mask','clip-before-mask']
try: occluded(*pos(opaque),[{**layer,'archive':'UNKNOWN.DAT'}])
except SystemExit as e: assert '不能解釋' in str(e); checks.append('unknown-archive-rejected')
else: raise AssertionError('unknown foreground accepted')
with tempfile.TemporaryDirectory(prefix='fd2-occlusion-') as name:
 p=Path(name); shutil.copyfile(original/'manifest.json',p/'manifest.json')
 dest=p/'animations/FDOTHER_032'; dest.mkdir(parents=True)
 shutil.copyfile(original/'animations/FDOTHER_032/animation.json',dest/'animation.json')
 shutil.copyfile(mask_path,dest/mask_path.name)
 mask.putpixel(opaque,0); mask.save(dest/mask_path.name)
 try: make_native_occlusion_test({},Path('/unused'),p,proof)(*pos(opaque),[layer])
 except SystemExit as e: assert 'SHA或定位不同' in str(e); checks.append('tampered-native-mask-rejected')
 else: raise AssertionError('tampered mask accepted')
catalog=json.loads((r/'remake/assets/themes/modern/catalog.json').read_text())
effects={a['source_resource']:a for a in catalog['assets'] if a['role']=='battle_effect_hd_set'}
effect=effects[65]; desc=effect['native_frames'][0]
with Image.open(args.source/effect['files'][0]) as im:
 alpha=im.getchannel('A'); q=next((x,y) for y in range(im.height) for x in range(im.width) if alpha.getpixel((x,y))>0)
layer65=dict(archive='FDOTHER.DAT',resource=65,frame=0,x=0,y=0,width=desc['width'],height=desc['height'],fill=-1,clip=None)
assert make_native_occlusion_test({},args.source,original,proof,effects)(*q,[layer65])
checks.append('registered-HD-effect-alpha')
out.write_text(json.dumps(dict(status='PASS',checks=checks,manifest_sha256=digest(original/'manifest.json'),scope='固定原生遮罩、HD alpha、clip與來源拒收；不降低實圖RGB門檻'),ensure_ascii=False,indent=2)+'\n')
print(out.read_text())
