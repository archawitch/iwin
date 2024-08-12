package main

import "errors"

var (
	ErrInvalidFormBody = errors.New("http: invalid request form body")
	ErrHardwareAddrNotFound = errors.New("http: hardware address not found")
	ErrMDNSCreation = errors.New("mdns: cannot create a mDNS service")
	ErrMDNSStarting = errors.New("mdns: cannot start a mDNS service")
)