#!/usr/bin/env python3
"""Write release provenance/instructions and SHA-256 for all uploadable files."""
import hashlib,json,pathlib,subprocess
root=pathlib.Path(__file__).resolve().parent.parent
out=root/'dist'
version=json.loads((root/'frontend/package.json').read_text())['version']
notes=(root/'packaging/release-notes.ru.md').read_text().replace('1.0.0-rc.1',version)
(out/'RELEASE-NOTES.md').write_text(notes)
def command(*argv):return subprocess.check_output(argv,cwd=root,text=True).strip()
info={'version':version,'architecture':'x86_64 / amd64','buildOS':pathlib.Path('/etc/os-release').read_text(),'go':command('go','version'),'node':command('node','--version'),'wails':'v2.16.0','minimumAppImageGlibc':'2.39','minimumSystemWebKit':'2.54.0','gitCommit':command('git','rev-parse','HEAD'),'goBuildInfo':command('go','version','-m','build/bin/mut'),'toolInputs':json.loads((root/'packaging/tools.lock.json').read_text()),'bundledPackages':json.loads((root/'build/appimage/mut.AppDir/usr/share/doc/mut/licenses/system/packages.json').read_text())}
(out/'BUILD-INFO.json').write_text(json.dumps(info,indent=2,ensure_ascii=False)+'\n')
files=sorted(p for p in out.iterdir() if p.name.startswith(f'mut-{version}-') or p.name in ['RELEASE-NOTES.md','BUILD-INFO.json'])
checksums=[]
for p in files:
 with p.open('rb') as f:checksums.append(hashlib.file_digest(f,'sha256').hexdigest()+'  '+p.name+'\n')
(out/'SHA256SUMS').write_text(''.join(checksums))
print('Ready to upload:',*[p.name for p in files],'SHA256SUMS',sep='\n')
