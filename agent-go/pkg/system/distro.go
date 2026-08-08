package system

import (
	"bufio"
	"os"
	"runtime"
	"strings"
)

// Distro represents Linux distribution
type Distro string

const (
	DistroDebian Distro = "debian"
	DistroUbuntu Distro = "ubuntu"
	DistroCentOS Distro = "centos"
	DistroRHEL   Distro = "rhel"
	DistroAlpine Distro = "alpine"
	DistroArch   Distro = "arch"
	DistroOther  Distro = "other"
)

// SystemInfo holds detected system characteristics
type SystemInfo struct {
	Distro       Distro
	Version      string
	Arch         string
	XrayArch     string
	SingboxArch  string
	PkgInstall   string
	PkgRemove    string
	PkgUpdate    string
}

// DetectSystem scans OS and architecture
func DetectSystem() *SystemInfo {
	info := &SystemInfo{
		Distro: DistroOther,
		Arch:   runtime.GOARCH,
	}

	// Map CPU architecture for Xray & Sing-box releases
	switch runtime.GOARCH {
	case "amd64":
		info.XrayArch = "64"
		info.SingboxArch = "linux-amd64"
	case "arm64":
		info.XrayArch = "arm64-v8a"
		info.SingboxArch = "linux-arm64"
	case "arm":
		info.XrayArch = "arm32-v7a"
		info.SingboxArch = "linux-armv7"
	case "s390x":
		info.XrayArch = "s390x"
		info.SingboxArch = "linux-s390x"
	default:
		info.XrayArch = "64"
		info.SingboxArch = "linux-amd64"
	}

	// Read /etc/os-release
	file, err := os.Open("/etc/os-release")
	if err == nil {
		defer file.Close()
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "ID=") {
				id := strings.ToLower(strings.Trim(strings.TrimPrefix(line, "ID="), `"`))
				switch id {
				case "debian":
					info.Distro = DistroDebian
					info.PkgInstall = "apt-get -y install"
					info.PkgRemove = "apt-get -y autoremove"
					info.PkgUpdate = "apt-get update"
				case "ubuntu":
					info.Distro = DistroUbuntu
					info.PkgInstall = "apt-get -y install"
					info.PkgRemove = "apt-get -y autoremove"
					info.PkgUpdate = "apt-get update"
				case "centos", "rocky", "almalinux", "fedora":
					info.Distro = DistroCentOS
					info.PkgInstall = "yum -y install"
					info.PkgRemove = "yum -y remove"
					info.PkgUpdate = "yum update -y"
				case "alpine":
					info.Distro = DistroAlpine
					info.PkgInstall = "apk add"
					info.PkgRemove = "apk del"
					info.PkgUpdate = "apk update"
				case "arch", "manjaro":
					info.Distro = DistroArch
					info.PkgInstall = "pacman -S --noconfirm"
					info.PkgRemove = "pacman -R --noconfirm"
					info.PkgUpdate = "pacman -Sy"
				}
			}
		}
	}

	return info
}
