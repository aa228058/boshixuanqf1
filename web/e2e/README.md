# TG Sender browser audit (Playwright)

Playwright runs headlessly on the Linux server. A desktop computer is not required; commands can be run from an SSH terminal app on a phone.

## 1. Install

After this branch is merged and pulled on the server:

```bash
cd /www/tg-sender/web/e2e
npm install
npx playwright install --with-deps chromium
```

If the production host is Alpine Linux, do not install Debian browser dependencies into the production app container. Use a compatible Debian/Ubuntu Playwright container or a separate test environment.

## 2. Save tenant session

```bash
cd /www/tg-sender/web/e2e
BASE_URL=https://boshixuan.duckdns.org npm run auth:setup -- tenant
```

Enter the tenant username and password at the SSH terminal prompts. Password input is masked. The script stores only browser storage state; it never writes the password. The `.auth` directory is mode 700 and the state file mode 600. The file contains live session credentials and must remain on the server.

Tenant access tokens are memory-only in the application. The saved HttpOnly refresh cookie plus the non-secret session marker allow the app to restore a tenant session in a fresh browser context.

## 3. Run tenant UI audit

```bash
BASE_URL=https://boshixuan.duckdns.org E2E_SKIP_WEBSERVER=1 npm test
```

If you have a safe test account workspace, optionally add its ID with `E2E_ACCOUNT_ID`. Do not use an account ID that would lead to production actions; the suite only navigates and captures pages.

## 4. Run admin UI audit

Admin access tokens are memory-only and there is no admin refresh flow. Therefore an admin `storageState` file cannot restore an authenticated admin session. The admin suite logs in within each test's browser context instead.

Enter credentials without putting the password in shell history:

```bash
read -r -p "管理员用户名: " E2E_ADMIN_USERNAME
read -r -s -p "管理员密码: " E2E_ADMIN_PASSWORD
printf '\\n'
export E2E_ADMIN_USERNAME E2E_ADMIN_PASSWORD
BASE_URL=https://boshixuan.duckdns.org E2E_SKIP_WEBSERVER=1 npm test -- --project=admin-desktop
unset E2E_ADMIN_USERNAME E2E_ADMIN_PASSWORD
```

The credentials are held in the test process environment only; do not enable shell tracing (`set -x`) while running this command.

## 5. Reports

```bash
npx playwright show-report
```

Reports, screenshots, traces, and videos are stored under `playwright-report/` and `test-results/`. They may contain private page data. Keep them private and do not publish them publicly.

## Coverage and safety

- Public: tenant login, admin login, registration, and not-found page.
- Tenant: all static tenant routes declared in the router, plus account workspace routes when `E2E_ACCOUNT_ID` is supplied.
- Admin: all static admin routes declared in the router, after a fresh login in each test.
- The suite navigates, checks rendering/runtime errors, checks selected horizontal overflow, and captures screenshots. It does not submit business forms, run tasks, send Telegram messages, invite users, delete records, import/export data, or change settings.

Dynamic detail routes requiring a real record ID are intentionally excluded. Add them only with a designated test record.
