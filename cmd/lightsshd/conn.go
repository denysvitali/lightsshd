package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/creack/pty"
	"github.com/gliderlabs/ssh"
	"github.com/sirupsen/logrus"
)

type ConnectionHandler struct {
	authorizedKeys string
	defaultCmd     []string
}

type ConnPK struct {
	pk      ssh.PublicKey
	comment string
}

func (h ConnectionHandler) Handle(session ssh.Session) {
	fullCmd := session.Command()
	if len(fullCmd) == 0 {
		fullCmd = h.defaultCmd
	}

	var args []string
	if len(fullCmd) > 1 {
		args = fullCmd[1:]
	}

	logger.WithFields(logrus.Fields{
		"user":        session.User(),
		"remote_addr": session.RemoteAddr(),
		"command":     fullCmd,
	}).Info("New SSH session")

	cmd := exec.Command(fullCmd[0], args...)
	ptyReq, _, _ := session.Pty()
	cmd.Env = append(cmd.Env, fmt.Sprintf("TERM=%s", ptyReq.Term))

	f, err := pty.Start(cmd)
	if err != nil {
		logger.WithError(err).Error("Failed to start PTY")
		return
	}
	defer f.Close()

	go func() {
		_, _ = io.Copy(f, session)
	}()

	_, _ = io.Copy(session, f)

	if err := cmd.Wait(); err != nil {
		logger.WithError(err).Debug("Command finished with error")
	}

	logger.WithFields(logrus.Fields{
		"user":        session.User(),
		"remote_addr": session.RemoteAddr(),
	}).Info("SSH session ended")
}

func (h ConnectionHandler) PKHandler(ctx ssh.Context, key ssh.PublicKey) bool {
	authorizedKeys, err := h.parseAuthorizedKeys()
	if err != nil {
		logger.WithError(err).Error("Unable to parse authorized keys")
		return false
	}

	for _, v := range authorizedKeys {
		if ssh.KeysEqual(v.pk, key) {
			logger.WithFields(logrus.Fields{
				"user":        ctx.User(),
				"remote_addr": ctx.RemoteAddr(),
				"key_comment": v.comment,
			}).Info("SSH authentication successful")
			return true
		}
	}

	logger.WithFields(logrus.Fields{
		"user":           ctx.User(),
		"remote_addr":    ctx.RemoteAddr(),
		"client_version": ctx.ClientVersion(),
	}).Warn("SSH authentication failed - public key not authorized")

	return false
}

func (h ConnectionHandler) parseAuthorizedKeys() ([]ConnPK, error) {
	var pks []ConnPK
	f, err := os.Open(h.authorizedKeys)
	if err != nil {
		return nil, fmt.Errorf("unable to open authorized keys file: %w", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Bytes()

		if len(line) == 0 || line[0] == '#' {
			continue
		}

		pk, comment, _, _, err := ssh.ParseAuthorizedKey(line)
		if err != nil {
			logger.WithFields(logrus.Fields{
				"line": lineNum,
				"file": h.authorizedKeys,
			}).WithError(err).Warn("Failed to parse authorized key line")
			continue
		}

		logger.WithFields(logrus.Fields{
			"comment": comment,
			"line":    lineNum,
		}).Debug("Loaded authorized key")

		pks = append(pks, ConnPK{
			pk:      pk,
			comment: comment,
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error scanning authorized keys file: %w", err)
	}

	logger.WithField("count", len(pks)).Info("Loaded authorized keys")
	return pks, nil
}