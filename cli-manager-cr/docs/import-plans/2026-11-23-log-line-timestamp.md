# 세션 로그의 줄 앞 시각 (`sessionLogTimestamp`) — 배경·설계·코드·검증

**작성**: 2026-11-23 · **상태**: 코드·자동 테스트 완료(D112). 앱에서의 확인은 수동 검증으로 남음 · **대상**: cli-manager-cr(DariTerm)
**묶음**: 빠진 기능 10건 중 7번. 앞 문서 [agent-forwarding](./2026-11-23-agent-forwarding.md), 뒤 문서 [keyword-highlight](./2026-11-23-keyword-highlight.md). 로그 관련 앞선 문서: [2026-11-18-session-log-sync-interval.md](./2026-11-18-session-log-sync-interval.md)(D100), [2026-11-18-session-log-toggle-open-sessions.md](./2026-11-18-session-log-toggle-open-sessions.md)(D99).

---

## 0. 한 줄 요약

설정 > 연결에 **"로그 줄마다 시각 표시"** 토글을 더했다(기본 꺼짐). 켜면 세션 로그 파일의 줄마다 앞에 `[2026-11-23 09:05:07.123] ` 이 붙는다. 화면에는 보이지 않고 파일에만 남는다.

```
[2026-11-23 09:05:07.123] $ ./deploy.sh
[2026-11-23 09:05:07.480] building…
[2026-11-23 09:05:19.002] done
```

## 1. 배경

- 세션 로그([internal/session/log.go](../../internal/session/log.go))는 ANSI 시퀀스와 CR 을 뺀 글만 남겼다. 어떤 줄이 언제 나왔는지 알 수 없어, 장애 시각을 맞춰 보거나 명령이 얼마나 걸렸는지 볼 수 없었다.

## 2. 설계 결정

| 항목 | 정한 것 | 이유 |
|---|---|---|
| 형식 | `[YYYY-MM-DD HH:mm:ss.SSS] ` — 이 PC 의 현지 시각, 밀리초, 뒤에 공백 하나 | 사람이 읽기 쉽고 글자 순서로 정렬된다. 길이가 고정(26자)이라 잘라 내기 쉽다 |
| 어느 시각 | **그 줄의 첫 글자를 받은 때** | 줄이 끝난 때로 하면 오래 걸린 명령의 시작을 알 수 없다 |
| 줄이 여러 묶음에 걸쳐 올 때 | 한 번만 붙는다(`bLineStart` 로 줄 시작 위치를 따라간다) | 출력은 줄 중간에서 잘려 온다 |
| 빈 줄 | 붙인다 | 규칙이 단순하다(줄마다 하나). 빈 줄이 나온 시각도 정보다 |
| 중간에 켜고 끌 때 | 쓰던 줄 중간에는 넣지 않고 **다음 줄부터** | 줄 가운데에 시각이 끼면 읽기 어렵다. 꺼져 있을 때도 줄 시작 위치는 계속 따라간다 |
| 화면 | 붙이지 않는다 | 터미널 화면은 서버가 그리는 것이다. 커서 이동과 섞이면 깨진다 |
| 진행 막대(같은 줄을 고쳐 쓰는 출력) | 줄바꿈이 올 때까지 한 줄이므로 시각도 한 번 | 로그는 CR 을 지우므로 그 줄은 원래 한 줄로 길게 남는다(전부터 그랬다) |
| 열린 세션에 적용 | 설정을 바꾸면 곧바로(`store.patch` → `Manager.SetLogTimestamp`) | "터미널 출력 기록"(D99)·"갱신 주기"(D100)와 같은 방식 |
| 시각 얻기 | `Logger.fnNow`(기본 `time.Now`) | 테스트에서 시각을 고정한다 |

## A. 바뀐 파일 한눈에

| 파일 | 종류 | 상태 |
|---|---|---|
| `frontend/src/ipc/types.ts` | 코드 | 수정 |
| `frontend/src/lib/defaults.ts` | 코드 | 수정 |
| `frontend/src/screens/SettingsConnection.tsx` | 코드 | 수정 |
| `internal/app/app.go` | 코드 | 수정 |
| `internal/app/channels_session.go` | 코드 | 수정 |
| `internal/session/control.go` | 코드 | 수정 |
| `internal/session/log.go` | 코드 | 수정 |
| `internal/session/manager.go` | 코드 | 수정 |
| `internal/session/session.go` | 코드 | 수정 |
| `internal/session/types.go` | 코드 | 수정 |
| `internal/store/store.go` | 코드 | 수정 |
| `internal/session/log_test.go` | 테스트 | 수정 |
| `frontend/src/ipc/contract/store.get.afterNullPatch.json` | 계약 고정 파일 | 수정 |
| `frontend/src/ipc/contract/store.get.json` | 계약 고정 파일 | 수정 |
| `spec/behavior/session.md` | 명세 | 수정 |
| `spec/data/store.md` | 명세 | 수정 |
| `spec/ui/settings.md` | 명세 | 수정 |
| `locales/de.json` | 문구 | 수정 |
| `locales/en.json` | 문구 | 수정 |
| `locales/es.json` | 문구 | 수정 |
| `locales/fr.json` | 문구 | 수정 |
| `locales/ja.json` | 문구 | 수정 |
| `locales/ko.json` | 문구 | 수정 |
| `locales/pt.json` | 문구 | 수정 |
| `locales/ru.json` | 문구 | 수정 |
| `locales/tw.json` | 문구 | 수정 |
| `locales/zh.json` | 문구 | 수정 |

아래의 `diff` 는 **이 묶음의 앞 문서들을 차례로 적용한 상태**를 기준으로 뽑았다. 앞 문서에서도 고친 파일(문맥 줄에 앞 기능의 코드가 섞여 있을 수 있다): `frontend/src/ipc/types.ts`, `internal/app/channels_session.go`, `spec/data/store.md`. 이 기능만 따로 넣을 때는 `+`/`-` 줄만 옮기면 된다.

## B. 코드

### `frontend/src/ipc/types.ts` — 수정

```diff
--- a/frontend/src/ipc/types.ts
+++ b/frontend/src/ipc/types.ts
@@ -104,6 +104,8 @@
   sessionLogKeepDays: number;
   /** 세션 로그를 디스크까지 내려 쓰는 주기(초), 0~5. 0 이면 받는 즉시 쓰고 내려 쓴다 (behavior/session.md §6). */
   sessionLogSyncSec: number;
+  /** 세션 로그의 줄마다 앞에 시각([YYYY-MM-DD HH:mm:ss.SSS])을 붙인다 (behavior/session.md §6). */
+  sessionLogTimestamp: boolean;
   bell: boolean;
   shiftEnterNewline: boolean;
   rememberWindow: boolean;
```

### `frontend/src/lib/defaults.ts` — 수정

```diff
--- a/frontend/src/lib/defaults.ts
+++ b/frontend/src/lib/defaults.ts
@@ -42,6 +42,7 @@
   sessionLog: false,
   sessionLogKeepDays: 30,
   sessionLogSyncSec: 2,
+  sessionLogTimestamp: false,
   bell: false,
   shiftEnterNewline: true,
   rememberWindow: true,
```

### `frontend/src/screens/SettingsConnection.tsx` — 수정

```diff
--- a/frontend/src/screens/SettingsConnection.tsx
+++ b/frontend/src/screens/SettingsConnection.tsx
@@ -13,6 +13,7 @@
   const [sessionLog, setSessionLog] = useSetting('sessionLog');
   const [keepDays] = useSetting('sessionLogKeepDays');
   const [syncSec] = useSetting('sessionLogSyncSec');
+  const [logStamp, setLogStamp] = useSetting('sessionLogTimestamp');
   const pushToast = useApp((s) => s.pushToast);
 
   const openLogs = async () => {
@@ -44,6 +45,12 @@
           checked={sessionLog}
           onChange={setSessionLog}
         />
+        <ToggleRow
+          title={t('conn.logTimestamp')}
+          desc={t('conn.logTimestampDesc')}
+          checked={!!logStamp}
+          onChange={setLogStamp}
+        />
       </Group>
       <Field label={t('conn.logKeepDays')} help={t('conn.logKeepDaysHelp')}>
         <NumberField
```

### `internal/app/app.go` — 수정

```diff
--- a/internal/app/app.go
+++ b/internal/app/app.go
@@ -152,6 +152,7 @@
 		iKeepBefore := pService.logKeepDays()
 		bLogBefore := pService.settingBool("sessionLog", false)
 		iSyncBefore := pService.logSyncSec()
+		bStampBefore := pService.settingBool("sessionLogTimestamp", false)
 		errPatch := pService.pStore.Patch(mapPatch)
 		pService.applyLanguage()
 		if bLogNow := pService.settingBool("sessionLog", false); bLogNow != bLogBefore {
@@ -161,6 +162,10 @@
 		if iSyncNow := pService.logSyncSec(); iSyncNow != iSyncBefore {
 			// 내려 쓰기 주기도 열려 있는 세션에 곧바로 적용한다 (session.md §6, D100).
 			pService.pSessions.SetLogSyncSec(iSyncNow)
+		}
+		if bStampNow := pService.settingBool("sessionLogTimestamp", false); bStampNow != bStampBefore {
+			// 줄 앞 시각 표시도 열려 있는 세션에 곧바로(다음 줄부터) 적용한다 (session.md §6, D112).
+			pService.pSessions.SetLogTimestamp(bStampNow)
 		}
 		if pService.logKeepDays() != iKeepBefore {
 			pService.pLogClean.Kick() // 보관 기간을 바꾸면 곧바로 한 번 정리한다 (session.md §6)
```

### `internal/app/channels_session.go` — 수정

```diff
--- a/internal/app/channels_session.go
+++ b/internal/app/channels_session.go
@@ -197,6 +197,7 @@
 		LogEnabled:    pService.settingBool("sessionLog", false),
 		LogAlias:      record.M_szAlias,
 		LogSyncSec:    pService.logSyncSec(),
+		LogTimestamp:  pService.settingBool("sessionLogTimestamp", false),
 		OpenDelay:     0,
 	}
 	if bHasSaved {
```

### `internal/session/control.go` — 수정

```diff
--- a/internal/session/control.go
+++ b/internal/session/control.go
@@ -135,6 +135,7 @@
 	bLost := pSession.userClosed || pSession.logger != nil // 그사이 닫혔거나 다른 호출이 먼저 열었다
 	if !bLost {
 		pLogger.SetSyncSec(pSession.iLogSyncSec) // 빈 파일이라 여기서 기다릴 일은 없다
+		pLogger.SetTimestamp(pSession.bLogStamp)
 		pSession.logger = pLogger
 	}
 	pSession.mtxData.Unlock()
@@ -157,6 +158,18 @@
 	}
 }
 
+// SetLogTimestamp 는 이 세션 로그의 줄 앞 시각 표시를 켜거나 끈다. 기록 중이면 다음 줄부터 적용하고,
+// 나중에 기록을 켤 때도 이 값을 쓴다 (설정 sessionLogTimestamp, session.md §6, D112).
+func (pSession *Session) SetLogTimestamp(bEnabled bool) {
+	pSession.mtxData.Lock()
+	pSession.bLogStamp = bEnabled
+	pLogger := pSession.logger
+	pSession.mtxData.Unlock()
+	if pLogger != nil {
+		pLogger.SetTimestamp(bEnabled)
+	}
+}
+
 // AckOutput 은 프런트가 출력 묶음 iCount 개를 처리했음을 반영한다 (spec/ipc.md session.ack).
 func (pSession *Session) AckOutput(iCount int) {
 	if iCount <= 0 {
```

### `internal/session/log.go` — 수정

```diff
--- a/internal/session/log.go
+++ b/internal/session/log.go
@@ -26,6 +26,9 @@
 	G_ciLogSyncSecMin     = 0
 	G_ciLogSyncSecMax     = 5
 	G_ciLogSyncSecDefault = 2
+	// G_cszLogStampLayout 은 줄 앞에 붙이는 시각의 형식이다 (설정 sessionLogTimestamp, session.md §6, D112).
+	// 이 PC 의 현지 시각, 밀리초까지. 뒤에 공백 하나가 붙는다.
+	G_cszLogStampLayout = "[2006-01-02 15:04:05.000] "
 	// g_ciLogCarryMax 는 미완성 ANSI 시퀀스로 보관하는 최대 길이다. 넘으면 시퀀스가 아니라고 보고 그대로 기록한다.
 	g_ciLogCarryMax = 256
 	// g_ciLogAliasMaxRunes 는 로그 파일 이름에 쓰는 별칭의 최대 길이다 (경로 260자 한도 대비).
@@ -104,11 +107,14 @@
 	szPath     string // 열린 파일의 절대 경로 (열린 로그 집합의 키)
 	pFile      *os.File
 	pWriter    *bufio.Writer
-	szCarry    string      // 아직 끝나지 않은 ANSI 시퀀스의 앞부분
-	timerFlush *time.Timer // 버퍼에 내용이 있거나 아직 내려 쓰지 않은 내용이 있을 때만 걸린다
-	iSyncSec   int         // 내려 쓰기 주기(초). 0 = 즉시
-	bUnsynced  bool        // 파일에는 썼지만 아직 내려 쓰지 않은 내용이 있다
-	tLastSync  time.Time   // 마지막으로 내려 쓴 시각 (처음에는 연 시각)
+	szCarry    string           // 아직 끝나지 않은 ANSI 시퀀스의 앞부분
+	timerFlush *time.Timer      // 버퍼에 내용이 있거나 아직 내려 쓰지 않은 내용이 있을 때만 걸린다
+	iSyncSec   int              // 내려 쓰기 주기(초). 0 = 즉시
+	bUnsynced  bool             // 파일에는 썼지만 아직 내려 쓰지 않은 내용이 있다
+	tLastSync  time.Time        // 마지막으로 내려 쓴 시각 (처음에는 연 시각)
+	bTimestamp bool             // 줄마다 앞에 시각을 붙인다 (설정 sessionLogTimestamp)
+	bLineStart bool             // 다음에 쓸 글자가 줄의 첫 글자다 (시각을 붙일 자리)
+	fnNow      func() time.Time // 시각을 얻는 함수 (시험에서 바꾼다)
 }
 
 // OpenLog 는 데이터 폴더의 logs 아래 이 세션만의 로그 파일을 새로 만든다 (소유자 전용 권한, session.md §6).
@@ -131,7 +137,7 @@
 			g_mOpenLogs[szPath] = struct{}{}
 			g_mtxOpenLogs.Unlock()
 			return &Logger{szPath: szPath, pFile: pFile, pWriter: bufio.NewWriterSize(pFile, G_ciLogBufferBytes),
-				iSyncSec: G_ciLogSyncSecDefault, tLastSync: time.Now()}, nil
+				iSyncSec: G_ciLogSyncSecDefault, tLastSync: time.Now(), bLineStart: true, fnNow: time.Now}, nil
 		}
 		if !errors.Is(errOpen, fs.ErrExist) {
 			return nil, errOpen
@@ -185,7 +191,7 @@
 	if szBody == "" {
 		return
 	}
-	_, _ = pLogger.pWriter.WriteString(StripForLog(szBody))
+	_, _ = pLogger.pWriter.WriteString(pLogger.stampLinesLocked(StripForLog(szBody)))
 	if pLogger.iSyncSec == 0 {
 		// 즉시 모드: 부하가 걸리더라도 받는 대로 파일에 쓰고 내려 쓴다 (사용자가 고른 동작).
 		pLogger.syncNowLocked()
@@ -194,6 +200,44 @@
 	if pLogger.timerFlush == nil {
 		pLogger.timerFlush = time.AfterFunc(G_cdurLogFlush, pLogger.flushTimer)
 	}
+}
+
+// stampLinesLocked 는 mtxData 를 잡은 상태에서, 정리된 로그 글에 줄 시작 위치를 따라가며
+// (켜져 있으면) 줄마다 앞에 시각을 붙인다. 시각은 그 줄의 첫 글자를 받은 때다.
+// 꺼져 있어도 줄 시작 위치는 따라가므로, 중간에 켜면 다음 줄부터 시각이 붙는다.
+func (pLogger *Logger) stampLinesLocked(szText string) string {
+	if szText == "" {
+		return ""
+	}
+	if !pLogger.bTimestamp {
+		pLogger.bLineStart = strings.HasSuffix(szText, "\n")
+		return szText
+	}
+	szStamp := pLogger.fnNow().Format(G_cszLogStampLayout)
+	var sbOut strings.Builder
+	sbOut.Grow(len(szText) + len(szStamp))
+	for len(szText) > 0 {
+		if pLogger.bLineStart {
+			sbOut.WriteString(szStamp)
+			pLogger.bLineStart = false
+		}
+		iNewline := strings.IndexByte(szText, '\n')
+		if iNewline < 0 {
+			sbOut.WriteString(szText)
+			break
+		}
+		sbOut.WriteString(szText[:iNewline+1])
+		szText = szText[iNewline+1:]
+		pLogger.bLineStart = true
+	}
+	return sbOut.String()
+}
+
+// SetTimestamp 는 줄마다 시각을 붙일지 바꾼다. 쓰던 줄 중간에는 붙이지 않고 다음 줄부터 적용한다.
+func (pLogger *Logger) SetTimestamp(bEnabled bool) {
+	pLogger.mtxData.Lock()
+	defer pLogger.mtxData.Unlock()
+	pLogger.bTimestamp = bEnabled
 }
 
 // syncNowLocked 는 mtxData 를 잡은 상태에서 버퍼를 파일에 쓰고 곧바로 내려 쓴다.
@@ -260,7 +304,7 @@
 		pLogger.timerFlush = nil
 	}
 	if pLogger.szCarry != "" {
-		_, _ = pLogger.pWriter.WriteString(StripForLog(pLogger.szCarry))
+		_, _ = pLogger.pWriter.WriteString(pLogger.stampLinesLocked(StripForLog(pLogger.szCarry)))
 		pLogger.szCarry = ""
 	}
 	errFlush := pLogger.pWriter.Flush()
```

### `internal/session/manager.go` — 수정

```diff
--- a/internal/session/manager.go
+++ b/internal/session/manager.go
@@ -116,6 +116,19 @@
 	}
 }
 
+// SetLogTimestamp 는 열려 있는 모든 세션의 로그 줄 앞 시각 표시를 바꾼다 (설정 sessionLogTimestamp 를 바꿨을 때).
+func (pManager *Manager) SetLogTimestamp(bEnabled bool) {
+	pManager.mtxData.Lock()
+	slSessions := make([]*Session, 0, len(pManager.mapSessions))
+	for _, pSession := range pManager.mapSessions {
+		slSessions = append(slSessions, pSession)
+	}
+	pManager.mtxData.Unlock()
+	for _, pSession := range slSessions {
+		pSession.SetLogTimestamp(bEnabled)
+	}
+}
+
 // Count 는 열린 세션 수를 돌려준다 (종료 확인 문구에 쓴다).
 func (pManager *Manager) Count() int {
 	pManager.mtxData.Lock()
```

### `internal/session/session.go` — 수정

```diff
--- a/internal/session/session.go
+++ b/internal/session/session.go
@@ -39,6 +39,7 @@
 	tConnected   time.Time // 지금 연결이 맺어진 시각 (서버 종료가 사용자 종료인지 가릴 때 쓴다)
 	tLastInput   time.Time // 지금 연결에 사용자 입력을 마지막으로 보낸 시각 (없으면 zero)
 	iLogSyncSec  int       // 세션 로그 내려 쓰기 주기(초). 설정을 바꾸면 SetLogSyncSec 으로 갱신된다
+	bLogStamp    bool      // 세션 로그 줄 앞에 시각을 붙일지. 설정을 바꾸면 SetLogTimestamp 로 갱신된다
 }
 
 // G_cdurUserExitWindow 는 서버가 셸을 끝냈을 때 "사용자가 끝낸 것"으로 보는 시간 창이다 (session.md §4).
@@ -55,6 +56,7 @@
 		decoder: NewDecoder(cfg.Encoding),
 
 		iLogSyncSec: ClampLogSyncSec(cfg.LogSyncSec),
+		bLogStamp:   cfg.LogTimestamp,
 	}
 	pSession.info.M_szState = G_cszStateConnecting
 	szId := cfg.Info.M_szId
```

### `internal/session/types.go` — 수정

```diff
--- a/internal/session/types.go
+++ b/internal/session/types.go
@@ -54,6 +54,8 @@
 	LogAlias      string
 	// LogSyncSec 은 세션 로그의 내려 쓰기 주기(초, 0~5)다. 0 이면 받는 즉시 쓰고 내려 쓴다 (session.md §6).
 	LogSyncSec int
+	// LogTimestamp 가 true 면 세션 로그의 줄마다 앞에 시각을 붙인다 (session.md §6, D112).
+	LogTimestamp bool
 	// StartupInput 은 접속 직후 셸에 보낼 입력이다 (tmux 한 줄 또는 자동 실행 명령). 비면 보내지 않는다.
 	StartupInput string
 	// StartupDelay 는 접속 후 StartupInput 을 보내기까지의 대기다 (tmux 500ms, 자동 실행 명령 400ms).
```

### `internal/store/store.go` — 수정

```diff
--- a/internal/store/store.go
+++ b/internal/store/store.go
@@ -41,26 +41,27 @@
 		`\bTRUNCATE\b`,
 		`systemctl\s+(stop|restart)`,
 	},
-	"autoReconnect":      true,
-	"autoLockMinutes":    15,
-	"uploadDir":          "~/.dariterm/uploads",
-	"uploadCleanupDays":  7,
-	"transferTempDir":    "",
-	"maxVerticalSplits":  3,
-	"insertPathQuote":    true,
-	"sessionLog":         false,
-	"sessionLogKeepDays": 30,
-	"sessionLogSyncSec":  2,
-	"bell":               false,
-	"shiftEnterNewline":  true,
-	"rememberWindow":     true,
-	"restoreLayout":      true,
-	"updateCheck":        true,
-	"syncRepo":           "",
-	"syncAuto":           true,
-	"osc52Copy":          true,
-	"sidebarWidth":       260,
-	"rightPanelWidth":    340,
+	"autoReconnect":       true,
+	"autoLockMinutes":     15,
+	"uploadDir":           "~/.dariterm/uploads",
+	"uploadCleanupDays":   7,
+	"transferTempDir":     "",
+	"maxVerticalSplits":   3,
+	"insertPathQuote":     true,
+	"sessionLog":          false,
+	"sessionLogKeepDays":  30,
+	"sessionLogSyncSec":   2,
+	"sessionLogTimestamp": false,
+	"bell":                false,
+	"shiftEnterNewline":   true,
+	"rememberWindow":      true,
+	"restoreLayout":       true,
+	"updateCheck":         true,
+	"syncRepo":            "",
+	"syncAuto":            true,
+	"osc52Copy":           true,
+	"sidebarWidth":        260,
+	"rightPanelWidth":     340,
 	// D78: 사용 용도·업무용 라이선스. usage ''=아직 안 고름 / personal / business(30일 평가), usageSince=업무용 고른 시각(ms).
 	"usage":            "",
 	"usageSince":       0,
```


## C. 테스트 코드

### `internal/session/log_test.go` — 수정

```diff
--- a/internal/session/log_test.go
+++ b/internal/session/log_test.go
@@ -369,3 +369,93 @@
 	return pLogger.iSyncSec
 }
 
+// 줄 앞 시각(D112): 켜면 줄마다 한 번, 그 줄의 첫 글자를 받은 때의 시각이 붙는다. 줄이 여러 조각에 걸쳐 와도 한 번이다.
+func TestLogTimestampPerLine(t *testing.T) {
+	pLogger, errOpen := OpenLog(t.TempDir(), "x", time.Now())
+	if errOpen != nil {
+		t.Fatal(errOpen)
+	}
+	defer pLogger.Close()
+	tClock := time.Date(2026, 11, 23, 9, 5, 7, 123_000_000, time.Local)
+	pLogger.fnNow = func() time.Time { return tClock }
+	pLogger.SetSyncSec(0)
+	pLogger.SetTimestamp(true)
+	pLogger.Write("first li")
+	tClock = tClock.Add(2 * time.Second)
+	pLogger.Write("ne\r\n\r\n\x1b[31msecond\x1b[0m\r\nthi")
+	tClock = tClock.Add(time.Second)
+	pLogger.Write("rd")
+	bData, _ := os.ReadFile(pLogger.Path())
+	szWant := "[2026-11-23 09:05:07.123] first line\n" +
+		"[2026-11-23 09:05:09.123] \n" +
+		"[2026-11-23 09:05:09.123] second\n" +
+		"[2026-11-23 09:05:09.123] third"
+	if string(bData) != szWant {
+		t.Fatalf("줄 앞 시각:\n%q\n기대:\n%q", string(bData), szWant)
+	}
+}
+
+// 기록 중에 켜고 끄면 쓰던 줄 중간에는 넣지 않고 다음 줄부터 적용한다. 닫을 때 남은 꼬리에도 같은 규칙이다.
+func TestLogTimestampToggleMidLine(t *testing.T) {
+	pLogger, errOpen := OpenLog(t.TempDir(), "x", time.Now())
+	if errOpen != nil {
+		t.Fatal(errOpen)
+	}
+	tClock := time.Date(2026, 1, 2, 3, 4, 5, 0, time.Local)
+	pLogger.fnNow = func() time.Time { return tClock }
+	pLogger.SetSyncSec(0)
+	pLogger.Write("plain\nhalf")
+	pLogger.SetTimestamp(true)
+	pLogger.Write(" line\nstamped\n")
+	pLogger.SetTimestamp(false)
+	pLogger.Write("off again\n")
+	pLogger.SetTimestamp(true)
+	pLogger.Write("tail\x1b[") // 미완성 시퀀스는 닫을 때 기록된다
+	szPath := pLogger.Path()
+	_ = pLogger.Close()
+	bData, _ := os.ReadFile(szPath)
+	szStamp := "[2026-01-02 03:04:05.000] "
+	szWant := "plain\nhalf line\n" + szStamp + "stamped\noff again\n" + szStamp + "tail\x1b["
+	if string(bData) != szWant {
+		t.Fatalf("중간에 켜고 끄기:\n%q\n기대:\n%q", string(bData), szWant)
+	}
+}
+
+// 세션에 준 설정은 기록을 켤 때 로그에 전해지고, 바꾸면 기록 중인 로그에도 적용된다 (D112).
+func TestSessionLogTimestampApplies(t *testing.T) {
+	szDir := t.TempDir()
+	pFake := newFakeTransport()
+	rec := &eventRecorder{}
+	pManager := NewManager(rec.emit)
+	cfg := sshConfig(func(func(string)) (Transport, error) { return pFake, nil }, false)
+	cfg.DataDir = szDir
+	cfg.LogAlias = "web"
+	cfg.LogEnabled = true
+	cfg.LogTimestamp = true
+	info, _ := pManager.Open(cfg)
+	defer pManager.Close(info.M_szId)
+	pSession, _ := pManager.Get(info.M_szId)
+	waitUntil(t, func() bool { return pSession.Snapshot().M_szState == G_cszStateConnected }, time.Second)
+	fnStamp := func() bool {
+		pSession.mtxData.Lock()
+		pLogger := pSession.logger
+		pSession.mtxData.Unlock()
+		pLogger.mtxData.Lock()
+		defer pLogger.mtxData.Unlock()
+		return pLogger.bTimestamp
+	}
+	if !fnStamp() {
+		t.Fatal("열 때 준 설정이 로그에 전해져야 함")
+	}
+	pManager.SetLogTimestamp(false)
+	if fnStamp() {
+		t.Fatal("바꾼 설정이 기록 중인 로그에 적용돼야 함")
+	}
+	pManager.SetLogging(false)
+	pManager.SetLogTimestamp(true)
+	pManager.SetLogging(true)
+	if !fnStamp() {
+		t.Fatal("꺼져 있을 때 바꾼 설정도 다시 켤 때 써야 함")
+	}
+}
+
```


## D. 계약 고정 파일 (백엔드 응답과 프런트 타입을 맞춰 보는 테스트가 읽는다)

### `frontend/src/ipc/contract/store.get.afterNullPatch.json` — 수정

```diff
--- a/frontend/src/ipc/contract/store.get.afterNullPatch.json
+++ b/frontend/src/ipc/contract/store.get.afterNullPatch.json
@@ -42,6 +42,7 @@
     "sessionLog": false,
     "sessionLogKeepDays": 30,
     "sessionLogSyncSec": 2,
+    "sessionLogTimestamp": false,
     "shiftEnterNewline": true,
     "sidebarWidth": 260,
     "syncAuto": true,
```

### `frontend/src/ipc/contract/store.get.json` — 수정

```diff
--- a/frontend/src/ipc/contract/store.get.json
+++ b/frontend/src/ipc/contract/store.get.json
@@ -42,6 +42,7 @@
     "sessionLog": false,
     "sessionLogKeepDays": 30,
     "sessionLogSyncSec": 2,
+    "sessionLogTimestamp": false,
     "shiftEnterNewline": true,
     "sidebarWidth": 260,
     "syncAuto": true,
```


## E. 문구

`locales/<언어>.json` 10개 파일에 같은 키를 넣는다(순서: ko, en, ja, zh, tw, de, fr, es, pt, ru). 한국어·영어 외에는 이 작업을 한 세션이 번역한 것이라 검수가 필요할 수 있다.

**`conn.logTimestamp`** (추가)

```json
  "conn.logTimestamp": "로그 줄마다 시각 표시",
  "conn.logTimestamp": "Timestamp each log line",
  "conn.logTimestamp": "ログの各行に時刻を付ける",
  "conn.logTimestamp": "为日志每行加时间戳",
  "conn.logTimestamp": "為記錄每行加上時間戳記",
  "conn.logTimestamp": "Zeitstempel für jede Protokollzeile",
  "conn.logTimestamp": "Horodater chaque ligne du journal",
  "conn.logTimestamp": "Marca de tiempo en cada línea del registro",
  "conn.logTimestamp": "Carimbo de data e hora em cada linha do log",
  "conn.logTimestamp": "Метка времени в каждой строке журнала",
```

**`conn.logTimestampDesc`** (추가)

```json
  "conn.logTimestampDesc": "기록하는 로그의 줄 앞에 [날짜 시각.밀리초]를 붙입니다. 화면에는 보이지 않고 파일에만 남습니다 (바꾸면 다음 줄부터 적용)",
  "conn.logTimestampDesc": "Prefixes each recorded log line with [date time.milliseconds]. It is written to the file only, not shown on screen (changes apply from the next line)",
  "conn.logTimestampDesc": "記録するログの各行の先頭に [日付 時刻.ミリ秒] を付けます。画面には表示されず、ファイルにのみ残ります（変更は次の行から適用）",
  "conn.logTimestampDesc": "在记录的日志每行前加上 [日期 时间.毫秒]。只写入文件，不显示在屏幕上（更改从下一行起生效）",
  "conn.logTimestampDesc": "在記錄的每行前加上 [日期 時間.毫秒]。只寫入檔案，不顯示在畫面上（變更從下一行起生效）",
  "conn.logTimestampDesc": "Stellt jeder aufgezeichneten Protokollzeile [Datum Uhrzeit.Millisekunden] voran. Nur in der Datei, nicht auf dem Bildschirm (Änderungen gelten ab der nächsten Zeile)",
  "conn.logTimestampDesc": "Ajoute [date heure.millisecondes] au début de chaque ligne enregistrée. Écrit dans le fichier uniquement, pas à l'écran (les changements s'appliquent à partir de la ligne suivante)",
  "conn.logTimestampDesc": "Antepone [fecha hora.milisegundos] a cada línea registrada. Solo se escribe en el archivo, no se muestra en pantalla (los cambios se aplican desde la línea siguiente)",
  "conn.logTimestampDesc": "Adiciona [data hora.milissegundos] no início de cada linha registrada. Gravado apenas no arquivo, não aparece na tela (as alterações valem a partir da próxima linha)",
  "conn.logTimestampDesc": "Добавляет [дата время.миллисекунды] в начало каждой записываемой строки. Только в файле, на экране не показывается (изменение действует со следующей строки)",
```


## F. 명세

### `spec/behavior/session.md` — 수정

```diff
--- a/spec/behavior/session.md
+++ b/spec/behavior/session.md
@@ -92,6 +92,7 @@
   - 값이 1~5 이면: 파일 쓰기는 지금처럼 약 1초 단위, 내려 쓰기는 **그 초마다 한 번**(그동안 새로 쓴 내용이 있을 때만). 출력이 멈춘 뒤에도 남은 내용은 주기가 되면 내려 쓴다.
   - 값이 0 이면: 출력을 받는 **즉시** 파일에 쓰고 내려 쓴다(모으지 않는다). 출력이 많은 세션에서는 디스크 부하와 출력 지연이 커질 수 있다.
   - 설정을 바꾸면 기록 중인 세션에도 곧바로 적용한다.
+- **줄 앞 시각**(2026-11-23, D112): 설정 `sessionLogTimestamp`(기본 꺼짐). 켜면 로그의 **줄마다 앞에** `[YYYY-MM-DD HH:mm:ss.SSS] `(이 PC 의 현지 시각, 밀리초까지, 뒤에 공백 하나)를 붙인다. 시각은 **그 줄의 첫 글자를 받은 때**다(줄이 여러 묶음에 걸쳐 와도 한 번만 붙는다). 빈 줄에도 붙는다. 화면 출력에는 붙이지 않는다(파일에만). 설정을 바꾸면 기록 중인 세션에도 곧바로 적용하되, 쓰던 줄 중간에는 넣지 않고 **다음 줄부터** 적용한다. 커서 이동으로 같은 줄을 고쳐 쓰는 출력(진행 막대 등)은 줄바꿈이 올 때까지 한 줄이므로 시각도 한 번이다.
 - **보관 기간**: 설정 `sessionLogKeepDays`(기본 30, 0 이면 지우지 않음). 앱은 시작 약 10초 뒤와 그 뒤 **1시간마다** 로그 폴더 바로 아래의 `.log` 파일 중 수정 시각이 그 일수보다 오래된 것을 지운다. 지금 열려 있는(기록 중인) 세션의 파일과 하위 폴더·다른 확장자 파일은 건드리지 않는다. 설정 값을 바꾸면 곧바로 한 번 정리한다. 지우지 못한 파일(다른 프로그램이 열고 있음 등)은 건너뛰고 다음 주기에 다시 본다. "터미널 출력 기록"이 꺼져 있어도 남은 파일은 정리한다.
 
 ## 7. 세션 유지 (tmux)
```

### `spec/data/store.md` — 수정

```diff
--- a/spec/data/store.md
+++ b/spec/data/store.md
@@ -97,6 +97,7 @@
 | insertPathQuote | true | 경로 삽입 시 따옴표 |
 | sessionLog | false | |
 | sessionLogKeepDays | 30 | 세션 로그 보관 일수. 0 이면 지우지 않음 ([../behavior/session.md](../behavior/session.md) §6) |
+| sessionLogTimestamp | false | 세션 로그의 줄마다 앞에 시각을 붙인다 ([../behavior/session.md](../behavior/session.md) §6, D112) |
 | sessionLogSyncSec | 2 | 기록 중인 세션 로그를 디스크까지 내려 쓰는 주기(초), 0~5. 0 이면 출력을 받는 즉시 쓰고 내려 쓴다 ([../behavior/session.md](../behavior/session.md) §6, 2026-11-18) |
 | bell | false | |
 | shiftEnterNewline | true | Shift+Enter 로 줄바꿈 입력 |
```

### `spec/ui/settings.md` — 수정

```diff
--- a/spec/ui/settings.md
+++ b/spec/ui/settings.md
@@ -51,6 +51,7 @@
   1. **끊기면 자동 재접속** — "연결이 끊기면 2·4·8·16·30초로 간격을 늘리며 최대 5번 재연결합니다"
   2. **탭·분할 레이아웃 복원** — "종료할 때 열린 탭과 창 배치를 저장하고, 다음 실행 때 같은 서버로 다시 연결합니다. 비밀번호·화면 내용은 저장하지 않으며, 목록에 없는 일회성 SSH 연결은 복원하지 않습니다"
   3. **세션 로그 저장** — "터미널 출력을 텍스트 파일로 남깁니다 (켜면 열려 있는 연결도 그때부터 기록)"
+  4. **로그 줄마다 시각 표시**(2026-11-23, D112) — "기록하는 로그의 줄 앞에 [날짜 시각.밀리초]를 붙입니다. 화면에는 보이지 않고 파일에만 남습니다 (바꾸면 다음 줄부터 적용)"
 - 필드 **로그 보관 기간 (일)**(숫자, 0~36500, 비면 30, 도움말 "이 일수보다 오래된 세션 로그 파일을 자동으로 지웁니다. 0이면 지우지 않습니다.") — 값은 `sessionLogKeepDays`(behavior/session.md §6 보관 기간).
 - 필드 **로그 파일 갱신 주기 (초)**(숫자, 0~5, 비면 2, 도움말 "기록 중인 로그를 디스크까지 내려 쓰는 주기입니다(0~5). 내려 써야 탐색기에 파일 크기와 수정 시각이 보입니다. 0이면 출력을 받는 즉시 쓰고 내려 써서 디스크 부하가 커집니다. 0이 아니면 파일에는 1초마다 씁니다.") — 값은 `sessionLogSyncSec`(behavior/session.md §6 내려 쓰기 주기). (2026-11-18)
 - 버튼 [로그 폴더 열기](폴더 아이콘) → 로그 폴더를 탐색기로 연다.
```



---

## G. 검증 (2026-11-23 실행)

| 테스트 ([internal/session/log_test.go](../../internal/session/log_test.go)) | 확인하는 것 |
|---|---|
| `TestLogTimestampPerLine` | 줄마다 한 번 붙는다 / 줄이 세 묶음에 걸쳐 와도 **첫 글자를 받은 때**의 시각 하나 / 빈 줄에도 붙는다 / ANSI 색 코드를 지운 뒤의 줄 기준이다 |
| `TestLogTimestampToggleMidLine` | 쓰던 줄 중간에 켜면 다음 줄부터 / 끄면 다음 줄부터 안 붙는다 / 닫을 때 남은 꼬리(미완성 시퀀스)에도 같은 규칙 |
| `TestSessionLogTimestampApplies` | 세션을 열 때 준 설정이 로그에 전해진다 / 바꾸면 기록 중인 로그에 적용 / 기록이 꺼져 있을 때 바꾼 값도 다시 켤 때 쓴다 |
| 기존 로그 테스트 전부 | 그대로 통과(꺼져 있으면 파일 내용이 전과 같다) |
| 계약 테스트(`TestContractChannels`) | `store.get` 의 `sessionLogTimestamp: false` 가 프런트 고정 파일과 같다 |

```bash
go test ./internal/session/ ./internal/app/ ./internal/store/    # 통과
(cd frontend && npx tsc --noEmit && npx vitest run)                # 통과
```

`store.patch` 에서 값이 바뀌었을 때 세션까지 이어지는 경로(app.go 의 4줄)를 직접 확인하는 자동 테스트는 없다(D99·D100 과 같은 상태).

## H. 수동 검증 (남은 일)

| # | 절차 | 기대 결과 |
|---|---|---|
| 1 | 설정 > 연결에서 "터미널 출력 기록"과 "로그 줄마다 시각 표시"를 켜고 세션에서 `date; sleep 3; date` | 로그의 두 `date` 줄 시각이 약 3초 차이 난다 |
| 2 | 화면 | 화면에는 시각이 보이지 않는다 |
| 3 | 세션을 연 채로 토글을 끈다 → 명령 몇 개 → 다시 켠다 | 끈 동안의 줄에는 시각이 없고, 켠 뒤의 줄부터 다시 붙는다. 줄 가운데에 끼어든 시각이 없다 |
| 4 | `ping -c 3 localhost` 처럼 천천히 나오는 출력 | 줄마다 1초씩 차이 나는 시각 |
| 5 | 토글이 꺼진 기존 사용자 | 로그 내용이 전과 같다 |

## I. 영향 범위·회귀 위험

| 영역 | 위험 | 대비 |
|---|---|---|
| 꺼져 있을 때(기본) | `stampLinesLocked` 가 글을 그대로 돌려준다(줄 시작 위치만 기억) | 기존 로그 테스트 |
| 로그 크기 | 줄마다 26바이트 늘어난다 | 짧은 줄이 많은 로그는 눈에 띄게 커진다 |
| 로그를 읽는 다른 도구 | 줄 앞에 글자가 생긴다 | 켠 사람만 해당. 앞 26자를 잘라 내면 원래 줄 |
| 성능 | 켜져 있으면 출력 묶음마다 시각 문자열 하나를 만들고 줄 수만큼 복사한다 | 재 보지 않았다. 로그 쓰기는 버퍼를 거친다 |
| 동기화·백업 가져오기로 설정이 바뀜 | `store.patch` 를 거치지 않아 기록 중인 세션에는 적용되지 않는다(새로 기록을 시작하는 세션부터) | D99·D100 과 같은 한계 |

## J. 체크리스트

- [x] `log.go`: 형식 상수, 필드 3개, `stampLinesLocked`, `SetTimestamp`, `Write`·`Close` 수정
- [x] `types.go`·`session.go`·`control.go`·`manager.go`: 전달과 적용
- [x] `app.go`·`channels_session.go`·`store.go`: 설정 읽기·바뀌면 적용·기본값
- [x] 프런트: 타입·기본값·계약 파일 2개·설정 화면 토글
- [x] 문구 2개 추가(10개 언어), 명세 behavior/session.md §6·data/store.md §4·ui/settings.md §4, decisions D112
- [x] 자동 테스트 3개 통과
- [ ] 수동 검증 1~5
