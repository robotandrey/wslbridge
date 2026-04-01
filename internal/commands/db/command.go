package dbcmd

import (
	"fmt"

	"wslbridge/internal/db"
	appruntime "wslbridge/internal/runtime"
)

// Command implements db-related CLI actions.
type Command struct{}

// Name returns the command name.
func (Command) Name() string { return "db" }

// Help returns the command description.
func (Command) Help() string {
	return "Manage service-discovery-driven local DB proxy (init|start|status|stop|add|remove)"
}

// Run executes db command.
func (Command) Run(rt appruntime.Runtime, args []string) error {
	svc := db.NewService(rt)

	if len(args) == 0 {
		return svc.Status()
	}

	switch args[0] {
	case "init":
		force := false
		for _, a := range args[1:] {
			switch a {
			case "--force":
				force = true
			default:
				return fmt.Errorf("unknown arg: %s", a)
			}
		}
		return svc.Init(force)
	case "start":
		force := false
		for _, a := range args[1:] {
			switch a {
			case "--force":
				force = true
			default:
				return fmt.Errorf("unknown arg: %s", a)
			}
		}
		return svc.Start(force)
	case "status":
		if len(args) > 1 {
			return fmt.Errorf("unknown arg: %s", args[1])
		}
		return svc.Status()
	case "stop":
		if len(args) > 1 {
			return fmt.Errorf("unknown arg: %s", args[1])
		}
		return svc.Stop()
	case "add":
		if len(args) > 2 {
			return fmt.Errorf("too many args for add")
		}
		service := ""
		if len(args) == 2 {
			service = args[1]
		}
		return svc.AddService(service)
	case "remove", "rm", "delete":
		if len(args) != 2 {
			return fmt.Errorf("usage: db remove <service>")
		}
		return svc.RemoveService(args[1])
	default:
		return fmt.Errorf("unknown action: %s (use: init | start | status | stop | add | remove)", args[0])
	}
}
