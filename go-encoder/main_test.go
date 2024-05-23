package main_test

import (
	"bytes"
	"os"
	"os/exec"
	"path"
	"testing"
)

func TestCLIEncoder(t *testing.T) {
	testbin := path.Join(t.TempDir(), "go-encoder")
	exec.Command("go", "build", "-o", testbin, ".").Run()

	fnames := []string{
		"0052503c-2849-4f41-ab51-db382103690c.wav",
		"ff970660-0ffd-461f-93de-379e95cd784a.wav",
	}
	for _, f := range fnames {
		t.Run("cache trashing file: "+f, func(t *testing.T) {
			i := path.Join("testdata", f)
			e := path.Join("testdata", f+".encoded")
			d := path.Join("testdata", f+".decoded")

			var stderr bytes.Buffer
			cmd := exec.Command(testbin, "-mode", "encode", "-in", i, "-out", e)
			cmd.Stderr = &stderr
			cmd.Run()
			t.Log("\n" + stderr.String())

			exec.Command(testbin, "-mode", "decode", "-in", e, "-out", d).Run()

			fa, _ := os.ReadFile(i)
			fb, _ := os.ReadFile(d)
			if !bytes.Equal(fa, fb) {
				t.Errorf("files are different: %s != %s", string(fa), string(fb))
			}

			fe, _ := os.ReadFile(e)

			t.Logf("compression ratio: %.2f", float64(len(fa))/float64(len(fe)))
		})
	}
}
