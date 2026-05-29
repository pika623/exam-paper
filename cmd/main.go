package main

import (
	"exam-paper/internal/app"
	"flag"
	"log"
)

func main() {
	host := flag.String("host", "0.0.0.0", "listen host")
	port := flag.Int("port", 16666, "listen port")
	flag.Parse()

	application, err := app.New()
	if err != nil {
		log.Fatal(err)
	}
	defer application.Close()

	if err := application.Run(app.Options{Host: *host, Port: *port}); err != nil {
		log.Fatal(err)
	}
}
