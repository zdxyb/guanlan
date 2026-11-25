package core

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type WxFinderResult struct {
	Path     string      `json:"path"`
	Backup   string      `json:"backup,omitempty"`
	Previous interface{} `json:"previous"`
	Changed  bool        `json:"changed"`
}

var (
	errWechatStillRunning = errors.New("已尝试结束微信，但进程仍在运行，请手动退出后再试")
	errFinderNotFound     = errors.New("未找到包含 featureSwitch.handleFinder 的 engineConfig.json")
)

func disableHandleFinder() (*WxFinderResult, error) {
	return disableHandleFinderIn(wxDataRoots(), true)
}

func disableHandleFinderIn(roots []string, checkProcess bool) (*WxFinderResult, error) {
	hits, err := collectEngineConfigs(roots)
	if err != nil {
		return nil, err
	}
	if len(hits) == 0 {
		return nil, errFinderNotFound
	}

	newest := hits[0]
	for _, hit := range hits[1:] {
		if hit.mod.After(newest.mod) || (hit.mod.Equal(newest.mod) && hit.path > newest.path) {
			newest = hit
		}
	}

	if isFalse(newest.previous) {
		return &WxFinderResult{
			Path:     newest.path,
			Previous: newest.previous,
			Changed:  false,
		}, nil
	}

	if checkProcess {
		if err := closeWechat(); err != nil {
			return nil, err
		}
		hits, err = collectEngineConfigs(roots)
		if err != nil {
			return nil, err
		}
		if len(hits) == 0 {
			return nil, errFinderNotFound
		}
		newest = hits[0]
		for _, hit := range hits[1:] {
			if hit.mod.After(newest.mod) || (hit.mod.Equal(newest.mod) && hit.path > newest.path) {
				newest = hit
			}
		}
		if isFalse(newest.previous) {
			return &WxFinderResult{
				Path:     newest.path,
				Previous: newest.previous,
				Changed:  false,
			}, nil
		}
	}

	backup := newest.path + ".bak." + time.Now().Format("20060102150405")
	if err := copyFile(newest.path, backup); err != nil {
		return nil, fmt.Errorf("备份 engineConfig.json 失败: %w", err)
	}

	newest.doc["featureSwitch"].(map[string]interface{})["handleFinder"] = false
	encoded, err := json.MarshalIndent(newest.doc, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("写入 engineConfig.json 失败: %w", err)
	}
	encoded = append(encoded, '\n')

	perm := os.FileMode(0644)
	if info, statErr := os.Stat(newest.path); statErr == nil {
		perm = info.Mode().Perm()
	}
	tmp := newest.path + ".tmp"
	if err := os.WriteFile(tmp, encoded, perm); err != nil {
		return nil, fmt.Errorf("写入 engineConfig.json 失败: %w", err)
	}
	if err := os.Rename(tmp, newest.path); err != nil {
		_ = os.Remove(newest.path)
		if err2 := os.Rename(tmp, newest.path); err2 != nil {
			_ = os.Remove(tmp)
			return nil, fmt.Errorf("写入 engineConfig.json 失败: %w", err2)
		}
	}

	return &WxFinderResult{
		Path:     newest.path,
		Backup:   backup,
		Previous: newest.previous,
		Changed:  true,
	}, nil
}

type engineConfigFile struct {
	path     string
	mod      time.Time
	doc      map[string]interface{}
	previous interface{}
}

func collectEngineConfigs(roots []string) ([]engineConfigFile, error) {
	seen := map[string]struct{}{}
	var hits []engineConfigFile
	for _, root := range roots {
		info, err := os.Stat(root)
		if err != nil || !info.IsDir() {
			continue
		}
		_ = filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
			if walkErr != nil || d.IsDir() {
				return nil
			}
			if !strings.EqualFold(d.Name(), "engineConfig.json") {
				return nil
			}
			abs, err := filepath.Abs(path)
			if err != nil {
				abs = path
			}
			if _, ok := seen[abs]; ok {
				return nil
			}
			seen[abs] = struct{}{}

			hit, ok, err := readEngineConfig(abs)
			if err != nil || !ok {
				return nil
			}
			hits = append(hits, hit)
			return nil
		})
	}
	return hits, nil
}

func readEngineConfig(path string) (engineConfigFile, bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return engineConfigFile{}, false, err
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	var doc map[string]interface{}
	if err := dec.Decode(&doc); err != nil {
		return engineConfigFile{}, false, nil
	}
	featureSwitch, ok := doc["featureSwitch"].(map[string]interface{})
	if !ok {
		return engineConfigFile{}, false, nil
	}
	previous, ok := featureSwitch["handleFinder"]
	if !ok {
		return engineConfigFile{}, false, nil
	}
	info, err := os.Stat(path)
	if err != nil {
		return engineConfigFile{}, false, err
	}
	return engineConfigFile{
		path:     path,
		mod:      info.ModTime(),
		doc:      doc,
		previous: previous,
	}, true, nil
}

func isFalse(v interface{}) bool {
	b, ok := v.(bool)
	return ok && !b
}

func wxDataRoots() []string {
	home, _ := os.UserHomeDir()
	configDir, _ := os.UserConfigDir()
	var roots []string
	switch runtime.GOOS {
	case "windows":
		if configDir != "" {
			roots = append(roots, filepath.Join(configDir, "Tencent"))
		}
		if local := os.Getenv("LOCALAPPDATA"); local != "" {
			roots = append(roots, filepath.Join(local, "Tencent"))
		}
		if cache, err := os.UserCacheDir(); err == nil && cache != "" {
			roots = append(roots, filepath.Join(cache, "Tencent"))
		}
	case "darwin":
		if home != "" {
			roots = append(roots, filepath.Join(home, "Library", "Containers", "com.tencent.xinWeChat", "Data"))
		}
		roots = append(roots, matchingChildDirs(configDir)...)
	default:
		if home != "" {
			roots = append(roots, filepath.Join(home, ".xwechat"))
			roots = append(roots, filepath.Join(home, ".var", "app", "com.tencent.WeChat"))
		}
		roots = append(roots, matchingChildDirs(configDir)...)
	}
	return uniquePaths(roots)
}

func matchingChildDirs(parent string) []string {
	if parent == "" {
		return nil
	}
	entries, err := os.ReadDir(parent)
	if err != nil {
		return nil
	}
	var out []string
	for _, entry := range entries {
		if entry.IsDir() && nameLooksLikeWechat(entry.Name()) {
			out = append(out, filepath.Join(parent, entry.Name()))
		}
	}
	return out
}

func nameLooksLikeWechat(name string) bool {
	n := strings.ToLower(name)
	return strings.Contains(n, "tencent") || strings.Contains(n, "wechat") || strings.Contains(n, "xwechat") || strings.Contains(n, "weixin")
}

func uniquePaths(paths []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, path := range paths {
		if path == "" {
			continue
		}
		clean := filepath.Clean(path)
		key := strings.ToLower(clean)
		if runtime.GOOS != "windows" {
			key = clean
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, clean)
	}
	return out
}

func closeWechat() error {
	running, err := wechatRunning()
	if err != nil {
		return err
	}
	if !running {
		return nil
	}
	killWechat()
	deadline := time.Now().Add(10 * time.Second)
	for {
		running, err = wechatRunning()
		if err != nil {
			return err
		}
		if !running {
			return nil
		}
		if time.Now().After(deadline) {
			return errWechatStillRunning
		}
		time.Sleep(300 * time.Millisecond)
	}
}

func killWechat() {
	if runtime.GOOS == "windows" {
		for _, name := range []string{"WeChatAppEx.exe", "Weixin.exe", "WeChat.exe"} {
			_ = exec.Command("taskkill", "/F", "/T", "/IM", name).Run()
		}
		return
	}
	for _, name := range []string{"WeChatAppEx", "Weixin", "WeChat", "weixin", "wechat"} {
		_ = exec.Command("pkill", "-x", name).Run()
	}
}

func wechatRunning() (bool, error) {
	names := []string{"Weixin.exe", "WeChat.exe", "WeChatAppEx.exe"}
	if runtime.GOOS != "windows" {
		names = []string{"weixin", "wechat", "wechatappex", "微信"}
	}

	var out []byte
	var err error
	if runtime.GOOS == "windows" {
		out, err = exec.Command("tasklist", "/FO", "CSV", "/NH").Output()
	} else {
		out, err = exec.Command("ps", "-e", "-o", "comm=").Output()
	}
	if err != nil {
		return false, fmt.Errorf("无法检查微信是否在运行: %w", err)
	}

	if runtime.GOOS == "windows" {
		text := strings.ToLower(string(out))
		for _, name := range names {
			if strings.Contains(text, `"`+strings.ToLower(name)+`"`) {
				return true, nil
			}
		}
		return false, nil
	}

	want := map[string]struct{}{}
	for _, name := range names {
		want[strings.ToLower(name)] = struct{}{}
	}
	for _, line := range strings.Split(string(out), "\n") {
		base := strings.ToLower(filepath.Base(strings.TrimSpace(line)))
		if _, ok := want[base]; ok {
			return true, nil
		}
	}
	return false, nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	info, err := in.Stat()
	if err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_EXCL|os.O_WRONLY, info.Mode().Perm())
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}
