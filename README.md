# belphegor

Belphegor is a clipboard manager that allows you to share your clipboard with other devices on the network.
___

# How it work
![p2p](logo.webp =100*20)

___
### Dependencies
- linux:
    * xclip or xsel (for skufs) or wl-clipboard (linux)
- macos:
    * pbpaste
- windows:
    * nothing


### Installation
#### Build from source
- Go 1.22 (to build a binary) // pls help to create gh actions
- jq
- git
- makefile
```sh
sudo make install
```

#### Prebuilt binaries
- https://github.com/labi-le/belphegor/releases


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
## Todo
[ ] Create github actions for build binary and running tests
