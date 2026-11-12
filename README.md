# Telegram Sender — UI/UX Workbench

This branch is a frontend-only workspace copied from the application's real `web/` project.

## Scope
- The original Vue 3 pages, layouts, reusable components, styles, themes, assets, router, and frontend tooling are preserved.
- The Go server, database, migrations, Docker deployment files, and Telegram protocol implementation are not included in this branch.
- This branch is intended for isolated frontend visual and interaction work. It is not the production application branch.

## Current status
This is the complete frontend source baseline. API/store calls still need to be replaced by local mock adapters before this branch can be considered fully standalone. Do not point it at production services.

## Run
```bash
npm install
npm run dev
```

The original frontend development proxy is retained until the mock adapter pass is complete; configure no production API target.
