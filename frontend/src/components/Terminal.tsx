import React, { useEffect, useRef } from 'react';
import { Terminal } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';
import { WebglAddon } from '@xterm/addon-webgl';
import { WebLinksAddon } from '@xterm/addon-web-links';
import '@xterm/xterm/css/xterm.css';

interface TerminalComponentProps {
  monoFont: string;
  isVisible: boolean;
}

const TerminalComponent: React.FC<TerminalComponentProps> = ({ monoFont, isVisible }) => {
  const containerRef = useRef<HTMLDivElement>(null);
  const terminalRef = useRef<Terminal | null>(null);
  const fitAddonRef = useRef<FitAddon | null>(null);

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
      requestAnimationFrame(() => fitAddon.fit());
    }

    terminalRef.current = term;

    const observer = new ResizeObserver(() => {
      fitAddonRef.current?.fit();
    });
    if (containerRef.current) {
      observer.observe(containerRef.current);
    }

    return () => {
      observer.disconnect();
      if (terminalRef.current === term) {
        term.dispose();
        terminalRef.current = null;
        fitAddonRef.current = null;
      }
    };
  }, [monoFont]);

  return (
    <div
      ref={containerRef}
      className="terminal-container"
      style={{ display: isVisible ? 'flex' : 'none' }}
    />
  );
};

export default TerminalComponent;
