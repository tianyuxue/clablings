# Security Policy

## Supported versions

Only the latest published release is supported with security fixes.

## Reporting a vulnerability

Please do not report security vulnerabilities through public GitHub issues.

Use GitHub's private vulnerability reporting on this repository:
<https://github.com/tianyuxue/clablings/security/advisories/new>

Include a description, reproduction steps, affected version, and impact. We will
acknowledge reports as soon as possible and coordinate a fix and disclosure.

## Scope notes

clablings is a local CLI. It shells out to `containerlab` and the Docker CLI,
generates a derived topology beside the exercise library, and can run exercise
commands and assertions on lab nodes. Reports about privilege escalation, command
injection via library content, unsafe handling of generated files, or unsafe
`sudo`/PATH behavior are in scope. Vulnerabilities in containerlab, Docker, or the
Linux kernel itself should be reported upstream.
