package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
	"github.com/xhd2015/less-gen/flags"
)

const debugPort = "9222"
const cookieName = "auth-token"

// Credentials file - can be overridden via env var or just return empty
var credentialsFile = os.Getenv("CREDENTIALS_FILE")

func loadAuthToken() string {
	if credentialsFile == "" {
		return ""
	}
	f, err := os.Open(credentialsFile)
	if err != nil {
		return ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			return line
		}
	}
	return ""
}

var defaultPort = 37651

const help = `
Usage: go run ./script/browser-debug [OPTIONS] <URL>

Arguments:
  URL                The URL to navigate to (required)

Options:
  --headless         Run browser in headless mode (no visible window)
  --new              Force start a new browser instance (ignore existing)
  --header <header>  Add custom HTTP header in "Key: Value" format (can be used multiple times)
  --port <port>      Backend port for API requests (default: 37651)

The tool reuses an existing Chrome instance on port ` + debugPort + ` if available.
To start fresh, use --new.

Auto-injects ai-critic auth token from ~/.ai-critic/server-credentials if available.

Examples:

  go run ./script/browser-debug http://localhost:37651
  go run ./script/browser-debug --headless http://localhost:37651
  go run ./script/browser-debug --header "Authorization: Bearer token123" http://localhost:37651
  go run ./script/browser-debug --header "X-Custom: value" --header "Cookie: session=abc" http://localhost:37651
`

var customHeaders map[string]string
var apiPort int

func apiRequest(method, path, body string, headers map[string]string) (string, error) {
	// For API requests, we need a base URL
	// If path is absolute URL, use it directly
	var url string
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		url = path
	} else {
		// Relative path - use the configured api port
		url = fmt.Sprintf("http://localhost:%d%s", apiPort, path)
	}

	var req *http.Request
	var err error
	if body != "" {
		req, err = http.NewRequest(method, url, bytes.NewBufferString(body))
	} else {
		req, err = http.NewRequest(method, url, nil)
	}
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Add custom headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		return "", fmt.Errorf("making request: %w", err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading response: %w", err)
	}
	return fmt.Sprintf("Status: %s\n%s", resp.Status, string(respBody)), nil
}

func main() {
	// Check for subcommands
	if len(os.Args) > 1 && os.Args[1] == "check" {
		err := runCheck()
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
		return
	}

	err := Handle(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func runCheck() error {
	fmt.Println("=== Browser Debug Tool - Dependency Check ===\n")

	hasErrors := false

	// Check 1: Chrome/Chromium
	fmt.Println("1. Checking Chrome/Chromium...")
	chromePath := findChromePath()
	if chromePath != "" {
		fmt.Printf("   ✓ Found: %s\n", chromePath)
	} else {
		fmt.Println("   ✗ Chrome/Chromium not found")
		fmt.Println()
		fmt.Println("   How to install:")
		switch runtime.GOOS {
		case "darwin":
			fmt.Println("   - macOS: brew install --cask google-chrome")
			fmt.Println("   - Or download from: https://www.google.com/chrome/")
		case "linux":
			fmt.Println("   - Ubuntu/Debian:")
			fmt.Println("     sudo apt-get update")
			fmt.Println("     sudo apt-get install chromium")
			fmt.Println("   - If 'chromium' is not found, try:")
			fmt.Println("     sudo snap install chromium")
			fmt.Println("   - Fedora: sudo dnf install chromium")
		default:
			fmt.Println("   - Download from: https://www.google.com/chrome/")
		}
		hasErrors = true
	}
	fmt.Println()

	// Check 2: Go dependencies
	fmt.Println("2. Checking Go dependencies...")
	deps := []struct {
		path string
		name string
	}{
		{"github.com/chromedp/chromedp", "chromedp"},
		{"github.com/chromedp/cdproto", "cdproto"},
	}

	missingDeps := []string{}
	for _, dep := range deps {
		cmd := exec.Command("go", "list", "-m", dep.path)
		err := cmd.Run()
		if err != nil {
			missingDeps = append(missingDeps, dep.name)
			fmt.Printf("   ✗ %s\n", dep.name)
		} else {
			fmt.Printf("   ✓ %s\n", dep.name)
		}
	}

	if len(missingDeps) > 0 {
		fmt.Println()
		fmt.Println("   How to install missing dependencies:")
		fmt.Println("   go get github.com/chromedp/chromedp")
		fmt.Println("   go get github.com/chromedp/cdproto")
		hasErrors = true
	}
	fmt.Println()

	// Summary
	fmt.Println("=== Summary ===")
	if hasErrors {
		fmt.Println("✗ Some dependencies are missing. Please install them following the instructions above.")
		return fmt.Errorf("dependency check failed")
	}
	fmt.Println("✓ All dependencies are installed and ready to use!")
	fmt.Println()
	fmt.Println("Usage: go run ./script/browser-debug <URL>")

	return nil
}

func tryConnectExisting(ctx context.Context) (context.Context, context.CancelFunc, bool) {
	// Try to connect to existing Chrome instance via remote debugging port
	devToolsURL := fmt.Sprintf("http://localhost:%s", debugPort)
	resp, err := http.Get(devToolsURL + "/json/version")
	if err != nil {
		return nil, nil, false
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, false
	}

	// Parse the webSocketDebuggerUrl from the response
	type versionInfo struct {
		WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
	}
	var info versionInfo
	if err := json.Unmarshal(body, &info); err != nil || info.WebSocketDebuggerURL == "" {
		return nil, nil, false
	}

	allocCtx, allocCancel := chromedp.NewRemoteAllocator(ctx, info.WebSocketDebuggerURL)
	chromeCtx, chromeCancel := chromedp.NewContext(allocCtx)

	// Test the connection
	if err := chromedp.Run(chromeCtx, chromedp.Evaluate(`"ok"`, new(string))); err != nil {
		chromeCancel()
		allocCancel()
		return nil, nil, false
	}

	cancel := func() {
		chromeCancel()
		allocCancel()
	}
	return chromeCtx, cancel, true
}

func findChromePath() string {
	if runtime.GOOS == "darwin" {
		paths := []string{
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
			"/Applications/Chromium.app/Contents/MacOS/Chromium",
		}
		for _, p := range paths {
			if _, err := os.Stat(p); err == nil {
				return p
			}
		}
	}
	// Try PATH
	if p, err := exec.LookPath("google-chrome"); err == nil {
		return p
	}
	if p, err := exec.LookPath("chromium"); err == nil {
		return p
	}
	if p, err := exec.LookPath("chromium-browser"); err == nil {
		return p
	}
	return ""
}

func launchChromeDetached(headless bool) error {
	chromePath := findChromePath()
	if chromePath == "" {
		return fmt.Errorf("Chrome/Chromium not found")
	}

	// iPhone 13 Pro dimensions: 390x844
	args := []string{
		"--remote-debugging-port=" + debugPort,
		"--no-first-run",
		"--no-default-browser-check",
		"--window-size=390,844",
		"--user-data-dir=" + os.TempDir() + "/browser-debug-profile",
		"--no-sandbox",
		"--disable-setuid-sandbox",
		"--disable-dev-shm-usage",
		"--disable-gpu",
		"--no-proxy-server",
		"--proxy-bypass-list=*",
	}
	if headless {
		args = append(args, "--headless=new")
	}
	// Open about:blank initially
	args = append(args, "about:blank")

	cmd := exec.Command(chromePath, args...)
	// Detach from parent process so Chrome survives after this tool exits
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("starting Chrome: %w", err)
	}

	// Release the process so it's not killed when Go exits
	if cmd.Process != nil {
		cmd.Process.Release()
	}

	return nil
}

func parseHeader(header string) (string, string, error) {
	parts := strings.SplitN(header, ":", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("header must be in 'Key: Value' format, got: %s", header)
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]), nil
}

func Handle(args []string) error {
	headless := false
	forceNew := false
	url := ""
	var headerList []string

	// Set default API port
	apiPort = defaultPort

	// Collect custom headers
	customHeaders = make(map[string]string)

	remainArgs, err := flags.
		Bool("--headless", &headless).
		Bool("--new", &forceNew).
		String("--url", &url).
		StringSlice("--header", &headerList).
		Int("--port", &apiPort).
		Help("-h,--help", help).
		Parse(args)

	// Parse headers from the slice
	for _, h := range headerList {
		key, val, err := parseHeader(h)
		if err != nil {
			return err
		}
		customHeaders[key] = val
	}

	if err != nil {
		return err
	}

	// Auto-inject auth token if not already set
	if _, ok := customHeaders["Cookie"]; !ok {
		token := loadAuthToken()
		if token != "" {
			customHeaders["Cookie"] = fmt.Sprintf("%s=%s", cookieName, token)
			fmt.Printf("Auto-injected auth cookie: %s\n", cookieName)
		}
	}

	// URL is mandatory - can be via --url flag or positional argument
	if url == "" && len(remainArgs) > 0 {
		url = remainArgs[0]
		remainArgs = remainArgs[1:]
	}

	if url == "" {
		return fmt.Errorf("URL is required. Use --url flag or positional argument.\n\n%s", help)
	}

	if len(remainArgs) > 0 {
		return fmt.Errorf("unrecognized extra args: %s", strings.Join(remainArgs, " "))
	}

	var ctx context.Context
	var cancel context.CancelFunc
	reused := false

	// Try to reuse existing browser instance
	if !forceNew {
		ctx, cancel, reused = tryConnectExisting(context.Background())
	}

	if !reused {
		// Launch Chrome as a detached process so it survives after this tool exits
		if err := launchChromeDetached(headless); err != nil {
			return fmt.Errorf("failed to launch Chrome: %w", err)
		}
		fmt.Printf("Started new Chrome instance (debugging port: %s)\n", debugPort)

		// Wait for Chrome to be ready and connect
		var connected bool
		for i := 0; i < 20; i++ {
			time.Sleep(500 * time.Millisecond)
			ctx, cancel, connected = tryConnectExisting(context.Background())
			if connected {
				break
			}
		}
		if !connected {
			return fmt.Errorf("Chrome started but could not connect to debugging port %s", debugPort)
		}
	} else {
		fmt.Printf("Reusing existing Chrome instance (port: %s)\n", debugPort)
	}
	defer cancel()

	// Set extra HTTP headers if any custom headers were provided
	if len(customHeaders) > 0 {
		headers := make(network.Headers)
		for key, value := range customHeaders {
			headers[key] = value
			fmt.Printf("Setting custom header: %s: %s\n", key, value)
		}
		if err := chromedp.Run(ctx, network.SetExtraHTTPHeaders(headers)); err != nil {
			log.Fatalf("Failed to set headers: %v", err)
		}
	}

	// Navigate to the URL
	fmt.Printf("Navigating to %s ...\n", url)
	if err := chromedp.Run(ctx, chromedp.Navigate(url)); err != nil {
		log.Fatalf("Failed to navigate: %v", err)
	}
	// Wait for page to load
	time.Sleep(5 * time.Second)

	if headless {
		fmt.Println("Running in headless mode (use --no-headless to show browser window)")
	}
	fmt.Println("Browser ready. Type JavaScript expressions to evaluate, or commands:")
	fmt.Println("  eval <js>         - evaluate JavaScript and print result")
	fmt.Println("  styles <selector> - show computed styles for an element")
	fmt.Println("  hierarchy <sel>   - show parent chain with flex/overflow styles")
	fmt.Println("  screenshot        - take a screenshot")
	fmt.Println("  scroll <selector> - scroll element into view")
	fmt.Println("  nav <url>         - navigate to URL")
	fmt.Println("  api GET <path>    - make API request")
	fmt.Println("  api POST <path> <body> - make API POST request")
	fmt.Println("  quit              - exit")
	fmt.Println()

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		if line == "quit" || line == "exit" {
			break
		}

		if strings.HasPrefix(line, "nav ") {
			navURL := strings.TrimSpace(line[4:])
			if err := chromedp.Run(ctx, chromedp.Navigate(navURL)); err != nil {
				fmt.Printf("Error: %v\n", err)
			} else {
				fmt.Println("Navigated.")
			}
			continue
		}

		if line == "screenshot" {
			var buf []byte
			if err := chromedp.Run(ctx, chromedp.FullScreenshot(&buf, 90)); err != nil {
				fmt.Printf("Error: %v\n", err)
				continue
			}
			outPath := fmt.Sprintf("/tmp/browser_debug_%d.png", time.Now().Unix())
			if err := os.WriteFile(outPath, buf, 0644); err != nil {
				fmt.Printf("Error writing: %v\n", err)
				continue
			}
			fmt.Printf("Screenshot saved to %s\n", outPath)
			continue
		}

		if strings.HasPrefix(line, "wait ") {
			durStr := strings.TrimSpace(line[5:])
			dur, err := time.ParseDuration(durStr)
			if err != nil {
				fmt.Printf("Invalid duration: %v\n", err)
				continue
			}
			fmt.Printf("Waiting %s...\n", dur)
			time.Sleep(dur)
			fmt.Println("Done.")
			continue
		}

		if strings.HasPrefix(line, "api ") {
			parts := strings.Fields(line[4:])
			if len(parts) < 2 {
				fmt.Println("Usage: api GET|POST <path> [body]")
				continue
			}
			method := strings.ToUpper(parts[0])
			path := parts[1]
			body := ""
			if len(parts) >= 3 {
				body = strings.Join(parts[2:], " ")
			}
			result, err := apiRequest(method, path, body, customHeaders)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
			} else {
				fmt.Println(result)
			}
			continue
		}

		if strings.HasPrefix(line, "styles ") {
			selector := strings.TrimSpace(line[7:])
			js := fmt.Sprintf(`(() => {
				const el = document.querySelector(%q);
				if (!el) return 'Element not found: ' + %q;
				const cs = window.getComputedStyle(el);
				return JSON.stringify({
					display: cs.display,
					flexDirection: cs.flexDirection,
					flex: cs.flex,
					flexGrow: cs.flexGrow,
					flexShrink: cs.flexShrink,
					minHeight: cs.minHeight,
					maxHeight: cs.maxHeight,
					height: cs.height,
					overflow: cs.overflow,
					overflowY: cs.overflowY,
					position: cs.position,
					top: cs.top,
					width: cs.width,
					scrollHeight: el.scrollHeight,
					clientHeight: el.clientHeight,
					offsetHeight: el.offsetHeight,
				}, null, 2);
			})()`, selector, selector)
			var result string
			if err := chromedp.Run(ctx, chromedp.Evaluate(js, &result)); err != nil {
				fmt.Printf("Error: %v\n", err)
			} else {
				fmt.Println(result)
			}
			continue
		}

		if strings.HasPrefix(line, "hierarchy ") {
			selector := strings.TrimSpace(line[10:])
			js := fmt.Sprintf(`(() => {
				let el = document.querySelector(%q);
				if (!el) return 'Element not found: ' + %q;
				const chain = [];
				while (el) {
					const cs = window.getComputedStyle(el);
					chain.push({
						tag: el.tagName.toLowerCase() + (el.className ? '.' + el.className.split(' ').join('.') : ''),
						display: cs.display,
						flexDirection: cs.flexDirection,
						flex: cs.flex,
						minHeight: cs.minHeight,
						height: cs.height,
						overflow: cs.overflow,
						overflowY: cs.overflowY,
						position: cs.position,
						scrollH: el.scrollHeight,
						clientH: el.clientHeight,
					});
					el = el.parentElement;
				}
				return chain.map((c, i) => {
					const indent = '  '.repeat(chain.length - 1 - i);
					return indent + c.tag + ' | display:' + c.display + ' flex:' + c.flex + ' minH:' + c.minHeight + ' h:' + c.height + ' overflow:' + c.overflow + '/' + c.overflowY + ' pos:' + c.position + ' scrollH:' + c.scrollH + ' clientH:' + c.clientH;
				}).reverse().join('\n');
			})()`, selector, selector)
			var result string
			if err := chromedp.Run(ctx, chromedp.Evaluate(js, &result)); err != nil {
				fmt.Printf("Error: %v\n", err)
			} else {
				fmt.Println(result)
			}
			continue
		}

		if strings.HasPrefix(line, "scroll ") {
			selector := strings.TrimSpace(line[7:])
			if err := chromedp.Run(ctx, chromedp.ScrollIntoView(selector)); err != nil {
				fmt.Printf("Error: %v\n", err)
			} else {
				fmt.Println("Scrolled into view.")
			}
			continue
		}

		// Default: evaluate as JavaScript
		jsExpr := line
		if strings.HasPrefix(line, "eval ") {
			jsExpr = strings.TrimSpace(line[5:])
		}
		var result string
		if err := chromedp.Run(ctx, chromedp.Evaluate(jsExpr, &result)); err != nil {
			fmt.Printf("Error: %v\n", err)
		} else {
			fmt.Println(result)
		}
	}

	fmt.Println("Bye!")
	return nil
}
