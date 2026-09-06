# Cert-Manager ClouDNS DNS01 Provider

A Cert-Manager DNS01 provider for ClouDNS.

## Note

This fork has `update-deps` branch merged and a few other changes that it works with recent versions of **Kubernetes** (1.23+) and **cert-manager** (1.7.2) to avoid errors and warnings.

## Changes in this fork

### TXT cleanup matches on record value

`Client.FindTxtRecord` previously queried ClouDNS by host and type only,
ignored the record's value, and returned the first match. Records come back
in a map, and Go randomises map iteration, so cleanup deleted an arbitrary
TXT record at that name rather than the one belonging to the challenge being
cleaned up. `CleanUp` already receives the challenge key and simply discarded
it; it is now passed through and compared.

This matters whenever something other than this cert-manager instance also
writes to `_acme-challenge.<zone>` — a second ACME client on the same domain,
for example. cert-manager serialises its own DNS01 challenges per name, so it
will not collide with itself; the hazard is another client's in-flight record
being deleted, which surfaces as "Incorrect TXT record found" on their side.

Verified end to end: with a foreign TXT record present at the challenge name,
cleanup removes only its own record and leaves the other intact.

### Credentials volume is optional

The chart mounts the ClouDNS credentials at `/creds`, and marks that volume
`optional: true` so the pod can start before the secret exists. Nothing reads
the credentials until a challenge arrives, and they are re-read per challenge,
so the webhook recovers on its own once the secret appears. This lets the
webhook be installed early during cluster bootstrap.

### Image tag follows the chart

`image.tag` defaults to `.Chart.AppVersion`, and releases are packaged with
`--app-version` set from the git tag, so the chart and the image it deploys
cannot drift apart.

### Builds

Releases are built on Forgejo Actions and published to GHCR as both a
container image and an OCI Helm chart. Images are `amd64` only.

## Configuration

Cert-Manager expects DNS01 providers to parse configuration from incoming webhook requests.

This can be used to have multiple Cert-Manager `Issuer` resources use the same instance of the provider with different credentials or configuration.

Because we currently don't need this and the LEGO library already has support for parsing environment variables (and files), we have opted to not use this.

The `testdata/config.json` file is there because the DNS01 provider conformance testing suite wants to mock the requests away, and needs a folder to load the data from.

### Environment Options

|Name|Required|Description|
|---|---|---|
|`GROUP_NAME`|yes|Used to organise cert-manager providers, this is usually a domain|
|`CLOUDNS_AUTH_ID_FILE`|yes|Path to file which contains ClouDNS Auth ID|
|`CLOUDNS_AUTH_PASSWORD_FILE`|yes|Path to file which contains ClouDNS Auth password|
|`CLOUDNS_TTL`|no, default: 60|ClouDNS TTL|
|`CLOUDNS_HTTP_TIMEOUT`|no, default: 30 seconds|ClouDNS API request timeout|

## Development

### Running DNS01 provider conformance testing suite

```bash
# Get kubebuilder
./scripts/fetch-test-binaries.sh

# Run testing suite
TEST_ZONE_NAME=<domain> CLOUDNS_AUTH_ID_FILE=.creds/auth_id CLOUDNS_AUTH_PASSWORD_FILE=.creds/auth_password go test -v .
```
