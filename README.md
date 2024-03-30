# belphegor

Belphegor is a clipboard manager that allows you to share your clipboard with other devices on the network.\
<img src="logo.webp" width="800">
___

### Features
- cipher (rsa)
- peer to peer
- discovering local nodes
- image sharing (`wayland` <—> `wayland`, `wayland` <—> `windows`)

___
### Dependencies
- linux:
    * xclip or xsel (for skufs) or wl-clipboard (linux)
- macos:
    * pbpaste
- windows:
    * nothing


### Installation

#### Prebuilt binaries
- https://github.com/labi-le/belphegor/releases

#### Build from source
- Go 1.22
- git
- makefile

```sh
git clone https://github.com/labi-le/belphegor.git
cd belphegor
sudo make install
```

### Usage
```
Usage of belphegor:
  -connect string
        Address in ip:port format to connect to the node
  -debug
        Show debug logs
  -discover_delay duration
        Delay between node discovery
  -help
        Show help
  -node_discover
        Find local nodes on the network and connect to them (default true)
  -port int
        Port to use. Default: random
  -scan_delay duration
        Delay between scan local clipboard
  -version
        Show version
```
### Todo
[x] Create github actions for build binary and running tests
