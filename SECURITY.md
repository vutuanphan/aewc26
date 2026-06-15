# Security Policy

## Reporting a vulnerability

Please **do not** open a public issue for security problems. Instead, use
GitHub's private vulnerability reporting (Security → Report a vulnerability) or
contact the maintainer privately. We'll acknowledge within a few days.

## Scope & posture

AEWC26 manages play-money among a closed, admin-created roster. It implements:

- bcrypt password hashing
- server-side sessions (random tokens, HttpOnly cookies)
- CSRF protection (double-submit token) on all state-changing POSTs
- parameterized SQL everywhere (no string-built queries from user input)
- `Secure` cookies when served over HTTPS

## Operator responsibilities

- Keep `.env` private; never commit it. Rotate `ODDS_API_KEY` if it leaks.
- Serve the app behind HTTPS (nginx/Caddy/Traefik).
- Use a strong `AEWC_ADMIN_PASSWORD`.
- Back up the SQLite volume (`/data`).

## Known limitations

- No login rate-limiting yet (contributions welcome).
- Single SQLite connection — designed for small private groups, not high
  concurrency.
