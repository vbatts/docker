package utils

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"testing"
)

func TestBufReader(t *testing.T) {
	reader, writer := io.Pipe()
	bufreader := NewBufReader(reader)

	// Write everything down to a Pipe
	// Usually, a pipe should block but because of the buffered reader,
	// the writes will go through
	done := make(chan bool)
	go func() {
		writer.Write([]byte("hello world"))
		writer.Close()
		done <- true
	}()

	// Drain the reader *after* everything has been written, just to verify
	// it is indeed buffering
	<-done
	output, err := ioutil.ReadAll(bufreader)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(output, []byte("hello world")) {
		t.Error(string(output))
	}
}

type dummyWriter struct {
	buffer      bytes.Buffer
	failOnWrite bool
}

func (dw *dummyWriter) Write(p []byte) (n int, err error) {
	if dw.failOnWrite {
		return 0, errors.New("Fake fail")
	}
	return dw.buffer.Write(p)
}

func (dw *dummyWriter) String() string {
	return dw.buffer.String()
}

func (dw *dummyWriter) Close() error {
	return nil
}

func TestWriteBroadcaster(t *testing.T) {
	writer := NewWriteBroadcaster()

	// Test 1: Both bufferA and bufferB should contain "foo"
	bufferA := &dummyWriter{}
	writer.AddWriter(bufferA, "")
	bufferB := &dummyWriter{}
	writer.AddWriter(bufferB, "")
	writer.Write([]byte("foo"))

	if bufferA.String() != "foo" {
		t.Errorf("Buffer contains %v", bufferA.String())
	}

	if bufferB.String() != "foo" {
		t.Errorf("Buffer contains %v", bufferB.String())
	}

	// Test2: bufferA and bufferB should contain "foobar",
	// while bufferC should only contain "bar"
	bufferC := &dummyWriter{}
	writer.AddWriter(bufferC, "")
	writer.Write([]byte("bar"))

	if bufferA.String() != "foobar" {
		t.Errorf("Buffer contains %v", bufferA.String())
	}

	if bufferB.String() != "foobar" {
		t.Errorf("Buffer contains %v", bufferB.String())
	}

	if bufferC.String() != "bar" {
		t.Errorf("Buffer contains %v", bufferC.String())
	}

	// Test3: Test eviction on failure
	bufferA.failOnWrite = true
	writer.Write([]byte("fail"))
	if bufferA.String() != "foobar" {
		t.Errorf("Buffer contains %v", bufferA.String())
	}
	if bufferC.String() != "barfail" {
		t.Errorf("Buffer contains %v", bufferC.String())
	}
	// Even though we reset the flag, no more writes should go in there
	bufferA.failOnWrite = false
	writer.Write([]byte("test"))
	if bufferA.String() != "foobar" {
		t.Errorf("Buffer contains %v", bufferA.String())
	}
	if bufferC.String() != "barfailtest" {
		t.Errorf("Buffer contains %v", bufferC.String())
	}

	writer.CloseWriters()
}

type devNullCloser int

func (d devNullCloser) Close() error {
	return nil
}

func (d devNullCloser) Write(buf []byte) (int, error) {
	return len(buf), nil
}

// This test checks for races. It is only useful when run with the race detector.
func TestRaceWriteBroadcaster(t *testing.T) {
	writer := NewWriteBroadcaster()
	c := make(chan bool)
	go func() {
		writer.AddWriter(devNullCloser(0), "")
		c <- true
	}()
	writer.Write([]byte("hello"))
	<-c
}

// Test the behavior of TruncIndex, an index for querying IDs from a non-conflicting prefix.
func TestTruncIndex(t *testing.T) {
	index := NewTruncIndex()
	// Get on an empty index
	if _, err := index.Get("foobar"); err == nil {
		t.Fatal("Get on an empty index should return an error")
	}

	// Spaces should be illegal in an id
	if err := index.Add("I have a space"); err == nil {
		t.Fatalf("Adding an id with ' ' should return an error")
	}

	id := "99b36c2c326ccc11e726eee6ee78a0baf166ef96"
	// Add an id
	if err := index.Add(id); err != nil {
		t.Fatal(err)
	}
	// Get a non-existing id
	assertIndexGet(t, index, "abracadabra", "", true)
	// Get the exact id
	assertIndexGet(t, index, id, id, false)
	// The first letter should match
	assertIndexGet(t, index, id[:1], id, false)
	// The first half should match
	assertIndexGet(t, index, id[:len(id)/2], id, false)
	// The second half should NOT match
	assertIndexGet(t, index, id[len(id)/2:], "", true)

	id2 := id[:6] + "blabla"
	// Add an id
	if err := index.Add(id2); err != nil {
		t.Fatal(err)
	}
	// Both exact IDs should work
	assertIndexGet(t, index, id, id, false)
	assertIndexGet(t, index, id2, id2, false)

	// 6 characters or less should conflict
	assertIndexGet(t, index, id[:6], "", true)
	assertIndexGet(t, index, id[:4], "", true)
	assertIndexGet(t, index, id[:1], "", true)

	// 7 characters should NOT conflict
	assertIndexGet(t, index, id[:7], id, false)
	assertIndexGet(t, index, id2[:7], id2, false)

	// Deleting a non-existing id should return an error
	if err := index.Delete("non-existing"); err == nil {
		t.Fatalf("Deleting a non-existing id should return an error")
	}

	// Deleting id2 should remove conflicts
	if err := index.Delete(id2); err != nil {
		t.Fatal(err)
	}
	// id2 should no longer work
	assertIndexGet(t, index, id2, "", true)
	assertIndexGet(t, index, id2[:7], "", true)
	assertIndexGet(t, index, id2[:11], "", true)

	// conflicts between id and id2 should be gone
	assertIndexGet(t, index, id[:6], id, false)
	assertIndexGet(t, index, id[:4], id, false)
	assertIndexGet(t, index, id[:1], id, false)

	// non-conflicting substrings should still not conflict
	assertIndexGet(t, index, id[:7], id, false)
	assertIndexGet(t, index, id[:15], id, false)
	assertIndexGet(t, index, id, id, false)
}

func assertIndexGet(t *testing.T, index *TruncIndex, input, expectedResult string, expectError bool) {
	if result, err := index.Get(input); err != nil && !expectError {
		t.Fatalf("Unexpected error getting '%s': %s", input, err)
	} else if err == nil && expectError {
		t.Fatalf("Getting '%s' should return an error", input)
	} else if result != expectedResult {
		t.Fatalf("Getting '%s' returned '%s' instead of '%s'", input, result, expectedResult)
	}
}

func assertKernelVersion(t *testing.T, a, b *KernelVersionInfo, result int) {
	if r := CompareKernelVersion(a, b); r != result {
		t.Fatalf("Unexpected kernel version comparison result. Found %d, expected %d", r, result)
	}
}

func TestCompareKernelVersion(t *testing.T) {
	assertKernelVersion(t,
		&KernelVersionInfo{Kernel: 3, Major: 8, Minor: 0},
		&KernelVersionInfo{Kernel: 3, Major: 8, Minor: 0},
		0)
	assertKernelVersion(t,
		&KernelVersionInfo{Kernel: 2, Major: 6, Minor: 0},
		&KernelVersionInfo{Kernel: 3, Major: 8, Minor: 0},
		-1)
	assertKernelVersion(t,
		&KernelVersionInfo{Kernel: 3, Major: 8, Minor: 0},
		&KernelVersionInfo{Kernel: 2, Major: 6, Minor: 0},
		1)
	assertKernelVersion(t,
		&KernelVersionInfo{Kernel: 3, Major: 8, Minor: 0},
		&KernelVersionInfo{Kernel: 3, Major: 8, Minor: 0},
		0)
	assertKernelVersion(t,
		&KernelVersionInfo{Kernel: 3, Major: 8, Minor: 5},
		&KernelVersionInfo{Kernel: 3, Major: 8, Minor: 0},
		1)
	assertKernelVersion(t,
		&KernelVersionInfo{Kernel: 3, Major: 0, Minor: 20},
		&KernelVersionInfo{Kernel: 3, Major: 8, Minor: 0},
		-1)
}

func TestHumanSize(t *testing.T) {

	size := strings.Trim(HumanSize(1000), " \t")
	expect := "1 kB"
	if size != expect {
		t.Errorf("1000 -> expected '%s', got '%s'", expect, size)
	}

	size = strings.Trim(HumanSize(1024), " \t")
	expect = "1.024 kB"
	if size != expect {
		t.Errorf("1024 -> expected '%s', got '%s'", expect, size)
	}
}

func TestRAMInBytes(t *testing.T) {
	assertRAMInBytes(t, "32", false, 32)
	assertRAMInBytes(t, "32b", false, 32)
	assertRAMInBytes(t, "32B", false, 32)
	assertRAMInBytes(t, "32k", false, 32*1024)
	assertRAMInBytes(t, "32K", false, 32*1024)
	assertRAMInBytes(t, "32kb", false, 32*1024)
	assertRAMInBytes(t, "32Kb", false, 32*1024)
	assertRAMInBytes(t, "32Mb", false, 32*1024*1024)
	assertRAMInBytes(t, "32Gb", false, 32*1024*1024*1024)

	assertRAMInBytes(t, "", true, -1)
	assertRAMInBytes(t, "hello", true, -1)
	assertRAMInBytes(t, "-32", true, -1)
	assertRAMInBytes(t, " 32 ", true, -1)
	assertRAMInBytes(t, "32 mb", true, -1)
	assertRAMInBytes(t, "32m b", true, -1)
	assertRAMInBytes(t, "32bm", true, -1)
}

func assertRAMInBytes(t *testing.T, size string, expectError bool, expectedBytes int64) {
	actualBytes, err := RAMInBytes(size)
	if (err != nil) && !expectError {
		t.Errorf("Unexpected error parsing '%s': %s", size, err)
	}
	if (err == nil) && expectError {
		t.Errorf("Expected to get an error parsing '%s', but got none (bytes=%d)", size, actualBytes)
	}
	if actualBytes != expectedBytes {
		t.Errorf("Expected '%s' to parse as %d bytes, got %d", size, expectedBytes, actualBytes)
	}
}

func TestParseHost(t *testing.T) {
	var (
		defaultHttpHost = "127.0.0.1"
		defaultUnix     = "/var/run/docker.sock"
	)
	if addr, err := ParseHost(defaultHttpHost, defaultUnix, "0.0.0.0"); err == nil {
		t.Errorf("tcp 0.0.0.0 address expected error return, but err == nil, got %s", addr)
	}
	if addr, err := ParseHost(defaultHttpHost, defaultUnix, "tcp://"); err == nil {
		t.Errorf("default tcp:// address expected error return, but err == nil, got %s", addr)
	}
	if addr, err := ParseHost(defaultHttpHost, defaultUnix, "0.0.0.1:5555"); err != nil || addr != "tcp://0.0.0.1:5555" {
		t.Errorf("0.0.0.1:5555 -> expected tcp://0.0.0.1:5555, got %s", addr)
	}
	if addr, err := ParseHost(defaultHttpHost, defaultUnix, ":6666"); err != nil || addr != "tcp://127.0.0.1:6666" {
		t.Errorf(":6666 -> expected tcp://127.0.0.1:6666, got %s", addr)
	}
	if addr, err := ParseHost(defaultHttpHost, defaultUnix, "tcp://:7777"); err != nil || addr != "tcp://127.0.0.1:7777" {
		t.Errorf("tcp://:7777 -> expected tcp://127.0.0.1:7777, got %s", addr)
	}
	if addr, err := ParseHost(defaultHttpHost, defaultUnix, ""); err != nil || addr != "unix:///var/run/docker.sock" {
		t.Errorf("empty argument -> expected unix:///var/run/docker.sock, got %s", addr)
	}
	if addr, err := ParseHost(defaultHttpHost, defaultUnix, "unix:///var/run/docker.sock"); err != nil || addr != "unix:///var/run/docker.sock" {
		t.Errorf("unix:///var/run/docker.sock -> expected unix:///var/run/docker.sock, got %s", addr)
	}
	if addr, err := ParseHost(defaultHttpHost, defaultUnix, "unix://"); err != nil || addr != "unix:///var/run/docker.sock" {
		t.Errorf("unix:///var/run/docker.sock -> expected unix:///var/run/docker.sock, got %s", addr)
	}
	if addr, err := ParseHost(defaultHttpHost, defaultUnix, "udp://127.0.0.1"); err == nil {
		t.Errorf("udp protocol address expected error return, but err == nil. Got %s", addr)
	}
	if addr, err := ParseHost(defaultHttpHost, defaultUnix, "udp://127.0.0.1:4243"); err == nil {
		t.Errorf("udp protocol address expected error return, but err == nil. Got %s", addr)
	}
}

func TestParseRepositoryTag(t *testing.T) {
	if repo, tag := ParseRepositoryTag("root"); repo != "root" || tag != "" {
		t.Errorf("Expected repo: '%s' and tag: '%s', got '%s' and '%s'", "root", "", repo, tag)
	}
	if repo, tag := ParseRepositoryTag("root:tag"); repo != "root" || tag != "tag" {
		t.Errorf("Expected repo: '%s' and tag: '%s', got '%s' and '%s'", "root", "tag", repo, tag)
	}
	if repo, tag := ParseRepositoryTag("user/repo"); repo != "user/repo" || tag != "" {
		t.Errorf("Expected repo: '%s' and tag: '%s', got '%s' and '%s'", "user/repo", "", repo, tag)
	}
	if repo, tag := ParseRepositoryTag("user/repo:tag"); repo != "user/repo" || tag != "tag" {
		t.Errorf("Expected repo: '%s' and tag: '%s', got '%s' and '%s'", "user/repo", "tag", repo, tag)
	}
	if repo, tag := ParseRepositoryTag("url:5000/repo"); repo != "url:5000/repo" || tag != "" {
		t.Errorf("Expected repo: '%s' and tag: '%s', got '%s' and '%s'", "url:5000/repo", "", repo, tag)
	}
	if repo, tag := ParseRepositoryTag("url:5000/repo:tag"); repo != "url:5000/repo" || tag != "tag" {
		t.Errorf("Expected repo: '%s' and tag: '%s', got '%s' and '%s'", "url:5000/repo", "tag", repo, tag)
	}
}

func TestGetResolvConf(t *testing.T) {
	resolvConfUtils, err := GetResolvConf()
	if err != nil {
		t.Fatal(err)
	}
	resolvConfSystem, err := ioutil.ReadFile("/etc/resolv.conf")
	if err != nil {
		t.Fatal(err)
	}
	if string(resolvConfUtils) != string(resolvConfSystem) {
		t.Fatalf("/etc/resolv.conf and GetResolvConf have different content.")
	}
}

func TestCheckLocalDns(t *testing.T) {
	for resolv, result := range map[string]bool{`# Dynamic
nameserver 10.0.2.3
search dotcloud.net`: false,
		`# Dynamic
#nameserver 127.0.0.1
nameserver 10.0.2.3
search dotcloud.net`: false,
		`# Dynamic
nameserver 10.0.2.3 #not used 127.0.1.1
search dotcloud.net`: false,
		`# Dynamic
#nameserver 10.0.2.3
#search dotcloud.net`: true,
		`# Dynamic
nameserver 127.0.0.1
search dotcloud.net`: true,
		`# Dynamic
nameserver 127.0.1.1
search dotcloud.net`: true,
		`# Dynamic
`: true,
		``: true,
	} {
		if CheckLocalDns([]byte(resolv)) != result {
			t.Fatalf("Wrong local dns detection: {%s} should be %v", resolv, result)
		}
	}
}

func assertParseRelease(t *testing.T, release string, b *KernelVersionInfo, result int) {
	var (
		a *KernelVersionInfo
	)
	a, _ = ParseRelease(release)

	if r := CompareKernelVersion(a, b); r != result {
		t.Fatalf("Unexpected kernel version comparison result. Found %d, expected %d", r, result)
	}
	if a.Flavor != b.Flavor {
		t.Fatalf("Unexpected parsed kernel flavor.  Found %s, expected %s", a.Flavor, b.Flavor)
	}
}

func TestParseRelease(t *testing.T) {
	assertParseRelease(t, "3.8.0", &KernelVersionInfo{Kernel: 3, Major: 8, Minor: 0}, 0)
	assertParseRelease(t, "3.4.54.longterm-1", &KernelVersionInfo{Kernel: 3, Major: 4, Minor: 54, Flavor: ".longterm-1"}, 0)
	assertParseRelease(t, "3.4.54.longterm-1", &KernelVersionInfo{Kernel: 3, Major: 4, Minor: 54, Flavor: ".longterm-1"}, 0)
	assertParseRelease(t, "3.8.0-19-generic", &KernelVersionInfo{Kernel: 3, Major: 8, Minor: 0, Flavor: "-19-generic"}, 0)
	assertParseRelease(t, "3.12.8tag", &KernelVersionInfo{Kernel: 3, Major: 12, Minor: 8, Flavor: "tag"}, 0)
	assertParseRelease(t, "3.12-1-amd64", &KernelVersionInfo{Kernel: 3, Major: 12, Minor: 0, Flavor: "-1-amd64"}, 0)
}

func TestParsePortMapping(t *testing.T) {
	data, err := PartParser("ip:public:private", "192.168.1.1:80:8080")
	if err != nil {
		t.Fatal(err)
	}

	if len(data) != 3 {
		t.FailNow()
	}
	if data["ip"] != "192.168.1.1" {
		t.Fail()
	}
	if data["public"] != "80" {
		t.Fail()
	}
	if data["private"] != "8080" {
		t.Fail()
	}
}

func TestGetNameservers(t *testing.T) {
	for resolv, result := range map[string][]string{`
nameserver 1.2.3.4
nameserver 40.3.200.10
search example.com`: {"1.2.3.4", "40.3.200.10"},
		`search example.com`: {},
		`nameserver 1.2.3.4
search example.com
nameserver 4.30.20.100`: {"1.2.3.4", "4.30.20.100"},
		``: {},
		`  nameserver 1.2.3.4   `: {"1.2.3.4"},
		`search example.com
nameserver 1.2.3.4
#nameserver 4.3.2.1`: {"1.2.3.4"},
		`search example.com
nameserver 1.2.3.4 # not 4.3.2.1`: {"1.2.3.4"},
	} {
		test := GetNameservers([]byte(resolv))
		if !StrSlicesEqual(test, result) {
			t.Fatalf("Wrong nameserver string {%s} should be %v. Input: %s", test, result, resolv)
		}
	}
}

func TestGetNameserversAsCIDR(t *testing.T) {
	for resolv, result := range map[string][]string{`
nameserver 1.2.3.4
nameserver 40.3.200.10
search example.com`: {"1.2.3.4/32", "40.3.200.10/32"},
		`search example.com`: {},
		`nameserver 1.2.3.4
search example.com
nameserver 4.30.20.100`: {"1.2.3.4/32", "4.30.20.100/32"},
		``: {},
		`  nameserver 1.2.3.4   `: {"1.2.3.4/32"},
		`search example.com
nameserver 1.2.3.4
#nameserver 4.3.2.1`: {"1.2.3.4/32"},
		`search example.com
nameserver 1.2.3.4 # not 4.3.2.1`: {"1.2.3.4/32"},
	} {
		test := GetNameserversAsCIDR([]byte(resolv))
		if !StrSlicesEqual(test, result) {
			t.Fatalf("Wrong nameserver string {%s} should be %v. Input: %s", test, result, resolv)
		}
	}
}

func TestGetSearchDomains(t *testing.T) {
	for resolv, result := range map[string][]string{
		`search example.com`:           {"example.com"},
		`search example.com # ignored`: {"example.com"},
		` 	  search 	 example.com 	  `: {"example.com"},
		` 	  search 	 example.com 	  # ignored`: {"example.com"},
		`search foo.example.com example.com`: {"foo.example.com", "example.com"},
		`	   search   	   foo.example.com 	 example.com 	`: {"foo.example.com", "example.com"},
		`	   search   	   foo.example.com 	 example.com 	# ignored`: {"foo.example.com", "example.com"},
		``:          {},
		`# ignored`: {},
		`nameserver 1.2.3.4
search foo.example.com example.com`: {"foo.example.com", "example.com"},
		`nameserver 1.2.3.4
search dup1.example.com dup2.example.com
search foo.example.com example.com`: {"foo.example.com", "example.com"},
		`nameserver 1.2.3.4
search foo.example.com example.com
nameserver 4.30.20.100`: {"foo.example.com", "example.com"},
	} {
		test := GetSearchDomains([]byte(resolv))
		if !StrSlicesEqual(test, result) {
			t.Fatalf("Wrong search domain string {%s} should be %v. Input: %s", test, result, resolv)
		}
	}
}

func StrSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	for i, v := range a {
		if v != b[i] {
			return false
		}
	}

	return true
}

func TestReplaceAndAppendEnvVars(t *testing.T) {
	var (
		d = []string{"HOME=/"}
		o = []string{"HOME=/root", "TERM=xterm"}
	)

	env := ReplaceOrAppendEnvValues(d, o)
	if len(env) != 2 {
		t.Fatalf("expected len of 2 got %d", len(env))
	}
	if env[0] != "HOME=/root" {
		t.Fatalf("expected HOME=/root got '%s'", env[0])
	}
	if env[1] != "TERM=xterm" {
		t.Fatalf("expected TERM=xterm got '%s'", env[1])
	}
}

// Reading a symlink to a directory must return the directory
func TestReadSymlinkedDirectoryExistingDirectory(t *testing.T) {
	var err error
	if err = os.Mkdir("/tmp/testReadSymlinkToExistingDirectory", 0777); err != nil {
		t.Errorf("failed to create directory: %s", err)
	}

	if err = os.Symlink("/tmp/testReadSymlinkToExistingDirectory", "/tmp/dirLinkTest"); err != nil {
		t.Errorf("failed to create symlink: %s", err)
	}

	var path string
	if path, err = ReadSymlinkedDirectory("/tmp/dirLinkTest"); err != nil {
		t.Fatalf("failed to read symlink to directory: %s", err)
	}

	if path != "/tmp/testReadSymlinkToExistingDirectory" {
		t.Fatalf("symlink returned unexpected directory: %s", path)
	}

	if err = os.Remove("/tmp/testReadSymlinkToExistingDirectory"); err != nil {
		t.Errorf("failed to remove temporary directory: %s", err)
	}

	if err = os.Remove("/tmp/dirLinkTest"); err != nil {
		t.Errorf("failed to remove symlink: %s", err)
	}
}

// Reading a non-existing symlink must fail
func TestReadSymlinkedDirectoryNonExistingSymlink(t *testing.T) {
	var path string
	var err error
	if path, err = ReadSymlinkedDirectory("/tmp/test/foo/Non/ExistingPath"); err == nil {
		t.Fatalf("error expected for non-existing symlink")
	}

	if path != "" {
		t.Fatalf("expected empty path, but '%s' was returned", path)
	}
}

// Reading a symlink to a file must fail
func TestReadSymlinkedDirectoryToFile(t *testing.T) {
	var err error
	var file *os.File

	if file, err = os.Create("/tmp/testReadSymlinkToFile"); err != nil {
		t.Fatalf("failed to create file: %s", err)
	}

	file.Close()

	if err = os.Symlink("/tmp/testReadSymlinkToFile", "/tmp/fileLinkTest"); err != nil {
		t.Errorf("failed to create symlink: %s", err)
	}

	var path string
	if path, err = ReadSymlinkedDirectory("/tmp/fileLinkTest"); err == nil {
		t.Fatalf("ReadSymlinkedDirectory on a symlink to a file should've failed")
	}

	if path != "" {
		t.Fatalf("path should've been empty: %s", path)
	}

	if err = os.Remove("/tmp/testReadSymlinkToFile"); err != nil {
		t.Errorf("failed to remove file: %s", err)
	}

	if err = os.Remove("/tmp/fileLinkTest"); err != nil {
		t.Errorf("failed to remove symlink: %s", err)
	}
}

func TestADDpaths(t *testing.T) {
	var set = []pathCheck{
		{"http://foo.com/image", true, IsURL},
		{"https://foo.com/image.tar", true, IsURL},
		{"foo.com/image.tar", false, IsURL},

		{"git://github.com/dotcloud/docker", true, IsGIT},
		{"github.com/dotcloud/docker", true, IsGIT},
		{"https://foo.com/foo.git", true, IsGIT},
		{"foo.com/repo.git", false, IsGIT},
		{"git@github.com:user/repo", true, IsGIT},
		{"git@github.com:user/repo.git", true, IsGIT},
		{"git@github.com/user/repo", false, IsGIT},
		{"git@github.com/user/repo.git", false, IsGIT},
		{"git@foo.com/repo", false, IsGIT},

		{"image://namespace/name:/build/file", true, IsIMAGE},
		{"image://name:/build/file", true, IsIMAGE},
		{"image://registry.com/namespace/name:/build/file", true, IsIMAGE},
		{"image://registry.com:5000/namespace/name:/build/file", true, IsIMAGE},
		{"image://registry.com:5000/namespace/name:/build/dir/", true, IsIMAGE},
		{"namespace/name:/build/file", false, IsIMAGE},
	}

	for _, item := range set {
		if item.meth(item.value) != item.result {
			t.Errorf("%s was supposed to return %q, but did not", item.value, item.result)
		}
	}
}

type pathCheck struct {
	value  string
	result bool
	meth   func(str string) bool
}

func TestImageURI(t *testing.T) {
	var set = []struct {
		value    string
		result   map[string]string
		hasError bool
	}{
		{"image://namespace/name:/build/file", map[string]string{"name": "namespace/name", "path": "/build/file"}, false},
		{"image://name:/build/file", map[string]string{"name": "name", "path": "/build/file"}, false},
		{"image://registry.com/namespace/name:/build/file", map[string]string{"name": "registry.com/namespace/name", "path": "/build/file"}, false},
		{"image://registry.com:5000/namespace/name:/build/file", map[string]string{"name": "registry.com:5000/namespace/name", "path": "/build/file"}, false},
		{"image://registry.com:5000/namespace/name:/build/dir/", map[string]string{"name": "registry.com:5000/namespace/name", "path": "/build/dir/"}, false},
		{"namespace/name:/build/file", map[string]string{}, true},
	}

	for _, item := range set {
		r, err := ParseImageURI(item.value)
		if err == nil && item.hasError {
			t.Errorf("%s should have errored", item.value)
			continue
		} else if err != nil && !item.hasError {
			t.Errorf("%s should _not_ have errored: %s", item.value, err)
			continue
		} else if err != nil && item.hasError {
			continue
		}
		if r["name"] != item.result["name"] || r["path"] != item.result["path"] {
			t.Errorf("%q should match %q", r, item.result)
		}
	}
}
