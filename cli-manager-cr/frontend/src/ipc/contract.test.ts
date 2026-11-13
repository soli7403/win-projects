// 계약 고정 테스트(프런트 쪽). 백엔드 Go 테스트가 만든 고정 JSON(src/ipc/contract/*.json)을 읽어
// (1) 런타임 검증기(spec/ipc.md·data/store.md 필드)를 통과하는지, (2) 상태 저장소·화면 로직이 크래시 없이 처리하는지 본다.
// 고정 파일 갱신: go test ./internal/app -run TestContract -update (프로젝트 루트에서 실행)
import { describe, it, expect, beforeEach } from 'vitest';
import appInfo from './contract/app.info.json';
import storeGet from './contract/store.get.json';
import storeAfterNull from './contract/store.get.afterNullPatch.json';
import vaultStatus from './contract/vault.status.json';
import vaultLocked from './contract/vault.status.locked.json';
import vaultUnlock from './contract/vault.unlock.json';
import vaultSetup from './contract/vault.setup.json';
import credList from './contract/cred.list.json';
import credGet from './contract/cred.get.json';
import credSave from './contract/cred.save.json';
import syncStatus from './contract/sync.status.json';
import updateStatus from './contract/update.status.json';
import sessionOpen from './contract/session.open.json';
import evTransfer from './contract/event.transfer.update.json';
import evSync from './contract/event.sync.status.json';
import evUpdate from './contract/event.update.available.json';
import evUiRequest from './contract/event.ui.request.json';
import { setBackend, type Backend } from '../ipc/backend';
import { normalizeStore, useApp } from '../state/app';
import { useTransfers } from '../state/transfers';
import { useRequests } from '../state/requests';
import { openSessionFor } from '../state/workspace';

type Errs = string[];
const isObj = (v: unknown): v is Record<string, unknown> => !!v && typeof v === 'object' && !Array.isArray(v);
function need(errs: Errs, cond: boolean, msg: string) { if (!cond) errs.push(msg); }
const str = (v: unknown) => typeof v === 'string';
const num = (v: unknown) => typeof v === 'number' && Number.isFinite(v);
const bool = (v: unknown) => typeof v === 'boolean';

// 검증기: spec/ipc.md 와 data/store.md 의 필드 구조 (필수 필드 + 타입). 선택 필드는 없거나 올바른 타입이면 된다.
const V = {
  appInfo(v: unknown): Errs {
    const e: Errs = []; need(e, isObj(v) && str(v.version) && str(v.platform) && str(v.dataDir), 'app.info'); return e;
  },
  vaultStatus(v: unknown): Errs {
    const e: Errs = [];
    need(e, isObj(v) && bool(v.initialized) && bool(v.unlocked) && bool(v.osUnlockAvailable) && bool(v.osUnlockEnabled), 'vault.status 필수 불리언');
    if (isObj(v) && v.osUnlockKind !== undefined) need(e, str(v.osUnlockKind), 'osUnlockKind 는 문자열');
    return e;
  },
  setup(v: unknown): Errs { const e: Errs = []; need(e, isObj(v) && str(v.recoveryCode), 'vault.setup.recoveryCode'); return e; },
  credList(v: unknown): Errs {
    const e: Errs = [];
    need(e, Array.isArray(v), 'cred.list 배열');
    (Array.isArray(v) ? v : []).forEach((c) => need(e, isObj(c) && str(c.id) && str(c.name) && (c.kind === 'password' || c.kind === 'key') && num(c.updatedAt), 'cred 메타'));
    return e;
  },
  credSecret(v: unknown): Errs {
    const e: Errs = [];
    if (v === null) return e;
    need(e, isObj(v), 'cred.get 객체 또는 null');
    if (isObj(v)) for (const k of ['username', 'password', 'privateKey', 'passphrase']) if (v[k] !== undefined) need(e, str(v[k]), `비밀 ${k} 문자열`);
    return e;
  },
  credMeta(v: unknown): Errs {
    const e: Errs = []; need(e, isObj(v) && str(v.id) && str(v.name) && num(v.updatedAt), 'cred.save 결과 메타'); return e;
  },
  store(v: unknown): Errs {
    const e: Errs = [];
    need(e, isObj(v), 'store 객체');
    if (!isObj(v)) return e;
    for (const k of ['groups', 'hosts', 'snippets', 'knownHosts']) need(e, Array.isArray(v[k]), `store.${k} 배열`);
    need(e, isObj(v.settings), 'store.settings 객체');
    for (const h of Array.isArray(v.hosts) ? v.hosts : []) {
      need(e, isObj(h) && str(h.id) && str(h.alias) && str(h.protocol) && str(h.host) && num(h.port), 'host 필수 필드');
      if (isObj(h)) need(e, Array.isArray(h.tags) && Array.isArray(h.forwards), 'host.tags/forwards 배열');
    }
    return e;
  },
  sessionInfo(v: unknown): Errs {
    const e: Errs = [];
    const states = ['connecting', 'connected', 'reconnecting', 'closed', 'error'];
    const o = isObj(v) ? v : {};
    need(e, str(o.id) && str(o.title) && str(o.protocol) && str(o.target) && states.includes(String(o.state)) && bool(o.canSftp), 'session.open 세션 정보');
    need(e, o.hostId === null || str(o.hostId), 'hostId 는 문자열 또는 null');
    return e;
  },
  syncStatus(v: unknown): Errs {
    const e: Errs = [];
    need(e, isObj(v) && bool(v.configured) && bool(v.gitAvailable), 'sync.status 필수');
    need(e, isObj(v) && isObj(v.last) && str(v.last.state), 'sync.status.last 는 항상 객체');
    return e;
  },
  syncResult(v: unknown): Errs {
    const e: Errs = []; need(e, isObj(v) && str(v.state), 'sync 결과 state'); return e;
  },
  updateStatus(v: unknown): Errs {
    const e: Errs = []; need(e, v === null || (isObj(v) && str(v.version) && str(v.url)), 'update.status 또는 null'); return e;
  },
  transfer(v: unknown): Errs {
    const e: Errs = [];
    need(e, isObj(v) && str(v.id) && str(v.sessionId) && str(v.name) && (v.direction === 'upload' || v.direction === 'download') && num(v.bytes) && num(v.total) && str(v.state) && str(v.remotePath), 'transfer.update');
    return e;
  },
  uiRequest(v: unknown): Errs {
    const e: Errs = [];
    need(e, isObj(v) && str(v.requestId) && str(v.kind), 'ui.request 공통 필드');
    return e;
  },
  updateAvailable(v: unknown): Errs {
    const e: Errs = []; need(e, isObj(v) && str(v.version) && str(v.url) && bool(v.announce), 'update.available'); return e;
  },
};

describe('고정 계약 JSON 검증기 통과', () => {
  it.each([
    ['app.info', V.appInfo, appInfo],
    ['vault.status', V.vaultStatus, vaultStatus],
    ['vault.status.locked', V.vaultStatus, vaultLocked],
    ['vault.setup', V.setup, vaultSetup],
    ['cred.list', V.credList, credList],
    ['cred.get', V.credSecret, credGet],
    ['cred.save', V.credMeta, credSave],
    ['store.get', V.store, storeGet],
    ['store.get(null 패치 뒤)', V.store, storeAfterNull],
    ['session.open', V.sessionInfo, sessionOpen],
    ['sync.status', V.syncStatus, syncStatus],
    ['update.status', V.updateStatus, updateStatus],
    ['event transfer.update', V.transfer, evTransfer],
    ['event sync.status', V.syncResult, evSync],
    ['event update.available', V.updateAvailable, evUpdate],
    ['event ui.request', V.uiRequest, evUiRequest],
  ])('%s', (_name, check, value) => {
    expect(check(value as unknown)).toEqual([]);
  });

  it('unlock 결과는 불리언', () => {
    expect(typeof vaultUnlock).toBe('boolean');
  });
});

describe('화면·상태 저장소가 계약 응답을 크래시 없이 처리', () => {
  let replies: Array<{ id: string; value: unknown }>;
  beforeEach(() => {
    replies = [];
    const table: Record<string, unknown> = {
      'store.get': storeAfterNull,
      'vault.status': vaultStatus,
      'session.open': sessionOpen,
    };
    const b: Backend = {
      async invoke<T = any>(channel: string): Promise<T> {
        if (channel in table) return table[channel] as T;
        throw new Error(`fixture 없음: ${channel}`);
      },
      termInput() {}, termResize() {},
      reply(id, value) { replies.push({ id, value }); },
      on: () => () => undefined,
    };
    setBackend(b);
  });

  it('store.get 응답의 배열 키가 null 이어도 normalizeStore 가 배열로 채운다', () => {
    const broken = { ...storeAfterNull, hosts: null, snippets: undefined } as unknown as typeof storeAfterNull;
    const s = normalizeStore(broken as never);
    expect(Array.isArray(s.hosts)).toBe(true);
    expect(Array.isArray(s.snippets)).toBe(true);
    expect(Array.isArray(s.groups)).toBe(true);
  });

  it('boot 이 저장 데이터와 보관함 상태를 읽는다', async () => {
    await useApp.getState().boot();
    const st = useApp.getState();
    expect(st.booted).toBe(true);
    expect(Array.isArray(st.store?.hosts)).toBe(true);
    expect(typeof st.vault?.initialized).toBe("boolean");
  });

  it('session.open 결과를 여는 흐름이 세션 정보를 돌려준다', async () => {
    const info = await openSessionFor(null, { host: 'example.test', protocol: 'ssh' });
    expect(info?.id).toBeTruthy();
  });

  it('transfer.update 이벤트 값을 전송 목록에 넣어도 오류가 없다', () => {
    useTransfers.getState().upsert(evTransfer as never);
    expect(useTransfers.getState().items.length).toBeGreaterThan(0);
  });

  it('ui.request 를 큐에 넣고 취소 답을 보내면 requestId 로 Reply 된다', () => {
    useRequests.getState().push(evUiRequest as never);
    expect(useRequests.getState().queue.length).toBe(1);
    useRequests.getState().answer(null);
    expect(replies).toEqual([{ id: (evUiRequest as { requestId: string }).requestId, value: null }]);
  });
});
