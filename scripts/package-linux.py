#!/usr/bin/env python3
"""Package already-built Linux binaries. No root or dpkg installation required."""
import gzip, hashlib, io, json, os, pathlib, shutil, subprocess, tarfile, tempfile
root=pathlib.Path(__file__).resolve().parent.parent
version=json.loads((root/'frontend/package.json').read_text())['version']
out=root/'dist';out.mkdir(exist_ok=True)
epoch=int(os.environ.get('SOURCE_DATE_EPOCH',subprocess.check_output(['git','log','-1','--format=%ct'],cwd=root,text=True).strip()))
def archive_bytes(stage):
 data=io.BytesIO()
 with gzip.GzipFile(fileobj=data,mode='wb',mtime=epoch,filename='') as gz,tarfile.open(fileobj=gz,mode='w') as t:
  for p in sorted(stage.rglob('*')):
   i=t.gettarinfo(p,arcname='./'+str(p.relative_to(stage)));i.uid=i.gid=0;i.uname=i.gname='root';i.mtime=epoch
   with p.open('rb') if p.is_file() else io.BytesIO() as source:t.addfile(i,source if p.is_file() else None)
 return data.getvalue()
for name in ['mut','vaultctl']:
 if not (root/'build/bin'/name).is_file(): raise SystemExit('Build '+name+' before packaging')
# Check linked-library availability on the build host; other systems need smoke tests.
linked=subprocess.check_output(['ldd',str(root/'build/bin/mut')],text=True)
if 'not found' in linked: raise SystemExit('Missing shared library: '+linked)
with tempfile.TemporaryDirectory(prefix='mut-package-') as temp:
 stage=pathlib.Path(temp)
 for name in ['mut','vaultctl']:
  dst=stage/'usr/bin'/name;dst.parent.mkdir(parents=True,exist_ok=True);shutil.copy2(root/'build/bin'/name,dst);dst.chmod(0o755)
 share=stage/'usr/share';doc=share/'doc/mut';doc.mkdir(parents=True)
 for src in ['LICENSE','README.md','CHANGELOG.md','docs/RELEASE.md','docs/RECOVERY.md','docs/LINUX_PACKAGES.md','docs/SECURITY_REVIEW_2026-10-14.md','spec/format-v1.md']:
  shutil.copy2(root/src,doc/pathlib.Path(src).name)
 licenses=doc/'licenses';licenses.mkdir()
 for src in (root/'frontend/public/licenses').glob('*'):shutil.copy2(src,licenses/src.name)
 if (root/'build/licenses').exists():shutil.copytree(root/'build/licenses',licenses/'go')
 shutil.copy2(root/'go.mod',doc/'go.mod');shutil.copy2(root/'frontend/package-lock.json',doc/'frontend-package-lock.json')
 icon=share/'icons/hicolor/256x256/apps';icon.mkdir(parents=True);subprocess.run(['gdk-pixbuf-thumbnailer','-s','256',str(root/'frontend/src/assets/mut-logo.png'),str(icon/'mut.png')],check=True) if shutil.which('gdk-pixbuf-thumbnailer') else shutil.copy2(root/'frontend/src/assets/mut-logo.png',icon/'mut.png')
 apps=share/'applications';apps.mkdir(parents=True)
 shutil.copy2(root/'packaging/mut.desktop',apps/'mut.desktop')
 if shutil.which('dpkg-shlibdeps'):
  debian=stage/'debian';debian.mkdir();(debian/'control').write_text('Source: mut\nSection: utils\nPriority: optional\nMaintainer: mut project\nStandards-Version: 4.7.0\n\nPackage: mut\nArchitecture: amd64\nDescription: Encrypted book library\n')
  deps=subprocess.check_output(['dpkg-shlibdeps','-O',str(stage/'usr/bin/mut')],cwd=stage,text=True).strip().removeprefix('shlibs:Depends=')
  shutil.rmtree(debian)
  deps=deps.replace('libwebkit2gtk-4.1-0 (>= 2.40.0)','libwebkit2gtk-4.1-0 (>= 2.54.0)')
  import re
  deps=re.sub(r'libwebkit2gtk-4.1-0(?: \(>= [^)]+\))?', 'libwebkit2gtk-4.1-0 (>= 2.54.0)', deps)
 else: deps='libc6 (>= 2.39), libgtk-3-0t64, libwebkit2gtk-4.1-0 (>= 2.54.0)'
 final_stage=root/'build/package-stage'
 if final_stage.exists():shutil.rmtree(final_stage)
 shutil.copytree(stage,final_stage)
 archive=out/f'mut-{version}-linux-amd64.tar.gz'
 data=archive_bytes(stage);archive.write_bytes(data)
 control=f'Package: mut\nVersion: {version.replace("-rc.","~rc") }\nArchitecture: amd64\nMaintainer: mut project\nSection: utils\nPriority: optional\nDepends: {deps}\nDescription: Offline encrypted personal book library\n Includes PDF, EPUB, FB2 readers and vaultctl recovery.\n'.encode()
 ctl=io.BytesIO()
 with gzip.GzipFile(fileobj=ctl,mode='wb',mtime=epoch,filename='') as gz,tarfile.open(fileobj=gz,mode='w') as t:
  i=tarfile.TarInfo('./control');i.size=len(control);i.mode=0o644;i.mtime=epoch;i.uname=i.gname='root';t.addfile(i,io.BytesIO(control))
 deb=out/f'mut-{version}-amd64.deb'
 with deb.open('wb') as f:
  f.write(b'!<arch>\n')
  for name,content in [('debian-binary',b'2.0\n'),('control.tar.gz',ctl.getvalue()),('data.tar.gz',data)]:
   header=f'{name+"/":<16}{0:<12}{0:<6}{0:<6}{"100644":<8}{len(content):<10}`\n'.encode();assert len(header)==60;f.write(header);f.write(content)
   if len(content)%2:f.write(b'\n')
 artifacts=[archive,deb]
 (out/'SHA256SUMS').write_text(''.join(hashlib.sha256(p.read_bytes()).hexdigest()+'  '+p.name+'\n' for p in artifacts))
 print('\n'.join(str(p) for p in artifacts))
