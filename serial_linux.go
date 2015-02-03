// +build linux,!cgo

package goserial

import (
	"fmt"
	"io"
	"os"
	"syscall"
	"unsafe"
)

const CBAUD = 0010017

func openPort(name string, c *Config) (rwc io.ReadWriteCloser, err error) {
	f, err := os.OpenFile(name, syscall.O_RDWR|syscall.O_NOCTTY|syscall.O_NONBLOCK, 0666)
	if err != nil {
		return nil, err
	}

	fd := f.Fd()
	t := syscall.Termios{}
	if _, _, errno := syscall.Syscall6(
		syscall.SYS_IOCTL,
		uintptr(fd),
		uintptr(syscall.TCGETS),
		uintptr(unsafe.Pointer(&t)),
		0,
		0,
		0,
	); errno != 0 {
		return nil, errno
	}

	var rate uint32
	switch c.Baud {
	case 115200:
		rate = syscall.B115200
	case 57600:
		rate = syscall.B57600
	case 38400:
		rate = syscall.B38400
	case 19200:
		rate = syscall.B19200
	case 9600:
		rate = syscall.B9600
	case 4800:
		rate = syscall.B4800
	default:
		return nil, fmt.Errorf("Unknown baud rate %v", c.Baud)
	}

	if rate == 0 {
		return
	}
	t.Cflag = (t.Cflag &^ CBAUD) | rate
	t.Ispeed = rate
	t.Ospeed = rate

	switch c.StopBits {
	case StopBits1:
		t.Cflag &^= syscall.CSTOPB
	case StopBits2:
		t.Cflag |= syscall.CSTOPB
	default:
		panic(c.StopBits)
	}

	t.Cflag &^= syscall.CSIZE
	switch c.Size {
	case Byte5:
		t.Cflag |= syscall.CS5
	case Byte6:
		t.Cflag |= syscall.CS6
	case Byte7:
		t.Cflag |= syscall.CS7
	case Byte8:
		t.Cflag |= syscall.CS8
	default:
		panic(c.Size)
	}

	switch c.Parity {
	case ParityNone:
		t.Cflag &^= syscall.PARENB
	case ParityEven:
		t.Cflag |= syscall.PARENB
	case ParityOdd:
		t.Cflag |= syscall.PARENB
		t.Cflag |= syscall.PARODD
	default:
		panic(c.Parity)
	}

	if c.CRLFTranslate {
		t.Iflag |= syscall.ICRNL
	} else {
		t.Iflag &^= syscall.ICRNL
	}

	// Select raw mode
	t.Lflag &^= syscall.ICANON | syscall.ECHO | syscall.ECHOE | syscall.ISIG
	t.Oflag &^= syscall.OPOST

	defer func() {
		if err != nil && f != nil {
			f.Close()
		}
	}()

	t.Cc[syscall.VMIN] = 1
	t.Cc[syscall.VTIME] = 0
	if _, _, errno := syscall.Syscall6(
		syscall.SYS_IOCTL,
		uintptr(fd),
		uintptr(syscall.TCSETS),
		uintptr(unsafe.Pointer(&t)),
		0,
		0,
		0,
	); errno != 0 {
		return nil, errno
	}

	if err = syscall.SetNonblock(int(fd), false); err != nil {
		return
	}

	return f, nil
}
