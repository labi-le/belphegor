# belphegor

Belphegor is a clipboard manager that allows you to share your clipboard with other devices on the network.

## Dependencies

- Go 1.22 (to build a binary) // pls help to create gh actions
- jq
- git
- xclip or xsel (for skufs) or wl-clipboard (linux)
- pbpaste (macos)
- makefile

It works unstable on macos, i don't have this system to fix the code

## Install

```sh
sudo make install
```

If you have Windows, then you will need to install [Make](https://stackoverflow.com/questions/2532234/how-to-run-a-makefile-in-windows) for auto installation, or you can run the command [yourself](Makefile#L27)

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
