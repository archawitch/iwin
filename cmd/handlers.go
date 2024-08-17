package main

import (
	"encoding/base64"
	"errors"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
)

// Handle device addition when the iOS device requested to register to the server
func (app *application) addDevice(w http.ResponseWriter, r *http.Request) {
	// get requested device's information
	device, err := app.getClientInfo(r)
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	// Check if the device is in the saved list or not
	exists, err := checkDeviceExist(device)
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	// If the device is already in the saved list, then reject the request
	if exists {
		response := map[string]any {
			"message": "Already connected!",
		}
		app.response(w, http.StatusBadRequest, response)
		return
	}

	// Add the device to the pending list
	err = savePendingDevice(device)
	if err != nil {
		app.serverError(w, err)
		return
	}

	app.response(w, http.StatusOK, map[string]any {
		"message": "Added your device to the pending list. Waiting for device verification...",
	})

	// Open a url to verify the device
	openURL("http://localhost:6789/devices")

	app.infoLog.Printf("Request for Registration from %s\n", r.RemoteAddr)
}

// Handle device connection when a valid device wanted to connect to the server
func (app *application) connect(w http.ResponseWriter, r *http.Request) {	
	// get requested device's information
	device, err := app.getClientInfo(r)
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	// Check if the device is in the saved list or not
	exists, err := checkDeviceExist(device)
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}
	
	// If not exist, reject the connection
	// Otherwise, generate, save a random token and send its secret to the device for authentication
	if !exists {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	secret := uuid.NewString()	// secret
	expires := time.Now().Add(time.Minute * 5)	// lives for 5 mins
	err = saveToken(Token{
		DeviceId: device.Identifier,
		Secret: secret,
		ExpiredAt: expires,
	})
	if err != nil {
		app.serverError(w, err)
		return
	}

	app.response(w, http.StatusOK, map[string]any {
		"message": "I'm ready, let's connect!",
		"s": base64.StdEncoding.EncodeToString([]byte(secret)),
	})

	app.infoLog.Printf("Connected to %s\n", r.RemoteAddr)
}

// Handle upload request when a valid device uploaded files to the server
func (app *application) upload(w http.ResponseWriter, r *http.Request) {	
	// Authenticate the device with its ID and secret
	found, err := verifyToken(r)
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}
	
	// If not found, then reject the request
	if !found {
		app.response(w, http.StatusBadRequest, map[string]any {"message": "Invalid token"})
		return
	}

	// Validate the request form
	err = r.ParseMultipartForm(50 << 20)	// maximum 50MB
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	// Get the form data
	var url string // URL sent by the client if any
	var text string // text sent by the client if any
	for key, vals := range r.MultipartForm.Value {
		for _, val := range vals {
			if key == "url" && val != "" {
				url = val
			} else if key == "text" && val != "" {
				text = val
			}
		}
	}

	var st settingsData

	// Get the destination folder path to save
	err = readJSONFile(&st, SETTINGS_FILE_PATH)
	if err != nil {
		app.serverError(w, err)
		return
	}

	// Otherwise, we can open the URL if any
	if url != "" {
		openURL(url)
	}

	// Or copy text to clipboard if any
	if text != "" {
		err = copyToClipboard(text)
		if err != nil {
			app.serverError(w, err)
			return
		}

		app.infoLog.Printf("Copied %s to clipboard\n", text)
	}

	// Or save the files if any
	if len(r.MultipartForm.File) > 0 { 
		err = saveFiles(r, st.Dst)
		if err != nil {
			app.serverError(w, err)
			return
		}

		// Open the destination folder
		openFolder(st.Dst)
	}

	app.response(w, http.StatusOK, map[string]any {
		"message": "Received all content successfully",
	})

	app.infoLog.Printf("Uploaded from %s\n", r.RemoteAddr)
}


/* --- DEVICES --- */

// Handle displaying all devices on the devices page
func (app *application) getDevices(w http.ResponseWriter, r *http.Request) {
	var pdDevices DeviceList
	var svDevices DeviceList
	
	// Get pending devices
	err := readJSONFile(&pdDevices, PENDING_DEVICES_FILE_PATH)
	if err != nil {
		app.serverError(w, err)
		return
	}

	// Get saved devices
	err = readJSONFile(&svDevices, DEVICES_FILE_PATH)
	if err != nil {
		app.serverError(w, err)
		return
	}

	deviceData := deviceData{Pending: pdDevices, Saved: svDevices}

	// Render devices.html page
	app.render(w, "devices", deviceData)
}

// Handle device verification when the user clicks to verify the device on the devices page
func (app *application) verifyDevicePost(w http.ResponseWriter, r *http.Request) {
	var form verifyPostForm
	
	err := app.decodePostFormUrlEncoded(r, &form)
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}
	
	err = saveDevice(form.Id, form.Allow)
	if err != nil {
		app.serverError(w, err)
		return
	}

	app.response(w, http.StatusOK, map[string]any {
		"message": "Updated device lists successfully",
	})
}

// Handle removing a specific device on the devices page
func (app *application) removeDevice(w http.ResponseWriter, r *http.Request) {
	var form removeDeviceForm
	
	err := app.decodePostFormUrlEncoded(r, &form)
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	// Remove the device from the list
	err = removeDevice(form.Id)
	if err != nil {
		app.serverError(w, err)
		return
	}
	
	app.response(w, http.StatusOK, map[string]any {
		"message": "Updated device lists successfully",
	})
}


/* --- SETTINGS --- */

// Handle retrieving and displaying the HTTP server settings to the settings page
func (app *application) settings(w http.ResponseWriter, r *http.Request) {
	var st settingsData
	
	// Get settings info
	err := readJSONFile(&st, SETTINGS_FILE_PATH)
	if err != nil {
		app.serverError(w, err)
		return
	}

	// Construct data to parse to the template
	QRCodeData := app.hostInfo.HostName + " " + app.hostInfo.IPAddr.String()
	data := &settingsForm{
		QRCodeData: QRCodeData,
		Dst: st.Dst,
	}

	// Render settings.html page
	app.render(w, "settings", data)
}

// Handle updating the HTTP server settings
func (app *application) settingsPost(w http.ResponseWriter, r *http.Request) {
	var form settingsPostForm
	
	err := app.decodePostFormUrlEncoded(r, &form)
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	// Validate the user input directory
	err = checkDirValid(form.Dst); 
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			app.clientError(w, http.StatusNotFound)
		} else {
			app.clientError(w, http.StatusBadRequest)
		}
		return
	}

	// If ok, then change the saved folder destination
	err = setDstPath(form.Dst)
	if err != nil {
		app.serverError(w, err)
		return
	}

	app.response(w, http.StatusOK, map[string]any {
		"message": "Saved new destination successfully",
	})
}

// Handle when the user clicks refresh to check current IP Address and restart the mDNS service
func (app *application) refresh(w http.ResponseWriter, r *http.Request) {
	err := app.refreshMDNSService()
	if err != nil {
		app.serverError(w, err)
		return
	}
	
	app.response(w, http.StatusOK, map[string]any{"message": "Refreshed successfully"})
}