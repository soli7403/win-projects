// Package zmodem 은 ZMODEM 파일 전송 프로토콜을 구현한다 (spec/behavior/zmodem.md, D114).
// 서버에서 sz 를 실행하면 받는 쪽(Receive), rz 를 실행하면 보내는 쪽(Send)이 된다.
// 형식은 공개 규격(Chuck Forsberg, "The ZMODEM Inter Application File Transfer Protocol")을 따른다.
// 전송 계층(SSH·시리얼)과 파일 저장 방식은 인터페이스로 받아 이 패키지는 프로토콜만 다룬다.
package zmodem

import (
	"encoding/binary"
	"errors"
	"hash/crc32"
)

// 프레임을 이루는 특수 바이트.
const (
	g_cbZpad   = '*'  // 헤더 앞의 채움 문자
	g_cbZdle   = 0x18 // 이스케이프 문자 (CAN 과 같은 값)
	g_cbZbin   = 'A'  // 이진 헤더, CRC-16
	g_cbZhex   = 'B'  // 16진 헤더, CRC-16
	g_cbZbin32 = 'C'  // 이진 헤더, CRC-32
	g_cbXon    = 0x11
	g_cbXoff   = 0x13
	g_cbZrub0  = 'l' // ZDLE 뒤에서 0x7f 를 뜻한다
	g_cbZrub1  = 'm' // ZDLE 뒤에서 0xff 를 뜻한다
)

// 헤더 종류.
const (
	G_cbZrqinit  = 0  // 보내는 쪽의 시작 요청 (sz 가 처음 보낸다)
	G_cbZrinit   = 1  // 받는 쪽의 준비 알림 (rz 가 처음 보낸다)
	g_cbZsinit   = 2  // 보내는 쪽의 초기 설정
	g_cbZack     = 3  // 확인
	g_cbZfile    = 4  // 파일 정보
	g_cbZskip    = 5  // 이 파일 건너뛰기
	g_cbZnak     = 6  // 마지막 헤더를 못 읽음
	g_cbZabort   = 7  // 전송 중단
	g_cbZfin     = 8  // 세션 끝
	g_cbZrpos    = 9  // 이 위치부터 다시 보내라
	g_cbZdata    = 10 // 데이터가 뒤따른다
	g_cbZeof     = 11 // 파일 끝
	g_cbZferr    = 12 // 파일 읽기·쓰기 오류
	g_cbZcrc     = 13 // 파일 CRC 요청·응답
	g_cbZcan     = 16 // CAN 여러 개로 중단됨
	g_cbZfreecnt = 17 // 남은 저장 공간 요청
)

// 데이터 조각(subpacket)의 끝 표시. ZDLE 뒤에 온다.
const (
	g_cbZcrce = 'h' // 프레임 끝, 헤더가 뒤따른다
	g_cbZcrcg = 'i' // 프레임이 이어진다, 응답 없음
	g_cbZcrcq = 'j' // 프레임이 이어진다, ZACK 을 기대한다
	g_cbZcrcw = 'k' // 프레임 끝, ZACK 을 기대한다
)

// ZRINIT 의 능력 표시(ZF0).
const (
	g_cbCanFdx  = 0x01 // 동시에 보내고 받을 수 있다
	g_cbCanOvIo = 0x02 // 디스크에 쓰는 동안에도 받을 수 있다
	g_cbCanFc32 = 0x20 // CRC-32 를 쓸 수 있다
)

// 한도.
const (
	// G_ciSubpacketBytes 는 보낼 때 데이터 조각 하나의 크기다. 모든 구현이 받는 1024 로 둔다.
	G_ciSubpacketBytes = 1024
	// g_ciSubpacketMax 는 받을 때 받아들이는 조각의 최대 크기다 (lrzsz 는 8192 까지 보낸다).
	g_ciSubpacketMax = 16 * 1024
	// g_ciAckEvery 는 보낼 때 몇 조각마다 확인(ZACK)을 기다릴지다. 약 256KB 마다 한 번.
	g_ciAckEvery = 256
	// g_ciGarbageMax 는 헤더를 기다리는 동안 버릴 수 있는 바이트 수다. 넘으면 상대가 ZMODEM 을 끝낸 것으로 본다.
	// 오류 뒤 다시 맞추는 동안에는 상대가 이미 보낸 데이터(확인 주기 하나 분량)를 통째로 버려야 하므로 넉넉히 둔다.
	g_ciGarbageMax = 4 * 1024 * 1024
	// g_ciErrorMax 는 전송 하나에서 참는 오류(CRC 불일치 등) 횟수다.
	g_ciErrorMax = 10
)

// 오류.
var (
	// ErrCancelled 는 이쪽(사용자)이 전송을 취소했을 때의 오류다.
	ErrCancelled = errors.New("zmodem: cancelled")
	// ErrRemoteCancelled 는 상대가 전송을 중단했을 때의 오류다.
	ErrRemoteCancelled = errors.New("zmodem: cancelled by the remote side")
	// ErrProtocol 은 상대의 응답이 규격과 맞지 않거나 오류가 너무 많을 때의 오류다.
	ErrProtocol = errors.New("zmodem: protocol error")
	// ErrNotZmodem 은 시작 표시처럼 보였지만 ZMODEM 헤더가 아니었을 때의 오류다 (화면 출력으로 되돌린다).
	ErrNotZmodem = errors.New("zmodem: not a zmodem header")
)

// StreamError 는 전송 계층(연결)에서 난 읽기·쓰기 오류를 감싼다. 호출자는 연결 끊김으로 처리한다.
type StreamError struct {
	Err error
}

func (pErr *StreamError) Error() string { return pErr.Err.Error() }
func (pErr *StreamError) Unwrap() error { return pErr.Err }

// header 는 헤더 하나다: 종류와 4바이트 값(위치 또는 플래그), 그리고 받은 형식(뒤따르는 데이터의 CRC 종류를 정한다).
type header struct {
	bType   byte
	arrData [4]byte
	bFormat byte
}

// pos 는 헤더 값을 파일 위치(리틀 엔디언)로 읽는다.
func (hdr header) pos() int64 {
	return int64(binary.LittleEndian.Uint32(hdr.arrData[:]))
}

// posData 는 파일 위치를 헤더 값으로 만든다.
func posData(i64Pos int64) [4]byte {
	var arrData [4]byte
	binary.LittleEndian.PutUint32(arrData[:], uint32(i64Pos))
	return arrData
}

// crc16 은 CRC-16/XMODEM(다항식 0x1021, 초기값 0)을 계산한다.
func crc16(bData []byte) uint16 {
	var uiCrc uint16
	for _, bOne := range bData {
		uiCrc ^= uint16(bOne) << 8
		for iBit := 0; iBit < 8; iBit++ {
			if uiCrc&0x8000 != 0 {
				uiCrc = uiCrc<<1 ^ 0x1021
			} else {
				uiCrc <<= 1
			}
		}
	}
	return uiCrc
}

// appendEscaped 는 bData 를 ZDLE 이스케이프해 bOut 에 붙인다.
// 제어 문자(0x00~0x1f, 0x80~0x9f)는 모두 이스케이프한다 — 터미널 계층이 건드릴 수 있는 문자를 하나도 그대로 보내지 않는다.
func appendEscaped(bOut []byte, bData []byte) []byte {
	for _, bOne := range bData {
		switch {
		case bOne == 0x7f:
			bOut = append(bOut, g_cbZdle, g_cbZrub0)
		case bOne == 0xff:
			bOut = append(bOut, g_cbZdle, g_cbZrub1)
		case bOne&0x60 == 0:
			bOut = append(bOut, g_cbZdle, bOne^0x40)
		default:
			bOut = append(bOut, bOne)
		}
	}
	return bOut
}

const g_cszHexDigits = "0123456789abcdef"

// encodeHexHeader 는 16진 헤더를 만든다: ZPAD ZPAD ZDLE 'B' + 16진 14자 + CR LF(8번째 비트를 세운 LF) [+ XON].
// ZACK 과 ZFIN 뒤에는 XON 을 붙이지 않는다.
func encodeHexHeader(bType byte, arrData [4]byte) []byte {
	bBody := []byte{bType, arrData[0], arrData[1], arrData[2], arrData[3]}
	uiCrc := crc16(bBody)
	bBody = append(bBody, byte(uiCrc>>8), byte(uiCrc))
	bOut := []byte{g_cbZpad, g_cbZpad, g_cbZdle, g_cbZhex}
	for _, bOne := range bBody {
		bOut = append(bOut, g_cszHexDigits[bOne>>4], g_cszHexDigits[bOne&0x0f])
	}
	bOut = append(bOut, '\r', 0x8a)
	if bType != g_cbZack && bType != g_cbZfin {
		bOut = append(bOut, g_cbXon)
	}
	return bOut
}

// encodeBinHeader 는 이진 헤더를 만든다. bCrc32 면 CRC-32('C'), 아니면 CRC-16('A').
func encodeBinHeader(bType byte, arrData [4]byte, bCrc32 bool) []byte {
	bBody := []byte{bType, arrData[0], arrData[1], arrData[2], arrData[3]}
	if bCrc32 {
		bBody = binary.LittleEndian.AppendUint32(bBody, crc32.ChecksumIEEE(bBody))
		return appendEscaped([]byte{g_cbZpad, g_cbZdle, g_cbZbin32}, bBody)
	}
	uiCrc := crc16(bBody)
	bBody = append(bBody, byte(uiCrc>>8), byte(uiCrc))
	return appendEscaped([]byte{g_cbZpad, g_cbZdle, g_cbZbin}, bBody)
}

// appendSubpacket 은 데이터 조각 하나를 bOut 에 붙인다: 이스케이프한 데이터 + ZDLE 끝 표시 + CRC(데이터와 끝 표시에 대한).
func appendSubpacket(bOut []byte, bData []byte, bEnd byte, bCrc32 bool) []byte {
	bOut = appendEscaped(bOut, bData)
	bOut = append(bOut, g_cbZdle, bEnd)
	if bCrc32 {
		uiCrc := crc32.Update(crc32.ChecksumIEEE(bData), crc32.IEEETable, []byte{bEnd})
		return appendEscaped(bOut, binary.LittleEndian.AppendUint32(nil, uiCrc))
	}
	uiCrc := crc16(append(append([]byte(nil), bData...), bEnd))
	return appendEscaped(bOut, []byte{byte(uiCrc >> 8), byte(uiCrc)})
}

// AbortSequence 는 상대에게 전송 중단을 알리는 바이트열이다: CAN 8개 + 백스페이스 8개.
func AbortSequence() []byte {
	return []byte{0x18, 0x18, 0x18, 0x18, 0x18, 0x18, 0x18, 0x18, 8, 8, 8, 8, 8, 8, 8, 8}
}

// TrailingOutput 은 받다가 중단한 전송의 끝자락(bTail)에서, 마지막 데이터 조각 뒤에 붙어 온 바이트를 돌려준다.
// 중단을 알린 뒤에도 상대가 이미 보낸 데이터가 한동안 흘러오고, 그 끝에 셸 프롬프트 같은 화면 출력이 바로 붙는다.
// 데이터 조각은 ZDLE + 끝 표시(ZCRCE·ZCRCG·ZCRCQ·ZCRCW) + CRC 로 끝나고, 조각 안의 ZDLE 은 항상 바꿔서 오므로
// 이 두 바이트는 조각 끝에만 나온다. 끝 표시가 없으면 bFound 가 false 다(끝자락에 데이터 조각이 없었다).
// CRC 가 끝자락에서 잘렸으면 빈 값을 돌려준다.
func TrailingOutput(bTail []byte, bCrc32 bool) (bAfter []byte, bFound bool) {
	iEnd := -1
	for iIndex := len(bTail) - 2; iIndex >= 0; iIndex-- {
		if bTail[iIndex] != g_cbZdle {
			continue
		}
		if bNext := bTail[iIndex+1]; bNext >= g_cbZcrce && bNext <= g_cbZcrcw {
			iEnd = iIndex + 2
			break
		}
	}
	if iEnd < 0 {
		return nil, false
	}
	iCrc := 2
	if bCrc32 {
		iCrc = 4
	}
	for ; iCrc > 0; iCrc-- {
		if iEnd < len(bTail) && bTail[iEnd] == g_cbZdle {
			iEnd++ // 바꿔서 온 CRC 바이트는 두 바이트다
		}
		iEnd++
		if iEnd > len(bTail) {
			return nil, true
		}
	}
	return bTail[iEnd:], true
}
