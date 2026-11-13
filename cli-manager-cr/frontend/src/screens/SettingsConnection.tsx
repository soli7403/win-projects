// 설정 4 · 연결. 근거: spec/ui/settings.md §4, spec/behavior/lifecycle.md §3.
import { FolderOpen } from 'lucide-react';
import { ToggleRow } from '../ui/Controls';
import { getBackend, errorMessage } from '../ipc/backend';
import { useApp } from '../state/app';
import { useSetting, Group } from './Settings';

export function SettingsConnection() {
  const [autoReconnect, setAutoReconnect] = useSetting('autoReconnect');
  const [restoreLayout, setRestoreLayout] = useSetting('restoreLayout');
  const [sessionLog, setSessionLog] = useSetting('sessionLog');
  const pushToast = useApp((s) => s.pushToast);

  const openLogs = async () => {
    try {
      await getBackend().invoke('app.openLogs');
    } catch (e) {
      pushToast({ kind: 'error', text: errorMessage(e) });
    }
  };

  return (
    <>
      <Group>
        <ToggleRow
          title="연결이 끊기면 다시 연결"
          desc="연결이 끊기면 2·4·8…초로 간격을 늘리며 최대 8번 재연결합니다"
          checked={autoReconnect}
          onChange={setAutoReconnect}
        />
        <ToggleRow
          title="마지막 화면 배치 복원"
          desc="종료할 때 열린 탭과 창 배치를 저장하고, 다음 실행 때 같은 서버로 다시 연결합니다. 비밀번호·화면 내용은 저장하지 않으며, 목록에 없는 일회성 SSH 연결은 복원하지 않습니다"
          checked={restoreLayout}
          onChange={setRestoreLayout}
        />
        <ToggleRow
          title="터미널 출력 기록"
          desc="터미널 출력을 텍스트 파일로 남깁니다 (켠 뒤 새로 여는 연결부터)"
          checked={sessionLog}
          onChange={setSessionLog}
        />
      </Group>
      <div>
        <button className="btn" type="button" onClick={() => void openLogs()}>
          <FolderOpen size={14} /> 로그 폴더 열기
        </button>
      </div>
    </>
  );
}
