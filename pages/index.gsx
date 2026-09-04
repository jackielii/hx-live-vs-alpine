package pages

import (
	"github.com/gsxhq/gsx"
	"github.com/gsxhq/vite"

	"github.com/jackielii/hx-live-vs-alpine/examples"
	"github.com/jackielii/hx-live-vs-alpine/ui"
)

// Index is the comparison page: a nav of anchors and one card per example
// with the Alpine and hx-live demos side by side.
component Index(exs []examples.Example) {
	{{ v := vite.FromContext(ctx) }}
	{{ assets := v.Entry("web/main.js") }}
	<!DOCTYPE html>
	<html lang="en">
		<head>
			<meta charset="UTF-8"/>
			<meta name="viewport" content="width=device-width, initial-scale=1.0"/>
			<title>hx-live vs Alpine</title>
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
		<body class="bg-background text-foreground">
			<div class="mx-auto grid max-w-7xl grid-cols-[220px_1fr] gap-10 px-6 py-10">
				<nav class="sticky top-10 flex h-fit flex-col gap-1">
					<h1 class="mb-3 text-lg font-semibold">hx-live vs Alpine</h1>
					{ for _, ex := range exs {
						<ui.Button variant="ghost" href={"#" + ex.Slug} class="justify-start">{ ex.Title }</ui.Button>
					} }
				</nav>
				<main class="flex flex-col gap-10">
					<p class="text-muted-foreground max-w-prose">
						Six examples from the Alpine.js docs, each ported like-for-like to htmx 4's hx-live extension.
						Every demo runs in its own iframe loading only its library. The source under each demo is the
						exact fragment inside that iframe.
					</p>
					{ for _, ex := range exs {
						<ui.Card id={ex.Slug}>
							<ui.CardHeader>
								<ui.CardTitle>{ ex.Title }</ui.CardTitle>
								<ui.CardDescription class="prose prose-sm max-w-none">{ gsx.Raw(ex.NotesHTML) }</ui.CardDescription>
							</ui.CardHeader>
							<ui.CardContent class="grid grid-cols-2 gap-6 px-4">
								<column lib={examples.Alpine} ex={ex}/>
								<column lib={examples.HxLive} ex={ex}/>
							</ui.CardContent>
						</ui.Card>
					} }
				</main>
			</div>
		</body>
	</html>
}

// column is one side of a card: badge, live iframe, escaped source.
component column(lib examples.Lib, ex examples.Example) {
	{{ src := "/frame/" + string(lib) + "/" + ex.Slug }}
	<div class="flex min-w-0 flex-col gap-3">
		<ui.Badge variant="secondary">{ lib.Label() }</ui.Badge>
		<iframe
			data-frame
			src={src}
			title={ex.Title + " — " + lib.Label()}
			class="w-full rounded-md border bg-white"
			style="height: 120px"
		></iframe>
		<pre class="overflow-x-auto rounded-md bg-muted p-3"><code>{ ex.Fragment(lib) }</code></pre>
	</div>
}
