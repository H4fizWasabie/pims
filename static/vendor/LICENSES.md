# Vendored third-party files

Self-hosted so the app works when a CDN is slow, blocked or gone. Update by replacing
the file and bumping the version here.

| Path | Project | Version | License |
|------|---------|---------|---------|
| `react.production.min.js`, `react-dom.production.min.js` | React | 18.3.1 | MIT |
| `xlsx.full.min.js` | SheetJS CE | 0.20.3 | Apache-2.0 |
| `fontawesome/` | Font Awesome Free | 6.4.0 | Fonts: SIL OFL 1.1, CSS: MIT, Icons: CC BY 4.0 (<https://fontawesome.com/license/free>) |
| `fonts/` | Poppins, Inter (Google Fonts, latin subset) | as served 2026-10 | SIL OFL 1.1 |

The Font Awesome CSS is trimmed to WOFF2 (the `.ttf` fallbacks are removed).
