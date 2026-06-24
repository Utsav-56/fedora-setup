package downloader

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/schollz/progressbar/v3"
)

// Config

type Config struct {
	Host   string
	Port   string
	Secret string
}

func DefaultConfig() Config {
	return Config{
		Host:   "localhost",
		Port:   "6800",
		Secret: "mysecret",
	}
}

// RPC Types

type rpcRequest struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      string        `json:"id"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
}

type rpcResponse struct {
	ID      string          `json:"id"`
	JSONRPC string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type downloadStatus struct {
	GID             string `json:"gid"`
	Status          string `json:"status"`
	TotalLength     string `json:"totalLength"`
	CompletedLength string `json:"completedLength"`
	DownloadSpeed   string `json:"downloadSpeed"`
	ErrorMessage    string `json:"errorMessage"`
	Files           []struct {
		Path string `json:"path"`
		URIs []struct {
			URI string `json:"uri"`
		} `json:"uris"`
	} `json:"files"`
}

// Aria2 Client

type Aria2Client struct {
	endpoint string
	secret   string
	http     *http.Client
}

func NewAria2Client(cfg Config) *Aria2Client {
	return &Aria2Client{
		endpoint: fmt.Sprintf("http://%s:%s/jsonrpc", cfg.Host, cfg.Port),
		secret:   cfg.Secret,
		http:     &http.Client{Timeout: 15 * time.Second},
	}
}

func (c *Aria2Client) call(method string, params ...interface{}) (json.RawMessage, error) {
	allParams := make([]interface{}, 0, len(params)+1)
	allParams = append(allParams, "token:"+c.secret)
	allParams = append(allParams, params...)

	req := rpcRequest{
		JSONRPC: "2.0",
		ID:      "1",
		Method:  method,
		Params:  allParams,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	resp, err := c.http.Post(c.endpoint, "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var rpcResp rpcResponse
	if err := json.Unmarshal(respBody, &rpcResp); err != nil {
		return nil, err
	}

	if rpcResp.Error != nil {
		return nil, fmt.Errorf("rpc [%d]: %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}

	return rpcResp.Result, nil
}

func (c *Aria2Client) AddURI(uris []string, options map[string]string) (string, error) {
	params := []interface{}{uris}
	if len(options) > 0 {
		params = append(params, options)
	}

	result, err := c.call("aria2.addUri", params...)
	if err != nil {
		return "", err
	}

	var gid string
	json.Unmarshal(result, &gid)
	return gid, nil
}

func (c *Aria2Client) TellStatus(gid string) (*downloadStatus, error) {
	result, err := c.call("aria2.tellStatus", gid)
	if err != nil {
		return nil, err
	}

	var status downloadStatus
	json.Unmarshal(result, &status)
	return &status, nil
}

func (c *Aria2Client) TellActive() ([]downloadStatus, error) {
	result, err := c.call("aria2.tellActive")
	if err != nil {
		return nil, err
	}

	var statuses []downloadStatus
	json.Unmarshal(result, &statuses)
	return statuses, nil
}

func (c *Aria2Client) TellWaiting(offset, num int) ([]downloadStatus, error) {
	result, err := c.call("aria2.tellWaiting", offset, num)
	if err != nil {
		return nil, err
	}

	var statuses []downloadStatus
	json.Unmarshal(result, &statuses)
	return statuses, nil
}

func (c *Aria2Client) TellStopped(offset, num int) ([]downloadStatus, error) {
	result, err := c.call("aria2.tellStopped", offset, num)
	if err != nil {
		return nil, err
	}

	var statuses []downloadStatus
	json.Unmarshal(result, &statuses)
	return statuses, nil
}

func (c *Aria2Client) ChangeGlobalOption(options map[string]string) error {
	_, err := c.call("aria2.changeGlobalOption", options)
	return err
}

func (c *Aria2Client) Pause(gid string) error {
	_, err := c.call("aria2.pause", gid)
	return err
}

func (c *Aria2Client) Unpause(gid string) error {
	_, err := c.call("aria2.unpause", gid)
	return err
}

func (c *Aria2Client) Remove(gid string) error {
	_, err := c.call("aria2.remove", gid)
	return err
}

func (c *Aria2Client) PurgeDownloadResult() error {
	_, err := c.call("aria2.purgeDownloadResult")
	return err
}

// Progress

type Progress struct {
	GID       string
	Name      string // filename or URL if unknown
	Status    string // active, waiting, paused, complete, error, removed
	Completed int64
	Total     int64
	Speed     int64
	Percent   float64
	Error     string
}

func (p Progress) Done() bool {
	return p.Status == "complete" || p.Status == "error" || p.Status == "removed"
}

func (p Progress) IsWaiting() bool {
	return p.Status == "waiting"
}

func (p Progress) IsActive() bool {
	return p.Status == "active"
}

// Task (single download handle)

type Task struct {
	GID      string
	Progress <-chan Progress
	Done     <-chan struct{}
	Err      error
}

func (t *Task) Wait() error {
	<-t.Done
	return t.Err
}

// DownloadRequest

type DownloadRequest struct {
	URL string
	Out string // filename only, e.g. "antigravity.tar.gz"
}

// BatchOptions (mirrors aria2c flags)

type BatchOptions struct {
	// ConcurrentDownloads = -j flag (max simultaneous files)
	ConcurrentDownloads int
	// ConnectionsPerFile = -x and -s flags (chunks per file)
	ConnectionsPerFile int
}

func DefaultBatchOptions() BatchOptions {
	return BatchOptions{
		ConcurrentDownloads: 3,
		ConnectionsPerFile:  16,
	}
}

// BatchResult

type BatchResult struct {
	Tasks   []*Task
	Updates <-chan BatchUpdate
	Done    <-chan struct{}
	Errors  []error
}

func (r *BatchResult) Wait() {
	<-r.Done
}

// BatchUpdate represents all download states at a point in time
type BatchUpdate struct {
	Active  []Progress
	Waiting []Progress
	Done    []Progress
	Failed  []Progress
}

// Downloader

type Downloader struct {
	client *Aria2Client
}

func NewDownloader(cfg Config) *Downloader {
	return &Downloader{
		client: NewAria2Client(cfg),
	}
}

// Download starts a single download
// path can be directory or full file path
func (d *Downloader) Download(url, path string) (*Task, error) {
	return d.startDownload(url, path, nil)
}

// DownloadMultiThread starts a single download with multiple connections
func (d *Downloader) DownloadMultiThread(url, path string, threads int) (*Task, error) {
	if threads < 1 {
		threads = 1
	}
	if threads > 16 {
		threads = 16
	}

	options := map[string]string{
		"split":                     strconv.Itoa(threads),
		"max-connection-per-server": strconv.Itoa(threads),
		"min-split-size":            "1M",
	}

	return d.startDownload(url, path, options)
}

// DownloadBatch mirrors: aria2c -j N -x M -s M file1 file2 file3
func (d *Downloader) DownloadBatch(
	dir string,
	opts BatchOptions,
	requests []DownloadRequest,
) (*BatchResult, error) {

	if len(requests) == 0 {
		return nil, fmt.Errorf("no downloads requested")
	}

	// Create directory
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create dir: %w", err)
	}

	// Set global concurrency (-j N)
	if err := d.client.ChangeGlobalOption(map[string]string{
		"max-concurrent-downloads": strconv.Itoa(opts.ConcurrentDownloads),
	}); err != nil {
		return nil, fmt.Errorf("set concurrency: %w", err)
	}

	// Build per-file options (-x M -s M)
	fileOptions := map[string]string{
		"dir":                       dir,
		"split":                     strconv.Itoa(opts.ConnectionsPerFile),
		"max-connection-per-server": strconv.Itoa(opts.ConnectionsPerFile),
		"min-split-size":            "1M",
	}

	// Add all downloads synchronously (aria2 queues them)
	tasks := make([]*Task, 0, len(requests))
	for _, req := range requests {
		options := make(map[string]string)
		for k, v := range fileOptions {
			options[k] = v
		}
		if req.Out != "" {
			options["out"] = req.Out
		}

		task, err := d.startDownload(req.URL, dir, options)
		if err != nil {
			// Log but continue with others
			fmt.Fprintf(os.Stderr, "Failed to queue %s: %v\n", req.URL, err)
			continue
		}
		tasks = append(tasks, task)
	}

	if len(tasks) == 0 {
		return nil, fmt.Errorf("all downloads failed to start")
	}

	// Create batch result with aggregated updates
	updates := make(chan BatchUpdate, 10)
	done := make(chan struct{})

	result := &BatchResult{
		Tasks:   tasks,
		Updates: updates,
		Done:    done,
	}

	// Start batch polling goroutine
	go d.pollBatch(tasks, updates, done)

	return result, nil
}

// startDownload is the internal function that:
// 1. Calls AddURI SYNCHRONOUSLY (gets GID immediately)
// 2. Starts a goroutine to poll progress
func (d *Downloader) startDownload(url, path string, options map[string]string) (*Task, error) {
	// Resolve path
	dir, out := resolvePath(path)

	// Ensure options exist
	if options == nil {
		options = make(map[string]string)
	}
	if _, ok := options["dir"]; !ok {
		options["dir"] = dir
	}
	if out != "" {
		options["out"] = out
	}

	// SYNCHRONOUS: Get GID before returning
	gid, err := d.client.AddURI([]string{url}, options)
	if err != nil {
		return nil, err
	}

	// Now create task with valid GID
	progressCh := make(chan Progress, 10)
	doneCh := make(chan struct{})

	task := &Task{
		GID:      gid,
		Progress: progressCh,
		Done:     doneCh,
	}

	// Start polling in background
	go d.pollProgress(gid, url, progressCh, doneCh, &task.Err)

	return task, nil
}

func (d *Downloader) pollProgress(gid, url string, ch chan<- Progress, done chan<- struct{}, errPtr *error) {
	defer close(ch)
	defer close(done)

	ticker := time.NewTicker(700 * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		status, err := d.client.TellStatus(gid)
		if err != nil {
			continue
		}

		completed, _ := strconv.ParseInt(status.CompletedLength, 10, 64)
		total, _ := strconv.ParseInt(status.TotalLength, 10, 64)
		speed, _ := strconv.ParseInt(status.DownloadSpeed, 10, 64)

		var percent float64
		if total > 0 {
			percent = float64(completed) / float64(total) * 100
		}

		// Try to get filename
		name := url
		if len(status.Files) > 0 && status.Files[0].Path != "" {
			name = filepath.Base(status.Files[0].Path)
		}

		p := Progress{
			GID:       gid,
			Name:      name,
			Status:    status.Status,
			Completed: completed,
			Total:     total,
			Speed:     speed,
			Percent:   percent,
			Error:     status.ErrorMessage,
		}

		ch <- p

		if p.Done() {
			if status.Status == "error" {
				*errPtr = fmt.Errorf("%s: %s", name, status.ErrorMessage)
			}
			return
		}
	}
}

func (d *Downloader) pollBatch(tasks []*Task, updates chan<- BatchUpdate, done chan<- struct{}) {
	defer close(updates)
	defer close(done)

	// Track completed GIDs
	completed := make(map[string]bool)
	failed := make(map[string]Progress)

	ticker := time.NewTicker(700 * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		var active, waiting, doneList []Progress
		var newFailed []Progress

		for _, task := range tasks {
			if _, alreadyFailed := failed[task.GID]; completed[task.GID] || alreadyFailed {
				continue
			}

			// Check if task errored
			select {
			case <-task.Done:
				if task.Err != nil {
					failed[task.GID] = Progress{
						GID:    task.GID,
						Status: "error",
						Error:  task.Err.Error(),
					}
					newFailed = append(newFailed, failed[task.GID])
				} else {
					completed[task.GID] = true
					doneList = append(doneList, Progress{
						GID:    task.GID,
						Status: "complete",
					})
				}
				continue
			default:
			}

			// Get status
			status, err := d.client.TellStatus(task.GID)
			if err != nil {
				continue
			}

			completedBytes, _ := strconv.ParseInt(status.CompletedLength, 10, 64)
			totalBytes, _ := strconv.ParseInt(status.TotalLength, 10, 64)
			speed, _ := strconv.ParseInt(status.DownloadSpeed, 10, 64)

			var percent float64
			if totalBytes > 0 {
				percent = float64(completedBytes) / float64(totalBytes) * 100
			}

			name := task.GID
			if len(status.Files) > 0 && status.Files[0].Path != "" {
				name = filepath.Base(status.Files[0].Path)
			}

			p := Progress{
				GID:       task.GID,
				Name:      name,
				Status:    status.Status,
				Completed: completedBytes,
				Total:     totalBytes,
				Speed:     speed,
				Percent:   percent,
			}

			switch status.Status {
			case "active":
				active = append(active, p)
			case "waiting":
				waiting = append(waiting, p)
			case "complete":
				completed[task.GID] = true
				doneList = append(doneList, p)
			case "error":
				failed[task.GID] = p
				newFailed = append(newFailed, p)
			}
		}

		// Build failed list from all accumulated failures
		var allFailed []Progress
		for _, p := range failed {
			allFailed = append(allFailed, p)
		}

		updates <- BatchUpdate{
			Active:  active,
			Waiting: waiting,
			Done:    doneList,
			Failed:  allFailed,
		}

		// Check if all done
		if len(completed)+len(failed) == len(tasks) {
			return
		}
	}
}

// Pause/Resume/Cancel
func (d *Downloader) Pause(gid string) error  { return d.client.Pause(gid) }
func (d *Downloader) Resume(gid string) error { return d.client.Unpause(gid) }
func (d *Downloader) Cancel(gid string) error { return d.client.Remove(gid) }

func resolvePath(path string) (dir, out string) {
	if ext := filepath.Ext(path); ext != "" {
		return filepath.Dir(path), filepath.Base(path)
	}
	return path, ""
}

// UI: Single Download Progress Bar

func ShowProgressBar(task *Task) error {
	var bar *progressbar.ProgressBar

	for p := range task.Progress {
		if bar == nil && p.Total > 0 {
			bar = progressbar.NewOptions64(
				p.Total,
				progressbar.OptionSetDescription(p.Name),
				progressbar.OptionShowBytes(true),
				progressbar.OptionSetWidth(40),
				progressbar.OptionThrottle(100*time.Millisecond),
				progressbar.OptionOnCompletion(func() { fmt.Fprint(os.Stderr, "\n") }),
				progressbar.OptionSetTheme(progressbar.Theme{
					Saucer: "█", SaucerHead: "█", SaucerPadding: "░",
					BarStart: "[", BarEnd: "]",
				}),
			)
		}

		if bar != nil {
			bar.Set64(p.Completed)
			bar.Describe(fmt.Sprintf("%s @ %s/s", p.Name, formatBytes(p.Speed)))
		}

		if p.Done() {
			if bar != nil {
				bar.Finish()
			}
			if p.Status == "complete" {
				fmt.Printf("✓ %s\n", p.Name)
			}
			return task.Err
		}
	}

	return task.Err
}

// UI: Batch Download Display

func ShowBatchProgress(result *BatchResult) error {
	// Track which files we've printed completion for
	printedComplete := make(map[string]bool)
	printedFailed := make(map[string]bool)

	for update := range result.Updates {
		// Clear and redraw (simple approach)
		fmt.Print("\033[H\033[2J") // Clear screen

		fmt.Println("        Download Manager")

		// Active downloads with bars
		if len(update.Active) > 0 {
			fmt.Println("\n▶ Active:")
			for _, p := range update.Active {
				bar := renderBar(p.Percent, 30)
				fmt.Printf("  %s\n", p.Name)
				fmt.Printf("  %s %.1f%% @ %s/s\n", bar, p.Percent, formatBytes(p.Speed))
			}
		}

		// Waiting downloads
		if len(update.Waiting) > 0 {
			fmt.Println("\n⏳ Waiting:")
			for _, p := range update.Waiting {
				fmt.Printf("  %s\n", p.Name)
			}
		}

		// Completed
		if len(update.Done) > 0 {
			fmt.Println("\n✓ Completed:")
			for _, p := range update.Done {
				if !printedComplete[p.GID] {
					fmt.Printf("  %s\n", p.Name)
					printedComplete[p.GID] = true
				}
			}
		}

		// Failed
		if len(update.Failed) > 0 {
			fmt.Println("\n✗ Failed:")
			for _, p := range update.Failed {
				if !printedFailed[p.GID] {
					fmt.Printf("  %s: %s\n", p.Name, p.Error)
					printedFailed[p.GID] = true
				}
			}
		}

		// Summary line
		total := len(update.Active) + len(update.Waiting) + len(update.Done) + len(update.Failed)
		fmt.Printf("\n───────────────────────────────────────\n")
		fmt.Printf("Progress: %d/%d complete", len(update.Done), total)
		if len(update.Failed) > 0 {
			fmt.Printf(", %d failed", len(update.Failed))
		}
		fmt.Println()
	}

	// Final state
	fmt.Print("\033[H\033[2J")
	fmt.Println("        All Downloads Complete")

	// Print results
	for _, task := range result.Tasks {
		if task.Err != nil {
			fmt.Printf("✗ %s: %v\n", task.GID, task.Err)
		} else {
			fmt.Printf("✓ %s\n", task.GID)
		}
	}

	return nil
}

// Simple text bar for TUI
func renderBar(percent float64, width int) string {
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}

	filled := int(percent / 100 * float64(width))
	empty := width - filled

	return "[" + strings.Repeat("█", filled) + strings.Repeat("░", empty) + "]"
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}

// Main Example
// func main() {
// 	// Example: Replicate the bash script behavior
// 	dl := NewDownloader(Config{
// 		Host:   "localhost",
// 		Port:   "6800",
// 		Secret: "mysecret",
// 	})

// 	// These would come from your has_ide / has_language checks
// 	downloads := []DownloadRequest{
// 		{URL: "https://example.com/antigravity.tar.gz", Out: "antigravity.tar.gz"},
// 		{URL: "https://example.com/zed.tar.gz", Out: "zed.tar.gz"},
// 		{URL: "https://example.com/jetbrains-toolbox.tar.gz", Out: "jetbrains-toolbox.tar.gz"},
// 		{URL: "https://example.com/dart.tar.xz", Out: "dart.tar.xz"},
// 		{URL: "https://example.com/go.tar.gz", Out: "go.tar.gz"},
// 	}

// 	// This is equivalent to:
// 	// aria2c --dir=/tmp/usetup -j 3 -x 16 -s 16 file1 file2 file3...
// 	result, err := dl.DownloadBatch(
// 		"/tmp/usetup",
// 		BatchOptions{
// 			ConcurrentDownloads: 3,  // -j 3
// 			ConnectionsPerFile:  16, // -x 16 -s 16
// 		},
// 		downloads,
// 	)

// 	if err != nil {
// 		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
// 		os.Exit(1)
// 	}

// 	// Show batch progress UI
// 	ShowBatchProgress(result)

// 	// Or just wait silently:
// 	// result.Wait()
// }
