param (
    [switch]$Uninstall
)

if (-not $Uninstall -and -not $PSBoundParameters.ContainsKey('Uninstall')) {
    Clear-Host
    Write-Host "Belphegor is a clipboard manager that allows you to share your clipboard with other devices on the network" -ForegroundColor Green
    Write-Host ""
    Write-Host "https://github.com/labi-le/belphegor" -ForegroundColor Red
    Write-Host ""
    Write-Host "1. install"
    Write-Host "2. uninstall"
    Write-Host "q. quit"
    Write-Host ""

    $choice = Read-Host "choose an option"

    switch ($choice) {
        "1" { }
        "2" { $Uninstall = $true }
        "Q" { return }
        "q" { return }
        Default { return }
    }
}

$RepoOwner = "labi-le"
$RepoName  = "belphegor"
$AppName   = "belphegor"
$InstallDir = "$env:APPDATA\$AppName"
$ExePath    = "$InstallDir\$AppName.exe"

$RegPath    = "HKCU:\Software\Microsoft\Windows\CurrentVersion\Run"

if ($Uninstall) {
    Write-Host "[-] starting uninstallation process..." -ForegroundColor Yellow

    if (Get-ItemProperty -Path $RegPath -Name $AppName -ErrorAction SilentlyContinue) {
        Write-Host "    removing from registry autostart..."
        Remove-ItemProperty -Path $RegPath -Name $AppName
    } else {
        Write-Host "    autostart entry not found."
    }

    $Proc = Get-Process -Name $AppName -ErrorAction SilentlyContinue
    if ($Proc) {
        Write-Host "    stopping the process..."
        Stop-Process -Name $AppName -Force
        Start-Sleep -Seconds 1
    }

    if (Test-Path $InstallDir) {
        Write-Host "    removing files from $InstallDir..."
        Remove-Item -Path $InstallDir -Recurse -Force
    }

    Write-Host "[+] uninstallation completed." -ForegroundColor Green
    return
}

Write-Host "[*] starting installation of $AppName..." -ForegroundColor Cyan

$Arch = $env:PROCESSOR_ARCHITECTURE
$SearchString = ""

if ($Arch -eq "AMD64") {
    $SearchString = "windows_amd64"
} elseif ($Arch -eq "ARM64") {
    $SearchString = "windows_arm64"
} else {
    Write-Error "architecture $Arch is not supported by this script."
    return
}

Write-Host "    architecture: $Arch (looking for: $SearchString)"

try {
    $ApiUrl = "https://api.github.com/repos/$RepoOwner/$RepoName/releases/latest"
    $Response = Invoke-RestMethod -Uri $ApiUrl -UseBasicParsing

    $Asset = $Response.assets | Where-Object { $_.name -like "*$SearchString*.exe" } | Select-Object -First 1

    if (-not $Asset) {
        throw "no suitable release file found for this architecture"
    }

    $DownloadUrl = $Asset.browser_download_url
    Write-Host "    link found: $DownloadUrl"
} catch {
    Write-Error "error retrieving data from github: $_"
    return
}

if (-not (Test-Path $InstallDir)) {
    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
}

Write-Host "    downloading file..."
try {
    Invoke-WebRequest -Uri $DownloadUrl -OutFile $ExePath
} catch {
    Write-Error "error downloading the file."
    return
}

Write-Host "    adding to registry autostart..."
try {
    Set-ItemProperty -Path $RegPath -Name $AppName -Value $ExePath -Force
} catch {
    Write-Error "could not add to startup registry. check permissions."
}

Write-Host "    launching application..."
Start-Process -FilePath $ExePath

Write-Host "[+] installation completed successfully." -ForegroundColor Green
