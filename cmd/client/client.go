/* Simple Test App to figure things out */

package main

import (
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"golang.org/x/net/context"
	"io/ioutil"

	"github.com/sjpotter/infranetes/pkg/common"
)

var (
	ip   = flag.String("ip", "127.0.0.1", "IP To Connect to")
	port = flag.Int("port", 2375, "Port to connect to")
	ca   = flag.String("ca", "/home/spotter/.docker/ca.pem", "CA File location")
)

func main() {
	flag.Parse()

	var opts []grpc.DialOption
	var creds credentials.TransportCredentials
	var sn = "127.0.0.1"

	creds, err := NewClientTLSFromFile(*ca, sn)
	if err != nil {
		fmt.Printf("failed: %v\n", err)
		return
	}
	opts = append(opts, grpc.WithTransportCredentials(creds))

	dial := fmt.Sprintf("%s:%d", *ip, *port)

	conn, err := grpc.Dial(dial, opts...)

	if err != nil {
		fmt.Printf("failed to grpc.Dial: %v\n", err)
		return
	}
	defer conn.Close()
	client := common.NewVMServerClient(conn)

	req := &common.RunCmdRequest{}
	req.Cmd = "ifconfig"
	req.Args = []string{"eth0:0", "10.10.10.10", "netmask", "255.255.255.255"}

	_, err = client.RunCmd(context.Background(), req)
	if err != nil {
		fmt.Printf("RunCmd failed: %v\n", err)
		return
	}
}

// NewClientTLSFromFile constructs a TLS from the input certificate file for client.
func NewClientTLSFromFile(certFile, serverName string) (credentials.TransportCredentials, error) {
	b, err := ioutil.ReadFile(certFile)
	if err != nil {
		return nil, err
	}
	cp := x509.NewCertPool()
	if !cp.AppendCertsFromPEM(b) {
		return nil, fmt.Errorf("credentials: failed to append certificates")
	}
	return credentials.NewTLS(&tls.Config{ServerName: serverName, RootCAs: cp, InsecureSkipVerify: true}), nil
}
