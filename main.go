// NetTraceX - A comprehensive network diagnostic toolkit with beautiful TUI
package main

import (
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nettracex/nettracex-tui/internal/config"
	"github.com/nettracex/nettracex-tui/internal/domain"
	"github.com/nettracex/nettracex-tui/internal/network"
	"github.com/nettracex/nettracex-tui/internal/tools/dns"
	"github.com/nettracex/nettracex-tui/internal/tools/ping"
	"github.com/nettracex/nettracex-tui/internal/tools/ssl"
	"github.com/nettracex/nettracex-tui/internal/tools/traceroute"
	"github.com/nettracex/nettracex-tui/internal/tools/whois"
	"github.com/nettracex/nettracex-tui/internal/tui"
)

// SimplePluginRegistry implements a basic plugin registry
type SimplePluginRegistry struct {
	tools map[string]domain.DiagnosticTool
}

func NewSimplePluginRegistry() *SimplePluginRegistry {
	return &SimplePluginRegistry{
		tools: make(map[string]domain.DiagnosticTool),
	}
}

func (r *SimplePluginRegistry) Register(tool domain.DiagnosticTool) error {
	r.tools[tool.Name()] = tool
	return nil
}

func (r *SimplePluginRegistry) Get(name string) (domain.DiagnosticTool, bool) {
	tool, exists := r.tools[name]
	return tool, exists
}

func (r *SimplePluginRegistry) List() []domain.DiagnosticTool {
	var tools []domain.DiagnosticTool
	for _, tool := range r.tools {
		tools = append(tools, tool)
	}
	return tools
}

func (r *SimplePluginRegistry) Unregister(name string) error {
	delete(r.tools, name)
	return nil
}

// SimpleTheme implements a basic theme
type SimpleTheme struct{}

func (t *SimpleTheme) GetColor(element string) string {
	colors := map[string]string{
		"primary":   "#62a0ea",
		"secondary": "#f6d32d",
		"success":   "#26a269",
		"warning":   "#f57c00",
		"error":     "#e01b24",
		"text":      "#ffffff",
		"background": "#1e1e1e",
	}
	if color, exists := colors[element]; exists {
		return color
	}
	return "#ffffff"
}

func (t *SimpleTheme) GetStyle(element string) map[string]interface{} {
	return make(map[string]interface{})
}

func (t *SimpleTheme) SetColor(element, color string) {
	// Not implemented for simple theme
}

// SimpleLogger implements a basic logger
type SimpleLogger struct{}

func (l *SimpleLogger) Debug(msg string, fields ...interface{}) {
	log.Printf("[DEBUG] %s %v", msg, fields)
}

func (l *SimpleLogger) Info(msg string, fields ...interface{}) {
	log.Printf("[INFO] %s %v", msg, fields)
}

func (l *SimpleLogger) Warn(msg string, fields ...interface{}) {
	log.Printf("[WARN] %s %v", msg, fields)
}

func (l *SimpleLogger) Error(msg string, fields ...interface{}) {
	log.Printf("[ERROR] %s %v", msg, fields)
}

func (l *SimpleLogger) Fatal(msg string, fields ...interface{}) {
	log.Fatalf("[FATAL] %s %v", msg, fields)
}

func main() {
	// Initialize configuration manager
	configManager := config.NewManager()
	if err := configManager.Load(); err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}
	
	cfg := configManager.GetConfig()
	
	// Initialize logger
	logger := &SimpleLogger{}
	
	// Initialize network client (using nil for error handler for now)
	networkClient := network.NewClient(&cfg.Network, nil, logger)
	
	// Initialize plugin registry
	registry := NewSimplePluginRegistry()
	
	// Register WHOIS tool
	whoisTool := whois.NewTool(networkClient, logger)
	if err := registry.Register(whoisTool); err != nil {
		log.Fatalf("Failed to register WHOIS tool: %v", err)
	}
	
	// Register Ping tool
	pingTool := ping.NewTool(networkClient, logger)
	if err := registry.Register(pingTool); err != nil {
		log.Fatalf("Failed to register Ping tool: %v", err)
	}
	
	// Register DNS tool
	dnsTool := dns.NewTool(networkClient, logger)
	if err := registry.Register(dnsTool); err != nil {
		log.Fatalf("Failed to register DNS tool: %v", err)
	}
	
	// Register Traceroute tool
	tracerouteTool := traceroute.NewTool(networkClient, logger)
	if err := registry.Register(tracerouteTool); err != nil {
		log.Fatalf("Failed to register Traceroute tool: %v", err)
	}
	
	// Register SSL tool
	sslTool := ssl.NewTool(networkClient, logger)
	if err := registry.Register(sslTool); err != nil {
		log.Fatalf("Failed to register SSL tool: %v", err)
	}
	
	// Initialize theme
	theme := &SimpleTheme{}
	
	// Create main TUI model
	mainModel := tui.NewMainModel(registry, cfg, configManager, theme)
	
	// Create Bubble Tea program
	program := tea.NewProgram(
		mainModel,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)
	
	// Start the TUI
	if _, err := program.Run(); err != nil {
		log.Printf("Error running TUI: %v", err)
		os.Exit(1)
	}
}