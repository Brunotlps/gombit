# Security Policy

## Reporting a vulnerability

**Do not open a public issue for a security vulnerability.**

Report privately through GitHub Security Advisories:

> [Report a vulnerability](https://github.com/gombit-dev/gombit/security/advisories/new)

If you can't use that form, email **leonardo.aa88@gmail.com** with `gombit
security` in the subject.

Please include:

- affected version (`gombit version --short`) or commit;
- the component — framework runtime, CLI/generator, generated application code,
  or the admin SPA;
- reproduction steps or a proof of concept;
- impact as you see it.

### What to expect

| Stage | Target |
| --- | --- |
| Acknowledgement | 3 business days |
| Initial assessment | 10 business days |
| Fix or mitigation plan | depends on severity; communicated in the assessment |
| Public disclosure | coordinated, up to 90 days from the report |

We'll credit you in the advisory and release notes unless you'd rather stay
anonymous.

## Supported versions

Gombit is pre-1.0. Fixes land on `main` and ship in the next release; there are
no long-term support branches.

| Version | Supported |
| --- | --- |
| `0.1.x` | ✅ |
| `< 0.1` (unreleased `main` before the first tag) | ❌ |

Pre-1.0 minor versions may contain breaking changes. Pin an exact version and
read the [CHANGELOG](CHANGELOG.md) before upgrading.

## Scope

**In scope** — vulnerabilities in this repository:

- the framework runtime (`framework`, `auth`, `admin`, `database`, `config`,
  `contract`, `cache`);
- authentication and session handling, including the cookie/CSRF mode
  ([`docs/auth-cookie.md`](docs/auth-cookie.md)) and admin permission
  enforcement;
- the admin SPA under `/admin/` (`internal/adminui`);
- the CLI and generators, including code that `gombit new` and
  `gombit make resource` **emit** — an insecure default in generated output is a
  vulnerability in Gombit;
- the release pipeline and published artifacts.

**Out of scope:**

- misconfiguration of an application *you* generated — for example running with
  a weak or shared `GOMBIT_JWT_SECRET`, disabling CSRF protection, or exposing
  `/docs` in production. `gombit doctor` flags several of these;
- **`VITE_*` environment variables.** By design everything under `VITE_*` is
  compiled into the frontend bundle and is public. Putting a secret there is a
  configuration error, not a framework vulnerability
  (build plan §5);
- vulnerabilities in third-party dependencies without a demonstrated impact on
  Gombit — report those upstream, though we're glad to hear about them;
- findings from automated scanners with no accompanying exploitability analysis;
- denial of service via unbounded self-inflicted load (for example running the
  dev server on a public interface).

## Hardening notes

The threat models for the areas most likely to matter are documented rather
than implicit:

- Bearer JWT and token handling: [`docs/auth.md`](docs/auth.md) — the access
  token is held **in memory, never `localStorage`**;
- cookie/session auth and the CSRF double-submit design:
  [`docs/auth-cookie.md`](docs/auth-cookie.md);
- admin permissions, groups, and the superuser bypass:
  [`docs/admin.md`](docs/admin.md);
- configuration, including which values are redacted by `gombit config show`:
  [`docs/config.md`](docs/config.md).

## Verifying a release

Release archives ship with `SHA256SUMS.txt`:

```bash
sha256sum -c SHA256SUMS.txt --ignore-missing
```

See [`docs/installation.md`](docs/installation.md).
