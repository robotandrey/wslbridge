package command

import (
	"fmt"
	"sort"
)

type Registry struct{ m map[string]Command }

func New(cmds ...Command) Registry {
	r := Registry{m: map[string]Command{}}
	for _, c := range cmds {
		r.m[c.Name()] = c
	}
	return r
}

func (r Registry) Get(name string) (Command, bool) {
	c, ok := r.m[name]
	return c, ok
}

func (r Registry) HelpLines() []string {
	names := make([]string, 0, len(r.m))
	for k := range r.m {
		names = append(names, k)
	}
	sort.Strings(names)

	lines := make([]string, 0, len(names))
	for _, n := range names {
		lines = append(lines, fmt.Sprintf("  %-16s %s", n, r.m[n].Help()))
	}
	return lines
}
