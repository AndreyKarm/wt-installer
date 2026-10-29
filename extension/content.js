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

// Installed state: post id (string) -> "camo" | "sight", from the server
let installedPosts = {};

// Install button -> its trash-can uninstall button
const removeButtons = new WeakMap();

// Inline SVG icons, built with DOM calls rather than innerHTML (AMO review
// flags innerHTML in content scripts). They use currentColor, so they follow
// the button's text color.
const SVG_NS = "http://www.w3.org/2000/svg";
const ICONS = {
  check: {
    viewBox: "0 0 16 16",
    attrs: { fill: "currentColor" },
    d: "M13.293 3.293a1 1 0 1 1 1.414 1.414l-8 8a1 1 0 0 1-1.414 0l-4-4a1 1 0 1 1 1.414-1.414L6 10.586z",
  },
  trash: {
    viewBox: "0 0 24 24",
    attrs: {
      fill: "none",
      stroke: "currentColor",
      "stroke-linecap": "round",
      "stroke-linejoin": "round",
      "stroke-width": "2",
    },
    d: "M4 6h16l-1.58 14.22A2 2 0 0 1 16.432 22H7.568a2 2 0 0 1-1.988-1.78zm3.345-2.853A2 2 0 0 1 9.154 2h5.692a2 2 0 0 1 1.81 1.147L18 6H6zM2 6h20m-12 5v5m4-5v5",
  },
};

function createIcon(name) {
  const icon = ICONS[name];
  const svg = document.createElementNS(SVG_NS, "svg");
  svg.setAttribute("viewBox", icon.viewBox);
  svg.setAttribute("aria-hidden", "true");
  svg.setAttribute("class", `wt-local-icon wt-local-icon-${name}`);
  const path = document.createElementNS(SVG_NS, "path");
  for (const [key, value] of Object.entries(icon.attrs)) {
    path.setAttribute(key, value);
  }
  path.setAttribute("d", icon.d);
  svg.appendChild(path);
  return svg;
}

function showSpinner(label) {
  label.textContent = "";
  const spinner = document.createElement("span");
  spinner.className = "wt-local-dl-spinner";
  label.appendChild(spinner);
}

// Shows the idle label: "Installed" for posts already on disk, otherwise
// the install title. Clicking an installed post still reinstalls it; the
// trash can next to it uninstalls.
function renderIdle(btn) {
  if (btn.dataset.pending) return;
  const label = btn.querySelector(".num");
  const isInstalled = Boolean(installedPosts[btn.dataset.postId]);
  btn.classList.toggle("wt-local-installed", isInstalled);
  btn.style.color = "";
  const removeBtn = removeButtons.get(btn);
  if (removeBtn) removeBtn.hidden = !isInstalled;
  if (isInstalled) {
    label.replaceChildren(createIcon("check"), "Installed");
    btn.title = "Already installed. Click to install again";
  } else {
    label.textContent = btn.dataset.defaultTitle;
    btn.title = btn.dataset.defaultTitle;
  }
}

async function refreshInstalled() {
  const result = await browser.runtime
    .sendMessage({ action: "wt_installed" })
    .catch(() => null);
  // Server offline: keep the last known state rather than clearing marks
  if (!result?.ok) return;
  installedPosts = result.installed ?? {};
  document.querySelectorAll(`.${BTN_MARKER}`).forEach(renderIdle);
}

// Shared button factory: returns the install button and its uninstall
// button, which callers insert right after it.
function createDownloadButtons(downloadType, postId, fileUrl) {
  const type = TYPE_MAP[downloadType];
  const defaultTitle = TITLE_MAP[downloadType] ?? "Install";

  const btn = document.createElement("a");
  btn.className = `button downloads button_item ${BTN_MARKER}`;
  btn.href = "#";
  btn.dataset.postId = String(postId);
  btn.dataset.defaultTitle = defaultTitle;

  const label = document.createElement("span");
  label.className = "num wt-local-label";
  btn.appendChild(label);

  const removeBtn = document.createElement("a");
  removeBtn.className = "button button_item wt-local-uninstall";
  removeBtn.href = "#";
  removeBtn.title = "Uninstall: delete its files from the game folders";
  removeBtn.setAttribute("aria-label", "Uninstall");
  removeBtn.appendChild(createIcon("trash"));
  removeButtons.set(btn, removeBtn);
  renderIdle(btn);

  removeBtn.addEventListener("click", async (e) => {
    e.preventDefault();
    if (btn.dataset.pending) return;
    if (
      !confirm(
        `Uninstall ${type} post #${postId}?\nThis deletes its files from your game folders.`
      )
    ) {
      return;
    }

    btn.dataset.pending = "true";
    removeBtn.hidden = true;
    showSpinner(label);

    const result = await browser.runtime.sendMessage({
      action: "wt_uninstall",
      postId,
    });
    delete btn.dataset.pending;

    if (result?.ok) {
      delete installedPosts[String(postId)];
      const detail = result.message ? `: ${result.message}` : "";
      showToast(`Uninstalled ${type} post #${postId}${detail}`, "success");
      wtLog("info", `Uninstalled ${type} post ${postId}${detail}`);
    } else {
      const reason =
        result?.error || result?.message || `HTTP ${result?.status}`;
      showToast(`Uninstall failed for post #${postId}: ${reason}`, "error");
      wtLog("error", `Failed to uninstall post ${postId}: ${reason}`);
    }
    renderIdle(btn);
  });

  btn.addEventListener("click", async (e) => {
    e.preventDefault();

    if (btn.dataset.pending) return;

    btn.dataset.pending = "true";
    removeBtn.hidden = true;
    showSpinner(label);

    const result = await browser.runtime.sendMessage({
      action: "wt_install",
      type,
      postId,
      fileUrl,
    });

    delete btn.dataset.pending;
    label.textContent = "";

    if (result?.ok) {
      installedPosts[String(postId)] = type;
      label.textContent = "Installed";
      btn.style.color = "#4caf50";
      const detail = result.message ? `: ${result.message}` : "";
      showToast(`${type} post #${postId}${detail}`, "success");
      wtLog("info", `${type} downloaded for post ${postId}${detail}`);
    } else {
      label.textContent = "Failed";
      btn.style.color = "#f44336";
      const reason =
        result?.error || result?.message || `HTTP ${result?.status}`;
      showToast(
        `Download ${type} failed for post #${postId}: ${reason}`,
        "error"
      );
      wtLog(
        "error",
        `Failed to download ${type} for post ${postId}: ${reason}`
      );
    }

    setTimeout(() => renderIdle(btn), 2000);
  });

  return [btn, removeBtn];
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

  leftButtons.append(...createDownloadButtons(downloadType, postId, fileUrl));
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

  existingDownload.after(...createDownloadButtons(downloadType, postId, fileUrl));
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
refreshInstalled();

// Pick up installs from other tabs and skins deleted by hand
document.addEventListener("visibilitychange", () => {
  if (document.visibilityState === "visible") refreshInstalled();
});

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