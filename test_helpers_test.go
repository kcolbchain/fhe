// Copyright (c) 2025, Lux Industries Inc
// SPDX-License-Identifier: BSD-3-Clause

package fhe

import (
	"math/big"
	"sync"
	"testing"
)

type testContext struct {
	params    Parameters
	sk        *SecretKey
	pk        *PublicKey
	bsk       *BootstrapKey
	enc       *Encryptor
	dec       *Decryptor
	eval      *Evaluator
	intParams *IntegerParams
	intEnc    *IntegerEncryptor
	intDec    *IntegerDecryptor
	intEval   *IntegerEvaluator
	bitEnc    *BitwiseEncryptor
	bitDec    *BitwiseDecryptor
	bitEval   *BitwiseEvaluator
}

var (
	sharedTestContext *testContext
	sharedTestOnce    sync.Once
	sharedTestErr     error
)

func newTestContext(t *testing.T) *testContext {
	t.Helper()

	sharedTestOnce.Do(func() {
		params, err := NewParametersFromLiteral(PN10QP27)
		if err != nil {
			sharedTestErr = err
			return
		}

		kg := NewKeyGenerator(params)
		sk, pk := kg.GenKeyPair()
		bsk := kg.GenBootstrapKey(sk)

		intParams, err := NewIntegerParams(params, 2)
		if err != nil {
			sharedTestErr = err
			return
		}

		sharedTestContext = &testContext{
			params:    params,
			sk:        sk,
			pk:        pk,
			bsk:       bsk,
			enc:       NewEncryptor(params, sk),
			dec:       NewDecryptor(params, sk),
			eval:      NewEvaluator(params, bsk),
			intParams: intParams,
			intEnc:    NewIntegerEncryptor(intParams, sk),
			intDec:    NewIntegerDecryptor(intParams, sk),
			intEval:   NewIntegerEvaluator(intParams, bsk),
			bitEnc:    NewBitwiseEncryptor(params, sk),
			bitDec:    NewBitwiseDecryptor(params, sk),
			bitEval:   NewBitwiseEvaluator(params, bsk, nil),
		}
	})

	if sharedTestErr != nil {
		t.Fatalf("creating shared test context: %v", sharedTestErr)
	}
	return sharedTestContext
}

func testBooleanEncryptDecrypt(t *testing.T, tc *testContext) {
	t.Helper()

	for _, value := range []bool{false, true} {
		ct, err := tc.enc.EncryptSafe(value)
		if err != nil {
			t.Fatalf("encrypt %v: %v", value, err)
		}
		if got := tc.dec.Decrypt(ct); got != value {
			t.Fatalf("decrypt %v = %v", value, got)
		}
	}
}

func testBooleanGates(t *testing.T, tc *testContext) {
	t.Helper()

	cases := []struct {
		name string
		a    bool
		b    bool
		run  func(*Ciphertext, *Ciphertext) (*Ciphertext, error)
		want bool
	}{
		{"AND false false", false, false, tc.eval.AND, false},
		{"AND true false", true, false, tc.eval.AND, false},
		{"AND true true", true, true, tc.eval.AND, true},
		{"OR false false", false, false, tc.eval.OR, false},
		{"OR true false", true, false, tc.eval.OR, true},
		{"XOR false true", false, true, tc.eval.XOR, true},
		{"XOR true true", true, true, tc.eval.XOR, false},
		{"NAND true true", true, true, tc.eval.NAND, false},
		{"NOR false false", false, false, tc.eval.NOR, true},
		{"XNOR true true", true, true, tc.eval.XNOR, true},
	}

	for _, tcases := range cases {
		t.Run(tcases.name, func(t *testing.T) {
			a, err := tc.enc.EncryptSafe(tcases.a)
			if err != nil {
				t.Fatalf("encrypt a: %v", err)
			}
			b, err := tc.enc.EncryptSafe(tcases.b)
			if err != nil {
				t.Fatalf("encrypt b: %v", err)
			}
			gotCt, err := tcases.run(a, b)
			if err != nil {
				t.Fatalf("gate: %v", err)
			}
			if got := tc.dec.Decrypt(gotCt); got != tcases.want {
				t.Fatalf("got %v, want %v", got, tcases.want)
			}
		})
	}

	t.Run("NOT", func(t *testing.T) {
		ct, err := tc.enc.EncryptSafe(true)
		if err != nil {
			t.Fatalf("encrypt: %v", err)
		}
		if got := tc.dec.Decrypt(tc.eval.NOT(ct)); got {
			t.Fatal("NOT(true) = true, want false")
		}
	})
}

func testIntegerEncryptDecrypt(t *testing.T, tc *testContext, types []FheUintType) {
	t.Helper()

	for _, fheType := range types {
		t.Run(fheType.String(), func(t *testing.T) {
			value := uint64(5)
			if fheType.NumBits() > 4 {
				value = 42
			}

			ct, err := tc.intEnc.EncryptUint64(value, fheType)
			if err != nil {
				t.Fatalf("encrypt integer: %v", err)
			}
			if got := tc.intDec.DecryptUint64(ct); got != value {
				t.Fatalf("decrypt integer = %d, want %d", got, value)
			}
		})
	}
}

func testIntegerArithmetic(t *testing.T, tc *testContext, a, b uint64, fheType FheUintType) {
	t.Helper()

	ctA := tc.bitEnc.EncryptUint64(a, fheType)
	ctB := tc.bitEnc.EncryptUint64(b, fheType)
	sum, err := tc.bitEval.Add(ctA, ctB)
	if err != nil {
		t.Fatalf("add: %v", err)
	}

	mask := uint64(1<<fheType.NumBits()) - 1
	want := (a + b) & mask
	if got := tc.bitDec.DecryptUint64(sum); got != want {
		t.Fatalf("%d + %d = %d, want %d", a, b, got, want)
	}
}

func testBigIntRoundtrip(t *testing.T, tc *testContext, value *big.Int, fheType FheUintType) {
	t.Helper()

	ct, err := tc.intEnc.EncryptBigInt(value, fheType)
	if err != nil {
		t.Fatalf("encrypt big int: %v", err)
	}

	mask := new(big.Int).Lsh(big.NewInt(1), uint(fheType.NumBits()))
	mask.Sub(mask, big.NewInt(1))
	want := new(big.Int).And(new(big.Int).Set(value), mask)

	if got := tc.intDec.DecryptBigInt(ct); got.Cmp(want) != 0 {
		t.Fatalf("decrypt big int = %s, want %s", got.String(), want.String())
	}
}

func testKeySerialization(t *testing.T, tc *testContext) {
	t.Helper()

	skData, err := tc.sk.MarshalBinary()
	if err != nil {
		t.Fatalf("marshal secret key: %v", err)
	}
	var sk SecretKey
	if err := sk.UnmarshalBinary(skData); err != nil {
		t.Fatalf("unmarshal secret key: %v", err)
	}

	ct, err := tc.enc.EncryptSafe(true)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	if got := NewDecryptor(tc.params, &sk).Decrypt(ct); !got {
		t.Fatal("decryption with unmarshaled secret key = false, want true")
	}

	ctData, err := ct.MarshalBinary()
	if err != nil {
		t.Fatalf("marshal ciphertext: %v", err)
	}
	var decoded Ciphertext
	if err := decoded.UnmarshalBinary(ctData); err != nil {
		t.Fatalf("unmarshal ciphertext: %v", err)
	}
	if got := tc.dec.Decrypt(&decoded); !got {
		t.Fatal("decryption of unmarshaled ciphertext = false, want true")
	}
}

func testRNG(t *testing.T, tc *testContext, fheType FheUintType) {
	t.Helper()

	rng := NewFheRNG(tc.params, tc.sk, []byte("fhe-test-seed"))
	ct := rng.RandomUint(fheType)
	if ct == nil {
		t.Fatal("RandomUint returned nil")
	}
	if ct.NumBits() != fheType.NumBits() {
		t.Fatalf("RandomUint bits = %d, want %d", ct.NumBits(), fheType.NumBits())
	}
	if rng.Counter() == 0 {
		t.Fatal("RandomUint did not advance counter")
	}

	bit := rng.RandomBit()
	if bit == nil {
		t.Fatal("RandomBit returned nil")
	}
	_ = tc.dec.Decrypt(bit)
}
