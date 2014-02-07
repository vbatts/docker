package execrpc

import (
	"fmt"
	"github.com/dotcloud/docker/execdriver"
	"net"
	"net/rpc"
	"os"
)

const DriverName = "execrpc"

type driver struct {
	client *rpc.Client
}

func NewDriver(address string) (*driver, error) {
	var client *rpc.Client
	addr, err := net.ResolveUnixAddr("unix", address)
	if err != nil {
		return nil, err
	}
	socket, err := net.DialUnix("unix", nil, addr)
	if err != nil {
		return nil, err
	}

	client = rpc.NewClient(socket)

	return &driver{client: client}, nil
}

func (d *driver) call(method string, args, reply interface{}) error {
	if d.client == nil {
		return fmt.Errorf("no rpc connection to container")
	}

	if err := d.client.Call("CmdDriver."+method, args, reply); err != nil {
		return fmt.Errorf("dockerinit rpc call %s failed: %s", method, err)
	}
	return nil
}

func (d *driver) Name() string {
	return fmt.Sprintf("%s", DriverName)
}

func (d *driver) Run(c *execdriver.Command, startCallback execdriver.StartCallback) (int, error) {
	wrapper := WrapCommand(c)
	var pid, exitCode, dummy int
	err := d.call("Run1", wrapper, &pid)
	if pid != -1 {
		c.Process, _ = os.FindProcess(pid)
		if startCallback != nil {
			startCallback(c)
		}
	}
	err = d.call("Run2", dummy, &exitCode)
	return exitCode, err
}

func (d *driver) Kill(c *execdriver.Command, sig int) error {
	return fmt.Errorf("Not supported")
}

func (d *driver) Restore(c *execdriver.Command) error {
	return fmt.Errorf("Not supported")
}

type info struct {
	ID     string
	driver *driver
}

func (i *info) IsRunning() bool {
	return true
}

func (d *driver) Info(id string) execdriver.Info {
	return &info{
		ID:     id,
		driver: d,
	}
}

func (d *driver) GetPidsForContainer(id string) ([]int, error) {
	return nil, fmt.Errorf("Not supported")
}
