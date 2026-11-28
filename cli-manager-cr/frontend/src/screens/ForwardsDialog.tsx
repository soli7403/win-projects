// 포트 포워딩 정보 창: 지금 연결에 설정된 포워딩과 그 결과를 표로 보인다.
// 근거: spec/ui/panes.md §2.6, spec/ui/dialogs.md §7 (D137, 전송량 D151).
import { useCallback, useEffect, useState } from 'react';
import { ArrowLeftRight } from 'lucide-react';
import { Modal } from '../ui/Modal';
import type { ForwardInfo } from '../ipc/types';
import { getBackend } from '../ipc/backend';
import { useApp } from '../state/app';
import { failedCount, forwardsText, trafficTotal } from '../lib/forwards';
import { formatBytes } from '../lib/format';
import { t, type MsgId } from '../i18n';

const KIND: Record<ForwardInfo['type'], MsgId> = { L: 'fwdinfo.kindL', R: 'fwdinfo.kindR', D: 'fwdinfo.kindD' };

export function ForwardsDialog({ sessionId, title, target, forwards: initial, onClose }: {
  /** 연결 수를 물을 세션. 없으면 묻지 않고 받은 목록만 보인다. */
  sessionId?: string;
  title: string;
  target: string;
  forwards: ForwardInfo[];
  onClose: () => void;
}) {
  // 지금 이어져 있는 연결 수는 세션 정보에 실려 오지 않는다(상태가 바뀔 때만 오므로).
  // 창을 열 때 한 번 읽는다. 주기적으로 다시 묻지 않는다 — 다시 보려면 [새로고침].
  const [live, setLive] = useState<ForwardInfo[] | null>(null);
  const load = useCallback(() => {
    if (!sessionId) return;
    void getBackend().invoke<ForwardInfo[]>('session.forwards', [sessionId])
      .then((list) => {
        if (Array.isArray(list) && list.length > 0) setLive(list);
      })
      .catch(() => undefined);
  }, [sessionId]);
  useEffect(() => load(), [load]);
  const forwards = live ?? initial;
  const failed = failedCount(forwards);
  const conns = forwards.reduce((sum, f) => sum + (f.conns ?? 0), 0);
  const total = trafficTotal(forwards);
  const copy = () => {
    const text = forwardsText(forwards, t('fwdinfo.socks'), (error) => t('fwdinfo.failed', { error }));
    void getBackend().invoke('clip.writeText', [text])
      .then(() => useApp.getState().pushToast({ kind: 'ok', text: t('fwdinfo.copied') }))
      .catch(() => undefined);
  };
  return (
    <Modal
      title={t('fwdinfo.title', { name: title })}
      icon={ArrowLeftRight}
      size="md"
      onClose={onClose}
      foot={
        <>
          {sessionId && <button className="btn" style={{ marginRight: 'auto' }} onClick={load}>{t('sftp.refresh')}</button>}
          <button className="btn" onClick={copy}>{t('common.copy')}</button>
          {/* 열리면 여기에 포커스를 둔다: 창이 떠 있는 동안 치는 키가 뒤의 터미널로 가지 않게. */}
          <button className="btn primary" autoFocus onClick={onClose}>{t('common.close')}</button>
        </>
      }
    >
      <div className="faint small" style={{ marginBottom: 10 }}>
        {target} · {failed > 0 ? t('fwdinfo.summaryFailed', { count: forwards.length, failed }) : t('fwdinfo.summary', { count: forwards.length })}
        {conns > 0 && ` · ${t('fwdinfo.conns', { count: conns })}`}
        {total.up + total.down > 0 && ` · ${t('fwdinfo.traffic', { up: formatBytes(total.up), down: formatBytes(total.down) })}`}
      </div>
      <table className="fwd-info">
        <thead>
          <tr>
            <th>{t('fwdinfo.colKind')}</th>
            <th>{t('fwdinfo.colBind')}</th>
            <th>{t('fwdinfo.colDest')}</th>
            <th>{t('fwdinfo.colState')}</th>
            <th>{t('fwdinfo.colTraffic')}</th>
          </tr>
        </thead>
        <tbody>
          {forwards.map((f, i) => (
            <tr key={`${f.type}-${f.bind}-${i}`} className={f.error ? 'failed' : undefined}>
              <td title={t(KIND[f.type])}><b>{f.type}</b> <span className="faint">{t(KIND[f.type])}</span></td>
              <td className="mono selectable">{f.bind}</td>
              <td className="mono selectable">{f.type === 'D' ? <span className="faint">{t('fwdinfo.socks')}</span> : f.dest}</td>
              <td>{f.error ? <span className="fwd-bad" title={f.error}>{t('fwdinfo.failed', { error: f.error })}</span> : <span className="fwd-ok">{t('fwdinfo.listening')}{(f.conns ?? 0) > 0 && <b>{` · ${t('fwdinfo.conns', { count: f.conns ?? 0 })}`}</b>}</span>}</td>
              <td className="mono small faint" style={{ whiteSpace: 'nowrap' }}>
                {!f.error && t('fwdinfo.traffic', { up: formatBytes(f.up ?? 0), down: formatBytes(f.down ?? 0) })}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
      <div className="help" style={{ marginTop: 10 }}>{t('fwdinfo.help')}</div>
    </Modal>
  );
}
