# Clablings

Interactive network practice and protocol mastery platform CLI.

```bash
clablings watch [--root DIR]        # watch mode
clablings run <exercise_id>         # jump to an exercise
clablings reset [--hard]            # fast / hard reset
clablings list [--json]             # progress
clablings test [<id>|--all] [--format text|json]  # headless verification
clablings debug [--sudo] [--root DIR] # launch clab-tui on this chapter's shared topology
clablings read [<chapter_id>] [--lang en|zh] # concept + RFC reading slices
```

The exercise library lives at the repository root `clab-exercises/`; see
`clab-exercises/README.md` for usage. The design document is at
`docs/superpowers/specs/2026-09-17-clablings-design.md`.

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
