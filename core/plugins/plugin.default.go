package plugins

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"path/filepath"
	"regexp"
	"res-downloader/core/shared"
	"strconv"
	"strings"

	"github.com/elazarl/goproxy"
	gonanoid "github.com/matoous/go-nanoid/v2"
)

type DefaultPlugin struct {
	bridge *shared.Bridge
}

func (p *DefaultPlugin) SetBridge(bridge *shared.Bridge) {
	p.bridge = bridge
}

func (p *DefaultPlugin) Domains() []string {
	return []string{"default"}
}

func (p *DefaultPlugin) OnRequest(r *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
	return r, nil
}

func (p *DefaultPlugin) OnResponse(resp *http.Response, ctx *goproxy.ProxyCtx) *http.Response {
	if resp == nil || resp.Request == nil || (resp.StatusCode != 200 && resp.StatusCode != 206 && resp.StatusCode != 304) {
		return resp
	}

	rawUrl := resp.Request.URL.String()
	if isSegmentURL(rawUrl) || strings.Contains(strings.ToLower(resp.Header.Get("Content-Type")), "video/mp2t") {
		return resp
	}

	classify, suffix := p.bridge.TypeSuffix(resp.Header.Get("Content-Type"))
	if found := playlistOrVideo(resp); found != "" {
		if strings.Contains(found, ".m3u8") {
			classify, suffix = "m3u8", ".m3u8"
			if found != rawUrl {
				p.emitMedia(found, "m3u8", ".m3u8", 0, resp)
				return resp
			}
		} else if classify == "" || classify == "stream" {
			classify, suffix = "video", ".mp4"
			if found != rawUrl {
				p.emitMedia(found, "video", ".mp4", 0, resp)
				return resp
			}
		}
	}
	if classify == "" {
		return resp
	}
	isAll, _ := p.bridge.GetResType("all")
	isClassify, _ := p.bridge.GetResType(classify)

	if suffix == "default" {
		ext := filepath.Ext(filepath.Base(strings.Split(strings.Split(rawUrl, "?")[0], "#")[0]))
		if ext != "" {
			suffix = ext
		}
	}

	urlSign := shared.Md5(rawUrl)
	if ok := p.bridge.MediaIsMarked(urlSign); !ok && (isAll || isClassify) {
		value, _ := strconv.ParseFloat(resp.Header.Get("content-length"), 64)
		id, err := gonanoid.New()
		if err != nil {
			id = urlSign
		}
		res := shared.MediaInfo{
			Id:          id,
			Url:         rawUrl,
			UrlSign:     urlSign,
			CoverUrl:    "",
			Size:        value,
			Domain:      shared.GetTopLevelDomain(rawUrl),
			Classify:    classify,
			Suffix:      suffix,
			Status:      shared.DownloadStatusReady,
			SavePath:    "",
			DecodeKey:   "",
			OtherData:   map[string]string{},
			Description: "",
			ContentType: resp.Header.Get("Content-Type"),
		}

		// Store entire request headers as JSON
		if headers, err := json.Marshal(resp.Request.Header); err == nil {
			res.OtherData["headers"] = string(headers)
		}

		p.bridge.MarkMedia(urlSign)
		go func(res shared.MediaInfo) {
			p.bridge.Send("newResources", res)
		}(res)
	}

	return resp
}

func isSegmentURL(rawUrl string) bool {
	path := strings.ToLower(strings.Split(rawUrl, "?")[0])
	return strings.HasSuffix(path, ".ts") || strings.Contains(path, ".ts/")
}

func playlistOrVideo(resp *http.Response) string {
	if resp == nil || resp.Request == nil {
		return ""
	}
	rawUrl := resp.Request.URL.String()
	path := strings.ToLower(strings.Split(resp.Request.URL.Path, "?" )[0])
	if strings.Contains(path, ".m3u8") || strings.Contains(path, "m3u8") {
		return rawUrl
	}
	if strings.HasSuffix(path, ".mp4") {
		return rawUrl
	}
	if !shouldSniffBody(resp) {
		return ""
	}
	body := peekBody(resp, 64*1024)
	trimmed := bytes.TrimSpace(body)
	if bytes.HasPrefix(trimmed, []byte("#EXTM3U")) {
		return rawUrl
	}
	if len(trimmed) == 0 || (trimmed[0] != '{' && trimmed[0] != '[' && trimmed[0] != '<') {
		return ""
	}
	text := strings.ReplaceAll(string(body), `\/`, `/`)
	text = strings.ReplaceAll(text, `\u0026`, `&`)
	return playURLPattern.FindString(text)
}

var playURLPattern = regexp.MustCompile(`https?://[^"'\\\s<>]+?\.(?:m3u8|mp4)(?:\?[^"'\\\s<>]*)?`)

func shouldSniffBody(resp *http.Response) bool {
	if resp.ContentLength > 2*1024*1024 {
		return false
	}
	ct := strings.ToLower(strings.Split(resp.Header.Get("Content-Type"), ";")[0])
	switch ct {
	case "", "application/octet-stream", "application/json", "text/plain", "text/html",
		"application/vnd.apple.mpegurl", "application/x-mpegurl", "audio/x-mpegurl":
		return true
	default:
		return strings.Contains(ct, "json") || strings.Contains(ct, "mpegurl")
	}
}

func peekBody(resp *http.Response, n int) []byte {
	if resp.Body == nil {
		return nil
	}
	buf := make([]byte, n)
	read, _ := io.ReadFull(resp.Body, buf)
	data := buf[:read]
	resp.Body = io.NopCloser(io.MultiReader(bytes.NewReader(data), resp.Body))
	return data
}

func (p *DefaultPlugin) emitMedia(rawUrl, classify, suffix string, size float64, resp *http.Response) {
	isAll, _ := p.bridge.GetResType("all")
	isClassify, _ := p.bridge.GetResType(classify)
	if !isAll && !isClassify {
		return
	}
	urlSign := shared.Md5(rawUrl)
	if p.bridge.MediaIsMarked(urlSign) {
		return
	}
	id, err := gonanoid.New()
	if err != nil {
		id = urlSign
	}
	res := shared.MediaInfo{
		Id:          id,
		Url:         rawUrl,
		UrlSign:     urlSign,
		Size:        size,
		Domain:      shared.GetTopLevelDomain(rawUrl),
		Classify:    classify,
		Suffix:      suffix,
		Status:      shared.DownloadStatusReady,
		OtherData:   map[string]string{},
		ContentType: resp.Header.Get("Content-Type"),
	}
	if headers, err := json.Marshal(resp.Request.Header); err == nil {
		res.OtherData["headers"] = string(headers)
	}
	p.bridge.MarkMedia(urlSign)
	go p.bridge.Send("newResources", res)
}
