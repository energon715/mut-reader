#!/usr/bin/env python3
"""Own-content, deterministic fixtures; no downloaded books."""
import pathlib, sys, zipfile
root=pathlib.Path(sys.argv[1]); root.mkdir(parents=True,exist_ok=True)
(root/'book.fb2').write_text('<FictionBook><description><title-info><book-title>Native FB2 test</book-title></title-info></description><body><section><title><p>Native chapter</p></title><p>Author-owned FB2 release fixture.</p></section></body></FictionBook>')
with zipfile.ZipFile(root/'book.epub','w') as z:
 z.writestr('META-INF/container.xml','<container><rootfiles><rootfile full-path="package.opf"/></rootfiles></container>')
 z.writestr('package.opf','<package><metadata><title>Native EPUB test</title></metadata><manifest><item id="one" href="one.xhtml" media-type="application/xhtml+xml"/></manifest><spine><itemref idref="one"/></spine></package>')
 z.writestr('one.xhtml','<html><head><meta charset="utf-8"/><script>window.shouldNotRun=true</script></head><body><h1>Native EPUB chapter</h1><p>Author-owned EPUB release fixture.</p><img src="https://example.test/tracking.png"/></body></html>')
objects=[b'<< /Type /Catalog /Pages 2 0 R >>', b'<< /Type /Pages /Kids [3 0 R 5 0 R] /Count 2 >>']
for page in ['one','two']:
 contents=f'BT /F1 24 Tf 60 700 Td (Native PDF test page {page}) Tj ET'.encode()
 objects.extend([f'<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << /Font << /F1 7 0 R >> >> /Contents {len(objects)+2} 0 R >>'.encode(), b'<< /Length '+str(len(contents)).encode()+b' >>\nstream\n'+contents+b'\nendstream'])
objects.append(b'<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>')
data=b'%PDF-1.4\n'; offsets=[0]
for i,obj in enumerate(objects,1):
 offsets.append(len(data)); data+=f'{i} 0 obj\n'.encode()+obj+b'\nendobj\n'
xref=len(data); data+=f'xref\n0 {len(offsets)}\n0000000000 65535 f \n'.encode()
for offset in offsets[1:]: data+=f'{offset:010} 00000 n \n'.encode()
data+=f'trailer\n<< /Size {len(offsets)} /Root 1 0 R >>\nstartxref\n{xref}\n%%EOF\n'.encode()
(root/'book.pdf').write_bytes(data)
# Several text pages, each taller than a desktop viewport, for scroll regressions.
with zipfile.ZipFile(root/'long.epub','w') as z:
 z.writestr('META-INF/container.xml','<container><rootfiles><rootfile full-path="package.opf"/></rootfiles></container>')
 z.writestr('package.opf','<package><metadata><title>Long reader fixture</title></metadata><manifest><item id="one" href="one.xhtml"/></manifest><spine><itemref idref="one"/></spine></package>')
 z.writestr('one.xhtml','<html><body><h1>Long reader chapter</h1>'+''.join('<p>Paragraph '+str(i)+'. '+('Author-owned long reading fixture. '*10)+'</p>' for i in range(80))+'</body></html>')
