package main

import (
	"log"
	"net"
	"time"

	"github.com/go-playground/form/v4"
	"github.com/grandcat/zeroconf"
)

/* --- SERVER --- */

type HostInfo struct {
	HostName			string
	IPAddr				net.IP
	HWAddr				net.HardwareAddr
	Iface					net.Interface
}

type application struct {
	infoLog				*log.Logger
	errorLog			*log.Logger
	formDecoder		*form.Decoder
	hostInfo			HostInfo
	mDNSSvc				*zeroconf.Server
}


/* --- DEVICES --- */

type DeviceInfo struct {
	Name 					string	`json:"name"`
	Identifier		string	`json:"identifier"`
}

type DeviceList struct {
	Devices		[]DeviceInfo	`json:"devices"`
}

type DeviceInfoForm struct {
	Name 					string	`form:"name"`
	Identifier		string	`form:"identifier"`
}


/* --- AUTHENTICATION --- */

type Token struct {
	DeviceId		string
	Secret			string		`json:"secret"`
	ExpiredAt		time.Time	`json:"expired_at"`
}

type TokenList struct {
	Tokens	[]Token	`json:"tokens"`
}


/* --- DEVICES FORMS --- */

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


/* --- SETTINGS FORMS --- */

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