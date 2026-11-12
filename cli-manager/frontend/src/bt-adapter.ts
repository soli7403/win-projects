// window.bt 호환 어댑터 — v1.1.0 렌더러가 쓰던 preload 브리지와 같은 인터페이스를 Wails 바인딩·이벤트로 연결한다.
/* eslint-disable @typescript-eslint/no-explicit-any */
declare global {
  interface Window { go: any; runtime: any }
}

const app = (): any => window.go.main.App

function b64(u: Uint8Array): string {
  let s = ''
  for (let i = 0; i < u.length; i += 0x8000) s += String.fromCharCode(...u.subarray(i, i + 0x8000))
  return btoa(s)
}

// Uint8Array 는 JSON 으로 그대로 못 보낸다 → {__b64} 로 감싸 Go 에서 풀어 쓴다.
const replacer = (_k: string, v: unknown): unknown => (v instanceof Uint8Array ? { __b64: b64(v) } : v)

function detectPlatform(): string {
  const ua = navigator.userAgent
  if (/Windows/i.test(ua)) return 'win32'
  if (/Mac/i.test(ua)) return 'darwin'
  return 'linux'
}

window.bt = {
  async call(ch: string, ...args: unknown[]) {
    try {
      const r: string = await app().Call(ch, JSON.stringify(args, replacer))
      return r === '' ? undefined : JSON.parse(r)
    } catch (e) {
      throw new Error(typeof e === 'string' ? e : (e as Error)?.message ?? String(e))
    }
  },
  on(ch: string, cb: (...a: never[]) => void) {
    return window.runtime.EventsOn(ch, (...a: never[]) => cb(...a)) as () => void
  },
  write: (id: string, data: string) => { void app().Write(id, data) },
  resize: (id: string, cols: number, rows: number) => { void app().Resize(id, cols, rows) },
  respond: (reqId: string, value: unknown) => { void app().Respond(reqId, JSON.stringify(value ?? null)) },
  // Wails 웹뷰는 File 의 실제 경로를 주지 않는다. 파일 드롭 경로는 Go 쪽 OnFileDrop 으로 대체 예정 (WIP 참조).
  pathForFile: (_f: File): string => '',
  platform: detectPlatform()
}

export {}
