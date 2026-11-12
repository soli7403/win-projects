import { create } from 'zustand'
import type { CopyJob, SavedLayout, CredentialMeta, Group, Host, SessionInfo, Settings, Snippet, StoreData, Transfer, UiRequest, VaultStatus, AdhocTarget } from '@shared/types'
import { DEFAULT_SETTINGS } from '@shared/types'
import { api } from './api'
import { LayoutNode, PaneNode, uid, panes, splitPane, removePane, grid, Dir, findPane, Edge, edgeDir, edgeBefore, isEmpty, setPaneSession, movePane, swapPanes } from './layout'

/** 새 세션을 놓을 자리: 새 탭 / 포커스 칸 분할 / 특정 칸의 가장자리 분할 / 빈 칸 채우기 */
export type PlaceTarget = 'tab' | Dir | { tabId: string; paneId: string; edge: Edge | 'fill' }

export interface Tab {
  id: string
  root: LayoutNode
  focused: string // pane id
  broadcast: boolean
  title?: string
  zoomed?: string | null
}

export interface MenuItem {
  label?: string
  shortcut?: string
  danger?: boolean
  disabled?: boolean
  separator?: boolean
  onClick?: () => void
}

export interface ConfirmReq {
  title: string
  message: string
  detail?: string
  okText?: string
  danger?: boolean
  resolve: (ok: boolean) => void
}

export interface Toast { id: string; kind: 'ok' | 'error' | 'info'; text: string }

type Dialog =
  | { kind: 'host'; host?: Host; groupId?: string | null }
  | { kind: 'group'; group?: Group; parentId?: string | null }
  | { kind: 'settings'; section?: string }
  | { kind: 'import' }
  | { kind: 'quick' }
  | { kind: 'place'; hostId: string }
  | { kind: 'preview'; sessionId: string; path: string }
  | null

interface State {
  loaded: boolean
  groups: Group[]
  hosts: Host[]
  snippets: Snippet[]
  settings: Settings
  savedLayout: SavedLayout | null
  /** 3.6: 새 버전이 있으면 정보 */
  update: { version: string; url: string } | null
  creds: CredentialMeta[]
  vault: VaultStatus
  sessions: Record<string, SessionInfo>
  tabs: Tab[]
  activeTab: string | null
  activity: Record<string, boolean>
  transfers: Record<string, Transfer>
  /** 끌어 놓기 복사 작업 (화면 맨 아래 목록) */
  copyJobs: Record<string, CopyJob>
  requests: UiRequest[]
  confirmReq: ConfirmReq | null
  promptReq: { title: string; label?: string; value: string; resolve: (v: string | null) => void } | null
  menu: { x: number; y: number; items: MenuItem[] } | null
  toasts: Toast[]
  dialog: Dialog
  sidebar: boolean
  rightPanel: 'sftp' | 'snippets' | null
  filter: string

  init(): Promise<void>
  reloadStore(): Promise<void>
  refreshCreds(): Promise<void>
  save(p: Partial<StoreData>): void
  setSettings(p: Partial<Settings>): void
  upsertHost(h: Host): void
  removeHosts(ids: string[]): void
  upsertGroup(g: Group): void
  removeGroup(id: string): void

  openHost(hostId: string, where?: PlaceTarget): Promise<void>
  /** 3.1: 이 서버의 세션이 열려 있으면 그 탭·패널로 이동(여러 개면 순환)하고 true, 없으면 false */
  activateHost(hostId: string): boolean
  /** 3.1: 열려 있으면 이동, 없으면 새로 연다 (더블클릭) */
  openOrActivate(hostId: string): Promise<void>
  openAdhoc(t: AdhocTarget, where?: PlaceTarget): Promise<void>
  openGroupGrid(groupId: string): Promise<void>
  /** hard=true 면 빈 칸을 남기지 않고 칸 자체를 제거 (3.8: 기본은 분할 상태에서 빈 칸으로 남김) */
  closePane(paneId: string, tabId?: string, hard?: boolean): Promise<void>
  /** 3.8: 칸 머리글 드래그 — 가운데면 맞바꾸고 가장자리면 그 방향으로 옮긴다 */
  dropPane(tabId: string, srcPaneId: string, targetPaneId: string, edge: Edge | 'center'): void
  closeTab(tabId: string, force?: boolean): Promise<void>
  focusPane(tabId: string, paneId: string): void
  setActiveTab(id: string): void
  updateTab(id: string, p: Partial<Tab>): void
  moveTab(from: number, to: number): void
  gatherAll(): void
  splitToTabs(tabId: string): void
  duplicatePane(where: 'tab' | Dir): Promise<void>

  focusedSession(): SessionInfo | null
  tabOfSession(sessionId: string): Tab | undefined

  confirm(o: Omit<ConfirmReq, 'resolve'>): Promise<boolean>
  prompt(title: string, value?: string, label?: string): Promise<string | null>
  toast(kind: Toast['kind'], text: string): void
  showMenu(e: { clientX: number; clientY: number }, items: MenuItem[]): void
}

const KEEP_NOTE = '세션 유지(tmux) 중인 창입니다. 창을 닫아도 그 안의 프로그램은 계속 실행되며, 다시 열면 그 화면에 그대로 붙습니다. 완전히 끝내려면 그 창에서 exit 를 입력하세요.'

const newId = (): string => crypto.randomUUID()

export const useApp = create<State>((set, get) => ({
  loaded: false,
  groups: [],
  hosts: [],
  snippets: [],
  settings: DEFAULT_SETTINGS,
  savedLayout: null,
  update: null,
  creds: [],
  vault: { initialized: false, unlocked: false, osUnlockAvailable: false, osUnlockEnabled: false },
  sessions: {},
  tabs: [],
  activeTab: null,
  activity: {},
  transfers: {},
  copyJobs: {},
  requests: [],
  confirmReq: null,
  promptReq: null,
  menu: null,
  toasts: [],
  dialog: null,
  sidebar: true,
  rightPanel: null,
  filter: '',

  async init() {
    const [data, vault] = await Promise.all([api.store.get(), api.vault.status()])
    set({ groups: data.groups, hosts: data.hosts, snippets: data.snippets, settings: data.settings, savedLayout: data.layout ?? null, vault, loaded: true })
    await get().refreshCreds()
  },

  async reloadStore() {
    const data = await api.store.get()
    set({ groups: data.groups, hosts: data.hosts, snippets: data.snippets, settings: data.settings })
    await get().refreshCreds()
  },

  async refreshCreds() {
    try {
      set({ creds: await api.cred.list() })
    } catch { /* */ }
  },

  save(p) {
    set(p as Partial<State>)
    api.store.patch(p)
  },

  setSettings(p) {
    const settings = { ...get().settings, ...p }
    get().save({ settings })
  },

  upsertHost(h) {
    const hosts = get().hosts.some((x) => x.id === h.id) ? get().hosts.map((x) => (x.id === h.id ? h : x)) : [...get().hosts, h]
    get().save({ hosts })
  },

  removeHosts(ids) {
    get().save({ hosts: get().hosts.filter((h) => !ids.includes(h.id)) })
  },

  upsertGroup(g) {
    const groups = get().groups.some((x) => x.id === g.id) ? get().groups.map((x) => (x.id === g.id ? g : x)) : [...get().groups, g]
    get().save({ groups })
  },

  removeGroup(id) {
    const { groups, hosts } = get()
    const doomed = new Set<string>([id])
    let changed = true
    while (changed) {
      changed = false
      for (const g of groups) if (g.parentId && doomed.has(g.parentId) && !doomed.has(g.id)) { doomed.add(g.id); changed = true }
    }
    const target = groups.find((g) => g.id === id)?.parentId ?? null
    get().save({
      groups: groups.filter((g) => !doomed.has(g.id)),
      hosts: hosts.map((h) => (h.groupId && doomed.has(h.groupId) ? { ...h, groupId: target } : h))
    })
  },

  async openHost(hostId, where = 'tab') {
    try {
      const info = await api.session.open({ hostId })
      placeSession(info, where)
    } catch (e) {
      get().toast('error', (e as Error).message)
    }
  },

  activateHost(hostId) {
    const s = get()
    const hits: { tabId: string; paneId: string }[] = []
    for (const t of s.tabs) for (const p of panes(t.root)) if (s.sessions[p.sessionId]?.hostId === hostId) hits.push({ tabId: t.id, paneId: p.id })
    if (!hits.length) return false
    const curTab = s.tabs.find((t) => t.id === s.activeTab)
    const curIdx = curTab ? hits.findIndex((h) => h.tabId === curTab.id && h.paneId === curTab.focused) : -1
    const target = hits[(curIdx + 1) % hits.length] // 지금 포커스가 이 서버가 아니면 curIdx=-1 → 첫 번째
    const tab = s.tabs.find((t) => t.id === target.tabId)!
    set((st) => ({
      tabs: st.tabs.map((t) => (t.id === tab.id ? { ...t, focused: target.paneId, zoomed: t.zoomed ? target.paneId : t.zoomed } : t)),
      activeTab: tab.id
    }))
    const ps = findPane(tab.root, target.paneId)
    if (ps) set((st) => ({ activity: { ...st.activity, [ps.sessionId]: false } }))
    return true
  },

  async openOrActivate(hostId) {
    const s = get()
    const tab = s.tabs.find((t) => t.id === s.activeTab)
    const fp = tab && findPane(tab.root, tab.focused)
    // 빈 칸을 선택한 상태면 그 자리에서 연다 (3.8)
    if (tab && fp && isEmpty(fp)) return get().openHost(hostId, { tabId: tab.id, paneId: fp.id, edge: 'fill' })
    if (!get().activateHost(hostId)) await get().openHost(hostId)
  },

  async openAdhoc(t, where = 'tab') {
    try {
      const info = await api.session.open({ adhoc: t })
      placeSession(info, where)
    } catch (e) {
      get().toast('error', (e as Error).message)
    }
  },

  async openGroupGrid(groupId) {
    const { hosts, groups } = get()
    const ids = new Set<string>([groupId])
    let changed = true
    while (changed) {
      changed = false
      for (const g of groups) if (g.parentId && ids.has(g.parentId) && !ids.has(g.id)) { ids.add(g.id); changed = true }
    }
    const targets = hosts.filter((h) => h.groupId && ids.has(h.groupId)).sort((a, b) => a.sort - b.sort)
    if (!targets.length) return get().toast('info', '이 폴더에 서버가 없습니다')
    if (targets.length > 16 && !(await get().confirm({ title: '여러 서버 열기', message: `${targets.length}개 서버에 동시에 접속할까요?` }))) return
    const ps: PaneNode[] = []
    for (const h of targets) {
      try {
        const info = await api.session.open({ hostId: h.id })
        set((s) => ({ sessions: { ...s.sessions, [info.id]: info } }))
        ps.push({ type: 'pane', id: uid(), sessionId: info.id })
      } catch { /* */ }
    }
    if (!ps.length) return
    const tab: Tab = { id: uid(), root: grid(ps), focused: ps[0].id, broadcast: false, title: groups.find((g) => g.id === groupId)?.name }
    set((s) => ({ tabs: [...s.tabs, tab], activeTab: tab.id }))
  },

  async closePane(paneId, tabId, hard = false) {
    const s = get()
    const tab = s.tabs.find((t) => (tabId ? t.id === tabId : !!findPane(t.root, paneId)))
    if (!tab) return
    const pane = findPane(tab.root, paneId)
    if (!pane) return
    const ps0 = s.sessions[pane.sessionId]
    // 3.3: 세션 유지(tmux) 중인 창을 닫을 때는 안에서 계속 실행된다는 안내를 먼저 보여 준다
    if (!hard && ps0?.keepSession && ps0.state === 'connected' && !(await get().confirm({ title: '창 닫기', message: KEEP_NOTE, okText: '닫기' }))) return
    if (pane.sessionId) api.session.close(pane.sessionId)
    // 분할 상태에서 세션이 있는 칸을 닫으면 빈 칸으로 남긴다. 빈 칸을 다시 닫으면 제거.
    const soft = !hard && !isEmpty(pane) && panes(tab.root).length > 1
    const root = soft ? setPaneSession(tab.root, paneId, '') : removePane(tab.root, paneId)
    set((st) => {
      const sessions = { ...st.sessions }
      delete sessions[pane.sessionId]
      if (!root) {
        const idx = st.tabs.findIndex((t) => t.id === tab.id)
        const tabs = st.tabs.filter((t) => t.id !== tab.id)
        const activeTab = st.activeTab === tab.id ? (tabs[Math.min(idx, tabs.length - 1)]?.id ?? null) : st.activeTab
        return { tabs, activeTab, sessions }
      }
      const focused = tab.focused === paneId ? panes(root)[0].id : tab.focused
      return { tabs: st.tabs.map((t) => (t.id === tab.id ? { ...t, root, focused, zoomed: null } : t)), sessions }
    })
  },

  async closeTab(tabId, force = false) {
    const tab = get().tabs.find((t) => t.id === tabId)
    if (!tab) return
    const ps = panes(tab.root)
    const live = ps.filter((p) => get().sessions[p.sessionId]?.state === 'connected').length
    const keepLive = ps.filter((p) => { const i = get().sessions[p.sessionId]; return i?.keepSession && i.state === 'connected' }).length
    if (!force && (live > 1 || keepLive > 0)) {
      const message = (live > 1 ? `이 탭의 연결 ${live}개를 모두 닫을까요?` : '이 탭을 닫을까요?') + (keepLive > 0 ? `

${KEEP_NOTE}` : '')
      if (!(await get().confirm({ title: '탭 닫기', message, okText: '닫기', danger: live > 1 }))) return
    }
    for (const p of ps) await get().closePane(p.id, tabId, true)
  },

  dropPane(tabId, srcId, targetId, edge) {
    const tab = get().tabs.find((t) => t.id === tabId)
    if (!tab || srcId === targetId) return
    const root = edge === 'center' ? swapPanes(tab.root, srcId, targetId) : movePane(tab.root, srcId, targetId, edge)
    set((st) => ({ tabs: st.tabs.map((t) => (t.id === tabId ? { ...t, root, focused: srcId, zoomed: null } : t)) }))
  },

  focusPane(tabId, paneId) {
    set((s) => ({ tabs: s.tabs.map((t) => (t.id === tabId ? { ...t, focused: paneId } : t)), activeTab: tabId }))
    const tab = get().tabs.find((t) => t.id === tabId)
    const p = tab && findPane(tab.root, paneId)
    if (p) set((s) => ({ activity: { ...s.activity, [p.sessionId]: false } }))
  },

  setActiveTab(id) {
    set({ activeTab: id })
    const tab = get().tabs.find((t) => t.id === id)
    if (tab) {
      const act = { ...get().activity }
      for (const p of panes(tab.root)) act[p.sessionId] = false
      set({ activity: act })
    }
  },

  updateTab(id, p) {
    set((s) => ({ tabs: s.tabs.map((t) => (t.id === id ? { ...t, ...p } : t)) }))
  },

  moveTab(from, to) {
    set((s) => {
      const tabs = [...s.tabs]
      const [t] = tabs.splice(from, 1)
      tabs.splice(to, 0, t)
      return { tabs }
    })
  },

  gatherAll() {
    const all = get().tabs.flatMap((t) => panes(t.root))
    if (all.length < 2) return
    const tab: Tab = { id: uid(), root: grid(all), focused: all[0].id, broadcast: false, title: '전체 보기' }
    set({ tabs: [tab], activeTab: tab.id })
  },

  splitToTabs(tabId) {
    const tab = get().tabs.find((t) => t.id === tabId)
    if (!tab) return
    const ps = panes(tab.root)
    if (ps.length < 2) return
    const idx = get().tabs.findIndex((t) => t.id === tabId)
    const newTabs: Tab[] = ps.map((p) => ({ id: uid(), root: p, focused: p.id, broadcast: false }))
    set((s) => {
      const tabs = [...s.tabs]
      tabs.splice(idx, 1, ...newTabs)
      return { tabs, activeTab: newTabs[0].id }
    })
  },

  async duplicatePane(where) {
    const info = get().focusedSession()
    if (!info) return
    if (info.hostId) return get().openHost(info.hostId, where)
    if (info.protocol === 'local') return get().openAdhoc({ host: '', protocol: 'local', cwd: info.target.replace(/^로컬 · /, '') }, where)
    const m = info.target.match(/^(?:(.+)@)?(.+):(\d+)$/)
    if (m) return get().openAdhoc({ username: m[1], host: m[2], port: +m[3], protocol: info.protocol }, where)
  },

  focusedSession() {
    const s = get()
    const tab = s.tabs.find((t) => t.id === s.activeTab)
    if (!tab) return null
    const p = findPane(tab.root, tab.focused)
    return p ? s.sessions[p.sessionId] ?? null : null
  },

  tabOfSession(sessionId) {
    return get().tabs.find((t) => panes(t.root).some((p) => p.sessionId === sessionId))
  },

  confirm(o) {
    return new Promise((resolve) => set({ confirmReq: { ...o, resolve } }))
  },

  prompt(title, value = '', label) {
    return new Promise((resolve) => set({ promptReq: { title, value, label, resolve } }))
  },

  toast(kind, text) {
    const id = newId()
    set((s) => ({ toasts: [...s.toasts.slice(-4), { id, kind, text }] }))
    setTimeout(() => set((s) => ({ toasts: s.toasts.filter((t) => t.id !== id) })), kind === 'error' ? 6000 : 3500)
  },

  showMenu(e, items) {
    set({ menu: { x: e.clientX, y: e.clientY, items } })
  }
}))

function placeSession(info: SessionInfo, where: PlaceTarget): void {
  const st = useApp.getState()
  useApp.setState((s) => ({ sessions: { ...s.sessions, [info.id]: info } }))
  const newTab = (): void => {
    const pane: PaneNode = { type: 'pane', id: uid(), sessionId: info.id }
    const t: Tab = { id: uid(), root: pane, focused: pane.id, broadcast: false }
    useApp.setState((s) => ({ tabs: [...s.tabs, t], activeTab: t.id }))
  }
  if (typeof where === 'object') {
    const tab = st.tabs.find((t) => t.id === where.tabId)
    const target = tab && findPane(tab.root, where.paneId)
    if (!tab || !target) return newTab()
    let root: LayoutNode
    let focused = where.paneId
    if (where.edge === 'fill') {
      root = setPaneSession(tab.root, where.paneId, info.id)
    } else {
      const pane: PaneNode = { type: 'pane', id: uid(), sessionId: info.id }
      root = splitPane(tab.root, where.paneId, edgeDir(where.edge), pane, edgeBefore(where.edge))
      focused = pane.id
    }
    useApp.setState((s) => ({ tabs: s.tabs.map((t) => (t.id === tab.id ? { ...t, root, focused, zoomed: null } : t)), activeTab: tab.id }))
    return
  }
  const tab = st.tabs.find((t) => t.id === st.activeTab)
  if (where === 'tab' || !tab) return newTab()
  const pane: PaneNode = { type: 'pane', id: uid(), sessionId: info.id }
  const root = splitPane(tab.root, tab.focused, where, pane)
  useApp.setState((s) => ({ tabs: s.tabs.map((t) => (t.id === tab.id ? { ...t, root, focused: pane.id, zoomed: null } : t)) }))
}

export function newHost(groupId: string | null = null): Host {
  return {
    id: crypto.randomUUID(),
    groupId,
    alias: '',
    protocol: 'ssh',
    host: '',
    port: 22,
    username: '',
    authType: 'password',
    credentialId: null,
    tags: [],
    encoding: 'utf-8',
    termType: 'xterm-256color',
    keepaliveSec: 30,
    forwards: [],
    sort: Date.now()
  }
}
