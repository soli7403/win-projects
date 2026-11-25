//go:build !windows

package importer

// XshellSessionsDir 는 이 PC 의 Xshell 세션 폴더를 돌려준다. Xshell 은 Windows 프로그램이라 그 밖의 OS 에서는 빈 문자열이다
// (내보내기 파일이나 복사한 폴더는 ScanXshell 에 경로를 주어 읽을 수 있다, import.md §1.6).
func XshellSessionsDir() string {
	return ""
}
