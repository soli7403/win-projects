// 설정 7 · 백업. 근거: spec/ui/settings.md §7, spec/data/sync-backup.md §3, spec/acceptance Y1·Y2·Y3.
import { getBackend, errorMessage } from '../ipc/backend';
import { useApp } from '../state/app';
import { useUi } from '../state/ui';

export function SettingsBackup() {
  const pushToast = useApp((s) => s.pushToast);
  const confirm = useUi((s) => s.confirm);

  const exportBackup = async () => {
    try {
      const path = await getBackend().invoke<string | false>('backup.export');
      if (path) pushToast({ kind: 'ok', text: `백업 저장: ${path}` });
    } catch (e) {
      pushToast({ kind: 'error', text: errorMessage(e) });
    }
  };

  const importBackup = async () => {
    const ok = await confirm({
      title: '백업 가져오기',
      body: '현재 서버 목록과 보관함이 백업 파일 내용으로 바뀝니다. 계속할까요?',
      okText: '가져오기',
      danger: true,
    });
    if (!ok) return;
    try {
      const done = await getBackend().invoke<boolean>('backup.import');
      if (!done) return;
      await useApp.getState().refreshStore();
      await useApp.getState().refreshVault();
      pushToast({ kind: 'ok', text: '백업을 가져왔습니다. 백업한 PC의 마스터 비밀번호로 잠금을 해제하세요.' });
    } catch (e) {
      pushToast({ kind: 'error', text: errorMessage(e) });
    }
  };

  return (
    <>
      <div className="set-notice">
        <div>
          백업 파일에는 서버 목록·설정·명령 모음과 <b>암호화된</b> 보관함이 들어 있습니다. 다른 PC에서 가져온 뒤 같은 마스터 비밀번호로 열 수 있습니다.
        </div>
      </div>
      <div style={{ display: 'flex', gap: 10 }}>
        <button className="btn primary" type="button" onClick={() => void exportBackup()}>백업 내보내기</button>
        <button className="btn" type="button" onClick={() => void importBackup()}>백업 가져오기</button>
      </div>
    </>
  );
}
