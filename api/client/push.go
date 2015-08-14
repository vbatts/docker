package client

import (
	"fmt"
	"net/url"
	"strings"

	flag "github.com/docker/docker/pkg/mflag"
	"github.com/docker/docker/pkg/parsers"
	"github.com/docker/docker/registry"
)

func (cli *DockerCli) confirmPush() bool {
	const prompt = "Do you really want to push to public registry? [y/n]: "
	answer := ""
	fmt.Fprintln(cli.out, "")

	for answer != "n" && answer != "y" {
		fmt.Fprint(cli.out, prompt)
		answer = strings.ToLower(strings.TrimSpace(readInput(cli.in, cli.out)))
	}

	if answer == "n" {
		fmt.Fprintln(cli.out, "Nothing pushed.")
	}

	return answer == "y"
}

// CmdPush pushes an image or repository to the registry.
//
// Usage: docker push NAME[:TAG]
func (cli *DockerCli) CmdPush(args ...string) error {
	cmd := cli.Subcmd("push", "NAME[:TAG]", "Push an image or a repository to the registry", true)
	force := cmd.Bool([]string{"f", "-force"}, false, "Push to public registry without confirmation")
	cmd.Require(flag.Exact, 1)

	cmd.ParseFlags(args, true)

	name := cmd.Arg(0)

	remote, tag := parsers.ParseRepositoryTag(name)

	// Resolve the Repository name from fqn to RepositoryInfo
	repoInfo, err := registry.ParseRepositoryInfo(remote)
	if err != nil {
		return err
	}

	v := url.Values{}
	v.Set("tag", tag)
	if *force {
		v.Set("force", "1")
	}

	push := func() error {
		_, _, err = cli.clientRequestAttemptLogin("POST", "/images/"+remote+"/push?"+v.Encode(), nil, cli.out, repoInfo.Index, "push")
		return err
	}
	if err = push(); err != nil {
		if v.Get("force") != "1" && strings.Contains(err.Error(), "Status 403") {
			if !cli.confirmPush() {
				return nil
			}
			v.Set("force", "1")
			if err = push(); err == nil {
				return nil
			}
		}
	}
	return err
}
