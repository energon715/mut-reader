#!/usr/bin/env python3
"""Build Linux x86-64 releases in a disposable Debian 13 user namespace."""
import argparse, os, pathlib, platform, shutil, subprocess, tarfile
root = pathlib.Path(__file__).resolve().parent.parent
cache = root / 'build/release-tools'
fs = root / 'build/.release-debian-rootfs'

def unpack(archive, target, strip=0):
    target.mkdir(parents=True, exist_ok=True)
    with tarfile.open(archive) as t:
        members = []
        for m in t.getmembers():
            parts = pathlib.PurePosixPath(m.name).parts
            if m.name.startswith('/') or '..' in parts:
                raise ValueError('Unsafe archive path: ' + m.name)
            if len(parts) <= strip or m.isdev() or m.isfifo():
                continue
            m.name = str(pathlib.PurePosixPath(*parts[strip:]))
            if m.issym():
                dest = target / m.name
                link = m.linkname
                absolute = (target / link.lstrip('/')) if link.startswith('/') else dest.parent / link
                normal = pathlib.Path(os.path.normpath(absolute))
                if not normal.is_relative_to(target):
                    raise ValueError('Unsafe archive link: ' + m.name)
                m.linkname = os.path.relpath(normal, dest.parent)
            elif m.islnk():
                linkparts = pathlib.PurePosixPath(m.linkname).parts
                if '..' in linkparts or m.linkname.startswith('/'):
                    raise ValueError('Unsafe hard link')
                m.linkname = str(pathlib.PurePosixPath(*linkparts[strip:]))
            m.uid = os.getuid(); m.gid = os.getgid(); m.uname = ''; m.gname = ''
            members.append(m)
        # Archive symlinks cannot redirect regular-file writes outside the target.
        for m in sorted(members, key=lambda x: x.issym() or x.islnk()):
            t.extract(m, target, set_attrs=True, numeric_owner=True, filter='fully_trusted')

def run(command):
    return subprocess.run(['unshare', '--user', '--map-auto', '--map-root-user', 'bwrap', '--uid', '0', '--gid', '0',
        '--unshare-pid', '--bind', str(fs), '/', '--proc', '/proc', '--dev', '/dev',
        '--bind', str(root), '/work', '--ro-bind', '/etc/resolv.conf', '/etc/resolv.conf',
        '--chdir', '/work', '--setenv', 'PATH', '/opt/go/bin:/opt/node/bin:/opt/gobin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin',
        '--setenv', 'GOPATH', '/opt/gopath', '--setenv', 'GOCACHE', '/opt/gocache',
        '--setenv', 'GOBIN', '/opt/gobin', '--setenv', 'SOURCE_DATE_EPOCH', subprocess.check_output(['git','log','-1','--format=%ct'],cwd=root,text=True).strip(),
        '--', '/bin/bash', '-euc', command], cwd=root, check=True)

if __name__ == '__main__':
    parser = argparse.ArgumentParser(); parser.add_argument('--setup-only', action='store_true'); parser.add_argument('--command')
    args = parser.parse_args()
    if platform.system()!='Linux' or platform.machine() not in ['x86_64','amd64']:raise SystemExit('Build these packages on a Linux x86-64 host')
    if not shutil.which('bwrap'): raise SystemExit('Install bubblewrap to use the rootless Debian build environment')
    subprocess.run(['python3', str(root/'scripts/fetch-release-tools.py')], check=True)
    if not (fs/'etc/os-release').exists():
        unpack(cache/'debian-trixie-slim.tar.gz', fs)
        (fs/'usr/sbin/policy-rc.d').write_text('#!/bin/sh\nexit 101\n'); (fs/'usr/sbin/policy-rc.d').chmod(0o755)
    if not (fs/'opt/go/bin/go').exists(): unpack(cache/'go1.27.2.linux-amd64.tar.gz', fs/'opt')
    if not (fs/'opt/node/bin/node').exists(): unpack(cache/'node-v24.21.0-linux-x64.tar.xz', fs/'opt/node', strip=1)
    if args.command:
        run(args.command)
    else:
        run('apt-get -o APT::Sandbox::User=root update; apt-get -o APT::Sandbox::User=root install -y --no-install-recommends ca-certificates build-essential pkg-config git python3 file binutils patchelf desktop-file-utils squashfs-tools libgtk-3-dev libwebkit2gtk-4.1-dev libglib2.0-bin dpkg-dev libfuse2 fonts-dejavu-core')
        if not args.setup_only: run('bash scripts/build-linux-inside.sh')
