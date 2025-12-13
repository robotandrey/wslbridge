package platform

import "wslbridge/internal/execx"

type Platform interface {
	Name() string
	EnsureDeps(r execx.Runner) error
}

func Detect() Platform {
	return Ubuntu{}
}
