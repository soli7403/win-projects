package zmodem

// Kind 는 출력에서 찾은 ZMODEM 시작 표시의 종류다.
type Kind int

const (
	KindNone          Kind = iota // 시작 표시 없음
	KindRemoteSends               // 서버가 보내려 한다(ZRQINIT, 서버에서 sz 실행) → 이쪽은 받는다
	KindRemoteAccepts             // 서버가 받으려 한다(ZRINIT, 서버에서 rz 실행) → 이쪽은 보낸다
)

// g_cszStartPrefix 는 16진 헤더의 앞부분이다: ZPAD ZPAD ZDLE 'B' '0'. 그 뒤 글자가 '0' 이면 ZRQINIT, '1' 이면 ZRINIT 다.
const g_cszStartPrefix = "**\x18B0"

// Detector 는 세션 출력 흐름에서 ZMODEM 시작 표시를 찾는다.
// 표시가 출력 묶음 경계에 걸쳐 와도 찾는다(앞 묶음에서 맞춘 글자 수를 기억한다).
// 표시 앞의 출력은 그대로 화면으로 보내야 하므로, 앞 묶음에 걸친 표시의 앞 글자('*' 등)는 이미 화면으로 나간 뒤다.
type Detector struct {
	iMatched int // 지금까지 맞춘 g_cszStartPrefix 의 글자 수
}

// Feed 는 출력 묶음 하나를 살핀다.
// 시작 표시가 없으면 (bData, KindNone, nil) 을 돌려준다.
// 있으면 표시 앞부분, 종류, 그리고 엔진에 넘길 바이트(다시 만든 표시 6바이트 + 묶음의 나머지)를 돌려준다.
func (pDetector *Detector) Feed(bData []byte) (bBefore []byte, kind Kind, bEngine []byte) {
	for iIndex, bOne := range bData {
		if pDetector.iMatched == len(g_cszStartPrefix) {
			pDetector.iMatched = 0
			if bOne == '0' || bOne == '1' {
				kind = KindRemoteSends
				if bOne == '1' {
					kind = KindRemoteAccepts
				}
				iStart := iIndex - len(g_cszStartPrefix) // 표시가 이 묶음에서 시작한 자리 (앞 묶음에 걸쳤으면 음수)
				if iStart < 0 {
					iStart = 0
				}
				bEngine = append([]byte(g_cszStartPrefix), bData[iIndex:]...)
				return bData[:iStart], kind, bEngine
			}
		}
		switch {
		case bOne == g_cszStartPrefix[pDetector.iMatched]:
			pDetector.iMatched++
		case bOne == '*':
			// "***" 처럼 별이 이어지면 마지막 두 개가 표시의 앞부분일 수 있다.
			if pDetector.iMatched >= 2 {
				pDetector.iMatched = 2
			} else {
				pDetector.iMatched = 1
			}
		default:
			pDetector.iMatched = 0
		}
	}
	return bData, KindNone, nil
}

// Reset 은 맞추던 상태를 지운다 (연결이 바뀌었을 때).
func (pDetector *Detector) Reset() {
	pDetector.iMatched = 0
}
