# belphegor

Belphegor is a clipboard manager that allows you to share your clipboard with other devices on the network.\
<img src="logo.webp" width="800">
___

### Features

- Hybrid encryption (RSA-2048*, AES-256-GCM) (bit size configurable)
- Peer to peer
- Discovering local nodes
- Image sharing (`wayland` <—> `wayland`, `wayland` <—> `windows`)

___

### Dependencies

- linux:
    * xclip or xsel (for skufs) or wl-clipboard (linux)
- macos:
    * pbpaste
- windows:
    * nothing

### Installation

- [Prebuilt binaries](https://github.com/labi-le/belphegor/releases)
- Nix flake
  <details> <summary>as profile</summary>

  ```sh
  nix profile install github:labi-le/belphegor
  ```
  </details>
  <details>
  <summary>import the module</summary>

  ```nix
  {
    # inputs
    belphegor.url = "github:labi-le/belphegor";
    # outputs
    overlay-belphegor = final: prev: {
      belphegor = belphegor.packages.${system}.default;
    };
  
    modules = [
      ({ config, pkgs, ... }: { nixpkgs.overlays = [ overlay-belphegor ]; })
    ];
  
    # add package
    environment.systemPackages = with pkgs; [
      belphegor
    ];
  }
  ```
  </details>

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
      --bit_size int              RSA key bit size (default 2048)
  -c, --connect string            Address in ip:port format to connect to the node
  -d, --debug                     Show debug logs
      --discover_delay duration   Delay between node discovery (default 5m0s)
  -h, --help                      Show help
      --hidden                    Hide console window (for windows user)
      --keep_alive duration       Interval for checking connections between nodes (default 1m0s)
      --max_peers int             Maximum number of peers to connect to (default 5)
      --node_discover             Find local nodes on the network and connect to them (default true)
      --notify                    Enable notifications (default true)
  -p, --port int                  Port to use. Default: random (default 7937)
      --scan_delay duration       Delay between scan local clipboard (default 2s)
  -v, --version                   Show version
      --write_timeout duration    Write timeout (default 5s)
```


### Autostart
  <details> <summary>sway</summary>

  ```conf
  exec belphegor
  ```
  </details>

### Todo

[x] Create github actions for build binary and running tests

[x] Add flake

[ ] Debug image sharing for xclip, xsel, pbpaste

[ ] Use wayland/x11 ipc clients instead of using utilities to monitor the clipboard

[ ] Add disable cipher option (maybe)

