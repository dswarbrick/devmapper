package main

import (
	"fmt"
	"net"
	"os"

	"github.com/dswarbrick/devmapper"
)

func main() {
	dm, err := devmapper.NewDevMapper()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	defer dm.Close()

	fmt.Printf("Kernel devmapper version: %s\n", dm.Version())

	devices, err := dm.ListDevices()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	for _, device := range devices {
		targets, err := dm.TableStatus(device.Dev)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		fmt.Printf("%#v %#v\n", device, targets)
	}

	// Experimental interaction with lvmetad Unix socket
	conn, err := net.DialUnix("unix", nil, &net.UnixAddr{"/run/lvm/lvmetad.socket", "unix"})
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	defer conn.Close()

	var buf [2048]byte
	var n int

	fmt.Printf("\nlvmetad socket: %#v\n", conn)

	conn.Write([]byte("request = \"hello\"\n"))
	conn.Write([]byte("\n##\n"))

	n, _ = conn.Read(buf[:])
	fmt.Printf("%s\n", buf[:n])

	conn.Write([]byte("request=\"get_global_info\"\ntoken = \"skip\"\n"))
	conn.Write([]byte("\n##\n"))

	n, _ = conn.Read(buf[:])
	fmt.Printf("%s\n", buf[:n])

	conn.Write([]byte("request=\"vg_list\"\ntoken =\"filter:3239235440\"\n"))
	conn.Write([]byte("\n##\n"))

	n, _ = conn.Read(buf[:])
	fmt.Printf("%s\n", buf[:n])

	conn.Write([]byte("request=\"vg_lookup\"\nuuid =\"VIn7Hm-7z8y-AMFJ-6CdJ-6la7-dPd7-h1I6eO\"\ntoken =\"filter:3239235440\"\n"))
	conn.Write([]byte("\n##\n"))

	n, _ = conn.Read(buf[:])
	fmt.Printf("%s\n", buf[:n])
}
