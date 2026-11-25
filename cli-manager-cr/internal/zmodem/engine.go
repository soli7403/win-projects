package zmodem

import (
	"encoding/binary"
	"hash/crc32"
)

// Stream 은 전송 계층이다 (세션의 연결). Read 는 받은 바이트를, Write 는 보낼 바이트를 다룬다.
type Stream interface {
	Read(bBuf []byte) (int, error)
	Write(bData []byte) error
}

// Options 는 전송 하나의 선택 사항이다.
type Options struct {
	// Cancelled 는 사용자가 취소했는지 알려 준다. nil 이면 취소할 수 없다.
	Cancelled func() bool
	// Progress 는 진행 상황을 알린다. nil 이면 알리지 않는다.
	Progress func(ev Progress)
}

// Progress 는 진행 상황 알림 하나다.
type Progress struct {
	Name  string
	Done  int64
	Total int64 // 모르면 -1
	State int   // G_ciStateStart · G_ciStateData · G_ciStateDone · G_ciStateSkipped
}

// 진행 상태.
const (
	G_ciStateStart = iota
	G_ciStateData
	G_ciStateDone
	G_ciStateSkipped
)

// engine 은 전송 하나의 상태다: 읽기 버퍼, 취소 확인, 오류 횟수.
type engine struct {
	stream  Stream
	opts    Options
	bIn     []byte // 읽어 둔 바이트
	iIn     int    // bIn 에서 다음에 읽을 자리
	bKeep   bool   // true 면 읽은 바이트를 버리지 않고 쌓는다 (첫 헤더가 아니면 통째로 되돌리기 위해)
	iCanRun int    // 연달아 받은 CAN(0x18) 수
	iErrors int
	// bResync 는 오류 뒤 ZRPOS 를 보내고 상대의 새 ZDATA 를 기다리는 중이라는 뜻이다. 그동안 흘러오는 것은
	// 이미 버리기로 한 데이터라, 그 안에서 헤더처럼 보이다 만 것들은 오류로 세지 않는다(ZRPOS 를 보낼 때 한 번 셌다).
	bResync bool
	// bDataCrc32 는 상대가 마지막으로 보낸 데이터 프레임이 CRC-32 였는지다 (중단 뒤 끝자락을 가를 때 쓴다).
	bDataCrc32 bool
	bDropped   []byte // 취소한 뒤 읽어서 버린 마지막 묶음
	arrRead    [32 * 1024]byte
}

func newEngine(stream Stream, bInitial []byte, opts Options) *engine {
	return &engine{stream: stream, opts: opts, bIn: append([]byte(nil), bInitial...), bKeep: true}
}

func (pEngine *engine) cancelled() bool {
	return pEngine.opts.Cancelled != nil && pEngine.opts.Cancelled()
}

func (pEngine *engine) progress(szName string, i64Done int64, i64Total int64, iState int) {
	if pEngine.opts.Progress != nil {
		pEngine.opts.Progress(Progress{Name: szName, Done: i64Done, Total: i64Total, State: iState})
	}
}

// readByte 는 바이트 하나를 읽는다. 버퍼가 비면 연결에서 읽는다.
// 사용자가 취소했으면 ErrCancelled, CAN 이 5개 이어지면 ErrRemoteCancelled 다.
func (pEngine *engine) readByte() (byte, error) {
	for pEngine.iIn >= len(pEngine.bIn) {
		if pEngine.cancelled() {
			return 0, ErrCancelled
		}
		nRead, errRead := pEngine.stream.Read(pEngine.arrRead[:])
		if nRead > 0 {
			if pEngine.bKeep {
				pEngine.bIn = append(pEngine.bIn, pEngine.arrRead[:nRead]...)
			} else {
				pEngine.bIn = append(pEngine.bIn[:0], pEngine.arrRead[:nRead]...)
				pEngine.iIn = 0
			}
		}
		if errRead != nil && nRead == 0 {
			return 0, &StreamError{Err: errRead}
		}
		if pEngine.cancelled() {
			// 취소한 뒤에 도착한 것은 전송 중이던 데이터다: 화면으로 되돌리지 않고 버린다.
			// 다만 그 끝에 셸 출력이 붙어 왔을 수 있어, 부른 쪽이 가를 수 있게 끝자락은 남겨 둔다 (Result.Dropped).
			pEngine.bDropped = append([]byte(nil), pEngine.bIn[pEngine.iIn:]...)
			pEngine.bIn, pEngine.iIn = pEngine.bIn[:0], 0
			return 0, ErrCancelled
		}
	}
	bOne := pEngine.bIn[pEngine.iIn]
	pEngine.iIn++
	if bOne == g_cbZdle {
		pEngine.iCanRun++
		if pEngine.iCanRun >= 5 {
			return 0, ErrRemoteCancelled
		}
	} else {
		pEngine.iCanRun = 0
	}
	return bOne, nil
}

// unreadByte 는 방금 읽은 바이트 하나를 되돌린다.
func (pEngine *engine) unreadByte() {
	if pEngine.iIn > 0 {
		pEngine.iIn--
	}
}

// leftover 는 읽어 두었지만 쓰지 않은 바이트다. 전송이 끝난 뒤 화면 출력으로 되돌린다.
func (pEngine *engine) leftover() []byte {
	return append([]byte(nil), pEngine.bIn[pEngine.iIn:]...)
}

func (pEngine *engine) write(bData []byte) error {
	if errWrite := pEngine.stream.Write(bData); errWrite != nil {
		return &StreamError{Err: errWrite}
	}
	return nil
}

func (pEngine *engine) sendHex(bType byte, arrData [4]byte) error {
	return pEngine.write(encodeHexHeader(bType, arrData))
}

// noteError 는 오류 하나를 센다. 한도를 넘으면 ErrProtocol 이다.
func (pEngine *engine) noteError() error {
	pEngine.iErrors++
	if pEngine.iErrors > g_ciErrorMax {
		return ErrProtocol
	}
	return nil
}

// readEscaped 는 이스케이프를 푼 바이트 하나를 읽는다. 데이터 조각의 끝 표시를 만나면 bEnd 에 그 표시를 넣어 돌려준다.
// 흐름 제어 문자(XON·XOFF)는 건너뛴다.
func (pEngine *engine) readEscaped() (bOne byte, bEnd byte, err error) {
	for {
		bRaw, errRead := pEngine.readByte()
		if errRead != nil {
			return 0, 0, errRead
		}
		switch bRaw {
		case g_cbXon, g_cbXoff, g_cbXon | 0x80, g_cbXoff | 0x80:
			continue
		case g_cbZdle:
		default:
			return bRaw, 0, nil
		}
		bNext, errNext := pEngine.readByte()
		if errNext != nil {
			return 0, 0, errNext
		}
		switch {
		case bNext == g_cbZcrce || bNext == g_cbZcrcg || bNext == g_cbZcrcq || bNext == g_cbZcrcw:
			return 0, bNext, nil
		case bNext == g_cbZrub0:
			return 0x7f, 0, nil
		case bNext == g_cbZrub1:
			return 0xff, 0, nil
		case bNext&0x60 == 0x40:
			return bNext ^ 0x40, 0, nil
		default:
			return 0, 0, ErrProtocol
		}
	}
}

// readEscapedN 은 이스케이프를 푼 바이트 iCount 개를 읽는다. 그 사이에 끝 표시가 나오면 오류다.
func (pEngine *engine) readEscapedN(iCount int) ([]byte, error) {
	bOut := make([]byte, 0, iCount)
	for len(bOut) < iCount {
		bOne, bEnd, errRead := pEngine.readEscaped()
		if errRead != nil {
			return nil, errRead
		}
		if bEnd != 0 {
			return nil, ErrProtocol
		}
		bOut = append(bOut, bOne)
	}
	return bOut, nil
}

func hexValue(bDigit byte) (byte, bool) {
	switch {
	case bDigit >= '0' && bDigit <= '9':
		return bDigit - '0', true
	case bDigit >= 'a' && bDigit <= 'f':
		return bDigit - 'a' + 10, true
	case bDigit >= 'A' && bDigit <= 'F':
		return bDigit - 'A' + 10, true
	}
	return 0, false
}

// tryHeaderBody 는 형식 글자(A·B·C) 뒤의 헤더 본문을 읽는다. CRC 가 맞지 않거나 모양이 틀리면 bOk=false 다.
func (pEngine *engine) tryHeaderBody(bFormat byte) (hdr header, bOk bool, err error) {
	hdr.bFormat = bFormat
	var bBody []byte
	switch bFormat {
	case g_cbZhex:
		bBody = make([]byte, 0, 7)
		for iIndex := 0; iIndex < 7; iIndex++ {
			bHigh, errHigh := pEngine.readByte()
			if errHigh != nil {
				return hdr, false, errHigh
			}
			bLow, errLow := pEngine.readByte()
			if errLow != nil {
				return hdr, false, errLow
			}
			bHighValue, bHighOk := hexValue(bHigh)
			bLowValue, bLowOk := hexValue(bLow)
			if !bHighOk || !bLowOk {
				return hdr, false, nil
			}
			bBody = append(bBody, bHighValue<<4|bLowValue)
		}
		if crc16(bBody[:5]) != binary.BigEndian.Uint16(bBody[5:]) {
			return hdr, false, nil
		}
		// 16진 헤더 뒤의 CR LF 를 먹는다 (LF 는 8번째 비트가 서 있을 수 있다). XON 은 다음 헤더를 찾을 때 건너뛴다.
		bCr, errCr := pEngine.readByte()
		if errCr != nil {
			return hdr, false, errCr
		}
		if bCr == '\r' || bCr == 0x8d {
			bLf, errLf := pEngine.readByte()
			if errLf != nil {
				return hdr, false, errLf
			}
			if bLf != '\n' && bLf != 0x8a {
				pEngine.unreadByte()
			}
		} else {
			pEngine.unreadByte()
		}
	case g_cbZbin:
		bRead, errRead := pEngine.readEscapedN(7)
		if errRead == ErrProtocol {
			return hdr, false, nil
		}
		if errRead != nil {
			return hdr, false, errRead
		}
		if crc16(bRead[:5]) != binary.BigEndian.Uint16(bRead[5:]) {
			return hdr, false, nil
		}
		bBody = bRead
	case g_cbZbin32:
		bRead, errRead := pEngine.readEscapedN(9)
		if errRead == ErrProtocol {
			return hdr, false, nil
		}
		if errRead != nil {
			return hdr, false, errRead
		}
		if crc32.ChecksumIEEE(bRead[:5]) != binary.LittleEndian.Uint32(bRead[5:]) {
			return hdr, false, nil
		}
		bBody = bRead
	default:
		return hdr, false, nil
	}
	hdr.bType = bBody[0]
	copy(hdr.arrData[:], bBody[1:5])
	return hdr, true, nil
}

// readFirstHeader 는 시작 표시로 본 자리에서 헤더 하나를 읽는다. 헤더가 아니면 읽은 것을 모두 되돌려 놓고 ErrNotZmodem 이다.
func (pEngine *engine) readFirstHeader() (header, error) {
	defer func() { pEngine.bKeep = false }()
	for _, bWant := range []byte{g_cbZpad, g_cbZpad, g_cbZdle, g_cbZhex} {
		bOne, errRead := pEngine.readByte()
		if errRead != nil {
			return header{}, errRead
		}
		if bOne != bWant {
			pEngine.iIn = 0
			return header{}, ErrNotZmodem
		}
	}
	hdr, bOk, errBody := pEngine.tryHeaderBody(g_cbZhex)
	if errBody != nil {
		return header{}, errBody
	}
	if !bOk {
		pEngine.iIn = 0
		return header{}, ErrNotZmodem
	}
	// 헤더가 맞다: 여기까지 읽은 부분은 더 되돌릴 일이 없다.
	pEngine.bIn = append([]byte(nil), pEngine.bIn[pEngine.iIn:]...)
	pEngine.iIn = 0
	return hdr, nil
}

// readHeader 는 다음 헤더를 찾을 때까지 읽는다. 헤더 사이의 다른 바이트는 버린다.
// 깨진 헤더는 오류로 세고 계속 찾는다. 헤더 없이 너무 많이 버리면 ErrProtocol 이다.
func (pEngine *engine) readHeader() (header, error) {
	iGarbage := 0
	for {
		bOne, errRead := pEngine.readByte()
		if errRead != nil {
			return header{}, errRead
		}
		if bOne != g_cbZpad {
			iGarbage++
			if iGarbage > g_ciGarbageMax {
				return header{}, ErrProtocol
			}
			continue
		}
		for bOne == g_cbZpad {
			if bOne, errRead = pEngine.readByte(); errRead != nil {
				return header{}, errRead
			}
		}
		if bOne != g_cbZdle {
			continue
		}
		bFormat, errFormat := pEngine.readByte()
		if errFormat != nil {
			return header{}, errFormat
		}
		if bFormat != g_cbZbin && bFormat != g_cbZhex && bFormat != g_cbZbin32 {
			continue
		}
		hdr, bOk, errBody := pEngine.tryHeaderBody(bFormat)
		if errBody != nil {
			return header{}, errBody
		}
		if !bOk {
			if !pEngine.bResync {
				if errCount := pEngine.noteError(); errCount != nil {
					return header{}, errCount
				}
			}
			continue
		}
		if hdr.bType == g_cbZcan || hdr.bType == g_cbZabort {
			return header{}, ErrRemoteCancelled
		}
		return hdr, nil
	}
}

// readSubpacket 은 데이터 조각 하나를 읽는다. CRC 종류는 앞선 헤더의 형식을 따른다(CRC-32 는 'C' 헤더 뒤에서만).
// CRC 가 맞지 않으면 bOk=false 다.
func (pEngine *engine) readSubpacket(bCrc32 bool) (bData []byte, bEnd byte, bOk bool, err error) {
	for {
		bOne, bEndMark, errRead := pEngine.readEscaped()
		if errRead == ErrProtocol {
			return nil, 0, false, nil
		}
		if errRead != nil {
			return nil, 0, false, errRead
		}
		if bEndMark != 0 {
			bEnd = bEndMark
			break
		}
		if len(bData) >= g_ciSubpacketMax {
			return nil, 0, false, nil
		}
		bData = append(bData, bOne)
	}
	if bCrc32 {
		bCrc, errCrc := pEngine.readEscapedN(4)
		if errCrc == ErrProtocol {
			return nil, 0, false, nil
		}
		if errCrc != nil {
			return nil, 0, false, errCrc
		}
		uiWant := crc32.Update(crc32.ChecksumIEEE(bData), crc32.IEEETable, []byte{bEnd})
		return bData, bEnd, uiWant == binary.LittleEndian.Uint32(bCrc), nil
	}
	bCrc, errCrc := pEngine.readEscapedN(2)
	if errCrc == ErrProtocol {
		return nil, 0, false, nil
	}
	if errCrc != nil {
		return nil, 0, false, errCrc
	}
	uiWant := crc16(append(append([]byte(nil), bData...), bEnd))
	return bData, bEnd, uiWant == binary.BigEndian.Uint16(bCrc), nil
}
