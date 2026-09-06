# Working in this repo

A fork of `mschirrmeister/cert-manager-webhook-cloudns`, which is unmaintained
(its last commit is `8930362`, June 2024). We own it now; there is no upstream
to merge from and no rebase burden.

Forgejo (`git.greyrock.io/todd/cert-manager-webhook-cloudns`) is the only git
remote. GitHub is reached through a server-side push mirror — never add a GitHub
remote. The branch is `master`, inherited from upstream and deliberately kept.

## Releasing

Releases are tag-driven. Pushing a `v*` tag runs `.forgejo/workflows/release.yaml`,
which tests, builds the image, packages the chart, and pushes both to GHCR.

**The git tag is the only version input.** `helm package` is called with
`--version` and `--app-version` derived from it, and `image.tag` defaults to
`.Chart.AppVersion`, so the chart and the image it deploys cannot drift apart.
Do not hand-edit the version in `Chart.yaml` to cut a release — tag instead.
`Chart.yaml`'s version is a floor, not the authority.

### Every release gets notes

Write them into the annotated tag message *before* pushing the tag — the tag is
the record, and there is no other place notes are kept.

```sh
git tag -a v2.3.0 -F - <<'EOF'
v2.3.0 — <one-line summary>

<what changed, and why it changed>

Upgrade notes:
- <anything a consumer must do, or "None.">
EOF
git push origin v2.3.0
```

Say what moved and what it fixes. "Bump dependencies" is not a release note —
name the dependency, the versions, and the reason. If a change alters behaviour
on upgrade, or requires anything of whoever consumes the chart, that belongs
under upgrade notes even when the answer is "None."

Nothing here needs the Forgejo API, so no Authorized Integration is required.
`permissions:` is a GitHub Actions concept that Forgejo ignores — don't add it.

## Testing

`go test ./...` **fails on the root package**, and does so on pristine upstream
too. `main_test.go` is cert-manager's DNS01 conformance suite and needs
kubebuilder envtest binaries (`etcd`, `kube-apiserver`) supplied via
`TEST_ASSET_*` — see `make verify` and `scripts/fetch-test-binaries.sh`. Do not
report that failure as a regression.

The meaningful local command is:

```sh
go test ./cloudns/... -count=10
```

`-count` matters. `TXTRecords` is a `map[string]TXTRecord` and Go randomises map
iteration, so the `FindTxtRecord` tests can pass or fail by luck in a single run.

## Builds are amd64 only

By choice — the cluster is a single amd64 node. The Forgejo runner also cannot
mount `binfmt_misc`, so `docker/setup-qemu-action` fails there outright with
`error: no such device`. **Do not add QEMU or arm64 platforms.** If arm64 is ever
genuinely needed, cross-compile in the Dockerfile with `ARG TARGETOS TARGETARCH`
and `--platform=$BUILDPLATFORM` rather than emulating — note the final stage's
`RUN apk add` would then also need replacing with a `COPY`, since a `RUN` in a
foreign-arch stage needs emulation.

## Linting workflows

`actionlint` needs `.forgejo/actionlint.yaml` or it rejects correct files —
Forgejo's full-URL `uses:` form and the `docker` runner label both trip it:

```sh
actionlint -config-file .forgejo/actionlint.yaml .forgejo/workflows/release.yaml
```

## Bumping dependencies

They cannot be bumped piecemeal. `github.com/cert-manager/cert-manager` dictates
which Kubernetes libraries are usable, so **bump it first and let everything else
follow**. Bumping lego on its own pulls otel/grpc transitives that break the
pinned `k8s.io/apiserver` with `undefined: otelhttp.WithPublicEndpoint`.

Keep the cert-manager library aligned with the cert-manager actually running in
the cluster. Raising it may raise `go.mod`'s `go` directive, and the Dockerfile's
`golang:` base must move with it or the image build fails.

lego v4 is frozen — no release since v5.0.0 in May 2026, so it gets no security
fixes. We are on v5. Three things moved in v5 and will bite anyone reading old
examples:

- `platform/config/env` → `platform/env`
- `dns01.FindZoneByFqdn(fqdn)` → `dns01.DefaultClient().FindZoneByFqdn(ctx, fqdn)`
- `platform/tester` → `internal/tester`, so it is unimportable. `cloudns/cloudns_test.go`
  reimplements the few methods it used on top of `t.Setenv`, which restores
  automatically. `t.Setenv(name, "")` reads as unset to lego's `env.Get`.

`env.Get` still resolves `<VAR>_FILE`, which is what the mounted `/creds`
credentials depend on. Verify that still holds before any lego upgrade.

## Why the TXT cleanup fix exists

`Client.FindTxtRecord` originally matched on host and type only, ignored the
record's value, and returned the first hit. Since records arrive in a map, it
deleted an arbitrary TXT record at that name rather than the challenge's own.
`CleanUp` already received the challenge key and discarded it; it is now passed
through and compared.

**The reason is not apex-plus-wildcard.** That was the original theory and it is
wrong: cert-manager will not run two DNS01 challenges for the same DNS name
concurrently, even across independent Orders, so it never collides with itself.
The real hazard is *other* ACME clients on the same zone — this cluster also has
traefik and codswallop holding ClouDNS credentials, none of which coordinate.
Deleting one of their in-flight records surfaces on their side as
"Incorrect TXT record found".

### Testing cleanup behaviour

You cannot get two concurrent records out of cert-manager. Use a decoy:

1. Add a TXT record at `_acme-challenge` in the zone, standing in for another client.
2. Create a Certificate for that domain against `letsencrypt-staging`.
3. Wait until both records exist.
4. Delete the Certificate — that triggers `CleanUp`.
5. The decoy must survive and only the challenge's own record disappear.

Query ClouDNS without handling credentials by reading them inside the pod:

```sh
kubectl -n cert-manager exec deploy/cert-manager-webhook-cloudns \
  -c cert-manager-webhook-cloudns -- sh -c \
  'ID=$(cat /creds/auth_id); PW=$(cat /creds/auth_password);
   wget -qO- "https://api.cloudns.net/dns/records.json?auth-id=$ID&auth-password=$PW&domain-name=greyrock.io&type=TXT"'
```

## Credentials and packages

CI authenticates to GHCR with Forgejo Actions secrets `GHCR_USERNAME` and
`GHCR_TOKEN` — uppercase, because Forgejo normalises secret names, and the token
must be a GitHub **classic** PAT with `write:packages`. Fine-grained tokens do
not work with GHCR.

`org.opencontainers.image.source` must point at the **GitHub** URL, not Forgejo,
and must be set as a manifest **annotation** — GHCR reads the annotation, not the
config label, when linking a package to its repository. A label alone leaves the
image package orphaned. Helm covers the chart half via `sources[0]` in `Chart.yaml`.

New GHCR packages are created **private** regardless of repo visibility. Both of
ours are public; a private one would need an imagePullSecret, a `secretRef` on
the OCIRepository, and a Renovate hostRule.

## The consuming side

home-ops pulls this chart at
`kubernetes/apps/cert-manager/cert-manager-webhook-cloudns/`. Two things there
are easy to get wrong:

- The `OCIRepository` needs the `layerSelector` with
  `mediaType: application/vnd.cncf.helm.chart.content.v1.tar+gzip`, or Flux
  treats the chart as a plain manifest tarball.
- `groupName: acme.ixon.cloud` is inherited from the original `ixoncloud` repo
  and is **not** a typo. Changing it means re-registering the APIService.

The chart marks the credentials volume `optional: true` so the pod can start
before its secret exists — that is what allows the webhook to be installed early
in the bootstrap helmfile, ahead of external-secrets. Nothing reads the
credentials until a challenge arrives and they are re-read per challenge, so it
recovers on its own once the secret appears.

Renovate owns the version pin in home-ops. Do not bump it by hand.

## A stalled challenge is usually not a fault

cert-manager's DNS01 self-check retries with backoff, so a challenge can sit on
"not yet propagated" for minutes after the record is already live and resolving.
Propagation to public resolvers is on the order of seconds. Wait it out before
concluding a resolver or the webhook is broken.

Check the record with DNS-over-HTTPS against two independent resolvers rather
than trusting one answer:

```sh
curl -sH 'accept: application/dns-json' 'https://1.1.1.1/dns-query?name=<name>&type=TXT'
curl -sH 'accept: application/dns-json' 'https://dns.google/resolve?name=<name>&type=TXT'
```

To see what the provider actually holds, ask the ClouDNS API rather than DNS.
