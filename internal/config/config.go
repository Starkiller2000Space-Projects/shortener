package config

import (
	"flag"
)

type Config struct {	
	RunAddr string  // address and port to run server
	ShowAddr string  // address and port to show for short urls
	IdSize int  // address and port to show for short urls
	ReadTimeout int  // server read timeout in seconds
	WriteTimeout int  // server write timeout in seconds
}


// parse all flags from command line
func LoadConfig() *Config{
	var config Config
    flag.StringVar(&config.RunAddr, "a", ":8080", "address and port to run server")
	flag.StringVar(&config.ShowAddr, "b", "", "address and port to show for short urls")
	flag.IntVar(&config.IdSize, "i", 8, "address and port to show for short urls")
	flag.IntVar(&config.ReadTimeout, "r", 30, "server read timeout in seconds")
	flag.IntVar(&config.WriteTimeout, "w", 30, "server write timeout in seconds")
    flag.Parse()
	return &config
}
