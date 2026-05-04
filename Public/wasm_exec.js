// Optimized Go/WASM Glue for 2026 Standards
(() => {
    if (typeof globalThis !== "undefined") {} else if (typeof global !== "undefined") { globalThis = global; } else if (typeof self !== "undefined") { globalThis = self; } else if (typeof window !== "undefined") { globalThis = window; } else { throw new Error("unable to locate global object"); }
    const encoder = new TextEncoder("utf-8");
    const decoder = new TextDecoder("utf-8");
    globalThis.Go = class {
        constructor() {
            this.argv = ["js"];
            this.env = {};
            this.exit = (code) => { if (code !== 0) { console.warn("exit code:", code); } };
            this._exitPromise = new Promise((resolve) => { this._resolveExit = resolve; });
            this._pendingEvent = null;
            this._scheduledTimeouts = new Map();
            this._nextCallbackTimeoutID = 1;
            const setInt64 = (addr, v) => { this.mem.setUint32(addr + 0, v, true); this.mem.setUint32(addr + 4, Math.floor(v / 4294967296), true); };
            const getInt64 = (addr) => { return this.mem.getUint32(addr + 0, true) + this.mem.getInt32(addr + 4, true) * 4294967296; };
            const loadValue = (addr) => {
                const f = this.mem.getFloat64(addr, true);
                if (f === 0) return undefined;
                if (!isNaN(f)) return f;
                return this._values[this.mem.getUint32(addr, true)];
            };
            const storeValue = (addr, v) => {
                const nanHead = 0x7FF80000;
                if (typeof v === "number" && v !== 0) {
                    if (isNaN(v)) { this.mem.setUint32(addr + 4, nanHead, true); this.mem.setUint32(addr, 0, true); return; }
                    this.mem.setFloat64(addr, v, true); return;
                }
                if (v === undefined) { this.mem.setFloat64(addr, 0, true); return; }
                let id = this._ids.get(v);
                if (id === undefined) {
                    id = this._idPool.pop() || this._values.length;
                    this._values[id] = v; this._ids.set(v, id);
                }
                this._refs.set(id, (this._refs.get(id) || 0) + 1);
                this.mem.setUint32(addr + 4, nanHead | 1, true); this.mem.setUint32(addr, id, true);
            };
            const loadSlice = (addr) => { return new Uint8Array(this._inst.exports.mem.buffer, getInt64(addr + 0), getInt64(addr + 8)); };
            const loadString = (addr) => { return decoder.decode(loadSlice(addr)); };
            const timeOrigin = Date.now() - performance.now();
            
            // This is the "gojs" block your engine looks for
            this.importObject = {
                gojs: {
                    "runtime.wasmExit": (sp) => {
                        const code = this.mem.getInt32(sp + 8, true);
                        this.exited = true;
                        delete this._inst;
                        delete this._values;
                        delete this._ids;
                        delete this._idPool;
                        this.exit(code);
                    },
                    "runtime.wasmWrite": (sp) => {
                        const p = getInt64(sp + 16);
                        const n = this.mem.getInt32(sp + 24, true);
                        console.log(decoder.decode(new Uint8Array(this._inst.exports.mem.buffer, p, n)));
                    },
                    "runtime.resetMemoryDataView": (sp) => {
                        this.mem = new DataView(this._inst.exports.mem.buffer);
                    },
                    "runtime.nanotime1": (sp) => { setInt64(sp + 8, (timeOrigin + performance.now()) * 1000000); },
                    "runtime.walltime": (sp) => { const msec = (new Date()).getTime(); setInt64(sp + 8, msec / 1000); this.mem.setUint32(sp + 16, (msec % 1000) * 1000000, true); },
                    "runtime.scheduleTimeoutEvent": (sp) => {
                        const id = this._nextCallbackTimeoutID++;
                        this._scheduledTimeouts.set(id, setTimeout(() => { this._resume(); }, getInt64(sp + 8) + 1));
                        this.mem.setUint32(sp + 16, id, true);
                    },
                    "runtime.clearTimeoutEvent": (sp) => { clearTimeout(this._scheduledTimeouts.get(this.mem.getUint32(sp + 8, true))); },
                    "runtime.getRandomData": (sp) => { crypto.getRandomValues(loadSlice(sp + 8)); },
                    "syscall/js.finalizeRef": (sp) => { /* logic */ },
                    "syscall/js.stringVal": (sp) => { storeValue(sp + 24, loadString(sp + 8)); },
                    "syscall/js.valueGet": (sp) => {
                        const v = loadValue(sp + 8);
                        const s = loadString(sp + 16);
                        storeValue(sp + 32, Reflect.get(v, s));
                    },
                    "syscall/js.valueSet": (sp) => {
                        const v = loadValue(sp + 8);
                        const s = loadString(sp + 16);
                        Reflect.set(v, s, loadValue(sp + 32));
                    },
                    "syscall/js.valueDelete": (sp) => {
                        Reflect.deleteProperty(loadValue(sp + 8), loadString(sp + 16));
                    },
                    "syscall/js.valueIndex": (sp) => {
                        storeValue(sp + 24, Reflect.get(loadValue(sp + 8), getInt64(sp + 16)));
                    },
                    "syscall/js.valueSetIndex": (sp) => {
                        Reflect.set(loadValue(sp + 8), getInt64(sp + 16), loadValue(sp + 24));
                    },
                    "syscall/js.valueCall": (sp) => {
                        try {
                            const v = loadValue(sp + 8);
                            const m = Reflect.get(v, loadString(sp + 16));
                            const args = loadSliceOfValues(sp + 32);
                            const result = Reflect.apply(m, v, args);
                            storeValue(sp + 56, result);
                            this.mem.setUint8(sp + 64, 1);
                        } catch (err) {
                            storeValue(sp + 56, err);
                            this.mem.setUint8(sp + 64, 0);
                        }
                    },
                    "syscall/js.valueInvoke": (sp) => {
                        try {
                            const v = loadValue(sp + 8);
                            const args = loadSliceOfValues(sp + 16);
                            const result = Reflect.apply(v, undefined, args);
                            storeValue(sp + 40, result);
                            this.mem.setUint8(sp + 48, 1);
                        } catch (err) {
                            storeValue(sp + 40, err);
                            this.mem.setUint8(sp + 48, 0);
                        }
                    },
                    "syscall/js.valueNew": (sp) => {
                        try {
                            const v = loadValue(sp + 8);
                            const args = loadSliceOfValues(sp + 16);
                            const result = Reflect.construct(v, args);
                            storeValue(sp + 40, result);
                            this.mem.setUint8(sp + 48, 1);
                        } catch (err) {
                            storeValue(sp + 40, err);
                            this.mem.setUint8(sp + 48, 0);
                        }
                    },
                    "syscall/js.valueLength": (sp) => {
                        setInt64(sp + 16, parseInt(loadValue(sp + 8).length));
                    },
                    "syscall/js.valuePrepareString": (sp) => {
                        const str = encoder.encode(String(loadValue(sp + 8)));
                        storeValue(sp + 16, str);
                        setInt64(sp + 24, str.length);
                    },
                    "syscall/js.valueLoadString": (sp) => {
                        const str = loadValue(sp + 8);
                        loadSlice(sp + 16).set(str);
                    },
                    "syscall/js.funcMakeStack": (sp) => {
                        const id = this.mem.getUint32(sp + 8, true);
                        this._values[id] = (...args) => {
                            this._pendingEvent = { id, this: this, args };
                            this._resume();
                            return this._pendingEvent.result;
                        };
                    },
                    "syscall/js.funcCall": (sp) => {
                        const e = this._pendingEvent;
                        if (!e) return;
                        e.result = loadValue(sp + 8);
                        this._resolveCallback();
                    },
                }
            };

            const loadSliceOfValues = (addr) => {
                const array = getInt64(addr + 0);
                const len = getInt64(addr + 8);
                const a = new Array(len);
                for (let i = 0; i < len; i++) {
                    a[i] = loadValue(array + i * 8);
                }
                return a;
            };
        }
        async run(instance) {
            this._inst = instance;
            this.mem = new DataView(this._inst.exports.mem.buffer);
            this._values = [NaN, 0, null, true, false, globalThis, this];
            this._ids = new Map([[NaN, 0], [0, 1], [null, 2], [true, 3], [false, 4], [globalThis, 5], [this, 6]]);
            this._idPool = []; this._refs = new Map();
            while (true) {
                const callbackPromise = new Promise((resolve) => { this._resolveCallback = resolve; });
                this._inst.exports._start();
                if (this._exited) break;
                await callbackPromise;
            }
        }
        _resume() { if (this._resolveCallback) this._resolveCallback(); }
    };
})();