package main

import (
	"context"
	"flag"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/signal"

	pb "github.com/UWFCybernauts/hackfortress-exchanger/build/exchange"
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
		unixListener net.Listener
		grpcConn     *grpc.ClientConn
		grpcClient   pb.ExchangeClient
		exiting      = false

		buf          = make([]byte, 512)
		interuptChan = make(chan os.Signal, 1)
		signalExit   = make(chan bool, 1)
	)

	signal.Notify(interuptChan, os.Interrupt)

	// We do this in a goroutine so we can end any waiting listeners in
	// the runtime rather than hanging on them to time out.
	go func() {
		<-interuptChan
		exiting = true
		signalExit <- true

		if unixConn != nil {
			unixConn.Close()
		}
		if unixListener != nil {
			unixListener.Close()
		}
		if grpcConn != nil {
			grpcConn.Close()
		}

		os.Remove(socketPath)
		ilog.Printf("Exiting...\n")
		signalExit <- true
	}()

	// Just remove the old sock dude
	os.Remove(socketPath)

	if newListener, err := net.Listen("unix", socketPath); err == nil {
		unixListener = newListener
		ilog.Printf("Successfully created UNIX Domain Socket Listener at %s\n", socketPath)
	} else {
		ilog.Printf("[CRITICAL] Failed to create UNIX Domain Socket Listener at %s", socketPath)
		elog.Printf(": %v", err)
		ilog.Printf("\nExiting...\n")
		interuptChan <- os.Interrupt
	}

	if newConn, err := grpc.Dial(gRPCServerAddress, grpc.WithInsecure()); err == nil {
		grpcConn = newConn
		grpcClient = pb.NewExchangeClient(newConn)
		ilog.Printf("Successfully created gRPC Exchange Client\n")

		var helloReq = pb.HelloRequest{}
		helloReq.Hello = "connect"
		if rsp, err := grpcClient.GRPCHello(context.Background(), &helloReq); err == nil {
			if rsp.Hello == "connected" {
				ilog.Printf("Successfully connected to gRPC Exchange Server at %s\n", gRPCServerAddress)
			} else {
				ilog.Printf("[CRITICAL] Got improper response from %s on gRPCHello", gRPCServerAddress)
				elog.Printf(": %v", err)
				ilog.Printf("\n")
				interuptChan <- os.Interrupt
			}
		} else {
			ilog.Printf("[CRITICAL] Failed to connect to gRPC Exchange Server at %s", gRPCServerAddress)
			elog.Printf(": %v", err)
			ilog.Printf("\n")
			interuptChan <- os.Interrupt
		}
	} else {
		ilog.Printf("[CRITICAL] Failed to create gRPC Exchange Client")
		elog.Printf(": %v", err)
		ilog.Printf("\n")
		interuptChan <- os.Interrupt
	}

runtime:
	for {
		select {
		case <-signalExit:
			break runtime
		default:
		}

		if unixConn != nil {
			ilog.Printf("Closing UNIX Domain socket connection\n")
			if err := unixConn.Close(); err != nil {
				if !exiting {
					ilog.Print("Failed to close UNIX Domain socket connection")
					elog.Printf(": %v", err)
					ilog.Printf("\n")
				}
				continue
			}
		}

		ilog.Printf("Waiting for UNIX Domain Socket connection\n")
		if newConn, err := unixListener.Accept(); err == nil {
			unixConn = newConn
			ilog.Printf("Accepted UNIX Domain Socket Connection\n")
		} else {
			if !exiting {
				elog.Printf("Failed to accept Connection: %v", err)
			}
			continue
		}

	readloop:
		for nread, err := unixConn.Read(buf); err == nil && nread != 0; nread, err = unixConn.Read(buf) {
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
						break readloop
					}
					ilog.Printf("Wrote scoring data to UNIX Domain Socket\n")
				} else {
					if !exiting {
						ilog.Printf("[CRITICAL] Failed to get remote scoring data from gRPC server")
						elog.Printf(": %v", err)
						ilog.Printf("\n")
						os.Exit(exitFailure)
					}
				}

			case "unregister":
				break readloop

			default:
				ilog.Printf("Unknown Command: '%s'\n", command)
				if _, err := unixConn.Write([]byte("unknown command")); err != nil {
					ilog.Printf("Failed to write data to UNIX Domain Socket")
					elog.Printf(": %v", err)
					ilog.Printf("\n")
					break readloop
				}
			}
		}
	}

	<-signalExit

}

func clearBufN(buf *[]byte, n int) {
	for i := 0; i < n; i++ {
		(*buf)[i] = 0
	}
}
