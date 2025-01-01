package nakamoto

import (
	// "math"
	"fmt"
	"io"
	"os"
	"testing"
)

func TestGetBlockReward(t *testing.T) {
	// Get block reward for the next 120 years.
	blocksIn8Years := 1 * 6 * 24 * 365 * 120
	xy := make([][2]float64, blocksIn8Years)
	for i := 0; i < blocksIn8Years; i++ {
		xy[i] = [2]float64{float64(i), float64(GetBlockReward(i))}
	}

	// Dump this to a csv for visualisation in the IPython notebook.
	f, err := os.Create("block_reward.csv")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	// Write line-by-line.
	_, err = io.WriteString(f, "block_height,reward\n")
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range xy {
		_, err := io.WriteString(f, fmt.Sprintf("%f,%f\n", v[0], v[1]))
		if err != nil {
			t.Fatal(err)
		}
	}
}
