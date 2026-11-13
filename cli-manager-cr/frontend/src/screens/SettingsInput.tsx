// 설정 2 · 입력·붙여넣기. 근거: spec/ui/settings.md §2, spec/data/store.md §4 (기본 패턴), spec/acceptance G3.
import { useEffect, useState } from 'react';
import { ToggleRow, Field } from '../ui/Controls';
import { DANGEROUS_PATTERNS } from '../lib/defaults';
import { getBackend } from '../ipc/backend';
import { useApp } from '../state/app';
import { saveSettings } from '../state/settings';
import { useSetting, Group } from './Settings';

/** 플랫폼 조회(Ctrl+V 토글은 Windows 에서만 보인다). 알 수 없으면 보인다. */
function usePlatform(): string | undefined {
  const [p, setP] = useState<string | undefined>(undefined);
  useEffect(() => {
    getBackend()
      .invoke<{ platform?: string }>('app.info')
      .then((i) => setP(i?.platform))
      .catch(() => setP(undefined));
  }, []);
  return p;
}

export function SettingsInput() {
  const platform = usePlatform();
  const [copyOnSelect, setCopyOnSelect] = useSetting('copyOnSelect');
  const [rightClickPaste, setRightClickPaste] = useSetting('rightClickPaste');
  const [ctrlVPaste, setCtrlVPaste] = useSetting('ctrlVPaste');
  const [shiftEnterNewline, setShiftEnterNewline] = useSetting('shiftEnterNewline');
  const [osc52Copy, setOsc52Copy] = useSetting('osc52Copy');
  const [confirmMultilinePaste, setConfirmMultilinePaste] = useSetting('confirmMultilinePaste');
  const [guardDangerousOnProd, setGuardDangerousOnProd] = useSetting('guardDangerousOnProd');
  const patterns = useApp((s) => s.store?.settings.dangerousPatterns) ?? [];
  const [text, setText] = useState(patterns.join('\n'));
  const [prevKey, setPrevKey] = useState(patterns.join('\n'));
  // 외부 변경(기본값 복원 등)이 생기면 입력 칸을 맞춘다.
  if (prevKey !== patterns.join('\n')) {
    setPrevKey(patterns.join('\n'));
    setText(patterns.join('\n'));
  }

  const savePatterns = (raw: string) => {
    const list = raw.split(/\r?\n/).map((l) => l.trim()).filter((l) => l !== '');
    void saveSettings({ dangerousPatterns: list });
  };

  return (
    <>
      <Group>
        <ToggleRow
          title="드래그하면 바로 복사"
          desc="드래그로 선택하면 즉시 클립보드에 복사합니다 (PuTTY 방식)"
          checked={copyOnSelect}
          onChange={setCopyOnSelect}
        />
        <ToggleRow
          title="마우스 오른쪽 버튼으로 붙여넣기"
          desc="끄면 오른쪽 클릭에 메뉴가 뜹니다. 켜져 있어도 Shift+오른쪽 클릭은 메뉴입니다"
          checked={rightClickPaste}
          onChange={setRightClickPaste}
        />
        {platform !== 'linux' && platform !== 'darwin' && (
          <ToggleRow
            title="Ctrl+V로 붙여넣기"
            desc="끄면 Ctrl+V 는 그대로 서버로 보내고, 붙여넣기는 Ctrl+Shift+V 로 합니다"
            checked={ctrlVPaste}
            onChange={setCtrlVPaste}
          />
        )}
        <ToggleRow
          title="Shift+Enter 로 줄바꿈"
          desc="실행하지 않고 줄만 바꿉니다. Claude Code 같은 CLI의 여러 줄 입력용입니다 (Alt+Enter 와 같은 값 전송)"
          checked={shiftEnterNewline}
          onChange={setShiftEnterNewline}
        />
        <ToggleRow
          title="서버의 클립보드 쓰기 허용 (OSC 52)"
          desc="서버 쪽 tmux·vim·Claude Code 등의 클립보드 쓰기 요청을 이 PC 클립보드에 반영합니다. 읽기 요청은 항상 거부합니다"
          checked={osc52Copy}
          onChange={setOsc52Copy}
        />
        <ToggleRow
          title="여러 줄 붙여넣기 전에 묻기"
          desc="두 줄 이상 붙여넣을 때 실행 전에 확인합니다"
          checked={confirmMultilinePaste}
          onChange={setConfirmMultilinePaste}
        />
        <ToggleRow
          title="운영 서버 위험 명령 경고"
          desc="환경이 '운영'인 서버에서 아래 패턴(정규식)에 맞는 명령을 실행하거나 붙여넣으면 먼저 확인합니다"
          checked={guardDangerousOnProd}
          onChange={setGuardDangerousOnProd}
        />
      </Group>

      <Field label="위험 명령 목록 (정규식, 한 줄에 하나)">
        <textarea
          className="input mono"
          aria-label="위험 명령 목록"
          rows={7}
          style={{ height: 150, padding: '8px 10px', resize: 'vertical', fontSize: 12 }}
          value={text}
          onChange={(e) => setText(e.target.value)}
          onBlur={() => savePatterns(text)}
        />
      </Field>
      <div>
        <button
          className="btn"
          type="button"
          onClick={() => {
            setText(DANGEROUS_PATTERNS.join('\n'));
            void saveSettings({ dangerousPatterns: [...DANGEROUS_PATTERNS] });
          }}
        >
          기본값으로
        </button>
      </div>
    </>
  );
}
