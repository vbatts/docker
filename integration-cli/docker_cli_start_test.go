package main

import (
	"fmt"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// Regression test for https://github.com/docker/docker/issues/7843
func TestStartAttachReturnsOnError(t *testing.T) {
	defer deleteAllContainers()

	cmd(t, "run", "-d", "--name", "test", "busybox")
	cmd(t, "stop", "test")

	// Expect this to fail because the above container is stopped, this is what we want
	if _, err := runCommand(exec.Command(dockerBinary, "run", "-d", "--name", "test2", "--link", "test:test", "busybox")); err == nil {
		t.Fatal("Expected error but got none")
	}

	ch := make(chan struct{})
	go func() {
		// Attempt to start attached to the container that won't start
		// This should return an error immediately since the container can't be started
		if _, err := runCommand(exec.Command(dockerBinary, "start", "-a", "test2")); err == nil {
			t.Fatal("Expected error but got none")
		}
		close(ch)
	}()

	select {
	case <-ch:
	case <-time.After(time.Second):
		t.Fatalf("Attach did not exit properly")
	}

	logDone("start - error on start with attach exits")
}

// gh#8555: Exit code should be passed through when using start -a
func TestStartAttachCorrectExitCode(t *testing.T) {
	defer deleteAllContainers()

	runCmd := exec.Command(dockerBinary, "run", "-d", "busybox", "sh", "-c", "sleep 2; exit 1")
	out, _, _, err := runCommandWithStdoutStderr(runCmd)
	if err != nil {
		t.Fatalf("failed to run container: %v, output: %q", err, out)
	}

	out = stripTrailingCharacters(out)

	// make sure the container has exited before trying the "start -a"
	waitCmd := exec.Command(dockerBinary, "wait", out)
	if out, _, err = runCommandWithOutput(waitCmd); err != nil {
		t.Fatal(out, err)
	}

	startCmd := exec.Command(dockerBinary, "start", "-a", out)
	startOut, exitCode, err := runCommandWithOutput(startCmd)
	if err != nil && !strings.Contains("exit status 1", fmt.Sprintf("%s", err)) {
		t.Fatalf("start command failed unexpectedly with error: %v, output: %q", err, startOut)
	}
	if exitCode != 1 {
		t.Fatalf("start -a did not respond with proper exit code: expected 1, got %d", exitCode)
	}

	logDone("start - correct exit code returned with -a")
}
