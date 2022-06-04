package main

import (
	"bufio"
	"fmt"
	"github.com/creack/pty"
	"github.com/gliderlabs/ssh"
	"io"
	"os"
	"os/exec"
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

	cmd := exec.Command(fullCmd[0], args...)
	ptyReq, _, _ := session.Pty()
	cmd.Env = append(cmd.Env, fmt.Sprintf("TERM=%s", ptyReq.Term))
	f, err := pty.Start(cmd)
	if err != nil {
		panic(err)
	}
	go func() {
		_, _ = io.Copy(f, session) // stdin
	}()
	_, _ = io.Copy(session, f) // stdout
	_ = cmd.Wait()
}

func (h ConnectionHandler) PKHandler(ctx ssh.Context, key ssh.PublicKey) bool {
	authorizedKeys, err := h.parseAuthorizedKeys()
	if err != nil {
		logger.Errorf("unable to parse authorized_keys: %v", err)
		return false
	}
	for _, v := range authorizedKeys {
		if ssh.KeysEqual(v.pk, key) {
			logger.Infof("PK success from key=%s", v.comment)
			return true
		}
	}
	logger.Errorf("permission denied (publickey) - user=%s, remote_addr=%s, client=%s",
		ctx.User(),
		ctx.RemoteAddr(),
		ctx.ClientVersion(),
	)
	return false
}

func (h ConnectionHandler) parseAuthorizedKeys() ([]ConnPK, error) {
	var pks []ConnPK
	f, err := os.Open(h.authorizedKeys)
	if err != nil {
		return nil, fmt.Errorf("unable to open authorized_keys file: %v", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := scanner.Bytes()
		pk, comment, options, rest, err := ssh.ParseAuthorizedKey(line)
		if err != nil {
			return nil, fmt.Errorf("unable to parse authorization key: %v", err)
		}

		logger.Infof("adding pk (c=%s, opt=%s, rest=%s)", comment, options, rest)
		pks = append(pks, ConnPK{
			pk:      pk,
			comment: comment,
		})
	}

	if scanner.Err() != nil {
		return nil, fmt.Errorf("error while scanning file: %v", err)
	}

	return pks, nil
}
