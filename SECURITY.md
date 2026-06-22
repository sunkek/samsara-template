# Security Policy

## Supported versions

This is a project template. Security fixes land on the default branch (`main`).
Adopters who have bootstrapped a project from this template should track `main`
for fixes and re-apply relevant patches to their fork.

## Reporting a vulnerability

**Do not open a public issue for security problems.**

Email **security@example.com** with:

- a description of the vulnerability and its impact,
- the affected component(s) and version/commit,
- reproduction steps or a proof of concept,
- any suggested remediation.

You will get an acknowledgement within **5 business days**. Once the issue is
confirmed, a fix and disclosure timeline will be agreed with you. Please give a
reasonable window to ship a fix before any public disclosure.

## Scope

In scope: the backend service, the auth domain, the bootstrap script, the
Docker/compose deployment manifests, and the CI configuration shipped in this
repository.

Out of scope: vulnerabilities in third-party dependencies (report those
upstream), and issues that require a misconfiguration explicitly warned against
in the docs (e.g. running with `CORS_ALLOW_ORIGINS=*` or `SSL_MODE=disable` in
production — see `CLAUDE.md`).
