package puppet

import (
	"errors"
	"testing"

	"gitea.obmondo.com/EnableIT/linuxaid-cli/internal/config"
	"gitea.obmondo.com/EnableIT/linuxaid-cli/internal/shell"
	"gitea.obmondo.com/EnableIT/linuxaid-cli/internal/shell/shelltest"
)

func TestOpensourceNodesRunCommandsWithoutWebtee(t *testing.T) {
	recorder := &shelltest.Recorder{}
	service := NewService(nil, recorder, config.Config{Certname: "node01.example", Opensource: true})

	if service.webtee != nil {
		t.Fatal("expected no webtee client on an opensource node")
	}

	if err := service.RunLogged("apt update"); err != nil {
		t.Fatalf("expected apt update to succeed, got %v", err)
	}

	if !recorder.Ran("apt update") {
		t.Errorf("expected apt update to run locally, ran %v", recorder.Commands())
	}
}

func TestCustomersGetAWebteeClient(t *testing.T) {
	service := NewService(nil, &shelltest.Recorder{}, config.Config{Certname: "node01.example"})

	if service.webtee == nil {
		t.Error("expected a webtee client for an Obmondo customer")
	}
}

func TestRunAgentOnOpensourceNodesReturnsPuppetsExitCode(t *testing.T) {
	const command = "puppet agent -t --noop --detailed-exitcodes"

	recorder := &shelltest.Recorder{Results: map[string]shell.Result{
		command: {ExitCode: ExitChanged, Err: errors.New("exit status 2")},
	}}
	service := NewService(nil, recorder, config.Config{Opensource: true})

	if exitCode := service.RunAgent(true, "noop", ""); exitCode != ExitChanged {
		t.Errorf("expected puppet's exit code %d, got %d", ExitChanged, exitCode)
	}

	if !recorder.Ran(command) {
		t.Errorf("expected the run to happen locally, ran %v", recorder.Commands())
	}
}
