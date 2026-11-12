// 시작 화면(탭이 하나도 없을 때 작업 영역). 근거: spec/ui/shell.md §4.
// 단축키 표(12개)는 behavior/shortcuts.md §4 가 필요하므로 4단계에서 채운다.
import { Download, Plus, Terminal, Zap, Network, Usb, Monitor } from 'lucide-react';
import type { Protocol } from '../ipc/types';
import { useApp } from '../state/app';
import { effectiveColor, recentHosts } from '../lib/hosts';
import { sidebarSubtitle } from '../lib/tree';
import { START_SHORTCUTS } from '../lib/shortcuts';
import { LogoMark } from '../ui/Logo';
import { openHost, runShortcut } from '../state/actions';
import { useUi } from '../state/ui';

const PROTOCOL_ICON: Record<Protocol, typeof Terminal> = {
  ssh: Terminal,
  telnet: Network,
  serial: Usb,
  local: Monitor,
};

export function Welcome() {
  const store = useApp((s) => s.store);
  const hosts = store?.hosts ?? [];
  const groups = store?.groups ?? [];
  const recent = recentHosts(hosts);

  return (
    <div className="welcome">
      <div className="welcome-card">
        <LogoMark size={48} />
        <h1>DariTerm</h1>
        <p className="muted" style={{ margin: 0 }}>
          서버를 더블클릭하거나 <span className="kbd">Ctrl+K</span> 로 바로 접속하세요.
        </p>
        <div className="welcome-actions">
          <button className="btn primary" onClick={() => useUi.getState().openQuick(null)}>
            <Zap size={15} />빠른 접속
          </button>
          <button className="btn" title="Ctrl+Alt+T" onClick={() => runShortcut('newLocal')}>
            <Terminal size={15} />로컬 터미널
          </button>
          <button className="btn" onClick={() => useUi.getState().openEditor({ kind: 'host', hostId: null, groupId: null })}>
            <Plus size={15} />새 서버 등록
          </button>
          <button className="btn" onClick={() => useUi.getState().openImportDialog()}>
            <Download size={15} />PuTTY에서 가져오기
          </button>
        </div>
        {recent.length > 0 && (
          <div className="recent-grid">
            {recent.map((h) => {
              const Icon = PROTOCOL_ICON[h.protocol];
              const color = effectiveColor(h, groups) ?? 'var(--border-strong)';
              return (
                <button key={h.id} className="recent-card" onClick={() => void openHost(h.id)}>
                  <span className="strip" style={{ background: color }} />
                  <div style={{ display: 'flex', alignItems: 'center', gap: 6, fontWeight: 700 }}>
                    <Icon size={14} />
                    <span className="selectable" style={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{h.alias}</span>
                  </div>
                  <div className="faint small selectable">{sidebarSubtitle(h)}</div>
                </button>
              );
            })}
          </div>
        )}
        <div className="shortcut-table" style={{ gridTemplateColumns: '1fr 1fr', width: '100%', gap: '6px 22px' }}>
          {START_SHORTCUTS.map((s) => (
            <div key={s.name} style={{ display: 'flex', justifyContent: 'space-between', gap: 8 }}>
              <span className="muted">{s.name}</span>
              <span className="kbd">{s.keys}</span>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
