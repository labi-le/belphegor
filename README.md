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
  -bit_size int
        RSA key bit size (default 2048)
  -connect string
        Address in ip:port format to connect to the node
  -debug
        Show debug logs
  -discover_delay duration
        Delay between node discovery (default 5m0s)
  -help
        Show help
  -keep_alive duration
        Interval for checking connections between nodes (default 1m0s)
  -max_peers int
        Maximum number of peers to connect to (default 5)
  -node_discover
        Find local nodes on the network and connect to them (default true)
  -notify
        Enable notifications (default true)
  -port int
        Port to use. Default: random (default 7663)
  -scan_delay duration
        Delay between scan local clipboard (default 2s)
  -version
        Show version
  -write_timeout duration
        Write timeout (default 5s)
```
### Todo
[x] Create github actions for build binary and running tests

[ ] Add the use of unix sockets to track connected nodes

[ ] Debug image sharing for xclip, xsel, pbpaste

[ ] Publish to aur package

[ ] Add disable cipher option
