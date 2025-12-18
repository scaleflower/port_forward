package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"pfm/internal/daemon"
	"pfm/internal/ipc"
	"pfm/internal/models"
)

// Run executes the CLI command
func Run(args []string) error {
	if len(args) < 1 {
		return showHelp()
	}

	cmd := args[0]
	subArgs := args[1:]

	switch cmd {
	case "service":
		return handleService(subArgs)
	case "rule", "rules":
		return handleRule(subArgs)
	case "chain", "chains":
		return handleChain(subArgs)
	case "status":
		return handleStatus()
	case "version":
		return handleVersion()
	case "help", "-h", "--help":
		return showHelp()
	default:
		return fmt.Errorf("unknown command: %s\nRun 'pfm help' for usage", cmd)
	}
}

func showHelp() error {
	help := `Port Forward Manager - CLI

Usage:
  pfm <command> [arguments]

Commands:
  service     Manage the background service
  rule        Manage port forwarding rules
  chain       Manage proxy chains
  status      Show service and rules status
  version     Show version information
  help        Show this help message

Service Commands:
  pfm service run         Run as foreground service (for systemd/init)
  pfm service install     Install as system service
  pfm service uninstall   Uninstall system service
  pfm service status      Show service status

Rule Commands:
  pfm rule list                    List all rules
  pfm rule show <id>               Show rule details
  pfm rule start <id>              Start a rule
  pfm rule stop <id>               Stop a rule
  pfm rule delete <id>             Delete a rule
  pfm rule create <json>           Create a rule from JSON

Chain Commands:
  pfm chain list                   List all chains
  pfm chain show <id>              Show chain details
  pfm chain delete <id>            Delete a chain

Examples:
  pfm service install              # Install and enable service
  pfm rule list                    # List all forwarding rules
  pfm rule start abc123            # Start rule with ID abc123
  pfm status                       # Show overall status
`
	fmt.Println(help)
	return nil
}

func handleService(args []string) error {
	if len(args) < 1 {
		fmt.Println("Usage: pfm service <run|install|uninstall|status>")
		return nil
	}

	switch args[0] {
	case "run":
		// This is handled in main.go directly
		return fmt.Errorf("service run should be handled by main")
	case "install":
		fmt.Println("Installing service...")
		if err := daemon.Install(); err != nil {
			return fmt.Errorf("failed to install service: %w", err)
		}
		fmt.Println("Service installed successfully")
		fmt.Println("Starting service...")
		if err := daemon.Start(); err != nil {
			return fmt.Errorf("service installed but failed to start: %w", err)
		}
		fmt.Println("Service started successfully")
		return nil
	case "uninstall":
		fmt.Println("Uninstalling service...")
		if err := daemon.Uninstall(); err != nil {
			return fmt.Errorf("failed to uninstall service: %w", err)
		}
		fmt.Println("Service uninstalled successfully")
		return nil
	case "status":
		if daemon.IsInstalled() {
			if daemon.IsRunning() {
				fmt.Println("Service: installed and running")
			} else {
				fmt.Println("Service: installed but stopped")
			}
		} else {
			fmt.Println("Service: not installed")
		}
		return nil
	default:
		return fmt.Errorf("unknown service command: %s", args[0])
	}
}

func handleRule(args []string) error {
	if len(args) < 1 {
		fmt.Println("Usage: pfm rule <list|show|start|stop|delete|create> [args]")
		return nil
	}

	client := ipc.NewClient()
	if err := client.Connect(); err != nil {
		return fmt.Errorf("failed to connect to service: %w\nIs the service running?", err)
	}
	defer client.Close()

	switch args[0] {
	case "list", "ls":
		rules, err := client.GetRules()
		if err != nil {
			return fmt.Errorf("failed to get rules: %w", err)
		}
		printRules(rules)
		return nil

	case "show", "get":
		if len(args) < 2 {
			return fmt.Errorf("usage: pfm rule show <id>")
		}
		rule, err := client.GetRule(args[1])
		if err != nil {
			return fmt.Errorf("failed to get rule: %w", err)
		}
		printRuleDetail(rule)
		return nil

	case "start":
		if len(args) < 2 {
			return fmt.Errorf("usage: pfm rule start <id>")
		}
		if err := client.StartRule(args[1]); err != nil {
			return fmt.Errorf("failed to start rule: %w", err)
		}
		fmt.Printf("Rule %s started\n", args[1])
		return nil

	case "stop":
		if len(args) < 2 {
			return fmt.Errorf("usage: pfm rule stop <id>")
		}
		if err := client.StopRule(args[1]); err != nil {
			return fmt.Errorf("failed to stop rule: %w", err)
		}
		fmt.Printf("Rule %s stopped\n", args[1])
		return nil

	case "delete", "rm":
		if len(args) < 2 {
			return fmt.Errorf("usage: pfm rule delete <id>")
		}
		if err := client.DeleteRule(args[1]); err != nil {
			return fmt.Errorf("failed to delete rule: %w", err)
		}
		fmt.Printf("Rule %s deleted\n", args[1])
		return nil

	case "create", "add":
		if len(args) < 2 {
			return fmt.Errorf("usage: pfm rule create '<json>'")
		}
		var rule models.Rule
		if err := json.Unmarshal([]byte(args[1]), &rule); err != nil {
			return fmt.Errorf("invalid JSON: %w", err)
		}
		id, err := client.CreateRule(&rule)
		if err != nil {
			return fmt.Errorf("failed to create rule: %w", err)
		}
		fmt.Printf("Rule created with ID: %s\n", id)
		return nil

	default:
		return fmt.Errorf("unknown rule command: %s", args[0])
	}
}

func handleChain(args []string) error {
	if len(args) < 1 {
		fmt.Println("Usage: pfm chain <list|show|delete> [args]")
		return nil
	}

	client := ipc.NewClient()
	if err := client.Connect(); err != nil {
		return fmt.Errorf("failed to connect to service: %w\nIs the service running?", err)
	}
	defer client.Close()

	switch args[0] {
	case "list", "ls":
		chains, err := client.GetChains()
		if err != nil {
			return fmt.Errorf("failed to get chains: %w", err)
		}
		printChains(chains)
		return nil

	case "show", "get":
		if len(args) < 2 {
			return fmt.Errorf("usage: pfm chain show <id>")
		}
		chains, err := client.GetChains()
		if err != nil {
			return fmt.Errorf("failed to get chains: %w", err)
		}
		for _, c := range chains {
			if c.ID == args[1] {
				printChainDetail(c)
				return nil
			}
		}
		return fmt.Errorf("chain not found: %s", args[1])

	case "delete", "rm":
		if len(args) < 2 {
			return fmt.Errorf("usage: pfm chain delete <id>")
		}
		if err := client.DeleteChain(args[1]); err != nil {
			return fmt.Errorf("failed to delete chain: %w", err)
		}
		fmt.Printf("Chain %s deleted\n", args[1])
		return nil

	default:
		return fmt.Errorf("unknown chain command: %s", args[0])
	}
}

func handleStatus() error {
	client := ipc.NewClient()
	if err := client.Connect(); err != nil {
		fmt.Println("Service: not running")
		return nil
	}
	defer client.Close()

	status, err := client.GetStatus()
	if err != nil {
		return fmt.Errorf("failed to get status: %w", err)
	}

	fmt.Println("Service Status")
	fmt.Println("==============")
	fmt.Printf("Running:      %v\n", status.Running)
	fmt.Printf("Version:      %s\n", status.Version)
	fmt.Printf("Active Rules: %d / %d\n", status.RulesActive, status.RulesTotal)

	// Also list rules
	rules, err := client.GetRules()
	if err == nil && len(rules) > 0 {
		fmt.Println("\nRules:")
		printRules(rules)
	}

	return nil
}

func handleVersion() error {
	fmt.Println("Port Forward Manager v1.0.0")
	fmt.Println("Core Engine: gost (go-gost/x)")
	return nil
}

func printRules(rules []*models.Rule) {
	if len(rules) == 0 {
		fmt.Println("No rules configured")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tTYPE\tLOCAL\tTARGET\tSTATUS")
	fmt.Fprintln(w, "--\t----\t----\t-----\t------\t------")

	for _, r := range rules {
		local := fmt.Sprintf(":%d", r.LocalPort)
		target := fmt.Sprintf("%s:%d", r.TargetHost, r.TargetPort)
		status := string(r.Status)
		if status == "" {
			status = "stopped"
		}

		// Shorten ID for display
		id := r.ID
		if len(id) > 8 {
			id = id[:8]
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			id, r.Name, r.Type, local, target, status)
	}
	w.Flush()
}

func printRuleDetail(r *models.Rule) {
	fmt.Printf("ID:          %s\n", r.ID)
	fmt.Printf("Name:        %s\n", r.Name)
	fmt.Printf("Type:        %s\n", r.Type)
	fmt.Printf("Protocol:    %s\n", r.Protocol)
	fmt.Printf("Local Port:  %d\n", r.LocalPort)
	fmt.Printf("Target:      %s:%d\n", r.TargetHost, r.TargetPort)
	fmt.Printf("Status:      %s\n", r.Status)
	fmt.Printf("Enabled:     %v\n", r.Enabled)
	if r.ChainID != "" {
		fmt.Printf("Chain ID:    %s\n", r.ChainID)
	}
	if r.ErrorMsg != "" {
		fmt.Printf("Error:       %s\n", r.ErrorMsg)
	}
}

func printChains(chains []*models.Chain) {
	if len(chains) == 0 {
		fmt.Println("No chains configured")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tHOPS")
	fmt.Fprintln(w, "--\t----\t----")

	for _, c := range chains {
		id := c.ID
		if len(id) > 8 {
			id = id[:8]
		}
		fmt.Fprintf(w, "%s\t%s\t%d\n", id, c.Name, len(c.Hops))
	}
	w.Flush()
}

func printChainDetail(c *models.Chain) {
	fmt.Printf("ID:          %s\n", c.ID)
	fmt.Printf("Name:        %s\n", c.Name)
	fmt.Printf("Hops:        %d\n", len(c.Hops))

	if len(c.Hops) > 0 {
		fmt.Println("\nHop Details:")
		for i, hop := range c.Hops {
			fmt.Printf("  [%d] %s - %s\n", i+1, hop.Protocol, hop.Addr)
			if hop.Auth != nil && hop.Auth.Username != "" {
				fmt.Printf("      Auth: %s\n", hop.Auth.Username)
			}
		}
	}
}

// IsServiceRunCommand checks if the args indicate service run mode
func IsServiceRunCommand(args []string) bool {
	return len(args) >= 2 && args[0] == "service" && args[1] == "run"
}

// IsCLICommand checks if args contain CLI commands (not GUI mode)
func IsCLICommand(args []string) bool {
	if len(args) == 0 {
		return false
	}

	cliCommands := []string{
		"service", "rule", "rules", "chain", "chains",
		"status", "version", "help", "-h", "--help",
	}

	cmd := strings.ToLower(args[0])
	for _, c := range cliCommands {
		if cmd == c {
			return true
		}
	}
	return false
}
