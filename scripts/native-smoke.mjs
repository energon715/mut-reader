import {connect} from './webkit-client.mjs';
import {mkdir,writeFile} from 'node:fs/promises';
import assert from 'node:assert/strict';
const base=process.argv[2];
if (!base?.startsWith('/tmp/mut-')) throw Error('Use a dedicated /tmp/mut-* fixture directory');
const c=await connect(); const q=JSON.stringify;
const old=await c.evaluate('localStorage.getItem("mut.paths")');
const fill=async(selector,value)=>c.evaluate(`(()=>{const e=document.querySelector(${q(selector)});if(!e)throw Error('Missing input');e.value=${q(value)};e.dispatchEvent(new Event('input',{bubbles:true}));e.dispatchEvent(new Event('change',{bubbles:true}));})()`);
const click=async(text)=>c.evaluate(`(()=>{const e=[...document.querySelectorAll('button')].filter(e=>e.textContent.trim()===${q(text)}).at(-1);if(!e)throw Error('Missing button: '+${q(text)});e.click()})()`);
const has=text=>`document.body.innerText.includes(${q(text)})`;
const api=(method,args=[])=>c.asyncEvaluate(`window.go.app.App.${method}(${args.map(q).join(',')})`);
const vault=base+'/vault-'+Date.now();
let ownsTestSession=false;
try {
 const defaults=await api("GetDefaultPaths");
 assert(defaults.library.startsWith(base+"/"),"Inspector must target the isolated test build");
 ownsTestSession=true;
 if(await c.evaluate('!!document.querySelector(".login-page")')) await click('Создать библиотеку');
 await c.wait('document.querySelectorAll("input[type=password]").length===2');
 await fill('input:not([type=password])',vault);
 await fill('input[type=password]','native-test-password');
 await fill('input[type=password]:last-of-type','native-test-password');
 // The passwords live in different labels; explicitly address both inputs.
 await c.evaluate(`document.querySelectorAll('input[type=password]').forEach(e=>{e.value='native-test-password';e.dispatchEvent(new Event('input',{bubbles:true}))})`);
 await click('Продолжить');
 await c.wait('document.querySelector(".recovery-box code")');
 const recovery=await c.evaluate('document.querySelector(".recovery-box code").textContent.trim()');
 await fill('input[placeholder="Последние 6 символов"]',recovery.replace(/\s+/g,'').slice(-6));
 await c.evaluate('document.querySelector("input[type=checkbox]").click()');
 await click('К библиотеке');
 await c.wait('document.querySelector(".library-layout") || document.querySelector(".empty-library")');
 console.log('PASS native create + recovery confirmation');
 await click('Добавить'); await c.wait('document.querySelector("textarea")');
 await fill('textarea',['fb2','epub','pdf'].map(ext=>base+'/book.'+ext).join('\n'));
 await click('Добавить'); await c.wait(has('Обработано 3 из 3'));
 await click('Вернуться к книгам');
 console.log('PASS native UI batch import');
 // Refresh library through the actual lock/unlock flow.
 await api('Lock'); await c.wait('document.querySelector("#archive-key")');
 await fill('input[aria-label="Папка библиотеки"]',vault);
 await fill('#archive-key','native-test-password'); await click('Войти в mut');
 await c.wait('document.querySelectorAll(".book-item").length===3');
 const books=await api('ListBooks');
 await mkdir(base+'/screenshots',{recursive:true});
 for(const b of books){
   await c.evaluate(`[...document.querySelectorAll('.book-item')].find(e=>e.getAttribute('aria-label')?.startsWith(${q('Открыть '+b.title+' · ')})).click()`);
   const pdf=b.format.toLowerCase()==='pdf';
   await c.wait(pdf?'document.querySelector(".textLayer")?.textContent.includes("page one") || document.querySelector(".pdf-wrapper [role=alert]")':has('Author-owned '+b.format.toUpperCase()+' release fixture.'));
   if(await c.evaluate('!!document.querySelector(".pdf-wrapper [role=alert]")'))throw Error('PDF renderer rejected page');
   if(pdf){
    await c.evaluate(`document.querySelector(${q('[aria-label="Следующая страница"]')}).click()`); await c.wait('document.querySelector(".textLayer")?.textContent.includes("page two")');
    assert(await c.evaluate(`(()=>{const e=document.querySelector('canvas');const p=e.getContext('2d').getImageData(0,0,e.width,e.height).data;for(let i=0;i<p.length;i+=4)if(p[i]<150&&p[i+1]<150&&p[i+2]<150&&p[i+3]>0)return true;return false})()`),'PDF canvas must have visible ink');
   }
   await c.evaluate(`document.querySelector(${q('.bookmark-toggle')}).click()`);
   await c.wait(`document.querySelector(${q('.bookmark-toggle')}).getAttribute('aria-pressed')==='true'`);
   const dimensions=await c.evaluate('({width:innerWidth,height:innerHeight})');
   const snap=await c.command('Page.snapshotRect',{x:0,y:0,...dimensions,coordinateSystem:'Viewport'});
   await writeFile(base+'/screenshots/'+b.format+'.png',Buffer.from(snap.dataURL.split(',')[1],'base64'));
   await click('Все книги');
   console.log('PASS native '+b.format+' render + bookmark');
 }
 assert.equal(await c.evaluate('window.shouldNotRun'),undefined);
 // Exercise both exits from BackupView; returning must keep the session.
 await click('Резервные копии');
 await c.wait(`document.querySelector(${q('input[aria-label="Куда сохранить копию"]')})`);
 await click('Назад'); await c.wait('document.querySelectorAll(".book-item").length===3');
 assert(await api('IsUnlocked'));
 await click('Резервные копии');
 await fill('input[aria-label="Куда сохранить копию"]',base+'/backups');
 await click('Создать и проверить');
 const backupPath=await c.wait('document.querySelector(".report-path")?.textContent.trim()');
 await click('Назад'); await c.wait('document.querySelectorAll(".book-item").length===3');
 assert(await api('IsUnlocked'));
 const backup={path:backupPath};
 console.log('PASS native backup cancel/completion return keeps session');
 const restored=base+'/restored-'+Date.now();
 await click('Резервные копии'); await click('Восстановить копию');
 await fill('input[aria-label="Папка резервной копии"]',backup.path);
 await fill('input[aria-label="Новая папка библиотеки"]',restored);
 await click('Код восстановления'); await fill('#restore-secret',recovery);
 await click('Проверить и восстановить'); await c.wait('document.querySelector(".report-path")');
 await click('Назад'); await c.wait('document.querySelectorAll(".book-item").length===3');
 await api('Lock'); await c.wait('document.querySelector("#archive-key")');
 await fill('input[aria-label="Папка библиотеки"]',restored); await fill('#archive-key','native-test-password'); await click('Войти в mut');
 await c.wait('document.querySelectorAll(".book-item").length===3');
 const restoredBooks=await api('ListBooks');
 assert(restoredBooks.every(b=>b.bookmarks.length===1));
 assert(restoredBooks.find(b=>b.format.toLowerCase()==='pdf').position.locator==='v1:pdf:2');
 const requests=c.events.filter(e=>e.method==='Network.requestWillBeSent'&&/^https?:/.test(e.params.request.url));
 assert.equal(requests.length,0,'No external network requests');
 console.log('PASS native backup + authenticated recovery restore + stored positions; no external requests');
 console.log(JSON.stringify({vault,restored,backup:backup.path,screenshots:base+'/screenshots'}));
 await writeFile(base+'/result.json',JSON.stringify({vault,restored,backup:backup.path}));
} catch(e){ console.error('Native smoke failed:',e.message); process.exitCode=1;
} finally {
 if(ownsTestSession) await api('Lock').catch(()=>{});
 await c.evaluate(old===null?'localStorage.removeItem("mut.paths")':`localStorage.setItem("mut.paths",${q(old)})`);
 c.close();
}
