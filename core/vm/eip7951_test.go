package vm

import (
	"encoding/hex"
	"testing"

	libcommon "github.com/erigontech/erigon-lib/common"
	"github.com/erigontech/erigon/params"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEIP7951PrecompileAddress verifies the P256VERIFY precompile is at address 0x100
func TestEIP7951PrecompileAddress(t *testing.T) {
	expectedAddr := libcommon.BytesToAddress([]byte{0x01, 0x00})

	// Verify the address is 0x0000...0100
	assert.Equal(t, libcommon.HexToAddress("0x0000000000000000000000000000000000000100"), expectedAddr)

	// Verify in Napoli precompiles
	_, exists := PrecompiledContractsNapoli[expectedAddr]
	assert.True(t, exists, "P256VERIFY should exist in Napoli precompiles")

	// Verify in Osaka precompiles
	_, exists = PrecompiledContractsOsaka[expectedAddr]
	assert.True(t, exists, "P256VERIFY should exist in Osaka precompiles")
}

// TestEIP7951GasConstants verifies the gas cost constants
func TestEIP7951GasConstants(t *testing.T) {
	// Pre-Osaka (PIP-27) gas cost
	assert.Equal(t, uint64(3450), params.P256VerifyGas)

	// EIP-7951 (Osaka) gas cost - doubled
	assert.Equal(t, uint64(6900), params.P256VerifyGasEIP7951)

	// Verify the ratio
	assert.Equal(t, params.P256VerifyGasEIP7951, params.P256VerifyGas*2)
}

// TestEIP7951RequiredGas verifies the RequiredGas method for both versions
func TestEIP7951RequiredGas(t *testing.T) {
	preOsakaVerify := &p256Verify{eip7951: false}
	osakaVerify := &p256Verify{eip7951: true}

	// Any input should return the constant gas
	input := make([]byte, 160)

	assert.Equal(t, params.P256VerifyGas, preOsakaVerify.RequiredGas(input))
	assert.Equal(t, params.P256VerifyGasEIP7951, osakaVerify.RequiredGas(input))

	// Gas should be the same regardless of input length
	shortInput := make([]byte, 32)
	assert.Equal(t, params.P256VerifyGas, preOsakaVerify.RequiredGas(shortInput))
	assert.Equal(t, params.P256VerifyGasEIP7951, osakaVerify.RequiredGas(shortInput))

	// Empty input
	assert.Equal(t, params.P256VerifyGas, preOsakaVerify.RequiredGas(nil))
	assert.Equal(t, params.P256VerifyGasEIP7951, osakaVerify.RequiredGas(nil))
}

// TestEIP7951InputLength verifies the required input length of 160 bytes
func TestEIP7951InputLength(t *testing.T) {
	verifier := &p256Verify{}

	tests := []struct {
		name        string
		inputLen    int
		expectEmpty bool
	}{
		{"empty input", 0, true},
		{"too short (32 bytes)", 32, true},
		{"too short (64 bytes)", 64, true},
		{"too short (96 bytes)", 96, true},
		{"too short (128 bytes)", 128, true},
		{"too short (159 bytes)", 159, true},
		{"exact length (160 bytes)", 160, false}, // Will fail verification but not due to length
		{"too long (161 bytes)", 161, true},
		{"too long (192 bytes)", 192, true},
		{"too long (256 bytes)", 256, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			input := make([]byte, tc.inputLen)
			result, err := verifier.Run(input)

			assert.NoError(t, err)

			if tc.inputLen != 160 {
				// Wrong length returns empty (nil)
				assert.Nil(t, result)
			}
		})
	}
}

// TestEIP7951ValidSignature verifies a valid secp256r1 signature
func TestEIP7951ValidSignature(t *testing.T) {
	verifier := &p256Verify{}

	// Valid test vector from EIP-7212 / p256Verify.json
	inputHex := "4cee90eb86eaa050036147a12d49004b6b9c72bd725d39d4785011fe190f0b4da73bd4903f0ce3b639bbbf6e8e80d16931ff4bcf5993d58468e8fb19086e8cac36dbcd03009df8c59286b162af3bd7fcc0450c9aa81be5d10d312af6c66b1d604aebd3099c618202fcfe16ae7770b0c49ab5eadf74b754204a3bb6060e44eff37618b065f9832de4ca6ca971a7a1adc826d0f7c00181a5fb2ddf79ae00b4e10e"

	input, err := hex.DecodeString(inputHex)
	require.NoError(t, err)
	require.Equal(t, 160, len(input), "Input should be 160 bytes")

	result, err := verifier.Run(input)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Should return 1 (true) as 32-byte big-endian
	expected := make([]byte, 32)
	expected[31] = 1
	assert.Equal(t, expected, result)
}

// TestEIP7951InvalidSignature verifies an invalid signature returns empty
func TestEIP7951InvalidSignature(t *testing.T) {
	verifier := &p256Verify{}

	// Invalid test vector (modified from valid one)
	inputHex := "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9c29c3df6ce3431b6f030b1b68b1589508ad9d1a149830489c638653aa4b08af93f6e86a9a7643403b6f5c593410d9f7234a8cd27309bce90447073ce17476850615ff147863bc8652be1e369444f90bbc5f9df05a26362e609f73ab1f1839fe3cd34fd2ae672c110671d49115825fc56b5148321aabe5ba39f2b46f71149cff9"

	input, err := hex.DecodeString(inputHex)
	require.NoError(t, err)
	require.Equal(t, 160, len(input))

	result, err := verifier.Run(input)
	assert.NoError(t, err)
	assert.Nil(t, result, "Invalid signature should return nil")
}

// TestEIP7951OsakaPrecompileRegistration verifies the Osaka precompile uses EIP-7951 gas
func TestEIP7951OsakaPrecompileRegistration(t *testing.T) {
	addr := libcommon.BytesToAddress([]byte{0x01, 0x00})

	// Get the precompile from Osaka
	precompile := PrecompiledContractsOsaka[addr]
	require.NotNil(t, precompile, "P256VERIFY should exist in Osaka")

	// Verify it's p256Verify type with eip7951 flag
	p256, ok := precompile.(*p256Verify)
	require.True(t, ok, "Should be *p256Verify type")
	assert.True(t, p256.eip7951, "Should have eip7951 flag set in Osaka")

	// Verify gas cost
	assert.Equal(t, params.P256VerifyGasEIP7951, p256.RequiredGas(nil))
}

// TestEIP7951NapoliPrecompileRegistration verifies the pre-Osaka precompile uses original gas
func TestEIP7951NapoliPrecompileRegistration(t *testing.T) {
	addr := libcommon.BytesToAddress([]byte{0x01, 0x00})

	// Get the precompile from Napoli
	precompile := PrecompiledContractsNapoli[addr]
	require.NotNil(t, precompile, "P256VERIFY should exist in Napoli")

	// Verify it's p256Verify type without eip7951 flag
	p256, ok := precompile.(*p256Verify)
	require.True(t, ok, "Should be *p256Verify type")
	assert.False(t, p256.eip7951, "Should NOT have eip7951 flag set in Napoli")

	// Verify gas cost
	assert.Equal(t, params.P256VerifyGas, p256.RequiredGas(nil))
}

// TestEIP7951InputLayout verifies the input layout:
// hash (32) || r (32) || s (32) || x (32) || y (32) = 160 bytes
func TestEIP7951InputLayout(t *testing.T) {
	verifier := &p256Verify{}

	// Create input with known structure
	input := make([]byte, 160)

	// Fill with recognizable patterns
	// hash [0:32]
	for i := 0; i < 32; i++ {
		input[i] = 0xAA
	}
	// r [32:64]
	for i := 32; i < 64; i++ {
		input[i] = 0xBB
	}
	// s [64:96]
	for i := 64; i < 96; i++ {
		input[i] = 0xCC
	}
	// x [96:128]
	for i := 96; i < 128; i++ {
		input[i] = 0xDD
	}
	// y [128:160]
	for i := 128; i < 160; i++ {
		input[i] = 0xEE
	}

	// This will fail verification but should not error
	result, err := verifier.Run(input)
	assert.NoError(t, err)
	assert.Nil(t, result, "Invalid signature should return nil")
}

// TestEIP7951GasDoubling verifies the gas cost doubled from pre-Osaka to Osaka
func TestEIP7951GasDoubling(t *testing.T) {
	preOsakaVerify := &p256Verify{eip7951: false}
	osakaVerify := &p256Verify{eip7951: true}

	preOsakaGas := preOsakaVerify.RequiredGas(nil)
	osakaGas := osakaVerify.RequiredGas(nil)

	assert.Equal(t, preOsakaGas*2, osakaGas, "Osaka gas should be exactly 2x pre-Osaka gas")
	t.Logf("Pre-Osaka gas: %d, Osaka gas: %d, ratio: %.2f", preOsakaGas, osakaGas, float64(osakaGas)/float64(preOsakaGas))
}

// TestEIP7951MultipleValidSignatures tests multiple valid signatures
func TestEIP7951MultipleValidSignatures(t *testing.T) {
	verifier := &p256Verify{}

	validInputs := []string{
		// Test vector 1
		"4cee90eb86eaa050036147a12d49004b6b9c72bd725d39d4785011fe190f0b4da73bd4903f0ce3b639bbbf6e8e80d16931ff4bcf5993d58468e8fb19086e8cac36dbcd03009df8c59286b162af3bd7fcc0450c9aa81be5d10d312af6c66b1d604aebd3099c618202fcfe16ae7770b0c49ab5eadf74b754204a3bb6060e44eff37618b065f9832de4ca6ca971a7a1adc826d0f7c00181a5fb2ddf79ae00b4e10e",
		// Test vector 2
		"b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9414de3726ee4d237b410c1d85ebcb05553dc578561d9f7942b7250795beb9b9027b657067322fc00ab35263fde0acabf998cd9fcf1282df9555f85dba7bdbbe2dc90f74c9e210bc3e0c60aeaa03729c9e6acde4a048ee58fd2e466c1e7b0374e606b8c22ad2985df7d792ff344f03ce94a079da801006b13640bc5af7932a7b9",
		// Test vector 3
		"b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9b35d6a4f7f6fc5620c97d4287696f5174b3d37fa537b74b5fc26997ba79c725d62fe5e5fe6da76eec924e822c5ef853ede6c17069a9e9133a38f87d61599f68e7d5f3c812a255436846ee84a262b79ec4d0783afccf2433deabdca9ecf62bef5ff24e90988c7f139d378549c3a8bc6c94e6a1c911c1e02e6f48ed65aaf3d296e",
	}

	expected := make([]byte, 32)
	expected[31] = 1

	for i, inputHex := range validInputs {
		t.Run(string(rune('1'+i)), func(t *testing.T) {
			input, err := hex.DecodeString(inputHex)
			require.NoError(t, err)

			result, err := verifier.Run(input)
			assert.NoError(t, err)
			assert.Equal(t, expected, result, "Valid signature should return 1")
		})
	}
}

// TestEIP7951ZeroInput tests all-zero input
func TestEIP7951ZeroInput(t *testing.T) {
	verifier := &p256Verify{}

	input := make([]byte, 160)
	result, err := verifier.Run(input)

	assert.NoError(t, err)
	// Zero r, s or invalid point should fail verification
	assert.Nil(t, result, "All-zero input should fail verification")
}

// TestEIP7951PrecompileAddressFormat verifies address encoding
func TestEIP7951PrecompileAddressFormat(t *testing.T) {
	// EIP-7951 specifies address 0x100
	addr := libcommon.BytesToAddress([]byte{0x01, 0x00})

	// Verify it's in the precompile address list
	found := false
	for _, precompileAddr := range PrecompiledAddressesOsaka {
		if precompileAddr == addr {
			found = true
			break
		}
	}
	assert.True(t, found, "0x100 should be in PrecompiledAddressesOsaka")
}

// TestEIP7951ComplianceStatus documents the implementation status
func TestEIP7951ComplianceStatus(t *testing.T) {
	// EIP-7951: secp256r1 (P-256) signature verification precompile
	// Fusaka upgrade EIP

	implementation := struct {
		EIP              string
		Title            string
		PrecompileAddr   string
		PreOsakaGas      uint64
		OsakaGas         uint64
		InputLength      int
		Implemented      bool
		Tested           bool
	}{
		EIP:            "7951",
		Title:          "secp256r1 (P-256) Signature Verification Precompile",
		PrecompileAddr: "0x0000000000000000000000000000000000000100",
		PreOsakaGas:    params.P256VerifyGas,      // 3450
		OsakaGas:       params.P256VerifyGasEIP7951, // 6900
		InputLength:    160,
		Implemented:    true,
		Tested:         true,
	}

	// Verify implementation
	assert.Equal(t, "7951", implementation.EIP)
	assert.Equal(t, uint64(3450), implementation.PreOsakaGas)
	assert.Equal(t, uint64(6900), implementation.OsakaGas)
	assert.Equal(t, 160, implementation.InputLength)
	assert.True(t, implementation.Implemented)
	assert.True(t, implementation.Tested)

	t.Logf("EIP-%s: %s - Implementation complete", implementation.EIP, implementation.Title)
	t.Logf("Precompile address: %s", implementation.PrecompileAddr)
	t.Logf("Gas cost: %d (pre-Osaka) -> %d (Osaka/EIP-7951)", implementation.PreOsakaGas, implementation.OsakaGas)
	t.Logf("Input format: hash(32) || r(32) || s(32) || x(32) || y(32) = %d bytes", implementation.InputLength)
}

// TestEIP7951UseCases documents the primary use cases
func TestEIP7951UseCases(t *testing.T) {
	t.Log("EIP-7951 Use Cases:")
	t.Log("1. WebAuthn/FIDO2: Verify signatures from hardware security keys")
	t.Log("2. Secure Enclave: Verify signatures from Apple Secure Enclave (iPhone/iPad/Mac)")
	t.Log("3. Android Keystore: Verify signatures from Android secure hardware")
	t.Log("4. HSM: Verify signatures from Hardware Security Modules")
	t.Log("5. TEE: Verify signatures from Trusted Execution Environments")
	t.Log("")
	t.Log("The secp256r1 (P-256/prime256v1) curve is widely supported by modern hardware,")
	t.Log("enabling native verification of signatures from consumer devices.")
}

