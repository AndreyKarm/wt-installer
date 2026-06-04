const POST_ATTR = "post_id";
const BTN_MARKER = "local-dl-btn";

function addButton(post) {
  if (post.querySelector(`.${BTN_MARKER}`)) return;

  const postId = post.getAttribute(POST_ATTR);
  if (!postId) return;

  const leftButtons = post.querySelector(".bottom .buttons .left");
  if (!leftButtons) return;

  // Grab the existing download button's href before we inject ours
  const existingDownload = leftButtons.querySelector(
    `.downloads.button_item:not(.${BTN_MARKER})`
  );
  const fileUrl = existingDownload?.href ?? null;

  const btn = document.createElement("a");
  btn.className = `downloads button_item ${BTN_MARKER}`;
  btn.title = "Download localy";
  btn.href = "#";

  const label = document.createElement("span");
  label.className = "num";
  label.textContent = "Download localy";
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
      type: "download_post",
      postId,
      fileUrl,
    });

    delete btn.dataset.pending;
    label.textContent = "";

    if (result?.ok) {
      label.textContent = "Downloaded";
      btn.style.color = "#4caf50";
    } else {
      label.textContent = "Failed";
      btn.style.color = "#f44336";
      console.error(
        "[WT Local Downloader] Failed for post",
        postId,
        result?.error ?? `HTTP ${result?.status}`
      );
    }

    setTimeout(() => {
      label.textContent = "Download localy";
      btn.style.color = "";
    }, 2000);
  });

  leftButtons.appendChild(btn);
}

function processPosts() {
  document
    .querySelectorAll(`.feed_item[${POST_ATTR}]`)
    .forEach(addButton);
}

// Handle posts already on the page
processPosts();

// Handle dynamically loaded posts (infinite scroll)
const observer = new MutationObserver(processPosts);
observer.observe(
  document.querySelector("#feedwrapper") ?? document.body,
  { childList: true, subtree: true }
);