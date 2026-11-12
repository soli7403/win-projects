// 서버 편집의 큰 부분 조각: 포트 포워딩 표, 시리얼 옵션, 점프 호스트 카드. 근거: spec/ui/host-editor.md §1.1(3), §1.3(1), §1.4.
import { Trash2 } from 'lucide-react';
import type { Forward } from '../ipc/types';
import { BAUD_RATES, type JumpDirect, type SerialEnter, type SerialFlow, type SerialOpts, type SerialParity } from '../lib/hostForm';
import { Field, Segmented, Switch } from '../ui/Controls';

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
      <div className="notice info">L: 내 PC의 포트 → 서버에서 접근 가능한 주소 (예: 로컬 3307 → DB 3306). R: 서버의 포트 → 내 PC 쪽 주소.</div>
      <table className="fwd-table" style={{ width: '100%', borderCollapse: 'separate', borderSpacing: '0 6px' }}>
        <thead>
          <tr className="faint small" style={{ textAlign: 'left' }}>
            <th style={{ width: 70 }}>종류</th>
            <th>바인드 주소</th>
            <th style={{ width: 90 }}>포트</th>
            <th>대상 호스트</th>
            <th style={{ width: 90 }}>대상 포트</th>
            <th style={{ width: 40 }} />
          </tr>
        </thead>
        <tbody>
          {rows.map((r) => (
            <tr key={r.id}>
              <td>
                <select className="input" aria-label="종류" value={r.type} onChange={(e) => patch(r.id, { type: e.target.value as 'L' | 'R' })}>
                  <option value="L">L</option>
                  <option value="R">R</option>
                </select>
              </td>
              <td><input className="input mono" aria-label="바인드 주소" value={r.bindHost} onChange={(e) => patch(r.id, { bindHost: e.target.value })} /></td>
              <td><input className="input" aria-label="포트" inputMode="numeric" value={String(r.bindPort)} onChange={(e) => patch(r.id, { bindPort: Number(e.target.value.replace(/\D/g, '')) || 0 })} /></td>
              <td><input className="input mono" aria-label="대상 호스트" value={r.destHost} onChange={(e) => patch(r.id, { destHost: e.target.value })} /></td>
              <td><input className="input" aria-label="대상 포트" inputMode="numeric" value={String(r.destPort)} onChange={(e) => patch(r.id, { destPort: Number(e.target.value.replace(/\D/g, '')) || 0 })} /></td>
              <td>
                <button type="button" className="icon-btn" aria-label="삭제" title="삭제" onClick={() => onChange(rows.filter((x) => x.id !== r.id))}>
                  <Trash2 size={14} />
                </button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
      <div>
        <button type="button" className="btn small-btn" onClick={() => onChange([...rows, newForward()])}>+ 포워딩 추가</button>
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
  { value: 'none', label: '없음' },
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
        <Field label="포트" htmlFor="serial-port">
          <div style={{ display: 'flex', gap: 8 }}>
            <input id="serial-port" className="input mono" list={listId} placeholder="COM3" value={value.path} onChange={(e) => set('path', e.target.value)} style={{ flex: 1 }} />
            <button type="button" className="btn link small" onClick={onRefresh}>새로고침</button>
          </div>
          <datalist id={listId}>
            {ports.map((p) => <option key={p.path} value={p.path}>{p.label}</option>)}
          </datalist>
        </Field>
        <Field label="속도 (baud)" htmlFor="serial-baud">
          <select id="serial-baud" className="input" value={value.baudRate} onChange={(e) => set('baudRate', Number(e.target.value))}>
            {BAUD_RATES.map((b) => <option key={b} value={b}>{b}</option>)}
          </select>
        </Field>
      </div>
      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 1fr', gap: 14 }}>
        <Field label="데이터 비트">
          <select className="input" value={value.dataBits} onChange={(e) => set('dataBits', Number(e.target.value) as SerialOpts['dataBits'])}>
            {[8, 7, 6, 5].map((n) => <option key={n} value={n}>{n}</option>)}
          </select>
        </Field>
        <Field label="패리티">
          <select className="input" value={value.parity} onChange={(e) => set('parity', e.target.value as SerialParity)}>
            {PARITY.map((o) => <option key={o.value} value={o.value}>{o.label}</option>)}
          </select>
        </Field>
        <Field label="정지 비트">
          <select className="input" value={value.stopBits} onChange={(e) => set('stopBits', Number(e.target.value) as SerialOpts['stopBits'])}>
            {[1, 1.5, 2].map((n) => <option key={n} value={n}>{n}</option>)}
          </select>
        </Field>
      </div>
      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 14 }}>
        <Field label="흐름 제어">
          <select className="input" value={value.flowControl} onChange={(e) => set('flowControl', e.target.value as SerialFlow)}>
            {FLOW.map((o) => <option key={o.value} value={o.value}>{o.label}</option>)}
          </select>
        </Field>
        <Field label="Enter 전송값">
          <Segmented value={value.enterSends} options={ENTER} onChange={(v) => set('enterSends', v)} label="Enter 전송값" />
        </Field>
      </div>
      <div className="toggle-row">
        <div>
          <div className="t-title">로컬 에코</div>
          <div className="t-desc">장비가 입력한 글자를 돌려주지 않을 때 켜세요</div>
        </div>
        <Switch checked={value.localEcho} onChange={(v) => set('localEcho', v)} label="로컬 에코" />
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
      <Field label="베스천 주소" help="user@host:port 형식으로 넣으면 사용자·호스트·포트로 나뉩니다">
        <input className="input mono" placeholder="ec2-user@203.0.113.10:22" value={address} onChange={(e) => onAddress(e.target.value)} />
      </Field>
      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12 }}>
        <Field label="포트">
          <input className="input" inputMode="numeric" placeholder="22" value={port} onChange={(e) => onPort(e.target.value.replace(/\D/g, ''))} />
        </Field>
        <Field label="사용자">
          <input className="input" value={username} onChange={(e) => onUsername(e.target.value)} />
        </Field>
      </div>
      <Field label="인증 방식">
        <Segmented
          value={authType}
          options={[{ value: 'password', label: '비밀번호' }, { value: 'key', label: '개인 키' }]}
          onChange={onAuthType}
          label="베스천 인증 방식"
        />
      </Field>
      {authType === 'password' ? (
        <Field
          label="비밀번호"
          help={hasSaved ? '저장됨 — 바꾸려면 새로 입력' : '볼트에 암호화해 저장 ("{별칭} 베스천")'}
        >
          <input className="input" type="password" placeholder={hasSaved ? '••••••••  (저장됨)' : '비밀번호'} value={password} onChange={(e) => onPassword(e.target.value)} />
        </Field>
      ) : (
        <>
          <Field label="개인 키">
            <div style={{ display: 'flex', gap: 10, alignItems: 'center' }}>
              <button type="button" className="btn small-btn" onClick={onPickKey}>키 파일 선택</button>
              <span className="faint small">{keyStatus || (hasSaved ? '저장됨' : '선택된 키 없음')}</span>
            </div>
          </Field>
          <Field label="키 암호 (있는 경우)">
            <input className="input" type="password" value={passphrase} onChange={(e) => onPassphrase(e.target.value)} />
          </Field>
        </>
      )}
    </div>
  );
}
