browser.runtime.onMessage.addListener((message) => {
  if (message.type === "download_post") {
    return fetch(`http://localhost:4316/post/${message.postId}`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ fileUrl: message.fileUrl }),
    })
      .then((res) => ({ ok: res.ok, status: res.status }))
      .catch((err) => ({ ok: false, error: err.message }));
  };
});