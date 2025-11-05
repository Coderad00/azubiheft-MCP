package azubiheft

import (
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

const (
	baseURL = "https://www.azubiheft.de"
)

// Session represents an authenticated session
type Session struct {
	client *http.Client
}

// Subject represents a subject/activity type
type Subject struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ReportEntry represents a single report entry
type ReportEntry struct {
	Seq      string `json:"seq"`
	Type     string `json:"type"`
	Duration string `json:"duration"`
	Text     string `json:"text"`
}

// NewSession creates a new session
func NewSession() *Session {
	jar, _ := cookiejar.New(nil)
	return &Session{
		client: &http.Client{
			Jar: jar,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return nil
			},
		},
	}
}

// Login authenticates the user
func (s *Session) Login(username, password string) error {
	// Get login page for tokens
	resp, err := s.client.Get(baseURL + "/Login.aspx")
	if err != nil {
		return fmt.Errorf("failed to get login page: %w", err)
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to parse login page: %w", err)
	}

	// Extract tokens
	viewState, _ := doc.Find("#__VIEWSTATE").Attr("value")
	viewStateGenerator, _ := doc.Find("#__VIEWSTATEGENERATOR").Attr("value")
	eventValidation, _ := doc.Find("#__EVENTVALIDATION").Attr("value")

	// Prepare form data
	formData := url.Values{
		"__VIEWSTATE":          {viewState},
		"__VIEWSTATEGENERATOR": {viewStateGenerator},
		"__EVENTVALIDATION":    {eventValidation},
		"ctl00$ContentPlaceHolder1$txt_Benutzername":     {username},
		"ctl00$ContentPlaceHolder1$txt_Passwort":         {password},
		"ctl00$ContentPlaceHolder1$chk_Persistent":       {"on"},
		"ctl00$ContentPlaceHolder1$cmd_Login":            {"Anmelden"},
		"ctl00$ContentPlaceHolder1$HiddenField_isMobile": {"false"},
	}

	// Submit login
	resp, err = s.client.PostForm(baseURL+"/Login.aspx", formData)
	if err != nil {
		return fmt.Errorf("failed to submit login: %w", err)
	}
	defer resp.Body.Close()

	// Check if login was successful
	if !s.IsLoggedIn() {
		return fmt.Errorf("login failed: invalid credentials")
	}

	return nil
}

// Logout terminates the session
func (s *Session) Logout() error {
	resp, err := s.client.Get(baseURL + "/Azubi/Abmelden.aspx")
	if err != nil {
		return fmt.Errorf("failed to logout: %w", err)
	}
	defer resp.Body.Close()

	return nil
}

// IsLoggedIn checks if the session is authenticated
func (s *Session) IsLoggedIn() bool {
	resp, err := s.client.Get(baseURL + "/Azubi/Default.aspx")
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false
	}

	return strings.Contains(string(body), `id="Abmelden"`)
}

// GetSubjects retrieves all subjects
func (s *Session) GetSubjects() ([]Subject, error) {
	resp, err := s.client.Get(baseURL + "/Azubi/SetupSchulfach.aspx")
	if err != nil {
		return nil, fmt.Errorf("failed to get subjects page: %w", err)
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse subjects page: %w", err)
	}

	subjects := []Subject{
		{ID: "1", Name: "Betrieb"},
		{ID: "2", Name: "Schule"},
		{ID: "3", Name: "ÜBA"},
		{ID: "4", Name: "Urlaub"},
		{ID: "5", Name: "Feiertag"},
		{ID: "6", Name: "Arbeitsunfähig"},
		{ID: "7", Name: "Frei"},
	}

	divSchulfach := doc.Find("#divSchulfach")
	if divSchulfach.Length() > 0 {
		divSchulfach.Find("input").Each(func(i int, sel *goquery.Selection) {
			id, hasID := sel.Attr("data-default")
			name, hasValue := sel.Attr("value")

			if hasID && hasValue && name != "" {
				subjects = append(subjects, Subject{
					ID:   id,
					Name: name,
				})
			}
		})
	}

	return subjects, nil
}

// AddSubject adds a new subject
func (s *Session) AddSubject(subjectName string) error {
	// Get current subjects and tokens
	resp, err := s.client.Get(baseURL + "/Azubi/SetupSchulfach.aspx")
	if err != nil {
		return fmt.Errorf("failed to get subjects page: %w", err)
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to parse subjects page: %w", err)
	}

	// Extract tokens
	viewState, _ := doc.Find("#__VIEWSTATE").Attr("value")
	viewStateGenerator, _ := doc.Find("#__VIEWSTATEGENERATOR").Attr("value")
	eventValidation, _ := doc.Find("#__EVENTVALIDATION").Attr("value")

	// Prepare form data
	formData := url.Values{
		"__VIEWSTATE":                        {viewState},
		"__VIEWSTATEGENERATOR":               {viewStateGenerator},
		"__EVENTVALIDATION":                  {eventValidation},
		"ctl00$ContentPlaceHolder1$cmd_Save": {"Speichern"},
	}

	// Add existing subjects
	doc.Find("input[id^='ctl00_ContentPlaceHolder1_txt']").Each(func(i int, sel *goquery.Selection) {
		id, _ := sel.Attr("id")
		value, _ := sel.Attr("value")
		if value != "" {
			formData.Set(id, value)
		}
	})

	timestamp := time.Now().Unix()
	formData.Set(fmt.Sprintf("txt%d", timestamp), subjectName)

	resp, err = s.client.PostForm(baseURL+"/Azubi/SetupSchulfach.aspx", formData)
	if err != nil {
		return fmt.Errorf("failed to add subject: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to add subject: status code %d", resp.StatusCode)
	}

	return nil
}

// DeleteSubject deletes a subject
func (s *Session) DeleteSubject(subjectID string) error {
	// Get current subjects and tokens
	resp, err := s.client.Get(baseURL + "/Azubi/SetupSchulfach.aspx")
	if err != nil {
		return fmt.Errorf("failed to get subjects page: %w", err)
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to parse subjects page: %w", err)
	}

	// Extract tokens
	viewState, _ := doc.Find("#__VIEWSTATE").Attr("value")
	viewStateGenerator, _ := doc.Find("#__VIEWSTATEGENERATOR").Attr("value")
	eventValidation, _ := doc.Find("#__EVENTVALIDATION").Attr("value")

	// Prepare form data
	formData := url.Values{
		"__VIEWSTATE":                              {viewState},
		"__VIEWSTATEGENERATOR":                     {viewStateGenerator},
		"__EVENTVALIDATION":                        {eventValidation},
		"ctl00$ContentPlaceHolder1$HiddenLöschIDs": {"," + subjectID},
		"ctl00$ContentPlaceHolder1$cmd_Save":       {"Speichern"},
	}

	doc.Find("input[id^='ctl00_ContentPlaceHolder1_txt']").Each(func(i int, sel *goquery.Selection) {
		id, _ := sel.Attr("id")
		value, _ := sel.Attr("value")

		re := regexp.MustCompile(`txt(\d+)`)
		matches := re.FindStringSubmatch(id)
		if len(matches) >= 2 && matches[1] != subjectID && value != "" {
			formData.Set(id, value)
		}
	})

	resp, err = s.client.PostForm(baseURL+"/Azubi/SetupSchulfach.aspx", formData)
	if err != nil {
		return fmt.Errorf("failed to delete subject: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to delete subject: status code %d", resp.StatusCode)
	}

	return nil
}

func (s *Session) GetReportWeekID(date time.Time) (string, error) {
	resp, err := s.client.Get(baseURL + "/Azubi/Ausbildungsnachweise.aspx")
	if err != nil {
		return "", fmt.Errorf("failed to get reports page: %w", err)
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to parse reports page: %w", err)
	}

	year, week := date.ISOWeek()
	var weekID string

	doc.Find("div.mo.NBox").Each(func(i int, sel *goquery.Selection) {
		onclick, exists := sel.Attr("onclick")
		if !exists {
			return
		}

		kwDiv := sel.Find("div.sKW")
		if kwDiv.Length() == 0 {
			return
		}

		kwParent := sel.Find("div.KW")
		if kwParent.Length() == 0 {
			return
		}

		yearDivs := kwParent.Find("div")
		if yearDivs.Length() < 3 {
			return
		}

		kwText := strings.TrimSpace(kwDiv.Text())
		kw, err := strconv.Atoi(kwText)
		if err != nil {
			return
		}

		yearText := strings.TrimSpace(yearDivs.Eq(2).Text())
		kwYear, err := strconv.Atoi(yearText)
		if err != nil {
			return
		}

		if kw == week && kwYear == year {
			re := regexp.MustCompile(`NachweisNr=(\d+)`)
			matches := re.FindStringSubmatch(onclick)
			if len(matches) >= 2 {
				weekID = matches[1]
			}
		}
	})

	if weekID == "" {
		return "", fmt.Errorf("no report found for week %d/%d", week, year)
	}

	return weekID, nil
}

func (s *Session) GetReport(date time.Time, includeFormatting bool) ([]ReportEntry, error) {
	dateStr := date.Format("20060102")
	resp, err := s.client.Get(baseURL + "/Azubi/Tagesbericht.aspx?Datum=" + dateStr)
	if err != nil {
		return nil, fmt.Errorf("failed to get report page: %w", err)
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse report page: %w", err)
	}

	var entries []ReportEntry

	doc.Find("div.d0.mo").Each(func(i int, entry *goquery.Selection) {
		seq, _ := entry.Attr("data-seq")
		duration := strings.TrimSpace(entry.Find("div.row2.d4").Text())

		if duration == "00:00" {
			return
		}

		activityType := entry.Find("div.row1.d3").Text()
		activityType = strings.TrimSpace(activityType)
		activityType = strings.TrimPrefix(activityType, "Art: ")

		reportTextDiv := entry.Find("div.row7.d5")
		var text string

		if includeFormatting {
			htmlContent, _ := reportTextDiv.Html()
			text = strings.ReplaceAll(htmlContent, "<br/>", "\n")
			text = strings.ReplaceAll(text, "<br>", "\n")
		} else {
			text = strings.TrimSpace(reportTextDiv.Text())
		}

		entries = append(entries, ReportEntry{
			Seq:      seq,
			Type:     activityType,
			Duration: duration,
			Text:     text,
		})
	})

	return entries, nil
}

func (s *Session) WriteReport(date time.Time, message, timeSpent string, entryType int) error {
	if timeSpent == "00:00" {
		return nil
	}

	weekID, err := s.GetReportWeekID(date)
	if err != nil {
		return err
	}

	dateStr := date.Format("20060102")
	timestamp := time.Now().Unix()

	lines := strings.Split(message, "\n")
	var formattedLines []string
	for _, line := range lines {
		formattedLines = append(formattedLines, "<div>"+line+"</div>")
	}
	formattedMessage := strings.Join(formattedLines, "")

	encodedMessage := url.QueryEscape(formattedMessage)
	encodedMessage = strings.ReplaceAll(encodedMessage, "+", "%20")

	formData := url.Values{
		"disablePaste": {"0"},
		"Seq":          {"0"},
		"Art_ID":       {strconv.Itoa(entryType)},
		"Abt_ID":       {"0"},
		"Dauer":        {timeSpent},
		"Inhalt":       {encodedMessage},
		"jsVer":        {"12"},
	}

	reqURL := fmt.Sprintf("%s/Azubi/XMLHttpRequest.ashx?Datum=%s&BrNr=%s&BrSt=1&BrVorh=Yes&T=%d",
		baseURL, dateStr, weekID, timestamp)

	req, err := http.NewRequest("POST", reqURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("x-my-ajax-request", "ajax")
	req.Header.Set("Origin", baseURL)
	req.Header.Set("Referer", baseURL)
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Pragma", "no-cache")
	req.Header.Set("Cache-Control", "no-cache")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to write report: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to write report: status code %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}

func (s *Session) DeleteReport(date time.Time, entryNumber *int) error {
	reports, err := s.GetReport(date, false)
	if err != nil {
		return err
	}

	if len(reports) == 0 {
		return nil
	}

	weekID, err := s.GetReportWeekID(date)
	if err != nil {
		return err
	}

	dateStr := date.Format("20060102")
	timestamp := time.Now().Unix()

	var entriesToDelete []ReportEntry
	if entryNumber == nil {
		entriesToDelete = reports
	} else {
		if *entryNumber < 1 || *entryNumber > len(reports) {
			return fmt.Errorf("invalid entry number: %d", *entryNumber)
		}
		entriesToDelete = []ReportEntry{reports[*entryNumber-1]}
	}

	for _, entry := range entriesToDelete {
		formData := url.Values{
			"disablePaste": {"0"},
			"Seq":          {fmt.Sprintf("-%s", entry.Seq)},
			"Art_ID":       {"0"},
			"Abt_ID":       {"0"},
			"Dauer":        {entry.Duration},
			"Inhalt":       {entry.Text},
			"jsVer":        {"12"},
		}

		reqURL := fmt.Sprintf("%s/Azubi/XMLHttpRequest.ashx?Datum=%s&BrNr=%s&BrSt=1&BrVorh=Yes&T=%d",
			baseURL, dateStr, weekID, timestamp)

		req, err := http.NewRequest("POST", reqURL, strings.NewReader(formData.Encode()))
		if err != nil {
			return fmt.Errorf("failed to create delete request: %w", err)
		}

		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("x-my-ajax-request", "ajax")
		req.Header.Set("Origin", baseURL)
		req.Header.Set("Referer", baseURL)

		resp, err := s.client.Do(req)
		if err != nil {
			return fmt.Errorf("failed to delete report: %w", err)
		}
		resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("failed to delete report: status code %d", resp.StatusCode)
		}
	}

	return nil
}
