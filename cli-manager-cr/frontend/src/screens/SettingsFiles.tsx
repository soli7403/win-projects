// 설정 3 · 이미지·파일 전송. 근거: spec/ui/settings.md §3, spec/data/store.md §4.
import { useState } from 'react';
import { ImageUp } from 'lucide-react';
import { ToggleRow, Field } from '../ui/Controls';
import { saveSettings } from '../state/settings';
import { NumberField, useSetting, Group } from './Settings';
import { getBackend } from '../ipc/backend';

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
          SSH 터미널에 <b>Ctrl+V</b>로 캡처 이미지를 붙여넣거나 파일을 끌어 놓으면, 서버 업로드 폴더에 올린 뒤 <b>그 경로를 터미널에 입력</b>합니다. 서버의 <b>Claude Code</b> 같은 CLI가 이 경로로 이미지를 읽습니다. 예: <span className="kbdish">이 화면의 오류 원인 알려줘</span> 입력 후 Ctrl+V.
        </div>
      </div>

      <Field label="서버에 올릴 폴더" help="업로드 파일을 둘 서버 경로입니다. ~ 는 홈 디렉터리이며, 없으면 만듭니다.">
        <input
          className="input mono"
          aria-label="서버에 올릴 폴더"
          value={draft}
          onChange={(e) => setDraft(e.target.value)}
          onBlur={() => {
            const v = draft.trim();
            if (v && v !== uploadDir) setUploadDir(v);
            else setDraft(uploadDir);
          }}
        />
      </Field>

      <Field label="올린 파일 자동 정리 (일)" help="이 일수보다 오래된 업로드를 지웁니다. DariTerm이 올린 파일(tg- 접두어)만 대상이며, 0이면 끕니다.">
        <NumberField label="올린 파일 자동 정리 (일)" value={cleanup} min={0} max={36500} fallback={7} onCommit={(v) => void saveSettings({ uploadCleanupDays: v })} />
      </Field>

      <Field
        label="서버 간 복사용 임시 폴더 (이 PC)"
        help="서버→서버 복사는 이 PC 폴더로 먼저 받은 뒤 대상 서버로 올리고, 끝나면 임시 파일을 지웁니다. 비우면 OS 임시 폴더를 씁니다."
      >
        <div style={{ display: 'flex', gap: 6 }}>
          <input
            className="input mono"
            style={{ flex: 1 }}
            aria-label="서버 간 복사용 임시 폴더"
            placeholder="(OS 임시 폴더)/dariterm-transfer"
            value={tempDraft}
            onChange={(e) => setTempDraft(e.target.value)}
            onBlur={() => {
              const v = tempDraft.trim();
              if (v !== (tempDir ?? '')) setTempDir(v);
            }}
          />
          <button className="btn small-btn" onClick={() => void pickTemp()}>찾아보기…</button>
        </div>
      </Field>

      <Group>
        <ToggleRow title="공백이 있는 경로는 따옴표로 감싸기" checked={quote} onChange={setQuote} />
      </Group>
    </>
  );
}
