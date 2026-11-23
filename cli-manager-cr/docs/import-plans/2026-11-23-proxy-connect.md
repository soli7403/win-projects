# 프록시 경유 접속 (HTTP CONNECT · SOCKS5) — 배경·설계·코드·검증

**작성**: 2026-11-23 · **상태**: 코드·자동 테스트 완료(D110). 실제 사내 프록시와의 확인은 수동 검증으로 남음 · **대상**: cli-manager-cr(DariTerm)
**묶음**: 빠진 기능 10건 중 5번. 앞 문서 [socks-dynamic-forward](./2026-11-23-socks-dynamic-forward.md), 뒤 문서 [agent-forwarding](./2026-11-23-agent-forwarding.md).

---

## 0. 한 줄 요약

서버에 바로 TCP 연결을 맺을 수 없는 환경(회사 방화벽 뒤)에서, **HTTP 프록시(CONNECT) 또는 SOCKS5 프록시를 거쳐** SSH 접속을 맺을 수 있게 했다. 서버 편집 > 고급에 "프록시 경유" 항목이 생겼다. 전에는 경유 수단이 SSH 경유 서버(ProxyJump)뿐이었다.

## 1. 배경

- [internal/sshconn/client.go](../../internal/sshconn/client.go) `Connect` 는 경유 서버가 없으면 `net.DialTimeout` 으로 대상에 바로 연결했다. 다른 길이 없었다.
- 밖으로 나가는 연결을 프록시로만 허용하는 회사 네트워크에서는 접속 자체가 불가능했다.

## 2. 설계 결정

| 항목 | 정한 것 | 이유 |
|---|---|---|
| 저장 | 서버에 `proxy: {type, host, port, username, credentialId}` | 서버마다 다르다(사내 서버는 바로, 외부 서버는 프록시로) |
| 종류 | `http`(CONNECT), `socks5` | 회사 프록시의 대부분. SOCKS4·ProxyCommand·NTLM/Kerberos 인증은 하지 않았다(§2.1) |
| 적용 자리 | **이 PC 가 직접 TCP 연결을 맺는 대상**에만. 경유 서버가 있으면 그 서버의 `proxy` 는 쓰지 않고, 경유 서버 체인의 맨 바깥 서버에 `proxy` 가 있으면 그 연결이 프록시를 거친다 | 경유 서버 뒤의 연결은 SSH 채널 위에서 열리므로 프록시가 낄 자리가 없다. OpenSSH 도 ProxyJump 와 ProxyCommand 를 한 호스트에 함께 쓰지 못한다 |
| 다시 연결할 때 | 비밀번호를 물어 새 연결을 맺는 경로(`dialNext`)도 프록시를 거친다 | 같은 함수 `dialDirect` 로 모았다 |
| 이름 풀기 | 대상 이름을 프록시에 그대로 넘긴다 | 이 PC 에서는 외부 DNS 가 안 될 수 있다 |
| 비밀번호 | 볼트의 계정(`{별칭} 프록시`)에 암호화해 저장, `proxy.credentialId` 로 연결 | 경유 서버 직접 입력과 같은 방식. store.json 에 비밀을 두지 않는다 |
| 볼트가 잠긴 동안 | 접속에 성공했던 프록시 비밀번호를 세션 사본(`sessionCreds.mapProxy`)에 두었다가 재접속에 쓴다. 사본도 없으면 잠금 해제를 기다린다 | 대상 계정과 같은 규칙(protocols.md §2.2 세션 사본) |
| 인증 실패 | `ErrProxyAuth` → 안내 문구 "프록시가 사용자 이름 또는 비밀번호를 받아들이지 않았습니다" | HTTP 407, SOCKS5 의 0xFF·인증 실패를 한 가지로 |
| 그 밖의 실패 | `proxy 주소: 원인` (HTTP 는 상태 줄, SOCKS5 는 응답 코드) | 프록시 문제인지 대상 문제인지 구분된다 |
| 시한 | 프록시 연결 + 터널 요청 합쳐 20초 | 기존 접속 시한과 같다 |
| 미리 온 데이터 | HTTP 응답을 `bufio.Reader` 로 읽으므로, 응답과 함께 도착한 SSH 서버의 첫 인사를 잃지 않도록 연결의 읽기를 그 reader 로 돌린다(`bufferedConn`) | 실제로 일어난다(프록시가 200 과 서버 인사를 한 묶음으로 보냄) |
| 기본 포트 | http 8080, socks5 1080 (포트 0 으로 저장됐을 때) | 관례 |

### 2.1 하지 않은 것

- SOCKS4 프록시, 프록시 명령(ProxyCommand), HTTP 프록시의 NTLM·Kerberos(Negotiate) 인증, 시스템 프록시 설정 자동 따르기, 텔넷의 프록시 경유.
- 직접 입력한 경유 서버(베스천)에 프록시 두기. 프록시가 필요하면 그 서버를 저장한 뒤 경유 서버로 고른다.
- 가져오기(PuTTY·SecureCRT·ssh config)에서 프록시 설정 읽기.

## A. 바뀐 파일 한눈에

| 파일 | 종류 | 상태 |
|---|---|---|
| `frontend/src/ipc/types.ts` | 코드 | 수정 |
| `frontend/src/lib/hostForm.ts` | 코드 | 수정 |
| `frontend/src/lib/hostSave.ts` | 코드 | 수정 |
| `frontend/src/screens/HostEditor.tsx` | 코드 | 수정 |
| `frontend/src/screens/HostEditorParts.tsx` | 코드 | 수정 |
| `internal/app/channels_session.go` | 코드 | 수정 |
| `internal/app/ssh_app.go` | 코드 | 수정 |
| `internal/app/ssh_credcache.go` | 코드 | 수정 |
| `internal/sshconn/client.go` | 코드 | 수정 |
| `internal/sshconn/proxy.go` | 코드 | 새 파일 |
| `internal/sshconn/types.go` | 코드 | 수정 |
| `frontend/src/lib/hostForm.test.ts` | 테스트 | 수정 |
| `internal/sshconn/proxy_test.go` | 테스트 | 새 파일 |
| `spec/data/store.md` | 명세 | 수정 |
| `spec/protocols.md` | 명세 | 수정 |
| `spec/ui/host-editor.md` | 명세 | 수정 |
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

아래의 `diff` 는 **이 묶음의 앞 문서들을 차례로 적용한 상태**를 기준으로 뽑았다. 앞 문서에서도 고친 파일(문맥 줄에 앞 기능의 코드가 섞여 있을 수 있다): `frontend/src/ipc/types.ts`, `frontend/src/lib/hostForm.test.ts`, `frontend/src/lib/hostForm.ts`, `frontend/src/screens/HostEditor.tsx`, `frontend/src/screens/HostEditorParts.tsx`, `spec/data/store.md`, `spec/protocols.md`, `spec/ui/host-editor.md`. 이 기능만 따로 넣을 때는 `+`/`-` 줄만 옮기면 된다.

## B. 코드

### `frontend/src/ipc/types.ts` — 수정

```diff
--- a/frontend/src/ipc/types.ts
+++ b/frontend/src/ipc/types.ts
@@ -37,6 +37,8 @@
   credentialId?: string | null;
   jumpHostId?: string | null;
   jumpDirect?: unknown;
+  /** 프록시 경유 설정(store.md §3 proxy). 없으면 바로 연결. */
+  proxy?: unknown;
   color?: string;
   env?: EnvKind;
   tags: string[];
```

### `frontend/src/lib/hostForm.ts` — 수정

```diff
--- a/frontend/src/lib/hostForm.ts
+++ b/frontend/src/lib/hostForm.ts
@@ -40,6 +40,20 @@
   authType: 'password' | 'key';
   credentialId?: string | null;
 }
+
+export type ProxyType = 'http' | 'socks5';
+
+/** 프록시 경유 설정(store.md §3 proxy). */
+export interface ProxyConf {
+  type: ProxyType;
+  host: string;
+  port: number;
+  username: string;
+  credentialId?: string | null;
+}
+
+/** 프록시 종류별 기본 포트. */
+export const PROXY_DEFAULT_PORT: Record<ProxyType, number> = { http: 8080, socks5: 1080 };
 
 /** 직접 입력 베스천을 가리키는 jumpHostId 값. */
 export const JUMP_DIRECT = 'direct';
@@ -104,6 +118,10 @@
   jumpDirect: boolean;
   jumpAddress: string;
   jumpHasSecret: boolean;
+  /** 프록시 경유를 켰는지, 그 주소, 새로 넣은 비밀번호가 있는지 (없으면 검사하지 않는다). */
+  proxyOn?: boolean;
+  proxyHost?: string;
+  proxyHasSecret?: boolean;
 }): { tab: 'basic' | 'auth' | 'adv'; message: string } | null {
   const isNet = input.protocol === 'ssh' || input.protocol === 'telnet';
   if (isNet && !input.host.trim()) return { tab: 'basic', message: t('hostForm.needHost') };
@@ -116,6 +134,10 @@
     if (input.jumpHasSecret && !input.vaultOpen) {
       return { tab: 'adv', message: t('hostForm.lockedJump') };
     }
+  }
+  if (input.protocol === 'ssh' && input.proxyOn) {
+    if (!(input.proxyHost ?? '').trim()) return { tab: 'adv', message: t('hostForm.needProxy') };
+    if (input.proxyHasSecret && !input.vaultOpen) return { tab: 'adv', message: t('hostForm.lockedProxy') };
   }
   return null;
 }
```

### `frontend/src/lib/hostSave.ts` — 수정

```diff
--- a/frontend/src/lib/hostSave.ts
+++ b/frontend/src/lib/hostSave.ts
@@ -1,7 +1,7 @@
 // 서버 저장 파이프라인: 계정 저장 → 점프 호스트 베스천 저장 → 서버 객체 조립. 근거: spec/ui/host-editor.md §1.5 3~8번.
 import type { AuthType, CredMeta, Host, Protocol } from '../ipc/types';
 import { getBackend } from '../ipc/backend';
-import { JUMP_DIRECT, localOptions, type JumpDirect, type SerialOpts } from './hostForm';
+import { JUMP_DIRECT, localOptions, PROXY_DEFAULT_PORT, type JumpDirect, type ProxyConf, type ProxyType, type SerialOpts } from './hostForm';
 import { t } from '../i18n';
 
 export interface SaveInput {
@@ -38,6 +38,8 @@
   /** 점프 호스트: '' 사용 안 함, 'direct' 직접 입력, 그 밖은 서버 id. */
   jumpChoice: string;
   jump: { host: string; port: number; username: string; authType: JumpDirect['authType']; password: string; keyText: string; passphrase: string };
+  /** 프록시 경유: type 이 '' 이면 사용 안 함. password 는 새로 넣었을 때만 값이 있다. */
+  proxy?: { type: ProxyType | ''; host: string; port: number; username: string; password: string };
 }
 
 /** 계정 저장(§1.5 5번). 기존 계정을 이 서버만 쓰면 갱신, 다른 서버와 공유 중이면 새 항목. */
@@ -113,6 +115,25 @@
     jumpHostId = i.jumpChoice;
   }
 
+  // 2-1) 프록시 경유(SSH 만). 비밀번호를 새로 넣었으면 "{별칭} 프록시" 계정으로 저장한다.
+  //      사용자 이름을 비우면 인증 없는 프록시로 보고 계정 연결을 끊는다.
+  let proxy: ProxyConf | undefined;
+  if (i.protocol === 'ssh' && i.proxy && i.proxy.type && i.proxy.host.trim()) {
+    const prev = i.original?.proxy as ProxyConf | undefined;
+    const user = i.proxy.username.trim();
+    let proxyCred = user ? prev?.credentialId ?? null : null;
+    if (user && i.proxy.password) {
+      proxyCred = await saveAccount('password', t('host.proxyAccountName', { alias: i.alias }), user, { password: i.proxy.password }, proxyCred ?? undefined);
+    }
+    proxy = {
+      type: i.proxy.type,
+      host: i.proxy.host.trim(),
+      port: i.proxy.port || PROXY_DEFAULT_PORT[i.proxy.type],
+      username: user,
+      credentialId: proxyCred,
+    };
+  }
+
   const base: Host = i.original ?? {
     id: crypto.randomUUID(),
     groupId: null,
@@ -142,6 +163,7 @@
     credentialId,
     jumpHostId,
     jumpDirect,
+    proxy,
     groupId: i.folderId || null,
     env: i.env === 'none' ? undefined : i.env,
     color: i.color,
```

### `frontend/src/screens/HostEditor.tsx` — 수정

```diff
--- a/frontend/src/screens/HostEditor.tsx
+++ b/frontend/src/screens/HostEditor.tsx
@@ -11,10 +11,10 @@
 import { ENV_INFO } from '../lib/defaults';
 import {
   fallbackAlias, isWslShell, JUMP_DIRECT, portAfterProtocolChange, serialDefaultsFor,
-  SERIAL_DEFAULTS, splitBastionInput, splitHostInput, tmuxPlaceholder, validateHostForm, type SerialOpts,
+  SERIAL_DEFAULTS, splitBastionInput, splitHostInput, tmuxPlaceholder, validateHostForm, type ProxyConf, type ProxyType, type SerialOpts,
 } from '../lib/hostForm';
 import { applyHost, buildHostFromForm } from '../lib/hostSave';
-import { ForwardsTable, JumpDirectCard, SerialFields } from './HostEditorParts';
+import { ForwardsTable, JumpDirectCard, ProxyCard, SerialFields } from './HostEditorParts';
 import { t } from '../i18n';
 
 type Tab = 'basic' | 'auth' | 'adv' | 'fwd';
@@ -121,6 +121,18 @@
     };
   });
   const jumpHasSaved = !!(original?.jumpDirect as { credentialId?: string | null } | undefined)?.credentialId;
+  // 프록시 경유(§1.3 2-1번)
+  const [proxy, setProxy] = useState(() => {
+    const p = original?.proxy as Partial<ProxyConf> | undefined;
+    return {
+      type: (p?.type === 'http' || p?.type === 'socks5' ? p.type : '') as ProxyType | '',
+      host: p?.host ?? '',
+      port: p?.port ? String(p.port) : '',
+      username: p?.username ?? '',
+      password: '',
+    };
+  });
+  const proxyHasSaved = !!(original?.proxy as ProxyConf | undefined)?.credentialId;
 
   // 포워딩·시리얼·로컬
   const [forwards, setForwards] = useState(original?.forwards ?? []);
@@ -232,6 +244,9 @@
       jumpDirect,
       jumpAddress: jump.address,
       jumpHasSecret: (jump.authType === 'password' && !!jump.password) || (jump.authType === 'key' && !!jump.keyText.trim()),
+      proxyOn: isSsh && !!proxy.type,
+      proxyHost: proxy.host,
+      proxyHasSecret: !!proxy.username.trim() && !!proxy.password,
     });
     if (bad) return fail(bad.message, bad.tab);
 
@@ -278,6 +293,7 @@
           keyText: jump.keyText,
           passphrase: jump.passphrase,
         },
+        proxy: { type: proxy.type, host: proxy.host, port: Number(proxy.port) || 0, username: proxy.username, password: proxy.password },
       });
       await patchStore({ hosts: applyHost(hosts, next, !!hostId) });
       await refreshStore();
@@ -493,6 +509,30 @@
                 keyStatus={jump.keyStatus}
               />
             )}
+            {isSsh && (
+              <Field label={t('host.proxy')} help={t('host.proxyHelp')}>
+                <select className="input" value={proxy.type} onChange={(e) => setProxy((p) => ({ ...p, type: e.target.value as ProxyType | '' }))}>
+                  <option value="">{t('host.none')}</option>
+                  <option value="http">{t('proxy.http')}</option>
+                  <option value="socks5">{t('proxy.socks5')}</option>
+                </select>
+              </Field>
+            )}
+            {isSsh && !!proxy.type && (
+              <ProxyCard
+                type={proxy.type}
+                host={proxy.host}
+                onHost={(v) => setProxy((p) => ({ ...p, host: v }))}
+                port={proxy.port}
+                onPort={(v) => setProxy((p) => ({ ...p, port: v }))}
+                username={proxy.username}
+                onUsername={(v) => setProxy((p) => ({ ...p, username: v }))}
+                password={proxy.password}
+                onPassword={(v) => setProxy((p) => ({ ...p, password: v }))}
+                hasSaved={proxyHasSaved}
+                jumpInUse={!!jumpChoice}
+              />
+            )}
             {tmuxCardShown && (
               <div className="toggle-row">
                 <div>
```

### `frontend/src/screens/HostEditorParts.tsx` — 수정

```diff
--- a/frontend/src/screens/HostEditorParts.tsx
+++ b/frontend/src/screens/HostEditorParts.tsx
@@ -1,7 +1,7 @@
 // 서버 편집의 큰 부분 조각: 포트 포워딩 표, 시리얼 옵션, 점프 호스트 카드. 근거: spec/ui/host-editor.md §1.1(3), §1.3(1), §1.4.
 import { Trash2 } from 'lucide-react';
 import type { Forward } from '../ipc/types';
-import { BAUD_RATES, type JumpDirect, type SerialEnter, type SerialFlow, type SerialOpts, type SerialParity } from '../lib/hostForm';
+import { BAUD_RATES, PROXY_DEFAULT_PORT, type JumpDirect, type ProxyType, type SerialEnter, type SerialFlow, type SerialOpts, type SerialParity } from '../lib/hostForm';
 import { Field, Segmented, Switch } from '../ui/Controls';
 import { t } from '../i18n';
 
@@ -238,3 +238,52 @@
   );
 }
 
+/** 프록시 경유 카드(§1.3 2-1번): 주소·포트·사용자 이름·비밀번호. 사용자 이름을 비우면 인증 없이 연결한다. */
+export function ProxyCard({
+  type,
+  host,
+  onHost,
+  port,
+  onPort,
+  username,
+  onUsername,
+  password,
+  onPassword,
+  hasSaved,
+  jumpInUse,
+}: {
+  type: ProxyType;
+  host: string;
+  onHost: (v: string) => void;
+  port: string;
+  onPort: (v: string) => void;
+  username: string;
+  onUsername: (v: string) => void;
+  password: string;
+  onPassword: (v: string) => void;
+  hasSaved: boolean;
+  jumpInUse: boolean;
+}) {
+  return (
+    <div className="card" style={{ display: 'flex', flexDirection: 'column', gap: 12, padding: 12, border: '1px solid var(--border)', borderRadius: 8 }}>
+      {jumpInUse && <div className="notice info">{t('proxy.jumpNote')}</div>}
+      <div style={{ display: 'grid', gridTemplateColumns: '2fr 1fr', gap: 12 }}>
+        <Field label={t('proxy.address')}>
+          <input className="input mono" placeholder="proxy.example.com" value={host} onChange={(e) => onHost(e.target.value)} />
+        </Field>
+        <Field label={t('fwd.port')}>
+          <input className="input" inputMode="numeric" placeholder={String(PROXY_DEFAULT_PORT[type])} value={port} onChange={(e) => onPort(e.target.value.replace(/\D/g, ''))} />
+        </Field>
+      </div>
+      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12 }}>
+        <Field label={t('proxy.user')} help={t('proxy.userHelp')}>
+          <input className="input" value={username} onChange={(e) => onUsername(e.target.value)} />
+        </Field>
+        <Field label={t('cred.password')} help={hasSaved ? t('host.savedChange') : t('proxy.saveHint')}>
+          <input className="input" type="password" disabled={!username.trim()} placeholder={hasSaved ? t('host.savedMask') : t('cred.password')} value={password} onChange={(e) => onPassword(e.target.value)} />
+        </Field>
+      </div>
+    </div>
+  );
+}
+
```

### `internal/app/channels_session.go` — 수정

```diff
--- a/internal/app/channels_session.go
+++ b/internal/app/channels_session.go
@@ -37,6 +37,16 @@
 	M_szAuthType       string            `json:"authType"`
 	M_iKeepaliveSec    int               `json:"keepaliveSec"`
 	M_slForwards       []forwardRecord   `json:"forwards"`
+	M_pProxy           *proxyRecord      `json:"proxy"`
+}
+
+// proxyRecord 는 서버의 프록시 경유 설정이다 (data/store.md §3 proxy, D110).
+type proxyRecord struct {
+	M_szType         string `json:"type"` // http | socks5 (그 밖·빈 값은 사용 안 함)
+	M_szHost         string `json:"host"`
+	M_iPort          int    `json:"port"`
+	M_szUsername     string `json:"username"`
+	M_szCredentialId string `json:"credentialId"`
 }
 
 // jumpDirectRecord 는 직접 입력한 점프 호스트 정보다 (data/store.md §3 jumpDirect).
```

### `internal/app/ssh_app.go` — 수정

```diff
--- a/internal/app/ssh_app.go
+++ b/internal/app/ssh_app.go
@@ -113,6 +113,7 @@
 		tgt.AuthType = "ask"
 	}
 	pService.fillCredential(tgt, record.M_szCredentialId, creds, pbNeedUnlock)
+	tgt.Proxy = pService.buildProxy(record.M_pProxy, tgt, creds, pbNeedUnlock)
 	switch {
 	case record.M_szJumpHostId == "direct" && record.M_pJumpDirect != nil:
 		jd := record.M_pJumpDirect
@@ -190,6 +191,48 @@
 	if result == credLocked && !bHadCopy && pbNeedUnlock != nil {
 		*pbNeedUnlock = true
 	}
+}
+
+// g_mProxyDefaultPort 는 프록시 종류별 기본 포트다 (포트를 0 으로 저장했을 때).
+var g_mProxyDefaultPort = map[string]int{sshconn.G_cszProxyHttp: 8080, sshconn.G_cszProxySocks5: 1080}
+
+// buildProxy 는 서버의 프록시 설정을 연결 쪽 형식으로 바꾼다. 설정이 없거나 종류·주소가 비면 nil 이다.
+// 비밀번호는 볼트의 연결된 계정에서 읽고, 볼트가 잠겨 있으면 세션 사본으로 채운다.
+// 사본도 없으면 *pbNeedUnlock 을 true 로 둔다 (protocols.md §2.7, D110).
+func (pService *Service) buildProxy(pRecord *proxyRecord, tgt *sshconn.Target, creds *sessionCreds, pbNeedUnlock *bool) *sshconn.Proxy {
+	if pRecord == nil || pRecord.M_szHost == "" {
+		return nil
+	}
+	iDefault, bKnown := g_mProxyDefaultPort[pRecord.M_szType]
+	if !bKnown {
+		return nil
+	}
+	pProxy := &sshconn.Proxy{Type: pRecord.M_szType, Host: pRecord.M_szHost, Port: pRecord.M_iPort, Username: pRecord.M_szUsername}
+	if pProxy.Port == 0 {
+		pProxy.Port = iDefault
+	}
+	if pRecord.M_szCredentialId == "" {
+		return pProxy
+	}
+	body, errGet := pService.pVault.Get(pRecord.M_szCredentialId)
+	switch {
+	case errGet == nil && body != nil:
+		pProxy.Password = body.M_szPassword
+		if pProxy.Username == "" {
+			pProxy.Username = body.M_szUsername
+		}
+	case errors.Is(errGet, vault.ErrLocked) && pService.vaultLocked():
+		szPassword, bHadCopy := "", false
+		if creds != nil {
+			szPassword, bHadCopy = creds.proxyPassword(tgt)
+		}
+		if bHadCopy {
+			pProxy.Password = szPassword
+		} else if pbNeedUnlock != nil {
+			*pbNeedUnlock = true
+		}
+	}
+	return pProxy
 }
 
 // forwardsOf 는 서버의 포워딩 설정을 연결 쪽 형식으로 바꾼다.
```

### `internal/app/ssh_credcache.go` — 수정

```diff
--- a/internal/app/ssh_credcache.go
+++ b/internal/app/ssh_credcache.go
@@ -23,10 +23,19 @@
 type sessionCreds struct {
 	mtxData  sync.Mutex
 	mapItems map[string]credCopy // 키: 깊이|host:port (점프 체인의 자리마다 하나)
+	mapProxy map[string]string   // 같은 키 → 그 자리에서 쓴 프록시 비밀번호 (D110)
 }
 
 func newSessionCreds() *sessionCreds {
-	return &sessionCreds{mapItems: map[string]credCopy{}}
+	return &sessionCreds{mapItems: map[string]credCopy{}, mapProxy: map[string]string{}}
+}
+
+// proxyPassword 는 tgt 자리에서 접속에 성공했던 프록시 비밀번호 사본을 돌려준다. 없으면 bOk=false.
+func (pCreds *sessionCreds) proxyPassword(tgt *sshconn.Target) (string, bool) {
+	pCreds.mtxData.Lock()
+	defer pCreds.mtxData.Unlock()
+	szPassword, bOk := pCreds.mapProxy[credKey(tgt)]
+	return szPassword, bOk
 }
 
 // credKey 는 점프 체인에서 대상의 자리를 나타내는 키다.
@@ -74,6 +83,9 @@
 			bPrivateKey:  append([]byte(nil), pCur.PrivateKey...),
 			szPassphrase: pCur.Passphrase,
 		}
+		if pCur.Proxy != nil {
+			pCreds.mapProxy[szKey] = pCur.Proxy.Password
+		}
 	}
 }
 
@@ -86,6 +98,7 @@
 		zeroBytes(item.bPrivateKey)
 	}
 	pCreds.mapItems = map[string]credCopy{}
+	pCreds.mapProxy = map[string]string{}
 }
 
 func zeroBytes(bData []byte) {
```

### `internal/sshconn/client.go` — 수정

```diff
--- a/internal/sshconn/client.go
+++ b/internal/sshconn/client.go
@@ -2,6 +2,7 @@
 
 import (
 	"dariterm/internal/i18n"
+	"errors"
 	"sync"
 
 	"github.com/pkg/sftp"
@@ -80,10 +81,13 @@
 			return nil, errJump
 		}
 	} else {
+		if tgt.Proxy != nil {
+			notice(i18n.T("be.proxy.via", "addr", net.JoinHostPort(tgt.Proxy.Host, strconv.Itoa(tgt.Proxy.Port))))
+		}
 		var errDial error
-		conn, errDial = net.DialTimeout("tcp", addr, G_cdurConnectTimeout)
+		conn, errDial = dialDirect(tgt, addr) // 프록시가 있으면 거쳐서 (D110)
 		if errDial != nil {
-			return nil, errDial
+			return nil, proxyError(errDial)
 		}
 	}
 
@@ -184,7 +188,16 @@
 	if jump != nil {
 		return jump.Dial("tcp", addr)
 	}
-	return net.DialTimeout("tcp", addr, G_cdurConnectTimeout)
+	conn, errDial := dialDirect(tgt, addr)
+	return conn, proxyError(errDial)
+}
+
+// proxyError 는 프록시 인증 실패를 안내 문구가 있는 오류로 바꾼다. 그 밖의 오류는 그대로 둔다.
+func proxyError(errDial error) error {
+	if errors.Is(errDial, ErrProxyAuth) {
+		return i18n.NewError("be.proxy.auth")
+	}
+	return errDial
 }
 
 func closeHops(hops []*ssh.Client) {
```

### `internal/sshconn/proxy.go` — 새 파일 (전체)

```go
package sshconn

import (
	"bufio"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"time"
)

// 프록시 경유 접속 (protocols.md §2.7, D110).
// 대상(또는 첫 경유 서버)까지의 TCP 연결을 HTTP 프록시(CONNECT) 또는 SOCKS5 프록시를 거쳐 맺는다.

// 프록시 종류 (data/store.md §3 proxy.type).
const (
	G_cszProxyHttp   = "http"
	G_cszProxySocks5 = "socks5"
)

// Proxy 는 프록시 설정 하나다.
type Proxy struct {
	Type     string // http | socks5
	Host     string
	Port     int
	Username string // 비면 인증 없이 시도한다
	Password string
}

// ErrProxyAuth 는 프록시가 사용자 이름·비밀번호를 받아들이지 않았을 때의 오류다.
var ErrProxyAuth = errors.New("proxy: authentication failed")

// bufferedConn 은 프록시 응답을 읽다가 미리 읽힌 바이트(SSH 서버의 첫 인사 등)를 잃지 않도록
// 읽기를 bufio.Reader 로 돌린 연결이다.
type bufferedConn struct {
	net.Conn
	pReader *bufio.Reader
}

func (pConn *bufferedConn) Read(bBuf []byte) (int, error) {
	return pConn.pReader.Read(bBuf)
}

// dialDirect 는 tgt 로 가는 첫 TCP 연결을 연다: 프록시가 있으면 프록시를 거치고, 없으면 바로 연결한다.
func dialDirect(tgt *Target, szAddr string) (net.Conn, error) {
	if tgt.Proxy == nil {
		return net.DialTimeout("tcp", szAddr, G_cdurConnectTimeout)
	}
	return dialViaProxy(tgt.Proxy, tgt.Host, tgt.Port, G_cdurConnectTimeout)
}

// dialViaProxy 는 프록시에 연결해 대상(szHost:iPort)까지 터널을 연다. 대상 이름은 프록시 쪽에서 푼다.
// durTimeout 은 프록시 연결과 터널 요청 전체의 시한이다.
func dialViaProxy(pProxy *Proxy, szHost string, iPort int, durTimeout time.Duration) (net.Conn, error) {
	szProxyAddr := net.JoinHostPort(pProxy.Host, strconv.Itoa(pProxy.Port))
	conn, errDial := net.DialTimeout("tcp", szProxyAddr, durTimeout)
	if errDial != nil {
		return nil, fmt.Errorf("proxy %s: %w", szProxyAddr, errDial)
	}
	_ = conn.SetDeadline(time.Now().Add(durTimeout))
	var tunnel net.Conn
	var errTunnel error
	switch pProxy.Type {
	case G_cszProxyHttp:
		tunnel, errTunnel = httpConnect(conn, pProxy, szHost, iPort)
	case G_cszProxySocks5:
		tunnel, errTunnel = socks5Connect(conn, pProxy, szHost, iPort)
	default:
		errTunnel = fmt.Errorf("unknown proxy type %q", pProxy.Type)
	}
	if errTunnel != nil {
		_ = conn.Close()
		if errors.Is(errTunnel, ErrProxyAuth) {
			return nil, errTunnel
		}
		return nil, fmt.Errorf("proxy %s: %w", szProxyAddr, errTunnel)
	}
	_ = conn.SetDeadline(time.Time{})
	return tunnel, nil
}

// httpConnect 는 HTTP 프록시에 CONNECT 를 보내 터널을 연다. 사용자 이름이 있으면 Basic 인증을 붙인다.
func httpConnect(conn net.Conn, pProxy *Proxy, szHost string, iPort int) (net.Conn, error) {
	szTarget := net.JoinHostPort(szHost, strconv.Itoa(iPort))
	szRequest := "CONNECT " + szTarget + " HTTP/1.1\r\nHost: " + szTarget + "\r\n"
	if pProxy.Username != "" {
		szToken := base64.StdEncoding.EncodeToString([]byte(pProxy.Username + ":" + pProxy.Password))
		szRequest += "Proxy-Authorization: Basic " + szToken + "\r\n"
	}
	szRequest += "\r\n"
	if _, errWrite := io.WriteString(conn, szRequest); errWrite != nil {
		return nil, errWrite
	}
	pReader := bufio.NewReader(conn)
	pResp, errResp := http.ReadResponse(pReader, &http.Request{Method: http.MethodConnect})
	if errResp != nil {
		return nil, errResp
	}
	// CONNECT 의 2xx 응답에는 본문이 없다. 뒤따르는 바이트는 터널 데이터다.
	if pResp.StatusCode == http.StatusProxyAuthRequired {
		return nil, ErrProxyAuth
	}
	if pResp.StatusCode < 200 || pResp.StatusCode > 299 {
		return nil, fmt.Errorf("CONNECT refused: %s", pResp.Status)
	}
	return &bufferedConn{Conn: conn, pReader: pReader}, nil
}

// socks5Connect 는 SOCKS5 프록시에 CONNECT 를 보내 터널을 연다 (RFC 1928, 사용자 이름·비밀번호 인증은 RFC 1929).
func socks5Connect(conn net.Conn, pProxy *Proxy, szHost string, iPort int) (net.Conn, error) {
	if len(szHost) > 255 || len(pProxy.Username) > 255 || len(pProxy.Password) > 255 {
		return nil, errors.New("socks5: name too long")
	}
	bMethods := []byte{5, 1, 0}
	if pProxy.Username != "" {
		bMethods = []byte{5, 2, 0, 2}
	}
	if _, errWrite := conn.Write(bMethods); errWrite != nil {
		return nil, errWrite
	}
	var arrChoice [2]byte
	if _, errRead := io.ReadFull(conn, arrChoice[:]); errRead != nil {
		return nil, errRead
	}
	if arrChoice[0] != 5 {
		return nil, errors.New("socks5: not a SOCKS5 proxy")
	}
	switch arrChoice[1] {
	case 0:
	case 2:
		if pProxy.Username == "" {
			return nil, ErrProxyAuth
		}
		bAuth := append([]byte{1, byte(len(pProxy.Username))}, pProxy.Username...)
		bAuth = append(append(bAuth, byte(len(pProxy.Password))), pProxy.Password...)
		if _, errWrite := conn.Write(bAuth); errWrite != nil {
			return nil, errWrite
		}
		var arrAuth [2]byte
		if _, errRead := io.ReadFull(conn, arrAuth[:]); errRead != nil {
			return nil, errRead
		}
		if arrAuth[1] != 0 {
			return nil, ErrProxyAuth
		}
	default:
		return nil, ErrProxyAuth // 0xFF: 내놓은 방법을 프록시가 받지 않는다
	}
	bRequest := []byte{5, 1, 0}
	if ip := net.ParseIP(szHost); ip != nil {
		if ip4 := ip.To4(); ip4 != nil {
			bRequest = append(append(bRequest, 1), ip4...)
		} else {
			bRequest = append(append(bRequest, 4), ip.To16()...)
		}
	} else {
		bRequest = append(append(bRequest, 3, byte(len(szHost))), szHost...)
	}
	bRequest = binary.BigEndian.AppendUint16(bRequest, uint16(iPort))
	if _, errWrite := conn.Write(bRequest); errWrite != nil {
		return nil, errWrite
	}
	var arrReply [4]byte // VER REP RSV ATYP
	if _, errRead := io.ReadFull(conn, arrReply[:]); errRead != nil {
		return nil, errRead
	}
	if arrReply[1] != 0 {
		return nil, fmt.Errorf("socks5: connect refused (code %d)", arrReply[1])
	}
	// 응답의 바인드 주소는 쓰지 않지만 길이만큼 읽어 버려야 뒤의 터널 데이터와 섞이지 않는다.
	iSkip := 0
	switch arrReply[3] {
	case 1:
		iSkip = 4
	case 4:
		iSkip = 16
	case 3:
		var arrLength [1]byte
		if _, errRead := io.ReadFull(conn, arrLength[:]); errRead != nil {
			return nil, errRead
		}
		iSkip = int(arrLength[0])
	default:
		return nil, errors.New("socks5: malformed reply")
	}
	if _, errRead := io.CopyN(io.Discard, conn, int64(iSkip+2)); errRead != nil {
		return nil, errRead
	}
	return conn, nil
}
```

### `internal/sshconn/types.go` — 수정

```diff
--- a/internal/sshconn/types.go
+++ b/internal/sshconn/types.go
@@ -54,6 +54,7 @@
 	KeepaliveSec int
 	TermType     string
 	Jump         *Target // 경유 서버 (없으면 nil)
+	Proxy        *Proxy  // 이 대상에 바로 연결할 때 거칠 프록시 (없으면 nil). 경유 서버가 있으면 쓰지 않는다 (D110)
 	Depth        int     // 점프 깊이 (내부용)
 }
 
```


## C. 테스트 코드

### `frontend/src/lib/hostForm.test.ts` — 수정

```diff
--- a/frontend/src/lib/hostForm.test.ts
+++ b/frontend/src/lib/hostForm.test.ts
@@ -40,6 +40,20 @@
     expect(r?.message).toBe('앱이 잠겨 있어 비밀번호를 저장할 수 없습니다');
   });
 
+  it('H4a: 프록시 경유를 켜면 주소가 있어야 하고, 잠긴 채로는 프록시 비밀번호를 저장할 수 없다 (D110)', () => {
+    const base = {
+      protocol: 'ssh' as const, host: '10.0.0.1', serialPath: '', authType: 'ask', hasSecret: false,
+      vaultOpen: true, jumpDirect: false, jumpAddress: '', jumpHasSecret: false,
+    };
+    expect(validateHostForm({ ...base, proxyOn: true, proxyHost: ' ' })).toEqual({ tab: 'adv', message: '프록시 주소를 입력하세요' });
+    expect(validateHostForm({ ...base, vaultOpen: false, proxyOn: true, proxyHost: 'px', proxyHasSecret: true }))
+      .toEqual({ tab: 'adv', message: '앱이 잠겨 있어 프록시 비밀번호를 저장할 수 없습니다' });
+    expect(validateHostForm({ ...base, proxyOn: true, proxyHost: 'px', proxyHasSecret: true })).toBeNull();
+    expect(validateHostForm({ ...base, proxyOn: false, proxyHost: '' })).toBeNull();
+    // 텔넷은 프록시를 쓰지 않으므로 검사하지 않는다.
+    expect(validateHostForm({ ...base, protocol: 'telnet', proxyOn: true, proxyHost: '' })).toBeNull();
+  });
+
   it('H5: 표시 이름을 비우면 호스트 주소가 표시 이름이 된다', () => {
     expect(fallbackAlias('', 'ssh', { host: '10.0.0.5' })).toBe('10.0.0.5');
     expect(fallbackAlias('  ', 'serial', { host: '', serialPath: 'COM4' })).toBe('COM4');
```

### `internal/sshconn/proxy_test.go` — 새 파일 (전체)

```go
package sshconn

import (
	"bufio"
	"encoding/base64"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"
)

// 프록시 경유 접속 시험 (D110).

// startHttpProxy 는 CONNECT 만 받는 가짜 HTTP 프록시다. szUserPass("user:pass")가 있으면 Basic 인증을 요구한다.
// szBanner 가 있으면 200 응답과 같은 쓰기에 붙여 보낸다(터널 데이터가 응답과 함께 도착하는 경우).
func startHttpProxy(t *testing.T, szUserPass string, szBanner string) (string, func() []string) {
	t.Helper()
	ln, errListen := net.Listen("tcp", "127.0.0.1:0")
	if errListen != nil {
		t.Fatal(errListen)
	}
	t.Cleanup(func() { _ = ln.Close() })
	var mtxData sync.Mutex
	var slTargets []string
	go func() {
		for {
			conn, errAccept := ln.Accept()
			if errAccept != nil {
				return
			}
			go func() {
				defer conn.Close()
				pReader := bufio.NewReader(conn)
				pReq, errReq := http.ReadRequest(pReader)
				if errReq != nil || pReq.Method != http.MethodConnect {
					_, _ = io.WriteString(conn, "HTTP/1.1 405 Method Not Allowed\r\n\r\n")
					return
				}
				if szUserPass != "" && pReq.Header.Get("Proxy-Authorization") != "Basic "+base64.StdEncoding.EncodeToString([]byte(szUserPass)) {
					_, _ = io.WriteString(conn, "HTTP/1.1 407 Proxy Authentication Required\r\nProxy-Authenticate: Basic realm=\"x\"\r\nContent-Length: 0\r\n\r\n")
					return
				}
				mtxData.Lock()
				slTargets = append(slTargets, pReq.Host)
				mtxData.Unlock()
				if strings.HasPrefix(pReq.Host, "blocked.invalid") {
					_, _ = io.WriteString(conn, "HTTP/1.1 403 Forbidden\r\nContent-Length: 0\r\n\r\n")
					return
				}
				outbound, errDial := net.DialTimeout("tcp", pReq.Host, 2*time.Second)
				if errDial != nil {
					_, _ = io.WriteString(conn, "HTTP/1.1 502 Bad Gateway\r\nContent-Length: 0\r\n\r\n")
					return
				}
				defer outbound.Close()
				_, _ = io.WriteString(conn, "HTTP/1.1 200 Connection established\r\n\r\n"+szBanner)
				go func() { _, _ = io.Copy(outbound, pReader) }()
				_, _ = io.Copy(conn, outbound)
			}()
		}
	}()
	return ln.Addr().String(), func() []string {
		mtxData.Lock()
		defer mtxData.Unlock()
		return append([]string(nil), slTargets...)
	}
}

func proxyOf(t *testing.T, szType string, szAddr string, szUser string, szPass string) *Proxy {
	t.Helper()
	szHost, szPort, _ := net.SplitHostPort(szAddr)
	var iPort int
	for _, ch := range szPort {
		iPort = iPort*10 + int(ch-'0')
	}
	return &Proxy{Type: szType, Host: szHost, Port: iPort, Username: szUser, Password: szPass}
}

// HTTP 프록시: CONNECT 로 대상까지 이어진다. 응답과 함께 도착한 터널 데이터도 잃지 않는다.
func TestHttpProxyConnect(t *testing.T) {
	iEchoPort := startEcho(t)
	szProxy, fnTargets := startHttpProxy(t, "", "SSH-2.0-banner\r\n")
	conn, errDial := dialViaProxy(proxyOf(t, G_cszProxyHttp, szProxy, "", ""), "127.0.0.1", iEchoPort, 3*time.Second)
	if errDial != nil {
		t.Fatal(errDial)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(3 * time.Second))
	if bGot := readN(t, conn, 16); string(bGot) != "SSH-2.0-banner\r\n" {
		t.Fatalf("응답과 함께 온 데이터를 잃으면 안 됨: %q", string(bGot))
	}
	expectEcho(t, conn, "through-http")
	if slTargets := fnTargets(); len(slTargets) != 1 || !strings.HasPrefix(slTargets[0], "127.0.0.1:") {
		t.Fatalf("CONNECT 대상: %v", slTargets)
	}
}

// HTTP 프록시: 인증이 맞으면 통과, 틀리거나 없으면 ErrProxyAuth, 그 밖의 거절은 상태 줄이 오류에 담긴다.
func TestHttpProxyAuthAndRefusal(t *testing.T) {
	iEchoPort := startEcho(t)
	szProxy, _ := startHttpProxy(t, "kai:s3cret", "")
	conn, errOk := dialViaProxy(proxyOf(t, G_cszProxyHttp, szProxy, "kai", "s3cret"), "127.0.0.1", iEchoPort, 3*time.Second)
	if errOk != nil {
		t.Fatalf("맞는 인증: %v", errOk)
	}
	_ = conn.SetDeadline(time.Now().Add(3 * time.Second))
	expectEcho(t, conn, "auth-ok")
	_ = conn.Close()

	if _, errBad := dialViaProxy(proxyOf(t, G_cszProxyHttp, szProxy, "kai", "wrong"), "127.0.0.1", iEchoPort, 3*time.Second); !errors.Is(errBad, ErrProxyAuth) {
		t.Fatalf("틀린 비밀번호는 인증 오류여야 함: %v", errBad)
	}
	if _, errNone := dialViaProxy(proxyOf(t, G_cszProxyHttp, szProxy, "", ""), "127.0.0.1", iEchoPort, 3*time.Second); !errors.Is(errNone, ErrProxyAuth) {
		t.Fatalf("인증 없이 가면 인증 오류여야 함: %v", errNone)
	}

	szOpen, _ := startHttpProxy(t, "", "")
	_, errBlocked := dialViaProxy(proxyOf(t, G_cszProxyHttp, szOpen, "", ""), "blocked.invalid", 22, 3*time.Second)
	if errBlocked == nil || !strings.Contains(errBlocked.Error(), "403") {
		t.Fatalf("거절 사유가 오류에 담겨야 함: %v", errBlocked)
	}
}

// SOCKS5 프록시(인증 없음): 이 패키지의 SOCKS 서버(동적 포워딩)를 프록시로 써서 맞물려 확인한다. 이름은 프록시가 푼다.
func TestSocks5ProxyConnect(t *testing.T) {
	iEchoPort := startEcho(t)
	szSocks, fnTargets := startSocks(t)
	conn, errDial := dialViaProxy(proxyOf(t, G_cszProxySocks5, szSocks, "", ""), "localhost", iEchoPort, 3*time.Second)
	if errDial != nil {
		t.Fatal(errDial)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(3 * time.Second))
	expectEcho(t, conn, "through-socks5")
	if slTargets := fnTargets(); len(slTargets) != 1 || !strings.HasPrefix(slTargets[0], "localhost:") {
		t.Fatalf("도메인 이름을 그대로 넘겨야 함: %v", slTargets)
	}
	if _, errUnreach := dialViaProxy(proxyOf(t, G_cszProxySocks5, szSocks, "", ""), "unreachable.invalid", 22, 3*time.Second); errUnreach == nil || !strings.Contains(errUnreach.Error(), "code 4") {
		t.Fatalf("프록시의 거절 코드가 오류에 담겨야 함: %v", errUnreach)
	}
}

// startSocks5Auth 는 사용자 이름·비밀번호 인증(RFC 1929)만 받는 가짜 SOCKS5 프록시다. 인증 뒤 대상에 잇는다.
func startSocks5Auth(t *testing.T, szUser string, szPass string) string {
	t.Helper()
	ln, errListen := net.Listen("tcp", "127.0.0.1:0")
	if errListen != nil {
		t.Fatal(errListen)
	}
	t.Cleanup(func() { _ = ln.Close() })
	go func() {
		for {
			conn, errAccept := ln.Accept()
			if errAccept != nil {
				return
			}
			go func() {
				defer conn.Close()
				bHead := make([]byte, 2)
				if _, errRead := io.ReadFull(conn, bHead); errRead != nil {
					return
				}
				bMethods := make([]byte, bHead[1])
				_, _ = io.ReadFull(conn, bMethods)
				if !strings.ContainsRune(string(bMethods), 2) {
					_, _ = conn.Write([]byte{5, 0xFF})
					return
				}
				_, _ = conn.Write([]byte{5, 2})
				_, _ = io.ReadFull(conn, bHead) // VER ULEN
				bUser := make([]byte, bHead[1])
				_, _ = io.ReadFull(conn, bUser)
				bLen := make([]byte, 1)
				_, _ = io.ReadFull(conn, bLen)
				bPass := make([]byte, bLen[0])
				_, _ = io.ReadFull(conn, bPass)
				if string(bUser) != szUser || string(bPass) != szPass {
					_, _ = conn.Write([]byte{1, 1})
					return
				}
				_, _ = conn.Write([]byte{1, 0})
				pReader := bufio.NewReader(conn)
				_, _ = pReader.Discard(3) // VER CMD RSV
				szTarget, errTarget := "", error(nil)
				bAtyp, _ := pReader.ReadByte()
				if bAtyp == 1 {
					bIp := make([]byte, 4)
					_, _ = io.ReadFull(pReader, bIp)
					szTarget = net.IP(bIp).String()
				} else {
					errTarget = errors.New("test proxy: only IPv4")
				}
				bPort := make([]byte, 2)
				_, _ = io.ReadFull(pReader, bPort)
				if errTarget != nil {
					return
				}
				outbound, errDial := net.DialTimeout("tcp", net.JoinHostPort(szTarget, itoaPort(bPort)), 2*time.Second)
				if errDial != nil {
					_, _ = conn.Write([]byte{5, 5, 0, 1, 0, 0, 0, 0, 0, 0})
					return
				}
				defer outbound.Close()
				// 응답의 바인드 주소를 도메인 이름 형식(ATYP 3)으로 보내, 길이가 다른 응답도 바르게 넘기는지 본다.
				_, _ = conn.Write([]byte{5, 0, 0, 3, 4, 't', 'e', 's', 't', 0, 0})
				go func() { _, _ = io.Copy(outbound, pReader) }()
				_, _ = io.Copy(conn, outbound)
			}()
		}
	}()
	return ln.Addr().String()
}

func itoaPort(bPort []byte) string {
	iPort := int(bPort[0])<<8 | int(bPort[1])
	szOut := ""
	for iPort > 0 {
		szOut = string(rune('0'+iPort%10)) + szOut
		iPort /= 10
	}
	return szOut
}

// SOCKS5 프록시(사용자 이름·비밀번호): 맞으면 통과, 틀리거나 사용자 이름이 없으면 ErrProxyAuth.
func TestSocks5ProxyUserPass(t *testing.T) {
	iEchoPort := startEcho(t)
	szProxy := startSocks5Auth(t, "kai", "s3cret")
	conn, errOk := dialViaProxy(proxyOf(t, G_cszProxySocks5, szProxy, "kai", "s3cret"), "127.0.0.1", iEchoPort, 3*time.Second)
	if errOk != nil {
		t.Fatalf("맞는 인증: %v", errOk)
	}
	_ = conn.SetDeadline(time.Now().Add(3 * time.Second))
	expectEcho(t, conn, "socks5-auth")
	_ = conn.Close()
	if _, errBad := dialViaProxy(proxyOf(t, G_cszProxySocks5, szProxy, "kai", "wrong"), "127.0.0.1", iEchoPort, 3*time.Second); !errors.Is(errBad, ErrProxyAuth) {
		t.Fatalf("틀린 비밀번호: %v", errBad)
	}
	if _, errNone := dialViaProxy(proxyOf(t, G_cszProxySocks5, szProxy, "", ""), "127.0.0.1", iEchoPort, 3*time.Second); !errors.Is(errNone, ErrProxyAuth) {
		t.Fatalf("사용자 이름 없이: %v", errNone)
	}
}

// 프록시에 닿지 못하면 프록시 주소가 담긴 오류다. 모르는 종류도 오류다. 프록시가 없으면 바로 연결한다.
func TestProxyDialErrorsAndDirect(t *testing.T) {
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	szDead := ln.Addr().String()
	_ = ln.Close()
	if _, errDead := dialViaProxy(proxyOf(t, G_cszProxyHttp, szDead, "", ""), "example.invalid", 22, time.Second); errDead == nil || !strings.Contains(errDead.Error(), "proxy "+szDead) {
		t.Fatalf("프록시 주소가 오류에 담겨야 함: %v", errDead)
	}
	iEchoPort := startEcho(t)
	szProxy, _ := startHttpProxy(t, "", "")
	if _, errType := dialViaProxy(proxyOf(t, "ftp", szProxy, "", ""), "127.0.0.1", iEchoPort, time.Second); errType == nil {
		t.Fatal("모르는 종류는 오류여야 함")
	}
	conn, errDirect := dialDirect(&Target{Host: "127.0.0.1", Port: iEchoPort}, net.JoinHostPort("127.0.0.1", itoaPort([]byte{byte(iEchoPort >> 8), byte(iEchoPort)})))
	if errDirect != nil {
		t.Fatal(errDirect)
	}
	_ = conn.SetDeadline(time.Now().Add(3 * time.Second))
	expectEcho(t, conn, "direct")
	_ = conn.Close()
}

// 통합: SSH 접속 전체가 프록시를 거쳐 이뤄진다. 비밀번호를 물어 다시 연결하는 경로(dialNext)도 프록시를 거친다.
// 프록시 인증이 틀리면 안내 문구가 있는 오류다.
func TestSSHConnectThroughProxy(t *testing.T) {
	iPort, fnStop := startServer(t)
	defer fnStop()
	for _, szType := range []string{G_cszProxyHttp, G_cszProxySocks5} {
		var szProxy string
		var fnTargets func() []string
		if szType == G_cszProxyHttp {
			szProxy, fnTargets = startHttpProxy(t, "kai:s3cret", "")
		} else {
			szProxy, fnTargets = startSocks(t)
		}
		tgt := targetFor(iPort, "")
		tgt.Proxy = proxyOf(t, szType, szProxy, "kai", "s3cret")
		var slNotices []string
		conn, errConnect := Connect(tgt, &memStore{m: map[string]string{}}, &fakePrompt{password: g_szPass, approve: true}, func(szLine string) { slNotices = append(slNotices, szLine) })
		if errConnect != nil {
			t.Fatalf("%s 프록시 경유 접속: %v", szType, errConnect)
		}
		_ = conn.Close()
		// 저장된 비밀번호가 없어 1차 시도가 인증 실패 → 비밀번호를 묻고 새 연결: 프록시를 두 번 거친다.
		if slTargets := fnTargets(); len(slTargets) != 2 {
			t.Fatalf("%s: 두 번의 연결이 모두 프록시를 거쳐야 함: %v", szType, slTargets)
		}
		if len(slNotices) == 0 {
			t.Fatalf("%s: 프록시 경유 안내 줄이 있어야 함", szType)
		}
	}
	szProxy, _ := startHttpProxy(t, "kai:s3cret", "")
	tgt := targetFor(iPort, g_szPass)
	tgt.Proxy = proxyOf(t, G_cszProxyHttp, szProxy, "kai", "wrong")
	if _, errAuth := Connect(tgt, &memStore{m: map[string]string{}}, &fakePrompt{approve: true}, func(string) {}); errAuth == nil || errors.Is(errAuth, ErrProxyAuth) {
		t.Fatalf("프록시 인증 실패는 안내 문구 오류로 바뀌어야 함: %v", errAuth)
	}
}
```


## E. 문구

`locales/<언어>.json` 10개 파일에 같은 키를 넣는다(순서: ko, en, ja, zh, tw, de, fr, es, pt, ru). 한국어·영어 외에는 이 작업을 한 세션이 번역한 것이라 검수가 필요할 수 있다.

**`host.proxyAccountName`** (추가)

```json
  "host.proxyAccountName": "{alias} 프록시",
  "host.proxyAccountName": "{alias} proxy",
  "host.proxyAccountName": "{alias} プロキシ",
  "host.proxyAccountName": "{alias} 代理",
  "host.proxyAccountName": "{alias} 代理",
  "host.proxyAccountName": "{alias} Proxy",
  "host.proxyAccountName": "{alias} proxy",
  "host.proxyAccountName": "{alias} proxy",
  "host.proxyAccountName": "{alias} proxy",
  "host.proxyAccountName": "{alias} прокси",
```

**`hostForm.needProxy`** (추가)

```json
  "hostForm.needProxy": "프록시 주소를 입력하세요",
  "hostForm.needProxy": "Enter the proxy address",
  "hostForm.needProxy": "プロキシのアドレスを入力してください",
  "hostForm.needProxy": "请输入代理地址",
  "hostForm.needProxy": "請輸入代理位址",
  "hostForm.needProxy": "Geben Sie die Proxy-Adresse ein",
  "hostForm.needProxy": "Saisissez l'adresse du proxy",
  "hostForm.needProxy": "Introduzca la dirección del proxy",
  "hostForm.needProxy": "Informe o endereço do proxy",
  "hostForm.needProxy": "Введите адрес прокси",
```

**`hostForm.lockedProxy`** (추가)

```json
  "hostForm.lockedProxy": "앱이 잠겨 있어 프록시 비밀번호를 저장할 수 없습니다",
  "hostForm.lockedProxy": "The app is locked, so the proxy password cannot be saved",
  "hostForm.lockedProxy": "アプリがロックされているため、プロキシのパスワードを保存できません",
  "hostForm.lockedProxy": "应用已锁定，无法保存代理密码",
  "hostForm.lockedProxy": "應用程式已鎖定，無法儲存代理密碼",
  "hostForm.lockedProxy": "Die App ist gesperrt, das Proxy-Passwort kann nicht gespeichert werden",
  "hostForm.lockedProxy": "L'application est verrouillée ; le mot de passe du proxy ne peut pas être enregistré",
  "hostForm.lockedProxy": "La aplicación está bloqueada; no se puede guardar la contraseña del proxy",
  "hostForm.lockedProxy": "O app está bloqueado; não é possível salvar a senha do proxy",
  "hostForm.lockedProxy": "Приложение заблокировано, пароль прокси сохранить нельзя",
```

**`host.proxy`** (추가)

```json
  "host.proxy": "프록시 경유",
  "host.proxy": "Connect through a proxy",
  "host.proxy": "プロキシ経由",
  "host.proxy": "通过代理连接",
  "host.proxy": "透過代理連線",
  "host.proxy": "Über Proxy verbinden",
  "host.proxy": "Connexion via un proxy",
  "host.proxy": "Conectar mediante proxy",
  "host.proxy": "Conectar por proxy",
  "host.proxy": "Подключение через прокси",
```

**`host.proxyHelp`** (추가)

```json
  "host.proxyHelp": "회사 방화벽처럼 서버에 바로 연결할 수 없을 때, 이 프록시를 거쳐 SSH 연결을 맺습니다.",
  "host.proxyHelp": "When the server cannot be reached directly (for example behind a corporate firewall), the SSH connection is made through this proxy.",
  "host.proxyHelp": "社内ファイアウォールなどでサーバーに直接接続できないとき、このプロキシを経由して SSH 接続します。",
  "host.proxyHelp": "当无法直接连接服务器（例如受公司防火墙限制）时，通过此代理建立 SSH 连接。",
  "host.proxyHelp": "當無法直接連線到伺服器（例如受公司防火牆限制）時，透過此代理建立 SSH 連線。",
  "host.proxyHelp": "Wenn der Server nicht direkt erreichbar ist (z. B. hinter einer Firmen-Firewall), wird die SSH-Verbindung über diesen Proxy aufgebaut.",
  "host.proxyHelp": "Lorsque le serveur n'est pas joignable directement (par exemple derrière un pare-feu d'entreprise), la connexion SSH passe par ce proxy.",
  "host.proxyHelp": "Cuando no se puede llegar al servidor directamente (por ejemplo, tras un cortafuegos corporativo), la conexión SSH se establece a través de este proxy.",
  "host.proxyHelp": "Quando o servidor não pode ser alcançado diretamente (por exemplo, atrás de um firewall corporativo), a conexão SSH é feita por este proxy.",
  "host.proxyHelp": "Если к серверу нельзя подключиться напрямую (например, из-за корпоративного файрвола), SSH-соединение устанавливается через этот прокси.",
```

**`proxy.http`** (추가)

```json
  "proxy.http": "HTTP 프록시 (CONNECT)",
  "proxy.http": "HTTP proxy (CONNECT)",
  "proxy.http": "HTTP プロキシ (CONNECT)",
  "proxy.http": "HTTP 代理 (CONNECT)",
  "proxy.http": "HTTP 代理 (CONNECT)",
  "proxy.http": "HTTP-Proxy (CONNECT)",
  "proxy.http": "Proxy HTTP (CONNECT)",
  "proxy.http": "Proxy HTTP (CONNECT)",
  "proxy.http": "Proxy HTTP (CONNECT)",
  "proxy.http": "HTTP-прокси (CONNECT)",
```

**`proxy.socks5`** (추가)

```json
  "proxy.socks5": "SOCKS5 프록시",
  "proxy.socks5": "SOCKS5 proxy",
  "proxy.socks5": "SOCKS5 プロキシ",
  "proxy.socks5": "SOCKS5 代理",
  "proxy.socks5": "SOCKS5 代理",
  "proxy.socks5": "SOCKS5-Proxy",
  "proxy.socks5": "Proxy SOCKS5",
  "proxy.socks5": "Proxy SOCKS5",
  "proxy.socks5": "Proxy SOCKS5",
  "proxy.socks5": "SOCKS5-прокси",
```

**`proxy.address`** (추가)

```json
  "proxy.address": "프록시 주소",
  "proxy.address": "Proxy address",
  "proxy.address": "プロキシのアドレス",
  "proxy.address": "代理地址",
  "proxy.address": "代理位址",
  "proxy.address": "Proxy-Adresse",
  "proxy.address": "Adresse du proxy",
  "proxy.address": "Dirección del proxy",
  "proxy.address": "Endereço do proxy",
  "proxy.address": "Адрес прокси",
```

**`proxy.user`** (추가)

```json
  "proxy.user": "프록시 사용자 이름",
  "proxy.user": "Proxy user name",
  "proxy.user": "プロキシのユーザー名",
  "proxy.user": "代理用户名",
  "proxy.user": "代理使用者名稱",
  "proxy.user": "Proxy-Benutzername",
  "proxy.user": "Nom d'utilisateur du proxy",
  "proxy.user": "Usuario del proxy",
  "proxy.user": "Usuário do proxy",
  "proxy.user": "Имя пользователя прокси",
```

**`proxy.userHelp`** (추가)

```json
  "proxy.userHelp": "프록시가 로그인을 요구할 때만 넣습니다. 비우면 인증 없이 연결합니다.",
  "proxy.userHelp": "Fill in only if the proxy requires a login. Leave empty to connect without authentication.",
  "proxy.userHelp": "プロキシがログインを要求する場合のみ入力します。空欄なら認証なしで接続します。",
  "proxy.userHelp": "仅在代理要求登录时填写。留空则不经认证连接。",
  "proxy.userHelp": "僅在代理要求登入時填寫。留空則不經驗證連線。",
  "proxy.userHelp": "Nur ausfüllen, wenn der Proxy eine Anmeldung verlangt. Leer lassen, um ohne Authentifizierung zu verbinden.",
  "proxy.userHelp": "À renseigner uniquement si le proxy exige une connexion. Laissez vide pour vous connecter sans authentification.",
  "proxy.userHelp": "Rellénelo solo si el proxy exige inicio de sesión. Déjelo vacío para conectar sin autenticación.",
  "proxy.userHelp": "Preencha somente se o proxy exigir login. Deixe vazio para conectar sem autenticação.",
  "proxy.userHelp": "Заполняйте, только если прокси требует вход. Оставьте пустым для подключения без аутентификации.",
```

**`proxy.saveHint`** (추가)

```json
  "proxy.saveHint": "암호화해 저장 (\"{표시 이름} 프록시\")",
  "proxy.saveHint": "Stored encrypted (\"{display name} proxy\")",
  "proxy.saveHint": "暗号化して保存（「{表示名} プロキシ」）",
  "proxy.saveHint": "加密保存（“{显示名称} 代理”）",
  "proxy.saveHint": "加密儲存（「{顯示名稱} 代理」）",
  "proxy.saveHint": "Verschlüsselt gespeichert (als „Anzeigename Proxy“)",
  "proxy.saveHint": "Enregistré chiffré (« {nom affiché} proxy »)",
  "proxy.saveHint": "Se guarda cifrada («{nombre visible} proxy»)",
  "proxy.saveHint": "Salva criptografada (\"{nome de exibição} proxy\")",
  "proxy.saveHint": "Сохраняется в зашифрованном виде («{отображаемое имя} прокси»)",
```

**`proxy.jumpNote`** (추가)

```json
  "proxy.jumpNote": "경유 서버를 쓰는 동안에는 이 프록시를 쓰지 않습니다. 경유 서버에 닿는 데 프록시가 필요하면 경유 서버 쪽에 프록시를 설정하세요.",
  "proxy.jumpNote": "This proxy is not used while a jump server is selected. If the jump server itself must be reached through a proxy, set the proxy on the jump server.",
  "proxy.jumpNote": "踏み台サーバーを使っている間、このプロキシは使いません。踏み台サーバーへの到達にプロキシが必要な場合は、踏み台サーバー側にプロキシを設定してください。",
  "proxy.jumpNote": "选择了跳板服务器时不使用此代理。如果需要通过代理才能连接跳板服务器，请在跳板服务器上设置代理。",
  "proxy.jumpNote": "選擇了跳板伺服器時不使用此代理。如果需要透過代理才能連上跳板伺服器，請在跳板伺服器上設定代理。",
  "proxy.jumpNote": "Dieser Proxy wird nicht verwendet, solange ein Sprungserver gewählt ist. Wenn der Sprungserver selbst nur über einen Proxy erreichbar ist, richten Sie den Proxy beim Sprungserver ein.",
  "proxy.jumpNote": "Ce proxy n'est pas utilisé tant qu'un serveur de rebond est sélectionné. Si le serveur de rebond doit lui-même être atteint via un proxy, configurez le proxy sur le serveur de rebond.",
  "proxy.jumpNote": "Este proxy no se usa mientras haya un servidor de salto seleccionado. Si el propio servidor de salto requiere un proxy, configure el proxy en el servidor de salto.",
  "proxy.jumpNote": "Este proxy não é usado enquanto um servidor de salto estiver selecionado. Se o próprio servidor de salto precisar de proxy, configure o proxy no servidor de salto.",
  "proxy.jumpNote": "Этот прокси не используется, пока выбран промежуточный сервер. Если до промежуточного сервера нужно добираться через прокси, настройте прокси у него.",
```

**`be.proxy.via`** (추가)

```json
  "be.proxy.via": "프록시 {addr} 경유 중...",
  "be.proxy.via": "Connecting through proxy {addr}...",
  "be.proxy.via": "プロキシ {addr} を経由中...",
  "be.proxy.via": "正在通过代理 {addr} 连接...",
  "be.proxy.via": "正在透過代理 {addr} 連線...",
  "be.proxy.via": "Verbindung über Proxy {addr}...",
  "be.proxy.via": "Connexion via le proxy {addr}...",
  "be.proxy.via": "Conectando a través del proxy {addr}...",
  "be.proxy.via": "Conectando pelo proxy {addr}...",
  "be.proxy.via": "Подключение через прокси {addr}...",
```

**`be.proxy.auth`** (추가)

```json
  "be.proxy.auth": "프록시가 사용자 이름 또는 비밀번호를 받아들이지 않았습니다",
  "be.proxy.auth": "The proxy rejected the user name or password",
  "be.proxy.auth": "プロキシがユーザー名またはパスワードを受け付けませんでした",
  "be.proxy.auth": "代理拒绝了用户名或密码",
  "be.proxy.auth": "代理拒絕了使用者名稱或密碼",
  "be.proxy.auth": "Der Proxy hat Benutzername oder Passwort abgelehnt",
  "be.proxy.auth": "Le proxy a refusé le nom d'utilisateur ou le mot de passe",
  "be.proxy.auth": "El proxy rechazó el usuario o la contraseña",
  "be.proxy.auth": "O proxy rejeitou o usuário ou a senha",
  "be.proxy.auth": "Прокси отклонил имя пользователя или пароль",
```


## F. 명세

### `spec/data/store.md` — 수정

```diff
--- a/spec/data/store.md
+++ b/spec/data/store.md
@@ -44,6 +44,7 @@
 | credentialId | 문자열 또는 null(선택) | 볼트 항목 id |
 | jumpHostId | 문자열 또는 null(선택) | 점프 호스트 서버 id. 값이 `direct` 이면 jumpDirect 사용 |
 | jumpDirect | 객체(선택) | host, port, username, authType(`password`/`key`), credentialId(선택) |
+| proxy | 객체(선택) | 프록시 경유(2026-11-23, D110): type(`http`/`socks5`), host, port, username(비면 인증 없음), credentialId(선택, 비밀번호 계정). 없거나 type·host 가 비면 바로 연결. SSH 만 쓴다 |
 | color | 문자열(선택) | |
 | env | 환경 값(선택) | |
 | tags | 문자열 배열 | |
```

### `spec/protocols.md` — 수정

```diff
--- a/spec/protocols.md
+++ b/spec/protocols.md
@@ -57,6 +57,17 @@
   - **R(원격)**: 서버 쪽에서 바인드 주소:포트로 대기 → 연결이 오면 이 PC 에서 대상 호스트:포트로 중계.
   - **D(동적)**(2026-11-23, D109): 이 PC 의 바인드 주소:포트를 **SOCKS 프록시**로 연다. 들어온 연결의 SOCKS 요청(SOCKS5 인증 없음·SOCKS4·SOCKS4a, 명령은 CONNECT 만)이 가리키는 대상으로 SSH 연결을 통해 중계한다. 대상 호스트·포트 설정은 쓰지 않는다. 도메인 이름은 이 PC 가 아니라 **서버 쪽에서** 풀린다. SOCKS5 에서 인증 없음을 내놓지 않은 클라이언트·CONNECT 가 아닌 명령·닿지 않는 대상은 SOCKS 오류 응답으로 거절하고, SOCKS 가 아닌 연결은 응답 없이 닫는다. 요청을 10초 안에 다 보내지 않는 연결은 닫는다. 프록시 자체에는 인증이 없으므로 바인드 주소를 `127.0.0.1` 밖으로 열면 그 주소에 닿는 누구나 쓸 수 있다. 성공 시 터미널에 "동적 포워딩(SOCKS 프록시) {바인드}:{포트}", 실패 시 "포트 포워딩 D{포트} 실패: {오류}".
 - 설정 성공 시 터미널에 "포워딩 {바인드}:{포트} → {대상}", 실패 시 "포트 포워딩 {L|R}{포트} 실패: {오류}" 를 보이고 다음 항목으로 계속한다. 세션이 끝나면 모든 대기 소켓을 닫는다.
+
+### 2.7 프록시 경유 (2026-11-23, D110)
+- 서버에 `proxy` 가 있고 **경유 서버(점프 호스트)가 없으면**, 대상까지의 TCP 연결을 프록시를 거쳐 맺는다. 경유 서버가 있으면 그 서버의 `proxy` 는 쓰지 않는다(연결이 경유 서버 위에서 열리기 때문). 대신 **경유 서버 체인의 맨 바깥 서버**(이 PC 가 직접 연결하는 서버)에 `proxy` 가 있으면 그 연결이 프록시를 거친다. 직접 입력한 경유 서버에는 프록시를 둘 수 없다.
+- 연결 전에 터미널에 안내 "프록시 {주소}:{포트} 경유 중..." 을 보인다. 프록시 연결과 터널 요청을 합쳐 제한시간 **20초**.
+- **HTTP 프록시**(`http`): `CONNECT 대상:포트` 요청. 사용자 이름이 있으면 Basic 인증을 붙인다. 응답이 2xx 이면 터널이 열린 것이고, 407 이면 인증 실패, 그 밖은 상태 줄을 오류로 보인다. 포트가 0 이면 8080.
+- **SOCKS5 프록시**(`socks5`): 사용자 이름이 없으면 인증 없음만, 있으면 인증 없음과 사용자 이름·비밀번호 인증을 내놓는다. CONNECT 만 쓴다. 포트가 0 이면 1080.
+- 두 방식 모두 **대상 이름은 프록시 쪽에서 푼다**(이 PC 에서 DNS 조회를 하지 않는다).
+- 인증 실패는 접속 실패 "프록시가 사용자 이름 또는 비밀번호를 받아들이지 않았습니다". 그 밖의 실패는 `proxy 주소: 원인` 형태의 오류다. 자동 재접속 규칙은 다른 접속 실패와 같다.
+- 비밀번호를 물어 다시 연결할 때(§2.2 5번)도 매번 프록시를 거친다.
+- **프록시 비밀번호**는 볼트의 계정(`proxy.credentialId`)에서 읽는다. 볼트가 잠겨 있으면 세션 사본(§2.2, 접속에 성공했던 프록시 비밀번호를 세션 메모리에 둔 것)을 쓰고, 사본도 없으면 접속하지 않고 잠금 해제를 기다린다.
+- 범위 밖: SOCKS4 프록시, 프록시 명령(ProxyCommand), 프록시의 NTLM·Kerberos 인증, 텔넷의 프록시 경유.
 
 ### 2.6 연결 유지(keepalive)
 `keepaliveSec` 가 0 보다 크면 그 간격으로 `keepalive@openssh.com` 요청을 보낸다. **연속 4번 실패하면 연결을 닫는다**(세션 종료 처리 → 자동 재접속 규칙 적용). 요청의 응답이 keepalive 간격 안에 오지 않으면 그 요청은 실패로 센다(상대가 조용히 사라져도 약 4 × 간격 안에 끊김을 알아챈다). 연결을 닫으면 keepalive 도 바로 멈춘다.
```

### `spec/ui/host-editor.md` — 수정

```diff
--- a/spec/ui/host-editor.md
+++ b/spec/ui/host-editor.md
@@ -48,6 +48,8 @@
 4. **접속 후 자동 실행 명령**(로컬이면 "시작할 때 실행할 명령") — 고정폭 입력. 도움말 로컬 "예: claude --continue", 그 외 "예: cd /var/www && sudo -i"(플레이스홀더도 같음). 세션 유지가 켜져 있으면 줄바꿈 후 "세션을 처음 만들 때만 실행됩니다 (이미 있는 세션에 다시 붙을 때는 실행되지 않음)".
 5. 로컬: "터미널 종류" 한 칸. 그 외: 3열 — "문자 인코딩"(옵션 4개: "UTF-8"(utf-8), "CP949 / EUC-KR (구형 장비)"(cp949), "Shift_JIS"(shift_jis), "Latin-1"(latin1). 저장 데이터에 `euc-kr` 값이 있어도 읽을 수 있어야 한다) / "터미널 종류" / "연결 확인 간격 (초)"(숫자, 비면 0).
 
+**프록시 경유**(SSH 만, 2026-11-23, D110): 경유 서버 항목 아래에 "프록시 경유" 드롭다운(없음 / HTTP 프록시 (CONNECT) / SOCKS5 프록시), 도움말 "회사 방화벽처럼 서버에 바로 연결할 수 없을 때, 이 프록시를 거쳐 SSH 연결을 맺습니다." 종류를 고르면 카드가 나온다: 2열(2:1) "프록시 주소"(고정폭, 플레이스홀더 `proxy.example.com`) / "포트"(숫자, 플레이스홀더는 종류별 기본 포트 8080·1080), 2열 "프록시 사용자 이름"(도움말 "프록시가 로그인을 요구할 때만 넣습니다. 비우면 인증 없이 연결합니다.") / "비밀번호"(사용자 이름이 비면 비활성. 저장된 것이 있으면 플레이스홀더가 가림 문자, 도움말은 경유 서버 카드와 같은 규칙). 경유 서버를 골라 둔 상태면 카드 맨 위에 안내 "경유 서버를 쓰는 동안에는 이 프록시를 쓰지 않습니다. …". 저장: 종류와 주소가 있으면 `proxy` 를 쓰고(포트 0 은 기본 포트), 사용자 이름과 비밀번호를 넣었으면 `{별칭} 프록시` 이름의 비밀번호 계정으로 저장해 연결한다. 사용자 이름을 비우면 계정 연결을 끊는다. 검사: 종류를 골랐는데 주소가 비면 "프록시 주소를 입력하세요", 비밀번호를 넣었는데 앱이 잠겨 있으면 "앱이 잠겨 있어 프록시 비밀번호를 저장할 수 없습니다"(둘 다 고급 탭으로 이동).
+
 ### 1.4 "포트 포워딩" 탭 (SSH 만)
 - 안내 박스: "L: 내 PC의 포트 → 서버에서 접근 가능한 주소 (예: 로컬 3307 → DB 3306). R: 서버의 포트 → 내 PC 쪽 주소. D: 내 PC의 포트를 SOCKS 프록시로 열어, 그 프록시를 쓰는 프로그램의 접속을 서버를 거쳐 내보냅니다(대상은 프로그램이 정합니다)."(2026-11-23, D109)
 - 표: 머리 종류 · 바인드 주소 · 포트 · 대상 호스트 · 대상 포트 · (삭제). 행마다 종류 드롭다운(L/R/D), 바인드 주소(고정폭), 포트(숫자), 대상 호스트(고정폭), 대상 포트(숫자), [삭제](휴지통 아이콘). 종류가 D 이면 대상 호스트·대상 포트 칸 대신 흐린 글자 "SOCKS 프록시 (대상 없음)" 을 두 칸 너비로 보인다(적어 둔 대상 값은 지우지 않아 종류를 되돌리면 다시 보인다).
```



---

## G. 검증 (2026-11-23 실행)

| 테스트 ([internal/sshconn/proxy_test.go](../../internal/sshconn/proxy_test.go)) | 확인하는 것 |
|---|---|
| `TestHttpProxyConnect` | 가짜 HTTP 프록시를 거쳐 대상까지 이어진다 / 200 응답과 **같은 묶음으로 온 데이터**(서버 인사 흉내)를 잃지 않는다 |
| `TestHttpProxyAuthAndRefusal` | Basic 인증이 맞으면 통과 / 틀리거나 없으면 `ErrProxyAuth` / 403 은 상태 줄이 오류에 담긴다 |
| `TestSocks5ProxyConnect` | 이 앱의 SOCKS 서버(동적 포워딩)를 프록시로 써서 맞물려 확인 / 도메인 이름이 그대로 넘어간다 / 프록시의 거절 코드가 오류에 담긴다 |
| `TestSocks5ProxyUserPass` | 사용자 이름·비밀번호 인증(RFC 1929) / 응답의 바인드 주소가 도메인 형식이어도 길이를 맞게 넘긴다 / 틀리면 `ErrProxyAuth` |
| `TestProxyDialErrorsAndDirect` | 프록시에 닿지 못하면 프록시 주소가 오류에 담긴다 / 모르는 종류는 오류 / 프록시가 없으면 바로 연결 |
| `TestSSHConnectThroughProxy` | **SSH 접속 전체**가 HTTP·SOCKS5 프록시를 거쳐 된다 / 비밀번호를 물어 다시 연결할 때도 프록시를 거친다(프록시가 본 연결 2번) / 프록시 경유 안내 줄이 나온다 / 프록시 인증 실패가 안내 문구 오류로 바뀐다 |

프런트 [frontend/src/lib/hostForm.test.ts](../../frontend/src/lib/hostForm.test.ts) `H4a`: 프록시를 켜면 주소가 있어야 한다 / 잠긴 채로는 프록시 비밀번호를 저장할 수 없다 / 텔넷은 검사하지 않는다.

```bash
go test ./internal/sshconn/ ./internal/app/     # 통과
(cd frontend && npx tsc --noEmit && npx vitest run)   # 통과
```

앱 계층의 `buildProxy`(볼트에서 비밀번호 읽기, 잠긴 동안 사본 쓰기)와 저장 화면(`ProxyCard`, `hostSave.ts` 의 계정 저장)은 **자동 테스트가 없다.** → 수동 검증 3~6번.

## H. 수동 검증 (남은 일)

| # | 절차 | 기대 결과 |
|---|---|---|
| 1 | 인증 없는 HTTP 프록시(예: 사내 프록시, 또는 다른 서버에 띄운 tinyproxy)를 서버 편집 > 고급 > 프록시 경유 > HTTP 프록시로 넣고 연결 | 터미널에 `프록시 {주소} 경유 중...` 이 보이고 접속된다 |
| 2 | SOCKS5 프록시(예: 다른 터미널의 `ssh -D 1080 …`, 또는 이 앱의 동적 포워딩)를 넣고 연결 | 접속된다 |
| 3 | 사용자 이름·비밀번호가 필요한 프록시: 둘 다 넣고 저장 → 다시 열기 | 비밀번호 칸이 가림 문자로 보이고, 설정 > 보안 > 저장된 계정에 `{별칭} 프록시` 가 있다. 접속된다 |
| 4 | 3번에서 비밀번호를 틀리게 | `프록시가 사용자 이름 또는 비밀번호를 받아들이지 않았습니다` 로 접속 실패 |
| 5 | 3번 서버를 연결한 채 앱을 잠그고 서버 쪽에서 연결을 끊는다 | 잠긴 채로 다시 접속된다(세션 사본) |
| 6 | 프록시 사용자 이름을 지우고 저장 | 계정 연결이 끊긴다(`proxy.credentialId` 가 null) |
| 7 | 경유 서버를 고른 서버에서 프록시 경유도 켠다 | 카드에 "경유 서버를 쓰는 동안에는 이 프록시를 쓰지 않습니다…" 안내. 접속은 경유 서버로 된다 |
| 8 | 프록시가 있는 서버 A 를 서버 B 의 경유 서버로 고르고 B 에 연결 | A 까지는 프록시를 거치고, B 는 A 위에서 열린다 |

## I. 영향 범위·회귀 위험

| 영역 | 위험 | 대비 |
|---|---|---|
| 프록시가 없는 서버(전부) | `dialDirect` 가 `tgt.Proxy == nil` 이면 전과 같은 `net.DialTimeout` 을 부른다 | 기존 SSH 테스트 전부, `TestProxyDialErrorsAndDirect` |
| 저장 데이터 | 서버에 `proxy` 필드가 생긴다(선택). 옛 버전 앱은 무시한다(바로 연결 시도) | — |
| 볼트 | `{별칭} 프록시` 이름의 비밀번호 계정이 생긴다. 서버를 지워도 계정은 남는다(경유 서버 계정과 같은 성질) | 설정 > 보안 > 저장된 계정에서 지운다 |
| HTTP 프록시 인증 | Basic 은 비밀번호가 프록시까지 평문(base64)으로 간다 | HTTP CONNECT 의 성질. 화면에 따로 경고하지 않았다 |

## J. 체크리스트

- [x] `proxy.go` 새 파일, `types.go`(Target.Proxy), `client.go`(dialDirect·proxyError)
- [x] 앱: `proxyRecord`, `buildProxy`, 세션 사본에 프록시 비밀번호
- [x] 프런트: `ProxyConf`·검사(hostForm.ts), 계정 저장(hostSave.ts), 고급 탭 선택·`ProxyCard`
- [x] 문구 13개 추가(10개 언어), 명세 protocols.md §2.7·data/store.md §3·ui/host-editor.md §1.3, decisions D110
- [x] 자동 테스트 6개 + 프런트 1개 통과
- [ ] 수동 검증 1~8
