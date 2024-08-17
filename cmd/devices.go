package main

import (
	"net/http"
)

// [devices] Parse a requested data from an iOS device into DeviceInfo type
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

// [devices] Check if the requested iOS device is in the saved list or not
func checkDeviceExist(client DeviceInfo) (exists bool, err error) {
	var devices DeviceList
	
	err = readJSONFile(&devices, DEVICES_FILE_PATH)
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

// [devices] Append new device to the pending list
func savePendingDevice(client DeviceInfo) error {
	var pdDevices DeviceList
	err := readJSONFile(&pdDevices, PENDING_DEVICES_FILE_PATH)
	if err != nil {
		return err
	}
	newDevice := DeviceInfo(client)
	pdDevices.Devices = append(pdDevices.Devices, newDevice)

	err = writeJSONFile(pdDevices, PENDING_DEVICES_FILE_PATH)
	if err != nil {
		return err
	}

	return nil
}

// [devices] Remove a pending device with identifier 'id', and append it on the saved list if allowed
func saveDevice(id string, isAllowed bool) error {
	// Get pending devices
	var pdDeviceList DeviceList
	err := readJSONFile(&pdDeviceList, PENDING_DEVICES_FILE_PATH)
	if err != nil {
		return err
	}

	// Remove the selected device from the pending devices
	var device DeviceInfo
	newPDDeviceList := DeviceList{Devices: []DeviceInfo{}}
	for _, dv := range(pdDeviceList.Devices) {
		if dv.Identifier == id {
			device = dv
			continue
		}
		newPDDeviceList.Devices = append(newPDDeviceList.Devices, dv)
	}

	err = writeJSONFile(newPDDeviceList, PENDING_DEVICES_FILE_PATH)
	if err != nil {
		return err
	}

	// if we allow the device to connect to the server, then add the device to the allowed device list
	// otherwise, we will not add it
	if isAllowed {	
		var deviceList DeviceList
		err = readJSONFile(&deviceList, DEVICES_FILE_PATH)
		if err != nil {
			return err
		}
		deviceList.Devices = append(deviceList.Devices, device)

		err = writeJSONFile(deviceList, DEVICES_FILE_PATH)
		if err != nil {
			return err
		}
	}

	return nil
}

// [devices] Remove a device with identifier 'id' from the saved list
func removeDevice(id string) error {
	// Get the saved devices
	var list DeviceList
	err := readJSONFile(&list, DEVICES_FILE_PATH)
	if err != nil {
		return err
	}

	// Remove the selected device from the list
	newList := DeviceList{Devices: []DeviceInfo{}}
	for _, dv := range(list.Devices) {
		if dv.Identifier != id {
			newList.Devices = append(newList.Devices, dv)
		}
	}

	// Save the saved device list back
	err = writeJSONFile(newList, DEVICES_FILE_PATH)
	if err != nil {
		return err
	}

	return nil
}