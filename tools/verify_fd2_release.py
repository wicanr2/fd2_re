#!/usr/bin/env python3
"""正式附件的獨立下載核對；需在含 Git／Python 的 Docker 工具鏈執行。"""
from pathlib import Path, PurePosixPath
import argparse,datetime,re,subprocess
import json,hashlib,urllib.request,zipfile,struct,plistlib,os
parser=argparse.ArgumentParser(description="下載正式 Release，核對來源、公開清單、封包結構與 SHA-256。")
parser.add_argument("--version",required=True)
parser.add_argument("--commit",required=True)
args=parser.parse_args()
v=args.version;commit=args.commit
assert re.fullmatch(r"v\.[0-9]+\.[0-9]+\.[0-9]+-[0-9]{8}",v)
datetime.datetime.strptime(v[-8:],"%Y%m%d")
assert re.fullmatch(r"[0-9a-f]{40}",commit)
os.chdir(Path(__file__).resolve().parents[1])
owner=(os.getuid(),os.getgid())
for parent in (Path("work"),Path("dist-all")):
 assert(parent.stat().st_uid,parent.stat().st_gid)==owner
work=Path("work/release-"+v)
work.mkdir(parents=True,exist_ok=True)
assert(work.stat().st_uid,work.stat().st_gid)==owner
request=urllib.request.Request("https://api.github.com/repos/wicanr2/fd2_re/releases/tags/"+v,
                              headers={"User-Agent":"fd2-release-verification"})
api=json.load(urllib.request.urlopen(request,timeout=30))
(work/"release-api.json").write_text(json.dumps(api,ensure_ascii=False,indent=2)+"\n",encoding="utf-8")
assert api["tag_name"]==v and not api["draft"] and not api["prerelease"]
root=Path("dist-all")/v
patch=root/"patch";smoke=root/"smoke"
for p in [root,patch,smoke]:
 p.mkdir(parents=True,exist_ok=True);assert(p.stat().st_uid,p.stat().st_gid)==owner
assets={a["name"]:a for a in api["assets"]}
expected={f"FD2-{v}-linux-x86_64.AppImage",f"FD2-{v}-windows-x86_64.zip",f"FD2-{v}-macos-universal.zip","SHA256SUMS.json","VALIDATION.json"}
assert set(assets)==expected,(set(assets),expected)
def sha(data):return hashlib.sha256(data).hexdigest()
def download(name,target):
 a=assets[name]
 if target.exists():
  assert(target.stat().st_uid,target.stat().st_gid)==owner
  data=target.read_bytes()
  assert len(data)==a["size"]
  assert a.get("digest")==("sha256:"+sha(data)),name
 else:
  req=urllib.request.Request(a["browser_download_url"],headers={"User-Agent":"fd2-release-verification"})
  with urllib.request.urlopen(req,timeout=90) as response:data=response.read()
  assert len(data)==a["size"]
  if a.get("digest"):assert a["digest"]=="sha256:"+sha(data),name
  temporary=target.with_name(target.name+".part")
  if temporary.exists():assert(temporary.stat().st_uid,temporary.stat().st_gid)==owner
  temporary.write_bytes(data);temporary.replace(target)
 return data
manifest=json.loads(download("SHA256SUMS.json",root/"SHA256SUMS.json"))
validation=json.loads(download("VALIDATION.json",smoke/"VALIDATION.json"))
assert manifest["version"]==v and manifest["source_commit"]==commit
assert validation["version"]==v and validation["source_commit"]==commit
assert len(manifest["files"])==3 and len(validation["platforms"])==3
tracked=subprocess.check_output(["git","ls-tree","-rz","--name-only",commit,"--","remake/assets"])
allow={x.decode()[len("remake/assets/"):] for x in tracked.split(b"\0") if x}
license_source=subprocess.check_output(["git","show",commit+":LICENSE"]).decode("utf-8").replace("\r\n","\n")
reports=[]
for row in manifest["files"]:
 name=row["file"];data=download(name,patch/name)
 assert sha(data)==row["sha256"] and len(data)==row["size"]
 native=next(p for p in validation["platforms"] if p["package"]==name)
 assert native["status"]=="passed" and native["source_commit"]==commit and native["sha256"]==sha(data)
 report={"file":name,"size":len(data),"sha256":sha(data),"native_check":"passed","distribution":"engine-only"}
 if name.endswith(".zip"):
  with zipfile.ZipFile(patch/name) as z:
   assert z.testzip() is None
   for entry in z.infolist():
    path=PurePosixPath(entry.filename)
    assert not path.is_absolute() and ".." not in path.parts
    if not entry.filename.isascii():assert entry.flag_bits & 0x800
   platform=native["platform"]
   base=f"FD2-{v}-{platform}/"
   resource=base+"FD2.app/Contents/Resources/" if platform=="macos-universal" else base
   info=json.loads(z.read(resource+"FD2-RELEASE.json"))
   assert info["version"]==v and info["source_commit"]==commit and info["platform"]==platform
   assert info["distribution"]=="engine-only" and info["original_assets_included"] is False
   names={n[len(resource+"assets/"):] for n in z.namelist() if n.startswith(resource+"assets/") and not n.endswith("/")}
   assert names==allow==set(info["assets"])
   for relative in names:
    content=z.read(resource+"assets/"+relative)
    assert sha(content)==info["assets"][relative]["sha256"]
    assert PurePosixPath(relative).suffix.lower() not in {".dat",".sav",".ogg",".wav",".mid",".b24",".exe"}
   assert z.read(resource+"LICENSE").decode("utf-8").replace("\r\n","\n")==license_source
   assert z.read(resource+"third-party/Go/LICENSE")
   assert json.loads(z.read(resource+"third-party/modules.json"))
   if platform=="windows-x86_64":
    readme=z.read(base+"README.txt");assert readme.startswith(b"\xef\xbb\xbf")
    assert b"\n" not in readme.replace(b"\r\n",b"")
    bat=z.read(base+"start-fd2.bat");assert bat.isascii() and b"\n" not in bat.replace(b"\r\n",b"")
    exe=z.read(base+"fd2.exe");offset=struct.unpack_from("<I",exe,0x3c)[0]
    assert exe[offset:offset+4]==b"PE\0\0" and struct.unpack_from("<H",exe,offset+4)[0]==0x8664
    assert struct.unpack_from("<H",exe,offset+24+68)[0]==2
   else:
    exe_name=base+"FD2.app/Contents/MacOS/fd2";exe=z.read(exe_name)
    assert exe[:4]==b"\xca\xfe\xba\xbe"
    cpus={struct.unpack_from(">I",exe,8+i*20)[0] for i in range(struct.unpack_from(">I",exe,4)[0])}
    assert cpus=={0x1000007,0x100000c}
    assert (z.getinfo(exe_name).external_attr>>16)&0o111
    plist=plistlib.loads(z.read(base+"FD2.app/Contents/Info.plist"))
    assert plist["FD2ReleaseVersion"]==v
   report["public_assets"]=len(names);report["archive_check"]="CRC, UTF-8, architecture, mode, licenses, exact public assets passed"
 else:
  assert data[:4]==b"\x7fELF";os.chmod(patch/name,0o755)
 reports.append(report)
write={"version":v,"source_commit":commit,"release_url":api["html_url"],"published_at":api["published_at"],"status":"download-verified","packages":reports}
(work/"download-verification.json").write_text(json.dumps(write,ensure_ascii=False,indent=2)+"\n",encoding="utf-8")
print(json.dumps(write,ensure_ascii=False,indent=2))
