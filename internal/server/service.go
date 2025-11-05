package azubiheftserver

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/konrad-maedler/azubiheft-mcp-server/internal/azubiheft"
)

// AzubiheftService manages sessions and provides MCP tool implementations
type AzubiheftService struct {
	sessions         map[string]*azubiheft.Session
	sessionsMutex    sync.RWMutex
	logger           *log.Logger
	defaultSessionID string // Auto-created session from env vars
}

// NewAzubiheftService creates a new service instance
func NewAzubiheftService(logger *log.Logger, username, password string) *AzubiheftService {
	service := &AzubiheftService{
		sessions: make(map[string]*azubiheft.Session),
		logger:   logger,
	}

	if username != "" && password != "" {
		logger.Printf("Auto-login with provided credentials for user: %s", username)
		session := azubiheft.NewSession()
		if err := session.Login(username, password); err != nil {
			logger.Printf("Warning: Auto-login failed: %v", err)
			logger.Println("You can still use manual login via the azubiheft_login tool")
		} else {
			sessionID := "default"
			service.sessionsMutex.Lock()
			service.sessions[sessionID] = session
			service.defaultSessionID = sessionID
			service.sessionsMutex.Unlock()
			logger.Printf("Auto-login successful! Default session ID: %s", sessionID)
			logger.Println("You can use 'default' as session_id or omit it in tool calls")
		}
	}

	return service
}

func (s *AzubiheftService) GetDefaultSessionID() string {
	return s.defaultSessionID
}

func (s *AzubiheftService) getSession(sessionID string) (*azubiheft.Session, error) {
	s.sessionsMutex.RLock()
	defer s.sessionsMutex.RUnlock()

	if sessionID == "" && s.defaultSessionID != "" {
		sessionID = s.defaultSessionID
	}

	session, exists := s.sessions[sessionID]
	if !exists {
		if s.defaultSessionID != "" {
			return nil, fmt.Errorf("invalid session ID (hint: use 'default' or omit session_id to use auto-login session)")
		}
		return nil, fmt.Errorf("invalid session ID")
	}
	return session, nil
}

func (s *AzubiheftService) Login(ctx context.Context, args map[string]interface{}) (string, error) {
	username, ok := args["username"].(string)
	if !ok {
		return "", fmt.Errorf("username is required")
	}

	password, ok := args["password"].(string)
	if !ok {
		return "", fmt.Errorf("password is required")
	}

	session := azubiheft.NewSession()
	if err := session.Login(username, password); err != nil {
		return "", fmt.Errorf("login failed: %w", err)
	}

	sessionID := uuid.New().String()

	s.sessionsMutex.Lock()
	s.sessions[sessionID] = session
	s.sessionsMutex.Unlock()

	s.logger.Printf("User logged in successfully, session ID: %s", sessionID)

	result := fmt.Sprintf("Login successful. Session ID: %s", sessionID)
	return result, nil
}

func (s *AzubiheftService) Logout(ctx context.Context, args map[string]interface{}) (string, error) {
	sessionID, ok := args["session_id"].(string)
	if !ok {
		return "", fmt.Errorf("session_id is required")
	}

	session, err := s.getSession(sessionID)
	if err != nil {
		return "", err
	}

	if err := session.Logout(); err != nil {
		return "", fmt.Errorf("logout failed: %w", err)
	}

	s.sessionsMutex.Lock()
	delete(s.sessions, sessionID)
	s.sessionsMutex.Unlock()

	s.logger.Printf("User logged out, session ID: %s", sessionID)

	result := "Logout successful"
	return result, nil
}

func (s *AzubiheftService) IsLoggedIn(ctx context.Context, args map[string]interface{}) (string, error) {
	sessionID, ok := args["session_id"].(string)
	if !ok {
		return "", fmt.Errorf("session_id is required")
	}

	session, err := s.getSession(sessionID)
	if err != nil {
		return "", err
	}

	loggedIn := session.IsLoggedIn()
	result := fmt.Sprintf("Logged in: %t", loggedIn)
	return result, nil
}

func (s *AzubiheftService) GetSubjects(ctx context.Context, args map[string]interface{}) (string, error) {
	sessionID, _ := args["session_id"].(string)

	session, err := s.getSession(sessionID)
	if err != nil {
		return "", err
	}

	subjects, err := session.GetSubjects()
	if err != nil {
		return "", fmt.Errorf("failed to get subjects: %w", err)
	}

	result := fmt.Sprintf("Subjects: %+v", subjects)
	return result, nil
}

func (s *AzubiheftService) AddSubject(ctx context.Context, args map[string]interface{}) (string, error) {
	sessionID, ok := args["session_id"].(string)
	if !ok {
		return "", fmt.Errorf("session_id is required")
	}

	subjectName, ok := args["subject_name"].(string)
	if !ok {
		return "", fmt.Errorf("subject_name is required")
	}

	session, err := s.getSession(sessionID)
	if err != nil {
		return "", err
	}

	if err := session.AddSubject(subjectName); err != nil {
		return "", fmt.Errorf("failed to add subject: %w", err)
	}

	result := fmt.Sprintf("Subject '%s' added successfully", subjectName)
	return result, nil
}

func (s *AzubiheftService) DeleteSubject(ctx context.Context, args map[string]interface{}) (string, error) {
	sessionID, ok := args["session_id"].(string)
	if !ok {
		return "", fmt.Errorf("session_id is required")
	}

	subjectID, ok := args["subject_id"].(string)
	if !ok {
		return "", fmt.Errorf("subject_id is required")
	}

	session, err := s.getSession(sessionID)
	if err != nil {
		return "", err
	}

	if err := session.DeleteSubject(subjectID); err != nil {
		return "", fmt.Errorf("failed to delete subject: %w", err)
	}

	result := fmt.Sprintf("Subject with ID '%s' deleted successfully", subjectID)
	return result, nil
}

func (s *AzubiheftService) GetReport(ctx context.Context, args map[string]interface{}) (string, error) {
	sessionID, ok := args["session_id"].(string)
	if !ok {
		return "", fmt.Errorf("session_id is required")
	}

	dateStr, ok := args["date"].(string)
	if !ok {
		return "", fmt.Errorf("date is required")
	}

	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return "", fmt.Errorf("invalid date format, use YYYY-MM-DD: %w", err)
	}

	includeFormatting := false
	if val, ok := args["include_formatting"].(bool); ok {
		includeFormatting = val
	}

	session, err := s.getSession(sessionID)
	if err != nil {
		return "", err
	}

	reports, err := session.GetReport(date, includeFormatting)
	if err != nil {
		return "", fmt.Errorf("failed to get report: %w", err)
	}

	result := fmt.Sprintf("Reports for %s: %+v", dateStr, reports)
	return result, nil
}

func (s *AzubiheftService) WriteReport(ctx context.Context, args map[string]interface{}) (string, error) {
	sessionID, ok := args["session_id"].(string)
	if !ok {
		return "", fmt.Errorf("session_id is required")
	}

	dateStr, ok := args["date"].(string)
	if !ok {
		return "", fmt.Errorf("date is required")
	}

	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return "", fmt.Errorf("invalid date format, use YYYY-MM-DD: %w", err)
	}

	message, ok := args["message"].(string)
	if !ok {
		return "", fmt.Errorf("message is required")
	}

	timeSpent, ok := args["time_spent"].(string)
	if !ok {
		return "", fmt.Errorf("time_spent is required")
	}

	entryType, ok := args["entry_type"].(float64)
	if !ok {
		return "", fmt.Errorf("entry_type is required")
	}

	session, err := s.getSession(sessionID)
	if err != nil {
		return "", err
	}

	if err := session.WriteReport(date, message, timeSpent, int(entryType)); err != nil {
		return "", fmt.Errorf("failed to write report: %w", err)
	}

	result := fmt.Sprintf("Report for %s written successfully", dateStr)
	return result, nil
}

func (s *AzubiheftService) DeleteReport(ctx context.Context, args map[string]interface{}) (string, error) {
	sessionID, ok := args["session_id"].(string)
	if !ok {
		return "", fmt.Errorf("session_id is required")
	}

	dateStr, ok := args["date"].(string)
	if !ok {
		return "", fmt.Errorf("date is required")
	}

	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return "", fmt.Errorf("invalid date format, use YYYY-MM-DD: %w", err)
	}

	var entryNumber *int
	if val, ok := args["entry_number"].(float64); ok {
		num := int(val)
		entryNumber = &num
	}

	session, err := s.getSession(sessionID)
	if err != nil {
		return "", err
	}

	if err := session.DeleteReport(date, entryNumber); err != nil {
		return "", fmt.Errorf("failed to delete report: %w", err)
	}

	result := fmt.Sprintf("Report(s) for %s deleted successfully", dateStr)
	return result, nil
}

func (s *AzubiheftService) GetWeekID(ctx context.Context, args map[string]interface{}) (string, error) {
	sessionID, ok := args["session_id"].(string)
	if !ok {
		return "", fmt.Errorf("session_id is required")
	}

	dateStr, ok := args["date"].(string)
	if !ok {
		return "", fmt.Errorf("date is required")
	}

	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return "", fmt.Errorf("invalid date format, use YYYY-MM-DD: %w", err)
	}

	session, err := s.getSession(sessionID)
	if err != nil {
		return "", err
	}

	weekID, err := session.GetReportWeekID(date)
	if err != nil {
		return "", fmt.Errorf("failed to get week ID: %w", err)
	}

	result := fmt.Sprintf("Week ID for %s: %s", dateStr, weekID)
	return result, nil
}
