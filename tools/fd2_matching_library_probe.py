#!/usr/bin/env python3
"""以完整SDK物件與固定IDA函式比對函式庫來源，不以遮罩或相鄰位址分類。"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import struct
import subprocess

INPUT_SHA = '222b7d067ad4450eb9c5f6e6bce1797d54bb050417ba39ced6067f8039f28c4f'
LIBRARY_SHA = '031c9b3467137d357dfbed0ab3fad4badd0fb862afd399dad473615ff2ccdb48'
LIBRARIES = {
    '10.0a': ('031c9b3467137d357dfbed0ab3fad4badd0fb862afd399dad473615ff2ccdb48','WATCOM/LIB386/DOS/CLIB3S.LIB'),
    '9.5': ('95b423adf7561bd638876fac51108c92243cc057e0084e98b98b7cd032b3edae','W9532_02/clib3s.dos/clib3s.lib'),
    '9.01': ('3a37d23600e6e9564a12492bc4088cdbcaa243850a4b45f203691db05ea441b8','disk02/clib3s.dos/clib3s.lib'),
}
CONVERTER_SHA = '5dc2d30a8bba85cff9717408f1a7b16128dbb835cebbd46f40268417ea32b6c1'


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def coff_sections(raw):
    if len(raw) < 20:
        raise ValueError('COFF標頭截斷')
    machine,count,_,sym_at,sym_count,optional,_=struct.unpack_from('<HHIIIHH',raw)
    if machine!=0x14c or optional or len(raw)<20+count*40:
        raise ValueError('非支援的i386 COFF')
    end=sym_at+sym_count*18
    if end+4>len(raw):raise ValueError('COFF符號表截斷')
    string_size=struct.unpack_from('<I',raw,end)[0]
    if string_size<4 or end+string_size>len(raw):raise ValueError('COFF字串表截斷')
    strings=raw[end:end+string_size];symbols=[];n=0
    while n<sym_count:
        name,value,section,kind,storage,aux=struct.unpack_from('<8sIhHBB',raw,sym_at+n*18)
        if n+aux>=sym_count:raise ValueError('COFF輔助符號截斷')
        if name[:4]==b'\0'*4:
            at=struct.unpack_from('<I',name,4)[0]
            if not 4<=at<len(strings) or b'\0' not in strings[at:]:raise ValueError('COFF長名稱截斷')
            name=strings[at:strings.index(b'\0',at)]
        else:name=name.rstrip(b'\0')
        symbols.append({'name':name.decode('ascii'),'offset':value,'section':section,'storage':storage,'index':n})
        n+=1+aux
    sections=[]
    for n in range(count):
        h=struct.unpack_from('<8sIIIIIIHHI',raw,20+n*40)
        name,_,_,size,at,rel_at,_,rel_count,_,flags=h
        if size and (at<20+count*40 or at+size>len(raw)):raise ValueError('COFF區段截斷')
        if rel_count and rel_at+rel_count*10>len(raw):raise ValueError('COFF重定位截斷')
        sections.append({'name':name.rstrip(b'\0').decode('ascii'),'size':size,'data':raw[at:at+size],
                         'relocation_count':rel_count,'flags':flags,'index':n+1,
                         'publics':[s for s in symbols if s['section']==n+1 and s['storage']==2]})
    return sections


def omf_sections(raw):
    """依原始record讀完整USE32 CODE段；任何FIXUPP皆拒收比對，不做重定位遮罩。"""
    names=[''];segments=[];publics=[];blocks={};fixup_records=0;at=0;ended=False;easy32=False
    def index(data,pos):
        if pos>=len(data):raise ValueError('OMF index截斷')
        value=data[pos];pos+=1
        if value&0x80:
            if pos>=len(data):raise ValueError('OMF長index截斷')
            value=((value&0x7f)<<8)|data[pos];pos+=1
        return value,pos
    def number(data,pos,wide):
        width=4 if wide else 2
        if pos+width>len(data):raise ValueError('OMF numeric截斷')
        return int.from_bytes(data[pos:pos+width],'little'),pos+width
    def string(data,pos):
        if pos>=len(data):raise ValueError('OMF string截斷')
        size=data[pos];pos+=1
        if pos+size>len(data):raise ValueError('OMF string內容截斷')
        return data[pos:pos+size].decode('ascii'),pos+size
    while at<len(raw):
        if at+3>len(raw):raise ValueError('OMF record標頭截斷')
        kind=raw[at];length=int.from_bytes(raw[at+1:at+3],'little');end=at+3+length
        if length<1 or end>len(raw):raise ValueError('OMF record內容截斷')
        record=raw[at:end]
        if sum(record)&255 and record[-1]!=0:raise ValueError('OMF checksum錯誤')
        data=record[3:-1];pos=0;base=kind&0xfe
        if base==0x88 and data==b'\x80\xaa80386':
            if segments:raise ValueError('EasyOMF signature在SEGDEF後，拒收混合格式')
            easy32=True
        wide=bool(kind&1) or easy32
        if at==0 and base not in (0x80,0x82):raise ValueError('OMF缺少THEADR')
        if base in (0x96,0xca):
            while pos<len(data):
                name,pos=string(data,pos);names.append(name)
        elif base==0x98:
            if not data:raise ValueError('OMF SEGDEF截斷')
            attr=data[0];pos=1
            if attr>>5==0:pos+=3
            size,pos=number(data,pos,wide)
            if attr&2 and not wide and size==0:size=65536
            name,pos=index(data,pos);cls,pos=index(data,pos);overlay,pos=index(data,pos)
            if pos!=len(data) or any(n>=len(names) for n in (name,cls,overlay)):raise ValueError('OMF SEGDEF名稱或長度不符')
            segments.append({'name':names[name],'class':names[cls],'size':size,'use32':bool(attr&1) or easy32})
        elif base==0x90:
            group,pos=index(data,pos);segment,pos=index(data,pos)
            if segment==0:pos+=2
            if segment>len(segments):raise ValueError('OMF PUBDEF segment超界')
            while pos<len(data):
                name,pos=string(data,pos);offset,pos=number(data,pos,wide);type_index,pos=index(data,pos)
                publics.append({'name':name,'offset':offset,'section':segment,'storage':2})
        elif base in (0xa0,0xa2):
            segment,pos=index(data,pos);offset,pos=number(data,pos,wide)
            if not 1<=segment<=len(segments):raise ValueError('OMF DATA segment超界')
            if base==0xa2:
                if segments[segment-1]['class'].upper()=='CODE':raise ValueError('OMF CODE LIDATA尚未支援')
            else:blocks.setdefault(segment,[]).append((offset,data[pos:]))
        elif base==0x9c:fixup_records+=1
        elif base==0x8a:
            if end!=len(raw):raise ValueError('OMF MODEND後仍有未核對資料')
            ended=True
        at=end
    if not ended:raise ValueError('OMF缺少MODEND')
    result=[]
    for n,seg in enumerate(segments,1):
        if seg['class'].upper()!='CODE' or not seg['size']:continue
        if not seg['use32']:raise ValueError('OMF CODE不是USE32')
        cursor=0;code=bytearray()
        for offset,data in sorted(blocks.get(n,[])):
            if offset!=cursor:raise ValueError('OMF CODE有缺口或重疊，不填補byte')
            code.extend(data);cursor+=len(data)
        if cursor!=seg['size']:raise ValueError('OMF CODE不涵蓋完整SEGDEF')
        ps=[p for p in publics if p['section']==n]
        if any(p['offset']>=cursor for p in ps):raise ValueError('OMF CODE公開入口超界')
        result.append({'name':seg['name'],'size':cursor,'data':bytes(code),'flags':0x20,'index':n,'publics':ps,
                       'relocation_count':-1 if fixup_records else 0,'fixup_record_count':fixup_records,
                       'omf_encoding':'easy_omf32' if easy32 else 'omf32'})
    return result


def main():
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--objects',type=Path,required=True)
    parser.add_argument('--library',type=Path,required=True)
    parser.add_argument('--library-version',choices=LIBRARIES,default='10.0a',help='固定原廠輸入包標籤，不辨識原作版本')
    parser.add_argument('--converter',type=Path,required=True)
    parser.add_argument('--evidence',type=Path,required=True)
    parser.add_argument('--original',type=Path,required=True)
    parser.add_argument('--output',type=Path,required=True)
    parser.add_argument('--representation',choices=('coff','omf'),default='omf')
    args=parser.parse_args()
    if not Path('/.dockerenv').exists():raise SystemExit('本工具只在Docker執行')
    library_sha,library_path=LIBRARIES[args.library_version]
    if sha(args.library)!=library_sha or sha(args.converter)!=CONVERTER_SHA:raise ValueError('SDK或converter與固定來源不符')
    library_bytes=args.library.read_bytes()
    original=args.original.read_bytes()
    if hashlib.sha256(original).hexdigest()!=INPUT_SHA:raise ValueError('原版版本不符')
    image=json.loads(args.evidence.read_text())
    if image['input']['sha256']!=INPUT_SHA or image['tool']['version']!='9.4':raise ValueError('IDA版本或輸入不符')
    args.output.mkdir(parents=True,exist_ok=True)
    if args.output.stat().st_uid!=os.getuid():raise ValueError('輸出擁有權不符')
    by_code={}
    for f in image['functions']:
        start=int(f['inventory']['start'],16);end=int(f['inventory']['end'],16)
        instructions=[i for c in f['chunks'] for i in c['instructions']]
        cursor=start;code=bytearray();continuous=True
        for i in instructions:
            raw=bytes.fromhex(i['loaded_bytes']);at=int(i['ida_linear_address'],16)
            offset=int(i['file_offset'],16);file_bytes=bytes.fromhex(i['file_bytes'])
            if original[offset:offset+len(file_bytes)]!=file_bytes:raise ValueError('IDA原始位元組版本不符')
            if at!=cursor:continuous=False
            cursor=at+len(raw);code.extend(raw)
        if continuous and cursor==end and len(code)==f['inventory']['size']:
            by_code.setdefault(bytes(code),[]).append(f)
    matches=[];members=[];conversion_failures=[];eligible=0
    for obj in sorted(args.objects.glob('*.o')):
        object_bytes=obj.read_bytes()
        library_offset=library_bytes.find(object_bytes)
        if library_offset<0:
            conversion_failures.append({'member':obj.name,'object_sha256':sha(obj),'error':'抽取物件不是固定LIB中的完整相同byte序列'});continue
        out=args.output/obj.stem;out.mkdir(exist_ok=True)
        converted=out/'member.cof'
        converted_sha=None
        if args.representation=='coff':
            p=subprocess.run([str(args.converter),'-fcoff',str(obj),str(converted)],capture_output=True,text=True,timeout=20)
            if p.returncode:
                conversion_failures.append({'member':obj.name,'error':p.stdout+p.stderr});continue
            converted_sha=sha(converted)
        try:
            sections=coff_sections(converted.read_bytes()) if args.representation=='coff' else omf_sections(object_bytes)
        except ValueError as error:
            conversion_failures.append({'member':obj.name,'object_sha256':sha(obj),'coff_sha256':converted_sha,'error':str(error)})
            continue
        code=[s for s in sections if s['flags']&0x20 and s['size']]
        descriptor={'member':obj.name,'object_sha256':sha(obj),'library_file_offset':hex(library_offset),'coff_sha256':converted_sha,
                    'object_size':len(object_bytes),
                    'code_sections':[{'name':s['name'],'size':s['size'],'relocation_count':s['relocation_count'],'publics':s['publics']} for s in code]}
        members.append(descriptor)
        # 只接受整個唯一code區段、唯一入口及完全沒有重定位；不清零或裁切任何byte。
        if len(code)!=1:continue
        section=code[0];publics=section['publics']
        if section['relocation_count'] or section['size']<32 or not publics or {s['offset'] for s in publics}!={0}:continue
        eligible+=1
        for f in by_code.get(section['data'],[]):
            matches.append({'ida_linear_address':f['inventory']['start'],'original_name':f['inventory']['ida_analysis_name'],
                            'size':len(section['data']),'member':obj.name,'vendor_symbols':[s['name'] for s in publics],
                            'object_sha256':sha(obj),'code_sha256':hashlib.sha256(section['data']).hexdigest(),
                            'library_file_offset':hex(library_offset),
                            'object_size':len(object_bytes),
                            'original_classification':f['inventory']['classification'],
                            'caller_functions':f['inventory']['direct_caller_functions'],
                            'equality_confidence':'已證實','module_classification':'尚待IDA caller／symbol審查',
                            'function_semantics':'unknown','comparison':'整個原始IDA函式與整個無重定位SDK code區段完全相同'})
    result={'schema_version':1,'input':image['input'],'tool':image['tool'],'evidence_sha256':sha(args.evidence),
            'library':{'source_package':args.library_version,'path':library_path,'sha256':library_sha,'size':args.library.stat().st_size,'original_version':'unknown'},
            'converter_sha256':CONVERTER_SHA,'driver_sha256':sha(Path(__file__)),
            'representation':args.representation,'converter_used':args.representation=='coff',
            'members':members,'conversion_failures':conversion_failures,'eligible_complete_members':eligible,'matches':matches,
            'classification_changed':False,'rights':'SDK物件、COFF及完整code只留本機；此probe不直接調整原始分類。'}
    (args.output/'library-probe.json').write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n')
    print(json.dumps({'members':len(members),'conversion_failures':len(conversion_failures),'eligible':eligible,'matches':matches},ensure_ascii=False))


if __name__=='__main__':main()
