# belphegor

Belphegor is a clipboard manager that allows you to share your clipboard with other devices on the network.

## Dependencies

- Go 1.21 (to build a binary)
- xlip or xsel or wl-clipboard (linux)
- pbpaste (macos)

## Install

```sh
sudo make install
```

## Usage
```
Usage:
	belphegor [flags]

Flags:
	-connect string | ip:port to connect to the node (e.g. 192.168.0.12:7777)
	-port int | the node will start on this port (e.g. 7777)
	-node_discover bool | find local nodes on the network and connect to them
	-scan_delay string | delay between scan local clipboard (e.g. 5s)
	-debug | show debug logs
	-version | show version
	-help | show help
```