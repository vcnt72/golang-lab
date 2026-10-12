package main

import (
	"github.com/vcnt72/http/server"
)

func main() {
	if err := server.Listen("8081"); err != nil {
		panic(err)
	}
}
