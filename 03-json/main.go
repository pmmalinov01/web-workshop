package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	logger := log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}
	if err := runServer(logger, port); err != nil {
		logger.Printf("Got error: %v", err)
		os.Exit(1)
	}
	logger.Println("Finished clean")
}

func runServer(logger *log.Logger, port string) error {
	// =========================================================================
	// App Starting

	logger.Printf("main : Listening on :%v", port)
	defer logger.Println("main : Completed")

	// =========================================================================
	// Start API Service

	api := http.Server{
		Addr:         ":" + port,
		Handler:      http.HandlerFunc(ListTodos),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}

	// Make a channel to listen for errors coming from the listener. Use a
	// buffered channel so the goroutine can exit if we don't collect this error.
	serverErrors := make(chan error, 1)

	// Start the service listening for requests.
	go func() {
		logger.Printf("main : API listening on %s", api.Addr)
		serverErrors <- api.ListenAndServe()
	}()

	// Make a channel to listen for an interrupt or terminate signal from the OS.
	// Use a buffered channel because the signal package requires it.
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGQUIT, syscall.SIGTERM)

	// =========================================================================
	// Shutdown

	// Blocking main and waiting for shutdown.
	select {
	case err := <-serverErrors:
		logger.Fatalf("error: listening and serving: %s", err)

	case <-shutdown:
		logger.Println("main : Start shutdown")

		// Give outstanding requests a deadline for completion.
		const timeout = 5 * time.Second
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		// Asking listener to shutdown and shed load.
		err := api.Shutdown(ctx)
		if err != nil {
			logger.Printf("main : Graceful shutdown did not complete in %v : %v", timeout, err)
			err = api.Close()
		}

		if err != nil {
			logger.Printf("main : could not stop server gracefully : %v", err)
			return err
		}
	}
	return nil
}

// Todo is an task we wish to remember.
type Todo struct {
	ID        int32  `json:"id"`
	Title     string `json:"title"`
	Completed bool   `json:"completed"`
	URL       string `json:"url"`
	Order     int32  `json:"order"`
}

// ListTodos is an HTTP Handler for returning a list of Products.
func ListTodos(w http.ResponseWriter, r *http.Request) {
	list := []Todo{
		{ID: 42, Title: "MyTask", Completed: false, Order: 1},
	}

	data, err := json.Marshal(list)
	if err != nil {
		log.Println("error marshalling result", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(data); err != nil {
		log.Println("error writing result", err)
	}
}
