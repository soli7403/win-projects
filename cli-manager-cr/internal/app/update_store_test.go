package app

import (
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"dariterm/internal/storeupdate"
)

// 스토어 업데이트 확인 시험 (lifecycle.md §6.1, D115). 스토어 호출은 가짜로 바꿔 끼운다.

// allEmitter 는 발행된 모든 이벤트를 모은다.
type allEmitter struct {
	mtxData sync.Mutex
	slNames []string
	slData  []any
}

func (pEmit *allEmitter) Emit(szEvent string, anyData ...any) {
	pEmit.mtxData.Lock()
	defer pEmit.mtxData.Unlock()
	pEmit.slNames = append(pEmit.slNames, szEvent)
	var anyFirst any
	if len(anyData) > 0 {
		anyFirst = anyData[0]
	}
	pEmit.slData = append(pEmit.slData, anyFirst)
}

func (pEmit *allEmitter) find(szEvent string) []map[string]any {
	pEmit.mtxData.Lock()
	defer pEmit.mtxData.Unlock()
	var slOut []map[string]any
	for iIndex, szName := range pEmit.slNames {
		if mapData, bOk := pEmit.slData[iIndex].(map[string]any); bOk && szName == szEvent {
			slOut = append(slOut, mapData)
		}
	}
	return slOut
}

// fakeStore 는 스토어 호출을 가짜로 바꾸고 시험이 끝나면 되돌린다.
type fakeStore struct {
	bStore       bool
	pUpdate      *storeupdate.Update
	errCheck     error
	szInstall    string
	errInstall   error
	iInstalls    int
	iPagesOpened int
	chInstall    chan struct{} // 닫힐 때까지 Install 이 기다린다 (nil 이면 바로 끝난다)
	mtxData      sync.Mutex
}

func useFakeStore(t *testing.T, pFake *fakeStore) {
	t.Helper()
	fnIs, fnCheck, fnInstall, fnOpen := g_fnIsStoreInstall, g_fnStoreCheck, g_fnStoreInstall, g_fnOpenStorePage
	g_fnIsStoreInstall = func() bool { return pFake.bStore }
	g_fnStoreCheck = func() (*storeupdate.Update, error) { return pFake.pUpdate, pFake.errCheck }
	g_fnStoreInstall = func() (string, error) {
		pFake.mtxData.Lock()
		pFake.iInstalls++
		pFake.mtxData.Unlock()
		if pFake.chInstall != nil {
			<-pFake.chInstall
		}
		return pFake.szInstall, pFake.errInstall
	}
	g_fnOpenStorePage = func() {
		pFake.mtxData.Lock()
		pFake.iPagesOpened++
		pFake.mtxData.Unlock()
	}
	t.Cleanup(func() {
		g_fnIsStoreInstall, g_fnStoreCheck, g_fnStoreInstall, g_fnOpenStorePage = fnIs, fnCheck, fnInstall, fnOpen
	})
}

func (pFake *fakeStore) counts() (int, int) {
	pFake.mtxData.Lock()
	defer pFake.mtxData.Unlock()
	return pFake.iInstalls, pFake.iPagesOpened
}

func newStoreTestService(t *testing.T) (*Service, *allEmitter) {
	t.Helper()
	pEmit := &allEmitter{}
	pService, errNew := NewService(t.TempDir(), pEmit)
	if errNew != nil {
		t.Fatal(errNew)
	}
	return pService, pEmit
}

func invokeMap(t *testing.T, pService *Service, szChannel string) map[string]any {
	t.Helper()
	szOut, errInvoke := pService.Router().Invoke(szChannel, "[]")
	if errInvoke != nil {
		t.Fatalf("%s: %v", szChannel, errInvoke)
	}
	var mapOut map[string]any
	_ = json.Unmarshal([]byte(szOut), &mapOut)
	return mapOut
}

func waitFor(t *testing.T, fnDone func() bool) {
	t.Helper()
	for tEnd := time.Now().Add(3 * time.Second); time.Now().Before(tEnd); time.Sleep(10 * time.Millisecond) {
		if fnDone() {
			return
		}
	}
	t.Fatal("기다리던 일이 일어나지 않음")
}

// 스토어 설치본이 아니고 배포 저장소도 없으면 업데이트 확인은 꺼져 있고, update.install 은 아무 일도 하지 않는다.
func TestUpdateOffWhenNotStoreAndNoRepo(t *testing.T) {
	t.Setenv("DARITERM_UPDATE_REPO", "")
	pFake := &fakeStore{bStore: false}
	useFakeStore(t, pFake)
	pService, _ := newStoreTestService(t)
	mapInfo := invokeMap(t, pService, "app.info")
	if mapInfo["updateEnabled"] != false || mapInfo["updateSource"] != "" {
		t.Fatalf("꺼져 있어야 함: %v %v", mapInfo["updateEnabled"], mapInfo["updateSource"])
	}
	if mapOut := invokeMap(t, pService, "update.install"); mapOut["started"] != false {
		t.Fatalf("스토어 설치본이 아니면 시작하지 않음: %v", mapOut)
	}
	if iInstalls, _ := pFake.counts(); iInstalls != 0 {
		t.Fatal("스토어를 부르면 안 됨")
	}
}

// 스토어 설치본: 새 버전이 있으면 newer + store 표시, 이벤트에도 store 가 붙고 update.status 로 다시 읽힌다.
func TestStoreUpdateCheckNewer(t *testing.T) {
	useFakeStore(t, &fakeStore{bStore: true, pUpdate: &storeupdate.Update{Version: "1.1.0"}})
	pService, pEmit := newStoreTestService(t)
	if mapInfo := invokeMap(t, pService, "app.info"); mapInfo["updateEnabled"] != true || mapInfo["updateSource"] != "store" {
		t.Fatalf("스토어 설치본이면 켜져 있어야 함: %v", mapInfo)
	}
	mapCheck := invokeMap(t, pService, "update.check")
	if mapCheck["status"] != "newer" || mapCheck["version"] != "1.1.0" || mapCheck["store"] != true || !strings.HasPrefix(mapCheck["url"].(string), "ms-windows-store://") {
		t.Fatalf("확인 결과: %v", mapCheck)
	}
	slEvents := pEmit.find("update.available")
	if len(slEvents) != 1 || slEvents[0]["store"] != true || slEvents[0]["announce"] != true || slEvents[0]["version"] != "1.1.0" {
		t.Fatalf("이벤트: %v", slEvents)
	}
	if mapStatus := invokeMap(t, pService, "update.status"); mapStatus["version"] != "1.1.0" || mapStatus["store"] != true {
		t.Fatalf("update.status: %v", mapStatus)
	}
}

// 스토어 설치본: 새 버전이 없으면 latest, 스토어가 답하지 않으면 error. 둘 다 이벤트를 내지 않는다.
func TestStoreUpdateCheckLatestAndError(t *testing.T) {
	pFake := &fakeStore{bStore: true}
	useFakeStore(t, pFake)
	pService, pEmit := newStoreTestService(t)
	if mapCheck := invokeMap(t, pService, "update.check"); mapCheck["status"] != "latest" {
		t.Fatalf("최신: %v", mapCheck)
	}
	pFake.errCheck = errors.New("no answer")
	if mapCheck := invokeMap(t, pService, "update.check"); mapCheck["status"] != "error" {
		t.Fatalf("오류: %v", mapCheck)
	}
	if len(pEmit.find("update.available")) != 0 {
		t.Fatal("새 버전이 없으면 이벤트를 내지 않아야 함")
	}
}

// update.install: 확인 창을 한 번만 띄우고(겹쳐 누르면 무시), 결과에 따라 토스트를 낸다.
// 완료·취소는 스토어 페이지를 열지 않고, 실패·오류 상태는 스토어 페이지를 대신 연다.
func TestStoreUpdateInstallOutcomes(t *testing.T) {
	for _, tc := range []struct {
		szName    string
		szState   string
		errRun    error
		szKind    string
		iPageOpen int
	}{
		{"완료", storeupdate.G_cszStateCompleted, nil, "ok", 0},
		{"취소", storeupdate.G_cszStateCanceled, nil, "info", 0},
		{"업데이트 없음", "NoUpdate", nil, "ok", 0},
		{"오류 상태", "ErrorWiFiRequired", nil, "error", 1},
		{"창을 띄우지 못함", "", errors.New("powershell failed"), "error", 1},
	} {
		pFake := &fakeStore{bStore: true, pUpdate: &storeupdate.Update{Version: "1.1.0"}, szInstall: tc.szState, errInstall: tc.errRun, chInstall: make(chan struct{})}
		useFakeStore(t, pFake)
		pService, pEmit := newStoreTestService(t)
		invokeMap(t, pService, "update.check")
		for iClick := 0; iClick < 3; iClick++ {
			if mapOut := invokeMap(t, pService, "update.install"); mapOut["started"] != true {
				t.Fatalf("%s: 시작해야 함: %v", tc.szName, mapOut)
			}
		}
		waitFor(t, func() bool { iInstalls, _ := pFake.counts(); return iInstalls == 1 })
		close(pFake.chInstall)
		waitFor(t, func() bool {
			for _, mapToast := range pEmit.find("toast") {
				if mapToast["kind"] == tc.szKind && !strings.Contains(mapToast["text"].(string), "1.1.0") {
					return true
				}
			}
			return false
		})
		iInstalls, iPages := pFake.counts()
		if iInstalls != 1 || iPages != tc.iPageOpen {
			t.Fatalf("%s: 확인 창 %d번(기대 1), 스토어 페이지 %d번(기대 %d)", tc.szName, iInstalls, iPages, tc.iPageOpen)
		}
		if tc.szState == "NoUpdate" {
			if szStatus, _ := pService.Router().Invoke("update.status", "[]"); szStatus != "null" && szStatus != "" {
				t.Fatalf("업데이트가 없다고 답하면 새 버전 표시를 지워야 함: %s", szStatus)
			}
		}
	}
}
