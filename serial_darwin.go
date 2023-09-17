package serial

import (
	"errors"
	"log"
	"os"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

type Port struct {
	f *os.File
}

func (p *Port) Read(b []byte) (n int, err error) {
	return p.f.Read(b)
}

func (p *Port) Write(b []byte) (n int, err error) {
	return p.f.Write(b)
}

func (p *Port) Flush() error {
	p.f.Sync()
	return nil
}

func (p *Port) Close() (err error) {
	return p.f.Close()
}

func openPort(name string, baud int, databits byte, parity Parity, stopbits StopBits, readTimeout time.Duration) (*Port, error) {
	log.Printf("openPort(%s, %d, %d, %d, %d, %d)\n", name, baud, databits, parity, stopbits, readTimeout)

	f, err := os.OpenFile(name, syscall.O_RDWR|syscall.O_NOCTTY|syscall.O_NONBLOCK, 0666)
	if err != nil {
		log.Printf("openPort: %s\n", err)
		return nil, err
	}

	fd := int(f.Fd())
	if !isTTY(fd) {
		f.Close()
		return nil, errors.New("File is not a tty")
	}

	termios, err := unix.IoctlGetTermios(fd, unix.TIOCGETA)
	if err != nil {
		f.Close()
		log.Printf("error getting termios: %s\n", err)
		return nil, err
	}

	termios.Cflag &= ^(uint64)(unix.BRKINT | unix.ICRNL | unix.INPCK | unix.ISTRIP | unix.IXOFF | unix.IXON | unix.PARMRK)
	termios.Cflag &= ^(uint64)(unix.CSIZE | unix.PARENB)
	termios.Cflag |= (unix.CLOCAL | unix.CREAD)

	speed := getBaudRate(baud)
	termios.Ispeed = speed
	termios.Ospeed = speed

	switch databits {
	case 5:
		termios.Cflag |= unix.CS5
	case 6:
		termios.Cflag |= unix.CS6
	case 7:
		termios.Cflag |= unix.CS7
	case 8:
		termios.Cflag |= unix.CS8
	default:
		return nil, ErrBadSize
	}

	switch parity {
	case ParityNone:
		// default is no parity
	case ParityOdd:
		termios.Cflag |= unix.PARENB
		termios.Cflag |= unix.PARODD
	case ParityEven:
		termios.Cflag |= unix.PARENB
		termios.Cflag &^= unix.PARODD
	default:
		return nil, ErrBadParity
	}

	switch stopbits {
	case Stop1:
		// as is, default is 1 bit
	case Stop2:
		termios.Cflag |= unix.CSTOPB
	default:
		return nil, ErrBadStopBits
	}

	termios.Lflag &^= unix.ICANON | unix.ECHO | unix.ECHOE | unix.ISIG
	termios.Oflag &^= unix.OPOST

	vmin, vtime := posixTimeoutValues(readTimeout)
	termios.Cc[unix.VMIN] = vmin
	termios.Cc[unix.VTIME] = vtime

	if err := unix.IoctlSetTermios(fd, unix.TIOCSETA, termios); err != nil {
		log.Printf("error setting termios: %s\n", err)
		f.Close()
		return nil, err
	}

	if err := unix.SetNonblock(fd, false); err != nil {
		log.Printf("error setting nonblock: %s\n", err)
		f.Close()
		return nil, err
	}

	return &Port{f: f}, nil
}

func isTTY(fd int) bool {
	_, err := unix.IoctlGetTermios(fd, unix.TIOCGETA)
	return err == nil
}

func getBaudRate(baud int) uint64 {
	baudRates := map[int]uint64{
		50:     unix.B50,
		75:     unix.B75,
		110:    unix.B110,
		134:    unix.B134,
		150:    unix.B150,
		200:    unix.B200,
		300:    unix.B300,
		600:    unix.B600,
		1200:   unix.B1200,
		2400:   unix.B2400,
		4800:   unix.B4800,
		9600:   unix.B9600,
		19200:  unix.B19200,
		38400:  unix.B38400,
		57600:  unix.B57600,
		115200: unix.B115200,
		230400: unix.B230400,
	}

	speed, ok := baudRates[baud]
	if !ok {
		return unix.B9600 // Default to 9600 baud
	}
	return speed
}
