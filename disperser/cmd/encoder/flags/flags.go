package flags

import (
	"github.com/Layr-Labs/eigenda/common"
	"github.com/Layr-Labs/eigenda/common/aws"
	"github.com/Layr-Labs/eigenda/encoding"
	"github.com/Layr-Labs/eigenda/encoding/kzg"
	"github.com/urfave/cli"
)

const (
	FlagPrefix   = "disperser-encoder"
	envVarPrefix = "DISPERSER_ENCODER"
)

var (
	/* Required Flags */
	GrpcPortFlag = cli.StringFlag{
		Name:     common.PrefixFlag(FlagPrefix, "grpc-port"),
		Usage:    "Port at which encoder listens for grpc calls",
		Required: true,
		EnvVar:   common.PrefixEnvVar(envVarPrefix, "GRPC_PORT"),
	}
	/* Optional Flags*/
	EncoderVersionFlag = cli.UintFlag{
		Name:     common.PrefixFlag(FlagPrefix, "encoder-version"),
		Usage:    "Encoder version. Options are 1 and 2.",
		Required: false,
		Value:    1,
		EnvVar:   common.PrefixEnvVar(envVarPrefix, "ENCODER_VERSION"),
	}
	S3BucketNameFlag = cli.StringFlag{
		Name:     common.PrefixFlag(FlagPrefix, "s3-bucket-name"),
		Usage:    "Name of the bucket to retrieve blobs and store encoded chunks",
		Required: false,
		EnvVar:   common.PrefixEnvVar(envVarPrefix, "S3_BUCKET_NAME"),
	}
	MetricsHTTPPort = cli.StringFlag{
		Name:     common.PrefixFlag(FlagPrefix, "metrics-http-port"),
		Usage:    "the http port which the metrics prometheus server is listening",
		Required: false,
		Value:    "9100",
		EnvVar:   common.PrefixEnvVar(envVarPrefix, "METRICS_HTTP_PORT"),
	}
	EnableMetrics = cli.BoolFlag{
		Name:     common.PrefixFlag(FlagPrefix, "enable-metrics"),
		Usage:    "start metrics server",
		Required: true,
		EnvVar:   common.PrefixEnvVar(envVarPrefix, "ENABLE_METRICS"),
	}
	MaxConcurrentRequestsFlag = cli.IntFlag{
		Name:     common.PrefixFlag(FlagPrefix, "max-concurrent-requests"),
		Usage:    "maximum number of concurrent requests",
		Required: false,
		Value:    16,
		EnvVar:   common.PrefixEnvVar(envVarPrefix, "MAX_CONCURRENT_REQUESTS"),
	}
	RequestPoolSizeFlag = cli.IntFlag{
		Name:     common.PrefixFlag(FlagPrefix, "request-pool-size"),
		Usage:    "maximum number of requests in the request pool",
		Required: false,
		Value:    32,
		EnvVar:   common.PrefixEnvVar(envVarPrefix, "REQUEST_POOL_SIZE"),
	}
	RequestQueueSizeFlag = cli.IntFlag{
		Name:     common.PrefixFlag(FlagPrefix, "request-queue-size"),
		Usage:    "maximum number of requests in the request queue",
		Required: false,
		Value:    32,
		EnvVar:   common.PrefixEnvVar(envVarPrefix, "REQUEST_QUEUE_SIZE"),
	}
	EnableGnarkChunkEncodingFlag = cli.BoolFlag{
		Name:     common.PrefixFlag(FlagPrefix, "enable-gnark-chunk-encoding"),
		Usage:    "if true, will produce chunks in Gnark, instead of Gob",
		Required: false,
		EnvVar:   common.PrefixEnvVar(envVarPrefix, "ENABLE_GNARK_CHUNK_ENCODING"),
	}
	GPUEnableFlag = cli.BoolFlag{
		Name:     common.PrefixFlag(FlagPrefix, "gpu-enable"),
		Usage:    "Enable GPU, falls back to CPU if not available",
		Required: false,
		EnvVar:   common.PrefixEnvVar(envVarPrefix, "GPU_ENABLE"),
	}
	BackendFlag = cli.StringFlag{
		Name:     common.PrefixFlag(FlagPrefix, "backend"),
		Usage:    "Backend to use for encoding",
		Required: false,
		Value:    string(encoding.GnarkBackend),
		EnvVar:   common.PrefixEnvVar(envVarPrefix, "BACKEND"),
	}
	PreventReencodingFlag = cli.BoolTFlag{
		Name:     common.PrefixFlag(FlagPrefix, "prevent-reencoding"),
		Usage:    "if true, will prevent reencoding of chunks by checking if the chunk already exists in the chunk store",
		Required: false,
		EnvVar:   common.PrefixEnvVar(envVarPrefix, "PREVENT_REENCODING"),
	}
	PprofHttpPort = cli.StringFlag{
		Name:     common.PrefixFlag(FlagPrefix, "pprof-http-port"),
		Usage:    "the http port which the pprof server is listening",
		Required: false,
		Value:    "6060",
		EnvVar:   common.PrefixEnvVar(envVarPrefix, "PPROF_HTTP_PORT"),
	}
	EnablePprof = cli.BoolFlag{
		Name:     common.PrefixFlag(FlagPrefix, "enable-pprof"),
		Usage:    "start prrof server",
		Required: false,
		EnvVar:   common.PrefixEnvVar(envVarPrefix, "ENABLE_PPROF"),
	}
)

var requiredFlags = []cli.Flag{
	GrpcPortFlag,
}

var optionalFlags = []cli.Flag{
	MetricsHTTPPort,
	EnableMetrics,
	MaxConcurrentRequestsFlag,
	RequestPoolSizeFlag,
	RequestQueueSizeFlag,
	EnableGnarkChunkEncodingFlag,
	EncoderVersionFlag,
	S3BucketNameFlag,
	GPUEnableFlag,
	BackendFlag,
	PreventReencodingFlag,
	PprofHttpPort,
	EnablePprof,
}

// Flags contains the list of configuration options available to the binary.
var Flags []cli.Flag

func init() {
	Flags = append(requiredFlags, optionalFlags...)
	Flags = append(Flags, aws.ClientFlags(envVarPrefix, FlagPrefix)...)
	Flags = append(Flags, kzg.CLIFlags(envVarPrefix)...)
	Flags = append(Flags, common.LoggerCLIFlags(envVarPrefix, FlagPrefix)...)
}
