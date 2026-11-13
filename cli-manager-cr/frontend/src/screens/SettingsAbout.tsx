// 설정 8 · 정보·단축키. 근거: spec/ui/settings.md §8, spec/behavior/shortcuts.md §4, spec/acceptance G6·G7.
// 결정 D63(D8 대체): 참고한 프로젝트(BrightTerm v1.1.0, MIT)의 고지를 "참고한 프로젝트" 필드에 보인다.
import { useEffect, useState } from 'react';
import { ToggleRow } from '../ui/Controls';
import { LogoMark } from '../ui/Logo';
import { SHORTCUT_LABELS } from '../lib/shortcuts';
import { getBackend } from '../ipc/backend';
import { useApp } from '../state/app';
import { useUpdate } from '../state/update';
import { useSetting, Group } from './Settings';
import { REFERENCE_NOTICE } from '../lib/notices';

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
      <div className="set-stack">
        <div className="set-label">참고한 프로젝트</div>
        <div>{REFERENCE_NOTICE.name} — {REFERENCE_NOTICE.summary}</div>
        <div className="faint" style={{ fontSize: 12 }}>{REFERENCE_NOTICE.note}</div>
        {/* MIT 고지 전문: 접어 두고 필요할 때 펼친다 */}
        <details>
          <summary className="faint" style={{ fontSize: 12, cursor: 'pointer' }}>라이선스 전문 보기</summary>
          <div className="detail-box" style={{ marginTop: 6 }}>{REFERENCE_NOTICE.license}</div>
        </details>
      </div>

      <Group>
        <ToggleRow
          title="업데이트 자동 확인"
          desc="시작할 때와 12시간마다 새 릴리스를 확인합니다. 서버 목록·IP·계정 정보는 보내지 않으며, 폐쇄망 등으로 확인이 안 되면 조용히 건너뜁니다"
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
