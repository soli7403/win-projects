# KI-001 — 같은 별칭 세션을 같은 분에 열면 세션 로그가 한 파일에 섞임 — 원인·수정 계획·코드·테스트

**작성**: 2026-11-16 · **상태**: 완료(2026-11-16, D84~D90) · **대상**: cli-manager-cr(DariTerm) · **Known issue**: [KI-001](../known-issues.md)
**관련**: [spec/behavior/session.md](../../spec/behavior/session.md) §6, [internal/session/log.go](../../internal/session/log.go), 공통 규칙 [../../../CLAUDE.md](../../../CLAUDE.md) §6 Fix Protocol

> 이 문서 하나로 **무엇이 문제인지 → 왜 그런지 → 어떻게 고치는지(코드 전체) → 어떻게 검증하는지**를 알 수 있게 쓴다.
> 규모가 작아(파일 2개 + 명세 1곳) WIP 는 만들지 않아도 된다. 중단되면 §7 체크리스트를 WIP 로 옮긴다.

---

## 0. 한 줄 요약

세션 로그 파일 이름이 `{별칭}-{YYYYMMDD-HHmm}.log` 로 **분까지만** 쓰이고, 같은 이름이 있으면 **이어 쓰기** 로 열린다. 그래서 같은 별칭(또는 같은 호스트 즉석 접속)을 같은 분 안에 두 번 이상 열면 여러 세션의 출력이 한 파일에 묶음 단위로 섞인다.

수정: 이름을 **초 단위**로 하고, 그래도 같은 이름이 있으면 `-2`, `-3` … 을 붙여 **항상 새 파일**을 만든다. 세션 하나 = 파일 하나를 보장한다. 재접속은 지금처럼 같은 파일에 이어 쓴다.

| 전 | 후 |
|---|---|
| `web-20261116-1432.log` (같은 분 두 번째 세션도 여기에 섞임) | `web-20261116-143205.log`, 같은 초 두 번째 세션은 `web-20261116-143205-2.log` |

---

## 1. 원인 분석 (Cause)

### 1.1 코드 경로

```
Session.start()                       internal/session/session.go:51-58
  → OpenLog(dataDir, alias, time.Now())   internal/session/log.go:36
      → LogFileName(alias, now)           log.go:25  "{별칭}-{YYYYMMDD-HHmm}.log"  ← 분까지만
      → os.OpenFile(path, O_APPEND|O_CREATE|O_WRONLY, 0o600)   log.go:41  ← 있으면 이어 씀
Session.writeOutput()                 session.go:73-84  → Logger.Write (세션마다 자기 Logger)
Session.Close()                       control.go  → Logger.Close
```

### 1.2 왜 섞이는가

1. [log.go:25-27](../../internal/session/log.go#L25-L27) `LogFileName` 은 `tNow.Format("20060102-1504")` 로 **분**까지만 쓴다. 같은 별칭을 같은 분에 열면 이름이 같다.
2. [log.go:41](../../internal/session/log.go#L41) `OpenLog` 는 `O_APPEND|O_CREATE` 라 같은 이름이 있으면 **새로 만들지 않고 뒤에 붙인다.** 두 세션이 각자 `*os.File` 을 들고 같은 파일에 쓴다.
3. 각 세션은 출력 묶음(12ms)마다 `Logger.Write` 를 부르므로 두 세션의 묶음이 번갈아 들어간다. 어느 줄이 어느 세션인지 구분할 표시가 없다.
4. 즉석 접속의 별칭은 호스트 이름이라 같은 호스트를 여러 개 열면 항상 이 경우다. 폴더 그리드 열기(여러 서버를 한꺼번에)에서 같은 별칭이 두 개 이상이면 역시 같은 분에 열린다.

### 1.3 명세와 코드의 차이 (같이 정리)

명세 §6 은 "접속(**재접속 포함**)마다 … 열어 이어 쓴다. 연결이 끊기거나 세션을 닫으면 파일을 닫는다" 인데, 코드는 **세션을 열 때 한 번만** 열고([session.go:51](../../internal/session/session.go#L51) `start` 에서만 호출) 자동 재접속 때는 같은 파일을 계속 쓰며 세션을 닫을 때만 닫는다. 이 수정에서는 **명세를 코드 쪽으로 맞춘다** (한 세션 = 한 파일, 재접속 전후가 한 파일에 이어짐). 이유: 사용자가 보기 편하고, 번호 접미 규칙과 충돌하지 않으며, 코드 변경이 없다.

---

## 2. 설계 결정

| 후보 | 장단점 | 결론 |
|---|---|---|
| **A. 초 단위 이름 + 중복 시 `-2`, `-3` 접미, `O_EXCL` 로 새 파일만 생성** | 이름이 사람이 읽기 쉬운 채로 남고 세션당 파일 하나가 보장된다. 변경은 함수 두 개 | **채택** |
| B. 세션 id 조각(예: 앞 8자)을 이름에 붙임 | 충돌은 없지만 이름이 지저분하고 정렬·찾기가 불편 | 기각 |
| C. 분 단위 그대로 두고 줄마다 세션 표시를 붙임 | 로그 내용이 바뀌고, 세션 분리라는 목적에 맞지 않음 | 기각 |

**규칙 (명세에 들어갈 문장)**:

- 파일 이름은 `{표시 이름}-{YYYYMMDD-HHmmss}.log`. 표시 이름의 `\ / : * ? " < > |` 는 `_`.
- 같은 이름의 파일이 이미 있으면 `-2`, `-3`, … 을 `.log` 앞에 붙여 **없는 이름을 찾아 새로 만든다** (`{표시 이름}-{YYYYMMDD-HHmmss}-2.log`). 1000번까지 시도하고 그래도 못 만들면 로그 없이 세션을 연다(지금도 열기 실패는 조용히 무시).
- 세션을 열 때 한 번 만들고, 재접속해도 같은 파일에 이어 쓰며, 세션을 닫을 때 닫는다. 설정을 바꾸면 새 세션부터 적용.

### 2.1 명세 변경 (명세 작성자 반영 — 구현 전에 먼저)

`spec/behavior/session.md` §6 의 첫 두 문장을 아래로 바꾼다:

```
## 6. 세션 로그
- 설정 "터미널 출력 기록"이 켜져 있을 때 **세션을 열 때 한 번** 로그 폴더(`<데이터 폴더>\logs`)에 `{표시 이름}-{YYYYMMDD-HHmmss}.log`(별칭의 `\ / : * ? " < > |` 는 `_` 로 바꿈)를 **새로 만든다**(권한 소유자 전용). 같은 이름이 이미 있으면 `.log` 앞에 `-2`, `-3`, … 을 붙여 없는 이름으로 만든다(세션 하나에 파일 하나). 1000번까지 시도해도 못 만들면 로그 없이 연다.
- 기록 내용: 화면 출력에서 **ANSI 이스케이프 시퀀스를 제거**하고 CR 도 제거한 텍스트. 자동·수동 재접속은 같은 파일에 이어 쓰고, 세션을 닫으면 파일을 닫는다. 설정을 바꾸면 **새 세션부터** 적용.
```

---

## 3. 수정 계획 — `internal/session/log.go`

변경은 이 파일의 `LogFileName` 과 `OpenLog` 두 함수뿐이다. `Logger` 타입·`Write`·`Close`·`StripForLog` 는 그대로.

### 3.1 import 추가

```go
import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)
```

### 3.2 `LogFileName` — 초 단위 + 순번 접미

```go
// before
// LogFileName 은 "{별칭}-{YYYYMMDD-HHmm}.log" 이름을 만든다.
func LogFileName(szAlias string, tNow time.Time) string {
	return g_reFileNameBad.ReplaceAllString(szAlias, "_") + "-" + tNow.Format("20060102-1504") + ".log"
}

// after
// G_ciLogNameMaxTries 는 같은 이름이 있을 때 순번을 붙여 다시 시도하는 최대 횟수다 (session.md §6).
const G_ciLogNameMaxTries = 1000

// LogFileName 은 "{별칭}-{YYYYMMDD-HHmmss}.log" 이름을 만든다 (session.md §6).
// iSeq 가 2 이상이면 ".log" 앞에 "-{iSeq}" 를 붙인다 (같은 초에 연 두 번째·세 번째 세션).
func LogFileName(szAlias string, tNow time.Time, iSeq int) string {
	szBase := g_reFileNameBad.ReplaceAllString(szAlias, "_") + "-" + tNow.Format("20060102-150405")
	if iSeq >= 2 {
		szBase += "-" + strconv.Itoa(iSeq)
	}
	return szBase + ".log"
}
```

### 3.3 `OpenLog` — 새 파일만 만들고, 있으면 순번을 올려 재시도

```go
// before
// OpenLog 는 데이터 폴더의 logs 아래 로그 파일을 연다 (소유자 전용 권한, 이어 쓰기).
func OpenLog(szDataDir string, szAlias string, tNow time.Time) (*Logger, error) {
	szDir := filepath.Join(szDataDir, "logs")
	if errMkdir := os.MkdirAll(szDir, 0o700); errMkdir != nil {
		return nil, errMkdir
	}
	pFile, errOpen := os.OpenFile(filepath.Join(szDir, LogFileName(szAlias, tNow)), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if errOpen != nil {
		return nil, errOpen
	}
	return &Logger{pFile: pFile}, nil
}

// after
// OpenLog 는 데이터 폴더의 logs 아래 이 세션만의 로그 파일을 새로 만든다 (소유자 전용 권한, session.md §6).
// 같은 이름이 이미 있으면 "-2", "-3", … 을 붙여 없는 이름을 찾는다. 세션 하나에 파일 하나를 보장한다.
func OpenLog(szDataDir string, szAlias string, tNow time.Time) (*Logger, error) {
	szDir := filepath.Join(szDataDir, "logs")
	if errMkdir := os.MkdirAll(szDir, 0o700); errMkdir != nil {
		return nil, errMkdir
	}
	var errLast error
	for iSeq := 1; iSeq <= G_ciLogNameMaxTries; iSeq++ {
		szPath := filepath.Join(szDir, LogFileName(szAlias, tNow, iSeq))
		// O_EXCL: 이미 있으면 실패한다 (Windows 는 CREATE_NEW). 두 세션이 동시에 같은 이름을 만들어도 한쪽만 성공한다.
		pFile, errOpen := os.OpenFile(szPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if errOpen == nil {
			return &Logger{pFile: pFile}, nil
		}
		if !errors.Is(errOpen, fs.ErrExist) {
			return nil, errOpen
		}
		errLast = errOpen
	}
	return nil, errLast
}
```

> `O_APPEND` 는 뺀다. 새로 만든 빈 파일에 한 핸들만 쓰므로 필요 없다. `os.OpenFile` 의 `O_EXCL` 은 Windows 에서 `CREATE_NEW` 로 번역되어 원자적으로 "없을 때만 생성" 이 된다 (OS 중립, 공통 규칙 §8).

### 3.4 호출처 `internal/session/session.go` — 변경 없음

[session.go:56](../../internal/session/session.go#L56) `OpenLog(pSession.cfg.DataDir, pSession.cfg.LogAlias, time.Now())` 의 서명은 그대로다. 실패 시 로그 없이 여는 동작(`errLog == nil` 일 때만 등록)도 그대로.

---

## 4. 테스트 — `internal/session/session_test.go`

### 4.1 기존 테스트 수정 (`TestLogStripAndFileName`, [session_test.go:280-298](../../internal/session/session_test.go#L280-L298))

초가 이름에 들어가므로 기대 파일 이름이 바뀐다. `time.Date(2026, 11, 10, 9, 5, 0, …)` 는 09:05:00 이라 `0905` → `090500`.

```go
// before (두 군데)
"a_b_c-20261110-0905.log"
// after
"a_b_c-20261110-090500.log"
```

### 4.2 새 테스트 추가

```go
// 같은 별칭·같은 초에 세 번 열면 파일 세 개가 따로 만들어지고 각자 자기 내용만 담는다 (KI-001, session.md §6).
func TestLogOpensSeparateFilePerSession(t *testing.T) {
	szDir := t.TempDir()
	tNow := time.Date(2026, 11, 16, 14, 32, 5, 0, time.Local)
	slLoggers := make([]*Logger, 0, 3)
	for iIndex := 0; iIndex < 3; iIndex++ {
		pLogger, errOpen := OpenLog(szDir, "web", tNow)
		if errOpen != nil {
			t.Fatalf("%d번째 열기 실패: %v", iIndex+1, errOpen)
		}
		pLogger.Write("session" + strconv.Itoa(iIndex+1) + "\n")
		slLoggers = append(slLoggers, pLogger)
	}
	for _, pLogger := range slLoggers {
		_ = pLogger.Close()
	}
	slWant := []string{
		"web-20261116-143205.log",
		"web-20261116-143205-2.log",
		"web-20261116-143205-3.log",
	}
	for iIndex, szName := range slWant {
		bData, errRead := os.ReadFile(filepath.Join(szDir, "logs", szName))
		if errRead != nil {
			t.Fatalf("%s 가 없음: %v", szName, errRead)
		}
		szWantBody := "session" + strconv.Itoa(iIndex+1) + "\n"
		if string(bData) != szWantBody {
			t.Fatalf("%s 내용이 섞임: %q (기대 %q)", szName, string(bData), szWantBody)
		}
	}
	slAll, _ := filepath.Glob(filepath.Join(szDir, "logs", "web-*.log"))
	if len(slAll) != 3 {
		t.Fatalf("파일이 3개여야 함: %v", slAll)
	}
}

// 이름 규칙: 초까지 쓰고, 순번 1 은 접미 없음, 2 부터 "-N".
func TestLogFileNameSeq(t *testing.T) {
	tNow := time.Date(2026, 11, 16, 14, 32, 5, 0, time.Local)
	if got := LogFileName("srv", tNow, 1); got != "srv-20261116-143205.log" {
		t.Fatalf("순번 1: %q", got)
	}
	if got := LogFileName("srv", tNow, 2); got != "srv-20261116-143205-2.log" {
		t.Fatalf("순번 2: %q", got)
	}
	if got := LogFileName("a/b:c", tNow, 10); got != "a_b_c-20261116-143205-10.log" {
		t.Fatalf("금지 문자·두 자리 순번: %q", got)
	}
}

// 세션을 통해 열어도 세션마다 파일이 하나씩 생긴다 (start → OpenLog 경로).
func TestSessionLogIsPerSession(t *testing.T) {
	szDir := t.TempDir()
	// 기존 테스트의 가짜 연결. 반드시 newFakeTransport() 로 만든다(채널이 nil 이면 Read 가 영원히 막힌다).
	fnDial := func(func(string)) (Transport, error) { return newFakeTransport(), nil }
	cfg := sshConfig(fnDial, false)
	cfg.DataDir = szDir
	cfg.LogEnabled = true
	cfg.LogAlias = "web"
	pFirst := newSession(cfg, func(string, ...any) {})
	pSecond := newSession(cfg, func(string, ...any) {})
	pFirst.start()
	pSecond.start()
	waitUntil(t, func() bool {
		slFiles, _ := filepath.Glob(filepath.Join(szDir, "logs", "web-*.log"))
		return len(slFiles) == 2
	}, time.Second)
	pFirst.Close()
	pSecond.Close()
}
```

> `newFakeTransport`([session_test.go:22](../../internal/session/session_test.go#L22))·`sshConfig`·`waitUntil` 은 기존 `session_test.go` 의 헬퍼다 (2026-11-16 확인). `os`·`strconv` import 가 없으면 추가한다.

### 4.3 실행

```bash
go test -race ./internal/session/ -run 'Log' -v
go test -race ./...
go build ./...
```

### 4.4 수동 검증 (빌드 후 `build/bin/dariterm.exe`)

| # | 절차 | 기대 결과 |
|---|---|---|
| 1 | 설정 → "터미널 출력 기록" 켬. 같은 서버를 1초 안에 탭 3개로 연다(탭 추가 3회 또는 폴더 그리드) | `%APPDATA%\DariTerm\data\logs\` 에 `web-YYYYMMDD-HHmmss.log`, `…-2.log`, `…-3.log` 세 파일 |
| 2 | 각 탭에서 `echo tab1` / `echo tab2` / `echo tab3` | 각 파일에 자기 줄만 있음 |
| 3 | 탭 1 에서 서버 쪽 sshd 재시작(또는 네트워크 끊기) → 자동 재접속 | 새 파일이 생기지 않고 재접속 전후 출력이 같은 파일에 이어짐 |
| 4 | 탭을 닫음 | 파일이 닫히고(다른 프로그램에서 열림), 더 이상 커지지 않음 |
| 5 | 설정을 끄고 새 세션 | 새 파일이 생기지 않음. 이미 열린 세션은 계속 기록 |
| 6 | 즉석 접속으로 같은 호스트 2개 | 호스트 이름 기반 파일 2개 |

---

## 5. 문서·기록

| 파일 | 내용 |
|---|---|
| `spec/behavior/session.md` §6 | §2.1 문구 (명세 작성자) |
| `.claude/known-issues.md` | KI-001 `active → resolved`, History 에 "2026-MM-DD: 초 단위 이름 + 순번 접미로 수정(log.go), 명세 §6 개정" |
| `docs/decisions.md` | **D85** 행: "세션 로그 파일은 세션 하나에 하나: `{별칭}-{YYYYMMDD-HHmmss}.log`, 겹치면 `-2`… 접미, `O_EXCL` 로 새 파일만. 재접속은 같은 파일에 이어 씀(명세를 코드에 맞춤). KI-001. 세션 id 접미(B)·줄마다 세션 표시(C)는 기각" (D84 는 출력 흐름 제어 계획이 쓴다) |

---

## 6. 영향 범위·회귀 위험

| 영역 | 위험 | 대비 |
|---|---|---|
| 기존 로그 파일 | 영향 없음. 옛 이름(분 단위)의 파일은 그대로 남고 새 파일만 새 규칙 | — |
| 사용자가 파일 이름 규칙에 의존 | 분 단위 이름으로 찾던 스크립트가 있으면 패턴을 `-HHmmss` 로 바꿔야 한다 | 명세·릴리스 노트에 적는다 |
| `O_EXCL` 미지원 파일 시스템 | 네트워크 드라이브 일부에서 `CREATE_NEW` 가 비원자적일 수 있음 | 데이터 폴더는 `%APPDATA%` 로컬. 같은 초 같은 이름은 재시도 루프가 받는다 |
| 1000번 재시도 | 같은 초에 같은 별칭 1000개는 현실에 없음. 넘으면 로그 없이 세션을 연다(기존 실패 처리와 동일) | 테스트 §4.2 |
| 명세 §6 "재접속마다 연다" | 명세를 코드에 맞춰 바꿈. 동작 변화 없음 | §2.1 |

---

## 7. 착수 체크리스트

- [ ] 1. 명세 §6 반영 (§2.1)
- [ ] 2. `log.go`: import, `G_ciLogNameMaxTries`, `LogFileName(…, iSeq)`, `OpenLog` 재시도 루프 (§3)
- [ ] 3. `session_test.go`: 기존 기대값 `090500` 으로 수정, 새 테스트 3개 추가 (§4)
- [ ] 4. `go build ./... && go test -race ./...` 통과
- [ ] 5. `wails build` → 수동 검증 §4.4 의 1·2·3 필수
- [ ] 6. known-issues KI-001 `resolved`, decisions D85
