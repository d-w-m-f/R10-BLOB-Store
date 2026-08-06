package test

import (
	"testing"

	"gateway/internal/services"
)

func TestErasureService_EncodeAndBracket_Success(t *testing.T) {
	es, err := services.NewErasureService(8, 4)
	if err != nil {
		t.Fatalf("Failed to create ErasureService: %v", err)
	}

	// Create a 1MB dummy payload
	originalData := make([]byte, 1024*1024)
	for i := range originalData {
		originalData[i] = byte(i % 256)
	}

	shards, err := es.EncodeAndBracket(originalData)
	if err != nil {
		t.Fatalf("EncodeAndBracket failed unexpectedly: %v", err)
	}

	if len(shards) != 12 {
		t.Errorf("Expected 12 shards, got %d", len(shards))
	}
}

func TestErasureService_SimulateDisasterAndVerify_Corruption(t *testing.T) {
	es, err := services.NewErasureService(8, 4)
	if err != nil {
		t.Fatalf("Failed to create ErasureService: %v", err)
	}

	originalData := []byte("This is a small test string that needs to be encoded and protected.")

	shards, err := es.Enc.Split(originalData)
	if err != nil {
		t.Fatalf("Failed to split: %v", err)
	}
	if err := es.Enc.Encode(shards); err != nil {
		t.Fatalf("Failed to encode: %v", err)
	}

	// Intentionally corrupt one of the shards to simulate a silent bit-rot before Bracketing
	shards[0][0] = shards[0][0] ^ 0xFF

	// Bracketing should now fail because the reconstructed data won't match
	err = es.SimulateDisasterAndVerify(originalData, shards)
	if err == nil {
		t.Fatal("Expected Bracketing to fail due to corruption, but it succeeded")
	}
}
