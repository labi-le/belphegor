# belphegor

Belphegor is a clipboard manager that allows you to share your clipboard with other devices on the network\
<img src="logo.png" width="500">
___

### Features

- Encryption
- P2P
- Discovering local nodes
- Image sharing
- File sharing

___

### Dependencies

- linux:
  - wayland
    * composer implements `ext_data_control_v1` (sway, kde, etc)
    * wl-clipboard
  - X11
    * XFixes

- macos:
  * nothing
- windows:
  * nothing (untested windows < 10)

### Limitations

- [the app can't work properly on gnome](https://github.com/labi-le/belphegor/issues/119#issuecomment-3749681212)

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
- <details> <summary>windows</summary>

  ```powershell
  irm https://raw.githubusercontent.com/labi-le/belphegor/refs/heads/main/install.ps1 | iex
  ```
  </details>

### Build from source

- Go 1.25.5
- git
- makefile

```sh
git clone https://github.com/labi-le/belphegor.git
cd belphegor
make build
```

### Usage

```
  -c, --connect string            Address in ip:port format to connect to the node
      --discover_delay duration   Delay between node discovery (default 5m0s)
      --file_save_path string     Folder where the files sent to us will be saved (default: Tmp dir)
  -h, --help                      Show help
      --hidden                    Hide console window (for windows user) (default true)
      --install-service           Install systemd-unit and start the service
      --keep_alive duration       Interval for checking connections between nodes (default 1m0s)
      --max_file_size string      Maximum number of discovered peers (default "500MiB")
      --max_peers int             Maximum number of discovered peers (default 5)
      --node_discover             Find local nodes on the network and connect to them (default true)
      --notify                    Enable notifications (default true)
  -p, --port int                  Port to use. Default: random
      --read_timeout duration     Write timeout (default 1m0s)
      --secret string             Key to connect between node (empty=all may connect)
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

  <details> <summary>systemd service</summary>
to install service, you need to have PATH in current ENV, otherwise the notifications will not work

  ```conf
  belphegor --install-service
  ```

#### uninstall

  ```conf
  systemctl --user disable --now belphegor
  ```

  </details>

### Todo

[x] Create github actions for build binary and running tests

[x] Add flake

[x] Add x11 support

[] Upnp (?)