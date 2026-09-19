# Clablings

Interactive network practice and protocol mastery platform CLI.

```bash
clablings watch [--root DIR]        # watch mode
clablings run <exercise_id>         # jump to an exercise
clablings reset [--hard]            # fast / hard reset
clablings list [--json]             # progress
clablings test [<id>|--all] [--format text|json]  # headless verification
```

The exercise library lives at the repository root `clab-exercises/`; see
`clab-exercises/README.md` for usage. The design document is at
`docs/superpowers/specs/2026-09-17-clablings-design.md`.

## Configuration

Optional config file `~/.config/clablings/config.yaml`:

```yaml
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
