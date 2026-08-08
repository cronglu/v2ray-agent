package core

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
	"v2ray-agent/pkg/system"
	"v2ray-agent/pkg/util"
)

var ghMirrors = []string{
	"", // Direct
	"https://gh-proxy.com/",
	"https://ghproxy.net/",
	"https://mirror.ghproxy.com/",
}

const (
	FallbackXrayVersion    = "v25.1.30"
	FallbackSingboxVersion = "v1.11.1"
)

type githubRelease struct {
	TagName string `json:"tag_name"`
}

// FetchLatestVersion fetches latest release tag with fallback
func FetchLatestVersion(repo string, fallback string) string {
	client := &http.Client{Timeout: 6 * time.Second}
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if err == nil {
		req.Header.Set("User-Agent", "xraycli-installer")
		resp, err := client.Do(req)
		if err == nil && resp.StatusCode == http.StatusOK {
			defer resp.Body.Close()
			var rel githubRelease
			if json.NewDecoder(resp.Body).Decode(&rel) == nil && rel.TagName != "" {
				return rel.TagName
			}
		}
	}
	util.PrintWarning(fmt.Sprintf("GitHub API 限制或网络波动，启用稳定保底版本: %s", fallback))
	return fallback
}

// DownloadFile downloads a URL with multi-proxy fallback
func DownloadFile(destPath string, rawURL string) error {
	client := &http.Client{Timeout: 45 * time.Second}

	for _, mirror := range ghMirrors {
		downloadURL := mirror + rawURL
		util.PrintInfo(fmt.Sprintf("正在拉取核心资源: %s", downloadURL))

		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, downloadURL, nil)
		if err != nil {
			continue
		}
		req.Header.Set("User-Agent", "xraycli-downloader")

		resp, err := client.Do(req)
		if err != nil || resp.StatusCode != http.StatusOK {
			continue
		}
		defer resp.Body.Close()

		out, err := os.Create(destPath)
		if err != nil {
			return err
		}
		_, err = io.Copy(out, resp.Body)
		out.Close()
		if err == nil {
			return nil
		}
	}
	return fmt.Errorf("all download sources failed for %s", rawURL)
}

// InstallXray downloads and installs latest Xray-core
func InstallXray(sysInfo *system.SystemInfo) error {
	version := FetchLatestVersion("XTLS/Xray-core", FallbackXrayVersion)
	installDir := "/etc/v2ray-agent/xray"
	_ = os.MkdirAll(installDir, 0755)

	zipName := fmt.Sprintf("Xray-linux-%s.zip", sysInfo.XrayArch)
	rawURL := fmt.Sprintf("https://github.com/XTLS/Xray-core/releases/download/%s/%s", version, zipName)
	zipPath := filepath.Join(installDir, zipName)

	if err := DownloadFile(zipPath, rawURL); err != nil {
		return fmt.Errorf("failed to download Xray: %w", err)
	}
	defer os.Remove(zipPath)

	// Unzip binary
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		if f.Name == "xray" || f.Name == "geoip.dat" || f.Name == "geosite.dat" {
			rc, err := f.Open()
			if err != nil {
				continue
			}
			targetPath := filepath.Join(installDir, f.Name)
			out, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err == nil {
				_, _ = io.Copy(out, rc)
				out.Close()
			}
			rc.Close()
			if f.Name == "xray" {
				_ = os.Chmod(targetPath, 0755)
			}
		}
	}

	util.PrintSuccess(fmt.Sprintf("Xray-core 安装成功 [%s]", version))
	return nil
}

// InstallSingBox downloads and installs latest Sing-box
func InstallSingBox(sysInfo *system.SystemInfo) error {
	version := FetchLatestVersion("SagerNet/sing-box", FallbackSingboxVersion)
	installDir := "/etc/v2ray-agent/sing-box"
	_ = os.MkdirAll(installDir, 0755)

	cleanVer := strings.TrimPrefix(version, "v")
	tarName := fmt.Sprintf("sing-box-%s-%s.tar.gz", cleanVer, sysInfo.SingboxArch)
	rawURL := fmt.Sprintf("https://github.com/SagerNet/sing-box/releases/download/%s/%s", version, tarName)
	tarPath := filepath.Join(installDir, tarName)

	if err := DownloadFile(tarPath, rawURL); err != nil {
		return fmt.Errorf("failed to download sing-box: %w", err)
	}
	defer os.Remove(tarPath)

	// Untar
	f, err := os.Open(tarPath)
	if err != nil {
		return err
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if strings.HasSuffix(header.Name, "sing-box") {
			targetPath := filepath.Join(installDir, "sing-box")
			out, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
			if err == nil {
				_, _ = io.Copy(out, tr)
				out.Close()
			}
			break
		}
	}

	util.PrintSuccess(fmt.Sprintf("Sing-box 核心安装成功 [%s]", version))
	return nil
}
