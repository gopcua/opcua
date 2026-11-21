package main

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/gopcua/opcua/contrib/userstore"
)

// cli runs the command with the given stdin and returns stdout and stderr.
func cli(t *testing.T, in string, args ...string) (string, string, error) {
	t.Helper()
	var out, errOut bytes.Buffer
	stdin, stdout, stderr = strings.NewReader(in), &out, &errOut
	isTerminal = func() bool { return false }
	bcryptCost = bcrypt.MinCost
	err := run(args)
	return out.String(), errOut.String(), err
}

func TestAddPasswdDel(t *testing.T) {
	f := filepath.Join(t.TempDir(), "users.yaml")

	out, _, err := cli(t, "s3cret\n", "-f", f, "add", "alice", "-role", "operator", "--role=viewer,admin", "-password-stdin")
	require.NoError(t, err)
	require.Contains(t, out, `added user "alice"`)

	b, err := os.ReadFile(f)
	require.NoError(t, err)
	require.NotContains(t, string(b), "s3cret", "plain text password written to file")
	fi, _ := os.Stat(f)
	require.Equal(t, os.FileMode(0o600), fi.Mode().Perm())

	uf, err := userstore.Load(f)
	require.NoError(t, err)
	require.Equal(t, []string{"operator", "viewer", "admin"}, uf.Users["alice"].Roles)
	require.NoError(t, bcrypt.CompareHashAndPassword([]byte(uf.Users["alice"].Password), []byte("s3cret")))

	_, _, err = cli(t, "x\n", "-f", f, "add", "alice", "-password-stdin")
	require.ErrorContains(t, err, "already exists")

	// flags before the user name, password without trailing newline
	_, _, err = cli(t, "pw2", "-f", f, "add", "-password-stdin", "bob")
	require.NoError(t, err)

	out, _, err = cli(t, "s3cret\n", "-f", f, "check", "alice")
	require.NoError(t, err)
	require.Equal(t, "ok\n", out)
	_, _, err = cli(t, "wrong\n", "-f", f, "check", "alice")
	require.ErrorContains(t, err, "invalid username or password")

	_, _, err = cli(t, "n3w\n", "-f", f, "passwd", "alice", "-password-stdin")
	require.NoError(t, err)
	_, _, err = cli(t, "s3cret\n", "-f", f, "check", "alice")
	require.Error(t, err)
	_, _, err = cli(t, "n3w\n", "-f", f, "check", "alice")
	require.NoError(t, err)
	_, _, err = cli(t, "x\n", "-f", f, "passwd", "nobody", "-password-stdin")
	require.ErrorContains(t, err, "no such user")

	_, _, err = cli(t, "", "-f", f, "disable", "alice")
	require.NoError(t, err)
	_, _, err = cli(t, "n3w\n", "-f", f, "check", "alice")
	require.ErrorContains(t, err, "disabled")
	_, _, err = cli(t, "", "-f", f, "enable", "alice")
	require.NoError(t, err)

	_, _, err = cli(t, "", "-f", f, "roles", "bob", "viewer")
	require.NoError(t, err)

	out, _, err = cli(t, "", "-f", f, "list")
	require.NoError(t, err)
	require.Regexp(t, `alice\s+operator,viewer,admin\s+active`, out)
	require.Regexp(t, `bob\s+viewer\s+active`, out)
	require.NotContains(t, out, "$2a$", "hashes must not be listed")

	_, _, err = cli(t, "", "-f", f, "del", "bob")
	require.NoError(t, err)
	_, _, err = cli(t, "", "-f", f, "del", "bob")
	require.ErrorContains(t, err, "no such user")
	uf, _ = userstore.Load(f)
	require.Equal(t, []string{"alice"}, uf.Usernames())
}

func TestPasswordValidation(t *testing.T) {
	f := filepath.Join(t.TempDir(), "users.yaml")
	_, _, err := cli(t, "", "-f", f, "add", "alice", "-password-stdin")
	require.ErrorContains(t, err, "no password")
	_, _, err = cli(t, "\n", "-f", f, "add", "alice", "-password-stdin")
	require.ErrorContains(t, err, "must not be empty")
	_, _, err = cli(t, strings.Repeat("x", 73)+"\n", "-f", f, "add", "alice", "-password-stdin")
	require.ErrorContains(t, err, "longer than 72")
	_, _, err = cli(t, "pw\n", "-f", f, "add", " alice", "-password-stdin")
	require.ErrorContains(t, err, "whitespace")
	_, err = os.Stat(f)
	require.ErrorIs(t, err, os.ErrNotExist, "nothing written on failure")
}

func TestTerminalPrompt(t *testing.T) {
	f := filepath.Join(t.TempDir(), "users.yaml")
	var prompts []string
	answers := []string{"pw1", "pw2"}
	setup := func() {
		isTerminal = func() bool { return true }
		readPassword = func(p string) (string, error) {
			prompts = append(prompts, p)
			a := answers[0]
			answers = answers[1:]
			return a, nil
		}
	}
	var out, errOut bytes.Buffer
	stdout, stderr = &out, &errOut
	bcryptCost = bcrypt.MinCost
	setup()
	require.ErrorContains(t, run([]string{"-f", f, "add", "alice"}), "do not match")
	require.Equal(t, []string{"New password: ", "Repeat password: "}, prompts)

	answers = []string{"same", "same"}
	require.NoError(t, run([]string{"-f", f, "add", "alice"}))
	uf, err := userstore.Load(f)
	require.NoError(t, err)
	require.NoError(t, bcrypt.CompareHashAndPassword([]byte(uf.Users["alice"].Password), []byte("same")))
}

func TestEnvFile(t *testing.T) {
	f := filepath.Join(t.TempDir(), "env-users.yaml")
	t.Setenv("OPCUA_USERS_FILE", f)
	_, _, err := cli(t, "pw\n", "add", "alice")
	require.NoError(t, err)
	_, err = os.Stat(f)
	require.NoError(t, err)
}

func TestCert(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "users.yaml")

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "line3-gateway"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	require.NoError(t, err)
	pemFile := filepath.Join(dir, "gw.pem")
	require.NoError(t, os.WriteFile(pemFile, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o600))
	derFile := filepath.Join(dir, "gw.der")
	require.NoError(t, os.WriteFile(derFile, der, 0o600))
	keyFile := filepath.Join(dir, "gw.key")
	require.NoError(t, os.WriteFile(keyFile, pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}), 0o600))
	tp := userstore.Thumbprint(der)

	out, _, err := cli(t, "", "-f", f, "cert", "add", pemFile, "-role", "operator")
	require.NoError(t, err)
	require.Contains(t, out, tp)
	_, _, err = cli(t, "", "-f", f, "cert", "add", derFile)
	require.ErrorContains(t, err, "already trusted")
	_, _, err = cli(t, "", "-f", f, "cert", "add", keyFile)
	require.ErrorContains(t, err, "want CERTIFICATE")

	uf, err := userstore.Load(f)
	require.NoError(t, err)
	c := uf.Certificate(tp)
	require.NotNil(t, c)
	require.Equal(t, "line3-gateway", c.Name)
	require.Equal(t, []string{"operator"}, c.Roles)

	_, _, err = cli(t, "", "-f", f, "cert", "disable", strings.ToUpper(tp))
	require.NoError(t, err)
	uf, _ = userstore.Load(f)
	require.True(t, uf.Certificate(tp).Disabled)

	out, _, err = cli(t, "", "-f", f, "list")
	require.NoError(t, err)
	require.Regexp(t, tp+`\s+line3-gateway\s+operator\s+disabled`, out)

	_, _, err = cli(t, "", "-f", f, "cert", "del", tp)
	require.NoError(t, err)
	uf, _ = userstore.Load(f)
	require.Empty(t, uf.Certificates)
}

func TestUsage(t *testing.T) {
	_, errOut, err := cli(t, "")
	require.Error(t, err)
	require.Contains(t, errOut, "usage: opcua-users")
	_, _, err = cli(t, "", "bogus")
	require.ErrorContains(t, err, "unknown command")
	_, _, err = cli(t, "", "-f", filepath.Join(t.TempDir(), "none.yaml"), "del", "x")
	require.ErrorIs(t, err, os.ErrNotExist)
}
