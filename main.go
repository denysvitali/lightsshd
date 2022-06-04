package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"github.com/alexflint/go-arg"
	"github.com/gliderlabs/ssh"
	"github.com/sirupsen/logrus"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

var args struct {
	LogLevel       string `arg:"-l"`
	Address        string `arg:"-L" default:"0.0.0.0:2222"`
	PidFile        string `arg:"-P"`
	HostKeyFile    string `arg:"-k" default:"/etc/lightsshd/ssh_host_ed25519_key"`
	AuthorizedKeys string `arg:"-a" default:"/etc/lightsshd/authorized_keys"`
}

var logger = logrus.New()

func main() {
	arg.MustParse(&args)
	switch strings.ToLower(args.LogLevel) {
	case "trace":
		logger.SetLevel(logrus.TraceLevel)
	case "debug":
		logger.SetLevel(logrus.DebugLevel)
	case "info":
		logger.SetLevel(logrus.InfoLevel)
	case "warn":
		logger.SetLevel(logrus.WarnLevel)
	default:
		logger.SetLevel(logrus.ErrorLevel)
	}

	if args.PidFile != "" {
		pidFile, err := os.Create(args.PidFile)
		if err != nil {
			logger.Fatalf("unable to open pidfile: %v", err)
		}
		_, _ = pidFile.WriteString(fmt.Sprintf("%d", os.Getpid()))
	}
	go signalHandler()

	connHandler := ConnectionHandler{
		authorizedKeys: args.AuthorizedKeys,
		defaultCmd:     []string{"/bin/bash"},
	}
	var opts []ssh.Option
	if args.HostKeyFile != "" {
		opts = append(opts, ssh.HostKeyFile(args.HostKeyFile))
		createHostKey(args.HostKeyFile)
	}
	opts = append(opts, ssh.PublicKeyAuth(connHandler.PKHandler))
	logrus.Fatalf("unable to listen: %v", ssh.ListenAndServe(args.Address, connHandler.Handle, opts...))
}

func signalHandler() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	signal.Notify(c, syscall.SIGTERM)
	signal.Notify(c, os.Kill)

	s := <-c
	logger.Infof("Got signal: %v", s)

	if args.PidFile != "" {
		err := os.Remove(args.PidFile)
		if err != nil {
			logger.Fatalf("unable to remove pidfile")
			return
		}
	}
	os.Exit(0)
}

func createHostKey(keyFile string) {
	// Create key file if it doesn't exist
	_, err := os.Stat(keyFile)
	if err != nil {
		if !os.IsNotExist(err) {
			logger.Fatalf("unable to stat file: %v", err)
		}
		// File doesn't exist, let's create it
		privateKeyFile, err := os.Create(keyFile)
		if err != nil {
			logger.Fatalf("unable to open file: %v", err)
		}
		defer privateKeyFile.Close()

		publicKeyFile, err := os.Create(fmt.Sprintf("%s.pub", keyFile))
		if err != nil {
			logger.Fatalf("unable to open file: %v", err)
		}
		defer publicKeyFile.Close()

		// Generate Ed25519 Key
		public, private, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			logger.Fatalf("unable to create ed25519 key: %v", err)
		}

		privateKeyPKCS8, err := x509.MarshalPKCS8PrivateKey(private)
		if err != nil {
			logger.Fatalf("unable to marshal private key to PKCS8: %v", err)
		}

		// Encode PEM private
		privateKeyFile.Write(pem.EncodeToMemory(&pem.Block{
			Type:  "PRIVATE KEY",
			Bytes: privateKeyPKCS8,
		}))
		publicKeyFile.Write(pem.EncodeToMemory(&pem.Block{
			Type:  "PUBLIC KEY",
			Bytes: public,
		}))
	}
}
