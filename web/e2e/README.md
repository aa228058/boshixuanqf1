# TG Sender browser audit (Playwright)

Playwright runs headlessly on the Linux server. A desktop computer is not required; you can run the commands from an SSH terminal app on your phone.

## 1. Install on the server

After this branch is merged and pulled on the server:

```bash
cd /www/tg-sender/web/e2e
npm install
npx playwright install --with-deps chromium
```

If the server uses Alpine Linux, use a Debian/Ubuntu Playwright container instead of installing Chromium system packages into the production app container.

## 2. Save tenant login state

Run this in your server SSH session:

```bash
cd /www/tg-sender/web/e2e
BASE_URL=https://boshixuan.duckdns.org npm run auth:setup -- tenant
```

Enter the tenant username and password at the terminal prompts. Password characters are masked. Credentials are not written to command history or passed as command-line arguments.

For the admin console, repeat with a separate admin session:

```bash
BASE_URL=https://boshixuan.duckdns.org npm run auth:setup -- admin
```

The resulting `.auth/user.json` and `.auth/admin.json` contain live session cookies/storage. Keep them on the server only. Never commit, publish, or upload them as CI artifacts.

## 3. Run against the live site

```bash
cd /www/tg-sender/web/e2e
BASE_URL=https://boshixuan.duckdns.org E2E_SKIP_WEBSERVER=1 npm test
```

Optional account-workspace routes can be included with `E2E_ACCOUNT_ID`.

## 4. Read the report

```bash
npx playwright show-report
```

The HTML report, screenshots, traces, and videos are stored under `playwright-report/` and `test-results/`. These artifacts may contain private page data; keep them private and do not publish them publicly.

## Safety

The current suite is read-only: it navigates pages, inspects DOM/layout, records browser errors, and captures screenshots. It does not submit business forms, execute tasks, send Telegram messages, invite users, delete records, import/export data, or change settings.
