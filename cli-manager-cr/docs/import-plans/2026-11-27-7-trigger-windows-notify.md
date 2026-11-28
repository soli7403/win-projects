# 출력 트리거의 Windows 알림 — 앱이 뒤에 있어도 알기

**작성**: 2026-11-27 · **상태**: 코드·자동 테스트·빌드 완료(D145). 실제 Windows 알림 센터에 뜨는지는 수동 검증으로 남음(§H) · **대상**: cli-manager-cr(DariTerm) · **Known issue**: 없음
**관련**: [2026-11-27-4-output-triggers.md](./2026-11-27-4-output-triggers.md)(D140, 그 문서 §H 끝의 "Windows 알림은 후속 과제"), [-11-](./2026-11-27-11-trigger-per-host.md)(서버별 규칙 — 같은 실행부를 고쳤다)

---

## 0. 한 줄 요약

출력 트리거의 "알림"은 앱 안의 토스트뿐이라, DariTerm 창이 뒤에 있으면 보이지 않았다. 알림을 켠 규칙이 맞았을 때 **앱 창에 포커스가 없으면** Windows 알림 센터에도 띄운다. 끌 수 있는 설정 하나(`triggerOsNotify`, 기본 켬)를 두고, 백엔드는 Wails v2.17.1 에 이미 있는 알림 기능(`runtime.SendNotification`, 내부는 go-toast)을 쓴다.

## 1. 요구와 결정

| 항목 | 결정 | 까닭 |
|---|---|---|
| 언제 띄우나 | 규칙의 "알림"이 켜져 있고, `document.hasFocus()` 가 거짓일 때 | 앱이 앞에 있으면 토스트로 충분하다. 둘 다 뜨면 겹친다 |
| 끌 수 있나 | 설정 > 출력 규칙의 토글 "앱이 뒤에 있으면 Windows 알림도 띄우기"(`triggerOsNotify`, 기본 **켬**) | 이 기능을 바라고 트리거를 켰을 사람이 대부분이다. 회의 중 등에 끄고 싶을 수 있다 |
| 규칙마다 고르나 | 아니다 | 줄에 체크가 넷이 되어 좁은 칸에서 넘친다. "알림" 하나에 "뒤에 있으면 OS 로도"를 얹는 것으로 충분하다 |
| 여러 줄이 한꺼번에 맞으면 | 한 번의 출력 묶음에서 맞은 줄은 **알림 하나**(본문에 줄바꿈으로) | 빌드 로그처럼 한 묶음에 여러 줄이 맞을 때 알림이 쏟아지지 않게 |
| 제목·본문 | 제목 `DariTerm · {세션 제목}`, 본문 = 맞은 줄들(줄마다 120자까지, 토스트와 같다) | 어느 창의 일인지 알림만 보고 알 수 있게 |
| 실패하면 | 한 번 실패하거나 띄울 수 없는 환경(`false`)이면 **앱이 켜져 있는 동안 다시 시도하지 않는다** | 실패할 때마다 백엔드 호출·오류가 반복되지 않게. 토스트는 그대로 |
| 알림 준비(레지스트리 등록) | 처음 알림을 띄울 때 한 번 `InitializeNotifications` | 트리거를 쓰지 않는 사람에게는 레지스트리를 쓰지 않는다 |
| 다른 대안 | 작업 표시줄 깜빡임 — 기각(무엇이 맞았는지 보이지 않는다) | |

## 2. 구현

### 2.1 백엔드 채널 `app.notify` — `internal/app/channels_clip.go`

창 동작 구현(`g_pWindowControl`, main.go 가 등록)이 새 인터페이스 `Notifier` 도 구현하면 그것을 부른다.

```go
// Notifier 는 OS 알림(Windows 알림 센터)을 띄울 수 있는 창 동작 구현이다 (출력 트리거, D145).
// main.go 의 Wails 구현이 갖는다. 없거나 실패하면 앱 안 알림만 보인다.
type Notifier interface {
	Notify(szTitle string, szBody string) error
}

// app.notify: OS 알림을 띄운다 (출력 트리거가 앱이 뒤에 있을 때 쓴다, D145). 띄웠으면 true.
pRouter.Handle("app.notify", func(slArgs []json.RawMessage) (any, error) {
	szTitle, errTitle := argString(slArgs, 0)
	if errTitle != nil {
		return nil, errTitle
	}
	szBody, _ := argString(slArgs, 1)
	notifier, bOk := g_pWindowControl.(Notifier)
	if !bOk {
		return false, nil
	}
	if errNotify := notifier.Notify(szTitle, szBody); errNotify != nil {
		return false, errNotify
	}
	return true, nil
})
```

### 2.2 Wails 알림 — `main.go`

```go
type wailsWindow struct {
	em *Emitter
	// 알림 준비는 처음 알림을 띄울 때 한 번만 한다 (트리거를 쓰지 않는 사용자에게는 레지스트리를 쓰지 않는다).
	onceNotify sync.Once
	errNotify  error
}

// Notify 는 OS 알림을 띄운다 (출력 트리거, D145). Windows 는 알림 센터의 토스트다.
func (pWin *wailsWindow) Notify(szTitle string, szBody string) error {
	ctxApp := pWin.em.context()
	if ctxApp == nil {
		return errors.New("app not started")
	}
	pWin.onceNotify.Do(func() {
		pWin.errNotify = runtime.InitializeNotifications(ctxApp)
	})
	if pWin.errNotify != nil {
		return pWin.errNotify
	}
	if !runtime.IsNotificationAvailable(ctxApp) {
		return errors.New("notifications unavailable")
	}
	return runtime.SendNotification(ctxApp, runtime.NotificationOptions{
		ID:    fmt.Sprintf("dariterm-%d", time.Now().UnixNano()),
		Title: szTitle,
		Body:  szBody,
	})
}
```

Wails v2.17.1 의 Windows 구현(`internal/frontend/desktop/windows/notifications.go`)을 읽어 확인한 것:

| 단계 | 하는 일 |
|---|---|
| `InitializeNotifications` | 실행 파일 이름을 앱 ID 로, `HKCU\Software\Classes\CLSID\{GUID}\LocalServer32` 에 활성화 경로를 쓰고, 알림 클릭용 COM 클래스 팩터리를 등록한다 |
| `SendNotification` | 토스트 XML 을 만들어 WinRT 로만 보낸다(go-toast 의 PowerShell 대체 경로는 쓰지 않는다) |
| 의존성 | go-toast 는 Wails 의 Windows 프런트엔드가 원래 끌어오는 모듈이라 새 의존성이 아니다(go.sum 에 이미 있음) |

### 2.3 설정 기본값

| 곳 | 값 |
|---|---|
| `internal/store/store.go` 기본 설정 | `"triggerOsNotify": true` |
| `frontend/src/lib/defaults.ts` | `triggerOsNotify: true` |
| `frontend/src/ipc/types.ts` `Settings` | `triggerOsNotify: boolean` |
| 계약 고정 파일 `frontend/src/ipc/contract/store.get*.json` | `go test ./internal/app -run TestContract -args -update` 로 다시 만듦 |

### 2.4 프런트 실행부 — `frontend/src/state/triggers.ts`

```ts
/** Windows 알림이 이번 실행에서 실패했으면 더 시도하지 않는다(실패할 때마다 오류를 만들지 않게). */
let osNotifyBroken = false;

/** 앱 창이 뒤에 있으면(포커스가 없으면) Windows 알림으로도 띄운다(D145). */
function notifyOs(title: string, body: string): void {
  if (osNotifyBroken) return;
  if (useApp.getState().store?.settings.triggerOsNotify === false) return;
  if (typeof document === 'undefined' || document.hasFocus()) return;
  void getBackend()
    .invoke<boolean>('app.notify', [title, body])
    .then((ok) => {
      if (ok === false) osNotifyBroken = true;
    })
    .catch(() => {
      osNotifyBroken = true;
    });
}
```

`run()` 안에서 알림을 켠 규칙의 줄을 모았다가 끝에서 한 번 부른다.

```ts
  let osLines: string[] = [];
  for (const f of fires) {
    const line = f.line.length > LINE_SHOWN_MAX ? `${f.line.slice(0, LINE_SHOWN_MAX)}…` : f.line;
    if (f.rule.notify) {
      push({ kind: 'info', text: t('trig.toast', { name, line }) });
      osLines = [...osLines, line];
    }
    // … 소리·자동 응답은 전과 같다
  }
  // 한 번에 맞은 줄들은 Windows 알림 하나로 묶는다.
  if (osLines.length > 0) notifyOs(t('trig.osTitle', { name }), osLines.join('\n'));
```

### 2.5 설정 화면 — `frontend/src/screens/SettingsRules.tsx`

출력 트리거 목록 위에 토글 한 줄(누르는 즉시 `saveSettings({ triggerOsNotify })`). 같은 작업에서 규칙 목록 편집부를 `TriggerRuleList` 로 빼 서버 편집에서도 쓰게 했다([-11-](./2026-11-27-11-trigger-per-host.md)).

```tsx
<div className="toggle-row">
  <div>
    <div className="t-title">{t('trig.osNotify')}</div>
    <div className="t-desc">{t('trig.osNotifyDesc')}</div>
  </div>
  <Switch checked={osNotify} onChange={(v) => void saveSettings({ triggerOsNotify: v })} label={t('trig.osNotify')} />
</div>
```

### 2.6 문구 (10개 언어)

| 키 | 한국어 |
|---|---|
| `trig.osNotify` | 앱이 뒤에 있으면 Windows 알림도 띄우기 |
| `trig.osNotifyDesc` | 알림을 켠 규칙이 맞았을 때 DariTerm 창이 앞에 있지 않으면 Windows 알림 센터에도 띄웁니다. 한 번에 맞은 줄은 알림 하나로 묶습니다. |
| `trig.osTitle` | DariTerm · {name} |

## 3. 명세

| 문서 | 바꾼 것 |
|---|---|
| spec/behavior/terminal.md §10 | "알림" 아래에 Windows 알림 규칙 |
| spec/ipc.md | 채널 `app.notify(title, body)` |
| spec/data/store.md §4 | `triggerOsNotify` 기본 true |
| spec/ui/settings.md §2b | 목록 위 토글과 설명 |

## 4. 테스트

`frontend/src/state/followups.test.ts` "트리거의 Windows 알림 (D145)"

| 테스트 | 확인하는 것 |
|---|---|
| 앱 창에 포커스가 없을 때만, 한 번에 맞은 줄을 알림 하나로 보낸다 | 포커스가 있으면 `app.notify` 를 부르지 않는다. 없으면 한 번, 인자 `['DariTerm · s1', 'ERR two\nFAIL three']` |
| 설정에서 끄면 보내지 않고, 한 번 실패하면 이번 실행에서는 다시 시도하지 않는다 | `triggerOsNotify: false` 면 0번. 켜고 실패시킨 뒤 다시 맞아도 호출이 늘지 않는다 |

계약 테스트(`internal/app/contract_test.go`)는 고정 파일을 갱신해 통과.

## 5. 검증 결과 (2026-11-27)

- `go build ./...`, `go vet ./...`, `go test -race ./...`(clipboard 의 KI-013 한 번 실패 → 재실행 통과), `npx tsc --noEmit`, vitest 465개, `wails build` 성공.
- **확인하지 못한 것**: 실제로 알림 센터에 뜨는지(화면을 볼 수 없다). 특히 **Microsoft Store(MSIX) 설치본**은 패키지 신원이 있는데 Wails 는 실행 파일 이름을 앱 ID 로 등록한다 — 스토어판에서 알림이 뜨지 않으면 `app.notify` 가 오류를 돌려주고 이후 시도를 멈추므로 앱 동작에는 영향이 없다. §H 4·5 로 확인한다.

## 6. 영향 범위와 위험

| 항목 | 내용 |
|---|---|
| 트리거를 쓰지 않는 사람 | 바뀌는 것 없음(알림 준비도 하지 않는다) |
| 레지스트리 | 처음 알림 때 `HKCU\Software\Classes\CLSID\{GUID}` 와 알림 범주 키를 쓴다(Wails 동작). MSIX 는 HKCU 쓰기가 패키지 안으로 가상화된다 |
| 알림 클릭 | 아무 동작도 걸지 않았다(알림을 눌러 그 창으로 가는 것은 후속 후보) |

## H. 수동 검증

| # | 하는 일 | 기대 | 결과 |
|---|---|---|---|
| 1 | 설정 > 출력 규칙 > 트리거 `DONE`(알림 켬), 토글 켜짐 확인 | 토글이 켜져 있다 | |
| 2 | 터미널에서 `sleep 5; echo DONE` 을 치고 곧바로 다른 프로그램 창을 앞으로 | 5초 뒤 Windows 알림 "DariTerm · {세션}" / "DONE" | |
| 3 | 같은 명령을 DariTerm 을 앞에 둔 채로 | 토스트만, Windows 알림은 없다 | |
| 4 | 설치 파일판에서 2 | 알림 센터에 남는다 | |
| 5 | 스토어판(MSIX)에서 2 | 알림이 뜨거나, 안 뜨면 토스트만 남고 오류 창은 없다 | |
| 6 | 토글을 끄고 2 | Windows 알림 없음 | |
| 7 | `printf 'DONE\nDONE again\n'` 이 한 번에 나오게(규칙 두 개 또는 같은 규칙 두 줄) | 알림 하나에 줄이 둘 | |

## 체크리스트

- [x] 백엔드 채널·Wails 알림 · [x] 설정 기본값·계약 파일 · [x] 실행부·설정 토글 · [x] 문구 10개 언어 · [x] 명세 · [x] 테스트 · [ ] §H
