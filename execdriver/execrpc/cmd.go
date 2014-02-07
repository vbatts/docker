package execrpc

import (
	"fmt"
	"github.com/dotcloud/docker/execdriver"
	"github.com/dotcloud/docker/execdriver/lxc"
	"github.com/dotcloud/docker/utils"
	"io/ioutil"
	"net"
	"net/rpc"
	"os"
	"path"
	"syscall"
	"time"
)

// This is a copy of execdriver.Command, which unfortunately doesn't can't be marshalled
// since it contains public os.File members that have no exported fields
// This could be done nicer if we change Command to not inherit from exec.Cmd
type CommandWrapper struct {
	// From exec.Cmd:
	Path        string
	Args        []string
	Env         []string
	Dir         string
	SysProcAttr *syscall.SysProcAttr

	// From execdriver.Command:
	ID         string
	Privileged bool
	User       string
	Rootfs     string
	InitPath   string
	Entrypoint string
	Arguments  []string
	WorkingDir string
	ConfigPath string
	Tty        bool
	Network    *execdriver.Network
	Config     []string
	Resources  *execdriver.Resources
}

func (wrapper *CommandWrapper) Unwrap() *execdriver.Command {
	d := &execdriver.Command{
		// From execdriver.Command:
		ID:         wrapper.ID,
		Privileged: wrapper.Privileged,
		User:       wrapper.User,
		Rootfs:     wrapper.Rootfs,
		InitPath:   wrapper.InitPath,
		Entrypoint: wrapper.Entrypoint,
		Arguments:  wrapper.Arguments,
		WorkingDir: wrapper.WorkingDir,
		ConfigPath: wrapper.ConfigPath,
		Tty:        wrapper.Tty,
		Network:    wrapper.Network,
		Config:     wrapper.Config,
		Resources:  wrapper.Resources,
	}

	// From exec.Cmd:
	d.Path = wrapper.Path
	d.Args = wrapper.Args
	d.Env = wrapper.Env
	d.Dir = wrapper.Dir
	d.SysProcAttr = wrapper.SysProcAttr

	return d
}

func WrapCommand(cmd *execdriver.Command) *CommandWrapper {
	return &CommandWrapper{
		// From exec.Cmd:
		Path:        cmd.Path,
		Args:        cmd.Args,
		Env:         cmd.Env,
		Dir:         cmd.Dir,
		SysProcAttr: cmd.SysProcAttr,

		// From execdriver.Command:
		ID:         cmd.ID,
		Privileged: cmd.Privileged,
		User:       cmd.User,
		Rootfs:     cmd.Rootfs,
		InitPath:   cmd.InitPath,
		Entrypoint: cmd.Entrypoint,
		Arguments:  cmd.Arguments,
		WorkingDir: cmd.WorkingDir,
		ConfigPath: cmd.ConfigPath,
		Tty:        cmd.Tty,
		Network:    cmd.Network,
		Config:     cmd.Config,
		Resources:  cmd.Resources,
	}
}

type CmdDriver struct {
	Address    string
	stdin      bool
	listener   *net.UnixListener
	realDriver execdriver.Driver

	startedLock chan int
	exitedLock  chan int
	err         error // if exit code is -1
	exitCode    int
}

func NewCmdDriver(stdin bool) (*CmdDriver, error) {
	realDriver, err := lxc.NewDriver("/var/lib/docker", false)
	if err != nil {
		return nil, err
	}

	baseDir := "/var/run/docker-client"
	if err := os.MkdirAll(baseDir, 0600); err != nil {
		return nil, err
	}

	socketDir, err := ioutil.TempDir(baseDir, "cli")
	if err != nil {
		return nil, err
	}

	address := path.Join(socketDir, "socket")
	addr := &net.UnixAddr{Net: "unix", Name: address}
	listener, err := net.ListenUnix("unix", addr)
	if err != nil {
		return nil, err
	}

	d := &CmdDriver{
		Address:     address,
		stdin:       stdin,
		listener:    listener,
		realDriver:  realDriver,
		startedLock: make(chan int),
		exitedLock:  make(chan int),
	}

	if err := rpc.Register(d); err != nil {
		return nil, err
	}

	return d, nil
}

func Serve(d *CmdDriver) error {
	for {
		conn, err := d.listener.AcceptUnix()
		if err != nil {
			utils.Debugf("rpc socket accept error: %s", err)
			continue
		}

		rpc.ServeConn(conn)
		conn.Close()
	}

	return nil
}

func (d *CmdDriver) started() {
	close(d.startedLock)
}

func (d *CmdDriver) exited() {
	close(d.exitedLock)
}

func (d *CmdDriver) Run1(wrapper *CommandWrapper, res *int) error {
	cmd := wrapper.Unwrap()

	if d.stdin {
		cmd.Stdin = os.Stdin
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	go func() {
		d.exitCode, d.err = d.realDriver.Run(cmd, func(*execdriver.Command) {
			d.started()
		})
		d.exited()

		// Todo: better synchronization with returning exit value to daemon (from Run2)
		// We want to ensure that the daemon can get the return value from Run2, but not
		// block longer than that. For safety we also need to have a timeout and be able
		// to handle the daemon dying and the rpc connection closing)
		time.Sleep(time.Second)

		if d.err != nil {
			fmt.Fprintf(os.Stderr, "Can't start container: %s\n", d.err)
		}
		os.RemoveAll(d.Address)
		os.Exit(d.exitCode)
	}()

	// block on started or exited (error)
	select {
	case <-d.startedLock:
	case <-d.exitedLock:
	}

	*res = -1
	if cmd.Process != nil {
		*res = cmd.Process.Pid
	}

	return nil
}

func (d *CmdDriver) Run2(_ int, res *int) error {
	// block on exited
	<-d.exitedLock

	if d.err != nil {
		return d.err
	}

	*res = d.exitCode

	return nil
}
