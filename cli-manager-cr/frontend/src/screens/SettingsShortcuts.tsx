// 설정 · 단축키(화면 순서 3번째). "정보"에 있던 단축키 표를 떼어 낸 구역 (D126).
// 전체 켜기/끄기, 명령마다 키 바꾸기(다른 명령이 쓰던 키를 고르면 그 명령은 빈 값으로 저장), 기본값으로 되돌리기.
// 근거: spec/ui/settings.md §2a, spec/behavior/shortcuts.md §5.
import { useEffect, useState } from 'react';
import { RotateCcw, X } from 'lucide-react';
import { ToggleRow } from '../ui/Controls';
import { useApp } from '../state/app';
import { saveSettings } from '../state/settings';
import { useSetting, Group } from './Settings';
import { SHORTCUT_COMMANDS, assignShortcut, comboOf, comboProblem, comboText, effectiveKeys, keyCapture, type ShortcutOverrides } from '../lib/keys';
import { FIXED_SHORTCUTS, shortcutText } from '../lib/shortcuts';
import { t } from '../i18n';

export function SettingsShortcuts() {
  const [enabledRaw, setEnabled] = useSetting('shortcutsEnabled');
  const enabled = enabledRaw !== false;
  const saved = useApp((s) => s.store?.settings.shortcuts) as ShortcutOverrides | undefined;
  const pushToast = useApp((s) => s.pushToast);
  /** 키를 받는 중인 명령 id. */
  const [recording, setRecording] = useState<string | null>(null);
  const keys = effectiveKeys(saved);
  const changed = SHORTCUT_COMMANDS.some((c) => keys[c.id] !== c.keys);

  const nameOf = (id: string) => {
    const c = SHORTCUT_COMMANDS.find((x) => x.id === id);
    return c ? t(c.label, c.labelVars) : id;
  };

  const assign = (id: string, combo: string) => {
    const r = assignShortcut(saved, id, combo);
    void saveSettings({ shortcuts: r.overrides });
    for (const other of r.displaced) pushToast({ kind: 'info', text: t('sc.displaced', { keys: comboText(combo), name: nameOf(other) }) });
  };

  // 키를 받는 동안: 모든 키 입력을 여기서 가로챈다(앱 단축키·모달의 Esc 닫기는 keyCapture 를 보고 쉰다).
  useEffect(() => {
    if (!recording) return undefined;
    keyCapture.active = true;
    const onKey = (e: KeyboardEvent) => {
      e.preventDefault();
      e.stopPropagation();
      if (e.code === 'Escape' && !e.ctrlKey && !e.altKey && !e.shiftKey) {
        setRecording(null);
        return;
      }
      const combo = comboOf(e);
      if (!combo) return; // Ctrl·Shift·Alt 만 누른 상태: 다음 키를 기다린다
      const problem = comboProblem(combo);
      if (problem) {
        pushToast({ kind: 'error', text: problem === 'reserved' ? t('sc.errReserved', { keys: comboText(combo) }) : t('sc.errModifier') });
        return;
      }
      assign(recording, combo);
      setRecording(null);
    };
    const cancel = () => setRecording(null);
    window.addEventListener('keydown', onKey, true);
    window.addEventListener('mousedown', cancel, true);
    window.addEventListener('blur', cancel);
    return () => {
      keyCapture.active = false;
      window.removeEventListener('keydown', onKey, true);
      window.removeEventListener('mousedown', cancel, true);
      window.removeEventListener('blur', cancel);
    };
    // assign 은 saved 를 읽으므로 saved 가 바뀌면 다시 건다.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [recording, saved]);

  return (
    <>
      <Group>
        <ToggleRow title={t('sc.enable')} desc={t('sc.enableDesc')} checked={enabled} onChange={setEnabled} />
      </Group>

      <div>
        <div className="sc-head">
          <div className="set-section-title" style={{ marginBottom: 0 }}>{t('sc.listTitle')}</div>
          <button className="btn small-btn" type="button" disabled={!changed} onClick={() => void saveSettings({ shortcuts: {} })}>
            {t('sc.resetAll')}
          </button>
        </div>
        <div className="help" style={{ marginBottom: 8 }}>{t('sc.editHint')}</div>
        <div className={`sc-edit${enabled ? '' : ' off'}`}>
          {SHORTCUT_COMMANDS.map((c) => {
            const name = t(c.label, c.labelVars);
            const text = comboText(keys[c.id]);
            const isRec = recording === c.id;
            return (
              <div className="sc-edit-row" key={c.id}>
                <span className="sc-name">{name}</span>
                <button
                  type="button"
                  className={`sc-key${isRec ? ' rec' : ''}${text ? '' : ' none'}`}
                  aria-label={`${name}: ${isRec ? t('sc.press') : text || t('sc.none')}`}
                  // 다른 곳(다른 줄 포함)을 누르면 창의 mousedown 이 먼저 받기를 끝내고, 그 뒤 이 click 이 새로 시작한다.
                  onClick={() => { if (!isRec) setRecording(c.id); }}
                >
                  {isRec ? t('sc.press') : text || t('sc.none')}
                </button>
                <button
                  type="button"
                  className="icon-btn"
                  title={t('sc.clear')}
                  aria-label={`${name}: ${t('sc.clear')}`}
                  disabled={!keys[c.id]}
                  onClick={() => assign(c.id, '')}
                >
                  <X size={14} />
                </button>
                <button
                  type="button"
                  className="icon-btn"
                  title={`${t('sc.reset')} (${comboText(c.keys)})`}
                  aria-label={`${name}: ${t('sc.reset')}`}
                  disabled={keys[c.id] === c.keys}
                  onClick={() => assign(c.id, c.keys)}
                >
                  <RotateCcw size={13} />
                </button>
              </div>
            );
          })}
        </div>
      </div>

      <div>
        <div className="set-section-title">{t('sc.fixedTitle')}</div>
        <div className="help" style={{ marginBottom: 8 }}>{t('sc.fixedDesc')}</div>
        <div className="sc-table">
          {FIXED_SHORTCUTS.map((s) => {
            const x = shortcutText(s);
            return (
              <span key={s.name} style={{ display: 'contents' }}>
                <span className="sc-name">{x.name}</span>
                <span className="kbd">{x.keys}</span>
              </span>
            );
          })}
        </div>
      </div>
    </>
  );
}
