import { useEffect, useRef, useState } from 'react';
import { Terminal } from 'xterm';
import { FitAddon } from 'xterm-addon-fit';
import 'xterm/css/xterm.css';

interface ContainerTerminalProps {
  containerId: string;
  orgId: string;
  connectionId: string;
}

export default function ContainerTerminal({ containerId, orgId, connectionId }: ContainerTerminalProps) {
  const terminalRef = useRef<HTMLDivElement>(null);
  const terminalInstance = useRef<Terminal | null>(null);
  const wsRef = useRef<WebSocket | null>(null);
  const fitAddonRef = useRef<FitAddon | null>(null);
  const [isConnected, setIsConnected] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!terminalRef.current) return;

    // Create terminal
    const terminal = new Terminal({
      cursorBlink: true,
      fontSize: 14,
      fontFamily: "'Geist Mono', 'SF Mono', 'Fira Code', monospace",
      theme: {
        background: '#0a0a0f',
        foreground: '#e0e0e8',
        cursor: '#00d4ff',
        cursorAccent: '#0a0a0f',
        selectionBackground: '#00d4ff33',
        black: '#0a0a0f',
        red: '#ff4444',
        green: '#00ff88',
        yellow: '#ffaa00',
        blue: '#00d4ff',
        magenta: '#a855f7',
        cyan: '#00d4ff',
        white: '#e0e0e8',
        brightBlack: '#8888a0',
        brightRed: '#ff6666',
        brightGreen: '#33ff99',
        brightYellow: '#ffcc33',
        brightBlue: '#33ddff',
        brightMagenta: '#bb77ff',
        brightCyan: '#33ddff',
        brightWhite: '#ffffff',
      },
    });

    const fitAddon = new FitAddon();
    terminal.loadAddon(fitAddon);

    terminal.open(terminalRef.current);
    fitAddon.fit();

    terminalInstance.current = terminal;
    fitAddonRef.current = fitAddon;

    // Connect WebSocket through nginx proxy
    const token = localStorage.getItem('token');
    const wsProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${wsProtocol}//${window.location.host}/ws/orgs/${orgId}/connections/${connectionId}/containers/${containerId}/exec?token=${token}`;
    
    const ws = new WebSocket(wsUrl);
    wsRef.current = ws;

    ws.onopen = () => {
      setIsConnected(true);
      setError(null);
      terminal.focus();
    };

    ws.onmessage = (event) => {
      if (event.data instanceof Blob) {
        event.data.text().then((text) => {
          terminal.write(text);
        });
      } else {
        terminal.write(event.data);
      }
    };

    ws.onerror = () => {
      setError('Connection failed');
      setIsConnected(false);
    };

    ws.onclose = () => {
      setIsConnected(false);
      terminal.write('\r\n\x1b[31m[Disconnected]\x1b[0m\r\n');
    };

    // Send keystrokes to WebSocket
    terminal.onData((data) => {
      if (ws.readyState === WebSocket.OPEN) {
        ws.send(data);
      }
    });

    // Handle resize
    const handleResize = () => {
      if (fitAddon && ws.readyState === WebSocket.OPEN) {
        fitAddon.fit();
        const dims = fitAddon.proposeDimensions();
        if (dims) {
          ws.send(JSON.stringify({
            type: 'resize',
            cols: dims.cols,
            rows: dims.rows,
          }));
        }
      }
    };

    window.addEventListener('resize', handleResize);

    // Initial resize
    setTimeout(handleResize, 100);

    return () => {
      window.removeEventListener('resize', handleResize);
      ws.close();
      terminal.dispose();
    };
  }, [containerId, orgId, connectionId]);

  return (
    <div style={{ 
      height: '100%', 
      display: 'flex', 
      flexDirection: 'column',
      background: '#0a0a0f',
      borderRadius: '8px',
      overflow: 'hidden',
      border: '1px solid var(--color-border)',
    }}>
      {/* Terminal Header */}
      <div style={{
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
        padding: '8px 12px',
        background: 'var(--color-surface)',
        borderBottom: '1px solid var(--color-border)',
      }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
          <div style={{
            width: 8,
            height: 8,
            borderRadius: '50%',
            background: isConnected ? '#00ff88' : '#ff4444',
          }} />
          <span style={{ fontSize: 12, color: 'var(--color-text-muted)' }}>
            {isConnected ? 'Connected' : error || 'Connecting...'}
          </span>
        </div>
        <div style={{ display: 'flex', gap: 8 }}>
          <span style={{ fontSize: 11, color: 'var(--color-text-muted)', fontFamily: 'monospace' }}>
            {containerId.substring(0, 12)}
          </span>
        </div>
      </div>

      {/* Terminal */}
      <div 
        ref={terminalRef} 
        style={{ 
          flex: 1, 
          padding: '4px',
        }} 
      />
    </div>
  );
}
