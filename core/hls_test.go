package core

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestHLSDownload(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/high.m3u8", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("#EXTM3U\n#EXTINF:1,\nseg0.ts\n#EXTINF:1,\nseg1.ts\n"))
	})
	mux.HandleFunc("/master.m3u8", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("#EXTM3U\n#EXT-X-STREAM-INF:BANDWIDTH=100\nlow.m3u8\n#EXT-X-STREAM-INF:BANDWIDTH=900\nhigh.m3u8\n"))
	})
	mux.HandleFunc("/low.m3u8", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("#EXTM3U\n#EXTINF:1,\nskip.ts\n"))
	})
	mux.HandleFunc("/seg0.ts", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("AAA")) })
	mux.HandleFunc("/seg1.ts", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("BBB")) })
	srv := httptest.NewServer(mux)
	defer srv.Close()

	path := t.TempDir() + "/out.ts"
	err := NewHLSDownloader(srv.URL+"/master.m3u8", path, nil).Start(nil)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "AAABBB" {
		t.Fatalf("got %q", data)
	}
	if strings.Contains(string(data), "skip") {
		t.Fatal("picked the low variant")
	}
}
