// 폴더 편집 대화상자(작은 모달). 근거: spec/ui/host-editor.md §2.
// 상위 폴더 목록은 자기 하위 폴더를 거르지 않는다(명세의 알려진 한계). 끌어 놓기에서만 순환을 막는다.
import { useState, type FormEvent } from 'react';
import type { EnvKind, Group } from '../ipc/types';
import { ENV_INFO } from '../lib/defaults';
import { useApp } from '../state/app';
import { useUi } from '../state/ui';
import { Modal } from '../ui/Modal';
import { ColorPicker, Field, Segmented } from '../ui/Controls';
import { groupPath, sortGroups } from '../lib/tree';
import { canSaveFolder } from '../lib/hostForm';

/** 환경 옵션 앞의 색 점(spec §1.1.5). */
function envOption(v: EnvKind, text: string) {
  return (
    <span style={{ display: 'inline-flex', alignItems: 'center', gap: 6 }}>
      <span style={{ width: 7, height: 7, borderRadius: 99, background: ENV_INFO[v].color }} />
      {text}
    </span>
  );
}

const ENVS: { value: EnvKind; label: string }[] = [
  { value: 'none', label: '일반' },
  { value: 'dev', label: '개발' },
  { value: 'stage', label: '기능테스트' },
  { value: 'perf', label: '성능테스트' },
  { value: 'prod', label: '운영' },
  { value: 'device', label: '장비' },
];

export function FolderEditor({ folderId, parentId }: { folderId: string | null; parentId: string | null }) {
  const groups = useApp((s) => s.store?.groups ?? []);
  const patchStore = useApp((s) => s.patchStore);
  const closeEditor = useUi((s) => s.closeEditor);
  const original = folderId ? groups.find((g) => g.id === folderId) ?? null : null;

  const [name, setName] = useState(original?.name ?? '');
  const [parent, setParent] = useState<string>(original ? original.parentId ?? '' : parentId ?? '');
  const [env, setEnv] = useState<EnvKind>(original?.env ?? 'none');
  const [color, setColor] = useState<string | undefined>(original?.color);

  const choices = sortGroups(groups).filter((g) => g.id !== folderId);
  const canSave = canSaveFolder(name);

  const save = async (e?: FormEvent) => {
    e?.preventDefault();
    if (!canSave) return;
    const base: Group = original ?? { id: crypto.randomUUID(), parentId: null, name: '', sort: groups.length };
    const next: Group = {
      ...base,
      name: name.trim(),
      parentId: parent || null,
      env: env === 'none' ? undefined : env,
      color,
    };
    const list = original ? groups.map((g) => (g.id === original.id ? next : g)) : [...groups, next];
    await patchStore({ groups: list });
    closeEditor();
  };

  return (
    <Modal
      title={original ? '폴더 편집' : '새 폴더'}
      size="sm"
      onClose={closeEditor}
      foot={
        <>
          <button className="btn" onClick={closeEditor}>취소</button>
          <button className="btn primary" disabled={!canSave} onClick={() => void save()}>저장</button>
        </>
      }
    >
      <form onSubmit={(e) => void save(e)} style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
        <Field label="이름">
          <input className="input" autoFocus placeholder="예: 헤펠레코리아" value={name} onChange={(e) => setName(e.target.value)} />
        </Field>
        <Field label="상위 폴더">
          <select className="input" value={parent} onChange={(e) => setParent(e.target.value)}>
            <option value="">(최상위)</option>
            {choices.map((g) => (
              <option key={g.id} value={g.id}>{groupPath(groups, g.id)}</option>
            ))}
          </select>
        </Field>
        <Field label="환경" help="폴더 안의 서버가 이 환경 색을 물려받습니다">
          <Segmented value={env} options={ENVS.map((o) => ({ value: o.value, label: envOption(o.value, o.label) }))} onChange={setEnv} label="환경" />
        </Field>
        <Field label="색상">
          <ColorPicker value={color} onChange={setColor} />
        </Field>
      </form>
    </Modal>
  );
}
