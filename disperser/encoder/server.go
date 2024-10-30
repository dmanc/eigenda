package encoder

import (
	"context"
	"errors"
	"time"

	pb "github.com/Layr-Labs/eigenda/disperser/api/grpc/encoder"
	"github.com/Layr-Labs/eigenda/encoding"
	"github.com/Layr-Labs/eigensdk-go/logging"
)

type EncoderServer struct {
	pb.UnimplementedEncoderServer
	pb.UnimplementedRSEncoderServer
	pb.UnimplementedKZGProverServer

	GPUEnabled bool

	config  ServerConfig
	logger  logging.Logger
	prover  encoding.Prover
	metrics *Metrics
	close   func()

	// General encoding request pool
	runningRequests chan struct{}
	requestPool     chan struct{}

	// RS encoding request pool
	rsRunningRequests chan struct{}
	rsRequestPool     chan struct{}

	// KZG encoding request pool
	kzgRunningRequests chan struct{}
	kzgRequestPool     chan struct{}
}

func NewEncoderServer(config ServerConfig, logger logging.Logger, prover encoding.Prover, metrics *Metrics) *EncoderServer {
	return &EncoderServer{
		config:  config,
		logger:  logger.With("component", "EncoderServer"),
		prover:  prover,
		metrics: metrics,

		runningRequests: make(chan struct{}, config.MaxConcurrentRequests),
		requestPool:     make(chan struct{}, config.RequestPoolSize),

		rsRunningRequests: make(chan struct{}, config.MaxConcurrentRequests),
		rsRequestPool:     make(chan struct{}, config.RequestPoolSize),

		kzgRunningRequests: make(chan struct{}, config.MaxConcurrentRequests),
		kzgRequestPool:     make(chan struct{}, config.RequestPoolSize),
	}
}

func (s *EncoderServer) EncodeBlob(ctx context.Context, req *pb.EncodeBlobRequest) (*pb.EncodeBlobReply, error) {
	startTime := time.Now()
	select {
	case s.requestPool <- struct{}{}:
	default:
		s.metrics.IncrementRateLimitedBlobRequestNum(len(req.GetData()))
		s.logger.Warn("rate limiting as request pool is full", "requestPoolSize", s.config.RequestPoolSize, "maxConcurrentRequests", s.config.MaxConcurrentRequests)
		return nil, errors.New("too many requests")
	}
	s.runningRequests <- struct{}{}
	defer s.popRequest()

	if ctx.Err() != nil {
		s.metrics.IncrementCanceledBlobRequestNum(len(req.GetData()))
		return nil, ctx.Err()
	}

	s.metrics.ObserveLatency("queuing", time.Since(startTime))
	reply, err := s.handleEncoding(req)
	if err != nil {
		s.metrics.IncrementFailedBlobRequestNum(len(req.GetData()))
	} else {
		s.metrics.IncrementSuccessfulBlobRequestNum(len(req.GetData()))
	}
	s.metrics.ObserveLatency("total", time.Since(startTime))

	return reply, err
}

func (s *EncoderServer) popRequest() {
	<-s.requestPool
	<-s.runningRequests
}

func (s *EncoderServer) handleEncoding(ctx context.Context, req *pb.EncodeBlobRequest) (*pb.EncodeBlobReply, error) {
	begin := time.Now()

	if len(req.Data) == 0 {
		return nil, errors.New("handleEncoding: missing data")
	}

	if req.EncodingParams == nil {
		return nil, errors.New("handleEncoding: missing encoding parameters")
	}

	// Convert to core EncodingParams
	var encodingParams = encoding.EncodingParams{
		ChunkLength: uint64(req.GetEncodingParams().GetChunkLength()),
		NumChunks:   uint64(req.GetEncodingParams().GetNumChunks()),
	}

	commits, chunks, err := s.prover.EncodeAndProve(req.GetData(), encodingParams)
	if err != nil {
		return nil, err
	}

	s.metrics.ObserveLatency("encoding", time.Since(begin))
	begin = time.Now()

	commitData, err := commits.Commitment.Serialize()
	if err != nil {
		return nil, err
	}

	lengthCommitData, err := commits.LengthCommitment.Serialize()
	if err != nil {
		return nil, err
	}

	lengthProofData, err := commits.LengthProof.Serialize()
	if err != nil {
		return nil, err
	}

	var chunksData [][]byte
	var format pb.ChunkEncodingFormat
	if s.config.EnableGnarkChunkEncoding {
		format = pb.ChunkEncodingFormat_GNARK
	} else {
		format = pb.ChunkEncodingFormat_GOB
	}

	for _, chunk := range chunks {
		var chunkSerialized []byte
		if s.config.EnableGnarkChunkEncoding {
			chunkSerialized, err = chunk.SerializeGnark()
		} else {
			chunkSerialized, err = chunk.Serialize()
		}
		if err != nil {
			return nil, err
		}
		// perform an operation
		chunksData = append(chunksData, chunkSerialized)
	}

	s.metrics.ObserveLatency("serialization", time.Since(begin))

	return &pb.EncodeBlobReply{
		Commitment: &pb.BlobCommitment{
			Commitment:       commitData,
			LengthCommitment: lengthCommitData,
			LengthProof:      lengthProofData,
			Length:           uint32(commits.Length),
		},
		Chunks:              chunksData,
		ChunkEncodingFormat: format,
	}, nil
}
