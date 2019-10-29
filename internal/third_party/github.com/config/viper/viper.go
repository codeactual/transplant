package viper

import (
	"fmt"
	"strconv"
	"strings"

	tp_strings "github.com/codeactual/transplant/internal/third_party/github.com/strings"
	"github.com/spf13/pflag"
	std_viper "github.com/spf13/viper"

	cage_strings "github.com/codeactual/transplant/internal/cage/strings"
)

// MergeConfig supplements the one-way binding provided by viper.BindPFlag(s) which only
// allows reading the bound config values via viper, e.g. GetString, but not through
// variables bound to configs by cobra, e.g. via StringVarP(). This allows reading of
// the bound values from the latter.
//
// Origin (except for "stringSlice" case):
//   https://github.com/spf13/viper/issues/35#issuecomment-71908327
//   https://github.com/xh3b4sd
//
// Changes:
//
// - Add stringSlice support from https://github.com/spf13/viper.
func MergeConfig(fs *pflag.FlagSet, v *std_viper.Viper) (lastErr error) {
	fs.VisitAll(func(f *pflag.Flag) {
		if f.Changed {
			return
		}

		flagValue := f.Value.String()

		switch f.Value.Type() {
		case "bool":
			viperValue := strconv.FormatBool(v.GetBool(f.Name))

			if flagValue != viperValue && viperValue != "" {
				lastErr = f.Value.Set(viperValue)
			}
		case "string":
			viperValue := v.GetString(f.Name)

			if flagValue != viperValue && viperValue != "" {
				lastErr = f.Value.Set(viperValue)
			}

		// Origin:
		//   https://github.com/spf13/viper/blob/6d33b5a963d922d182c91e8a1c88d81fd150cfd4/viper.go#L1060
		//   MIT: https://github.com/spf13/viper/blob/6d33b5a963d922d182c91e8a1c88d81fd150cfd4/LICENSE
		case "stringSlice":
			viperValue := v.GetStringSlice(f.Name)

			s := strings.TrimPrefix(flagValue, "[")
			s = strings.TrimSuffix(s, "]")
			res, _ := tp_strings.ReadAsCSV(s)

			a := cage_strings.NewSet().AddSlice(viperValue)
			b := cage_strings.NewSet().AddSlice(res)

			if !a.Equals(b) && len(viperValue) != 0 {
				lastErr = f.Value.Set(fmt.Sprintf("%v", viperValue)) // write back in expected format
			}

		case "int64", "int32", "int16", "int8", "int":
			viperValue := strconv.FormatInt(int64(v.GetInt(f.Name)), 10)

			if flagValue != viperValue && viperValue != "" {
				lastErr = f.Value.Set(viperValue)
			}
		case "uint64", "uint32", "uint16", "uint8", "uint":
			viperValue := strconv.FormatUint(uint64(v.GetInt(f.Name)), 10)

			if flagValue != viperValue && viperValue != "" {
				lastErr = f.Value.Set(viperValue)
			}
		case "float64":
			viperValue := strconv.FormatFloat(v.GetFloat64(f.Name), 'f', 6, 64)

			if flagValue != viperValue && viperValue != "" {
				lastErr = f.Value.Set(viperValue)
			}
		default:
			panic(fmt.Sprintf("unsupported flag type %s for flag %s", f.Value.Type(), f.Name))
		}
	})
	return lastErr
}
