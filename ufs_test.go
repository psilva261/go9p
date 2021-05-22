// +build !plan9

package go9p

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path"
	"testing"
)

func TestAttachOpenReaddir(t *testing.T) {
	var err error
	flag.Parse()
	ufs := new(Ufs)
	ufs.Dotu = false
	ufs.Id = "ufs"
	ufs.Root = *root
	ufs.Debuglevel = *debug
	ufs.Start(ufs)

	t.Log("ufs starting\n")
	// determined by build tags
	//extraFuncs()
	go func() {
		if err = ufs.StartNetListener("tcp", *addr); err != nil {
			t.Fatalf("Can not start listener: %v", err)
		}
	}()
	/* this may take a few tries ... */
	var conn net.Conn
	for i := 0; i < 16; i++ {
		if conn, err = net.Dial("tcp", *addr); err != nil {
			t.Logf("Try go connect, %d'th try, %v", i, err)
		} else {
			t.Logf("Got a conn, %v\n", conn)
			break
		}
	}
	if err != nil {
		t.Fatalf("Connect failed after many tries ...")
	}

	root := OsUsers.Uid2User(0)

	dir, err := ioutil.TempDir("", "go9p")
	if err != nil {
		t.Fatalf("got %v, want nil", err)
	}
	defer os.RemoveAll(dir)

	// Now create a whole bunch of files to test readdir
	for i := 0; i < numDir; i++ {
		f := fmt.Sprintf(path.Join(dir, fmt.Sprintf("%d", i)))
		if err := ioutil.WriteFile(f, []byte(f), 0600); err != nil {
			t.Fatalf("Create %v: got %v, want nil", f, err)
		}
	}

	var clnt *Clnt
	for i := 0; i < 16; i++ {
		clnt, err = Mount("tcp", *addr, dir, 8192, root)
	}
	if err != nil {
		t.Fatalf("Connect failed: %v\n", err)
	}

	defer clnt.Unmount()
	t.Logf("attached, rootfid %v\n", clnt.Root)

	dirfid := clnt.FidAlloc()
	if _, err = clnt.Walk(clnt.Root, dirfid, []string{"."}); err != nil {
		t.Fatalf("%v", err)
	}
	if err = clnt.Open(dirfid, 0); err != nil {
		t.Fatalf("%v", err)
	}
	var b []byte
	var i, amt int
	var offset uint64
	for i < numDir {
		if b, err = clnt.Read(dirfid, offset, 64*1024); err != nil {
			t.Fatalf("%v", err)
		}
		for b != nil && len(b) > 0 {
			if _, b, amt, err = UnpackDir(b, ufs.Dotu); err != nil {
				break
			} else {
				i++
				offset += uint64(amt)
			}
		}
	}
	if i != numDir {
		t.Fatalf("Reading %v: got %d entries, wanted %d", dir, i, numDir)
	}

	// Alternate form, using readdir and File
	var dirfile *File
	if dirfile, err = clnt.FOpen(".", OREAD); err != nil {
		t.Fatalf("%v", err)
	}
	i, amt, offset = 0, 0, 0
	for i < numDir {
		if d, err := dirfile.Readdir(numDir); err != nil {
			t.Fatalf("%v", err)
		} else {
			i += len(d)
		}
	}
	if i != numDir {
		t.Fatalf("Readdir %v: got %d entries, wanted %d", dir, i, numDir)
	}

	// now test partial reads.
	// Read 128 bytes at a time. Remember the last successful offset.
	// if UnpackDir fails, read again from that offset
	t.Logf("NOW TRY PARTIAL")
	i, amt, offset = 0, 0, 0
	for {
		var b []byte
		var d *Dir
		if b, err = clnt.Read(dirfid, offset, 128); err != nil {
			t.Fatalf("%v", err)
		}
		if len(b) == 0 {
			break
		}
		t.Logf("b %v\n", b)
		for b != nil && len(b) > 0 {
			t.Logf("len(b) %v\n", len(b))
			if d, b, amt, err = UnpackDir(b, ufs.Dotu); err != nil {
				// this error is expected ...
				t.Logf("unpack failed (it's ok!). retry at offset %v\n",
					offset)
				break
			} else {
				t.Logf("d %v\n", d)
				offset += uint64(amt)
			}
		}
	}

	t.Logf("NOW TRY WAY TOO SMALL")
	i, amt, offset = 0, 0, 0
	for {
		var b []byte
		if b, err = clnt.Read(dirfid, offset, 32); err != nil {
			t.Logf("dirread fails as expected: %v\n", err)
			break
		}
		if offset == 0 && len(b) == 0 {
			t.Fatalf("too short dirread returns 0 (no error)")
		}
		if len(b) == 0 {
			break
		}
		// todo: add entry accumulation and validation here..
		offset += uint64(len(b))
	}
}
