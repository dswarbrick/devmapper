package main

import (
	"fmt"
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

	devList, _ := dm.ListDevices()
	fmt.Printf("devices: %#v\n", devList)
}
