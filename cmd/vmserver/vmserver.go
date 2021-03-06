package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/golang/glog"

	"github.com/apporbit/infranetes/cmd/vmserver/flags"
	"github.com/apporbit/infranetes/pkg/vmserver"

	// Registered providers
	_ "github.com/apporbit/infranetes/pkg/vmserver/docker"
	_ "github.com/apporbit/infranetes/pkg/vmserver/fake"
	_ "github.com/apporbit/infranetes/pkg/vmserver/systemd"
)

const (
	infranetesVersion = "0.1"
)

func main() {
	flag.Parse()

	if *flags.Version {
		fmt.Printf("infranetes version: %s\n", infranetesVersion)
		os.Exit(0)
	}

	glog.Infof("contprovider = %v", *flags.ContProvider)

	contProvider, err := vmserver.NewContainerProvider(flags.ContProvider)
	if err != nil {
		fmt.Printf("Couldn't create image provider: %v\n", err)
		os.Exit(1)
	}

	server, err := vmserver.NewVMServer(flags.Cert, flags.Key, contProvider)
	if err != nil {
		fmt.Println("Initialize infranetes vm server failed: ", err)
		os.Exit(1)
	}

	fmt.Println(server.Serve(*flags.Listen))
}
