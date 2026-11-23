package zmodem

import (
	"hash/crc32"
	"io"
	"strconv"
	"time"
)

// FileSource 는 보낼 파일 하나다.
type FileSource struct {
	Name    string // 상대에게 알릴 이름 (경로 없이 파일 이름만)
	Size    int64
	ModTime time.Time
	// Open 은 내용을 읽을 곳을 연다. 상대가 중간부터 다시 달라고 할 수 있어 위치를 옮길 수 있어야 한다.
	Open func() (io.ReadSeekCloser, error)
}

// G_durStaleRinit 은 ZFILE 을 보낸 뒤 이 시간 안에 온 ZRINIT 을 "밀려 있던 것"으로 보는 창이다.
// 파일을 고르는 동안 rz 가 ZRINIT 을 여러 번 다시 보내는데, 그것들에 ZFILE 로 일일이 답하지 않기 위해서다.
// 창이 지난 뒤의 ZRINIT 은 상대가 ZFILE 을 받지 못한 것이므로 ZFILE 을 다시 보낸다. 시험에서 줄일 수 있게 변수다.
var G_durStaleRinit = 2 * time.Second

// Send 는 서버(rz)로 파일들을 보낸다. bInitial 은 Detector 가 돌려준 바이트(시작 표시 포함)다.
// fnPick 은 시작 표시가 진짜 ZRINIT 임을 확인한 뒤 불러 보낼 파일을 고르게 한다(빈 목록이면 취소).
// 실패하면(취소 포함) 상대에게 중단을 알린다. 연결 오류(*StreamError)는 그대로 돌려준다.
func Send(stream Stream, bInitial []byte, fnPick func() ([]FileSource, error), opts Options) (Result, error) {
	pEngine := newEngine(stream, bInitial, opts)
	var result Result
	hdr, errFirst := pEngine.readFirstHeader()
	if errFirst == nil && hdr.bType != G_cbZrinit {
		errFirst = ErrNotZmodem
	}
	if errFirst != nil {
		result.Leftover = pEngine.leftover()
		return result, errFirst
	}
	bCrc32 := hdr.arrData[3]&g_cbCanFc32 != 0
	slFiles, errPick := fnPick()
	errRun := errPick
	if errRun == nil && (len(slFiles) == 0 || pEngine.cancelled()) {
		errRun = ErrCancelled // 고르지 않았거나, 고르는 동안 취소했다
	}
	if errRun == nil {
		errRun = pEngine.sendLoop(slFiles, bCrc32, &result)
	}
	result.Leftover = pEngine.leftover()
	if errRun != nil {
		if _, bStream := errRun.(*StreamError); !bStream && errRun != ErrRemoteCancelled {
			_ = stream.Write(AbortSequence())
		}
	}
	return result, errRun
}

// sendLoop 는 파일을 차례로 보내고 세션을 끝낸다.
func (pEngine *engine) sendLoop(slFiles []FileSource, bCrc32 bool, pResult *Result) error {
	for iIndex, file := range slFiles {
		if errFile := pEngine.sendFile(file, len(slFiles)-iIndex, bCrc32, pResult); errFile != nil {
			return errFile
		}
	}
	// 끝: ZFIN 을 보내고 상대의 ZFIN 을 받은 뒤 "OO"(over and out)를 보낸다.
	if errSend := pEngine.sendHex(g_cbZfin, [4]byte{}); errSend != nil {
		return errSend
	}
	for {
		hdr, errHeader := pEngine.readHeader()
		if errHeader != nil {
			return errHeader
		}
		if hdr.bType == g_cbZfin {
			return pEngine.write([]byte("OO"))
		}
	}
}

// fileHeaderFrame 은 ZFILE 헤더와 파일 정보 데이터를 만든다. 변환 방식은 이진(ZCBIN)이다.
func fileHeaderFrame(file FileSource, iFilesLeft int, bCrc32 bool) []byte {
	szInfo := file.Name + "\x00" + strconv.FormatInt(file.Size, 10) + " " +
		strconv.FormatInt(file.ModTime.Unix(), 8) + " 100644 0 " + strconv.Itoa(iFilesLeft) + " " + strconv.FormatInt(file.Size, 10) + "\x00"
	bFrame := encodeBinHeader(g_cbZfile, [4]byte{0, 0, 0, 1}, bCrc32)
	return appendSubpacket(bFrame, []byte(szInfo), g_cbZcrcw, bCrc32)
}

// sendFile 은 파일 하나를 보낸다: ZFILE → (ZRPOS 를 받으면) ZDATA … ZEOF → ZRINIT.
func (pEngine *engine) sendFile(file FileSource, iFilesLeft int, bCrc32 bool, pResult *Result) error {
	reader, errOpen := file.Open()
	if errOpen != nil {
		return errOpen
	}
	defer reader.Close()
	bFileFrame := fileHeaderFrame(file, iFilesLeft, bCrc32)
	if errSend := pEngine.write(bFileFrame); errSend != nil {
		return errSend
	}
	tFileSent := time.Now()
	pEngine.progress(file.Name, 0, file.Size, G_ciStateStart)
	// ZFILE 에 대한 답을 기다린다.
	var i64Pos int64
	for bWaiting := true; bWaiting; {
		hdr, errHeader := pEngine.readHeader()
		if errHeader != nil {
			return errHeader
		}
		switch hdr.bType {
		case g_cbZrpos:
			i64Pos = hdr.pos()
			bWaiting = false
		case g_cbZskip:
			pResult.Skipped++
			pEngine.progress(file.Name, 0, file.Size, G_ciStateSkipped)
			return nil
		case G_cbZrinit:
			if time.Since(tFileSent) < G_durStaleRinit {
				continue // 파일을 고르는 동안 밀려 있던 ZRINIT
			}
			fallthrough
		case g_cbZnak:
			if errCount := pEngine.noteError(); errCount != nil {
				return errCount
			}
			if errSend := pEngine.write(bFileFrame); errSend != nil {
				return errSend
			}
			tFileSent = time.Now()
		case g_cbZcrc:
			// 상대가 이미 가진 파일과 같은지 보려고 파일 전체의 CRC-32 를 묻는다.
			uiCrc, errCrc := fileCrc32(reader)
			if errCrc != nil {
				return errCrc
			}
			if errSend := pEngine.sendHex(g_cbZcrc, posData(int64(uiCrc))); errSend != nil {
				return errSend
			}
		case g_cbZfin:
			return ErrRemoteCancelled
		default:
		}
	}
	for {
		i64End, errData := pEngine.sendData(file, reader, i64Pos, bCrc32)
		if errData != nil {
			return errData
		}
		if i64End >= 0 {
			// 상대가 중간 위치부터 다시 달라고 했다.
			i64Pos = i64End
			continue
		}
		break
	}
	pResult.Files++
	pResult.Bytes += file.Size
	pEngine.progress(file.Name, file.Size, file.Size, G_ciStateDone)
	return nil
}

// fileCrc32 는 파일 전체의 CRC-32 를 계산하고 읽는 위치를 처음으로 되돌린다.
func fileCrc32(reader io.ReadSeeker) (uint32, error) {
	if _, errSeek := reader.Seek(0, io.SeekStart); errSeek != nil {
		return 0, errSeek
	}
	hasher := crc32.NewIEEE()
	if _, errCopy := io.Copy(hasher, reader); errCopy != nil {
		return 0, errCopy
	}
	_, errSeek := reader.Seek(0, io.SeekStart)
	return hasher.Sum32(), errSeek
}

// sendData 는 i64Pos 부터 파일 끝까지 보내고 ZEOF 뒤의 답을 기다린다.
// 다 끝났으면 -1, 상대가 다른 위치부터 다시 달라고 했으면 그 위치를 돌려준다.
func (pEngine *engine) sendData(file FileSource, reader io.ReadSeeker, i64Pos int64, bCrc32 bool) (int64, error) {
	if _, errSeek := reader.Seek(i64Pos, io.SeekStart); errSeek != nil {
		return -1, errSeek
	}
	if errSend := pEngine.write(encodeBinHeader(g_cbZdata, posData(i64Pos), bCrc32)); errSend != nil {
		return -1, errSend
	}
	bChunk := make([]byte, G_ciSubpacketBytes)
	bFrame := make([]byte, 0, G_ciSubpacketBytes*2+16)
	iSinceAck := 0
	for {
		if pEngine.cancelled() {
			return -1, ErrCancelled
		}
		nRead, errRead := io.ReadFull(reader, bChunk)
		bLast := errRead == io.EOF || errRead == io.ErrUnexpectedEOF
		if errRead != nil && !bLast {
			return -1, errRead
		}
		iSinceAck++
		bEnd := byte(g_cbZcrcg)
		switch {
		case bLast:
			bEnd = g_cbZcrce
		case iSinceAck >= g_ciAckEvery:
			bEnd = g_cbZcrcq
		}
		bFrame = appendSubpacket(bFrame[:0], bChunk[:nRead], bEnd, bCrc32)
		if errSend := pEngine.write(bFrame); errSend != nil {
			return -1, errSend
		}
		i64Pos += int64(nRead)
		pEngine.progress(file.Name, i64Pos, file.Size, G_ciStateData)
		if bLast {
			break
		}
		if bEnd == g_cbZcrcq {
			// 확인을 기다린다: ZACK 이면 계속, ZRPOS 면 그 위치부터 다시.
			iSinceAck = 0
			for bWaiting := true; bWaiting; {
				hdr, errHeader := pEngine.readHeader()
				if errHeader != nil {
					return -1, errHeader
				}
				switch hdr.bType {
				case g_cbZack:
					bWaiting = false
				case g_cbZrpos:
					if errCount := pEngine.noteError(); errCount != nil {
						return -1, errCount
					}
					return hdr.pos(), nil
				case g_cbZskip:
					return -1, nil
				case g_cbZfin:
					return -1, ErrRemoteCancelled
				default:
				}
			}
		}
	}
	if errSend := pEngine.write(encodeBinHeader(g_cbZeof, posData(i64Pos), bCrc32)); errSend != nil {
		return -1, errSend
	}
	for {
		hdr, errHeader := pEngine.readHeader()
		if errHeader != nil {
			return -1, errHeader
		}
		switch hdr.bType {
		case G_cbZrinit, g_cbZskip:
			return -1, nil // 파일을 다 받았다 (다음 파일 또는 끝)
		case g_cbZrpos:
			if errCount := pEngine.noteError(); errCount != nil {
				return -1, errCount
			}
			return hdr.pos(), nil
		case g_cbZfin:
			return -1, ErrRemoteCancelled
		default: // ZACK 등은 넘긴다
		}
	}
}
