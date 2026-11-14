// 시작 화면(탭이 하나도 없을 때 작업 영역). 근거: spec/ui/shell.md §4.
// 단축키 표(12개)는 behavior/shortcuts.md §4 가 필요하므로 4단계에서 채운다.
import { Download, Plus, Terminal, Zap } from 'lucide-react';
import { useApp } from '../state/app';
import { effectiveColor, recentHosts } from '../lib/hosts';
import { sidebarSubtitle } from '../lib/tree';
import { START_SHORTCUTS, shortcutText } from '../lib/shortcuts';
import { LogoMark } from '../ui/Logo';
import { openHost, runShortcut } from '../state/actions';
import { useUi } from '../state/ui';
import { PROTOCOL_ICON, protocolIconColor } from '../ui/protocolIcon';
import { t } from '../i18n';

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
          {t('welcome.hint').split('{key}')[0]}<span className="kbd">Ctrl+K</span>{t('welcome.hint').split('{key}')[1]}
        </p>
        <div className="welcome-actions">
          <button className="btn primary" onClick={() => useUi.getState().openQuick(null)}>
            <Zap size={15} />{t('welcome.quick')}
          </button>
          <button className="btn" title="Ctrl+Alt+T" onClick={() => runShortcut('newLocal')}>
            <Terminal size={15} />{t('host.localTerminal')}
          </button>
          <button className="btn" onClick={() => useUi.getState().openEditor({ kind: 'host', hostId: null, groupId: null })}>
            <Plus size={15} />{t('welcome.newServer')}
          </button>
          <button className="btn" onClick={() => useUi.getState().openImportDialog()}>
            <Download size={15} />{t('welcome.importPutty')}
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
                    <Icon size={14} color={protocolIconColor(h)} />
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
              <span className="muted">{shortcutText(s).name}</span>
              <span className="kbd">{shortcutText(s).keys}</span>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
