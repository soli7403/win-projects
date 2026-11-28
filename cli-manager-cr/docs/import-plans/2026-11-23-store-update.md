# Microsoft Store 설치본의 새 버전 확인과 업데이트 확인 창 — 배경·설계·코드·검증

**작성**: 2026-11-23 · **상태**: 코드·자동 테스트 완료(D115). 버전 확인은 **설치된 스토어 패키지의 신원으로 실제 스토어 응답까지 확인**했다. 업데이트 확인 창(설치)은 띄워 보지 않았다(§2.2) · **대상**: cli-manager-cr(DariTerm)
**묶음**: 빠진 기능 10건 중 10번(작업 중 사용자 추가 요청). 앞 문서 [zmodem](./2026-11-23-zmodem.md). 스토어 배포: [../store/README.md](../store/README.md).

---

## 0. 한 줄 요약

Microsoft Store 로 설치한 실행에서는 **스토어에 새 버전이 있는지 묻고**, 있으면 상태 표시줄의 "새 버전 {버전}" 버튼으로 **Windows 의 업데이트 확인 창**을 띄워 내려받기·설치를 확인할 수 있게 했다. 스토어 설치본이 아닌 실행(설치 파일, 빌드 폴더, Windows 가 아닌 OS)은 전과 같다.

원래 계획한 "코드 서명 + 자체 자동 업데이트"는 **하지 않았다**: 스토어로 배포하면 패키지 서명과 업데이트 설치를 스토어가 맡는다(사용자 안내, 2026-11-23).

## 1. 배경

- 새 버전 확인([internal/app/channels_update.go](../../internal/app/channels_update.go))은 배포 저장소(GitHub 릴리스)를 보는 방식인데, 공개 저장소가 없어 **꺼져 있었다**(D80). 스토어 설치본은 설정에 업데이트 항목 자체가 보이지 않았다.
- 스토어는 앱을 자동으로 업데이트하지만 시점을 Windows 가 정한다(보통 하루 안, 사용자가 자동 업데이트를 꺼 두면 안 된다). 새 버전이 나왔는지 앱 안에서 알 수 없었다.

## 2. 설계 결정

| 항목 | 정한 것 | 이유 |
|---|---|---|
| 언제 스토어에 묻나 | **Windows 의 스토어 설치본일 때만** (`GetCurrentPackageFullName` 으로 패키지된 앱인지 판정) | 사용자 지시. 설치 파일로 설치했거나 개발 빌드면 스토어에 물을 수 없다(패키지 신원이 없다) |
| 묻는 방법 | Windows 의 스토어 기능 `Windows.Services.Store.StoreContext` 의 `GetAppAndOptionalStorePackageUpdatesAsync` | 스토어가 제공하는 정식 방법. 설치된 것보다 새 패키지만 돌려주므로 앱이 버전을 비교할 필요가 없다 |
| 부르는 방법 | **숨긴 PowerShell 스크립트**를 거친다 | StoreContext 는 WinRT API 다. Go 에서 직접 부르려면 COM 가상 함수 표를 번호로 불러야 하는데, 번호 하나가 틀리면 앱이 죽고 이 PC 에서 시험할 방법이 없다. 스크립트는 실패해도 오류를 돌려줄 뿐이다. 이 앱은 파일 대화상자·클립보드 이미지도 같은 방식으로 한다 |
| 패키지 신원 | 패키지된 앱이 띄운 자식 프로세스는 같은 신원을 물려받는다 | 그래서 자식 PowerShell 의 `StoreContext.GetDefault()` 가 이 앱의 것으로 동작한다(§G.2 에서 확인). **[2026-11-26 바로잡음] 이 가정은 틀렸다 — 자식은 따로 정하지 않으면 패키지 밖에서 돈다. [2026-11-26-1-store-update-child-identity.md](./2026-11-26-1-store-update-child-identity.md) (D134)** |
| 업데이트 확인(설치) | `RequestDownloadAndInstallStorePackageUpdatesAsync` — Windows 가 확인 창을 보이고, 확인하면 내려받아 설치한다 | 요청한 "update confirm". 앱이 직접 내려받거나 설치하지 않는다 |
| 확인 창의 주인 | 앱의 주 창 핸들을 `IInitializeWithWindow` 로 알려 준다(스크립트 안의 작은 C# 선언) | 데스크톱 앱은 이것을 하지 않으면 창을 띄울 수 없다 |
| 실패했을 때 | 오류 토스트 + **스토어의 제품 페이지를 연다**(`ms-windows-store://pdp/?productid=9N8WBPMRNBWV`) | 확인 창이 안 떠도 스토어 앱에서 직접 업데이트할 수 있다 |
| 기존 흐름과 합치기 | 같은 채널(`update.check`·`update.status`·이벤트 `update.available`)에 `store: true` 만 더한다. 주기(시작 5초 뒤, 12시간마다)·자동 확인 끄기·처음 발견한 버전만 알림은 그대로 | 화면 코드를 거의 바꾸지 않는다 |
| 새 채널 | `update.install` → `{started}` | 스토어 설치본이면 확인 창을 띄우고 곧바로 돌아온다(결과는 토스트). 아니면 `started: false` |
| 겹쳐 누르기 | 확인 창이 떠 있는 동안 다시 눌러도 하나만 | `atomic.Bool` |
| 버전 표기 | 스토어의 네 자리(`1.1.0.0`)에서 끝의 `.0` 을 뗀다 | 앱이 쓰는 세 자리 표기에 맞춘다 |
| 시험 | 스토어 호출 네 개를 함수 변수로 두어 테스트에서 바꿔 끼운다 | 스토어 없이 흐름 전체를 확인 |

### 2.1 스크립트가 하는 일

확인 스크립트는 업데이트마다 `UPDATE|패키지 이름|주.부.빌드.수정|True/False` 한 줄, 끝에 `COUNT|개수` 를 적는다. `COUNT` 줄이 없으면 끝까지 돌지 못한 것이므로 "업데이트 없음"이 아니라 오류로 본다. 설치 스크립트는 `STATE|상태이름`(`Completed`, `Canceled`, `NoUpdate`, 그 밖에 Windows 의 오류 상태 이름)을 적는다.

### 2.2 확인한 것과 확인하지 못한 것

| | 상태 |
|---|---|
| 스토어 설치본 판정 | 이 PC 의 개발 빌드에서 false(자동 테스트). 패키지된 실행에서 true 가 되는지는 수동 검증 |
| **버전 확인 스크립트** | **확인함**: 이 PC 에 설치된 스토어 패키지(`YangHunLee.DariTerm_1.0.0.0`)의 신원으로 스크립트를 실행(`Invoke-CommandInDesktopPackage`)했더니 스토어가 `COUNT|0`(업데이트 없음)으로 답했다. 패키지 신원이 없는 곳에서는 스토어 호출에서 오류로 끝난다(이것도 확인) |
| 새 버전이 **있을 때**의 응답 | 확인 못 함: 지금 스토어의 최신이 설치된 1.0.0 이다. 다음 버전을 스토어에 올린 뒤에야 볼 수 있다 |
| 설치 스크립트 | C# 선언이 컴파일되고 창 핸들 연결까지 도는 것은 확인(패키지 신원 없이). **실제 확인 창은 띄워 보지 않았다** — 띄우면 이 PC 의 앱이 업데이트될 수 있어 사용자가 직접 해야 한다 |
| 앱이 띄운 자식 프로세스의 신원 | 위 확인은 Windows 의 시험 도구로 패키지 안에서 실행한 것이다. 실제 앱이 띄운 PowerShell 도 같은 신원을 받는다는 것은 Windows 의 문서상 동작이며, 이 빌드를 스토어에 올려 확인해야 한다 |

### 2.3 눈에 띈 점

이 PC 에 설치된 스토어 패키지는 **1.0.0.0** 인데 저장소의 `wails.json` `productVersion` 은 **0.3.0** 이다. 다음 스토어 패키지는 1.0.0 보다 커야 올라간다(`make msix VERSION=1.0.1` 이상). 이 문서의 작업은 버전을 건드리지 않았다.

## A. 바뀐 파일 한눈에

| 파일 | 종류 | 상태 |
|---|---|---|
| `frontend/src/ipc/c1types.ts` | 코드 | 수정 |
| `frontend/src/ipc/fakeBackend.ts` | 코드 | 수정 |
| `frontend/src/screens/SettingsAbout.tsx` | 코드 | 수정 |
| `frontend/src/state/update.ts` | 코드 | 수정 |
| `frontend/src/ui/UpdateBadge.tsx` | 코드 | 수정 |
| `internal/app/app.go` | 코드 | 수정 |
| `internal/app/channels_update.go` | 코드 | 수정 |
| `internal/storeupdate/storeupdate.go` | 코드 | 새 파일 |
| `internal/storeupdate/storeupdate_other.go` | 코드 | 새 파일 |
| `internal/storeupdate/storeupdate_windows.go` | 코드 | 새 파일 |
| `internal/update/update.go` | 코드 | 수정 |
| `internal/app/update_store_test.go` | 테스트 | 새 파일 |
| `internal/storeupdate/storeupdate_test.go` | 테스트 | 새 파일 |
| `frontend/src/ipc/contract/app.info.json` | 계약 고정 파일 | 수정 |
| `spec/behavior/lifecycle.md` | 명세 | 수정 |
| `spec/ipc.md` | 명세 | 수정 |
| `locales/de.json` | 문구 | 수정 |
| `locales/en.json` | 문구 | 수정 |
| `locales/es.json` | 문구 | 수정 |
| `locales/fr.json` | 문구 | 수정 |
| `locales/ja.json` | 문구 | 수정 |
| `locales/ko.json` | 문구 | 수정 |
| `locales/pt.json` | 문구 | 수정 |
| `locales/ru.json` | 문구 | 수정 |
| `locales/tw.json` | 문구 | 수정 |
| `locales/zh.json` | 문구 | 수정 |

아래의 `diff` 는 **이 묶음의 앞 문서들을 차례로 적용한 상태**를 기준으로 뽑았다. 앞 문서에서도 고친 파일(문맥 줄에 앞 기능의 코드가 섞여 있을 수 있다): `internal/app/app.go`. 이 기능만 따로 넣을 때는 `+`/`-` 줄만 옮기면 된다.

## B. 코드

### `frontend/src/ipc/c1types.ts` — 수정

```diff
--- a/frontend/src/ipc/c1types.ts
+++ b/frontend/src/ipc/c1types.ts
@@ -49,11 +49,14 @@
   version: string;
   url: string;
   announce?: boolean;
+  /** Microsoft Store 에서 찾은 새 버전(lifecycle.md §6.1). 누르면 update.install 로 Windows 의 업데이트 확인 창을 띄운다. */
+  store?: boolean;
 }
 
 export interface UpdateCheckResult {
   status: 'newer' | 'latest' | 'error' | 'unconfigured';
   version?: string;
   url?: string;
+  store?: boolean;
 }
 
```

### `frontend/src/ipc/fakeBackend.ts` — 수정

```diff
--- a/frontend/src/ipc/fakeBackend.ts
+++ b/frontend/src/ipc/fakeBackend.ts
@@ -267,6 +267,7 @@
     'import.apply': ([list]) => ({ hosts: Array.isArray(list) ? list.length : 0, keys: 0, skipped: [] }),
     'sync.run': () => ({ state: 'ok', lastSync: Date.now(), changed: false }),
     'update.check': () => ({ status: 'latest' }),
+    'update.install': () => ({ started: false }),
     'backup.export': () => false,
     'backup.import': () => false,
     // SFTP 채널(fakeSftp.ts). 세션 목록은 이 파일의 sessions 를 그대로 쓴다.
```

### `frontend/src/screens/SettingsAbout.tsx` — 수정

```diff
--- a/frontend/src/screens/SettingsAbout.tsx
+++ b/frontend/src/screens/SettingsAbout.tsx
@@ -21,7 +21,7 @@
 export function SettingsAbout() {
   const [updateCheck, setUpdateCheck] = useSetting('updateCheck');
   const checkNow = useUpdate((s) => s.checkNow);
-  const [info, setInfo] = useState<{ version?: string; dataDir?: string; updateEnabled?: boolean }>({});
+  const [info, setInfo] = useState<{ version?: string; dataDir?: string; updateEnabled?: boolean; updateSource?: string }>({});
   const pushToast = useApp((s) => s.pushToast);
   const links = aboutLinks();
   const openLink = (url: string) => {
@@ -38,7 +38,7 @@
 
   useEffect(() => {
     getBackend()
-      .invoke<{ version?: string; dataDir?: string; updateEnabled?: boolean }>('app.info')
+      .invoke<{ version?: string; dataDir?: string; updateEnabled?: boolean; updateSource?: string }>('app.info')
       .then((i) => setInfo(i ?? {}))
       .catch((e) => pushToast({ kind: 'error', text: String(e?.message ?? e) }));
     // pushToast 는 안정적인 참조다.
@@ -112,7 +112,7 @@
         <Group>
           <ToggleRow
             title={t('about.updateCheck')}
-            desc={t('about.updateCheckDesc')}
+            desc={t(info.updateSource === 'store' ? 'about.updateCheckStoreDesc' : 'about.updateCheckDesc')}
             checked={updateCheck}
             onChange={setUpdateCheck}
           />
```

### `frontend/src/state/update.ts` — 수정

```diff
--- a/frontend/src/state/update.ts
+++ b/frontend/src/state/update.ts
@@ -19,8 +19,8 @@
   setAvailable: (u) => set({ available: u }),
   async loadStatus() {
     try {
-      const r = await getBackend().invoke<{ status?: string; version: string; url: string } | null>('update.status');
-      if (r && r.version) set({ available: { version: r.version, url: r.url } });
+      const r = await getBackend().invoke<{ status?: string; version: string; url: string; store?: boolean } | null>('update.status');
+      if (r && r.version) set({ available: { version: r.version, url: r.url, store: !!r.store } });
     } catch {
       /* 조회 실패는 조용히 넘어간다 */
     }
@@ -30,7 +30,7 @@
     try {
       const r = await getBackend().invoke<UpdateCheckResult>('update.check');
       if (r.status === 'newer') {
-        set({ available: { version: r.version ?? '', url: r.url ?? '' } });
+        set({ available: { version: r.version ?? '', url: r.url ?? '', store: !!r.store } });
         push({ kind: 'info', text: t('update.newer', { version: r.version ?? '' }) });
       } else if (r.status === 'latest') push({ kind: 'ok', text: t('update.latest') });
       else if (r.status === 'unconfigured') push({ kind: 'ok', text: t('update.unconfigured') });
@@ -44,7 +44,7 @@
 /** update.available 이벤트를 받아 상태와 안내 토스트를 갱신한다. 구독 해제 함수를 돌려준다. */
 export function subscribeUpdateEvents(): () => void {
   const off = getBackend().on('update.available', (u: UpdateInfo) => {
-    useUpdate.getState().setAvailable({ version: u.version, url: u.url });
+    useUpdate.getState().setAvailable({ version: u.version, url: u.url, store: !!u.store });
     if (u.announce) {
       useApp.getState().pushToast({ kind: 'info', text: t('update.announce', { version: u.version }) });
     }
```

### `frontend/src/ui/UpdateBadge.tsx` — 수정

```diff
--- a/frontend/src/ui/UpdateBadge.tsx
+++ b/frontend/src/ui/UpdateBadge.tsx
@@ -1,4 +1,5 @@
-// 상태 표시줄의 "새 버전 {버전}" 버튼. 누르면 배포 페이지를 외부 브라우저로 연다. 근거: spec/ui/shell.md §3.
+// 상태 표시줄의 "새 버전 {버전}" 버튼. 누르면 배포 페이지를 외부 브라우저로 연다.
+// Microsoft Store 설치본이면 Windows 의 업데이트 확인 창을 띄운다. 근거: spec/ui/shell.md §3, spec/behavior/lifecycle.md §6.1.
 import { getBackend, errorMessage } from '../ipc/backend';
 import { useApp } from '../state/app';
 import { useUpdate } from '../state/update';
@@ -8,15 +9,20 @@
   const info = useUpdate((s) => s.available);
   if (!info) return null;
   const open = async () => {
-    if (!info.url) return;
     try {
+      if (info.store) {
+        // 스토어 설치본: Windows 의 업데이트 확인 창을 띄운다(결과는 토스트로 온다). lifecycle.md §6.1
+        await getBackend().invoke('update.install');
+        return;
+      }
+      if (!info.url) return;
       await getBackend().invoke('app.openExternal', [info.url]);
     } catch (e) {
       useApp.getState().pushToast({ kind: 'error', text: errorMessage(e) });
     }
   };
   return (
-    <button className="update-badge" type="button" title={t('update.openTitle')} onClick={() => void open()}>
+    <button className="update-badge" type="button" title={t(info.store ? 'update.storeTitle' : 'update.openTitle')} onClick={() => void open()}>
       {t('update.badge', { version: info.version })}
     </button>
   );
```

### `internal/app/app.go` — 수정

```diff
--- a/internal/app/app.go
+++ b/internal/app/app.go
@@ -214,6 +214,8 @@
 			"domainJoined": license.DomainJoined(),
 			// 업데이트 확인 기능을 쓰는 빌드인지(D80). false 면 화면에서 업데이트 항목을 감춘다.
 			"updateEnabled": UpdateEnabled(),
+			// 새 버전을 묻는 곳: "store"(Microsoft Store 설치본, D115) · "release"(배포 저장소) · ""(확인하지 않음).
+			"updateSource": updateSource(),
 		}, nil
 	})
 }
```

### `internal/app/channels_update.go` — 수정

```diff
--- a/internal/app/channels_update.go
+++ b/internal/app/channels_update.go
@@ -7,11 +7,52 @@
 	"os"
 	"path/filepath"
 	"sync"
+	"sync/atomic"
 	"time"
 
 	"dariterm/internal/jsonfile"
+	"dariterm/internal/storeupdate"
 	"dariterm/internal/update"
 )
+
+// 새 버전을 어디에 묻는지 (lifecycle.md §6).
+const (
+	g_cszUpdateSourceStore   = "store"   // Microsoft Store 설치본: 스토어에 묻는다 (D115)
+	g_cszUpdateSourceRelease = "release" // 배포 저장소가 정해진 빌드: 저장소의 최신 릴리스를 본다 (D80)
+)
+
+// 스토어 호출. 시험에서 바꿔 끼운다.
+var (
+	g_fnIsStoreInstall = storeupdate.IsStoreInstall
+	g_fnStoreCheck     = storeupdate.Check
+	g_fnStoreInstall   = storeupdate.Install
+	g_fnOpenStorePage  = func() { _ = openWithShell(storeupdate.G_cszStorePageUri) }
+)
+
+// updateSource 는 이 실행이 새 버전을 어디에 묻는지 정한다. Windows 의 스토어 설치본이면 스토어,
+// 아니고 배포 저장소가 정해져 있으면 저장소, 둘 다 아니면 빈 값(확인하지 않음)이다.
+func updateSource() string {
+	if g_fnIsStoreInstall() {
+		return g_cszUpdateSourceStore
+	}
+	if updateRepo() != "" {
+		return g_cszUpdateSourceRelease
+	}
+	return ""
+}
+
+// storeCheckResult 는 스토어에 물은 결과를 확인 결과 형식으로 바꾼다.
+func storeCheckResult() update.Result {
+	pUpdate, errCheck := g_fnStoreCheck()
+	switch {
+	case errCheck != nil:
+		return update.Result{M_szStatus: update.G_cszStatusError}
+	case pUpdate == nil:
+		return update.Result{M_szStatus: update.G_cszStatusLatest}
+	default:
+		return update.Result{M_szStatus: update.G_cszStatusNewer, M_szVersion: pUpdate.Version, M_szUrl: storeupdate.G_cszStorePageUri, M_bStore: true}
+	}
+}
 
 // G_cszUpdateRepo 는 빌드 때 -ldflags -X 로 정하는 배포 저장소(owner/name)다. 비어 있으면 환경 변수를 본다.
 var G_cszUpdateRepo = ""
@@ -24,10 +65,11 @@
 
 // updateState 는 마지막으로 찾은 새 버전과 종료 신호다.
 type updateState struct {
-	mtxData  sync.Mutex
-	pNewer   *update.Result
-	chStop   chan struct{}
-	onceStop sync.Once
+	mtxData     sync.Mutex
+	pNewer      *update.Result
+	chStop      chan struct{}
+	onceStop    sync.Once
+	bInstalling atomic.Bool // 스토어 업데이트 확인 창이 떠 있는 중 (겹쳐 띄우지 않는다)
 }
 
 // updateRepo 는 배포 저장소 주소를 정한다 (환경 변수 DARITERM_UPDATE_REPO 가 빌드 값보다 우선).
@@ -53,7 +95,12 @@
 
 // checkUpdate 는 확인을 한 번 하고 필요하면 안내한다. bManual 이면 항상 안내 토스트를 띄운다.
 func (pService *Service) checkUpdate(bManual bool) update.Result {
-	res := update.Check(updateApi(), updateRepo(), G_cszAppVersion, &http.Client{Timeout: update.G_cdurRequestTimeout})
+	var res update.Result
+	if updateSource() == g_cszUpdateSourceStore {
+		res = storeCheckResult()
+	} else {
+		res = update.Check(updateApi(), updateRepo(), G_cszAppVersion, &http.Client{Timeout: update.G_cdurRequestTimeout})
+	}
 	if res.M_szStatus != update.G_cszStatusNewer {
 		return res
 	}
@@ -69,7 +116,11 @@
 	}
 	bFirst := file.M_szNotified != res.M_szVersion
 	bAnnounce := bFirst || bManual
-	pService.pRouter.Emit("update.available", map[string]any{"version": res.M_szVersion, "url": res.M_szUrl, "announce": bAnnounce})
+	mapEvent := map[string]any{"version": res.M_szVersion, "url": res.M_szUrl, "announce": bAnnounce}
+	if res.M_bStore {
+		mapEvent["store"] = true
+	}
+	pService.pRouter.Emit("update.available", mapEvent)
 	if bAnnounce {
 		pService.pRouter.Emit("toast", map[string]any{"kind": "info", "text": i18n.T("update.announce", "version", res.M_szVersion)})
 	}
@@ -92,10 +143,38 @@
 	pService.checkUpdate(false)
 }
 
-// UpdateEnabled 는 업데이트 확인 기능을 쓰는 빌드인지다: 배포 저장소 주소가 있어야 켜진다 (D80).
-// 지금은 공개 릴리스 저장소가 없어 꺼져 있다. 켜려면 G_cszUpdateRepo 를 정한다(빌드 -ldflags -X 또는 이 값).
+// UpdateEnabled 는 업데이트 확인 기능을 쓰는 실행인지다: Microsoft Store 설치본이거나(D115)
+// 배포 저장소 주소가 있어야 켜진다(D80). 설치 파일로 설치했고 저장소도 없으면 꺼져 있다.
 func UpdateEnabled() bool {
-	return updateRepo() != ""
+	return updateSource() != ""
+}
+
+// runStoreInstall 은 Windows 의 스토어 업데이트 확인 창을 띄우고 결과를 토스트로 알린다 (lifecycle.md §6.1).
+// 사용자가 확인하면 Windows 가 내려받아 설치하며, 그 과정에서 앱이 닫힐 수 있다.
+// 창을 띄우지 못했거나 Windows 가 오류 상태를 알려 주면 Microsoft Store 의 제품 페이지를 대신 연다.
+func (pService *Service) runStoreInstall() {
+	fnToast := func(szKind string, szText string) {
+		pService.pRouter.Emit("toast", map[string]any{"kind": szKind, "text": szText})
+	}
+	szState, errInstall := g_fnStoreInstall()
+	switch {
+	case errInstall != nil:
+		fnToast("error", i18n.T("be.update.storeFailed"))
+		g_fnOpenStorePage()
+	case szState == storeupdate.G_cszStateCompleted:
+		fnToast("ok", i18n.T("be.update.storeDone"))
+	case szState == storeupdate.G_cszStateCanceled:
+		fnToast("info", i18n.T("be.update.storeCanceled"))
+	case szState == "NoUpdate":
+		pState := pService.pUpdate
+		pState.mtxData.Lock()
+		pState.pNewer = nil
+		pState.mtxData.Unlock()
+		fnToast("ok", i18n.T("update.latest"))
+	default:
+		fnToast("error", i18n.T("be.update.storeState", "state", szState))
+		g_fnOpenStorePage()
+	}
 }
 
 // startUpdateLoop 은 시작 5초 뒤 첫 확인, 이후 12시간마다 확인한다. 기능이 꺼진 빌드면 아무것도 하지 않는다.
@@ -140,6 +219,21 @@
 	pRouter.Handle("update.check", func(slArgs []json.RawMessage) (any, error) {
 		return pService.checkUpdate(true), nil
 	})
+	// update.install: 스토어 설치본이면 Windows 의 업데이트 확인 창을 띄운다(끝을 기다리지 않고 바로 돌아온다).
+	// 그 밖의 실행에서는 아무 일도 하지 않고 started=false 를 돌려준다 — 화면이 내려받기 페이지를 연다.
+	pRouter.Handle("update.install", func(slArgs []json.RawMessage) (any, error) {
+		if updateSource() != g_cszUpdateSourceStore {
+			return map[string]any{"started": false}, nil
+		}
+		pState := pService.pUpdate
+		if pState.bInstalling.CompareAndSwap(false, true) {
+			go func() {
+				defer pState.bInstalling.Store(false)
+				pService.runStoreInstall()
+			}()
+		}
+		return map[string]any{"started": true}, nil
+	})
 	pRouter.Handle("update.status", func(slArgs []json.RawMessage) (any, error) {
 		pState := pService.pUpdate
 		pState.mtxData.Lock()
```

### `internal/storeupdate/storeupdate.go` — 새 파일 (전체)

```go
// Package storeupdate 는 Microsoft Store 로 설치한 앱의 새 버전을 스토어에 묻고, 업데이트 확인 창을 띄운다
// (spec/behavior/lifecycle.md §6.1, D115). Windows 의 스토어 설치본에서만 동작하고, 그 밖에서는 아무 일도 하지 않는다.
package storeupdate

import (
	"errors"
	"strings"
)

// 스토어의 제품 정보 (docs/store/README.md §0 — Partner Center 가 정한 값).
const (
	// G_cszProductId 는 스토어 제품 ID 다.
	G_cszProductId = "9N8WBPMRNBWV"
	// G_cszPackageName 은 패키지 이름(Identity Name)이다. 스토어가 돌려준 업데이트 목록에서 이 앱의 것을 고를 때 쓴다.
	G_cszPackageName = "YangHunLee.DariTerm"
	// G_cszStorePageUri 는 Microsoft Store 앱에서 이 제품 페이지를 여는 주소다.
	G_cszStorePageUri = "ms-windows-store://pdp/?productid=" + G_cszProductId
)

// 설치 요청의 결과 상태 (Windows 의 StorePackageUpdateState 이름 그대로).
const (
	G_cszStateCompleted = "Completed"
	G_cszStateCanceled  = "Canceled"
)

// ErrNotStore 는 스토어 설치본이 아닌 곳에서 불렀을 때의 오류다.
var ErrNotStore = errors.New("storeupdate: not a Microsoft Store install")

// Update 는 스토어에 올라온 새 버전 하나다.
type Update struct {
	Version   string // "1.2.3" (끝의 ".0" 네 번째 자리는 뗀다)
	Mandatory bool   // 스토어에서 필수 업데이트로 표시한 버전
}

// parseCheckOutput 은 확인 스크립트의 출력 줄을 해석한다.
// 줄 형식: "UPDATE|패키지 이름|주.부.빌드.수정|True 또는 False", 마지막에 "COUNT|개수".
// "COUNT" 줄이 없으면 스크립트가 끝까지 돌지 못한 것이므로 오류다. 이 앱의 업데이트가 없으면 (nil, nil) 이다.
func parseCheckOutput(slLines []string) (*Update, error) {
	var pFound *Update
	bDone := false
	for _, szLine := range slLines {
		slParts := strings.Split(strings.TrimSpace(szLine), "|")
		switch {
		case len(slParts) == 2 && slParts[0] == "COUNT":
			bDone = true
		case len(slParts) == 4 && slParts[0] == "UPDATE":
			if !strings.EqualFold(slParts[1], G_cszPackageName) || pFound != nil {
				continue // 선택 패키지 등 다른 것의 업데이트
			}
			pFound = &Update{Version: displayVersion(slParts[2]), Mandatory: strings.EqualFold(slParts[3], "True")}
		}
	}
	if !bDone {
		return nil, errors.New("storeupdate: store did not answer")
	}
	return pFound, nil
}

// parseInstallOutput 은 설치 스크립트의 출력에서 결과 상태("STATE|이름")를 찾는다.
func parseInstallOutput(slLines []string) (string, error) {
	for _, szLine := range slLines {
		if szState, bFound := strings.CutPrefix(strings.TrimSpace(szLine), "STATE|"); bFound && szState != "" {
			return szState, nil
		}
	}
	return "", errors.New("storeupdate: store did not answer")
}

// displayVersion 은 스토어의 네 자리 버전에서 끝의 ".0" 을 떼어 앱이 쓰는 세 자리 표기로 바꾼다.
func displayVersion(szVersion string) string {
	slParts := strings.Split(szVersion, ".")
	if len(slParts) == 4 && slParts[3] == "0" {
		return strings.Join(slParts[:3], ".")
	}
	return szVersion
}
```

### `internal/storeupdate/storeupdate_other.go` — 새 파일 (전체)

```go
//go:build !windows

package storeupdate

// IsStoreInstall 은 Windows 가 아니면 항상 false 다 (Microsoft Store 는 Windows 에만 있다).
func IsStoreInstall() bool { return false }

// Check 는 Windows 가 아니면 쓸 수 없다.
func Check() (*Update, error) { return nil, ErrNotStore }

// Install 은 Windows 가 아니면 쓸 수 없다.
func Install() (string, error) { return "", ErrNotStore }
```

### `internal/storeupdate/storeupdate_windows.go` — 새 파일 (전체)

```go
//go:build windows

package storeupdate

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"dariterm/internal/procutil"
)

// Windows 의 스토어 기능(Windows.Services.Store.StoreContext)은 WinRT API 라서, 이 앱의 다른 OS 대화상자와 같이
// 숨긴 PowerShell 을 거쳐 부른다. 패키지된 앱이 띄운 자식 프로세스는 같은 패키지 신원을 물려받으므로
// StoreContext 가 이 앱의 것으로 동작한다. 스크립트가 실패하면 오류를 돌려줄 뿐 앱에는 영향이 없다.

const (
	// g_ciNoPackage 는 GetCurrentPackageFullName 이 "패키지된 앱이 아님"을 알리는 값(APPMODEL_ERROR_NO_PACKAGE)이다.
	g_ciNoPackage = 15700
	// G_cdurCheckTimeout 은 새 버전 확인의 시한이다.
	G_cdurCheckTimeout = 30 * time.Second
	// G_cdurInstallTimeout 은 내려받기·설치(사용자 확인 포함)의 시한이다.
	G_cdurInstallTimeout = 60 * time.Minute
)

var (
	g_procPackageFullName = syscall.NewLazyDLL("kernel32.dll").NewProc("GetCurrentPackageFullName")
	g_procEnumWindows     = syscall.NewLazyDLL("user32.dll").NewProc("EnumWindows")
	g_procWindowThreadPid = syscall.NewLazyDLL("user32.dll").NewProc("GetWindowThreadProcessId")
	g_procIsWindowVisible = syscall.NewLazyDLL("user32.dll").NewProc("IsWindowVisible")
	g_procGetWindow       = syscall.NewLazyDLL("user32.dll").NewProc("GetWindow")
	g_procCurrentPid      = syscall.NewLazyDLL("kernel32.dll").NewProc("GetCurrentProcessId")

	g_onceStore sync.Once
	g_bStore    bool
)

// IsStoreInstall 은 이 프로세스가 패키지된 앱(Microsoft Store 설치본)으로 실행 중인지 알려 준다.
// 설치 파일(NSIS)로 설치했거나 빌드 폴더에서 바로 실행한 경우는 false 다.
func IsStoreInstall() bool {
	g_onceStore.Do(func() {
		if g_procPackageFullName.Find() != nil {
			return // Windows 8 이전: 패키지라는 것이 없다
		}
		var uiLength uint32
		rCode, _, _ := g_procPackageFullName.Call(uintptr(unsafe.Pointer(&uiLength)), 0)
		g_bStore = rCode != g_ciNoPackage && uiLength > 0
	})
	return g_bStore
}

// g_cszPreamble 은 두 스크립트가 함께 쓰는 앞부분이다: WinRT 형식을 불러오고, 비동기 작업을 기다리는 함수를 만든다.
const g_cszPreamble = `$ErrorActionPreference = 'Stop'
Add-Type -AssemblyName System.Runtime.WindowsRuntime
$null = [Windows.Services.Store.StoreContext, Windows.Services.Store, ContentType = WindowsRuntime]
$null = [Windows.Services.Store.StorePackageUpdate, Windows.Services.Store, ContentType = WindowsRuntime]
$null = [Windows.Services.Store.StorePackageUpdateResult, Windows.Services.Store, ContentType = WindowsRuntime]
$null = [Windows.Services.Store.StorePackageUpdateStatus, Windows.Services.Store, ContentType = WindowsRuntime]
function Wait-Async($op, $kind, $types) {
  $m = [System.WindowsRuntimeSystemExtensions].GetMethods() | Where-Object {
    $_.Name -eq 'AsTask' -and $_.GetParameters().Count -eq 1 -and $_.GetParameters()[0].ParameterType.Name -eq $kind
  } | Select-Object -First 1
  $task = $m.MakeGenericMethod($types).Invoke($null, @($op))
  $null = $task.Wait(-1)
  $task.Result
}
$ctx = [Windows.Services.Store.StoreContext]::GetDefault()
$listType = [System.Collections.Generic.IReadOnlyList[Windows.Services.Store.StorePackageUpdate]]
`

// g_cszCheckScript 는 스토어에 이 앱의 업데이트 목록을 묻고 한 줄씩 적는다.
const g_cszCheckScript = g_cszPreamble + `$list = Wait-Async ($ctx.GetAppAndOptionalStorePackageUpdatesAsync()) 'IAsyncOperation` + "`" + `1' @($listType)
foreach ($u in $list) {
  $v = $u.Package.Id.Version
  Write-Output ("UPDATE|{0}|{1}.{2}.{3}.{4}|{5}" -f $u.Package.Id.Name, $v.Major, $v.Minor, $v.Build, $v.Revision, $u.Mandatory)
}
Write-Output ("COUNT|{0}" -f @($list).Count)
`

// g_cszInstallScript 는 업데이트를 내려받아 설치하도록 요청한다. Windows 가 사용자에게 확인 창을 보인다.
// 데스크톱 앱은 확인 창의 주인이 될 창 핸들을 먼저 알려 줘야 한다(IInitializeWithWindow). __HWND__ 자리에 핸들 값을 넣는다.
const g_cszInstallScript = g_cszPreamble + `Add-Type -TypeDefinition @"
using System;
using System.Runtime.InteropServices;
[ComImport, Guid("3E68D4BD-7135-4D10-8018-9FB6D9F33FA1"), InterfaceType(ComInterfaceType.InterfaceIsIUnknown)]
public interface IDtInitializeWithWindow { void Initialize(IntPtr hwnd); }
public static class DtStoreWindow {
  public static void Attach(object target, long hwnd) { ((IDtInitializeWithWindow)target).Initialize(new IntPtr(hwnd)); }
}
"@
[DtStoreWindow]::Attach($ctx, __HWND__)
$list = Wait-Async ($ctx.GetAppAndOptionalStorePackageUpdatesAsync()) 'IAsyncOperation` + "`" + `1' @($listType)
if (@($list).Count -eq 0) { Write-Output 'STATE|NoUpdate'; exit 0 }
$result = Wait-Async ($ctx.RequestDownloadAndInstallStorePackageUpdatesAsync($list)) 'IAsyncOperationWithProgress` + "`" + `2' @([Windows.Services.Store.StorePackageUpdateResult], [Windows.Services.Store.StorePackageUpdateStatus])
Write-Output ("STATE|{0}" -f $result.OverallState)
`

// runScript 는 숨긴 PowerShell 로 스크립트를 실행하고 표준 출력 줄을 돌려준다.
func runScript(szScript string, durTimeout time.Duration) ([]string, error) {
	ctxRun, fnCancel := context.WithTimeout(context.Background(), durTimeout)
	defer fnCancel()
	pCmd := exec.CommandContext(ctxRun, "powershell.exe", "-NoProfile", "-NonInteractive", "-STA", "-Command", szScript)
	procutil.HideConsole(pCmd)
	bOut, errRun := pCmd.Output()
	if errRun != nil {
		return nil, fmt.Errorf("storeupdate: %w", errRun)
	}
	var slLines []string
	for _, szLine := range strings.Split(strings.ReplaceAll(string(bOut), "\r\n", "\n"), "\n") {
		if strings.TrimSpace(szLine) != "" {
			slLines = append(slLines, szLine)
		}
	}
	return slLines, nil
}

// Check 는 스토어에 새 버전이 있는지 묻는다. 없으면 (nil, nil) 이다.
func Check() (*Update, error) {
	if !IsStoreInstall() {
		return nil, ErrNotStore
	}
	slLines, errRun := runScript(g_cszCheckScript, G_cdurCheckTimeout)
	if errRun != nil {
		return nil, errRun
	}
	return parseCheckOutput(slLines)
}

// Install 은 Windows 의 업데이트 확인 창을 띄워 내려받기·설치를 요청하고, 끝난 상태를 돌려준다
// ("Completed", "Canceled", "NoUpdate", 그 밖에 Windows 가 알려 준 오류 상태 이름).
// 설치가 진행되면 Windows 가 앱을 닫을 수 있으므로 이 함수는 돌아오지 않을 수도 있다.
func Install() (string, error) {
	if !IsStoreInstall() {
		return "", ErrNotStore
	}
	szScript := strings.Replace(g_cszInstallScript, "__HWND__", fmt.Sprintf("%d", mainWindow()), 1)
	slLines, errRun := runScript(szScript, G_cdurInstallTimeout)
	if errRun != nil {
		return "", errRun
	}
	return parseInstallOutput(slLines)
}

// mainWindow 는 이 프로세스의 보이는 최상위 창 핸들을 찾는다 (주인 없는 창 = 앱의 주 창). 못 찾으면 0.
func mainWindow() uintptr {
	rPid, _, _ := g_procCurrentPid.Call()
	var hFound uintptr
	fnEach := syscall.NewCallback(func(hWnd uintptr, lParam uintptr) uintptr {
		var uiPid uint32
		_, _, _ = g_procWindowThreadPid.Call(hWnd, uintptr(unsafe.Pointer(&uiPid)))
		if uintptr(uiPid) != rPid {
			return 1
		}
		if rVisible, _, _ := g_procIsWindowVisible.Call(hWnd); rVisible == 0 {
			return 1
		}
		if rOwner, _, _ := g_procGetWindow.Call(hWnd, 4); rOwner != 0 { // GW_OWNER
			return 1
		}
		hFound = hWnd
		return 0 // 찾았다: 그만 돈다
	})
	_, _, _ = g_procEnumWindows.Call(fnEach, 0)
	return hFound
}
```

### `internal/update/update.go` — 수정

```diff
--- a/internal/update/update.go
+++ b/internal/update/update.go
@@ -31,6 +31,8 @@
 	M_szStatus  string `json:"status"`
 	M_szVersion string `json:"version,omitempty"`
 	M_szUrl     string `json:"url,omitempty"`
+	// M_bStore 는 Microsoft Store 에서 찾은 새 버전이라는 뜻이다 (설치는 update.install 로 요청한다, D115).
+	M_bStore bool `json:"store,omitempty"`
 }
 
 // ParseVersion 은 태그 문자열을 숫자 세 개(major, minor, patch)로 바꾼다.
```


## C. 테스트 코드

### `internal/app/update_store_test.go` — 새 파일 (전체)

```go
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
```

### `internal/storeupdate/storeupdate_test.go` — 새 파일 (전체)

```go
package storeupdate

import (
	"errors"
	"testing"
)

// 확인 스크립트 출력 해석: 이 앱의 업데이트를 고르고, 네 자리 버전의 끝 ".0" 을 뗀다.
func TestParseCheckOutput(t *testing.T) {
	pUpdate, errParse := parseCheckOutput([]string{
		"UPDATE|Some.OptionalPackage|9.9.9.0|False",
		"UPDATE|YangHunLee.DariTerm|0.2.1.0|True",
		"COUNT|2",
	})
	if errParse != nil || pUpdate == nil || pUpdate.Version != "0.2.1" || !pUpdate.Mandatory {
		t.Fatalf("업데이트 해석: %+v %v", pUpdate, errParse)
	}
	pNone, errNone := parseCheckOutput([]string{"COUNT|0"})
	if errNone != nil || pNone != nil {
		t.Fatalf("업데이트 없음: %+v %v", pNone, errNone)
	}
	pOther, errOther := parseCheckOutput([]string{"UPDATE|Other.App|1.0.0.0|False", "COUNT|1"})
	if errOther != nil || pOther != nil {
		t.Fatalf("다른 패키지의 업데이트는 이 앱의 것이 아님: %+v %v", pOther, errOther)
	}
	// 대소문자가 달라도 같은 패키지로 본다. 네 번째 자리가 0 이 아니면 그대로 둔다.
	pCase, _ := parseCheckOutput([]string{"UPDATE|yanghunlee.dariterm|1.2.3.4|false", "COUNT|1"})
	if pCase == nil || pCase.Version != "1.2.3.4" || pCase.Mandatory {
		t.Fatalf("버전 표기: %+v", pCase)
	}
}

// 스크립트가 끝까지 돌지 못했으면(COUNT 줄 없음) "업데이트 없음"으로 보지 않고 오류다.
func TestParseCheckOutputIncompleteIsError(t *testing.T) {
	for _, slLines := range [][]string{nil, {"some error text"}, {"UPDATE|YangHunLee.DariTerm|0.2.0.0|False"}} {
		if _, errParse := parseCheckOutput(slLines); errParse == nil {
			t.Fatalf("오류여야 함: %v", slLines)
		}
	}
}

func TestParseInstallOutput(t *testing.T) {
	if szState, errParse := parseInstallOutput([]string{"noise", " STATE|Completed "}); errParse != nil || szState != G_cszStateCompleted {
		t.Fatalf("완료 상태: %q %v", szState, errParse)
	}
	if szState, _ := parseInstallOutput([]string{"STATE|Canceled"}); szState != G_cszStateCanceled {
		t.Fatalf("취소 상태: %q", szState)
	}
	if _, errParse := parseInstallOutput([]string{"STATE|", "x"}); errParse == nil {
		t.Fatal("상태 줄이 없으면 오류여야 함")
	}
}

// 시험은 스토어 설치본이 아닌 곳에서 돈다: 스토어 확인을 하지 않고 ErrNotStore 다.
func TestNotStoreInstallDoesNothing(t *testing.T) {
	if IsStoreInstall() {
		t.Skip("패키지된 환경에서 실행 중")
	}
	if _, errCheck := Check(); !errors.Is(errCheck, ErrNotStore) {
		t.Fatalf("Check: %v", errCheck)
	}
	if _, errInstall := Install(); !errors.Is(errInstall, ErrNotStore) {
		t.Fatalf("Install: %v", errInstall)
	}
}
```


## D. 계약 고정 파일 (백엔드 응답과 프런트 타입을 맞춰 보는 테스트가 읽는다)

### `frontend/src/ipc/contract/app.info.json` — 수정

```diff
--- a/frontend/src/ipc/contract/app.info.json
+++ b/frontend/src/ipc/contract/app.info.json
@@ -4,6 +4,7 @@
   "osLanguage": "\u003clang\u003e",
   "platform": "\u003cos\u003e",
   "updateEnabled": false,
+  "updateSource": "",
   "version": "0.1.0"
 }
 
```


## E. 문구

`locales/<언어>.json` 10개 파일에 같은 키를 넣는다(순서: ko, en, ja, zh, tw, de, fr, es, pt, ru). 한국어·영어 외에는 이 작업을 한 세션이 번역한 것이라 검수가 필요할 수 있다.

**`update.storeTitle`** (추가)

```json
  "update.storeTitle": "Microsoft Store 업데이트 확인 창 열기",
  "update.storeTitle": "Open the Microsoft Store update prompt",
  "update.storeTitle": "Microsoft Store の更新確認ウィンドウを開く",
  "update.storeTitle": "打开 Microsoft Store 更新确认窗口",
  "update.storeTitle": "開啟 Microsoft Store 更新確認視窗",
  "update.storeTitle": "Microsoft Store-Aktualisierungsdialog öffnen",
  "update.storeTitle": "Ouvrir la fenêtre de mise à jour du Microsoft Store",
  "update.storeTitle": "Abrir el aviso de actualización de Microsoft Store",
  "update.storeTitle": "Abrir o aviso de atualização da Microsoft Store",
  "update.storeTitle": "Открыть окно обновления Microsoft Store",
```

**`about.updateCheckStoreDesc`** (추가)

```json
  "about.updateCheckStoreDesc": "시작할 때와 12시간마다 Microsoft Store에 새 버전이 있는지 확인합니다. 새 버전이 있으면 상태 표시줄의 버튼으로 업데이트를 확인할 수 있습니다. 서버 목록·IP·계정 정보는 보내지 않습니다.",
  "about.updateCheckStoreDesc": "Checks the Microsoft Store for a new version at startup and every 12 hours. When one is available, use the button in the status bar to confirm the update. Your server list, IP addresses and account details are never sent.",
  "about.updateCheckStoreDesc": "起動時と 12 時間ごとに、Microsoft Store に新しいバージョンがあるか確認します。新しいバージョンがあると、ステータスバーのボタンから更新を確認できます。サーバー一覧・IP・アカウント情報は送信しません。",
  "about.updateCheckStoreDesc": "启动时以及每 12 小时检查 Microsoft Store 是否有新版本。有新版本时，可通过状态栏的按钮确认更新。不会发送服务器列表、IP 或账户信息。",
  "about.updateCheckStoreDesc": "啟動時以及每 12 小時檢查 Microsoft Store 是否有新版本。有新版本時，可透過狀態列的按鈕確認更新。不會傳送伺服器清單、IP 或帳戶資訊。",
  "about.updateCheckStoreDesc": "Prüft beim Start und alle 12 Stunden im Microsoft Store, ob eine neue Version vorliegt. Ist eine verfügbar, bestätigen Sie die Aktualisierung über die Schaltfläche in der Statusleiste. Serverliste, IP-Adressen und Kontodaten werden nie gesendet.",
  "about.updateCheckStoreDesc": "Vérifie dans le Microsoft Store la présence d'une nouvelle version au démarrage et toutes les 12 heures. Lorsqu'une version est disponible, confirmez la mise à jour avec le bouton de la barre d'état. La liste des serveurs, les adresses IP et les informations de compte ne sont jamais envoyées.",
  "about.updateCheckStoreDesc": "Comprueba en Microsoft Store si hay una versión nueva al iniciar y cada 12 horas. Cuando la haya, confirme la actualización con el botón de la barra de estado. Nunca se envían la lista de servidores, las IP ni los datos de cuenta.",
  "about.updateCheckStoreDesc": "Verifica na Microsoft Store se há uma nova versão ao iniciar e a cada 12 horas. Quando houver, confirme a atualização pelo botão da barra de status. A lista de servidores, os IPs e os dados de conta nunca são enviados.",
  "about.updateCheckStoreDesc": "При запуске и каждые 12 часов проверяет в Microsoft Store наличие новой версии. Если она есть, подтвердите обновление кнопкой в строке состояния. Список серверов, IP-адреса и учётные данные не отправляются.",
```

**`be.update.storeDone`** (추가)

```json
  "be.update.storeDone": "업데이트를 설치했습니다. 앱을 다시 시작하면 새 버전으로 실행됩니다",
  "be.update.storeDone": "The update was installed. Restart the app to run the new version",
  "be.update.storeDone": "更新をインストールしました。アプリを再起動すると新しいバージョンで起動します",
  "be.update.storeDone": "更新已安装。重新启动应用后将运行新版本",
  "be.update.storeDone": "更新已安裝。重新啟動應用程式後將執行新版本",
  "be.update.storeDone": "Die Aktualisierung wurde installiert. Starten Sie die App neu, um die neue Version zu verwenden",
  "be.update.storeDone": "La mise à jour a été installée. Redémarrez l'application pour utiliser la nouvelle version",
  "be.update.storeDone": "La actualización se instaló. Reinicie la aplicación para usar la nueva versión",
  "be.update.storeDone": "A atualização foi instalada. Reinicie o app para usar a nova versão",
  "be.update.storeDone": "Обновление установлено. Перезапустите приложение, чтобы использовать новую версию",
```

**`be.update.storeCanceled`** (추가)

```json
  "be.update.storeCanceled": "업데이트를 취소했습니다",
  "be.update.storeCanceled": "The update was cancelled",
  "be.update.storeCanceled": "更新をキャンセルしました",
  "be.update.storeCanceled": "已取消更新",
  "be.update.storeCanceled": "已取消更新",
  "be.update.storeCanceled": "Die Aktualisierung wurde abgebrochen",
  "be.update.storeCanceled": "La mise à jour a été annulée",
  "be.update.storeCanceled": "La actualización se canceló",
  "be.update.storeCanceled": "A atualização foi cancelada",
  "be.update.storeCanceled": "Обновление отменено",
```

**`be.update.storeFailed`** (추가)

```json
  "be.update.storeFailed": "업데이트 확인 창을 열지 못해 Microsoft Store를 엽니다",
  "be.update.storeFailed": "Could not open the update prompt; opening the Microsoft Store instead",
  "be.update.storeFailed": "更新確認ウィンドウを開けなかったため、Microsoft Store を開きます",
  "be.update.storeFailed": "无法打开更新确认窗口，将改为打开 Microsoft Store",
  "be.update.storeFailed": "無法開啟更新確認視窗，將改為開啟 Microsoft Store",
  "be.update.storeFailed": "Der Aktualisierungsdialog konnte nicht geöffnet werden; stattdessen wird der Microsoft Store geöffnet",
  "be.update.storeFailed": "Impossible d'ouvrir la fenêtre de mise à jour ; ouverture du Microsoft Store à la place",
  "be.update.storeFailed": "No se pudo abrir el aviso de actualización; se abre Microsoft Store en su lugar",
  "be.update.storeFailed": "Não foi possível abrir o aviso de atualização; abrindo a Microsoft Store",
  "be.update.storeFailed": "Не удалось открыть окно обновления; вместо этого открывается Microsoft Store",
```

**`be.update.storeState`** (추가)

```json
  "be.update.storeState": "업데이트를 설치하지 못했습니다 ({state}). Microsoft Store를 엽니다",
  "be.update.storeState": "The update could not be installed ({state}). Opening the Microsoft Store",
  "be.update.storeState": "更新をインストールできませんでした（{state}）。Microsoft Store を開きます",
  "be.update.storeState": "无法安装更新（{state}）。正在打开 Microsoft Store",
  "be.update.storeState": "無法安裝更新（{state}）。正在開啟 Microsoft Store",
  "be.update.storeState": "Die Aktualisierung konnte nicht installiert werden ({state}). Der Microsoft Store wird geöffnet",
  "be.update.storeState": "La mise à jour n'a pas pu être installée ({state}). Ouverture du Microsoft Store",
  "be.update.storeState": "No se pudo instalar la actualización ({state}). Se abre Microsoft Store",
  "be.update.storeState": "Não foi possível instalar a atualização ({state}). Abrindo a Microsoft Store",
  "be.update.storeState": "Не удалось установить обновление ({state}). Открывается Microsoft Store",
```


## F. 명세

### `spec/behavior/lifecycle.md` — 수정

```diff
--- a/spec/behavior/lifecycle.md
+++ b/spec/behavior/lifecycle.md
@@ -50,6 +50,17 @@
 - **버전 비교**: 태그 앞의 `v` 와 `-`/`+` 뒤 접미를 떼고 숫자 3개(major.minor.patch, 부족하면 0)로 비교한다. 최신 > 현재면 `newer`(버전은 `v` 뗀 문자열, 릴리스 페이지 URL 포함), 아니면 `latest`.
 - `newer` 이면 상태 표시줄의 "새 버전 {버전}" 버튼을 위해 기억하고 "새 버전 발견" 이벤트를 보낸다. **처음 발견한 버전이거나 수동 확인이면 안내 토스트도 띄운다**: `새 버전 {버전} 이 있습니다 (상태 표시줄을 눌러 확인)`. 마지막으로 안내한 버전은 데이터 폴더의 `update.json`(`{notified: 버전}`)에 기록해 같은 버전은 다시 안내하지 않는다.
 
+### 6.1 Microsoft Store 설치본의 새 버전 확인 (2026-11-23, D115)
+
+- **Windows 에서 Microsoft Store 로 설치한 실행**(패키지된 앱)은 새 버전을 배포 저장소가 아니라 **Microsoft Store 에 묻는다**. 설치 파일로 설치했거나 빌드 폴더에서 바로 실행한 경우, Windows 가 아닌 OS 는 해당하지 않는다(위 §6 의 규칙 그대로 — 배포 저장소가 없으면 꺼짐).
+- 스토어 설치본에서는 설정 > 정보의 업데이트 항목이 보인다. 토글 설명은 "시작할 때와 12시간마다 Microsoft Store에 새 버전이 있는지 확인합니다. …" 로 바뀐다. 주기(시작 5초 뒤, 12시간마다)·자동 확인 끄기·[업데이트 확인]·처음 발견한 버전만 알림은 §6 과 같다.
+- 확인: 스토어에 이 앱의 업데이트 목록을 묻는다(시한 30초). 이 앱(패키지 이름 `YangHunLee.DariTerm`)의 업데이트가 있으면 `newer`(버전은 스토어의 네 자리에서 끝의 `.0` 을 뗀 것, 주소는 스토어 제품 페이지 `ms-windows-store://pdp/?productid=9N8WBPMRNBWV`, `store: true`), 없으면 `latest`, 스토어가 답하지 않으면 `error`. 스토어는 설치된 것보다 새 버전만 알려 주므로 앱이 버전을 비교하지 않는다.
+- **업데이트 확인(설치)**: 상태 표시줄의 "새 버전 {버전}" 버튼(도움말 "Microsoft Store 업데이트 확인 창 열기")을 누르면 Windows 의 업데이트 확인 창이 앱 창 위에 뜬다. 사용자가 확인하면 Windows 가 내려받아 설치하며, 그 과정에서 앱이 닫힐 수 있다(시한 60분). 창이 떠 있는 동안 버튼을 다시 눌러도 겹쳐 띄우지 않는다.
+  - 끝난 뒤 토스트: 설치됨 "업데이트를 설치했습니다. 앱을 다시 시작하면 새 버전으로 실행됩니다"(ok) / 사용자가 취소 "업데이트를 취소했습니다"(info) / 그사이 업데이트가 없어짐 "최신 버전입니다"(ok, "새 버전" 버튼도 지운다).
+  - 확인 창을 띄우지 못했거나 Windows 가 오류 상태를 알려 주면(Wi-Fi 필요, 배터리 부족 등) 오류 토스트를 내고 **Microsoft Store 의 제품 페이지를 대신 연다**(거기서 직접 업데이트할 수 있다).
+- 앱이 스토어에 보내는 것은 Windows 의 스토어 기능이 보내는 것뿐이다(앱 식별 정보). 서버 목록·IP·계정 정보는 보내지 않는다.
+- 열려 있는 세션이 있을 때 설치를 확인하면 Windows 가 앱을 닫으면서 세션이 끊긴다. 확인 창에서 취소하면 아무 일도 일어나지 않는다.
+
 ## 7. 제품 이름·기본값에서 기존 앱과 다른 점 (clean-room 선택)
 - 업로드 파일 접두 `dt-`, 백업 확장자 `.dtbackup`, IPC 채널 이름, 로고는 새로 정한다.
 - 데이터 폴더 이름 `DariTerm`, 환경 변수 `DARITERM_*`, 업로드 기본 폴더 `~/.dariterm/uploads` 는 이 제품의 이름이다.
```

### `spec/ipc.md` — 수정

```diff
--- a/spec/ipc.md
+++ b/spec/ipc.md
@@ -116,7 +116,8 @@
 | `sync.run` | 모드(`""` / `pull-vault` / `overwrite`) | 상태 `{state, message?, lastSync?, changed?}` |
 | `sync.status` | | `{configured, gitAvailable, last}` — **`last` 는 항상 객체**이며 동기화한 적이 없으면 `{"state":""}`(나머지 필드 생략). null 이 아니다 |
 | `update.check` | | `{status: newer\|latest\|error\|unconfigured, version?, url?}` (수동 확인 — 새 버전이면 안내 이벤트에 "알림 필요" 표시) |
-| `update.status` | | 마지막으로 발견한 새 버전 `{status, version, url}` 또는 null |
+| `update.status` | | 마지막으로 발견한 새 버전 `{status, version, url, store?}` 또는 null |
+| `update.install` | | `{started}` — Microsoft Store 설치본이면 Windows 의 업데이트 확인 창을 띄우고 곧바로 `started: true`(결과는 `toast` 이벤트). 그 밖의 실행은 아무 일도 하지 않고 `started: false` (lifecycle.md §6.1, D115) |
 
 ## 3. 인증·확인 요청 (백엔드가 묻는 방식)
 백엔드가 `ui.request` 이벤트로 질문을 보내고 프런트가 `Reply(requestId, 값JSON)` 로 답한다. **요청 객체의 id 필드 이름은 `requestId`** 이며 `kind` 와 아래 표의 필드를 함께 가진다. **응답이 `null` 이거나 10분 안에 없으면 "취소"** 로 본다. 종류와 응답 값:
@@ -140,5 +141,5 @@
 | `store.changed` | | 백엔드가 저장 데이터를 바꿨을 때(가져오기·동기화·비밀번호 저장 등) — 프런트가 다시 읽는다 |
 | `toast` | `{kind: ok\|error\|info, text}` | 백엔드가 띄우는 알림 |
 | `sync.status` | 동기화 상태 — **`sync.run` 의 결과와 같은 형태** `{state, message?, lastSync?, changed?}` | 동기화 종료마다 |
-| `update.available` | `{version, url, announce}` | 새 버전 발견(announce 는 처음 발견했거나 수동 확인일 때 true) |
+| `update.available` | `{version, url, announce, store?}` | 새 버전 발견(announce 는 처음 발견했거나 수동 확인일 때 true. store 는 Microsoft Store 에서 찾았을 때만 true 로 붙는다) |
 
```



---

## G. 검증 (2026-11-23 실행)

### G.1 자동 테스트

| 테스트 | 확인하는 것 |
|---|---|
| [storeupdate_test.go](../../internal/storeupdate/storeupdate_test.go) `TestParseCheckOutput` | 이 앱의 업데이트를 고른다(다른 패키지 것은 버림, 대소문자 무시) / 끝의 `.0` 을 뗀다 / 없으면 nil |
| `TestParseCheckOutputIncompleteIsError` | `COUNT` 줄이 없으면 오류(업데이트 없음으로 보지 않는다) |
| `TestParseInstallOutput` | 상태 줄 찾기 |
| `TestNotStoreInstallDoesNothing` | 스토어 설치본이 아니면 스토어를 부르지 않고 `ErrNotStore` |
| [update_store_test.go](../../internal/app/update_store_test.go) `TestUpdateOffWhenNotStoreAndNoRepo` | 스토어 설치본이 아니고 저장소도 없으면 꺼져 있고 `update.install` 은 `started: false` |
| `TestStoreUpdateCheckNewer` | 스토어 설치본이면 켜진다(`updateSource: "store"`) / 새 버전이면 `newer`·`store: true`·스토어 주소 / 이벤트에 `store` / `update.status` 로 다시 읽힌다 |
| `TestStoreUpdateCheckLatestAndError` | 없으면 `latest`, 스토어가 답하지 않으면 `error`, 둘 다 이벤트 없음 |
| `TestStoreUpdateInstallOutcomes` | 세 번 눌러도 확인 창은 한 번 / 완료·취소·업데이트 없음은 스토어 페이지를 열지 않는다 / 오류 상태·창 실패는 스토어 페이지를 연다 / "업데이트 없음"이면 새 버전 표시를 지운다 |
| 계약 테스트 | `app.info` 의 `updateSource` |

```bash
go test ./internal/storeupdate/ ./internal/app/ ./internal/update/    # 통과
GOOS=linux go build ./internal/storeupdate/                           # 통과
(cd frontend && npx tsc --noEmit && npx vitest run)                   # 통과
```

### G.2 실제 스토어 패키지 신원으로 실행 (버전 확인만)

```powershell
# 설치된 패키지 확인
Get-AppxPackage -Name 'YangHunLee.DariTerm'      # YangHunLee.DariTerm_1.0.0.0_x64__bmtfree9a9kzr, SignatureKind=Store

# 확인 스크립트를 패키지 신원으로 실행 (결과를 파일로)
Invoke-CommandInDesktopPackage -PackageFamilyName 'YangHunLee.DariTerm_bmtfree9a9kzr' -AppId 'DariTerm' `
  -Command 'powershell.exe' -Args '-NoProfile -NonInteractive -STA -ExecutionPolicy Bypass -WindowStyle Hidden -Command "& ''<check.ps1>'' *> ''<out.txt>''"'
```

결과: `COUNT|0` — 스토어가 답했고, 1.0.0 이 최신이라 업데이트가 없다. 같은 스크립트를 패키지 신원 없이 돌리면 스토어 호출에서 예외로 끝난다(앱은 이것을 `error` 로 처리한다).

## H. 수동 검증 (남은 일)

| # | 절차 | 기대 결과 |
|---|---|---|
| 1 | 이 빌드를 MSIX 로 만들어(버전 1.0.1 이상) 스토어에 올리고, 스토어에서 설치한 앱을 실행 → 설정 > 정보 | "업데이트 자동 확인" 토글과 [업데이트 확인] 이 보인다. 설명이 Microsoft Store 문구다 |
| 2 | [업데이트 확인] | 스토어에 더 새 버전이 없으면 "최신 버전입니다" |
| 3 | 그 뒤 더 새 버전을 스토어에 올린다(인증 통과 뒤) → 앱에서 [업데이트 확인] | "새 버전 {버전} 이 있습니다", 상태 표시줄에 "새 버전 {버전}" 버튼 |
| 4 | 그 버튼을 누른다 | Windows 의 업데이트 확인 창이 앱 창 위에 뜬다 |
| 5 | 확인 창에서 취소 | "업데이트를 취소했습니다". 앱은 그대로 |
| 6 | 확인 창에서 확인 | 내려받아 설치된다. 앱이 닫히면 다시 실행했을 때 새 버전이다 |
| 7 | 설치 파일(NSIS)로 설치한 실행, 개발 빌드 | 업데이트 항목이 보이지 않는다(전과 같음) |
| 8 | "업데이트 자동 확인"을 끄고 12시간 넘게 둔다 | 자동으로 묻지 않는다. [업데이트 확인] 은 된다 |

4~6번이 안 되면(확인 창이 뜨지 않으면) 오류 토스트와 함께 스토어 제품 페이지가 열려야 한다. 그것도 수동 검증 대상이다.

## I. 영향 범위·회귀 위험

| 영역 | 위험 | 대비 |
|---|---|---|
| 스토어 설치본이 아닌 실행 | `updateSource()` 가 전과 같은 값(저장소가 있으면 release, 없으면 빈 값)을 돌려준다 | `TestUpdateOffWhenNotStoreAndNoRepo`, 기존 계약 테스트 |
| 스토어 설치본 | **전에는 없던 업데이트 항목이 설정에 생기고**, 시작 5초 뒤와 12시간마다 PowerShell 을 한 번 띄운다(수 초, 창 없음) | "업데이트 자동 확인"을 끄면 주기 확인을 하지 않는다 |
| 스크립트 실패 | `error` 로 처리, 자동 확인에서는 조용히 넘어간다 | `TestStoreUpdateCheckLatestAndError` |
| 설치 중 열린 세션 | 확인하면 Windows 가 앱을 닫아 세션이 끊긴다 | 확인 창이 한 번 묻는다. 앱이 따로 경고하지는 않는다 — 정리 거리 |
| 개인정보 | 스토어에 가는 것은 Windows 의 스토어 기능이 보내는 것뿐이다 | 설정 설명과 명세에 명시. 개인정보 처리방침([../store/privacy-policy.md](../store/privacy-policy.md))에 스토어 확인을 적을지는 검토 거리 |
| PowerShell 을 막은 PC | 스크립트가 실행되지 않아 `error` | 회사 정책으로 PowerShell 이 막힌 곳에서는 이 기능을 쓸 수 없다(파일 대화상자도 같은 사정) |

## J. 체크리스트

- [x] `internal/storeupdate/` 새 패키지(공통·Windows·그 밖)
- [x] `channels_update.go`: `updateSource`, `storeCheckResult`, `runStoreInstall`, 채널 `update.install`, 이벤트에 `store`
- [x] `update.go`: 결과에 `M_bStore`. `app.go`: `app.info` 에 `updateSource`
- [x] 프런트: 타입, `update.ts`, `UpdateBadge.tsx`, `SettingsAbout.tsx`, 가짜 백엔드, 계약 파일
- [x] 문구 6개 추가(10개 언어), 명세 behavior/lifecycle.md §6.1·ipc.md, decisions D115
- [x] 자동 테스트 8개 통과, 버전 확인 스크립트를 실제 패키지 신원으로 실행
- [ ] 수동 검증 1~8 (스토어에 올린 뒤)
