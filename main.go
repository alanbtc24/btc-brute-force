package main

import (
	"encoding/hex"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcutil"
	"github.com/decred/dcrd/dcrec/secp256k1/v4"
)

var (
	totalKeysChecked int64
	startTime        = time.Now()
	done             = make(chan struct{})
	pool             = sync.Pool{
		New: func() interface{} {
			return make([]byte, 32) // Tamanho da chave
		},
	}
)

func checkKey(address string, start, end []byte, wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		select {
		case <-done:
			return
		default:
			if compare(start, end) > 0 {
				close(done)
				return
			}

			privKey := secp256k1.PrivKeyFromBytes(start)
			pubKey := privKey.PubKey().SerializeCompressed()
			addressPubKey, err := btcutil.NewAddressPubKey(pubKey, &chaincfg.MainNetParams)
			if err != nil {
				fmt.Println("Erro ao criar endereço público:", err)
				return
			}

			if addressPubKey.EncodeAddress() == address {
				fmt.Printf("\nChave Encontrada: %x\n", start)
				fmt.Printf("Endereço: %s\n", addressPubKey.EncodeAddress())
				close(done)
				return
			}

			totalKeysChecked++
			increment(start)
		}
	}
}

func increment(key []byte) {
	for i := len(key) - 1; i >= 0; i-- {
		if key[i] < 255 {
			key[i]++
			return
		}
		key[i] = 0
	}
}

func compare(a, b []byte) int {
	for i := range a {
		if a[i] != b[i] {
			return int(a[i]) - int(b[i])
		}
	}
	return 0
}

func formatNumber(num int64) string {
	if num >= 1e12 {
		return fmt.Sprintf("%.2f TRILHÕES", float64(num)/1e12)
	} else if num >= 1e9 {
		return fmt.Sprintf("%.2f BILHÕES", float64(num)/1e9)
	} else if num >= 1e6 {
		return fmt.Sprintf("%.2f MILHÕES", float64(num)/1e6)
	} else if num >= 1e3 {
		return fmt.Sprintf("%.2f MIL", float64(num)/1e3)
	} else {
		return fmt.Sprintf("%d", num)
	}
}

func main() {
	if len(os.Args) != 4 {
		fmt.Println("Uso: btc-bf.exe <endereço> <início> <fim>")
		return
	}

	address := os.Args[1]
	start, err := hex.DecodeString(os.Args[2])
	if err != nil {
		fmt.Println("Erro ao analisar o início do intervalo:", err)
		return
	}
	end, err := hex.DecodeString(os.Args[3])
	if err != nil {
		fmt.Println("Erro ao analisar o fim do intervalo:", err)
		return
	}

	numCPUs := 12        // Número de CPUs
	goroutinesPerCPU := 3 // Goroutines por CPU
	totalGoroutines := numCPUs * goroutinesPerCPU
	var wg sync.WaitGroup

	// Ajusta o tamanho do segmento para garantir uma divisão mais eficiente
	segmentSize := (1 << (len(start) * 8)) / totalGoroutines
	for i := 0; i < totalGoroutines; i++ {
		wg.Add(1)
		go func(offset int) {
			defer wg.Done()
			localStart := pool.Get().([]byte)
			localEnd := pool.Get().([]byte)
			defer pool.Put(localStart)
			defer pool.Put(localEnd)

			copy(localStart, start)
			copy(localEnd, end)

			for i := range localStart {
				localStart[i] += byte(offset * segmentSize)
				if (offset+1)*segmentSize < 256 {
					localEnd[i] += byte((offset+1) * segmentSize)
				}
			}

			checkKey(address, localStart, localEnd, &wg)
		}(i)
	}

	// Goroutine para exibir estatísticas
	go func() {
		for {
			select {
			case <-done:
				return
			default:
				// Limpa a tela e exibe as estatísticas
				fmt.Printf("\033[H\033[J")
				fmt.Printf("      CHAT GPT - Dev 01/08/2024 BRASIL\n")
				fmt.Printf("      ------------------------------\n")
				fmt.Printf("    Chaves Verificadas: %15s\n", formatNumber(totalKeysChecked))
				fmt.Printf("  Chaves por Segundo: %10.2f\n", float64(totalKeysChecked)/time.Since(startTime).Seconds())
				time.Sleep(1 * time.Second)
			}
		}
	}()

	wg.Wait() // Aguarda todas as goroutines terminarem

	// Garantir que a goroutine de exibição de estatísticas encerre corretamente
	close(done)
	time.Sleep(1 * time.Second)
}



