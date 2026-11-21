package userstore

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/gopcua/opcua/server"
	"github.com/gopcua/opcua/ua"
)

// low cost keeps the tests fast; production uses DefaultCost
const testCost = bcrypt.MinCost

func hash(t *testing.T, pw string) string {
	t.Helper()
	h, err := HashPassword(pw, testCost)
	require.NoError(t, err)
	return h
}

func writeUsers(t *testing.T, path string, f *File) {
	t.Helper()
	require.NoError(t, f.Save(path))
}

type clock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *clock) now() time.Time { c.mu.Lock(); defer c.mu.Unlock(); return c.t }
func (c *clock) add(d time.Duration) {
	c.mu.Lock()
	c.t = c.t.Add(d)
	c.mu.Unlock()
}

func newStore(t *testing.T, f *File, o Options) (*Store, string, *clock) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "users.yaml")
	writeUsers(t, path, f)
	c := &clock{t: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	if o.now == nil {
		o.now = c.now
	}
	if o.ReloadInterval == 0 {
		o.ReloadInterval = -1
	}
	if o.Logf == nil {
		o.Logf = t.Logf
	}
	s, err := Open(path, o)
	require.NoError(t, err)
	t.Cleanup(func() { s.Close() })
	return s, path, c
}

func TestHashPassword(t *testing.T) {
	h, err := HashPassword("pw", testCost)
	require.NoError(t, err)
	require.NoError(t, bcrypt.CompareHashAndPassword([]byte(h), []byte("pw")))

	_, err = HashPassword("", testCost)
	require.Error(t, err)
	_, err = HashPassword(strings.Repeat("x", MaxPasswordLength+1), testCost)
	require.ErrorContains(t, err, "longer than 72")
	_, err = HashPassword(strings.Repeat("x", MaxPasswordLength), testCost)
	require.NoError(t, err)

	h, err = HashPassword("pw", 0)
	require.NoError(t, err)
	cost, _ := bcrypt.Cost([]byte(h))
	require.Equal(t, DefaultCost, cost)
}

func TestParse(t *testing.T) {
	h := hash(t, "pw")
	tp := strings.Repeat("ab", 20)

	t.Run("valid", func(t *testing.T) {
		f, err := Parse([]byte(fmt.Sprintf(`
users:
  alice:
    password: %q
    roles: [operator, viewer]
  bob:
    password: %q
    disabled: true
certificates:
  - thumbprint: %q
    name: gw
    roles: [operator]
`, h, h, strings.ToUpper(strings.Join(splitPairs(tp), ":")))))
		require.NoError(t, err)
		require.Equal(t, []string{"alice", "bob"}, f.Usernames())
		require.Equal(t, []string{"operator", "viewer"}, f.Users["alice"].Roles)
		require.True(t, f.Users["bob"].Disabled)
		require.Equal(t, tp, f.Certificates[0].Thumbprint, "thumbprint normalized")
		require.NotNil(t, f.Certificate(strings.ToUpper(tp)))
	})
	t.Run("empty document", func(t *testing.T) {
		f, err := Parse(nil)
		require.NoError(t, err)
		require.Empty(t, f.Users)
		f, err = Parse([]byte("# only a comment\n"))
		require.NoError(t, err)
		require.Empty(t, f.Users)
	})

	bad := map[string]string{
		"plain text password": "users:\n  a:\n    password: secret\n",
		"unknown field":       fmt.Sprintf("users:\n  a:\n    password: %q\n    pasword: x\n", h),
		"empty user":          "users:\n  a:\n",
		"bad username":        fmt.Sprintf("users:\n  \" a\":\n    password: %q\n", h),
		"bad thumbprint":      "certificates:\n  - thumbprint: xyz\n",
		"duplicate thumbprint": fmt.Sprintf("certificates:\n  - thumbprint: %s\n  - thumbprint: %s\n",
			tp, strings.ToUpper(tp)),
		"not yaml": "users: [",
	}
	for name, doc := range bad {
		t.Run(name, func(t *testing.T) {
			_, err := Parse([]byte(doc))
			require.Error(t, err)
		})
	}
}

func splitPairs(s string) []string {
	var out []string
	for i := 0; i < len(s); i += 2 {
		out = append(out, s[i:i+2])
	}
	return out
}

func TestSaveAndUpdate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "users.yaml")
	h := hash(t, "pw")

	// Update creates the file
	require.NoError(t, Update(path, func(f *File) error {
		f.Users["alice"] = &User{Password: h, Roles: []string{"admin"}}
		return nil
	}))
	fi, err := os.Stat(path)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o600), fi.Mode().Perm())
	b, _ := os.ReadFile(path)
	require.True(t, strings.HasPrefix(string(b), "# gopcua users file"))
	require.NotContains(t, string(b), "pw\n")

	f, err := Load(path)
	require.NoError(t, err)
	require.Equal(t, h, f.Users["alice"].Password)

	// fn error: file unchanged, lock released
	require.Error(t, Update(path, func(f *File) error {
		delete(f.Users, "alice")
		return errors.New("abort")
	}))
	f, _ = Load(path)
	require.NotNil(t, f.Users["alice"])

	// invalid content is not written
	require.Error(t, Update(path, func(f *File) error {
		f.Users["bob"] = &User{Password: "plain"}
		return nil
	}))
	f, _ = Load(path)
	require.Nil(t, f.Users["bob"])

	// no temp files left behind
	ents, _ := os.ReadDir(filepath.Dir(path))
	require.Len(t, ents, 1)
}

func TestUpdateConcurrent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "users.yaml")
	h := hash(t, "pw")
	var wg sync.WaitGroup
	errs := make(chan error, 20)
	for i := range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- Update(path, func(f *File) error {
				f.Users[fmt.Sprintf("u%02d", i)] = &User{Password: h}
				return nil
			})
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}
	f, err := Load(path)
	require.NoError(t, err)
	require.Len(t, f.Users, 20, "no update lost")
}

func TestUpdateLockTimeout(t *testing.T) {
	path := filepath.Join(t.TempDir(), "users.yaml")
	require.NoError(t, os.WriteFile(path+".lock", []byte("1\n"), 0o600))
	old := lockTimeout
	lockTimeout = 100 * time.Millisecond
	defer func() { lockTimeout = old }()
	err := Update(path, func(*File) error { return nil })
	require.ErrorContains(t, err, "locked")
}

func TestAuthenticate(t *testing.T) {
	s, _, _ := newStore(t, &File{Users: map[string]*User{
		"alice": {Password: hash(t, "s3cret"), Roles: []string{"operator"}},
		"bob":   {Password: hash(t, "pw"), Disabled: true},
	}}, Options{})

	require.NoError(t, s.Authenticate("alice", "s3cret"))
	require.NoError(t, s.Authenticate("alice", "s3cret"), "cached")
	require.ErrorIs(t, s.Authenticate("alice", "wrong"), server.ErrAccessDenied)
	require.ErrorIs(t, s.Authenticate("alice", ""), server.ErrAccessDenied)
	require.ErrorIs(t, s.Authenticate("nobody", "s3cret"), server.ErrAccessDenied)
	require.ErrorIs(t, s.Authenticate("bob", "pw"), server.ErrAccessDenied, "disabled")
	require.ErrorIs(t, s.Authenticate("ALICE", "s3cret"), server.ErrAccessDenied, "case sensitive")

	require.Equal(t, []string{"operator"}, s.Roles(&server.Identity{TokenType: ua.UserTokenTypeUserName, UserName: "alice"}))
	require.Nil(t, s.Roles(&server.Identity{TokenType: ua.UserTokenTypeUserName, UserName: "bob"}))
	require.Nil(t, s.Roles(&server.Identity{TokenType: ua.UserTokenTypeAnonymous}))
	require.Nil(t, s.Roles(nil))

	u := s.User("alice")
	u.Roles[0] = "mutated"
	require.Equal(t, "operator", s.User("alice").Roles[0], "User returns a copy")
	require.Nil(t, s.User("nobody"))
}

func TestUnknownUserTiming(t *testing.T) {
	// unknown users must run a bcrypt compare too
	path := filepath.Join(t.TempDir(), "users.yaml")
	h, err := HashPassword("pw", DefaultCost)
	require.NoError(t, err)
	writeUsers(t, path, &File{Users: map[string]*User{"alice": {Password: h}}})
	s, err := Open(path, Options{ReloadInterval: -1, CacheTTL: -1, MaxFailures: -1, Logf: t.Logf})
	require.NoError(t, err)
	defer s.Close()

	measure := func(user string) time.Duration {
		start := time.Now()
		_ = s.Authenticate(user, "wrong")
		return time.Since(start)
	}
	known, unknown := measure("alice"), measure("nobody")
	t.Logf("known=%s unknown=%s", known, unknown)
	require.Greater(t, unknown, known/4, "unknown user returns too fast")
}

func TestCacheInvalidatedOnReload(t *testing.T) {
	s, path, _ := newStore(t, &File{Users: map[string]*User{
		"alice": {Password: hash(t, "old")},
	}}, Options{})
	require.NoError(t, s.Authenticate("alice", "old"))

	writeUsers(t, path, &File{Users: map[string]*User{"alice": {Password: hash(t, "new")}}})
	require.NoError(t, s.Reload())
	require.ErrorIs(t, s.Authenticate("alice", "old"), server.ErrAccessDenied, "old password still cached")
	require.NoError(t, s.Authenticate("alice", "new"))

	writeUsers(t, path, &File{Users: map[string]*User{"alice": {Password: hash(t, "new"), Disabled: true}}})
	require.NoError(t, s.Reload())
	require.Error(t, s.Authenticate("alice", "new"), "disabled user still cached")
}

func TestCacheExpiry(t *testing.T) {
	s, path, c := newStore(t, &File{Users: map[string]*User{
		"alice": {Password: hash(t, "pw")},
	}}, Options{CacheTTL: time.Minute})
	require.NoError(t, s.Authenticate("alice", "pw"))

	// swap the in-memory hash without clearing the cache: the cached login
	// keeps working until it expires
	writeUsers(t, path, &File{Users: map[string]*User{"alice": {Password: hash(t, "other")}}})
	f, err := Load(path)
	require.NoError(t, err)
	s.mu.Lock()
	s.file = f
	s.mu.Unlock()
	require.NoError(t, s.Authenticate("alice", "pw"))
	c.add(2 * time.Minute)
	require.Error(t, s.Authenticate("alice", "pw"))
}

func TestLockout(t *testing.T) {
	s, _, c := newStore(t, &File{Users: map[string]*User{
		"alice": {Password: hash(t, "pw")},
	}}, Options{MaxFailures: 3, FailureWindow: time.Minute, LockoutDuration: 10 * time.Second})

	for range 3 {
		require.ErrorIs(t, s.Authenticate("alice", "wrong"), server.ErrAccessDenied)
	}
	// locked: even the right password fails, without a bcrypt compare
	require.ErrorIs(t, s.Authenticate("alice", "pw"), ErrLockedOut)

	c.add(11 * time.Second)
	require.NoError(t, s.Authenticate("alice", "pw"), "lockout expired")

	// success reset the counter
	require.Error(t, s.Authenticate("alice", "wrong"))
	require.Error(t, s.Authenticate("alice", "wrong"))
	require.NoError(t, s.Authenticate("alice", "pw"))

	// failures outside the window don't add up
	require.Error(t, s.Authenticate("alice", "wrong"))
	require.Error(t, s.Authenticate("alice", "wrong"))
	c.add(2 * time.Minute)
	require.Error(t, s.Authenticate("alice", "wrong"))
	require.NoError(t, s.Authenticate("alice", "pw"))

	// lockout doubles when failing right after a lockout
	for range 3 {
		require.Error(t, s.Authenticate("alice", "wrong"))
	}
	c.add(11 * time.Second)
	require.ErrorIs(t, s.Authenticate("alice", "wrong"), server.ErrAccessDenied)
	require.ErrorIs(t, s.Authenticate("alice", "pw"), ErrLockedOut)
	c.add(15 * time.Second)
	require.ErrorIs(t, s.Authenticate("alice", "pw"), ErrLockedOut, "second lockout is 20s")
	c.add(6 * time.Second)
	require.NoError(t, s.Authenticate("alice", "pw"))

	// unknown users are throttled too
	for range 3 {
		require.Error(t, s.Authenticate("nobody", "x"))
	}
	require.ErrorIs(t, s.Authenticate("nobody", "x"), ErrLockedOut)
	// other users are not affected
	require.NoError(t, s.Authenticate("alice", "pw"))
}

func TestLockoutDisabled(t *testing.T) {
	s, _, _ := newStore(t, &File{Users: map[string]*User{
		"alice": {Password: hash(t, "pw")},
	}}, Options{MaxFailures: -1})
	for range 20 {
		require.Error(t, s.Authenticate("alice", "wrong"))
	}
	require.NoError(t, s.Authenticate("alice", "pw"))
}

func TestWatchReload(t *testing.T) {
	path := filepath.Join(t.TempDir(), "users.yaml")
	writeUsers(t, path, &File{Users: map[string]*User{"alice": {Password: hash(t, "pw")}}})

	var logMu sync.Mutex
	var logs []string
	s, err := Open(path, Options{ReloadInterval: 10 * time.Millisecond, Logf: func(f string, a ...any) {
		logMu.Lock()
		logs = append(logs, fmt.Sprintf(f, a...))
		logMu.Unlock()
	}})
	require.NoError(t, err)
	defer s.Close()
	require.Error(t, s.Authenticate("bob", "pw2"))

	// add a user: picked up without a restart
	require.NoError(t, Update(path, func(f *File) error {
		f.Users["bob"] = &User{Password: hash(t, "pw2")}
		return nil
	}))
	require.Eventually(t, func() bool { return s.User("bob") != nil }, 2*time.Second, 10*time.Millisecond)
	require.NoError(t, s.Authenticate("bob", "pw2"))

	// a broken file keeps the last good version
	require.NoError(t, os.WriteFile(path, []byte("users: [broken"), 0o600))
	require.Eventually(t, func() bool {
		logMu.Lock()
		defer logMu.Unlock()
		for _, l := range logs {
			if strings.Contains(l, "keeping the last loaded users") {
				return true
			}
		}
		return false
	}, 2*time.Second, 10*time.Millisecond)
	require.NoError(t, s.Authenticate("bob", "pw2"))

	// deleting the file keeps the users too
	require.NoError(t, os.Remove(path))
	time.Sleep(50 * time.Millisecond)
	require.NoError(t, s.Authenticate("alice", "pw"))
}

func TestOpenErrors(t *testing.T) {
	_, err := Open(filepath.Join(t.TempDir(), "missing.yaml"))
	require.ErrorIs(t, err, os.ErrNotExist)

	path := filepath.Join(t.TempDir(), "users.yaml")
	require.NoError(t, os.WriteFile(path, []byte("users:\n  a:\n    password: plain\n"), 0o600))
	_, err = Open(path)
	require.ErrorContains(t, err, "not a bcrypt hash")
}

func TestOpenWarnsOnPermissions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "users.yaml")
	writeUsers(t, path, &File{})
	require.NoError(t, os.Chmod(path, 0o644))
	var warned bool
	s, err := Open(path, Options{ReloadInterval: -1, Logf: func(f string, a ...any) {
		warned = warned || strings.Contains(fmt.Sprintf(f, a...), "chmod 600")
	}})
	require.NoError(t, err)
	s.Close()
	require.True(t, warned)
}

func genCert(t *testing.T, cn string, notBefore, notAfter time.Time) *x509.Certificate {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject:      pkix.Name{CommonName: cn},
		NotBefore:    notBefore,
		NotAfter:     notAfter,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	require.NoError(t, err)
	c, err := x509.ParseCertificate(der)
	require.NoError(t, err)
	return c
}

func TestAuthenticateCertificate(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	good := genCert(t, "gw", now.Add(-time.Hour), now.Add(time.Hour))
	off := genCert(t, "off", now.Add(-time.Hour), now.Add(time.Hour))
	expired := genCert(t, "old", now.Add(-2*time.Hour), now.Add(-time.Hour))
	unknown := genCert(t, "unknown", now.Add(-time.Hour), now.Add(time.Hour))

	s, _, _ := newStore(t, &File{Certificates: []*Certificate{
		{Thumbprint: Thumbprint(good.Raw), Name: "gw", Roles: []string{"operator"}},
		{Thumbprint: Thumbprint(off.Raw), Disabled: true},
		{Thumbprint: Thumbprint(expired.Raw)},
	}}, Options{})

	require.NoError(t, s.AuthenticateCertificate(good))
	require.ErrorIs(t, s.AuthenticateCertificate(off), server.ErrAccessDenied)
	require.ErrorIs(t, s.AuthenticateCertificate(expired), server.ErrAccessDenied)
	require.ErrorIs(t, s.AuthenticateCertificate(unknown), server.ErrAccessDenied)
	require.ErrorIs(t, s.AuthenticateCertificate(nil), server.ErrAccessDenied)

	require.Equal(t, []string{"operator"}, s.Roles(&server.Identity{TokenType: ua.UserTokenTypeCertificate, Certificate: good}))
	require.Nil(t, s.Roles(&server.Identity{TokenType: ua.UserTokenTypeCertificate, Certificate: off}))
	require.Nil(t, s.Roles(&server.Identity{TokenType: ua.UserTokenTypeCertificate, Certificate: unknown}))
}

func TestConcurrentAuthAndReload(t *testing.T) {
	s, path, _ := newStore(t, &File{Users: map[string]*User{
		"alice": {Password: hash(t, "pw")},
	}}, Options{MaxFailures: -1})
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 20 {
				_ = s.Authenticate("alice", "pw")
				_ = s.Authenticate("alice", "nope")
				_ = s.Roles(&server.Identity{UserName: "alice"})
			}
		}()
	}
	for range 10 {
		require.NoError(t, Update(path, func(*File) error { return nil }))
		require.NoError(t, s.Reload())
	}
	wg.Wait()
}
