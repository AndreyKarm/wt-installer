const MAX_EXT_LOGS = 200;
let extLogs = [];

function addExtLog(level, source, message) {
  extLogs.push({
    time: new Date().toLocaleTimeString("en-GB", { hour12: false }),
    level,
    source,
    message: String(message),
  });
  if (extLogs.length > MAX_EXT_LOGS) {
    extLogs = extLogs.slice(-MAX_EXT_LOGS);
  }
}

// Capture background script's own console output
const _log = console.log.bind(console);
const _warn = console.warn.bind(console);
const _error = console.error.bind(console);
console.log = (...args) => {
  addExtLog("info", "bg", args.join(" "));
  _log(...args);
};
console.warn = (...args) => {
  addExtLog("warn", "bg", args.join(" "));
  _warn(...args);
};
console.error = (...args) => {
  addExtLog("error", "bg", args.join(" "));
  _error(...args);
};

browser.runtime.onMessage.addListener((message) => {
  if (message.action === "wt_install") {
    return fetch(`http://localhost:4316/download`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        type: message.type,
        postId: message.postId,
        fileUrl: message.fileUrl,
      }),
    })
      .then((r) => ({ ok: r.ok, status: r.status }))
      .catch((err) => ({ ok: false, error: err.message }));
  }

  if (message.action === "wt_server_version") {
    return fetch("http://localhost:4316/version")
      .then((r) => r.text())
      .then((text) => ({ ok: true, version: text.replace("Version: ", "") }))
      .catch((err) => ({ ok: false, error: err.message }));
  }

  if (message.action === "wt_open_folder") {
    return fetch(
      `http://localhost:4316/openfolder?target=${encodeURIComponent(message.target)}`,
      { method: "POST" }
    )
      .then((r) => ({ ok: r.ok, status: r.status }))
      .catch((err) => ({ ok: false, error: err.message }));
  }

  if (message.action === "wt_add_log") {
    addExtLog(
      message.level ?? "info",
      message.source ?? "content",
      message.message
    );
    return Promise.resolve({ ok: true });
  }

  if (message.action === "wt_get_ext_logs") {
    return Promise.resolve({ ok: true, logs: [...extLogs] });
  }

  if (message.action === "wt_clear_ext_logs") {
    extLogs = [];
    return Promise.resolve({ ok: true });
  }
})