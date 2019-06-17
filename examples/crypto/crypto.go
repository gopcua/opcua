// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package main

import (
	"bufio"
	"context"
	"crypto/rsa"
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"syscall"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/debug"
	"github.com/gopcua/opcua/ua"

	"golang.org/x/crypto/ssh/terminal"
)

var (
	endpoint = flag.String("endpoint", "opc.tcp://localhost:4840", "OPC UA Endpoint URL")
	certfile = flag.String("cert", "cert.pem", "Path to certificate file")
	keyfile  = flag.String("key", "key.pem", "Path to PEM Private Key file")
	gencert  = flag.Bool("gen-cert", false, "Generate a new certificate")
	policy   = flag.String("sec-policy", "auto", "Security Policy URL or one of None, Basic128Rsa15, Basic256, Basic256Sha256")
	mode     = flag.String("sec-mode", "auto", "Security Mode: one of None, Sign, SignAndEncrypt")
	auth     = flag.String("auth-mode", "Anonymous", "Authentication Mode: one of Anonymous, UserName, Certificate")
	appuri   = flag.String("app-uri", "urn:gopcua:client", "Application URI")
	list     = flag.Bool("list", false, "List the policies supported by the endpoint and exit")
	username = flag.String("user", "", "Username to use in auth-mode UserName; will prompt for input if omitted")
	password = flag.String("pass", "", "Password to use in auth-mode UserName; will prompt for input if omitted")
)

func main() {
	flag.BoolVar(&debug.Enable, "debug", false, "enable debug logging")
	flag.Parse()
	log.SetFlags(0)

	ctx := context.Background()

	// Get a list of the endpoints for our target server
	endpoints, err := opcua.GetEndpoints(*endpoint)
	if err != nil {
		log.Fatal(err)
	}

	// User asked for just the list of options: print and quit
	if *list {
		printEndpointOptions(endpoints)
		return
	}

	// Get the options to pass into the client based on the flags passed into the executable
	opts := clientOptsFromFlags(endpoints)

	// Create a Client with the selected options
	c := opcua.NewClient(*endpoint, opts...)
	if err := c.Connect(ctx); err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	// Use our connection (read the server's time)
	v, err := c.Node(ua.NewNumericNodeID(0, 2258)).Value()
	if err != nil {
		log.Fatal(err)
	}
	if v != nil {
		fmt.Printf("Server's Time | Conn 1 %s | ", v.Value)
	} else {
		log.Print("v == nil")
	}

	// Detach our session and try re-establish it on a different secure channel
	s, err := c.DetachSession()
	if err != nil {
		log.Fatalf("Error detaching session: %s", err)
	}

	d := opcua.NewClient(*endpoint, opts...)

	// Create a channel only and do not activate it automatically
	d.Dial(ctx)
	defer d.Close()

	// Activate the previous session on the new channel
	err = d.ActivateSession(s)
	if err != nil {
		log.Fatalf("Error reactivating session: %s", err)
	}

	// Read the time again to prove our session is still OK
	v, err = d.Node(ua.NewNumericNodeID(0, 2258)).Value()
	if err != nil {
		log.Fatal(err)
	}
	if v != nil {
		fmt.Printf("Conn 2: %s\n", v.Value)
	} else {
		log.Print("v == nil")
	}
}

func clientOptsFromFlags(endpoints []*ua.EndpointDescription) []opcua.Option {
	opts := []opcua.Option{}

	// ApplicationURI is automatically read from the cert so is not required if a cert if provided
	if *certfile == "" && !*gencert {
		opts = append(opts, opcua.ApplicationURI(*appuri))
	}

	var cert []byte
	if *gencert || (*certfile != "" && *keyfile != "") {
		if *gencert {
			generate_cert(*appuri, 2048, *certfile, *keyfile)
		}
		debug.Printf("Loading cert/key from %s/%s", *certfile, *keyfile)
		c, err := tls.LoadX509KeyPair(*certfile, *keyfile)
		if err != nil {
			log.Printf("Failed to load certificate: %s", err)
		} else {
			pk, ok := c.PrivateKey.(*rsa.PrivateKey)
			if !ok {
				log.Fatalf("Invalid private key")
			}
			cert = c.Certificate[0]
			opts = append(opts, opcua.PrivateKey(pk), opcua.Certificate(cert))
		}
	}

	var secPolicy string
	switch {
	case *policy == "auto":
		// set it later
	case strings.HasPrefix(*policy, ua.SecurityPolicyURIPrefix):
		secPolicy = *policy
		*policy = ""
	case *policy == "None" || *policy == "Basic128Rsa15" || *policy == "Basic256" || *policy == "Basic256Sha256" || *policy == "Aes128_Sha256_RsaOaep" || *policy == "Aes256_Sha256_RsaPss":
		secPolicy = ua.SecurityPolicyURIPrefix + *policy
		*policy = ""
	default:
		log.Fatalf("Invalid security policy: %s", *policy)
	}

	// Select the most appropriate authentication mode from server capabilities and user input
	authMode, authOption := authFromFlags(cert)
	opts = append(opts, authOption)

	var secMode ua.MessageSecurityMode
	switch strings.ToLower(*mode) {
	case "auto":
	case "none":
		secMode = ua.MessageSecurityModeNone
		*mode = ""
	case "sign":
		secMode = ua.MessageSecurityModeSign
		*mode = ""
	case "signandencrypt":
		secMode = ua.MessageSecurityModeSignAndEncrypt
		*mode = ""
	default:
		log.Fatalf("Invalid security mode: %s", *mode)
	}

	// Allow input of only one of sec-mode,sec-policy when choosing 'None'
	if secMode == ua.MessageSecurityModeNone || secPolicy == ua.SecurityPolicyURINone {
		secMode = ua.MessageSecurityModeNone
		secPolicy = ua.SecurityPolicyURINone
	}

	// Find the best endpoint based on our input and server recommendation (highest SecurityMode+SecurityLevel)
	var serverEndpoint *ua.EndpointDescription
	switch {
	case *mode == "auto" && *policy == "auto": // No user selection, choose best
		for _, e := range endpoints {
			if serverEndpoint == nil || (e.SecurityMode >= serverEndpoint.SecurityMode && e.SecurityLevel >= serverEndpoint.SecurityLevel) {
				serverEndpoint = e
			}
		}

	case *mode != "auto" && *policy == "auto": // User only cares about mode, select highest securitylevel with that mode
		for _, e := range endpoints {
			if e.SecurityMode == secMode && (serverEndpoint == nil || e.SecurityLevel >= serverEndpoint.SecurityLevel) {
				serverEndpoint = e
			}
		}

	case *mode == "auto" && *policy != "auto": // User only cares about policy, select highest securitylevel with that policy
		for _, e := range endpoints {
			if e.SecurityPolicyURI == secPolicy && (serverEndpoint == nil || e.SecurityLevel >= serverEndpoint.SecurityLevel) {
				serverEndpoint = e
			}
		}

	default: // User cares about both
		fmt.Println("secMode: ", secMode, "secPolicy:", secPolicy)
		for _, e := range endpoints {
			if e.SecurityPolicyURI == secPolicy && e.SecurityMode == secMode && (serverEndpoint == nil || e.SecurityLevel >= serverEndpoint.SecurityLevel) {
				serverEndpoint = e
			}
		}
	}

	if serverEndpoint == nil { // Didn't find an endpoint with matching policy and mode.
		log.Printf("unable to find suitable server endpoint with selected sec-policy and sec-mode")
		printEndpointOptions(endpoints)
		log.Fatalf("quitting")
	}

	secPolicy = serverEndpoint.SecurityPolicyURI
	secMode = serverEndpoint.SecurityMode

	// Check that the selected endpoint is a valid combo
	err := validateEndpointConfig(endpoints, secPolicy, secMode, authMode)
	if err != nil {
		log.Fatalf("error validating input: %s", err)
	}

	opts = append(opts, opcua.SecurityFromEndpoint(serverEndpoint, authMode))

	log.Printf("Using config:\nEndpoint: %s\nSecurity mode: %s, %s\nAuth mode : %s\n", serverEndpoint.EndpointURL, serverEndpoint.SecurityPolicyURI, serverEndpoint.SecurityMode, authMode)
	return opts
}

func authFromFlags(cert []byte) (ua.UserTokenType, opcua.Option) {
	var err error

	var authMode ua.UserTokenType
	var authOption opcua.Option
	switch strings.ToLower(*auth) {
	case "anonymous":
		authMode = ua.UserTokenTypeAnonymous
		authOption = opcua.AuthAnonymous()

	case "username":
		authMode = ua.UserTokenTypeUserName

		if *username == "" {
			fmt.Print("Enter username: ")
			*username, err = bufio.NewReader(os.Stdin).ReadString('\n')
			*username = strings.TrimSuffix(*username, "\n")
			if err != nil {
				log.Fatalf("error reading username input: %s", err)
			}
		}

		passPrompt := true
		flag.Visit(func(f *flag.Flag) {
			if f.Name == "pass" {
				passPrompt = false
			}
		})

		if passPrompt {
			fmt.Print("Enter password: ")
			passInput, err := terminal.ReadPassword(int(syscall.Stdin))
			if err != nil {
				log.Fatalf("Error reading password: %s", err)
			}
			*password = string(passInput)
			fmt.Print("\n")
		}
		authOption = opcua.AuthUsername(*username, *password)

	case "certificate":
		authMode = ua.UserTokenTypeCertificate
		authOption = opcua.AuthCertificate(cert)

	case "issuedtoken":
		// todo: this is unsupported, fail here or fail in the opcua package?
		authMode = ua.UserTokenTypeIssuedToken
		authOption = opcua.AuthIssuedToken([]byte(nil))

	default:
		log.Printf("unknown auth-mode, defaulting to Anonymous")
		authMode = ua.UserTokenTypeAnonymous
		authOption = opcua.AuthAnonymous()

	}

	return authMode, authOption
}

func validateEndpointConfig(endpoints []*ua.EndpointDescription, secPolicy string, secMode ua.MessageSecurityMode, authMode ua.UserTokenType) error {
	for _, e := range endpoints {
		if e.SecurityMode == secMode && e.SecurityPolicyURI == secPolicy {
			for _, t := range e.UserIdentityTokens {
				if t.TokenType == authMode {
					return nil
				}
			}
		}
	}

	err := fmt.Errorf("server does not support an endpoint with security : %s , %s", secPolicy, secMode)
	printEndpointOptions(endpoints)
	return err
}

func printEndpointOptions(endpoints []*ua.EndpointDescription) {
	log.Print("Valid options for the endpoint are:")
	log.Print("         sec-policy    |    sec-mode     |      auth-modes\n")
	log.Print("-----------------------|-----------------|---------------------------\n")
	for _, e := range endpoints {
		p := strings.TrimPrefix(e.SecurityPolicyURI, "http://opcfoundation.org/UA/SecurityPolicy#")
		m := strings.TrimPrefix(e.SecurityMode.String(), "MessageSecurityMode")
		var tt []string
		for _, t := range e.UserIdentityTokens {
			tok := strings.TrimPrefix(t.TokenType.String(), "UserTokenType")

			// Just show one entry if a server has multiple varieties of one TokenType (eg. different algorithms)
			dup := false
			for _, v := range tt {
				if tok == v {
					dup = true
					break
				}
			}
			if !dup {
				tt = append(tt, tok)
			}
		}
		log.Printf("%22s | %-15s | (%s)", p, m, strings.Join(tt, ","))
	}
}
