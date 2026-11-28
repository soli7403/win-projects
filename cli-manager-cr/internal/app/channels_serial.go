package app

import (
	"encoding/json"
	"os"

	"dariterm/internal/dialog"
	"dariterm/internal/i18n"
	"dariterm/internal/session"
)

// 시리얼 도구 채널: HEX 보기·HEX 보내기·XMODEM (spec/ipc.md §2 session.hex*·session.xmodem*, D163·D164).

// pickXmodemSend 는 보낼 파일을 고르는 대화상자다. 시험에서 바꾼다.
var pickXmodemSend = func() (string, error) {
	szHome, _ := os.UserHomeDir()
	return dialog.OpenFileOf(i18n.T("be.dlg.xmSend"), szHome, i18n.T("be.dlg.allFiles")+" (*.*)|*.*")
}

func (pService *Service) registerSerialTools() {
	pRouter := pService.pRouter
	sessionArg := func(slArgs []json.RawMessage) (*session.Session, error) {
		szId, errArg := argString(slArgs, 0)
		if errArg != nil {
			return nil, errArg
		}
		return pService.pSessions.Get(szId)
	}
	pRouter.Handle("session.setHexView", func(slArgs []json.RawMessage) (any, error) {
		pSession, errGet := sessionArg(slArgs)
		if errGet != nil {
			return nil, errGet
		}
		bOn, errOn := argBool(slArgs, 1)
		if errOn != nil {
			return nil, errOn
		}
		pSession.SetHexView(bOn)
		return nil, nil
	})
	pRouter.Handle("session.sendHex", func(slArgs []json.RawMessage) (any, error) {
		pSession, errGet := sessionArg(slArgs)
		if errGet != nil {
			return nil, errGet
		}
		szText, errText := argString(slArgs, 1)
		if errText != nil {
			return nil, errText
		}
		bData, errParse := session.ParseHex(szText)
		if errParse != nil {
			return nil, i18n.NewError("be.hex.invalid")
		}
		if errSend := pSession.SendBytes(bData); errSend != nil {
			return nil, errSend
		}
		return len(bData), nil
	})
	pRouter.Handle("session.xmodemSend", func(slArgs []json.RawMessage) (any, error) {
		pSession, errGet := sessionArg(slArgs)
		if errGet != nil {
			return nil, errGet
		}
		var opts struct {
			M_bOneK bool `json:"oneK"`
		}
		if len(slArgs) > 1 {
			if errOpts := argInto(slArgs, 1, &opts); errOpts != nil {
				return nil, errOpts
			}
		}
		szPath, errPick := pickXmodemSend()
		if errPick != nil {
			return nil, errPick
		}
		if szPath == "" {
			return false, nil // 취소
		}
		if errStart := pSession.XmodemSend(szPath, opts.M_bOneK); errStart != nil {
			return nil, errStart
		}
		return true, nil
	})
	pRouter.Handle("session.xmodemReceive", func(slArgs []json.RawMessage) (any, error) {
		pSession, errGet := sessionArg(slArgs)
		if errGet != nil {
			return nil, errGet
		}
		if pService.pDialogs == nil {
			return nil, i18n.NewError("be.serial.notConnected")
		}
		szPath, bOk := pService.pDialogs.SaveFile("xmodem.bin", i18n.T("be.dlg.allFiles"), "*.*")
		if !bOk || szPath == "" {
			return false, nil
		}
		if errStart := pSession.XmodemReceive(szPath); errStart != nil {
			return nil, errStart
		}
		return true, nil
	})
	pRouter.Handle("session.xmodemCancel", func(slArgs []json.RawMessage) (any, error) {
		pSession, errGet := sessionArg(slArgs)
		if errGet != nil {
			return nil, errGet
		}
		pSession.XmodemCancel()
		return nil, nil
	})
}
