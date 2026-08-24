package main

import (
	"fmt"
	"log"
	"os"

	"orbitlink/internal/api"
)

func main() {
	config, err := loadConfig()
	if err != nil {
		log.Fatal(err)
	}
	runtime, err := api.NewRuntime(config.stateRoot)
	if err != nil {
		log.Fatal(err)
	}
	server, err := api.Listen(config.listen, runtime)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Fprintf(os.Stdout, "OrbitLink listening on http://%s\n", server.Address())
	root, stop := shutdownContext()
	defer stop()
	<-root.Done()
	ctx, cancel := closeContext()
	defer cancel()
	if err := server.Close(ctx); err != nil {
		log.Printf("shutdown: %v", err)
	}
}
