// 仅按 CONTRACT.md 的固定协议组合提供测试响应，不代表真实供应商兼容。
const http = require('node:http');
http.createServer(async (req, res) => {
  const send = (status, body) => { res.writeHead(status, {'Content-Type':'application/json'}); res.end(JSON.stringify(body)); };
  let body = req.body;
  if (body === undefined) {
    let raw = ''; for await (const chunk of req) raw += chunk;
    try { body = raw ? JSON.parse(raw) : {}; } catch { return send(400, {error:'invalid-json'}); }
  }
  if (typeof body === 'string') { try { body = JSON.parse(body); } catch { return send(400, {error:'invalid-json'}); } }
  const path = req.url.split('?')[0];
  if (path === '/v1/messages' && req.method === 'POST') {
    if (req.headers['x-api-key'] !== 'fixture-model') return send(401, {type:'error'});
    if (body.model !== 'fixture-model-name' || body.max_tokens !== 1) return send(400, {type:'error'});
    return send(200, {id:'fixture-message',type:'message',role:'assistant',model:body.model,content:[{type:'text',text:'OK'}],stop_reason:'max_tokens',usage:{input_tokens:1,output_tokens:1}});
  }
  if (path !== '/mcp') return send(501, {error:'not-modeled'});
  if (req.headers.authorization !== 'Bearer fixture-mcp') return send(401, {error:'invalid-fixture-token'});
  if (req.method === 'DELETE') { res.writeHead(204); return res.end(); }
  if (req.method !== 'POST') return send(501, {error:'not-modeled'});
  if (body.method === 'initialize') {
    res.setHeader('Mcp-Session-Id', 'fixture-session');
    return send(200, {jsonrpc:'2.0',id:body.id,result:{protocolVersion:'2025-03-26',capabilities:{tools:{}},serverInfo:{name:'fixture',version:'1'}}});
  }
  if (req.headers['mcp-session-id'] !== 'fixture-session') return send(400, {error:'missing-session'});
  if (body.method === 'notifications/initialized') { res.writeHead(202); return res.end(); }
  if (body.method === 'tools/list') return send(200, {jsonrpc:'2.0',id:body.id,result:{tools:[{name:'approved_read',description:'受控只读工具',inputSchema:{type:'object'}}]}});
  return send(501, {error:'not-modeled'});
}).listen(process.env.PORT || 4500);
