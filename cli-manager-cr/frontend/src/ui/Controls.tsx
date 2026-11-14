// 공통 입력 요소: 필드·스위치·토글 행·세그먼트·색 선택·환경 알약·키 표시.
// 근거: spec/ui/dialogs.md §1.4, spec/ui/tokens.md §3, spec/data/store.md §6.
import type { ReactNode } from 'react';
import { COLOR_PALETTE, ENV_INFO } from '../lib/defaults';
import type { EnvKind } from '../ipc/types';
import { t } from '../i18n';

export function Field({ label, help, children, htmlFor }: { label?: string; help?: string; children: ReactNode; htmlFor?: string }) {
  return (
    <div className="field">
      {label && <label htmlFor={htmlFor} className="field-label">{label}</label>}
      {children}
      {help && <div className="help">{help}</div>}
    </div>
  );
}

export function Switch({ checked, onChange, label }: { checked: boolean; onChange: (v: boolean) => void; label?: string }) {
  return (
    <button
      type="button"
      role="switch"
      aria-checked={checked}
      aria-label={label}
      className="switch"
      onClick={() => onChange(!checked)}
    />
  );
}

/** 토글 행: 왼쪽에 라벨과 설명, 오른쪽에 스위치. */
export function ToggleRow({ title, desc, checked, onChange, disabled }: {
  title: string;
  desc?: string;
  checked: boolean;
  onChange: (v: boolean) => void;
  disabled?: boolean;
}) {
  return (
    <div className="toggle-row" style={{ opacity: disabled ? 0.5 : 1 }}>
      <div>
        <div className="t-title">{title}</div>
        {desc && <div className="t-desc">{desc}</div>}
      </div>
      <Switch checked={checked} onChange={(v) => !disabled && onChange(v)} label={title} />
    </div>
  );
}

export interface SegOption<T extends string> {
  value: T;
  label: ReactNode;
}

export function Segmented<T extends string>({ value, options, onChange, label }: {
  value: T;
  options: SegOption<T>[];
  onChange: (v: T) => void;
  label?: string;
}) {
  return (
    <div className="segmented" role="group" aria-label={label}>
      {options.map((o) => (
        <button key={o.value} type="button" aria-pressed={value === o.value} onClick={() => onChange(o.value)}>
          {o.label}
        </button>
      ))}
    </div>
  );
}

/** 색 선택: 맨 앞 "자동"(값을 지움) + 팔레트 14색. */
export function ColorPicker({ value, onChange }: { value: string | undefined; onChange: (v: string | undefined) => void }) {
  const swatch = (color: string | undefined, title: string, auto = false) => {
    const selected = auto ? !value : value === color;
    return (
      <button
        key={title}
        type="button"
        title={title}
        aria-label={title}
        aria-pressed={selected}
        onClick={() => onChange(color)}
        style={{
          width: 22,
          height: 22,
          borderRadius: 6,
          border: selected ? '2px solid var(--fg)' : '1px solid var(--border-strong)',
          background: auto
            ? 'repeating-linear-gradient(45deg, var(--panel-2) 0 4px, var(--border) 4px 8px)'
            : color,
          borderStyle: auto ? 'dashed' : 'solid',
          cursor: 'pointer',
          padding: 0,
        }}
      />
    );
  };
  return (
    <div style={{ display: 'flex', gap: 6, flexWrap: 'wrap' }} role="group" aria-label={t('color.label')}>
      {swatch(undefined, t('color.auto'), true)}
      {COLOR_PALETTE.map((c) => swatch(c, c))}
    </div>
  );
}

export function EnvPill({ env }: { env: EnvKind }) {
  const info = ENV_INFO[env];
  return (
    <span className="pill" style={{ background: `${info.color}33`, color: info.color }}>{t(info.label)}</span>
  );
}

export function Kbd({ children }: { children: ReactNode }) {
  return <span className="kbd">{children}</span>;
}
