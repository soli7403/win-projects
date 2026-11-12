package ipc

import (
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"
)

type fakeEmitter struct {
	mtxData sync.Mutex
	slEvent []string
	slData  []any
}

func (pFake *fakeEmitter) Emit(szEvent string, anyData ...any) {
	pFake.mtxData.Lock()
	defer pFake.mtxData.Unlock()
	pFake.slEvent = append(pFake.slEvent, szEvent)
	if len(anyData) > 0 {
		pFake.slData = append(pFake.slData, anyData[0])
	} else {
		pFake.slData = append(pFake.slData, nil)
	}
}

// 알 수 없는 채널·인자 해석 실패 오류 문구, 결과 없으면 빈 문자열. spec/ipc.md §1.
func TestInvokeErrorsAndEmptyResult(t *testing.T) {
	pRouter := NewRouter(nil)
	pRouter.Handle("ping", func(slArgs []json.RawMessage) (any, error) { return nil, nil })

	if _, errInvoke := pRouter.Invoke("nope", "[]"); errInvoke == nil || errInvoke.Error() != "구현되지 않은 채널: nope" {
		t.Fatalf("알 수 없는 채널 오류 문구 불일치: %v", errInvoke)
	}
	if _, errInvoke := pRouter.Invoke("ping", "{not json"); errInvoke == nil || !strings.HasPrefix(errInvoke.Error(), "인자 해석 실패(ping)") {
		t.Fatalf("인자 해석 실패 문구 불일치: %v", errInvoke)
	}
	szResult, errInvoke := pRouter.Invoke("ping", "[]")
	if errInvoke != nil || szResult != "" {
		t.Fatalf("결과 없음은 빈 문자열이어야 함: %q %v", szResult, errInvoke)
	}
}

// 핸들러 패닉은 앱을 멈추지 않고 "내부 오류(채널): 내용" 으로 바뀐다.
func TestInvokeRecoversPanic(t *testing.T) {
	pRouter := NewRouter(nil)
	pRouter.Handle("boom", func(slArgs []json.RawMessage) (any, error) { panic("폭발") })
	_, errInvoke := pRouter.Invoke("boom", "[]")
	if errInvoke == nil || errInvoke.Error() != "내부 오류(boom): 폭발" {
		t.Fatalf("패닉 변환 문구 불일치: %v", errInvoke)
	}
}

// 질문 → 답 전달, null 과 시간 초과는 취소. spec/ipc.md §3.
func TestAskReplyAndCancel(t *testing.T) {
	fake := &fakeEmitter{}
	pRouter := NewRouter(fake)
	pRouter.Asker().SetTimeout(200 * time.Millisecond)

	chResult := make(chan bool, 1)
	go func() {
		rawValue, bOk := pRouter.Asker().Ask("passphrase", map[string]any{"sessionTitle": "t"})
		chResult <- bOk && string(rawValue) == `"secret"`
	}()
	// 이벤트에 실린 requestId 를 찾아 답한다.
	var szId string
	for i := 0; i < 100 && szId == ""; i++ {
		time.Sleep(5 * time.Millisecond)
		fake.mtxData.Lock()
		if len(fake.slData) > 0 {
			szId = fake.slData[0].(map[string]any)["requestId"].(string)
		}
		fake.mtxData.Unlock()
	}
	if szId == "" {
		t.Fatal("ui.request 이벤트가 나가지 않음")
	}
	pRouter.Reply(szId, `"secret"`)
	if !<-chResult {
		t.Fatal("답이 전달되지 않음")
	}

	_, bCancelled := pRouter.Asker().Ask("username", map[string]any{"sessionTitle": "t"})
	if bCancelled {
		t.Fatal("시간 초과는 취소여야 함")
	}
}
