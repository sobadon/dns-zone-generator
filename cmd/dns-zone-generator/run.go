package main

import (
	"encoding/json"
	"fmt"
	"net/netip"
	"os"
	"path"
	"strings"

	"github.com/pkg/errors"
	"github.com/sobadon/dns-zone-generator/zoneutil"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type SourceJSON struct {
	Hosts []HostJSON `json:"hosts"`
}

type HostJSON struct {
	// FQDN (with dot or no dot)
	// e.g. host1.zone1.example. , host2.zone2.example
	Name     string `json:"name"`
	IPv4Addr string `json:"ipv4_addr"`
	IPv6Addr string `json:"ipv6_addr"`
}

type Source struct {
	Hosts []Host
}

type Host struct {
	Name     string     `json:"name"`
	IPv4Addr netip.Addr `json:"ipv4_addr"`
	IPv6Addr netip.Addr `json:"ipv6_addr"`
}

func run(cmd *cobra.Command, _ []string) error {
	loggerConfig := zap.NewProductionConfig()
	loggerConfig.Encoding = "json"
	loggerConfig.EncoderConfig.EncodeTime = zapcore.RFC3339NanoTimeEncoder
	loggerConfig.DisableStacktrace = true

	// logLevelStr := viper.GetString(logLevelKey)
	logLevelStr := "debug"
	logLevel, err := zap.ParseAtomicLevel(logLevelStr)
	if err != nil {
		return errors.Wrapf(err, "failed to parse log level: %s", logLevelStr)
	}
	loggerConfig.Level = zap.NewAtomicLevelAt(logLevel.Level())
	l, err := loggerConfig.Build()
	if err != nil {
		return err
	}

	sourceFilePath := viper.GetString(sourceKey)
	source, err := loadSourceJSON(sourceFilePath)
	if err != nil {
		return err
	}

	destDir := viper.GetString(destDirKey)

	forwardZones := viper.GetStringSlice(forwardZonesKey)
	reverseZonesStr := viper.GetStringSlice(reverseZonesKey)
	if len(forwardZones) == 0 && len(reverseZonesStr) == 0 {
		return errors.New("no zones specified. please provide at least one zone")
	}

	reverseZones := []netip.Prefix{}
	for _, zone := range reverseZonesStr {
		zoneIPPrefix, err := netip.ParsePrefix(zone)
		if err != nil {
			return errors.Wrapf(err, "failed to parse reverse zone: %s", zone)
		}
		if zoneIPPrefix.Addr().Is4() && zoneIPPrefix.Bits() != 24 {
			return errors.Errorf("invalid reverse zone prefix size: %d. ipv4 prefix only /24 are supported", zoneIPPrefix.Bits())
		}
		if zoneIPPrefix.Addr().Is6() && zoneIPPrefix.Bits() != 32 {
			return errors.Errorf("invalid reverse zone prefix size: %d. ipv6 prefix only /32 are supported", zoneIPPrefix.Bits())
		}
		reverseZones = append(reverseZones, zoneIPPrefix)
	}

	for _, zoneName := range forwardZones {
		zoneText, err := generateForwardZoneText(zoneName, source)
		if err != nil {
			return errors.Wrapf(err, "failed to generate forward zone text for %s", zoneName)
		}
		zoneFileName := zoneutil.BuildZoneFileNameForwardZone(zoneName)
		if err := writeZoneFile(zoneText, destDir, zoneFileName); err != nil {
			return errors.Wrapf(err, "failed to write forward zone file for %s", zoneName)
		}
		l.Info("Forward zone file generated", zap.String("zone", zoneName))
	}

	for _, zoneIPPrefix := range reverseZones {
		zoneText, err := generateReverseZoneText(zoneIPPrefix, source)
		if err != nil {
			return errors.Wrapf(err, "failed to generate reverse zone text for %s", zoneIPPrefix)
		}
		zoneFileName := zoneutil.BuildZoneFileNameReverseZone(zoneIPPrefix)
		if err := writeZoneFile(zoneText, destDir, zoneFileName); err != nil {
			return errors.Wrapf(err, "failed to write reverse zone file for %s", zoneIPPrefix)
		}
		l.Info("Reverse zone file generated", zap.String("zone", zoneFileName))
	}

	return nil
}

func loadSourceJSON(filePath string) (*Source, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to open source file: %s", filePath)
	}
	defer f.Close()

	var sourceJSON SourceJSON
	if err := json.NewDecoder(f).Decode(&sourceJSON); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal source from %s", filePath)
	}
	var source Source
	for _, host := range sourceJSON.Hosts {
		var ipv4Addr, ipv6Addr netip.Addr
		var err error
		if host.IPv4Addr != "" {
			ipv4Addr, err = netip.ParseAddr(host.IPv4Addr)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to parse ipv4 address: %s", host.IPv4Addr)
			}
		}
		if host.IPv6Addr != "" {
			ipv6Addr, err = netip.ParseAddr(host.IPv6Addr)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to parse ipv6 address: %s", host.IPv6Addr)
			}
		}
		source.Hosts = append(source.Hosts, Host{
			Name:     host.Name,
			IPv4Addr: ipv4Addr,
			IPv6Addr: ipv6Addr,
		})
	}
	return &source, nil
}

func generateForwardZoneText(zoneName string, source *Source) (string, error) {
	zoneText := ""
	for _, host := range source.Hosts {
		if strings.HasSuffix(host.Name, zoneName) && host.IPv4Addr.IsValid() {
			zoneText += fmt.Sprintf("%s IN A %s\n", zoneutil.MustWithDot(host.Name), host.IPv4Addr.String())
		}
	}
	return zoneText, nil
}

func generateReverseZoneText(zoneIPPrefix netip.Prefix, source *Source) (string, error) {
	zoneText := ""
	for _, host := range source.Hosts {
		reversedName := ""
		if zoneIPPrefix.Contains(host.IPv4Addr) {
			reversedName = zoneutil.ReverseName4(host.IPv4Addr)
		} else if zoneIPPrefix.Contains(host.IPv6Addr) {
			reversedName = zoneutil.ReverseName6(host.IPv6Addr)
		} else {
			// その IP アドレスは管轄外のゾーン
			continue
		}
		zoneText += fmt.Sprintf("%s IN PTR %s\n", reversedName, zoneutil.MustWithDot(host.Name))
	}
	return zoneText, nil
}

func writeZoneFile(zoneText string, destDir string, zoneFileName string) error {
	zoneFilePath := path.Join(destDir, zoneFileName)
	err := os.WriteFile(zoneFilePath, []byte(zoneText), 0644)
	if err != nil {
		return errors.Wrapf(err, "failed to write zone file: %s", zoneFilePath)
	}
	return nil
}
