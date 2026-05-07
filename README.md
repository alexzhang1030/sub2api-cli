# sub2api-cli

Go CLI for viewing today's Sub2API usage in a terminal dashboard.

## Install

```bash
go install github.com/alex/sub2api-cli@latest
```

From this checkout:

```bash
go build -o sub2api .
```

## Release

CI runs on pushes to `main` or `master` and on pull requests:

```text
go test ./...
go build -trimpath -o sub2api .
```

Publishing is tag-based. Create and push a version tag:

```bash
git tag v0.1.0
git push origin v0.1.0
```

The release workflow builds archives for macOS, Linux, and Windows, then publishes them to GitHub Releases with `checksums.txt`.

You can also run the release workflow manually from GitHub Actions with a tag such as `v0.1.0`.

## OAuth setup

Browser login receives Sub2API tokens through a local callback such as:

```text
http://127.0.0.1:<port>/callback
```

Configure the Sub2API OAuth frontend callback URL for the provider you use so it can redirect to the CLI local callback. The CLI starts a temporary local server, opens the provider flow, then stores returned credentials in the system Keychain.

Supported providers:

```text
github, google, oidc, linuxdo, wechat
```

## Usage

Login:

```bash
sub2api login --base-url https://sub2api.example.com --provider github
```

Import tokens from an already logged-in browser session:

```js
copy(JSON.stringify({
  auth_token: localStorage.getItem("auth_token"),
  refresh_token: localStorage.getItem("refresh_token"),
  token_expires_at: localStorage.getItem("token_expires_at")
}))
```

```bash
pbpaste | sub2api login token --base-url https://sub2api.example.com --provider oidc --timezone Asia/Shanghai
```

Render today's dashboard:

```bash
sub2api today
```

Show current user:

```bash
sub2api whoami
```

Logout:

```bash
sub2api logout
```

Profiles:

```bash
sub2api --profile work login --base-url https://sub2api.example.com --provider github --timezone Asia/Shanghai
sub2api --profile work today
```

## Data

The CLI uses the current-user dashboard endpoints:

```text
GET  /api/v1/usage/dashboard/stats
GET  /api/v1/usage/dashboard/trend
GET  /api/v1/usage/dashboard/models
POST /api/v1/auth/refresh
```

Config metadata is stored in:

```text
~/.config/sub2api-cli/config.toml
```

Tokens are stored in the system Keychain under service `sub2api-cli`.

## Common errors

`missing access_token`: the OAuth provider flow returned to a web frontend callback that did not forward tokens to the local CLI callback.

`OIDC login timed out`: the Sub2API instance kept the token in the web pending-auth flow. Use `sub2api login token` to import browser localStorage tokens, or configure a CLI-friendly OAuth callback on the server.

`profile not found`: run `sub2api login` for that profile.

`oauth login timed out`: finish the browser flow within two minutes, or retry `sub2api login`.
