// 스니펫 탭과 편집 대화상자. 근거: spec/ui/sftp-snippets.md §4, spec/behavior/terminal.md §7, spec/data/store.md §7.
import { useState, type MouseEvent } from 'react';
import { Pencil, Plus, Search, Trash2 } from 'lucide-react';
import { Modal } from '../ui/Modal';
import { Field, ToggleRow } from '../ui/Controls';
import { useApp } from '../state/app';
import { useUi } from '../state/ui';
import type { Snippet } from '../ipc/types';
import { notifyNoFocus, sendSnippet } from '../lib/termSend';

function newSnippetId(): string {
  return `sn-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 7)}`;
}

function matches(s: Snippet, q: string): boolean {
  if (!q) return true;
  const needle = q.toLowerCase();
  return s.name.toLowerCase().includes(needle) || s.body.toLowerCase().includes(needle);
}

export function SnippetsPanel() {
  const snippets = useApp((s) => s.store?.snippets ?? []);
  const patchStore = useApp((s) => s.patchStore);
  const confirm = useUi((s) => s.confirm);
  const [query, setQuery] = useState('');
  const [editing, setEditing] = useState<Snippet | 'new' | null>(null);

  const shown = snippets.filter((s) => matches(s, query.trim()));

  const save = async (next: Snippet[]) => {
    await patchStore({ snippets: next });
  };

  const onSaved = async (draft: Snippet) => {
    const exists = snippets.some((s) => s.id === draft.id);
    const next = exists ? snippets.map((s) => (s.id === draft.id ? draft : s)) : [...snippets, draft];
    setEditing(null);
    await save(next);
  };

  const onDelete = async (s: Snippet, ev: MouseEvent) => {
    ev.stopPropagation();
    const ok = await confirm({
      title: '명령 삭제',
      body: `"${s.name}"을(를) 삭제할까요?`,
      okText: '삭제',
      danger: true,
    });
    if (ok) await save(snippets.filter((x) => x.id !== s.id));
  };

  const onRun = (s: Snippet) => {
    if (!sendSnippet(s.body, s.sendEnter)) notifyNoFocus();
  };

  return (
    <div className="rpanel-snip" style={{ display: 'flex', flexDirection: 'column', flex: 1, minHeight: 0 }}>
      <div className="snip-tools">
        <div style={{ position: 'relative', flex: 1, minWidth: 0 }}>
          <Search size={13} style={{ position: 'absolute', left: 8, top: 8, color: 'var(--fg-3)' }} />
          <input
            className="input snip-search"
            style={{ paddingLeft: 26 }}
            placeholder="명령 찾기"
            aria-label="명령 찾기"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
          />
        </div>
        <button className="icon-btn small" title="새 명령" aria-label="새 명령" onClick={() => setEditing('new')}>
          <Plus size={15} />
        </button>
      </div>

      {snippets.length === 0 ? (
        <div className="snip-empty">
          <div>자주 쓰는 명령을 저장해 두고 클릭 한 번으로 실행하세요.</div>
          <div className="faint">일괄 입력이 켜진 탭에서는 모든 창에 입력됩니다.</div>
        </div>
      ) : (
        <div className="snip-list">
          {shown.map((s) => (
            <div
              key={s.id}
              className="snip-item"
              title="누르면 지금 창에 입력합니다"
              role="button"
              tabIndex={0}
              onClick={() => onRun(s)}
              onKeyDown={(e) => {
                if (e.key === 'Enter') onRun(s);
              }}
            >
              <div className="snip-main">
                <div className="snip-name">{s.name || '(이름 없음)'}</div>
                <div className="snip-body">{s.body}</div>
              </div>
              <div className="snip-actions">
                <button
                  className="icon-btn small"
                  title="수정"
                  aria-label="수정"
                  onClick={(e) => {
                    e.stopPropagation();
                    setEditing(s);
                  }}
                >
                  <Pencil size={13} />
                </button>
                <button className="icon-btn small" title="삭제" aria-label="삭제" onClick={(e) => void onDelete(s, e)}>
                  <Trash2 size={13} />
                </button>
              </div>
            </div>
          ))}
        </div>
      )}

      {editing && (
        <SnippetEditor
          initial={editing === 'new' ? { id: newSnippetId(), name: '', body: '', sendEnter: true } : editing}
          isNew={editing === 'new'}
          onCancel={() => setEditing(null)}
          onSave={(d) => void onSaved(d)}
        />
      )}
    </div>
  );
}

function SnippetEditor({
  initial,
  isNew,
  onCancel,
  onSave,
}: {
  initial: Snippet;
  isNew: boolean;
  onCancel: () => void;
  onSave: (s: Snippet) => void;
}) {
  const [name, setName] = useState(initial.name);
  const [body, setBody] = useState(initial.body);
  const [sendEnter, setSendEnter] = useState(initial.sendEnter);
  const canSave = body.trim().length > 0;
  const submit = () => {
    if (!canSave) return;
    onSave({ id: initial.id, name: name.trim(), body, sendEnter });
  };
  return (
    <Modal
      title={isNew ? '새 명령' : '명령 수정'}
      size="sm"
      onClose={onCancel}
      foot={
        <>
          <button className="btn" onClick={onCancel}>취소</button>
          <button className="btn primary" onClick={submit} disabled={!canSave}>저장</button>
        </>
      }
    >
      <Field label="이름">
        <input
          className="input"
          autoFocus
          placeholder="예: 로그 보기"
          value={name}
          onChange={(e) => setName(e.target.value)}
        />
      </Field>
      <Field label="명령" help="여러 줄은 순서대로 입력됩니다">
        <textarea
          className="input mono"
          rows={5}
          placeholder="tail -f /var/log/nginx/error.log"
          value={body}
          onChange={(e) => setBody(e.target.value)}
        />
      </Field>
      <ToggleRow title="입력하고 바로 실행" checked={sendEnter} onChange={setSendEnter} />
    </Modal>
  );
}
