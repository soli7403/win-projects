package sshconn

import (
	"dariterm/internal/i18n"
	"errors"
	"strings"

	"golang.org/x/crypto/ssh"
)

// askedSecret 은 사용자가 입력한 비밀번호다. 인증에 성공했을 때만 저장 여부를 따진다 (protocols.md §2.2).
type askedSecret struct {
	Password string
	Save     bool
	Asked    bool
}

// errCancelled 는 사용자가 묻는 창을 취소했을 때의 오류다.
var errCancelled = i18n.NewError("be.cancelled")

// authMethodsFor 는 protocols.md §2.2 의 순서로 인증 방법 목록을 만든다.
// 순서: 개인 키 → 에이전트(agent·key 일 때) → 저장된 비밀번호 → 키보드-인터랙티브 → 비밀번호 직접 입력(최대 4번).
// agentSigners 가 nil 이면 에이전트를 건너뛴다. notice 는 터미널 안내 줄을 보낸다.
func authMethodsFor(tgt *Target, username string, prompt Prompter, agentSigners func() ([]ssh.Signer, error), asked *askedSecret, notice func(string)) []ssh.AuthMethod {
	var methods []ssh.AuthMethod
	if len(tgt.PrivateKey) > 0 {
		methods = append(methods, ssh.PublicKeysCallback(keyLoader(tgt, prompt, notice)))
	}
	if agentSigners != nil && (tgt.AuthType == "agent" || tgt.AuthType == "key") {
		methods = append(methods, ssh.PublicKeysCallback(agentSigners))
	}
	if tgt.Password != "" {
		methods = append(methods, ssh.Password(tgt.Password))
	}
	methods = append(methods, ssh.KeyboardInteractive(keyboardHandler(tgt, username, prompt, asked)))
	// 비밀번호 직접 입력(최대 4번)은 Connect 에서 연결을 새로 맺어 가며 처리한다.
	// x/crypto/ssh 는 같은 이름의 인증 방법을 한 연결에서 두 번 쓰지 않기 때문이다.
	return methods
}

// askPassword 는 비밀번호 요청 창을 띄운다. 인증 성공 시 저장할지는 asked 에 남겨 두었다가 호출자가 판단한다.
func askPassword(tgt *Target, username string, prompt Prompter, asked *askedSecret, bFailed bool) (string, error) {
	szPrompt := i18n.T("be.ssh.passwordPrompt", "user", username, "host", tgt.Host)
	szPassword, bSave, bOk := prompt.Password(tgt.Title, szPrompt, username, bFailed)
	if !bOk {
		return "", errCancelled
	}
	asked.Password = szPassword
	asked.Save = bSave && tgt.AllowSave
	asked.Asked = true
	return szPassword, nil
}

// keyboardHandler 는 키보드-인터랙티브 질문을 처리한다 (protocols.md §2.2 4번).
// 질문 하나·에코 없음·"pass" 포함이면 비밀번호 요청으로 처리한다.
func keyboardHandler(tgt *Target, username string, prompt Prompter, asked *askedSecret) ssh.KeyboardInteractiveChallenge {
	return func(szUser string, szInstruction string, slQuestions []string, slEchos []bool) ([]string, error) {
		if len(slQuestions) == 1 && !slEchos[0] && strings.Contains(strings.ToLower(slQuestions[0]), "pass") {
			szPassword, errAsk := askPassword(tgt, username, prompt, asked, false)
			if errAsk != nil {
				return nil, errAsk
			}
			return []string{szPassword}, nil
		}
		slPrompts := make([]KeyboardPrompt, 0, len(slQuestions))
		for iIndex, szQuestion := range slQuestions {
			slPrompts = append(slPrompts, KeyboardPrompt{Prompt: szQuestion, Echo: slEchos[iIndex]})
		}
		slAnswers, bOk := prompt.Keyboard(tgt.Title, "", szInstruction, slPrompts)
		if !bOk {
			return nil, errCancelled
		}
		return slAnswers, nil
	}
}

// keyLoader 는 개인 키를 읽는다. 암호가 걸려 있으면 저장된 암호 → 요청(최대 3번) 순으로 푼다 (protocols.md §2.2 1번).
func keyLoader(tgt *Target, prompt Prompter, notice func(string)) func() ([]ssh.Signer, error) {
	return func() ([]ssh.Signer, error) {
		signer, errParse := ssh.ParsePrivateKey(tgt.PrivateKey)
		if errParse == nil {
			return []ssh.Signer{signer}, nil
		}
		var errMissing *ssh.PassphraseMissingError
		if !errors.As(errParse, &errMissing) {
			notice(i18n.T("be.ssh.keyParse", "error", errParse))
			return nil, errParse
		}
		szPass := tgt.Passphrase
		errLast := errParse
		for iTry := 0; iTry < G_iPassphraseMax; iTry++ {
			if szPass == "" {
				szAsk, bOk := prompt.Passphrase(tgt.Title)
				if !bOk {
					return nil, errCancelled
				}
				szPass = szAsk
			}
			signer, errWith := ssh.ParsePrivateKeyWithPassphrase(tgt.PrivateKey, []byte(szPass))
			if errWith == nil {
				tgt.Passphrase = szPass // 세션 사본용 (rememberWorking 참고)
				return []ssh.Signer{signer}, nil
			}
			errLast = errWith
			szPass = ""
		}
		notice(i18n.T("be.ssh.keyParse", "error", errLast))
		return nil, errLast
	}
}
