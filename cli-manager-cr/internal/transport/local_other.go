//go:build !windows

package transport

import (
	"dariterm/internal/i18n"
)

// ErrLocalUnsupported 는 이 운영체제에서 로컬 터미널을 아직 지원하지 않는다는 뜻이다 (1차 범위는 Windows).
var ErrLocalUnsupported = i18n.NewError("be.local.unsupported")

// LocalConn 은 비 Windows 빌드의 자리표시자다.
type LocalConn struct{}

// OpenLocal 은 비 Windows 에서 항상 실패한다.
func OpenLocal(opts LocalOptions) (*LocalConn, error) {
	return nil, ErrLocalUnsupported
}

// Read 는 연결이 없으므로 오류를 돌려준다.
func (pLocal *LocalConn) Read(bBuf []byte) (int, error) { return 0, ErrLocalUnsupported }

// Write 는 연결이 없으므로 오류를 돌려준다.
func (pLocal *LocalConn) Write(bData []byte) error { return ErrLocalUnsupported }

// Resize 는 아무 일도 하지 않는다.
func (pLocal *LocalConn) Resize(iCols int, iRows int) error { return nil }

// Close 는 아무 일도 하지 않는다.
func (pLocal *LocalConn) Close() error { return nil }
