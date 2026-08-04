package sshutil

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/cirruslabs/orchard/pkg/client"
	"golang.org/x/crypto/ssh"
)

func DialVM(
	ctx context.Context,
	c *client.Client,
	vmName string,
	port uint16,
	waitSeconds uint16,
	username string,
	password string,
) (*ssh.Client, error) {
	var sshClient *ssh.Client

	err := retry.Do(func() error {
		conn, err := c.VMs().PortForward(ctx, vmName, port, waitSeconds)
		if err != nil {
			return err
		}

		addr := fmt.Sprintf("%s:%d", vmName, port)
		sshConfig := &ssh.ClientConfig{
			User: username,
			Auth: []ssh.AuthMethod{
				ssh.Password(password),
			},
			HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
				return nil
			},
			Timeout: 30 * time.Second,
		}

		sshConn, chans, reqs, err := ssh.NewClientConn(conn, addr, sshConfig)
		if err != nil {
			_ = conn.Close()
			return err
		}

		sshClient = ssh.NewClient(sshConn, chans, reqs)
		// Keep the Orchard port-forward / SSH session alive during long TestFlight waits.
		go keepAlive(ctx, sshClient, 30*time.Second)
		return nil
	}, retry.Context(ctx), retry.Attempts(0), retry.Delay(time.Second), retry.DelayType(retry.FixedDelay))
	if err != nil {
		return nil, fmt.Errorf("SSH to VM %q failed: %w", vmName, err)
	}

	return sshClient, nil
}

func keepAlive(ctx context.Context, client *ssh.Client, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_, _, err := client.SendRequest("keepalive@openssh.com", true, nil)
			if err != nil {
				return
			}
		}
	}
}

func RunScript(sshClient *ssh.Client, scriptPath string, shell string) error {
	scriptFile, err := os.Open(scriptPath)
	if err != nil {
		return err
	}
	defer scriptFile.Close()

	session, err := sshClient.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	session.Stdin = scriptFile
	session.Stdout = os.Stdout
	session.Stderr = os.Stderr

	if shell != "" {
		if err := session.Start(shell); err != nil {
			return err
		}
	} else {
		if err := session.Shell(); err != nil {
			return err
		}
	}
	return session.Wait()
}

func PropagateSSHExitError(err error) {
	var exitErr *ssh.ExitError
	if !errors.As(err, &exitErr) {
		return
	}
	exitCodeFile, ok := os.LookupEnv("BUILD_EXIT_CODE_FILE")
	if !ok {
		return
	}
	if writeErr := os.WriteFile(exitCodeFile, fmt.Appendf(nil, "%d\n", exitErr.ExitStatus()), 0o644); writeErr != nil {
		log.Printf("failed to propagate SSH exit code to %q: %v", exitCodeFile, writeErr)
	}
}
