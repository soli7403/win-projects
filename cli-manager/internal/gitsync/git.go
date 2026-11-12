package gitsync

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// 인증은 이 PC 에 설치된 git 과 그 설정(SSH 키, 자격 증명 도우미)을 그대로 쓴다. 앱은 git 자격 증명을 저장하지 않는다.

func gitRun(szDir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = szDir
	// 입력을 기다리는 프롬프트로 멈추지 않게 한다 (자격 증명 도우미/SSH 에이전트는 그대로 동작)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0", "GIT_ASKPASS=", "GCM_INTERACTIVE=never", "LC_ALL=C")
	hideWindow(cmd)
	var so, se bytes.Buffer
	cmd.Stdout, cmd.Stderr = &so, &se
	err := cmd.Run()
	if err != nil {
		msg := strings.TrimSpace(se.String())
		if msg == "" {
			msg = err.Error()
		}
		return so.String(), fmt.Errorf("git %s: %s", args[0], lastLines(msg, 3))
	}
	return strings.TrimSpace(so.String()), nil
}

func lastLines(s string, n int) string {
	l := strings.Split(strings.TrimSpace(s), "\n")
	if len(l) > n {
		l = l[len(l)-n:]
	}
	return strings.Join(l, " / ")
}

// GitAvailable 은 git 실행 파일이 있는지.
func GitAvailable() bool {
	_, err := exec.LookPath("git")
	return err == nil
}

// ensureRepo 는 작업용 클론을 준비하고 원격 최신 상태로 맞춘다. 돌려주는 값은 사용할 브랜치 이름.
func ensureRepo(szDir, szURL string) (string, error) {
	if _, err := os.Stat(filepath.Join(szDir, ".git")); err != nil {
		if err := os.MkdirAll(filepath.Dir(szDir), 0o700); err != nil {
			return "", err
		}
		if _, err := gitRun(filepath.Dir(szDir), "clone", szURL, szDir); err != nil {
			return "", err
		}
	} else if cur, err := gitRun(szDir, "remote", "get-url", "origin"); err != nil || cur != szURL {
		if _, err := gitRun(szDir, "remote", "set-url", "origin", szURL); err != nil {
			return "", err
		}
	}
	if _, err := gitRun(szDir, "fetch", "origin"); err != nil {
		// 빈 저장소는 fetch 가 성공하지만, 네트워크/인증 실패는 여기서 드러난다
		return "", err
	}
	for _, br := range []string{"main", "master"} {
		if _, err := gitRun(szDir, "rev-parse", "--verify", "-q", "refs/remotes/origin/"+br); err == nil {
			if _, err := gitRun(szDir, "checkout", "-q", "-B", br, "origin/"+br); err != nil {
				return "", err
			}
			if _, err := gitRun(szDir, "reset", "-q", "--hard", "origin/"+br); err != nil {
				return "", err
			}
			return br, nil
		}
	}
	// 빈 원격: 새 main 브랜치에서 시작
	if _, err := gitRun(szDir, "checkout", "-q", "-B", "main"); err != nil {
		return "", err
	}
	return "main", nil
}

// commitPush 는 동기화 파일만 커밋하고 푸시한다.
func commitPush(szDir, szBranch, szFile string) error {
	if _, err := gitRun(szDir, "add", szFile); err != nil {
		return err
	}
	if out, _ := gitRun(szDir, "status", "--porcelain", "--", szFile); out == "" {
		return nil // 바뀐 것 없음
	}
	if _, err := gitRun(szDir, "-c", "user.name=TermGo", "-c", "user.email=termgo@localhost", "commit", "-q", "-m", "TermGo sync"); err != nil {
		return err
	}
	_, err := gitRun(szDir, "push", "-q", "origin", "HEAD:refs/heads/"+szBranch)
	return err
}

var errNoFile = errors.New("동기화 파일 없음")
