# Browser Debug Tool

Interactive browser debugging tool using Chrome DevTools Protocol (via `chromedp`). Launches a Chrome instance, navigates to a URL, and provides an interactive REPL for inspecting the DOM, computed styles, taking screenshots, and making API requests.

## Usage

```sh
# Check dependencies are installed
go run ./script/browser-debug check

# Basic usage - URL is required
go run ./script/browser-debug http://localhost:3000

# Headless mode (no window, works over SSH)
go run ./script/browser-debug --headless http://localhost:3000

# With custom authentication headers
go run ./script/browser-debug --header "Authorization: Bearer token123" http://localhost:3000
go run ./script/browser-debug --header "X-API-Key: secret" --header "Cookie: session=abc" http://localhost:3000

# Combine options
go run ./script/browser-debug --headless --header "Authorization: Bearer xyz" http://localhost:3000
```

## Options

- `URL` (required) — The URL to navigate to
- `--headless` — Run browser in headless mode (no visible window, works over SSH)
- `--new` — Force start a new browser instance (ignore existing)
- `--header "Key: Value"` — Add custom HTTP header (can be used multiple times)

## Interactive Commands

Once the browser is ready, use these commands in the REPL:

| Command | Description |
|---------|-------------|
| `eval <js>` | Evaluate JavaScript expression and print result |
| `styles <selector>` | Show computed styles (display, flex, overflow, height, etc.) for an element |
| `hierarchy <selector>` | Show parent chain with flex/overflow/height styles (useful for debugging layout) |
| `screenshot` | Take a full-page screenshot, saved to `/tmp/browser_debug_*.png` |
| `scroll <selector>` | Scroll an element into view |
| `nav <url>` | Navigate to a new URL |
| `api GET <url>` | Make GET request (headers set via --header are included) |
| `api POST <url> <body>` | Make POST request (headers set via --header are included) |
| `wait <duration>` | Wait for a duration (e.g., `wait 3s`) |
| `quit` / `exit` | Exit the tool |

Typing any other text evaluates it as JavaScript directly.

## Authentication

Authentication headers can be passed using the `--header` flag. These headers are:
- Sent with all browser page requests automatically
- Included in `api` command requests

Example:
```sh
go run ./script/browser-debug --header "Authorization: Bearer token123" --header "X-Custom: value" http://localhost:3000
```

## Requirements

- Chrome/Chromium installed (chromedp will find it automatically)
- Go 1.21+ with dependencies:
  ```sh
  go get github.com/chromedp/chromedp
  go get github.com/chromedp/cdproto
  go get github.com/xhd2015/less-gen/flags
  ```

## Examples

```
> styles .container
{
  "display": "flex",
  "flexDirection": "column",
  "overflow": "hidden",
  "overflowY": "auto",
  "scrollHeight": 1039,
  "clientHeight": 716
}

> hierarchy .container
html | display:block ...
  body | display:block ...
    div#root | display:flex ...
      ...
        div.container | display:flex flex:... overflow:hidden/auto ...

> screenshot
Screenshot saved to /tmp/browser_debug_1739349123.png

> api GET http://localhost:3000/api/info
Status: 200 OK
{"code":0,"data":{...}}
```
