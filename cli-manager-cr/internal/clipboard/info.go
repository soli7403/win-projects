package clipboard

// Info 는 clip.info 결과다 (spec/ipc.md §2). 우선순위: 존재하는 파일 목록 → 텍스트 → 이미지 → 비어 있음.
type Info struct {
	M_szType  string   `json:"type"`
	M_slFiles []string `json:"files,omitempty"`
	M_szText  string   `json:"text,omitempty"`
}

// GetInfo 는 클립보드 종류를 판정한다.
func GetInfo() Info {
	if slFiles, errFiles := Files(); errFiles == nil && len(slFiles) > 0 {
		return Info{M_szType: "files", M_slFiles: slFiles}
	}
	if szText, errText := ReadText(); errText == nil && szText != "" {
		return Info{M_szType: "text", M_szText: szText}
	}
	if HasImage() {
		return Info{M_szType: "image"}
	}
	return Info{M_szType: "empty"}
}
