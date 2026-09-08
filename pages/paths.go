package pages

import "github.com/jackielii/hx-live-vs-alpine/examples"

// FramePath is the site path of one demo frame. Directory-style so a static
// host serves it as <path>/index.html without redirects.
func FramePath(lib examples.Lib, slug string) string {
	return "/frame/" + string(lib) + "/" + slug + "/"
}
