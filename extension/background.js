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
});