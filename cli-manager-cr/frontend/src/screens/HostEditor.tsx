// 서버 편집 대화상자. 근거: spec/ui/host-editor.md §1.1~§1.6.
import { useEffect, useRef, useState } from 'react';
import type { AuthType, CredMeta, EnvKind, Host, Protocol } from '../ipc/types';
import { getBackend, errorMessage } from '../ipc/backend';
import { useApp } from '../state/app';
import { useUi } from '../state/ui';
import { openHost } from '../state/actions';
import { Modal } from '../ui/Modal';
import { ColorPicker, Field, Segmented, Switch } from '../ui/Controls';
import { groupPath, sortGroups } from '../lib/tree';
import { ENV_INFO } from '../lib/defaults';
import {
  fallbackAlias, isPpkV3, isWslShell, JUMP_DIRECT, PPK_V3_MESSAGE, portAfterProtocolChange, serialDefaultsFor,
  SERIAL_DEFAULTS, splitBastionInput, splitHostInput, tmuxPlaceholder, validateHostForm, type SerialOpts,
} from '../lib/hostForm';
import { applyHost, buildHostFromForm } from '../lib/hostSave';
import { ForwardsTable, JumpDirectCard, SerialFields } from './HostEditorParts';

type Tab = 'basic' | 'auth' | 'adv' | 'fwd';

const PROTOCOLS: { value: Protocol; label: string }[] = [
  { value: 'ssh', label: 'SSH' },
  { value: 'telnet', label: 'Telnet' },
  { value: 'serial', label: '시리얼 (COM/RS-485)' },
  { value: 'local', label: '로컬 (이 PC)' },
];
/** 환경 옵션 앞의 색 점(spec §1.1.5). */
function envOption(v: EnvKind, text: string) {
  return (
    <span style={{ display: 'inline-flex', alignItems: 'center', gap: 6 }}>
      <span style={{ width: 7, height: 7, borderRadius: 99, background: ENV_INFO[v].color }} />
      {text}
    </span>
  );
}

const ENVS: { value: EnvKind; label: string }[] = [
  { value: 'none', label: '기본' },
  { value: 'dev', label: '개발' },
  { value: 'stage', label: '검증' },
  { value: 'perf', label: '부하 테스트' },
  { value: 'prod', label: '운영' },
  { value: 'device', label: '네트워크 장비' },
];
const AUTHS: { value: AuthType; label: string }[] = [
  { value: 'password', label: '비밀번호' },
  { value: 'key', label: '키 파일' },
  { value: 'agent', label: 'SSH 에이전트 (Pageant)' },
  { value: 'ask', label: '접속할 때마다 입력' },
];
const ENCODINGS: { value: string; label: string }[] = [
  { value: 'utf-8', label: 'UTF-8' },
  { value: 'cp949', label: 'CP949 / EUC-KR (구형 장비)' },
  { value: 'shift_jis', label: 'Shift_JIS' },
  { value: 'latin1', label: 'Latin-1' },
];

interface Props {
  hostId: string | null;
  groupId: string | null;
}

export function HostEditor({ hostId, groupId }: Props) {
  const store = useApp((s) => s.store);
  const hosts = store?.hosts ?? [];
  const groups = store?.groups ?? [];
  const vaultOpen = useApp((s) => !!s.vault?.unlocked);
  const patchStore = useApp((s) => s.patchStore);
  const refreshStore = useApp((s) => s.refreshStore);
  const closeEditor = useUi((s) => s.closeEditor);
  const original = hostId ? hosts.find((h) => h.id === hostId) ?? null : null;
  const isNew = !original;

  const [tab, setTab] = useState<Tab>('basic');
  const [protocol, setProtocol] = useState<Protocol>(original?.protocol ?? 'ssh');
  const [alias, setAlias] = useState(original?.alias ?? '');
  const [host, setHost] = useState(original?.host ?? '');
  const [port, setPort] = useState(String(original?.port ?? 22));
  const [username, setUsername] = useState(original?.username ?? '');
  const [folder, setFolder] = useState<string>(original?.groupId ?? groupId ?? '');
  const [tags, setTags] = useState((original?.tags ?? []).join(', '));
  const [env, setEnv] = useState<EnvKind>(original?.env ?? 'none');
  const [color, setColor] = useState<string | undefined>(original?.color);
  const [notes, setNotes] = useState(original?.notes ?? '');
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);

  // 인증
  const [authType, setAuthType] = useState<AuthType>(original?.authType ?? 'password');
  const [accountMode, setAccountMode] = useState<'own' | 'shared'>('own');
  const [sharedCredId, setSharedCredId] = useState<string | null>(null);
  const [creds, setCreds] = useState<CredMeta[]>([]);
  const [password, setPassword] = useState('');
  const [privateKey, setPrivateKey] = useState('');
  const [keyStatus, setKeyStatus] = useState('');
  const [passphrase, setPassphrase] = useState('');
  const modeDecided = useRef(false);

  // 고급
  const [encoding, setEncoding] = useState<string>(original?.encoding ?? 'utf-8');
  const [termType, setTermType] = useState(original?.termType ?? 'xterm-256color');
  const [keepalive, setKeepalive] = useState(String(original?.keepaliveSec ?? 30));
  const [startup, setStartup] = useState(original?.startupCommand ?? '');
  const [keepSession, setKeepSession] = useState<boolean | undefined>(original?.keepSession);
  const [tmuxName, setTmuxName] = useState(original?.tmuxName ?? '');
  const [tmuxAvail, setTmuxAvail] = useState(false);
  const [jumpChoice, setJumpChoice] = useState<string>(original?.jumpHostId ?? '');
  const [jump, setJump] = useState(() => {
    const d = original?.jumpDirect as { host?: string; port?: number; username?: string; authType?: 'password' | 'key' } | undefined;
    return {
      address: d?.host ? `${d.username ? d.username + '@' : ''}${d.host}${d.port && d.port !== 22 ? ':' + d.port : ''}` : '',
      port: d?.port && d.port !== 22 ? String(d.port) : '',
      username: d?.username ?? '',
      authType: (d?.authType ?? 'password') as 'password' | 'key',
      password: '',
      keyText: '',
      keyStatus: '',
      passphrase: '',
    };
  });
  const jumpHasSaved = !!(original?.jumpDirect as { credentialId?: string | null } | undefined)?.credentialId;

  // 포워딩·시리얼·로컬
  const [forwards, setForwards] = useState(original?.forwards ?? []);
  const [serial, setSerial] = useState<SerialOpts>({ ...SERIAL_DEFAULTS, ...(original?.serial as Partial<SerialOpts> | undefined) });
  const [ports, setPorts] = useState<{ path: string; label: string }[]>([]);
  const [localCwd, setLocalCwd] = useState(original?.local?.cwd ?? '');
  const [localShell, setLocalShell] = useState(original?.local?.shell ?? '');
  const [shells, setShells] = useState<{ path: string; label: string }[]>([]);

  const isSsh = protocol === 'ssh';
  const isLocal = protocol === 'local';
  const isSerial = protocol === 'serial';
  const isNet = protocol === 'ssh' || protocol === 'telnet';
  const isWindows = typeof navigator !== 'undefined' && /Windows/i.test(navigator.userAgent);
  const tmuxCardShown = isSsh || (isLocal && (!isWindows || isWslShell(localShell)));
  const tmuxOn = keepSession ?? (isNew && isLocal && tmuxAvail);

  // 공유 계정 시작 규칙(§1.2): 계정 이름이 별칭과 다르고 같은 계정을 쓰는 서버가 2개 이상.
  useEffect(() => {
    const b = getBackend();
    void b.invoke<CredMeta[]>('cred.list').then((list) => {
      setCreds(list ?? []);
      if (!modeDecided.current && original?.credentialId) {
        modeDecided.current = true;
        const cred = (list ?? []).find((c) => c.id === original.credentialId);
        const users = hosts.filter((h) => h.credentialId === original.credentialId).length;
        if (cred && cred.name !== original.alias && users >= 2) {
          setAccountMode('shared');
          setSharedCredId(original.credentialId);
        }
      }
    }).catch(() => undefined);
    void b.invoke<{ path: string; label: string }[]>('local.shells').then((l) => setShells(l ?? [])).catch(() => undefined);
    if (isNew) void b.invoke<boolean>('local.tmuxAvailable').then((v) => setTmuxAvail(!!v)).catch(() => undefined);
    if (protocol === 'serial') refreshPorts();
    // 최초 1회만 읽는다.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const refreshPorts = () => {
    void getBackend().invoke<{ path: string; label: string }[]>('serial.list').then((l) => setPorts(l ?? [])).catch(() => undefined);
  };

  // 프로토콜 전환 규칙(§1.1 1·5번).
  const changeProtocol = (next: Protocol) => {
    setProtocol(next);
    const p = Number(port) || 0;
    setPort(String(portAfterProtocolChange(next, p)));
    if (next === 'serial') {
      const r = serialDefaultsFor(serial, env);
      setSerial(r.serial);
      setEnv(r.env ?? 'none');
      refreshPorts();
    }
    if (next !== 'ssh' && tab === 'auth') setTab('basic');
    if (next !== 'ssh' && tab === 'fwd') setTab('basic');
  };

  // 호스트 칸 입력 나누기(§1.1).
  const onHostChange = (v: string) => {
    const r = splitHostInput(v);
    if (r.username !== undefined) setUsername(r.username);
    setHost(r.host);
    if (r.port !== undefined) setPort(r.port);
  };

  const pickDir = async () => {
    try {
      const p = await getBackend().invoke<string | null>('dialog.chooseDir');
      if (p) setLocalCwd(p);
    } catch (e) {
      setError(errorMessage(e));
    }
  };

  const pickKey = async (target: 'own' | 'jump') => {
    try {
      const r = await getBackend().invoke<{ path: string; text: string } | null>('dialog.readKeyFile');
      if (!r) return;
      if (isPpkV3(r.text)) {
        setError(PPK_V3_MESSAGE);
        return;
      }
      if (target === 'own') {
        setPrivateKey(r.text);
        setKeyStatus(r.path);
      } else {
        setJump((j) => ({ ...j, keyText: r.text, keyStatus: '키를 불러왔습니다' }));
      }
    } catch (e) {
      setError(errorMessage(e));
    }
  };

  // 검사 실패 시 해당 탭으로 이동하고 오류를 보인다(§1.5 4번).
  const fail = (msg: string, t: Tab) => {
    setError(msg);
    setTab(t);
  };

  const save = async (connect = false): Promise<void> => {
    setError('');
    const aliasFinal = fallbackAlias(alias, protocol, { host, serialPath: serial.path, localCwd });
    const hasSecret = accountMode === 'own' && ((authType === 'password' && !!password) || (authType === 'key' && !!privateKey.trim()));
    const jumpDirect = isSsh && jumpChoice === JUMP_DIRECT;
    const bad = validateHostForm({
      protocol,
      host,
      serialPath: serial.path,
      authType,
      hasSecret,
      vaultOpen,
      jumpDirect,
      jumpAddress: jump.address,
      jumpHasSecret: (jump.authType === 'password' && !!jump.password) || (jump.authType === 'key' && !!jump.keyText.trim()),
    });
    if (bad) return fail(bad.message, bad.tab);

    setBusy(true);
    try {
      const b = isNet && jumpDirect ? splitBastionInput(jump.address) : null;
      const next = await buildHostFromForm({
        original,
        hostId,
        allHosts: hosts,
        protocol,
        alias: aliasFinal,
        host,
        port: Number(port) || 0,
        username,
        folderId: folder || null,
        tags: tags.split(',').map((t) => t.trim()).filter(Boolean),
        env,
        color,
        notes,
        authType,
        accountMode,
        sharedCredId,
        password,
        privateKey,
        passphrase,
        encoding: encoding as Host['encoding'],
        termType,
        keepaliveSec: Number(keepalive) || 0,
        startupCommand: startup,
        keepSession: !!tmuxOn,
        tmuxName,
        forwards,
        serial,
        localShell,
        localCwd,
        jumpChoice,
        jump: {
          host: b?.host ?? '',
          port: Number(b?.port ?? jump.port) || 22,
          username: b?.username ?? jump.username,
          authType: jump.authType,
          password: jump.password,
          keyText: jump.keyText,
          passphrase: jump.passphrase,
        },
      });
      await patchStore({ hosts: applyHost(hosts, next, !!hostId) });
      await refreshStore();
      closeEditor();
      if (connect) void openHost(next.id);
    } catch (e) {
      setError(errorMessage(e));
    } finally {
      setBusy(false);
    }
  };

  const folders = sortGroups(groups);
  const tabs: { id: Tab; label: string }[] = [
    { id: 'basic', label: '일반' },
    ...(isSsh ? [{ id: 'auth' as Tab, label: '로그인' }] : []),
    { id: 'adv', label: '고급' },
    ...(isSsh ? [{ id: 'fwd' as Tab, label: '포트 포워딩' }] : []),
  ];
  const curTab = tabs.some((t) => t.id === tab) ? tab : 'basic';
  const shellOptions = [
    { value: '', label: `기본 (${shells[0]?.label ?? '기본 셸'})` },
    ...shells.map((s) => ({ value: s.path, label: s.label })),
  ];
  const credOptions = creds.map((c) => ({ value: c.id, label: `${c.name}${c.username ? ` (${c.username})` : ''} · ${c.kind === 'key' ? '키' : '비밀번호'}` }));

  return (
    <Modal
      title={original ? `서버 수정 · ${original.alias}` : '서버 추가'}
      size="md"
      onClose={closeEditor}
      foot={
        <>
          <div className="err-text" style={{ marginRight: 'auto' }}>{error}</div>
          <button className="btn" onClick={closeEditor}>취소</button>
          <button className="btn primary" disabled={busy} onClick={() => void save()}>저장</button>
          <button className="btn primary" disabled={busy} onClick={() => void save(true)}>저장하고 연결</button>
        </>
      }
    >
      <div style={{ display: 'flex', gap: 4, margin: '-4px -18px 0', padding: '0 18px', borderBottom: '1px solid var(--border)' }} role="tablist">
        {tabs.map((t) => (
          <button
            key={t.id}
            role="tab"
            aria-selected={curTab === t.id}
            onClick={() => setTab(t.id)}
            className="btn link"
            style={{
              height: 34,
              borderRadius: 0,
              borderBottom: curTab === t.id ? '2px solid var(--accent)' : '2px solid transparent',
              color: curTab === t.id ? 'var(--fg)' : 'var(--fg-2)',
              padding: '0 10px',
            }}
          >
            {t.label}
          </button>
        ))}
      </div>

      <div style={{ minHeight: 380, display: 'flex', flexDirection: 'column', gap: 14 }}>
        {curTab === 'basic' && (
          <>
            <Field label="프로토콜">
              <Segmented value={protocol} options={PROTOCOLS} onChange={changeProtocol} label="프로토콜" />
            </Field>
            <Field label="표시 이름 (탭과 목록에 보임)">
              <input className="input" autoFocus placeholder="예: 운영-WEB01" value={alias} onChange={(e) => setAlias(e.target.value)} />
            </Field>
            {isLocal && (
              <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 14 }}>
                <Field label="시작 폴더" help="비워 두면 홈 폴더에서 열립니다">
                  <div style={{ display: 'flex', gap: 6 }}>
                    <input className="input mono" style={{ flex: 1 }} placeholder="C:\Work\프로젝트" value={localCwd} onChange={(e) => setLocalCwd(e.target.value)} />
                    <button type="button" className="icon-btn" title="폴더 선택" aria-label="폴더 선택" onClick={() => void pickDir()}>…</button>
                  </div>
                </Field>
                <Field label="셸">
                  <select className="input" value={localShell} onChange={(e) => setLocalShell(e.target.value)}>
                    {shellOptions.map((o) => <option key={o.value} value={o.value}>{o.label}</option>)}
                  </select>
                </Field>
              </div>
            )}
            {isNet && (
              <div style={{ display: 'grid', gridTemplateColumns: '2fr 1fr 1fr', gap: 14 }}>
                <Field label="주소 (IP 또는 도메인)">
                  <input className="input mono" placeholder="10.0.0.11  또는  user@host:22" value={host} onChange={(e) => onHostChange(e.target.value)} />
                </Field>
                <Field label="포트">
                  <input className="input" inputMode="numeric" value={port} onChange={(e) => setPort(e.target.value.replace(/\D/g, ''))} />
                </Field>
                <Field label="사용자 이름">
                  <input className="input" placeholder="root" value={username} onChange={(e) => setUsername(e.target.value)} />
                </Field>
              </div>
            )}
            {isSerial && (
              <SerialFields value={serial} ports={ports} onChange={setSerial} onRefresh={refreshPorts} />
            )}
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 14 }}>
              <Field label="폴더">
                <select className="input" value={folder} onChange={(e) => setFolder(e.target.value)}>
                  <option value="">(최상위)</option>
                  {folders.map((g) => (
                    <option key={g.id} value={g.id}>{groupPath(groups, g.id)}</option>
                  ))}
                </select>
              </Field>
              <Field label="태그 (쉼표로 구분)">
                <input className="input" placeholder="web, nginx" value={tags} onChange={(e) => setTags(e.target.value)} />
              </Field>
            </div>
            <Field label="환경" help="운영 서버는 빨간 테두리로 표시되고 위험 명령 실행 전에 확인을 받습니다">
              <Segmented value={env} options={ENVS.map((o) => ({ value: o.value, label: envOption(o.value, o.label) }))} onChange={setEnv} label="환경" />
            </Field>
            <Field label="색상 (지정하지 않으면 환경/폴더 색을 따름)">
              <ColorPicker value={color} onChange={setColor} />
            </Field>
            <Field label="메모">
              <textarea className="input" rows={2} style={{ height: 'auto', padding: 8 }} value={notes} onChange={(e) => setNotes(e.target.value)} />
            </Field>
          </>
        )}
        {curTab === 'auth' && isSsh && (
          <>
            {!vaultOpen && <div className="notice warn">앱이 잠겨 있습니다. 비밀번호나 키를 저장하려면 먼저 잠금을 해제하세요.</div>}
            <Field label="인증 방식">
              <Segmented value={authType} options={AUTHS} onChange={setAuthType} label="인증 방식" />
            </Field>
            {(authType === 'password' || authType === 'key') && (
              <Field label="계정 저장 방식">
                <Segmented
                  value={accountMode}
                  options={[{ value: 'own', label: '이 서버에만 저장' }, { value: 'shared', label: '저장된 계정 사용' }]}
                  onChange={setAccountMode}
                  label="계정 저장 방식"
                />
              </Field>
            )}
            {accountMode === 'shared' && (authType === 'password' || authType === 'key') && (
              <Field label="저장된 계정" help="같은 계정을 쓰는 서버 여러 대가 하나의 비밀번호를 공유합니다. 비밀번호를 바꾸면 모두 적용됩니다.">
                <select className="input" value={sharedCredId ?? ''} onChange={(e) => setSharedCredId(e.target.value || null)}>
                  <option value="">선택하세요</option>
                  {credOptions.map((o) => <option key={o.value} value={o.value}>{o.label}</option>)}
                </select>
              </Field>
            )}
            {accountMode === 'own' && authType === 'password' && (
              <Field
                label="비밀번호"
                help={original?.credentialId ? '저장된 비밀번호가 있습니다. 바꾸려면 새로 입력하세요.' : '비워 두면 처음 연결할 때 묻고, 저장 여부를 선택할 수 있습니다.'}
              >
                <input className="input" type="password" placeholder={original?.credentialId ? '••••••••  (저장됨)' : '비밀번호'} value={password} onChange={(e) => setPassword(e.target.value)} />
              </Field>
            )}
            {accountMode === 'own' && authType === 'key' && (
              <>
                <Field label="키 파일">
                  <div style={{ display: 'flex', gap: 10, alignItems: 'center' }}>
                    <button type="button" className="btn small-btn" onClick={() => void pickKey('own')}>키 파일 선택</button>
                    <span className="faint small">{keyStatus || (original?.credentialId ? '저장된 키 사용 중' : '선택된 키 없음')}</span>
                  </div>
                </Field>
                <Field label="또는 키 내용 붙여넣기" help="OpenSSH, PEM, PuTTY PPK v2 지원. 키 내용은 암호화되어 저장됩니다.">
                  <textarea className="input mono" rows={4} style={{ height: 'auto', padding: 8 }} placeholder="-----BEGIN OPENSSH PRIVATE KEY-----" value={privateKey} onChange={(e) => setPrivateKey(e.target.value)} />
                </Field>
                <Field label="키 암호 (있으면)">
                  <input className="input" type="password" value={passphrase} onChange={(e) => setPassphrase(e.target.value)} />
                </Field>
              </>
            )}
            {authType === 'agent' && (
              <div className="notice info">Windows에서는 Pageant, 그 외에는 SSH_AUTH_SOCK 에이전트를 사용합니다. 에이전트 인증이 실패하면 비밀번호를 묻습니다.</div>
            )}
            {authType === 'ask' && (
              <div className="notice info">연결할 때마다 비밀번호를 묻습니다. 입력 창에서 "저장"을 선택하면 다음부터 자동으로 로그인됩니다.</div>
            )}
          </>
        )}

        {curTab === 'adv' && (
          <>
            {isSsh && (
              <Field label="경유 서버 (ProxyJump)" help="이 서버에 연결하기 전에 먼저 거쳐 갈 서버입니다 (ProxyJump).">
                <select className="input" value={jumpChoice} onChange={(e) => setJumpChoice(e.target.value)}>
                  <option value="">없음</option>
                  <option value={JUMP_DIRECT}>직접 입력…</option>
                  {hosts.filter((h) => h.id !== hostId && h.protocol === 'ssh').map((h) => (
                    <option key={h.id} value={h.id}>{`${h.alias} (${h.host})`}</option>
                  ))}
                </select>
              </Field>
            )}
            {isSsh && jumpChoice === JUMP_DIRECT && (
              <JumpDirectCard
                address={jump.address}
                onAddress={(v) => setJump((j) => ({ ...j, address: v, ...(() => { const r = splitBastionInput(v); return r.username !== undefined || r.port !== undefined ? { username: r.username ?? j.username, port: r.port ?? j.port } : {}; })() }))}
                port={jump.port}
                onPort={(v) => setJump((j) => ({ ...j, port: v }))}
                username={jump.username}
                onUsername={(v) => setJump((j) => ({ ...j, username: v }))}
                authType={jump.authType}
                onAuthType={(v) => setJump((j) => ({ ...j, authType: v }))}
                password={jump.password}
                onPassword={(v) => setJump((j) => ({ ...j, password: v }))}
                keyText={jump.keyText}
                onPickKey={() => void pickKey('jump')}
                passphrase={jump.passphrase}
                onPassphrase={(v) => setJump((j) => ({ ...j, passphrase: v }))}
                hasSaved={jumpHasSaved}
                keyStatus={jump.keyStatus}
              />
            )}
            {tmuxCardShown && (
              <div className="toggle-row">
                <div>
                  <div className="t-title">연결 유지 (tmux)</div>
                  <div className="t-desc">
                    {'창이나 앱을 닫아도 tmux 안의 프로그램은 계속 실행되고, 다시 열면 그 화면에 그대로 붙습니다. '}
                    {isLocal && isWindows ? 'Windows 로컬은 WSL 셸일 때만 지원합니다.' : '대상에 tmux 가 있어야 합니다.'}
                  </div>
                </div>
                <Switch checked={tmuxOn} onChange={(v) => setKeepSession(v)} label="연결 유지 (tmux)" />
              </div>
            )}
            {tmuxCardShown && tmuxOn && (
              <Field label="tmux 세션 이름" help="비워 두면 서버 표시 이름(또는 로컬 폴더 이름)으로 만듭니다">
                <input className="input mono" placeholder={tmuxPlaceholder(alias, protocol, localCwd)} value={tmuxName} onChange={(e) => setTmuxName(e.target.value)} />
              </Field>
            )}
            <Field
              label={isLocal ? '시작할 때 실행할 명령' : '연결 후 실행할 명령'}
              help={isLocal ? '예: claude --continue' : '예: cd /var/www && sudo -i'}
            >
              <input className="input mono" placeholder={isLocal ? 'claude --continue' : 'cd /var/www && sudo -i'} value={startup} onChange={(e) => setStartup(e.target.value)} />
            </Field>
            {isNet && (
              <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 1fr', gap: 14 }}>
                <Field label="문자 인코딩">
                  <select className="input" value={encoding} onChange={(e) => setEncoding(e.target.value)}>
                    {ENCODINGS.map((o) => <option key={o.value} value={o.value}>{o.label}</option>)}
                    {encoding === 'euc-kr' && <option value="euc-kr">CP949 / EUC-KR (구형 장비)</option>}
                  </select>
                </Field>
                <Field label="터미널 종류">
                  <input className="input mono" value={termType} onChange={(e) => setTermType(e.target.value)} />
                </Field>
                <Field label="연결 확인 간격 (초)">
                  <input className="input" inputMode="numeric" value={keepalive} onChange={(e) => setKeepalive(e.target.value.replace(/\D/g, ''))} />
                </Field>
              </div>
            )}
            {isLocal && (
              <Field label="터미널 종류">
                <input className="input mono" value={termType} onChange={(e) => setTermType(e.target.value)} />
              </Field>
            )}
          </>
        )}

        {curTab === 'fwd' && isSsh && <ForwardsTable rows={forwards} onChange={setForwards} />}
      </div>
    </Modal>
  );
}
