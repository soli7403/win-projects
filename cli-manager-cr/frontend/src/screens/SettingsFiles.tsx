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
          SSH 터미널에서 <b>Ctrl+V</b>로 이미지(캡처)를 붙여넣거나 파일을 터미널로 끌어다 놓으면, 서버의 업로드 폴더로 전송한 뒤 그 <b>경로를 터미널에 입력</b>합니다. 서버에서 실행 중인 <b>Claude Code</b> 등 CLI는 이 경로로 이미지를 읽을 수 있습니다. 예: <span className="kbdish">이 화면의 오류 원인 알려줘</span> 입력 후 Ctrl+V.
        </div>
      </div>

      <Field label="서버 업로드 폴더" help="~ 는 서버의 홈 디렉터리입니다. 폴더가 없으면 자동으로 만듭니다.">
        <input
          className="input mono"
          aria-label="서버 업로드 폴더"
          value={draft}
          onChange={(e) => setDraft(e.target.value)}
          onBlur={() => {
            const v = draft.trim();
            if (v && v !== uploadDir) setUploadDir(v);
            else setDraft(uploadDir);
          }}
        />
      </Field>

      <Field label="오래된 업로드 자동 삭제 (일)" help="DariTerm이 올린 파일(tg- 로 시작)만 삭제합니다. 0이면 삭제하지 않습니다.">
        <NumberField label="오래된 업로드 자동 삭제 (일)" value={cleanup} min={0} max={36500} fallback={7} onCommit={(v) => void saveSettings({ uploadCleanupDays: v })} />
      </Field>

      <Field
        label="서버 ↔ 서버 복사 임시 폴더 (이 PC)"
        help="파일 패널에서 다른 서버 탭으로 끌어 복사할 때 이 폴더에 먼저 받은 뒤 대상 서버로 올리고, 끝나면 임시 사본을 지웁니다. 비우면 OS 임시 폴더를 씁니다."
      >
        <div style={{ display: 'flex', gap: 6 }}>
          <input
            className="input mono"
            style={{ flex: 1 }}
            aria-label="서버 ↔ 서버 복사 임시 폴더"
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
