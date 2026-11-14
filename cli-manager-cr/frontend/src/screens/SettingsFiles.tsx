// 설정 3 · 이미지·파일 전송. 근거: spec/ui/settings.md §3, spec/data/store.md §4.
import { useState } from 'react';
import { ImageUp } from 'lucide-react';
import { ToggleRow, Field } from '../ui/Controls';
import { saveSettings } from '../state/settings';
import { Rich } from '../i18n/Rich';
import { NumberField, useSetting, Group } from './Settings';
import { getBackend } from '../ipc/backend';
import { t } from '../i18n';

export function SettingsFiles() {
  const [uploadDir, setUploadDir] = useSetting('uploadDir');
  const [cleanup] = useSetting('uploadCleanupDays');
  const [quote, setQuote] = useSetting('insertPathQuote');
  const [tempDir, setTempDir] = useSetting('transferTempDir');
  const [draft, setDraft] = useState(uploadDir);
  const [prev, setPrev] = useState(uploadDir);
  if (prev !== uploadDir) {
    setPrev(uploadDir);
    setDraft(uploadDir);
  }
  // 서버 ↔ 서버 복사 임시 폴더 (behavior/file-panel.md §4). 빈 값 허용(= OS 임시 폴더).
  const [tempDraft, setTempDraft] = useState(tempDir ?? '');
  const [tempPrev, setTempPrev] = useState(tempDir);
  if (tempPrev !== tempDir) {
    setTempPrev(tempDir);
    setTempDraft(tempDir ?? '');
  }
  const pickTemp = async () => {
    const d = await getBackend().invoke<string | null>('dialog.chooseDir').catch(() => null);
    if (d) setTempDir(d);
  };

  return (
    <>
      <div className="set-notice">
        <ImageUp size={16} style={{ flex: 'none', marginTop: 2 }} />
        <div>
          <Rich text={t('files.intro')} />
        </div>
      </div>

      <Field label={t('files.uploadDir')} help={t('files.uploadDirHelp')}>
        <input
          className="input mono"
          aria-label={t('files.uploadDir')}
          value={draft}
          onChange={(e) => setDraft(e.target.value)}
          onBlur={() => {
            const v = draft.trim();
            if (v && v !== uploadDir) setUploadDir(v);
            else setDraft(uploadDir);
          }}
        />
      </Field>

      <Field label={t('files.cleanup')} help={t('files.cleanupHelp')}>
        <NumberField label={t('files.cleanup')} value={cleanup} min={0} max={36500} fallback={7} onCommit={(v) => void saveSettings({ uploadCleanupDays: v })} />
      </Field>

      <Field
        label={t('files.tempDir')}
        help={t('files.tempDirHelp')}
      >
        <div style={{ display: 'flex', gap: 6 }}>
          <input
            className="input mono"
            style={{ flex: 1 }}
            aria-label={t('files.tempDirAria')}
            placeholder={t('files.tempDirPlaceholder')}
            value={tempDraft}
            onChange={(e) => setTempDraft(e.target.value)}
            onBlur={() => {
              const v = tempDraft.trim();
              if (v !== (tempDir ?? '')) setTempDir(v);
            }}
          />
          <button className="btn small-btn" onClick={() => void pickTemp()}>{t('common.browseEllipsis')}</button>
        </div>
      </Field>

      <Group>
        <ToggleRow title={t('files.quotePaths')} checked={quote} onChange={setQuote} />
      </Group>
    </>
  );
}
