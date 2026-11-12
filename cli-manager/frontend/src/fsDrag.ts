// 파일 패널 끌어 놓기 복사: 드래그 중인 항목, 탭/칸 위에 머물면 전환, 복사 작업 시작.
import { useApp, type Tab } from './state'
import { api } from './api'
import { panes } from './layout'

export const FS_MIME = 'bt/fs'

export interface FsDrag {
  sessionId: string
  /** 원본 항목들이 있는 폴더 (같은 폴더에 놓으면 무시) */
  dir: string
  paths: string[]
}

let g_cur: FsDrag | null = null

/** 세션별 파일 패널에서 마지막으로 본 폴더 (탭에 바로 놓을 때 대상) */
export const lastDir = new Map<string, string>()

export const isFsDrag = (e: React.DragEvent<HTMLElement>): boolean => !!g_cur && e.dataTransfer!.types.includes(FS_MIME)

export function startFsDrag(e: React.DragEvent<HTMLElement>, d: FsDrag): void {
  g_cur = d
  e.dataTransfer!.setData(FS_MIME, '1')
  e.dataTransfer!.effectAllowed = 'copy'
}

// 드래그를 시작한 행은 탭이 바뀌면 사라져 dragend 가 안 올 수 있다 → 창 단위로 정리
function endDrag(): void {
  setTimeout(() => { g_cur = null }, 0)
  hoverEnd()
}
window.addEventListener('dragend', endDrag)
window.addEventListener('drop', endDrag)

let g_hoverKey = ''
let g_hoverTimer: ReturnType<typeof setTimeout> | undefined

/** 드래그 중 같은 대상 위에 0.5초 머물면 fn 실행 (탭 전환·칸 포커스) */
export function hoverStart(key: string, fn: () => void): void {
  if (g_hoverKey === key) return
  hoverEnd()
  g_hoverKey = key
  g_hoverTimer = setTimeout(fn, 500)
}

export function hoverEnd(key?: string): void {
  if (key && key !== g_hoverKey) return
  clearTimeout(g_hoverTimer)
  g_hoverKey = ''
}

/** 파일 패널을 쓸 수 있는 세션인지 (SSH 의 SFTP 또는 로컬 터미널) */
export function canBrowse(sessionId: string | undefined): boolean {
  const s = sessionId ? useApp.getState().sessions[sessionId] : undefined
  return !!s && (s.canSftp || s.protocol === 'local') && s.state === 'connected'
}

/** 탭의 포커스 칸 세션 */
export function tabSession(t: Tab): string | undefined {
  const ps = panes(t.root)
  const p = ps.find((x) => x.id === t.focused) ?? ps[0]
  return p?.sessionId || undefined
}

/** 드래그 중인 항목을 dstSession 의 dir 로 복사하는 작업을 시작한다 (dir 이 비면 그 세션의 마지막/시작 폴더).
 * 같은 이름이 있으면 확인 후 덮어쓴다. 진행은 화면 맨 아래 복사 목록(CopyJobs)에 한 줄로 나온다 */
export async function dropFsDrag(dstSession: string, dir: string): Promise<void> {
  const d = g_cur
  g_cur = null
  hoverEnd()
  if (!d || !d.paths.length) return
  const st = useApp.getState()
  if (!canBrowse(dstSession)) { st.toast('error', '이 탭에는 복사할 수 없습니다 (연결된 SSH 또는 로컬 터미널만 가능)'); return }
  const szDir = dir || lastDir.get(dstSession) || ''
  if (d.sessionId === dstSession && (d.dir === szDir || d.paths.includes(szDir))) return // 같은 폴더 / 끌던 폴더 자신에 놓음
  try {
    const r = await api.sftp.copy(d.sessionId, d.paths, dstSession, szDir)
    if (r.conflicts.length) {
      const names = r.conflicts.slice(0, 5).join(', ') + (r.conflicts.length > 5 ? ` 외 ${r.conflicts.length - 5}개` : '')
      const ok = await st.confirm({ title: '덮어쓰기', message: `대상 폴더에 같은 이름이 있습니다: ${names}\n덮어쓸까요?`, okText: '덮어쓰기', danger: true })
      if (ok) await api.sftp.copy(d.sessionId, d.paths, dstSession, szDir, true)
    }
  } catch (e) {
    st.toast('error', `복사 실패: ${(e as Error).message}`)
  }
}
