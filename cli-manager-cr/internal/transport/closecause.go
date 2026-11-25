package transport

import (
	"dariterm/internal/i18n"
	"errors"
	"io"
	"net"
)

// closedError 는 끊긴 까닭을 붙인 끊김 오류다 (protocols.md §4·§5, D127).
// errors.Is 로는 errBase(ErrRemoteClosed·ErrPortClosed)와 같고, Error() 는 지금 언어의 "까닭이 든 문구"다.
// 문구는 Error() 를 부를 때 만든다(고정 오류 i18n.Error 와 같이 시작 때 언어로 굳지 않는다).
type closedError struct {
	errBase error  // 종류: ErrRemoteClosed 또는 ErrPortClosed
	szId    string // 문구 키
	szCause string // 문구의 {cause} 에 넣을 원래 오류 글. 비면 넣지 않는다
}

// Error 는 지금 언어의 문구다.
func (pErr *closedError) Error() string {
	if pErr.szCause == "" {
		return i18n.T(pErr.szId)
	}
	return i18n.T(pErr.szId, "cause", pErr.szCause)
}

// Unwrap 은 끊김의 종류를 돌려준다 (errors.Is(err, ErrRemoteClosed) 가 그대로 참이다).
func (pErr *closedError) Unwrap() error {
	return pErr.errBase
}

// causeText 는 OS·라이브러리가 준 오류에서 사용자가 볼 까닭만 뽑는다.
// 네트워크 오류는 "read tcp 내주소->상대주소: " 머리를 떼고 OS 가 말한 까닭만 남긴다.
func causeText(errCause error) string {
	var pOpError *net.OpError
	if errors.As(errCause, &pOpError) && pOpError.Err != nil {
		return pOpError.Err.Error()
	}
	return errCause.Error()
}

// telnetClosed 는 텔넷 읽기가 errRead 로 끝났을 때의 끊김 오류를 만든다.
// 이쪽에서 닫은 것(Close)이면 까닭 없는 ErrRemoteClosed, 서버가 정상적으로 닫았으면(EOF) "서버가 연결을 닫았습니다",
// 그 밖에는 "네트워크 오류 ({까닭})" 이다.
func telnetClosed(errRead error) error {
	if errRead == nil || errors.Is(errRead, net.ErrClosed) {
		return ErrRemoteClosed
	}
	if errors.Is(errRead, io.EOF) {
		return &closedError{errBase: ErrRemoteClosed, szId: "be.remote.closedByPeer"}
	}
	return &closedError{errBase: ErrRemoteClosed, szId: "be.remote.closedCause", szCause: causeText(errRead)}
}

// serialClosed 는 시리얼 읽기·쓰기가 errIo 로 끝났을 때의 끊김 오류를 만든다.
// 이쪽에서 닫은 것(bUserClosed)이면 까닭 없는 ErrPortClosed, 그 밖에는 "포트가 닫혔습니다 ({까닭})" 이다.
func serialClosed(errIo error, bUserClosed bool) error {
	if errIo == nil || bUserClosed {
		return ErrPortClosed
	}
	return &closedError{errBase: ErrPortClosed, szId: "be.serial.closedCause", szCause: causeText(errIo)}
}
