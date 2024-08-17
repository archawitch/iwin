# iWin (Windows)

iWin is a tool for file sharing from iOS devices to Windows PCs. It is built with Go and Swift, and it follows the mDNS protocol and an HTTP-based Client-Server architecture. There are two parts that are used in this project: the first part is here and the other part is [iWin Share](https://github.com/archawitch/iwin-share).

## Getting Started

You can follow these instructions to get a copy of this project and run it on your local machine for development or testing purposes.

### Prerequisites

- Go (Golang) installed on your Windows PC
- Xcode installed on your MAC
- An iOS device

### Installation

1. Clone this repository to your Windows machine.
2. Follow the instructions at [iWin Share](https://github.com/archawitch/iwin-share) to install iWin share on your iOS device.
3. After setting up your iOS device, you can follow these steps to register your device with the Windows PC.
    1. Open a terminal in your cloned folder and run `go run ./cmd`.
    2. Open a new tab in your browser and navigate to _localhost:6789_.
    3. Open the iWin share app on your iOS device and scan the QR code showing in the browser.
    4. A verification popup will appear. To register your device with the PC, click the "allow" button.
    5. Go back to the settings page and paste your desired uploaded folder in the text box. Then, click "save" button.

## Usage

To run this application, you can run `go run ./cmd` in your terminal to start the HTTP server and advertise the mDNS service to the local network. For file sharing from your iOS device, please visit [iWin Share Usage](https://github.com/archawitch/iwin-share#usage).

## Notes

- The iOS device and the Windows PC need to be on the same local network.
- If you want to check if the mDNS service is currently running or not, you can enter `dns-sd -B _iw._tcp` on the terminal.
- If your PC is not running the mDNS service, you can fix this by clicking the "refresh" button on the settings page.
