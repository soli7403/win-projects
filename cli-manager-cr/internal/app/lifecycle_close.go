package app

import (
	"dariterm/internal/i18n"

	"dariterm/internal/session"
)

// CloseConfirmText 는 연결된 세션이 있을 때 종료 확인 본문을 만든다 (lifecycle.md §5 문구 원문).
// iKeep 은 세션 유지(tmux) 중인 창 수다. 0이면 둘째 문단이 없다.
func CloseConfirmText(iSessions int, iKeep int) string {
	szText := i18n.T("be.close.body", "count", iSessions)
	if iKeep > 0 {
		szText += "\n\n" + i18n.T("be.close.keep", "count", iKeep)
	}
	return szText
}

// CountKeepSessions 는 세션 정보 목록에서 세션 유지 중인 창 수를 센다.
func CountKeepSessions(slInfos []session.Info) int {
	iCount := 0
	for _, info := range ConnectedSessions(slInfos) {
		if info.M_bKeepSession {
			iCount++
		}
	}
	return iCount
}

// SessionInfos 는 열린 세션 정보 목록을 돌려준다 (종료 확인에 쓴다).
func (pService *Service) SessionInfos() []session.Info {
	return pService.pSessions.Snapshots()
}

// NeedsCloseConfirm 은 창 닫기에서 종료 확인을 띄울지 정한다. 연결된 세션이 없으면 확인 없이 닫는다 (acceptance X4b).
func NeedsCloseConfirm(iSessions int) bool {
	return iSessions > 0
}

// IsQuitChoice 는 종료 확인 대화상자의 반환값이 "종료 선택"인지 판정한다 (lifecycle.md §5 구현 주의(Windows)).
// Windows 는 지정한 이름 대신 시스템 버튼을 쓰고 "Yes" 를 돌려주므로 "종료", "Yes", "예", "예(Y)" 를 종료로 본다.
// 그 밖의 값(취소, No, 아니요(N), 빈 값)이나 오류는 취소다.
func IsQuitChoice(szChoice string, bErr bool) bool {
	if bErr {
		return false
	}
	if szChoice == i18n.T("be.close.quit") {
		return true
	}
	switch szChoice {
	case "종료", "Yes", "예", "예(Y)": // i18n-ok Windows 시스템 버튼·예전 값
		return true
	}
	return false
}

// ConnectedSessions 는 상태가 connected 인 세션만 거른다. 종료 확인의 집계 대상이다 (lifecycle.md §5).
// 접속 중·접속 실패·끊김 상태는 확인 문구의 세션 수에 넣지 않는다.
func ConnectedSessions(slInfos []session.Info) []session.Info {
	slOut := []session.Info{}
	for _, info := range slInfos {
		if info.M_szState == session.G_cszStateConnected {
			slOut = append(slOut, info)
		}
	}
	return slOut
}
