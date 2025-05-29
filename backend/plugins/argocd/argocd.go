package main

import (
	impl "github.com/apache/incubator-devlake/plugins/argocd/impl"
)

// Plugin entry point for ArgoCD
var PluginEntry impl.ArgoCDPlugin

func main() {
	// Plugin entry point - needed for go build with plugin mode
}
