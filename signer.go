package x402

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
	"github.com/tyler-smith/go-bip32"
	"github.com/tyler-smith/go-bip39"
)

// PaymentSigner signs x402 payment authorizations
type PaymentSigner interface {
	// SignPayment signs a payment authorization for the given requirement
	SignPayment(ctx context.Context, req PaymentRequirement) (*PaymentPayload, error)

	// GetAddress returns the signer's address
	GetAddress() string

	// SupportsNetwork returns true if the signer supports the given network
	SupportsNetwork(network string) bool

	// HasAsset returns true if the signer has the given asset on the network
	HasAsset(asset, network string) bool
}

// PrivateKeySigner signs with a raw private key
type PrivateKeySigner struct {
	privateKey *ecdsa.PrivateKey
	address    common.Address
}

// NewPrivateKeySigner creates a signer from a hex-encoded private key
func NewPrivateKeySigner(privateKeyHex string) (*PrivateKeySigner, error) {
	// Remove 0x prefix if present
	privateKeyHex = strings.TrimPrefix(privateKeyHex, "0x")

	privateKeyBytes, err := hex.DecodeString(privateKeyHex)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidPrivateKey, err)
	}

	privateKey, err := crypto.ToECDSA(privateKeyBytes)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidPrivateKey, err)
	}

	address := crypto.PubkeyToAddress(privateKey.PublicKey)

	return &PrivateKeySigner{
		privateKey: privateKey,
		address:    address,
	}, nil
}

func (s *PrivateKeySigner) GetAddress() string {
	return s.address.Hex()
}

func (s *PrivateKeySigner) SupportsNetwork(network string) bool {
	_, ok := NetworkChainIDs[network]
	return ok
}

func (s *PrivateKeySigner) HasAsset(asset, network string) bool {
	// This would normally check blockchain balance
	// For now, assume we have the asset
	return true
}

func (s *PrivateKeySigner) SignPayment(ctx context.Context, req PaymentRequirement) (*PaymentPayload, error) {
	// Generate nonce
	nonceBytes := crypto.Keccak256([]byte(fmt.Sprintf("%d-%s-%s",
		time.Now().UnixNano(), req.Resource, s.address.Hex())))
	nonce := "0x" + hex.EncodeToString(nonceBytes)

	// Create time window with some buffer for clock skew
	// Make validAfter 5 seconds in the past to account for clock differences
	validAfter := time.Now().Add(-5 * time.Second).Unix()
	validBefore := time.Now().Add(time.Duration(req.MaxTimeoutSeconds) * time.Second).Unix()

	// Create EIP-712 typed data
	chainID := GetChainID(req.Network)

	// Parse value
	value := new(big.Int)
	value.SetString(req.MaxAmountRequired, 10)

	typedData := apitypes.TypedData{
		Types: apitypes.Types{
			"EIP712Domain": []apitypes.Type{
				{Name: "name", Type: "string"},
				{Name: "version", Type: "string"},
				{Name: "chainId", Type: "uint256"},
				{Name: "verifyingContract", Type: "address"},
			},
			"TransferWithAuthorization": []apitypes.Type{
				{Name: "from", Type: "address"},
				{Name: "to", Type: "address"},
				{Name: "value", Type: "uint256"},
				{Name: "validAfter", Type: "uint256"},
				{Name: "validBefore", Type: "uint256"},
				{Name: "nonce", Type: "bytes32"},
			},
		},
		PrimaryType: "TransferWithAuthorization",
		Domain: apitypes.TypedDataDomain{
			Name:              req.Extra["name"],
			Version:           req.Extra["version"],
			ChainId:           (*math.HexOrDecimal256)(chainID),
			VerifyingContract: req.Asset,
		},
		Message: apitypes.TypedDataMessage{
			"from":        s.address.Hex(),
			"to":          common.HexToAddress(req.PayTo).Hex(),
			"value":       (*math.HexOrDecimal256)(value),
			"validAfter":  (*math.HexOrDecimal256)(big.NewInt(validAfter)),
			"validBefore": (*math.HexOrDecimal256)(big.NewInt(validBefore)),
			"nonce":       nonce,
		},
	}

	// Sign the typed data
	sigHash, _, err := apitypes.TypedDataAndHash(typedData)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrSigningFailed, err)
	}

	signature, err := crypto.Sign(sigHash, s.privateKey)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrSigningFailed, err)
	}

	// Adjust V value for Ethereum signature standard
	signature[64] += 27

	return &PaymentPayload{
		X402Version: 1,
		Scheme:      req.Scheme,
		Network:     req.Network,
		Payload: PaymentPayloadData{
			Signature: "0x" + hex.EncodeToString(signature),
			Authorization: PaymentAuthorization{
				From:        s.address.Hex(),
				To:          req.PayTo,
				Value:       req.MaxAmountRequired,
				ValidAfter:  fmt.Sprintf("%d", validAfter),
				ValidBefore: fmt.Sprintf("%d", validBefore),
				Nonce:       nonce,
			},
		},
	}, nil
}

// derivePrivateKey derives a private key from a seed using BIP-32 HD derivation
func derivePrivateKey(seed []byte, path accounts.DerivationPath) (*ecdsa.PrivateKey, error) {
	// Create master key from seed
	masterKey, err := bip32.NewMasterKey(seed)
	if err != nil {
		return nil, fmt.Errorf("failed to create master key: %w", err)
	}

	// Follow the derivation path
	key := masterKey
	for _, n := range path {
		key, err = key.NewChildKey(n)
		if err != nil {
			return nil, fmt.Errorf("failed to derive child key: %w", err)
		}
	}

	// Convert to ECDSA private key
	privateKey, err := crypto.ToECDSA(key.Key)
	if err != nil {
		return nil, fmt.Errorf("failed to convert to ECDSA key: %w", err)
	}

	return privateKey, nil
}

// MnemonicSigner signs with a key derived from a mnemonic phrase
type MnemonicSigner struct {
	*PrivateKeySigner
}

// NewMnemonicSigner creates a signer from a BIP-39 mnemonic phrase
func NewMnemonicSigner(mnemonic string, derivationPath string) (*MnemonicSigner, error) {
	if !bip39.IsMnemonicValid(mnemonic) {
		return nil, ErrInvalidMnemonic
	}

	if derivationPath == "" {
		derivationPath = "m/44'/60'/0'/0/0" // Default Ethereum path
	}

	// Parse the derivation path using go-ethereum's parser
	path, err := accounts.ParseDerivationPath(derivationPath)
	if err != nil {
		return nil, fmt.Errorf("invalid derivation path: %w", err)
	}

	// Create seed from mnemonic
	seed := bip39.NewSeed(mnemonic, "")

	// Use BIP-32 HD derivation with go-ethereum's path parser
	privateKey, err := derivePrivateKey(seed, path)
	if err != nil {
		return nil, fmt.Errorf("failed to derive private key: %w", err)
	}

	address := crypto.PubkeyToAddress(privateKey.PublicKey)

	return &MnemonicSigner{
		PrivateKeySigner: &PrivateKeySigner{
			privateKey: privateKey,
			address:    address,
		},
	}, nil
}

// KeystoreSigner signs with a key from an encrypted keystore file
type KeystoreSigner struct {
	*PrivateKeySigner
}

// NewKeystoreSigner creates a signer from an encrypted keystore JSON
func NewKeystoreSigner(keystoreJSON []byte, password string) (*KeystoreSigner, error) {
	key, err := keystore.DecryptKey(keystoreJSON, password)
	if err != nil {
		if err == keystore.ErrDecrypt {
			return nil, ErrWrongPassword
		}
		return nil, fmt.Errorf("%w: %v", ErrInvalidKeystore, err)
	}

	return &KeystoreSigner{
		PrivateKeySigner: &PrivateKeySigner{
			privateKey: key.PrivateKey,
			address:    key.Address,
		},
	}, nil
}

// MockSigner is a test signer that generates fake signatures
type MockSigner struct {
	address string
}

// NewMockSigner creates a mock signer for testing
func NewMockSigner(address string) *MockSigner {
	if !strings.HasPrefix(address, "0x") {
		address = "0x" + address
	}
	return &MockSigner{address: address}
}

func (m *MockSigner) GetAddress() string {
	return m.address
}

func (m *MockSigner) SupportsNetwork(network string) bool {
	return true
}

func (m *MockSigner) HasAsset(asset, network string) bool {
	return true
}

func (m *MockSigner) SignPayment(ctx context.Context, req PaymentRequirement) (*PaymentPayload, error) {
	// Generate deterministic fake signature for testing
	fakeSignature := strings.Repeat("00", 65)

	return &PaymentPayload{
		X402Version: 1,
		Scheme:      req.Scheme,
		Network:     req.Network,
		Payload: PaymentPayloadData{
			Signature: "0x" + fakeSignature,
			Authorization: PaymentAuthorization{
				From:        m.address,
				To:          req.PayTo,
				Value:       req.MaxAmountRequired,
				ValidAfter:  fmt.Sprintf("%d", time.Now().Unix()),
				ValidBefore: fmt.Sprintf("%d", time.Now().Add(60*time.Second).Unix()),
				Nonce:       "0x" + strings.Repeat("11", 32),
			},
		},
	}, nil
}
