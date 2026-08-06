package bus

import (
	"fmt"
	"os"
	"strconv"
	"syscall"
	"unsafe"

	iface "github.com/sergz72/go-peripheral-library/interface"
)

const (
	i2cSlave  = 0x0703
	i2cTenBit = 0x0704
	i2cRdwr   = 0x0707
	i2cMRd    = 0x0001
)

type I2CMaster struct {
	f *os.File
}

type i2cMsg struct {
	addr      uint16
	flags     uint16
	len       uint16
	__padding uint16
	buf       uintptr
}

type i2cRdwrIoctlData struct {
	msgs  uintptr
	nmsgs uint32
}

func (c *I2CMaster) Close() {
	_ = c.f.Close()
}

func NewI2CMaster(busNumber int) (*I2CMaster, error) {
	deviceName := "/dev/i2c-" + strconv.Itoa(busNumber)
	f, err := os.OpenFile(deviceName, syscall.O_RDWR, 0666)
	if err != nil {
		return nil, err
	}
	return &I2CMaster{f}, nil
}

func (c *I2CMaster) transfer(msgs []i2cMsg) error {
	data := i2cRdwrIoctlData{
		msgs:  uintptr(unsafe.Pointer(&msgs[0])),
		nmsgs: uint32(len(msgs)),
	}
	return iface.IOCTL(c.f.Fd(), i2cRdwr, uintptr(unsafe.Pointer(&data)))
}

func (c *I2CMaster) SetSlaveAddress(address uint) error {
	return iface.IOCTL(c.f.Fd(), i2cSlave, uintptr(address))
}

func (c *I2CMaster) Write(data []byte) error {
	_, err := c.f.Write(data)
	return err
}

func (c *I2CMaster) Read(numBytes int) ([]byte, error) {
	data := make([]byte, numBytes)
	_, err := c.f.Read(data)
	return data, err
}

func (c *I2CMaster) Transfer(address uint16, wdata []byte, rlength uint16) ([]byte, error) {
	rdata := make([]byte, rlength)
	msg := []i2cMsg{
		{
			addr:  address,
			flags: 0,
			len:   uint16(len(wdata)),
			buf:   uintptr(unsafe.Pointer(&wdata[0])),
		},
		{
			addr:  address,
			flags: i2cMRd,
			len:   rlength,
			buf:   uintptr(unsafe.Pointer(&rdata[0])),
		},
	}
	err := c.transfer(msg)
	if err != nil {
		return nil, err
	}
	return rdata, nil
}

func I2cScan(busNumber int) error {
	fmt.Println("     0  1  2  3  4  5  6  7  8  9  a  b  c  d  e  f")
	fmt.Print("00:   ")
	c, err := NewI2CMaster(busNumber)
	if err != nil {
		return err
	}
	defer c.Close()
	for address := 1; address <= 0x7F; address++ {
		if (address & 0x0F) == 0 {
			fmt.Printf("\n%.2x:", address)
		}
		err = c.SetSlaveAddress(uint(address))
		if err != nil {
			fmt.Print(" --")
		} else {
			err = c.Write([]byte{0})
			if err != nil {
				fmt.Print(" --")
			} else {
				fmt.Printf(" %.2x", address)
			}
		}
	}
	fmt.Println()
	return nil
}
