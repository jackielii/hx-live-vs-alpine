import "./frame.css";
import "./frame-resize.js";
// Order matters: hx-live.js reads the global `htmx` at eval time.
import "./htmx-setup.js";
import "htmx.org/dist/ext/hx-live.js";
