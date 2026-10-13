// WebKitGTK remote inspector protocol. Only use with a local test build.
export async function connect(endpoint = 'http://127.0.0.1:9223') {
  const html = await (await fetch(endpoint)).text();
  const path = html.match(/\/socket\/\d+\/\d+\/WebPage/)?.[0];
  if (!path) throw Error('No inspectable WebKit page');
  const socket = new WebSocket(endpoint.replace('http', 'ws') + path);
  let target, id = 0;
  const pending = new Map();
  const events = [];
  socket.addEventListener('message', ({data}) => {
    const msg = JSON.parse(data);
    if (msg.method === 'Target.targetCreated' && msg.params.targetInfo.type === 'page') target = msg.params.targetInfo.targetId;
    if (msg.method !== 'Target.dispatchMessageFromTarget') return;
    const inner = JSON.parse(msg.params.message);
    if (inner.id) { const p = pending.get(inner.id); pending.delete(inner.id); inner.error ? p?.reject(Error(JSON.stringify(inner.error))) : p?.resolve(inner.result); }
    else { events.push(inner); if(process.env.MUT_INSPECT_LOG && inner.method === "Console.messageAdded") console.log(inner.params.message.text); }
  });
  await new Promise((resolve,reject) => {socket.addEventListener('open', resolve); socket.addEventListener('error', reject);});
  for (let n=0; !target && n<100; n++) await new Promise(r=>setTimeout(r,50));
  if (!target) throw Error('No page target');
  async function command(method, params = {}) {
    const n=++id;
    const response = new Promise((resolve,reject)=>pending.set(n,{resolve,reject}));
    socket.send(JSON.stringify({id:n+100000,method:'Target.sendMessageToTarget',params:{targetId:target,message:JSON.stringify({id:n,method,params})}}));
    let timer;
    try { return await Promise.race([response,new Promise((_,reject)=>timer=setTimeout(()=>reject(Error('Inspector timeout: '+method)),10000))]); }
    finally { clearTimeout(timer); pending.delete(n); }
  }
  async function evaluate(expression) {
    const result = await command('Runtime.evaluate',{expression,returnByValue:true});
    if (result.wasThrown) throw Error(result.result.description);
    return result.result.value;
  }
  async function wait(expression, timeout=20000) {
    const start=Date.now();
    while (Date.now()-start<timeout) {
      const value = await evaluate(expression);
      if(value) return value;
      await new Promise(r=>setTimeout(r,100));
    }
    throw Error('Timed out: '+expression);
  }
  async function asyncEvaluate(expression) {
    const token='__mut_test_'+(++id);
    await evaluate(`window[${JSON.stringify(token)}]={done:false}; Promise.resolve(${expression}).then(value=>window[${JSON.stringify(token)}]={done:true,value},e=>window[${JSON.stringify(token)}]={done:true,error:String(e)}); undefined`);
    const result=await wait(`window[${JSON.stringify(token)}]?.done && window[${JSON.stringify(token)}]`,60000);
    await evaluate(`delete window[${JSON.stringify(token)}]`);
    if(result.error) throw Error(result.error);
    return result.value;
  }
  await command('Console.enable'); await command('Network.enable');
  return {command,evaluate,asyncEvaluate,wait,events,close:()=>socket.close()};
}
