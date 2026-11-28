// 짧은 알림 소리(터미널 벨, 출력 트리거). 소리를 낼 수 없는 환경에서는 조용히 넘어간다.
export function beep(): void {
  try {
    const ctx = new AudioContext();
    const osc = ctx.createOscillator();
    osc.frequency.value = 880;
    osc.connect(ctx.destination);
    osc.start();
    osc.stop(ctx.currentTime + 0.08);
    osc.onended = () => void ctx.close();
  } catch {
    /* 소리를 낼 수 없으면 무시 */
  }
}
