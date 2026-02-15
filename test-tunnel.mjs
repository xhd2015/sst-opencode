import puppeteer from "puppeteer"

const browser = await puppeteer.launch({
  headless: "new",
  args: ["--no-sandbox", "--disable-setuid-sandbox", "--disable-proxy"],
})

const page = await browser.newPage()

page.on("console", (msg) => {
  console.log("CONSOLE:", msg.type(), msg.text())
})

page.on("pageerror", (err) => {
  console.log("PAGE ERROR:", err.message)
})

page.on("response", (response) => {
  if (response.status() >= 400) {
    console.log("ERROR:", response.url(), response.status())
  }
})

try {
  await page.goto("https://port-6096-ae2842d.xhd2015.xyz/", {
    waitUntil: "networkidle2",
    timeout: 30000,
  })

  await new Promise((r) => setTimeout(r, 3000))

  const rootContent = await page.$eval("#root", (el) => el.innerHTML)
  console.log("\n=== Root content ===")
  console.log("Length:", rootContent.length)
  console.log(rootContent.substring(0, 500))
} catch (e) {
  console.log("Error:", e.message)
}

await browser.close()
