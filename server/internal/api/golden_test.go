package api

// Golden HTTP tests: a behavioural snapshot of the whole public route table.
//
// Purpose. The server is being refactored (global DB handle -> injected stores,
// free functions -> injected structs, handlers -> structs). Almost none of that
// code has tests today, so these goldens are the contract the refactor must not
// break: any change in status code or response body shows up as a golden diff.
//
// Regenerate deliberately, never casually:
//
//	go test ./internal/api/ -run TestGolden -update
//
// A diff in these files during a pure refactor is a bug, not an expected change.

import (
	"bytes"
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"github.com/srad/mediasink/server/config"
	"github.com/srad/mediasink/server/internal/db"
)

var update = flag.Bool("update", false, "rewrite the golden files instead of comparing against them")

const (
	testAPIVersion = "1.0"
	testVersion    = "v-test"
	testCommit     = "c-test"
)

// Shared across the whole suite. api.Setup starts `go ws.WsListen()` internally,
// so building a router per test would leak one goroutine per test. Phase 4 of the
// refactor removes that side effect; until then the router is built exactly once.
var (
	testRouter http.Handler
	testToken  string
	testTmpDir string

	// The seeded channel's stream URL. It points at a local test server rather than
	// a public host, so the one route that shells out to yt-dlp
	// (POST /channels/:id/resume) makes no network call.
	//
	// A test-local httptest.Server, deliberately not a route on the real router:
	// a test-only endpoint mounted in router.go would ship to production.
	testStreamURL string
)

func TestMain(m *testing.M) {
	tmp, err := os.MkdirTemp("", "mediasink-golden-")
	if err != nil {
		fmt.Fprintf(os.Stderr, "tempdir: %v\n", err)
		os.Exit(1)
	}
	testTmpDir = tmp
	defer os.RemoveAll(tmp)

	for k, v := range map[string]string{
		"DB_FILENAME":  filepath.Join(tmp, "golden.db"),
		"REC_PATH":     tmp,
		"DATA_DIR":     ".previews",
		"DATA_DISK":    tmp,
		"NET_ADAPTER":  "lo",
		"SECRET":       "golden-test-secret",
		"DB_ADAPTER":   "",
		"LOG_LEVEL":    "fatal",
		"STREAM_DEBUG": "",
	} {
		os.Setenv(k, v)
	}

	// Handlers log errors on every 4xx/5xx; the suite deliberately exercises those.
	log.SetLevel(log.FatalLevel)
	gin.SetMode(gin.TestMode)

	cfg, _, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		os.Exit(1)
	}

	// Serves the seeded channel's stream URL. Its content does not matter: yt-dlp
	// has no extractor for it either way. What matters is that the request stays on
	// loopback, so the outcome does not depend on reaching a public host.
	streamSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, "<html><body>golden test stream page</body></html>")
	}))
	defer streamSrv.Close()
	testStreamURL = streamSrv.URL + "/live"

	if _, err := db.Open(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "open database: %v\n", err)
		os.Exit(1)
	}

	var frontendFS embed.FS // zero value: no embedded frontend needed for API routes
	testRouter = Setup(cfg, testVersion, testCommit, testAPIVersion, frontendFS)

	testToken = seedUserAndLogin()

	code := m.Run()
	os.RemoveAll(tmp)
	os.Exit(code)
}

// seedUserAndLogin creates the account the authenticated cases run as.
func seedUserAndLogin() string {
	creds := `{"username":"golden@example.com","password":"golden-pass-123"}`

	res := call(http.MethodPost, "/api/v2/auth/signup", creds, "")
	if res.status != http.StatusOK {
		panic(fmt.Sprintf("seed signup failed: %d %s", res.status, res.body))
	}

	res = call(http.MethodPost, "/api/v2/auth/login", creds, "")
	if res.status != http.StatusOK {
		panic(fmt.Sprintf("seed login failed: %d %s", res.status, res.body))
	}

	var payload struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal([]byte(res.body), &payload); err != nil || payload.Token == "" {
		panic(fmt.Sprintf("seed login returned no token: %s", res.body))
	}
	return payload.Token
}

type response struct {
	status int
	body   string
}

func call(method, path, body, token string) response {
	var reader *strings.Reader
	if body == "" {
		reader = strings.NewReader("")
	} else {
		reader = strings.NewReader(body)
	}

	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("X-API-Version", testAPIVersion)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	rec := httptest.NewRecorder()
	testRouter.ServeHTTP(rec, req)
	return response{status: rec.Code, body: strings.TrimSpace(rec.Body.String())}
}

// volatileKeys hold values that legitimately differ between runs. They are
// redacted rather than dropped so that a field appearing or disappearing is
// still caught by the golden diff.
var volatileKeys = map[string]bool{
	"token":       true,
	"createdAt":   true,
	"updatedAt":   true,
	"CreatedAt":   true,
	"UpdatedAt":   true,
	"startTime":   true,
	"endTime":     true,
	"timestamp":   true,
	"pid":         true,
	"duration":    true,
	"size":        true,
	"free":        true,
	"used":        true,
	"total":       true,
	"avail":       true,
	"transmitted": true,
	"received":    true,
}

// ytDLPFailure matches the error QueryStreamURLs builds when the yt-dlp subprocess
// fails, up to and including the URL. Everything after it is the exec error plus
// yt-dlp's own stderr.
var ytDLPFailure = regexp.MustCompile(`^(yt-dlp failed for URL \S+): [\s\S]*$`)

// redactSubprocessError normalises the tail of a subprocess failure message.
//
// This case previously pinned the literal text `exec: "yt-dlp": executable file not
// found in $PATH`, which only holds on a machine that does not have yt-dlp. Anywhere
// it IS installed the binary really ran and the golden failed — so the suite passed
// by accident of a missing dependency, and could never run in a provisioned
// container or CI. The detail is a property of the environment and of yt-dlp's
// version, never of this code, so it is redacted like any other volatile value.
// What the case still pins: the route returns 500, and the message names the URL.
func redactSubprocessError(s string) string {
	if m := ytDLPFailure.FindStringSubmatch(s); m != nil {
		return m[1] + ": <subprocess-error>"
	}
	return s
}

// redact walks decoded JSON replacing volatile values with a placeholder, and
// rewrites the temp directory so paths are stable across runs. Deliberately
// structural rather than textual — no regex over response bodies, with the single
// exception of redactSubprocessError above.
func redact(v any) any {
	switch t := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, val := range t {
			if volatileKeys[k] {
				out[k] = "<redacted>"
				continue
			}
			out[k] = redact(val)
		}
		return out
	case []any:
		out := make([]any, len(t))
		for i, val := range t {
			out[i] = redact(val)
		}
		return out
	case string:
		s := t
		if testTmpDir != "" && strings.Contains(s, testTmpDir) {
			s = strings.ReplaceAll(s, testTmpDir, "<tmp>")
		}
		// The test server's port is assigned by the kernel, so it differs per run.
		if testStreamURL != "" && strings.Contains(s, testStreamURL) {
			s = strings.ReplaceAll(s, testStreamURL, "<teststream>")
		}
		return redactSubprocessError(s)
	default:
		return v
	}
}

// jsonShape replaces every scalar leaf with its type name and collapses arrays to
// a single representative element. Used for endpoints that report live host
// telemetry (CPU load, disk usage, per-core arrays): their values are both
// time-varying and machine-dependent, so snapshotting them would produce a golden
// that only passes on one machine at one instant. The shape still catches a field
// being renamed, added or removed during the refactor, which is what matters here.
func jsonShape(v any) any {
	switch t := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, val := range t {
			out[k] = jsonShape(val)
		}
		return out
	case []any:
		if len(t) == 0 {
			return []any{}
		}
		return []any{jsonShape(t[0])}
	case string:
		return "<string>"
	case float64:
		return "<number>"
	case bool:
		return "<bool>"
	case nil:
		return nil
	default:
		return "<unknown>"
	}
}

// normalise renders a response body deterministically: pretty-printed and
// key-sorted when it is JSON, left alone (minus temp paths) when it is not.
func normalise(body string) string {
	if body == "" {
		return ""
	}
	var decoded any
	if err := json.Unmarshal([]byte(body), &decoded); err != nil {
		if testTmpDir != "" {
			body = strings.ReplaceAll(body, testTmpDir, "<tmp>")
		}
		return body
	}
	buf, err := json.MarshalIndent(redact(decoded), "", "  ")
	if err != nil {
		return body
	}
	return string(buf)
}

// shapeOf renders only the structure of a JSON body. See jsonShape.
func shapeOf(body string) string {
	if body == "" {
		return ""
	}
	var decoded any
	if err := json.Unmarshal([]byte(body), &decoded); err != nil {
		return normalise(body)
	}
	buf, err := json.MarshalIndent(jsonShape(decoded), "", "  ")
	if err != nil {
		return normalise(body)
	}
	return string(buf)
}

type routeCase struct {
	name   string
	method string
	path   string
	body   string
	// shapeOnly records the response structure instead of its values, for
	// endpoints whose payload is live host telemetry.
	shapeOnly bool
}

func (c routeCase) label() string {
	if c.name != "" {
		return c.name
	}
	return c.method + " " + c.path
}

func runGolden(t *testing.T, goldenName string, cases []routeCase, token string) {
	t.Helper()

	var out bytes.Buffer
	for _, c := range cases {
		res := call(c.method, c.path, c.body, token)
		fmt.Fprintf(&out, "=== %s\n", c.label())
		fmt.Fprintf(&out, "--- status: %d\n", res.status)

		rendered := normalise(res.body)
		label := "body"
		if c.shapeOnly {
			rendered = shapeOf(res.body)
			label = "body shape"
		}
		if rendered != "" {
			fmt.Fprintf(&out, "--- %s:\n%s\n", label, rendered)
		} else {
			fmt.Fprintf(&out, "--- %s: <empty>\n", label)
		}
		out.WriteString("\n")
	}

	goldenPath := filepath.Join("testdata", goldenName)
	if *update {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatalf("mkdir testdata: %v", err)
		}
		if err := os.WriteFile(goldenPath, out.Bytes(), 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}
		t.Logf("updated %s (%d cases)", goldenPath, len(cases))
		return
	}

	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden %s: %v (run: go test ./internal/api/ -run TestGolden -update)", goldenPath, err)
	}
	if got := out.String(); got != string(want) {
		t.Errorf("response snapshot changed for %s.\n%s", goldenPath, firstDiff(string(want), got))
	}
}

// firstDiff reports the first differing line, so a failure names the offending
// route instead of dumping the whole file.
func firstDiff(want, got string) string {
	wantLines := strings.Split(want, "\n")
	gotLines := strings.Split(got, "\n")
	for i := 0; i < len(wantLines) || i < len(gotLines); i++ {
		var w, g string
		if i < len(wantLines) {
			w = wantLines[i]
		}
		if i < len(gotLines) {
			g = gotLines[i]
		}
		if w != g {
			return fmt.Sprintf("first difference at line %d:\n  want: %q\n  got:  %q", i+1, w, g)
		}
	}
	return "files differ in length only"
}

// allRoutes mirrors the route table in router.go. Kept in declaration order so
// that a reviewer can diff it against Setup() by eye.
func allRoutes() []routeCase {
	return []routeCase{
		{method: http.MethodPost, path: "/api/v2/auth/logout"},

		{method: http.MethodGet, path: "/api/v2/user/profile"},

		{method: http.MethodGet, path: "/api/v2/admin/version"},
		{method: http.MethodPost, path: "/api/v2/admin/import"},
		{method: http.MethodGet, path: "/api/v2/admin/import"},
		{method: http.MethodPost, path: "/api/v2/admin/chapters/regenerate"},

		{method: http.MethodGet, path: "/api/v2/channels"},
		{method: http.MethodPost, path: "/api/v2/channels"},
		{method: http.MethodGet, path: "/api/v2/channels/1"},
		{method: http.MethodDelete, path: "/api/v2/channels/1"},
		{method: http.MethodPatch, path: "/api/v2/channels/1"},
		{method: http.MethodPost, path: "/api/v2/channels/1/resume"},
		{method: http.MethodPost, path: "/api/v2/channels/1/pause"},
		{method: http.MethodPatch, path: "/api/v2/channels/1/fav"},
		{method: http.MethodPatch, path: "/api/v2/channels/1/unfav"},
		{method: http.MethodPost, path: "/api/v2/channels/1/upload"},
		{method: http.MethodPatch, path: "/api/v2/channels/1/tags"},
		{method: http.MethodPost, path: "/api/v2/channels/1/merge"},

		{method: http.MethodPost, path: "/api/v2/jobs/1"},
		{method: http.MethodPost, path: "/api/v2/jobs/stop/1"},
		{method: http.MethodDelete, path: "/api/v2/jobs/1"},
		{method: http.MethodPost, path: "/api/v2/jobs/list"},
		{method: http.MethodPost, path: "/api/v2/jobs/resume"},
		{method: http.MethodPost, path: "/api/v2/jobs/pause"},
		{method: http.MethodGet, path: "/api/v2/jobs/worker"},

		{method: http.MethodPost, path: "/api/v2/recorder/resume"},
		{method: http.MethodPost, path: "/api/v2/recorder/pause"},
		{method: http.MethodGet, path: "/api/v2/recorder"},

		{method: http.MethodPost, path: "/api/v2/videos/updateinfo"},
		{method: http.MethodPost, path: "/api/v2/videos/isupdating"},
		{method: http.MethodGet, path: "/api/v2/videos"},
		{method: http.MethodPost, path: "/api/v2/videos/filter"},
		{method: http.MethodGet, path: "/api/v2/videos/random/5"},
		{method: http.MethodGet, path: "/api/v2/videos/bookmarks"},
		{method: http.MethodGet, path: "/api/v2/videos/enhance/descriptions"},
		{method: http.MethodGet, path: "/api/v2/videos/1"},
		{method: http.MethodGet, path: "/api/v2/videos/1/preview/manifest"},
		{method: http.MethodGet, path: "/api/v2/videos/1/download"},
		{method: http.MethodPatch, path: "/api/v2/videos/1/fav"},
		{method: http.MethodPatch, path: "/api/v2/videos/1/unfav"},
		{method: http.MethodPost, path: "/api/v2/videos/1/720/convert"},
		{method: http.MethodPost, path: "/api/v2/videos/1/cut"},
		{method: http.MethodPost, path: "/api/v2/videos/1/preview"},
		{method: http.MethodPost, path: "/api/v2/videos/1/enhance"},
		{method: http.MethodPost, path: "/api/v2/videos/1/estimate-enhancement"},
		{method: http.MethodDelete, path: "/api/v2/videos/1"},

		{method: http.MethodPost, path: "/api/v2/previews/regenerate"},
		{method: http.MethodGet, path: "/api/v2/previews/regenerate"},

		{method: http.MethodPost, path: "/api/v2/analysis/search/image"},
		{method: http.MethodPost, path: "/api/v2/analysis/group"},
		{method: http.MethodPost, path: "/api/v2/analysis/all"},
		{method: http.MethodPost, path: "/api/v2/analysis/1"},
		{method: http.MethodGet, path: "/api/v2/analysis/1"},

		// Live host telemetry: values vary per machine and per second, so only the
		// response shape is snapshotted.
		{method: http.MethodGet, path: "/api/v2/info/1", shapeOnly: true},
		{method: http.MethodGet, path: "/api/v2/info/disk", shapeOnly: true},

		{method: http.MethodGet, path: "/api/v2/processes", shapeOnly: true},
	}
}

// TestGoldenAuthGate asserts that every protected route rejects an unauthenticated
// request. Cheap and completely safe — the request never reaches a handler — while
// still covering the entire route table, so an accidentally-unprotected or
// accidentally-deleted route shows up immediately.
func TestGoldenAuthGate(t *testing.T) {
	cases := allRoutes()
	sort.Slice(cases, func(i, j int) bool { return cases[i].label() < cases[j].label() })
	runGolden(t, "auth_gate.golden", cases, "")
}

// TestGoldenVersionGate asserts the X-API-Version precondition. Uses a raw request
// because call() always sets the header.
func TestGoldenVersionGate(t *testing.T) {
	var out bytes.Buffer
	for _, path := range []string{"/api/v2/videos", "/api/v2/channels", "/api/v2/auth/login"} {
		req := httptest.NewRequest(http.MethodGet, path, strings.NewReader(""))
		rec := httptest.NewRecorder()
		testRouter.ServeHTTP(rec, req)
		fmt.Fprintf(&out, "=== GET %s (no X-API-Version)\n--- status: %d\n--- body:\n%s\n\n",
			path, rec.Code, normalise(strings.TrimSpace(rec.Body.String())))
	}

	goldenPath := filepath.Join("testdata", "version_gate.golden")
	if *update {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatalf("mkdir testdata: %v", err)
		}
		if err := os.WriteFile(goldenPath, out.Bytes(), 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}
		return
	}
	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden: %v (run with -update)", err)
	}
	if got := out.String(); got != string(want) {
		t.Errorf("version gate snapshot changed.\n%s", firstDiff(string(want), got))
	}
}

// TestGoldenAuthenticated snapshots authenticated responses for the routes that
// are safe to execute in a test process.
//
// Excluded on purpose, with reasons:
//   - /ws                     websocket upgrade, not expressible via httptest recorder
//   - /admin/import           starts a filesystem import in the background
//   - /recorder/resume        starts the recorder worker pool and ffmpeg children
//   - /analysis/all           enqueues analysis jobs for every recording
//   - /previews/regenerate    starts a background regeneration pass
//   - /channels/:id/upload    multipart upload; covered by the auth-gate case only
//   - /jobs/resume            starts the job workers
func TestGoldenAuthenticated(t *testing.T) {
	skip := map[string]bool{
		"POST /api/v2/admin/import":        true,
		"POST /api/v2/recorder/resume":     true,
		"POST /api/v2/analysis/all":        true,
		"POST /api/v2/previews/regenerate": true,
		"POST /api/v2/channels/1/upload":   true,
		"POST /api/v2/jobs/resume":         true,
		// Mutating routes that would make later cases order-dependent.
		"DELETE /api/v2/channels/1": true,
		"DELETE /api/v2/videos/1":   true,
		"DELETE /api/v2/jobs/1":     true,
	}

	var cases []routeCase
	for _, c := range allRoutes() {
		if skip[c.label()] {
			continue
		}
		cases = append(cases, c)
	}

	// Bodies for the routes that parse one, so they exercise handler logic rather
	// than stopping at the bind error.
	bodies := map[string]string{
		"POST /api/v2/channels":      `{"channelName":"golden_channel","displayName":"Golden","skipStart":0,"minDuration":10,"url":"` + testStreamURL + `","isPaused":true,"fav":false}`,
		"POST /api/v2/jobs/list":     `{"skip":0,"take":10,"states":[],"sortOrder":"asc"}`,
		"POST /api/v2/videos/filter": `{"column":"created_at","order":"desc","skip":0,"take":5}`,
	}
	for i := range cases {
		if b, ok := bodies[cases[i].label()]; ok {
			cases[i].body = b
		}
	}

	sort.Slice(cases, func(i, j int) bool { return cases[i].label() < cases[j].label() })
	runGolden(t, "authenticated.golden", cases, testToken)
}
