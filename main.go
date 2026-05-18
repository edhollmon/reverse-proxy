package main

import (
	"fmt"
	"log"

	services "github.com/edhollmon/reverse-proxy/internal/config"
)

func main() {
	cs := services.NewConfigService()
	if err := cs.LoadDefaultConfig(); err != nil {
		log.Fatal(err)
	}
	fmt.Print(cs)
}
