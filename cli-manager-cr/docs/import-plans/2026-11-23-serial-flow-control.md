# 시리얼 흐름 제어 (RTS/CTS · XON/XOFF) — 배경·설계·코드·검증

**작성**: 2026-11-23 · **상태**: 코드·자동 테스트 완료(D108). 실제 장비 확인은 수동 검증으로 남음 · **대상**: cli-manager-cr(DariTerm)
**묶음**: 빠진 기능 10건 중 3번. 앞 문서 [pageant-agent](./2026-11-23-pageant-agent.md), 뒤 문서 [socks-dynamic-forward](./2026-11-23-socks-dynamic-forward.md).

---

## 0. 한 줄 요약

서버 편집의 시리얼 "흐름 제어"(없음 / RTS/CTS / XON/XOFF)는 **저장만 하고 쓰지 않았다**(화면에도 그렇게 안내). 이제 고른 방식대로 동작한다: 장치가 받을 수 있을 때만 입력을 보낸다.

| 방식 | 동작 |
|---|---|
| 없음 | 전과 같다(한 번에 보낸다) |
| RTS/CTS | 열 때 RTS 를 올린다. 보낼 때 **장치의 CTS 가 켜져 있을 때만** 보낸다 |
| XON/XOFF | 장치가 XOFF(0x13)를 보내면 XON(0x11)이 올 때까지 보내지 않는다. 두 문자는 화면 출력에서 뺀다 |

두 방식 모두 입력을 16바이트 조각으로 나눠 조각마다 장치 상태를 다시 본다. 장치가 2초 넘게 받지 못하면 그 입력을 버리고 노랑 안내를 낸다.

## 1. 배경 (원인)

- [internal/transport/serial.go](../../internal/transport/serial.go) `SerialOptions` 에 흐름 제어 필드가 없었고, [internal/app/dialers.go](../../internal/app/dialers.go) 도 저장값 `serial.flowControl` 을 넘기지 않았다.
- 쓰는 라이브러리 `go.bug.st/serial` v1.8.0 의 `Mode` 에는 흐름 제어 설정이 없다(속도·데이터 비트·패리티·정지 비트·초기 RTS/DTR 뿐). 드라이버에 흐름 제어를 맡기려면 Windows 의 DCB 를 직접 고쳐야 하는데, 라이브러리가 포트 핸들을 내주지 않는다.

## 2. 설계 결정

| 항목 | 정한 것 | 이유 |
|---|---|---|
| 어디서 하나 | **앱이 한다**: 보내기 전에 `GetModemStatusBits().CTS` 를 보거나(RTS/CTS), 받은 XOFF/XON 을 기억해 본다(XON/XOFF) | 라이브러리의 공개 기능만 쓰므로 Windows·Linux·macOS 에서 같은 코드가 돈다(공통 규칙 §8 OS 중립). 드라이버 방식은 OS 마다 따로 짜야 하고 라이브러리 내부를 건드려야 한다 |
| 조각 크기 | 16바이트 | 조각 사이에만 상태를 보므로 작을수록 장치가 "그만"이라고 한 뒤 더 나가는 양이 줄어든다. 115200bps 에서 16바이트는 약 1.4ms |
| 기다리는 시간 | 2초(`G_durSerialFlowWait`), 조각이 하나 나가면 다시 2초 | 입력 경로가 오래 막히지 않게. 케이블이 빠졌거나 설정이 틀리면 CTS 가 영영 안 올라온다 |
| 못 보낸 입력 | **버린다** + 노랑 안내 한 줄(막힌 동안 한 번만) | 쌓아 두었다가 한참 뒤에 한꺼번에 나가면 더 위험하다(장비 콘솔에 옛 명령이 실행된다) |
| 버린 입력의 로컬 에코 | 하지 않는다 | 화면에 보이면 보낸 것으로 오해한다 |
| CTS 를 읽을 수 없는 장치 | 받을 수 있다고 본다 | 일부 USB 변환기는 모뎀 상태를 주지 않는다. 영영 막히는 것보다 낫다 |
| 받는 방향 | 조절하지 않는다(RTS 를 내리지 않고 XOFF 를 보내지 않는다) | 앱은 받은 것을 곧바로 읽어 가므로 PC 쪽이 넘칠 일이 없다 |
| 포트 닫기 | 기다리던 보내기를 곧바로 끝낸다 | 세션을 닫았는데 2초 매달리지 않게 |

### 2.1 드라이버 방식과의 차이 (알려진 한계)

- 조각이 나가는 도중에 장치가 CTS 를 내리면 그 조각(최대 16바이트)은 끝까지 나간다. 드라이버 방식은 바이트 단위로 멈춘다. 받는 쪽 버퍼가 16바이트도 남기지 않고 "그만"을 알리는 장치에서는 넘칠 수 있다.
- XON/XOFF 는 장치가 보내는 **이진 데이터 안의 0x11·0x13 도 흐름 제어 문자로 본다**(화면에서 빠진다). XON/XOFF 방식 자체의 성질이다. 이진 전송(ZMODEM 등)을 할 장치에는 RTS/CTS 를 쓴다.

## A. 바뀐 파일 한눈에

| 파일 | 종류 | 상태 |
|---|---|---|
| `frontend/src/screens/HostEditorParts.tsx` | 코드 | 수정 |
| `internal/app/dialers.go` | 코드 | 수정 |
| `internal/transport/serial.go` | 코드 | 수정 |
| `internal/transport/serial_flow_test.go` | 테스트 | 새 파일 |
| `internal/transport/serial_test.go` | 테스트 | 수정 |
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

아래의 `diff` 는 **이 묶음의 앞 문서들을 차례로 적용한 상태**를 기준으로 뽑았다. 앞 문서에서도 고친 파일(문맥 줄에 앞 기능의 코드가 섞여 있을 수 있다): `spec/protocols.md`, `spec/ui/host-editor.md`. 이 기능만 따로 넣을 때는 `+`/`-` 줄만 옮기면 된다.

## B. 코드

### `frontend/src/screens/HostEditorParts.tsx` — 수정

```diff
--- a/frontend/src/screens/HostEditorParts.tsx
+++ b/frontend/src/screens/HostEditorParts.tsx
@@ -128,7 +128,7 @@
         </Field>
       </div>
       <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 14 }}>
-        <Field label={t('serial.flow')} help={t('serial.flowNotYet')}>
+        <Field label={t('serial.flow')} help={t('serial.flowHelp')}>
           <select className="input" value={value.flowControl} onChange={(e) => set('flowControl', e.target.value as SerialFlow)}>
             {FLOW.map((o) => <option key={o.value} value={o.value}>{o.value === 'none' ? t('serial.flowNone') : o.label}</option>)}
           </select>
```

### `internal/app/dialers.go` — 수정

```diff
--- a/internal/app/dialers.go
+++ b/internal/app/dialers.go
@@ -64,6 +64,7 @@
 			opts.EnterSends = record.M_pSerial.M_szEnterSends
 		}
 		opts.LocalEcho = record.M_pSerial.M_bLocalEcho
+		opts.FlowControl = record.M_pSerial.M_szFlowControl // none · rtscts · xonxoff (protocols.md §5, D108)
 	}
 	if opts.Path == "" {
 		return nil, i18n.NewError("be.serial.noDevice")
```

### `internal/transport/serial.go` — 수정

```diff
--- a/internal/transport/serial.go
+++ b/internal/transport/serial.go
@@ -16,6 +16,25 @@
 
 // G_cdurSerialReadTick 은 읽기 시한이다. 이 주기로 Read 가 돌아와 대기 중인 로컬 에코를 내보낸다 (protocols.md §5).
 const G_cdurSerialReadTick = 50 * time.Millisecond
+
+// 흐름 제어 (protocols.md §5, D108). 드라이버가 아니라 앱이 한다: 보내기 전에 장치가 받을 수 있는지 확인한다.
+const (
+	// G_cszFlowNone·G_cszFlowRtsCts·G_cszFlowXonXoff 는 서버 설정 serial.flowControl 의 값이다.
+	G_cszFlowNone    = "none"
+	G_cszFlowRtsCts  = "rtscts"
+	G_cszFlowXonXoff = "xonxoff"
+	// g_ciSerialFlowChunk 는 흐름 제어가 켜져 있을 때 한 번에 보내는 바이트 수다. 조각 사이마다 장치 상태를 다시 본다.
+	g_ciSerialFlowChunk = 16
+	// g_cdurSerialFlowPoll 은 장치가 받을 수 있게 되기를 기다리며 상태를 다시 보는 간격이다.
+	g_cdurSerialFlowPoll = 5 * time.Millisecond
+	// g_cbXon·g_cbXoff 는 XON/XOFF 흐름 제어 문자(DC1·DC3)다.
+	g_cbXon  = 0x11
+	g_cbXoff = 0x13
+)
+
+// G_durSerialFlowWait 는 장치가 받을 수 있게 되기를 기다리는 최대 시간이다. 넘으면 남은 입력을 버리고 안내한다.
+// 시험에서 줄일 수 있게 변수로 둔다.
+var G_durSerialFlowWait = 2 * time.Second
 
 // SerialOptions 는 시리얼 열기 설정이다 (protocols.md §5, data/store.md §3 serial).
 type SerialOptions struct {
@@ -26,6 +45,8 @@
 	StopBits   float64
 	EnterSends string
 	LocalEcho  bool
+	// FlowControl 은 흐름 제어 방식이다: none(기본) · rtscts · xonxoff.
+	FlowControl string
 }
 
 // SerialConn 은 열린 시리얼 포트 하나다.
@@ -35,6 +56,9 @@
 	mtxWrite    sync.Mutex
 	mtxState    sync.Mutex
 	bPendingOut []byte
+	bPeerXoff   bool // XON/XOFF: 장치가 XOFF 를 보내 멈춰 달라고 한 상태 (mtxState)
+	bFlowWarned bool // 흐름 제어로 입력을 버렸다는 안내를 이미 냈다 (다시 보낼 수 있게 되면 지운다) (mtxState)
+	bClosed     bool // Close 가 불렸다: 기다리던 쓰기를 끝낸다 (mtxState)
 }
 
 // OpenSerial 은 시리얼 장치를 연다. 속도·데이터 비트는 0 이면 기본값 (9600, 8).
@@ -62,7 +86,16 @@
 		_ = port.Close()
 		return nil, fmt.Errorf("%s: %v", opts.Path, errTimeout)
 	}
-	return &SerialConn{port: port, opts: opts}, nil
+	return newSerialConn(port, opts), nil
+}
+
+// newSerialConn 은 열린 포트를 감싼다. RTS/CTS 면 RTS 를 올려 "이 PC 는 받을 수 있다"고 알린다
+// (이 앱은 받은 데이터를 곧바로 읽어 가므로 RTS 를 내리지 않는다).
+func newSerialConn(port serial.Port, opts SerialOptions) *SerialConn {
+	if opts.FlowControl == G_cszFlowRtsCts {
+		_ = port.SetRTS(true)
+	}
+	return &SerialConn{port: port, opts: opts}
 }
 
 func parityOf(szName string) serial.Parity {
@@ -121,14 +154,97 @@
 		if nRaw == 0 {
 			continue // 시한: 데이터 없음. 위에서 대기분을 다시 본다.
 		}
-		nCopy := copy(bBuf, bRaw[:nRaw])
-		if nCopy < nRaw {
+		bData := bRaw[:nRaw]
+		if pSerial.opts.FlowControl == G_cszFlowXonXoff {
+			bData = pSerial.takeXonXoff(bData)
+			if len(bData) == 0 {
+				continue // 흐름 제어 문자뿐이었다
+			}
+		}
+		nCopy := copy(bBuf, bData)
+		if nCopy < len(bData) {
 			pSerial.mtxState.Lock()
-			pSerial.bPendingOut = append(pSerial.bPendingOut, bRaw[nCopy:nRaw]...)
+			pSerial.bPendingOut = append(pSerial.bPendingOut, bData[nCopy:]...)
 			pSerial.mtxState.Unlock()
 		}
 		return nCopy, nil
 	}
+}
+
+// takeXonXoff 는 받은 데이터에서 XON·XOFF 를 빼고(화면에 보이지 않게) 장치의 멈춤 상태를 갱신한다.
+func (pSerial *SerialConn) takeXonXoff(bData []byte) []byte {
+	bOut := bData[:0]
+	pSerial.mtxState.Lock()
+	defer pSerial.mtxState.Unlock()
+	for _, bOne := range bData {
+		switch bOne {
+		case g_cbXoff:
+			pSerial.bPeerXoff = true
+		case g_cbXon:
+			pSerial.bPeerXoff = false
+		default:
+			bOut = append(bOut, bOne)
+		}
+	}
+	return bOut
+}
+
+// peerReady 는 장치가 지금 받을 수 있는지 본다. RTS/CTS 는 CTS 선, XON/XOFF 는 마지막으로 받은 문자로 판단한다.
+// CTS 를 읽을 수 없는 장치(일부 USB 변환기)는 받을 수 있다고 본다 — 영영 막히는 것보다 낫다.
+func (pSerial *SerialConn) peerReady() bool {
+	switch pSerial.opts.FlowControl {
+	case G_cszFlowRtsCts:
+		pBits, errBits := pSerial.port.GetModemStatusBits()
+		return errBits != nil || pBits == nil || pBits.CTS
+	case G_cszFlowXonXoff:
+		pSerial.mtxState.Lock()
+		defer pSerial.mtxState.Unlock()
+		return !pSerial.bPeerXoff
+	default:
+		return true
+	}
+}
+
+// waitPeerReady 는 장치가 받을 수 있게 될 때까지 tDeadline 까지 기다린다. 받을 수 있으면 true,
+// 시한이 지났거나 포트를 닫았으면 false 다.
+func (pSerial *SerialConn) waitPeerReady(tDeadline time.Time) bool {
+	for {
+		pSerial.mtxState.Lock()
+		bClosed := pSerial.bClosed
+		pSerial.mtxState.Unlock()
+		if bClosed {
+			return false
+		}
+		if pSerial.peerReady() {
+			return true
+		}
+		if !time.Now().Before(tDeadline) {
+			return false
+		}
+		time.Sleep(g_cdurSerialFlowPoll)
+	}
+}
+
+// writeFlow 는 흐름 제어를 지키며 보낸다: 작은 조각으로 나눠 조각마다 장치가 받을 수 있는지 확인한다.
+// 실제로 보낸 길이를 돌려준다. 장치가 G_durSerialFlowWait 동안 받지 못하면 남은 것은 버린다.
+func (pSerial *SerialConn) writeFlow(bSend []byte) (int, error) {
+	tDeadline := time.Now().Add(G_durSerialFlowWait)
+	nSent := 0
+	for nSent < len(bSend) {
+		if !pSerial.waitPeerReady(tDeadline) {
+			return nSent, nil
+		}
+		nEnd := nSent + g_ciSerialFlowChunk
+		if nEnd > len(bSend) {
+			nEnd = len(bSend)
+		}
+		if _, errWrite := pSerial.port.Write(bSend[nSent:nEnd]); errWrite != nil {
+			return nSent, errWrite
+		}
+		nSent = nEnd
+		tDeadline = time.Now().Add(G_durSerialFlowWait) // 조각이 나갔으면 다음 조각은 새로 기다린다
+	}
+	return nSent, nil
 }
 
 // enterBytes 는 Enter 전송값(CR/LF/CRLF)을 돌려준다 (protocols.md §5).
@@ -148,11 +264,24 @@
 	szEnter := enterBytes(pSerial.opts.EnterSends)
 	szText := strings.ReplaceAll(string(bData), "\r", szEnter)
 	bSend := []byte(szText)
+	bFlow := pSerial.opts.FlowControl == G_cszFlowRtsCts || pSerial.opts.FlowControl == G_cszFlowXonXoff
+	nSent := len(bSend)
+	var errWrite error
 	pSerial.mtxWrite.Lock()
-	_, errWrite := pSerial.port.Write(bSend)
+	if bFlow {
+		nSent, errWrite = pSerial.writeFlow(bSend)
+	} else {
+		_, errWrite = pSerial.port.Write(bSend)
+	}
 	pSerial.mtxWrite.Unlock()
 	if errWrite != nil {
 		return ErrPortClosed
+	}
+	if bFlow {
+		pSerial.noteFlowResult(nSent == len(bSend))
+		if nSent < len(bSend) {
+			return nil // 장치가 받지 못해 버렸다: 에코도 하지 않는다 (안내는 noteFlowResult 가 냈다)
+		}
 	}
 	if pSerial.opts.LocalEcho {
 		// CR 과 CRLF 는 화면에서 줄바꿈으로 보인다.
@@ -166,6 +295,22 @@
 	return nil
 }
 
+// noteFlowResult 는 흐름 제어로 입력을 버렸을 때 화면에 노랑 안내 한 줄을 낸다.
+// 막힌 동안에는 한 번만 내고, 다시 보낼 수 있게 되면 다음에 또 낼 수 있게 한다.
+func (pSerial *SerialConn) noteFlowResult(bAllSent bool) {
+	pSerial.mtxState.Lock()
+	defer pSerial.mtxState.Unlock()
+	if bAllSent {
+		pSerial.bFlowWarned = false
+		return
+	}
+	if pSerial.bFlowWarned || pSerial.bClosed {
+		return
+	}
+	pSerial.bFlowWarned = true
+	pSerial.bPendingOut = append(pSerial.bPendingOut, []byte("\r\n\x1b[33m● "+i18n.T("be.serial.flowBlocked")+"\x1b[0m\r\n")...)
+}
+
 // Resize 는 시리얼에 아무 영향이 없다 (protocols.md §5).
 func (pSerial *SerialConn) Resize(iCols int, iRows int) error {
 	return nil
@@ -173,6 +318,9 @@
 
 // Close 는 포트를 닫는다.
 func (pSerial *SerialConn) Close() error {
+	pSerial.mtxState.Lock()
+	pSerial.bClosed = true // 흐름 제어로 기다리던 쓰기를 끝낸다
+	pSerial.mtxState.Unlock()
 	return pSerial.port.Close()
 }
 
```


## C. 테스트 코드

### `internal/transport/serial_flow_test.go` — 새 파일 (전체)

```go
package transport

import (
	"errors"
	"strings"
	"testing"
	"time"
)

// 흐름 제어 시험 (protocols.md §5, D108).

func (pFake *fakeSerialPort) setCts(bCts bool) {
	pFake.mtxData.Lock()
	pFake.bCts = bCts
	pFake.mtxData.Unlock()
}

func (pFake *fakeSerialPort) written() string {
	pFake.mtxData.Lock()
	defer pFake.mtxData.Unlock()
	return strings.Join(pFake.slWrites, "")
}

func (pFake *fakeSerialPort) writeCount() int {
	pFake.mtxData.Lock()
	defer pFake.mtxData.Unlock()
	return len(pFake.slWrites)
}

// withFlowWait 는 시험 동안 "장치를 기다리는 최대 시간"을 줄인다.
func withFlowWait(t *testing.T, durWait time.Duration) {
	t.Helper()
	durOld := G_durSerialFlowWait
	G_durSerialFlowWait = durWait
	t.Cleanup(func() { G_durSerialFlowWait = durOld })
}

// RTS/CTS: 열 때 RTS 를 올리고, CTS 가 내려가 있으면 보내지 않고 기다리다가 올라오면 16바이트 조각으로 보낸다.
func TestSerialRtsCtsWaitsForCts(t *testing.T) {
	withFlowWait(t, 2*time.Second)
	pFake := &fakeSerialPort{chIn: make(chan []byte, 1)}
	pSerial := newSerialConn(pFake, SerialOptions{FlowControl: G_cszFlowRtsCts, EnterSends: "CR"})
	pFake.mtxData.Lock()
	bRts := pFake.bRts
	pFake.mtxData.Unlock()
	if !bRts {
		t.Fatal("RTS/CTS 면 열 때 RTS 를 올려야 함")
	}
	szData := strings.Repeat("0123456789", 4) // 40바이트 → 16·16·8
	chDone := make(chan error, 1)
	go func() { chDone <- pSerial.Write([]byte(szData)) }()
	time.Sleep(60 * time.Millisecond)
	if pFake.writeCount() != 0 {
		t.Fatalf("CTS 가 내려가 있으면 보내지 않아야 함: %q", pFake.written())
	}
	pFake.setCts(true)
	select {
	case errWrite := <-chDone:
		if errWrite != nil {
			t.Fatal(errWrite)
		}
	case <-time.After(time.Second):
		t.Fatal("CTS 가 올라오면 보내야 함")
	}
	if pFake.written() != szData || pFake.writeCount() != 3 {
		t.Fatalf("16바이트 조각 3개로 전부 보내야 함: %d개 %q", pFake.writeCount(), pFake.written())
	}
}

// RTS/CTS: 장치가 끝내 받지 못하면 입력을 버리고 노랑 안내를 한 번만 낸다. 다시 받을 수 있게 되면 정상으로 돌아온다.
func TestSerialRtsCtsDropsAfterWaitAndWarnsOnce(t *testing.T) {
	withFlowWait(t, 40*time.Millisecond)
	pFake := &fakeSerialPort{chIn: make(chan []byte, 1)}
	pSerial := newSerialConn(pFake, SerialOptions{FlowControl: G_cszFlowRtsCts, EnterSends: "CR", LocalEcho: true})
	for iTry := 0; iTry < 2; iTry++ {
		if errWrite := pSerial.Write([]byte("lost")); errWrite != nil {
			t.Fatalf("버려도 오류는 아님: %v", errWrite)
		}
	}
	if pFake.writeCount() != 0 {
		t.Fatalf("받지 못하는 장치에는 보내지 않아야 함: %q", pFake.written())
	}
	bBuf := make([]byte, 512)
	nRead, _ := pSerial.Read(bBuf)
	szNotice := string(bBuf[:nRead])
	if strings.Count(szNotice, "●") != 1 {
		t.Fatalf("안내는 한 번만: %q", szNotice)
	}
	if strings.Contains(szNotice, "lost") {
		t.Fatalf("버린 입력은 에코하지 않아야 함: %q", szNotice)
	}
	pFake.setCts(true)
	if errWrite := pSerial.Write([]byte("ok")); errWrite != nil || pFake.written() != "ok" {
		t.Fatalf("다시 받을 수 있으면 보내야 함: %v %q", errWrite, pFake.written())
	}
	nRead, _ = pSerial.Read(bBuf)
	if string(bBuf[:nRead]) != "ok" {
		t.Fatalf("보낸 입력은 에코해야 함: %q", string(bBuf[:nRead]))
	}
	// 다시 막히면 안내를 또 낼 수 있다.
	pFake.setCts(false)
	_ = pSerial.Write([]byte("x"))
	nRead, _ = pSerial.Read(bBuf)
	if strings.Count(string(bBuf[:nRead]), "●") != 1 {
		t.Fatalf("다시 막히면 안내를 다시 내야 함: %q", string(bBuf[:nRead]))
	}
}

// XON/XOFF: 장치가 XOFF 를 보내면 멈추고 XON 을 보내면 다시 보낸다. 두 문자는 화면 출력에서 빠진다.
func TestSerialXonXoff(t *testing.T) {
	withFlowWait(t, 2*time.Second)
	pFake := &fakeSerialPort{chIn: make(chan []byte, 4)}
	pSerial := newSerialConn(pFake, SerialOptions{FlowControl: G_cszFlowXonXoff, EnterSends: "CR"})
	pFake.chIn <- []byte("a\x13b")
	bBuf := make([]byte, 64)
	nRead, errRead := pSerial.Read(bBuf)
	if errRead != nil || string(bBuf[:nRead]) != "ab" {
		t.Fatalf("XOFF 는 출력에서 빠져야 함: %q %v", string(bBuf[:nRead]), errRead)
	}
	chDone := make(chan error, 1)
	go func() { chDone <- pSerial.Write([]byte("hello")) }()
	time.Sleep(60 * time.Millisecond)
	if pFake.writeCount() != 0 {
		t.Fatalf("XOFF 뒤에는 보내지 않아야 함: %q", pFake.written())
	}
	// XON 만 온 조각: Read 는 돌려줄 글자가 없으므로 다음 데이터까지 계속 읽는다.
	pFake.chIn <- []byte("\x11")
	pFake.chIn <- []byte("c")
	nRead, _ = pSerial.Read(bBuf)
	if string(bBuf[:nRead]) != "c" {
		t.Fatalf("XON 은 출력에서 빠져야 함: %q", string(bBuf[:nRead]))
	}
	select {
	case errWrite := <-chDone:
		if errWrite != nil || pFake.written() != "hello" {
			t.Fatalf("XON 뒤에는 보내야 함: %v %q", errWrite, pFake.written())
		}
	case <-time.After(time.Second):
		t.Fatal("XON 뒤에는 보내야 함")
	}
}

// 기다리는 중에 포트를 닫으면 쓰기가 곧바로 끝난다. CTS 를 읽을 수 없는 장치는 받을 수 있다고 본다.
func TestSerialFlowCloseAndUnreadableCts(t *testing.T) {
	withFlowWait(t, 5*time.Second)
	pFake := &fakeSerialPort{chIn: make(chan []byte, 1)}
	pSerial := newSerialConn(pFake, SerialOptions{FlowControl: G_cszFlowRtsCts})
	chDone := make(chan struct{})
	go func() { _ = pSerial.Write([]byte("x")); close(chDone) }()
	time.Sleep(30 * time.Millisecond)
	_ = pSerial.Close()
	select {
	case <-chDone:
	case <-time.After(time.Second):
		t.Fatal("닫으면 기다리던 쓰기가 끝나야 함")
	}

	pFake2 := &fakeSerialPort{chIn: make(chan []byte, 1), errBits: errors.New("not supported")}
	pSerial2 := newSerialConn(pFake2, SerialOptions{FlowControl: G_cszFlowRtsCts})
	if errWrite := pSerial2.Write([]byte("go")); errWrite != nil || pFake2.written() != "go" {
		t.Fatalf("CTS 를 읽을 수 없으면 그대로 보내야 함: %v %q", errWrite, pFake2.written())
	}
}

// 흐름 제어 없음: 전과 같이 한 번에 보낸다(조각내지 않는다).
func TestSerialNoFlowWritesWhole(t *testing.T) {
	pFake := &fakeSerialPort{chIn: make(chan []byte, 1)}
	pSerial := newSerialConn(pFake, SerialOptions{})
	_ = pSerial.Write([]byte(strings.Repeat("z", 100)))
	if pFake.writeCount() != 1 {
		t.Fatalf("한 번에 보내야 함: %d", pFake.writeCount())
	}
}
```

### `internal/transport/serial_test.go` — 수정

```diff
--- a/internal/transport/serial_test.go
+++ b/internal/transport/serial_test.go
@@ -16,6 +16,9 @@
 	mtxData  sync.Mutex
 	slWrites []string
 	bClosed  bool
+	bCts     bool  // 장치의 CTS 선 (흐름 제어 시험용)
+	bRts     bool  // 이 PC 가 올린 RTS
+	errBits  error // 모뎀 상태를 읽지 못하는 장치 흉내
 }
 
 func (pFake *fakeSerialPort) SetMode(mode *serial.Mode) error { return nil }
@@ -43,9 +46,19 @@
 func (pFake *fakeSerialPort) ResetInputBuffer() error  { return nil }
 func (pFake *fakeSerialPort) ResetOutputBuffer() error { return nil }
 func (pFake *fakeSerialPort) SetDTR(bDtr bool) error   { return nil }
-func (pFake *fakeSerialPort) SetRTS(bRts bool) error   { return nil }
+func (pFake *fakeSerialPort) SetRTS(bRts bool) error {
+	pFake.mtxData.Lock()
+	defer pFake.mtxData.Unlock()
+	pFake.bRts = bRts
+	return nil
+}
 func (pFake *fakeSerialPort) GetModemStatusBits() (*serial.ModemStatusBits, error) {
-	return &serial.ModemStatusBits{}, nil
+	pFake.mtxData.Lock()
+	defer pFake.mtxData.Unlock()
+	if pFake.errBits != nil {
+		return nil, pFake.errBits
+	}
+	return &serial.ModemStatusBits{CTS: pFake.bCts}, nil
 }
 func (pFake *fakeSerialPort) SetReadTimeout(durTimeout time.Duration) error { return nil }
 func (pFake *fakeSerialPort) Break(durBreak time.Duration) error            { return nil }
```


## E. 문구

`locales/<언어>.json` 10개 파일에 같은 키를 넣는다(순서: ko, en, ja, zh, tw, de, fr, es, pt, ru). 한국어·영어 외에는 이 작업을 한 세션이 번역한 것이라 검수가 필요할 수 있다.

**`serial.flowHelp`** (추가)

```json
  "serial.flowHelp": "RTS/CTS는 장치의 CTS 신호가 켜져 있을 때만, XON/XOFF는 장치가 XOFF를 보내지 않았을 때만 입력을 보냅니다. 장치가 2초 넘게 받지 못하면 그 입력은 버립니다.",
  "serial.flowHelp": "RTS/CTS sends input only while the device asserts CTS; XON/XOFF sends only while the device has not sent XOFF. Input the device cannot accept for more than 2 seconds is discarded.",
  "serial.flowHelp": "RTS/CTS はデバイスの CTS 信号がオンのときだけ、XON/XOFF はデバイスが XOFF を送っていないときだけ入力を送ります。デバイスが 2 秒を超えて受け取れない入力は破棄します。",
  "serial.flowHelp": "RTS/CTS 仅在设备的 CTS 信号有效时发送输入；XON/XOFF 仅在设备未发送 XOFF 时发送。设备超过 2 秒仍无法接收的输入将被丢弃。",
  "serial.flowHelp": "RTS/CTS 僅在裝置的 CTS 訊號有效時傳送輸入；XON/XOFF 僅在裝置未傳送 XOFF 時傳送。裝置超過 2 秒仍無法接收的輸入會被捨棄。",
  "serial.flowHelp": "RTS/CTS sendet Eingaben nur, solange das Gerät CTS setzt; XON/XOFF nur, solange das Gerät kein XOFF gesendet hat. Eingaben, die das Gerät länger als 2 Sekunden nicht annehmen kann, werden verworfen.",
  "serial.flowHelp": "RTS/CTS n'envoie la saisie que lorsque l'appareil active CTS ; XON/XOFF seulement tant que l'appareil n'a pas envoyé XOFF. Une saisie que l'appareil ne peut pas accepter pendant plus de 2 secondes est abandonnée.",
  "serial.flowHelp": "RTS/CTS envía la entrada solo mientras el dispositivo activa CTS; XON/XOFF solo mientras el dispositivo no haya enviado XOFF. La entrada que el dispositivo no puede aceptar durante más de 2 segundos se descarta.",
  "serial.flowHelp": "RTS/CTS envia a entrada somente enquanto o dispositivo ativa CTS; XON/XOFF somente enquanto o dispositivo não tiver enviado XOFF. A entrada que o dispositivo não consegue aceitar por mais de 2 segundos é descartada.",
  "serial.flowHelp": "RTS/CTS отправляет ввод, только пока устройство держит сигнал CTS; XON/XOFF — только пока устройство не прислало XOFF. Ввод, который устройство не может принять дольше 2 секунд, отбрасывается.",
```

**`be.serial.flowBlocked`** (추가)

```json
  "be.serial.flowBlocked": "장치가 받을 준비가 되지 않아 입력을 보내지 못했습니다 (흐름 제어)",
  "be.serial.flowBlocked": "The device is not ready to receive, so the input was not sent (flow control)",
  "be.serial.flowBlocked": "デバイスが受信可能な状態でないため、入力を送信できませんでした（フロー制御）",
  "be.serial.flowBlocked": "设备尚未准备好接收，输入未发送（流控制）",
  "be.serial.flowBlocked": "裝置尚未準備好接收，輸入未傳送（流量控制）",
  "be.serial.flowBlocked": "Das Gerät ist nicht empfangsbereit, die Eingabe wurde nicht gesendet (Flusssteuerung)",
  "be.serial.flowBlocked": "L'appareil n'est pas prêt à recevoir ; la saisie n'a pas été envoyée (contrôle de flux)",
  "be.serial.flowBlocked": "El dispositivo no está listo para recibir; la entrada no se envió (control de flujo)",
  "be.serial.flowBlocked": "O dispositivo não está pronto para receber; a entrada não foi enviada (controle de fluxo)",
  "be.serial.flowBlocked": "Устройство не готово к приёму, ввод не отправлен (управление потоком)",
```

**`serial.flowNotYet`** (삭제)

10개 언어 파일 모두에서 이 키를 지운다.


## F. 명세

### `spec/protocols.md` — 수정

```diff
--- a/spec/protocols.md
+++ b/spec/protocols.md
@@ -101,7 +101,7 @@
 
 ## 5. 시리얼
 
-- 장치 열기: 속도(0 이면 9600), 데이터 비트(0 이면 8), 패리티(none/even/odd/mark/space), 정지 비트(1/1.5/2). 흐름 제어 설정값(`flowControl`)은 저장하지만 현재 동작에 쓰지 않는다(알려진 한계). 서버 편집 화면에 이 사실을 안내한다(ui/host-editor.md §1). 열기 실패 시 오류 메시지는 `{장치}: {원인}`.
+- 장치 열기: 속도(0 이면 9600), 데이터 비트(0 이면 8), 패리티(none/even/odd/mark/space), 정지 비트(1/1.5/2). **흐름 제어**(`flowControl`, 2026-11-23, D108 — 전에는 저장만 하고 쓰지 않았다): `none`(기본)은 입력을 그대로 보낸다. `rtscts` 는 열 때 RTS 선을 올려 두고, 보낼 때 **장치의 CTS 선이 켜져 있을 때만** 보낸다. `xonxoff` 는 장치가 XOFF(0x13)를 보낸 뒤부터 XON(0x11)을 보낼 때까지 보내지 않으며, 장치가 보낸 XON·XOFF 는 화면 출력에서 뺀다. 두 방식 모두 입력을 **16바이트 조각**으로 나눠 조각마다 장치 상태를 다시 본다. 장치가 **2초** 넘게 받을 수 없으면 그 입력의 남은 부분을 버리고 터미널에 노랑 안내 "장치가 받을 준비가 되지 않아 입력을 보내지 못했습니다 (흐름 제어)" 를 낸다(막혀 있는 동안 한 번만, 다시 보낼 수 있게 된 뒤 또 막히면 다시). 버린 입력은 로컬 에코도 하지 않는다. CTS 상태를 읽을 수 없는 장치는 받을 수 있는 것으로 본다. 포트를 닫으면 기다리던 보내기는 곧바로 끝난다. 받는 방향(장치 → 이 PC)은 조절하지 않는다(RTS 를 내리거나 XOFF 를 보내지 않는다 — 앱이 받은 데이터를 곧바로 읽어 가기 때문). 열기 실패 시 오류 메시지는 `{장치}: {원인}`.
 - 읽기 단위 4KB. 포트가 닫히면 이유 "포트가 닫혔습니다"(사용자가 닫은 경우는 이유 없음). **시리얼은 자동 재접속을 하지 않는다**.
 - 읽기는 50ms 시한으로 돌아와 그사이 쌓인 로컬 에코를 바로 내보낸다(장치가 아무것도 보내지 않아도 입력한 글자가 즉시 보인다).
 - 쓰기: 입력의 CR(`\r`)을 **Enter 전송값**(CR/LF/CRLF)으로 바꿔 보낸다. **로컬 에코**가 켜져 있으면 보낸 내용을 그대로 화면에도 출력하되 CR 과 CRLF 는 화면에서 줄바꿈(`\r\n`)으로 보이게 한다.
```

### `spec/ui/host-editor.md` — 수정

```diff
--- a/spec/ui/host-editor.md
+++ b/spec/ui/host-editor.md
@@ -19,7 +19,7 @@
 3. 프로토콜별 영역:
    - **로컬**: 2열 — "시작 폴더"(고정폭 입력 + [폴더 선택] 버튼(아이콘, 툴팁 "폴더 선택"), 도움말 "비워 두면 홈 폴더에서 열립니다", 플레이스홀더 `C:\Work\프로젝트`) / "셸"(드롭다운: 첫 항목 "기본 ({기본 셸 이름})" = 값 없음, 이어서 이 PC에서 찾은 셸 목록)
    - **SSH / Telnet**: 3열 — "주소 (IP 또는 도메인)"(플레이스홀더 "10.0.0.11  또는  user@host:22") / "포트"(숫자) / "사용자 이름"(플레이스홀더 "root"). **호스트 칸에 `사용자@호스트[:포트]` 를 입력하면 자동으로 사용자·호스트·포트 칸으로 나눠 채운다.**
-   - **시리얼**: 2열 — "포트"(장치 경로 입력 + 이 PC의 장치 목록 자동완성(항목에 설명 라벨), 라벨 옆 [새로고침] 링크, 플레이스홀더 "COM3") / "속도 (baud)"(드롭다운 1200, 2400, 4800, 9600, 19200, 38400, 57600, 115200, 230400, 460800, 921600). 2열 — "데이터 / 패리티 / 정지 비트"(드롭다운 3개: 8·7·6·5 / none·even·odd·mark·space / 1·1.5·2), "흐름 제어 / Enter 전송값"(드롭다운 2개: 없음·RTS/CTS·XON/XOFF / CR·LF·CRLF). 그 아래 작은 도움말 "현재 버전은 흐름 제어 설정을 적용하지 않습니다(저장만)". 토글 "입력 글자 표시 (로컬 에코)"(설명 "장비가 입력한 글자를 돌려주지 않을 때 켜세요"). 프로토콜을 시리얼로 고르면 장치 목록을 조회한다.
+   - **시리얼**: 2열 — "포트"(장치 경로 입력 + 이 PC의 장치 목록 자동완성(항목에 설명 라벨), 라벨 옆 [새로고침] 링크, 플레이스홀더 "COM3") / "속도 (baud)"(드롭다운 1200, 2400, 4800, 9600, 19200, 38400, 57600, 115200, 230400, 460800, 921600). 2열 — "데이터 / 패리티 / 정지 비트"(드롭다운 3개: 8·7·6·5 / none·even·odd·mark·space / 1·1.5·2), "흐름 제어 / Enter 전송값"(드롭다운 2개: 없음·RTS/CTS·XON/XOFF / CR·LF·CRLF). 그 아래 작은 도움말 "RTS/CTS는 장치의 CTS 신호가 켜져 있을 때만, XON/XOFF는 장치가 XOFF를 보내지 않았을 때만 입력을 보냅니다. 장치가 2초 넘게 받지 못하면 그 입력은 버립니다."(2026-11-23, D108). 토글 "입력 글자 표시 (로컬 에코)"(설명 "장비가 입력한 글자를 돌려주지 않을 때 켜세요"). 프로토콜을 시리얼로 고르면 장치 목록을 조회한다.
 4. 2열 — "폴더"(드롭다운: "(최상위)" + 모든 폴더를 `상위 / 하위` 경로 형식으로 이름순 정렬) / "태그 (쉼표로 구분)"(쉼표로 나눠 공백 제거·빈 항목 제거, 플레이스홀더 "web, nginx").
 5. **환경**(세그먼트: 일반·개발·기능테스트·성능테스트·운영·장비, 각 옵션 앞에 해당 색 점; 순서는 none, dev, stage, perf, prod, device). 도움말 "운영 서버는 빨간 테두리로 표시되고 위험 명령 실행 전에 확인을 받습니다".
 6. **색상 (지정하지 않으면 환경/폴더 색을 따름)** — 색 선택(데이터 명세 §6).
```



---

## G. 검증 (2026-11-23 실행)

| 테스트 ([internal/transport/serial_flow_test.go](../../internal/transport/serial_flow_test.go)) | 확인하는 것 |
|---|---|
| `TestSerialRtsCtsWaitsForCts` | 열 때 RTS 를 올린다 / CTS 가 내려가 있으면 보내지 않고 기다린다 / 올라오면 40바이트를 16·16·8 조각으로 전부 보낸다 |
| `TestSerialRtsCtsDropsAfterWaitAndWarnsOnce` | 장치가 끝내 받지 못하면 버리고 오류는 아니다 / 안내는 막힌 동안 한 번 / 버린 입력은 에코하지 않는다 / 다시 받을 수 있으면 정상 / 다시 막히면 안내를 다시 낸다 |
| `TestSerialXonXoff` | XOFF 뒤에는 보내지 않고 XON 뒤에 보낸다 / 두 문자는 화면 출력에서 빠진다 / XON 만 온 조각은 건너뛰고 다음 데이터를 돌려준다 |
| `TestSerialFlowCloseAndUnreadableCts` | 기다리는 중에 닫으면 곧바로 끝난다 / CTS 를 읽을 수 없으면 그대로 보낸다 |
| `TestSerialNoFlowWritesWhole` | 흐름 제어 없음은 전과 같이 한 번에 보낸다 |

```bash
go test ./internal/transport/     # 통과 (기존 시리얼·텔넷·로컬 테스트 포함)
```

가짜 포트(`fakeSerialPort`)로 한 시험이다. **실제 장비·케이블로는 확인하지 않았다.**

## H. 수동 검증 (남은 일)

| # | 절차 | 기대 결과 |
|---|---|---|
| 1 | RTS/CTS 를 쓰는 장비(또는 널 모뎀 케이블로 이은 두 포트)에 RTS/CTS 로 연결해 긴 글을 붙여넣는다 | 글자가 빠지지 않는다 |
| 2 | CTS 선이 없는 3선 케이블에 RTS/CTS 로 연결해 글자를 친다 | 2초 뒤 `● 장치가 받을 준비가 되지 않아 입력을 보내지 못했습니다 (흐름 제어)` 가 한 번 나온다. 흐름 제어를 "없음"으로 바꾸면 정상 |
| 3 | XON/XOFF 로 연결하고 장비 쪽에서 Ctrl+S(XOFF)에 해당하는 상태를 만든 뒤 입력 | 보내지 않다가 XON 뒤에 나간다 |
| 4 | 흐름 제어 "없음"인 기존 서버 | 전과 같다 |

## I. 영향 범위·회귀 위험

| 영역 | 위험 | 대비 |
|---|---|---|
| 흐름 제어 "없음"(기본값, 기존 서버 대부분) | 코드 경로가 전과 같다 | `TestSerialNoFlowWritesWhole`, 기존 시리얼 테스트 |
| 전에 RTS/CTS·XON/XOFF 를 골라 저장해 둔 서버 | 전에는 무시됐으므로 **이번부터 동작이 바뀐다**. CTS 선이 없으면 입력이 버려진다 | 수동 검증 2번. 안내 문구가 이유를 알려 준다 |
| 입력 경로 | 흐름 제어로 기다리는 동안(최대 2초) 그 세션의 입력 처리가 멈춘다. 접속 직후 자동 입력(`StartupInput`)은 세션 잠금을 잡은 채 쓰므로 그동안 상태 조회도 기다린다 | 2초 상한 |
| 잠금 | `mtxWrite`(쓰기) 안에서 `mtxState`(상태)를 잠깐씩 잡는다. 반대 순서는 없다 | — |

## J. 체크리스트

- [x] `serial.go`: 옵션·상태 필드, `newSerialConn`, `takeXonXoff`·`peerReady`·`waitPeerReady`·`writeFlow`·`noteFlowResult`, `Write`·`Read`·`Close` 수정
- [x] `dialers.go`: 저장값 전달 한 줄
- [x] 프런트: 도움말 키 교체 한 줄. 문구 1개 삭제·2개 추가(10개 언어)
- [x] 명세 protocols.md §5, ui/host-editor.md §1.1, decisions D108
- [x] 자동 테스트 5개 통과
- [ ] 수동 검증 1~4 (실제 장비)
