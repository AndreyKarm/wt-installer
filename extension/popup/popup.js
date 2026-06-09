const BASE = "http://localhost:4316";

// Version info
document.getElementById("ext-version").textContent =
  `v${browser.runtime.getManifest().version}`;

fetch(`${BASE}/version`)
  .then((r) => r.text())
  .then((text) => {
    const el = document.getElementById("srv-version");
    el.textContent = `v${text.replace("Version: ", "")}`;
    el.style.color = "#4caf50";
  })
  .catch(() => {
    const el = document.getElementById("srv-version");
    el.textContent = "Offline";
    el.style.color = "#f44336";
  });

// Folder buttons
async function openFolder(target, btn) {
  btn.disabled = true;
  try {
    await fetch(`${BASE}/openfolder?target=${encodeURIComponent(target)}`, {
      method: "POST",
    });
  } catch (err) {
    console.error("[WT Local Downloader] openFolder failed:", err);
  } finally {
    btn.disabled = false;
  }
}

document.getElementById("btn-camo").addEventListener("click", (e) => {
  openFolder("camo", e.currentTarget);
});
document.getElementById("btn-installer").addEventListener("click", (e) => {
  openFolder("installer", e.currentTarget);
});
document.getElementById("btn-sight").addEventListener("click", (e) => {
  openFolder("sight", e.currentTarget);
});

// Logs
let currentTab = "server";
let logsOpen = false;
let autoRefresh = null;

const logsPanel = document.getElementById("logs-panel");
const logsActions = document.getElementById("logs-actions");
const logsArrow = document.getElementById("logs-arrow");
const logEntries = document.getElementById("log-entries");

document.getElementById("btn-toggle-logs").addEventListener("click", () => {
  logsOpen = !logsOpen;
  logsPanel.hidden = !logsOpen;
  logsActions.hidden = !logsOpen;
  logsArrow.textContent = logsOpen ? "▼" : "▶";

  if (logsOpen) {
    fetchLogs();
    autoRefresh = setInterval(fetchLogs, 2000);
  } else {
    clearInterval(autoRefresh);
    autoRefresh = null;
  }
});

document.querySelectorAll(".log-tab").forEach((tab) => {
  tab.addEventListener("click", () => {
    document
      .querySelectorAll(".log-tab")
      .forEach((t) => t.classList.remove("active"));
    tab.classList.add("active");
    currentTab = tab.dataset.tab;
    fetchLogs();
  });
});

document.getElementById("btn-refresh-logs").addEventListener("click", fetchLogs);

document
  .getElementById("btn-clear-logs")
  .addEventListener("click", async () => {
    if (currentTab === "server") {
      await fetch(`${BASE}/logs/clear`, { method: "POST" }).catch(() => { });
    } else {
      await browser.runtime.sendMessage({ action: "wt_clear_ext_logs" });
    }
    fetchLogs();
  });

async function fetchLogs() {
  if (currentTab === "server") {
    try {
      const r = await fetch(`${BASE}/logs`);
      const logs = await r.json();
      renderServerLogs(logs);
    } catch {
      renderEmpty("No logs yet.");
    }
  } else {
    try {
      const result = await browser.runtime.sendMessage({
        action: "wt_get_ext_logs",
      });
      renderExtLogs(result.logs ?? []);
    } catch {
      renderEmpty("Could not fetch extension logs");
    }
  }
}

function renderServerLogs(logs) {
  if (!logs.length) {
    renderEmpty("No server logs yet.");
    return;
  }
  logEntries.innerHTML = [...logs]
    .reverse()
    .map(
      ({ time, message }) =>
        `<div class="log-entry">` +
        `<span class="log-time">${esc(time)}</span>` +
        `<span class="log-msg">${esc(message)}</span>` +
        `</div>`
    )
    .join("");
}

function renderExtLogs(logs) {
  if (!logs.length) {
    renderEmpty("No extension logs yet.");
    return;
  }
  logEntries.innerHTML = [...logs]
    .reverse()
    .map(
      ({ time, level, source, message }) =>
        `<div class="log-entry log-${esc(level)}">` +
        `<span class="log-time">${esc(time)}</span>` +
        `<span class="log-src log-src-${esc(source)}">${esc(source)}</span>` +
        `<span class="log-msg">${esc(message)}</span>` +
        `</div>`
    )
    .join("");
}

function renderEmpty(msg) {
  logEntries.innerHTML = `<span class="log-empty">${esc(msg)}</span>`;
}

function esc(str) {
  return String(str)
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;");
}