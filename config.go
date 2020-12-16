package main

type Config struct {
	Port     uint16 `json:"port"`
	HTTPPort uint16 `json:"http_port"`
	Sites    []Site `json:"sites"`
}

type Site struct {
	Domain   string `json:"domain"`
	CertPath string `json:"cert_path"`
	KeyPath  string `json:"key_path"`

	Files        *FileServer   `json:"files"`
	ReverseProxy *ReverseProxy `json:"reverse_proxy"`
}

type FileServer struct {
	Root      string `json:"root"`
	UserPaths bool   `json:"user_paths"`
	AutoIndex bool   `json:"auto_index"`
}

type ReverseProxy struct {
	To     []string `json:"to"`
	Domain string   `json:"domain"`
	Yolo   bool     `json:"yolo"`
}
