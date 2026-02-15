const puppeteer = require("puppeteer")

;(async () => {
  const browser = await puppeteer.launch({
    headless: "new",
    args: ["--no-sandbox", "--disable-setuid-sandbox"],
  })

  const page = await browser.newPage()

  // Capture console messages
  page.on("console", (msg) => {
    console.log("CONSOLE:", msg.type(), msg.text())
  })

  // Capture page errors
  page.on("pageerror", (err) => {
    console.log("PAGE ERROR:", err.message)
  })

  // Capture failed requests
  page.on("requestfailed", (req) => {
    const failure = req.failure()
    console.log("FAILED REQUEST:", req.url(), failure ? failure.errorText : "unknown")
  })

  try {
    await page.goto("https://opencode-dev-server-ae2842d.xhd2015.xyz/", {
      waitUntil: "networkidle0",
      timeout: 30000,
    })

    // Wait a bit for JS to execute
    await new Promise((r) => setTimeout(r, 3000))

    const html = await page.content()
    console.log("Page HTML length:", html.length)

    // Check if #root has content
    const rootContent = await page.$eval("#root", (el) => el.innerHTML)
    console.log("Root content length:", rootContent.length)
    console.log("Root content:", rootContent.substring(0, 500))
  } catch (e) {
    console.log("Error:", e.message)
  }

  await browser.close()
})()
