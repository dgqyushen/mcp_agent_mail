package main

import (
	"flag"
	"fmt"
	"os"

	"agent-mail/config"
	mcp "agent-mail/mcp"
	"agent-mail/model"
)

func main() {
	cfgPath := flag.String("config", "", "path to config file (default: ~/.agent-mail/config.json)")
	flag.Parse()

	path := *cfgPath
	if path == "" {
		path = model.DefaultConfigPath()
	}

	cfg, err := config.Load(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	s := mcp.New(cfg, path)
	if err := s.ServeStdio(); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
	}
}
