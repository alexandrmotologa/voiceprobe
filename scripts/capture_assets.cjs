const fs = require('fs');
const path = require('path');
const { spawn, execSync } = require('child_process');
const puppeteer = require('puppeteer-core');

const CHROME_PATH = 'C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe';
const PORT = 7985;
const BASE_URL = `http://localhost:${PORT}`;
const OUTPUT_DIR = path.resolve(__dirname, '../docs/images');
const FRAMES_DIR = path.resolve(__dirname, '../temp_frames');

async function sleep(ms) {
  return new Promise((r) => setTimeout(r, ms));
}

async function startServer() {
  console.log(`Starting voiceprobe on port ${PORT}...`);
  const serverProc = spawn(
    path.resolve(__dirname, '../voiceprobe.exe'),
    ['listen', '-http-port', String(PORT), '-rtp-port', '8010'],
    {
      cwd: path.resolve(__dirname, '..'),
      stdio: 'pipe',
    }
  );

  serverProc.stdout.on('data', (d) => {
    console.log('[server]', d.toString().trim());
  });
  serverProc.stderr.on('data', (d) => {
    console.error('[server:err]', d.toString().trim());
  });

  // Wait for server ready
  for (let i = 0; i < 20; i++) {
    await sleep(500);
    try {
      const resp = await fetch(`${BASE_URL}/api/v1/status`);
      if (resp.ok) {
        console.log('✓ voiceprobe engine is ready on port', PORT);
        return serverProc;
      }
    } catch (e) {
      // retry
    }
  }
  throw new Error('Server failed to start in 10s');
}

async function main() {
  if (!fs.existsSync(OUTPUT_DIR)) {
    fs.mkdirSync(OUTPUT_DIR, { recursive: true });
  }
  if (!fs.existsSync(FRAMES_DIR)) {
    fs.mkdirSync(FRAMES_DIR, { recursive: true });
  }

  const server = await startServer();

  console.log('Launching headless Chrome via puppeteer-core...');
  const browser = await puppeteer.launch({
    executablePath: CHROME_PATH,
    headless: 'new',
    args: [
      '--no-sandbox',
      '--disable-setuid-sandbox',
      '--disable-dev-shm-usage',
      '--disable-gpu',
      '--window-size=1600,1000',
    ],
  });

  try {
    const page = await browser.newPage();

    // 1. Desktop Dashboard Screenshot
    console.log('Navigating to Desktop view...');
    await page.setViewport({ width: 1440, height: 940, deviceScaleFactor: 2 });
    await page.goto(BASE_URL, { waitUntil: 'domcontentloaded' });
    await sleep(2000);

    // Trigger synthetic cloning attack simulation
    console.log('Triggering synthetic attack simulation...');
    await page.evaluate(async () => {
      await fetch('/api/v1/simulate', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ type: 'cloned', duration_sec: 15 }),
      });
    });

    // Let telemetry flow for 3.5 seconds so threat gauge hits critical and waterfall animates
    console.log('Waiting for acoustic telemetry to stabilize in CRITICAL...');
    await sleep(3500);

    const desktopPath = path.join(OUTPUT_DIR, 'voiceprobe_dashboard.png');
    await page.screenshot({ path: desktopPath, fullPage: false });
    console.log('✓ Captured docs/images/voiceprobe_dashboard.png');

    // 2. Mobile View Screenshot
    console.log('Navigating to Mobile view (390x844)...');
    await page.setViewport({ width: 390, height: 844, deviceScaleFactor: 2 });
    await sleep(1000);
    const mobilePath = path.join(OUTPUT_DIR, 'voiceprobe_mobile.png');
    await page.screenshot({ path: mobilePath, fullPage: false });
    console.log('✓ Captured docs/images/voiceprobe_mobile.png');

    // 3. File Analyzer Modal Screenshot
    console.log('Opening File Analyzer Modal...');
    await page.setViewport({ width: 1440, height: 940, deviceScaleFactor: 2 });
    await sleep(500);

    // Click the upload/analyze icon in Header
    await page.evaluate(() => {
      const uploadBtn = document.querySelector('button[title*="Inspect audio file"]');
      if (uploadBtn) uploadBtn.click();
    });
    await sleep(800);

    // Click "Test Cloned Deepfake" preset
    console.log('Triggering file analysis for cloned sample...');
    await page.evaluate(() => {
      const buttons = Array.from(document.querySelectorAll('button'));
      const testClonedBtn = buttons.find((b) => b.textContent.includes('Test Cloned Deepfake'));
      if (testClonedBtn) testClonedBtn.click();
    });

    // Wait for analysis to complete and results to render
    await sleep(2500);

    // Switch filter to "Vocoder Band" to show interactive capability
    await page.evaluate(() => {
      const buttons = Array.from(document.querySelectorAll('button'));
      const vocoderBtn = buttons.find((b) => b.textContent.includes('Vocoder Band'));
      if (vocoderBtn) vocoderBtn.click();
    });
    await sleep(500);

    const analyzerPath = path.join(OUTPUT_DIR, 'voiceprobe_analyzer.png');
    await page.screenshot({ path: analyzerPath, fullPage: false });
    console.log('✓ Captured docs/images/voiceprobe_analyzer.png');

    // Close File Analyzer Modal
    await page.evaluate(() => {
      const closeBtn = document.querySelector('div[role="dialog"] button, .fixed button');
      if (closeBtn) closeBtn.click();
    });
    await sleep(500);

    // 4. Incidents Drawer / NIST SP 800-63B Certificate
    console.log('Navigating directly to NIST Forensic Certificate...');
    // Trigger simulation again to ensure incident is recorded
    await page.evaluate(async () => {
      await fetch('/api/v1/simulate', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ type: 'cloned', duration_sec: 4 }),
      });
    });
    await sleep(4200);

    // Find latest incident wav
    const incidentsResp = await (await fetch(`${BASE_URL}/api/v1/incidents`)).json();
    if (incidentsResp && incidentsResp.length > 0) {
      const latestWav = incidentsResp[incidentsResp.length - 1].wav_file;
      console.log('Loading NIST Audit Report for', latestWav);
      await page.goto(`${BASE_URL}/api/v1/incidents/${latestWav}/report`, { waitUntil: 'domcontentloaded' });
      await sleep(1000);
      const auditPath = path.join(OUTPUT_DIR, 'voiceprobe_audit.png');
      await page.screenshot({ path: auditPath, fullPage: false });
      console.log('✓ Captured docs/images/voiceprobe_audit.png');
    }

    // 5. Generate Animated Demo GIF
    console.log('Recording sequence of frames for voiceprobe_demo.gif...');
    await page.setViewport({ width: 1200, height: 750, deviceScaleFactor: 1.5 });
    await page.goto(BASE_URL, { waitUntil: 'domcontentloaded' });
    await sleep(2000);

    // Trigger simulation
    await page.evaluate(async () => {
      await fetch('/api/v1/simulate', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ type: 'cloned', duration_sec: 10 }),
      });
    });

    // Capture 36 frames over 3.6 seconds (10 fps)
    for (let i = 0; i < 36; i++) {
      const frameFile = path.join(FRAMES_DIR, `frame_${String(i).padStart(3, '0')}.png`);
      await page.screenshot({ path: frameFile });
      await sleep(100);
    }
    console.log('✓ Captured 36 frames for demo GIF');

    // Compile GIF with ffmpeg
    const gifPath = path.join(OUTPUT_DIR, 'voiceprobe_demo.gif');
    console.log('Compiling demo GIF with ffmpeg...');
    const ffmpegCmd = `ffmpeg -y -framerate 10 -i "${path.join(FRAMES_DIR, 'frame_%03d.png')}" -vf "fps=10,scale=880:-1:flags=lanczos,split[s0][s1];[s0]palettegen=max_colors=128[p];[s1][p]paletteuse=dither=bayer" "${gifPath}"`;
    execSync(ffmpegCmd, { stdio: 'inherit' });
    console.log('✓ Generated docs/images/voiceprobe_demo.gif');

    // Cleanup temp frames
    fs.rmSync(FRAMES_DIR, { recursive: true, force: true });
    console.log('✓ Cleaned up temporary frames');
  } finally {
    await browser.close();
    server.kill();
    console.log('✓ Closed browser and stopped voiceprobe engine.');
  }
}

main().catch((err) => {
  console.error('Asset capture failed:', err);
  process.exit(1);
});
