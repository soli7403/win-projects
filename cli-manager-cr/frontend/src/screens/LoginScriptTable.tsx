// 서버 편집 "자동화" 탭: 로그인 스크립트 단계 표. 근거: spec/ui/host-editor.md §1.4a (D139, 보관함 값 D148).
import { ArrowDown, ArrowUp, Trash2 } from 'lucide-react';
import type { CredMeta, LoginStep } from '../ipc/types';
import { LOGIN_PASSWORD_TOKEN, LOGIN_STEP_MAX, LOGIN_STEP_TIMEOUT_SEC, moveStep, newLoginStep } from '../lib/loginScript';
import { t } from '../i18n';

interface Props {
  rows: LoginStep[];
  onChange: (rows: LoginStep[]) => void;
  isSsh: boolean;
  /** 보관함의 항목 목록(서버 편집이 읽어 둔 cred.list). 비밀번호 종류만 고를 수 있다. */
  secrets?: CredMeta[];
  /** 보관함이 열려 있는지. 잠겨 있으면 목록이 비어 고른 항목 이름을 알 수 없다. */
  vaultOpen?: boolean;
}

export function LoginScriptTable({ rows, onChange, isSsh, secrets = [], vaultOpen = true }: Props) {
  const patch = (id: string, p: Partial<LoginStep>) => onChange(rows.map((r) => (r.id === id ? { ...r, ...p } : r)));
  const pwItems = secrets.filter((c) => c.kind === 'password');
  /** 고른 항목의 이름. 목록에 없으면 볼트 상태에 따라 "(지워진 항목)" 또는 "(보관함 잠김)". */
  const secretName = (id: string) => {
    const c = pwItems.find((x) => x.id === id);
    if (c) return c.name;
    return vaultOpen ? t('login.secretGone') : t('login.secretLockedName');
  };
  // 표는 열 너비를 고정한다(D157 작업 중 발견): 자동 배치면 보관함 값의 긴 안내 글이 "보낼 글자" 열을 넓혀
  // "기다릴 글자" 열이 글자 하나 폭으로 줄어들었다.
  return (
    <>
      <div className="notice info">{t('login.help', { sec: LOGIN_STEP_TIMEOUT_SEC })}</div>
      <div className="notice warn">
        {t('login.plainWarn')}
        {isSsh && ` ${t('login.passwordHint', { token: LOGIN_PASSWORD_TOKEN })}`}
        {` ${t('login.secretHint')}`}
      </div>
      {rows.length > 0 && (
        <table className="fwd-table" style={{ width: '100%', tableLayout: 'fixed', borderCollapse: 'separate', borderSpacing: '0 6px' }}>
          <thead>
            <tr className="faint small" style={{ textAlign: 'left' }}>
              <th style={{ width: 26 }}>#</th>
              <th style={{ width: '22%' }}>{t('login.expect')}</th>
              <th>{t('login.send')}</th>
              <th style={{ width: 128 }}>{t('login.secret')}</th>
              <th style={{ width: 88 }} />
            </tr>
          </thead>
          <tbody>
            {rows.map((r, i) => (
              <tr key={r.id}>
                <td className="faint small">{i + 1}</td>
                <td>
                  <input className="input mono" aria-label={t('login.expectN', { n: i + 1 })} placeholder={t('login.expectPlaceholder')} value={r.expect} onChange={(e) => patch(r.id, { expect: e.target.value })} />
                </td>
                <td>
                  {r.secretId ? (
                    <div className="input faint small" style={{ display: 'flex', alignItems: 'center', opacity: 0.8 }} title={t('login.secretSends', { name: secretName(r.secretId) })}>
                      🔑 {t('login.secretSends', { name: secretName(r.secretId) })}
                    </div>
                  ) : (
                    <input className="input mono" aria-label={t('login.sendN', { n: i + 1 })} placeholder={t('login.sendPlaceholder')} value={r.send} onChange={(e) => patch(r.id, { send: e.target.value })} />
                  )}
                </td>
                <td>
                  <select
                    className="input"
                    aria-label={t('login.secretN', { n: i + 1 })}
                    value={r.secretId ?? ''}
                    onChange={(e) => patch(r.id, { secretId: e.target.value || undefined })}
                  >
                    <option value="">{t('login.secretNone')}</option>
                    {pwItems.map((c) => (
                      <option key={c.id} value={c.id}>{c.username ? `${c.name} (${c.username})` : c.name}</option>
                    ))}
                    {r.secretId && !pwItems.some((c) => c.id === r.secretId) && <option value={r.secretId}>{secretName(r.secretId)}</option>}
                  </select>
                </td>
                <td style={{ whiteSpace: 'nowrap' }}>
                  <button type="button" className="icon-btn" aria-label={t('login.up')} title={t('login.up')} disabled={i === 0} onClick={() => onChange(moveStep(rows, i, -1))}>
                    <ArrowUp size={14} />
                  </button>
                  <button type="button" className="icon-btn" aria-label={t('login.down')} title={t('login.down')} disabled={i === rows.length - 1} onClick={() => onChange(moveStep(rows, i, 1))}>
                    <ArrowDown size={14} />
                  </button>
                  <button type="button" className="icon-btn" aria-label={t('common.delete')} title={t('common.delete')} onClick={() => onChange(rows.filter((x) => x.id !== r.id))}>
                    <Trash2 size={14} />
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
      <div>
        <button type="button" className="btn small-btn" disabled={rows.length >= LOGIN_STEP_MAX} onClick={() => onChange([...rows, newLoginStep()])}>
          {t('login.add')}
        </button>
      </div>
    </>
  );
}
