package storeupdate

import (
	"errors"
	"testing"
)

// 확인 스크립트 출력 해석: 이 앱의 업데이트를 고르고, 네 자리 버전의 끝 ".0" 을 뗀다.
func TestParseCheckOutput(t *testing.T) {
	pUpdate, errParse := parseCheckOutput([]string{
		"UPDATE|Some.OptionalPackage|9.9.9.0|False",
		"UPDATE|YangHunLee.DariTerm|0.2.1.0|True",
		"COUNT|2",
	})
	if errParse != nil || pUpdate == nil || pUpdate.Version != "0.2.1" || !pUpdate.Mandatory {
		t.Fatalf("업데이트 해석: %+v %v", pUpdate, errParse)
	}
	pNone, errNone := parseCheckOutput([]string{"COUNT|0"})
	if errNone != nil || pNone != nil {
		t.Fatalf("업데이트 없음: %+v %v", pNone, errNone)
	}
	pOther, errOther := parseCheckOutput([]string{"UPDATE|Other.App|1.0.0.0|False", "COUNT|1"})
	if errOther != nil || pOther != nil {
		t.Fatalf("다른 패키지의 업데이트는 이 앱의 것이 아님: %+v %v", pOther, errOther)
	}
	// 대소문자가 달라도 같은 패키지로 본다. 네 번째 자리가 0 이 아니면 그대로 둔다.
	pCase, _ := parseCheckOutput([]string{"UPDATE|yanghunlee.dariterm|1.2.3.4|false", "COUNT|1"})
	if pCase == nil || pCase.Version != "1.2.3.4" || pCase.Mandatory {
		t.Fatalf("버전 표기: %+v", pCase)
	}
}

// 스크립트가 끝까지 돌지 못했으면(COUNT 줄 없음) "업데이트 없음"으로 보지 않고 오류다.
func TestParseCheckOutputIncompleteIsError(t *testing.T) {
	for _, slLines := range [][]string{nil, {"some error text"}, {"UPDATE|YangHunLee.DariTerm|0.2.0.0|False"}} {
		if _, errParse := parseCheckOutput(slLines); errParse == nil {
			t.Fatalf("오류여야 함: %v", slLines)
		}
	}
}

func TestParseInstallOutput(t *testing.T) {
	if szState, errParse := parseInstallOutput([]string{"noise", " STATE|Completed "}); errParse != nil || szState != G_cszStateCompleted {
		t.Fatalf("완료 상태: %q %v", szState, errParse)
	}
	if szState, _ := parseInstallOutput([]string{"STATE|Canceled"}); szState != G_cszStateCanceled {
		t.Fatalf("취소 상태: %q", szState)
	}
	if _, errParse := parseInstallOutput([]string{"STATE|", "x"}); errParse == nil {
		t.Fatal("상태 줄이 없으면 오류여야 함")
	}
}

// 시험은 스토어 설치본이 아닌 곳에서 돈다: 스토어 확인을 하지 않고 ErrNotStore 다.
func TestNotStoreInstallDoesNothing(t *testing.T) {
	if IsStoreInstall() {
		t.Skip("패키지된 환경에서 실행 중")
	}
	if _, errCheck := Check(); !errors.Is(errCheck, ErrNotStore) {
		t.Fatalf("Check: %v", errCheck)
	}
	if _, errInstall := Install(); !errors.Is(errInstall, ErrNotStore) {
		t.Fatalf("Install: %v", errInstall)
	}
}
