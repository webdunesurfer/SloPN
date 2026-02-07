<script>
  import { onMount } from 'svelte';
  import { Connect, Disconnect, GetStatus, GetGUIVersion } from '../wailsjs/go/main/App';
  import { EventsOn } from '../wailsjs/runtime/runtime';

  let server = "38.242.216.161:4242";
  let token = "8a1b06c4-13a4-4b00-a0e4-79d9ff804eb0";
  let fullTunnel = true;
  let guiVersion = "0.1.1";
  
  let status = { state: 'disconnected', helper_version: '---', server_version: '---' };
  let stats = { bytes_sent: 0, bytes_recv: 0, uptime_seconds: 0 };
  let errorMsg = "";
  let errorTimeout;

  function showError(msg) {
    errorMsg = msg;
    if (errorTimeout) clearTimeout(errorTimeout);
    errorTimeout = setTimeout(() => {
      errorMsg = "";
    }, 10000);
  }

  onMount(async () => {
    // Fetch GUI version
    guiVersion = await GetGUIVersion();

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
      stats = data;
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
      <h1>SloPN VPN</h1>
    </div>

    <div class="card status-card">
      <div class="status-indicator {status.state}"></div>
      <div class="status-info">
        <p class="label">Status</p>
        <p class="value">{status.state.toUpperCase()}</p>
      </div>
      <button class="toggle-btn {status.state}" on:click={handleToggle}>
        {status.state === 'disconnected' ? 'CONNECT' : 'DISCONNECT'}
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
        </div>
        <div class="card small">
          <p class="label">Received</p>
          <p class="value">{formatBytes(stats.bytes_recv)}</p>
        </div>
      </div>
    {/if}

    <div class="card config-card">
      <div class="input-group">
        <label for="server">Server Address</label>
        <input id="server" bind:value={server} disabled={status.state !== 'disconnected'} />
      </div>
      <div class="input-group">
        <label for="token">Auth Token</label>
        <input id="token" type="password" bind:value={token} disabled={status.state !== 'disconnected'} />
      </div>
      <div class="input-group checkbox">
        <input id="full" type="checkbox" bind:checked={fullTunnel} disabled={status.state !== 'disconnected'} />
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
  }

  .container {
    padding: 20px;
    max-width: 400px;
    margin: 0 auto;
    display: flex;
    flex-direction: column;
    min-height: 100vh;
  }

  .header {
    display: flex;
    align-items: center;
    gap: 12px;
    margin-bottom: 20px;
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
    grid-template-columns: 1fr 1fr;
    gap: 10px;
    margin-bottom: 12px;
  }

  .card.small { padding: 12px; margin-bottom: 0; }

  .config-card { display: flex; flex-direction: column; gap: 12px; }

  .input-group label { display: block; font-size: 0.7rem; color: #888; margin-bottom: 4px; }
  .input-group input {
    width: 100%;
    background: #1a1a1a;
    border: 1px solid #444;
    color: white;
    padding: 8px;
    border-radius: 6px;
    box-sizing: border-box;
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
    margin-top: 20px;
    padding: 10px;
    text-align: center;
    font-size: 0.7rem;
    color: #888;
    background: #111;
    border-radius: 8px;
  }
</style>
