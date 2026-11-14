// 설정 화면: 크게 보기 모달, 왼쪽 구역 내비 8개 + 오른쪽 본문. 모든 변경은 즉시 저장. 근거: spec/ui/settings.md.
import { useState, type ReactNode, type KeyboardEvent } from 'react';
import { Palette, Keyboard, ImageUp, Plug, RefreshCw, Shield, Archive, Info } from 'lucide-react';
import { Modal } from '../ui/Modal';
import { useUi, type SettingsSection } from '../state/ui';
import { useApp } from '../state/app';
import { saveSettings } from '../state/settings';
import type { Settings as SettingsT } from '../ipc/types';
import { SettingsAppearance } from './SettingsAppearance';
import { SettingsInput } from './SettingsInput';
import { SettingsFiles } from './SettingsFiles';
import { SettingsConnection } from './SettingsConnection';
import { SettingsSync } from './SettingsSync';
import { SettingsSecurity } from './SettingsSecurity';
import { SettingsBackup } from './SettingsBackup';
import { SettingsAbout } from './SettingsAbout';
import '../styles/settings.css';
import { t, type MsgId } from '../i18n';

const SECTIONS: { id: SettingsSection; label: MsgId; icon: typeof Palette }[] = [
  { id: 'appearance', label: 'settings.sec.appearance', icon: Palette },
  { id: 'input', label: 'settings.sec.input', icon: Keyboard },
  { id: 'files', label: 'settings.sec.files', icon: ImageUp },
  { id: 'connection', label: 'settings.sec.connection', icon: Plug },
  { id: 'sync', label: 'settings.sec.sync', icon: RefreshCw },
  { id: 'security', label: 'settings.sec.security', icon: Shield },
  { id: 'backup', label: 'settings.sec.backup', icon: Archive },
  { id: 'about', label: 'settings.sec.about', icon: Info },
];

/** 숫자 입력: 입력 중에는 글자 그대로 두고, 떠날 때(blur)·Enter 에 범위로 보정해 저장한다. */
export function NumberField({ value, min, max, fallback, step, onCommit, label, width }: {
  value: number;
  min: number;
  max: number;
  fallback: number;
  step?: number;
  onCommit: (v: number) => void;
  label?: string;
  width?: number | string;
}) {
  const [draft, setDraft] = useState(String(value));
  const [prev, setPrev] = useState(value);
  // 외부에서 값이 바뀌면(다른 곳 저장) 입력 칸도 맞춘다.
  if (prev !== value) {
    setPrev(value);
    setDraft(String(value));
  }
  const commit = () => {
    const trimmed = draft.trim();
    let n = trimmed === '' ? fallback : Number(trimmed);
    if (!Number.isFinite(n)) n = fallback;
    n = Math.min(max, Math.max(min, n));
    if (step) n = Math.round(n / step) * step;
    if (step) n = Number(n.toFixed(2));
    setDraft(String(n));
    if (n !== value) onCommit(n);
  };
  const onKey = (e: KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter') (e.target as HTMLInputElement).blur();
  };
  return (
    <input
      className="input"
      type="number"
      aria-label={label}
      style={{ width }}
      min={min}
      max={max}
      step={step}
      value={draft}
      onChange={(e) => setDraft(e.target.value)}
      onBlur={commit}
      onKeyDown={onKey}
    />
  );
}

/** 설정 한 항목을 저장하는 얇은 도우미. 값이 같으면 저장하지 않는다. */
export function useSetting<K extends keyof SettingsT>(key: K): [SettingsT[K], (v: SettingsT[K]) => void] {
  const value = useApp((s) => s.store?.settings[key]) as SettingsT[K];
  return [value, (v) => void saveSettings({ [key]: v } as Partial<SettingsT>)];
}

export function Group({ children }: { children: ReactNode }) {
  return <div className="set-card">{children}</div>;
}

export function Settings() {
  const open = useUi((s) => s.settings.open);
  const section = useUi((s) => s.settings.section);
  const setSection = (id: SettingsSection) => useUi.setState((s) => ({ settings: { ...s.settings, section: id } }));
  const close = useUi((s) => s.closeSettings);
  const settings = useApp((s) => s.store?.settings);
  if (!open || !settings) return null;
  const body: Record<SettingsSection, ReactNode> = {
    appearance: <SettingsAppearance />,
    input: <SettingsInput />,
    files: <SettingsFiles />,
    connection: <SettingsConnection />,
    sync: <SettingsSync />,
    security: <SettingsSecurity />,
    backup: <SettingsBackup />,
    about: <SettingsAbout />,
  };
  return (
    <Modal title={t('settings.title')} size="lg" onClose={close} hideClose={false}>
      <div className="set-wrap">
        <nav className="set-nav" aria-label={t('settings.nav')}>
          {SECTIONS.map((s) => {
            const Icon = s.icon;
            return (
              <button
                key={s.id}
                type="button"
                className={`set-nav-item${section === s.id ? ' on' : ''}`}
                aria-pressed={section === s.id}
                onClick={() => setSection(s.id)}
              >
                <Icon size={15} />
                {t(s.label)}
              </button>
            );
          })}
        </nav>
        <div className="set-body">{body[section]}</div>
      </div>
    </Modal>
  );
}
