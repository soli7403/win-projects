// 설정 8 · 정보·단축키. 근거: spec/ui/settings.md §8, spec/behavior/shortcuts.md §4, spec/acceptance G6·G7.
// 결정 D8: "원본" 필드는 두지 않는다.
import { useEffect, useState } from 'react';
import { ToggleRow } from '../ui/Controls';
import { LogoMark } from '../ui/Logo';
import { SHORTCUT_LABELS } from '../lib/shortcuts';
import { getBackend } from '../ipc/backend';
import { useApp } from '../state/app';
import { useUpdate } from '../state/update';
import { useSetting, Group } from './Settings';

export function SettingsAbout() {
  const [updateCheck, setUpdateCheck] = useSetting('updateCheck');
  const checkNow = useUpdate((s) => s.checkNow);
  const [info, setInfo] = useState<{ version?: string; dataDir?: string }>({});
  const pushToast = useApp((s) => s.pushToast);

  useEffect(() => {
    getBackend()
      .invoke<{ version?: string; dataDir?: string }>('app.info')
      .then((i) => setInfo(i ?? {}))
      .catch((e) => pushToast({ kind: 'error', text: String(e?.message ?? e) }));
    // pushToast 는 안정적인 참조다.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  return (
    <>
      <div style={{ display: 'flex', gap: 14, alignItems: 'center' }}>
        <LogoMark size={44} />
        <div>
          <div style={{ fontSize: 16, fontWeight: 700 }}>DariTerm {info.version ?? ''}</div>
          <div className="faint" style={{ fontSize: 12.5 }}>SSH · Telnet · Serial · 로컬 터미널</div>
        </div>
      </div>

      <div className="set-stack">
        <div className="set-label">개발자</div>
        <div>kai</div>
      </div>
      <div className="set-stack">
        <div className="set-label">라이선스</div>
        <div>MIT · Copyright © 2026 kai</div>
      </div>

      <Group>
        <ToggleRow
          title="업데이트 자동 확인"
          desc="앱을 켤 때와 12시간마다 배포 저장소의 최신 릴리스를 확인합니다. 서버 목록·IP·계정 등 사용자 정보는 보내지 않으며, 확인에 실패해도(폐쇄망 등) 조용히 넘어갑니다"
          checked={updateCheck}
          onChange={setUpdateCheck}
        />
        <div style={{ paddingBottom: 10 }}>
          <button className="btn small-btn" type="button" onClick={() => void checkNow()}>업데이트 확인</button>
        </div>
      </Group>

      <div className="set-stack">
        <div className="set-label">데이터 폴더</div>
        <div className="detail-box">{info.dataDir ?? ''}</div>
      </div>

      <div>
        <div className="set-section-title" style={{ marginBottom: 8 }}>단축키</div>
        <div className="sc-table">
          {SHORTCUT_LABELS.map((s) => (
            <ShortcutRow key={s.name} name={s.name} keys={s.keys} />
          ))}
        </div>
      </div>
    </>
  );
}

function ShortcutRow({ name, keys }: { name: string; keys: string }) {
  return (
    <>
      <span className="sc-name">{name}</span>
      <span className="kbd">{keys}</span>
    </>
  );
}
