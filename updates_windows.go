//go:build windows

package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const downloadURL = "https://qingying.luming.host/#download"

type updateResult struct {
	version string
	err     error
}

type updateState struct {
	checking   bool
	version    string
	checkErr   bool
	checkedAt  time.Time
	settingErr string
	cancel     context.CancelFunc
	results    chan updateResult
}

func releaseClient() *http.Client {
	return &http.Client{
		Timeout: 8 * time.Second,
		// The official latest-release redirect contains the version. Avoid
		// downloading the release page or using GitHub's rate-limited API.
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
}

func fetchLatestRelease(ctx context.Context, client *http.Client, endpoint string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "SimpleKeymap/"+appVersion)
	response, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode < 300 || response.StatusCode >= 400 {
		return "", fmt.Errorf("unexpected release response: %d", response.StatusCode)
	}
	target, err := response.Location()
	if err != nil || target.Scheme != "https" || target.Host != "github.com" || target.User != nil || target.RawQuery != "" || target.Fragment != "" {
		return "", fmt.Errorf("unexpected release location")
	}
	prefix := "/indiefun/simple-keymap/releases/tag/v"
	version := strings.TrimPrefix(target.Path, prefix)
	if !strings.HasPrefix(target.Path, prefix) || len(version) > 32 || !releaseVersion.MatchString(version) {
		return "", fmt.Errorf("invalid release version")
	}
	return version, nil
}

// Numeric components are compared by length, avoiding integer overflow and
// ensuring that, for example, 1.10.0 is newer than 1.9.0.
func compareReleaseVersions(current, latest string) (int, bool) {
	if !releaseVersion.MatchString(current) || !releaseVersion.MatchString(latest) {
		return 0, false
	}
	a, b := strings.Split(current, "."), strings.Split(latest, ".")
	for i := range a {
		if len(a[i]) < len(b[i]) || len(a[i]) == len(b[i]) && a[i] < b[i] {
			return -1, true
		}
		if len(a[i]) > len(b[i]) || len(a[i]) == len(b[i]) && a[i] > b[i] {
			return 1, true
		}
	}
	return 0, true
}

func (a *application) updateSettings() *updateState {
	if a.updates == nil {
		a.updates = &updateState{}
	}
	return a.updates
}

func (a *application) startUpdateCheck() {
	state := a.updateSettings()
	if state.checking || a.closing {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	state.checking, state.checkErr, state.cancel = true, false, cancel
	results := make(chan updateResult, 1)
	state.results = results
	a.refreshUpdates()
	hwnd := a.hwnd
	go func() {
		version, err := fetchLatestRelease(ctx, releaseClient(), repositoryURL+"/releases/latest")
		if ctx.Err() != nil {
			return
		}
		results <- updateResult{version, err}
		postMessage.Call(hwnd, wmUpdateResult, 0, 0)
	}()
}

func (a *application) receiveUpdateResult() {
	if a.closing || a.updates == nil {
		return
	}
	state := a.updates
	select {
	case result := <-state.results:
		state.checking, state.checkErr = false, result.err != nil
		state.checkedAt = time.Now()
		state.version = result.version
		if state.cancel != nil {
			state.cancel()
			state.cancel = nil
		}
		a.refreshUpdates()
	default:
	}
}

func (a *application) refreshUpdates() {
	label := "检查更新"
	state := a.updateSettings()
	if order, valid := compareReleaseVersions(appVersion, state.version); valid && order < 0 {
		label = "发现新版本"
	}
	setText(a.controls[idUpdates], label)
	if a.info != nil && a.info.page == "updates" {
		a.updateInformation("updates")
	}
}

func (a *application) toggleStartupUpdates() {
	state := a.updateSettings()
	next := a.cfg
	next.CheckUpdatesOnStartup = !next.CheckUpdatesOnStartup
	if err := saveConfig(a.path, next); err != nil {
		state.settingErr = "设置未保存：" + errorCause(err) + "。请处理后再次点击。"
	} else {
		a.cfg, a.configSaveError, state.settingErr = next, "", ""
		a.status = "更新偏好已保存。"
		a.layoutMain()
	}
	a.refreshUpdates()
}

func (a *application) updatesContent() string {
	state := a.updateSettings()
	message := "点击「立即检查」获取最新正式版本。"
	switch {
	case state.checking:
		message = "正在检查更新…"
	case state.checkErr:
		message = "检查失败，请检查网络后重试，也可以直接前往官网下载。"
	case state.version != "":
		order, valid := compareReleaseVersions(appVersion, state.version)
		switch {
		case !valid:
			message = "当前为开发版本，最新正式版为 " + state.version + "。"
		case order < 0:
			message = "发现新版本 " + state.version + "，可前往官网下载。"
		case order == 0:
			message = "当前已是最新正式版本。"
		default:
			message = "当前版本较新，最新公开正式版为 " + state.version + "。"
		}
	}
	body := "当前版本：" + appVersion + "\n\n" + message
	if !state.checkedAt.IsZero() && !state.checking {
		body += "\n检查时间：" + state.checkedAt.Format("2006-01-02 15:04")
	}
	body += "\n\n点击「前往下载」获取新版。关闭旧版后，将新版完整解压运行，已有映射和开关设置会保留。\n\n启用下方选项后，启动时会联网检查版本，发现新版时在顶部提示。更新由你下载并替换。"
	if state.settingErr != "" {
		body += "\n\n" + state.settingErr
	}
	return body
}
