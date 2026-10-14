#!/usr/bin/env python3
"""Assemble matching WebKit/GTK processes and runtime modules in an AppDir."""
import hashlib,json,os,pathlib,shutil,subprocess
root=pathlib.Path(__file__).resolve().parent.parent
work=root/'build/appimage'; app=work/'mut.AppDir'; tools=root/'build/release-tools'
version=json.loads((root/'frontend/package.json').read_text())['version']
work.mkdir(parents=True,exist_ok=True)
if app.exists(): shutil.rmtree(app)
# Share the exact desktop payload, documentation and binaries with the DEB.
shutil.copytree(root/'build/package-stage',app)

def extract_tool(name):
    directory=work/name
    if not (directory/'squashfs-root/AppRun').exists():
        directory.mkdir(exist_ok=True)
        subprocess.run([str(tools/(name+'-x86_64.AppImage')),'--appimage-extract'],cwd=directory,stdout=subprocess.DEVNULL,check=True)
    return directory/'squashfs-root/AppRun'

origins={}
def copy(source,dest):
    source=pathlib.Path(source); dest=app/dest; dest.parent.mkdir(parents=True,exist_ok=True)
    if source.is_dir():
        shutil.copytree(source,dest,dirs_exist_ok=True,symlinks=False)
        for original in source.rglob('*'):
            if original.is_file(): origins[str(dest/original.relative_to(source))]=str(original.resolve())
    else:
        shutil.copy2(source,dest); origins[str(dest)]=str(source.resolve())

multi=pathlib.Path('/usr/lib/x86_64-linux-gnu')
webkit=multi/'webkit2gtk-4.1'
# ALSA is on linuxdeploy's generic exclusion list, but isn't present in minimal desktops.
for name in ['libasound.so.2','libgpg-error.so.0']:
    copy(multi/name,'usr/lib/'+name)
copy('/usr/share/alsa','usr/share/alsa')
copy(webkit,'usr/lib/x86_64-linux-gnu/webkit2gtk-4.1')
copy(webkit/'injected-bundle','usr/lib/webkit2gtk-4.1/injected-bundle')
copy(multi/'gio/modules','usr/lib/gio/modules')
copy(multi/'gdk-pixbuf-2.0/2.10.0/loaders','usr/lib/gdk-pixbuf-2.0/loaders')
copy(multi/'gstreamer-1.0','usr/lib/gstreamer-1.0')
copy(multi/'gstreamer1.0/gstreamer-1.0/gst-plugin-scanner','usr/libexec/gst-plugin-scanner')
for directory in ['glib-2.0/schemas','mime','icons/Adwaita','icons/hicolor','webkitgtk-4.1']:
    source=pathlib.Path('/usr/share')/directory
    if source.exists(): copy(source,'usr/share/'+directory)
# linuxdeploy finds each ELF dependency recursively; explicitly include dlopen modules.
elf=[]
for p in app.rglob('*'):
    if p.is_file() and not p.is_symlink():
        with p.open('rb') as f:
            if f.read(4)==b'\x7fELF': elf.append(p)
command=[str(extract_tool('linuxdeploy')),'--appdir',str(app),'--desktop-file',str(root/'packaging/mut.desktop'),'--icon-file',str(root/'frontend/src/assets/mut-logo.png')]
# The original asset name differs from Icon=mut: stage a named copy.
icon=work/'mut.png';subprocess.run(['gdk-pixbuf-thumbnailer','-s','256',str(root/'frontend/src/assets/mut-logo.png'),str(icon)],check=True)
command[-1]=str(icon)
for excluded in ['libwayland-*','libdrm*','libgbm*','libEGL*','libGL*','libGLES*','libvulkan*','libudev*']:
    command += ['--exclude-library',excluded]
for p in elf:
    command += ['--executable' if os.access(p,os.X_OK) and '.so' not in p.name else '--library',str(p)]
subprocess.run(command,check=True,env=dict(os.environ,NO_STRIP='1'))
# A small GLib spawn hook relocates absolute helper paths without modifying WebKit.
subprocess.run(['cc','-O2','-fPIC','-shared','-Wl,-z,relro,-z,now','-Wall','-Wextra','-Werror',str(root/'packaging/webkit-relocate.c'),'-ldl','-o',str(app/'usr/lib/libmut-webkit-relocate.so')],check=True)
(app/'AppRun').unlink(missing_ok=True)
shutil.copy2(root/'packaging/AppRun',app/'AppRun');(app/'AppRun').chmod(0o755)
subprocess.run(['glib-compile-schemas',str(app/'usr/share/glib-2.0/schemas')],check=True)
subprocess.run(['gio-querymodules',str(app/'usr/lib/gio/modules')],check=True)
# Generate a relocatable pixbuf cache (module paths relative to MODULEDIR).
loaders=app/'usr/lib/gdk-pixbuf-2.0/loaders'
cache=subprocess.check_output([str(multi/'gdk-pixbuf-2.0/gdk-pixbuf-query-loaders'),*[str(x) for x in loaders.glob('*.so')]],text=True)
cache=cache.replace(str(loaders)+'/', '')
(app/'usr/lib/gdk-pixbuf-2.0/loaders.cache').write_text(cache)
# Track libraries included in the image, their Debian source packages and notices.
licenses=app/'usr/share/doc/mut/licenses/system';licenses.mkdir(parents=True,exist_ok=True)
packages={}
# The dpkg file index includes dynamically loaded modules and copied data files.
owners={}
for listing in pathlib.Path('/var/lib/dpkg/info').glob('*.list'):
    pkg=listing.name.removesuffix('.list')
    for path in listing.read_text().splitlines(): owners[path]=pkg
for p in list(app.rglob('*')):
    if not p.is_file():continue
    candidates=[multi/p.name, pathlib.Path('/usr/bin')/p.name]
    if str(p) in origins: candidates.insert(0,pathlib.Path(origins[str(p)]))
    for src in candidates:
        if not src.exists():continue
        pkg=owners.get(str(src)) or owners.get(str(src.resolve()))
        if not pkg or pkg.split(':')[0]=='mut':continue
        if pkg in packages:break
        meta=subprocess.check_output(['dpkg-query','-W','-f=${Package}\t${Version}\t${source:Package}\t${source:Version}\n',pkg],text=True).strip().split('\t')
        packages[pkg]=dict(zip(['package','version','source','sourceVersion'],meta))
        copyright=pathlib.Path('/usr/share/doc')/pkg.split(':')[0]/'copyright'
        if copyright.exists():copy(copyright,'usr/share/doc/mut/licenses/system/'+pkg.replace(':','_')+'-copyright')
        break
(licenses/'packages.json').write_text(json.dumps(list(packages.values()),indent=2)+'\n')
shutil.copy2(root/'scripts/package-appimage.py',licenses/'package-appimage.py')
shutil.copy2(root/'packaging/webkit-relocate.c',licenses/'webkit-relocate.c')
shutil.copy2(root/'packaging/tools.lock.json',licenses/'tools.lock.json')
(licenses/'README.txt').write_text('Bundled runtime libraries are distribution binaries (RPATH relocation only).\nDebian copyright notices and exact source/version inventory are included. Corresponding Debian sources: https://sources.debian.org/ and https://snapshot.debian.org/package/\nShared libraries remain replaceable after --appimage-extract (LGPL relinking).\n')
out=root/'dist'/f'mut-{version}-linux-x86_64.AppImage'
env=os.environ.copy();env['ARCH']='x86_64'
subprocess.run([str(extract_tool('appimagetool')),'--runtime-file',str(tools/'runtime-x86_64'),str(app),str(out)],env=env,check=True)
# Add the AppImage to the existing package checksum list.
files=sorted(p for p in (root/'dist').iterdir() if p.suffix in ['.deb','.AppImage'] or p.name.endswith('.tar.gz'))
(root/'dist/SHA256SUMS').write_text(''.join(hashlib.sha256(p.read_bytes()).hexdigest()+'  '+p.name+'\n' for p in files))
print(out)
