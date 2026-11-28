package session

import (
	"dariterm/internal/i18n"
	"errors"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

// 로그인 스크립트 (session.md §5.1, D139): 접속 직후 서버 출력에서 "기다릴 글자"가 나오면 "보낼 글자"를 입력한다.
// 단계는 위에서 아래로 한 번씩만 실행한다. 접속(재접속 포함)마다 처음부터 다시 한다.

// LoginStep 은 로그인 스크립트의 한 단계다 (data/store.md §3 loginScript).
type LoginStep struct {
	Expect string // 기다릴 글자. 비면 기다리지 않고 바로 보낸다
	Send   string // 보낼 글자. 뒤에 Enter(CR)를 붙여 보낸다
	// SecretId 가 있으면 Send 대신 보관함 항목(이 id)의 비밀번호를 보낸다 (D148).
	SecretId string
}

// ErrSecretLocked 는 보관함이 잠겨 있어 로그인 스크립트의 비밀값을 읽지 못했다는 뜻이다 (SecretResolver 가 돌려준다).
var ErrSecretLocked = errors.New("login script: vault locked")

// SecretResolver 는 보관함 항목 id 의 비밀번호를 돌려준다 (앱이 구현한다). 볼트가 잠겨 있고 세션 사본도 없으면 오류다.
type SecretResolver func(szId string) (string, error)

// G_ciLoginStepMax 는 로그인 스크립트의 최대 단계 수다. 넘는 단계는 버린다.
const G_ciLoginStepMax = 20

// G_durLoginStepTimeout 은 한 단계가 기다리는 최대 시간이다. 넘으면 스크립트를 멈춘다.
// 테스트에서 줄일 수 있게 변수로 둔다.
var G_durLoginStepTimeout = 20 * time.Second

// G_cszLoginPassword 는 보낼 글자에 쓰는 자리표시자다: 이 서버에 로그인할 때 쓴 비밀번호로 바뀐다.
const G_cszLoginPassword = "{{password}}"

// g_ciLoginTailMax 는 기다릴 글자를 찾는 최근 출력의 최대 길이(바이트)다.
const g_ciLoginTailMax = 4096

// SecretProvider 는 로그인에 쓴 비밀번호를 알려 줄 수 있는 Transport 다 (SSH).
// 로그인 스크립트의 {{password}} 를 채우는 데만 쓴다. 없으면 빈 글자를 돌려준다.
type SecretProvider interface {
	LoginSecret() string
}

// TrimLoginSteps 는 저장된 단계에서 쓸 수 있는 것만 남긴다: 기다릴 글자·보낼 글자·보관함 값이 모두 빈 단계는 빼고,
// G_ciLoginStepMax 개를 넘는 단계는 버린다.
func TrimLoginSteps(slSteps []LoginStep) []LoginStep {
	slOut := make([]LoginStep, 0, len(slSteps))
	for _, step := range slSteps {
		if step.Expect == "" && step.Send == "" && step.SecretId == "" {
			continue
		}
		if len(slOut) >= G_ciLoginStepMax {
			break
		}
		slOut = append(slOut, step)
	}
	return slOut
}

// loginRunner 는 접속 하나에서 로그인 스크립트의 진행 상태다. 접속마다 새로 만든다.
type loginRunner struct {
	mtxData sync.Mutex
	slSteps []LoginStep
	iStep   int    // 다음에 실행할 단계
	szCarry string // 조각 끝의 미완성 ANSI 시퀀스 (다음 조각과 이어 본다)
	szTail  string // ANSI 를 지운 최근 출력 (소문자). 맞은 자리까지는 잘라 낸다
	bDone   bool   // 끝났거나 멈췄다
}

func newLoginRunner(slSteps []LoginStep) *loginRunner {
	return &loginRunner{slSteps: slSteps}
}

// feed 는 feedSteps 와 같고 맞은 단계의 "보낼 글자"만 돌려준다 (시험용).
func (pRunner *loginRunner) feed(szText string) (slSends []string, bFinished bool) {
	slSteps, bFinished := pRunner.feedSteps(szText)
	for _, step := range slSteps {
		slSends = append(slSends, step.Send)
	}
	return slSends, bFinished
}

// feedSteps 는 출력 조각을 받아, 이번에 조건이 맞은 단계들을 순서대로 돌려준다.
// bFinished 는 이 호출로 마지막 단계까지 끝났다는 뜻이다. 대소문자는 구분하지 않는다.
// 한 번 맞은 글자는 다음 단계의 조건에 다시 쓰지 않는다(같은 프롬프트가 두 단계를 한꺼번에 넘기지 않게).
func (pRunner *loginRunner) feedSteps(szText string) (slMatched []LoginStep, bFinished bool) {
	pRunner.mtxData.Lock()
	defer pRunner.mtxData.Unlock()
	if pRunner.bDone {
		return nil, false
	}
	szBody, szCarry := splitAnsiTail(pRunner.szCarry + szText)
	pRunner.szCarry = szCarry
	pRunner.szTail += strings.ToLower(g_reAnsi.ReplaceAllString(szBody, ""))
	if iOver := len(pRunner.szTail) - g_ciLoginTailMax; iOver > 0 {
		for iOver < len(pRunner.szTail) && !utf8.RuneStart(pRunner.szTail[iOver]) {
			iOver++
		}
		pRunner.szTail = pRunner.szTail[iOver:]
	}
	for pRunner.iStep < len(pRunner.slSteps) {
		step := pRunner.slSteps[pRunner.iStep]
		if step.Expect != "" {
			szNeedle := strings.ToLower(step.Expect)
			iAt := strings.Index(pRunner.szTail, szNeedle)
			if iAt < 0 {
				break
			}
			pRunner.szTail = pRunner.szTail[iAt+len(szNeedle):]
		}
		slMatched = append(slMatched, step)
		pRunner.iStep++
	}
	if pRunner.iStep >= len(pRunner.slSteps) {
		pRunner.bDone = true
		return slMatched, true
	}
	return slMatched, false
}

// stop 은 스크립트를 멈춘다. 멈추기 전에 진행 중이었으면 기다리던 단계의 "기다릴 글자"와 true 를 돌려준다.
func (pRunner *loginRunner) stop() (szWaiting string, bWasActive bool) {
	pRunner.mtxData.Lock()
	defer pRunner.mtxData.Unlock()
	if pRunner.bDone {
		return "", false
	}
	pRunner.bDone = true
	if pRunner.iStep < len(pRunner.slSteps) {
		szWaiting = pRunner.slSteps[pRunner.iStep].Expect
	}
	return szWaiting, true
}

// stepIndex 는 다음에 실행할 단계 번호와 끝났는지를 돌려준다.
func (pRunner *loginRunner) stepIndex() (int, bool) {
	pRunner.mtxData.Lock()
	defer pRunner.mtxData.Unlock()
	return pRunner.iStep, pRunner.bDone
}

// startLogin 은 접속 직후의 자동 입력을 시작한다. 로그인 스크립트가 있으면 그것을 먼저 끝까지 실행한 뒤에
// 접속 후 입력(tmux 한 줄 또는 자동 실행 명령)을 보내고, 없으면 접속 후 입력만 예약한다 (session.md §5·§5.1).
func (pSession *Session) startLogin(conn Transport, iGen int) {
	if len(pSession.cfg.LoginScript) == 0 {
		pSession.scheduleStartup(iGen)
		return
	}
	pRunner := newLoginRunner(pSession.cfg.LoginScript)
	pSession.mtxData.Lock()
	if iGen != pSession.gen {
		pSession.mtxData.Unlock()
		return
	}
	pSession.login = pRunner
	pSession.mtxData.Unlock()
	pSession.armLoginTimeout(pRunner, iGen)
	// 첫 단계의 기다릴 글자가 비어 있으면 출력 없이도 바로 보낸다.
	pSession.feedLogin(conn, "", iGen)
}

// feedLogin 은 읽기 루프가 화면 출력 조각마다 부른다. 조건이 맞은 단계의 글자를 서버로 보낸다.
func (pSession *Session) feedLogin(conn Transport, szText string, iGen int) {
	pSession.mtxData.Lock()
	pRunner := pSession.login
	pSession.mtxData.Unlock()
	if pRunner == nil {
		return
	}
	slSteps, bFinished := pRunner.feedSteps(szText)
	for _, step := range slSteps {
		szSend := step.Send
		if step.SecretId != "" {
			// 보관함 항목의 비밀번호를 보낸다 (D148). 읽지 못하면(잠김·지워짐) 빈 글자를 보내지 않고 멈춘다.
			szSecret := ""
			var errSecret error
			if pSession.cfg.ResolveSecret != nil {
				szSecret, errSecret = pSession.cfg.ResolveSecret(step.SecretId)
			}
			if szSecret == "" {
				pRunner.stop()
				pSession.clearLogin(pRunner)
				szKey := "be.login.secretMissing"
				if errors.Is(errSecret, ErrSecretLocked) {
					szKey = "be.login.secretLocked"
				}
				pSession.writeOutput("\r\n" + notice(G_cszColorWarn, i18n.T(szKey)))
				return
			}
			szSend = szSecret
		} else if strings.Contains(szSend, G_cszLoginPassword) {
			szSecret := ""
			if provider, bOk := conn.(SecretProvider); bOk {
				szSecret = provider.LoginSecret()
			}
			if szSecret == "" {
				// 넣을 비밀번호가 없다: 빈 비밀번호를 보내 계정이 잠기는 일이 없게 여기서 멈춘다.
				pRunner.stop()
				pSession.clearLogin(pRunner)
				pSession.writeOutput("\r\n" + notice(G_cszColorWarn, i18n.T("be.login.noPassword")))
				return
			}
			szSend = strings.ReplaceAll(szSend, G_cszLoginPassword, szSecret)
		}
		pSession.mtxData.Lock()
		bCurrent := iGen == pSession.gen && pSession.transport != nil
		pSession.mtxData.Unlock()
		if !bCurrent {
			return
		}
		_ = pSession.send(conn, Encode(pSession.cfg.Encoding, szSend+"\r"))
	}
	if bFinished {
		pSession.clearLogin(pRunner)
		pSession.scheduleStartup(iGen)
		return
	}
	if len(slSteps) > 0 {
		pSession.armLoginTimeout(pRunner, iGen) // 다음 단계의 시한을 새로 잰다
	}
}

// armLoginTimeout 은 지금 단계의 시한을 건다. 시한 안에 다음 단계로 넘어가지 못하면 스크립트를 멈추고 알린다.
func (pSession *Session) armLoginTimeout(pRunner *loginRunner, iGen int) {
	iStep, bDone := pRunner.stepIndex()
	if bDone {
		return
	}
	durTimeout := G_durLoginStepTimeout
	timer := time.AfterFunc(durTimeout, func() {
		if iNow, _ := pRunner.stepIndex(); iNow != iStep {
			return // 그사이 다음 단계로 넘어갔다
		}
		szWaiting, bWasActive := pRunner.stop()
		if !bWasActive || !pSession.isCurrent(iGen) {
			return
		}
		pSession.clearLogin(pRunner)
		pSession.writeOutput("\r\n" + notice(G_cszColorWarn, i18n.T("be.login.timeout", "text", szWaiting, "sec", int(durTimeout/time.Second))))
	})
	pSession.mtxData.Lock()
	if pSession.timerLogin != nil {
		pSession.timerLogin.Stop()
	}
	pSession.timerLogin = timer
	pSession.mtxData.Unlock()
}

// clearLogin 은 끝났거나 멈춘 스크립트를 세션에서 뗀다 (다른 접속의 스크립트로 바뀌었으면 건드리지 않는다).
func (pSession *Session) clearLogin(pRunner *loginRunner) {
	pSession.mtxData.Lock()
	defer pSession.mtxData.Unlock()
	if pSession.login != pRunner {
		return
	}
	pSession.login = nil
	if pSession.timerLogin != nil {
		pSession.timerLogin.Stop()
		pSession.timerLogin = nil
	}
}

// cancelLoginOnInput 은 사용자가 직접 입력하면 진행 중인 로그인 스크립트를 멈춘다.
// 프롬프트가 예상과 달라 사용자가 손으로 넘기는 동안 스크립트가 뒤늦게 글자를 보내는 일을 막는다.
// ESC 로 시작하는 입력은 세지 않는다: 터미널이 서버의 질의에 스스로 답하는 응답(커서 위치 등)이 그 모양이다.
func (pSession *Session) cancelLoginOnInput(szText string) {
	if szText == "" || strings.HasPrefix(szText, "\x1b") {
		return
	}
	pSession.mtxData.Lock()
	pRunner := pSession.login
	pSession.mtxData.Unlock()
	if pRunner == nil {
		return
	}
	if _, bWasActive := pRunner.stop(); !bWasActive {
		return
	}
	pSession.clearLogin(pRunner)
	pSession.writeOutput("\r\n" + notice(G_cszColorInfo, i18n.T("be.login.cancelled")))
}
