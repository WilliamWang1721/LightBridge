package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type cliError struct {
	Message string `json:"message"`
}

type cliResult struct {
	OK     bool            `json:"ok"`
	Action string          `json:"action"`
	Data   json.RawMessage `json:"data,omitempty"`
	Error  *cliError       `json:"error,omitempty"`
}

type client struct {
	baseURL string
	apiKey  string
	http    *http.Client
}

func main() {
	if len(os.Args) < 2 {
		writeResultAndExit(cliResult{OK: false, Action: "help", Error: &cliError{Message: "command is required"}}, 2)
	}
	cmd := os.Args[1]
	switch cmd {
	case "list":
		runList(os.Args[2:])
	case "apply":
		runApply(os.Args[2:])
	case "configure":
		runConfigure(os.Args[2:])
	case "activate":
		runActivate(os.Args[2:])
	case "deactivate":
		runDeactivate(os.Args[2:])
	case "delete":
		runDelete(os.Args[2:])
	case "validate-package":
		runValidatePackage(os.Args[2:])
	default:
		writeResultAndExit(cliResult{OK: false, Action: cmd, Error: &cliError{Message: "unknown command"}}, 2)
	}
}

func baseFlags(name string, args []string) (*flag.FlagSet, *client, *bool) {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	baseURL := fs.String("base-url", os.Getenv("LIGHTBRIDGE_BASE_URL"), "LightBridge base URL")
	adminAPIKey := fs.String("admin-api-key", os.Getenv("LIGHTBRIDGE_ADMIN_API_KEY"), "LightBridge Admin API Key")
	jsonOut := fs.Bool("json", false, "write JSON output")
	_ = fs.Parse(args)
	c := &client{
		baseURL: strings.TrimRight(strings.TrimSpace(*baseURL), "/"),
		apiKey:  strings.TrimSpace(*adminAPIKey),
		http:    &http.Client{Timeout: 60 * time.Second},
	}
	return fs, c, jsonOut
}

func runList(args []string) {
	fs, c, _ := baseFlags("list", args)
	if err := c.requireAuth(); err != nil {
		fail("list", err, 2)
	}
	data, err := c.do(http.MethodGet, "/api/v1/admin/ui-themes", nil, "")
	if err != nil {
		fail("list", err, 1)
	}
	_ = fs
	ok("list", data)
}

func runApply(args []string) {
	fs, c, _ := baseFlags("apply", args)
	githubURL := fs.String("github", "", "GitHub repository URL")
	zipPath := fs.String("zip", "", "theme zip path")
	configPath := fs.String("config", "", "config JSON file")
	activate := fs.Bool("activate", false, "activate after import")
	replace := fs.Bool("replace", false, "replace existing theme")
	if err := fs.Parse(args); err != nil {
		fail("apply", err, 2)
	}
	if err := c.requireAuth(); err != nil {
		fail("apply", err, 2)
	}
	var data []byte
	var err error
	switch {
	case strings.TrimSpace(*githubURL) != "":
		body, _ := json.Marshal(map[string]any{"url": *githubURL, "replace": *replace})
		data, err = c.do(http.MethodPost, "/api/v1/admin/ui-themes/import-github", bytes.NewReader(body), "application/json")
	case strings.TrimSpace(*zipPath) != "":
		data, err = c.uploadZip(*zipPath, *replace)
	default:
		fail("apply", fmt.Errorf("--github or --zip is required"), 2)
	}
	if err != nil {
		fail("apply", err, 1)
	}
	themeID := extractThemeID(data)
	if *configPath != "" {
		if themeID == "" {
			fail("apply", fmt.Errorf("cannot configure imported theme: missing theme id in response"), 1)
		}
		configData, err := os.ReadFile(*configPath)
		if err != nil {
			fail("apply", err, 1)
		}
		body, _ := json.Marshal(map[string]json.RawMessage{"config": configData})
		data, err = c.do(http.MethodPut, "/api/v1/admin/ui-themes/"+url.PathEscape(themeID)+"/config", bytes.NewReader(body), "application/json")
		if err != nil {
			fail("apply", err, 1)
		}
	}
	if *activate {
		if themeID == "" {
			themeID = extractThemeID(data)
		}
		if themeID == "" {
			fail("apply", fmt.Errorf("cannot activate imported theme: missing theme id in response"), 1)
		}
		data, err = c.do(http.MethodPut, "/api/v1/admin/ui-themes/"+url.PathEscape(themeID)+"/activate", nil, "")
		if err != nil {
			fail("apply", err, 1)
		}
	}
	ok("apply", data)
}

func runConfigure(args []string) {
	fs, c, _ := baseFlags("configure", args)
	theme := fs.String("theme", "", "theme id")
	configPath := fs.String("config", "", "config JSON file")
	if err := fs.Parse(args); err != nil {
		fail("configure", err, 2)
	}
	if err := c.requireAuth(); err != nil {
		fail("configure", err, 2)
	}
	if *theme == "" || *configPath == "" {
		fail("configure", fmt.Errorf("--theme and --config are required"), 2)
	}
	configData, err := os.ReadFile(*configPath)
	if err != nil {
		fail("configure", err, 1)
	}
	body, _ := json.Marshal(map[string]json.RawMessage{"config": configData})
	data, err := c.do(http.MethodPut, "/api/v1/admin/ui-themes/"+url.PathEscape(*theme)+"/config", bytes.NewReader(body), "application/json")
	if err != nil {
		fail("configure", err, 1)
	}
	ok("configure", data)
}

func runActivate(args []string) {
	fs, c, _ := baseFlags("activate", args)
	theme := fs.String("theme", "", "theme id")
	if err := fs.Parse(args); err != nil {
		fail("activate", err, 2)
	}
	if err := c.requireAuth(); err != nil {
		fail("activate", err, 2)
	}
	if *theme == "" {
		fail("activate", fmt.Errorf("--theme is required"), 2)
	}
	data, err := c.do(http.MethodPut, "/api/v1/admin/ui-themes/"+url.PathEscape(*theme)+"/activate", nil, "")
	if err != nil {
		fail("activate", err, 1)
	}
	ok("activate", data)
}

func runDeactivate(args []string) {
	fs, c, _ := baseFlags("deactivate", args)
	theme := fs.String("theme", "", "theme id")
	if err := fs.Parse(args); err != nil {
		fail("deactivate", err, 2)
	}
	if err := c.requireAuth(); err != nil {
		fail("deactivate", err, 2)
	}
	if *theme == "" {
		fail("deactivate", fmt.Errorf("--theme is required"), 2)
	}
	data, err := c.do(http.MethodPut, "/api/v1/admin/ui-themes/"+url.PathEscape(*theme)+"/deactivate", nil, "")
	if err != nil {
		fail("deactivate", err, 1)
	}
	ok("deactivate", data)
}

func runDelete(args []string) {
	fs, c, _ := baseFlags("delete", args)
	theme := fs.String("theme", "", "theme id")
	if err := fs.Parse(args); err != nil {
		fail("delete", err, 2)
	}
	if err := c.requireAuth(); err != nil {
		fail("delete", err, 2)
	}
	if *theme == "" {
		fail("delete", fmt.Errorf("--theme is required"), 2)
	}
	data, err := c.do(http.MethodDelete, "/api/v1/admin/ui-themes/"+url.PathEscape(*theme), nil, "")
	if err != nil {
		fail("delete", err, 1)
	}
	ok("delete", data)
}

func runValidatePackage(args []string) {
	fs := flag.NewFlagSet("validate-package", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	zipPath := fs.String("zip", "", "theme zip path")
	if err := fs.Parse(args); err != nil {
		fail("validate-package", err, 2)
	}
	if *zipPath == "" {
		fail("validate-package", fmt.Errorf("--zip is required"), 2)
	}
	data, err := os.ReadFile(*zipPath)
	if err != nil {
		fail("validate-package", err, 1)
	}
	if len(data) == 0 || len(data) > 10<<20 {
		fail("validate-package", fmt.Errorf("zip size must be between 1 byte and 10MB"), 1)
	}
	ok("validate-package", json.RawMessage(`{"valid":true}`))
}

func (c *client) requireAuth() error {
	if c.baseURL == "" {
		return fmt.Errorf("--base-url or LIGHTBRIDGE_BASE_URL is required")
	}
	if c.apiKey == "" {
		return fmt.Errorf("--admin-api-key or LIGHTBRIDGE_ADMIN_API_KEY is required")
	}
	return nil
}

func (c *client) do(method, apiPath string, body io.Reader, contentType string) ([]byte, error) {
	req, err := http.NewRequest(method, c.baseURL+apiPath, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-api-key", c.apiKey)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}
	return data, nil
}

func (c *client) uploadZip(zipPath string, replace bool) ([]byte, error) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile("file", filepath.Base(zipPath))
	if err != nil {
		return nil, err
	}
	file, err := os.Open(zipPath)
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(part, io.LimitReader(file, 10<<20+1)); err != nil {
		_ = file.Close()
		return nil, err
	}
	_ = file.Close()
	if err := writer.Close(); err != nil {
		return nil, err
	}
	apiPath := "/api/v1/admin/ui-themes/upload"
	if replace {
		apiPath += "?replace=true"
	}
	return c.do(http.MethodPost, apiPath, &buf, writer.FormDataContentType())
}

func extractThemeID(data []byte) string {
	var envelope struct {
		Data struct {
			Theme struct {
				ID string `json:"id"`
			} `json:"theme"`
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return ""
	}
	if envelope.Data.Theme.ID != "" {
		return envelope.Data.Theme.ID
	}
	return envelope.Data.ID
}

func ok(action string, data []byte) {
	writeResultAndExit(cliResult{OK: true, Action: action, Data: data}, 0)
}

func fail(action string, err error, code int) {
	writeResultAndExit(cliResult{OK: false, Action: action, Error: &cliError{Message: err.Error()}}, code)
}

func writeResultAndExit(result cliResult, code int) {
	out, _ := json.MarshalIndent(result, "", "  ")
	_, _ = fmt.Fprintln(os.Stdout, string(out))
	os.Exit(code)
}
