import { useEffect, useLayoutEffect, useRef, useState } from 'react'
import { Columns2, Rows2, X, Maximize2, Minimize2, FolderTree, RotateCw, Search, ChevronUp, ChevronDown, ImageUp, Radio } from 'lucide-react'
import { useApp, Tab } from '../state'
import { LayoutNode, PaneNode, panes, setSizes, isEmpty, type Edge } from '../layout'
import { attach, detach, fitEntry, getEntry, smartPaste, uploadAndInsert, disposeEntry } from '../terms'
import { api } from '../api'
import { ENV_COLORS, ENV_LABELS } from '@shared/types'
import { TERMINAL_THEMES } from '../themes'
import { Welcome } from './Welcome'
import { SC } from '../platform'
import { isFsDrag, hoverStart, hoverEnd, dropFsDrag } from '../fsDrag'

export function Workspace(): JSX.Element {
  const tabs = useApp((s) => s.tabs)
  const activeTab = useApp((s) => s.activeTab)
  return (
    <div className="workspace">
      {tabs.length === 0 && <Welcome />}
      {tabs.map((t) => (
        <div key={t.id} className={`tab-view ${t.id === activeTab ? '' : 'hidden'}`}>
          <TabView tab={t} active={t.id === activeTab} />
        </div>
      ))}
    </div>
  )
}

function TabView({ tab, active }: { tab: Tab; active: boolean }): JSX.Element {
  useEffect(() => {
    if (!active) return
    const id = requestAnimationFrame(() => {
      for (const p of panes(tab.root)) {
        const e = getEntry(p.sessionId)
        if (e) fitEntry(e)
      }
      const fp = panes(tab.root).find((p) => p.id === tab.focused)
      if (fp) getEntry(fp.sessionId)?.term.focus()
    })
    return () => cancelAnimationFrame(id)
  }, [active, tab.root, tab.focused, tab.zoomed])
  if (tab.zoomed) {
    const p = panes(tab.root).find((x) => x.id === tab.zoomed)
    if (p) return isEmpty(p) ? <EmptyPane tab={tab} pane={p} /> : <Pane tab={tab} pane={p} />
  }
  return <Node tab={tab} node={tab.root} />
}

function Node({ tab, node }: { tab: Tab; node: LayoutNode }): JSX.Element {
  if (node.type === 'pane') return isEmpty(node) ? <EmptyPane tab={tab} pane={node} /> : <Pane tab={tab} pane={node} />
  return <Split tab={tab} node={node} />
}

function Split({ tab, node }: { tab: Tab; node: Extract<LayoutNode, { type: 'split' }> }): JSX.Element {
  const ref = useRef<HTMLDivElement>(null)
  const [drag, setDrag] = useState<number | null>(null)
  const startDrag = (i: number, e: React.MouseEvent<HTMLElement>): void => {
    e.preventDefault()
    const el = ref.current
    if (!el) return
    const rect = el.getBoundingClientRect()
    const total = node.dir === 'row' ? rect.width : rect.height
    const start = node.dir === 'row' ? e.clientX : e.clientY
    const s0 = [...node.sizes]
    setDrag(i)
    const move = (ev: MouseEvent): void => {
      const delta = (((node.dir === 'row' ? ev.clientX : ev.clientY) - start) / total) * 100
      const a = Math.max(8, s0[i] + delta)
      const b = Math.max(8, s0[i] + s0[i + 1] - a)
      const sizes = [...s0]
      sizes[i] = s0[i] + s0[i + 1] - b
      sizes[i + 1] = b
      const st = useApp.getState()
      const t = st.tabs.find((x) => x.id === tab.id)
      if (t) st.updateTab(tab.id, { root: setSizes(t.root, node.id, sizes) })
    }
    const up = (): void => {
      setDrag(null)
      window.removeEventListener('mousemove', move)
      window.removeEventListener('mouseup', up)
    }
    window.addEventListener('mousemove', move)
    window.addEventListener('mouseup', up)
  }
  return (
    <div ref={ref} className={`split s-${node.dir}`}>
      {node.children.map((c, i) => (
        <SplitChild key={c.id} size={node.sizes[i]} last={i === node.children.length - 1} dragging={drag === i} onDrag={(e) => startDrag(i, e)}>
          <Node tab={tab} node={c} />
        </SplitChild>
      ))}
    </div>
  )
}

function SplitChild(props: { size: number; last: boolean; dragging: boolean; onDrag: (e: React.MouseEvent<HTMLElement>) => void; children: React.ReactNode }): JSX.Element {
  return (
    <>
      <div className="split-child" style={{ flex: `${props.size} 1 0` }}>{props.children}</div>
      {!props.last && <div className={`divider ${props.dragging ? 'drag' : ''}`} onMouseDown={props.onDrag} />}
    </>
  )
}

function Pane({ tab, pane }: { tab: Tab; pane: PaneNode }): JSX.Element {
  const info = useApp((s) => s.sessions[pane.sessionId])
  const settings = useApp((s) => s.settings)
  const focused = tab.focused === pane.id
  const multi = panes(tab.root).length > 1
  const bodyRef = useRef<HTMLDivElement>(null)
  const [dropping, setDropping] = useState(false)
  const dnd = usePaneDnd(tab, pane)
  const [find, setFind] = useState<string | null>(null)
  const findRef = useRef<HTMLInputElement>(null)
  const st = useApp.getState

  useLayoutEffect(() => {
    const el = bodyRef.current
    if (!el) return
    const e = attach(pane.sessionId, el)
    const ro = new ResizeObserver(() => fitEntry(e))
    ro.observe(el)
    return () => {
      ro.disconnect()
      detach(pane.sessionId, el)
    }
  }, [pane.sessionId])

  // dispose terminal when session is gone for good
  useEffect(() => () => {
    setTimeout(() => {
      if (!useApp.getState().sessions[pane.sessionId]) disposeEntry(pane.sessionId)
    }, 0)
  }, [pane.sessionId])

  useEffect(() => {
    const h = (ev: Event): void => {
      const d = (ev as CustomEvent).detail
      if (d === pane.sessionId) {
        setFind('')
        setTimeout(() => findRef.current?.focus(), 20)
      }
    }
    window.addEventListener('bt:find', h)
    return () => window.removeEventListener('bt:find', h)
  }, [pane.sessionId])

  const color = info?.color
  const env = info?.env && info.env !== 'none' ? info.env : null
  const termBg = TERMINAL_THEMES[settings.terminalTheme]?.background ?? '#0d1117'

  const onContext = (e: React.MouseEvent<HTMLElement>): void => {
    e.preventDefault()
    const entry = getEntry(pane.sessionId)
    const hasSel = !!entry?.term.hasSelection()
    if (settings.rightClickPaste && !e.shiftKey) {
      if (hasSel && !settings.copyOnSelect) {
        api.clip.writeText(entry!.term.getSelection())
        entry!.term.clearSelection()
        return
      }
      smartPaste(pane.sessionId)
      return
    }
    st().showMenu(e, [
      { label: '복사', shortcut: SC.copy, disabled: !hasSel, onClick: () => entry && api.clip.writeText(entry.term.getSelection()) },
      { label: '붙여넣기', shortcut: SC.paste, onClick: () => smartPaste(pane.sessionId) },
      { label: info?.protocol === 'local' ? '클립보드 이미지 붙여넣기 (경로 입력)' : '클립보드 이미지를 서버로 붙여넣기', shortcut: SC.pasteImage, disabled: !info?.canSftp && info?.protocol !== 'local', onClick: () => smartPaste(pane.sessionId, true) },
      { separator: true },
      { label: '모두 선택', onClick: () => entry?.term.selectAll() },
      { label: '화면 지우기', onClick: () => entry?.term.clear() },
      { label: '찾기', shortcut: SC.find, onClick: () => window.dispatchEvent(new CustomEvent('bt:find', { detail: pane.sessionId })) },
      { separator: true },
      { label: '오른쪽으로 분할 (같은 서버)', shortcut: SC.splitRight, onClick: () => st().duplicatePane('row') },
      { label: '아래로 분할 (같은 서버)', shortcut: SC.splitDown, onClick: () => st().duplicatePane('col') },
      { label: info?.protocol === 'local' ? '파일 탐색 열기' : 'SFTP 열기', shortcut: SC.sftp, disabled: !info?.canSftp && info?.protocol !== 'local', onClick: () => useApp.setState({ rightPanel: 'sftp' }) },
      { label: '다시 접속', disabled: info?.state === 'connected', onClick: () => api.session.reconnect(pane.sessionId) },
      { separator: true },
      { label: '패널 닫기', danger: true, onClick: () => st().closePane(pane.id, tab.id) }
    ])
  }

  const doFind = (dir: 'next' | 'prev'): void => {
    const e = getEntry(pane.sessionId)
    if (!e || !find) return
    const opt = { caseSensitive: false, decorations: { matchOverviewRuler: '#f59e0b', activeMatchColorOverviewRuler: '#ef4444', matchBackground: '#f59e0b55', activeMatchBackground: '#ef4444aa' } }
    if (dir === 'next') e.search.findNext(find, opt)
    else e.search.findPrevious(find, opt)
  }

  return (
    <div
      className={`pane ${focused ? 'focused' : ''} ${env === 'prod' ? 'prod' : ''}`}
      data-pane={pane.id}
      style={{ ['--pane-color' as string]: color ?? 'var(--accent)', ['--term-bg' as string]: termBg }}
      onMouseDown={() => { if (!focused || useApp.getState().activeTab !== tab.id) st().focusPane(tab.id, pane.id) }}
      onDragOver={(e) => { if (hasDnd(e)) return dnd.handlers.onDragOver?.(e); if (e.dataTransfer!.types.includes('Files')) { e.preventDefault(); setDropping(true) } }}
      onDragLeave={(e) => { dnd.handlers.onDragLeave?.(e); if (!e.currentTarget.contains(e.relatedTarget as Node)) setDropping(false) }}
      onDrop={(e) => {
        if (hasDnd(e)) return dnd.handlers.onDrop?.(e)
        e.preventDefault()
        setDropping(false)
        const paths = Array.from(e.dataTransfer!.files).map((f) => api.pathForFile(f)).filter(Boolean)
        if (paths.length) uploadAndInsert(pane.sessionId, { kind: 'files', paths })
      }}
    >
      {(
        <div className="pane-head" draggable={multi} onDragStart={(e) => { e.dataTransfer!.setData('bt/pane', `${tab.id}:${pane.id}`); e.dataTransfer!.effectAllowed = 'move' }} style={color ? { background: `color-mix(in srgb, ${color} ${focused ? 22 : 12}%, var(--bg-3))`, borderBottomColor: color } : undefined}>
          <span className={`dot ${info?.state ?? 'closed'}`} />
          {env && <span className="env-tag" style={{ background: ENV_COLORS[env] }}>{ENV_LABELS[env]}</span>}
          {info?.keepSession && <span className="keep-badge" title="tmux 세션 유지 중 — 창을 닫아도 안의 프로그램은 계속 실행됩니다">유지</span>}
          <span className="ph-title">{info?.title ?? '세션'}</span>
          <span className="ph-target">{info?.target}</span>
          {tab.broadcast && <Radio size={12} color="var(--warn)" />}
          <span className="ph-actions">
            {info && info.state !== 'connected' && <button className="ibtn sm" title="다시 접속" onClick={() => api.session.reconnect(pane.sessionId)}><RotateCw size={13} /></button>}
            {(info?.canSftp || info?.protocol === 'local') && <button className="ibtn sm" title={info?.protocol === 'local' ? `클립보드 이미지를 저장하고 경로 입력 (${SC.pasteImage})` : `클립보드 이미지 → 서버 업로드 후 경로 입력 (${SC.pasteImage})`} onClick={() => smartPaste(pane.sessionId, true)}><ImageUp size={13} /></button>}
            <button className="ibtn sm" title={`찾기 (${SC.find})`} onClick={() => { setFind(''); setTimeout(() => findRef.current?.focus(), 20) }}><Search size={13} /></button>
            {(info?.canSftp || info?.protocol === 'local') && <button className="ibtn sm" title={info?.protocol === 'local' ? `파일 탐색 (${SC.sftp})` : `SFTP (${SC.sftp})`} onClick={() => { st().focusPane(tab.id, pane.id); useApp.setState({ rightPanel: 'sftp' }) }}><FolderTree size={13} /></button>}
            <button className="ibtn sm" title={`오른쪽으로 분할 (${SC.splitRight})`} onClick={() => { st().focusPane(tab.id, pane.id); st().duplicatePane('row') }}><Columns2 size={13} /></button>
            <button className="ibtn sm" title={`아래로 분할 (${SC.splitDown})`} onClick={() => { st().focusPane(tab.id, pane.id); st().duplicatePane('col') }}><Rows2 size={13} /></button>
            {multi && (
              <button className="ibtn sm" title={tab.zoomed ? `원래대로 (${SC.zoom})` : `크게 보기 (${SC.zoom})`} onClick={() => st().updateTab(tab.id, { zoomed: tab.zoomed ? null : pane.id })}>
                {tab.zoomed ? <Minimize2 size={13} /> : <Maximize2 size={13} />}
              </button>
            )}
            <button className="ibtn sm" title="닫기" onClick={() => st().closePane(pane.id, tab.id)}><X size={13} /></button>
          </span>
        </div>
      )}
      <div className="pane-body" ref={bodyRef} onContextMenu={onContext} />
      {multi && info && <div className="watermark" style={{ color: color ?? 'var(--text)' }}>{info.title}</div>}
      {find !== null && (
        <div className="findbar" onMouseDown={(e) => e.stopPropagation()}>
          <input ref={findRef} className="input" placeholder="터미널에서 찾기" value={find}
            onChange={(e) => setFind(e.currentTarget.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter') doFind(e.shiftKey ? 'prev' : 'next')
              if (e.key === 'Escape') { setFind(null); getEntry(pane.sessionId)?.search.clearDecorations(); getEntry(pane.sessionId)?.term.focus() }
            }} />
          <button className="ibtn sm" onClick={() => doFind('prev')}><ChevronUp size={14} /></button>
          <button className="ibtn sm" onClick={() => doFind('next')}><ChevronDown size={14} /></button>
          <button className="ibtn sm" onClick={() => { setFind(null); getEntry(pane.sessionId)?.search.clearDecorations() }}><X size={14} /></button>
        </div>
      )}
      <DropOverlay zone={dnd.zone} />
      {dropping && (
        <div className="pane-drop"><div><ImageUp size={18} /> {info?.canSftp ? `서버(${settings.uploadDir})로 업로드하고 경로를 입력합니다` : info?.protocol === 'local' ? '파일 경로를 입력합니다 (이미지는 저장 후 경로)' : '파일 경로를 입력합니다'}</div></div>
      )}
    </div>
  )
}

// ---------- 3.8: 칸 드래그·끌어 놓기 ----------

type Zone = Edge | 'center'

/** 칸 위 마우스 위치 → 가운데 / 가장자리(상하좌우) */
function zoneAt(e: React.DragEvent<HTMLElement>, el: HTMLElement): Zone {
  const r = el.getBoundingClientRect()
  const x = (e.clientX - r.left) / r.width
  const y = (e.clientY - r.top) / r.height
  const m = 0.22
  const d = [x, 1 - x, y, 1 - y]
  const min = Math.min(...d)
  if (min > m) return 'center'
  const i = d.indexOf(min)
  return (['left', 'right', 'up', 'down'] as const)[i]
}

const hasDnd = (e: React.DragEvent<HTMLElement>): boolean => e.dataTransfer!.types.includes('bt/pane') || e.dataTransfer!.types.includes('bt/host')

/** 칸 하나에 칸 머리글/사이드바 서버를 끌어 놓는 동작. empty 칸은 가운데 놓기 = 그 자리에서 열기 */
function usePaneDnd(tab: Tab, pane: PaneNode): { zone: Zone | null; handlers: React.HTMLAttributes<HTMLDivElement> } {
  const [zone, setZone] = useState<Zone | null>(null)
  const handlers: React.HTMLAttributes<HTMLDivElement> = {
    onDragOver: (e) => {
      if (isFsDrag(e)) {
        // 파일 패널 항목: 머물면 이 칸 포커스(파일 패널이 이 세션으로 바뀜), 놓으면 이 세션의 마지막 폴더로 복사
        if (isEmpty(pane)) return
        e.preventDefault()
        e.stopPropagation()
        e.dataTransfer!.dropEffect = 'copy'
        if (tab.focused !== pane.id) hoverStart(`pane:${pane.id}`, () => useApp.getState().focusPane(tab.id, pane.id))
        return
      }
      if (!hasDnd(e)) return
      e.preventDefault()
      e.stopPropagation()
      setZone(zoneAt(e, e.currentTarget))
    },
    onDragLeave: (e) => {
      if (!e.currentTarget.contains(e.relatedTarget as Node)) { setZone(null); hoverEnd(`pane:${pane.id}`) }
    },
    onDrop: (e) => {
      if (isFsDrag(e)) {
        if (isEmpty(pane)) return
        e.preventDefault()
        e.stopPropagation()
        void dropFsDrag(pane.sessionId, '')
        return
      }
      if (!hasDnd(e)) return
      e.preventDefault()
      e.stopPropagation()
      const z = zoneAt(e, e.currentTarget)
      setZone(null)
      const st = useApp.getState()
      const srcPane = e.dataTransfer!.getData('bt/pane')
      const hostId = e.dataTransfer!.getData('bt/host')
      if (srcPane) {
        const [tabId, paneId] = srcPane.split(':')
        if (tabId === tab.id) st.dropPane(tab.id, paneId, pane.id, z)
      } else if (hostId) {
        if (isEmpty(pane) && z === 'center') void st.openHost(hostId, { tabId: tab.id, paneId: pane.id, edge: 'fill' })
        else if (z !== 'center') void st.openHost(hostId, { tabId: tab.id, paneId: pane.id, edge: z })
      }
    }
  }
  return { zone, handlers }
}

function DropOverlay({ zone }: { zone: Zone | null }): JSX.Element | null {
  if (!zone) return null
  return <div className={`dnd-zone z-${zone}`} />
}

function EmptyPane({ tab, pane }: { tab: Tab; pane: PaneNode }): JSX.Element {
  const focused = tab.focused === pane.id
  const { zone, handlers } = usePaneDnd(tab, pane)
  const st = useApp.getState
  return (
    <div
      className={`pane empty ${focused ? 'focused' : ''}`}
      data-pane={pane.id}
      onMouseDown={() => st().focusPane(tab.id, pane.id)}
      {...handlers}
    >
      <div className="pane-head">
        <span className="dot closed" />
        <span className="ph-title">빈 칸</span>
        <span className="ph-actions">
          <button className="ibtn sm" title="닫기 (칸 제거)" onClick={() => st().closePane(pane.id, tab.id)}><X size={13} /></button>
        </span>
      </div>
      <div className="empty-body">
        <div>서버를 이 칸으로 끌어다 놓거나,<br />이 칸을 선택하고 사이드바에서 서버를 더블클릭하세요</div>
        <button className="btn sm" onClick={() => useApp.setState({ dialog: { kind: 'quick' } })}>빠른 접속…</button>
      </div>
      <DropOverlay zone={zone} />
    </div>
  )
}
