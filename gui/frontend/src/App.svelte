<script>
  import { onMount, tick } from 'svelte';
  import { Connect, Disconnect, GetStatus, GetGUIVersion, GetInitialConfig, GetSavedConfig, SaveConfig } from '../wailsjs/go/main/App';
  import { EventsOn } from '../wailsjs/runtime/runtime';

  let server = "";
  let token = "";
  let fullTunnel = true;
  let guiVersion = "0.3.3";

  function handleConfigChange() {
    SaveConfig(server, token, fullTunnel);
  }
  
  let status = { state: 'disconnected', helper_version: '---', server_version: '---' };
  let stats = { bytes_sent: 0, bytes_recv: 0, uptime_seconds: 0 };
  let lastStats = { bytes_sent: 0, bytes_recv: 0 };
  let history = { sent: [], recv: [] };
  let logs = "";
  let lastLogs = "";
  let errorMsg = "";
  let errorTimeout;
  let logElement;

  function showError(msg) {
    errorMsg = msg;
    if (errorTimeout) clearTimeout(errorTimeout);
    errorTimeout = setTimeout(() => {
      errorMsg = "";
    }, 10000);
  }

  $: if (logs !== lastLogs && logElement) {
    lastLogs = logs;
    tick().then(() => {
      logElement.scrollTop = logElement.scrollHeight;
    });
  }

  onMount(async () => {
    // Fetch GUI version
    guiVersion = await GetGUIVersion();

    // 1. Try to load user saved config from Go (Keyring + settings.json)
    const saved = await GetSavedConfig();
    if (saved.server) server = saved.server;
    if (saved.token) token = saved.token;
    if (saved.full_tunnel !== undefined) fullTunnel = saved.full_tunnel;

    // 2. If still empty, try to get config from installer
    if (!server || !token) {
      const initConfig = await GetInitialConfig();
      if (!server && initConfig.server) server = initConfig.server;
      if (!token && initConfig.token) token = initConfig.token;
      
      // Save these so they persist next time
      handleConfigChange();
    }

    // Initial status fetch
    try {
      status = await GetStatus();
    } catch (e) {
      status = { state: 'disconnected', helper_version: '---', server_version: '---' };
    }

    // Listen for helper presence
    EventsOn("helper_status", (state) => {
      if (state === "missing") {
        if (!errorMsg) errorMsg = "Helper not detected. Please start slopn-helper.";
      } else {
        if (errorMsg === "Helper not detected. Please start slopn-helper.") {
          errorMsg = "";
        }
      }
    });

    // Listen for updates from Go
    EventsOn("vpn_status", (data) => {
      status = data;
    });

    EventsOn("vpn_stats", (data) => {
      // Calculate speeds
      if (lastStats.bytes_sent > 0) {
        const sentSpeed = Math.max(0, data.bytes_sent - lastStats.bytes_sent);
        const recvSpeed = Math.max(0, data.bytes_recv - lastStats.bytes_recv);
        
        history.sent = [...history.sent.slice(-29), sentSpeed];
        history.recv = [...history.recv.slice(-29), recvSpeed];
      } else {
        history.sent = [...history.sent.slice(-29), 0];
        history.recv = [...history.recv.slice(-29), 0];
      }
      
      lastStats = { bytes_sent: data.bytes_sent, bytes_recv: data.bytes_recv };
      stats = data;
    });

    EventsOn("vpn_logs", (data) => {
      logs = data;
    });
  });

  async function handleToggle() {
    errorMsg = "";
    try {
      if (status.state === 'disconnected') {
        const res = await Connect(server, token, fullTunnel);
        if (res !== "success") {
          showError(res);
        }
      } else {
        const res = await Disconnect();
        if (res !== "success") {
          showError(res);
        }
      }
    } catch (e) {
      showError(e.message || "Request failed");
    }
  }

  function formatBytes(bytes) {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  }

  function formatUptime(seconds) {
    const h = Math.floor(seconds / 3600);
    const m = Math.floor((seconds % 3600) / 60);
    const s = seconds % 60;
    return `${h}h ${m}m ${s}s`;
  }
</script>

<main>
  <div class="container">
    <div class="header">
      <img src="./assets/images/logo-universal.png" class="logo" alt="logo" />
      <h1>SloPN</h1>
    </div>

    <div class="card status-card">
      <div class="status-indicator {status.state}"></div>
      <div class="status-info">
        <p class="label">Status</p>
        <p class="value">{status.state.toUpperCase()}</p>
      </div>
      <button class="toggle-btn {status.state}" on:click={handleToggle} disabled={status.state === 'connecting'}>
        {status.state === 'disconnected' ? 'CONNECT' : (status.state === 'connecting' ? 'CONNECTING...' : 'DISCONNECT')}
      </button>
    </div>

    {#if status.state === 'connected'}
      <div class="stats-grid">
        <div class="card small">
          <p class="label">Assigned VIP</p>
          <p class="value highlight">{status.assigned_vip}</p>
        </div>
        <div class="card small">
          <p class="label">Uptime</p>
          <p class="value">{formatUptime(stats.uptime_seconds)}</p>
        </div>
        <div class="card small">
          <p class="label">Sent</p>
          <p class="value">{formatBytes(stats.bytes_sent)}</p>
          <div class="sparkline">
            <svg viewBox="0 0 100 20" preserveAspectRatio="none">
              <polyline
                fill="none"
                stroke="#00ff88"
                stroke-width="1"
                points={history.sent.map((v, i) => `${(i / 29) * 100},${20 - (Math.min(v, 1000000) / 1000000) * 20}`).join(' ')}
              />
            </svg>
          </div>
        </div>
        <div class="card small">
          <p class="label">Received</p>
          <p class="value">{formatBytes(stats.bytes_recv)}</p>
          <div class="sparkline">
            <svg viewBox="0 0 100 20" preserveAspectRatio="none">
              <polyline
                fill="none"
                stroke="#00ff88"
                stroke-width="1"
                points={history.recv.map((v, i) => `${(i / 29) * 100},${20 - (Math.min(v, 1000000) / 1000000) * 20}`).join(' ')}
              />
            </svg>
          </div>
        </div>
      </div>
    {/if}

    <div class="card logs-card">
      <p class="label">Engine Logs</p>
      <div class="logs-container" bind:this={logElement}>
        {logs || 'Waiting for logs...'}
      </div>
    </div>

    <div class="card config-card">
      <div class="input-group">
        <label for="server">Server Address</label>
        <input id="server" bind:value={server} on:input={handleConfigChange} disabled={status.state !== 'disconnected'} />
      </div>
      <div class="input-group">
        <label for="token">Auth Token</label>
        <input id="token" type="password" bind:value={token} on:input={handleConfigChange} disabled={status.state !== 'disconnected'} />
      </div>
      <div class="input-group checkbox">
        <input id="full" type="checkbox" bind:checked={fullTunnel} on:change={handleConfigChange} disabled={status.state !== 'disconnected'} />
        <label for="full">Full Tunnel (Route All Traffic)</label>
      </div>
    </div>

    {#if errorMsg}
      <div class="error-banner">
        {errorMsg}
      </div>
    {/if}

    <div class="footer">
      <p>GUI v{guiVersion} | Engine v{status.helper_version || '---'} | Server v{status.server_version || '---'}</p>
    </div>
  </div>
</main>

<style>
  :global(body) {
    background-color: #1a1a1a;
    color: white;
    font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
    margin: 0;
    padding: 0;
    overflow: hidden;
  }

  .container {
    padding: 15px;
    max-width: 800px;
    margin: 0 auto;
    display: flex;
    flex-direction: column;
    height: 100vh;
    box-sizing: border-box;
  }

  .header {
    display: flex;
    align-items: center;
    gap: 12px;
    margin-bottom: 15px;
  }

  .logo {
    width: 32px;
    height: 32px;
  }

  h1 {
    font-size: 1.2rem;
    margin: 0;
    color: #00ff88;
  }

  .card {
    background: #2a2a2a;
    border-radius: 12px;
    padding: 16px;
    margin-bottom: 12px;
    box-shadow: 0 4px 6px rgba(0,0,0,0.3);
  }

  .status-card {
    display: flex;
    align-items: center;
    gap: 15px;
  }

  .status-indicator {
    width: 12px;
    height: 12px;
    border-radius: 50%;
    background: gray;
  }
  .status-indicator.connected { background: #00ff88; box-shadow: 0 0 10px #00ff88; }
  .status-indicator.connecting { background: #ffcc00; }
  .status-indicator.disconnected { background: #ff4444; }

  .status-info { flex-grow: 1; }

  .label {
    font-size: 0.7rem;
    color: #888;
    margin: 0;
    text-transform: uppercase;
  }

  .value {
    font-size: 1rem;
    font-weight: bold;
    margin: 0;
  }

  .value.highlight { color: #00ff88; }

  .sparkline {
    margin-top: 8px;
    height: 20px;
    width: 100%;
    background: rgba(0, 255, 136, 0.05);
    border-radius: 2px;
  }

  .sparkline svg {
    width: 100%;
    height: 100%;
    display: block;
  }

  .toggle-btn {
    border: none;
    border-radius: 8px;
    padding: 10px 16px;
    font-weight: bold;
    cursor: pointer;
    transition: background 0.2s;
  }

  .toggle-btn.disconnected { background: #00ff88; color: black; }
  .toggle-btn.connected { background: #ff4444; color: white; }
  .toggle-btn:disabled { opacity: 0.5; cursor: not-allowed; }

  .stats-grid {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    gap: 10px;
    margin-bottom: 12px;
  }

  .card.small { padding: 12px; margin-bottom: 0; }

  .logs-card {
    padding: 10px;
    display: flex;
    flex-direction: column;
    height: 120px;
    margin-bottom: 10px;
  }

  .logs-container {
    margin-top: 5px;
    background: #111;
    border-radius: 4px;
    padding: 8px;
    font-family: 'Menlo', 'Monaco', 'Courier New', monospace;
    font-size: 0.6rem;
    line-height: 1.2;
    color: #00ff88;
    white-space: pre-wrap;
    word-break: break-all;
    overflow-y: auto;
    height: 80px;
    text-align: left;
    pointer-events: auto;
  }

  .config-card { 
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 10px;
    margin-bottom: 10px;
    padding: 12px;
  }

  .input-group.checkbox {
    grid-column: span 2;
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .input-group label { display: block; font-size: 0.65rem; color: #888; margin-bottom: 2px; }
  .input-group input {
    width: 100%;
    background: #1a1a1a;
    border: 1px solid #444;
    color: white;
    padding: 6px;
    border-radius: 6px;
    box-sizing: border-box;
    font-size: 0.8rem;
  }

  .input-group.checkbox { display: flex; align-items: center; gap: 8px; }
  .input-group.checkbox input { width: auto; }

  .error-banner {
    background: rgba(255, 68, 68, 0.2);
    border: 1px solid #ff4444;
    color: #ff4444;
    padding: 10px;
    border-radius: 8px;
    font-size: 0.8rem;
    margin-top: 10px;
  }

  .footer {
    margin-top: auto;
    padding: 8px;
    text-align: center;
    font-size: 0.65rem;
    color: #888;
    background: #111;
    border-radius: 8px;
  }
</style>
