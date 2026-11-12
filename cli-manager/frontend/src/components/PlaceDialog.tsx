import { ArrowUp, ArrowDown, ArrowLeft, ArrowRight, Plus, SquarePlus } from 'lucide-react'
import { useApp, type Tab } from '../state'
import { LayoutNode, panes, isEmpty, type Edge } from '../layout'
import { Modal } from './ui'

/** 3.8: "위치 골라서 연결…" — 열린 탭마다 현재 배치를 축소 그림으로 보여 주고, 방향 화살표/빈 칸을 눌러 열 위치를 고른다. */
export function PlaceDialog({ hostId }: { hostId: string }): JSX.Element {
  const { tabs, hosts, sessions } = useApp()
  const host = hosts.find((h) => h.id === hostId)
  const close = (): void => useApp.setState({ dialog: null })

  const choose = (tab: Tab, paneId: string, edge: Edge | 'fill'): void => {
    close()
    void useApp.getState().openHost(hostId, { tabId: tab.id, paneId, edge })
  }

  const Mini = ({ tab, node }: { tab: Tab; node: LayoutNode }): JSX.Element => {
    if (node.type === 'split') {
      return (
        <div className="mini-split" style={{ flexDirection: node.dir === 'row' ? 'row' : 'column' }}>
          {node.children.map((c, i) => (
            <div key={c.id} style={{ flex: `${node.sizes[i]} 1 0`, minWidth: 0, minHeight: 0, display: 'flex' }}>
              <Mini tab={tab} node={c} />
            </div>
          ))}
        </div>
      )
    }
    if (isEmpty(node)) {
      return (
        <button className="mini-pane mini-empty" onClick={() => choose(tab, node.id, 'fill')} title="이 빈 칸에서 열기">
          <SquarePlus size={16} /> 여기에
        </button>
      )
    }
    const arrow = (edge: Edge, icon: JSX.Element, cls: string): JSX.Element => (
      <button className={`mini-arrow ${cls}`} title="이 방향으로 분할해 열기" onClick={() => choose(tab, node.id, edge)}>{icon}</button>
    )
    return (
      <div className="mini-pane">
        <span className="mini-title">{sessions[node.sessionId]?.title ?? '세션'}</span>
        {arrow('up', <ArrowUp size={13} />, 'up')}
        {arrow('down', <ArrowDown size={13} />, 'down')}
        {arrow('left', <ArrowLeft size={13} />, 'left')}
        {arrow('right', <ArrowRight size={13} />, 'right')}
      </div>
    )
  }

  return (
    <Modal title={`위치 골라서 연결 — ${host?.alias ?? ''}`} onClose={close} size="lg">
      <div className="modal-body" style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
        <button className="btn" style={{ alignSelf: 'flex-start' }} onClick={() => { close(); void useApp.getState().openHost(hostId, 'tab') }}>
          <Plus size={14} /> 새 탭으로 열기
        </button>
        {tabs.length === 0 && <div className="muted">열린 탭이 없습니다. 새 탭으로 여세요.</div>}
        <div className="mini-grid">
          {tabs.map((t, i) => (
            <div key={t.id} className="mini-tab">
              <div className="mini-tab-title">{t.title || sessions[panes(t.root)[0]?.sessionId]?.title || `탭 ${i + 1}`}</div>
              <div className="mini-canvas"><Mini tab={t} node={t.root} /></div>
            </div>
          ))}
        </div>
      </div>
    </Modal>
  )
}
