#!/usr/bin/env python3
"""在 Pillow Docker 依原開場分鏡編製 HD 候選；正式清冊由驗收後另行接納。"""

import argparse
from collections import Counter
import hashlib
import json
import os
from pathlib import Path
import shutil

from PIL import Image, ImageChops, ImageDraw, ImageOps


COUNTS = {1: 51, 3: 28, 4: 12, 5: 35, 6: 12, 7: 17, 8: 12}
CONTRACT = 'opening_native_schedule_rgba_hd6_v1'
MASTERS = {
    'scroll': 'opening-scroll-hd-draft-v2.png',
    'sol': 'opening-sol-hd-draft-v2.png',
    'ares': 'opening-ares-hd-draft-v1.png',
    'guardian': 'opening-guardian-hd-draft-v1.png',
    'castle': 'opening-castle-hd-draft-v1.png',
    'title': 'opening-title-hd-draft-v1.png',
    'number2': 'opening-number2-hd-draft-v1.png',
    'beam30': 'opening-ani5-frame30-hd-draft-v1.png',
}


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def dump(path, value):
    path.write_text(json.dumps(value, ensure_ascii=False, indent=2) + '\n')


def source_paths(archive, resource):
    if archive == 'ANI.DAT':
        return [f'animations/ANI_{resource:03}/animation.json']
    if resource == 69:
        return [f'surfaces/FDOTHER_{i:03}/resource.json' for i in range(69, 74)] + [
            'palette/fdother_101.json', 'palette/fdother_102.json']
    palettes = {7: 8, 74: 76, 75: 76, 100: 99}
    surface = 'surfaces/FDOTHER_007/entry_000/resource.json' if resource == 7 else f'surfaces/FDOTHER_{resource:03}/resource.json'
    return [surface, f'palette/fdother_{palettes[resource]:03}.json']


def source_digest(original, archive, resource):
    text = ''.join(f'{name} {sha(original / name)}\n' for name in source_paths(archive, resource))
    return hashlib.sha256(text.encode()).hexdigest()


def native_clip(original, resource):
    folder = original / f'animations/ANI_{resource:03}'
    doc = json.loads((folder / 'animation.json').read_text())
    expected = 96 if resource == 0 else COUNTS[resource]
    if doc['frame_count'] != expected or doc['source']['sha256'] != 'be909c71d0f1121b6632ae931d978e990f6d54c830f4e0509cd6862187c4d963':
        raise ValueError('原版ANI版本或影格數不同')
    frames = [Image.open(folder / row['frame']).copy() for row in doc['frames']]
    if any(im.mode != 'P' or im.size != (320, 200) for im in frames):
        raise ValueError('ANI索引畫布不同')
    return frames


def surface(original, resource, palette):
    folder = original / f'surfaces/FDOTHER_{resource:03}'
    if resource == 7:
        folder /= 'entry_000'
    im = Image.open(folder / 'frame.png').copy()
    dac = json.loads((original / f'palette/fdother_{palette:03}.json').read_text())['dac_6bit_components']
    im.putpalette([(v << 2) | (v >> 4) for v in dac])
    return im.convert('RGBA')


def alpha_crop(im):
    im = im.convert('RGBA')
    box = im.getchannel('A').point(lambda v: 255 if v >= 16 else 0).getbbox()
    if box is None:
        raise ValueError('透明母稿沒有主體')
    return im.crop(box)


def body_box(im):
    """人物幕右／左空背景列固定；以該列逐行消除背景，僅作美術錨點。"""
    rgb = im.convert('RGB')
    edge = 319 if rgb.getpixel((319, 100)) == rgb.getpixel((0, 100)) else (
        319 if rgb.getpixel((319, 100)) == rgb.getpixel((319, 101)) else 0)
    # ANI4在左，ANI6在右；以較少變化的外側列選空背景。
    if len(set(rgb.crop((0, 0, 1, 200)).getdata())) < len(set(rgb.crop((319, 0, 320, 200)).getdata())):
        edge = 0
    else:
        edge = 319
    mask = Image.new('L', (320, 200))
    mask.putdata([255 if rgb.getpixel((x, y)) != rgb.getpixel((edge, y)) else 0
                  for y in range(200) for x in range(320)])
    return mask.getbbox(), edge


def place(canvas, picture, box):
    x0, y0, x1, y1 = box
    canvas.alpha_composite(picture.resize((max(1, (x1-x0)*6), max(1, (y1-y0)*6)), Image.Resampling.LANCZOS), (x0*6, y0*6))


def pose_background(native, edge):
    rgb = native.convert('RGB')
    band = Image.new('RGB', (1, 200))
    band.putdata([rgb.getpixel((edge, y)) for y in range(200)])
    return band.resize((1920, 1200), Image.Resampling.BICUBIC).convert('RGBA')


def blue_pose(picture):
    alpha = picture.getchannel('A')
    light = ImageOps.grayscale(picture)
    blue = ImageOps.colorize(light, (0, 0, 8), (45, 70, 255)).convert('RGBA')
    blue.putalpha(alpha)
    return blue


def static_pose_frames(native, master, box, edge):
    out = []
    for i, frame in enumerate(native):
        if i in (6, 7):
            out.append(Image.new('RGBA', (1920, 1200), 'white'))
            continue
        canvas = pose_background(frame, edge)
        place(canvas, blue_pose(master) if i < 6 else master, box)
        out.append(canvas)
    return out


def sprite_candidates(original, resources):
    out = []
    for resource in resources:
        folder = original / f'animations/FIGANI_{resource:03}'
        doc = json.loads((folder / 'animation.json').read_text())
        for i, row in enumerate(doc['frames']):
            im = Image.open(folder / row['path'])
            mask = Image.open(folder / row['mask_path'])
            pix, coverage = im.tobytes(), mask.tobytes()
            runs = []
            for y in range(im.height):
                x = 0
                while x < im.width:
                    if not coverage[y*im.width+x]:
                        x += 1
                        continue
                    end = x+1
                    while end < im.width and coverage[y*im.width+end]:
                        end += 1
                    if end-x >= 16:
                        runs.append((end-x, x, y, pix[y*im.width+x:y*im.width+end]))
                    x = end
            runs.sort(reverse=True)
            # Separated rows avoid accepting a long uniform stripe as a match.
            patterns, ys = [], []
            for length, x, y, pattern in runs:
                if len(set(pattern)) >= 4 and all(abs(y-yy) >= 7 for yy in ys):
                    patterns.append((x, y, pattern)); ys.append(y)
                if len(patterns) == 3:
                    break
            points = [(n % im.width, n // im.width, value) for n, value in enumerate(pix) if coverage[n]]
            vertical = []
            for x in range(0, im.width, max(1, im.width//12)):
                for y in range(max(0, im.height-15)):
                    if all(coverage[(y+k)*im.width+x] for k in range(16)):
                        pattern = bytes(pix[(y+k)*im.width+x] for k in range(16))
                        if len(set(pattern)) >= 4:
                            vertical.append((x, y, pattern))
                            break
            out.append(dict(resource=resource, frame=i, points=points, patterns=patterns,
                            vertical_patterns=vertical))
    return out


def locate_figures(native, candidates, dragon=False):
    target = native.tobytes()
    columns = [bytes(target[y*320+x] for y in range(200)) for x in range(320)]
    matches = []
    for sprite in candidates:
        coords = set()
        for x, y, pattern in sprite['patterns']:
            at = target.find(pattern)
            while at >= 0:
                if at % 320 + len(pattern) <= 320:
                    coords.add((at % 320-x, at // 320-y))
                at = target.find(pattern, at+1)
        # FIGANI98's three pure beam fragments have mostly uniform horizontal
        # runs. Body poses already supply discriminating horizontal rows.
        vertical = sprite['vertical_patterns'] if sprite['resource'] == 98 and sprite['frame'] >= 9 else ()
        for x, y, pattern in vertical:
            for tx, column in enumerate(columns):
                at = column.find(pattern)
                while at >= 0:
                    coords.add((tx-x, at-y))
                    at = column.find(pattern, at+1)
        for dx, dy in coords:
            exact, mismatched = [], Counter()
            total, rejected = 0, False
            mismatch_limit = max(12, len(sprite['points'])*.005)
            bad = 0
            for x, y, value in sprite['points']:
                x, y = x+dx, y+dy
                if not (0 <= x < 320 and 0 <= y < 200):
                    continue
                total += 1
                if target[y*320+x] == value:
                    exact.append(y*320+x)
                else:
                    mismatched[value] += 1
                    if not (dragon and sprite['resource'] in (312, 313, 314) and value == 31):
                        bad += 1
                        if bad > mismatch_limit:
                            rejected = True
                            break
            if rejected or total < 500:
                continue
            permitted = dragon and sprite['resource'] in (312, 313, 314) and not (set(mismatched)-{31})
            ratio = len(exact)/total
            if ratio >= .995 or permitted and ratio >= .9:
                matches.append(dict(resource=sprite['resource'], frame=sprite['frame'], x=dx, y=dy,
                                    exact_pixels=len(exact), visible_pixels=total, ratio=ratio,
                                    mismatched_source_colours=dict(mismatched), coverage=set(exact)))
    matches.sort(key=lambda r: (-r['exact_pixels'], -r['ratio'], r['resource'], r['frame']))
    selected, coverage = [], set()
    for match in matches:
        fresh = match['coverage']-coverage
        if len(fresh) < max(200, len(match['coverage'])*.25):
            continue
        coverage |= match.pop('coverage')
        selected.append(match)
    # Native special trails may overlap; paint smaller old poses before the
    # largest visible final actor. No game-side transparency rule is inferred.
    selected.sort(key=lambda r: r['exact_pixels'])
    return selected


def located_coverage(placements, candidates):
    points = set()
    by_key = {(row['resource'], row['frame']): row for row in candidates}
    for row in placements:
        source = by_key[row['resource'], row['frame']]
        for x, y, value in source['points']:
            x, y = x+row['x'], y+row['y']
            if 0 <= x < 320 and 0 <= y < 200:
                points.add(y*320+x)
    return points


def locate_enemy(native, candidates, origin, foreground):
    """固定站位已由首幀證實；先排除前景再比較敵人，避免遮擋被當成消失。"""
    target = native.tobytes()
    rows = []
    diagnostics = []
    dx, dy = origin
    for source in candidates:
        good = total = 0
        mismatches = Counter()
        for x, y, value in source['points']:
            x, y = x+dx, y+dy
            if not (0 <= x < 320 and 0 <= y < 200) or y*320+x in foreground:
                continue
            if source['resource'] in (312, 313, 314) and value == 31:
                continue
            total += 1
            if target[y*320+x] == value:
                good += 1
            else:
                mismatches[value] += 1
        if total:
            diagnostics.append((good/total, good, total, source['resource'], source['frame']))
        if total >= 300 and good/total >= .97:
            rows.append(dict(resource=source['resource'], frame=source['frame'], x=dx, y=dy,
                             exact_pixels=good, visible_pixels=total, ratio=good/total,
                             mismatched_source_colours=dict(mismatches),
                             method='pixel-match-after-foreground-occlusion'))
    if not rows:
        raise ValueError(f'排除前景後仍無受來源支持的敵人影格：{sorted(diagnostics, reverse=True)[:4]}')
    return max(rows, key=lambda row: (row['exact_pixels'], row['ratio'], -row['resource']))


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--original', type=Path, required=True)
    parser.add_argument('--modern', type=Path, required=True)
    parser.add_argument('--drafts', type=Path, required=True)
    parser.add_argument('--catalog', type=Path, required=True)
    parser.add_argument('--out', type=Path, required=True)
    args = parser.parse_args()
    assert Path('/.dockerenv').is_file()
    assert not args.out.exists()
    assert args.out.parent.stat().st_uid == os.getuid()
    for name in MASTERS.values():
        assert (args.drafts/name).is_file(), name
    args.out.mkdir()
    catalog = json.loads(args.catalog.read_text())
    admitted = catalog['assets']
    master_names, master_images = {}, {}
    for kind, filename in MASTERS.items():
        name = f'opening-{kind}-master-nov04-r1.png'
        shutil.copy2(args.drafts/filename, args.out/name)
        master_names[kind] = name
        master_images[kind] = Image.open(args.out/name).convert('RGBA')
    fragments, plans = [], []

    def add_bank(archive, resource, pictures, methods, masters):
        key = f'{"ani" if archive == "ANI.DAT" else "fdother"}-{resource:03}'
        files, hashes = [], []
        geometry = pictures[0].size
        for i, picture in enumerate(pictures):
            assert picture.size == geometry
            assert picture.getchannel('A').getextrema() == (255, 255)
            name = f'opening-{key}-hd6-nov04-r1-f{i:03}.png'
            picture.save(args.out/name, compress_level=6)
            files.append(name); hashes.append(sha(args.out/name))
        plan_name = f'opening-{key}-plan-nov04-r1.json'
        plan = dict(schema_version=1, kind='opening_hd_art_derivation', source_archive=archive,
                    source_resource=resource, source_metadata_sha256=source_digest(args.original, archive, resource),
                    compiler='tools/build_fd2_opening_hd.py', compiler_sha256=sha(Path(__file__)),
                    original_parity=False, frames=methods)
        dump(args.out/plan_name, plan)
        sources = [master_names[m] for m in masters] + [plan_name]
        asset = dict(asset_id=f'modern.opening.{key}.hd6', role='opening_presentation_hd_set',
                     status='runtime_candidate', master_file=sources[0], width=geometry[0], height=geometry[1],
                     frame_count=len(files), files=files, frame_sha256=hashes,
                     source_archive=archive, source_resource=resource, source_metadata_sha256=plan['source_metadata_sha256'],
                     source_files=sources, source_sha256=[sha(args.out/name) for name in sources],
                     consumer_contract=CONTRACT, layout_policy='full_native_canvas_hd6', alpha_contract='opaque',
                     source_refs=['docs/knowledge-base/61-modern-theme-runtime-spec.md'])
        fragments.append(asset); plans.append(plan)

    # Brand signature remains the exact original design, with mechanical
    # interpolation only. It is never sent through a generative lettering pass.
    publisher = surface(args.original, 74, 76).resize((1920, 1200), Image.Resampling.LANCZOS)
    publisher_name = 'opening-publisher-master-nov04-r1.png'
    publisher.save(args.out/publisher_name)
    master_names['publisher'] = publisher_name
    add_bank('FDOTHER.DAT', 74, [publisher], [dict(method='source-brand-preserved-lanczos', resource=74)], ['publisher'])
    for resource, kind in ((7, 'title'), (75, 'castle'), (100, 'guardian')):
        canvas = Image.new('RGBA', (1920, 1200), 'black')
        canvas.alpha_composite(master_images[kind].resize((1920, 1200), Image.Resampling.LANCZOS))
        add_bank('FDOTHER.DAT', resource, [canvas], [dict(method='source-guided-full-master', palette={7: 8, 75: 76, 100: 99}[resource])], [kind])
    scroll = master_images['scroll'].resize((1920, 4410), Image.Resampling.LANCZOS)
    # Pulse changes only the colour emphasis, following the source's blue/red
    # armour distinction. The black/red source remains the normal full master.
    pulse = Image.blend(scroll, ImageOps.colorize(ImageOps.grayscale(scroll), (0, 0, 0), (150, 175, 255)).convert('RGBA'), .32)
    add_bank('FDOTHER.DAT', 69, [scroll, pulse], [dict(method='source-guided-master'), dict(method='art-pulse-colour-approximation', native_palette=102)], ['scroll'])

    native4, native6 = native_clip(args.original, 4), native_clip(args.original, 6)
    solbox, soledge = body_box(native4[-1]); aresbox, aresedge = body_box(native6[-1])
    sol, ares = alpha_crop(master_images['sol']), alpha_crop(master_images['ares'])
    for resource, native, master, box, edge, kind in ((4, native4, sol, solbox, soledge, 'sol'), (6, native6, ares, aresbox, aresedge, 'ares')):
        pictures = static_pose_frames(native, master, box, edge)
        add_bank('ANI.DAT', resource, pictures,
                 [dict(method='native-reveal-blue-white-colour', frame=i, anchor=box, background_edge=edge) for i in range(len(pictures))], [kind])
    native8 = native_clip(args.original, 8)
    pictures = []
    for frame in native8:
        canvas = pose_background(frame, 0)
        place(canvas, ares, aresbox); place(canvas, sol, solbox)
        pictures.append(canvas)
    add_bank('ANI.DAT', 8, pictures, [dict(method='native-paired-pose', frame=i, sol=solbox, ares=aresbox) for i in range(12)], ['sol', 'ares'])

    guardian = master_images['guardian'].resize((1920, 1200), Image.Resampling.LANCZOS)
    native3 = native_clip(args.original, 3)
    pictures = []
    for frame in native3:
        # The source introduces only the closed blue face. Other armour stays
        # a black silhouette until the separate FDOTHER100 full-colour cut.
        canvas = pose_background(frame, 0)
        silhouette = Image.new('RGBA', guardian.size, 'black')
        silhouette.putalpha(guardian.getchannel('A'))
        canvas.alpha_composite(silhouette)
        face = blue_pose(guardian)
        alpha = face.getchannel('A')
        gate = Image.new('L', face.size)
        ImageDraw.Draw(gate).ellipse((140*6, 49*6, 180*6, 86*6), fill=255)
        face.putalpha(ImageChops.multiply(alpha, gate))
        canvas.alpha_composite(face)
        pictures.append(canvas)
    add_bank('ANI.DAT', 3, pictures, [dict(method='native-dark-armour-blue-face-art-mask', frame=i) for i in range(28)], ['guardian'])

    def admitted_picture(role, resource, frame=0, archive='FIGANI.DAT'):
        asset = next(a for a in admitted if a['role'] == role and a['source_archive'] == archive and a['source_resource'] == resource)
        name = asset['files'][frame] if 'files' in asset else asset['file']
        expected = asset['frame_sha256'][frame] if 'files' in asset else asset['sha256']
        assert sha(args.modern/name) == expected
        return Image.open(args.modern/name).convert('RGBA'), name, expected

    for resource, figures, enemies, bg, bgx, platform, py in ((5, (96, 98), (312, 313, 314), 14, 69, 14, 157), (7, (108, 109), (366,), 8, 0, 8, 158)):
        native = native_clip(args.original, resource)
        candidates = sprite_candidates(args.original, figures)
        enemy_candidates = sprite_candidates(args.original, enemies)
        background, bgname, bgsha = admitted_picture('battle_surface_hd', bg, archive='BG.DAT')
        stage, stagename, stagesha = admitted_picture('battle_surface_hd', platform, archive='TAI.DAT')
        pictures, methods = [], []
        for i, frame in enumerate(native):
            if resource == 5 and i == 30:
                canvas = Image.new('RGBA', (1920, 1200), 'black')
                canvas.alpha_composite(master_images['beam30'].resize((1920, 1200), Image.Resampling.LANCZOS))
                methods.append(dict(method='direct-source-frame-guided-keyframe-redraw', frame=i, figures=[],
                                    native_frame_sha256=hashlib.sha256(frame.tobytes()).hexdigest(),
                                    master=master_names['beam30'],
                                    note='吐息大幅遮擋的過渡幀直接依原影格重繪，不猜補其內部分層或原生透明規則。'))
                pictures.append(canvas)
                continue
            placements = locate_figures(frame, candidates, dragon=resource == 5)
            if not placements:
                raise ValueError(f'ANI{resource}第{i}幀沒有受來源支持的人物')
            canvas = Image.new('RGBA', (1920, 1200), 'black')
            canvas.alpha_composite(background, (bgx*6, 50*6))
            canvas.alpha_composite(stage, (164*6, py*6))
            foreground = located_coverage(placements, candidates)
            body_placements = [p for p in placements if not (resource == 5 and p['resource'] == 98 and p['frame'] >= 9)]
            body_foreground = located_coverage(body_placements, candidates)
            luminous = {n for n, value in enumerate(frame.tobytes()) if value in range(80, 88) or value in range(192, 200)} if resource == 5 else set()
            enemy = locate_enemy(frame, enemy_candidates, (0, 0) if resource == 5 else (7, 2), foreground | luminous)
            placements = [enemy] + placements
            if resource == 5:
                # The intro adds a narrow hot region that is absent from the
                # reusable BG14 art. Preserve its literal observed coverage;
                # no new environment or game-side palette rule is invented.
                glow = Image.new('RGBA', (320, 200))
                colours = frame.convert('RGB').tobytes()
                rgba = [(*colours[n*3:n*3+3], 255) if n in luminous and n not in body_foreground else (0, 0, 0, 0)
                        for n in range(320*200)]
                # The old actor silhouette must not leave a cave-shaped hole
                # beside the newly drawn actor. Extend each observed wide beam
                # row only under the old actor, then let the HD alpha occlude it.
                filled = 0
                for y in range(200):
                    xs = [x for x in range(320) if rgba[y*320+x][3]]
                    if len(xs) < 120:
                        continue
                    colour = Counter(rgba[y*320+x] for x in xs).most_common(1)[0][0]
                    for x in range(min(xs), max(xs)+1):
                        n = y*320+x
                        if n in body_foreground:
                            rgba[n] = colour
                            filled += 1
                glow.putdata(rgba)
                luminous_layer = glow.resize((1920, 1200), Image.Resampling.LANCZOS)
            else:
                luminous_layer = None
            for match in placements:
                if resource == 5 and match['resource'] == 98 and match['frame'] >= 9:
                    match['render_method'] = 'native-luminous-coverage-hd6-interpolation'
                    continue
                pic, filename, expected = admitted_picture('battle_animation_hd_set', match['resource'], match['frame'])
                canvas.alpha_composite(pic, (match['x']*6, match['y']*6))
                match['hd_file'], match['hd_sha256'] = filename, expected
                if match is enemy and luminous_layer is not None:
                    canvas.alpha_composite(luminous_layer)
            methods.append(dict(method='native-pixel-supported-figani-placement-art-adaptation', frame=i,
                                background=dict(file=bgname, sha256=bgsha, x=bgx, y=50),
                                platform=dict(file=stagename, sha256=stagesha, x=164, y=py), figures=placements,
                                luminous_old_actor_coverage_filled=filled if resource == 5 else 0,
                                source_colour31_note='僅記錄ANI5局部背景差異；不推導或修改FIGANI正式透明規則。'))
            pictures.append(canvas)
        # Source admission includes the already catalogued HD components too.
        component_names = sorted({v['hd_file'] for row in methods for v in row['figures'] if 'hd_file' in v} | {bgname, stagename})
        for name in component_names:
            shutil.copy2(args.modern/name, args.out/name)
        kind = f'combat{resource}'
        master_names[kind] = component_names[0]
        add_bank('ANI.DAT', resource, pictures, methods, [kind])
        entry = fragments[-1]
        # Every component is explicit allow-list provenance, not just the first.
        plan_name = entry['source_files'][-1]
        entry['source_files'] = component_names + [plan_name]
        if resource == 5:
            entry['source_files'].insert(-1, master_names['beam30'])
        entry['source_sha256'] = [sha(args.out/name) for name in entry['source_files']]

    native1 = native_clip(args.original, 1)
    number = alpha_crop(master_images['number2'])
    title = master_images['title'].resize((1920, 1200), Image.Resampling.LANCZOS)
    pictures, methods = [], []
    for i, frame in enumerate(native1):
        rgb = frame.convert('RGB')
        mask = Image.new('L', (320, 200))
        mask.putdata([255 if b > max(r, g) and b > 20 else 0 for r, g, b in rgb.getdata()])
        box = mask.getbbox()
        canvas = Image.new('RGBA', (1920, 1200), 'black')
        # Red wordmark is revealed by the source. It is deliberately kept
        # literal, with interpolated outlines, beneath the new metal master.
        red = Image.new('RGBA', (320, 200))
        red.putdata([(r, g, b, 255) if r > max(g, b)*2 and r > 0 else (0, 0, 0, 0) for r, g, b in rgb.getdata()])
        canvas.alpha_composite(red.resize((1920, 1200), Image.Resampling.LANCZOS))
        if box:
            place(canvas, number, box)
        pictures.append(canvas)
        methods.append(dict(method='native-number-visible-bounds-art-scaling', frame=i, box=box,
                            approximation='原可見輪廓框的2D縮放；不是原版3D逐面投影。'))
    add_bank('ANI.DAT', 1, pictures, methods, ['number2', 'title'])
    dump(args.out/'catalog-fragment.json', fragments)
    dump(args.out/'build-receipt.json', dict(status='DRAFT-COMPILED-NOT-ADMITTED', compiler_sha256=sha(Path(__file__)),
         catalog_input_sha256=sha(args.catalog), bank_count=len(fragments), frame_count=sum(a['frame_count'] for a in fragments),
         anchors=dict(sol=solbox, ares=aresbox), outputs={p.name: sha(p) for p in sorted(args.out.iterdir()) if p.is_file()}))
    print(json.dumps(dict(banks=len(fragments), frames=sum(a['frame_count'] for a in fragments), anchors=dict(sol=solbox, ares=aresbox)), ensure_ascii=False))


if __name__ == '__main__':
    main()
