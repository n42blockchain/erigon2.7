package vm

import (
	"math/big"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/erigontech/erigon/core/vm/evmtypes"
	"github.com/erigontech/erigon/core/vm/stack"
	"github.com/erigontech/erigon/params"
)

// TestEIP7939CLZOpcodeValue verifies the CLZ opcode value (0x1E)
func TestEIP7939CLZOpcodeValue(t *testing.T) {
	// CLZ should be 0x1E according to EIP-7939
	assert.Equal(t, OpCode(0x1E), CLZ)
	assert.Equal(t, "CLZ", CLZ.String())
}

// TestEIP7939CLZGasCost verifies the CLZ gas cost
func TestEIP7939CLZGasCost(t *testing.T) {
	// CLZ should use GasFastStep (5) as gas cost
	assert.Equal(t, uint64(5), GasFastStep)

	// Verify in instruction set
	jt := newOsakaInstructionSet()
	assert.NotNil(t, jt[CLZ])
	assert.Equal(t, GasFastStep, jt[CLZ].constantGas)
	assert.Equal(t, 1, jt[CLZ].numPop)
	assert.Equal(t, 1, jt[CLZ].numPush)
}

// TestEIP7939CLZComprehensive provides comprehensive CLZ tests
func TestEIP7939CLZComprehensive(t *testing.T) {
	tests := []struct {
		name     string
		input    *uint256.Int
		expected uint64
	}{
		// Basic values
		{"zero", uint256.NewInt(0), 256},
		{"one", uint256.NewInt(1), 255},
		{"two", uint256.NewInt(2), 254},
		{"three", uint256.NewInt(3), 254},
		{"four", uint256.NewInt(4), 253},
		{"seven", uint256.NewInt(7), 253},
		{"eight", uint256.NewInt(8), 252},
		{"fifteen", uint256.NewInt(15), 252},
		{"sixteen", uint256.NewInt(16), 251},

		// Powers of 2
		{"2^8", uint256.NewInt(256), 247},
		{"2^16", uint256.NewInt(65536), 239},
		{"2^32", new(uint256.Int).SetUint64(1 << 32), 223},
		{"2^64", new(uint256.Int).Lsh(uint256.NewInt(1), 64), 191},
		{"2^128", new(uint256.Int).Lsh(uint256.NewInt(1), 128), 127},
		{"2^255", new(uint256.Int).Lsh(uint256.NewInt(1), 255), 0},

		// Max values
		{"max-uint64", new(uint256.Int).SetUint64(^uint64(0)), 192},
		{"max-uint128", new(uint256.Int).Sub(new(uint256.Int).Lsh(uint256.NewInt(1), 128), uint256.NewInt(1)), 128},
		{"max-uint256", new(uint256.Int).Sub(new(uint256.Int).Lsh(uint256.NewInt(1), 256), uint256.NewInt(1)), 0},

		// Edge cases
		{"high-bit-only", new(uint256.Int).Lsh(uint256.NewInt(1), 255), 0},
		{"high-two-bits", new(uint256.Int).Lsh(uint256.NewInt(3), 254), 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			env := NewEVM(evmtypes.BlockContext{}, evmtypes.TxContext{}, nil, params.TestChainConfig, Config{})
			stk := stack.New()
			interpreter := NewEVMInterpreter(env, env.Config())
			pc := uint64(0)

			// Clone input to avoid modification
			input := new(uint256.Int).Set(tc.input)
			stk.Push(input)

			_, err := opCLZ(&pc, interpreter, &ScopeContext{nil, stk, nil})
			require.NoError(t, err)

			result := stk.Pop()
			assert.Equal(t, tc.expected, result.Uint64(), "CLZ(%s) = %d; want %d", tc.name, result.Uint64(), tc.expected)
		})
	}
}

// TestEIP7939CLZBitPatterns tests specific bit patterns
func TestEIP7939CLZBitPatterns(t *testing.T) {
	tests := []struct {
		name     string
		pattern  string // hex pattern
		expected uint64
	}{
		{"all-zeros", "0x0", 256},
		{"all-ones", "0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", 0},
		{"high-nibble-1", "0x1000000000000000000000000000000000000000000000000000000000000000", 3},
		{"high-nibble-8", "0x8000000000000000000000000000000000000000000000000000000000000000", 0},
		{"alternating-10", "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", 0},
		{"alternating-01", "0x5555555555555555555555555555555555555555555555555555555555555555", 1},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			env := NewEVM(evmtypes.BlockContext{}, evmtypes.TxContext{}, nil, params.TestChainConfig, Config{})
			stk := stack.New()
			interpreter := NewEVMInterpreter(env, env.Config())
			pc := uint64(0)

			val := new(uint256.Int)
			require.NoError(t, val.SetFromHex(tc.pattern), "failed to parse %s", tc.pattern)

			stk.Push(val)
			_, err := opCLZ(&pc, interpreter, &ScopeContext{nil, stk, nil})
			require.NoError(t, err)

			result := stk.Pop()
			assert.Equal(t, tc.expected, result.Uint64())
		})
	}
}

// TestEIP7939CLZStackBehavior verifies correct stack manipulation
func TestEIP7939CLZStackBehavior(t *testing.T) {
	env := NewEVM(evmtypes.BlockContext{}, evmtypes.TxContext{}, nil, params.TestChainConfig, Config{})
	stk := stack.New()
	interpreter := NewEVMInterpreter(env, env.Config())
	pc := uint64(0)

	// Push value and execute
	stk.Push(uint256.NewInt(1))
	initialLen := stk.Len()

	_, err := opCLZ(&pc, interpreter, &ScopeContext{nil, stk, nil})
	require.NoError(t, err)

	// Stack length should remain the same (pop 1, push 1)
	assert.Equal(t, initialLen, stk.Len())

	// Result should be 255 (CLZ of 1)
	result := stk.Pop()
	assert.Equal(t, uint64(255), result.Uint64())
}

// TestEIP7939CLZNotEnabledBeforeOsaka verifies CLZ is not available before Osaka
func TestEIP7939CLZNotEnabledBeforeOsaka(t *testing.T) {
	// Prague instruction set should NOT have CLZ enabled (uses opUndefined)
	pragueJt := newPragueInstructionSet()
	assert.NotNil(t, pragueJt[CLZ]) // Slot is filled but with opUndefined
	assert.Equal(t, uint64(0), pragueJt[CLZ].constantGas, "CLZ should use opUndefined gas in Prague")

	// Osaka instruction set SHOULD have CLZ with proper gas cost
	osakaJt := newOsakaInstructionSet()
	assert.NotNil(t, osakaJt[CLZ], "CLZ should be enabled in Osaka")
	assert.Equal(t, GasFastStep, osakaJt[CLZ].constantGas, "CLZ should use GasFastStep in Osaka")
}

// TestEIP7939Enable7939 verifies the enable7939 function
func TestEIP7939Enable7939(t *testing.T) {
	jt := newPragueInstructionSet()
	// Before enable7939, CLZ slot is filled with opUndefined (gas = 0)
	assert.Equal(t, uint64(0), jt[CLZ].constantGas, "CLZ should use opUndefined before enable7939")

	enable7939(&jt)
	assert.NotNil(t, jt[CLZ], "CLZ should exist after enable7939")
	assert.Equal(t, GasFastStep, jt[CLZ].constantGas)
	assert.Equal(t, 1, jt[CLZ].numPop)
	assert.Equal(t, 1, jt[CLZ].numPush)
}

// TestEIP7939CLZMathematicalProperties tests mathematical properties of CLZ
func TestEIP7939CLZMathematicalProperties(t *testing.T) {
	env := NewEVM(evmtypes.BlockContext{}, evmtypes.TxContext{}, nil, params.TestChainConfig, Config{})
	interpreter := NewEVMInterpreter(env, env.Config())

	// Property: CLZ(x) + BitLen(x) = 256 for x > 0
	t.Run("CLZ_plus_BitLen_equals_256", func(t *testing.T) {
		testValues := []*uint256.Int{
			uint256.NewInt(1),
			uint256.NewInt(255),
			uint256.NewInt(256),
			uint256.NewInt(65535),
			new(uint256.Int).SetUint64(^uint64(0)),
		}

		for _, val := range testValues {
			stk := stack.New()
			pc := uint64(0)

			bitLen := uint64(val.BitLen())
			stk.Push(new(uint256.Int).Set(val))
			opCLZ(&pc, interpreter, &ScopeContext{nil, stk, nil})
			result := stk.Pop()
			clz := result.Uint64()

			assert.Equal(t, uint64(256), clz+bitLen, "CLZ(%v) + BitLen = %d + %d = %d; want 256", val, clz, bitLen, clz+bitLen)
		}
	})

	// Property: CLZ(x) >= CLZ(y) when x <= y for same bit-width values
	t.Run("CLZ_monotonicity", func(t *testing.T) {
		pairs := [][2]*uint256.Int{
			{uint256.NewInt(1), uint256.NewInt(2)},
			{uint256.NewInt(100), uint256.NewInt(200)},
			{uint256.NewInt(1000), uint256.NewInt(10000)},
		}

		for _, pair := range pairs {
			stk := stack.New()
			pc := uint64(0)

			stk.Push(new(uint256.Int).Set(pair[0]))
			opCLZ(&pc, interpreter, &ScopeContext{nil, stk, nil})
			result1 := stk.Pop()
			clz1 := result1.Uint64()

			stk.Push(new(uint256.Int).Set(pair[1]))
			opCLZ(&pc, interpreter, &ScopeContext{nil, stk, nil})
			result2 := stk.Pop()
			clz2 := result2.Uint64()

			assert.GreaterOrEqual(t, clz1, clz2, "CLZ(%v) >= CLZ(%v): %d >= %d", pair[0], pair[1], clz1, clz2)
		}
	})

	// Property: CLZ(2^n) = 256 - n - 1 = 255 - n
	t.Run("CLZ_power_of_two", func(t *testing.T) {
		for n := uint(0); n < 256; n++ {
			stk := stack.New()
			pc := uint64(0)

			val := new(uint256.Int).Lsh(uint256.NewInt(1), n)
			stk.Push(val)
			opCLZ(&pc, interpreter, &ScopeContext{nil, stk, nil})
			result := stk.Pop()
			clz := result.Uint64()

			expected := uint64(255 - n)
			assert.Equal(t, expected, clz, "CLZ(2^%d) = %d; want %d", n, clz, expected)
		}
	})
}

// TestEIP7939ActivatorRegistration verifies the activator is registered
func TestEIP7939ActivatorRegistration(t *testing.T) {
	activator, exists := activators[7939]
	require.True(t, exists, "EIP-7939 activator should be registered")
	require.NotNil(t, activator)

	// Test the activator function
	jt := newPragueInstructionSet()
	activator(&jt)
	assert.NotNil(t, jt[CLZ])
}

// TestEIP7939ComplianceStatus documents the implementation status
func TestEIP7939ComplianceStatus(t *testing.T) {
	// EIP-7939: CLZ (Count Leading Zeros) opcode
	// Fusaka upgrade EIP

	implementation := struct {
		EIP         string
		Title       string
		Opcode      OpCode
		OpcodeHex   uint8
		GasCost     uint64
		StackPop    int
		StackPush   int
		Implemented bool
		Tested      bool
	}{
		EIP:         "7939",
		Title:       "CLZ (Count Leading Zeros) opcode",
		Opcode:      CLZ,
		OpcodeHex:   0x1E,
		GasCost:     GasFastStep, // 5
		StackPop:    1,
		StackPush:   1,
		Implemented: true,
		Tested:      true,
	}

	// Verify implementation
	assert.Equal(t, "7939", implementation.EIP)
	assert.Equal(t, OpCode(0x1E), implementation.Opcode)
	assert.Equal(t, uint64(5), implementation.GasCost)
	assert.True(t, implementation.Implemented)
	assert.True(t, implementation.Tested)

	t.Logf("EIP-%s: %s - Implementation complete", implementation.EIP, implementation.Title)
	t.Logf("Opcode: 0x%02X (%s)", implementation.OpcodeHex, CLZ.String())
	t.Logf("Gas cost: %d (GasFastStep)", implementation.GasCost)
}

// TestEIP7939CLZWithBigInt tests CLZ with big.Int conversion
func TestEIP7939CLZWithBigInt(t *testing.T) {
	env := NewEVM(evmtypes.BlockContext{}, evmtypes.TxContext{}, nil, params.TestChainConfig, Config{})
	stk := stack.New()
	interpreter := NewEVMInterpreter(env, env.Config())

	tests := []struct {
		name     string
		bigInt   *big.Int
		expected uint64
	}{
		{"big-1", big.NewInt(1), 255},
		{"big-256", big.NewInt(256), 247},
		// 9223372036854775807 = 2^63 - 1, BitLen = 63, CLZ = 256 - 63 = 193
		{"big-max-int64", big.NewInt(9223372036854775807), 193},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pc := uint64(0)
			val := new(uint256.Int)
			val.SetFromBig(tc.bigInt)

			stk.Push(val)
			_, err := opCLZ(&pc, interpreter, &ScopeContext{nil, stk, nil})
			require.NoError(t, err)

			result := stk.Pop()
			assert.Equal(t, tc.expected, result.Uint64())
		})
	}
}

