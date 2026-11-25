package zmodem

import (
	"strconv"
	"strings"
	"time"
)

// FileInfo 는 상대가 보내려는 파일의 정보다 (ZFILE 헤더의 데이터).
type FileInfo struct {
	Name    string    // 상대가 준 이름 그대로 (경로가 붙어 있을 수 있다 — 받는 쪽이 다듬는다)
	Size    int64     // 모르면 -1
	ModTime time.Time // 모르면 zero
}

// FileWriter 는 받는 파일 하나를 쓰는 곳이다.
type FileWriter interface {
	Write(bData []byte) (int, error)
	// Commit 은 파일을 다 받았을 때 부른다 (닫고 최종 이름으로 확정).
	Commit() error
	// Abort 는 받다가 그만둘 때 부른다 (닫고 지운다).
	Abort()
}

// Sink 는 받은 파일을 저장하는 쪽이다.
type Sink interface {
	// Accept 는 파일 하나를 받을지 정한다. (nil, nil) 이면 그 파일을 건너뛴다. 오류면 전송 전체를 중단한다.
	Accept(info FileInfo) (FileWriter, error)
}

// Result 는 전송 결과다.
type Result struct {
	Files    int    // 다 주고받은 파일 수
	Skipped  int    // 건너뛴 파일 수
	Bytes    int64  // 주고받은 바이트 수
	Leftover []byte // ZMODEM 이 아닌 것으로 남은 바이트 (화면 출력으로 되돌린다)
	// DataCrc32 는 받던 데이터 프레임이 CRC-32 였는지다. 받다가 중단했을 때 TrailingOutput 에 넘긴다.
	DataCrc32 bool
	// Dropped 는 취소한 뒤 읽어서 버린 마지막 묶음이다 (화면으로 되돌리지 않는다. 끝자락을 가를 때만 쓴다).
	Dropped []byte
}

// Receive 는 서버가 보내는 파일들을 받는다. bInitial 은 Detector 가 돌려준 바이트(시작 표시 포함)다.
// 시작 표시가 실제 헤더가 아니면 ErrNotZmodem 과 함께 읽은 바이트를 Leftover 로 돌려준다.
// 실패하면(취소 포함) 상대에게 중단을 알린다. 연결 오류(*StreamError)는 그대로 돌려준다.
func Receive(stream Stream, bInitial []byte, sink Sink, opts Options) (Result, error) {
	pEngine := newEngine(stream, bInitial, opts)
	var result Result
	hdr, errFirst := pEngine.readFirstHeader()
	if errFirst == nil && hdr.bType != G_cbZrqinit {
		errFirst = ErrNotZmodem
	}
	if errFirst != nil {
		result.Leftover = pEngine.leftover()
		return result, errFirst
	}
	errRun := pEngine.receiveLoop(sink, &result)
	result.Leftover = pEngine.leftover()
	result.DataCrc32 = pEngine.bDataCrc32
	result.Dropped = pEngine.bDropped
	if errRun != nil {
		if _, bStream := errRun.(*StreamError); !bStream && errRun != ErrRemoteCancelled {
			_ = stream.Write(AbortSequence())
		}
	}
	return result, errRun
}

// sendRinit 는 받을 준비가 됐음을 알린다: 버퍼 크기 0(끊김 없이 받음), 능력 = 양방향·디스크 중 수신·CRC-32.
func (pEngine *engine) sendRinit() error {
	return pEngine.sendHex(G_cbZrinit, [4]byte{0, 0, 0, g_cbCanFdx | g_cbCanOvIo | g_cbCanFc32})
}

// receiveLoop 는 파일 사이의 헤더를 처리한다: ZFILE 이면 파일을 받고, ZFIN 이면 끝낸다.
func (pEngine *engine) receiveLoop(sink Sink, pResult *Result) error {
	if errSend := pEngine.sendRinit(); errSend != nil {
		return errSend
	}
	for {
		hdr, errHeader := pEngine.readHeader()
		if errHeader != nil {
			return errHeader
		}
		switch hdr.bType {
		case G_cbZrqinit:
			if errSend := pEngine.sendRinit(); errSend != nil {
				return errSend
			}
		case g_cbZsinit:
			// 보내는 쪽의 초기 설정(주의 끌기 문자열): 읽고 확인만 한다.
			_, _, bOk, errData := pEngine.readSubpacket(hdr.bFormat == g_cbZbin32)
			if errData != nil {
				return errData
			}
			bReply := byte(g_cbZack)
			if !bOk {
				bReply = g_cbZnak
			}
			if errSend := pEngine.sendHex(bReply, [4]byte{1, 0, 0, 0}); errSend != nil {
				return errSend
			}
		case g_cbZfreecnt:
			if errSend := pEngine.sendHex(g_cbZack, [4]byte{0xff, 0xff, 0xff, 0x7f}); errSend != nil {
				return errSend
			}
		case g_cbZfile:
			bInfo, _, bOk, errData := pEngine.readSubpacket(hdr.bFormat == g_cbZbin32)
			if errData != nil {
				return errData
			}
			if !bOk {
				if errCount := pEngine.noteError(); errCount != nil {
					return errCount
				}
				if errSend := pEngine.sendHex(g_cbZnak, [4]byte{}); errSend != nil {
					return errSend
				}
				continue
			}
			if errFile := pEngine.receiveFile(parseFileInfo(bInfo), sink, pResult); errFile != nil {
				return errFile
			}
		case g_cbZfin:
			if errSend := pEngine.sendHex(g_cbZfin, [4]byte{}); errSend != nil {
				return errSend
			}
			pEngine.eatOverAndOut()
			return nil
		default:
			// 이 자리에서 뜻이 없는 헤더(늦게 온 ZDATA·ZEOF 등)는 넘긴다.
		}
	}
}

// eatOverAndOut 은 상대가 마지막에 보내는 "OO" 를 먹는다. 이미 버퍼에 있을 때만 본다 —
// 없으면 기다리지 않는다(다음 출력과 함께 오면 화면에 "OO" 두 글자가 보일 수 있다).
func (pEngine *engine) eatOverAndOut() {
	// 16진 헤더 뒤의 XON 이 남아 있으면 먼저 넘긴다.
	for pEngine.iIn < len(pEngine.bIn) && pEngine.bIn[pEngine.iIn] == g_cbXon {
		pEngine.iIn++
	}
	if pEngine.iIn+1 < len(pEngine.bIn) && pEngine.bIn[pEngine.iIn] == 'O' && pEngine.bIn[pEngine.iIn+1] == 'O' {
		pEngine.iIn += 2
	}
}

// parseFileInfo 는 ZFILE 데이터를 푼다: "이름\0크기 수정시각(8진) 모드(8진) …\0".
func parseFileInfo(bInfo []byte) FileInfo {
	info := FileInfo{Size: -1}
	szName, szRest, _ := strings.Cut(string(bInfo), "\x00")
	info.Name = szName
	szRest, _, _ = strings.Cut(szRest, "\x00")
	slFields := strings.Fields(szRest)
	if len(slFields) >= 1 {
		if i64Size, errSize := strconv.ParseInt(slFields[0], 10, 64); errSize == nil && i64Size >= 0 {
			info.Size = i64Size
		}
	}
	if len(slFields) >= 2 {
		if i64Time, errTime := strconv.ParseInt(slFields[1], 8, 64); errTime == nil && i64Time > 0 {
			info.ModTime = time.Unix(i64Time, 0)
		}
	}
	return info
}

// receiveFile 은 파일 하나를 받는다. 저장하는 쪽이 건너뛰라고 하면 ZSKIP 을 보낸다.
func (pEngine *engine) receiveFile(info FileInfo, sink Sink, pResult *Result) error {
	writer, errAccept := sink.Accept(info)
	if errAccept != nil {
		return errAccept
	}
	if writer == nil {
		pResult.Skipped++
		pEngine.progress(info.Name, 0, info.Size, G_ciStateSkipped)
		return pEngine.sendHex(g_cbZskip, [4]byte{})
	}
	pEngine.progress(info.Name, 0, info.Size, G_ciStateStart)
	i64Pos, errData := pEngine.receiveData(info, writer)
	if errData != nil {
		writer.Abort()
		return errData
	}
	if errCommit := writer.Commit(); errCommit != nil {
		return errCommit
	}
	pResult.Files++
	pResult.Bytes += i64Pos
	pEngine.progress(info.Name, i64Pos, info.Size, G_ciStateDone)
	return pEngine.sendRinit() // 다음 파일(ZFILE) 또는 끝(ZFIN)을 기다린다
}

// receiveData 는 파일 내용을 받는다. 받은 길이를 돌려준다.
// CRC 가 틀리거나 위치가 어긋나면 ZRPOS 로 지금 위치부터 다시 보내 달라고 한다.
func (pEngine *engine) receiveData(info FileInfo, writer FileWriter) (int64, error) {
	var i64Pos int64
	if errSend := pEngine.sendHex(g_cbZrpos, posData(0)); errSend != nil {
		return 0, errSend
	}
	for {
		hdr, errHeader := pEngine.readHeader()
		if errHeader != nil {
			return i64Pos, errHeader
		}
		switch hdr.bType {
		case g_cbZdata:
			if hdr.pos() != i64Pos&0xffffffff {
				if pEngine.bResync {
					continue // 다시 맞추는 중: 버리는 데이터 안에서 우연히 맞은 헤더이거나 옛 프레임이다
				}
				if errCount := pEngine.noteError(); errCount != nil {
					return i64Pos, errCount
				}
				pEngine.bResync = true
				if errSend := pEngine.sendHex(g_cbZrpos, posData(i64Pos)); errSend != nil {
					return i64Pos, errSend
				}
				continue
			}
			pEngine.bResync = false
			pEngine.bDataCrc32 = hdr.bFormat == g_cbZbin32
			bFrameDone := false
			for !bFrameDone {
				bData, bEnd, bOk, errData := pEngine.readSubpacket(hdr.bFormat == g_cbZbin32)
				if errData != nil {
					return i64Pos, errData
				}
				if !bOk {
					if errCount := pEngine.noteError(); errCount != nil {
						return i64Pos, errCount
					}
					pEngine.bResync = true
					if errSend := pEngine.sendHex(g_cbZrpos, posData(i64Pos)); errSend != nil {
						return i64Pos, errSend
					}
					break
				}
				if len(bData) > 0 {
					if _, errWrite := writer.Write(bData); errWrite != nil {
						_ = pEngine.sendHex(g_cbZferr, [4]byte{})
						return i64Pos, errWrite
					}
					i64Pos += int64(len(bData))
					pEngine.progress(info.Name, i64Pos, info.Size, G_ciStateData)
				}
				switch bEnd {
				case g_cbZcrcw:
					bFrameDone = true
					fallthrough
				case g_cbZcrcq:
					if errSend := pEngine.sendHex(g_cbZack, posData(i64Pos)); errSend != nil {
						return i64Pos, errSend
					}
				case g_cbZcrce:
					bFrameDone = true
				}
			}
		case g_cbZeof:
			if hdr.pos() == i64Pos&0xffffffff {
				pEngine.bResync = false
				return i64Pos, nil
			}
			if pEngine.bResync {
				continue // 다시 맞추는 중에 온 옛 ZEOF: 이미 ZRPOS 를 보냈으니 새 ZDATA 를 기다린다
			}
			// 위치가 다른 ZEOF 는 놓친 데이터가 있다는 뜻이다: 지금 위치부터 다시 받는다.
			if errCount := pEngine.noteError(); errCount != nil {
				return i64Pos, errCount
			}
			pEngine.bResync = true
			if errSend := pEngine.sendHex(g_cbZrpos, posData(i64Pos)); errSend != nil {
				return i64Pos, errSend
			}
		case g_cbZfile:
			// 우리 ZRPOS 를 못 받고 ZFILE 을 다시 보냈다: 데이터는 읽어 버리고 ZRPOS 를 다시 보낸다.
			if _, _, _, errData := pEngine.readSubpacket(hdr.bFormat == g_cbZbin32); errData != nil {
				return i64Pos, errData
			}
			if errSend := pEngine.sendHex(g_cbZrpos, posData(i64Pos)); errSend != nil {
				return i64Pos, errSend
			}
		case g_cbZfin:
			return i64Pos, ErrRemoteCancelled // 파일 중간에 세션을 끝냈다
		default:
		}
	}
}
