// 서버 편집의 큰 부분 조각: 포트 포워딩 표, 시리얼 옵션, 점프 호스트 카드. 근거: spec/ui/host-editor.md §1.1(3), §1.3(1), §1.4.
import { Trash2 } from 'lucide-react';
import type { Forward } from '../ipc/types';
import { BAUD_RATES, PROXY_DEFAULT_PORT, type JumpDirect, type ProxyType, type SerialEnter, type SerialFlow, type SerialOpts, type SerialParity } from '../lib/hostForm';
import { Field, Segmented, Switch } from '../ui/Controls';
import { t } from '../i18n';

export const newForward = (): Forward => ({
  id: crypto.randomUUID(),
  type: 'L',
  bindHost: '127.0.0.1',
  bindPort: 8080,
  destHost: 'localhost',
  destPort: 80,
});

/** 포트 포워딩 표(§1.4). */
export function ForwardsTable({ rows, onChange }: { rows: Forward[]; onChange: (rows: Forward[]) => void }) {
  const patch = (id: string, p: Partial<Forward>) => onChange(rows.map((r) => (r.id === id ? { ...r, ...p } : r)));
  return (
    <>
      <div className="notice info">{t('fwd.help')}</div>
      <table className="fwd-table" style={{ width: '100%', borderCollapse: 'separate', borderSpacing: '0 6px' }}>
        <thead>
          <tr className="faint small" style={{ textAlign: 'left' }}>
            <th style={{ width: 70 }}>{t('fwd.type')}</th>
            <th>{t('fwd.bindHost')}</th>
            <th style={{ width: 90 }}>{t('fwd.port')}</th>
            <th>{t('fwd.destHost')}</th>
            <th style={{ width: 90 }}>{t('fwd.destPort')}</th>
            <th style={{ width: 40 }} />
          </tr>
        </thead>
        <tbody>
          {rows.map((r) => (
            <tr key={r.id}>
              <td>
                <select className="input" aria-label={t('fwd.type')} value={r.type} onChange={(e) => patch(r.id, { type: e.target.value as Forward['type'] })}>
                  <option value="L">L</option>
                  <option value="R">R</option>
                  <option value="D">D</option>
                </select>
              </td>
              <td><input className="input mono" aria-label={t('fwd.bindHost')} value={r.bindHost} onChange={(e) => patch(r.id, { bindHost: e.target.value })} /></td>
              <td><input className="input" aria-label={t('fwd.port')} inputMode="numeric" value={String(r.bindPort)} onChange={(e) => patch(r.id, { bindPort: Number(e.target.value.replace(/\D/g, '')) || 0 })} /></td>
              {r.type === 'D' ? (
                // 동적 포워딩은 대상이 없다(SOCKS 프록시를 쓰는 프로그램이 정한다). 적어 둔 대상 값은 지우지 않는다.
                <td colSpan={2} className="faint small">{t('fwd.dynamicDest')}</td>
              ) : (
                <>
                  <td><input className="input mono" aria-label={t('fwd.destHost')} value={r.destHost} onChange={(e) => patch(r.id, { destHost: e.target.value })} /></td>
                  <td><input className="input" aria-label={t('fwd.destPort')} inputMode="numeric" value={String(r.destPort)} onChange={(e) => patch(r.id, { destPort: Number(e.target.value.replace(/\D/g, '')) || 0 })} /></td>
                </>
              )}
              <td>
                <button type="button" className="icon-btn" aria-label={t('common.delete')} title={t('common.delete')} onClick={() => onChange(rows.filter((x) => x.id !== r.id))}>
                  <Trash2 size={14} />
                </button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
      <div>
        <button type="button" className="btn small-btn" onClick={() => onChange([...rows, newForward()])}>{t('fwd.add')}</button>
      </div>
    </>
  );
}

const PARITY: { value: SerialParity; label: string }[] = [
  { value: 'none', label: 'none' },
  { value: 'even', label: 'even' },
  { value: 'odd', label: 'odd' },
  { value: 'mark', label: 'mark' },
  { value: 'space', label: 'space' },
];
const FLOW: { value: SerialFlow; label: string }[] = [
  { value: 'none', label: 'none' }, // 그릴 때 serial.flowNone 으로 바꾼다
  { value: 'rtscts', label: 'RTS/CTS' },
  { value: 'xonxoff', label: 'XON/XOFF' },
];
const ENTER: { value: SerialEnter; label: string }[] = [
  { value: 'CR', label: 'CR' },
  { value: 'LF', label: 'LF' },
  { value: 'CRLF', label: 'CRLF' },
];

/** 시리얼 옵션(§1.1 3번). 장치 목록은 serial.list 로 받아 자동완성에 쓴다. */
export function SerialFields({
  value,
  ports,
  onChange,
  onRefresh,
}: {
  value: SerialOpts;
  ports: { path: string; label: string }[];
  onChange: (v: SerialOpts) => void;
  onRefresh: () => void;
}) {
  const set = <K extends keyof SerialOpts>(k: K, v: SerialOpts[K]) => onChange({ ...value, [k]: v });
  const listId = 'serial-ports';
  return (
    <>
      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 14 }}>
        <Field label={t('fwd.port')} htmlFor="serial-port">
          <div style={{ display: 'flex', gap: 8 }}>
            <input id="serial-port" className="input mono" list={listId} placeholder="COM3" value={value.path} onChange={(e) => set('path', e.target.value)} style={{ flex: 1 }} />
            <button type="button" className="btn link small" onClick={onRefresh}>{t('sftp.refresh')}</button>
          </div>
          <datalist id={listId}>
            {ports.map((p) => <option key={p.path} value={p.path}>{p.label}</option>)}
          </datalist>
        </Field>
        <Field label={t('serial.baud')} htmlFor="serial-baud">
          <select id="serial-baud" className="input" value={value.baudRate} onChange={(e) => set('baudRate', Number(e.target.value))}>
            {BAUD_RATES.map((b) => <option key={b} value={b}>{b}</option>)}
          </select>
        </Field>
      </div>
      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 1fr', gap: 14 }}>
        <Field label={t('serial.dataBits')}>
          <select className="input" value={value.dataBits} onChange={(e) => set('dataBits', Number(e.target.value) as SerialOpts['dataBits'])}>
            {[8, 7, 6, 5].map((n) => <option key={n} value={n}>{n}</option>)}
          </select>
        </Field>
        <Field label={t('serial.parity')}>
          <select className="input" value={value.parity} onChange={(e) => set('parity', e.target.value as SerialParity)}>
            {PARITY.map((o) => <option key={o.value} value={o.value}>{o.label}</option>)}
          </select>
        </Field>
        <Field label={t('serial.stopBits')}>
          <select className="input" value={value.stopBits} onChange={(e) => set('stopBits', Number(e.target.value) as SerialOpts['stopBits'])}>
            {[1, 1.5, 2].map((n) => <option key={n} value={n}>{n}</option>)}
          </select>
        </Field>
      </div>
      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 14 }}>
        <Field label={t('serial.flow')} help={t('serial.flowHelp')}>
          <select className="input" value={value.flowControl} onChange={(e) => set('flowControl', e.target.value as SerialFlow)}>
            {FLOW.map((o) => <option key={o.value} value={o.value}>{o.value === 'none' ? t('serial.flowNone') : o.label}</option>)}
          </select>
        </Field>
        <Field label={t('serial.enter')}>
          <Segmented value={value.enterSends} options={ENTER} onChange={(v) => set('enterSends', v)} label={t('serial.enter')} />
        </Field>
      </div>
      <div className="toggle-row">
        <div>
          <div className="t-title">{t('serial.echo')}</div>
          <div className="t-desc">{t('serial.echoDesc')}</div>
        </div>
        <Switch checked={value.localEcho} onChange={(v) => set('localEcho', v)} label={t('serial.echo')} />
      </div>
    </>
  );
}

/** 점프 호스트 직접 입력 카드(§1.3 2번). */
export function JumpDirectCard({
  address,
  onAddress,
  port,
  onPort,
  username,
  onUsername,
  authType,
  onAuthType,
  password,
  onPassword,
  keyText,
  onPickKey,
  passphrase,
  onPassphrase,
  hasSaved,
  keyStatus,
}: {
  address: string;
  onAddress: (v: string) => void;
  port: string;
  onPort: (v: string) => void;
  username: string;
  onUsername: (v: string) => void;
  authType: JumpDirect['authType'];
  onAuthType: (v: JumpDirect['authType']) => void;
  password: string;
  onPassword: (v: string) => void;
  keyText: string;
  onPickKey: () => void;
  passphrase: string;
  onPassphrase: (v: string) => void;
  hasSaved: boolean;
  keyStatus: string;
}) {
  void keyText;
  return (
    <div className="card" style={{ display: 'flex', flexDirection: 'column', gap: 12, padding: 12, border: '1px solid var(--border)', borderRadius: 8 }}>
      <Field label={t('jump.address')} help={t('jump.addressHelp')}>
        <input className="input mono" placeholder="ec2-user@203.0.113.10:22" value={address} onChange={(e) => onAddress(e.target.value)} />
      </Field>
      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12 }}>
        <Field label={t('fwd.port')}>
          <input className="input" inputMode="numeric" placeholder="22" value={port} onChange={(e) => onPort(e.target.value.replace(/\D/g, ''))} />
        </Field>
        <Field label={t('host.user')}>
          <input className="input" value={username} onChange={(e) => onUsername(e.target.value)} />
        </Field>
      </div>
      <Field label={t('host.authMethod')}>
        <Segmented
          value={authType}
          options={[{ value: 'password', label: t('cred.password') }, { value: 'key', label: t('host.keyFile') }]}
          onChange={onAuthType}
          label={t('jump.authAria')}
        />
      </Field>
      {authType === 'password' ? (
        <Field
          label={t('cred.password')}
          help={hasSaved ? t('host.savedChange') : t('jump.saveHint')}
        >
          <input className="input" type="password" placeholder={hasSaved ? t('host.savedMask') : t('cred.password')} value={password} onChange={(e) => onPassword(e.target.value)} />
        </Field>
      ) : (
        <>
          <Field label={t('host.keyFile')}>
            <div style={{ display: 'flex', gap: 10, alignItems: 'center' }}>
              <button type="button" className="btn small-btn" onClick={onPickKey}>{t('host.pickKey')}</button>
              <span className="faint small">{keyStatus || (hasSaved ? t('host.saved') : t('host.noKey'))}</span>
            </div>
          </Field>
          <Field label={t('host.passphrase')}>
            <input className="input" type="password" value={passphrase} onChange={(e) => onPassphrase(e.target.value)} />
          </Field>
        </>
      )}
    </div>
  );
}

/** 프록시 경유 카드(§1.3 2-1번): 주소·포트·사용자 이름·비밀번호. 사용자 이름을 비우면 인증 없이 연결한다. */
export function ProxyCard({
  type,
  host,
  onHost,
  port,
  onPort,
  username,
  onUsername,
  password,
  onPassword,
  hasSaved,
  jumpInUse,
}: {
  type: ProxyType;
  host: string;
  onHost: (v: string) => void;
  port: string;
  onPort: (v: string) => void;
  username: string;
  onUsername: (v: string) => void;
  password: string;
  onPassword: (v: string) => void;
  hasSaved: boolean;
  jumpInUse: boolean;
}) {
  return (
    <div className="card" style={{ display: 'flex', flexDirection: 'column', gap: 12, padding: 12, border: '1px solid var(--border)', borderRadius: 8 }}>
      {jumpInUse && <div className="notice info">{t('proxy.jumpNote')}</div>}
      <div style={{ display: 'grid', gridTemplateColumns: '2fr 1fr', gap: 12 }}>
        <Field label={t('proxy.address')}>
          <input className="input mono" placeholder="proxy.example.com" value={host} onChange={(e) => onHost(e.target.value)} />
        </Field>
        <Field label={t('fwd.port')}>
          <input className="input" inputMode="numeric" placeholder={String(PROXY_DEFAULT_PORT[type])} value={port} onChange={(e) => onPort(e.target.value.replace(/\D/g, ''))} />
        </Field>
      </div>
      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12 }}>
        <Field label={t('proxy.user')} help={t('proxy.userHelp')}>
          <input className="input" value={username} onChange={(e) => onUsername(e.target.value)} />
        </Field>
        <Field label={t('cred.password')} help={hasSaved ? t('host.savedChange') : t('proxy.saveHint')}>
          <input className="input" type="password" disabled={!username.trim()} placeholder={hasSaved ? t('host.savedMask') : t('cred.password')} value={password} onChange={(e) => onPassword(e.target.value)} />
        </Field>
      </div>
    </div>
  );
}
