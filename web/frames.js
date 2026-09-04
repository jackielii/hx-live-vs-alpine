// Runs in the outer page. Each demo iframe posts its content height; match the
// sender to its iframe and set the height so nothing clips.
window.addEventListener("message", (e) => {
  if (!e.data || e.data.type !== "hx-vs-alpine:height") return;
  const h = e.data.height;
  if (!Number.isFinite(h) || h <= 0) return;
  for (const frame of document.querySelectorAll("iframe[data-frame]")) {
    if (frame.contentWindow === e.source) frame.style.height = h + "px";
  }
});
