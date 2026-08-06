package services

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/klauspost/reedsolomon"
)

type ErasureService struct {
	DataShards   int
	ParityShards int
	Enc          reedsolomon.Encoder
}

// NewErasureService initializes the Reed-Solomon encoder for 8+4 by default
func NewErasureService(dataShards, parityShards int) (*ErasureService, error) {
	enc, err := reedsolomon.New(dataShards, parityShards)
	if err != nil {
		return nil, err
	}
	return &ErasureService{
		DataShards:   dataShards,
		ParityShards: parityShards,
		Enc:          enc,
	}, nil
}

// EncodeAndBracket splits the payload, calculates parity, and then synchronously
// performs a verification "Bracket" test by simulating the loss of shards.
// Returns the 12 shards ready for network distribution.
func (es *ErasureService) EncodeAndBracket(data []byte) ([][]byte, error) {
	if len(data) == 0 {
		return nil, errors.New("cannot encode empty data")
	}

	// 1. Sharding & Parity Generation
	shards, err := es.Enc.Split(data)
	if err != nil {
		return nil, err
	}

	err = es.Enc.Encode(shards)
	if err != nil {
		return nil, err
	}

	// 2. Synchronous Bracketing (Integrity Verification)
	if err := es.SimulateDisasterAndVerify(data, shards); err != nil {
		return nil, errors.New("bracketing failed: data corruption detected during simulation: " + err.Error())
	}

	return shards, nil
}

// SimulateDisasterAndVerify takes the generated shards, intentionally deletes a subset of them,
// reconstructs them, and compares byte-by-byte with the original data.
// Executa 5 testes determinísticos que formam uma base matemática para garantir a validade da matriz.
func (es *ErasureService) SimulateDisasterAndVerify(originalData []byte, originalShards [][]byte) error {
	totalShards := es.DataShards + es.ParityShards
	
	// Em RS(8,4), Shards de Dados = 0 a 7. Paridade = 8 a 11.
	// Checkups requeridos (Shards a serem mantidos para a reconstrução):
	checkups := [][]int{
		{8, 9, 10, 11, 4, 5, 6, 7},
		{8, 9, 10, 11, 3, 5, 6, 7},
		{8, 9, 10, 11, 2, 5, 6, 7},
		{8, 9, 10, 11, 1, 5, 6, 7},
		{8, 9, 10, 11, 0, 5, 6, 7},
	}

	for i, keepIndices := range checkups {
		// Deep copy para não mutar os originais
		testShards := make([][]byte, totalShards)
		for j := range originalShards {
			testShards[j] = make([]byte, len(originalShards[j]))
			copy(testShards[j], originalShards[j])
		}

		// Mapeia quem deve ser mantido
		keepMap := make(map[int]bool)
		for _, idx := range keepIndices {
			keepMap[idx] = true
		}

		// Nula todos os outros (simulando perda no disco/rede)
		for j := 0; j < totalShards; j++ {
			if !keepMap[j] {
				testShards[j] = nil
			}
		}

		// Reconstrói
		if err := es.Enc.Reconstruct(testShards); err != nil {
			return fmt.Errorf("falha ao reconstruir no checkup %d: %v", i, err)
		}

		// Junta pra verificar
		buf := new(bytes.Buffer)
		if err := es.Enc.Join(buf, testShards, len(originalData)); err != nil {
			return fmt.Errorf("falha ao juntar no checkup %d: %v", i, err)
		}

		if !bytes.Equal(buf.Bytes(), originalData) {
			return fmt.Errorf("reconstrução falhou no checkup %d", i)
		}
	}

	return nil
}
