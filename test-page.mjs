import puppeteer from "puppeteer"

const browser = await puppeteer.launch({
  headless: "new",
  args: ["--no-sandbox", "--disable-setuid-sandbox"],
})

const page = await browser.newPage()

page.on("console", (msg) => {
  console.log("CONSOLE:", msg.type(), msg.text())
})

page.on("pageerror", (err) => {
  console.log("PAGE ERROR:", err.message)
})

page.on("requestfailed", (req) => {
  const failure = req.failure()
  console.log("FAILED REQUEST:", req.url(), failure ? failure.errorText : "unknown")
})

page.on("response", (response) => {
  if (response.status() >= 400) {
    console.log("ERROR RESPONSE:", response.url(), response.status())
  }
})

try {
  await page.goto("https://opencode-dev-server-ae2842d.xhd2015.xyz/", {
    waitUntil: "domcontentloaded",
    timeout: 15000,
  })

  await new Promise((r) => setTimeout(r, 5000))

  const rootContent = await page.$eval("#root", (el) => el.innerHTML)
  console.log("\n=== Root content ===")
  console.log("Length:", rootContent.length)
} catch (e) {
  console.log("Error:", e.message)
}

await browser.close()
