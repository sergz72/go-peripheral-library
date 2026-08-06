package device

import (
	"fmt"
	"time"

	"github.com/sergz72/go-peripheral-library/bus"
	"github.com/sergz72/go-peripheral-library/utils"
)

const (
	scd4xRawDataSize = 9
	scd4xSensorAddr  = 0x62
)

type SCD4X struct {
	client *bus.I2CMaster
	crc    *utils.Crc8
}

type SCD4XResult struct {
	Temperature int16
	Humidity    uint16
	Co2         uint
}

func (r *SCD4XResult) Print() {
	fmt.Printf("Temperature %d Humidity %d CO2 %d\n", r.Temperature, r.Humidity, r.Co2)
}

func NewSCD4X(busNumber int) (*SCD4X, error) {
	client, err := bus.NewI2CMaster(busNumber)
	if err != nil {
		return nil, err
	}
	err = client.SetSlaveAddress(scd4xSensorAddr)
	if err != nil {
		client.Close()
		return nil, err
	}
	return &SCD4X{client: client, crc: utils.NewCrc31()}, nil
}

func (d *SCD4X) Close() {
	d.client.Close()
}

func (d *SCD4X) validateRawDataItem(rawData []byte, offset int, name string) error {
	if d.crc.Compute(rawData[offset:offset+2]) != rawData[offset+2] {
		return fmt.Errorf("%s CRC error", name)
	}
	return nil
}

func (d *SCD4X) validateRawData(rawData []byte) error {
	err := d.validateRawDataItem(rawData, 0, "CO2")
	if err != nil {
		return err
	}
	err = d.validateRawDataItem(rawData, 3, "temperature")
	if err != nil {
		return err
	}
	return d.validateRawDataItem(rawData, 6, "humidity")
}

func (d *SCD4X) computeValues(rawData []byte) SCD4XResult {
	var r SCD4XResult
	r.Co2 = ((uint(rawData[0]) << 8) | uint(rawData[1])) * 100
	t := (int(rawData[3]) << 8) | int(rawData[4])
	h := (uint(rawData[6]) << 8) | uint(rawData[7])
	r.Temperature = int16(-4500 + 17500*t/65535)
	r.Humidity = uint16(10000 * h / 65535)
	return r
}

func (d *SCD4X) StartMeasurement() error {
	return d.client.Write([]byte{0x21, 0x9D})
}

func (d *SCD4X) PowerDown() error {
	return d.client.Write([]byte{0x36, 0xE0})
}

func (d *SCD4X) WakeUp() error {
	err := d.client.Write([]byte{0x36, 0xF6})
	time.Sleep(30 * time.Millisecond)
	return err
}

func (d *SCD4X) getStatus() (uint16, error) {
	err := d.client.Write([]byte{0xE4, 0xB8})
	if err != nil {
		return 0, err
	}
	time.Sleep(2 * time.Millisecond)
	var data []byte
	data, err = d.client.Read(3)
	if err != nil {
		return 0, err
	}
	err = d.validateRawDataItem(data, 0, "status")
	if err != nil {
		return 0, err
	}
	return (uint16(data[0]) << 8) | uint16(data[1]), nil
}

func (d *SCD4X) ReadMeasurement() (SCD4XResult, error) {
	for {
		status, err := d.getStatus()
		if err != nil {
			return SCD4XResult{}, err
		}
		if (status & 0x3FF) != 0 {
			break
		}
		time.Sleep(time.Second)
	}

	err := d.client.Write([]byte{0xEC, 0x05})
	if err != nil {
		return SCD4XResult{}, err
	}
	time.Sleep(2 * time.Millisecond)
	var data []byte
	data, err = d.client.Read(scd4xRawDataSize)
	if err != nil {
		return SCD4XResult{}, err
	}
	err = d.validateRawData(data)
	if err != nil {
		return SCD4XResult{}, err
	}
	return d.computeValues(data), nil
}

func (d *SCD4X) Get() (SCD4XResult, error) {
	_ = d.WakeUp()
	err := d.StartMeasurement()
	if err != nil {
		return SCD4XResult{}, err
	}
	time.Sleep(6 * time.Second)
	result, err := d.ReadMeasurement()
	if err != nil {
		return SCD4XResult{}, err
	}
	return result, d.PowerDown()
}

func (d *SCD4X) Test() error {
	for range 3 {
		result, err := d.Get()
		if err != nil {
			return err
		}
		result.Print()
	}
	return nil
}
