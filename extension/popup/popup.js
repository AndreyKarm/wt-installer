// localhost matches the manifest's host_permissions. Switching to 127.0.0.1
// would be a new permission, and Firefox may ask users to approve it before
// the update installs.
const BASE = "http://localhost:4316";

// Oldest tray app with everything this extension uses (/installed,
// /uninstall, /gamedirs). Older servers still install, but marks,
// uninstall and choosing the game folder fail.
const MIN_SERVER_VERSION = "0.1.9";

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
        `The tray app is outdated. "Installed" marks, uninstall and ` +
        `choosing the game folder need v${MIN_SERVER_VERSION} or newer. ` +
        `Restart it to update.`;
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

// Game installs: which War Thunder folders skins go into
const gameList = document.getElementById("game-list");
const gameError = document.getElementById("game-error");
const gameAddForm = document.getElementById("game-add-form");
const gameAddPath = document.getElementById("game-add-path");

const SOURCE_LABELS = {
  launcher: "Launcher",
  steam: "Steam",
  nearby: "Found",
  custom: "Added",
};

function showGameError(msg) {
  gameError.textContent = msg ?? "";
  gameError.hidden = !msg;
}

// gameDirsRequest calls /gamedirs and renders the list it answers with.
async function gameDirsRequest(body) {
  const options = body
    ? {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      }
    : {};
  let r;
  try {
    r = await fetch(`${BASE}/gamedirs`, options);
  } catch {
    renderGamesMessage("Tray app is offline.");
    return false;
  }
  if (r.status === 404) {
    renderGamesMessage("Update the tray app to choose game folders.");
    return false;
  }
  if (!r.ok) {
    showGameError((await r.text()).trim() || `Error ${r.status}`);
    return false;
  }
  showGameError("");
  renderGames(await r.json());
  return true;
}

function renderGamesMessage(msg) {
  gameList.replaceChildren(el("span", "game-empty", msg));
  gameAddForm.hidden = true;
}

function folderName(path) {
  return path.replace(/[\\/]+$/, "").split(/[\\/]/).pop() || path;
}

function renderGames(games) {
  gameAddForm.hidden = false;
  if (!games.length) {
    gameList.replaceChildren(
      el("span", "game-empty", "No War Thunder install found. Add its folder below.")
    );
    return;
  }
  gameList.replaceChildren(
    ...games.map((game) => {
      const row = el("label", "game-row");
      row.title = game.path;
      if (!game.found) row.classList.add("game-missing");

      const box = document.createElement("input");
      box.type = "checkbox";
      box.checked = game.enabled;
      box.disabled = !game.found;
      box.addEventListener("change", async () => {
        box.disabled = true;
        const ok = await gameDirsRequest({
          path: game.path,
          enabled: box.checked,
        });
        if (!ok) {
          box.checked = !box.checked;
          box.disabled = false;
        }
      });

      const text = el("span", "game-text");
      const name = el("span", "game-name", folderName(game.path));
      const source = SOURCE_LABELS[game.source] ?? game.source;
      name.append(
        el("span", "game-source", game.found ? source : `${source}, missing`)
      );
      text.append(name, el("span", "game-path", game.path));
      row.append(box, text);

      if (game.source === "custom") {
        const remove = el("button", "icon-btn game-remove", "✕");
        remove.type = "button";
        remove.title = "Remove this folder";
        remove.addEventListener("click", (e) => {
          e.preventDefault();
          gameDirsRequest({ path: game.path, action: "remove" });
        });
        row.append(remove);
      }
      return row;
    })
  );
}

gameAddForm.addEventListener("submit", async (e) => {
  e.preventDefault();
  const path = gameAddPath.value.trim();
  if (!path) return;
  if (await gameDirsRequest({ path, action: "add" })) {
    gameAddPath.value = "";
  }
});

gameDirsRequest();

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