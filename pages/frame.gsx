// Package pages holds the gsx pages. Each renders its own <html> shell because
// Go cannot import package main.
package pages

import (
	"github.com/gsxhq/gsx"
	"github.com/gsxhq/vite"

	"github.com/jackielii/hx-live-vs-alpine/examples"
)

// Frame is the document loaded inside one demo iframe. It loads exactly one
// library's Vite entry and injects the raw fragment unchanged.
component Frame(lib examples.Lib, ex examples.Example) {
	{{ v := vite.FromContext(ctx) }}
	{{ assets := v.Entry(lib.Entry()) }}
	<!DOCTYPE html>
	<html lang="en">
		<head>
			<meta charset="UTF-8"/>
			<meta name="viewport" content="width=device-width, initial-scale=1.0"/>
			<title>{ ex.Title } — { lib.Label() }</title>
			{ for _, href := range assets.CSS {
				<link rel="stylesheet" href={href}/>
			} }
			{ for _, src := range assets.Preloads {
				<link rel="modulepreload" href={src}/>
			} }
			{ for _, src := range assets.JS {
				<script type="module" src={src}></script>
			} }
		</head>
		<body>
			{ gsx.Raw(ex.Fragment(lib)) }
		</body>
	</html>
}
