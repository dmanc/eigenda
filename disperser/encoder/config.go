package encoder

const (
	Localhost = "0.0.0.0"
)

type ServerConfig struct {
	GrpcPort                 string
	MaxConcurrentRequests    int
	RequestPoolSize          int
	EnableGnarkChunkEncoding bool
	EnableKzg                bool
	EnableRs                 bool
	Backend                  string
	EnableGPU                bool
}
