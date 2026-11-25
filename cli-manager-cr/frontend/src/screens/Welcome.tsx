// 시작 화면(탭이 하나도 없을 때 작업 영역). 근거: spec/ui/shell.md §4.
// 단축키 표(12줄)와 안내의 키는 지금 설정의 키를 보인다(behavior/shortcuts.md §4·§5, D126).
import { Download, Plus, Terminal, Zap } from 'lucide-react';
import { useApp } from '../state/app';
import { effectiveColor, recentHosts } from '../lib/hosts';
import { sidebarSubtitle } from '../lib/tree';
import { startShortcutRows } from '../lib/shortcuts';
import { useShortcutLabels } from '../state/shortcuts';
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
  const labels = useShortcutLabels();
  // 안내 문장의 키: 빠른 연결을 여는 키 중 지금 있는 것. 하나도 없으면(끄거나 비움) 문장을 보이지 않는다.
  const quickKey = labels.quick || labels.newTab || labels.quickAlt || labels.quickOpen;

  return (
    <div className="welcome">
      <div className="welcome-card">
        <LogoMark size={48} />
        <h1>DariTerm</h1>
        {quickKey && (
          <p className="muted" style={{ margin: 0 }}>
            {t('welcome.hint').split('{key}')[0]}<span className="kbd">{quickKey}</span>{t('welcome.hint').split('{key}')[1]}
          </p>
        )}
        <div className="welcome-actions">
          <button className="btn primary" onClick={() => useUi.getState().openQuick(null)}>
            <Zap size={15} />{t('welcome.quick')}
          </button>
          <button className="btn" title={labels.newLocal || undefined} onClick={() => runShortcut('newLocal')}>
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
          {startShortcutRows(labels).map((s) => (
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
