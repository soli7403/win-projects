// 설정·가져오기·동기화·업데이트 화면의 IPC 페이로드 타입. 근거: spec/ipc.md §2, spec/import.md §4.
export type ImportSource = 'putty' | 'securecrt' | 'xshell' | 'ssh-config' | 'aws';

/** 후보 안의 서버 정보(data/store.md 서버 필드의 부분 집합). spec/ipc.md "가져오기 후보 객체". 모든 필드는 없을 수 있다. */
export interface ImportHost {
  alias?: string;
  protocol?: 'ssh' | 'telnet' | 'serial' | 'local' | string;
  host?: string;
  port?: number;
  username?: string;
  authType?: string;
  keepaliveSec?: number;
  encoding?: string;
  termType?: string;
  forwards?: unknown[];
  serial?: { path?: string; baudRate?: number; [k: string]: unknown };
  startupCommand?: string;
  notes?: string;
}

/** 가져오기 후보 한 건. 화면은 이 구조를 그대로 읽고 import.apply 에도 변경 없이 되돌려 보낸다. */
export interface ImportCandidate {
  source: ImportSource;
  name: string;
  host: ImportHost;
  keyFile?: string;
  jumpVia?: string;
  /** 경유 서버가 여러 단계일 때 바깥부터의 주소 목록(ssh config 의 ProxyJump a,b, D153). 마지막이 jumpVia. */
  jumpChain?: string[];
  note?: string;
  /** 넣을 폴더 아래에 만들 하위 폴더 이름들(SecureCRT·Xshell, spec/import.md §1.5·§1.6). */
  folderPath?: string[];
}

export interface ImportApplyResult {
  hosts: number;
  keys: number;
  skipped: string[];
}

export type SyncMode = '' | 'pull-vault' | 'overwrite';

export interface SyncRunResult {
  state: 'ok' | 'unconfigured' | 'locked' | 'nogit' | 'vault-mismatch' | 'error' | string;
  message?: string;
  lastSync?: number;
  changed?: boolean;
}

export interface UpdateInfo {
  version: string;
  url: string;
  announce?: boolean;
  /** Microsoft Store 에서 찾은 새 버전(lifecycle.md §6.1). 누르면 update.install 로 Windows 의 업데이트 확인 창을 띄운다. */
  store?: boolean;
}

export interface UpdateCheckResult {
  status: 'newer' | 'latest' | 'error' | 'unconfigured';
  version?: string;
  url?: string;
  store?: boolean;
}
