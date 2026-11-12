// 3.4 탭·분할 레이아웃 저장/복원.
// 저장하는 것은 배치 구조와 각 칸의 서버 식별자(또는 로컬 폴더)뿐 — 비밀번호·화면 내용은 저장하지 않는다.
import { api } from './api'
import { useApp, type Tab } from './state'
import { LayoutNode, PaneNode, uid, isEmpty } from './layout'
import type { SavedLayout, SavedNode } from '@shared/types'

let restoreDone = false
let timer: ReturnType<typeof setTimeout> | null = null
let lastJson = ''

/** 현재 탭 → 저장 형식. 저장할 수 없는 즉석 SSH/텔넷 칸은 뺀다 (비밀번호가 필요하므로). */
export function serializeLayout(): SavedLayout {
  const st = useApp.getState()
  const conv = (n: LayoutNode): SavedNode | null => {
    if (n.type === 'pane') {
      if (isEmpty(n)) return { type: 'pane', empty: true }
      const s = st.sessions[n.sessionId]
      if (!s) return null
      if (s.hostId) return { type: 'pane', hostId: s.hostId }
      if (s.protocol === 'local') return { type: 'pane', local: { cwd: s.target.replace(/^로컬 · /, '') } }
      return null
    }
    const kept: { node: SavedNode; size: number }[] = []
    n.children.forEach((c, i) => { const r = conv(c); if (r) kept.push({ node: r, size: n.sizes[i] }) })
    if (!kept.length) return null
    if (kept.length === 1) return kept[0].node
    const total = kept.reduce((a, k) => a + k.size, 0)
    return { type: 'split', dir: n.dir, sizes: kept.map((k) => (k.size / total) * 100), children: kept.map((k) => k.node) }
  }
  const tabs: SavedLayout['tabs'] = []
  let active = 0
  for (const t of st.tabs) {
    const root = conv(t.root)
    if (!root) continue
    if (t.id === st.activeTab) active = tabs.length
    tabs.push({ title: t.title, broadcast: t.broadcast || undefined, root })
  }
  return { tabs, active }
}

/** 탭이 바뀔 때마다(0.3초 모아서) 저장. 복원이 끝나기 전에는 저장하지 않는다 (시작 직후 빈 상태로 덮어쓰지 않도록). */
export function startLayoutPersist(): void {
  useApp.subscribe((s, prev) => {
    if (!restoreDone || (s.tabs === prev.tabs && s.activeTab === prev.activeTab && s.sessions === prev.sessions)) return
    if (s.settings.restoreLayout === false) return
    if (timer) clearTimeout(timer)
    timer = setTimeout(() => {
      const layout = serializeLayout()
      const json = JSON.stringify(layout)
      if (json === lastJson) return // 세션 상태만 바뀐 경우는 다시 쓰지 않는다
      lastJson = json
      void api.store.patch({ layout } as never)
    }, 300)
  })
}

/** 저장된 레이아웃을 다시 열어 접속한다 (볼트가 열린 뒤 한 번). */
export async function restoreLayoutOnce(): Promise<void> {
  if (restoreDone) return
  restoreDone = true
  const st = useApp.getState()
  const saved = st.savedLayout
  if (st.settings.restoreLayout === false || !saved || !saved.tabs.length) return
  const build = async (n: SavedNode): Promise<LayoutNode | null> => {
    if (n.type === 'pane') {
      const pane: PaneNode = { type: 'pane', id: uid(), sessionId: '' }
      try {
        if (n.hostId && st.hosts.some((h) => h.id === n.hostId)) {
          const info = await api.session.open({ hostId: n.hostId })
          useApp.setState((s) => ({ sessions: { ...s.sessions, [info.id]: info } }))
          pane.sessionId = info.id
        } else if (n.local) {
          const info = await api.session.open({ adhoc: { host: '', protocol: 'local', cwd: n.local.cwd } })
          useApp.setState((s) => ({ sessions: { ...s.sessions, [info.id]: info } }))
          pane.sessionId = info.id
        } else if (!n.empty) {
          return null // 지워진 서버
        }
      } catch { return null }
      return pane
    }
    const kids = await Promise.all(n.children.map(build))
    const keptIdx = kids.map((k, i) => (k ? i : -1)).filter((i) => i >= 0)
    if (!keptIdx.length) return null
    if (keptIdx.length === 1) return kids[keptIdx[0]]
    const total = keptIdx.reduce((a, i) => a + n.sizes[i], 0)
    return { type: 'split', id: uid(), dir: n.dir, children: keptIdx.map((i) => kids[i]!), sizes: keptIdx.map((i) => (n.sizes[i] / total) * 100) }
  }
  const tabs: Tab[] = []
  let activeIdx = 0
  for (let i = 0; i < saved.tabs.length; i++) {
    const t = saved.tabs[i]
    const root = await build(t.root)
    if (!root) continue
    if (i === saved.active) activeIdx = tabs.length
    const first = firstPane(root)
    tabs.push({ id: uid(), root, focused: first.id, broadcast: !!t.broadcast, title: t.title })
  }
  if (tabs.length) useApp.setState({ tabs, activeTab: tabs[Math.min(activeIdx, tabs.length - 1)].id })
}

function firstPane(n: LayoutNode): PaneNode {
  return n.type === 'pane' ? n : firstPane(n.children[0])
}
