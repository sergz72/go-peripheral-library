package bus

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"

	iface "github.com/sergz72/go-peripheral-library/interface"
)

type SPIMaster struct {
	f           *os.File
	speedHz     uint32
	bitsPerWord uint8
}

type spiIocTransfer struct {
	txBuf          uint64
	rxBuf          uint64
	length         uint32
	speedHz        uint32
	delayUsecs     uint16
	bitsPerWord    uint8
	csChange       uint8
	txNBits        uint8
	rxNBits        uint8
	wordDelayUsecs uint8
	pad            uint8
}

const spiIocMagic = 107

// Write of SPI mode (SPI_MODE_0..SPI_MODE_3)
func spiIocWrMode() uintptr {
	return iface.IOW(spiIocMagic, 1, 1)
}

// Write SPI bit justification
func spiIocWrLsbFirst() uintptr {
	return iface.IOW(spiIocMagic, 2, 1)
}

// Write SPI device word length (1..N)
func spiIocWrBitsPerWord() uintptr {
	return iface.IOW(spiIocMagic, 3, 1)
}

// Write SPI device default max speed hz
func spiIocWrMaxSpeedHz() uintptr {
	return iface.IOW(spiIocMagic, 4, 4)
}

func spiMessageSize(n uintptr) uintptr {
	if n*unsafe.Sizeof(spiIocTransfer{}) < 1<<iface.IocSizebits {
		return n * unsafe.Sizeof(spiIocTransfer{})
	}
	return 0
}

// Write custom SPI message
func spiIocMessage(n uintptr) uintptr {
	return iface.IOW(spiIocMagic, 0, spiMessageSize(n))
}

func (s *SPIMaster) Close() {
	_ = s.f.Close()
}

func NewSPIMaster(busNumber int, deviceNumber int, bitsPerWord uint8, speedHz uint32) (*SPIMaster, error) {
	deviceName := fmt.Sprintf("/dev/spidev%d.%d", busNumber, deviceNumber)
	f, err := os.OpenFile(deviceName, syscall.O_RDWR, 0666)
	if err != nil {
		return nil, err
	}
	return &SPIMaster{f: f, speedHz: speedHz, bitsPerWord: bitsPerWord}, nil
}

func (s *SPIMaster) SetMode(mode uint8) error {
	return iface.IOCTL(s.f.Fd(), spiIocWrMode(), uintptr(unsafe.Pointer(&mode)))
}

func (s *SPIMaster) SetBitsPerWord(bpw uint8) error {
	s.bitsPerWord = bpw
	return iface.IOCTL(s.f.Fd(), spiIocWrBitsPerWord(), uintptr(unsafe.Pointer(&bpw)))
}

func (s *SPIMaster) SetSpeed(speed uint32) error {
	s.speedHz = speed
	return iface.IOCTL(s.f.Fd(), spiIocWrMaxSpeedHz(), uintptr(unsafe.Pointer(&speed)))
}

func (s *SPIMaster) Transfer(wdata []byte) ([]byte, error) {
	rdata := make([]byte, len(wdata))

	// generates message
	transfer := spiIocTransfer{
		txBuf:          uint64(uintptr(unsafe.Pointer(&wdata[0]))),
		rxBuf:          uint64(uintptr(unsafe.Pointer(&rdata[0]))),
		length:         uint32(len(wdata)),
		delayUsecs:     0,
		bitsPerWord:    s.bitsPerWord,
		speedHz:        s.speedHz,
		txNBits:        0,
		rxNBits:        0,
		wordDelayUsecs: 0,
		csChange:       0,
		pad:            0,
	}

	// sends message over SPI
	err := iface.IOCTL(s.f.Fd(), spiIocMessage(1), uintptr(unsafe.Pointer(&transfer)))
	if err != nil {
		return nil, err
	}

	return rdata, nil
}

func (s *SPIMaster) Test() error {
	wData := []byte{0x55, 0xAA, 0xA5}
	rData, err := s.Transfer(wData)
	if err != nil {
		return err
	}
	fmt.Printf("wdata: %v\n", wData)
	fmt.Printf("rdata: %v\n", rData)
	return nil
}
