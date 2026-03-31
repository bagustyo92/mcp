package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"bitbucket.org/mid-kelola-indonesia/pull-request-maker-mcp/internal/bitbucket"
	"bitbucket.org/mid-kelola-indonesia/pull-request-maker-mcp/internal/config"
	"bitbucket.org/mid-kelola-indonesia/pull-request-maker-mcp/internal/gchat"
	"bitbucket.org/mid-kelola-indonesia/pull-request-maker-mcp/internal/prdesc"
	"bitbucket.org/mid-kelola-indonesia/pull-request-maker-mcp/internal/tools"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	configPath := flag.String("config", "", "Path to config.yaml (default: config.yaml next to binary)")
	flag.Parse()

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Create dependencies
	bbClient := bitbucket.NewClient()
	templateLoader := prdesc.NewTemplateLoader()
	gchatNotifier := gchat.NewNotifier()

	// Create MCP server
	server := mcp.NewServer(
		&mcp.Implementation{
			Name:    "pull-request-maker-mcp",
			Version: "1.0.0",
		},
		nil,
	)

	// Register tools
	tools.RegisterGetPRDiff(server, cfg, bbClient)
	tools.RegisterGetBranchDiff(server, cfg, bbClient, templateLoader)
	tools.RegisterCreatePR(server, cfg, bbClient, templateLoader)
	tools.RegisterPostPRComments(server, cfg, bbClient)
	tools.RegisterNotifyGChat(server, cfg, bbClient, gchatNotifier)

	// Graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigCh
		cancel()
	}()

	// Run with stdio transport
	if err := server.Run(ctx, &mcp.StdioTransport{}); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
