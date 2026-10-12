package server

import (
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/vcnt72/http/request"
	"github.com/vcnt72/http/response"
)

func Listen(port string) error {
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%s", port))
	if err != nil {
		return fmt.Errorf("error on listen: %w", err)
	}

	fmt.Printf("Listening on address: %s\n", ln.Addr())

	var wg sync.WaitGroup
	sigChan := make(chan os.Signal, 1)

	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		<-sigChan
		if err := ln.Close(); err != nil {
			log.Println("error on closing listener: ", err)
		}
	}()

	for {
		conn, err := ln.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				log.Println("connection closed")
				break
			}
			panic(fmt.Errorf("error on connection: %w", err))
		}

		wg.Go(func() {
			execRequest(conn)
		})

	}

	wg.Wait()
	return nil
}

func execRequest(conn net.Conn) {
	defer func() {
		fmt.Println("connection close")
		if err := conn.Close(); err != nil {
			log.Println(err)
		}
	}()

	resp := response.NewWriter(conn)
	req, err := request.Read(conn)
	if err != nil {
		panic(err)
	}

	fmt.Println("The request ", req.String())
	resp.Headers().Set("Content-Type", "text/plain")
	resp.WriteHeader(http.StatusOK)

	_, err = resp.Write([]byte("OK1\n"))
	if err != nil {
		log.Println(fmt.Errorf("error on write buffer: %w", err))
	}

	_, err = resp.Write([]byte("OK2\n"))
	if err != nil {
		log.Println(fmt.Errorf("error on write buffer: %w", err))
	}

	defer resp.Flush()

	fmt.Println("Success Write")
}
