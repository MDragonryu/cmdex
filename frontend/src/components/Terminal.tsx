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
}

export interface TerminalHandle {
    clear: () => void;
}

const TerminalComponent = forwardRef<TerminalHandle, TerminalComponentProps>(
    ({ monoFont, isVisible }, ref) => {
  const containerRef = useRef<HTMLDivElement>(null);
  const terminalRef = useRef<Terminal | null>(null);
  const fitAddonRef = useRef<FitAddon | null>(null);

  useImperativeHandle(ref, () => ({
      clear: () => {
          Write('\x0c').catch((err) => console.error('clear failed:', err));
      },
  }));

  useEffect(() => {
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

    return () => {
      cleanupOutput();
      cleanupExit();
    };
  }, []);

  useEffect(() => {
      const term = terminalRef.current;
      if (!term) return;

      let keystrokeBuffer = '';

      const flushBuffer = () => {
          if (keystrokeBuffer.length > 0) {
              const batch = keystrokeBuffer;
              keystrokeBuffer = '';
              Write(batch).catch((err) => console.error('TerminalService.Write failed:', err));
          }
      };

      const handleData = (data: string) => {
          keystrokeBuffer += data;
          flushBuffer();
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
      style={{ display: isVisible ? 'flex' : 'none' }}
    />
  );
  }
);

export default TerminalComponent;
