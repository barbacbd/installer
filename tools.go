//go:build tools
// +build tools

// Official workaround to track tool dependencies with go modules:
// https://github.com/golang/go/wiki/Modules#how-can-i-track-tool-dependencies-for-a-module

// TODO: go 1.23 (added for grep) should be the last version where we need to do this. With
// go 1.24 we should be able to use the go tool command https://tip.golang.org/doc/go1.24#go-command

package tools

import (
	_ "github.com/daixiang0/gci" // dependency of hack/go-fmt.sh
	// used to generate mocks
	_ "go.uber.org/mock/mockgen"
	// dependency of generating CRD for install-config
	_ "sigs.k8s.io/controller-tools/cmd/controller-gen"
)
