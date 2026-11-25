package core

import (
	"bufio"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type hlsDownloader struct {
	ctx      context.Context
	cancel   context.CancelFunc
	url      string
	fileName string
	headers  map[string]string
	client   *http.Client
}

func NewHLSDownloader(rawUrl, filename string, headers map[string]string) *hlsDownloader {
	ctx, cancel := context.WithCancel(context.Background())
	return &hlsDownloader{
		ctx:      ctx,
		cancel:   cancel,
		url:      rawUrl,
		fileName: filename,
		headers:  hlsHeaders(headers),
		client:   &http.Client{Timeout: 60 * time.Second},
	}
}

func (h *hlsDownloader) Cancel() {
	if h.cancel != nil {
		h.cancel()
	}
}

func (h *hlsDownloader) Start(progress func(done, total int)) error {
	playlist, err := h.loadMediaPlaylist(h.url)
	if err != nil {
		return err
	}
	file, err := os.Create(h.fileName)
	if err != nil {
		return err
	}
	defer file.Close()

	var key []byte
	var iv []byte
	seg := 0
	total := 0
	for _, line := range playlist.lines {
		if !strings.HasPrefix(line, "#") && strings.TrimSpace(line) != "" {
			total++
		}
	}
	for _, line := range playlist.lines {
		if err := h.ctx.Err(); err != nil {
			return fmt.Errorf("cancelled")
		}
		if strings.HasPrefix(line, "#EXT-X-KEY:") {
			key, iv, err = h.parseKey(line, playlist.base)
			if err != nil {
				return err
			}
			continue
		}
		if strings.HasPrefix(line, "#") || strings.TrimSpace(line) == "" {
			continue
		}
		segURL := resolveURL(playlist.base, strings.TrimSpace(line))
		data, err := h.get(segURL)
		if err != nil {
			return err
		}
		if key != nil {
			data, err = decryptAES128(data, key, segmentIV(iv, playlist.sequence+seg))
			if err != nil {
				return err
			}
		}
		if _, err := file.Write(data); err != nil {
			return err
		}
		seg++
		if progress != nil {
			progress(seg, total)
		}
	}
	return nil
}

type mediaPlaylist struct {
	base     *url.URL
	lines    []string
	sequence int
}

func (h *hlsDownloader) loadMediaPlaylist(rawUrl string) (*mediaPlaylist, error) {
	body, err := h.get(rawUrl)
	if err != nil {
		return nil, err
	}
	base, err := url.Parse(rawUrl)
	if err != nil {
		return nil, err
	}
	text := string(body)
	if strings.Contains(text, "#EXT-X-STREAM-INF") {
		best := bestVariant(text)
		if best == "" {
			return nil, fmt.Errorf("播放列表里没有可下载的视频")
		}
		return h.loadMediaPlaylist(resolveURL(base, best))
	}
	seq := 0
	if i := strings.Index(text, "#EXT-X-MEDIA-SEQUENCE:"); i >= 0 {
		fmt.Sscanf(text[i:], "#EXT-X-MEDIA-SEQUENCE:%d", &seq)
	}
	return &mediaPlaylist{base: base, lines: splitLines(text), sequence: seq}, nil
}

func bestVariant(text string) string {
	bestBandwidth := -1
	bestURL := ""
	lines := splitLines(text)
	for i, line := range lines {
		if !strings.HasPrefix(line, "#EXT-X-STREAM-INF") || i+1 >= len(lines) {
			continue
		}
		bw := 0
		if j := strings.Index(line, "BANDWIDTH="); j >= 0 {
			fmt.Sscanf(line[j:], "BANDWIDTH=%d", &bw)
		}
		next := strings.TrimSpace(lines[i+1])
		if strings.HasPrefix(next, "#") || next == "" {
			continue
		}
		if bw >= bestBandwidth {
			bestBandwidth = bw
			bestURL = next
		}
	}
	return bestURL
}

func (h *hlsDownloader) parseKey(line string, base *url.URL) ([]byte, []byte, error) {
	if strings.Contains(line, "METHOD=NONE") {
		return nil, nil, nil
	}
	if !strings.Contains(line, "METHOD=AES-128") {
		return nil, nil, fmt.Errorf("这个播放列表使用了无法直接保存的加密")
	}
	uri := attr(line, "URI")
	if uri == "" {
		return nil, nil, fmt.Errorf("播放列表缺少解密地址")
	}
	key, err := h.get(resolveURL(base, uri))
	if err != nil {
		return nil, nil, err
	}
	var iv []byte
	if raw := attr(line, "IV"); strings.HasPrefix(raw, "0x") || strings.HasPrefix(raw, "0X") {
		iv, _ = hex.DecodeString(raw[2:])
	}
	return key, iv, nil
}

func attr(line, name string) string {
	token := name + `="`
	if i := strings.Index(line, token); i >= 0 {
		rest := line[i+len(token):]
		if j := strings.Index(rest, `"`); j >= 0 {
			return rest[:j]
		}
	}
	token = name + "="
	if i := strings.Index(line, token); i >= 0 {
		rest := line[i+len(token):]
		if j := strings.IndexAny(rest, ","); j >= 0 {
			return rest[:j]
		}
		return rest
	}
	return ""
}

func segmentIV(explicit []byte, sequence int) []byte {
	if len(explicit) == aes.BlockSize {
		return explicit
	}
	iv := make([]byte, aes.BlockSize)
	iv[12] = byte(sequence >> 24)
	iv[13] = byte(sequence >> 16)
	iv[14] = byte(sequence >> 8)
	iv[15] = byte(sequence)
	return iv
}

func decryptAES128(data, key, iv []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 || len(data)%aes.BlockSize != 0 {
		return nil, fmt.Errorf("加密分片长度不正确")
	}
	out := make([]byte, len(data))
	cipher.NewCBCDecrypter(block, iv).CryptBlocks(out, data)
	return pkcs7Unpad(out), nil
}

func pkcs7Unpad(data []byte) []byte {
	if len(data) == 0 {
		return data
	}
	pad := int(data[len(data)-1])
	if pad == 0 || pad > aes.BlockSize || pad > len(data) {
		return data
	}
	return data[:len(data)-pad]
}

func (h *hlsDownloader) get(rawUrl string) ([]byte, error) {
	req, err := http.NewRequestWithContext(h.ctx, http.MethodGet, rawUrl, nil)
	if err != nil {
		return nil, err
	}
	for k, v := range h.headers {
		req.Header.Set(k, v)
	}
	resp, err := h.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		return nil, fmt.Errorf("下载失败 %s", resp.Status)
	}
	return io.ReadAll(resp.Body)
}

func hlsHeaders(headers map[string]string) map[string]string {
	keep := map[string]string{}
	for _, name := range []string{"User-Agent", "Referer", "Cookie", "Origin"} {
		for k, v := range headers {
			if strings.EqualFold(k, name) && v != "" {
				keep[name] = v
			}
		}
	}
	return keep
}

func resolveURL(base *url.URL, ref string) string {
	u, err := url.Parse(ref)
	if err != nil || base == nil {
		return ref
	}
	return base.ResolveReference(u).String()
}

func splitLines(text string) []string {
	var lines []string
	sc := bufio.NewScanner(strings.NewReader(text))
	for sc.Scan() {
		lines = append(lines, strings.TrimSpace(sc.Text()))
	}
	return lines
}
