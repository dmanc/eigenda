package client

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Layr-Labs/eigenda/api/clients/codecs"
	"github.com/Layr-Labs/eigenda/api/clients/v2"
	"github.com/Layr-Labs/eigenda/encoding"
	"github.com/Layr-Labs/eigenda/encoding/kzg/prover"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/Layr-Labs/eigenda/api/clients/v2/verification"
	"github.com/Layr-Labs/eigenda/common/geth"
	"github.com/Layr-Labs/eigenda/common/testutils/random"
	"github.com/Layr-Labs/eigenda/core"
	auth "github.com/Layr-Labs/eigenda/core/auth/v2"
	"github.com/Layr-Labs/eigenda/core/eth"
	"github.com/Layr-Labs/eigenda/core/thegraph"
	corev2 "github.com/Layr-Labs/eigenda/core/v2"
	"github.com/Layr-Labs/eigenda/encoding/kzg"
	"github.com/Layr-Labs/eigenda/encoding/kzg/verifier"
	"github.com/Layr-Labs/eigensdk-go/logging"
	"github.com/docker/go-units"
	gethcommon "github.com/ethereum/go-ethereum/common"
)

const (
	SRSPathG1         = "/g1.point"
	SRSPathG2         = "/g2.point"
	SRSPathG2PowerOf2 = "/g2.point.powerOf2"
	SRSPathSRSTables  = "/SRSTables"
)

// TestClient encapsulates the various clients necessary for interacting with EigenDA.
type TestClient struct {
	config                    *TestClientConfig
	logger                    logging.Logger
	disperserClient           clients.DisperserClient
	payloadDisperser          *clients.PayloadDisperser
	relayClient               clients.RelayClient
	relayPayloadRetriever     *clients.RelayPayloadRetriever
	indexedChainState         core.IndexedChainState
	retrievalClient           clients.RetrievalClient
	validatorPayloadRetriever *clients.ValidatorPayloadRetriever
	certVerifier              *verification.CertVerifier
	privateKey                string
	metricsRegistry           *prometheus.Registry
	metrics                   *testClientMetrics
	blobCodec                 codecs.BlobCodec
}

// NewTestClient creates a new TestClient instance.
func NewTestClient(
	logger logging.Logger,
	metrics *testClientMetrics,
	config *TestClientConfig) (*TestClient, error) {

	if config.SRSNumberToLoad == 0 {
		config.SRSNumberToLoad = config.MaxBlobSize / 32
	}

	// Construct the disperser client

	privateKey, err := loadPrivateKey(config.KeyPath, config.KeyVar)
	if err != nil {
		return nil, fmt.Errorf("failed to load private key: %w", err)
	}

	signer, err := auth.NewLocalBlobRequestSigner(privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create signer: %w", err)
	}
	signerAccountId, err := signer.GetAccountID()
	if err != nil {
		return nil, fmt.Errorf("failed to get account ID: %w", err)
	}
	accountId := gethcommon.HexToAddress(signerAccountId)
	logger.Infof("Account ID: %s", accountId.String())

	g1Path, err := config.ResolveSRSPath(SRSPathG1)
	if err != nil {
		return nil, fmt.Errorf("failed to get path to G1 file: %w", err)
	}
	g2Path, err := config.ResolveSRSPath(SRSPathG2)
	if err != nil {
		return nil, fmt.Errorf("failed to get path to G2 file: %w", err)
	}
	g2PowerOf2Path, err := config.ResolveSRSPath(SRSPathG2PowerOf2)
	if err != nil {
		return nil, fmt.Errorf("failed to get path to G2 power of 2 file: %w", err)
	}
	srsTablesPath, err := config.ResolveSRSPath(SRSPathSRSTables)
	if err != nil {
		return nil, fmt.Errorf("failed to get path to SRS tables: %w", err)
	}

	kzgConfig := &kzg.KzgConfig{
		LoadG2Points:    true,
		G1Path:          g1Path,
		G2Path:          g2Path,
		G2PowerOf2Path:  g2PowerOf2Path,
		CacheDir:        srsTablesPath,
		SRSOrder:        config.SRSOrder,
		SRSNumberToLoad: config.SRSNumberToLoad,
		NumWorker:       32,
	}

	kzgProver, err := prover.NewProver(kzgConfig, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create KZG prover: %w", err)
	}

	disperserConfig := &clients.DisperserClientConfig{
		Hostname:          config.DisperserHostname,
		Port:              fmt.Sprintf("%d", config.DisperserPort),
		UseSecureGrpcFlag: true,
	}

	disperserClient, err := clients.NewDisperserClient(disperserConfig, signer, kzgProver, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create disperser client: %w", err)
	}
	err = disperserClient.PopulateAccountant(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to populate accountant: %w", err)
	}

	ethClientConfig := geth.EthClientConfig{
		RPCURLs:          config.EthRPCURLs,
		PrivateKeyString: privateKey,
		NumConfirmations: 0,
		NumRetries:       3,
	}
	ethClient, err := geth.NewMultiHomingClient(ethClientConfig, accountId, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create Ethereum client: %w", err)
	}

	certVerifier, err := verification.NewCertVerifier(
		logger,
		ethClient,
		time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to create cert verifier: %w", err)
	}

	blobCodec, err := codecs.CreateCodec(codecs.PolynomialFormEval, codecs.PayloadEncodingVersion0)
	if err != nil {
		return nil, fmt.Errorf("failed to create blob codec: %w", err)
	}

	payloadClientConfig := clients.GetDefaultPayloadClientConfig()

	payloadDisperserConfig := clients.PayloadDisperserConfig{
		PayloadClientConfig: *payloadClientConfig,
		DisperseBlobTimeout: 1337 * time.Hour, // this suite enforces its own timeouts
	}
	payloadDisperser, err := clients.NewPayloadDisperser(
		logger,
		payloadDisperserConfig,
		blobCodec,
		disperserClient,
		certVerifier)
	if err != nil {
		return nil, fmt.Errorf("failed to create payload disperser: %w", err)
	}

	// Construct the relay client

	ethReader, err := eth.NewReader(
		logger,
		ethClient,
		config.BLSOperatorStateRetrieverAddr,
		config.EigenDAServiceManagerAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to create Ethereum reader: %w", err)
	}

	relayURLS, err := ethReader.GetRelayURLs(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get relay URLs: %w", err)
	}

	// If the relay client attempts to call GetChunks(), it will use this bogus signer.
	// This is expected to be rejected by the relays, since this client is not authorized to call GetChunks().
	rand := random.NewTestRandom()
	keypair, err := rand.BLS()
	if err != nil {
		return nil, fmt.Errorf("failed to generate BLS keypair: %w", err)
	}

	var fakeSigner clients.MessageSigner = func(ctx context.Context, data [32]byte) (*core.Signature, error) {
		return keypair.SignMessage(data), nil
	}

	relayConfig := &clients.RelayClientConfig{
		Sockets:            relayURLS,
		UseSecureGrpcFlag:  true,
		MaxGRPCMessageSize: units.GiB,
		OperatorID:         &core.OperatorID{0},
		MessageSigner:      fakeSigner,
	}
	relayClient, err := clients.NewRelayClient(relayConfig, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create relay client: %w", err)
	}

	blobVerifier, err := verifier.NewVerifier(kzgConfig, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create blob verifier: %w", err)
	}

	relayPayloadRetrieverConfig := &clients.RelayPayloadRetrieverConfig{
		PayloadClientConfig: *payloadClientConfig,
		RelayTimeout:        1337 * time.Hour, // this suite enforces its own timeouts
	}

	relayPayloadRetriever, err := clients.NewRelayPayloadRetriever(
		logger,
		rand.Rand,
		*relayPayloadRetrieverConfig,
		relayClient,
		blobCodec,
		blobVerifier.Srs.G1)
	if err != nil {
		return nil, fmt.Errorf("failed to create relay payload retriever: %w", err)
	}

	// Construct the retrieval client

	chainState := eth.NewChainState(ethReader, ethClient)
	icsConfig := thegraph.Config{
		Endpoint:     config.SubgraphURL,
		PullInterval: 100 * time.Millisecond,
		MaxRetries:   5,
	}
	indexedChainState := thegraph.MakeIndexedChainState(icsConfig, chainState, logger)

	validatorPayloadRetrieverConfig := &clients.ValidatorPayloadRetrieverConfig{
		PayloadClientConfig:           *payloadClientConfig,
		MaxConnectionCount:            20,
		BlsOperatorStateRetrieverAddr: config.BLSOperatorStateRetrieverAddr,
		EigenDAServiceManagerAddr:     config.EigenDAServiceManagerAddr,
		RetrievalTimeout:              1337 * time.Hour, // this suite enforces its own timeouts
	}

	retrievalClient := clients.NewRetrievalClient(
		logger,
		ethReader,
		indexedChainState,
		blobVerifier,
		int(validatorPayloadRetrieverConfig.MaxConnectionCount))

	validatorPayloadRetriever, err := clients.NewValidatorPayloadRetriever(
		logger,
		*validatorPayloadRetrieverConfig,
		blobCodec,
		retrievalClient,
		blobVerifier.Srs.G1)
	if err != nil {
		return nil, fmt.Errorf("failed to create validator payload retriever: %w", err)
	}

	return &TestClient{
		config:                    config,
		logger:                    logger,
		disperserClient:           disperserClient,
		payloadDisperser:          payloadDisperser,
		relayClient:               relayClient,
		relayPayloadRetriever:     relayPayloadRetriever,
		indexedChainState:         indexedChainState,
		retrievalClient:           retrievalClient,
		validatorPayloadRetriever: validatorPayloadRetriever,
		certVerifier:              certVerifier,
		privateKey:                privateKey,
		metricsRegistry:           metrics.registry,
		metrics:                   metrics,
		blobCodec:                 blobCodec,
	}, nil
}

// loadPrivateKey loads the private key from the file/env var specified in the config.
func loadPrivateKey(keyPath string, keyVar string) (string, error) {
	if keyPath != "" {
		privateKeyFile, err := ResolveTildeInPath(keyPath)
		if err != nil {
			return "", fmt.Errorf("failed to resolve tilde in path: %w", err)
		}
		privateKey, err := os.ReadFile(privateKeyFile)
		if err != nil {
			return "", fmt.Errorf("failed to read private key file: %w", err)
		}

		return formatPrivateKey(string(privateKey)), nil
	}

	if keyVar == "" {
		return "", fmt.Errorf("either KeyPath or KeyVar must be set")
	}
	privateKey := os.Getenv(keyVar)
	if privateKey == "" {
		return "", fmt.Errorf("key not found in environment variable %s", keyVar)
	}
	return formatPrivateKey(privateKey), nil
}

// formatPrivateKey formats the private key by removing leading/trailing whitespace and "0x" prefix.
func formatPrivateKey(privateKey string) string {
	privateKey = strings.Trim(privateKey, "\n \t")
	privateKey, _ = strings.CutPrefix(privateKey, "0x")
	return privateKey
}

// GetConfig returns the test client's configuration.
func (c *TestClient) GetConfig() *TestClientConfig {
	return c.config
}

// GetLogger returns the test client's logger.
func (c *TestClient) GetLogger() logging.Logger {
	return c.logger
}

// GetDisperserClient returns the test client's disperser client.
func (c *TestClient) GetDisperserClient() clients.DisperserClient {
	return c.disperserClient
}

// GetPayloadDisperser returns the test client's payload disperser.
func (c *TestClient) GetPayloadDisperser() *clients.PayloadDisperser {
	return c.payloadDisperser
}

// GetRelayClient returns the test client's relay client.
func (c *TestClient) GetRelayClient() clients.RelayClient {
	return c.relayClient
}

// GetRelayPayloadRetriever returns the test client's relay payload retriever.
func (c *TestClient) GetRelayPayloadRetriever() *clients.RelayPayloadRetriever {
	return c.relayPayloadRetriever
}

// GetIndexedChainState returns the test client's indexed chain state.
func (c *TestClient) GetIndexedChainState() core.IndexedChainState {
	return c.indexedChainState
}

// GetRetrievalClient returns the test client's retrieval client.
func (c *TestClient) GetRetrievalClient() clients.RetrievalClient {
	return c.retrievalClient
}

// GetValidatorPayloadRetriever returns the test client's validator payload retriever.
func (c *TestClient) GetValidatorPayloadRetriever() *clients.ValidatorPayloadRetriever {
	return c.validatorPayloadRetriever
}

// GetCertVerifier returns the test client's cert verifier.
func (c *TestClient) GetCertVerifier() *verification.CertVerifier {
	return c.certVerifier
}

// GetPrivateKey returns the test client's private key.
func (c *TestClient) GetPrivateKey() string {
	return c.privateKey
}

// GetMetricsRegistry returns the test client's metrics registry.
func (c *TestClient) GetMetricsRegistry() *prometheus.Registry {
	return c.metricsRegistry
}

// Stop stops the test client.
func (c *TestClient) Stop() {
	c.metrics.stop()
}

// DisperseAndVerify sends a payload to the disperser. Waits until the payload is confirmed and then reads
// it back from the relays and the validators.
func (c *TestClient) DisperseAndVerify(
	ctx context.Context,
	certVerifierAddress string,
	payload []byte,
) error {

	start := time.Now()
	eigenDACert, err := c.DispersePayload(ctx, certVerifierAddress, payload)
	if err != nil {
		return fmt.Errorf("failed to disperse payload: %w", err)
	}
	c.metrics.reportCertificationTime(time.Since(start))

	blobKey, err := eigenDACert.ComputeBlobKey()
	if err != nil {
		return fmt.Errorf("failed to compute blob key: %w", err)
	}

	// read blob from a single relay (assuming success, otherwise will retry)
	payloadBytesFromRelayRetriever, err := c.relayPayloadRetriever.GetPayload(ctx, eigenDACert)
	if err != nil {
		return fmt.Errorf("failed to read blob from relay: %w", err)
	}
	if !bytes.Equal(payload, payloadBytesFromRelayRetriever) {
		return fmt.Errorf("payloads do not match")
	}

	// read blob from a single quorum (assuming success, otherwise will retry)
	payloadBytesFromValidatorRetriever, err := c.validatorPayloadRetriever.GetPayload(ctx, eigenDACert)
	if err != nil {
		return fmt.Errorf("failed to read blob from validators: %w", err)
	}
	if !bytes.Equal(payload, payloadBytesFromValidatorRetriever) {
		return fmt.Errorf("payloads do not match")
	}

	// read blob from ALL relays
	err = c.ReadBlobFromRelays(ctx, *blobKey, eigenDACert.BlobInclusionInfo.BlobCertificate.RelayKeys, payload)
	if err != nil {
		return fmt.Errorf("failed to read blob from relays: %w", err)
	}

	blobHeader := eigenDACert.BlobInclusionInfo.BlobCertificate.BlobHeader
	commitment, err := verification.BlobCommitmentsBindingToInternal(&blobHeader.Commitment)
	if err != nil {
		return fmt.Errorf("failed to convert blob commitments: %w", err)
	}

	// read blob from ALL quorums
	err = c.ReadBlobFromValidators(
		ctx,
		*blobKey,
		blobHeader.Version,
		*commitment,
		eigenDACert.BlobInclusionInfo.BlobCertificate.BlobHeader.QuorumNumbers,
		payload)
	if err != nil {
		return fmt.Errorf("failed to read blob from validators: %w", err)
	}

	return nil
}

// DispersePayload sends a payload to the disperser. Returns the blob key.
func (c *TestClient) DispersePayload(
	ctx context.Context,
	certVerifierAddress string,
	payload []byte,
) (*verification.EigenDACert, error) {
	c.logger.Debugf("Dispersing payload of length %d", len(payload))
	start := time.Now()

	cert, err := c.GetPayloadDisperser().SendPayload(ctx, certVerifierAddress, payload)

	if err != nil {
		return nil, fmt.Errorf("failed to disperse payload: %w", err)
	}
	c.metrics.reportDispersalTime(time.Since(start))

	return cert, nil
}

// ReadBlobFromRelays reads a blob from the relays and compares it to the given payload.
func (c *TestClient) ReadBlobFromRelays(
	ctx context.Context,
	key corev2.BlobKey,
	relayKeys []corev2.RelayKey,
	expectedPayload []byte) error {

	for _, relayID := range relayKeys {
		start := time.Now()

		c.logger.Debugf("Reading blob from relay %d", relayID)
		blobFromRelay, err := c.relayClient.GetBlob(ctx, relayID, key)
		if err != nil {
			return fmt.Errorf("failed to read blob from relay: %w", err)
		}

		c.metrics.reportRelayReadTime(time.Since(start), relayID)

		payloadBytesFromRelay, err := c.blobCodec.DecodeBlob(blobFromRelay)
		if err != nil {
			return fmt.Errorf("failed to decode blob: %w", err)
		}

		if !bytes.Equal(payloadBytesFromRelay, expectedPayload) {
			return fmt.Errorf("payloads do not match")
		}
	}

	return nil
}

// ReadBlobFromValidators reads a blob from the validators and compares it to the given payload.
func (c *TestClient) ReadBlobFromValidators(
	ctx context.Context,
	blobKey corev2.BlobKey,
	blobVersion corev2.BlobVersion,
	blobCommitments encoding.BlobCommitments,
	quorums []core.QuorumID,
	expectedPayload []byte) error {

	currentBlockNumber, err := c.indexedChainState.GetCurrentBlockNumber(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current block number: %w", err)
	}

	for _, quorumID := range quorums {
		c.logger.Debugf("Reading blob from validators for quorum %d", quorumID)

		start := time.Now()

		retrievedBlob, err := c.retrievalClient.GetBlob(
			ctx,
			blobKey,
			blobVersion,
			blobCommitments,
			uint64(currentBlockNumber),
			quorumID)
		if err != nil {
			return fmt.Errorf("failed to read blob from validators: %w", err)
		}

		c.metrics.reportValidatorReadTime(time.Since(start), quorumID)

		retrievedPayload, err := c.blobCodec.DecodeBlob(retrievedBlob)
		if err != nil {
			return fmt.Errorf("failed to decode blob: %w", err)
		}
		if !bytes.Equal(retrievedPayload, expectedPayload) {
			return fmt.Errorf("payloads do not match")
		}
	}

	return nil
}
