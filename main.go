package main

import (
	"fmt"
	"log"

	services "github.com/edhollmon/reverse-proxy/internal/config"
	"github.com/edhollmon/reverse-proxy/internal/server"
)

func main() {
	cs := services.NewConfigService()
	if err := cs.LoadDefaultConfig(); err != nil {
		log.Fatal(err)
	}
	fmt.Println("Config loaded: ", cs)

	rp := server.NewReverseProxy()
	rp.Start()

	fmt.Println("Server shutting down")
}
