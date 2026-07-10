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

// Shared button factory
function createDownloadButton(downloadType, postId, fileUrl) {
  const type = TYPE_MAP[downloadType];
  const defaultTitle = TITLE_MAP[downloadType] ?? "Install";

  const btn = document.createElement("a");
  btn.className = `button downloads button_item ${BTN_MARKER}`;
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
      showToast(
        `Download ${type} failed for post #${postId}: ${reason}`,
        "error"
      );
      wtLog(
        "error",
        `Failed to download ${type} for post ${postId}: ${reason}`
      );
    }

    setTimeout(() => {
      label.textContent = defaultTitle;
      btn.style.color = "";
    }, 2000);
  });

  return btn;
}

function getDownloadType(el) {
  return Object.keys(TYPE_MAP).find((cls) => el.classList.contains(cls));
}

// Button Injection — feed list
function addButton(post) {
  if (post.querySelector(`.${BTN_MARKER}`)) return;

  const postId = parseInt(post.getAttribute(POST_ATTR), 10);
  if (!postId) return;

  const downloadType = getDownloadType(post);
  if (!downloadType) return;

  const leftButtons = post.querySelector(".bottom .buttons .left");
  if (!leftButtons) return;

  const existingDownload = leftButtons.querySelector(
    `.downloads.button_item:not(.${BTN_MARKER})`
  );
  const fileUrl = existingDownload?.href ?? null;

  const btn = createDownloadButton(downloadType, postId, fileUrl);
  leftButtons.appendChild(btn);
}

// Button Injection — fullview / lightbox popup
function addPopupButton(wrapper) {
  const container = wrapper.querySelector(".description .buttons");
  if (!container || container.querySelector(`.${BTN_MARKER}`)) return;

  const postId = parseInt(wrapper.getAttribute(POST_ATTR), 10);
  if (!postId) return;

  const downloadType = getDownloadType(wrapper);
  if (!downloadType) return;

  const existingDownload = container.querySelector(
    "a.button.downloadButton"
  );
  if (!existingDownload) return;

  const fileUrl = existingDownload.href ?? null;

  const btn = createDownloadButton(downloadType, postId, fileUrl);
  existingDownload.insertAdjacentElement("afterend", btn);
}

// Observer — feed
function processPosts() {
  document.querySelectorAll(`.feed_item[${POST_ATTR}]`).forEach(addButton);
}

// Observer — fullview popup
function processPopup() {
  document
    .querySelectorAll(`#clb .wrapper[${POST_ATTR}]`)
    .forEach(addPopupButton);
}

processPosts();
processPopup();

const feedObserver = new MutationObserver(processPosts);
feedObserver.observe(document.querySelector("#feedwrapper") ?? document.body, {
  childList: true,
  subtree: true,
});

const popupObserver = new MutationObserver(processPopup);
const clb = document.getElementById("clb");
if (clb) {
  popupObserver.observe(clb, { childList: true, subtree: true });
}

// Logs
function wtLog(level, ...args) {
  const message = args.map(String).join(" ");
  if (level === "error") console.error("[WT]", message);
  else console.log("[WT]", message);
  browser.runtime
    .sendMessage({
      action: "wt_add_log",
      level,
      source: "content",
      message,
    })
    .catch(() => { });
}