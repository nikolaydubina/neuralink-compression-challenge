package wav_test

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/nikolaydubina/neuralink-compression-challenge/go-encoder/wav"
)

func TestWAVReaderWriter(t *testing.T) {
	filename := "testdata/0ab237b7-fb12-4687-afed-8d1e2070d621.wav"

	f, err := os.Open(filename)
	if err != nil {
		t.Error(err)
	}

	r := wav.NewWAVReader(f)
	if err := r.ReadHeader(); err != nil {
		t.Error(err)
	}

	out, err := os.CreateTemp(t.TempDir(), "same*")
	if err != nil {
		t.Error(err)
	}

	w := wav.NewWAVWriter(r.Header, out)
	w.WriteHeader()

	for sample, err := r.Next(); err != io.EOF; sample, err = r.Next() {
		w.WriteSample(sample)
	}

	f.Close()
	out.Close()

	t.Run("output file is the same", func(t *testing.T) {
		exp, err := os.ReadFile(filename)
		if err != nil {
			t.Error(err)
		}

		got, err := os.ReadFile(out.Name())
		if err != nil {
			t.Error(err)
		}

		if !bytes.Equal(exp, got) {
			t.Error("output file is not the same")
		}
	})
}
