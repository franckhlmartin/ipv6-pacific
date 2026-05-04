# Commit

Standard checklist for preparing a commit (bookerpal-style, adapted for this repository). Execute in order.

## 1. Check changes since last push

- Run `./scripts/commit-check-changes.sh` to list changes since the last push (it runs `git fetch` and `git diff --name-status` against the upstream branch).
- Or manually: `git fetch origin`, then `git diff --name-status origin/<branch>..HEAD` (use your current branch name).

Summarize what changed (backend, frontend, config, env, docs, scripts, tests).

## 2. Update documentation if needed before any commit or push

Based on the changed files, update docs only when the change warrants it:

- **New or removed env vars** → update `.env.example` and any relevant doc in `docs/` (e.g. `docs/development.md`, `docs/security.md`).
- **New HTTP handlers, services, or routes** → document in `README.md` and/or add `docs/api-reference.md` (or extend an existing API doc) if the public API surface grows.
- **New features or behavior** → add or update a section in `README.md` or the appropriate file under `docs/` (match the tone of existing docs).
- **Public-facing UI changes** → if you maintain a user-facing changelog (e.g. under `cmd/web/static` or `frontend/`), add an entry for changes that help users use the site.
- **New scripts or notable commands** → document in `docs/development.md` or a dedicated doc under `docs/`.
- **Schema or database changes** → update the relevant doc if this project gains persisted schema documentation (none today).

Do not change documentation for trivial tweaks (typos, style, refactors with no user-facing or structural impact). Prefer small, focused doc updates.

**Agent note:** After doc edits, let the user review documentation changes before proceeding to commit and push.

## 3. Commit and push

- Run `git status` to confirm what will be committed.
- Stage all intended changes: `git add …` (or `git add -A` if the full set is correct).
- Commit with a clear message that describes the change (e.g. "Add X", "Fix Y", "Update docs for Z"). Prefer conventional-style messages.
- **Push:** Run `git push origin <branch>`. In Cursor’s sandbox, push requires **full permissions** (e.g. `required_permissions: ["all"]` on the shell invocation); otherwise Git may fail to authenticate to GitHub (“could not read Username” / “Device not configured”).

Never commit secrets, `.env`, `.env.local`, or other sensitive files. If the user has uncommitted changes they do not want included, ask before staging.
