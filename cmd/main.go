package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"
	"time"

	"github.com/go-playground/form/v4"
	"github.com/grandcat/zeroconf"
)

/* --- STRUCT --- */

// A collection of the server information
type HostInfo struct {
	HostName			string
	IPAddr				net.IP
	HWAddr				net.HardwareAddr
	Iface					net.Interface
}

// A collection of the server services
type application struct {
	infoLog				*log.Logger
	errorLog			*log.Logger
	formDecoder		*form.Decoder
	hostInfo			HostInfo
	mDNSSvc				*zeroconf.Server
}

/* --- FILE PATHS --- */

var dvPath string = "configs/devices/saved_devices.json"				// path to saved devices file
var pDVPath string = "configs/devices/requested_devices.json"		// path to pending devices file
var stPath string = "configs/settings/settings.json"						// path to settings file

func main() { 
	port := ":6789"
	
	// INFO LOG
	fileInfo, err := openLogFile("./logs/info.log")
	if err != nil {
			log.Fatal(err)
	}
	infoLog := log.New(fileInfo, "INFO\t", log.Ldate|log.Ltime)

	// ERROR LOG
	fileErr, err := openLogFile("./logs/errors.log")
	if err != nil {
			log.Fatal(err)
	}
	errorLog := log.New(fileErr, "ERROR\t", log.Ldate|log.Ltime|log.Lshortfile)

	// FORM DECODER
	formDecoder := form.NewDecoder()
	
	// HOST INFO
	hostInfo, err := getHostInfo()
	if err != nil {
		log.Fatal(err)
	}

	// CREATE mDNS SERVICE
	var mDNSSvc *zeroconf.Server

	// CREATE AN APP SERVICE
	app := &application{
		infoLog: infoLog,
		errorLog: errorLog,
		formDecoder: formDecoder,
		hostInfo: hostInfo,
		mDNSSvc: mDNSSvc,
	}

	// ERROR CHANNEL USED TO SEND ERRORS BETWEEN THE MAIN FUNCTION AND THE SERVERS
	errCh := make(chan error)

	// CREATE HTTP SERVER
	srv := &http.Server{
		Addr: port,
		ErrorLog: errorLog,
		Handler: app.routes(),
		IdleTimeout: time.Minute * 1,
		ReadTimeout: time.Second * 30,
		WriteTimeout: time.Minute * 1,
	}

	// START HTTP SERVER AND SHUTDOWN IT WHEN THE PROGRAM EXIT
	go func() {
		app.infoLog.Println("Starting HTTP server on", port)
		err := srv.ListenAndServe();
		
		errCh <- err
	}()
	defer func ()  {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second * 5)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			log.Fatal(err)
		}
		app.infoLog.Println("HTTP server stopped")
	}()

	// START mDNS SERVER AND SHUTDOWN IT WHEN THE PROGRAM EXIT
	go func() {
		var err error
		app.mDNSSvc, err = app.advertiseMDNSService()
		if err != nil {
			app.errorLog.Println("Failed to advertise service:", err)
			errCh <- err
			return
		}
		<-time.After(time.Minute * 10)

		for {
			// Periodically re-advertise the service every 10 minutes
			err = app.refreshMDNSService()
			if err != nil {
				app.errorLog.Println("Failed to start mDNS service:", err)
				errCh <- err
				return
			}
			<-time.After(time.Minute * 10)
		}
	}()
	defer func(){
		app.mDNSSvc.Shutdown()
		app.infoLog.Println("mDNS service stopped")
	}()
	
	log.Println("Started the server successfully")
	
	// EXIT THE PROGRAM
	var srvErr error
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	select {
	case <-sig:
		// Exit by user
	case srvErr = <-errCh:
		// Exit by error
	}

	// LOG ERRORS IF ANY
	if srvErr != nil {
		trace := fmt.Sprintf("%s\n%s", err.Error(), debug.Stack())
		app.errorLog.Output(2, trace)
		log.Fatal(err)
	}

	log.Println("Shutting down the server...")
}
