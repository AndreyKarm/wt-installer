const EXT_VERSION = browser.runtime.getManifest().version;
const POST_ATTR = "post_id";
const BTN_MARKER = "local-dl-btn";
const TYPE_MAP = {
  camouflage: "camo",
  sight: "sight",
};
const TITLE_MAP = {
  camouflage: "Install Camo",
  sight: "Install Sight",
};

// Toast
function getToastContainer() {
  const existing = document.getElementById("wt-local-toast-container");
  if (existing) return existing;
  const container = document.createElement("div");
  container.id = "wt-local-toast-container";
  document.body.appendChild(container);
  return container;
}

/**
 * @param {string} message
 * @param {"success"|"error"|"info"} type
 * @param {number} duration ms before auto-dismiss
 */
function showToast(message, type = "info", duration = 4000) {
  const container = getToastContainer();

  const toast = document.createElement("div");
  toast.className = `wt-local-toast ${type}`;
  toast.textContent = message;
  container.appendChild(toast);

  requestAnimationFrame(() => {
    requestAnimationFrame(() => toast.classList.add("visible"));
  });

  setTimeout(() => {
    toast.classList.remove("visible");
    toast.addEventListener("transitionend", () => toast.remove(), {
      once: true,
    });
  }, duration);
}


document.addEventListener("keydown", (e) => {
  if (e.ctrlKey && e.shiftKey && e.key === "Y") {
    e.preventDefault();
    const types = ["info", "success", "error"];
    types.forEach((t, i) =>
      setTimeout(() => showToast(`Test toast (${t})`, t), i * 600)
    );
  }
});

// Button Injection
function addButton(post) {
  if (post.querySelector(`.${BTN_MARKER}`)) return;

  const postId = parseInt(post.getAttribute(POST_ATTR), 10);
  if (!postId) return;

  const downloadType = Object.keys(TYPE_MAP).find((cls) =>
    post.classList.contains(cls)
  );
  if (!downloadType) return;

  const defaultTitle = TITLE_MAP[downloadType] ?? "Install";

  const leftButtons = post.querySelector(".bottom .buttons .left");
  if (!leftButtons) return;

  const existingDownload = leftButtons.querySelector(
    `.downloads.button_item:not(.${BTN_MARKER})`
  );
  const fileUrl = existingDownload?.href ?? null;

  const btn = document.createElement("a");
  btn.className = `downloads button_item ${BTN_MARKER}`;
  btn.title = defaultTitle;
  btn.href = "#";

  const label = document.createElement("span");
  label.className = "num";
  label.textContent = defaultTitle;
  btn.appendChild(label);

  btn.addEventListener("click", async (e) => {
    e.preventDefault();

    if (btn.dataset.pending) return;

    btn.dataset.pending = "true";
    label.textContent = "";
    const spinner = document.createElement("span");
    spinner.className = "wt-local-dl-spinner";
    label.appendChild(spinner);

    const type = TYPE_MAP[downloadType];

    const result = await browser.runtime.sendMessage({
      action: "wt_install",
      type,
      postId,
      fileUrl,
    });

    delete btn.dataset.pending;
    label.textContent = "";

    if (result?.ok) {
      label.textContent = "Installed";
      btn.style.color = "#4caf50";
      showToast(`Successfully installed ${type} post #${postId}`, "success");
      wtLog("info", `${type} downloaded for post ${postId}`);
    } else {
      label.textContent = "Failed";
      btn.style.color = "#f44336";
      const reason = result?.error ?? `HTTP ${result?.status}`;
      showToast(`Download ${type} failed for post #${postId}: ${reason}`, "error");
      wtLog("error", `Failed to download ${type} for post ${postId}: ${reason}`);
    }

    setTimeout(() => {
      label.textContent = defaultTitle;
      btn.style.color = "";
    }, 2000);
  });

  leftButtons.appendChild(btn);
}

// Observer
function processPosts() {
  document
    .querySelectorAll(`.feed_item[${POST_ATTR}]`)
    .forEach(addButton);
}

processPosts();

const observer = new MutationObserver(processPosts);
observer.observe(
  document.querySelector("#feedwrapper") ?? document.body,
  { childList: true, subtree: true }
);

// Logs
function wtLog(level, ...args) {
  const message = args.map(String).join(" ");
  if (level === "error") console.error("[WT]", message);
  else console.log("[WT]", message);
  browser.runtime.sendMessage({
    action: "wt_add_log",
    level,
    source: "content",
    message,
  }).catch(() => { });
}