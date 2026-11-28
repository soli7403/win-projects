import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import './styles/app.css';
import { App } from './App';
import { installHelpCapture } from './lib/helpCapture';

// 도움말 그림 자동 캡처 때만(데모 프로필의 화면 저장소 표시) 화면을 여는 손잡이를 단다. D158.
installHelpCapture();

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <App />
  </StrictMode>,
);
