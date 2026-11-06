# TG Sender browser audit (Playwright)

This is an isolated browser-test package. It does not add dependencies to the Vue app package.

## Install

```bash
cd web/e2e
npm install
npx playwright install chromium
```

## Save an authenticated session

Use a trusted local machine. This opens a visible browser; sign in manually, then return to the terminal and press Enter:

```bash
BASE_URL=https://boshixuan.duckdns.org npm run auth:setup
```

The resulting `.auth/user.json` contains session credentials. Never commit it, upload it to CI artifacts, or share it.\n\nFor the admin console, save a separate session:\n\n```bash\nBASE_URL=https://boshixuan.duckdns.org LOGIN_PATH=/admin/login AUTH_STATE_PATH=.auth/admin.json npm run auth:setup\n```

## Run

Build the frontend first, then run tests against Vite preview:

```bash
cd web
npm install
npm run build
cd e2e
npm test
```

Without `.auth/user.json`, only public-page smoke tests run. With it, desktop and mobile authenticated route checks are enabled. Set `E2E_ACCOUNT_ID` to include account-workspace routes.

For a different target, set `BASE_URL`. Set `E2E_SKIP_WEBSERVER=1` when the target server is already running.

## Safety

Tests are read-only. They navigate pages, inspect DOM/layout, and take screenshots. They do not submit forms, click task execution controls, send Telegram messages, invite users, delete records, import/export data, or change settings.

Reports and screenshots are local artifacts under `playwright-report/` and `test-results/`.
