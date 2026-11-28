# 화면 보호기를 Windows 잠금으로 오인하지 않게 (2026-11-17 계획서의 안 C)

**작성**: 2026-11-27 · **상태**: 코드·자동 테스트(이 PC 의 실제 Windows 세션에서 읽기 확인)·빌드 완료(D154). 화면 보호기를 띄운 상태의 확인은 수동 검증으로 남음(§H) · **대상**: cli-manager-cr(DariTerm) · **Known issue**: 없음
**관련**: [2026-11-17-reconnect-password-after-screensaver.md](./2026-11-17-reconnect-password-after-screensaver.md) §3.4·안 C(§4-1 실측 전이라 보류였다)

---

## 0. 한 줄 요약

자동 잠금이 "Windows 가 잠겼는지"를 **입력 데스크톱을 열 수 있는지**로 추정했다. 화면 보호기는 별도 데스크톱에서 돌아, "다시 시작할 때 로그온 화면 표시"를 끈 화면 보호기에서도 접근 거부가 나 **앱이 잠길 수 있었다**. 이제 **Windows 세션의 잠금 상태**(`WTSQuerySessionInformationW` 의 `WTSSessionInfoEx` → `SessionFlags`)를 읽어 판정한다. 읽지 못할 때만 예전 추정을 쓴다.

## 1. 결정

| 항목 | 결정 | 까닭 |
|---|---|---|
| 판정 | `SessionFlags == WTS_SESSIONSTATE_LOCK(0)` 이면 잠김, `WTS_SESSIONSTATE_UNLOCK(1)` 이면 풀림 | 화면 보호기와 무관한 실제 잠금 상태다 |
| 모를 때 | `WTS_SESSIONSTATE_UNKNOWN`·호출 실패·DLL 없음 → 예전 추정(입력 데스크톱 접근 거부) | 서비스·일부 원격 환경 |
| 검사 주기 | 그대로 20초 | 구조를 바꾸지 않는다 |
| Windows 7 | 값이 거꾸로 나오는 알려진 문제가 있으나 대상(Windows 10/11)이 아니다 | |
| 대안 | `WTSRegisterSessionNotification` 으로 알림 받기 — 기각(창 메시지 루프에 걸어야 한다. 20초 검사로 충분) | |
| 실측 전제(2026-11-17 R7) | 사용자 지시("1~10번 모두")로 실측 없이 넣는다. 실제 잠금은 예전과 같이 잡고, 화면 보호기만 다르게 보므로 해가 없다 | |

함께 고친 것: 예전 코드가 `OpenInputDesktop` 의 데스크톱 핸들을 `CloseHandle` 로 닫았다 — 데스크톱 핸들은 `CloseDesktop` 으로 닫아야 한다.

## 2. 구현 — `internal/app/lifecycle_windows.go`

```go
var (
	…
	g_procCloseDesktop    = syscall.NewLazyDLL("user32.dll").NewProc("CloseDesktop")
	g_procWtsQuerySession = syscall.NewLazyDLL("wtsapi32.dll").NewProc("WTSQuerySessionInformationW")
	g_procWtsFreeMemory   = syscall.NewLazyDLL("wtsapi32.dll").NewProc("WTSFreeMemory")
)

// WTSQuerySessionInformationW 로 세션 잠금 상태를 읽는 데 쓰는 값 (wtsapi32.h).
const (
	g_cuWtsCurrentSession   = 0xFFFFFFFF // WTS_CURRENT_SESSION: 이 프로세스의 세션
	g_ciWtsSessionInfoEx    = 25         // WTSSessionInfoEx
	g_ciWtsInfoExLevel1     = 1          // WTSINFOEXW.Level
	g_ciWtsSessionStateLock = 0          // WTS_SESSIONSTATE_LOCK
	g_ciWtsSessionStateOpen = 1          // WTS_SESSIONSTATE_UNLOCK
	// WTSINFOEXW 는 Level(DWORD) 뒤에 8바이트 정렬된 공용체가 온다. 그 안의 WTSINFOEX_LEVEL1_W 는
	// SessionId(4) · SessionState(4) · SessionFlags(4) 순이라 SessionFlags 는 맨 앞에서 16바이트 자리다.
	g_ciWtsSessionFlagsOffset = 16
)

// desktopLockedOS 는 Windows 세션이 잠겼는지 본다 (lifecycle.md §2, D154).
func desktopLockedOS() bool {
	if bLocked, bKnown := sessionLockStateOS(); bKnown {
		return bLocked
	}
	return inputDesktopDeniedOS()
}

// sessionLockStateOS 는 이 프로세스가 도는 Windows 세션의 잠금 상태를 읽는다. 알 수 없으면 bKnown=false.
func sessionLockStateOS() (bLocked bool, bKnown bool) {
	if g_procWtsQuerySession.Find() != nil || g_procWtsFreeMemory.Find() != nil {
		return false, false
	}
	var pBuf *byte
	var uBytes uint32
	rRet, _, _ := g_procWtsQuerySession.Call(0, uintptr(g_cuWtsCurrentSession), uintptr(g_ciWtsSessionInfoEx),
		uintptr(unsafe.Pointer(&pBuf)), uintptr(unsafe.Pointer(&uBytes)))
	if rRet == 0 || pBuf == nil {
		return false, false
	}
	defer g_procWtsFreeMemory.Call(uintptr(unsafe.Pointer(pBuf)))
	if uBytes < g_ciWtsSessionFlagsOffset+4 {
		return false, false
	}
	bInfo := unsafe.Slice(pBuf, uBytes)
	if binary.LittleEndian.Uint32(bInfo[0:4]) != g_ciWtsInfoExLevel1 {
		return false, false
	}
	switch int32(binary.LittleEndian.Uint32(bInfo[g_ciWtsSessionFlagsOffset : g_ciWtsSessionFlagsOffset+4])) {
	case g_ciWtsSessionStateLock:
		return true, true
	case g_ciWtsSessionStateOpen:
		return false, true
	default: // WTS_SESSIONSTATE_UNKNOWN
		return false, false
	}
}

// inputDesktopDeniedOS 는 예전 판정이다: 입력 데스크톱을 열 수 없고 접근 거부이면 잠긴 것으로 본다.
func inputDesktopDeniedOS() bool { … OpenInputDesktop, 열리면 CloseDesktop … }
```

구조체 자리 계산(공용체에 `LARGE_INTEGER` 가 있어 8바이트 정렬):

| 자리 | 필드 |
|---|---|
| 0 | `Level` (DWORD) |
| 4 | (정렬 채움) |
| 8 | `SessionId` |
| 12 | `SessionState` (WTS_CONNECTSTATE_CLASS) |
| **16** | **`SessionFlags`** |

`unsafe.Pointer(uintptr)` 변환을 쓰지 않으려고(go vet `unsafeptr`) 버퍼를 `*byte` 로 받아 `unsafe.Slice` 로 읽는다.

비 Windows(`lifecycle_other.go`)는 그대로(늘 잠기지 않음).

## 3. 명세

spec/behavior/lifecycle.md §2 "Windows 잠금 판정" 항목.

## 4. 테스트 — `internal/app/lifecycle_windows_test.go`

```go
// TestSessionLockStateReadable 은 Windows 세션 잠금 상태를 실제로 읽을 수 있고, 테스트를 돌리는 지금(풀린 세션)은
// "잠기지 않음"으로 나오는지 본다 (D154). 대화형 세션이 아닌 곳(서비스·일부 CI)에서는 상태를 알 수 없어 건너뛴다.
func TestSessionLockStateReadable(t *testing.T) {
	bLocked, bKnown := sessionLockStateOS()
	if !bKnown {
		t.Skip("이 환경에서는 Windows 세션 잠금 상태를 알 수 없다 (대화형 세션이 아님)")
	}
	if bLocked {
		t.Fatalf("풀린 세션에서 잠김으로 읽었다 — SessionFlags 자리(%d)가 틀렸을 수 있다", g_ciWtsSessionFlagsOffset)
	}
	if desktopLockedOS() {
		t.Fatal("desktopLockedOS 가 풀린 세션을 잠김으로 본다")
	}
}
```

2026-11-27 이 PC(Windows 11 Pro 10.0.26300)에서 **건너뛰지 않고 PASS** — 상태를 실제로 읽었고 "풀림"이었다. 자리가 틀렸다면 Level 검사나 값(0/1 외)에서 걸러져 건너뛰거나, 다른 값이 나와 실패했을 것이다.

## 5. 검증 결과 (2026-11-27)

`go vet`, `go test -race ./...`, `wails build` 성공. 잠긴 상태의 값(0)은 자동 시험으로 만들 수 없어 §H.

## H. 수동 검증

| # | 하는 일 | 기대 | 결과 |
|---|---|---|---|
| 1 | 설정 > 보안 > 자동 잠금 15분. Windows 화면 보호기 1분, "다시 시작할 때 로그온 화면 표시" **끔** | | |
| 2 | 1분 넘게 손대지 않아 화면 보호기가 뜨게 → 마우스로 해제(2분 안) | **앱이 잠기지 않았다**(전에는 잠길 수 있었다) | |
| 3 | 같은 설정에서 "로그온 화면 표시" **켬** → 화면 보호기 뒤 Windows 암호로 해제 | 앱이 잠겨 있다(실제 Windows 잠금) | |
| 4 | Win+L → 다시 로그인 | 앱이 잠겨 있다 | |
| 5 | 자동 잠금 시간만큼 입력 없음(화면 보호기 꺼 둠) | 유휴 시간으로 잠긴다(전과 같음) | |

## 체크리스트

- [x] 세션 상태 판정 · [x] 예전 추정은 대비책으로 · [x] CloseDesktop 수정 · [x] 명세 · [x] 실제 PC 시험 · [ ] §H(특히 2)
- [x] 2026-11-17 계획서의 체크리스트 "안 C 구현"을 이 문서로 표시
