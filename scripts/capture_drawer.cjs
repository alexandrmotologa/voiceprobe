const path = require('path');
const { spawn } = require('child_process');
const puppeteer = require('puppeteer-core');

const CHROME_PATH = 'C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe';
const PORT = 7986;
const BASE_URL = `http://localhost:${PORT}`;
const OUTPUT_DIR = path.resolve(__dirname, '../docs/images');

async function sleep(ms) {
  return new Promise((r) => setTimeout(r, ms));
}

async function main() {
  const server = spawn(
    path.resolve(__dirname, '../voiceprobe.exe'),
    ['listen', '-http-port', String(PORT), '-rtp-port', '8012'],
    { cwd: path.resolve(__dirname, '..'), stdio: 'ignore' }
  );

  // Wait for server ready
  for (let i = 0; i < 20; i++) {
    await sleep(300);
    try {
      const resp = await fetch(`${BASE_URL}/api/v1/status`);
      if (resp.ok) break;
    } catch (e) {}
  }

  const browser = await puppeteer.launch({
    executablePath: CHROME_PATH,
    headless: 'new',
    args: ['--no-sandbox', '--window-size=1600,1000'],
  });

  try {
    const page = await browser.newPage();
    await page.setViewport({ width: 1440, height: 940, deviceScaleFactor: 2 });
    await page.goto(BASE_URL, { waitUntil: 'domcontentloaded' });
    await sleep(1500);

    // Open Incident Archives Drawer
    await page.evaluate(() => {
      const btn = document.querySelector('button[title*="Incident archives"]');
      if (btn) btn.click();
    });
    await sleep(1000);

    const auditPath = path.join(OUTPUT_DIR, 'voiceprobe_audit.png');
    await page.screenshot({ path: auditPath, fullPage: false });
    console.log('✓ Captured authentic docs/images/voiceprobe_audit.png (Incidents Archive Drawer)');
  } finally {
    await browser.close();
    server.kill();
  }
}

main().catch(err => {
  console.error('Drawer capture failed:', err);
  process.exit(1);
});
