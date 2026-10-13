#!/usr/bin/env python3
"""Package already-built Linux binaries. No root or dpkg installation required."""
import hashlib, io, json, pathlib, shutil, subprocess, tarfile, tempfile
root=pathlib.Path(__file__).resolve().parent.parent
version=json.loads((root/'frontend/package.json').read_text())['version']
out=root/'dist';out.mkdir(exist_ok=True)
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
 for src in ['LICENSE','README.md','CHANGELOG.md','docs/RELEASE.md','docs/RECOVERY.md','spec/format-v1.md']:
  shutil.copy2(root/src,doc/pathlib.Path(src).name)
 licenses=doc/'licenses';licenses.mkdir()
 for src in (root/'frontend/public/licenses').glob('*'):shutil.copy2(src,licenses/src.name)
 if (root/'build/licenses').exists():shutil.copytree(root/'build/licenses',licenses/'go')
 shutil.copy2(root/'go.mod',doc/'go.mod');shutil.copy2(root/'frontend/package-lock.json',doc/'frontend-package-lock.json')
 icon=share/'icons/hicolor/256x256/apps';icon.mkdir(parents=True);shutil.copy2(root/'frontend/src/assets/mut-logo.png',icon/'mut.png')
 apps=share/'applications';apps.mkdir(parents=True)
 (apps/'mut.desktop').write_text('[Desktop Entry]\nType=Application\nName=mut\nComment=Local encrypted book library\nExec=mut\nIcon=mut\nTerminal=false\nCategories=Office;Viewer;\nStartupWMClass=mut\n')
 archive=out/f'mut-{version}-linux-amd64.tar.gz'
 with tarfile.open(archive,'w:gz') as t:
  for p in sorted(stage.rglob('*')):t.add(p,arcname=str(p.relative_to(stage)),recursive=False)
 data=io.BytesIO()
 with tarfile.open(fileobj=data,mode='w:gz') as t:
  for p in sorted(stage.rglob('*')):t.add(p,arcname='./'+str(p.relative_to(stage)),recursive=False)
 control=f'Package: mut\nVersion: {version.replace("-rc.","~rc") }\nArchitecture: amd64\nMaintainer: mut project\nSection: utils\nPriority: optional\nDepends: libgtk-3-0, libwebkit2gtk-4.1-0\nDescription: Offline encrypted personal book library\n Includes PDF, EPUB, FB2 readers and vaultctl recovery.\n'.encode()
 ctl=io.BytesIO()
 with tarfile.open(fileobj=ctl,mode='w:gz') as t:
  i=tarfile.TarInfo('./control');i.size=len(control);i.mode=0o644;t.addfile(i,io.BytesIO(control))
 deb=out/f'mut-{version}-amd64.deb'
 with deb.open('wb') as f:
  f.write(b'!<arch>\n')
  for name,content in [('debian-binary',b'2.0\n'),('control.tar.gz',ctl.getvalue()),('data.tar.gz',data.getvalue())]:
   header=f'{name+"/":<16}{0:<12}{0:<6}{0:<6}{"100644":<8}{len(content):<10}`\n'.encode();assert len(header)==60;f.write(header);f.write(content)
   if len(content)%2:f.write(b'\n')
 artifacts=[archive,deb]
 (out/'SHA256SUMS').write_text(''.join(hashlib.sha256(p.read_bytes()).hexdigest()+'  '+p.name+'\n' for p in artifacts))
 print('\n'.join(str(p) for p in artifacts))
