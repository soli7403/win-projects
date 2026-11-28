// storecheck 는 스토어 업데이트 확인(internal/storeupdate)을 앱 밖에서 한 번 돌려 결과와 오류 내용을 파일에 적는 검증 도구다.
// 패키지 신원이 있어야 스토어가 답하므로 설치된 스토어판의 신원으로 실행한다 (D134):
//
//	go build -o <폴더>\storecheck.exe ./tools/storecheck
//	Invoke-CommandInDesktopPackage -PackageFamilyName 'YangHunLee.DariTerm_bmtfree9a9kzr' -AppId 'DariTerm' `
//	  -Command '<폴더>\storecheck.exe' -Args '"<폴더>\storecheck.txt"'
//
// 앱과 같은 경로(이 프로그램 → 숨긴 PowerShell → 스토어)를 타므로, 앱의 [업데이트 확인] 이 실패할 때 까닭을 볼 수 있다.
package main

import (
	"fmt"
	"os"
	"time"

	"dariterm/internal/storeupdate"
)

func main() {
	szOut := "storecheck.txt"
	if len(os.Args) > 1 {
		szOut = os.Args[1]
	}
	timeStart := time.Now()
	szText := fmt.Sprintf("store=%v\n", storeupdate.IsStoreInstall())
	pUpdate, errCheck := storeupdate.Check()
	szText += fmt.Sprintf("elapsed=%s\nupdate=%+v\nerr=%v\n", time.Since(timeStart).Round(time.Millisecond), pUpdate, errCheck)
	if errWrite := os.WriteFile(szOut, []byte(szText), 0o600); errWrite != nil {
		fmt.Fprintln(os.Stderr, errWrite)
		os.Exit(1)
	}
}
