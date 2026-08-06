package _interface

import (
	"syscall"
)

const (
	iocNrbits   = 8
	iocTypebits = 8

	IocSizebits = 14
	iocDirbits  = 2

	iocNrmask   = (1 << iocNrbits) - 1
	iocTypemask = (1 << iocTypebits) - 1
	iocSizemask = (1 << IocSizebits) - 1
	iocDirmask  = (1 << iocDirbits) - 1

	iocNrshift   = 0
	iocTypeshift = iocNrshift + iocNrbits
	iocSizeshift = iocTypeshift + iocTypebits
	iocDirshift  = iocSizeshift + IocSizebits

	// Direction bits
	iocNone  = 0
	iocWrite = 1
	iocRead  = 2
)

// ...and for the drivers/sound files...
const (
	iocIn       = iocWrite << iocDirshift
	iocOut      = iocRead << iocDirshift
	iocInout    = (iocWrite | iocRead) << iocDirshift
	iocsizeMask = iocSizemask << iocSizeshift
)

func IOC(dir, t, nr, size uintptr) uintptr {
	return (dir << iocDirshift) | (t << iocTypeshift) | (nr << iocNrshift) | (size << iocSizeshift)
}

// used to create ioctl numbers

func IO(t, nr uintptr) uintptr {
	return IOC(iocNone, t, nr, 0)
}

func IOR(t, nr, size uintptr) uintptr {
	return IOC(iocRead, t, nr, size)
}

func IOW(t, nr, size uintptr) uintptr {
	return IOC(iocWrite, t, nr, size)
}

func IOWR(t, nr, size uintptr) uintptr {
	return IOC(iocRead|iocWrite, t, nr, size)
}

func IOR_BAD(t, nr, size uintptr) uintptr {
	return IOC(iocRead, t, nr, size)
}

func IOW_BAD(t, nr, size uintptr) uintptr {
	return IOC(iocWrite, t, nr, size)
}

func IOWR_BAD(t, nr, size uintptr) uintptr {
	return IOC(iocRead|iocWrite, t, nr, size)
}

func IOCTL(fd, op, arg uintptr) error {
	_, _, ep := syscall.Syscall(syscall.SYS_IOCTL, fd, op, arg)
	if ep != 0 {
		return ep
	}
	return nil
}
