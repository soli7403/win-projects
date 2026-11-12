// 위치 골라서 연결 대화상자. 근거: spec/ui/dialogs.md §3.
// 열린 탭마다 분할 구조를 축소해 그리고, 칸의 방향 화살표(분할) 또는 "여기에"(빈 칸)로 연다.
import { ArrowDown, ArrowLeft, ArrowRight, ArrowUp } from 'lucide-react';
import type { LayoutNode } from '../lib/layout';
import { useApp } from '../state/app';
import { useUi } from '../state/ui';
import { useWorkspace, openSessionFor, tabTitle, type Tab } from '../state/workspace';
import { openHost } from '../state/actions';
import { Modal } from '../ui/Modal';
import type { Dir } from '../lib/layout';

export function LocationPicker() {
  const loc = useUi((s) => s.locationPicker);
  const close = useUi((s) => s.closeLocation);
  const tabs = useWorkspace((s) => s.tabs);
  const sessions = useWorkspace((s) => s.sessions);
  const host = useApp((s) => s.store?.hosts.find((h) => h.id === loc?.hostId));
  if (!loc || !host) return null;

  const place = async (paneId: string | null, dir: Dir, before: boolean, empty: boolean) => {
    close();
    const info = await openSessionFor(loc.hostId);
    if (!info) return;
    const ws = useWorkspace.getState();
    if (empty && paneId) ws.fillPane(paneId, info);
    else if (paneId) ws.openInPane(paneId, dir, info, before);
    else ws.openInNewTab(info);
  };

  return (
    <Modal title={`위치 골라서 연결 — ${host.alias}`} size="lg" onClose={close}
      foot={<button className="btn primary" onClick={() => { close(); void openHost(loc.hostId); }}>새 탭으로 열기</button>}>
      {tabs.length === 0 && <div className="faint">열린 탭이 없습니다. 새 탭으로 여세요.</div>}
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(280px, 1fr))', gap: 14 }}>
        {tabs.map((t) => (
          <div key={t.id} className="loc-card">
            <div className="faint small" style={{ marginBottom: 6 }}>{tabTitle(t, sessions)}</div>
            <div className="loc-mini">
              <MiniNode tab={t} node={t.root} onPlace={place} sessions={sessions} />
            </div>
          </div>
        ))}
      </div>
    </Modal>
  );
}

type Sessions = ReturnType<typeof useWorkspace.getState>['sessions'];

function MiniNode({ tab, node, onPlace, sessions }: {
  tab: Tab;
  node: LayoutNode;
  onPlace: (paneId: string | null, dir: Dir, before: boolean, empty: boolean) => void;
  sessions: Sessions;
}) {
  if (node.type === 'pane') {
    const title = node.sessionId ? sessions[node.sessionId]?.info.title ?? '세션' : null;
    if (!title) {
      return (
        <div className="loc-pane">
          <button className="btn small-btn" title="이 빈 칸에서 열기" onClick={() => onPlace(node.id, 'row', false, true)}>여기에</button>
        </div>
      );
    }
    return (
      <div className="loc-pane">
        <span className="small">{title}</span>
        <div className="loc-arrows">
          <button className="icon-btn small" title="이 방향으로 분할해 열기" aria-label="위로 분할" onClick={() => onPlace(node.id, 'col', true, false)}><ArrowUp size={12} /></button>
          <button className="icon-btn small" title="이 방향으로 분할해 열기" aria-label="왼쪽으로 분할" onClick={() => onPlace(node.id, 'row', true, false)}><ArrowLeft size={12} /></button>
          <button className="icon-btn small" title="이 방향으로 분할해 열기" aria-label="오른쪽으로 분할" onClick={() => onPlace(node.id, 'row', false, false)}><ArrowRight size={12} /></button>
          <button className="icon-btn small" title="이 방향으로 분할해 열기" aria-label="아래로 분할" onClick={() => onPlace(node.id, 'col', false, false)}><ArrowDown size={12} /></button>
        </div>
      </div>
    );
  }
  const horizontal = node.dir === 'row';
  return (
    <div style={{ display: 'flex', flexDirection: horizontal ? 'row' : 'column', gap: 3, flex: 1, minHeight: 0, minWidth: 0 }}>
      {node.children.map((c, i) => (
        <div key={c.id} style={{ flex: `0 0 calc(${node.sizes[i]}% - 2px)`, display: 'flex', minWidth: 0, minHeight: 0 }}>
          <MiniNode tab={tab} node={c} onPlace={onPlace} sessions={sessions} />
        </div>
      ))}
    </div>
  );
}
