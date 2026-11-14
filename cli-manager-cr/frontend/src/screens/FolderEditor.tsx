// 폴더 편집 대화상자(작은 모달). 근거: spec/ui/host-editor.md §2.
// 상위 폴더 목록은 자기 하위 폴더를 거르지 않는다(명세의 알려진 한계). 끌어 놓기에서만 순환을 막는다.
import { useState, type FormEvent } from 'react';
import type { EnvKind, Group } from '../ipc/types';
import { ENV_INFO } from '../lib/defaults';
import { t, type MsgId } from '../i18n';
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

/** 환경 선택 순서. 이름은 ENV_INFO 의 문구 키(D75). */
const ENVS: { value: EnvKind; label: MsgId }[] = [
  { value: 'none', label: 'env.none' },
  { value: 'dev', label: 'env.dev' },
  { value: 'stage', label: 'env.stage' },
  { value: 'perf', label: 'env.perf' },
  { value: 'prod', label: 'env.prod' },
  { value: 'device', label: 'env.device' },
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
      title={original ? t('folder.edit') : t('quick.newFolder')}
      size="sm"
      onClose={closeEditor}
      foot={
        <>
          <button className="btn" onClick={closeEditor}>{t('common.cancel')}</button>
          <button className="btn primary" disabled={!canSave} onClick={() => void save()}>{t('common.save')}</button>
        </>
      }
    >
      <form onSubmit={(e) => void save(e)} style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
        <Field label={t('snip.name')}>
          <input className="input" autoFocus placeholder={t('folder.namePlaceholder')} value={name} onChange={(e) => setName(e.target.value)} />
        </Field>
        <Field label={t('folder.parent')}>
          <select className="input" value={parent} onChange={(e) => setParent(e.target.value)}>
            <option value="">{t('folder.root')}</option>
            {choices.map((g) => (
              <option key={g.id} value={g.id}>{groupPath(groups, g.id)}</option>
            ))}
          </select>
        </Field>
        <Field label={t('folder.env')} help={t('folder.envHelp')}>
          <Segmented value={env} options={ENVS.map((o) => ({ value: o.value, label: envOption(o.value, t(o.label)) }))} onChange={setEnv} label={t('folder.env')} />
        </Field>
        <Field label={t('color.label')}>
          <ColorPicker value={color} onChange={setColor} />
        </Field>
      </form>
    </Modal>
  );
}
