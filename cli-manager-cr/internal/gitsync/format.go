package gitsync

import (
	"encoding/json"
	"errors"

	"dariterm/internal/vault"
)

// G_cszSyncFormat 은 저장소 파일의 형식 이름이다 (data/sync-backup.md §5.1).
const G_cszSyncFormat = "dariterm-sync-1"

// G_cszSyncFileName 은 저장소 루트의 파일 이름이다.
const G_cszSyncFileName = "dariterm-sync.json"

// ErrFormat 은 동기화 파일 형식을 알 수 없을 때 돌려준다.
var ErrFormat = errors.New("동기화 파일 형식을 알 수 없습니다")

// syncFile 은 저장소 파일의 구조다 (data/sync-backup.md §5.1).
type syncFile struct {
	M_szFormat string          `json:"format"`
	M_rawVault json.RawMessage `json:"vault"`
	M_payload  vault.Blob      `json:"payload"`
}

// EncodeSyncFile 은 볼트 헤더와 암호화된 payload 를 저장소 파일 바이트로 만든다 (들여쓰기 1칸).
func EncodeSyncFile(rawHeader json.RawMessage, payload vault.Blob) ([]byte, error) {
	return json.MarshalIndent(syncFile{M_szFormat: G_cszSyncFormat, M_rawVault: rawHeader, M_payload: payload}, "", " ")
}

// DecodeSyncFile 은 저장소 파일을 해석한다. 형식 이름이 다르거나 JSON 이 깨졌으면 ErrFormat.
func DecodeSyncFile(bData []byte) (json.RawMessage, vault.Blob, error) {
	var file syncFile
	if errJson := json.Unmarshal(bData, &file); errJson != nil {
		return nil, vault.Blob{}, ErrFormat
	}
	if file.M_szFormat != G_cszSyncFormat {
		return nil, vault.Blob{}, ErrFormat
	}
	return file.M_rawVault, file.M_payload, nil
}

// EncodeData 는 동기화 데이터를 payload 평문(JSON)으로 만든다.
func EncodeData(data Data) ([]byte, error) {
	return json.Marshal(data)
}

// DecodeData 는 payload 평문(JSON)을 데이터로 바꾼다. 빠진 컬렉션은 빈 맵으로 채운다.
func DecodeData(bPlain []byte) (Data, error) {
	data := NewData()
	if errJson := json.Unmarshal(bPlain, &data); errJson != nil {
		return NewData(), errJson
	}
	if data.M_mapGroups == nil {
		data.M_mapGroups = Collection{}
	}
	if data.M_mapHosts == nil {
		data.M_mapHosts = Collection{}
	}
	if data.M_mapSnippets == nil {
		data.M_mapSnippets = Collection{}
	}
	if data.M_mapCreds == nil {
		data.M_mapCreds = Collection{}
	}
	return data, nil
}
