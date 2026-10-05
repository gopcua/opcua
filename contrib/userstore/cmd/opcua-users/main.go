// Copyright 2018-2026 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

// Command opcua-users manages a gopcua users.yaml file.
//
//	opcua-users [-f users.yaml] <command> [flags] [args]
//
//	add <user> [-role r]... [-password-stdin]   add a user (prompts for the password)
//	passwd <user> [-password-stdin]             change a password
//	del <user>                                  delete a user
//	disable <user> | enable <user>              block / unblock a login, keep the entry
//	roles <user> [r ...]                        replace the roles of a user
//	list                                        list users and certificates
//	cert add <cert.pem|cert.der> [-name n] [-role r]...
//	cert del <thumbprint>
//	cert disable <thumbprint> | cert enable <thumbprint>
//	check <user> [-password-stdin]              verify a password against the file
//
// The file path defaults to $OPCUA_USERS_FILE or ./users.yaml. Passwords are
// never accepted as command-line arguments: they are read from the terminal
// (twice, without echo) or, with -password-stdin, from the first line of
// standard input.
package main

import (
	"bufio"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"golang.org/x/crypto/bcrypt"
	"golang.org/x/term"

	"github.com/gopcua/opcua/contrib/userstore"
)

type roleFlags []string

func (r *roleFlags) String() string { return strings.Join(*r, ",") }
func (r *roleFlags) Set(v string) error {
	for _, s := range strings.Split(v, ",") {
		if s = strings.TrimSpace(s); s != "" {
			*r = append(*r, s)
		}
	}
	return nil
}

var (
	stdin  io.Reader = os.Stdin
	stdout io.Writer = os.Stdout
	stderr io.Writer = os.Stderr
	// readPassword reads a password without echo from the terminal.
	readPassword = func(prompt string) (string, error) {
		fmt.Fprint(stderr, prompt)
		b, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Fprintln(stderr)
		return string(b), err
	}
	isTerminal = func() bool { return term.IsTerminal(int(os.Stdin.Fd())) }
	bcryptCost = userstore.DefaultCost
)

const usage = `usage: opcua-users [-f users.yaml] <command> [flags] [args]

commands:
  add <user> [-role r]... [-password-stdin]   add a user (prompts for the password)
  passwd <user> [-password-stdin]             change a password
  del <user>                                  delete a user
  disable <user> | enable <user>              block / unblock a login, keep the entry
  roles <user> [role ...]                     replace the roles of a user
  list                                        list users and certificates
  cert add <cert.pem|cert.der> [-name n] [-role r]...
  cert del <thumbprint>
  cert disable <thumbprint> | cert enable <thumbprint>
  check <user> [-password-stdin]              verify a password against the file

The file defaults to $OPCUA_USERS_FILE or ./users.yaml.
`

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(stderr, "opcua-users:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	fs := flag.NewFlagSet("opcua-users", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() { fmt.Fprint(stderr, usage) }
	def := os.Getenv("OPCUA_USERS_FILE")
	if def == "" {
		def = "users.yaml"
	}
	path := fs.String("f", def, "users file")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() == 0 {
		fs.Usage()
		return errors.New("missing command")
	}
	cmd, rest := fs.Arg(0), fs.Args()[1:]

	switch cmd {
	case "add":
		return cmdAdd(*path, rest)
	case "passwd":
		return cmdPasswd(*path, rest)
	case "del", "delete", "rm":
		return cmdUser(*path, rest, "del", func(f *userstore.File, name string) error {
			delete(f.Users, name)
			return nil
		})
	case "disable", "enable":
		disabled := cmd == "disable"
		return cmdUser(*path, rest, cmd, func(f *userstore.File, name string) error {
			f.Users[name].Disabled = disabled
			return nil
		})
	case "roles":
		if len(rest) < 1 {
			return errors.New("usage: roles <user> [role ...]")
		}
		roles := rest[1:]
		return cmdUser(*path, rest[:1], "roles", func(f *userstore.File, name string) error {
			f.Users[name].Roles = roles
			return nil
		})
	case "list", "ls":
		return cmdList(*path)
	case "cert":
		return cmdCert(*path, rest)
	case "check":
		return cmdCheck(*path, rest)
	case "help", "-h", "--help":
		fs.Usage()
		return nil
	default:
		fs.Usage()
		return fmt.Errorf("unknown command %q", cmd)
	}
}

// parse parses flags that may appear before or after the positional args.
func parse(fs *flag.FlagSet, args []string) ([]string, error) {
	var pos []string
	for {
		if err := fs.Parse(args); err != nil {
			return nil, err
		}
		if fs.NArg() == 0 {
			return pos, nil
		}
		pos = append(pos, fs.Arg(0))
		args = fs.Args()[1:]
	}
}

func newPassword(fromStdin bool) (string, error) {
	if fromStdin || !isTerminal() {
		line, err := bufio.NewReader(stdin).ReadString('\n')
		if err != nil && !(errors.Is(err, io.EOF) && line != "") {
			return "", errors.New("no password on stdin")
		}
		return strings.TrimRight(line, "\r\n"), nil
	}
	p1, err := readPassword("New password: ")
	if err != nil {
		return "", err
	}
	p2, err := readPassword("Repeat password: ")
	if err != nil {
		return "", err
	}
	if p1 != p2 {
		return "", errors.New("passwords do not match")
	}
	return p1, nil
}

func cmdAdd(path string, args []string) error {
	fs := flag.NewFlagSet("add", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var roles roleFlags
	fs.Var(&roles, "role", "role, repeatable or comma separated")
	fromStdin := fs.Bool("password-stdin", false, "read the password from stdin")
	pos, err := parse(fs, args)
	if err != nil {
		return err
	}
	if len(pos) != 1 {
		return errors.New("usage: add <user> [-role r ...] [-password-stdin]")
	}
	name := pos[0]
	if err := userstore.ValidateUsername(name); err != nil {
		return err
	}
	// fail before prompting if the user exists
	if f, err := userstore.Load(path); err == nil && f.Users[name] != nil {
		return fmt.Errorf("user %q already exists, use passwd to change the password", name)
	}
	pass, err := newPassword(*fromStdin)
	if err != nil {
		return err
	}
	hash, err := userstore.HashPassword(pass, bcryptCost)
	if err != nil {
		return err
	}
	err = userstore.Update(path, func(f *userstore.File) error {
		if f.Users[name] != nil {
			return fmt.Errorf("user %q already exists", name)
		}
		f.Users[name] = &userstore.User{Password: hash, Roles: roles}
		return nil
	})
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "added user %q to %s\n", name, path)
	return nil
}

func cmdPasswd(path string, args []string) error {
	fs := flag.NewFlagSet("passwd", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fromStdin := fs.Bool("password-stdin", false, "read the password from stdin")
	pos, err := parse(fs, args)
	if err != nil {
		return err
	}
	if len(pos) != 1 {
		return errors.New("usage: passwd <user> [-password-stdin]")
	}
	name := pos[0]
	f, err := userstore.Load(path)
	if err != nil {
		return err
	}
	if f.Users[name] == nil {
		return fmt.Errorf("no such user %q", name)
	}
	pass, err := newPassword(*fromStdin)
	if err != nil {
		return err
	}
	hash, err := userstore.HashPassword(pass, bcryptCost)
	if err != nil {
		return err
	}
	err = userstore.Update(path, func(f *userstore.File) error {
		u := f.Users[name]
		if u == nil {
			return fmt.Errorf("no such user %q", name)
		}
		u.Password = hash
		return nil
	})
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "password of %q changed\n", name)
	return nil
}

func cmdUser(path string, args []string, verb string, fn func(*userstore.File, string) error) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: %s <user>", verb)
	}
	name := args[0]
	if _, err := os.Stat(path); err != nil {
		return err
	}
	err := userstore.Update(path, func(f *userstore.File) error {
		if f.Users[name] == nil {
			return fmt.Errorf("no such user %q", name)
		}
		return fn(f, name)
	})
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "%s: %s ok\n", name, verb)
	return nil
}

func cmdList(path string) error {
	f, err := userstore.Load(path)
	if err != nil {
		return err
	}
	w := tabwriter.NewWriter(stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "USER\tROLES\tSTATUS")
	for _, n := range f.Usernames() {
		u := f.Users[n]
		fmt.Fprintf(w, "%s\t%s\t%s\n", n, strings.Join(u.Roles, ","), status(u.Disabled))
	}
	if len(f.Certificates) > 0 {
		fmt.Fprintln(w, "\nCERTIFICATE\tNAME\tROLES\tSTATUS")
		for _, c := range f.Certificates {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", c.Thumbprint, c.Name, strings.Join(c.Roles, ","), status(c.Disabled))
		}
	}
	return w.Flush()
}

func status(disabled bool) string {
	if disabled {
		return "disabled"
	}
	return "active"
}

func cmdCheck(path string, args []string) error {
	fs := flag.NewFlagSet("check", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fromStdin := fs.Bool("password-stdin", false, "read the password from stdin")
	pos, err := parse(fs, args)
	if err != nil {
		return err
	}
	if len(pos) != 1 {
		return errors.New("usage: check <user> [-password-stdin]")
	}
	f, err := userstore.Load(path)
	if err != nil {
		return err
	}
	var pass string
	if *fromStdin || !isTerminal() {
		pass, err = newPassword(true)
	} else {
		pass, err = readPassword("Password: ")
	}
	if err != nil {
		return err
	}
	u := f.Users[pos[0]]
	if u == nil || bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(pass)) != nil {
		return errors.New("invalid username or password")
	}
	if u.Disabled {
		return errors.New("password ok, but the user is disabled")
	}
	fmt.Fprintln(stdout, "ok")
	return nil
}

func cmdCert(path string, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: cert add|del|disable|enable <args>")
	}
	switch sub, rest := args[0], args[1:]; sub {
	case "add":
		fs := flag.NewFlagSet("cert add", flag.ContinueOnError)
		fs.SetOutput(stderr)
		name := fs.String("name", "", "label, defaults to the certificate subject CN")
		var roles roleFlags
		fs.Var(&roles, "role", "role, repeatable or comma separated")
		pos, err := parse(fs, rest)
		if err != nil {
			return err
		}
		if len(pos) != 1 {
			return errors.New("usage: cert add <cert.pem|cert.der> [-name n] [-role r ...]")
		}
		cert, err := readCert(pos[0])
		if err != nil {
			return err
		}
		if *name == "" {
			*name = cert.Subject.CommonName
		}
		tp := userstore.Thumbprint(cert.Raw)
		err = userstore.Update(path, func(f *userstore.File) error {
			if f.Certificate(tp) != nil {
				return fmt.Errorf("certificate %s already trusted", tp)
			}
			f.Certificates = append(f.Certificates, &userstore.Certificate{Thumbprint: tp, Name: *name, Roles: roles})
			return nil
		})
		if err != nil {
			return err
		}
		fmt.Fprintf(stdout, "trusted certificate %s (%s), valid until %s\n", tp, cert.Subject, cert.NotAfter.Format("2006-01-02"))
		return nil
	case "del", "delete", "rm", "disable", "enable":
		if len(rest) != 1 {
			return fmt.Errorf("usage: cert %s <thumbprint>", sub)
		}
		if _, err := os.Stat(path); err != nil {
			return err
		}
		err := userstore.Update(path, func(f *userstore.File) error {
			c := f.Certificate(rest[0])
			if c == nil {
				return fmt.Errorf("no certificate %s", rest[0])
			}
			switch sub {
			case "disable", "enable":
				c.Disabled = sub == "disable"
			default:
				for i := range f.Certificates {
					if f.Certificates[i] == c {
						f.Certificates = append(f.Certificates[:i], f.Certificates[i+1:]...)
						break
					}
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
		fmt.Fprintf(stdout, "certificate %s: %s ok\n", rest[0], sub)
		return nil
	default:
		return fmt.Errorf("unknown cert command %q", sub)
	}
}

func readCert(file string) (*x509.Certificate, error) {
	b, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	if blk, _ := pem.Decode(b); blk != nil {
		if blk.Type != "CERTIFICATE" {
			return nil, fmt.Errorf("%s: PEM block is %q, want CERTIFICATE", file, blk.Type)
		}
		b = blk.Bytes
	}
	cert, err := x509.ParseCertificate(b)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", file, err)
	}
	return cert, nil
}
