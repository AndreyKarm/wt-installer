// localhost matches the manifest's host_permissions. Switching to 127.0.0.1
// would be a new permission, and Firefox may ask users to approve it before
// the update installs.
const BASE = "http://localhost:4316";

// Oldest tray app with everything this extension uses (/installed,
// /uninstall). Older servers still install, but marks and uninstall fail.
const MIN_SERVER_VERSION = "0.1.8";

// compareVersions returns 1 if a > b, -1 if a < b, 0 if equal. Same rules
// as the tray app's updater: numeric parts, missing parts count as 0.
function compareVersions(a, b) {
  const pa = a.replace(/^v/, "").split(".");
  const pb = b.replace(/^v/, "").split(".");
  for (let i = 0; i < Math.max(pa.length, pb.length); i++) {
    const x = parseInt(pa[i], 10) || 0;
    const y = parseInt(pb[i], 10) || 0;
    if (x !== y) return x > y ? 1 : -1;
  }
  return 0;
}

// Version info
document.getElementById("ext-version").textContent =
  `v${browser.runtime.getManifest().version}`;

fetch(`${BASE}/version`)
  .then((r) => r.text())
  .then((text) => {
    const version = text.replace("Version: ", "").trim();
    const el = document.getElementById("srv-version");
    el.textContent = `v${version}`;
    el.style.color = "#4caf50";

    if (compareVersions(version, MIN_SERVER_VERSION) < 0) {
      el.style.color = "#ffa726";
      const warning = document.getElementById("srv-warning");
      warning.textContent =
        `The tray app is outdated. "Installed" marks and uninstall need ` +
        `v${MIN_SERVER_VERSION} or newer. Restart it to update.`;
      warning.hidden = false;
    }
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
document.getElementById("btn-logs").addEventListener("click", (e) => {
  openFolder("logs", e.currentTarget);
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

// Log rows are built with DOM calls and textContent, never innerHTML: the
// text comes from the server and the page, and AMO review flags innerHTML.
function el(tag, className, text) {
  const node = document.createElement(tag);
  node.className = className;
  if (text !== undefined) node.textContent = String(text);
  return node;
}

// Only known values become class names, so log data can't inject classes.
const LOG_LEVELS = ["info", "warn", "error"];
const LOG_SOURCES = ["content", "bg"];

function renderServerLogs(logs) {
  if (!logs.length) {
    renderEmpty("No server logs yet.");
    return;
  }
  logEntries.replaceChildren(
    ...[...logs].reverse().map(({ time, message }) => {
      const row = el("div", "log-entry");
      row.append(el("span", "log-time", time), el("span", "log-msg", message));
      return row;
    })
  );
}

function renderExtLogs(logs) {
  if (!logs.length) {
    renderEmpty("No extension logs yet.");
    return;
  }
  logEntries.replaceChildren(
    ...[...logs].reverse().map(({ time, level, source, message }) => {
      const lvl = LOG_LEVELS.includes(level) ? level : "info";
      const src = LOG_SOURCES.includes(source) ? source : "content";
      const row = el("div", `log-entry log-${lvl}`);
      row.append(
        el("span", "log-time", time),
        el("span", `log-src log-src-${src}`, source),
        el("span", "log-msg", message)
      );
      return row;
    })
  );
}

function renderEmpty(msg) {
  logEntries.replaceChildren(el("span", "log-empty", msg));
}