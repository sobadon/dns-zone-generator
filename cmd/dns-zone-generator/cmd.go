package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	sourceKey       = "source"
	destDirKey      = "dest-dir"
	forwardZonesKey = "forward-zones"
	reverseZonesKey = "reverse-zones"
)

func main() {
	var rootCmd = &cobra.Command{
		Use: "dns-zone-generator",
		Long: `aaa

example:
  ./dns-zone-generator --dest-dir ./dest --forward-zones zone1.example --reverse-zones 192.0.2.0/24 --reverse-zones 2001:db8::/32 --source temp/hosts.json
`,
		// PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// 	perfListenAddr, err := cmd.Flags().GetString("perf-listen-addr")
		// 	if err != nil {
		// 		return err
		// 	}
		// 	http.DefaultServeMux.Handle("/debug/fgprof", fgprof.Handler())
		// 	go func() {
		// 		fmt.Println(http.ListenAndServe(perfListenAddr, nil))
		// 	}()
		// 	return nil
		// },
		RunE: run,
	}
	// rootCmd.PersistentFlags().String("perf-listen-addr", "127.0.0.1:6060", "pprof listen address")
	// rootCmd.Flags().String(logLevelKey, "debug", "log level")

	rootCmd.Flags().String(sourceKey, "input.json", "input json file")
	rootCmd.Flags().String(destDirKey, "./output", "output directory")
	rootCmd.Flags().StringArray(forwardZonesKey, []string{"zone1.example.", "zone2.example"}, "forward zone(s)")
	// 実装が面倒なので、とりあえず制約
	rootCmd.Flags().StringArray(reverseZonesKey, []string{"192.0.2.0/24", "2001:db8::/32"}, "reverse zone(s). limitation: ipv4 prefix size must be /24, ipv6 prefix size must be /32, /40, /48")

	viper.SetEnvPrefix("DNS_ZONE_GENERATOR")
	viper.BindPFlags(rootCmd.Flags())
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))

	if err := rootCmd.Execute(); err != nil {
		fmt.Printf("%+v\n", err)
		os.Exit(1)
	}
}
