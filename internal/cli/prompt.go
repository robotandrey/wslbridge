package cli

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

type Prompter struct {
	in  *bufio.Reader
	out io.Writer
}

func NewPrompter(in io.Reader, out io.Writer) *Prompter {
	return &Prompter{in: bufio.NewReader(in), out: out}
}

func (p *Prompter) AskString(label, def, current string, validate func(string) error) (string, error) {
	for {
		if current != "" {
			_, _ = fmt.Fprintf(p.out, "%s [default: %s] (current: %s): ", label, def, current)
		} else {
			_, _ = fmt.Fprintf(p.out, "%s [default: %s]: ", label, def)
		}

		line, err := p.in.ReadString('\n')
		if err != nil {
			return "", err
		}
		val := strings.TrimSpace(line)

		if val == "" {
			if current != "" {
				val = current
			} else {
				val = def
			}
		}

		if validate != nil {
			if err := validate(val); err != nil {
				_, _ = fmt.Fprintf(p.out, "invalid value: %v\n", err)
				continue
			}
		}
		return val, nil
	}
}
