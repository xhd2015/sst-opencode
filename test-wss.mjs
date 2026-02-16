import puppeteer from "puppeteer"

const browser = await puppeteer.launch({
  headless: "new",
  args: ["--no-sandbox", "--disable-setuid-sandbox", "--disable-proxy"],
})

const page = await browser.newPage()

let wsConnected = false

page.on("console", (msg) => {
  const text = msg.text()
  console.log("CONSOLE:", msg.type(), text)
  if (text.includes("[vite] connected") || text.includes("connecting")) {
    wsConnected = true
  }
})

page.on("pageerror", (err) => {
  console.log("PAGE ERROR:", err.message)
})

try {
  await page.goto("https://port-4444-ae2842d.xhd2015.xyz/", {
    waitUntil: "networkidle2",
    timeout: 15000,
  })

  await new Promise((r) => setTimeout(r, 5000))

  if (wsConnected) {
    console.log("\n✓ WebSocket connected successfully!")
  } else {
    console.log("\n✗ WebSocket NOT connected")
  }

  const rootContent = await page.$eval("#root", (el) => el.innerHTML)
  console.log("Root content length:", rootContent.length)
} catch (e) {
  console.log("Error:", e.message)
}

await browser.close()
