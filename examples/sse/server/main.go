// Tencent is pleased to support the open source community by making trpc-mcp-go available.
//
// Copyright (C) 2025 THL A29 Limited, a Tencent company.  All rights reserved.
//
// trpc-mcp-go is licensed under the Apache License Version 2.0.

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	mcp "trpc.group/trpc-go/trpc-mcp-go"
)

func main() {
	// Create SSE server.
	server := mcp.NewSSEServer(
		"SSE Compatibility Server",          // Server name.
		"1.0.0",                             // Server version.
		mcp.WithSSEEndpoint("/sse"),         // Explicitly set SSE endpoint.
		mcp.WithMessageEndpoint("/message"), // Explicitly set message endpoint.
	)

	// Register tools.
	greetTool := mcp.NewTool("greet",
		mcp.WithDescription("Greet a user by name"),
		mcp.WithString("name", mcp.Description("Name of the person to greet")),
	)
	server.RegisterTool(greetTool, handleGreet)

	weatherTool := mcp.NewTool("weather",
		mcp.WithDescription("Get weather information for a city"),
		mcp.WithString("city", mcp.Description("City name (Beijing, Shanghai, Shenzhen, Guangzhou)")),
	)
	server.RegisterTool(weatherTool, handleWeather)

	log.Printf("Registered tools: greet, weather")
	log.Printf("SSE endpoint: /sse")
	log.Printf("Message endpoint: /message")

	// Set graceful exit.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals.
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-signalChan
		log.Println("Received shutdown signal, gracefully shutting down...")
		cancel()
	}()

	// Start server.
	log.Printf("Starting SSE server on port 4000...")
	go func() {
		if err := server.Start(":4000"); err != nil {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Wait for exit signal.
	<-ctx.Done()
	log.Println("Shutting down server...")

	// Graceful exit.
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server shutdown failed: %v", err)
	}

	log.Println("Server gracefully stopped")
}

// handleGreet handles greet tool callback function.
func handleGreet(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Get session information.
	session, ok := mcp.GetSessionFromContext(ctx)
	if !ok || session == nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.NewTextContent("Warning: Session info not found, but you may continue."),
			},
		}, nil
	}

	// Extract name parameter.
	name := "Client user"
	if nameArg, ok := req.Params.Arguments["name"]; ok {
		if nameStr, ok := nameArg.(string); ok && nameStr != "" {
			name = nameStr
		}
	}

	// Try to get server instance and send notification.
	if server, ok := mcp.GetServerFromContext(ctx).(interface {
		SendNotification(string, string, map[string]interface{}) error
	}); ok {
		// Send notification to current session.
		err := server.SendNotification(
			session.GetID(),
			"greeting.echo",
			map[string]interface{}{
				"message": fmt.Sprintf("Server received greeting for: %s", name),
				"time":    time.Now().Format(time.RFC3339),
			},
		)
		if err != nil {
			log.Printf("Failed to send notification: %v", err)
		} else {
			log.Printf("Notification sent successfully to session: %s", session.GetID())
		}
	} else {
		log.Printf("Server instance not available in context or does not support SendNotification")
	}

	// Return greeting message.
	log.Printf(
		"Hello, %s! (Session ID: %s)",
		name, session.GetID()[:8]+"...",
	)
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.NewTextContent(fmt.Sprintf(
				"Hello, %s! (Session ID: %s)",
				name, session.GetID()[:8]+"...",
			)),
		},
	}, nil
}

// handleWeather handles weather tool callback function.
func handleWeather(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Extract city parameter.
	city := "Beijing"
	if cityArg, ok := req.Params.Arguments["city"]; ok {
		if cityStr, ok := cityArg.(string); ok && cityStr != "" {
			city = cityStr
		}
	}

	// Simulate weather information.
	weatherInfo := map[string]string{
		"Beijing":   "Sunny, 25°C",
		"Shanghai":  "Cloudy, 22°C",
		"Shenzhen":  "Rainy, 28°C",
		"Guangzhou": "Partly cloudy, 30°C",
	}

	weather, ok := weatherInfo[city]
	if !ok {
		weather = "Unknown, please check a supported city"
	}

	log.Printf("Weather request for city: %s, result: %s", city, weather)

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.NewTextContent(fmt.Sprintf("Weather in %s: %s", city, weather)),
		},
	}, nil
}
