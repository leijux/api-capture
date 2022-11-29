package config

import "flag"

type Config struct {
	URL         string `json:"URL,omitempty"`
	Debug       bool   `json:"Debug,omitempty"`
	BrowserPath string `json:"BrowserPath,omitempty"`
}

func Init() *Config {
	var cfg Config
	flag.StringVar(&cfg.URL, "url", "https://studygolang.com/topics", "")
	flag.BoolVar(&cfg.Debug, "debug", false, "")
	flag.StringVar(&cfg.BrowserPath, "path", "", "浏览器路径")

	flag.Parse()
	return &cfg
}
