# 프록시 명령 (ProxyCommand)

**작성**: 2026-11-27 · **상태**: 코드·자동 테스트(실제 SSH 서버까지 통합 시험 포함)·빌드 완료(D152). 실제 사내 터널 프로그램으로의 확인은 수동 검증으로 남음(§H) · **대상**: cli-manager-cr(DariTerm) · **Known issue**: 없음
**관련**: [2026-11-23-proxy-connect.md](./2026-11-23-proxy-connect.md)(D110, "범위 밖: 프록시 명령"), [-15-](./2026-11-27-15-sshconfig-proxyjump-chain.md)(ssh config 의 ProxyCommand 가져오기)

---

## 0. 한 줄 요약

서버까지의 통로를 TCP 대신 **외부 명령의 표준 입출력**으로 만든다. OpenSSH 의 `ProxyCommand` 와 같다 — `ssh -W %h:%p bastion`, `ncat --proxy proxy:8080 %h %p`, `cloudflared access ssh --hostname %h` 처럼 회사·클라우드가 정한 터널 프로그램을 그대로 쓸 수 있다. 서버 편집 > 고급 > 프록시 경유에서 "프록시 명령 (ProxyCommand)" 을 고른다.

## 1. 결정

| 항목 | 결정 | 까닭 |
|---|---|---|
| 저장 | 기존 `proxy` 에 `type: "command"`, `command: "…"` | 프록시 경유의 한 종류로 두면 "경유 서버가 있으면 맨 바깥 서버의 프록시만"(D110) 규칙을 그대로 쓴다 |
| 자리표시자 | `%h` 호스트, `%p` 포트, `%r` 사용자 이름, `%%` → `%`. 그 밖의 `%x` 는 그대로 | OpenSSH 의 자주 쓰는 것. 모르는 것을 지우면 명령이 망가진다 |
| 실행 | Windows `cmd.exe /d /s /c "{명령 줄}"`(창 없음, `ComSpec` 우선), 그 밖 `/bin/sh -c` | 사용자가 명령 프롬프트에서 시험한 줄과 해석이 같아야 한다. 인자를 우리가 나누면 따옴표 규칙이 달라진다 |
| 첫 응답 시한 | 20초(연결 시한과 같다) 안에 첫 바이트가 없으면 명령을 끝낸다 | 비밀번호를 묻는 명령(창이 없어 입력할 수 없다)이 영원히 멈추지 않게 |
| 실패 까닭 | 명령의 표준 오류 마지막 512바이트를 기억해, 출력이 닫히면 `EOF ({프로그램}: {표준 오류})` 로 돌려준다 | 그냥 `ssh: handshake failed: EOF` 만 보이면 왜인지 알 수 없다 |
| 안내 줄 | "프록시 명령 {프로그램} 으로 연결하는 중..." — 프로그램 이름(첫 낱말, 따옴표 안)만 | 인자에 토큰·비밀번호가 들 수 있다 |
| 계정 | 쓰지 않는다(`host`·`port`·`username`·`credentialId` 무시) | 인증은 명령이 알아서 한다 |
| 비밀번호를 물어 다시 연결 | 명령을 새로 띄운다 | TCP 와 같다 |
| 대안 | 인자를 나눠 직접 실행 — 기각(해석 차이). 명령 줄 전체를 안내에 — 기각(비밀 노출) | |

## 2. 구현

### 2.1 `internal/sshconn/proxy_command.go` (새 파일, 요점)

```go
// G_cszProxyCommand 는 프록시 종류 "명령"이다 (data/store.md §3 proxy.type).
const G_cszProxyCommand = "command"

// g_ciCommandStderrMax 는 오류 안내에 붙일 명령의 표준 오류 끝부분 최대 길이(바이트)다.
const g_ciCommandStderrMax = 512

// G_durCommandFirstByte 는 명령을 띄운 뒤 첫 응답(서버 인사)을 기다리는 최대 시간이다. 넘으면 명령을 끝낸다.
var G_durCommandFirstByte = G_cdurConnectTimeout

// ExpandProxyCommand 는 명령의 자리표시자를 바꾼다: %h 대상 호스트, %p 포트, %r 사용자 이름, %% 는 % 하나.
func ExpandProxyCommand(szCommand string, szHost string, iPort int, szUser string) string { … }

// commandProgram 은 안내에 보일 프로그램 이름이다 (명령의 첫 낱말). 인자에 비밀이 들 수 있어 전체는 보이지 않는다.
func commandProgram(szCommand string) string { … }

// commandConn 은 외부 명령의 표준 출력을 읽고 표준 입력에 쓰는 net.Conn 이다.
// 시한(SetDeadline)은 지원하지 않는다 — 대신 첫 바이트를 G_durCommandFirstByte 안에 받지 못하면 명령을 끝낸다.
type commandConn struct {
	pCmd      *exec.Cmd
	stdin     io.WriteCloser
	stdout    io.ReadCloser
	pStderr   *tailBuffer
	szProgram string
	onceClose sync.Once
	onceFirst sync.Once
	timerWait *time.Timer
	bTimedOut bool
	mtxState  sync.Mutex
}

// dialCommand 는 명령을 띄워 그 표준 입출력을 연결로 돌려준다.
func dialCommand(szCommand string, szHost string, iPort int, szUser string) (net.Conn, error) {
	szLine := ExpandProxyCommand(strings.TrimSpace(szCommand), szHost, iPort, szUser)
	if szLine == "" {
		return nil, i18n.NewError("be.proxy.cmdEmpty")
	}
	pCmd := shellCommand(szLine)
	stdin, errIn := pCmd.StdinPipe()
	…
	stdout, errOut := pCmd.StdoutPipe()
	…
	pStderr := &tailBuffer{}
	pCmd.Stderr = pStderr
	szProgram := commandProgram(szLine)
	if errStart := pCmd.Start(); errStart != nil {
		return nil, i18n.Errorf("be.proxy.cmdStart", errStart, "program", szProgram, "error", errStart.Error())
	}
	pConn := &commandConn{pCmd: pCmd, stdin: stdin, stdout: stdout, pStderr: pStderr, szProgram: szProgram}
	// 시한 타이머는 잠금 아래에서 건다: 타이머가 아주 빨리 울려도 timerWait 를 다 넣은 뒤에 읽는다.
	pConn.mtxState.Lock()
	pConn.timerWait = time.AfterFunc(G_durCommandFirstByte, func() {
		pConn.mtxState.Lock()
		pConn.bTimedOut = true
		pConn.mtxState.Unlock()
		_ = pConn.Close()
	})
	pConn.mtxState.Unlock()
	return pConn, nil
}

// Read 는 명령의 표준 출력을 읽는다. 명령이 끝나 더 읽을 것이 없으면 표준 오류의 끝부분을 붙인 EOF 를 돌려준다.
func (pConn *commandConn) Read(bBuf []byte) (int, error) {
	nRead, errRead := pConn.stdout.Read(bBuf)
	if nRead > 0 {
		pConn.onceFirst.Do(pConn.stopTimer)
	}
	if errRead == nil {
		return nRead, nil
	}
	… 시한으로 끝났으면 be.proxy.cmdTimeout, EOF 면 "%w (%s: %s)" 로 표준 오류를 붙인다
}

// Close 는 표준 입력을 닫고 명령을 끝낸다. 여러 번 불러도 한 번만 한다.
func (pConn *commandConn) Close() error {
	pConn.onceClose.Do(func() {
		pConn.stopTimer()
		_ = pConn.stdin.Close()
		if pConn.pCmd.Process != nil {
			_ = pConn.pCmd.Process.Kill()
		}
		go func() { _ = pConn.pCmd.Wait() }() // 끝난 명령을 거둔다 (좀비 방지)
	})
	return nil
}
// LocalAddr·RemoteAddr 는 표시용 주소, SetDeadline 들은 아무것도 하지 않는다(x/crypto/ssh 의 NewClientConn 은 쓰지 않는다).
```

### 2.2 OS 별 실행 (build tag)

`internal/sshconn/proxy_command_windows.go`:

```go
// shellCommand 는 명령 줄을 cmd.exe 로 실행하는 명령을 만든다 (명령 프롬프트에 친 것과 같은 해석).
// 인자 따옴표를 Go 가 다시 감싸지 않도록 명령 줄을 그대로 넘긴다.
func shellCommand(szLine string) *exec.Cmd {
	szComSpec := os.Getenv("ComSpec")
	if szComSpec == "" {
		szComSpec = filepath.Join(os.Getenv("SystemRoot"), "System32", "cmd.exe")
	}
	pCmd := exec.Command(szComSpec)
	pCmd.SysProcAttr = &syscall.SysProcAttr{
		CmdLine:       `"` + szComSpec + `" /d /s /c "` + szLine + `"`,
		CreationFlags: g_ciCreateNoWindow,
	}
	return pCmd
}
```

`/d` 는 AutoRun 레지스트리를 건너뛰고, `/s` 는 바깥 따옴표 한 쌍만 벗겨 안의 따옴표(`"C:\Program Files\…\ncat.exe"`)를 그대로 둔다.

`internal/sshconn/proxy_command_unix.go`: `exec.Command("/bin/sh", "-c", szLine)`.

### 2.3 연결 경로 — `proxy.go`, `client.go`

```go
type Proxy struct {
	Type, Host string; Port int; Username, Password string
	Command  string // Type 이 command 일 때 실행할 명령 줄 (%h %p %r, D152). 그때 Host·Port 는 쓰지 않는다
}

func dialDirect(tgt *Target, szAddr string) (net.Conn, error) {
	if tgt.Proxy == nil {
		return net.DialTimeout("tcp", szAddr, G_cdurConnectTimeout)
	}
	if tgt.Proxy.Type == G_cszProxyCommand {
		return dialCommand(tgt.Proxy.Command, tgt.Host, tgt.Port, tgt.Username)
	}
	return dialViaProxy(tgt.Proxy, tgt.Host, tgt.Port, G_cdurConnectTimeout)
}

// client.go Connect — 안내 줄
		if tgt.Proxy != nil && tgt.Proxy.Type == G_cszProxyCommand {
			notice(i18n.T("be.proxy.viaCommand", "program", commandProgram(tgt.Proxy.Command)))
		} else if tgt.Proxy != nil {
			notice(i18n.T("be.proxy.via", "addr", …))
		}
```

### 2.4 앱 — `internal/app/channels_session.go`, `ssh_app.go`

```go
type proxyRecord struct {
	M_szType    string `json:"type"`    // http | socks5 | command
	M_szCommand string `json:"command"` // type 이 command 일 때 실행할 명령 줄 (D152)
	…
}

func (pService *Service) buildProxy(…) *sshconn.Proxy {
	if pRecord != nil && pRecord.M_szType == sshconn.G_cszProxyCommand {
		// 프록시 명령(D152): 주소·계정 없이 명령 줄만 쓴다. 비어 있으면 바로 연결한다.
		if strings.TrimSpace(pRecord.M_szCommand) == "" {
			return nil
		}
		return &sshconn.Proxy{Type: sshconn.G_cszProxyCommand, Command: pRecord.M_szCommand}
	}
	… (http·socks5 는 전과 같음)
```

### 2.5 화면 — `HostEditor.tsx`, `lib/hostForm.ts`, `lib/hostSave.ts`, `HostEditorParts.tsx`

- 드롭다운 마지막 항목 `proxy.command` "프록시 명령 (ProxyCommand)". 고르면 주소·계정 카드 대신 "명령" 입력(고정폭, 플레이스홀더 `ssh -W %h:%p bastion`)과 도움말 `proxy.commandHelp`. 경유 서버가 있으면 기존 안내 `proxy.jumpNote`.
- 검사: 명령이 비면 "고급" 탭으로, `hostForm.needProxyCommand` "프록시 명령을 입력하세요".
- 저장: `proxy = { type: 'command', host: '', port: 0, username: '', credentialId: null, command }`.
- 타입: `ProxyType = 'http' | 'socks5' | 'command'`, `ProxyConf.command?`, 기본 포트 표는 `Exclude<ProxyType, 'command'>` 만, `proxyTypeOf()` 로 저장값을 읽는다.

### 2.6 문구 (10개 언어)

| 키 | 한국어 |
|---|---|
| `proxy.command` · `proxy.commandLine` | 프록시 명령 (ProxyCommand) · 명령 |
| `proxy.commandHelp` | 이 명령의 입출력을 서버까지의 통로로 씁니다. %h 는 서버 주소, %p 는 포트, %r 은 사용자 이름으로 바뀝니다. 명령 프롬프트(cmd)에서 실행한 것과 같습니다. |
| `hostForm.needProxyCommand` | 프록시 명령을 입력하세요 |
| `be.proxy.viaCommand` | 프록시 명령 {program} 으로 연결하는 중... |
| `be.proxy.cmdEmpty` · `be.proxy.cmdStart` · `be.proxy.cmdTimeout` | 비어 있음 · 실행 못 함 `{program}: {error}` · `{sec}`초 응답 없음 |

## 3. 명세

spec/protocols.md §2.7a(새 절), §2.7 의 "범위 밖"에서 프록시 명령을 뺐다. spec/data/store.md §3 `proxy`, spec/ui/host-editor.md §1.3.

## 4. 테스트 — `internal/sshconn/proxy_command_test.go`

시험은 **이 테스트 실행 파일 자신을 "명령"으로 띄운다**(`TestHelperProxyCommand`, 환경 변수 `DT_PROXYCMD_MODE` 로 echo·fail·silent·bridge). 그래서 Windows 에서는 실제로 `cmd.exe /d /s /c "…"` 를 거친다.

| 테스트 | 확인 |
|---|---|
| TestExpandProxyCommand | `%h %p %r %%`, 모르는 `%x` 와 끝의 `%` 는 그대로 |
| TestCommandProgram | 첫 낱말, 따옴표 경로(`"C:\Program Files\Nmap\ncat.exe"`) |
| TestDialCommandEcho | 쓰면 되돌아온다(입출력이 연결처럼) |
| TestDialCommandFailShowsStderr | 명령이 곧바로 끝나면 EOF 이고 오류에 표준 오류(`boom: no route to bastion`) |
| TestDialCommandFirstByteTimeout | 시한(시험에서 0.3초) 안에 아무것도 안 오면 읽기가 오류로 끝난다 |
| TestDialCommandEmpty | 빈 명령은 띄우지 않는다 |
| **TestSSHConnectThroughProxyCommand** | **통합**: 도우미가 `%h %p` 로 시험용 SSH 서버에 TCP 중계 → `Connect` 가 핸드셰이크·비밀번호 재시도(명령을 다시 띄움)까지 성공, 안내 줄이 있다 |

`go vet`, `go test -race -count=1 -run 'ProxyCommand|DialCommand|CommandProgram' ./internal/sshconn/` 통과.

## 5. 검증 결과 (2026-11-27)

전체 `go test -race ./...`(clipboard KI-013 재실행 통과), tsc, vitest 465개, `wails build` 성공.

## 6. 위험

- 명령은 사용자 권한으로 실행된다. 서버 목록을 git 동기화로 받는 다른 PC 에서도 그 명령이 실행된다 — 동기화 저장소는 비공개이고 파일은 앱 잠금 비밀번호로 암호화돼 있지만, 남이 만든 백업을 가져올 때는 프록시 명령을 확인해야 한다. 도움말에 "그 프로그램이 이 PC 에 설치돼 있어야 합니다"만 적었다(경고는 후속 검토).
- 명령이 표준 출력에 SSH 가 아닌 글자를 먼저 내면(배너 등) 핸드셰이크가 실패한다 — OpenSSH 도 같다.

## H. 수동 검증

| # | 하는 일 | 기대 | 결과 |
|---|---|---|---|
| 1 | 서버 편집 > 고급 > 프록시 경유 > 프록시 명령, 명령 `ssh -W %h:%p 경유서버` (Windows OpenSSH 와 키가 있어야 함) | 저장된다 | |
| 2 | 연결 | "프록시 명령 ssh 으로 연결하는 중..." 뒤 정상 접속 | |
| 3 | 명령을 `ssh -W %h:%p 없는호스트` 로 | 실패 안내에 ssh 의 오류 끝부분 | |
| 4 | 명령을 `cmd /c pause` 처럼 아무 출력 없이 멈추는 것으로 | 20초 뒤 "응답하지 않아 끝냈습니다" | |
| 5 | 명령 칸을 비우고 저장 | "프록시 명령을 입력하세요", 고급 탭으로 | |
| 6 | 경유 서버를 정한 서버에서 프록시 명령 | "경유 서버가 있으면…" 안내, 경유 서버 쪽 프록시만 쓴다 | |

## 체크리스트

- [x] 명령 연결 · [x] OS 별 실행 · [x] 연결 경로·안내 · [x] 앱 · [x] 화면·검사·저장 · [x] 문구 · [x] 명세 · [x] 테스트(통합 포함) · [ ] §H
