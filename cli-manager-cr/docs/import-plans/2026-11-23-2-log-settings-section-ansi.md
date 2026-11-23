# 설정에 "로그" 구역 만들기 + 세션 로그에 ANSI 그대로 저장 (`sessionLogAnsi`) — 배경·설계·코드·검증

**작성**: 2026-11-23 · **상태**: 코드·자동 테스트·빌드 완료(D120). 앱 화면에서의 확인은 수동 검증으로 남음(§H) · **대상**: cli-manager-cr(DariTerm) · **Known issue**: 없음
**관련**: 로그의 앞선 문서 [2026-11-23-log-line-timestamp.md](./2026-11-23-log-line-timestamp.md)(D112, 줄 앞 시각), [2026-11-18-session-log-sync-interval.md](./2026-11-18-session-log-sync-interval.md)(D100), [2026-11-18-session-log-toggle-open-sessions.md](./2026-11-18-session-log-toggle-open-sessions.md)(D99). 명세 [spec/ui/settings.md](../../spec/ui/settings.md) §4·§4a, [spec/behavior/session.md](../../spec/behavior/session.md) §6, [spec/data/store.md](../../spec/data/store.md) §4

> 이 문서 하나로 **무엇을 바꿨는지 → 왜 그렇게 정했는지 → 코드 전체 → 어떻게 검증하는지**를 알 수 있게 쓴다.

---

## 0. 한 줄 요약

두 가지를 했다.

1. **설정 화면에 "로그" 구역을 새로 두고**, "연결"에 있던 로그 항목을 모두 그리로 옮겼다. 구역은 8개에서 9개가 됐다.
2. 로그 구역에 **"ANSI 그대로 저장"** 토글을 더했다(**기본 꺼짐**). 켜면 세션 로그에 색·커서 이동 같은 ANSI 제어 코드를 지우지 않고 받은 그대로 남긴다.

| 구역 | 바꾸기 전 | 바꾼 뒤 |
|---|---|---|
| 연결 | 자동 재연결 · 배치 복원 · 터미널 출력 기록 · 로그 줄마다 시각 표시 · 로그 보관 기간 · 로그 파일 갱신 주기 · [로그 폴더 열기] | 자동 재연결 · 배치 복원 |
| **로그** (새로, 연결 바로 다음) | — | 터미널 출력 기록 · 로그 줄마다 시각 표시 · **ANSI 그대로 저장**(새로) · 로그 보관 기간 · 로그 파일 갱신 주기 · [로그 폴더 열기] |

같은 출력 `ls --color` 한 줄이 로그 파일에 남는 모습(`␛` 는 ESC 문자 0x1B, `␍` 는 CR 0x0D):

```
꺼짐(기본):  docs  main.go␊
켜짐:        ␛[0m␛[01;34mdocs␛[0m  main.go␍␊
```

## 1. 사용자 요청

2026-11-23: "설정에서 연결 메뉴의 항목 중에서 로그 관련 항목들을 로그 메뉴로 독립하도록 하며, ansi 로 저장할지도 옵션으로 설정해서 ansi on 이면 ansi 로 저장해줘. default 는 off 로 한다."

바로 앞 질문("출력이 ANSI 로 나오면 text 로만 저장되나?")의 답: 지금까지는 **항상** ANSI 시퀀스와 CR 을 지운 글자만 저장했다([internal/session/log.go](../../internal/session/log.go) `StripForLog`). 원본을 남길 방법이 없었다.

### 1.1 용어

| 말 | 뜻 |
|---|---|
| ANSI 시퀀스 | 서버가 화면을 꾸미려고 보내는 제어 코드. ESC(0x1B)로 시작한다. 예: `ESC[31m`(빨강), `ESC[2J`(화면 지우기), `ESC]0;제목 BEL`(창 제목) |
| CR | 줄 맨 앞으로 돌아가는 글자(0x0D). 서버 출력의 줄 끝은 보통 CR+LF 이고, 진행 막대는 CR 만 보내 같은 줄을 덮어 쓴다 |
| 출력 묶음 | 서버에서 한 번에 받은 조각. 시퀀스 가운데에서 잘려 올 수 있다 |
| 꼬리 보관 | 묶음 끝이 아직 끝나지 않은 시퀀스면 떼어 두었다가 다음 묶음 앞에 붙이는 일(최대 256바이트, D88) |

## 2. 설계 결정

### 2.1 "로그" 구역

| 항목 | 정한 것 | 이유 |
|---|---|---|
| 구역 이름·자리 | "로그", **연결 바로 다음**(5번째) | 원래 연결에 있던 것이라 찾던 자리 근처에 둔다 |
| 아이콘 | 두루마리 문서(lucide `ScrollText`) | 기록·로그를 뜻하는 흔한 모양. 다른 구역과 겹치지 않는다 |
| 옮긴 것 | 로그 항목 5개 전부(토글 2 + 숫자 2 + 버튼 1) | 요청 그대로. 연결에는 자동 재연결·배치 복원만 남는다 |
| 설정 키 | **그대로**(`sessionLog`·`sessionLogKeepDays`·`sessionLogSyncSec`·`sessionLogTimestamp`) | 화면에서 보이는 자리만 옮긴 것이다. 저장 파일·동기화·백업과 호환이 깨지지 않는다 |
| 문구 키 | **그대로**(`conn.sessionLog`·`conn.logKeepDays` 등). 새 키도 같은 묶음(`conn.logAnsi`)에 둔다. 구역 이름만 `settings.sec.log` | 키 이름을 `log.*` 로 바꾸면 10개 언어 파일에서 9개 키를 옮겨야 하는데 화면에 보이는 것은 달라지지 않는다 |
| 내부 구역 id | `log` (`SettingsSection` 에 추가) | — |
| 명세의 절 번호 | `spec/ui/settings.md` 에 **§4a** 로 넣고 §5~§8 은 그대로 | 코드 주석과 다른 문서가 §6·§7·§8 을 가리킨다 |

### 2.2 "ANSI 그대로 저장"

| 항목 | 정한 것 | 이유 |
|---|---|---|
| 설정 키·기본값 | `sessionLogAnsi`, **false** | 요청 그대로. 기존 사용자의 로그는 달라지지 않는다 |
| 켰을 때 남기는 것 | ANSI 시퀀스 **와 CR** 을 지우지 않는다 = 받은 그대로(글자 인코딩만 지금처럼 UTF-8) | 켜는 까닭은 터미널에서 `cat`·`less -R` 로 다시 볼 때 화면처럼 보이게 하려는 것이다. CR 을 지우면 진행 막대처럼 같은 줄을 덮어 쓰는 출력이 다시 볼 때 한 줄로 길게 늘어진다 |
| 파일 이름·확장자 | 그대로 `{별칭}-{시각}.log` | 오래된 로그 정리(`internal/logclean`)가 `.log` 만 본다. 확장자를 바꾸면 정리에서 빠진다 |
| 쓰는 주기·보관 기간 | 그대로 | 내용만 다르다 |
| 미완성 시퀀스 꼬리 보관 | **켜져 있어도 한다** | 꺼져 있을 때만 하면, 시퀀스 앞 절반을 그대로 쓴 직후에 설정을 끄면 뒤 절반(`1m` 같은 조각)이 글자로 남는다. 늘 시퀀스 단위로 끊어 두면 켜고 끄는 순간에도 시퀀스 하나는 통째로 남거나 통째로 지워진다 |
| 열린 세션에 적용 | 바꾸면 곧바로, **다음 출력부터**(`store.patch` → `Manager.SetLogAnsi`) | "터미널 출력 기록"(D99)·"갱신 주기"(D100)·"줄 앞 시각"(D112)과 같은 방식 |
| 바꿀 때 파일 | 같은 파일에 이어 쓴다(바꾼 자리 앞뒤로 두 형식이 섞인다) | 설정 하나 바꿨다고 파일이 갈리면 한 세션의 기록이 흩어진다. 새 파일을 원하면 "터미널 출력 기록"을 껐다 켜면 된다 |
| 줄 앞 시각과 함께 | 둘 다 된다. 시각은 지금처럼 **LF 뒤의 첫 글자 앞**에 붙는다 | 줄 시작 판단(`bLineStart`)을 건드리지 않는다. 줄이 색 코드로 시작하면 시각이 그 앞에 온다 |
| 화면 | 달라지지 않는다 | 로그 파일에만 해당한다 |

### 2.3 바꾼 뒤의 흐름

```
서버 출력 ─ 디코더(UTF-8) ─┬─ 화면(xterm)                      ← 그대로
                           └─ Logger.Write
                                ├ 꼬리 보관분 + 이번 묶음 → splitAnsiTail   (늘 한다)
                                ├ cleanLocked:  sessionLogAnsi 꺼짐 → StripForLog(시퀀스·CR 제거)
                                │               sessionLogAnsi 켜짐 → 그대로
                                ├ stampLinesLocked: 줄 앞 시각(켜져 있으면)
                                └ 버퍼 → 파일
```

설정이 전해지는 길(줄 앞 시각 D112 와 같다):

```
설정 화면 토글 → store.patch → app.go: 값이 바뀌었으면 Manager.SetLogAnsi
                                         └ 세션마다 Session.SetLogAnsi → Logger.SetAnsi
세션 열기 → channels_session.go: OpenConfig.LogAnsi → Session.bLogAnsi → (로그를 열 때) Logger.SetAnsi
```

### 2.4 영향 범위

- **저장 형식**: `settings` 에 키 하나(`sessionLogAnsi`)가 늘었다. 옛 파일에는 없으므로 기본값 `false` 로 채워진다(다른 설정과 같은 방식).
- **IPC**: 채널은 그대로. `store.get` 응답의 `settings` 에 키가 하나 늘어 계약 고정 파일 2개를 고쳤다.
- **꺼져 있을 때(기본)**: 로그 파일 내용이 전과 같다.

## A. 바뀐 파일 한눈에

| 파일 | 종류 | 상태 |
|---|---|---|
| `internal/session/log.go` | 코드(Go) | 수정 |
| `internal/session/types.go` | 코드(Go) | 수정 |
| `internal/session/session.go` | 코드(Go) | 수정 |
| `internal/session/control.go` | 코드(Go) | 수정 |
| `internal/session/manager.go` | 코드(Go) | 수정 |
| `internal/app/app.go` | 코드(Go) | 수정 |
| `internal/app/channels_session.go` | 코드(Go) | 수정 |
| `internal/store/store.go` | 코드(Go) | 수정 |
| `frontend/src/ipc/types.ts` | 코드 | 수정 |
| `frontend/src/lib/defaults.ts` | 코드 | 수정 |
| `frontend/src/state/ui.ts` | 코드 | 수정 |
| `frontend/src/screens/Settings.tsx` | 코드 | 수정 |
| `frontend/src/screens/SettingsConnection.tsx` | 코드 | 수정(로그 항목을 뺌) |
| `frontend/src/screens/SettingsLog.tsx` | 코드 | **새 파일** |
| `internal/session/log_test.go` | 테스트 | 수정(4개 추가) |
| `frontend/src/screens/SettingsLog.test.tsx` | 테스트 | **새 파일**(6개) |
| `frontend/src/ipc/contract/store.get.json` | 계약 고정 파일 | 수정 |
| `frontend/src/ipc/contract/store.get.afterNullPatch.json` | 계약 고정 파일 | 수정 |
| `locales/*.json` (10개) | 문구 | 키 3개 추가 |
| `spec/ui/settings.md` | 명세 | 수정 |
| `spec/behavior/session.md` | 명세 | 수정 |
| `spec/data/store.md` | 명세 | 수정 |
| `docs/scenarios.md`, `docs/scenarios.en.md` | 문서 | 수정 |
| `docs/decisions.md` | 문서 | D120 추가 |

아래의 `diff` 는 **같은 날의 앞 문서들(D106~D119)을 적용한 상태**를 기준으로 뽑았다.

## B. 코드

### `internal/session/log.go` — 수정

```diff
--- a/internal/session/log.go
+++ b/internal/session/log.go
@@ -71,6 +71,7 @@
 }
 
 // StripForLog 는 로그에 남길 텍스트를 만든다: ANSI 시퀀스와 CR 을 제거한다.
+// "ANSI 그대로 저장"(sessionLogAnsi)이 켜져 있으면 쓰지 않는다 (Logger.cleanLocked).
 func StripForLog(szText string) string {
 	szOut := g_reAnsi.ReplaceAllString(szText, "")
 	return strings.ReplaceAll(szOut, "\r", "")
@@ -114,6 +115,7 @@
 	tLastSync  time.Time        // 마지막으로 내려 쓴 시각 (처음에는 연 시각)
 	bTimestamp bool             // 줄마다 앞에 시각을 붙인다 (설정 sessionLogTimestamp)
 	bLineStart bool             // 다음에 쓸 글자가 줄의 첫 글자다 (시각을 붙일 자리)
+	bAnsi      bool             // ANSI 시퀀스와 CR 을 지우지 않고 받은 그대로 쓴다 (설정 sessionLogAnsi)
 	fnNow      func() time.Time // 시각을 얻는 함수 (시험에서 바꾼다)
 }
 
@@ -191,7 +193,7 @@
 	if szBody == "" {
 		return
 	}
-	_, _ = pLogger.pWriter.WriteString(pLogger.stampLinesLocked(StripForLog(szBody)))
+	_, _ = pLogger.pWriter.WriteString(pLogger.stampLinesLocked(pLogger.cleanLocked(szBody)))
 	if pLogger.iSyncSec == 0 {
 		// 즉시 모드: 부하가 걸리더라도 받는 대로 파일에 쓰고 내려 쓴다 (사용자가 고른 동작).
 		pLogger.syncNowLocked()
@@ -202,6 +204,24 @@
 	}
 }
 
+// cleanLocked 는 mtxData 를 잡은 상태에서 로그에 쓸 글을 만든다.
+// 평소에는 ANSI 시퀀스와 CR 을 지우고, "ANSI 그대로 저장"이 켜져 있으면 받은 그대로 돌려준다.
+// 어느 쪽이든 szText 는 완성된 시퀀스만 담고 있다(미완성 꼬리는 Write 가 떼어 보관한다).
+func (pLogger *Logger) cleanLocked(szText string) string {
+	if pLogger.bAnsi {
+		return szText
+	}
+	return StripForLog(szText)
+}
+
+// SetAnsi 는 ANSI 시퀀스(색·커서 이동 등)와 CR 을 지우지 않고 그대로 쓸지 바꾼다 (설정 sessionLogAnsi, D120).
+// 다음에 쓰는 조각부터 적용한다. 시퀀스 하나가 반은 지워지고 반은 남는 일은 없다(조각은 시퀀스 단위로 끊는다).
+func (pLogger *Logger) SetAnsi(bEnabled bool) {
+	pLogger.mtxData.Lock()
+	defer pLogger.mtxData.Unlock()
+	pLogger.bAnsi = bEnabled
+}
+
 // stampLinesLocked 는 mtxData 를 잡은 상태에서, 정리된 로그 글에 줄 시작 위치를 따라가며
 // (켜져 있으면) 줄마다 앞에 시각을 붙인다. 시각은 그 줄의 첫 글자를 받은 때다.
 // 꺼져 있어도 줄 시작 위치는 따라가므로, 중간에 켜면 다음 줄부터 시각이 붙는다.
@@ -304,7 +324,7 @@
 		pLogger.timerFlush = nil
 	}
 	if pLogger.szCarry != "" {
-		_, _ = pLogger.pWriter.WriteString(pLogger.stampLinesLocked(StripForLog(pLogger.szCarry)))
+		_, _ = pLogger.pWriter.WriteString(pLogger.stampLinesLocked(pLogger.cleanLocked(pLogger.szCarry)))
 		pLogger.szCarry = ""
 	}
 	errFlush := pLogger.pWriter.Flush()
```

### `internal/session/types.go` — 수정

```diff
--- a/internal/session/types.go
+++ b/internal/session/types.go
@@ -56,6 +56,8 @@
 	LogSyncSec int
 	// LogTimestamp 가 true 면 세션 로그의 줄마다 앞에 시각을 붙인다 (session.md §6, D112).
 	LogTimestamp bool
+	// LogAnsi 가 true 면 세션 로그에 ANSI 시퀀스와 CR 을 지우지 않고 받은 그대로 쓴다 (session.md §6, D120).
+	LogAnsi bool
 	// Zmodem 은 ZMODEM 파일 전송에 쓰는 앱 쪽 기능이다. nil 이면 이 세션은 ZMODEM 을 쓰지 않는다 (zmodem.md, D114).
 	Zmodem *ZmodemHooks
 	// StartupInput 은 접속 직후 셸에 보낼 입력이다 (tmux 한 줄 또는 자동 실행 명령). 비면 보내지 않는다.
```

### `internal/session/session.go` — 수정

```diff
--- a/internal/session/session.go
+++ b/internal/session/session.go
@@ -43,6 +43,7 @@
 	tLastInput   time.Time   // 지금 연결에 사용자 입력을 마지막으로 보낸 시각 (없으면 zero)
 	iLogSyncSec  int         // 세션 로그 내려 쓰기 주기(초). 설정을 바꾸면 SetLogSyncSec 으로 갱신된다
 	bLogStamp    bool        // 세션 로그 줄 앞에 시각을 붙일지. 설정을 바꾸면 SetLogTimestamp 로 갱신된다
+	bLogAnsi     bool        // 세션 로그에 ANSI 시퀀스를 그대로 쓸지. 설정을 바꾸면 SetLogAnsi 로 갱신된다
 	zActive      atomic.Bool // ZMODEM 전송 중 (터미널 입력을 서버로 보내지 않는다)
 	zCancel      atomic.Bool // 사용자가 ZMODEM 전송을 취소했다
 }
@@ -62,6 +63,7 @@
 
 		iLogSyncSec: ClampLogSyncSec(cfg.LogSyncSec),
 		bLogStamp:   cfg.LogTimestamp,
+		bLogAnsi:    cfg.LogAnsi,
 	}
 	pSession.info.M_szState = G_cszStateConnecting
 	szId := cfg.Info.M_szId
```

### `internal/session/control.go` — 수정

```diff
--- a/internal/session/control.go
+++ b/internal/session/control.go
@@ -143,6 +143,7 @@
 	if !bLost {
 		pLogger.SetSyncSec(pSession.iLogSyncSec) // 빈 파일이라 여기서 기다릴 일은 없다
 		pLogger.SetTimestamp(pSession.bLogStamp)
+		pLogger.SetAnsi(pSession.bLogAnsi)
 		pSession.logger = pLogger
 	}
 	pSession.mtxData.Unlock()
@@ -177,6 +178,18 @@
 	}
 }
 
+// SetLogAnsi 는 이 세션 로그에 ANSI 시퀀스와 CR 을 그대로 쓸지 바꾼다. 기록 중이면 다음 조각부터 적용하고,
+// 나중에 기록을 켤 때도 이 값을 쓴다 (설정 sessionLogAnsi, session.md §6, D120).
+func (pSession *Session) SetLogAnsi(bEnabled bool) {
+	pSession.mtxData.Lock()
+	pSession.bLogAnsi = bEnabled
+	pLogger := pSession.logger
+	pSession.mtxData.Unlock()
+	if pLogger != nil {
+		pLogger.SetAnsi(bEnabled)
+	}
+}
+
 // AckOutput 은 프런트가 출력 묶음 iCount 개를 처리했음을 반영한다 (spec/ipc.md session.ack).
 func (pSession *Session) AckOutput(iCount int) {
 	if iCount <= 0 {
```

### `internal/session/manager.go` — 수정

```diff
--- a/internal/session/manager.go
+++ b/internal/session/manager.go
@@ -129,6 +129,19 @@
 	}
 }
 
+// SetLogAnsi 는 열려 있는 모든 세션의 로그에 ANSI 시퀀스를 그대로 쓸지 바꾼다 (설정 sessionLogAnsi 를 바꿨을 때).
+func (pManager *Manager) SetLogAnsi(bEnabled bool) {
+	pManager.mtxData.Lock()
+	slSessions := make([]*Session, 0, len(pManager.mapSessions))
+	for _, pSession := range pManager.mapSessions {
+		slSessions = append(slSessions, pSession)
+	}
+	pManager.mtxData.Unlock()
+	for _, pSession := range slSessions {
+		pSession.SetLogAnsi(bEnabled)
+	}
+}
+
 // Count 는 열린 세션 수를 돌려준다 (종료 확인 문구에 쓴다).
 func (pManager *Manager) Count() int {
 	pManager.mtxData.Lock()
```

### `internal/app/app.go` — 수정

```diff
--- a/internal/app/app.go
+++ b/internal/app/app.go
@@ -153,6 +153,7 @@
 		bLogBefore := pService.settingBool("sessionLog", false)
 		iSyncBefore := pService.logSyncSec()
 		bStampBefore := pService.settingBool("sessionLogTimestamp", false)
+		bAnsiBefore := pService.settingBool("sessionLogAnsi", false)
 		errPatch := pService.pStore.Patch(mapPatch)
 		pService.applyLanguage()
 		if bLogNow := pService.settingBool("sessionLog", false); bLogNow != bLogBefore {
@@ -167,6 +168,10 @@
 			// 줄 앞 시각 표시도 열려 있는 세션에 곧바로(다음 줄부터) 적용한다 (session.md §6, D112).
 			pService.pSessions.SetLogTimestamp(bStampNow)
 		}
+		if bAnsiNow := pService.settingBool("sessionLogAnsi", false); bAnsiNow != bAnsiBefore {
+			// ANSI 그대로 저장도 열려 있는 세션에 곧바로(다음 출력부터) 적용한다 (session.md §6, D120).
+			pService.pSessions.SetLogAnsi(bAnsiNow)
+		}
 		if pService.logKeepDays() != iKeepBefore {
 			pService.pLogClean.Kick() // 보관 기간을 바꾸면 곧바로 한 번 정리한다 (session.md §6)
 		}
```

### `internal/app/channels_session.go` — 수정

```diff
--- a/internal/app/channels_session.go
+++ b/internal/app/channels_session.go
@@ -199,6 +199,7 @@
 		LogAlias:      record.M_szAlias,
 		LogSyncSec:    pService.logSyncSec(),
 		LogTimestamp:  pService.settingBool("sessionLogTimestamp", false),
+		LogAnsi:       pService.settingBool("sessionLogAnsi", false),
 		OpenDelay:     0,
 		Zmodem:        pService.zmodemHooksFor(record),
 	}
```

### `internal/store/store.go` — 수정

```diff
--- a/internal/store/store.go
+++ b/internal/store/store.go
@@ -55,6 +55,7 @@
 	"sessionLogKeepDays":  30,
 	"sessionLogSyncSec":   2,
 	"sessionLogTimestamp": false,
+	"sessionLogAnsi":      false, // 세션 로그에 ANSI 시퀀스를 그대로 남긴다 (session.md §6, D120)
 	"bell":                false,
 	"shiftEnterNewline":   true,
 	"rememberWindow":      true,
```

### `frontend/src/ipc/types.ts` — 수정

```diff
--- a/frontend/src/ipc/types.ts
+++ b/frontend/src/ipc/types.ts
@@ -126,6 +126,8 @@
   sessionLogSyncSec: number;
   /** 세션 로그의 줄마다 앞에 시각([YYYY-MM-DD HH:mm:ss.SSS])을 붙인다 (behavior/session.md §6). */
   sessionLogTimestamp: boolean;
+  /** 세션 로그에 ANSI 시퀀스(색·커서 이동)와 CR 을 지우지 않고 받은 그대로 남긴다 (behavior/session.md §6, D120). */
+  sessionLogAnsi: boolean;
   bell: boolean;
   shiftEnterNewline: boolean;
   rememberWindow: boolean;
```

### `frontend/src/lib/defaults.ts` — 수정

```diff
--- a/frontend/src/lib/defaults.ts
+++ b/frontend/src/lib/defaults.ts
@@ -46,6 +46,7 @@
   sessionLogKeepDays: 30,
   sessionLogSyncSec: 2,
   sessionLogTimestamp: false,
+  sessionLogAnsi: false,
   bell: false,
   shiftEnterNewline: true,
   rememberWindow: true,
```

### `frontend/src/state/ui.ts` — 수정

```diff
--- a/frontend/src/state/ui.ts
+++ b/frontend/src/state/ui.ts
@@ -45,7 +45,7 @@
   | { kind: 'host'; hostId: string | null; groupId: string | null }
   | { kind: 'folder'; folderId: string | null; parentId: string | null };
 
-export type SettingsSection = 'appearance' | 'input' | 'files' | 'connection' | 'sync' | 'security' | 'backup' | 'about';
+export type SettingsSection = 'appearance' | 'input' | 'files' | 'connection' | 'log' | 'sync' | 'security' | 'backup' | 'about';
 
 interface UiState {
   /** 설정 화면(크게 보기 모달). section 은 열 때 보일 구역(기본 모양). */
```

### `frontend/src/screens/Settings.tsx` — 수정

```diff
--- a/frontend/src/screens/Settings.tsx
+++ b/frontend/src/screens/Settings.tsx
@@ -1,6 +1,6 @@
-// 설정 화면: 크게 보기 모달, 왼쪽 구역 내비 8개 + 오른쪽 본문. 모든 변경은 즉시 저장. 근거: spec/ui/settings.md.
+// 설정 화면: 크게 보기 모달, 왼쪽 구역 내비 9개 + 오른쪽 본문. 모든 변경은 즉시 저장. 근거: spec/ui/settings.md.
 import { useState, type ReactNode, type KeyboardEvent } from 'react';
-import { Palette, Keyboard, ImageUp, Plug, RefreshCw, Shield, Archive, Info } from 'lucide-react';
+import { Palette, Keyboard, ImageUp, Plug, ScrollText, RefreshCw, Shield, Archive, Info } from 'lucide-react';
 import { Modal } from '../ui/Modal';
 import { useUi, type SettingsSection } from '../state/ui';
 import { useApp } from '../state/app';
@@ -10,6 +10,7 @@
 import { SettingsInput } from './SettingsInput';
 import { SettingsFiles } from './SettingsFiles';
 import { SettingsConnection } from './SettingsConnection';
+import { SettingsLog } from './SettingsLog';
 import { SettingsSync } from './SettingsSync';
 import { SettingsSecurity } from './SettingsSecurity';
 import { SettingsBackup } from './SettingsBackup';
@@ -22,6 +23,7 @@
   { id: 'input', label: 'settings.sec.input', icon: Keyboard },
   { id: 'files', label: 'settings.sec.files', icon: ImageUp },
   { id: 'connection', label: 'settings.sec.connection', icon: Plug },
+  { id: 'log', label: 'settings.sec.log', icon: ScrollText },
   { id: 'sync', label: 'settings.sec.sync', icon: RefreshCw },
   { id: 'security', label: 'settings.sec.security', icon: Shield },
   { id: 'backup', label: 'settings.sec.backup', icon: Archive },
@@ -98,6 +100,7 @@
     input: <SettingsInput />,
     files: <SettingsFiles />,
     connection: <SettingsConnection />,
+    log: <SettingsLog />,
     sync: <SettingsSync />,
     security: <SettingsSecurity />,
     backup: <SettingsBackup />,
```

### `frontend/src/screens/SettingsConnection.tsx` — 수정

```diff
--- a/frontend/src/screens/SettingsConnection.tsx
+++ b/frontend/src/screens/SettingsConnection.tsx
@@ -1,82 +1,27 @@
 // 설정 4 · 연결. 근거: spec/ui/settings.md §4, spec/behavior/lifecycle.md §3.
-import { FolderOpen } from 'lucide-react';
-import { ToggleRow, Field } from '../ui/Controls';
-import { getBackend, errorMessage } from '../ipc/backend';
-import { useApp } from '../state/app';
-import { NumberField, useSetting, Group } from './Settings';
-import { saveSettings } from '../state/settings';
+// 세션 로그 항목은 "로그" 구역(SettingsLog.tsx)으로 옮겼다 (D120).
+import { ToggleRow } from '../ui/Controls';
+import { useSetting, Group } from './Settings';
 import { t } from '../i18n';
 
 export function SettingsConnection() {
   const [autoReconnect, setAutoReconnect] = useSetting('autoReconnect');
   const [restoreLayout, setRestoreLayout] = useSetting('restoreLayout');
-  const [sessionLog, setSessionLog] = useSetting('sessionLog');
-  const [keepDays] = useSetting('sessionLogKeepDays');
-  const [syncSec] = useSetting('sessionLogSyncSec');
-  const [logStamp, setLogStamp] = useSetting('sessionLogTimestamp');
-  const pushToast = useApp((s) => s.pushToast);
-
-  const openLogs = async () => {
-    try {
-      await getBackend().invoke('app.openLogs');
-    } catch (e) {
-      pushToast({ kind: 'error', text: errorMessage(e) });
-    }
-  };
 
   return (
-    <>
-      <Group>
-        <ToggleRow
-          title={t('conn.autoReconnect')}
-          desc={t('conn.autoReconnectDesc')}
-          checked={autoReconnect}
-          onChange={setAutoReconnect}
-        />
-        <ToggleRow
-          title={t('conn.restoreLayout')}
-          desc={t('conn.restoreLayoutDesc')}
-          checked={restoreLayout}
-          onChange={setRestoreLayout}
-        />
-        <ToggleRow
-          title={t('conn.sessionLog')}
-          desc={t('conn.sessionLogDesc')}
-          checked={sessionLog}
-          onChange={setSessionLog}
-        />
-        <ToggleRow
-          title={t('conn.logTimestamp')}
-          desc={t('conn.logTimestampDesc')}
-          checked={!!logStamp}
-          onChange={setLogStamp}
-        />
-      </Group>
-      <Field label={t('conn.logKeepDays')} help={t('conn.logKeepDaysHelp')}>
-        <NumberField
-          label={t('conn.logKeepDays')}
-          value={keepDays}
-          min={0}
-          max={36500}
-          fallback={30}
-          onCommit={(v) => void saveSettings({ sessionLogKeepDays: v })}
-        />
-      </Field>
-      <Field label={t('conn.logSyncSec')} help={t('conn.logSyncSecHelp')}>
-        <NumberField
-          label={t('conn.logSyncSec')}
-          value={syncSec}
-          min={0}
-          max={5}
-          fallback={2}
-          onCommit={(v) => void saveSettings({ sessionLogSyncSec: v })}
-        />
-      </Field>
-      <div>
-        <button className="btn" type="button" onClick={() => void openLogs()}>
-          <FolderOpen size={14} /> {t('conn.openLogs')}
-        </button>
-      </div>
-    </>
+    <Group>
+      <ToggleRow
+        title={t('conn.autoReconnect')}
+        desc={t('conn.autoReconnectDesc')}
+        checked={autoReconnect}
+        onChange={setAutoReconnect}
+      />
+      <ToggleRow
+        title={t('conn.restoreLayout')}
+        desc={t('conn.restoreLayoutDesc')}
+        checked={restoreLayout}
+        onChange={setRestoreLayout}
+      />
+    </Group>
   );
 }
```

### `frontend/src/screens/SettingsLog.tsx` — 새 파일

```tsx
// 설정 · 로그(화면 순서 5번째). 세션 로그(터미널 출력 기록) 항목을 "연결"에서 떼어 낸 구역 (D120). 근거: spec/ui/settings.md §4a, spec/behavior/session.md §6.
import { FolderOpen } from 'lucide-react';
import { ToggleRow, Field } from '../ui/Controls';
import { getBackend, errorMessage } from '../ipc/backend';
import { useApp } from '../state/app';
import { NumberField, useSetting, Group } from './Settings';
import { saveSettings } from '../state/settings';
import { t } from '../i18n';

export function SettingsLog() {
  const [sessionLog, setSessionLog] = useSetting('sessionLog');
  const [keepDays] = useSetting('sessionLogKeepDays');
  const [syncSec] = useSetting('sessionLogSyncSec');
  const [logStamp, setLogStamp] = useSetting('sessionLogTimestamp');
  const [logAnsi, setLogAnsi] = useSetting('sessionLogAnsi');
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
          title={t('conn.sessionLog')}
          desc={t('conn.sessionLogDesc')}
          checked={sessionLog}
          onChange={setSessionLog}
        />
        <ToggleRow
          title={t('conn.logTimestamp')}
          desc={t('conn.logTimestampDesc')}
          checked={!!logStamp}
          onChange={setLogStamp}
        />
        <ToggleRow
          title={t('conn.logAnsi')}
          desc={t('conn.logAnsiDesc')}
          checked={!!logAnsi}
          onChange={setLogAnsi}
        />
      </Group>
      <Field label={t('conn.logKeepDays')} help={t('conn.logKeepDaysHelp')}>
        <NumberField
          label={t('conn.logKeepDays')}
          value={keepDays}
          min={0}
          max={36500}
          fallback={30}
          onCommit={(v) => void saveSettings({ sessionLogKeepDays: v })}
        />
      </Field>
      <Field label={t('conn.logSyncSec')} help={t('conn.logSyncSecHelp')}>
        <NumberField
          label={t('conn.logSyncSec')}
          value={syncSec}
          min={0}
          max={5}
          fallback={2}
          onCommit={(v) => void saveSettings({ sessionLogSyncSec: v })}
        />
      </Field>
      <div>
        <button className="btn" type="button" onClick={() => void openLogs()}>
          <FolderOpen size={14} /> {t('conn.openLogs')}
        </button>
      </div>
    </>
  );
}
```


## C. 테스트 코드

### `internal/session/log_test.go` — 수정

```diff
--- a/internal/session/log_test.go
+++ b/internal/session/log_test.go
@@ -458,3 +458,111 @@
 		t.Fatal("꺼져 있을 때 바꾼 설정도 다시 켤 때 써야 함")
 	}
 }
+
+// "ANSI 그대로 저장"이 꺼져 있으면(기본) 지금처럼 시퀀스와 CR 을 지우고, 켜면 받은 그대로 쓴다 (D120).
+func TestLogAnsiKeepsSequences(t *testing.T) {
+	const szIn = "\x1b[1;31mred\x1b[0m\r\n\x1b]0;title\x07plain\r\n"
+	for _, tc := range []struct {
+		szName string
+		bAnsi  bool
+		szWant string
+	}{
+		{"꺼짐", false, "red\nplain\n"},
+		{"켜짐", true, szIn},
+	} {
+		pLogger, errOpen := OpenLog(t.TempDir(), "x", time.Now())
+		if errOpen != nil {
+			t.Fatal(errOpen)
+		}
+		pLogger.SetSyncSec(0)
+		pLogger.SetAnsi(tc.bAnsi)
+		pLogger.Write(szIn)
+		bData, _ := os.ReadFile(pLogger.Path())
+		_ = pLogger.Close()
+		if string(bData) != tc.szWant {
+			t.Fatalf("%s:\n%q\n기대:\n%q", tc.szName, string(bData), tc.szWant)
+		}
+	}
+}
+
+// 시퀀스가 묶음 경계에서 잘려 와도 켜져 있으면 한 덩어리로 남고, 중간에 끄면 그 시퀀스는 통째로 지워진다.
+// (반은 남고 반은 지워진 조각이 로그에 생기지 않는다.) 닫을 때 남은 꼬리도 그때의 설정을 따른다.
+func TestLogAnsiToggleKeepsSequencesWhole(t *testing.T) {
+	pLogger, errOpen := OpenLog(t.TempDir(), "x", time.Now())
+	if errOpen != nil {
+		t.Fatal(errOpen)
+	}
+	pLogger.SetSyncSec(0)
+	pLogger.SetAnsi(true)
+	pLogger.Write("a\x1b[3")   // 미완성: 보관
+	pLogger.Write("1mb\x1b[0") // 앞 시퀀스는 완성돼 그대로 기록, 뒤는 다시 보관
+	pLogger.SetAnsi(false)
+	pLogger.Write("mc\r\n\x1b[32mgreen") // 보관분 + 이번 조각은 지워서 기록
+	pLogger.SetAnsi(true)
+	pLogger.Write("\x1b[0m\r\nend\x1b[") // 다시 그대로. 끝의 미완성 꼬리는 닫을 때 그대로 쓴다
+	_ = pLogger.Close()
+	bData, _ := os.ReadFile(pLogger.Path())
+	szWant := "a\x1b[31mb" + "c\ngreen" + "\x1b[0m\r\nend\x1b["
+	if string(bData) != szWant {
+		t.Fatalf("켜고 끄기:\n%q\n기대:\n%q", string(bData), szWant)
+	}
+}
+
+// 줄 앞 시각과 함께 켜면 시각은 줄바꿈(LF) 기준으로 지금처럼 붙고, 시퀀스와 CR 은 그대로 남는다.
+func TestLogAnsiWithTimestamp(t *testing.T) {
+	pLogger, errOpen := OpenLog(t.TempDir(), "x", time.Now())
+	if errOpen != nil {
+		t.Fatal(errOpen)
+	}
+	defer pLogger.Close()
+	tClock := time.Date(2026, 11, 23, 9, 5, 7, 123_000_000, time.Local)
+	pLogger.fnNow = func() time.Time { return tClock }
+	pLogger.SetSyncSec(0)
+	pLogger.SetTimestamp(true)
+	pLogger.SetAnsi(true)
+	pLogger.Write("\x1b[31mone\x1b[0m\r\ntwo\r\n")
+	bData, _ := os.ReadFile(pLogger.Path())
+	szWant := "[2026-11-23 09:05:07.123] \x1b[31mone\x1b[0m\r\n" +
+		"[2026-11-23 09:05:07.123] two\r\n"
+	if string(bData) != szWant {
+		t.Fatalf("시각 + ANSI:\n%q\n기대:\n%q", string(bData), szWant)
+	}
+}
+
+// 세션을 열 때 준 설정이 로그에 전해지고, 바꾸면 기록 중인 로그에도, 꺼져 있다가 다시 켠 로그에도 적용된다.
+func TestSessionLogAnsiApplies(t *testing.T) {
+	szDir := t.TempDir()
+	pFake := newFakeTransport()
+	rec := &eventRecorder{}
+	pManager := NewManager(rec.emit)
+	cfg := sshConfig(func(func(string)) (Transport, error) { return pFake, nil }, false)
+	cfg.DataDir = szDir
+	cfg.LogAlias = "web"
+	cfg.LogEnabled = true
+	cfg.LogAnsi = true
+	info, _ := pManager.Open(cfg)
+	defer pManager.Close(info.M_szId)
+	pSession, _ := pManager.Get(info.M_szId)
+	waitUntil(t, func() bool { return pSession.Snapshot().M_szState == G_cszStateConnected }, time.Second)
+	fnAnsi := func() bool {
+		pSession.mtxData.Lock()
+		pLogger := pSession.logger
+		pSession.mtxData.Unlock()
+		pLogger.mtxData.Lock()
+		defer pLogger.mtxData.Unlock()
+		return pLogger.bAnsi
+	}
+	if !fnAnsi() {
+		t.Fatal("열 때 준 설정이 로그에 전해져야 함")
+	}
+	pManager.SetLogAnsi(false)
+	if fnAnsi() {
+		t.Fatal("바꾼 설정이 기록 중인 로그에 적용돼야 함")
+	}
+	pManager.SetLogging(false)
+	pManager.SetLogAnsi(true)
+	pManager.SetLogging(true)
+	if !fnAnsi() {
+		t.Fatal("꺼져 있을 때 바꾼 설정도 다시 켤 때 써야 함")
+	}
+}
```

### `frontend/src/screens/SettingsLog.test.tsx` — 새 파일

```tsx
// 설정의 "로그" 구역(D120): 세션 로그 항목은 "연결"이 아니라 "로그"에 있고, "ANSI 그대로 저장"은 기본 꺼짐이며 누르면 저장된다.
// 근거: spec/ui/settings.md §4·§4a, spec/behavior/session.md §6.
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import { setBackend, type Backend } from '../ipc/backend';
import { useApp } from '../state/app';
import { useUi } from '../state/ui';
import { DEFAULT_SETTINGS } from '../lib/defaults';
import type { Store } from '../ipc/types';
import { Settings } from './Settings';

const invoke = vi.fn(async () => undefined);

/** 로그 구역에만 있어야 하는 항목 이름. */
const LOG_ITEMS = ['터미널 출력 기록', '로그 줄마다 시각 표시', 'ANSI 그대로 저장', '로그 보관 기간 (일)', '로그 파일 갱신 주기 (초)'];

beforeEach(() => {
  invoke.mockClear();
  setBackend({ invoke, termInput: vi.fn(), termResize: vi.fn(), reply: vi.fn(), on: () => () => undefined } as unknown as Backend);
  useApp.setState({ store: { version: 1, groups: [], hosts: [], snippets: [], knownHosts: [], settings: { ...DEFAULT_SETTINGS } } as Store });
});

function openSection(section: 'connection' | 'log') {
  useUi.setState({ settings: { open: true, section } });
  render(<Settings />);
}

/** store.patch 로 보낸 설정 값들(보낸 순서대로). */
function patched(key: string): unknown[] {
  return invoke.mock.calls
    .filter((c) => (c as unknown[])[0] === 'store.patch')
    .map((c) => ((c as unknown[])[1] as [{ settings: Record<string, unknown> }])[0].settings[key]);
}

describe('설정 > 로그 (D120)', () => {
  it('구역 목록에 "연결" 다음으로 "로그"가 있다 (9개)', () => {
    openSection('connection');
    const names = within(screen.getByRole('navigation')).getAllByRole('button').map((b) => b.textContent);
    expect(names).toEqual(['화면', '입력', '파일 전송', '연결', '로그', 'git 동기화', '보안', '백업', '정보']);
  });

  it('"연결"에는 로그 항목이 없고 자동 재연결·배치 복원만 남는다', () => {
    openSection('connection');
    for (const name of LOG_ITEMS) expect(screen.queryByText(name), name).toBeNull();
    expect(screen.queryByText('로그 폴더 열기')).toBeNull();
    expect(screen.getAllByRole('switch')).toHaveLength(2);
  });

  it('"로그"에 로그 항목이 모두 있다', () => {
    openSection('log');
    for (const name of LOG_ITEMS) expect(screen.getAllByText(name).length, name).toBeGreaterThan(0);
    expect(screen.getByText('로그 폴더 열기')).toBeTruthy();
    expect(screen.getAllByRole('switch')).toHaveLength(3);
  });

  it('"연결"에서 "로그"를 누르면 로그 구역으로 바뀐다', () => {
    openSection('connection');
    fireEvent.click(within(screen.getByRole('navigation')).getByText('로그'));
    expect(screen.getByText('ANSI 그대로 저장')).toBeTruthy();
  });

  it('"ANSI 그대로 저장"은 기본 꺼짐이고, 누르면 sessionLogAnsi: true 로 저장된다', async () => {
    expect(DEFAULT_SETTINGS.sessionLogAnsi).toBe(false);
    openSection('log');
    const sw = screen.getByRole('switch', { name: 'ANSI 그대로 저장' });
    expect(sw.getAttribute('aria-checked')).toBe('false');
    fireEvent.click(sw);
    await waitFor(() => expect(patched('sessionLogAnsi')).toEqual([true]));
  });

  it('[로그 폴더 열기]는 app.openLogs 를 부른다', async () => {
    openSection('log');
    fireEvent.click(screen.getByText('로그 폴더 열기'));
    await waitFor(() => expect(invoke).toHaveBeenCalledWith('app.openLogs'));
  });
});
```


## D. 계약 고정 파일 (백엔드 응답과 프런트 타입을 맞춰 보는 테스트가 읽는다)

### `frontend/src/ipc/contract/store.get.json` — 수정

```diff
--- a/frontend/src/ipc/contract/store.get.json
+++ b/frontend/src/ipc/contract/store.get.json
@@ -41,6 +41,7 @@
     "rightPanelWidth": 340,
     "scrollback": 10000,
     "sessionLog": false,
+    "sessionLogAnsi": false,
     "sessionLogKeepDays": 30,
     "sessionLogSyncSec": 2,
     "sessionLogTimestamp": false,
```

### `frontend/src/ipc/contract/store.get.afterNullPatch.json` — 수정

```diff
--- a/frontend/src/ipc/contract/store.get.afterNullPatch.json
+++ b/frontend/src/ipc/contract/store.get.afterNullPatch.json
@@ -41,6 +41,7 @@
     "rightPanelWidth": 340,
     "scrollback": 10000,
     "sessionLog": false,
+    "sessionLogAnsi": false,
     "sessionLogKeepDays": 30,
     "sessionLogSyncSec": 2,
     "sessionLogTimestamp": false,
```


## E. 문구

`locales/<언어>.json` 10개 파일에 같은 키를 넣는다(순서: ko, en, ja, zh, tw, de, fr, es, pt, ru). 한국어·영어 외에는 이 작업을 한 세션이 번역한 것이라 검수가 필요할 수 있다.

넣는 자리: `settings.sec.log` 는 `settings.sec.connection` 바로 다음 줄, `conn.logAnsi`·`conn.logAnsiDesc` 는 `conn.logTimestampDesc` 바로 다음 줄. 로그 구역으로 옮긴 기존 항목의 문구 키(`conn.sessionLog` 등)는 바꾸지 않았다.

**`settings.sec.log`** (추가)

```json
  "settings.sec.log": "로그",
  "settings.sec.log": "Logs",
  "settings.sec.log": "ログ",
  "settings.sec.log": "日志",
  "settings.sec.log": "記錄",
  "settings.sec.log": "Protokoll",
  "settings.sec.log": "Journaux",
  "settings.sec.log": "Registros",
  "settings.sec.log": "Logs",
  "settings.sec.log": "Журнал",
```

**`conn.logAnsi`** (추가)

```json
  "conn.logAnsi": "ANSI 그대로 저장",
  "conn.logAnsi": "Keep ANSI codes",
  "conn.logAnsi": "ANSI をそのまま保存",
  "conn.logAnsi": "保留 ANSI 代码",
  "conn.logAnsi": "保留 ANSI 代碼",
  "conn.logAnsi": "ANSI-Codes beibehalten",
  "conn.logAnsi": "Conserver les codes ANSI",
  "conn.logAnsi": "Conservar códigos ANSI",
  "conn.logAnsi": "Manter códigos ANSI",
  "conn.logAnsi": "Сохранять коды ANSI",
```

**`conn.logAnsiDesc`** (추가)

```json
  "conn.logAnsiDesc": "색·커서 이동 같은 ANSI 제어 코드를 지우지 않고 받은 그대로 로그에 남깁니다. 끄면(기본) 글자만 남깁니다. 켠 로그는 터미널에서 cat 이나 less -R 로 보면 색이 살아납니다 (바꾸면 다음 출력부터 적용)",
  "conn.logAnsiDesc": "Writes the log exactly as received, without removing ANSI control codes such as colors and cursor movement. When off (default), only the text is kept. View such a log with cat or less -R in a terminal to see the colors (changes apply from the next output)",
  "conn.logAnsiDesc": "色やカーソル移動などの ANSI 制御コードを取り除かず、受信したままログに残します。オフ（既定）では文字だけを残します。オンにしたログはターミナルで cat や less -R で見ると色が再現されます（変更は次の出力から適用）",
  "conn.logAnsiDesc": "不删除颜色、光标移动等 ANSI 控制代码，按收到的原样写入日志。关闭（默认）时只保留文字。开启后的日志可在终端中用 cat 或 less -R 查看以显示颜色（更改从下一次输出起生效）",
  "conn.logAnsiDesc": "不移除顏色、游標移動等 ANSI 控制代碼，依收到的原樣寫入記錄。關閉（預設）時只保留文字。開啟後的記錄可在終端機中用 cat 或 less -R 檢視以顯示顏色（變更從下一次輸出起生效）",
  "conn.logAnsiDesc": "Schreibt das Protokoll so, wie es empfangen wurde, ohne ANSI-Steuercodes wie Farben und Cursorbewegungen zu entfernen. Aus (Standard): nur der Text bleibt. Ein solches Protokoll zeigt mit cat oder less -R im Terminal die Farben (Änderungen gelten ab der nächsten Ausgabe)",
  "conn.logAnsiDesc": "Écrit le journal tel qu'il est reçu, sans retirer les codes de contrôle ANSI comme les couleurs et les déplacements du curseur. Désactivé (par défaut), seul le texte est conservé. Affichez un tel journal avec cat ou less -R dans un terminal pour voir les couleurs (les changements s'appliquent à partir de la sortie suivante)",
  "conn.logAnsiDesc": "Escribe el registro tal como se recibe, sin quitar los códigos de control ANSI como colores y movimientos del cursor. Desactivado (predeterminado), solo se guarda el texto. Vea ese registro con cat o less -R en un terminal para ver los colores (los cambios se aplican desde la siguiente salida)",
  "conn.logAnsiDesc": "Grava o log exatamente como recebido, sem remover códigos de controle ANSI como cores e movimentos do cursor. Desativado (padrão), apenas o texto é mantido. Veja esse log com cat ou less -R em um terminal para ver as cores (as alterações valem a partir da próxima saída)",
  "conn.logAnsiDesc": "Записывает журнал в том виде, в каком он получен, не удаляя управляющие коды ANSI (цвета, перемещение курсора). Выключено (по умолчанию) — остаётся только текст. Такой журнал показывает цвета при просмотре в терминале через cat или less -R (изменение действует со следующего вывода)",
```


## F. 명세·문서

### `spec/ui/settings.md` — 수정

```diff
--- a/spec/ui/settings.md
+++ b/spec/ui/settings.md
@@ -2,7 +2,7 @@
 
 > 작성 방식: 기존 앱 실행 관찰 + 열람. 작성일 2026-11-10. 상태: **draft**. 기본값·키 이름은 data/store.md §4.
 
-크게 보기 모달, 제목 "설정". 왼쪽 **내비게이션**(구역 8개, 선택된 구역 강조) + 오른쪽 **본문**. 모든 변경은 **즉시 적용·저장**된다(저장 버튼 없음). 열 때 구역을 지정할 수 있고 기본은 "화면".
+크게 보기 모달, 제목 "설정". 왼쪽 **내비게이션**(구역 9개, 선택된 구역 강조) + 오른쪽 **본문**. 모든 변경은 **즉시 적용·저장**된다(저장 버튼 없음). 열 때 구역을 지정할 수 있고 기본은 "화면".
 
 | 순서 | 구역 | 아이콘 |
 |---|---|---|
@@ -10,10 +10,13 @@
 | 2 | 입력·붙여넣기 | 키보드 |
 | 3 | 이미지·파일 전송 | 이미지 업로드 |
 | 4 | 연결 | 플러그 |
-| 5 | 동기화 (git) | 새로고침 |
-| 6 | 보안·계정 | 방패 |
-| 7 | 백업 | 보관함 |
-| 8 | 정보·단축키 | 정보 |
+| 5 | 로그 | 두루마리 문서 |
+| 6 | 동기화 (git) | 새로고침 |
+| 7 | 보안·계정 | 방패 |
+| 8 | 백업 | 보관함 |
+| 9 | 정보·단축키 | 정보 |
+
+> 2026-11-23 (D120): "연결"에 있던 세션 로그 항목을 **"로그" 구역**으로 떼어 냈다. 아래 절 번호는 다른 문서·코드가 가리키고 있어 바꾸지 않았다 — 로그 구역은 **§4a** 다(화면의 순서는 위 표).
 
 ## 1. 모양
 - **언어**(맨 위, 목록 선택, D75): 도움말 "화면과 안내 문구의 언어입니다. 바꾸면 바로 적용됩니다." 목록은 behavior/i18n.md §1 의 10개(그 언어로 쓴 이름). 지금 화면 언어가 골라져 있다.
@@ -53,8 +56,13 @@
 - 카드 토글:
   1. **끊기면 자동 재접속** — "연결이 끊기면 2·4·8·16·30초로 간격을 늘리며 최대 5번 재연결합니다"
   2. **탭·분할 레이아웃 복원** — "종료할 때 열린 탭과 창 배치를 저장하고, 다음 실행 때 같은 서버로 다시 연결합니다. 비밀번호·화면 내용은 저장하지 않으며, 목록에 없는 일회성 SSH 연결은 복원하지 않습니다"
-  3. **세션 로그 저장** — "터미널 출력을 텍스트 파일로 남깁니다 (켜면 열려 있는 연결도 그때부터 기록)"
-  4. **로그 줄마다 시각 표시**(2026-11-23, D112) — "기록하는 로그의 줄 앞에 [날짜 시각.밀리초]를 붙입니다. 화면에는 보이지 않고 파일에만 남습니다 (바꾸면 다음 줄부터 적용)"
+
+## 4a. 로그
+(2026-11-23, D120. 항목의 동작은 behavior/session.md §6.)
+- 카드 토글:
+  1. **세션 로그 저장** — "터미널 출력을 텍스트 파일로 남깁니다 (켜면 열려 있는 연결도 그때부터 기록)"
+  2. **로그 줄마다 시각 표시**(2026-11-23, D112) — "기록하는 로그의 줄 앞에 [날짜 시각.밀리초]를 붙입니다. 화면에는 보이지 않고 파일에만 남습니다 (바꾸면 다음 줄부터 적용)"
+  3. **ANSI 그대로 저장**(2026-11-23, D120, 기본 꺼짐) — "색·커서 이동 같은 ANSI 제어 코드를 지우지 않고 받은 그대로 로그에 남깁니다. 끄면(기본) 글자만 남깁니다. 켠 로그는 터미널에서 cat 이나 less -R 로 보면 색이 살아납니다 (바꾸면 다음 출력부터 적용)" — 값은 `sessionLogAnsi`.
 - 필드 **로그 보관 기간 (일)**(숫자, 0~36500, 비면 30, 도움말 "이 일수보다 오래된 세션 로그 파일을 자동으로 지웁니다. 0이면 지우지 않습니다.") — 값은 `sessionLogKeepDays`(behavior/session.md §6 보관 기간).
 - 필드 **로그 파일 갱신 주기 (초)**(숫자, 0~5, 비면 2, 도움말 "기록 중인 로그를 디스크까지 내려 쓰는 주기입니다(0~5). 내려 써야 탐색기에 파일 크기와 수정 시각이 보입니다. 0이면 출력을 받는 즉시 쓰고 내려 써서 디스크 부하가 커집니다. 0이 아니면 파일에는 1초마다 씁니다.") — 값은 `sessionLogSyncSec`(behavior/session.md §6 내려 쓰기 주기). (2026-11-18)
 - 버튼 [로그 폴더 열기](폴더 아이콘) → 로그 폴더를 탐색기로 연다.
```

### `spec/behavior/session.md` — 수정

```diff
--- a/spec/behavior/session.md
+++ b/spec/behavior/session.md
@@ -85,7 +85,7 @@
 ## 6. 세션 로그
 - 설정 "터미널 출력 기록"(`sessionLog`)이 켜져 있을 때 **세션을 열 때 한 번** 로그 폴더(`<데이터 폴더>\logs`)에 `{표시 이름}-{YYYYMMDD-HHmmss}.log` 를 **새로 만든다**(권한 소유자 전용, 열린 뒤에 설정을 켠 경우는 아래). 같은 이름이 이미 있으면 `.log` 앞에 `-2`, `-3`, … 을 붙여 없는 이름으로 만든다(세션 하나에 파일 하나). 1000번까지 시도해도 못 만들면 로그 없이 연다.
 - 파일 이름의 표시 이름: 별칭의 `\ / : * ? " < > |` 와 제어 문자는 `_` 로 바꾸고, 앞뒤 공백·점을 지우며, 80자로 자른다. 비면 `session`.
-- 기록 내용: 화면 출력에서 **ANSI 이스케이프 시퀀스를 제거**하고 CR 도 제거한 텍스트. 시퀀스가 출력 묶음 경계에서 잘려도 이어 붙여 지운다(종결 없이 256바이트를 넘는 ESC 는 시퀀스로 보지 않고 그대로 둔다). 디스크에는 **약 1초 단위로 모아서** 쓰고, 세션을 닫을 때 남은 것을 쓴다(앱이 비정상 종료하면 마지막 1초 분량이 빠질 수 있다).
+- 기록 내용(기본, 설정 `sessionLogAnsi` 가 꺼져 있을 때): 화면 출력에서 **ANSI 이스케이프 시퀀스를 제거**하고 CR 도 제거한 텍스트. 시퀀스가 출력 묶음 경계에서 잘려도 이어 붙여 지운다(종결 없이 256바이트를 넘는 ESC 는 시퀀스로 보지 않고 그대로 둔다). 디스크에는 **약 1초 단위로 모아서** 쓰고, 세션을 닫을 때 남은 것을 쓴다(앱이 비정상 종료하면 마지막 1초 분량이 빠질 수 있다).
 - 자동·수동 재접속은 같은 파일에 이어 쓰고, 세션을 닫으면 파일을 닫는다.
 - **설정을 바꾸면 열려 있는 세션에도 곧바로 적용**(2026-11-18): 켜면 열려 있는 세션마다 그 순간의 시각으로 새 로그 파일을 만들어 **그때부터의 출력**을 기록한다(이전 출력은 넣지 않는다, 이미 기록 중인 세션은 그대로). 끄면 열려 있는 세션의 기록을 멈추고 남은 것을 쓴 뒤 파일을 닫는다. 다시 켜면 새 파일이다.
 - **내려 쓰기 주기**(2026-11-18): 설정 `sessionLogSyncSec`(초, 0~5, 기본 2, 범위 밖 값은 가까운 끝으로). 로그 파일은 세션 동안 열어 두는데, Windows 는 열려 있는 파일의 크기·수정 시각을 **디스크까지 내려 쓰거나 닫을 때만** 폴더 목록에 반영한다. 그래서 기록 중에도 파일 크기가 보이도록 주기적으로 내려 쓴다.
@@ -93,6 +93,7 @@
   - 값이 0 이면: 출력을 받는 **즉시** 파일에 쓰고 내려 쓴다(모으지 않는다). 출력이 많은 세션에서는 디스크 부하와 출력 지연이 커질 수 있다.
   - 설정을 바꾸면 기록 중인 세션에도 곧바로 적용한다.
 - **줄 앞 시각**(2026-11-23, D112): 설정 `sessionLogTimestamp`(기본 꺼짐). 켜면 로그의 **줄마다 앞에** `[YYYY-MM-DD HH:mm:ss.SSS] `(이 PC 의 현지 시각, 밀리초까지, 뒤에 공백 하나)를 붙인다. 시각은 **그 줄의 첫 글자를 받은 때**다(줄이 여러 묶음에 걸쳐 와도 한 번만 붙는다). 빈 줄에도 붙는다. 화면 출력에는 붙이지 않는다(파일에만). 설정을 바꾸면 기록 중인 세션에도 곧바로 적용하되, 쓰던 줄 중간에는 넣지 않고 **다음 줄부터** 적용한다. 커서 이동으로 같은 줄을 고쳐 쓰는 출력(진행 막대 등)은 줄바꿈이 올 때까지 한 줄이므로 시각도 한 번이다.
+- **ANSI 그대로 저장**(2026-11-23, D120): 설정 `sessionLogAnsi`(기본 꺼짐). 켜면 ANSI 이스케이프 시퀀스(색·커서 이동·창 제목 등)와 CR 을 **지우지 않고 받은 그대로** 쓴다(글자 인코딩은 지금처럼 UTF-8 로 바꾼 뒤의 것). 파일 이름·확장자(`.log`)·쓰는 주기·보관 기간은 꺼져 있을 때와 같다. 시퀀스가 출력 묶음 경계에서 잘려 오면 지금처럼 이어 붙인 뒤에 쓴다(시퀀스 하나는 늘 한 번에 쓴다). 설정을 바꾸면 기록 중인 세션에도 곧바로, **다음 출력부터** 적용한다 — 같은 파일에 이어 쓰므로 바꾼 자리 앞뒤로 두 형식이 섞이지만, 시퀀스 하나가 반만 남는 일은 없다. 줄 앞 시각과 함께 켜면 시각은 지금처럼 줄바꿈(LF) 뒤의 첫 글자 앞에 붙는다(시퀀스가 줄의 첫 글자면 시각이 그 앞에 온다). 화면 출력은 달라지지 않는다.
 - **보관 기간**: 설정 `sessionLogKeepDays`(기본 30, 0 이면 지우지 않음). 앱은 시작 약 10초 뒤와 그 뒤 **1시간마다** 로그 폴더 바로 아래의 `.log` 파일 중 수정 시각이 그 일수보다 오래된 것을 지운다. 지금 열려 있는(기록 중인) 세션의 파일과 하위 폴더·다른 확장자 파일은 건드리지 않는다. 설정 값을 바꾸면 곧바로 한 번 정리한다. 지우지 못한 파일(다른 프로그램이 열고 있음 등)은 건너뛰고 다음 주기에 다시 본다. "터미널 출력 기록"이 꺼져 있어도 남은 파일은 정리한다.
 
 ## 7. 세션 유지 (tmux)
```

### `spec/data/store.md` — 수정

```diff
--- a/spec/data/store.md
+++ b/spec/data/store.md
@@ -101,6 +101,7 @@
 | sessionLog | false | |
 | sessionLogKeepDays | 30 | 세션 로그 보관 일수. 0 이면 지우지 않음 ([../behavior/session.md](../behavior/session.md) §6) |
 | sessionLogTimestamp | false | 세션 로그의 줄마다 앞에 시각을 붙인다 ([../behavior/session.md](../behavior/session.md) §6, D112) |
+| sessionLogAnsi | false | 세션 로그에 ANSI 시퀀스와 CR 을 지우지 않고 받은 그대로 남긴다 ([../behavior/session.md](../behavior/session.md) §6, D120) |
 | sessionLogSyncSec | 2 | 기록 중인 세션 로그를 디스크까지 내려 쓰는 주기(초), 0~5. 0 이면 출력을 받는 즉시 쓰고 내려 쓴다 ([../behavior/session.md](../behavior/session.md) §6, 2026-11-18) |
 | bell | false | |
 | shiftEnterNewline | true | Shift+Enter 로 줄바꿈 입력 |
```

### `docs/scenarios.md` — 수정

```diff
--- a/docs/scenarios.md
+++ b/docs/scenarios.md
@@ -107,7 +107,7 @@
 | 6-8 | 글자 크기 | Ctrl+= / Ctrl+- / Ctrl+0 | 8~32, 0 은 14 로. 설정에 저장. |
 | 6-9 | 여러 줄 입력(Claude Code 등) | Shift+Enter | 실행하지 않고 줄만 바꿈(설정, 기본 켬). |
 | 6-10 | 서버가 클립보드에 씀 | tmux·vim 등이 OSC 52 로 복사 | 이 PC 클립보드에 넣는다(설정, 읽기 요청은 늘 거부). |
-| 6-11 | 출력 기록 | 설정 > 연결 "터미널 출력 기록" 켬(기본 꺼짐) | 켜면 열려 있는 연결도 그때부터, 새 연결은 처음부터 텍스트 파일로 저장. 끄면 기록을 멈춤. [로그 폴더 열기]. |
+| 6-11 | 출력 기록 | 설정 > 로그 "터미널 출력 기록" 켬(기본 꺼짐) | 켜면 열려 있는 연결도 그때부터, 새 연결은 처음부터 텍스트 파일로 저장. 끄면 기록을 멈춤. [로그 폴더 열기]. "ANSI 그대로 저장"을 켜면(기본 꺼짐) 색·커서 이동 코드를 지우지 않고 받은 그대로 남긴다. |
 | 6-12 | 화면 비우기·전체 선택 | 칸 우클릭 | 화면 비우기, 전체 선택, 복사, 붙여넣기, 이미지 붙여넣기, 찾기, 같은 서버를 오른쪽에/아래에, SFTP(파일) 창 열기, 재연결, 창 닫기. |
 
 ## 7. 탭·창 나누기
@@ -175,14 +175,15 @@
 | 12-4 | tmux 세션 유지 | 서버 고급 "연결 유지 (tmux)": 창·앱을 닫아도 서버의 프로그램이 계속 돌고, 다시 열면 같은 화면에 붙는다. tmux 가 없으면 안내 후 평소처럼. Windows 로컬은 WSL 셸만. |
 | 12-5 | 로컬 셸이 끝남 | "셸이 종료되었습니다" → Enter 로 새 셸. |
 
-## 13. 설정 (톱니바퀴, 구역 8개)
+## 13. 설정 (톱니바퀴, 구역 9개)
 
 | 구역 | 주요 항목 |
 |---|---|
 | 화면 | 언어 · 앱 테마(다크/라이트 2종) · 터미널 색상 6종(DariTerm Dark·Light, PuTTY Classic, Solarized Dark, Monokai, Nord) · 글꼴·크기·줄 간격 · 창 크기·위치 기억 · [기본 크기로] · 커서 모양 · 기록 줄 수 · 위아래 창 최대 개수 · 커서 깜빡임 · 경고음 |
 | 입력 | 드래그하면 바로 복사 · 오른쪽 버튼 붙여넣기 · Ctrl+V 붙여넣기 · Shift+Enter 줄바꿈 · OSC 52 · 여러 줄 붙여넣기 확인 · 운영 서버 위험 명령 경고와 패턴 목록 |
 | 파일 전송 | 서버 업로드 폴더 · 자동 정리 일수 · 서버 간 복사 임시 폴더 · 경로 따옴표 |
-| 연결 | 자동 재연결 · 마지막 화면 배치 복원 · 터미널 출력 기록 · 로그 폴더 |
+| 연결 | 자동 재연결 · 마지막 화면 배치 복원 |
+| 로그 | 터미널 출력 기록 · 로그 줄마다 시각 표시 · ANSI 그대로 저장 · 로그 보관 기간 · 로그 파일 갱신 주기 · 로그 폴더 |
 | git 동기화 | 저장소 주소 · 지금 동기화 · 자동 동기화 |
 | 보안 | 자동 잠금(분) · Windows 로그인으로 열기 · 비밀번호 변경 · 저장된 계정 |
 | 백업 | 내보내기 · 가져오기 |
```

### `docs/scenarios.en.md` — 수정

```diff
--- a/docs/scenarios.en.md
+++ b/docs/scenarios.en.md
@@ -106,7 +106,7 @@
 | 6-8 | Font size | Ctrl+= / Ctrl+- / Ctrl+0 | 8–32; Ctrl+0 resets to 14. Saved in settings. |
 | 6-9 | Multi-line input (Claude Code, etc.) | Shift+Enter | New line without running (setting, on by default). |
 | 6-10 | Server writes the clipboard | tmux, vim, etc. copy via OSC 52 | Put on this PC's clipboard (setting; read requests are always refused). |
-| 6-11 | Log output | Settings > Connection "Log terminal output" on (off by default) | Open connections start saving output to text files from that moment, new connections from the start. Turning it off stops recording. [Open log folder]. |
+| 6-11 | Log output | Settings > Logs "Log terminal output" on (off by default) | Open connections start saving output to text files from that moment, new connections from the start. Turning it off stops recording. [Open log folder]. With "Keep ANSI codes" on (off by default) the log keeps color and cursor codes exactly as received. |
 | 6-12 | Pane context menu | Right-click a pane | Clear screen, Select all, Copy, Paste, Paste image, Find, Same server to the right/below, Open SFTP (file) panel, Reconnect, Close pane. |
 
 ## 7. Tabs and split panes
@@ -174,14 +174,15 @@
 | 12-4 | Keep session with tmux | Server Advanced "Keep session (tmux)": programs on the server keep running after you close the pane or app, and reopening reattaches to the same screen. Without tmux it says so and works normally. On local Windows, WSL shells only. |
 | 12-5 | Local shell exits | "Shell exited" → Enter starts a new shell. |
 
-## 13. Settings (gear icon, 8 sections)
+## 13. Settings (gear icon, 9 sections)
 
 | Section | Main items |
 |---|---|
 | Appearance | Language · app theme (Dark/Light) · 6 terminal color themes (DariTerm Dark·Light, PuTTY Classic, Solarized Dark, Monokai, Nord) · font, size, line spacing · remember window size and position · [Reset to default size] · cursor style · scrollback lines · maximum vertical panes · blinking cursor · bell sound |
 | Input | Copy on select · paste with right click · paste with Ctrl+V · new line with Shift+Enter · OSC 52 · ask before pasting multiple lines · production dangerous-command warning and pattern list |
 | File transfer | Server upload folder · auto-delete days · temp folder for server-to-server copies · quote paths |
-| Connection | Reconnect when disconnected · restore last layout · log terminal output · log folder |
+| Connection | Reconnect when disconnected · restore last layout |
+| Logs | Log terminal output · timestamp each log line · keep ANSI codes · keep session logs (days) · log file update interval · log folder |
 | git sync | Repository address · Sync now · automatic sync |
 | Security | Auto-lock (minutes) · unlock with Windows sign-in · change password · saved accounts |
 | Backup | Export · Import |
```

---

## G. 검증 (2026-11-23 실행)

### G.1 백엔드 — [internal/session/log_test.go](../../internal/session/log_test.go) 에 4개 추가

| 테스트 | 확인하는 것 |
|---|---|
| `TestLogAnsiKeepsSequences` | 꺼짐(기본): 색 코드·창 제목 시퀀스·CR 이 지워진 글자만 남는다 / 켜짐: 파일 내용이 받은 것과 **한 바이트도 다르지 않다** |
| `TestLogAnsiToggleKeepsSequencesWhole` | 시퀀스가 묶음 경계에서 잘려 와도 켜져 있으면 한 덩어리로 남는다 / 보관 중이던 미완성 시퀀스는 그 뒤에 끄면 통째로 지워진다(조각이 남지 않는다) / 다시 켜면 다음 출력부터 그대로 / 닫을 때 남은 꼬리도 그때의 설정을 따른다 |
| `TestLogAnsiWithTimestamp` | 줄 앞 시각과 함께 켜면 시각은 LF 기준으로 줄마다 하나, 시퀀스와 CR 은 그대로 |
| `TestSessionLogAnsiApplies` | 세션을 열 때 준 설정이 로그에 전해진다 / 바꾸면 기록 중인 로그에 적용 / 기록이 꺼져 있을 때 바꾼 값도 다시 켤 때 쓴다 |
| 기존 로그 테스트 전부 | 그대로 통과(꺼져 있으면 파일 내용이 전과 같다) |
| 계약 테스트(`internal/app`) | `store.get` 의 `sessionLogAnsi: false` 가 프런트 고정 파일과 같다 |

### G.2 프런트 — 새 파일 [frontend/src/screens/SettingsLog.test.tsx](../../frontend/src/screens/SettingsLog.test.tsx) 6개

| 테스트 | 확인하는 것 |
|---|---|
| 구역 목록에 "연결" 다음으로 "로그"가 있다 (9개) | 내비게이션의 순서: 화면·입력·파일 전송·연결·**로그**·git 동기화·보안·백업·정보 |
| "연결"에는 로그 항목이 없다 | 로그 항목 5개와 [로그 폴더 열기]가 보이지 않고 스위치는 2개 |
| "로그"에 로그 항목이 모두 있다 | 항목 5개 + 버튼, 스위치 3개 |
| "연결"에서 "로그"를 누르면 바뀐다 | 구역 전환 |
| "ANSI 그대로 저장"은 기본 꺼짐, 누르면 저장 | 기본값 `false` / 스위치가 꺼져 있음 / 누르면 `store.patch` 로 `sessionLogAnsi: true` |
| [로그 폴더 열기] | `app.openLogs` 호출(옮긴 뒤에도 동작) |

### G.3 실행한 명령

```bash
go build ./... && go vet ./internal/session/ ./internal/app/ ./internal/store/   # 통과
go test ./...                          # 전부 통과 (session 18초, app·i18n·store 포함)
(cd frontend && npx tsc --noEmit)      # 통과
(cd frontend && npx vitest run)        # PowerShell 에서. 45개 파일 291개 통과 (앞 문서까지 44개 파일 285개 + 이 문서 6개)
go run github.com/wailsapp/wails/v2/cmd/wails@v2.17.1 build   # build/bin/dariterm.exe
```

문구 검사(10개 언어에 같은 키, 자리표시자 일치)는 `go test ./internal/i18n` 과 프런트의 `src/i18n` 테스트가 본다 — 둘 다 통과.

자동 테스트가 확인하지 못하는 것:
- `store.patch` 에서 값이 바뀌었을 때 세션까지 이어지는 경로(app.go 의 4줄). D99·D100·D112 와 같은 상태다.
- 실제 서버 출력으로 만든 로그를 터미널에서 다시 봤을 때 색이 살아나는지.
- 설정 화면의 모양(아이콘, 9번째 구역이 들어가도 내비게이션이 넘치지 않는지).

## H. 수동 검증 (남은 일)

| # | 절차 | 기대 결과 |
|---|---|---|
| 1 | 설정을 연다 | 왼쪽에 구역 9개. "연결" 다음에 "로그"(두루마리 아이콘). 목록이 잘리거나 스크롤이 생기지 않는다 |
| 2 | "연결"을 누른다 | 자동 재연결·배치 복원 토글 두 개만 있다 |
| 3 | "로그"를 누른다 | 터미널 출력 기록 · 로그 줄마다 시각 표시 · ANSI 그대로 저장(꺼져 있음) · 로그 보관 기간 · 로그 파일 갱신 주기 · [로그 폴더 열기] |
| 4 | 옮기기 전에 켜 둔 값(출력 기록, 보관 기간 등)이 있는 PC | 값이 그대로 보인다(설정 키는 바뀌지 않았다) |
| 5 | "터미널 출력 기록"만 켜고 세션에서 `ls --color=always` | 로그 파일에 색 코드가 없다(전과 같다) |
| 6 | "ANSI 그대로 저장"을 켜고 새 세션에서 `ls --color=always` → 로그 폴더의 그 파일을 다른 터미널에서 `cat` 또는 `less -R` | 색이 화면과 같게 보인다. 메모장으로 열면 `[01;34m` 같은 코드가 보인다 |
| 7 | 세션을 연 채로 토글을 켰다 껐다 하며 명령을 몇 개 실행 | 같은 파일에 이어 쓰인다. 켠 동안의 출력만 코드가 있고, 반쯤 잘린 코드 조각(`1m` 등)이 글자로 남지 않는다 |
| 8 | "ANSI 그대로 저장"과 "로그 줄마다 시각 표시"를 함께 켬 | 줄마다 앞에 시각이 붙고 그 뒤에 색 코드가 그대로 온다 |
| 9 | 켠 상태에서 `vim`·`top` 을 잠깐 실행 | 로그가 커진다(화면을 다시 그리는 코드가 모두 들어간다). `cat` 으로 보면 화면이 재생되듯 지나간다 |
| 10 | 다른 언어(영어 등)로 바꿔 설정을 본다 | 구역 이름 "Logs", 토글 "Keep ANSI codes" 와 설명이 그 언어로 나온다 |

## I. 영향 범위·회귀 위험

| 영역 | 위험 | 대비 |
|---|---|---|
| 꺼져 있을 때(기본) | `cleanLocked` 가 지금처럼 `StripForLog` 를 부른다 | 기존 로그 테스트 전부 통과 |
| 설정 화면의 다른 구역 | 구역 목록에 한 줄이 늘었을 뿐 다른 구역의 내용은 건드리지 않았다 | `tsc`(구역 id 를 빠뜨리면 컴파일 오류), 구역 순서 테스트 |
| 설정 값 | 키를 옮기지 않았다. 옛 저장 파일은 `sessionLogAnsi` 가 없어 `false` 로 채워진다 | 계약 테스트, `store` 테스트 |
| 켰을 때 로그 크기 | 색이 많은 출력이나 `vim`·`top` 같은 화면 프로그램은 로그가 몇 배 커질 수 있다 | 켠 사람만 해당. 보관 기간 설정은 그대로 적용된다 |
| 켰을 때 로그를 읽는 도구 | `grep`·메모장에는 코드가 글자로 보인다. 진행 막대는 CR 로 덮어 쓴 것이 모두 남는다 | 설명 문구에 `cat`·`less -R` 로 보라고 적었다 |
| 켰을 때 보안 | 창 제목 변경·클립보드 쓰기(OSC 52) 같은 시퀀스도 그대로 남는다. 이런 로그를 터미널에서 `cat` 하면 그 터미널이 시퀀스를 다시 실행한다(제목이 바뀌는 등) | 서버가 보낸 것을 그대로 다시 보는 것이라 원래 세션에서 일어난 일과 같다. `less -R` 은 색 코드만 통과시킨다 |
| 꼬리 보관(켜져 있을 때) | 종결 없이 256바이트를 넘는 ESC 는 지금처럼 시퀀스로 보지 않고 그대로 쓴다. 켜져 있을 때는 어차피 그대로 쓰므로 내용이 달라지지 않는다 | `TestLogAnsiToggleKeepsSequencesWhole` |
| 동기화·백업 가져오기로 설정이 바뀜 | `store.patch` 를 거치지 않아 기록 중인 세션에는 적용되지 않는다(새로 기록을 시작하는 세션부터) | D99·D100·D112 와 같은 한계 |
| 번역 | 한국어·영어 외 8개 언어의 문구 3개는 이 작업을 한 세션이 번역했다 | 검수가 필요할 수 있다 |
| 스토어 스크린샷·도움말 | 설정 > 연결 화면을 찍은 자료가 있으면 로그 항목이 빠진 모습으로 바뀐다 | docs/scenarios(한·영)는 고쳤다. 스크린샷은 확인하지 않았다 |

되돌리기: §B·§C·§D 의 `-` 줄로 되돌리고 새 파일 2개를 지운다. 저장 파일에 남은 `sessionLogAnsi` 는 읽는 코드가 없으면 무시된다.

## J. 체크리스트

- [x] `log.go`: 필드 `bAnsi`, `cleanLocked`, `SetAnsi`, `Write`·`Close` 수정
- [x] `types.go`·`session.go`·`control.go`·`manager.go`: 전달과 적용
- [x] `app.go`(바뀌면 열린 세션에 적용)·`channels_session.go`(세션을 열 때 전달)·`store.go`(기본값 false)
- [x] 프런트: 타입·기본값, 구역 id `log`, `SettingsLog.tsx` 새로, `SettingsConnection.tsx` 에서 로그 항목 빼기, `Settings.tsx` 구역 목록
- [x] 문구 3개(10개 언어), 계약 고정 파일 2개
- [x] Go 테스트 4개, 프런트 테스트 6개 추가. `go test ./...`·`tsc`·`vitest`(291개) 통과, 빌드
- [x] 명세 ui/settings.md(구역 9개, §4a)·behavior/session.md §6·data/store.md, 시나리오 ko·en, decisions D120
- [ ] 수동 검증 1~10
