package sshconn

import (
	"fmt"
	"io"
	"net"
	"sync"

	"golang.org/x/crypto/ssh"

	"termgo/internal/model"
	"termgo/internal/transport"
)

// Shell 은 SSH 셸 채널 하나 (transport.Transport 구현). SFTP 등은 Conn.Client 를 공유한다.
type Shell struct {
	Conn      *Conn
	sess      *ssh.Session
	stdin     io.WriteCloser
	once      *transport.OnceClose
	mu        sync.Mutex
	listeners []net.Listener
	bClosing  bool
}

// StartShell 은 접속(점프 포함) → PTY 요청 → 셸 시작 → 포워딩 설정까지 수행한다.
func StartShell(d Deps, h model.Host, c Context, cb transport.Callbacks) (*Shell, error) {
	conn, err := Dial(d, h, c, 0)
	if err != nil {
		return nil, err
	}
	sess, err := conn.Client.NewSession()
	if err != nil {
		conn.Close()
		return nil, err
	}
	szTerm := h.TermType
	if szTerm == "" {
		szTerm = "xterm-256color"
	}
	modes := ssh.TerminalModes{ssh.ECHO: 1, ssh.TTY_OP_ISPEED: 14400, ssh.TTY_OP_OSPEED: 14400}
	if err := sess.RequestPty(szTerm, max(2, c.Rows), max(2, c.Cols), modes); err != nil {
		conn.Close()
		return nil, err
	}
	stdin, err := sess.StdinPipe()
	if err != nil {
		conn.Close()
		return nil, err
	}
	stdout, err := sess.StdoutPipe()
	if err != nil {
		conn.Close()
		return nil, err
	}
	stderr, err := sess.StderrPipe()
	if err != nil {
		conn.Close()
		return nil, err
	}
	if err := sess.Shell(); err != nil {
		conn.Close()
		return nil, err
	}
	pstrS := &Shell{Conn: conn, sess: sess, stdin: stdin, once: transport.NewOnceClose(cb.OnClose)}
	pump := func(r io.Reader) {
		arr := make([]byte, 32*1024)
		for {
			n, err := r.Read(arr)
			if n > 0 {
				cb.OnData(append([]byte(nil), arr[:n]...))
			}
			if err != nil {
				return
			}
		}
	}
	go pump(stdout)
	go pump(stderr)
	go func() {
		_ = sess.Wait()
		pstrS.finish("")
	}()
	go func() {
		_ = conn.Client.Wait()
		pstrS.finish("연결이 종료되었습니다")
	}()
	pstrS.setupForwards(h, c)
	return pstrS, nil
}

func (pstrS *Shell) finish(szReason string) {
	pstrS.mu.Lock()
	bClosing := pstrS.bClosing
	for _, l := range pstrS.listeners {
		_ = l.Close()
	}
	pstrS.mu.Unlock()
	if bClosing {
		szReason = ""
	}
	pstrS.once.Fire(szReason)
	pstrS.Conn.Close()
}

func (pstrS *Shell) setupForwards(h model.Host, c Context) {
	cl := pstrS.Conn.Client
	for _, f := range h.Forwards {
		szBind := f.BindHost
		if szBind == "" {
			szBind = "127.0.0.1"
		}
		szDest := net.JoinHostPort(f.DestHost, fmt.Sprint(f.DestPort))
		var l net.Listener
		var err error
		if f.Type == "L" {
			l, err = net.Listen("tcp", net.JoinHostPort(szBind, fmt.Sprint(f.BindPort)))
		} else {
			l, err = cl.Listen("tcp", net.JoinHostPort(szBind, fmt.Sprint(f.BindPort)))
		}
		if err != nil {
			c.info(fmt.Sprintf("포트 포워딩 %s%d 실패: %v", f.Type, f.BindPort, err))
			continue
		}
		c.info(fmt.Sprintf("포워딩 %s:%d → %s", szBind, f.BindPort, szDest))
		pstrS.mu.Lock()
		pstrS.listeners = append(pstrS.listeners, l)
		pstrS.mu.Unlock()
		isLocal := f.Type == "L"
		go func() {
			for {
				in, err := l.Accept()
				if err != nil {
					return
				}
				go func() {
					var out net.Conn
					var err error
					if isLocal {
						out, err = cl.Dial("tcp", szDest)
					} else {
						out, err = net.Dial("tcp", szDest)
					}
					if err != nil {
						_ = in.Close()
						return
					}
					go func() { _, _ = io.Copy(out, in); _ = out.Close() }()
					_, _ = io.Copy(in, out)
					_ = in.Close()
				}()
			}
		}()
	}
}

// Write 는 셸 입력.
func (pstrS *Shell) Write(arr []byte) error {
	_, err := pstrS.stdin.Write(arr)
	return err
}

// Resize 는 PTY 창 크기를 바꾼다.
func (pstrS *Shell) Resize(iCols, iRows int) {
	if iCols > 1 && iRows > 1 {
		_ = pstrS.sess.WindowChange(iRows, iCols)
	}
}

// Close 는 사용자가 연결을 닫을 때.
func (pstrS *Shell) Close() {
	pstrS.mu.Lock()
	pstrS.bClosing = true
	pstrS.mu.Unlock()
	pstrS.finish("")
}

var _ transport.Transport = (*Shell)(nil)
