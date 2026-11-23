// Package storeupdate 는 Microsoft Store 로 설치한 앱의 새 버전을 스토어에 묻고, 업데이트 확인 창을 띄운다
// (spec/behavior/lifecycle.md §6.1, D115). Windows 의 스토어 설치본에서만 동작하고, 그 밖에서는 아무 일도 하지 않는다.
package storeupdate

import (
	"errors"
	"strings"
)

// 스토어의 제품 정보 (docs/store/README.md §0 — Partner Center 가 정한 값).
const (
	// G_cszProductId 는 스토어 제품 ID 다.
	G_cszProductId = "9N8WBPMRNBWV"
	// G_cszPackageName 은 패키지 이름(Identity Name)이다. 스토어가 돌려준 업데이트 목록에서 이 앱의 것을 고를 때 쓴다.
	G_cszPackageName = "YangHunLee.DariTerm"
	// G_cszStorePageUri 는 Microsoft Store 앱에서 이 제품 페이지를 여는 주소다.
	G_cszStorePageUri = "ms-windows-store://pdp/?productid=" + G_cszProductId
)

// 설치 요청의 결과 상태 (Windows 의 StorePackageUpdateState 이름 그대로).
const (
	G_cszStateCompleted = "Completed"
	G_cszStateCanceled  = "Canceled"
)

// ErrNotStore 는 스토어 설치본이 아닌 곳에서 불렀을 때의 오류다.
var ErrNotStore = errors.New("storeupdate: not a Microsoft Store install")

// Update 는 스토어에 올라온 새 버전 하나다.
type Update struct {
	Version   string // "1.2.3" (끝의 ".0" 네 번째 자리는 뗀다)
	Mandatory bool   // 스토어에서 필수 업데이트로 표시한 버전
}

// parseCheckOutput 은 확인 스크립트의 출력 줄을 해석한다.
// 줄 형식: "UPDATE|패키지 이름|주.부.빌드.수정|True 또는 False", 마지막에 "COUNT|개수".
// "COUNT" 줄이 없으면 스크립트가 끝까지 돌지 못한 것이므로 오류다. 이 앱의 업데이트가 없으면 (nil, nil) 이다.
func parseCheckOutput(slLines []string) (*Update, error) {
	var pFound *Update
	bDone := false
	for _, szLine := range slLines {
		slParts := strings.Split(strings.TrimSpace(szLine), "|")
		switch {
		case len(slParts) == 2 && slParts[0] == "COUNT":
			bDone = true
		case len(slParts) == 4 && slParts[0] == "UPDATE":
			if !strings.EqualFold(slParts[1], G_cszPackageName) || pFound != nil {
				continue // 선택 패키지 등 다른 것의 업데이트
			}
			pFound = &Update{Version: displayVersion(slParts[2]), Mandatory: strings.EqualFold(slParts[3], "True")}
		}
	}
	if !bDone {
		return nil, errors.New("storeupdate: store did not answer")
	}
	return pFound, nil
}

// parseInstallOutput 은 설치 스크립트의 출력에서 결과 상태("STATE|이름")를 찾는다.
func parseInstallOutput(slLines []string) (string, error) {
	for _, szLine := range slLines {
		if szState, bFound := strings.CutPrefix(strings.TrimSpace(szLine), "STATE|"); bFound && szState != "" {
			return szState, nil
		}
	}
	return "", errors.New("storeupdate: store did not answer")
}

// displayVersion 은 스토어의 네 자리 버전에서 끝의 ".0" 을 떼어 앱이 쓰는 세 자리 표기로 바꾼다.
func displayVersion(szVersion string) string {
	slParts := strings.Split(szVersion, ".")
	if len(slParts) == 4 && slParts[3] == "0" {
		return strings.Join(slParts[:3], ".")
	}
	return szVersion
}
