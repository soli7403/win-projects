// updatecheck 는 릴리스 저장소 방식의 새 버전 확인(internal/update)을 앱 밖에서 한 번 돌려 결과를 적는 검증 도구다 (D135).
// 앱과 같은 코드로 공개 API 에 묻는다. 사용:
//
//	go run ./tools/updatecheck <owner/name> <지금 버전>     예: go run ./tools/updatecheck soli7403/dariterm-releases 1.0.0
package main

import (
	"fmt"
	"os"

	"dariterm/internal/update"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: updatecheck <owner/name> <current version>")
		os.Exit(2)
	}
	res := update.Check("https://api.github.com", os.Args[1], os.Args[2], nil)
	fmt.Printf("repo=%s current=%s -> status=%s version=%s url=%s\n", os.Args[1], os.Args[2], res.M_szStatus, res.M_szVersion, res.M_szUrl)
}
