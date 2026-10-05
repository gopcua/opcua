# userstore

A file-backed user store for the [gopcua](https://github.com/gopcua/opcua)
server: usernames with **bcrypt** password hashes and trusted **X509 user
certificates**, kept in `users.yaml` and managed with the `opcua-users`
command.

It is a separate Go module so the core `github.com/gopcua/opcua` module stays
free of the bcrypt and YAML dependencies.

## Adding a user

```bash
go install github.com/gopcua/opcua/contrib/userstore/cmd/opcua-users@latest
# or, in this repo: cd contrib/userstore && go build ./cmd/opcua-users

opcua-users -f users.yaml add alice -role operator    # prompts twice for the password
opcua-users -f users.yaml add viewer1 -role viewer
opcua-users -f users.yaml list
```

```
USER     ROLES     STATUS
alice    operator  active
viewer1  viewer    active
```

The file is created on the first `add`. A running server picks up changes
within 2 seconds, so you never need to restart it.

| Command | |
|---|---|
| `add <user> [-role r]... [-password-stdin]` | add a user |
| `passwd <user> [-password-stdin]` | change a password |
| `disable <user>` / `enable <user>` | block / unblock a login, keep the entry |
| `roles <user> [role ...]` | replace the roles |
| `del <user>` | delete a user |
| `list` | users and certificates (never shows hashes) |
| `check <user>` | verify a password against the file |
| `cert add <cert.pem\|cert.der> [-name n] [-role r]...` | trust a user certificate |
| `cert disable\|enable\|del <thumbprint>` | manage a certificate |

* The file defaults to `$OPCUA_USERS_FILE`, or `./users.yaml` if that's unset.
* Passwords are **never** taken as command-line arguments, because those end
  up in shell history and `ps`. They are typed at a hidden prompt, twice. For
  scripts and Docker, use `-password-stdin`, which reads the first line of
  standard input:

```bash
printf '%s\n' "$ALICE_PASSWORD" | opcua-users add alice -role operator -password-stdin
```

* Passwords must be 1–72 bytes. That's the bcrypt limit; longer passwords are
  rejected instead of being silently cut off.

## users.yaml

```yaml
# gopcua users file. Manage with: opcua-users -f <file> add|passwd|del|list ...
# Passwords are bcrypt hashes. Do not put plain text passwords here.
users:
    alice:
        password: $2a$12$0a5mJlbnb3l0K4vI4nYxB.2mJzB0yQwH5l2yXgJb5s8vEw3cS7m8G
        roles: [operator]
    viewer1:
        password: $2a$12$hR8bq9XWm8E9t4WlqJk2NuyQ6bq3o2j5mKfZ0c5bY2Gz6Y7c1aX3e
        roles: [viewer]
        disabled: true
certificates:
    - thumbprint: 3f2a9c0d6b1e8f7a5c4b3d2e1f0a9b8c7d6e5f4a
      name: line3-gateway
      roles: [operator]
```

* `password` must be a bcrypt hash (`$2a$`/`$2b$`/`$2y$`). A plain-text
  password makes the file invalid. You can also create hashes with
  `htpasswd -nbBC 12 user pw`.
* `thumbprint` is the SHA-1 of the DER certificate, the usual OPC UA
  identifier. Upper case and `:`-separated forms are accepted.
* Roles are plain strings. The store only keeps them and returns them from
  `Store.Roles`; enforcing them is up to the application.
* Unknown fields are rejected, so a typo like `pasword:` is caught.
* The command writes the file with mode `0600`, atomically (temp file +
  rename), and takes a `users.yaml.lock` file so concurrent edits can't
  overwrite each other. `Open` warns if the file is readable by other users.

## Server

```go
store, err := userstore.Open("users.yaml")
if err != nil {
	log.Fatal(err)
}
defer store.Close()

srv := server.New(
	server.EndPoint("0.0.0.0", 4840),
	server.Certificate(cert),
	server.PrivateKey(key),
	server.EnableSecurity("Basic256Sha256", ua.MessageSecurityModeSignAndEncrypt),
	server.UserNameAuth(store.UserNameAuthenticator()),
	server.X509Auth(store.X509Authenticator()), // optional: certificates from users.yaml
)
```

### Login behavior

* **Reload:** the file is checked every 2s (`Options.ReloadInterval`), and
  `Store.Reload()` reloads it immediately, e.g. from a SIGHUP handler. If the
  file is broken or missing, the last valid version stays active and an error
  is logged, so a typo can't lock everyone out.
* **Cache:** successful logins are cached in memory for 5 minutes
  (`CacheTTL`), so reconnecting clients don't pay the ~250 ms bcrypt cost each
  time. The cache key is an HMAC of user + password with a random key that
  only exists in memory. The cache is cleared on every reload, so password
  changes and `disable` apply immediately.
* **Throttling:** after 5 failed logins within 1 minute, a user is locked out
  for 1 minute. Another failure right after the lockout locks them again for
  twice as long, up to 15 minutes. While locked out, logins are rejected
  without running bcrypt. Settings: `MaxFailures`, `FailureWindow`,
  `LockoutDuration`. Unknown usernames are throttled too.
* **Timing:** an unknown username still goes through a bcrypt compare (against
  a dummy hash), so attackers can't tell which usernames exist by how fast the
  server answers.
* Every failure reaches the client as `Bad_UserAccessDenied`, so a lockout
  looks the same as a wrong password. The server log has the reason; passwords
  are never logged.
* **Certificates:** accepted if the thumbprint is listed, the entry isn't
  disabled, and the certificate is within its validity period. The server has
  already checked that the client holds the private key.
* **Roles:** `store.Roles(identity)` returns the roles of a username or of a
  listed certificate.

## Tests

```bash
cd contrib/userstore
go test -race ./...
```

* File format, validation, atomic save, concurrent `Update`, lock timeout.
* Authentication, cache invalidation and expiry, lockout and backoff, timing
  for unknown users, reload (including broken and deleted files), permissions
  warning, certificates.
* `opcua-users`: every command, stdin and terminal prompts, invalid input.
* `e2e_test.go`: a real gopcua client logging in to a server backed by
  `users.yaml` (Basic256Sha256/SignAndEncrypt). It adds and disables users
  while the server runs, and checks that a slow login doesn't block other
  clients.
