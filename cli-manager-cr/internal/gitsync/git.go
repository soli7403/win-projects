package gitsync

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"dariterm/internal/procutil"
)

// G_cdurGitCallTimeout 은 git 호출 한 번의 제한 시간이다 (안전장치, 명세에 수치 없음 — 넉넉히 5분).
const G_cdurGitCallTimeout = 5 * time.Minute

// g_reCredentialUrl 은 scheme://사용자:비밀번호@ 형태의 주소를 찾는다 (sync-backup.md §5.4 사전 검사 2).
var g_reCredentialUrl = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9+.\-]*://[^/?#@]*:[^/?#@]*@`)

// HasCredentialInUrl 은 저장소 주소에 비밀번호·토큰이 들어 있는지 본다.
func HasCredentialInUrl(szRepo string) bool {
	return g_reCredentialUrl.MatchString(szRepo)
}

// GitAvailable 은 git 실행 파일이 PATH 에 있는지 본다.
func GitAvailable() bool {
	_, errLook := exec.LookPath("git")
	return errLook == nil
}

// gitRun 은 git 을 szDir 에서 실행한다. 실패하면 "git <명령>: <끝 3줄을 ' / ' 로 이은 내용>" 오류를 돌려준다.
func gitRun(szDir string, slArgs ...string) (string, error) {
	ctxCall, fnCancel := context.WithTimeout(context.Background(), G_cdurGitCallTimeout)
	defer fnCancel()
	pCmd := exec.CommandContext(ctxCall, "git", slArgs...)
	if szDir != "" {
		pCmd.Dir = szDir
	}
	pCmd.Env = append(os.Environ(), procutil.G_cslEnvQuiet...)
	procutil.HideConsole(pCmd)
	var bOut bytes.Buffer
	var bErr bytes.Buffer
	pCmd.Stdout = &bOut
	pCmd.Stderr = &bErr
	errRun := pCmd.Run()
	if errRun != nil {
		szDetail := tailLines(bErr.String(), 3)
		if szDetail == "" {
			szDetail = errRun.Error()
		}
		return "", errors.New(fmt.Sprintf("git %s: %s", slArgs[0], szDetail))
	}
	return bOut.String(), nil
}

// tailLines 은 텍스트의 끝 n 줄을 " / " 로 이어 붙인다 (빈 줄 제외).
func tailLines(szText string, iCount int) string {
	slRaw := strings.Split(strings.TrimSpace(szText), "\n")
	slLines := []string{}
	for _, szLine := range slRaw {
		if szTrim := strings.TrimSpace(szLine); szTrim != "" {
			slLines = append(slLines, szTrim)
		}
	}
	if len(slLines) > iCount {
		slLines = slLines[len(slLines)-iCount:]
	}
	return strings.Join(slLines, " / ")
}
