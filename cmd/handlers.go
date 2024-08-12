package main

import (
	"errors"
	"net/http"
	"os"
)

type DeviceInfo struct {
	Name 			string	`json:"name"`
	Identifier		string	`json:"identifier"`
}

type DeviceList struct {
	Devices		[]DeviceInfo	`json:"devices"`
}

type DeviceInfoForm struct {
	Name 			string	`form:"name"`
	Identifier		string	`form:"identifier"`
}

// Used to add the requested device to the pending list
func (app *application) addDevice(w http.ResponseWriter, r *http.Request) {
	// get requested device's information
	client, err := app.getClientInfo(r)
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	// Check if the device is in the saved list or not
	found, err := checkDevice(client)
	if err != nil {
		app.serverError(w, err)
		return
	}

	// If the device is already in the saved list, then reject the request
	if found {
		response := map[string]any {
			"message": "Already connected!",
		}
		app.response(w, http.StatusBadRequest, response)
		return
	}

	// Add the device to pending list
	var pdDevices DeviceList
	err = readJSONFile(&pdDevices, pDVPath)
	if err != nil {
		app.serverError(w, err)
		return
	}
	newDevice := DeviceInfo(client)
	pdDevices.Devices = append(pdDevices.Devices, newDevice)

	err = writeJSONFile(pdDevices, pDVPath)
	if err != nil {
		app.serverError(w, err)
		return
	}

	app.response(w, http.StatusOK, map[string]any {
		"message": "Added your device to the pending list. Waiting for device verification...",
	})

	// Open a url to verify the device
	openURL("http://localhost:6789/devices")
}

// Used to authenticate the requested iOS device
func (app *application) connect(w http.ResponseWriter, r *http.Request) {	
	// Get requested device info
	client, err := app.getClientInfo(r)
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	// Check if the device is in the saved list or not
	found, err := checkDevice(client)
	if err != nil {
		app.serverError(w, err)
		return
	}
	
	// If not found, reject the connection
	// Otherwise, accept the connection
	if !found {
		app.response(w, http.StatusNotFound, map[string]any {
			"message": "Device not found",
		})
		return
	}

	app.response(w, http.StatusOK, map[string]any {
		"message": "I'm ready, let's connect!",
	})
}

// Used to upload content from the requested iOS device to the PC
func (app *application) upload(w http.ResponseWriter, r *http.Request) {
	var st settingsData

	// Get the destination folder path to save
	err := readJSONFile(&st, stPath)
	if err != nil {
		app.serverError(w, err)
		return
	}

	// Validate the request form
	err = r.ParseMultipartForm(60 << 20)	// maximum 60MB
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	// Get the form data
	var id string	// client's identifier
	var url string // URL sent by the client if any
	var text string // text sent by the client if any
	for key, vals := range r.MultipartForm.Value {
		for _, val := range vals {
			if key == "identifier" {
				id = val
			} else if key == "url" {
				if val != "" {
					url = val
				}
			} else if key == "text" {
				if val != "" {
					text = val
				}
			}
		}
	}

	// If the requestor did not attach its ID with the form, then reject the request
	if id == "" {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	// Check if the device is in saved devices or not
	found, err := checkDevice(DeviceInfo{Name: "", Identifier: id})
	if err != nil {
		app.serverError(w, err)
		return
	}
	
	// If not found, then reject the request
	if !found {
		app.response(w, http.StatusNotFound, map[string]any {"message": "Device not found"})
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
		"message": "Files saved successfully",
	})
}


/* --- AUTHENTICATION --- */

type deviceData struct {
	Pending	DeviceList
	Saved		DeviceList
}

type verifyPostForm struct {
	Id			string 	`form:"id"`
	Allow		bool		`form:"allow"`	
}

type removeDeviceForm struct {
	Id			string 	`form:"id"`
}

// Used to retrieve and display all devices to the devices page
func (app *application) devices(w http.ResponseWriter, r *http.Request) {
	var pdDevices DeviceList
	var svDevices DeviceList
	
	// Get pending devices
	err := readJSONFile(&pdDevices, pDVPath)
	if err != nil {
		app.serverError(w, err)
		return
	}

	// Get saved devices
	err = readJSONFile(&svDevices, dvPath)
	if err != nil {
		app.serverError(w, err)
		return
	}

	deviceData := deviceData{Pending: pdDevices, Saved: svDevices}

	// Render devices.html page
	app.render(w, "devices", deviceData)
}

// Used to verify the device from the devices page
func (app *application) verifyDevicePost(w http.ResponseWriter, r *http.Request) {
	var form verifyPostForm
	
	err := app.decodePostFormUrlEncoded(r, &form)
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}
	
	// Get pending devices
	var pdDeviceList DeviceList
	err = readJSONFile(&pdDeviceList, pDVPath)
	if err != nil {
		app.serverError(w, err)
		return
	}

	// Remove the selected device from the pending devices
	var device DeviceInfo
	newPDDeviceList := &DeviceList{Devices: []DeviceInfo{}}
	for _, dv := range(pdDeviceList.Devices) {
		if dv.Identifier == form.Id {
			device = dv
			continue
		}
		newPDDeviceList.Devices = append(newPDDeviceList.Devices, dv)
	}

	err = writeJSONFile(*newPDDeviceList, pDVPath)
	if err != nil {
		app.serverError(w, err)
		return
	}

	// if we allow the device to connect to the server, then add the device to the allowed device list
	// otherwise, we will not add it
	if form.Allow {	
		var deviceList DeviceList
		err = readJSONFile(&deviceList, dvPath)
		if err != nil {
			app.serverError(w, err)
			return
		}
		deviceList.Devices = append(deviceList.Devices, device)

		err = writeJSONFile(deviceList, dvPath)
		if err != nil {
			app.serverError(w, err)
			return
		}
	}

	app.response(w, http.StatusOK, map[string]any {
		"message": "Updated device lists successfully",
	})
}

// Used to remove the device from the devices page
func (app *application) removeDevice(w http.ResponseWriter, r *http.Request) {
	var form removeDeviceForm
	
	err := app.decodePostFormUrlEncoded(r, &form)
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}
	
	// Get the saved devices
	var deviceList DeviceList
	err = readJSONFile(&deviceList, dvPath)
	if err != nil {
		app.serverError(w, err)
		return
	}

	// Remove the selected device from the list
	newDeviceList := &DeviceList{Devices: []DeviceInfo{}}
	for _, dv := range(deviceList.Devices) {
		if dv.Identifier != form.Id {
			newDeviceList.Devices = append(newDeviceList.Devices, dv)
		}
	}

	// Save the saved device list back
	err = writeJSONFile(*newDeviceList, dvPath)
	if err != nil {
		app.serverError(w, err)
		return
	}
	
	app.response(w, http.StatusOK, map[string]any {
		"message": "Updated device lists successfully",
	})
}


/* --- SETTINGS --- */

type settingsData struct {
	Dst    			string `json:"destination"`
}

type settingsForm struct {
	QRCodeData 	string
	Dst    			string
}

type settingsPostForm struct {
	Dst		string `form:"dst"`
}

// Used to retrieve and display the HTTP server settings to the settings page
func (app *application) settings(w http.ResponseWriter, r *http.Request) {
	var st settingsData
	
	// Get settings info
	err := readJSONFile(&st, stPath)
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

// Used to update the HTTP server settings
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

// Used to check current IP Address and restart the mDNS service
func (app *application) refresh(w http.ResponseWriter, r *http.Request) {
	err := app.refreshMDNSService()
	if err != nil {
		app.serverError(w, err)
		return
	}
	
	app.response(w, http.StatusOK, map[string]any{"message": "Refreshed successfully"})
}