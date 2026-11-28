// 표시 형식. 근거: spec/ui/dialogs.md §1.5.

/** 바이트: 1024 미만 `n B`, 1MB 미만 `x.x KB`, 1GB 미만 `x.x MB`, 그 이상 `x.xx GB`. */
export function formatBytes(n: number): string {
  const KB = 1024;
  const MB = KB * 1024;
  const GB = MB * 1024;
  if (n < KB) return `${n} B`;
  if (n < MB) return `${(n / KB).toFixed(1)} KB`;
  if (n < GB) return `${(n / MB).toFixed(1)} MB`;
  return `${(n / GB).toFixed(2)} GB`;
}

const pad = (n: number) => String(n).padStart(2, '0');

/**
 * 날짜: 올해이면 `MM-DD HH:mm`, 아니면 `YYYY-MM-DD`, 값이 0 이면 빈 칸.
 * ms 는 밀리초 타임스탬프.
 */
export function formatDate(ms: number, now: Date = new Date()): string {
  if (!ms) return '';
  const d = new Date(ms);
  if (d.getFullYear() === now.getFullYear()) {
    return `${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`;
  }
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;
}

/** 경과 시간 HH:MM:SS (상태 표시줄). */
export function formatClock(ms: number): string {
  const d = new Date(ms);
  return `${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`;
}

/** 경과 시간 HH:MM:SS(상태 표시줄 연결 경과). ms 는 경과 밀리초. */
export function formatDuration(ms: number): string {
  const total = Math.max(0, Math.floor(ms / 1000));
  const h = Math.floor(total / 3600);
  const m = Math.floor((total % 3600) / 60);
  const s = total % 60;
  return `${pad(h)}:${pad(m)}:${pad(s)}`;
}

const COMPACT_UNITS = ['K', 'M', 'G', 'T', 'P'];

/**
 * 좁은 자리(창 머리글)용 바이트 표기: **늘 5글자 이하**라 값이 바뀌어도 폭이 거의 그대로다 (spec/ui/panes.md §2.7, D160).
 * 1000 미만 `n B`, 그 위로는 1024 단위로 나눠 10 미만이면 소수 한 자리(`9.9 K`), 아니면 정수(`512 M`).
 * 정수로 반올림해 1000 이 되면 다음 단위로 올린다(`1000 K` 대신 `1.0 M`).
 */
export function formatBytesCompact(n: number): string {
  if (!Number.isFinite(n) || n < 1000) return `${Math.max(0, Math.floor(Number.isFinite(n) ? n : 0))} B`;
  let v = n;
  for (const unit of COMPACT_UNITS) {
    v /= 1024;
    if (v < 9.95) return `${v.toFixed(1)} ${unit}`;
    if (Math.round(v) < 1000) return `${Math.round(v)} ${unit}`;
  }
  return `${Math.round(v)} ${COMPACT_UNITS[COMPACT_UNITS.length - 1]}`;
}
