// 서버 가져오기 대화상자. 근거: spec/ui/dialogs.md §5, spec/import.md §1.5·§4, spec/acceptance I1·I6·I9.
import { useEffect, useMemo, useState } from 'react';
import { Download, KeyRound, Cloud, FolderOpen } from 'lucide-react';
import { Modal } from '../ui/Modal';
import { Field } from '../ui/Controls';
import { getBackend, errorMessage } from '../ipc/backend';
import { useApp } from '../state/app';
import { useUi } from '../state/ui';
import type { ImportCandidate, ImportApplyResult, ImportHost } from '../ipc/c1types';
import { t } from '../i18n';
import { Rich } from '../i18n/Rich';

type Cand = ImportCandidate & { key: string };

const SOURCE_LABEL: Record<string, string> = { putty: 'PuTTY', securecrt: 'SecureCRT', 'ssh-config': 'ssh config', aws: 'AWS' };

/** 후보의 서버 정보. host 가 빠져 있어도 예외 없이 빈 객체로 본다(spec/ipc.md "가져오기 후보 객체"). */
export function hostOf(c: Partial<ImportCandidate> | null | undefined): ImportHost {
  return (c && c.host && typeof c.host === 'object' ? c.host : {}) as ImportHost;
}

/** 같은 호스트·포트·사용자의 서버가 이미 있는지(초기 선택 제외 기준, dialogs.md §5). */
export function sameHostExists(c: ImportCandidate, hosts: { host: string; port: number; username: string }[]): boolean {
  const h0 = hostOf(c);
  return hosts.some((h) => h.host === h0.host && h.port === h0.port && (h.username ?? '') === (h0.username ?? ''));
}

function kindOf(c: ImportCandidate): string {
  return String(hostOf(c).protocol ?? 'ssh').toUpperCase();
}

/** 하위 폴더 경로 표시(`폴더/폴더/`). 없으면 빈 문자열. */
export function folderPrefix(c: Partial<ImportCandidate>): string {
  const path = Array.isArray(c.folderPath) ? c.folderPath.filter((n) => typeof n === 'string' && n.trim() !== '') : [];
  return path.length > 0 ? `${path.join('/')}/` : '';
}

function addrOf(c: ImportCandidate): string {
  const h = hostOf(c);
  if (h.protocol === 'serial') return `${h.serial?.path ?? h.host ?? ''} @ ${h.serial?.baudRate ?? 9600}`;
  return `${h.username ? `${h.username}@` : ''}${h.host ?? ''}:${h.port ?? 22}`;
}

export function ImportDialog() {
  const open = useUi((s) => s.importOpen);
  const close = useUi((s) => s.closeImportDialog);
  if (!open) return null;
  return <ImportBody onClose={close} />;
}

function ImportBody({ onClose }: { onClose: () => void }) {
  const vaultLocked = useApp((s) => !s.vault?.unlocked);
  const pushToast = useApp((s) => s.pushToast);
  const [cands, setCands] = useState<Cand[]>([]);
  const [sel, setSel] = useState<Set<string>>(new Set());
  const [folder, setFolder] = useState(() => t('import.defaultFolder'));
  const [scanning, setScanning] = useState(true);
  const [awsBusy, setAwsBusy] = useState(false);
  const [crtBusy, setCrtBusy] = useState(false);
  const [applying, setApplying] = useState(false);

  // 후보 자동 검색. 초기 선택: 이미 있는 서버를 뺀 나머지 전부.
  useEffect(() => {
    let alive = true;
    getBackend()
      .invoke<ImportCandidate[]>('import.scan')
      .then((list) => {
        if (!alive) return;
        const withKey = (Array.isArray(list) ? list : []).map((c, i) => ({ ...c, key: `${c.source}-${i}` }));
        setCands(withKey);
        // PuTTY 후보 없이 SecureCRT 후보만 있으면 기본 폴더 이름을 SecureCRT 로 (dialogs.md §5).
        if (!withKey.some((c) => c.source === 'putty') && withKey.some((c) => c.source === 'securecrt')) setFolder('SecureCRT');
        setSel(new Set(withKey.filter((c) => !sameHostExists(c, useApp.getState().store?.hosts ?? [])).map((c) => c.key)));
      })
      .catch((e) => pushToast({ kind: 'error', text: errorMessage(e) }))
      .finally(() => alive && setScanning(false));
    return () => {
      alive = false;
    };
    // 검색은 열릴 때 한 번만 한다.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const allChecked = cands.length > 0 && cands.every((c) => sel.has(c.key));
  const toggle = (k: string) =>
    setSel((prev) => {
      const next = new Set(prev);
      if (next.has(k)) next.delete(k);
      else next.add(k);
      return next;
    });
  const toggleAll = () => setSel(allChecked ? new Set() : new Set(cands.map((c) => c.key)));

  // 한 출처의 후보를 새 결과로 대체하고(다른 출처는 유지) 새 후보를 중복 제외하고 선택, 폴더 이름을 바꾼다.
  const replaceSource = (source: string, found: ImportCandidate[], folderName: string) => {
    const base = cands.filter((c) => c.source !== source);
    const added: Cand[] = found.map((c, i) => ({ ...c, key: `${source}-${Date.now()}-${i}` }));
    const merged = [...base, ...added];
    setCands(merged);
    setSel((prev) => {
      const next = new Set([...prev].filter((k) => merged.some((c) => c.key === k)));
      added.forEach((c) => {
        if (!sameHostExists(c, useApp.getState().store?.hosts ?? [])) next.add(c.key);
      });
      return next;
    });
    setFolder(folderName);
  };

  // AWS 조회: 결과는 AWS 후보를 대체, 폴더 이름은 "AWS".
  const scanAws = async () => {
    setAwsBusy(true);
    try {
      const found = await getBackend().invoke<ImportCandidate[]>('import.scanAws', []);
      if (!Array.isArray(found) || found.length === 0) {
        pushToast({ kind: 'info', text: t('import.awsNone') });
        return;
      }
      replaceSource('aws', found, 'AWS');
    } catch (e) {
      pushToast({ kind: 'error', text: t('import.awsFailed', { error: errorMessage(e) }) });
    } finally {
      setAwsBusy(false);
    }
  };

  // SecureCRT 폴더 선택: 취소(null)면 아무것도 안 함. 결과는 SecureCRT 후보를 대체, 폴더 이름은 "SecureCRT".
  const scanSecureCrt = async () => {
    setCrtBusy(true);
    try {
      const found = await getBackend().invoke<ImportCandidate[] | null>('import.scanSecureCrt', []);
      if (found === null || found === undefined) return;
      if (!Array.isArray(found) || found.length === 0) {
        pushToast({ kind: 'info', text: t('import.crtNone') });
        return;
      }
      replaceSource('securecrt', found, 'SecureCRT');
    } catch (e) {
      pushToast({ kind: 'error', text: t('import.crtFailed', { error: errorMessage(e) }) });
    } finally {
      setCrtBusy(false);
    }
  };

  const chosen = useMemo(() => cands.filter((c) => sel.has(c.key)), [cands, sel]);

  const apply = async () => {
    if (chosen.length === 0 || applying) return;
    setApplying(true);
    // 후보 객체는 변경 없이 그대로 보낸다(내부 key 만 뺀다).
    const payload = chosen.map(({ key: _key, ...rest }) => rest);
    try {
      const r = await getBackend().invoke<ImportApplyResult>('import.apply', [payload, folder.trim()]);
      await useApp.getState().refreshStore();
      await useApp.getState().refreshVault();
      pushToast({
        kind: 'ok',
        text: t('import.done', { count: r.hosts }) + (r.keys > 0 ? t('import.doneKeys', { count: r.keys }) : ''),
      });
      if (r.skipped && r.skipped.length > 0) pushToast({ kind: 'error', text: r.skipped.join('\n') });
      onClose();
    } catch (e) {
      pushToast({ kind: 'error', text: errorMessage(e) });
    } finally {
      setApplying(false);
    }
  };

  return (
    <Modal
      title={t('import.title')}
      icon={Download}
      size="lg"
      onClose={onClose}
      foot={
        <>
          <span className="faint" style={{ marginRight: 'auto', alignSelf: 'center' }}>{t('import.selectedCount', { sel: sel.size, total: cands.length })}</span>
          <button className="btn" type="button" onClick={onClose}>{t('common.cancel')}</button>
          <button className="btn primary" type="button" disabled={chosen.length === 0 || applying} onClick={() => void apply()}>
            {applying ? t('import.applying') : t('import.apply', { count: chosen.length })}
          </button>
        </>
      }
    >
      <div className="set-notice" style={{ lineHeight: 1.6 }}>
        <Rich text={t('import.intro')} />
        {vaultLocked && t('import.lockedNote')}
      </div>

      <div style={{ display: 'flex', gap: 12, alignItems: 'center' }}>
        <button className="btn" type="button" disabled={awsBusy} onClick={() => void scanAws()} style={{ flex: 'none' }}>
          <Cloud size={14} /> {awsBusy ? t('import.awsBusy') : t('import.aws')}
        </button>
        <span className="set-note">{t('import.awsNote')}</span>
      </div>

      <div style={{ display: 'flex', gap: 12, alignItems: 'center' }}>
        <button className="btn" type="button" disabled={crtBusy} onClick={() => void scanSecureCrt()} style={{ flex: 'none' }}>
          <FolderOpen size={14} /> {crtBusy ? t('import.crtBusy') : t('import.crt')}
        </button>
        <span className="set-note">{t('import.crtNote')}</span>
      </div>

      {scanning && <div className="faint">{t('import.scanning')}</div>}
      {!scanning && cands.length === 0 && (
        <div>
          <div>{t('import.none')}</div>
          <div className="set-note">{t('import.noneNote')}</div>
        </div>
      )}

      {cands.length > 0 && (
        <>
          <Field label={t('import.folder')} help={t('import.folderHelp')}>
            <input className="input" aria-label={t('import.folder')} value={folder} onChange={(e) => setFolder(e.target.value)} />
          </Field>
          <div className="import-wrap">
            <table className="import-table">
              <thead>
                <tr>
                  <th style={{ width: 36 }}>
                    <input type="checkbox" aria-label={t('pane.selectAll')} checked={allChecked} onChange={toggleAll} />
                  </th>
                  <th>{t('sftp.colName')}</th>
                  <th>{t('import.address')}</th>
                  <th>{t('import.kind')}</th>
                  <th>{t('import.source')}</th>
                </tr>
              </thead>
              <tbody>
                {cands.map((c) => (
                  <tr key={c.key}>
                    <td>
                      <input type="checkbox" aria-label={t('import.selectOne', { name: c.name })} checked={sel.has(c.key)} onChange={() => toggle(c.key)} />
                    </td>
                    <td style={{ fontWeight: 700 }}>
                      {folderPrefix(c) && <span className="faint" style={{ fontWeight: 400 }}>{folderPrefix(c)}</span>}
                      {c.name ?? hostOf(c).alias ?? ''}
                    </td>
                    <td>
                      <span className="mono" style={{ fontSize: 12 }}>{addrOf(c)}</span>
                      {c.keyFile && <KeyRound size={12} style={{ marginLeft: 6, verticalAlign: '-2px' }} aria-label={t('import.hasKey')} />}
                    </td>
                    <td>{kindOf(c)}</td>
                    <td title={c.note ?? undefined}>
                      {SOURCE_LABEL[c.source] ?? String(c.source ?? '')}
                      {c.source === 'aws' && c.jumpVia ? t('import.viaJump') : ''}
                      {typeof c.note === 'string' && c.note.includes(t('import.noKeyFile')) && <span className="danger-note">{t('import.noKey')}</span>}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </>
      )}
    </Modal>
  );
}
