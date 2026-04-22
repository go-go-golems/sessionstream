# sessionstream-systemlab

`sessionstream-systemlab` is a separate app used to explain, exercise, and validate the public API boundaries of the `sessionstream` framework.

Goals:

- keep the playground separate from substrate code,
- consume only public `sessionstream` APIs,
- provide narrated labs for each implementation phase,
- make debugging and onboarding easier.

## Boundary contract

Systemlab may:

- import `github.com/go-go-golems/sessionstream/pkg/sessionstream` public packages and sibling `pkg/sessionstream/...` subpackages,
- expose its own HTTP endpoints and UI shell,
- exercise the same public seams later transports will use.

Systemlab may not:

- import `pkg/webchat` internals,
- introduce SEM-specific substrate types into `sessionstream`,
- reach around the public Hub/store/transport seams.

Current phases implemented:

- Phase 0 foundations / status page
- Phase 1 command -> event -> projection lab
- Phase 2 ordering / ordinals lab
- Phase 3 hydration / reconnect lab
- Phase 4 chat demo lab
- Phase 5 persistence / restart lab

Current chapter coverage in Systemlab:

- Phase 0 through Phase 5 have long-form markdown chapters served by the app
- Phase 0 through Phase 5 are framework-oriented labs owned by the `sessionstream` repo
- The old Phase 6 `cmd/web-chat` migration console is intentionally not part of this extracted Systemlab because it remains a downstream `pinocchio` concern

## Frontend and chapter file layout

The Systemlab browser UI is intentionally split so future labs do not accumulate into one large inline HTML file, and the long-form intern chapters live as editable markdown beside the app:

- `static/index.html` — app shell only
- `static/app.css` — shared styling
- `static/partials/*.html` — page-level markup fragments
- `static/js/main.js` — bootstrap + navigation
- `static/js/pages/*.js` — per-page behavior
- `static/js/api.js` / `static/js/dom.js` — shared helpers
- `chapters/*.md` — long-form textbook chapters served by the app and rendered onto the matching phase pages

When adding a new lab, prefer adding:

- a new partial,
- a new page module,
- and, when needed, a matching markdown chapter in `chapters/`

instead of growing `index.html` or one global script.

Run locally:

```bash
make systemlab-run
```

Validation helpers:

```bash
make test
make systemlab-build
make boundary-check
```

Then open:

- `http://localhost:8091/`
