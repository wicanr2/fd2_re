#!/usr/bin/env python3
"""以唯讀來源連結組合隔離執行素材，不複製私人包或修改工作樹素材。"""
import argparse
from pathlib import Path


def stage(project, original, catalog, out):
    assert Path('/.dockerenv').exists(), 'Docker required'
    assert project.is_dir() and (original / 'manifest.json').is_file() and catalog.is_file()
    assert not out.exists() and out.parent.is_dir()
    out.mkdir()
    catalog_parts = ('themes', 'modern', 'catalog.json')

    def merge(target, left, right, relative=()):
        if relative == catalog_parts:
            target.symlink_to(catalog.resolve())
            return
        force = catalog_parts[:len(relative)] == relative
        if force or left and left.is_dir() and right and right.is_dir():
            target.mkdir()
            names = set()
            for source in (left, right):
                if source and source.is_dir():
                    names.update(p.name for p in source.iterdir())
            if force:
                names.add(catalog_parts[len(relative)])
            for name in sorted(names):
                l = left / name if left else None
                r = right / name if right else None
                merge(target / name, l if l and l.exists() else None,
                      r if r and r.exists() else None, (*relative, name))
        else:
            source = right or left
            assert source is not None and source.exists(), relative
            target.symlink_to(source.resolve())

    merge(out / 'assets', original, project)
    for name in ('maps/map0/map.json', 'animations/FDOTHER_065/animation.json',
                 'ui/fdother_005_item_panel/resource.json', 'themes/modern/catalog.json'):
        assert (out / 'assets' / name).is_file(), name
    print('隔離素材覆蓋已建立：' + str(out))


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ('project-assets', 'original-pack', 'catalog', 'out'):
        parser.add_argument('--' + name, type=lambda value: Path(value).resolve(), required=True)
    a = parser.parse_args()
    stage(a.project_assets, a.original_pack, a.catalog, a.out)


if __name__ == '__main__':
    main()
