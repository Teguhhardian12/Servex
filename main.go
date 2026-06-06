package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/mark3labs/mcp-go/server"
	"github.com/rxyz/servex/internal/memory"
	mcpserver "github.com/rxyz/servex/internal/server"
)

func main() {
	stdio := flag.Bool("stdio", false, "Run in MCP stdio mode (for Claude Code, Cursor, etc.)")
	httpAddr := flag.String("http", "", "Run HTTP+SSE server on this address (e.g. :8080)")
	dbPath := flag.String("db", "servex.db", "SQLite database path")
	flag.Parse()

	store, err := memory.Open(*dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer store.Close()

	mcpSrv := mcpserver.NewMCPServer(store)

	if *httpAddr != "" {
		runHTTP(mcpSrv, *httpAddr)
		return
	}

	if *stdio {
		runStdio(mcpSrv)
		return
	}

	// Default: stdio mode
	runStdio(mcpSrv)
}

func runStdio(mcpSrv *server.MCPServer) {
	srv := server.NewStdioServer(mcpSrv)
	log.SetOutput(os.Stderr) // MCP uses stdout for protocol
	log.Println("Servex MCP server starting (stdio mode)...")
	if err := srv.Listen(context.Background(), os.Stdin, os.Stdout); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func runHTTP(mcpSrv *server.MCPServer, addr string) {
	srv := server.NewSSEServer(mcpSrv)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		fmt.Fprintln(os.Stderr, "\nShutting down...")
		srv.Shutdown(context.Background())
	}()

	fmt.Fprintf(os.Stderr, "Servex MCP server listening on %s\n", addr)
	if err := srv.Start(addr); err != nil {
		log.Fatalf("HTTP server error: %v", err)
	}
}
