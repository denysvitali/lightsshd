package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/charmbracelet/lipgloss"
	"github.com/gliderlabs/ssh"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

var logger = logrus.New()

type Config struct {
	LogLevel       string
	Address        string
	PidFile        string
	HostKeyFile    string
	AuthorizedKeys string
}

var config Config

var rootCmd = &cobra.Command{
	Use:   "lightsshd",
	Short: "A lightweight SSH daemon",
	Long: lipgloss.NewStyle().
		Foreground(lipgloss.Color("12")).
		Bold(true).
		Render("LightSSHD - A lightweight SSH daemon\n\nA simple, secure SSH server implementation in Go."),
	Version: fmt.Sprintf("%s (commit: %s, built: %s)", version, commit, date),
	Run:     runServer,
}

func init() {
	rootCmd.Flags().StringVarP(&config.LogLevel, "log-level", "l", "error", "Log level (trace, debug, info, warn, error)")
	rootCmd.Flags().StringVarP(&config.Address, "listen", "L", "0.0.0.0:2222", "Address to listen on")
	rootCmd.Flags().StringVarP(&config.PidFile, "pid-file", "P", "", "PID file path")
	rootCmd.Flags().StringVarP(&config.HostKeyFile, "host-key", "k", "/etc/lightsshd/ssh_host_ed25519_key", "Host key file path")
	rootCmd.Flags().StringVarP(&config.AuthorizedKeys, "authorized-keys", "a", "/etc/lightsshd/authorized_keys", "Authorized keys file path")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		logger.Fatalf("Failed to execute command: %v", err)
	}
}

func runServer(cmd *cobra.Command, args []string) {
	setupLogger()

	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("10")).
		Bold(true)

	logger.Info(style.Render("Starting LightSSHD server..."))

	if config.PidFile != "" {
		if err := writePidFile(); err != nil {
			logger.Fatalf("Unable to write PID file: %v", err)
		}
	}

	go signalHandler()

	connHandler := ConnectionHandler{
		authorizedKeys: config.AuthorizedKeys,
		defaultCmd:     []string{"/bin/bash"},
	}

	var opts []ssh.Option
	if config.HostKeyFile != "" {
		opts = append(opts, ssh.HostKeyFile(config.HostKeyFile))
		createHostKey(config.HostKeyFile)
	}
	opts = append(opts, ssh.PublicKeyAuth(connHandler.PKHandler))

	logger.WithFields(logrus.Fields{
		"address":         config.Address,
		"host_key":        config.HostKeyFile,
		"authorized_keys": config.AuthorizedKeys,
	}).Info("Server configuration")

	logger.Fatalf("Unable to listen: %v", ssh.ListenAndServe(config.Address, connHandler.Handle, opts...))
}

func setupLogger() {
	switch strings.ToLower(config.LogLevel) {
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

	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
		ForceColors:   true,
	})
}

func writePidFile() error {
	pidFile, err := os.Create(config.PidFile)
	if err != nil {
		return fmt.Errorf("unable to create PID file: %w", err)
	}
	defer pidFile.Close()

	_, err = pidFile.WriteString(fmt.Sprintf("%d", os.Getpid()))
	if err != nil {
		return fmt.Errorf("unable to write PID: %w", err)
	}

	return nil
}

func signalHandler() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM, os.Kill)

	s := <-c
	logger.WithField("signal", s).Info("Received signal, shutting down...")

	if config.PidFile != "" {
		if err := os.Remove(config.PidFile); err != nil {
			logger.WithError(err).Error("Unable to remove PID file")
		}
	}

	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("9")).
		Bold(true)

	logger.Info(style.Render("LightSSHD server stopped"))
	os.Exit(0)
}

func createHostKey(keyFile string) {
	if _, err := os.Stat(keyFile); err == nil {
		return
	} else if !os.IsNotExist(err) {
		logger.Fatalf("Unable to stat host key file: %v", err)
	}

	logger.WithField("keyFile", keyFile).Info("Generating new host key...")

	privateKeyFile, err := os.Create(keyFile)
	if err != nil {
		logger.Fatalf("Unable to create private key file: %v", err)
	}
	defer privateKeyFile.Close()

	publicKeyFile, err := os.Create(fmt.Sprintf("%s.pub", keyFile))
	if err != nil {
		logger.Fatalf("Unable to create public key file: %v", err)
	}
	defer publicKeyFile.Close()

	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		logger.Fatalf("Unable to generate Ed25519 key: %v", err)
	}

	privateKeyPKCS8, err := x509.MarshalPKCS8PrivateKey(private)
	if err != nil {
		logger.Fatalf("Unable to marshal private key to PKCS8: %v", err)
	}

	if err := pem.Encode(privateKeyFile, &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: privateKeyPKCS8,
	}); err != nil {
		logger.Fatalf("Unable to encode private key: %v", err)
	}

	if err := pem.Encode(publicKeyFile, &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: public,
	}); err != nil {
		logger.Fatalf("Unable to encode public key: %v", err)
	}

	logger.Info("Host key generated successfully")
}