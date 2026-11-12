// 서버 가져오기 대화상자. 근거: spec/ui/dialogs.md §5, spec/import.md §4, spec/acceptance I1·I6·I9.
import { useEffect, useMemo, useState } from 'react';
import { Download, KeyRound, Cloud } from 'lucide-react';
import { Modal } from '../ui/Modal';
import { Field } from '../ui/Controls';
import { getBackend, errorMessage } from '../ipc/backend';
import { useApp } from '../state/app';
import { useUi } from '../state/ui';
import type { ImportCandidate, ImportApplyResult, ImportHost } from '../ipc/c1types';

type Cand = ImportCandidate & { key: string };

const SOURCE_LABEL: Record<string, string> = { putty: 'PuTTY', 'ssh-config': 'ssh config', aws: 'AWS' };

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
  const [folder, setFolder] = useState('PuTTY 가져오기');
  const [scanning, setScanning] = useState(true);
  const [awsBusy, setAwsBusy] = useState(false);
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

  // AWS 조회: 기존 AWS 후보를 대체하고(다른 출처는 유지) 새 후보를 중복 제외하고 선택, 폴더 이름은 "AWS".
  const scanAws = async () => {
    setAwsBusy(true);
    try {
      const found = await getBackend().invoke<ImportCandidate[]>('import.scanAws', []);
      if (!Array.isArray(found) || found.length === 0) {
        pushToast({ kind: 'info', text: 'AWS 에서 가져올 인스턴스를 찾지 못했습니다' });
        return;
      }
      const base = cands.filter((c) => c.source !== 'aws');
      const awsList: Cand[] = found.map((c, i) => ({ ...c, key: `aws-${Date.now()}-${i}` }));
      const merged = [...base, ...awsList];
      setCands(merged);
      setSel((prev) => {
        const next = new Set([...prev].filter((k) => merged.some((c) => c.key === k)));
        awsList.forEach((c) => {
          if (!sameHostExists(c, useApp.getState().store?.hosts ?? [])) next.add(c.key);
        });
        return next;
      });
      setFolder('AWS');
    } catch (e) {
      pushToast({ kind: 'error', text: `AWS 조회 실패: ${errorMessage(e)} — 터미널에서 aws configure 로 자격 증명을 확인하세요` });
    } finally {
      setAwsBusy(false);
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
        text: `${r.hosts}개 서버를 가져왔습니다${r.keys > 0 ? ` (키 ${r.keys}개 볼트에 저장)` : ''}`,
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
      title="서버 가져오기"
      icon={Download}
      size="lg"
      onClose={onClose}
      foot={
        <>
          <span className="faint" style={{ marginRight: 'auto', alignSelf: 'center' }}>{sel.size}/{cands.length}개 선택</span>
          <button className="btn" type="button" onClick={onClose}>취소</button>
          <button className="btn primary" type="button" disabled={chosen.length === 0 || applying} onClick={() => void apply()}>
            {applying ? '가져오는 중...' : `${chosen.length}개 가져오기`}
          </button>
        </>
      }
    >
      <div className="set-notice" style={{ lineHeight: 1.6 }}>
        PuTTY 저장 세션(레지스트리)과 <code className="kbdish">~/.ssh/config</code>를 읽어 옵니다. 호스트·포트·사용자·키 파일·포트 포워딩·시리얼 설정·인코딩이 옮겨집니다. PuTTY는 비밀번호를 저장하지 않으므로, 비밀번호는 처음 접속할 때 한 번 입력하면 저장됩니다.
        {vaultLocked && ' (볼트가 잠겨 있으면 키 파일은 볼트에 복사되지 않습니다)'}
      </div>

      <div style={{ display: 'flex', gap: 12, alignItems: 'center' }}>
        <button className="btn" type="button" disabled={awsBusy} onClick={() => void scanAws()} style={{ flex: 'none' }}>
          <Cloud size={14} /> {awsBusy ? 'AWS 전 리전 조회 중...' : 'AWS EC2 불러오기'}
        </button>
        <span className="set-note">aws CLI 자격 증명으로 인스턴스를 읽습니다. 사설 IP 서버는 같은 VPC 의 배스천을 점프 호스트로 잇고, 키는 ~/.ssh/&lt;키 이름&gt;.pem 에서 찾습니다.</span>
      </div>

      {scanning && <div className="faint">검색 중...</div>}
      {!scanning && cands.length === 0 && (
        <div>
          <div>가져올 세션을 찾지 못했습니다.</div>
          <div className="set-note">PuTTY에 저장된 세션이 없거나 ~/.ssh/config가 없습니다.</div>
        </div>
      )}

      {cands.length > 0 && (
        <>
          <Field label="넣을 폴더 이름" help="이름이 prod-/stg-/dev- 로 시작하면 운영/기능테스트/개발 환경이 자동 지정됩니다. 비우면 최상위에 넣습니다.">
            <input className="input" aria-label="넣을 폴더 이름" value={folder} onChange={(e) => setFolder(e.target.value)} />
          </Field>
          <div className="import-wrap">
            <table className="import-table">
              <thead>
                <tr>
                  <th style={{ width: 36 }}>
                    <input type="checkbox" aria-label="전체 선택" checked={allChecked} onChange={toggleAll} />
                  </th>
                  <th>이름</th>
                  <th>주소</th>
                  <th>종류</th>
                  <th>출처</th>
                </tr>
              </thead>
              <tbody>
                {cands.map((c) => (
                  <tr key={c.key}>
                    <td>
                      <input type="checkbox" aria-label={`${c.name} 선택`} checked={sel.has(c.key)} onChange={() => toggle(c.key)} />
                    </td>
                    <td style={{ fontWeight: 700 }}>{c.name ?? hostOf(c).alias ?? ''}</td>
                    <td>
                      <span className="mono" style={{ fontSize: 12 }}>{addrOf(c)}</span>
                      {c.keyFile && <KeyRound size={12} style={{ marginLeft: 6, verticalAlign: '-2px' }} aria-label="키 파일 있음" />}
                    </td>
                    <td>{kindOf(c)}</td>
                    <td title={c.note ?? undefined}>
                      {SOURCE_LABEL[c.source] ?? String(c.source ?? '')}
                      {c.source === 'aws' && c.jumpVia ? ' · 배스천 경유' : ''}
                      {typeof c.note === 'string' && c.note.includes('키 파일 없음') && <span className="danger-note"> (키 없음)</span>}
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
