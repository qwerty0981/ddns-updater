/*
Copyright © 2021 Cerras

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package cmd

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/antchfx/xmlquery"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
)

var cfgFile string
var verbose bool

func fatal(msg string) {
	fmt.Println(msg)
	os.Exit(1)
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "ddns-updater",
	Short: "A tool to trigger a DNS update in a Namecheap DDNS entry",
	Run: func(cmd *cobra.Command, args []string) {
		namecheapHost := viper.GetString("namecheap.host")
		namecheapDomain := viper.GetString("namecheap.domain")
		namecheapToken := viper.GetString("namecheap.token")

		if namecheapHost == "" {
			fatal("Namecheap host must be specified! Remember to specify it with -n")
		}

		if namecheapDomain == "" {
			fatal("Namecheap domain must be specified! Remember to specify it with -d")
		}

		if namecheapToken == "" {
			fatal("Namecheap token must be specified! Remember to specify it with -t")
		}

		ipResolvers := viper.GetStringSlice("ipResolvers")
		cacheFile := viper.GetString("cacheFile")

		if verbose {
			fmt.Println("Config info:")
			fmt.Printf("  Namecheap domain: '%s'\n", namecheapDomain)
			fmt.Printf("  Namecheap host: '%s'\n", namecheapHost)
			fmt.Printf("  Namecheap token: '%s'\n", namecheapToken)
			fmt.Printf("  Ip resolvers: '%s'\n", strings.Join(ipResolvers, ","))
			fmt.Printf("  Cache file: '%s'\n", cacheFile)
			fmt.Print("\n\n")
		}

		oldIp := ""

		// First check for the cache file and load the ip if it exists
		if cacheFile != "" {
			if _, err := os.Stat(cacheFile); os.IsNotExist(err) {
				if verbose {
					fmt.Println("No cache file found")
				}
			} else {
				if verbose {
					fmt.Println("IP Cache found! Reading...")
				}

				bytes, err := os.ReadFile(cacheFile)
				if err != nil {
					fatal("Failed to read file: " + err.Error())
				}

				oldIp = strings.Trim(string(bytes), "\n")

				if verbose {
					fmt.Println("Old Ip is: " + oldIp)
				}
			}
		}

		// Next attempt to resolve the current ip using the list of provided ip resolvers
		currentIp := ""
		for _, domain := range ipResolvers {
			ip, err := getIp(domain)
			if err == nil {
				if verbose {
					fmt.Println("Successfully resolved current IP: " + ip)
				}

				currentIp = ip
				break
			}
			fmt.Println(err)
		}

		if currentIp == "" {
			fatal("Failed to resolve IP using any provided ip resolvers!")
		}

		// If the IPs are the same exit
		if oldIp == currentIp {
			fmt.Println("IPs are the same...")
			os.Exit(0)
		}

		err := updateIp(currentIp, namecheapHost, namecheapDomain, namecheapToken)
		if err != nil {
			fatal(err.Error())
		}

		// If there is a configured cache file, save the new ip
		if cacheFile != "" {
			err = os.WriteFile(cacheFile, []byte(currentIp), 0644)
			if err != nil {
				fatal("Failed to write to cache file: " + err.Error())
			}
		}

		fmt.Println("Success!")
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.ddns-updater.yaml)")

	rootCmd.Flags().StringP("cacheFile", "f", "ip-cache", "Filename for the ip cache file. Leave blank to not cache the local ip")
	viper.BindPFlag("cacheFile", rootCmd.Flags().Lookup("cacheFile"))
	rootCmd.Flags().StringSliceP("ipResolvers", "i", []string{"http://icanhazip.com", "http://checkip.amazonaws.com"}, "Domain(s) that will return the ip address of the calling client")
	viper.BindPFlag("ipResolvers", rootCmd.Flags().Lookup("ipResolvers"))

	rootCmd.Flags().StringP("namecheapHost", "n", "", "Host name for the DNS entry (normally the subdomain you are using)")
	viper.BindPFlag("namecheap.host", rootCmd.Flags().Lookup("namecheapHost"))
	rootCmd.Flags().StringP("namecheapDomain", "d", "", "DNS domain you are using (the domain you are paying for)")
	viper.BindPFlag("namecheap.domain", rootCmd.Flags().Lookup("namecheapDomain"))
	rootCmd.Flags().StringP("namecheapToken", "t", "", "Namecheap token used to authorize the request. Check the README for more info")
	viper.BindPFlag("namecheap.token", rootCmd.Flags().Lookup("namecheapToken"))

	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		cobra.CheckErr(err)

		// Search config in the local and home directory for ".ddns-updater"
		viper.AddConfigPath(home)
		viper.AddConfigPath(".")
		viper.SetConfigName(".ddns-updater")
		viper.SetConfigType("yaml")
	}

	viper.SetEnvPrefix("DDNS")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			fatal(err.Error())
		}
	}
}

func getIp(domain string) (string, error) {
	resp, err := http.Get(domain)
	if err != nil {
		return "", errors.New("Failed to make request to get IP: " + err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", errors.New("Non 200 status code returned when attempting to resolve public IP: " + fmt.Sprint(resp.StatusCode))
	}

	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", errors.New("Failed to read response body: " + err.Error())
	}

	return strings.TrimSpace(string(bytes)), nil
}

func updateIp(ip, host, domain, token string) error {
	u, err := url.Parse("https://dynamicdns.park-your-domain.com/update")
	if err != nil {
		panic(1)
	}

	vals := url.Values{
		"host":     {host},
		"domain":   {domain},
		"password": {token},
	}

	u.RawQuery = vals.Encode()

	client := &http.Client{}

	req, err := http.NewRequest("GET", u.String(), nil)

	req.Header.Add("Accept", "application/json")

	// resp, err := http.Get(u.String())
	resp, err := client.Do(req)
	if err != nil {
		return errors.New("Failed to make ddns update request: " + err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.New("Request returned non 200 status code: " + resp.Status)
	}

	doc, err := xmlquery.Parse(resp.Body)

	responseError := xmlquery.FindOne(doc, "//ErrCount")

	numOfErrors, err := strconv.Atoi(responseError.InnerText())
	if err != nil {
		return errors.New("Failed to convert namecheap response error count to integer")
	}

	if numOfErrors > 0 {
		responseErrorText := xmlquery.FindOne(doc, "//Err1").InnerText()
		return errors.New("Namecheap api returned error(s): " + responseErrorText)
	}

	return nil
}
