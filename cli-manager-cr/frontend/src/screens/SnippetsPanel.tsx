// 명령 모음(스니펫) 패널과 편집 대화상자. 명령과 매크로를 한 목록에 둔다.
// 근거: spec/ui/sftp-snippets.md §4, spec/behavior/terminal.md §7(명령)·§7.1(변수)·§11(매크로), spec/data/store.md §7.
import { useEffect, useState, type MouseEvent } from 'react';
import { Circle, ListPlus, Pencil, Plus, Search, Square, Trash2 } from 'lucide-react';
import { Modal } from '../ui/Modal';
import { Field, ToggleRow } from '../ui/Controls';
import { useApp } from '../state/app';
import { useUi } from '../state/ui';
import { runMacroOnTargets, stopMacro, toggleRecording, useMacros } from '../state/macros';
import type { Snippet } from '../ipc/types';
import { focusedSessionId, notifyNoFocus, sendSnippet } from '../lib/termSend';
import { fillVars, initialValues, parseVars, parseVarsAll, rememberValues, type SnippetVar } from '../lib/snippetVars';
import { isMacro } from '../lib/macro';
import { VarsDialog } from './VarsDialog';
import { MacroEditor } from './MacroEditor';
import { t } from '../i18n';

function newSnippetId(): string {
  return `sn-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 7)}`;
}

function matches(s: Snippet, q: string): boolean {
  if (!q) return true;
  const needle = q.toLowerCase();
  return s.name.toLowerCase().includes(needle) || s.body.toLowerCase().includes(needle);
}

/** 항목의 변수: 명령은 본문에서, 매크로는 모든 단계의 보낼 글자에서. */
function varsOf(s: Snippet): SnippetVar[] {
  return isMacro(s) ? parseVarsAll(s.steps.map((x) => x.send)) : parseVars(s.body);
}

/** 변수 값을 채운 미리 보기 글. */
function previewOf(s: Snippet, values: Record<string, string>): string {
  return isMacro(s) ? s.steps.map((x) => fillVars(x.send, values)).join('\n') : fillVars(s.body, values);
}

type Editing =
  | { kind: 'command'; item: Snippet; isNew: boolean }
  | { kind: 'macro'; item: Snippet; isNew: boolean; recorded?: boolean };

export function SnippetsPanel() {
  const snippets = useApp((s) => s.store?.snippets ?? []);
  const patchStore = useApp((s) => s.patchStore);
  const confirm = useUi((s) => s.confirm);
  const running = useMacros((s) => s.running);
  const recording = useMacros((s) => s.recording);
  const recorded = useMacros((s) => s.recorded);
  const [query, setQuery] = useState('');
  const [editing, setEditing] = useState<Editing | null>(null);
  /** 변수 값을 묻는 중인 항목(§4.3). */
  const [asking, setAsking] = useState<Snippet | null>(null);

  // 녹화가 끝나면(어디서 멈췄든) 모은 단계로 새 매크로 편집 창을 연다.
  useEffect(() => {
    if (!recorded) return;
    useMacros.setState({ recorded: null });
    setEditing({ kind: 'macro', isNew: true, recorded: true, item: { id: newSnippetId(), name: '', body: '', sendEnter: true, steps: recorded } });
  }, [recorded]);

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
      title: isMacro(s) ? t('macro.deleteTitle') : t('snip.deleteTitle'),
      body: t('snip.deleteBody', { name: s.name }),
      okText: t('common.delete'),
      danger: true,
    });
    if (ok) await save(snippets.filter((x) => x.id !== s.id));
  };

  /**
   * 값을 채워 실제로 보낸다. 명령·매크로 모두 지금 입력 대상에 간다: 일괄 입력이 켜진 탭이면 그 탭의 모든 창(D146).
   * 매크로의 운영 서버 확인은 runMacroOnTargets 가 한다.
   */
  const execute = async (s: Snippet, values: Record<string, string>) => {
    if (!isMacro(s)) {
      if (!sendSnippet(fillVars(s.body, values), s.sendEnter)) notifyNoFocus();
      return;
    }
    await runMacroOnTargets(s, values);
  };

  const onRun = (s: Snippet) => {
    // 보낼 창이 없으면 값을 묻기 전에 알린다.
    if (!focusedSessionId()) {
      notifyNoFocus();
      return;
    }
    // 변수가 있으면 값을 먼저 묻는다 (terminal.md §7.1).
    if (varsOf(s).length > 0) {
      setAsking(s);
      return;
    }
    void execute(s, {});
  };

  const onRunWith = (s: Snippet, values: Record<string, string>) => {
    rememberValues(s.id, values);
    setAsking(null);
    void execute(s, values);
  };

  const blank = (): Snippet => ({ id: newSnippetId(), name: '', body: '', sendEnter: true });

  /** 이 매크로를 돌리고 있는 세션(있으면 그 진행 상태). */
  const runOf = (s: Snippet) => Object.entries(running).find(([, r]) => r.macroId === s.id);

  return (
    <div className="rpanel-snip" style={{ display: 'flex', flexDirection: 'column', flex: 1, minHeight: 0 }}>
      <div className="snip-tools">
        <div style={{ position: 'relative', flex: 1, minWidth: 0 }}>
          <Search size={13} style={{ position: 'absolute', left: 8, top: 8, color: 'var(--fg-3)' }} />
          <input
            className="input snip-search"
            style={{ paddingLeft: 26 }}
            placeholder={t('snip.search')}
            aria-label={t('snip.search')}
            value={query}
            onChange={(e) => setQuery(e.target.value)}
          />
        </div>
        <button
          className={`icon-btn small${recording ? ' recording' : ''}`}
          title={recording ? t('macro.recordStop', { count: recording.count }) : t('macro.record')}
          aria-label={recording ? t('macro.recordStop', { count: recording.count }) : t('macro.record')}
          aria-pressed={!!recording}
          onClick={toggleRecording}
        >
          {recording ? <Square size={13} /> : <Circle size={13} />}
        </button>
        <button className="icon-btn small" title={t('macro.new')} aria-label={t('macro.new')} onClick={() => setEditing({ kind: 'macro', isNew: true, item: blank() })}>
          <ListPlus size={15} />
        </button>
        <button className="icon-btn small" title={t('snip.new')} aria-label={t('snip.new')} onClick={() => setEditing({ kind: 'command', isNew: true, item: blank() })}>
          <Plus size={15} />
        </button>
      </div>

      {snippets.length === 0 ? (
        <div className="snip-empty">
          <div>{t('snip.empty')}</div>
          <div className="faint">{t('snip.emptyBroadcast')}</div>
        </div>
      ) : (
        <div className="snip-list">
          {shown.map((s) => {
            const macro = isMacro(s);
            const run = macro ? runOf(s) : undefined;
            return (
              <div
                key={s.id}
                className="snip-item"
                title={macro ? t('macro.runTitle') : t('snip.runTitle')}
                role="button"
                tabIndex={0}
                onClick={() => onRun(s)}
                onKeyDown={(e) => {
                  if (e.key === 'Enter') onRun(s);
                }}
              >
                <div className="snip-main">
                  <div className="snip-name">
                    {s.name || t('snip.noName')}
                    {/* 종류 배지: 매크로는 강조색, 명령은 초록 — 한 목록에 섞여 있어도 한눈에 가려진다 (D143). */}
                    {macro
                      ? <span className="snip-badge">{t('macro.badge', { count: s.steps.length })}</span>
                      : <span className="snip-badge cmd">{t('snip.badge')}</span>}
                  </div>
                  <div className="snip-body">
                    {run ? t('macro.running', { step: run[1].step + 1, total: run[1].total }) : macro ? s.steps.map((x) => x.send).join(' ⏎ ') : s.body}
                  </div>
                </div>
                <div className="snip-actions">
                  {run ? (
                    <button
                      className="icon-btn small"
                      title={t('macro.stop')}
                      aria-label={t('macro.stop')}
                      onClick={(e) => {
                        e.stopPropagation();
                        stopMacro(run[0]);
                      }}
                    >
                      <Square size={13} />
                    </button>
                  ) : (
                    <>
                      <button
                        className="icon-btn small"
                        title={t('common.edit')}
                        aria-label={t('common.edit')}
                        onClick={(e) => {
                          e.stopPropagation();
                          setEditing({ kind: macro ? 'macro' : 'command', item: s, isNew: false });
                        }}
                      >
                        <Pencil size={13} />
                      </button>
                      <button className="icon-btn small" title={t('common.delete')} aria-label={t('common.delete')} onClick={(e) => void onDelete(s, e)}>
                        <Trash2 size={13} />
                      </button>
                    </>
                  )}
                </div>
              </div>
            );
          })}
        </div>
      )}

      {asking && (
        <VarsDialog
          title={asking.name || t('snip.noName')}
          vars={varsOf(asking)}
          initial={initialValues(asking.id, varsOf(asking))}
          preview={(values) => previewOf(asking, values)}
          onCancel={() => setAsking(null)}
          onRun={(values) => onRunWith(asking, values)}
        />
      )}

      {editing?.kind === 'command' && (
        <SnippetEditor initial={editing.item} isNew={editing.isNew} onCancel={() => setEditing(null)} onSave={(d) => void onSaved(d)} />
      )}
      {editing?.kind === 'macro' && (
        <MacroEditor initial={editing.item} isNew={editing.isNew} recorded={editing.recorded} onCancel={() => setEditing(null)} onSave={(d) => void onSaved(d)} />
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
      title={isNew ? t('snip.new') : t('snip.edit')}
      size="sm"
      onClose={onCancel}
      foot={
        <>
          <button className="btn" onClick={onCancel}>{t('common.cancel')}</button>
          <button className="btn primary" onClick={submit} disabled={!canSave}>{t('common.save')}</button>
        </>
      }
    >
      <Field label={t('snip.name')}>
        <input
          className="input"
          autoFocus
          placeholder={t('snip.namePlaceholder')}
          value={name}
          onChange={(e) => setName(e.target.value)}
        />
      </Field>
      <Field label={t('snip.command')} help={`${t('snip.commandHelp')} · ${t('snip.varHelp', { a: '{{host}}', b: '{{port:22}}' })}`}>
        <textarea
          className="input mono"
          rows={5}
          placeholder="tail -f /var/log/nginx/error.log"
          value={body}
          onChange={(e) => setBody(e.target.value)}
        />
      </Field>
      <ToggleRow title={t('snip.sendEnter')} checked={sendEnter} onChange={setSendEnter} />
    </Modal>
  );
}
