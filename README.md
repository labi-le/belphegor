# belphegor

Belphegor is a clipboard manager that allows you to share your clipboard with other devices on the network\
<img src="logo.png" width="500">
___

### Features

- Encryption
- P2P
- Discovering local nodes
- Image sharing (`wayland` <—> `wayland`, `wayland` <—> `windows`)

___

### Dependencies

- linux:
  - wayland
    * composer implements `ext_data_control_v1`
    * wl-clipboard

- macos:
  * pbpaste (i have no mac for testing)
- windows:
  * nothing (untested windows < 10)

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

- Go 1.25.5
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
  -c, --connect string            Address in ip:port format to connect to the node
      --discover_delay duration   Delay between node discovery (default 5m0s)
  -h, --help                      Show help
      --hidden                    Hide console window (for windows user)
      --keep_alive duration       Interval for checking connections between nodes (default 1m0s)
      --max_peers int             Maximum number of discovered peers (default 5)
      --node_discover             Find local nodes on the network and connect to them (default true)
      --notify                    Enable notifications (default true)
  -p, --port int                  Port to use. Default: random
      --read_timeout duration     Write timeout (default 1m0s)
  -s, --secret string             Key to connect between node (empty=all may connect)
      --verbose                   Verbose logs
  -v, --version                   Show version
      --write_timeout duration    Write timeout (default 1m0s)
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

[] Add x11 support

[] Upnp (?)