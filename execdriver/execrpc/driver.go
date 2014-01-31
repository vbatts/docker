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
	address string
}

func NewDriver(address string) *driver {
	return &driver{address: address}
}

func (d *driver) getClient() (*rpc.Client, error) {
	addr, err := net.ResolveUnixAddr("unix", d.address)
	if err != nil {
		return nil, err
	}
	socket, err := net.DialUnix("unix", nil, addr)
	if err != nil {
		return nil, err
	}

	return rpc.NewClient(socket), nil
}

func (d *driver) Name() string {
	return fmt.Sprintf("%s", DriverName)
}

func (d *driver) Run(c *execdriver.Command, startCallback execdriver.StartCallback) (int, error) {
	client, err := d.getClient()
	if err != nil {
		return -1, err
	}

	defer client.Close()

	wrapper := WrapCommand(c)
	var pid, exitCode, dummy int
	err = client.Call("CmdDriver.Start", wrapper, &pid)
	if pid != -1 {
		c.Process, _ = os.FindProcess(pid)
		if startCallback != nil {
			startCallback(c)
		}
	}
	err = client.Call("CmdDriver.Wait", dummy, &exitCode)
	return exitCode, err
}

func (d *driver) Kill(c *execdriver.Command, sig int) error {
	client, err := d.getClient()
	if err != nil {
		return err
	}
	defer client.Close()

	var dummy int
	err = client.Call("CmdDriver.Kill", sig, &dummy)
	if err != nil {
		return err
	}

	return nil
}

func (d *driver) Restore(c *execdriver.Command) error {
	client, err := d.getClient()
	if err != nil {
		return err
	}
	defer client.Close()

	var exitCode int
	err = client.Call("CmdDriver.Wait", 0, &exitCode)
	return err
}

type info struct {
	ID     string
	driver *driver
}

func (i *info) IsRunning() (r bool) {
	client, err := i.driver.getClient()
	if err != nil {
		return false
	}
	defer client.Close()

	var running bool
	err = client.Call("CmdDriver.IsRunning", 0, &running)
	if err != nil {
		return false
	}

	return running
}

func (d *driver) Info(id string) execdriver.Info {
	return &info{
		ID:     id,
		driver: d,
	}
}

func (d *driver) GetPidsForContainer(id string) ([]int, error) {
	client, err := d.getClient()
	if err != nil {
		return nil, err
	}
	defer client.Close()

	var pids []int
	err = client.Call("CmdDriver.GetPids", 0, &pids)
	if err != nil {
		return nil, err
	}

	return pids, nil
}
