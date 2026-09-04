package pages

import "github.com/jackielii/hx-live-vs-alpine/examples"

// Index is the comparison page. Filled in by the next task.
component Index(exs []examples.Example) {
	<!DOCTYPE html>
	<html lang="en"><body><p>{ len(exs) } examples</p></body></html>
}
