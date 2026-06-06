const BASE = "http://localhost:4316";

document.getElementById("ext-version").textContent =
  `v${browser.runtime.getManifest().version}`;

// Fetch server version on open
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

// Open folder buttons
async function openFolder(target, btn) {
  btn.disabled = true;
  try {
    await fetch(
      `${BASE}/openfolder?target=${encodeURIComponent(target)}`,
      { method: "POST" }
    );
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