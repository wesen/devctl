# Screenshot Playbook

This playbook documents how to capture and update devctl TUI screenshots.

## Goal

Produce consistent, colored PNG screenshots of the devctl TUI for README usage.

## Prerequisites

- `tmux` (used for deterministic captures)
- A `devctl` binary in `PATH` (or set `DEVCTL_BIN`)
- A repo with `.devctl.yaml` configured (or set `DEMO_REPO`)
- Python deps for rendering: `pyte` and `Pillow`

Install deps:

```bash
python3 -m pip install --user pyte pillow
```

## Capture + Render (recommended flow)

Run from the devctl repo root:

```bash
# Capture ANSI screens
DEVCTL_BIN=/tmp/devctl \
DEMO_REPO=/tmp/devctl-demo-repo \
./ttmp/2026/01/13/MO-018-ADD-DEVCTL-README--add-devctl-readme/scripts/01-capture-tui-screens.sh

# Render PNGs from ANSI dumps
python3 ./ttmp/2026/01/13/MO-018-ADD-DEVCTL-README--add-devctl-readme/scripts/02-ansi-to-png.py \
  --input-dir docs/screenshots
```

Outputs:

- `docs/screenshots/devctl-tui-dashboard.png`
- `docs/screenshots/devctl-tui-pipeline.png`
- `docs/screenshots/devctl-tui-plugins.png`

Remove ANSI intermediates when done:

```bash
rm -f docs/screenshots/*.ansi
```

## Building a devctl binary (optional)

If you do not already have a `devctl` binary in `PATH`:

```bash
go build -o /tmp/devctl ./cmd/devctl
```

## Using a different demo repo

You can use any repo with a valid `.devctl.yaml` and plugin. Set `DEMO_REPO` to the repo root. The capture script runs:

- `devctl tui --alt-screen=false`
- `u` to start services
- `tab` to switch views

If the TUI uses different keybindings, adjust the script in the ticket `scripts/` folder.

## Validation

- Open the PNGs to confirm readability and color correctness.
- Ensure the README references the correct paths.
- Run `devctl down` for the demo repo if the capture script was interrupted.

## Notes

- The capture script writes ANSI dumps to `docs/screenshots/` and does not overwrite PNGs unless you render again.
- Keep the screenshot filenames stable so README references do not break.
