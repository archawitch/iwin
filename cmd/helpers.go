package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/go-playground/form/v4"
	"github.com/grandcat/zeroconf"
)

/* --- SERVER --- */

// Reply an Internal Server (500) response to the client and log the error to errors.log
func (app *application) serverError(w http.ResponseWriter, err error) {
	trace := fmt.Sprintf("%s\n%s", err.Error(), debug.Stack())
	app.errorLog.Output(2, trace)

	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
}

// Reply an error response with specific status code to the client
func (app *application) clientError(w http.ResponseWriter, status int) {
	http.Error(w, http.StatusText(status), status)
}

// Reply a Not Found (404) response to the client
func (app *application) notFound(w http.ResponseWriter) {
	app.clientError(w, http.StatusNotFound)
}

// Reply a JSON response with arbitrary message and status code to the client
func (app *application) response(w http.ResponseWriter, status int, response map[string]any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	json.NewEncoder(w).Encode(response)
}


// Parse an form-urlencoded form data into a specific data type
func (app *application) decodePostFormUrlEncoded(r *http.Request, dst any) error {
	err := r.ParseForm()
	if err != nil {
		return err
	}

	err = app.formDecoder.Decode(dst, r.PostForm)
	if err != nil {
		var invalidDecoderError *form.InvalidDecoderError

		if errors.As(err, &invalidDecoderError) {
			panic(err)
		}

		return err
	}

	return nil
}

// Display a HTML page with a specific data
func (app *application) render(w http.ResponseWriter, page string, data any) {	
	page = fmt.Sprintf("./ui/html/%s.html", page)
	files := []string{
		page,
	}

	ts, err := template.ParseFiles(files...)
	if err != nil {
		app.serverError(w, err)
		return
	}

	err = ts.ExecuteTemplate(w, "base", data)
	if err != nil {
		app.serverError(w, err)
	}
}

// Advertise the mDNS service on port 9876
func (app *application) advertiseMDNSService() (*zeroconf.Server, error) {
	port := 9876

	ipAddr := strings.Join(strings.Split(app.hostInfo.IPAddr.String(), "."), "--")
	instance := app.hostInfo.HostName + "__" + ipAddr

	app.infoLog.Printf("Starting mDNS service on :%d", port)
	srv, err := zeroconf.Register(
		instance, 
		"_iw._tcp",
		"local.",
		port,
		nil,
		[]net.Interface{app.hostInfo.Iface,},
	)
	if err != nil {
		return nil, err
	}
	
	return srv, nil
}

// Refresh the mDNS service
func (app *application) refreshMDNSService() error {
	// Update IP Address if it has changed
	err := app.updateHostInfo()
	if err != nil {
		return err
	}
	app.infoLog.Println("Updated host info successfully")

	app.mDNSSvc.Shutdown()
	app.infoLog.Println("Service stopped, re-advertising...")

	// Wait for a moment before next advertisement
	<-time.After(time.Second * 2)

	// Re-advertise the service
	svc, err := app.advertiseMDNSService()
	if err != nil {
		return err
	}

	app.mDNSSvc = svc
	app.infoLog.Println("Advertised mDNS service successfully, current IP:", app.hostInfo.IPAddr.String())

	return nil
}

// Open a log file located on a given path
func openLogFile(path string) (*os.File, error) {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}

	return file, nil
}

/* --- AUTHENTICATION --- */

// Parse a requested data from an iOS device into DeviceInfo type
func (app *application) getClientInfo(r *http.Request) (DeviceInfo, error) {
	// get requested device's information
	var client DeviceInfoForm
	err := app.decodePostFormUrlEncoded(r, &client)
	if err != nil {
		return DeviceInfo(client), err
	}
	if client.Identifier == "" || client.Name == "" {
		return DeviceInfo(client), ErrInvalidFormBody
	}

	return DeviceInfo(client), nil
}

// Check if the requested iOS device is in the saved list or not
func checkDevice(client DeviceInfo) (bool, error) {
	var devices DeviceList
	
	err := readJSONFile(&devices, dvPath)
	if err != nil {
		return false, err
	}
	
	// Check if the device is existed or not
	for _, device := range devices.Devices {
		// Found the device
		if(device.Identifier == client.Identifier) {
			return true, nil
		}
	}

	return false, nil
}

/* --- SETTINGS --- */

// Used to update the server information (eg. IP Address)
func (app *application) updateHostInfo() error {
	// Update IP Address if it has changed
	hostInfo, err := getHostInfo()
	if err != nil {
		return err
	}
	if !app.hostInfo.IPAddr.Equal(hostInfo.IPAddr) {
		app.hostInfo = hostInfo
	}

	return nil
}

// Check whether a given path is a valid path based on the server
func checkDirValid(path string) error {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return os.ErrNotExist
		}
		
		return err
	}

	return nil
}

// Set the server's uploaded path
func setDstPath(newPath string) error {
	// read saved devices from json file
	var st settingsData
	
	err := readJSONFile(&st, stPath)
	if err != nil {
		return err
	}

	st.Dst = newPath

	// save the json file back
	err = writeJSONFile(st, stPath)
	if err != nil {
		return err
	}

	return nil
}


/* --- HANDLE UPLOADED DATA --- */

// Save uploaded files into a given folder
func saveFiles(r *http.Request, path string) error {
	for _, fs := range r.MultipartForm.File {
		for _, fh := range fs {
			f, err := fh.Open()
			if err != nil {
				return err
			}
			defer f.Close()

			dstF, err := os.Create(filepath.Join(path, fh.Filename))
			if err != nil {
				return err
			}
			defer dstF.Close()

			_, err = io.Copy(dstF, f)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// Open a given folder
func openFolder(path string) {
	cmd := exec.Command(`explorer`, path)
	cmd.Run()
}

// Open a given URL on the browser
func openURL(url string) {
	args := []string{"/c", "start", url}
	exec.Command("cmd", args...).Run()
}

// Copy a given text to the clipboard
func copyToClipboard(text string) error {
	err := clipboard.WriteAll(text)
	if err != nil {
		return err
	}

	return nil
}
 

/* --- MISCELLANEOUS --- */

// Retrieve current information of the server as HostInfo type
func getHostInfo() (HostInfo, error) {
	var hostInfo HostInfo

	// get hostname
	hostname, err := os.Hostname()
	if err != nil {
		return hostInfo, err
	}

	// get IP address
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return hostInfo, err
	}
	defer conn.Close()

	ipAddr := conn.LocalAddr().(*net.UDPAddr).IP

	// get MAC address
	var hostIface net.Interface
	var hwAddr net.HardwareAddr
	ifaces, err := net.Interfaces()
	if err != nil {
		return hostInfo, err
	}

	for _, iface := range ifaces {
		addrs, err := iface.Addrs()
		if err != nil {
			return hostInfo, err
		}

		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
				case *net.IPNet:
					ip = v.IP
				case *net.IPAddr:
					ip = v.IP
			}

			if ipAddr.Equal(ip) {
				hostIface = iface
				hwAddr = iface.HardwareAddr
				break
			}
		}
	}

	if hwAddr == nil {
		return hostInfo, ErrHardwareAddrNotFound
	}

	hostInfo.HostName = hostname
	hostInfo.IPAddr = ipAddr
	hostInfo.HWAddr = hwAddr
	hostInfo.Iface = hostIface

	return hostInfo, nil
}

// Parse a JSON located on a given path into a specific data type
func readJSONFile(data any, path string) error {
	file, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	err = json.Unmarshal(file, data)
	if err != nil {
		return err
	}

	return nil
}

// Save data to a specific JSON file
func writeJSONFile(data any, path string) error {
	updatedFile, err := json.MarshalIndent(data, "", " ")
	if err != nil {
		return err
	}

	err = os.WriteFile(path, updatedFile, 0644)
	if err != nil {
		return err
	}

	return nil
}
