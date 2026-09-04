// Runs inside each demo iframe. Reports the document height to the parent so
// the outer page can size the iframe to its content.
function report() {
  const height = Math.ceil(document.documentElement.getBoundingClientRect().height);
  window.parent.postMessage({ type: "hx-vs-alpine:height", height }, "*");
}
new ResizeObserver(report).observe(document.documentElement);
window.addEventListener("load", report);
report();
