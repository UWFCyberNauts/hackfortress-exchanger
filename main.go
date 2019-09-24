package main

import (
	"context"
	"flag"
	"io/ioutil"
	"log"
	"net"
	"os"

	pb "github.com/AWildBeard/hackfortress-exchanger/build/exchange"
	"google.golang.org/grpc"
)

const (
	exitSuccess = 0
	exitFailure = 1
)

var (
	socketPath        string
	gRPCServerAddress string
	debug             bool

	ilog *log.Logger
	elog *log.Logger
)

func init() {
	flag.StringVar(&socketPath, "socket-path", "/tmp/hackfortress-exchanger.sock", "Specify a cusotom socket path")
	flag.StringVar(&gRPCServerAddress, "grpc-address", "", "Set the gRPC remote server address")
	flag.BoolVar(&debug, "debug", false, "Enable debug output")
}

func main() {
	flag.Parse()

	if debug {
		elog = log.New(os.Stdout, "", 0)
	} else {
		elog = log.New(ioutil.Discard, "", 0)
	}

	ilog = log.New(os.Stdout, "", 0)

	if len(gRPCServerAddress) == 0 {
		ilog.Printf("You must specify the -grpc-address flag.\n")
		flag.PrintDefaults()
		os.Exit(exitFailure)
	}

	var (
		unixConn     net.Conn
		grpcClient   pb.ExchangeClient
		unixListener net.Listener

		buf = make([]byte, 512)
	)

	// Just remove the old sock dude
	os.Remove(socketPath)

	if newListener, err := net.Listen("unix", socketPath); err == nil {
		unixListener = newListener
		ilog.Printf("Successfully created UNIX Domain Socket unixListener at %s\n", socketPath)
	} else {
		ilog.Printf("Failed to create UNIX Domain Socket unixListener at %s", socketPath)
		elog.Printf(": %v", err)
		ilog.Printf("\nExiting...\n")
		os.Exit(exitFailure)
	}

	if newConn, err := grpc.Dial(gRPCServerAddress, grpc.WithInsecure()); err == nil {
		defer newConn.Close()
		grpcClient = pb.NewExchangeClient(newConn)
		ilog.Printf("Successfully created gRPC Exchange grpcClient connection to %s\n", gRPCServerAddress)
	} else {
		ilog.Printf("Failed to create gRPC Exchange Client connection to %s", gRPCServerAddress)
		elog.Printf(": %v", err)
		ilog.Printf("\n")
		os.Exit(exitFailure)
	}

	for true {
		if unixConn != nil {
			if err := unixConn.Close(); err != nil {
				ilog.Printf("Failed to close UNIX Domain socket connection")
				elog.Printf(": %v", err)
				ilog.Printf("\n")
			}
		}

		ilog.Printf("Waiting for UNIX Domain Socket connection\n")
		if newConn, err := unixListener.Accept(); err == nil {
			unixConn = newConn
			ilog.Printf("Accepted UNIX Domain Socket Connection\n")
		} else {
			elog.Printf("Failed to accept unixConnection: %v", err)
			continue
		}

		for nread, err := unixConn.Read(buf); err == nil; nread, err = unixConn.Read(buf) {
			if nread == 0 {
				break
			}

			var command = string(buf)
			clearBufN(&buf, nread)

			switch command {
			case "get":
				ilog.Printf("'get' command recieved. Retrieving scoring data\n")

				var req = pb.DataRequest{}
				req.Request = "scoring data"

				if rsp, err := grpcClient.GetData(context.Background(), &req); err == nil {
					if _, err := unixConn.Write([]byte(rsp.GetResponse())); err != nil {
						ilog.Printf("Failed to write scoring data to UNIX Domain Socket")
						elog.Printf(": %v", err)
						ilog.Printf("\n")
						os.Exit(exitFailure)
					}
					ilog.Printf("Wrote scoring data to UNIX Domain Socket\n")
				} else {
					ilog.Printf("Failed to get remote scoring data from gRPC server")
					elog.Printf(": %v", err)
					ilog.Printf("\n")
					os.Exit(exitFailure)
				}

			case "unregister":
				ilog.Printf("UNIX Domain Socket Connection closing...\n")
				break

			default:
				ilog.Printf("Unknown Command: '%s'\n", command)
				if _, err := unixConn.Write([]byte("unknown command")); err != nil {
					ilog.Printf("Failed to write data to UNIX Domain Socket")
					elog.Printf(": %v", err)
					ilog.Printf("\nClosing Connection\n")
					break
				}
			}
		}
	}
}

func clearBuf(buf *[]byte) {
	for i := range *buf {
		(*buf)[i] = 0
	}
}

func clearBufN(buf *[]byte, n int) {
	for i := 0; i < n; i++ {
		(*buf)[i] = 0
	}
}
