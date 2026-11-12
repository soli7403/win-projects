package app

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"termgo/internal/gitsync"
	"termgo/internal/ipc"
	"termgo/internal/session"
	"termgo/internal/sftpfs"
)

// g_bUpdateContract 가 켜지면 고정 파일(frontend/src/ipc/contract/*.json)을 현재 응답으로 다시 쓴다.
// 사용: go test ./internal/app -run TestContract -update
var g_bUpdateContract = flag.Bool("update", false, "계약 고정 파일을 현재 응답으로 갱신")

const g_szContractDir = "../../frontend/src/ipc/contract"

// contractNormalize 는 실행마다 달라지는 값(무작위 id, 복구 코드, 시각, 임시 폴더, OS)을 자리표시자로 바꾼다.
func contractNormalize(anyValue any, szKey string) any {
	switch v := anyValue.(type) {
	case map[string]any:
		out := map[string]any{}
		for k, child := range v {
			out[k] = contractNormalize(child, k)
		}
		return out
	case []any:
		out := make([]any, len(v))
		for i, child := range v {
			out[i] = contractNormalize(child, szKey)
		}
		return out
	case string:
		switch szKey {
		case "recoveryCode":
			return "<recovery>"
		case "id", "hostId", "sessionId", "requestId":
			if v != "" {
				return "<id>"
			}
		case "dataDir":
			return "<dir>"
		case "platform":
			return "<os>"
		case "lastSync", "message":
			return v
		}
		return v
	case float64:
		if szKey == "updatedAt" || szKey == "connectedAt" || szKey == "mtime" || szKey == "lastSync" {
			return 0
		}
		return v
	}
	return anyValue
}

// contractCheck 은 응답 JSON 을 정규화해 고정 파일과 비교하거나(기본), -update 면 쓴다.
func contractCheck(t *testing.T, szName string, szRaw string) {
	t.Helper()
	var anyValue any
	if szRaw == "" {
		anyValue = nil
	} else if errJson := json.Unmarshal([]byte(szRaw), &anyValue); errJson != nil {
		t.Fatalf("%s: JSON 해석 실패: %v (%s)", szName, errJson, szRaw)
	}
	bOut, errMarshal := json.MarshalIndent(contractNormalize(anyValue, ""), "", "  ")
	if errMarshal != nil {
		t.Fatal(errMarshal)
	}
	bOut = append(bOut, '\n')
	szPath := filepath.Join(g_szContractDir, szName+".json")
	if *g_bUpdateContract {
		if errMk := os.MkdirAll(g_szContractDir, 0o755); errMk != nil {
			t.Fatal(errMk)
		}
		if errWrite := os.WriteFile(szPath, bOut, 0o644); errWrite != nil {
			t.Fatal(errWrite)
		}
		return
	}
	bWant, errRead := os.ReadFile(szPath)
	if errRead != nil {
		t.Fatalf("%s: 고정 파일이 없습니다 (go test -update 로 만든다): %v", szName, errRead)
	}
	if !bytes.Equal(bytes.TrimSpace(bWant), bytes.TrimSpace(bOut)) {
		t.Fatalf("%s: 응답이 고정 파일과 다릅니다\n--- 고정\n%s\n--- 현재\n%s", szName, bWant, bOut)
	}
}

// contractInvoke 는 채널을 호출하고 결과 문자열을 돌려준다. 오류면 테스트를 멈춘다.
func contractInvoke(t *testing.T, pService *Service, szChannel string, szArgs string) string {
	t.Helper()
	szOut, errInvoke := pService.Router().Invoke(szChannel, szArgs)
	if errInvoke != nil {
		t.Fatalf("%s: 오류 %v", szChannel, errInvoke)
	}
	return szOut
}

// fakeDialerTransport 는 접속이 끝나지 않는 가짜 연결이다 (세션 정보 모양만 본다).
type fakeDialerTransport struct{ chStop chan struct{} }

func (f *fakeDialerTransport) Read(bBuf []byte) (int, error) { <-f.chStop; return 0, errContractClosed }
func (f *fakeDialerTransport) Write(bData []byte) error      { return nil }
func (f *fakeDialerTransport) Resize(int, int) error         { return nil }
func (f *fakeDialerTransport) Close() error                  { close(f.chStop); return nil }

type contractErr string

func (e contractErr) Error() string { return string(e) }

const errContractClosed = contractErr("closed")

// TestContractChannels 는 외부 의존이 없는 채널의 응답 모양을 고정 파일과 비교한다.
func TestContractChannels(t *testing.T) {
	pService, errNew := NewService(t.TempDir(), nil)
	if errNew != nil {
		t.Fatal(errNew)
	}
	defer pService.pSessions.CloseAll()

	contractCheck(t, "app.info", contractInvoke(t, pService, "app.info", "[]"))
	contractCheck(t, "store.get", contractInvoke(t, pService, "store.get", "[]"))
	contractCheck(t, "store.patch", contractInvoke(t, pService, "store.patch", `[{"snippets":null}]`))
	contractCheck(t, "store.get.afterNullPatch", contractInvoke(t, pService, "store.get", "[]"))
	contractCheck(t, "vault.status", contractInvoke(t, pService, "vault.status", "[]"))
	contractCheck(t, "vault.setup", contractInvoke(t, pService, "vault.setup", `["correct-horse-1"]`))
	contractCheck(t, "cred.save", contractInvoke(t, pService, "cred.save", `[{"name":"bastion","kind":"password","username":"ops","password":"pw"}]`))
	contractCheck(t, "cred.list", contractInvoke(t, pService, "cred.list", "[]"))
	szSecret := contractInvoke(t, pService, "cred.get", `["`+firstCredId(t, pService)+`"]`)
	contractCheck(t, "cred.get", szSecret)
	contractInvoke(t, pService, "vault.lock", "[]")
	contractCheck(t, "vault.status.locked", contractInvoke(t, pService, "vault.status", "[]"))
	contractCheck(t, "vault.unlock", contractInvoke(t, pService, "vault.unlock", `["correct-horse-1"]`))
	contractCheck(t, "sync.status", contractInvoke(t, pService, "sync.status", "[]"))
	contractCheck(t, "update.status", contractInvoke(t, pService, "update.status", "[]"))

	// 세션 정보 모양: 가짜 연결을 쓰는 관리자로 연다 (실제 접속·셸은 쓰지 않는다).
	pManager := session.NewManager(func(string, ...any) {})
	defer pManager.CloseAll()
	info, errOpen := pManager.Open(session.OpenConfig{
		Info: session.Info{M_szTitle: "ops@example.test", M_szProtocol: "ssh", M_szTarget: "ops@example.test:22", M_szState: "connecting", M_szEnv: "none", M_bCanSftp: true},
		Dialer: func(func(string)) (session.Transport, error) {
			return &fakeDialerTransport{chStop: make(chan struct{})}, nil
		},
	})
	if errOpen != nil {
		t.Fatal(errOpen)
	}
	bInfo, _ := json.Marshal(info)
	contractCheck(t, "session.open", string(bInfo))
}

// firstCredId 는 cred.list 의 첫 항목 id 를 돌려준다.
func firstCredId(t *testing.T, pService *Service) string {
	t.Helper()
	var slMeta []map[string]any
	if errJson := json.Unmarshal([]byte(contractInvoke(t, pService, "cred.list", "[]")), &slMeta); errJson != nil || len(slMeta) == 0 {
		t.Fatalf("cred.list 해석 실패: %v", errJson)
	}
	return slMeta[0]["id"].(string)
}

// marshalFor 는 값을 JSON 문자열로 바꾼다.
func marshalFor(t *testing.T, anyValue any) string {
	t.Helper()
	bOut, errMarshal := json.Marshal(anyValue)
	if errMarshal != nil {
		t.Fatal(errMarshal)
	}
	return string(bOut)
}

// recordEmitter 는 발행된 이벤트를 모아 두는 가짜 발행기다.
type recordEmitter struct {
	mtxData sync.Mutex
	slEvent []map[string]any
}

func (r *recordEmitter) Emit(szEvent string, anyData ...any) {
	r.mtxData.Lock()
	defer r.mtxData.Unlock()
	if szEvent == "ui.request" && len(anyData) > 0 {
		if mapEvent, bOk := anyData[0].(map[string]any); bOk {
			r.slEvent = append(r.slEvent, mapEvent)
		}
	}
}

// TestContractEvents 는 백엔드→프런트 이벤트 페이로드 모양을 고정 파일과 비교한다.
func TestContractEvents(t *testing.T) {
	contractCheck(t, "event.transfer.update", marshalFor(t, sftpfs.Transfer{
		M_szId: "tr-1", M_szSessionId: "s-1", M_szName: "a.txt", M_szDirection: "upload",
		M_i64Bytes: 10, M_i64Total: 10, M_szState: "done", M_szRemotePath: "/home/x/a.txt", M_szLocalPath: "C:\\a.txt",
	}))
	contractCheck(t, "event.sync.status", marshalFor(t, gitsync.Result{M_szState: "ok", M_i64LastSync: 1, M_bChanged: true}))
	contractCheck(t, "event.update.available", marshalFor(t, map[string]any{"version": "0.2.0", "url": "https://example.test/release", "announce": true}))

	// ui.request: 취소(시간 초과)로 끝내면서 발행된 이벤트 모양을 본다.
	rec := &recordEmitter{}
	pRouter := ipc.NewRouter(rec)
	pRouter.Asker().SetTimeout(20 * time.Millisecond)
	pRouter.Asker().Ask("hostkey", map[string]any{
		"sessionTitle": "ops@example.test", "hostPort": "example.test:22", "keyType": "ssh-ed25519",
		"fingerprint": "SHA256:abc", "changed": false,
	})
	if len(rec.slEvent) != 1 {
		t.Fatalf("ui.request 발행 개수 %d", len(rec.slEvent))
	}
	contractCheck(t, "event.ui.request", marshalFor(t, rec.slEvent[0]))
}
