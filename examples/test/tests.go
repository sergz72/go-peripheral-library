package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/sergz72/go-peripheral-library/bus"
	"github.com/sergz72/go-peripheral-library/device"
)

const (
	CommandNone = iota
	CommandTestScd41
	CommandTestCC1101
	CommandI2CScan
	CommandSPITest
	CommandGPIOTest
)

func main() {
	var configFileName string
	var gpioParameters string
	configFileNameExpected := false
	i2cBusNumberExpected := false
	spiBusNumberExpected := false
	gpioParametersExpected := false
	var busNumber int
	var deviceNumber int
	command := CommandNone
	for _, arg := range os.Args {
		if configFileNameExpected {
			configFileName = arg
			configFileNameExpected = false
			continue
		}
		if gpioParametersExpected {
			gpioParameters = arg
			gpioParametersExpected = false
			continue
		}
		if i2cBusNumberExpected {
			var err error
			busNumber, err = strconv.Atoi(arg)
			if err != nil {
				log.Fatal(err)
			}
			i2cBusNumberExpected = false
			continue
		}
		if spiBusNumberExpected {
			var err error
			parts := strings.Split(arg, ",")
			if len(parts) != 2 {
				log.Fatal("SPI bus and device number expected")
			}
			busNumber, err = strconv.Atoi(parts[0])
			if err != nil {
				log.Fatal(err)
			}
			deviceNumber, err = strconv.Atoi(parts[1])
			if err != nil {
				log.Fatal(err)
			}
			spiBusNumberExpected = false
			continue
		}
		if strings.HasPrefix(arg, "--") {
			switch arg {
			case "--test-scd41":
				command = CommandTestScd41
				i2cBusNumberExpected = true
			case "--test-cc1101":
				command = CommandTestCC1101
				configFileNameExpected = true
			case "--i2c-scan":
				command = CommandI2CScan
				i2cBusNumberExpected = true
			case "--test-spi":
				command = CommandSPITest
				spiBusNumberExpected = true
			case "--test-gpio":
				command = CommandGPIOTest
				gpioParametersExpected = true
			default:
				fmt.Println("Usage: tests [--test-scd41 i2c_no][--test-cc1101 configFileName][--i2c-scan i2c_no][--test-spi spi_no][--test-gpio parameters]")
				os.Exit(1)
			}
		}
	}
	if command == CommandNone {
		fmt.Println("no test selected")
		os.Exit(1)
	}
	if configFileNameExpected || (command == CommandTestCC1101 && configFileName == "") {
		fmt.Println("configuration file name expected")
		os.Exit(1)
	}
	if i2cBusNumberExpected {
		fmt.Println("I2C bus number expected")
		os.Exit(1)
	}
	if spiBusNumberExpected {
		fmt.Println("SPI bus number expected")
		os.Exit(1)
	}
	if gpioParametersExpected {
		fmt.Println("GPIO parameters expected")
		os.Exit(1)
	}
	var err error
	switch command {
	case CommandTestScd41:
		scd, err := device.NewSCD4X(busNumber)
		if err != nil {
			log.Fatal(err)
		}
		err = scd.Test()
	case CommandI2CScan:
		err = bus.I2cScan(busNumber)
	case CommandSPITest:
		spi, err := bus.NewSPIMaster(busNumber, deviceNumber, 8, 1000000)
		if err != nil {
			log.Fatal(err)
		}
		err = spi.Test()
	case CommandGPIOTest:
		err = bus.GPIOTest(gpioParameters)
	case CommandTestCC1101:
		cc1101, err := device.CC1101Init(configFileName)
		if err != nil {
			log.Fatal(err)
		}
		err = cc1101.Test()
	default:
		err = errors.New("invalid argument")
	}
	if err != nil {
		log.Fatal(err)
	}
}
