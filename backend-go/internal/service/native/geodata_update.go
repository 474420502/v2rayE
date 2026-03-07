package native

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const (
	defaultDLCURL         = "https://github.com/v2fly/domain-list-community/releases/latest/download/dlc.dat"
	defaultDLCChecksumURL = "https://github.com/v2fly/domain-list-community/releases/latest/download/dlc.dat.sha256sum"
	defaultGeoIPURL       = "https://github.com/v2fly/geoip/releases/latest/download/geoip.dat"
	defaultGeoIPChecksum  = "https://github.com/v2fly/geoip/releases/latest/download/geoip.dat.sha256sum"
)

var sha256Pattern = regexp.MustCompile(`(?i)^[a-f0-9]{64}$`)

func (s *Service) UpdateRoutingGeoData() (map[string]interface{}, error) {
	return s.updateAllGeoData(false)
}

func (s *Service) ensureGeoSiteData() (map[string]interface{}, error) {
	return s.updateAllGeoData(true)
}

func (s *Service) updateAllGeoData(onlyMissing bool) (map[string]interface{}, error) {
	updatedAny := false
	result := map[string]interface{}{
		"updatedAt": time.Now().UTC().Format(time.RFC3339),
	}

	geositeUpdated, geositePath, geositeSHA, geositeBytes, err := ensureGeoAsset(
		"geosite.dat",
		defaultDLCURL,
		defaultDLCChecksumURL,
		onlyMissing,
	)
	if err != nil {
		return nil, fmt.Errorf("update geosite.dat failed: %w", err)
	}
	if geositeUpdated {
		updatedAny = true
	}
	result["geositeUpdated"] = geositeUpdated
	result["geositePath"] = geositePath
	result["geositeSha256"] = geositeSHA
	result["geositeBytes"] = geositeBytes

	geoipUpdated, geoipPath, geoipSHA, geoipBytes, err := ensureGeoAsset(
		"geoip.dat",
		defaultGeoIPURL,
		defaultGeoIPChecksum,
		onlyMissing,
	)
	if err != nil {
		return nil, fmt.Errorf("update geoip.dat failed: %w", err)
	}
	if geoipUpdated {
		updatedAny = true
	}
	result["geoipUpdated"] = geoipUpdated
	result["geoipPath"] = geoipPath
	result["geoipSha256"] = geoipSHA
	result["geoipBytes"] = geoipBytes

	result["updated"] = updatedAny
	result["sourceGeosite"] = defaultDLCURL
	result["checksumSourceGeosite"] = defaultDLCChecksumURL
	result["sourceGeoip"] = defaultGeoIPURL
	result["checksumSourceGeoip"] = defaultGeoIPChecksum
	result["hasGeoSite"] = hasGeoSiteAsset()
	result["hasGeoIP"] = hasGeoIPAsset()
	result["geoDataAvailable"] = hasGeoDataAssets()

	return result, nil
}

func ensureGeoAsset(fileName, dataURL, checksumURL string, onlyMissing bool) (bool, string, string, int, error) {
	if onlyMissing && hasGeoAsset(fileName) {
		return false, firstExistingGeoAssetPath(fileName), "", 0, nil
	}

	content, err := downloadBytes(dataURL)
	if err != nil {
		return false, "", "", 0, fmt.Errorf("download data: %w", err)
	}

	expected, err := downloadChecksum(checksumURL)
	if err != nil {
		return false, "", "", 0, fmt.Errorf("download checksum: %w", err)
	}

	actualBytes := sha256.Sum256(content)
	actual := hex.EncodeToString(actualBytes[:])
	if !strings.EqualFold(expected, actual) {
		return false, "", "", 0, fmt.Errorf("checksum mismatch: expected %s got %s", expected, actual)
	}

	targetPath, err := writeGeoDataAsset(fileName, content)
	if err != nil {
		return false, "", "", 0, fmt.Errorf("save %s: %w", fileName, err)
	}

	return true, targetPath, actual, len(content), nil
}

func firstExistingGeoAssetPath(fileName string) string {
	for _, dir := range geoDataSearchDirs() {
		if dir == "" {
			continue
		}
		candidate := filepath.Join(dir, fileName)
		if fileExists(candidate) {
			return candidate
		}
	}
	return ""
}

func downloadBytes(url string) ([]byte, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("http %d", resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 32<<20))
}

func downloadChecksum(url string) (string, error) {
	data, err := downloadBytes(url)
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		if sha256Pattern.MatchString(fields[0]) {
			return strings.ToLower(fields[0]), nil
		}
	}
	return "", fmt.Errorf("no sha256 hash found")
}

func writeGeoDataAsset(fileName string, data []byte) (string, error) {
	var lastErr error
	for _, dir := range geoDataSearchDirs() {
		if dir == "" {
			continue
		}
		if err := os.MkdirAll(dir, 0o755); err != nil {
			lastErr = err
			continue
		}
		target := filepath.Join(dir, fileName)
		tmp := target + ".tmp"
		if err := os.WriteFile(tmp, data, 0o644); err != nil {
			lastErr = err
			continue
		}
		if err := os.Rename(tmp, target); err != nil {
			_ = os.Remove(tmp)
			lastErr = err
			continue
		}
		return target, nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("no writable geodata directory")
	}
	return "", lastErr
}
