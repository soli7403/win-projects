// 미리보기 대화상자(크게). 근거: spec/ui/sftp-snippets.md §3, spec/protocols.md §3(미리보기 조건).
import { useEffect, useState } from 'react';
import { FileText } from 'lucide-react';
import { Modal } from '../ui/Modal';
import { useUi } from '../state/ui';
import { getBackend, errorMessage } from '../ipc/backend';
import { typePaths, editLocally, downloadPaths } from './sftpOps';

const NO_PREVIEW = '미리보기를 지원하지 않는 파일입니다 (이진 파일 또는 512KB 초과).';

type Loaded =
  | { kind: 'loading' }
  | { kind: 'none' }
  | { kind: 'image'; data: string }
  | { kind: 'text'; data: string }
  | { kind: 'error'; message: string };

export function PreviewDialog() {
  const target = useUi((s) => s.preview);
  const close = useUi((s) => s.closePreview);
  const [loaded, setLoaded] = useState<Loaded>({ kind: 'loading' });

  useEffect(() => {
    if (!target) return;
    let cancelled = false;
    setLoaded({ kind: 'loading' });
    getBackend()
      .invoke<{ kind: 'image' | 'text'; data: string } | null>('sftp.preview', [target.sessionId, target.path])
      .then((r) => {
        if (cancelled) return;
        if (!r) setLoaded({ kind: 'none' });
        else setLoaded({ kind: r.kind, data: r.data });
      })
      .catch((e) => {
        if (!cancelled) setLoaded({ kind: 'error', message: errorMessage(e) });
      });
    return () => {
      cancelled = true;
    };
  }, [target]);

  if (!target) return null;
  const name = target.path.slice(target.path.lastIndexOf('/') + 1);

  const onType = () => {
    close();
    typePaths([target.path]);
  };
  const onEdit = async () => {
    if (await editLocally(target.sessionId, target.path)) close();
  };
  const onDownload = () => void downloadPaths(target.sessionId, [target.path]);

  let body;
  if (loaded.kind === 'loading') body = <div className="preview-msg">불러오는 중...</div>;
  else if (loaded.kind === 'none') body = <div className="preview-msg">{NO_PREVIEW}</div>;
  else if (loaded.kind === 'error') body = <div className="preview-msg" role="alert">{loaded.message}</div>;
  else if (loaded.kind === 'image') body = <img className="preview-img" src={loaded.data} alt={name} />;
  else body = <pre className="preview-text selectable">{loaded.data}</pre>;

  return (
    <Modal
      title={name}
      icon={FileText}
      size="lg"
      onClose={close}
      foot={
        <div className="preview-foot">
          <span className="preview-path" title={target.path}>{target.path}</span>
          <button className="btn" onClick={onType}>경로를 터미널에 입력</button>
          <button className="btn" onClick={() => void onEdit()}>로컬 앱으로 편집</button>
          <button className="btn primary" onClick={onDownload}>다운로드</button>
        </div>
      }
    >
      <div className="preview-body">{body}</div>
    </Modal>
  );
}
