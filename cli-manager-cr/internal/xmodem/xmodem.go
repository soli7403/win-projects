// Package xmodem 은 XMODEM 파일 전송(체크섬·CRC·1K)의 보내기·받기다 (spec/behavior/xmodem.md, D164).
//
// 받는 쪽이 전송을 이끈다: 받는 쪽이 'C'(CRC) 또는 NAK(체크섬)를 보내면 보내는 쪽이 블록을 보낸다.
// 블록: SOH(128바이트)·STX(1024바이트), 번호, 255-번호, 데이터, 체크섬 1바이트 또는 CRC-16 2바이트(큰 끝).
// 마지막 블록은 SUB(0x1A)로 채우고, EOT 로 끝낸다. CAN 두 개는 취소다.
package xmodem

import (
	"errors"
	"io"
	"time"
)

// 제어 문자.
const (
	SOH byte = 0x01
	STX byte = 0x02
	EOT byte = 0x04
	ACK byte = 0x06
	NAK byte = 0x15
	CAN byte = 0x18
	SUB byte = 0x1A
	CRC byte = 'C'
)

// ErrTimeout 은 Port.ReadByteWithin 가 시한 안에 아무것도 받지 못했다는 오류다.
var ErrTimeout = errors.New("xmodem: timeout")

// 전송 오류.
var (
	ErrCancelled       = errors.New("xmodem: cancelled")
	ErrRemoteCancelled = errors.New("xmodem: cancelled by remote")
	ErrNoStart         = errors.New("xmodem: remote did not start")
	ErrTooManyErrors   = errors.New("xmodem: too many errors")
	ErrSequence        = errors.New("xmodem: block out of sequence")
)

// Port 는 전송에 쓰는 연결이다.
type Port interface {
	// ReadByteWithin 은 한 바이트를 durWait 동안 기다린다. 시한이 지나면 ErrTimeout, 연결이 끝나면 그 오류다.
	ReadByteWithin(durWait time.Duration) (byte, error)
	// Write 는 바이트를 보낸다.
	Write(bData []byte) error
}

// Options 는 전송 선택 사항이다.
type Options struct {
	// OneK 면 보내기에서 1024바이트 블록을 쓴다(받는 쪽이 CRC 를 고를 때만, 아니면 128).
	OneK bool
	// Cancelled 는 사용자가 취소했는지다. 기다리는 틈마다 묻는다.
	Cancelled func() bool
	// Progress 는 지금까지 보낸·받은 바이트를 알린다. total 은 모르면 0.
	Progress func(i64Done int64, i64Total int64)
}

// 시간 값. 시험에서 줄일 수 있게 변수다.
var (
	// G_durStart 동안 받는 쪽이 시작하지 않으면(보내기) ErrNoStart.
	G_durStart = 60 * time.Second
	// G_durBlock 은 블록 응답(보내기)·블록(받기)을 기다리는 시한이다.
	G_durBlock = 10 * time.Second
	// G_durStartEach 는 받기에서 'C'·NAK 를 다시 보내는 간격이다.
	G_durStartEach = 3 * time.Second
	// G_durByte 는 블록 안의 바이트 사이 시한이다.
	G_durByte = 1 * time.Second
	// G_durPoll 은 취소를 살피는 간격이다.
	G_durPoll = 200 * time.Millisecond
)

const (
	g_ciMaxErrors      = 10 // 한 블록을 다시 보내는·받는 최대 횟수
	g_ciStartTriesMax  = 10 // 받기에서 시작 문자를 보내는 최대 횟수
	g_ciCancelSendSize = 8  // 취소할 때 보내는 CAN 수
)

// g_iCrcStartTries 는 받기에서 'C' 를 보내 보는 횟수다. 그 뒤는 NAK(체크섬). 시험에서 바꾼다.
var g_iCrcStartTries = 3

// CRC16 은 XMODEM 의 CRC-16(CCITT, 다항식 0x1021, 처음 값 0)이다.
func CRC16(bData []byte) uint16 {
	var u16Crc uint16
	for _, b := range bData {
		u16Crc ^= uint16(b) << 8
		for i := 0; i < 8; i++ {
			if u16Crc&0x8000 != 0 {
				u16Crc = u16Crc<<1 ^ 0x1021
			} else {
				u16Crc <<= 1
			}
		}
	}
	return u16Crc
}

// Checksum 은 데이터 바이트의 합(mod 256)이다.
func Checksum(bData []byte) byte {
	var bSum byte
	for _, b := range bData {
		bSum += b
	}
	return bSum
}

// cancelled 는 opts 의 취소 질문이다.
func (opts *Options) cancelled() bool {
	return opts.Cancelled != nil && opts.Cancelled()
}

func (opts *Options) progress(i64Done int64, i64Total int64) {
	if opts.Progress != nil {
		opts.Progress(i64Done, i64Total)
	}
}

// sendCancel 은 상대에게 취소를 알린다.
func sendCancel(port Port) {
	bCan := make([]byte, g_ciCancelSendSize)
	for i := range bCan {
		bCan[i] = CAN
	}
	_ = port.Write(bCan)
}

// readWait 는 durWait 동안 한 바이트를 기다리며 G_durPoll 마다 취소를 살핀다.
func readWait(port Port, durWait time.Duration, opts *Options) (byte, error) {
	tmEnd := time.Now().Add(durWait)
	for {
		if opts.cancelled() {
			return 0, ErrCancelled
		}
		durLeft := time.Until(tmEnd)
		if durLeft <= 0 {
			return 0, ErrTimeout
		}
		if durLeft > G_durPoll {
			durLeft = G_durPoll
		}
		b, errRead := port.ReadByteWithin(durLeft)
		if errors.Is(errRead, ErrTimeout) {
			continue
		}
		return b, errRead
	}
}

// secondCan 은 CAN 하나를 받은 뒤 곧바로 CAN 이 하나 더 오는지 본다(둘이면 취소).
func secondCan(port Port) bool {
	b, errRead := port.ReadByteWithin(G_durByte)
	return errRead == nil && b == CAN
}

// purge 는 줄이 조용해질 때까지 받은 바이트를 버린다(깨진 블록 뒤 NAK 전에).
func purge(port Port) error {
	for {
		_, errRead := port.ReadByteWithin(G_durByte)
		if errors.Is(errRead, ErrTimeout) {
			return nil
		}
		if errRead != nil {
			return errRead
		}
	}
}

// Send 는 r 의 내용을 보낸다. i64Size 는 진행 표시용(모르면 0)이다. 보낸 데이터 바이트 수를 돌려준다.
func Send(port Port, r io.Reader, i64Size int64, opts Options) (int64, error) {
	bCrc, errStart := waitStart(port, &opts)
	if errStart != nil {
		return 0, errStart
	}
	iBlockSize := 128
	if opts.OneK && bCrc {
		iBlockSize = 1024
	}
	bBuf := make([]byte, iBlockSize)
	var i64Done int64
	bBlock := byte(1)
	for {
		nRead, errRead := io.ReadFull(r, bBuf)
		if nRead == 0 {
			if errRead == io.EOF || errRead == io.ErrUnexpectedEOF {
				break
			}
			if errRead != nil {
				sendCancel(port)
				return i64Done, errRead
			}
		}
		iSize := iBlockSize
		if iSize == 1024 && nRead <= 128 {
			iSize = 128 // 끝의 작은 조각은 128바이트 블록으로 (채움을 줄인다)
		}
		bData := make([]byte, iSize)
		copy(bData, bBuf[:nRead])
		for i := nRead; i < iSize; i++ {
			bData[i] = SUB
		}
		if errBlock := sendBlock(port, bBlock, bData, bCrc, &opts); errBlock != nil {
			return i64Done, errBlock
		}
		i64Done += int64(nRead)
		opts.progress(i64Done, i64Size)
		bBlock++
		if errRead == io.EOF || errRead == io.ErrUnexpectedEOF {
			break
		}
		if errRead != nil {
			sendCancel(port)
			return i64Done, errRead
		}
	}
	return i64Done, sendEot(port, &opts)
}

// waitStart 는 받는 쪽의 시작 문자를 기다린다. 'C' 면 CRC, NAK 면 체크섬이다.
func waitStart(port Port, opts *Options) (bool, error) {
	tmEnd := time.Now().Add(G_durStart)
	for {
		b, errRead := readWait(port, time.Until(tmEnd), opts)
		switch {
		case errors.Is(errRead, ErrCancelled):
			sendCancel(port)
			return false, ErrCancelled
		case errors.Is(errRead, ErrTimeout):
			return false, ErrNoStart
		case errRead != nil:
			return false, errRead
		}
		switch b {
		case CRC:
			return true, nil
		case NAK:
			return false, nil
		case CAN:
			if secondCan(port) {
				return false, ErrRemoteCancelled
			}
		}
	}
}

// sendBlock 은 블록 하나를 ACK 를 받을 때까지(최대 g_ciMaxErrors 번) 보낸다.
func sendBlock(port Port, bBlock byte, bData []byte, bCrc bool, opts *Options) error {
	bPacket := make([]byte, 0, len(bData)+5)
	bHead := SOH
	if len(bData) == 1024 {
		bHead = STX
	}
	bPacket = append(bPacket, bHead, bBlock, 255-bBlock)
	bPacket = append(bPacket, bData...)
	if bCrc {
		u16Crc := CRC16(bData)
		bPacket = append(bPacket, byte(u16Crc>>8), byte(u16Crc))
	} else {
		bPacket = append(bPacket, Checksum(bData))
	}
	for iTry := 0; iTry < g_ciMaxErrors; iTry++ {
		if errWrite := port.Write(bPacket); errWrite != nil {
			return errWrite
		}
		bAnswer, errAnswer := waitAnswer(port, opts)
		if errAnswer != nil {
			return errAnswer
		}
		if bAnswer == ACK {
			return nil
		}
		// NAK 또는 시한: 다시 보낸다.
	}
	sendCancel(port)
	return ErrTooManyErrors
}

// waitAnswer 는 블록·EOT 에 대한 답(ACK·NAK)을 기다린다. 시한이 지나면 NAK 로 본다.
func waitAnswer(port Port, opts *Options) (byte, error) {
	tmEnd := time.Now().Add(G_durBlock)
	for {
		b, errRead := readWait(port, time.Until(tmEnd), opts)
		switch {
		case errors.Is(errRead, ErrCancelled):
			sendCancel(port)
			return 0, ErrCancelled
		case errors.Is(errRead, ErrTimeout):
			return NAK, nil
		case errRead != nil:
			return 0, errRead
		}
		switch b {
		case ACK, NAK:
			return b, nil
		case CAN:
			if secondCan(port) {
				return 0, ErrRemoteCancelled
			}
		}
		// 그 밖의 바이트(받는 쪽이 늦게 보낸 'C' 등)는 버리고 계속 기다린다.
	}
}

// sendEot 는 EOT 를 ACK 를 받을 때까지 보낸다. 첫 EOT 에 NAK 를 주는 받는 쪽도 있다.
func sendEot(port Port, opts *Options) error {
	for iTry := 0; iTry < g_ciMaxErrors; iTry++ {
		if errWrite := port.Write([]byte{EOT}); errWrite != nil {
			return errWrite
		}
		bAnswer, errAnswer := waitAnswer(port, opts)
		if errAnswer != nil {
			return errAnswer
		}
		if bAnswer == ACK {
			return nil
		}
	}
	return ErrTooManyErrors
}

// Receive 는 받은 데이터를 w 에 쓴다. 마지막 블록 끝의 SUB 채움은 떼어 낸다. 받은 바이트 수를 돌려준다.
func Receive(port Port, w io.Writer, opts Options) (int64, error) {
	var i64Done int64
	var bHeld []byte // 다음 블록(또는 EOT)이 올 때까지 쥐고 있는 마지막 블록 (끝의 채움을 떼려고)
	flush := func(bLast bool) error {
		if bHeld == nil {
			return nil
		}
		bOut := bHeld
		if bLast {
			for len(bOut) > 0 && bOut[len(bOut)-1] == SUB {
				bOut = bOut[:len(bOut)-1]
			}
		}
		if _, errWrite := w.Write(bOut); errWrite != nil {
			return errWrite
		}
		i64Done += int64(len(bOut))
		bHeld = nil
		return nil
	}
	fail := func(errFail error) (int64, error) {
		if !errors.Is(errFail, ErrRemoteCancelled) {
			sendCancel(port)
		}
		return i64Done, errFail
	}

	// 시작: 'C' 를 몇 번 보내 보고, 답이 없으면 NAK(체크섬)로 바꾼다.
	bCrc := true
	var bHead byte
	bStarted := false
	for iTry := 0; iTry < g_ciStartTriesMax && !bStarted; iTry++ {
		bCrc = iTry < g_iCrcStartTries
		bStart := NAK
		if bCrc {
			bStart = CRC
		}
		if errWrite := port.Write([]byte{bStart}); errWrite != nil {
			return 0, errWrite
		}
		b, errRead := readWait(port, G_durStartEach, &opts)
		switch {
		case errors.Is(errRead, ErrCancelled):
			return fail(ErrCancelled)
		case errors.Is(errRead, ErrTimeout):
			continue
		case errRead != nil:
			return 0, errRead
		}
		switch b {
		case SOH, STX, EOT:
			bHead, bStarted = b, true
		case CAN:
			if secondCan(port) {
				return fail(ErrRemoteCancelled)
			}
		}
	}
	if !bStarted {
		return fail(ErrNoStart)
	}

	bExpect := byte(1)
	iErrors := 0
	for {
		switch bHead {
		case EOT:
			if errFlush := flush(true); errFlush != nil {
				return fail(errFlush)
			}
			if errWrite := port.Write([]byte{ACK}); errWrite != nil {
				return i64Done, errWrite
			}
			return i64Done, nil
		case CAN:
			if secondCan(port) {
				return fail(ErrRemoteCancelled)
			}
		case SOH, STX:
			iSize := 128
			if bHead == STX {
				iSize = 1024
			}
			bBlock, bData, bOk, errBlock := readBlock(port, iSize, bCrc)
			if errBlock != nil {
				return fail(errBlock)
			}
			switch {
			case !bOk:
				iErrors++
				if errPurge := purge(port); errPurge != nil {
					return fail(errPurge)
				}
				if errWrite := port.Write([]byte{NAK}); errWrite != nil {
					return i64Done, errWrite
				}
			case bBlock == bExpect-1:
				// 이미 받은 블록(우리 ACK 가 사라졌다): 다시 ACK 만 한다.
				if errWrite := port.Write([]byte{ACK}); errWrite != nil {
					return i64Done, errWrite
				}
			case bBlock != bExpect:
				return fail(ErrSequence)
			default:
				if errFlush := flush(false); errFlush != nil {
					return fail(errFlush)
				}
				bHeld = bData
				bExpect++
				iErrors = 0
				opts.progress(i64Done+int64(len(bHeld)), 0)
				if errWrite := port.Write([]byte{ACK}); errWrite != nil {
					return i64Done, errWrite
				}
			}
		}
		if iErrors >= g_ciMaxErrors {
			return fail(ErrTooManyErrors)
		}
		// 다음 블록 머리를 기다린다. 시한이 지나면 NAK 로 다시 달라고 한다.
		for {
			b, errRead := readWait(port, G_durBlock, &opts)
			if errors.Is(errRead, ErrCancelled) {
				return fail(ErrCancelled)
			}
			if errors.Is(errRead, ErrTimeout) {
				iErrors++
				if iErrors >= g_ciMaxErrors {
					return fail(ErrTooManyErrors)
				}
				if errWrite := port.Write([]byte{NAK}); errWrite != nil {
					return i64Done, errWrite
				}
				continue
			}
			if errRead != nil {
				return i64Done, errRead
			}
			if b == SOH || b == STX || b == EOT || b == CAN {
				bHead = b
				break
			}
			// 블록 사이의 쓰레기 바이트는 버린다.
		}
	}
}

// readBlock 은 머리 다음의 번호·보수·데이터·검사 값을 읽는다. 모양이 틀리거나 검사 값이 다르면 bOk=false.
func readBlock(port Port, iSize int, bCrc bool) (byte, []byte, bool, error) {
	iCheck := 1
	if bCrc {
		iCheck = 2
	}
	bRaw := make([]byte, 2+iSize+iCheck)
	for i := range bRaw {
		b, errRead := port.ReadByteWithin(G_durByte)
		if errors.Is(errRead, ErrTimeout) {
			return 0, nil, false, nil
		}
		if errRead != nil {
			return 0, nil, false, errRead
		}
		bRaw[i] = b
	}
	bBlock, bComp := bRaw[0], bRaw[1]
	bData := bRaw[2 : 2+iSize]
	if bBlock+bComp != 0xFF {
		return 0, nil, false, nil
	}
	if bCrc {
		u16Got := uint16(bRaw[2+iSize])<<8 | uint16(bRaw[3+iSize])
		if u16Got != CRC16(bData) {
			return bBlock, nil, false, nil
		}
	} else if bRaw[2+iSize] != Checksum(bData) {
		return bBlock, nil, false, nil
	}
	return bBlock, append([]byte(nil), bData...), true, nil
}
