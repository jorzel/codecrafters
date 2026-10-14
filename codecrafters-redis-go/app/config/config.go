package config

const (
	MasterRole  = "master"
	ReplicaRole = "replica"
)

type InitSpec struct {
	ReplicaOf  *Node  `json:"replica_of,omitempty"`
	ServerPort int    `json:"port"`
	Dir        string `json:"dir,omitempty"`
	DbFileName string `json:"dbfilename,omitempty"`
}

type Node struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

type Config struct {
	ReplicaOf  *Node  `json:"replica_of"`
	ServerPort int    `json:"port"`
	Dir        string `json:"dir,omitempty"`
	DbFileName string `json:"dbfilename,omitempty"`
}

func NewConfig(initSpec InitSpec) (*Config, error) {
	return &Config{
		ReplicaOf:  initSpec.ReplicaOf,
		ServerPort: initSpec.ServerPort,
		Dir:        initSpec.Dir,
		DbFileName: initSpec.DbFileName,
	}, nil
}
