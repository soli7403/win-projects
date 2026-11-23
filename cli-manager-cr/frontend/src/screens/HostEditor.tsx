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
  fallbackAlias, isWslShell, JUMP_DIRECT, portAfterProtocolChange, serialDefaultsFor,
  SERIAL_DEFAULTS, splitBastionInput, splitHostInput, tmuxPlaceholder, validateHostForm, type ProxyConf, type ProxyType, type SerialOpts,
} from '../lib/hostForm';
import { applyHost, buildHostFromForm } from '../lib/hostSave';
import { ForwardsTable, JumpDirectCard, ProxyCard, SerialFields } from './HostEditorParts';
import { t } from '../i18n';

type Tab = 'basic' | 'auth' | 'adv' | 'fwd';

// 선택 목록은 그릴 때 언어에 맞춰 만든다(D75).
const PROTOCOLS = (): { value: Protocol; label: string }[] => [
  { value: 'ssh', label: 'SSH' },
  { value: 'telnet', label: 'Telnet' },
  { value: 'serial', label: t('proto.serial') },
  { value: 'local', label: t('proto.local') },
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

const ENVS = (): { value: EnvKind; label: string }[] => [
  { value: 'none', label: t('env.none') },
  { value: 'dev', label: t('env.dev') },
  { value: 'stage', label: t('env.stage') },
  { value: 'perf', label: t('env.perf') },
  { value: 'prod', label: t('env.prod') },
  { value: 'device', label: t('env.device') },
];
const AUTHS = (): { value: AuthType; label: string }[] => [
  { value: 'password', label: t('cred.password') },
  { value: 'key', label: t('host.keyFile') },
  { value: 'agent', label: t('auth.agent') },
  { value: 'ask', label: t('auth.ask') },
];
const ENCODINGS = (): { value: string; label: string }[] => [
  { value: 'utf-8', label: 'UTF-8' },
  { value: 'cp949', label: t('enc.cp949') },
  { value: 'shift_jis', label: 'Shift_JIS' },
  { value: 'euc-jp', label: 'EUC-JP' },
  { value: 'gbk', label: 'GBK' },
  { value: 'big5', label: 'Big5' },
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
  // 프록시 경유(§1.3 2-1번)
  const [proxy, setProxy] = useState(() => {
    const p = original?.proxy as Partial<ProxyConf> | undefined;
    return {
      type: (p?.type === 'http' || p?.type === 'socks5' ? p.type : '') as ProxyType | '',
      host: p?.host ?? '',
      port: p?.port ? String(p.port) : '',
      username: p?.username ?? '',
      password: '',
    };
  });
  const [agentForward, setAgentForward] = useState(!!original?.agentForward);
  const proxyHasSaved = !!(original?.proxy as ProxyConf | undefined)?.credentialId;

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
      if (target === 'own') {
        setPrivateKey(r.text);
        setKeyStatus(r.path);
      } else {
        setJump((j) => ({ ...j, keyText: r.text, keyStatus: t('host.keyLoaded') }));
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
      proxyOn: isSsh && !!proxy.type,
      proxyHost: proxy.host,
      proxyHasSecret: !!proxy.username.trim() && !!proxy.password,
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
        proxy: { type: proxy.type, host: proxy.host, port: Number(proxy.port) || 0, username: proxy.username, password: proxy.password },
        agentForward,
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
    { id: 'basic', label: t('host.tabBasic') },
    ...(isSsh ? [{ id: 'auth' as Tab, label: t('host.tabAuth') }] : []),
    { id: 'adv', label: t('host.tabAdv') },
    ...(isSsh ? [{ id: 'fwd' as Tab, label: t('host.tabFwd') }] : []),
  ];
  const curTab = tabs.some((t) => t.id === tab) ? tab : 'basic';
  const shellOptions = [
    { value: '', label: t('host.defaultShell', { name: shells[0]?.label ?? t('host.defaultShellName') }) },
    ...shells.map((s) => ({ value: s.path, label: s.label })),
  ];
  const credOptions = creds.map((c) => ({ value: c.id, label: `${c.name}${c.username ? ` (${c.username})` : ''} · ${c.kind === 'key' ? t('cred.key') : t('cred.password')}` }));

  return (
    <Modal
      title={original ? t('host.editTitle', { name: original.alias }) : t('sb.addServer')}
      size="md"
      onClose={closeEditor}
      foot={
        <>
          <div className="err-text" style={{ marginRight: 'auto' }}>{error}</div>
          <button className="btn" onClick={closeEditor}>{t('common.cancel')}</button>
          <button className="btn primary" disabled={busy} onClick={() => void save()}>{t('common.save')}</button>
          <button className="btn primary" disabled={busy} onClick={() => void save(true)}>{t('host.saveConnect')}</button>
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
            <Field label={t('host.protocol')}>
              <Segmented value={protocol} options={PROTOCOLS()} onChange={changeProtocol} label={t('host.protocol')} />
            </Field>
            <Field label={t('host.alias')}>
              <input className="input" autoFocus placeholder={t('host.aliasPlaceholder')} value={alias} onChange={(e) => setAlias(e.target.value)} />
            </Field>
            {isLocal && (
              <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 14 }}>
                <Field label={t('host.startDir')} help={t('host.startDirHelp')}>
                  <div style={{ display: 'flex', gap: 6 }}>
                    <input className="input mono" style={{ flex: 1 }} placeholder={t('host.startDirPlaceholder')} value={localCwd} onChange={(e) => setLocalCwd(e.target.value)} />
                    <button type="button" className="icon-btn" title={t('host.pickDir')} aria-label={t('host.pickDir')} onClick={() => void pickDir()}>…</button>
                  </div>
                </Field>
                <Field label={t('host.shell')}>
                  <select className="input" value={localShell} onChange={(e) => setLocalShell(e.target.value)}>
                    {shellOptions.map((o) => <option key={o.value} value={o.value}>{o.label}</option>)}
                  </select>
                </Field>
              </div>
            )}
            {isNet && (
              <div style={{ display: 'grid', gridTemplateColumns: '2fr 1fr 1fr', gap: 14 }}>
                <Field label={t('host.address')}>
                  <input className="input mono" placeholder={t('host.addressPlaceholder')} value={host} onChange={(e) => onHostChange(e.target.value)} />
                </Field>
                <Field label={t('fwd.port')}>
                  <input className="input" inputMode="numeric" value={port} onChange={(e) => setPort(e.target.value.replace(/\D/g, ''))} />
                </Field>
                <Field label={t('host.username')}>
                  <input className="input" placeholder="root" value={username} onChange={(e) => setUsername(e.target.value)} />
                </Field>
              </div>
            )}
            {isSerial && (
              <SerialFields value={serial} ports={ports} onChange={setSerial} onRefresh={refreshPorts} />
            )}
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 14 }}>
              <Field label={t('host.folder')}>
                <select className="input" value={folder} onChange={(e) => setFolder(e.target.value)}>
                  <option value="">{t('folder.root')}</option>
                  {folders.map((g) => (
                    <option key={g.id} value={g.id}>{groupPath(groups, g.id)}</option>
                  ))}
                </select>
              </Field>
              <Field label={t('host.tags')}>
                <input className="input" placeholder="web, nginx" value={tags} onChange={(e) => setTags(e.target.value)} />
              </Field>
            </div>
            <Field label={t('folder.env')} help={t('host.envHelp')}>
              <Segmented value={env} options={ENVS().map((o) => ({ value: o.value, label: envOption(o.value, o.label) }))} onChange={setEnv} label={t('folder.env')} />
            </Field>
            <Field label={t('host.color')}>
              <ColorPicker value={color} onChange={setColor} />
            </Field>
            <Field label={t('host.memo')}>
              <textarea className="input" rows={2} style={{ height: 'auto', padding: 8 }} value={notes} onChange={(e) => setNotes(e.target.value)} />
            </Field>
          </>
        )}
        {curTab === 'auth' && isSsh && (
          <>
            {!vaultOpen && <div className="notice warn">{t('host.lockedNotice')}</div>}
            <Field label={t('host.authMethod')}>
              <Segmented value={authType} options={AUTHS()} onChange={setAuthType} label={t('host.authMethod')} />
            </Field>
            {(authType === 'password' || authType === 'key') && (
              <Field label={t('host.credMode')}>
                <Segmented
                  value={accountMode}
                  options={[{ value: 'own', label: t('host.credOwn') }, { value: 'shared', label: t('host.credShared') }]}
                  onChange={setAccountMode}
                  label={t('host.credMode')}
                />
              </Field>
            )}
            {accountMode === 'shared' && (authType === 'password' || authType === 'key') && (
              <Field label={t('host.savedCred')} help={t('host.savedCredHelp')}>
                <select className="input" value={sharedCredId ?? ''} onChange={(e) => setSharedCredId(e.target.value || null)}>
                  <option value="">{t('host.choose')}</option>
                  {credOptions.map((o) => <option key={o.value} value={o.value}>{o.label}</option>)}
                </select>
              </Field>
            )}
            {accountMode === 'own' && authType === 'password' && (
              <Field
                label={t('cred.password')}
                help={original?.credentialId ? t('host.pwSavedHelp') : t('host.pwEmptyHelp')}
              >
                <input className="input" type="password" placeholder={original?.credentialId ? t('host.savedMask') : t('cred.password')} value={password} onChange={(e) => setPassword(e.target.value)} />
              </Field>
            )}
            {accountMode === 'own' && authType === 'key' && (
              <>
                <Field label={t('host.keyFile')}>
                  <div style={{ display: 'flex', gap: 10, alignItems: 'center' }}>
                    <button type="button" className="btn small-btn" onClick={() => void pickKey('own')}>{t('host.pickKey')}</button>
                    <span className="faint small">{keyStatus || (original?.credentialId ? t('host.keyInUse') : t('host.noKey'))}</span>
                  </div>
                </Field>
                <Field label={t('host.keyPaste')} help={t('host.keyPasteHelp')}>
                  <textarea className="input mono" rows={4} style={{ height: 'auto', padding: 8 }} placeholder="-----BEGIN OPENSSH PRIVATE KEY-----" value={privateKey} onChange={(e) => setPrivateKey(e.target.value)} />
                </Field>
                <Field label={t('host.passphrase')}>
                  <input className="input" type="password" value={passphrase} onChange={(e) => setPassphrase(e.target.value)} />
                </Field>
              </>
            )}
            {authType === 'agent' && (
              <div className="notice info">{t('host.agentNote')}</div>
            )}
            {authType === 'ask' && (
              <div className="notice info">{t('host.askNote')}</div>
            )}
          </>
        )}

        {curTab === 'adv' && (
          <>
            {isSsh && (
              <Field label={t('host.jump')} help={t('host.jumpHelp')}>
                <select className="input" value={jumpChoice} onChange={(e) => setJumpChoice(e.target.value)}>
                  <option value="">{t('host.none')}</option>
                  <option value={JUMP_DIRECT}>{t('host.jumpManual')}</option>
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
            {isSsh && (
              <Field label={t('host.proxy')} help={t('host.proxyHelp')}>
                <select className="input" value={proxy.type} onChange={(e) => setProxy((p) => ({ ...p, type: e.target.value as ProxyType | '' }))}>
                  <option value="">{t('host.none')}</option>
                  <option value="http">{t('proxy.http')}</option>
                  <option value="socks5">{t('proxy.socks5')}</option>
                </select>
              </Field>
            )}
            {isSsh && !!proxy.type && (
              <ProxyCard
                type={proxy.type}
                host={proxy.host}
                onHost={(v) => setProxy((p) => ({ ...p, host: v }))}
                port={proxy.port}
                onPort={(v) => setProxy((p) => ({ ...p, port: v }))}
                username={proxy.username}
                onUsername={(v) => setProxy((p) => ({ ...p, username: v }))}
                password={proxy.password}
                onPassword={(v) => setProxy((p) => ({ ...p, password: v }))}
                hasSaved={proxyHasSaved}
                jumpInUse={!!jumpChoice}
              />
            )}
            {isSsh && (
              <div className="toggle-row">
                <div>
                  <div className="t-title">{t('host.agentFwd')}</div>
                  <div className="t-desc">{t('host.agentFwdDesc')}</div>
                </div>
                <Switch checked={agentForward} onChange={setAgentForward} label={t('host.agentFwd')} />
              </div>
            )}
            {tmuxCardShown && (
              <div className="toggle-row">
                <div>
                  <div className="t-title">{t('host.tmux')}</div>
                  <div className="t-desc">
                    {t('host.tmuxDesc')}
                    {isLocal && isWindows ? t('host.tmuxWsl') : t('host.tmuxNeed')}
                  </div>
                </div>
                <Switch checked={tmuxOn} onChange={(v) => setKeepSession(v)} label={t('host.tmux')} />
              </div>
            )}
            {tmuxCardShown && tmuxOn && (
              <Field label={t('host.tmuxName')} help={t('host.tmuxNameHelp')}>
                <input className="input mono" placeholder={tmuxPlaceholder(alias, protocol, localCwd)} value={tmuxName} onChange={(e) => setTmuxName(e.target.value)} />
              </Field>
            )}
            <Field
              label={isLocal ? t('host.startCmd') : t('host.afterCmd')}
              help={isLocal ? t('host.startCmdEx') : t('host.afterCmdEx')}
            >
              <input className="input mono" placeholder={isLocal ? 'claude --continue' : 'cd /var/www && sudo -i'} value={startup} onChange={(e) => setStartup(e.target.value)} />
            </Field>
            {isNet && (
              <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 1fr', gap: 14 }}>
                <Field label={t('host.encoding')}>
                  <select className="input" value={encoding} onChange={(e) => setEncoding(e.target.value)}>
                    {ENCODINGS().map((o) => <option key={o.value} value={o.value}>{o.label}</option>)}
                    {encoding === 'euc-kr' && <option value="euc-kr">{t('enc.cp949')}</option>}
                  </select>
                </Field>
                <Field label={t('host.termType')}>
                  <input className="input mono" value={termType} onChange={(e) => setTermType(e.target.value)} />
                </Field>
                <Field label={t('host.keepalive')}>
                  <input className="input" inputMode="numeric" value={keepalive} onChange={(e) => setKeepalive(e.target.value.replace(/\D/g, ''))} />
                </Field>
              </div>
            )}
            {isLocal && (
              <Field label={t('host.termType')}>
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
