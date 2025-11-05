package main

import (
	"log"
	"os"

	"github.com/konrad-maedler/azubiheft-mcp-server/internal/mcp"
	"github.com/konrad-maedler/azubiheft-mcp-server/internal/server"
)

func main() {
	logger := log.New(os.Stderr, "[azubiheft-mcp] ", log.LstdFlags)

	username := os.Getenv("AZUBIHEFT_USERNAME")
	password := os.Getenv("AZUBIHEFT_PASSWORD")

	if username != "" && password != "" {
		logger.Println("Credentials found in environment variables")
	} else {
		logger.Println("No credentials in environment - manual login required")
	}

	mcpServer := mcp.NewServer("Azubiheft MCP Server", "1.0.0", logger)
	azubiheftService := azubiheftserver.NewAzubiheftService(logger, username, password)
	registerTools(mcpServer, azubiheftService)

	logger.Println("Starting Azubiheft MCP Server...")
	if err := mcpServer.Serve(); err != nil {
		logger.Fatalf("Server error: %v", err)
	}
}

func registerTools(s *mcp.Server, service *azubiheftserver.AzubiheftService) {
	s.RegisterTool(
		"azubiheft_login",
		"Authenticates a user and creates a session",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"username": map[string]interface{}{
					"type":        "string",
					"description": "The user's username",
				},
				"password": map[string]interface{}{
					"type":        "string",
					"description": "The user's password",
				},
			},
			"required": []string{"username", "password"},
		},
		service.Login,
	)

	s.RegisterTool(
		"azubiheft_logout",
		"Terminates the current user session",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"session_id": map[string]interface{}{
					"type":        "string",
					"description": "Session ID from login",
				},
			},
			"required": []string{"session_id"},
		},
		service.Logout,
	)

	s.RegisterTool(
		"azubiheft_is_logged_in",
		"Checks if a user is currently logged in",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"session_id": map[string]interface{}{
					"type":        "string",
					"description": "Session ID to check",
				},
			},
			"required": []string{"session_id"},
		},
		service.IsLoggedIn,
	)

	s.RegisterTool(
		"azubiheft_get_subjects",
		"Retrieves the complete list of subjects (both static and user-defined). If credentials were provided via environment variables, session_id can be omitted.",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"session_id": map[string]interface{}{
					"type":        "string",
					"description": "Session ID from login (optional if using auto-login)",
				},
			},
		},
		service.GetSubjects,
	)

	s.RegisterTool(
		"azubiheft_add_subject",
		"Adds a new custom subject to the user's subject list",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"session_id": map[string]interface{}{
					"type":        "string",
					"description": "Session ID from login",
				},
				"subject_name": map[string]interface{}{
					"type":        "string",
					"description": "Name of the new subject",
				},
			},
			"required": []string{"session_id", "subject_name"},
		},
		service.AddSubject,
	)

	s.RegisterTool(
		"azubiheft_delete_subject",
		"Removes a subject from the user's subject list",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"session_id": map[string]interface{}{
					"type":        "string",
					"description": "Session ID from login",
				},
				"subject_id": map[string]interface{}{
					"type":        "string",
					"description": "ID of the subject to delete",
				},
			},
			"required": []string{"session_id", "subject_id"},
		},
		service.DeleteSubject,
	)

	s.RegisterTool(
		"azubiheft_get_report",
		"Retrieves all report entries for a specific date",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"session_id": map[string]interface{}{
					"type":        "string",
					"description": "Session ID from login",
				},
				"date": map[string]interface{}{
					"type":        "string",
					"description": "Date in YYYY-MM-DD format",
				},
				"include_formatting": map[string]interface{}{
					"type":        "boolean",
					"description": "Whether to include HTML formatting (default: false)",
				},
			},
			"required": []string{"session_id", "date"},
		},
		service.GetReport,
	)

	s.RegisterTool(
		"azubiheft_write_report",
		"Writes a single report entry for a specific date",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"session_id": map[string]interface{}{
					"type":        "string",
					"description": "Session ID from login",
				},
				"date": map[string]interface{}{
					"type":        "string",
					"description": "Date in YYYY-MM-DD format",
				},
				"message": map[string]interface{}{
					"type":        "string",
					"description": "Content of the report",
				},
				"time_spent": map[string]interface{}{
					"type":        "string",
					"description": "Duration in HH:MM format (must not be 00:00)",
				},
				"entry_type": map[string]interface{}{
					"type":        "number",
					"description": "Subject ID (1-7 for static, higher for user-defined)",
				},
			},
			"required": []string{"session_id", "date", "message", "time_spent", "entry_type"},
		},
		service.WriteReport,
	)

	s.RegisterTool(
		"azubiheft_delete_report",
		"Deletes one or all report entries for a specific date",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"session_id": map[string]interface{}{
					"type":        "string",
					"description": "Session ID from login",
				},
				"date": map[string]interface{}{
					"type":        "string",
					"description": "Date in YYYY-MM-DD format",
				},
				"entry_number": map[string]interface{}{
					"type":        "number",
					"description": "Entry number to delete (omit to delete all)",
				},
			},
			"required": []string{"session_id", "date"},
		},
		service.DeleteReport,
	)

	s.RegisterTool(
		"azubiheft_get_week_id",
		"Retrieves the week ID for a given date (required for report operations)",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"session_id": map[string]interface{}{
					"type":        "string",
					"description": "Session ID from login",
				},
				"date": map[string]interface{}{
					"type":        "string",
					"description": "Date in YYYY-MM-DD format",
				},
			},
			"required": []string{"session_id", "date"},
		},
		service.GetWeekID,
	)
}
