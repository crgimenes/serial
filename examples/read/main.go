package main

import (
	"flag"
	"log"

	"serial"
)

func main() {
	serialPort := flag.String("serial", "", "Serial port to use")
	serialBaud := flag.Int("baud", 115200, "Baud rate")

	flag.Parse()

	c := &serial.Config{Name: *serialPort, Baud: *serialBaud}
	s, err := serial.OpenPort(c)
	if err != nil {
		log.Fatal(err)
	}

	//n, err := s.Write([]byte("test"))
	//if err != nil {
	//        log.Fatal(err)
	//}

	for {
		buf := make([]byte, 128)
		n, err := s.Read(buf)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("%q", string(buf[:n]))
	}
}
