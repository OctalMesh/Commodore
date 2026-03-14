package tui

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

/*
Finder is satisfied by any type that can locate a project / module root.
Both the Foreman's RootFinder and every Brigadier's ModuleFinder satisfy it.
*/
type Finder interface {
	Find() (string, error)
}

type phase int8

const (
	phaseChecking phase = iota
	phaseStarting
	phaseRunning
	phaseStopping
	phaseDone
	phaseError
)

const (
	logViewHeight = 15  // visible lines in the log viewport
	logMaxLines   = 500 // max buffered log lines (older ones dropped)
	hintDuration  = 3 * time.Second
	pageAll       = "all" // synthetic page that shows every service
)

type checkDoneMsg struct {
	root string
	err  error
}
type tiltStartedMsg struct{ cmd *exec.Cmd }
type logLineMsg struct{ line string }
type logsDoneMsg struct{}
type tiltExitMsg struct{ err error }
type tiltDownDoneMsg struct{}
type tickMsg time.Time

// Model is the BubbleTea model shared by every OctalWeb CLI's up / down command.
// It shows a spinner during pre-flight checks, then streams Tilt output into a
// bounded, scrollable viewport with a key-hint bar below it.
type Model struct {
	brand    string
	finder   Finder
	argv     []string
	tiltfile string // path extracted from argv --file, used by tilt down
	phase    phase
	spinner  spinner.Model
	vp       viewport.Model
	// paged log storage
	pages        []string            // ordered: ["all", svc1, svc2, ...]
	pageLogs     map[string][]string // log lines per page key
	pageScroll   map[string]int      // saved YOffset per page
	activePage   int                 // index into pages
	err          error
	cmd          *exec.Cmd
	logCh        chan string
	doneCh       chan error
	root         string
	legacyMode   bool
	statusHint   string
	hintAt       time.Time
	quitting     bool
	elapsed      time.Duration
	startAt      time.Time
	width        int
	labelStart   string
	labelRunning string
}

func newModel(brand string, f Finder, argv []string, labelStart, labelRunning string) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = StyleSpinner
	vp := viewport.New(80, logViewHeight)

	return Model{
		brand:        brand,
		finder:       f,
		argv:         argv,
		phase:        phaseChecking,
		spinner:      s,
		vp:           vp,
		pages:        []string{pageAll},
		pageLogs:     map[string][]string{pageAll: nil},
		pageScroll:   map[string]int{},
		activePage:   0,
		logCh:        make(chan string, 512),
		doneCh:       make(chan error, 1),
		tiltfile:     extractTiltfile(argv),
		labelStart:   labelStart,
		labelRunning: labelRunning,
	}
}

// NewModel constructs the TUI model for the `up` command.
func NewModel(brand string, f Finder, argv []string) Model {
	return newModel(brand, f, argv, "Starting Tilt...", "● running")
}

// NewDownModel constructs the TUI model for the `down` command.
func NewDownModel(brand string, f Finder, argv []string) Model {
	return newModel(brand, f, argv, "Stopping Tilt...", "● stopping")
}

/*
FinalErr returns the tilt process exit error.
Returns nil when the stop was user-initiated (Ctrl+C) - that is not an error.
*/
func (m Model) FinalErr() error {
	if m.quitting {
		return nil
	}
	return m.err
}

// LegacyMode reports whether the user pressed 't' to switch to tilt's legacy terminal mode.
func (m Model) LegacyMode() bool { return m.legacyMode }

// Root returns the located project / module root (available after Tilt started).
func (m Model) Root() string { return m.root }

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, doCheck(m.finder))
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.vp.Width = msg.Width
		m.vp.SetContent(m.renderCurrentPage())

	case tea.KeyMsg:
		switch {
		case msg.Type == tea.KeyCtrlC:
			return m.handleStop()

		case m.phase == phaseRunning:
			switch msg.String() {
			case " ":
				openBrowser("http://localhost:10350")
				m.statusHint = "opening tilt UI in browser..."
				m.hintAt = time.Now()
			case "t":
				m.legacyMode = true
				m.statusHint = "switching to legacy terminal mode..."
				m.hintAt = time.Now()
				return m.handleStop()
			case "g":
				m.vp.GotoTop()
			case "G":
				m.vp.GotoBottom()
			case "[":
				if len(m.pages) > 1 {
					m.pageScroll[m.pages[m.activePage]] = m.vp.YOffset
					m.activePage = (m.activePage - 1 + len(m.pages)) % len(m.pages)
					m.vp.SetContent(m.renderCurrentPage())
					if pos, ok := m.pageScroll[m.pages[m.activePage]]; ok {
						m.vp.YOffset = pos
					} else {
						m.vp.GotoBottom()
					}
				}
			case "]":
				if len(m.pages) > 1 {
					m.pageScroll[m.pages[m.activePage]] = m.vp.YOffset
					m.activePage = (m.activePage + 1) % len(m.pages)
					m.vp.SetContent(m.renderCurrentPage())
					if pos, ok := m.pageScroll[m.pages[m.activePage]]; ok {
						m.vp.YOffset = pos
					} else {
						m.vp.GotoBottom()
					}
				}
			default:
				var vpCmd tea.Cmd
				m.vp, vpCmd = m.vp.Update(msg)
				return m, vpCmd
			}

		case m.phase == phaseStopping:
			var vpCmd tea.Cmd
			m.vp, vpCmd = m.vp.Update(msg)
			return m, vpCmd
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case checkDoneMsg:
		if msg.err != nil {
			m.err = msg.err
			m.phase = phaseError
			return m, tea.Quit
		}
		m.root = msg.root
		m.phase = phaseStarting
		return m, tea.Batch(
			m.spinner.Tick,
			doStart(msg.root, m.argv, m.logCh, m.doneCh),
		)

	case tiltStartedMsg:
		m.cmd = msg.cmd
		m.phase = phaseRunning
		m.startAt = time.Now()
		return m, tea.Batch(waitForLog(m.logCh), tick())

	case logLineMsg:
		svc := parseService(msg.line)

		// Always append to the synthetic "all" page.
		m.pageLogs[pageAll] = append(m.pageLogs[pageAll], msg.line)
		if len(m.pageLogs[pageAll]) > logMaxLines {
			m.pageLogs[pageAll] = m.pageLogs[pageAll][len(m.pageLogs[pageAll])-logMaxLines:]
		}

		// Append to the per-service page, creating it on first sight.
		if svc != "" {
			if _, exists := m.pageLogs[svc]; !exists {
				m.pages = append(m.pages, svc)
				m.pageLogs[svc] = nil
			}
			m.pageLogs[svc] = append(m.pageLogs[svc], msg.line)
			if len(m.pageLogs[svc]) > logMaxLines {
				m.pageLogs[svc] = m.pageLogs[svc][len(m.pageLogs[svc])-logMaxLines:]
			}
		}

		// Refresh viewport only when the active page is affected.
		activeKey := m.pages[m.activePage]
		if activeKey == pageAll || activeKey == svc {
			atBottom := m.vp.AtBottom()
			m.vp.SetContent(m.renderCurrentPage())
			if atBottom {
				m.vp.GotoBottom()
			}
		}
		return m, waitForLog(m.logCh)

	case logsDoneMsg:
		return m, waitForDone(m.doneCh)

	case tiltDownDoneMsg:
		m.phase = phaseDone
		return m, tea.Quit

	case tiltExitMsg:
		m.err = msg.err
		if m.quitting {
			// tilt up exited during shutdown; tiltDownDoneMsg will trigger Quit.
			return m, nil
		}
		if msg.err == nil {
			m.phase = phaseDone
		} else {
			m.phase = phaseError
		}
		return m, tea.Quit

	case tickMsg:
		if m.phase == phaseRunning {
			m.elapsed = time.Since(m.startAt)
			if m.statusHint != "" && time.Since(m.hintAt) >= hintDuration {
				m.statusHint = ""
			}
			return m, tick()
		}
	}

	return m, nil
}

// renderCurrentPage builds the log content string for the active page.
func (m Model) renderCurrentPage() string {
	if len(m.pages) == 0 {
		return StyleMuted.Render("  Waiting for Tilt output...")
	}
	logs := m.pageLogs[m.pages[m.activePage]]
	if len(logs) == 0 {
		return StyleMuted.Render("  Waiting for Tilt output...")
	}
	lines := make([]string, len(logs))
	for i, l := range logs {
		lines[i] = StyleLog.Render("  " + l)
	}
	return strings.Join(lines, "\n")
}

func (m Model) handleStop() (tea.Model, tea.Cmd) {
	if m.cmd != nil && m.cmd.Process != nil {
		m.quitting = true
		m.phase = phaseStopping
		return m, tea.Batch(
			doTiltDown(m.root, m.tiltfile, m.cmd),
			m.spinner.Tick,
		)
	}
	return m, tea.Quit
}

/*
doTiltDown runs `tilt down` to gracefully stop all managed services and
containers, then kills the tilt-up process.
The whole sequence is bounded by a 60 s timeout; after that the process is
force-killed regardless.
*/
func doTiltDown(root, tiltfile string, upCmd *exec.Cmd) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		if tiltfile != "" {
			down := exec.CommandContext(ctx, "tilt", "down", "--file", tiltfile)
			down.Dir = root
			_ = down.Run()
		}

		// Kill the tilt-up process after tilt-down has finished (or timed out).
		if upCmd != nil && upCmd.Process != nil {
			_ = upCmd.Process.Kill()
		}

		return tiltDownDoneMsg{}
	}
}

// extractTiltfile pulls the --file value out of a tilt argv slice.
func extractTiltfile(argv []string) string {
	for i, arg := range argv {
		if arg == "--file" && i+1 < len(argv) {
			return argv[i+1]
		}
	}
	return ""
}

// openBrowser opens url in the OS default browser.
func openBrowser(url string) {
	switch runtime.GOOS {
	case "windows":
		_ = exec.Command("cmd", "/c", "start", url).Start()
	case "darwin":
		_ = exec.Command("open", url).Start()
	default:
		_ = exec.Command("xdg-open", url).Start()
	}
}

func (m Model) View() string {
	switch m.phase {
	case phaseChecking:
		return fmt.Sprintf("  %s  %s\n",
			m.spinner.View(),
			StyleStep.Render("Checking environment..."),
		)
	case phaseStarting:
		return fmt.Sprintf("  %s  %s\n",
			m.spinner.View(),
			StyleStep.Render(m.labelStart),
		)
	case phaseRunning, phaseStopping:
		return m.logView()
	case phaseDone:
		return fmt.Sprintf("  %s  %s\n",
			StyleSuccess.Render("✓"),
			StyleSuccess.Render("Done."),
		)
	case phaseError:
		label := "unknown error"
		if m.err != nil {
			label = m.err.Error()
		}
		return fmt.Sprintf("  %s  %s\n",
			StyleError.Render("✗"),
			StyleError.Render(label),
		)
	}
	return ""
}

func (m Model) logView() string {
	w := m.vp.Width
	if w < 1 {
		w = 80
	}
	sep := StyleMuted.Render(strings.Repeat("─", w))
	var b strings.Builder
	b.WriteString(m.statusBar())
	b.WriteByte('\n')
	b.WriteString(sep)
	b.WriteByte('\n')
	b.WriteString(m.vp.View())
	b.WriteByte('\n')
	b.WriteString(sep)
	b.WriteByte('\n')
	if tabs := m.tabBar(); tabs != "" {
		b.WriteString(tabs)
		b.WriteByte('\n')
		b.WriteString(sep)
		b.WriteByte('\n')
	}
	b.WriteString(m.keyBar())
	b.WriteByte('\n')
	return b.String()
}

// tabBar renders a horizontal list of page tabs below the log separator.
// Hidden when there is only the synthetic "all" page.
func (m Model) tabBar() string {
	if len(m.pages) <= 1 {
		return ""
	}
	pipe := StyleMuted.Render("│")
	parts := make([]string, len(m.pages))
	for i, pg := range m.pages {
		label := " " + pg + " "
		if i == m.activePage {
			parts[i] = StyleTabActive.Render(label)
		} else {
			parts[i] = StyleTab.Render(label)
		}
	}
	return "  " + strings.Join(parts, pipe)
}

// statusBar renders a full-width running status line.
func (m Model) statusBar() string {
	var runLabel string
	if m.phase == phaseStopping {
		runLabel = m.spinner.View() + "  " + StyleWarn.Render("stopping...")
	} else {
		runLabel = StyleStatusRunning.Render(m.labelRunning)
	}
	left := fmt.Sprintf("  %s  %s  ",
		StyleBrand.Render("◉ "+m.brand),
		runLabel,
	)
	right := StyleMuted.Render(fmtElapsed(m.elapsed) + "  ")

	gap := m.width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}
	return left + strings.Repeat(" ", gap) + right
}

func (m Model) keyBar() string {
	if m.statusHint != "" {
		return StyleStep.Render("  " + m.statusHint)
	}
	base := "  space browser  t legacy  ctrl+c stop  ↑/↓ j/k scroll  g top  G bottom  pgup/pgdn page"
	if len(m.pages) > 1 {
		base += "  [ ] switch tab"
	}
	return StyleMuted.Render(base)
}

func doCheck(f Finder) tea.Cmd {
	return func() tea.Msg {
		if _, err := exec.LookPath("tilt"); err != nil {
			return checkDoneMsg{err: fmt.Errorf(
				"tilt not found on PATH - install: https://docs.tilt.dev/install.html",
			)}
		}
		root, err := f.Find()
		return checkDoneMsg{root: root, err: err}
	}
}

func doStart(root string, argv []string, logCh chan<- string, doneCh chan<- error) tea.Cmd {
	return func() tea.Msg {
		args := make([]string, 0, len(argv)+1)
		args = append(args, argv...)
		args = append(args, "--stream=true")

		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = root
		setSysProcAttr(cmd)

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return checkDoneMsg{err: fmt.Errorf("stdout pipe: %w", err)}
		}
		stderr, err := cmd.StderrPipe()
		if err != nil {
			return checkDoneMsg{err: fmt.Errorf("stderr pipe: %w", err)}
		}
		if err := cmd.Start(); err != nil {
			return checkDoneMsg{err: fmt.Errorf("tilt start: %w", err)}
		}

		var wg sync.WaitGroup
		wg.Add(2)
		go func() { defer wg.Done(); drainTo(stdout, logCh) }()
		go func() { defer wg.Done(); drainTo(stderr, logCh) }()
		go func() {
			wg.Wait()
			doneCh <- cmd.Wait()
			close(logCh)
		}()

		return tiltStartedMsg{cmd: cmd}
	}
}

func drainTo(r io.Reader, ch chan<- string) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		ch <- sc.Text()
	}
}

func waitForLog(ch <-chan string) tea.Cmd {
	return func() tea.Msg {
		line, ok := <-ch
		if !ok {
			return logsDoneMsg{}
		}
		return logLineMsg{line: line}
	}
}

func waitForDone(ch <-chan error) tea.Cmd {
	return func() tea.Msg {
		return tiltExitMsg{err: <-ch}
	}
}

func tick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// parseService extracts the service name from a Tilt log line.
// Tilt streams lines in the format: "  service-name │ log content"
// using U+2502 (BOX DRAWINGS LIGHT VERTICAL) as the separator.
// Returns an empty string for lines that don't match the format.
func parseService(line string) string {
	trimmed := strings.TrimLeft(line, " ")
	const sep = " \u2502 "
	idx := strings.Index(trimmed, sep)
	if idx <= 0 {
		return ""
	}
	return strings.TrimSpace(trimmed[:idx])
}

func fmtElapsed(d time.Duration) string {
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	if h > 0 {
		return fmt.Sprintf("%dh %dm %ds", h, m, s)
	}
	if m > 0 {
		return fmt.Sprintf("%dm %ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}
