package pages

import (
	"github.com/gsxhq/gsx"
	"github.com/gsxhq/vite"

	"github.com/jackielii/hx-live-vs-alpine/examples"
	"github.com/jackielii/hx-live-vs-alpine/site"
	"github.com/jackielii/hx-live-vs-alpine/ui"
)

// Index is the comparison page: a feature matrix, a grouped nav, and one card
// per example with the Alpine and hx-live demos side by side.
component Index(exs []examples.Example, feats []examples.FeatureRow) {
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
				<nav class="sticky top-10 flex h-fit max-h-[calc(100vh-5rem)] flex-col gap-1 overflow-y-auto">
					<h1 class="text-lg font-semibold">hx-live vs Alpine</h1>
					<a class="mb-3 text-xs text-muted-foreground underline" href="https://github.com/jackielii/hx-live-vs-alpine">[source]</a>
					<ui.Button variant="ghost" size="sm" href="#matrix" class="h-auto justify-start whitespace-normal text-left">Feature matrix</ui.Button>
					{ for _, g := range examples.Groups {
						<h2 class="mt-3 mb-1 text-xs font-semibold uppercase text-muted-foreground">{ g.Label() }</h2>
						{ for _, ex := range examples.ByGroup(exs, g) {
							<ui.Button variant="ghost" size="sm" href={"#" + ex.Slug} class="h-auto justify-start whitespace-normal text-left">{ ex.Title }</ui.Button>
						} }
					} }
				</nav>
				<main class="flex flex-col gap-10">
					<p class="text-muted-foreground max-w-prose">
						Every Alpine.js core feature, each with its docs example and a like-for-like htmx 4 hx-live port
						where one exists. Every demo runs in its own iframe loading only its library. The source under each
						demo is the exact fragment inside that iframe.
					</p>
					<matrix feats={feats}/>
					{ for _, ex := range exs {
						<ui.Card id={ex.Slug}>
							<ui.CardHeader>
								<ui.CardTitle class="flex items-center gap-2">
									{ ex.Title }
									<statusBadge status={ex.Status}/>
									<a class="text-xs font-normal text-muted-foreground underline" href={"https://github.com/jackielii/hx-live-vs-alpine/tree/main/examples/" + ex.Slug}>[source]</a>
								</ui.CardTitle>
								{ if ex.HasDemo() {
									<ui.CardDescription class="space-y-2">{ gsx.Raw(ex.NotesHTML) }</ui.CardDescription>
								} }
							</ui.CardHeader>
							<ui.CardContent class="grid grid-cols-2 gap-6 px-4">
								<column lib={examples.Alpine} ex={ex}/>
								{ if ex.HasDemo() {
									<column lib={examples.HxLive} ex={ex}/>
								} else {
									<noEquivalent ex={ex}/>
								} }
							</ui.CardContent>
						</ui.Card>
					} }
				</main>
			</div>
		</body>
	</html>
}

// matrix is the feature table: one line per Alpine feature, grouped.
component matrix(feats []examples.FeatureRow) {
	<table id="matrix" class="w-full text-sm">
		<thead>
			<tr class="text-left text-muted-foreground">
				<th class="py-1 pr-4 font-medium">Feature</th>
				<th class="py-1 pr-4 font-medium">Status</th>
				<th class="py-1 font-medium">Card</th>
			</tr>
		</thead>
		{ for _, g := range examples.Groups {
			<tbody>
				<tr>
					<th colspan="3" class="pt-4 pb-1 text-left text-xs font-semibold uppercase text-muted-foreground">{ g.Label() }</th>
				</tr>
				{ for _, f := range feats {
					{ if f.Group == g {
						<tr class="border-t">
							<td class="py-1 pr-4 font-mono text-xs">{ f.Feature }</td>
							<td class="py-1 pr-4"><statusBadge status={f.Status}/></td>
							<td class="py-1"><a class="underline" href={"#" + f.Slug}>{ f.Title }</a></td>
						</tr>
					} }
				} }
			</tbody>
		} }
	</table>
}

// statusBadge maps a status to a badge variant.
component statusBadge(status examples.Status) {
	{ if status == examples.Equivalent {
		<ui.Badge>{ string(status) }</ui.Badge>
	} else {
		{ if status == examples.Workaround {
			<ui.Badge variant="secondary">{ string(status) }</ui.Badge>
		} else {
			<ui.Badge variant="outline">{ string(status) }</ui.Badge>
		} }
	} }
}

// column is one side of a card: badge, live iframe, escaped source.
component column(lib examples.Lib, ex examples.Example) {
	{{ src := site.URL(ctx, FramePath(lib, ex.Slug)) }}
	<div class="flex min-w-0 flex-col gap-3">
		<ui.Badge variant="secondary">{ lib.Label() }</ui.Badge>
		<iframe
			data-frame
			src={src}
			title={ex.Title + " — " + lib.Label()}
			class="w-full rounded-md border bg-white box-content"
			style="height: 120px"
		></iframe>
		<pre class="overflow-x-auto rounded-md bg-muted p-3"><code>{ ex.Fragment(lib) }</code></pre>
	</div>
}

// noEquivalent is the right column of a card whose feature hx-live lacks.
component noEquivalent(ex examples.Example) {
	<div class="flex min-w-0 flex-col gap-3">
		<ui.Badge variant="outline">No hx-live equivalent</ui.Badge>
		<div class="space-y-2 text-sm text-muted-foreground">{ gsx.Raw(ex.NotesHTML) }</div>
	</div>
}
