// 固定受信适配器；仅使用受限 profile 已注入的官方 Agent 服务。
import { randomUUID } from 'node:crypto';
import { installModelSelection } from __AGENT_MODULE__;
import { createUserMessage } from __LLM_MODULE__;
import { SessionSeq } from __SESSION_MODULE__;

export const name = 'harbor-conversation';
export const inject = ['agentDefaultModel', 'agents', 'sessions'];

export function apply(ctx) {
  const exit = ctx.get('appExit');
  if (!exit) throw new Error('Harbor conversation requires appExit');
  const send = (event) => process.stdout.write(JSON.stringify(event) + '\n');
  async function run() {
    await ctx.get('loader')?.await();
    const selection = ctx.get('agentDefaultModel').currentSelection();
    const fs = ctx.get('fs');
    const cwd = fs ? fs.processPath(await fs.resolve('.')) : process.cwd();
    const sessionId = `harbor-${randomUUID()}`;
    const { agent } = await ctx.get('agents').create({
      sessionId,
      meta: { cwd },
      agentOptions: { provider: selection.provider, model: selection.model },
      setup: (agentCtx) => { installModelSelection(agentCtx, { current: selection, assembled: undefined }); },
    });
    await agent.whenIdle();
    send({ type: 'ready', sessionId });
    let round = 0;
    let pending = '';
    // 不使用 readline 的后台无限队列：完成本轮后才消费下一条消息。
    process.stdin.setEncoding('utf8');
    for await (const chunk of process.stdin) {
      pending += chunk;
      let newline;
      while ((newline = pending.indexOf('\n')) !== -1) {
        if (newline > 256 * 1024) throw new Error('invalid message');
        const line = pending.slice(0, newline);
        pending = pending.slice(newline + 1);
        const message = JSON.parse(line);
        if (typeof message.text !== 'string' || !message.text.trim() || Buffer.byteLength(message.text) > 32768 || message.text.includes('\0')) throw new Error('invalid message');
        const firstSeq = agent.session.seq;
        let failure;
        const stopErrors = ctx.on('agent/error', ({ agent: subject }) => { if (subject === agent) failure = agent.session.seq; });
        const stopTerminals = ctx.on('session/event', (session, event) => { if (session === agent.session && event.type === 'turn/end' && failure !== undefined && event.seq >= failure) failure = undefined; });
        const stopStream = ctx.on('agent/assistant-stream', ({ agent: subject, frame }) => {
          if (subject !== agent || frame.type !== 'chunk') return;
          const chunk = frame.chunk;
          if ((chunk.type === 'text-delta' || chunk.type === 'reasoning-delta') && chunk.text) send({ type: chunk.type, text: chunk.text });
        });
        try {
          agent.followup(createUserMessage({ content: [{ type: 'text', text: message.text }], source: { kind: 'user' } }));
          await agent.whenIdle();
          await ctx.get('sessions').flush(agent.session);
          let text = '';
          let reason;
          for (let seq = firstSeq; seq < agent.session.seq; seq++) {
            const event = agent.session.eventAt(SessionSeq(seq));
            if (!event) throw new Error('invalid session event');
            if (event.type === 'assistant/message') {
              const joined = event.data.message.content.filter((b) => b.type === 'text').map((b) => b.text).join('');
              if (joined) text = joined;
            }
            if (event.type === 'turn/end') reason = event.data.reason;
          }
          if (failure !== undefined || reason?.kind !== 'completed') throw new Error('agent turn failed');
          send({ type: 'final', sessionId, round: ++round, text });
        } finally {
          stopErrors();
          stopTerminals();
          stopStream();
        }
      }
      if (Buffer.byteLength(pending) > 256 * 1024) throw new Error('invalid message');
    }
    if (pending.trim()) throw new Error('incomplete message');
    exit(0);
  }
  run().catch(() => { send({ type: 'error', message: 'dsh 会话执行失败。' }); exit(1); });
}
