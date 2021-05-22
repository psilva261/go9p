// +build !plan9

package go9p

import (
	"io/ioutil"
	"syscall"
	"testing"
)

var f *File
var b = make([]byte, 1048576/8)

// Not sure we want this, and the test has issues. Revive it if we ever find a use for it.
func TestPipefs(t *testing.T) {
	pipefs := new(Pipefs)
	pipefs.Dotu = false
	pipefs.Msize = 1048576
	pipefs.Id = "pipefs"
	pipefs.Root = *root
	pipefs.Debuglevel = *debug
	pipefs.Start(pipefs)

	t.Logf("pipefs starting\n")
	d, err := ioutil.TempDir("", "TestPipeFS")
	if err != nil {
		t.Fatalf("%v", err)
	}
	defer func() {
		if err := os.Remove(d); err != nil {
			t.Fatalf("%v", err)
		}
	}()
	fn := path.Join(d, "fifo")
	if err := syscall.Mkfifo(fn, 0600); err != nil {
		t.Fatalf("%v", err)
	}
	defer func() {
		if err := os.Remove(fn); err != nil {
			t.Fatalf("%v", err)
		}
	}()
	// determined by build tags
	//extraFuncs()
	go func() {
		err := pipefs.StartNetListener("tcp", *pipefsaddr)
		if err != nil {
			t.Fatalf("StartNetListener failed: %v\n", err)
		}
	}()
	root := OsUsers.Uid2User(0)

	var c *Clnt
	for i := 0; i < 16; i++ {
		c, err = Mount("tcp", *pipefsaddr, "/", uint32(len(b)), root)
	}
	if err != nil {
		t.Fatalf("Connect failed: %v\n", err)
	}
	t.Logf("Connected to %v\n", *c)
	if f, err = c.FOpen(fn, ORDWR); err != nil {
		t.Fatalf("Open failed: %v\n", err)
	} else {
		for i := 0; i < 1048576/8; i++ {
			b[i] = byte(i)
		}
		t.Logf("f %v \n", f)
		if n, err := f.Write(b); err != nil {
			t.Fatalf("write failed: %v\n", err)
		} else {
			t.Logf("Wrote %v bytes\n", n)
		}
		if n, err := f.Read(b); err != nil {
			t.Fatalf("read failed: %v\n", err)
		} else {
			t.Logf("read %v bytes\n", n)
		}

	}
}

func BenchmarkPipeFS(bb *testing.B) {
	for i := 0; i < bb.N; i++ {
		if _, err := f.Write(b); err != nil {
			bb.Errorf("write failed: %v\n", err)
		}
		if _, err := f.Read(b); err != nil {
			bb.Errorf("read failed: %v\n", err)
		}
	}
}
