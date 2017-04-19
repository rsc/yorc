package commands

import (
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"novaforge.bull.com/starlings-janus/janus/config"
	"novaforge.bull.com/starlings-janus/janus/log"
	"novaforge.bull.com/starlings-janus/janus/server"
)

func init() {
	RootCmd.AddCommand(serverCmd)
	setConfig()
	cobra.OnInitialize(initConfig)
}

var cfgFile string

var serverCmd = &cobra.Command{

	Use:          "server",
	Short:        "Perform the server command",
	Long:         `Perform the server command`,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {

		configuration := getConfig()
		shutdownCh := make(chan struct{})
		return server.RunServer(configuration, shutdownCh)
	},
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// enable ability to specify config file via flag
		viper.SetConfigFile(cfgFile)
	}
	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		log.Debugln("Using config file:", viper.ConfigFileUsed())
	} else {
		log.Debugln("Config not found... ")
	}
}

func setConfig() {

	//Flags definition for Janus server
	serverCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file (default is /etc/janus/config.janus.json)")
	serverCmd.PersistentFlags().StringP("working_directory", "w", "", "The name of the working directory of the Janus server")
	serverCmd.PersistentFlags().Int("workers_number", config.DefaultWorkersNumber, "Number of workers in the Janus server. If not set the default value will be used")

	// Flags definition for Janus HTTP REST API
	serverCmd.PersistentFlags().Int("http_port", config.DefaultHTTPPort, "Port number for the Janus HTTP REST API. If omitted or set to '0' then the default port number is used, any positive integer will be used as it, and finally any negative value will let use a random port.")
	serverCmd.PersistentFlags().String("http_address", config.DefaultHTTPAddress, "Listening address for the Janus HTTP REST API.")
	serverCmd.PersistentFlags().String("key_file", "", "File path to a PEM-encoded private key. The key is used to enable SSL for the Janus HTTP REST API. This must be provided along with cert_file. If one of key_file or cert_file is not provided then SSL is disabled.")
	serverCmd.PersistentFlags().String("cert_file", "", "File path to a PEM-encoded certificate. The certificate is used to enable SSL for the Janus HTTP REST API. This must be provided along with key_file. If one of key_file or cert_file is not provided then SSL is disabled.")

	//Flags definition for OpenStack
	serverCmd.PersistentFlags().StringP("os_auth_url", "a", "", "will use the 1.1 *compute api*")
	serverCmd.PersistentFlags().StringP("os_tenant_id", "i", "", "The ID of the tenant")
	serverCmd.PersistentFlags().StringP("os_tenant_name", "n", "", "The name of the tenant")
	serverCmd.PersistentFlags().StringP("os_user_name", "u", "", "The username to authenticate")
	serverCmd.PersistentFlags().StringP("os_password", "p", "", "The password to authenticate")
	serverCmd.PersistentFlags().StringP("os_region", "r", "", "The region name")
	serverCmd.PersistentFlags().StringP("os_prefix", "x", "", "Prefix of the user")
	serverCmd.PersistentFlags().StringP("os_private_network_name", "m", "", "Name of the private network")
	serverCmd.PersistentFlags().StringP("os_public_network_name", "e", "", "Name of the public network")
	serverCmd.PersistentFlags().StringSliceP("os_default_security_groups", "g", make([]string, 0), "Default security groups to be added to created VMs")

	//Flags definition for Consul
	serverCmd.PersistentFlags().StringP("consul_address", "", "", "Address of the HTTP interface for Consul (format: <host>:<port>)")
	serverCmd.PersistentFlags().StringP("consul_token", "t", "", "The token by default")
	serverCmd.PersistentFlags().StringP("consul_datacenter", "d", "", "The datacenter of Consul node")

	serverCmd.PersistentFlags().Int("consul_publisher_max_routines", config.DefaultConsulPubMaxRoutines, "Maximum number of parallelism used to store TOSCA definitions in Consul. If you increase the default value you may need to tweak the ulimit max open files. If set to 0 or less the default value will be used")

	//Flags config fot Kubernetes
	serverCmd.PersistentFlags().StringP("kube_master_ip", "", "", "Address where the HTTP API of Kubernetes is exposed (format: <host>:<port>")

	//Bind Flags for OpenStack
	viper.BindPFlag("os_auth_url", serverCmd.PersistentFlags().Lookup("os_auth_url"))
	viper.BindPFlag("os_tenant_id", serverCmd.PersistentFlags().Lookup("os_tenant_id"))
	viper.BindPFlag("os_tenant_name", serverCmd.PersistentFlags().Lookup("os_tenant_name"))
	viper.BindPFlag("os_user_name", serverCmd.PersistentFlags().Lookup("os_user_name"))
	viper.BindPFlag("os_password", serverCmd.PersistentFlags().Lookup("os_password"))
	viper.BindPFlag("os_region", serverCmd.PersistentFlags().Lookup("os_region"))
	viper.BindPFlag("os_prefix", serverCmd.PersistentFlags().Lookup("os_prefix"))
	viper.BindPFlag("os_private_network_name", serverCmd.PersistentFlags().Lookup("os_private_network_name"))
	viper.BindPFlag("os_public_network_name", serverCmd.PersistentFlags().Lookup("os_public_network_name"))
	viper.BindPFlag("os_default_security_groups", serverCmd.PersistentFlags().Lookup("os_default_security_groups"))
	//Bind flags for Consul
	viper.BindPFlag("consul_address", serverCmd.PersistentFlags().Lookup("consul_address"))
	viper.BindPFlag("consul_token", serverCmd.PersistentFlags().Lookup("consul_token"))
	viper.BindPFlag("consul_datacenter", serverCmd.PersistentFlags().Lookup("consul_datacenter"))

	viper.BindPFlag("consul_publisher_max_routines", serverCmd.PersistentFlags().Lookup("consul_publisher_max_routines"))

	//Bind Flags for Janus server
	viper.BindPFlag("working_directory", serverCmd.PersistentFlags().Lookup("working_directory"))
	viper.BindPFlag("workers_number", serverCmd.PersistentFlags().Lookup("workers_number"))

	//Bind Flags Janus HTTP REST API
	viper.BindPFlag("http_port", serverCmd.PersistentFlags().Lookup("http_port"))
	viper.BindPFlag("http_address", serverCmd.PersistentFlags().Lookup("http_address"))
	viper.BindPFlag("cert_file", serverCmd.PersistentFlags().Lookup("cert_file"))
	viper.BindPFlag("key_file", serverCmd.PersistentFlags().Lookup("key_file"))

	//Environment Variables
	viper.SetEnvPrefix("janus") // will be uppercased automatically - Become "JANUS_"
	viper.AutomaticEnv()        // read in environment variables that match
	viper.BindEnv("working_directory")
	viper.BindEnv("workers_number")
	viper.BindEnv("http_port")
	viper.BindEnv("http_address")
	viper.BindEnv("key_file")
	viper.BindEnv("cert_file")
	viper.BindEnv("os_auth_url", "OS_AUTH_URL")
	viper.BindEnv("os_tenant_id", "OS_TENANT_ID")
	viper.BindEnv("os_tenant_name", "OS_TENANT_NAME")
	viper.BindEnv("os_user_name", "OS_USERNAME")
	viper.BindEnv("os_password", "OS_PASSWORD")
	viper.BindEnv("os_region", "OS_REGION_NAME")
	viper.BindEnv("os_prefix")
	viper.BindEnv("os_private_network_name")
	viper.BindEnv("os_public_network_name")
	viper.BindEnv("os_default_security_groups")
	viper.BindEnv("consul_publisher_max_routines")
	viper.BindEnv("consul_address")

	//Setting Defaults
	viper.SetDefault("working_directory", "work")
	viper.SetDefault("http_port", config.DefaultHTTPPort)
	viper.SetDefault("http_address", config.DefaultHTTPAddress)
	viper.SetDefault("os_prefix", "janus-")
	viper.SetDefault("os_region", "RegionOne")
	viper.SetDefault("os_default_security_groups", make([]string, 0))
	viper.SetDefault("consul_address", "") // Use consul api default
	viper.SetDefault("consul_datacenter", "dc1")
	viper.SetDefault("consul_token", "anonymous")
	viper.SetDefault("consul_publisher_max_routines", config.DefaultConsulPubMaxRoutines)
	viper.SetDefault("workers_number", config.DefaultWorkersNumber)

	//Configuration file directories
	viper.SetConfigName("config.janus") // name of config file (without extension)
	viper.AddConfigPath("/etc/janus/")  // adding home directory as first search path
	viper.AddConfigPath(".")

}

func getConfig() config.Configuration {
	configuration := config.Configuration{}
	configuration.WorkingDirectory = viper.GetString("working_directory")
	configuration.WorkersNumber = viper.GetInt("workers_number")
	configuration.HTTPPort = viper.GetInt("http_port")
	configuration.HTTPAddress = viper.GetString("http_address")
	configuration.CertFile = viper.GetString("cert_file")
	configuration.KeyFile = viper.GetString("key_file")
	configuration.OSAuthURL = viper.GetString("os_auth_url")
	configuration.OSTenantID = viper.GetString("os_tenant_id")
	configuration.OSTenantName = viper.GetString("os_tenant_name")
	configuration.OSUserName = viper.GetString("os_user_name")
	configuration.OSPassword = viper.GetString("os_password")
	configuration.OSRegion = viper.GetString("os_region")
	configuration.ResourcesPrefix = viper.GetString("os_prefix")
	configuration.OSPrivateNetworkName = viper.GetString("os_private_network_name")
	configuration.OSPublicNetworkName = viper.GetString("os_public_network_name")
	configuration.ConsulAddress = viper.GetString("consul_address")
	configuration.ConsulDatacenter = viper.GetString("consul_datacenter")
	configuration.ConsulToken = viper.GetString("consul_token")
	configuration.ConsulPubMaxRoutines = viper.GetInt("consul_publisher_max_routines")
	configuration.OSDefaultSecurityGroups = make([]string, 0)
	configuration.KubeMasterIp = viper.GetString("kube_master_ip")
	for _, secgFlag := range viper.GetStringSlice("os_default_security_groups") {
		// Don't know why but Cobra gives a slice with only one element containing coma separated input flags
		configuration.OSDefaultSecurityGroups = append(configuration.OSDefaultSecurityGroups, strings.Split(secgFlag, ",")...)
	}
	return configuration
}
