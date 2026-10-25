// 输出和快照使用独立水位：较新的快照不能吞掉尚未展示的输出。
const encoder = new TextEncoder();
const bytes = (text) => encoder.encode(text).length;
const LIMIT = 1024 * 1024;

export class OutputTail {
  constructor(limit = LIMIT) {
    this.limit = limit;
    this.items = new Map();
    this.truncated = false;
  }
  append(seq, text) {
    if (this.items.has(seq)) return;
    this.items.set(seq, String(text));
    let size = [...this.items.values()].reduce(
      (sum, value) => sum + bytes(value),
      0,
    );
    for (const [id, value] of [...this.items].sort((a, b) => a[0] - b[0])) {
      if (size <= this.limit) break;
      const length = bytes(value);
      if (size - length >= this.limit) {
        this.items.delete(id);
        size -= length;
      } else {
        let remaining = length;
        let offset = 0;
        for (const rune of value) {
          if (size - length + remaining <= this.limit) break;
          remaining -= bytes(rune);
          offset += rune.length;
        }
        this.items.set(id, value.slice(offset));
        size = size - length + remaining;
      }
      this.truncated = true;
    }
  }
  get text() {
    return [...this.items]
      .sort((a, b) => a[0] - b[0])
      .map(([, text]) => text)
      .join("");
  }
}

export class SessionEvents {
  constructor() {
    this.listeners = new Set();
    this.id = "";
    this.stateSeq = -1;
    this.seen = new Set();
    this.pending = null;
  }
  subscribe(callback) {
    this.listeners.add(callback);
    return () => this.listeners.delete(callback);
  }
  emit(event) {
    for (const callback of this.listeners) callback(event);
  }
  beginCreate() {
    if (this.pending) return false;
    this.pending = [];
    this.pendingBytes = 0;
    this.pendingMarkers = new Map();
    return true;
  }
  cancelCreate() {
    const events = this.pending || [];
    this.pending = null;
    for (const event of events) this.receive(event);
  }
  receive(event) {
    if (!event?.sessionId || !Number.isSafeInteger(event.seq) || event.seq < 1)
      return;
    if (this.pending) {
      this.pending.push(event);
      this.pendingBytes += bytes(event.output?.text || "");
      while (this.pendingBytes > LIMIT) {
        const index = this.pending.findIndex(
          (value) =>
            value.output && this.pendingMarkers.get(value.sessionId) !== value,
        );
        this.pendingBytes -= bytes(this.pending[index].output.text);
        const old = this.pending[index];
        if (!this.pendingMarkers.has(old.sessionId)) {
          const marker = {
            ...old,
            output: { stream: "stderr", text: "[早到输出已截断]\n" },
          };
          this.pending[index] = marker;
          this.pendingMarkers.set(old.sessionId, marker);
          this.pendingBytes += bytes(marker.output.text);
        } else this.pending.splice(index, 1);
      }
      return;
    }
    if (event.sessionId !== this.id || this.seen.has(event.seq)) return;
    this.seen.add(event.seq);
    if (event.type === "state_changed") this.applySnapshot(event.state);
    else this.emit(event);
  }
  applySnapshot(snapshot) {
    if (
      !snapshot ||
      snapshot.sessionId !== this.id ||
      snapshot.seq < this.stateSeq
    )
      return;
    this.stateSeq = snapshot.seq;
    this.emit({
      sessionId: this.id,
      seq: snapshot.seq,
      type: "state_changed",
      state: snapshot,
    });
  }
  activate(snapshot) {
    const events = this.pending || [];
    this.pending = null;
    if (this.id !== snapshot.sessionId) {
      this.id = snapshot.sessionId;
      this.stateSeq = -1;
      this.seen.clear();
    }
    this.applySnapshot(snapshot);
    for (const event of events.sort((a, b) => a.seq - b.seq))
      this.receive(event);
  }
}
