# SSH 에이전트 전달 (`ssh -A`) — 배경·설계·코드·검증

**작성**: 2026-11-23 · **상태**: 코드·자동 테스트 완료(D111). 실제 서버에서의 확인은 수동 검증으로 남음 · **대상**: cli-manager-cr(DariTerm)
**묶음**: 빠진 기능 10건 중 6번. 앞 문서 [proxy-connect](./2026-11-23-proxy-connect.md), 뒤 문서 [log-line-timestamp](./2026-11-23-log-line-timestamp.md). 이 PC 의 에이전트에 연결하는 부분은 [pageant-agent](./2026-11-23-pageant-agent.md).

---

## 0. 한 줄 요약

서버 편집 > 고급에 **"SSH 에이전트 전달"** 토글을 더했다. 켜면 서버 안에서 `ssh 다른서버`·`git pull` 을 할 때 이 PC 의 SSH 에이전트(OpenSSH·Pageant)에 든 키로 인증한다. 개인 키를 서버에 복사해 둘 필요가 없다. OpenSSH 의 `ssh -A`(ForwardAgent)와 같다. 기본은 꺼짐.

## 1. 배경

- [internal/sshconn/shell.go](../../internal/sshconn/shell.go) `OpenShell` 은 PTY 요청과 셸 시작만 했다. 에이전트 전달 요청(`auth-agent-req@openssh.com`)을 보내는 코드가 없었다.
- 경유 서버를 거쳐 들어간 뒤 그 안에서 또 다른 서버로 넘어가는 작업 방식에서 필요하다.

## 2. 설계 결정

| 항목 | 정한 것 | 이유 |
|---|---|---|
| 저장 | 서버에 `agentForward: true`(끄면 필드 없음) | 서버마다 정한다. 기본 꺼짐 |
| 요청 시점 | 셸을 열 때, PTY 요청 **전** | OpenSSH 와 같은 순서 |
| 구현 | `x/crypto/ssh/agent` 의 `ForwardToAgent`(서버가 여는 에이전트 채널을 이 PC 의 에이전트에 잇기) + `RequestAgentForwarding`(세션에 요청) | 라이브러리가 제공한다 |
| 어느 에이전트 | 연결되는 것 중 **첫 번째 하나**(Windows: OpenSSH 에이전트 → Pageant) | 전달 채널 하나에는 에이전트 하나만 이을 수 있다 |
| 연결마다 한 번 | `ForwardToAgent` 는 같은 연결에 두 번 부르면 오류다 → `sync.Once` 로 한 번만 등록하고 결과를 기억 | 한 연결에서 셸을 다시 열 수 있다 |
| 실패했을 때 | **셸은 그대로 연다.** 터미널에 안내 한 줄만(에이전트 없음 / 서버가 거절) | 에이전트 전달이 안 된다고 접속까지 막을 이유가 없다 |
| 성공했을 때 | 안내 없음 | 조용히 |
| 범위 | 대상 서버의 세션만. 경유 서버에는 전달하지 않는다 | 필요한 곳은 최종 서버다 |
| 인증 방식과의 관계 | 따로다. 비밀번호로 로그인한 서버에도 전달할 수 있다 | OpenSSH 와 같다 |
| 함수 모양 | `OpenShell`(서명 유지) → `OpenShellWith(…, ShellOptions{AgentForward, Notice})` | 기존 호출·테스트를 고치지 않는다 |
| 위험 안내 | 토글 설명에 "서버 관리자도 연결된 동안 그 키를 쓸 수 있으므로 믿는 서버에서만 켜세요" | 에이전트 전달의 알려진 위험 |

## A. 바뀐 파일 한눈에

| 파일 | 종류 | 상태 |
|---|---|---|
| `frontend/src/ipc/types.ts` | 코드 | 수정 |
| `frontend/src/lib/hostSave.ts` | 코드 | 수정 |
| `frontend/src/screens/HostEditor.tsx` | 코드 | 수정 |
| `internal/app/channels_session.go` | 코드 | 수정 |
| `internal/app/ssh_app.go` | 코드 | 수정 |
| `internal/sshconn/agent.go` | 코드 | 수정 |
| `internal/sshconn/client.go` | 코드 | 수정 |
| `internal/sshconn/shell.go` | 코드 | 수정 |
| `internal/sshconn/agent_forward_test.go` | 테스트 | 새 파일 |
| `spec/data/store.md` | 명세 | 수정 |
| `spec/protocols.md` | 명세 | 수정 |
| `spec/ui/host-editor.md` | 명세 | 수정 |
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

아래의 `diff` 는 **이 묶음의 앞 문서들을 차례로 적용한 상태**를 기준으로 뽑았다. 앞 문서에서도 고친 파일(문맥 줄에 앞 기능의 코드가 섞여 있을 수 있다): `frontend/src/ipc/types.ts`, `frontend/src/lib/hostSave.ts`, `frontend/src/screens/HostEditor.tsx`, `internal/app/channels_session.go`, `internal/app/ssh_app.go`, `internal/sshconn/agent.go`, `internal/sshconn/client.go`, `spec/data/store.md`, `spec/protocols.md`, `spec/ui/host-editor.md`. 이 기능만 따로 넣을 때는 `+`/`-` 줄만 옮기면 된다.

## B. 코드

### `frontend/src/ipc/types.ts` — 수정

```diff
--- a/frontend/src/ipc/types.ts
+++ b/frontend/src/ipc/types.ts
@@ -39,6 +39,8 @@
   jumpDirect?: unknown;
   /** 프록시 경유 설정(store.md §3 proxy). 없으면 바로 연결. */
   proxy?: unknown;
+  /** SSH 에이전트 전달(protocols.md §2.8). 켜면 이 PC 에이전트의 키를 서버에서 쓸 수 있다. */
+  agentForward?: boolean;
   color?: string;
   env?: EnvKind;
   tags: string[];
```

### `frontend/src/lib/hostSave.ts` — 수정

```diff
--- a/frontend/src/lib/hostSave.ts
+++ b/frontend/src/lib/hostSave.ts
@@ -40,6 +40,8 @@
   jump: { host: string; port: number; username: string; authType: JumpDirect['authType']; password: string; keyText: string; passphrase: string };
   /** 프록시 경유: type 이 '' 이면 사용 안 함. password 는 새로 넣었을 때만 값이 있다. */
   proxy?: { type: ProxyType | ''; host: string; port: number; username: string; password: string };
+  /** SSH 에이전트 전달. */
+  agentForward?: boolean;
 }
 
 /** 계정 저장(§1.5 5번). 기존 계정을 이 서버만 쓰면 갱신, 다른 서버와 공유 중이면 새 항목. */
@@ -164,6 +166,7 @@
     jumpHostId,
     jumpDirect,
     proxy,
+    agentForward: i.protocol === 'ssh' && i.agentForward ? true : undefined,
     groupId: i.folderId || null,
     env: i.env === 'none' ? undefined : i.env,
     color: i.color,
```

### `frontend/src/screens/HostEditor.tsx` — 수정

```diff
--- a/frontend/src/screens/HostEditor.tsx
+++ b/frontend/src/screens/HostEditor.tsx
@@ -132,6 +132,7 @@
       password: '',
     };
   });
+  const [agentForward, setAgentForward] = useState(!!original?.agentForward);
   const proxyHasSaved = !!(original?.proxy as ProxyConf | undefined)?.credentialId;
 
   // 포워딩·시리얼·로컬
@@ -294,6 +295,7 @@
           passphrase: jump.passphrase,
         },
         proxy: { type: proxy.type, host: proxy.host, port: Number(proxy.port) || 0, username: proxy.username, password: proxy.password },
+        agentForward,
       });
       await patchStore({ hosts: applyHost(hosts, next, !!hostId) });
       await refreshStore();
@@ -533,6 +535,15 @@
                 jumpInUse={!!jumpChoice}
               />
             )}
+            {isSsh && (
+              <div className="toggle-row">
+                <div>
+                  <div className="t-title">{t('host.agentFwd')}</div>
+                  <div className="t-desc">{t('host.agentFwdDesc')}</div>
+                </div>
+                <Switch checked={agentForward} onChange={setAgentForward} label={t('host.agentFwd')} />
+              </div>
+            )}
             {tmuxCardShown && (
               <div className="toggle-row">
                 <div>
```

### `internal/app/channels_session.go` — 수정

```diff
--- a/internal/app/channels_session.go
+++ b/internal/app/channels_session.go
@@ -38,6 +38,7 @@
 	M_iKeepaliveSec    int               `json:"keepaliveSec"`
 	M_slForwards       []forwardRecord   `json:"forwards"`
 	M_pProxy           *proxyRecord      `json:"proxy"`
+	M_bAgentForward    bool              `json:"agentForward"` // SSH 에이전트 전달 (D111)
 }
 
 // proxyRecord 는 서버의 프록시 경유 설정이다 (data/store.md §3 proxy, D110).
```

### `internal/app/ssh_app.go` — 수정

```diff
--- a/internal/app/ssh_app.go
+++ b/internal/app/ssh_app.go
@@ -66,7 +66,8 @@
 			return nil, errConnect
 		}
 		creds.remember(tgt)
-		shell, errShell := sshconn.OpenShell(conn, record.M_szTermType, 120, 32)
+		shell, errShell := sshconn.OpenShellWith(conn, record.M_szTermType, 120, 32,
+			sshconn.ShellOptions{AgentForward: record.M_bAgentForward, Notice: notify})
 		if errShell != nil {
 			_ = conn.Close()
 			return nil, errShell
```

### `internal/sshconn/agent.go` — 수정

```diff
--- a/internal/sshconn/agent.go
+++ b/internal/sshconn/agent.go
@@ -18,10 +18,13 @@
 // errAgentMessage 는 에이전트와 주고받은 메시지 모양이 틀렸을 때의 오류다.
 var errAgentMessage = errors.New("ssh agent: malformed message")
 
+// g_fnAvailableAgents 는 연결할 수 있는 에이전트 목록을 돌려주는 함수다. 시험에서 가짜 에이전트로 바꾼다.
+var g_fnAvailableAgents = availableAgents
+
 // agentSignersFor 는 연결할 수 있는 모든 에이전트의 서명자를 모아 돌려주는 함수를 만든다.
 // 에이전트가 하나도 없으면 nil 을 돌려 이 인증 방법을 건너뛰게 한다.
 func agentSignersFor() func() ([]ssh.Signer, error) {
-	slAgents := availableAgents()
+	slAgents := g_fnAvailableAgents()
 	if len(slAgents) == 0 {
 		return nil
 	}
```

### `internal/sshconn/client.go` — 수정

```diff
--- a/internal/sshconn/client.go
+++ b/internal/sshconn/client.go
@@ -29,6 +29,8 @@
 	sftpClient *sftp.Client
 	chClosed   chan struct{} // Close 때 닫힌다 (keepalive 루프 종료용)
 	onceClose  sync.Once
+	onceAgent  sync.Once // 에이전트 전달 채널 처리기는 연결마다 한 번만 등록한다 (D111)
+	errAgent   error     // onceAgent 의 결과
 }
 
 // Close 는 대상과 모든 점프 연결을 닫는다 (protocols.md §2.4). 두 번 불러도 안전하다.
```

### `internal/sshconn/shell.go` — 수정

```diff
--- a/internal/sshconn/shell.go
+++ b/internal/sshconn/shell.go
@@ -6,6 +6,9 @@
 	"sync"
 
 	"golang.org/x/crypto/ssh"
+	"golang.org/x/crypto/ssh/agent"
+
+	"dariterm/internal/i18n"
 
 	"dariterm/internal/transport"
 )
@@ -20,9 +23,57 @@
 	mtxClose  sync.Once
 }
 
+// ShellOptions 는 셸을 열 때의 선택 사항이다.
+type ShellOptions struct {
+	// AgentForward 가 true 면 이 PC 의 SSH 에이전트를 서버에서 쓸 수 있게 전달한다 (protocols.md §2.8, D111).
+	AgentForward bool
+	// Notice 는 터미널 안내 줄을 보낸다 (에이전트 전달을 못 했을 때). nil 이면 안내하지 않는다.
+	Notice func(string)
+}
+
+// errNoAgent 는 전달할 에이전트가 이 PC 에 없을 때의 내부 표시다.
+var errNoAgent = errors.New("no ssh agent")
+
+// enableAgentForward 는 이 연결에서 서버가 여는 에이전트 채널을 이 PC 의 첫 번째 에이전트로 잇는다.
+// 연결마다 한 번만 등록하고 결과를 기억한다.
+func (pConn *Conn) enableAgentForward() error {
+	pConn.onceAgent.Do(func() {
+		slAgents := g_fnAvailableAgents()
+		if len(slAgents) == 0 {
+			pConn.errAgent = errNoAgent
+			return
+		}
+		pConn.errAgent = agent.ForwardToAgent(pConn.Client, slAgents[0])
+	})
+	return pConn.errAgent
+}
+
+// requestAgentForward 는 세션에 에이전트 전달을 요청한다. 실패해도 셸은 그대로 연다(안내만 한다).
+func requestAgentForward(conn *Conn, session *ssh.Session, notice func(string)) {
+	if notice == nil {
+		notice = func(string) {}
+	}
+	if errEnable := conn.enableAgentForward(); errEnable != nil {
+		if errors.Is(errEnable, errNoAgent) {
+			notice(i18n.T("be.ssh.agentFwdNoAgent"))
+		} else {
+			notice(i18n.T("be.ssh.agentFwdRefused", "error", errEnable))
+		}
+		return
+	}
+	if errRequest := agent.RequestAgentForwarding(session); errRequest != nil {
+		notice(i18n.T("be.ssh.agentFwdRefused", "error", errRequest))
+	}
+}
+
 // OpenShell 은 PTY 를 요청하고 셸을 시작한다 (protocols.md §2.1 3·4번).
 // termType 이 비면 xterm-256color, 크기는 각각 최소 2 다.
 func OpenShell(conn *Conn, termType string, iCols int, iRows int) (*Shell, error) {
+	return OpenShellWith(conn, termType, iCols, iRows, ShellOptions{})
+}
+
+// OpenShellWith 는 선택 사항(에이전트 전달)을 주어 셸을 연다.
+func OpenShellWith(conn *Conn, termType string, iCols int, iRows int, opts ShellOptions) (*Shell, error) {
 	if termType == "" {
 		termType = "xterm-256color"
 	}
@@ -35,6 +86,9 @@
 	session, errSession := conn.Client.NewSession()
 	if errSession != nil {
 		return nil, errSession
+	}
+	if opts.AgentForward {
+		requestAgentForward(conn, session, opts.Notice)
 	}
 	modes := ssh.TerminalModes{
 		ssh.ECHO:          1,
```


## C. 테스트 코드

### `internal/sshconn/agent_forward_test.go` — 새 파일 (전체)

```go
package sshconn

import (
	"crypto/ed25519"
	"crypto/rand"
	"net"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

// 에이전트 전달 시험 (D111).

// startAgentServer 는 에이전트 전달 요청을 받는 시험용 SSH 서버다.
// 세션에 auth-agent-req@openssh.com 이 오면(bAccept 일 때 수락) 서버 쪽에서 에이전트 채널을 열어
// 키 주석 목록을 chKeys 로 보낸다. 요청이 온 횟수는 chRequests 로 알린다.
func startAgentServer(t *testing.T, bAccept bool) (int, chan []string, chan struct{}) {
	t.Helper()
	_, hostKey, _ := ed25519.GenerateKey(rand.Reader)
	signer, _ := ssh.NewSignerFromKey(hostKey)
	cfg := &ssh.ServerConfig{NoClientAuth: true}
	cfg.AddHostKey(signer)
	ln, errListen := net.Listen("tcp", "127.0.0.1:0")
	if errListen != nil {
		t.Fatal(errListen)
	}
	t.Cleanup(func() { _ = ln.Close() })
	chKeys := make(chan []string, 4)
	chRequests := make(chan struct{}, 4)
	go func() {
		for {
			c, errAccept := ln.Accept()
			if errAccept != nil {
				return
			}
			go func() {
				sconn, chans, reqs, errHs := ssh.NewServerConn(c, cfg)
				if errHs != nil {
					return
				}
				defer sconn.Close()
				go ssh.DiscardRequests(reqs)
				for newCh := range chans {
					if newCh.ChannelType() != "session" {
						_ = newCh.Reject(ssh.UnknownChannelType, "unknown")
						continue
					}
					ch, chReqs, errCh := newCh.Accept()
					if errCh != nil {
						continue
					}
					go func() {
						defer ch.Close()
						for req := range chReqs {
							switch req.Type {
							case "auth-agent-req@openssh.com":
								chRequests <- struct{}{}
								_ = req.Reply(bAccept, nil)
								if !bAccept {
									continue
								}
								go func() {
									agentCh, agentReqs, errOpen := sconn.OpenChannel("auth-agent@openssh.com", nil)
									if errOpen != nil {
										chKeys <- []string{"open failed: " + errOpen.Error()}
										return
									}
									go ssh.DiscardRequests(agentReqs)
									defer agentCh.Close()
									slKeys, errList := agent.NewClient(agentCh).List()
									if errList != nil {
										chKeys <- []string{"list failed: " + errList.Error()}
										return
									}
									var slComments []string
									for _, pKey := range slKeys {
										slComments = append(slComments, pKey.Comment)
									}
									chKeys <- slComments
								}()
							case "pty-req", "shell":
								_ = req.Reply(true, nil)
							default:
								_ = req.Reply(false, nil)
							}
						}
					}()
				}
			}()
		}
	}()
	return ln.Addr().(*net.TCPAddr).Port, chKeys, chRequests
}

// withAgents 는 시험 동안 "이 PC 의 에이전트" 목록을 바꾼다.
func withAgents(t *testing.T, slAgents []agent.ExtendedAgent) {
	t.Helper()
	fnOld := g_fnAvailableAgents
	g_fnAvailableAgents = func() []agent.ExtendedAgent { return slAgents }
	t.Cleanup(func() { g_fnAvailableAgents = fnOld })
}

func keyringWith(t *testing.T, szComment string) agent.ExtendedAgent {
	t.Helper()
	keyring := agent.NewKeyring()
	_, privKey, _ := ed25519.GenerateKey(rand.Reader)
	if errAdd := keyring.Add(agent.AddedKey{PrivateKey: privKey, Comment: szComment}); errAdd != nil {
		t.Fatal(errAdd)
	}
	return keyring.(agent.ExtendedAgent)
}

func connectNoAuth(t *testing.T, iPort int) *Conn {
	t.Helper()
	tgt := &Target{Title: "t", Host: "127.0.0.1", Port: iPort, Username: "u", AuthType: "ask"}
	conn, errConnect := Connect(tgt, &memStore{m: map[string]string{}}, &fakePrompt{approve: true}, func(string) {})
	if errConnect != nil {
		t.Fatal(errConnect)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

// 에이전트 전달을 켜면 서버가 이 PC 에이전트의 키 목록을 볼 수 있다. 같은 연결에서 셸을 두 번 열어도 된다.
func TestAgentForwardReachesLocalAgent(t *testing.T) {
	withAgents(t, []agent.ExtendedAgent{keyringWith(t, "local-agent-key")})
	iPort, chKeys, _ := startAgentServer(t, true)
	conn := connectNoAuth(t, iPort)
	for iRound := 0; iRound < 2; iRound++ {
		var slNotices []string
		shell, errShell := OpenShellWith(conn, "xterm", 80, 24, ShellOptions{AgentForward: true, Notice: func(szLine string) { slNotices = append(slNotices, szLine) }})
		if errShell != nil {
			t.Fatal(errShell)
		}
		select {
		case slKeys := <-chKeys:
			if len(slKeys) != 1 || slKeys[0] != "local-agent-key" {
				t.Fatalf("서버가 본 키 목록: %v", slKeys)
			}
		case <-time.After(3 * time.Second):
			t.Fatal("서버가 전달된 에이전트에 닿지 못함")
		}
		if len(slNotices) != 0 {
			t.Fatalf("성공하면 안내가 없어야 함: %v", slNotices)
		}
		_ = shell.Close()
	}
}

// 끄면(기본) 에이전트 전달을 요청하지 않는다.
func TestAgentForwardOffByDefault(t *testing.T) {
	withAgents(t, []agent.ExtendedAgent{keyringWith(t, "k")})
	iPort, _, chRequests := startAgentServer(t, true)
	conn := connectNoAuth(t, iPort)
	shell, errShell := OpenShell(conn, "xterm", 80, 24)
	if errShell != nil {
		t.Fatal(errShell)
	}
	defer shell.Close()
	select {
	case <-chRequests:
		t.Fatal("끄면 요청하지 않아야 함")
	case <-time.After(150 * time.Millisecond):
	}
}

// 에이전트가 없거나 서버가 거절하면 안내 한 줄을 내고 셸은 그대로 연다.
func TestAgentForwardFailuresOnlyNotify(t *testing.T) {
	iPort, _, chRequests := startAgentServer(t, false)

	withAgents(t, nil)
	conn := connectNoAuth(t, iPort)
	var slNotices []string
	shell, errShell := OpenShellWith(conn, "xterm", 80, 24, ShellOptions{AgentForward: true, Notice: func(szLine string) { slNotices = append(slNotices, szLine) }})
	if errShell != nil {
		t.Fatalf("에이전트가 없어도 셸은 열려야 함: %v", errShell)
	}
	_ = shell.Close()
	if len(slNotices) != 1 {
		t.Fatalf("에이전트 없음 안내 한 줄: %v", slNotices)
	}
	select {
	case <-chRequests:
		t.Fatal("에이전트가 없으면 서버에 요청하지 않아야 함")
	case <-time.After(100 * time.Millisecond):
	}

	withAgents(t, []agent.ExtendedAgent{keyringWith(t, "k")})
	conn2 := connectNoAuth(t, iPort)
	slNotices = nil
	shell2, errShell2 := OpenShellWith(conn2, "xterm", 80, 24, ShellOptions{AgentForward: true, Notice: func(szLine string) { slNotices = append(slNotices, szLine) }})
	if errShell2 != nil {
		t.Fatalf("서버가 거절해도 셸은 열려야 함: %v", errShell2)
	}
	_ = shell2.Close()
	if len(slNotices) != 1 || !strings.Contains(slNotices[0], "forwarding request denied") {
		t.Fatalf("거절 안내 한 줄(원인 포함): %v", slNotices)
	}
}
```


## E. 문구

`locales/<언어>.json` 10개 파일에 같은 키를 넣는다(순서: ko, en, ja, zh, tw, de, fr, es, pt, ru). 한국어·영어 외에는 이 작업을 한 세션이 번역한 것이라 검수가 필요할 수 있다.

**`host.agentFwd`** (추가)

```json
  "host.agentFwd": "SSH 에이전트 전달",
  "host.agentFwd": "SSH agent forwarding",
  "host.agentFwd": "SSH エージェント転送",
  "host.agentFwd": "SSH 代理转发",
  "host.agentFwd": "SSH 代理程式轉送",
  "host.agentFwd": "SSH-Agent-Weiterleitung",
  "host.agentFwd": "Transfert de l'agent SSH",
  "host.agentFwd": "Reenvío del agente SSH",
  "host.agentFwd": "Encaminhamento do agente SSH",
  "host.agentFwd": "Перенаправление SSH-агента",
```

**`host.agentFwdDesc`** (추가)

```json
  "host.agentFwdDesc": "이 PC의 SSH 에이전트(OpenSSH·Pageant)에 든 키를 이 서버 안에서도 쓸 수 있게 합니다(서버에서 다른 서버로 ssh·git 할 때). 서버 관리자도 연결된 동안 그 키를 쓸 수 있으므로 믿는 서버에서만 켜세요.",
  "host.agentFwdDesc": "Lets this server use the keys held by the SSH agent on this PC (OpenSSH or Pageant), for example to ssh or git from the server to another server. The server's administrators can also use those keys while you are connected, so enable this only on servers you trust.",
  "host.agentFwdDesc": "この PC の SSH エージェント（OpenSSH・Pageant）にある鍵を、このサーバー内でも使えるようにします（サーバーから別のサーバーへ ssh・git するとき）。接続中はサーバーの管理者もその鍵を使えるため、信頼できるサーバーでのみ有効にしてください。",
  "host.agentFwdDesc": "允许此服务器使用本机 SSH 代理（OpenSSH、Pageant）中的密钥（例如从服务器 ssh 或 git 到另一台服务器）。连接期间服务器管理员也能使用这些密钥，请仅在信任的服务器上开启。",
  "host.agentFwdDesc": "允許此伺服器使用本機 SSH 代理程式（OpenSSH、Pageant）中的金鑰（例如從伺服器 ssh 或 git 到另一台伺服器）。連線期間伺服器管理員也能使用這些金鑰，請僅在信任的伺服器上開啟。",
  "host.agentFwdDesc": "Erlaubt diesem Server, die Schlüssel des SSH-Agenten auf diesem PC (OpenSSH oder Pageant) zu verwenden, etwa für ssh oder git vom Server zu einem anderen Server. Auch die Administratoren des Servers können diese Schlüssel während der Verbindung nutzen – nur bei vertrauenswürdigen Servern aktivieren.",
  "host.agentFwdDesc": "Permet à ce serveur d'utiliser les clés de l'agent SSH de ce PC (OpenSSH ou Pageant), par exemple pour ssh ou git depuis le serveur vers un autre serveur. Les administrateurs du serveur peuvent aussi utiliser ces clés pendant la connexion : n'activez cette option que sur des serveurs de confiance.",
  "host.agentFwdDesc": "Permite que este servidor use las claves del agente SSH de este PC (OpenSSH o Pageant), por ejemplo para hacer ssh o git desde el servidor a otro servidor. Los administradores del servidor también pueden usar esas claves mientras esté conectado; actívelo solo en servidores de confianza.",
  "host.agentFwdDesc": "Permite que este servidor use as chaves do agente SSH deste PC (OpenSSH ou Pageant), por exemplo para ssh ou git do servidor para outro servidor. Os administradores do servidor também podem usar essas chaves enquanto você estiver conectado; ative apenas em servidores confiáveis.",
  "host.agentFwdDesc": "Позволяет этому серверу использовать ключи SSH-агента этого ПК (OpenSSH или Pageant), например для ssh или git с сервера на другой сервер. Администраторы сервера тоже могут пользоваться этими ключами, пока вы подключены, поэтому включайте только на доверенных серверах.",
```

**`be.ssh.agentFwdNoAgent`** (추가)

```json
  "be.ssh.agentFwdNoAgent": "에이전트 전달: 이 PC에서 실행 중인 SSH 에이전트가 없어 전달하지 않습니다",
  "be.ssh.agentFwdNoAgent": "Agent forwarding: no SSH agent is running on this PC, so nothing is forwarded",
  "be.ssh.agentFwdNoAgent": "エージェント転送: この PC で SSH エージェントが動作していないため転送しません",
  "be.ssh.agentFwdNoAgent": "代理转发：本机没有正在运行的 SSH 代理，未进行转发",
  "be.ssh.agentFwdNoAgent": "代理程式轉送：本機沒有執行中的 SSH 代理程式，未進行轉送",
  "be.ssh.agentFwdNoAgent": "Agent-Weiterleitung: Auf diesem PC läuft kein SSH-Agent, es wird nichts weitergeleitet",
  "be.ssh.agentFwdNoAgent": "Transfert de l'agent : aucun agent SSH n'est en cours d'exécution sur ce PC, rien n'est transféré",
  "be.ssh.agentFwdNoAgent": "Reenvío del agente: no hay ningún agente SSH en ejecución en este PC; no se reenvía nada",
  "be.ssh.agentFwdNoAgent": "Encaminhamento do agente: não há agente SSH em execução neste PC; nada é encaminhado",
  "be.ssh.agentFwdNoAgent": "Перенаправление агента: на этом ПК не запущен SSH-агент, перенаправление не выполняется",
```

**`be.ssh.agentFwdRefused`** (추가)

```json
  "be.ssh.agentFwdRefused": "에이전트 전달을 켜지 못했습니다: {error}",
  "be.ssh.agentFwdRefused": "Agent forwarding could not be enabled: {error}",
  "be.ssh.agentFwdRefused": "エージェント転送を有効にできませんでした: {error}",
  "be.ssh.agentFwdRefused": "无法启用代理转发：{error}",
  "be.ssh.agentFwdRefused": "無法啟用代理程式轉送：{error}",
  "be.ssh.agentFwdRefused": "Agent-Weiterleitung konnte nicht aktiviert werden: {error}",
  "be.ssh.agentFwdRefused": "Impossible d'activer le transfert de l'agent : {error}",
  "be.ssh.agentFwdRefused": "No se pudo activar el reenvío del agente: {error}",
  "be.ssh.agentFwdRefused": "Não foi possível ativar o encaminhamento do agente: {error}",
  "be.ssh.agentFwdRefused": "Не удалось включить перенаправление агента: {error}",
```


## F. 명세

### `spec/data/store.md` — 수정

```diff
--- a/spec/data/store.md
+++ b/spec/data/store.md
@@ -44,6 +44,7 @@
 | credentialId | 문자열 또는 null(선택) | 볼트 항목 id |
 | jumpHostId | 문자열 또는 null(선택) | 점프 호스트 서버 id. 값이 `direct` 이면 jumpDirect 사용 |
 | jumpDirect | 객체(선택) | host, port, username, authType(`password`/`key`), credentialId(선택) |
+| agentForward | 불리언(선택) | SSH 에이전트 전달(2026-11-23, D111). 없으면 끔. SSH 만 쓴다 |
 | proxy | 객체(선택) | 프록시 경유(2026-11-23, D110): type(`http`/`socks5`), host, port, username(비면 인증 없음), credentialId(선택, 비밀번호 계정). 없거나 type·host 가 비면 바로 연결. SSH 만 쓴다 |
 | color | 문자열(선택) | |
 | env | 환경 값(선택) | |
```

### `spec/protocols.md` — 수정

```diff
--- a/spec/protocols.md
+++ b/spec/protocols.md
@@ -69,6 +69,14 @@
 - **프록시 비밀번호**는 볼트의 계정(`proxy.credentialId`)에서 읽는다. 볼트가 잠겨 있으면 세션 사본(§2.2, 접속에 성공했던 프록시 비밀번호를 세션 메모리에 둔 것)을 쓰고, 사본도 없으면 접속하지 않고 잠금 해제를 기다린다.
 - 범위 밖: SOCKS4 프록시, 프록시 명령(ProxyCommand), 프록시의 NTLM·Kerberos 인증, 텔넷의 프록시 경유.
 
+### 2.8 에이전트 전달 (2026-11-23, D111)
+- 서버의 `agentForward` 가 켜져 있으면 셸을 열 때(PTY 요청 전에) **에이전트 전달**을 요청한다. 서버 안의 프로그램(ssh, git 등)이 서명을 요청하면 그 요청이 SSH 연결을 타고 이 PC 로 와서 **이 PC 의 SSH 에이전트**가 서명한다. 키 자체는 서버로 가지 않는다.
+- 쓰는 에이전트는 §2.2 2번의 순서에서 **처음 연결되는 하나**다(Windows: OpenSSH 에이전트 → Pageant).
+- 이 PC 에 에이전트가 없으면 터미널에 안내 "에이전트 전달: 이 PC에서 실행 중인 SSH 에이전트가 없어 전달하지 않습니다", 서버가 거절하면(서버 설정에서 막은 경우 등) "에이전트 전달을 켜지 못했습니다: {오류}" 를 보인다. **어느 경우든 셸은 그대로 연다.** 성공하면 안내가 없다.
+- 대상 서버의 세션에만 적용한다. 경유 서버(점프 호스트)에는 전달하지 않는다. 재접속할 때마다 다시 요청한다.
+- 인증 방식과는 따로다: 비밀번호로 로그인한 서버에도 전달할 수 있다.
+- 위험: 연결된 동안 서버의 관리자 권한을 가진 사람은 전달된 에이전트로 서명을 요청할 수 있다. 화면 도움말에 이 사실을 적는다(ui/host-editor.md §1.3).
+
 ### 2.6 연결 유지(keepalive)
 `keepaliveSec` 가 0 보다 크면 그 간격으로 `keepalive@openssh.com` 요청을 보낸다. **연속 4번 실패하면 연결을 닫는다**(세션 종료 처리 → 자동 재접속 규칙 적용). 요청의 응답이 keepalive 간격 안에 오지 않으면 그 요청은 실패로 센다(상대가 조용히 사라져도 약 4 × 간격 안에 끊김을 알아챈다). 연결을 닫으면 keepalive 도 바로 멈춘다.
 
```

### `spec/ui/host-editor.md` — 수정

```diff
--- a/spec/ui/host-editor.md
+++ b/spec/ui/host-editor.md
@@ -50,6 +50,8 @@
 
 **프록시 경유**(SSH 만, 2026-11-23, D110): 경유 서버 항목 아래에 "프록시 경유" 드롭다운(없음 / HTTP 프록시 (CONNECT) / SOCKS5 프록시), 도움말 "회사 방화벽처럼 서버에 바로 연결할 수 없을 때, 이 프록시를 거쳐 SSH 연결을 맺습니다." 종류를 고르면 카드가 나온다: 2열(2:1) "프록시 주소"(고정폭, 플레이스홀더 `proxy.example.com`) / "포트"(숫자, 플레이스홀더는 종류별 기본 포트 8080·1080), 2열 "프록시 사용자 이름"(도움말 "프록시가 로그인을 요구할 때만 넣습니다. 비우면 인증 없이 연결합니다.") / "비밀번호"(사용자 이름이 비면 비활성. 저장된 것이 있으면 플레이스홀더가 가림 문자, 도움말은 경유 서버 카드와 같은 규칙). 경유 서버를 골라 둔 상태면 카드 맨 위에 안내 "경유 서버를 쓰는 동안에는 이 프록시를 쓰지 않습니다. …". 저장: 종류와 주소가 있으면 `proxy` 를 쓰고(포트 0 은 기본 포트), 사용자 이름과 비밀번호를 넣었으면 `{별칭} 프록시` 이름의 비밀번호 계정으로 저장해 연결한다. 사용자 이름을 비우면 계정 연결을 끊는다. 검사: 종류를 골랐는데 주소가 비면 "프록시 주소를 입력하세요", 비밀번호를 넣었는데 앱이 잠겨 있으면 "앱이 잠겨 있어 프록시 비밀번호를 저장할 수 없습니다"(둘 다 고급 탭으로 이동).
 
+**SSH 에이전트 전달**(SSH 만, 2026-11-23, D111): 프록시 경유 아래에 토글 "SSH 에이전트 전달"(기본 꺼짐), 설명 "이 PC의 SSH 에이전트(OpenSSH·Pageant)에 든 키를 이 서버 안에서도 쓸 수 있게 합니다(서버에서 다른 서버로 ssh·git 할 때). 서버 관리자도 연결된 동안 그 키를 쓸 수 있으므로 믿는 서버에서만 켜세요." 저장: 켜면 `agentForward: true`, 끄면 필드를 쓰지 않는다.
+
 ### 1.4 "포트 포워딩" 탭 (SSH 만)
 - 안내 박스: "L: 내 PC의 포트 → 서버에서 접근 가능한 주소 (예: 로컬 3307 → DB 3306). R: 서버의 포트 → 내 PC 쪽 주소. D: 내 PC의 포트를 SOCKS 프록시로 열어, 그 프록시를 쓰는 프로그램의 접속을 서버를 거쳐 내보냅니다(대상은 프로그램이 정합니다)."(2026-11-23, D109)
 - 표: 머리 종류 · 바인드 주소 · 포트 · 대상 호스트 · 대상 포트 · (삭제). 행마다 종류 드롭다운(L/R/D), 바인드 주소(고정폭), 포트(숫자), 대상 호스트(고정폭), 대상 포트(숫자), [삭제](휴지통 아이콘). 종류가 D 이면 대상 호스트·대상 포트 칸 대신 흐린 글자 "SOCKS 프록시 (대상 없음)" 을 두 칸 너비로 보인다(적어 둔 대상 값은 지우지 않아 종류를 되돌리면 다시 보인다).
```



---

## G. 검증 (2026-11-23 실행)

시험용 SSH 서버가 `auth-agent-req@openssh.com` 을 받으면 서버 쪽에서 에이전트 채널을 열어 키 목록을 읽는다. 이 PC 의 에이전트는 `g_fnAvailableAgents` 를 바꿔 메모리 키링으로 대신한다.

| 테스트 ([internal/sshconn/agent_forward_test.go](../../internal/sshconn/agent_forward_test.go)) | 확인하는 것 |
|---|---|
| `TestAgentForwardReachesLocalAgent` | 켜면 **서버가 이 PC 에이전트의 키 목록을 본다** / 같은 연결에서 셸을 두 번 열어도 된다 / 성공하면 안내가 없다 |
| `TestAgentForwardOffByDefault` | 끄면(기본) 서버에 요청하지 않는다 |
| `TestAgentForwardFailuresOnlyNotify` | 에이전트가 없으면 서버에 요청하지 않고 안내 한 줄, 셸은 열린다 / 서버가 거절하면 원인이 담긴 안내 한 줄, 셸은 열린다 |

```bash
go test ./internal/sshconn/ ./internal/app/     # 통과
(cd frontend && npx tsc --noEmit)
```

## H. 수동 검증 (남은 일)

| # | 절차 | 기대 결과 |
|---|---|---|
| 1 | 이 PC 의 에이전트(OpenSSH `ssh-add` 또는 Pageant)에 키를 올린다. 서버 A 의 "SSH 에이전트 전달"을 켜고 연결 → A 에서 `ssh-add -l` | 이 PC 의 키가 보인다 |
| 2 | A 에서 그 키를 받는 서버 B 로 `ssh B` | 비밀번호 없이 접속된다 |
| 3 | 에이전트를 모두 끄고 A 에 연결 | `에이전트 전달: 이 PC에서 실행 중인 SSH 에이전트가 없어 전달하지 않습니다` 가 보이고 셸은 정상 |
| 4 | 서버 설정에서 `AllowAgentForwarding no` 인 서버 | `에이전트 전달을 켜지 못했습니다: …` 가 보이고 셸은 정상 |
| 5 | 토글을 끄고 연결 → `ssh-add -l` | 에이전트에 연결할 수 없다고 나온다(전과 같음) |
| 6 | 전달을 켠 서버에서 연결이 끊겨 자동 재접속 | 재접속 뒤에도 `ssh-add -l` 에 키가 보인다 |

## I. 영향 범위·회귀 위험

| 영역 | 위험 | 대비 |
|---|---|---|
| 전달을 켜지 않은 서버(전부) | `opts.AgentForward` 가 false 면 새 코드를 지나지 않는다 | `TestAgentForwardOffByDefault`, 기존 셸 테스트 |
| 보안 | 켠 서버의 관리자 권한을 가진 사람이 연결된 동안 에이전트에 서명을 요청할 수 있다(키 자체는 가져가지 못한다) | 기본 꺼짐, 토글 설명에 명시. Pageant 의 "키 사용 전 확인"을 쓰면 요청마다 확인 창이 뜬다 |
| 에이전트 연결 | Windows OpenSSH 파이프는 연결마다 하나 열어 SSH 연결이 살아 있는 동안 쓴다 | 닫는 코드는 없다(pageant-agent 문서 §2.1 과 같은 정리 거리) |

## J. 체크리스트

- [x] `shell.go`: `ShellOptions`, `enableAgentForward`, `requestAgentForward`, `OpenShellWith`
- [x] `client.go`: 연결에 `onceAgent`·`errAgent`, `agent.go`: `g_fnAvailableAgents`
- [x] 앱: `M_bAgentForward`, `OpenShellWith` 호출
- [x] 프런트: `Host.agentForward`, 고급 탭 토글, 저장
- [x] 문구 4개 추가(10개 언어), 명세 protocols.md §2.8·data/store.md §3·ui/host-editor.md §1.3, decisions D111
- [x] 자동 테스트 3개 통과
- [ ] 수동 검증 1~6
