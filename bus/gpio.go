package bus

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unsafe"

	iface "github.com/sergz72/go-peripheral-library/interface"
	"github.com/sergz72/go-peripheral-library/utils"
)

/*
 * The maximum size of name and label arrays.
 *
 * Must be a multiple of 8 to ensure 32/64-bit alignment of structs.
 */
const GPIOMaxNameSize = 32

/*
 * Maximum number of requested lines.
 *
 * Must be no greater than 64, as bitmaps are restricted here to 64-bits
 * for simplicity, and a multiple of 2 to ensure 32/64-bit alignment of
 * structs.
 */
const GPIOV2LinesMax = 64

/*
 * The maximum number of configuration attributes associated with a line
 * request.
 */
const GPIOV2LineNumAttrsMax = 10

/**
 * struct gpiochip_info - Information about a certain GPIO chip
 * @name: the Linux kernel name of this GPIO chip
 * @label: a functional name for this GPIO chip, such as a product
 * number, may be empty (i.e. label[0] == '\0')
 * @lines: number of GPIO lines on this chip
 */
type gpioChipInfo struct {
	name  [GPIOMaxNameSize]byte
	label [GPIOMaxNameSize]byte
	lines uint32
}

/**
 * enum gpio_v2_line_flag - &struct gpio_v2_line_attribute.flags values
 * @GPIO_V2_LINE_FLAG_USED: line is not available for request
 * @GPIO_V2_LINE_FLAG_ACTIVE_LOW: line active state is physical low
 * @GPIO_V2_LINE_FLAG_INPUT: line is an input
 * @GPIO_V2_LINE_FLAG_OUTPUT: line is an output
 * @GPIO_V2_LINE_FLAG_EDGE_RISING: line detects rising (inactive to active)
 * edges
 * @GPIO_V2_LINE_FLAG_EDGE_FALLING: line detects falling (active to
 * inactive) edges
 * @GPIO_V2_LINE_FLAG_OPEN_DRAIN: line is an open drain output
 * @GPIO_V2_LINE_FLAG_OPEN_SOURCE: line is an open source output
 * @GPIO_V2_LINE_FLAG_BIAS_PULL_UP: line has pull-up bias enabled
 * @GPIO_V2_LINE_FLAG_BIAS_PULL_DOWN: line has pull-down bias enabled
 * @GPIO_V2_LINE_FLAG_BIAS_DISABLED: line has bias disabled
 */
const (
	GPIOV2LineFlagUsed         = 1
	GPIOV2LineFlagActiveLow    = 2
	GPIOV2LineFlagInput        = 4
	GPIOV2LineFlagOutput       = 8
	GPIOV2LineFlagEdgeRising   = 16
	GPIOV2LineFlagEdgeFalling  = 32
	GPIOV2LineFlagOpenDrain    = 64
	GPIOV2LineFlagOpenSource   = 128
	GPIOV2LineFlagBiasPullUp   = 256
	GPIOV2LineFlagBiasPullDown = 512
	GPIOV2LineFlagBiasDisabled = 1024
)

/**
 * struct gpio_v2_line_values - Values of GPIO lines
 * @bits: a bitmap containing the value of the lines, set to 1 for active
 * and 0 for inactive.
 * @mask: a bitmap identifying the lines to get or set, with each bit
 * number corresponding to the index into &struct
 * gpio_v2_line_request.offsets.
 */
type gpioV2LineValues struct {
	bits uint64
	mask uint64
}

/**
 * enum gpio_v2_line_attr_id - &struct gpio_v2_line_attribute.id values
 * identifying which field of the attribute union is in use.
 * @GPIO_V2_LINE_ATTR_ID_FLAGS: flags field is in use
 * @GPIO_V2_LINE_ATTR_ID_OUTPUT_VALUES: values field is in use
 * @GPIO_V2_LINE_ATTR_ID_DEBOUNCE: debounce_period_us field is in use
 */
const (
	GPIOV2LineAttrIdFlags        = 1
	GPIOV2LineAttrIdOutputValues = 2
	GPIOV2LineAttrIdDebounce     = 3
)

/**
 * struct gpio_v2_line_attribute - a configurable attribute of a line
 * @id: attribute identifier with value from &enum gpio_v2_line_attr_id
 * @padding: reserved for future use and must be zero filled
 * @flags: if id is %GPIO_V2_LINE_ATTR_ID_FLAGS, the flags for the GPIO
 * line, with values from &enum gpio_v2_line_flag, such as
 * %GPIO_V2_LINE_FLAG_ACTIVE_LOW, %GPIO_V2_LINE_FLAG_OUTPUT etc, added
 * together.  This overrides the default flags contained in the &struct
 * gpio_v2_line_config for the associated line.
 * @values: if id is %GPIO_V2_LINE_ATTR_ID_OUTPUT_VALUES, a bitmap
 * containing the values to which the lines will be set, with each bit
 * number corresponding to the index into &struct
 * gpio_v2_line_request.offsets.
 * @debounce_period_us: if id is %GPIO_V2_LINE_ATTR_ID_DEBOUNCE, the
 * desired debounce period, in microseconds
 */
type GpioV2LineAttribute struct {
	id          uint32
	padding     uint32
	flagsValues uint64
}

/**
 * struct gpio_v2_line_config_attribute - a configuration attribute
 * associated with one or more of the requested lines.
 * @attr: the configurable attribute
 * @mask: a bitmap identifying the lines to which the attribute applies,
 * with each bit number corresponding to the index into &struct
 * gpio_v2_line_request.offsets.
 */
type gpioV2LineConfigAttribute struct {
	attr GpioV2LineAttribute
	mask uint64
}

/**
 * struct gpio_v2_line_config - Configuration for GPIO lines
 * @flags: flags for the GPIO lines, with values from &enum
 * gpio_v2_line_flag, such as %GPIO_V2_LINE_FLAG_ACTIVE_LOW,
 * %GPIO_V2_LINE_FLAG_OUTPUT etc, added together.  This is the default for
 * all requested lines but may be overridden for particular lines using
 * @attrs.
 * @num_attrs: the number of attributes in @attrs
 * @padding: reserved for future use and must be zero filled
 * @attrs: the configuration attributes associated with the requested
 * lines.  Any attribute should only be associated with a particular line
 * once.  If an attribute is associated with a line multiple times then the
 * first occurrence (i.e. lowest index) has precedence.
 */
type gpioV2LineConfig struct {
	flags    uint64
	numAttrs uint32
	/* Pad to fill implicit padding and reserve space for future use. */
	padding [5]uint32
	attrs   [GPIOV2LineNumAttrsMax]gpioV2LineConfigAttribute
}

/**
 * struct gpio_v2_line_request - Information about a request for GPIO lines
 * @offsets: an array of desired lines, specified by offset index for the
 * associated GPIO chip
 * @consumer: a desired consumer label for the selected GPIO lines such as
 * "my-bitbanged-relay"
 * @config: requested configuration for the lines.
 * @num_lines: number of lines requested in this request, i.e. the number
 * of valid fields in the %GPIO_V2_LINES_MAX sized arrays, set to 1 to
 * request a single line
 * @event_buffer_size: a suggested minimum number of line events that the
 * kernel should buffer.  This is only relevant if edge detection is
 * enabled in the configuration. Note that this is only a suggested value
 * and the kernel may allocate a larger buffer or cap the size of the
 * buffer. If this field is zero then the buffer size defaults to a minimum
 * of @num_lines * 16.
 * @padding: reserved for future use and must be zero filled
 * @fd: if successful this field will contain a valid anonymous file handle
 * after a %GPIO_GET_LINE_IOCTL operation, zero or negative value means
 * error
 */
type GpioV2LineRequest struct {
	offsets         [GPIOV2LinesMax]uint32
	consumer        [GPIOMaxNameSize]byte
	config          gpioV2LineConfig
	numLines        uint32
	eventBufferSize uint32
	/* Pad to fill implicit padding and reserve space for future use. */
	padding [5]uint32
	fd      int32
}

/**
 * struct gpio_v2_line_info - Information about a certain GPIO line
 * @name: the name of this GPIO line, such as the output pin of the line on
 * the chip, a rail or a pin header name on a board, as specified by the
 * GPIO chip, may be empty (i.e. name[0] == '\0')
 * @consumer: a functional name for the consumer of this GPIO line as set
 * by whatever is using it, will be empty if there is no current user but
 * may also be empty if the consumer doesn't set this up
 * @offset: the local offset on this GPIO chip, fill this in when
 * requesting the line information from the kernel
 * @num_attrs: the number of attributes in @attrs
 * @flags: flags for the GPIO lines, with values from &enum
 * gpio_v2_line_flag, such as %GPIO_V2_LINE_FLAG_ACTIVE_LOW,
 * %GPIO_V2_LINE_FLAG_OUTPUT etc, added together.
 * @attrs: the configuration attributes associated with the line
 * @padding: reserved for future use
 */
type GpioV2LineInfo struct {
	name     [GPIOMaxNameSize]byte
	consumer [GPIOMaxNameSize]byte
	offset   uint32
	numAttrs uint32
	flags    uint64
	attrs    [GPIOV2LineNumAttrsMax]GpioV2LineAttribute
	/* Space reserved for future use. */
	padding [4]uint32
}

/**
 * enum gpio_v2_line_changed_type - &struct gpio_v2_line_changed.event_type
 * values
 * @GPIO_V2_LINE_CHANGED_REQUESTED: line has been requested
 * @GPIO_V2_LINE_CHANGED_RELEASED: line has been released
 * @GPIO_V2_LINE_CHANGED_CONFIG: line has been reconfigured
 */
const (
	GPIOV2LineChangedRequested = 1
	GPIOV2LineChangedReleased  = 2
	GPIOV2LineChangedConfig    = 3
)

/**
 * struct gpio_v2_line_info_changed - Information about a change in status
 * of a GPIO line
 * @info: updated line information
 * @timestamp_ns: estimate of time of status change occurrence, in nanoseconds
 * @event_type: the type of change with a value from &enum
 * gpio_v2_line_changed_type
 * @padding: reserved for future use
 */
type gpioV2LineInfoChanged struct {
	info        GpioV2LineInfo
	timestampNs uint64
	eventType   uint32
	/* Pad struct to 64-bit boundary and reserve space for future use. */
	padding [5]uint32
}

/**
 * enum gpio_v2_line_event_id - &struct gpio_v2_line_event.id values
 * @GPIO_V2_LINE_EVENT_RISING_EDGE: event triggered by a rising edge
 * @GPIO_V2_LINE_EVENT_FALLING_EDGE: event triggered by a falling edge
 */
const (
	GPIOV2LineEventRisingEdge  = 1
	GPIOV2LineEventFallingEdge = 2
)

/**
 * struct gpio_v2_line_event - The actual event being pushed to userspace
 * @timestamp_ns: best estimate of time of event occurrence, in nanoseconds.
 * The @timestamp_ns is read from %CLOCK_MONOTONIC and is intended to allow
 * the accurate measurement of the time between events. It does not provide
 * the wall-clock time.
 * @id: event identifier with value from &enum gpio_v2_line_event_id
 * @offset: the offset of the line that triggered the event
 * @seqno: the sequence number for this event in the sequence of events for
 * all the lines in this line request
 * @line_seqno: the sequence number for this event in the sequence of
 * events on this particular line
 * @padding: reserved for future use
 */
type gpioV2LineEvent struct {
	timestampNs uint64
	id          uint32
	offset      uint32
	seqno       uint32
	lineSeqno   uint32
	/* Space reserved for future use. */
	padding [6]uint32
}

/*
 * v1 and v2 ioctl()s
 */
func gPIOGetChipInfoIoctl() uintptr {
	return iface.IOR(0xB4, 0x01, unsafe.Sizeof(gpioChipInfo{}))
}

func gPIOGetLineInfoUnwatchIoctl() uintptr {
	return iface.IOWR(0xB4, 0x0C, 4)
}

/*
 * v2 ioctl()s
 */
func gPIOV2GetLineInfoIoctl() uintptr {
	return iface.IOWR(0xB4, 0x05, unsafe.Sizeof(GpioV2LineInfo{}))
}

func gPIOV2GetLineinfoWatchIoctl() uintptr {
	return iface.IOWR(0xB4, 0x06, unsafe.Sizeof(GpioV2LineInfo{}))
}

func gPIOV2GetLineIoctl() uintptr {
	return iface.IOWR(0xB4, 0x07, unsafe.Sizeof(GpioV2LineRequest{}))
}

func gPIOV2LineSetConfigIoctl() uintptr {
	return iface.IOWR(0xB4, 0x0D, unsafe.Sizeof(gpioV2LineConfig{}))
}

func gPIOV2LineGetValuesIoctl() uintptr {
	return iface.IOWR(0xB4, 0x0E, unsafe.Sizeof(gpioV2LineValues{}))
}

func gPIOV2LineSetValuesIoctl() uintptr {
	return iface.IOWR(0xB4, 0x0F, unsafe.Sizeof(gpioV2LineValues{}))
}

type GPIO struct {
	f *os.File
}

func (g *GPIO) Close() {
	_ = g.f.Close()
}

type GPIOPin struct {
	fd int32
}

func (g *GPIOPin) Close() {
	_ = syscall.Close(int(g.fd))
}

func (g *GPIOPin) Read() (bool, error) {
	var values gpioV2LineValues
	values.mask = 1
	err := iface.IOCTL(uintptr(g.fd), gPIOV2LineGetValuesIoctl(), uintptr(unsafe.Pointer(&values)))
	if err != nil {
		return false, err
	}
	return (values.bits & 1) != 0, nil
}

func (g *GPIOPin) Write(level bool) error {
	var values gpioV2LineValues
	if level {
		values.bits = 1
	} else {
		values.bits = 0
	}
	values.mask = 1
	return iface.IOCTL(uintptr(g.fd), gPIOV2LineSetValuesIoctl(), uintptr(unsafe.Pointer(&values)))
}

func NewGPIO(chipNumber int) (*GPIO, error) {
	deviceName := "/dev/gpiochip" + strconv.Itoa(chipNumber)
	f, err := os.OpenFile(deviceName, syscall.O_RDWR, 0666)
	if err != nil {
		return nil, err
	}
	return &GPIO{f}, nil
}

func (g *GPIO) GetChipInfo() (string, string, int, error) {
	var chipInfo gpioChipInfo
	err := iface.IOCTL(g.f.Fd(), gPIOGetChipInfoIoctl(), uintptr(unsafe.Pointer(&chipInfo)))
	if err != nil {
		return "", "", 0, err
	}
	return utils.ZeroTerminatedToString(chipInfo.name[:]), utils.ZeroTerminatedToString(chipInfo.label[:]), int(chipInfo.lines), nil
}

func (g *GPIO) GetLineInfo(offset int) (string, string, *GpioV2LineInfo, error) {
	var lineInfo GpioV2LineInfo
	lineInfo.offset = uint32(offset)
	err := iface.IOCTL(g.f.Fd(), gPIOV2GetLineInfoIoctl(), uintptr(unsafe.Pointer(&lineInfo)))
	if err != nil {
		return "", "", nil, err
	}
	return utils.ZeroTerminatedToString(lineInfo.name[:]), utils.ZeroTerminatedToString(lineInfo.consumer[:]), &lineInfo, nil
}

func (g *GPIO) LineRequest(request *GpioV2LineRequest) error {
	return iface.IOCTL(g.f.Fd(), gPIOV2GetLineIoctl(), uintptr(unsafe.Pointer(request)))
}

func (g *GPIO) SetLineOutput(offset int, level bool) (*GPIOPin, error) {
	var request GpioV2LineRequest
	request.numLines = 1
	request.offsets[0] = uint32(offset)
	var value uint64
	if level {
		value = 1
	} else {
		value = 0
	}
	request.config.flags = GPIOV2LineFlagOutput
	request.config.numAttrs = 1
	request.config.attrs[0].attr.id = GPIOV2LineAttrIdOutputValues
	request.config.attrs[0].attr.flagsValues = value
	request.config.attrs[0].mask = 1
	err := g.LineRequest(&request)
	if err != nil {
		return nil, err
	}
	return &GPIOPin{request.fd}, nil
}

func (g *GPIO) SetLineInput(offset int, flags uint64) (*GPIOPin, error) {
	var request GpioV2LineRequest
	request.numLines = 1
	request.offsets[0] = uint32(offset)
	request.config.flags = GPIOV2LineFlagInput | flags
	err := g.LineRequest(&request)
	if err != nil {
		return nil, err
	}
	return &GPIOPin{request.fd}, nil
}

func GPIOTest(parameters string) error {
	parts := strings.Split(parameters, ",")
	if len(parts) < 2 {
		return errors.New("invalid chipinfo parameter")
	}
	id, err := strconv.Atoi(parts[0])
	if err != nil {
		return err
	}
	if id < 0 {
		return errors.New("invalid chip id")
	}
	gpio, err := NewGPIO(id)
	if err != nil {
		return err
	}
	defer gpio.Close()
	switch parts[1] {
	case "chipinfo":
		if len(parts) != 2 {
			return errors.New("invalid chipinfo parameter")
		}
		name, label, lines, err := gpio.GetChipInfo()
		if err != nil {
			return err
		}
		fmt.Printf("chipinfo: name=%s label=%s lines=%d\n", name, label, lines)
		return nil
	case "lineinfo":
		if len(parts) != 3 {
			return errors.New("invalid lineinfo parameter")
		}
		offset, err := strconv.Atoi(parts[2])
		if err != nil {
			return err
		}
		if offset < 0 {
			return errors.New("invalid line id")
		}
		name, consumer, _, err := gpio.GetLineInfo(offset)
		if err != nil {
			return err
		}
		fmt.Printf("lineinfo: name=%s consumer=%s\n", name, consumer)
		return nil
	case "lineset":
		if len(parts) < 4 {
			return errors.New("invalid lineset parameter")
		}
		offset, err := strconv.Atoi(parts[2])
		if err != nil {
			return err
		}
		if offset < 0 {
			return errors.New("invalid line id")
		}
		switch parts[3] {
		case "out":
			if len(parts) != 5 || (parts[4] != "1" && parts[4] != "0") {
				return errors.New("invalid lineset out parameter")
			}
			level := parts[4] == "1"
			pin, err := gpio.SetLineOutput(offset, level)
			if err != nil {
				return err
			}
			pin.Close()
			return nil
		case "toggle":
			if len(parts) != 4 {
				return errors.New("invalid lineset toggle parameter")
			}
			level := false
			pin, err := gpio.SetLineOutput(offset, level)
			if err != nil {
				return err
			}
			defer pin.Close()
			for range 4 {
				time.Sleep(time.Second)
				level = !level
				err = pin.Write(level)
				if err != nil {
					return err
				}
			}
			return nil
		case "in_float":
			fallthrough
		case "in_pullup":
			fallthrough
		case "in_pulldown":
			if len(parts) != 4 {
				return errors.New("invalid lineset in parameter")
			}
			var flags uint64 = 0
			if parts[3] == "in_pullup" {
				flags = GPIOV2LineFlagBiasPullUp
			} else if parts[3] == "in_pulldown" {
				flags = GPIOV2LineFlagBiasPullDown
			}
			pin, err := gpio.SetLineInput(offset, flags)
			if err != nil {
				return err
			}
			defer pin.Close()
			level, err := pin.Read()
			if err != nil {
				return err
			}
			fmt.Printf("pin level %v\n", level)
			return nil
		default:
			return errors.New("invalid lineset parameter name")
		}
	default:
		return fmt.Errorf("invalid argument %s", parts[0])
	}
}
