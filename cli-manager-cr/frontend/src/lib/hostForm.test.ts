// 수용 시나리오 H1·H2·H3·H5·H6·H7·H9·H10 의 규칙 부분. 근거: spec/acceptance/04-hosts-sftp-settings-sync.md.
import { describe, expect, it } from 'vitest';
import {
  canSaveFolder,
  fallbackAlias,
  isPpkV3,
  portAfterProtocolChange,
  serialDefaultsFor,
  SERIAL_DEFAULTS,
  splitBastionInput,
  splitHostInput,
  validateHostForm,
} from './hostForm';

describe('서버 편집 규칙', () => {
  it('H1: 텔넷으로 바꾸면 포트 22 → 23, SSH 로 돌아오면 23 → 22', () => {
    expect(portAfterProtocolChange('telnet', 22)).toBe(23);
    expect(portAfterProtocolChange('ssh', 23)).toBe(22);
    expect(portAfterProtocolChange('telnet', 2323)).toBe(2323);
  });

  it('H2: user@host:port 입력을 사용자·호스트·포트로 나눈다', () => {
    expect(splitHostInput('root@10.0.0.9:2200')).toEqual({ host: '10.0.0.9', port: '2200', username: 'root' });
    expect(splitHostInput('10.0.0.11')).toEqual({ host: '10.0.0.11' });
  });

  it('H3: 호스트를 비우고 저장하면 기본 탭에서 오류', () => {
    const r = validateHostForm({
      protocol: 'ssh', host: '  ', serialPath: '', authType: 'password', hasSecret: false,
      vaultOpen: true, jumpDirect: false, jumpAddress: '', jumpHasSecret: false,
    });
    expect(r).toEqual({ tab: 'basic', message: '호스트 주소를 입력하세요' });
  });

  it('H4: 보관함이 잠겨 있고 비밀번호가 있으면 인증 탭에서 막는다', () => {
    const r = validateHostForm({
      protocol: 'ssh', host: '10.0.0.1', serialPath: '', authType: 'password', hasSecret: true,
      vaultOpen: false, jumpDirect: false, jumpAddress: '', jumpHasSecret: false,
    });
    expect(r?.tab).toBe('auth');
    expect(r?.message).toBe('보관함이 잠겨 있어 비밀번호를 저장할 수 없습니다');
  });

  it('H5: 표시 이름을 비우면 호스트 주소가 표시 이름이 된다', () => {
    expect(fallbackAlias('', 'ssh', { host: '10.0.0.5' })).toBe('10.0.0.5');
    expect(fallbackAlias('  ', 'serial', { host: '', serialPath: 'COM4' })).toBe('COM4');
    expect(fallbackAlias('', 'local', { host: '', localCwd: 'C:\\Work\\프로젝트' })).toBe('프로젝트');
    expect(fallbackAlias('', 'local', { host: '' })).toBe('로컬 터미널');
  });

  it('H6: 시리얼로 바꾸면 환경이 장비가 되고 기본값 COM1 9600 8 none 1', () => {
    const r = serialDefaultsFor(undefined, 'none');
    expect(r.env).toBe('device');
    expect(r.serial).toEqual(SERIAL_DEFAULTS);
    expect(r.serial).toMatchObject({ path: 'COM1', baudRate: 9600, dataBits: 8, parity: 'none', stopBits: 1 });
    expect(serialDefaultsFor(undefined, 'prod').env).toBe('prod');
  });

  it('H7: 점프 직접 입력 주소를 나눈다. 사용자나 포트가 없으면 그대로 호스트', () => {
    expect(splitBastionInput('ec2-user@203.0.113.10:22')).toEqual({ host: '203.0.113.10', port: '22', username: 'ec2-user' });
    expect(splitBastionInput('[2001:db8::1]:2222')).toEqual({ host: '2001:db8::1', port: '2222', username: undefined });
    expect(splitBastionInput('bastion.local')).toEqual({ host: 'bastion.local' });
  });

  it('H9: PPK v3 키를 알아본다', () => {
    expect(isPpkV3('PuTTY-User-Key-File-3: ssh-ed25519\nEncryption: none')).toBe(true);
    expect(isPpkV3('PuTTY-User-Key-File-2: ssh-rsa')).toBe(false);
  });

  it('H10: 폴더 이름이 비면 저장할 수 없다', () => {
    expect(canSaveFolder('   ')).toBe(false);
    expect(canSaveFolder(' 헤펠레 ')).toBe(true);
  });
});
