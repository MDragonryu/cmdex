import React, { useEffect, useRef, forwardRef, useImperativeHandle } from 'react';
import { Terminal } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';
import { WebglAddon } from '@xterm/addon-webgl';
import { WebLinksAddon } from '@xterm/addon-web-links';
import '@xterm/xterm/css/xterm.css';
import { Events } from '@wailsio/runtime';
import { eventNames } from '../wails/events';
import { Write, Resize } from '../../bindings/cmdex/terminalservice';

interface TerminalComponentProps {
  monoFont: string;
  isVisible: boolean;
  theme: string;
}

export interface TerminalHandle {
    clear: () => void;
    getSelection: () => string;
    getLastOutput: () => string;
}

const TerminalComponent = forwardRef<TerminalHandle, TerminalComponentProps>(
    ({ monoFont, isVisible, theme }, ref) => {
  const containerRef = useRef<HTMLDivElement>(null);
  const terminalRef = useRef<Terminal | null>(null);
  const fitAddonRef = useRef<FitAddon | null>(null);
  const isFirstMountRef = useRef(true);

    function hexToRgba(hex: string, alpha: number): string {
        hex = hex.replace('#', '');
        if (hex.length === 3) {
            hex = hex[0] + hex[0] + hex[1] + hex[1] + hex[2] + hex[2];
        }
        const r = parseInt(hex.substring(0, 2), 16);
        const g = parseInt(hex.substring(2, 4), 16);
        const b = parseInt(hex.substring(4, 6), 16);
        return `rgba(${r}, ${g}, ${b}, ${alpha})`;
    }

  useImperativeHandle(ref, () => ({
      clear: () => {
          Write('\x0c').catch((err) => console.error('clear failed:', err));
      },
      getSelection: () => terminalRef.current?.getSelection() || '',
      getLastOutput: () => {
          const buffer = terminalRef.current?.buffer.active;
          if (!buffer) return '';

          const stripAnsi = (str: string) => str
              .replace(/\x1B\[[0-9;]*[mGKHFJA-Za-z]/g, '')
              .replace(/\x1B\][^\x07]*\x07/g, '')
              .replace(/\r/g, '');

          const promptRegex = /[$#%❯>➤λ→⟩»◇](\s|$)/;

          const isLineContent = (i: number): boolean => {
              const l = buffer.getLine(i);
              if (!l) return false;
              if (l.isWrapped) {
                  let p = i - 1;
                  while (p >= 0 && buffer.getLine(p)?.isWrapped) p--;
                  if (p >= 0) {
                      const pl = buffer.getLine(p);
                      if (pl && promptRegex.test(pl.translateToString(true).trim())) return false;
                  }
              }
              return stripAnsi(l.translateToString(true)).trim().length > 0;
          };

          const cursorPos = buffer.cursorY + buffer.baseY;
          let promptIdx = -1;
          let prevPromptIdx = -1;
          let scanFrom = cursorPos;

          while (scanFrom >= 0) {
              let found = -1;
              for (let i = scanFrom; i >= 0; i--) {
                  const line = buffer.getLine(i);
                  if (!line) continue;
                  if (promptRegex.test(line.translateToString(true).trim())) {
                      found = i;
                      break;
                  }
              }

              if (found === -1) break;

              if (promptIdx === -1) {
                  promptIdx = found;
                  scanFrom = found - 1;
                  continue;
              }

              let hasContent = false;
              for (let i = found + 1; i < promptIdx; i++) {
                  if (isLineContent(i)) { hasContent = true; break; }
              }

              if (hasContent) {
                  prevPromptIdx = found;
                  break;
              }

              promptIdx = found;
              scanFrom = found - 1;
          }

          if (promptIdx === -1) return '';

          const outputStart = prevPromptIdx !== -1 ? prevPromptIdx + 1 : promptIdx + 1;
          const outputEnd = prevPromptIdx !== -1 ? promptIdx - 1 : cursorPos;

          const outputLines: string[] = [];
          for (let i = outputStart; i <= outputEnd; i++) {
              const line = buffer.getLine(i);
              if (!line) continue;
              if (line.isWrapped) {
                  let parent = i - 1;
                  while (parent >= 0 && buffer.getLine(parent)?.isWrapped) parent--;
                  if (parent >= 0) {
                      const parentLine = buffer.getLine(parent);
                      if (parentLine && promptRegex.test(parentLine.translateToString(true).trim())) continue;
                  }
              }
              const text = line.translateToString(true);
              const stripped = stripAnsi(text);
              if (stripped.length > 0) outputLines.push(stripped);
          }

          return outputLines.join('\n').trim();
      },
  }));

  useEffect(() => {
    const skipTransition = isFirstMountRef.current;
    if (isFirstMountRef.current) {
        isFirstMountRef.current = false;
    }

    const container = containerRef.current;
    if (!skipTransition && container) {
        container.style.opacity = '0';
        container.style.transition = 'opacity var(--transition-fast)';
    }

    const term = new Terminal({
      cursorBlink: true,
      cursorStyle: 'block',
      fontSize: 14,
      fontFamily: monoFont || 'JetBrains Mono, Fira Code, monospace',
      fontWeight: '400',
      scrollback: 5000,
      convertEol: true,
      allowProposedApi: true,
      allowTransparency: false,
      theme: {
        background: '#1e1e1e',
        foreground: '#d4d4d4',
        cursor: '#d4d4d4',
        cursorAccent: '#1e1e1e',
        selectionBackground: '#264f78',
        black: '#000000',
        red: '#cd3131',
        green: '#0dbc79',
        yellow: '#e5e510',
        blue: '#2472c8',
        magenta: '#bc3fbc',
        cyan: '#11a8cd',
        white: '#e5e5e5',
        brightBlack: '#666666',
        brightRed: '#f44747',
        brightGreen: '#4ec9b0',
        brightYellow: '#d7ba7d',
        brightBlue: '#569cd6',
        brightMagenta: '#c586c0',
        brightCyan: '#4ec9b0',
        brightWhite: '#ffffff',
      },
    });

    const fitAddon = new FitAddon();
    term.loadAddon(fitAddon);
    fitAddonRef.current = fitAddon;

    const webLinksAddon = new WebLinksAddon((event, uri) => {
      window.open(uri, '_blank');
    });
    term.loadAddon(webLinksAddon);

    try {
      const webglAddon = new WebglAddon();
      webglAddon.onContextLoss(() => webglAddon.dispose());
      term.loadAddon(webglAddon);
    } catch {
      // WebGL unavailable — canvas renderer used by default
    }

    if (containerRef.current) {
      term.open(containerRef.current);
      requestAnimationFrame(() => {
        fitAddon.fit();
        Resize(term.cols, term.rows).catch((err) => console.error('resize failed:', err));
        if (!skipTransition) {
            containerRef.current!.style.opacity = '1';
        }
      });
    }

    terminalRef.current = term;

    const resizeDisposable = term.onResize(({ cols, rows }) => {
      Resize(cols, rows).catch((err) => console.error('resize failed:', err));
    });

    const observer = new ResizeObserver(() => {
      fitAddonRef.current?.fit();
    });
    if (containerRef.current) {
      observer.observe(containerRef.current);
    }

    return () => {
      resizeDisposable.dispose();
      observer.disconnect();
      if (terminalRef.current === term) {
        term.dispose();
        terminalRef.current = null;
        fitAddonRef.current = null;
      }
    };
  }, [monoFont]);

  useEffect(() => {
    const term = terminalRef.current;
    if (!term) return;

    requestAnimationFrame(() => {
        const current = terminalRef.current;
        if (!current) return;

        const styles = getComputedStyle(document.documentElement);
        const background = styles.getPropertyValue('--background').trim();
        const foreground = styles.getPropertyValue('--foreground').trim();
        const primary = styles.getPropertyValue('--primary').trim();
        const cursorAccent = background;
        const selectionBg = hexToRgba(primary, 0.4);

        current.options.theme = {
            ...current.options.theme,
            background,
            foreground,
            cursor: primary,
            cursorAccent,
            selectionBackground: selectionBg,
        };
    });
  }, [theme]);

  useEffect(() => {
    const term = terminalRef.current;
    if (!term) return;

    const cleanupOutput = Events.On(eventNames.ptyOutput, (event: { data: { data: string } }) => {
      const output = event?.data?.data;
      if (output) {
        term.write(output);
      }
    });

    const cleanupExit = Events.On(eventNames.ptyExit, (event: { data: { exitCode: number; wasIntentional: boolean } }) => {
      const { exitCode, wasIntentional } = event?.data ?? {};
      console.log(`Shell exited: code=${exitCode}, intentional=${wasIntentional}`);
    });

    const cleanupCmd = Events.On(eventNames.cmdOutput, (event: { data: { stream: string; data: string } }) => {
      const chunk = event?.data;
      if (!chunk?.data) return;
      if (chunk.stream === 'stderr') {
        term.write('\x1b[31m' + chunk.data + '\x1b[0m');
      } else {
        term.write(chunk.data);
      }
    });

    return () => {
      cleanupOutput();
      cleanupExit();
      cleanupCmd();
    };
  }, []);

  useEffect(() => {
      const term = terminalRef.current;
      if (!term) return;

      let keystrokeBuffer = '';
      let flushScheduled = false;

      const flushBuffer = () => {
        flushScheduled = false;
        if (keystrokeBuffer.length > 0) {
          const batch = keystrokeBuffer;
          keystrokeBuffer = '';
          Write(batch).catch((err) =>
            console.error('TerminalService.Write failed:', err)
          );
        }
      };

      const handleData = (data: string) => {
          // Accumulate keystrokes
        keystrokeBuffer += data;

        // Schedule a single flush on the next microtask tick,
        // so multiple same-tick keystrokes are sent as one batch
        if (!flushScheduled) {
          flushScheduled = true;
          Promise.resolve().then(flushBuffer);
        }
      };

      const inputDisposable = term.onData(handleData);

      return () => {
        inputDisposable.dispose();
        flushBuffer();
      };
  }, []);

  return (
    <div
      ref={containerRef}
      className="terminal-container"
      style={{ display: isVisible ? '' : 'none' }}
    />
  );
  }
);

export default TerminalComponent;
