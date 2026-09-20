# Clablings

Interactive network practice and protocol mastery platform CLI.

```bash
clablings watch [--root DIR]        # watch mode
clablings run <exercise_id>         # jump to an exercise
clablings reset [--hard]            # fast / hard reset
clablings list [--json]             # progress
clablings test [<id>|--all] [--answers FILE] [--format text|json]  # headless verification
clablings debug [--sudo] [--root DIR] # launch clab-tui on this chapter's shared topology
clablings read [<chapter_id>] [--lang en|zh] # concept + RFC reading slices
clablings lint [--root DIR]          # validate the library
```

The exercise library lives at the repository root `clab-exercises/`; see
`clab-exercises/README.md` for usage. The design document is at
`docs/superpowers/specs/2026-09-17-clablings-design.md`.

## Watch keys

Watch prints a context block once per exercise: a progress bar, the exercise
title, its brief, and the key legend. Actions never reprint it.

- `Enter` verify the current exercise (on a quiz: start the quiz)
- `h` show hints
- `c` show the chapter's concept and RFC reading slices
- `p` / `n` go to the previous / next exercise (no status change)
- `r` / `f` hard / fast reset
- `q` quit (keep containers)
- `x` quit and destroy containers
- `Ctrl+C` quit

On `quiz` exercises, watch asks you to press `Enter` to start. Then enter option
numbers (comma-separated for multi-select); wrong answers show explanations and
re-prompt until correct, then advance. `h`/`c` show help and reprint the
question (not the legend).

## Configuration

Optional config file `~/.config/clablings/config.yaml`:

```yaml
language: en             # content language: en or zh
containerlab:
  bin_path: ""          # empty = resolve from PATH
docker:
  bin_path: ""          # empty = resolve from PATH
watch:
  poll_interval: 250ms
  debounce: 300ms
build:
  default_image: gcc:13
```

On a read failure or missing field an English warning is printed and the
default value is used.

## Notes

- Labs are isolated per library instance: the lab name is
  `<rootbasename>-<sha1(abs root)[:8]>-<topology name>` (override the instance id
  with `CLABLINGS_INSTANCE`). Container names are `clab-<lab>-<node>`.
- Because this containerlab build ignores `--name` on `destroy`/`redeploy`, each
  deploy generates a derived topology `clablings.<lab>.clab.yml` beside the
  original (gitignored in the library repo) and runs containerlab against it.
  The library directory must therefore be writable.
- `clablings debug` execs `clab-tui` as the current user; pass `--sudo` to launch
  it via `sudo <clab-tui> --file <derived>` (root). The first `sudo` may prompt
  for a password in the same terminal.
- Running clablings with `sudo` makes `git checkout` and the derived topology
  root-owned; run `sudo chown -R "$USER" <library>` afterwards if you need to
  edit the library as your normal user.
- Containers deployed by older builds (static topology names) are not matched by
  the instance-derived destroy; remove them with `containerlab destroy -t <topology>`.
- Content language defaults to English; select Chinese with `--lang zh`,
  `CLABLINGS_LANG=zh`, or `language: zh` in the config file (precedence in that
  order). CLI labels stay English. `clablings read` shows a chapter's concept
  and RFC reading slices.
- Install with `make install` (defaults to `~/.local/bin`; override with
  `PREFIX=...`/`DESTDIR=...`). Ensure the destination is on your `PATH`.
