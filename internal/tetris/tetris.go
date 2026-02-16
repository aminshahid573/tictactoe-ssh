package tetris

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ─────────────────────────────────────────────
//  Constants
// ─────────────────────────────────────────────

const (
	boardWidth  = 10
	boardHeight = 20
)

// ─────────────────────────────────────────────
//  Palette  (single source of truth)
// ─────────────────────────────────────────────

const (
	colBg       = lipgloss.Color("#0A0A18")
	colPanelBg  = lipgloss.Color("#0F0F22")
	colBorder   = lipgloss.Color("#2D2B55")
	colAccent   = lipgloss.Color("#7C3AED")
	colAccentHi = lipgloss.Color("#C084FC")
	colDim      = lipgloss.Color("#3D3B5C")
	colMid      = lipgloss.Color("#8B83B8")
	colGhost    = lipgloss.Color("#1A1832")
	colGhostFg  = lipgloss.Color("#2E2B52")
	colRed      = lipgloss.Color("#FF2D55")
	colYellow   = lipgloss.Color("#FFE600")
)

// ─────────────────────────────────────────────
//  Types
// ─────────────────────────────────────────────

type Point struct{ x, y int }

type Tetromino struct {
	cells [][]bool
	color lipgloss.Color
}

var tetrominoes = []Tetromino{
	{cells: [][]bool{{true, true, true, true}}, color: "#00F5FF"},
	{cells: [][]bool{{true, true}, {true, true}}, color: "#FFE600"},
	{cells: [][]bool{{false, true, false}, {true, true, true}}, color: "#CC00FF"},
	{cells: [][]bool{{false, true, true}, {true, true, false}}, color: "#00FF6A"},
	{cells: [][]bool{{true, true, false}, {false, true, true}}, color: "#FF2D55"},
	{cells: [][]bool{{true, false, false}, {true, true, true}}, color: "#0A84FF"},
	{cells: [][]bool{{false, false, true}, {true, true, true}}, color: "#FF9F0A"},
}

type boardCell struct {
	filled bool
	color  lipgloss.Color
}

type GameState int

const (
	StatePlaying GameState = iota
	StatePaused
	StateGameOver
)

// ─────────────────────────────────────────────
//  Model
// ─────────────────────────────────────────────

type Model struct {
	board      [boardHeight][boardWidth]boardCell
	current    Tetromino
	currentPos Point
	next       Tetromino
	held       *Tetromino
	canHold    bool
	score      int
	lines      int
	level      int
	State      GameState
	tickSpeed  time.Duration
	rng        *rand.Rand
	ghostY     int
	TermW      int
	TermH      int
	WantsQuit  bool
}

func InitialModel() Model {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	m := Model{
		rng:       r,
		tickSpeed: 600 * time.Millisecond,
		canHold:   true,
		TermW:     80,
		TermH:     24,
		State:     StatePlaying,
	}
	m.current = m.randomPiece()
	m.currentPos = m.spawnPos(m.current)
	m.next = m.randomPiece()
	m.computeGhost()
	return m
}

func (m *Model) randomPiece() Tetromino {
	t := tetrominoes[m.rng.Intn(len(tetrominoes))]
	cp := make([][]bool, len(t.cells))
	for i, row := range t.cells {
		cp[i] = make([]bool, len(row))
		copy(cp[i], row)
	}
	t.cells = cp
	return t
}

func (m *Model) spawnPos(t Tetromino) Point {
	return Point{x: boardWidth/2 - len(t.cells[0])/2, y: 0}
}

// ─────────────────────────────────────────────
//  Messages
// ─────────────────────────────────────────────

type TickMsg time.Time

func TickCmd(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg { return TickMsg(t) })
}

// ─────────────────────────────────────────────
//  Init
// ─────────────────────────────────────────────

func (m Model) Init() tea.Cmd {
	return TickCmd(m.tickSpeed)
}

// ─────────────────────────────────────────────
//  Update
// ─────────────────────────────────────────────

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.TermW = msg.Width
		m.TermH = msg.Height
		return m, nil

	case tea.KeyMsg:
		if m.State == StateGameOver {
			switch msg.String() {
			case "r":
				nm := InitialModel()
				nm.TermW, nm.TermH = m.TermW, m.TermH
				return nm, nm.Init()
			case "q", "ctrl+c":
				m.WantsQuit = true
				return m, nil
			}
			return m, nil
		}
		switch msg.String() {
		case "q", "ctrl+c":
			m.WantsQuit = true
			return m, nil
		case "p":
			if m.State == StatePlaying {
				m.State = StatePaused
			} else {
				m.State = StatePlaying
			}
			return m, TickCmd(m.tickSpeed)
		}
		if m.State == StatePaused {
			return m, nil
		}
		switch msg.String() {
		case "left", "a":
			m.moveH(-1)
		case "right", "d":
			m.moveH(1)
		case "down", "s":
			if !m.moveDown() {
				m.place()
				return m, TickCmd(m.tickSpeed)
			}
		case "up", "w":
			m.rotate()
		case "x": // Changed drop key
			m.hardDrop()
		case "c":
			m.holdPiece()
		}
		m.computeGhost()
		return m, nil

	case TickMsg:
		if m.State != StatePlaying {
			return m, TickCmd(m.tickSpeed)
		}
		if !m.moveDown() {
			m.place()
		}
		if m.State == StateGameOver {
			return m, nil
		}
		return m, TickCmd(m.tickSpeed)
	}

	return m, nil
}

// ─────────────────────────────────────────────
//  Game Logic
// ─────────────────────────────────────────────

func (m *Model) collides(t Tetromino, pos Point) bool {
	for y, row := range t.cells {
		for x, filled := range row {
			if !filled {
				continue
			}
			nx, ny := pos.x+x, pos.y+y
			if nx < 0 || nx >= boardWidth || ny >= boardHeight {
				return true
			}
			if ny >= 0 && m.board[ny][nx].filled {
				return true
			}
		}
	}
	return false
}

func (m *Model) moveH(dx int) {
	np := Point{m.currentPos.x + dx, m.currentPos.y}
	if !m.collides(m.current, np) {
		m.currentPos = np
	}
}

func (m *Model) moveDown() bool {
	np := Point{m.currentPos.x, m.currentPos.y + 1}
	if !m.collides(m.current, np) {
		m.currentPos = np
		return true
	}
	return false
}

func (m *Model) hardDrop() {
	for !m.collides(m.current, Point{m.currentPos.x, m.currentPos.y + 1}) {
		m.currentPos.y++
	}
	m.place()
}

func (m *Model) rotate() {
	rows, cols := len(m.current.cells), len(m.current.cells[0])
	rot := make([][]bool, cols)
	for i := range rot {
		rot[i] = make([]bool, rows)
	}
	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			rot[x][rows-1-y] = m.current.cells[y][x]
		}
	}
	newT := m.current
	newT.cells = rot
	for _, kick := range []Point{{0, 0}, {-1, 0}, {1, 0}, {-2, 0}, {2, 0}, {0, -1}} {
		np := Point{m.currentPos.x + kick.x, m.currentPos.y + kick.y}
		if !m.collides(newT, np) {
			m.current = newT
			m.currentPos = np
			return
		}
	}
}

func (m *Model) computeGhost() {
	gy := m.currentPos.y
	for !m.collides(m.current, Point{m.currentPos.x, gy + 1}) {
		gy++
	}
	m.ghostY = gy
}

func (m *Model) place() {
	for y, row := range m.current.cells {
		for x, filled := range row {
			if !filled {
				continue
			}
			ny, nx := m.currentPos.y+y, m.currentPos.x+x
			if ny < 0 {
				m.State = StateGameOver
				return
			}
			m.board[ny][nx] = boardCell{filled: true, color: m.current.color}
		}
	}
	m.clearLines()
	m.canHold = true
	m.current = m.next
	m.currentPos = m.spawnPos(m.current)
	m.next = m.randomPiece()
	m.computeGhost()
	if m.collides(m.current, m.currentPos) {
		m.State = StateGameOver
	}
	m.level = m.lines/10 + 1
	// Fixed speed: kept constant at 600ms
	m.tickSpeed = 600 * time.Millisecond
}

func (m *Model) clearLines() {
	cleared := 0
	newBoard := [boardHeight][boardWidth]boardCell{}
	row := boardHeight - 1
	for y := boardHeight - 1; y >= 0; y-- {
		full := true
		for x := 0; x < boardWidth; x++ {
			if !m.board[y][x].filled {
				full = false
				break
			}
		}
		if !full {
			newBoard[row] = m.board[y]
			row--
		} else {
			cleared++
		}
	}
	m.board = newBoard
	scoreTable := []int{0, 100, 300, 500, 800}
	if cleared <= 4 {
		m.score += scoreTable[cleared] * m.level
	}
	m.lines += cleared
}

func (m *Model) holdPiece() {
	if !m.canHold {
		return
	}
	m.canHold = false
	if m.held == nil {
		h := m.current
		m.held = &h
		m.current = m.next
		m.next = m.randomPiece()
	} else {
		m.current, *m.held = *m.held, m.current
	}
	m.currentPos = m.spawnPos(m.current)
	m.computeGhost()
}

// ─────────────────────────────────────────────
//  Rendering helpers — bg-aware, no black gaps
// ─────────────────────────────────────────────

// padRight pads to exact terminal-width w with spaces on given bg
func padRight(s string, w int, bg lipgloss.Color) string {
	vis := lipgloss.Width(s)
	if vis >= w {
		return s
	}
	return s + lipgloss.NewStyle().Background(bg).Render(strings.Repeat(" ", w-vis))
}

func filledCell(c lipgloss.Color) string {
	return lipgloss.NewStyle().Background(c).Foreground(c).Render("  ")
}

func emptyCell() string {
	return lipgloss.NewStyle().Background(colBg).Foreground(colDim).Render("░░")
}

func ghostCell() string {
	return lipgloss.NewStyle().Background(colGhost).Foreground(colGhostFg).Render("▒▒")
}

func previewEmpty() string {
	return lipgloss.NewStyle().Background(colPanelBg).Foreground(colDim).Render("··")
}

func previewFilled(c lipgloss.Color) string {
	return lipgloss.NewStyle().Background(c).Foreground(c).Render("  ")
}

// ─────────────────────────────────────────────
//  View
// ─────────────────────────────────────────────

func (m Model) View() string {
	leftCol := m.buildLeft()
	boardCol := m.buildBoard()
	rightCol := m.buildRight()

	// 2-char bg-filled spacer between columns
	spacer := lipgloss.NewStyle().Background(colBg).Render("  ")

	game := lipgloss.JoinHorizontal(lipgloss.Top,
		leftCol, spacer, boardCol, spacer, rightCol,
	)

	title := lipgloss.NewStyle().
		Foreground(colAccentHi).Background(colBg).Bold(true).
		Render("  ◈  T E T R I S  ◈  ")

	var status string
	switch m.State {
	case StateGameOver:
		status = lipgloss.NewStyle().Foreground(colRed).Background(colBg).Bold(true).Render("  ✕ GAME OVER") +
			lipgloss.NewStyle().Foreground(colMid).Background(colBg).Render("   [R] restart  [Q] quit")
	case StatePaused:
		status = lipgloss.NewStyle().Foreground(colYellow).Background(colBg).Bold(true).Render("  ⏸  PAUSED") +
			lipgloss.NewStyle().Foreground(colMid).Background(colBg).Render("   [P] resume  [Q] quit")
	default:
		status = lipgloss.NewStyle().Foreground(colDim).Background(colBg).
			Render("  [←→] move  [↑] rotate  [↓] soft  [X] drop  [C] hold  [P] pause  [Q] quit")
	}

	blankLine := lipgloss.NewStyle().Background(colBg).Render(" ")

	// Use Center alignment for JoinVertical to prevent displacement when status width changes
	ui := lipgloss.JoinVertical(lipgloss.Center,
		title, blankLine, game, blankLine, status,
	)

	// Fill the entire terminal with colBg — no black spots anywhere
	return lipgloss.NewStyle().
		Width(m.TermW).
		Height(m.TermH).
		Background(colBg).
		Align(lipgloss.Center, lipgloss.Center).
		Render(ui)
}

// ─────────────────────────────────────────────
//  Board column
// ─────────────────────────────────────────────

func (m Model) buildBoard() string {
	boardVisW := boardWidth * 2 // each cell = 2 chars

	grid := [boardHeight][boardWidth]string{}
	for y := 0; y < boardHeight; y++ {
		for x := 0; x < boardWidth; x++ {
			if m.board[y][x].filled {
				grid[y][x] = filledCell(m.board[y][x].color)
			} else {
				grid[y][x] = emptyCell()
			}
		}
	}

	// Ghost
	for y, row := range m.current.cells {
		for x, filled := range row {
			if !filled {
				continue
			}
			gy, gx := m.ghostY+y, m.currentPos.x+x
			if gy >= 0 && gy < boardHeight && gx >= 0 && gx < boardWidth && !m.board[gy][gx].filled {
				grid[gy][gx] = ghostCell()
			}
		}
	}

	// Active piece
	for y, row := range m.current.cells {
		for x, filled := range row {
			if !filled {
				continue
			}
			py, px := m.currentPos.y+y, m.currentPos.x+x
			if py >= 0 && py < boardHeight && px >= 0 && px < boardWidth {
				grid[py][px] = filledCell(m.current.color)
			}
		}
	}

	rows := make([]string, boardHeight)
	for y := 0; y < boardHeight; y++ {
		var line string
		for x := 0; x < boardWidth; x++ {
			line += grid[y][x]
		}
		rows[y] = padRight(line, boardVisW, colBg)
	}

	content := lipgloss.JoinVertical(lipgloss.Left, rows...)

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colAccent).
		BorderBackground(colBg). // ← critical: border gaps use colBg not black
		Background(colBg).
		Render(content)
}

// ─────────────────────────────────────────────
//  Panel helpers
// ─────────────────────────────────────────────

const panelInnerW = 14 // chars inside panel border (not counting border itself)

func buildPanel(lines []string) string {
	padded := make([]string, len(lines))
	for i, l := range lines {
		padded[i] = padRight(l, panelInnerW, colPanelBg)
	}
	content := lipgloss.JoinVertical(lipgloss.Left, padded...)
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colBorder).
		BorderBackground(colBg). // ← border gaps use colBg not black
		Background(colPanelBg).
		Render(content)
}

func panelLabel(s string) string {
	return lipgloss.NewStyle().Foreground(colMid).Background(colPanelBg).Bold(true).Render(s)
}

func panelValue(s string) string {
	return lipgloss.NewStyle().Foreground(colAccentHi).Background(colPanelBg).Bold(true).Render(s)
}

func panelSpacer() string {
	return lipgloss.NewStyle().Background(colPanelBg).Render(" ")
}

func keyLine(key, label string) string {
	k := lipgloss.NewStyle().
		Foreground(colBg).Background(colAccent).Bold(true).Padding(0, 1).
		Render(key)
	lbl := lipgloss.NewStyle().Foreground(colMid).Background(colPanelBg).
		Render(" " + label)
	return lipgloss.JoinHorizontal(lipgloss.Center, k, lbl)
}

func (m Model) buildPreview(t Tetromino, dimmed bool) []string {
	grid := [4][4]string{}
	for y := range grid {
		for x := range grid[y] {
			grid[y][x] = previewEmpty()
		}
	}
	offY := (4 - len(t.cells)) / 2
	offX := (4 - len(t.cells[0])) / 2
	for y, row := range t.cells {
		for x, filled := range row {
			if filled {
				c := t.color
				if dimmed {
					c = colDim
				}
				grid[offY+y][offX+x] = previewFilled(c)
			}
		}
	}
	rows := make([]string, 4)
	for y := 0; y < 4; y++ {
		var line string
		for x := 0; x < 4; x++ {
			line += grid[y][x]
		}
		rows[y] = padRight(line, panelInnerW, colPanelBg)
	}
	return rows
}

func (m Model) buildEmptyPreview() []string {
	rows := make([]string, 4)
	for y := 0; y < 4; y++ {
		var line string
		for x := 0; x < 4; x++ {
			line += previewEmpty()
		}
		rows[y] = padRight(line, panelInnerW, colPanelBg)
	}
	return rows
}

func (m Model) buildLevelBar() string {
	filled := m.lines % 10
	bar := ""
	for i := 0; i < 10; i++ {
		if i < filled {
			bar += lipgloss.NewStyle().Foreground(colAccentHi).Background(colPanelBg).Render("█")
		} else {
			bar += lipgloss.NewStyle().Foreground(colDim).Background(colPanelBg).Render("░")
		}
	}
	return padRight("  "+bar, panelInnerW, colPanelBg)
}

// ─────────────────────────────────────────────
//  Left panel
// ─────────────────────────────────────────────

func (m Model) buildLeft() string {
	lines := []string{panelLabel("  HOLD")}
	if m.held != nil {
		lines = append(lines, m.buildPreview(*m.held, !m.canHold)...)
		if !m.canHold {
			lines = append(lines, padRight(
				lipgloss.NewStyle().Foreground(colDim).Background(colPanelBg).Render("  (used)"),
				panelInnerW, colPanelBg))
		} else {
			lines = append(lines, panelSpacer())
		}
	} else {
		lines = append(lines, m.buildEmptyPreview()...)
		lines = append(lines, panelSpacer())
	}
	lines = append(lines, panelSpacer())
	lines = append(lines, panelLabel("  PROGRESS"))
	lines = append(lines, m.buildLevelBar())
	lines = append(lines, panelSpacer())
	lines = append(lines, panelLabel("  SCORE"))
	lines = append(lines, panelValue(fmt.Sprintf("  %d", m.score)))
	lines = append(lines, panelSpacer())
	return buildPanel(lines)
}

// ─────────────────────────────────────────────
//  Right panel
// ─────────────────────────────────────────────

func (m Model) buildRight() string {
	lines := []string{panelLabel("  NEXT")}
	lines = append(lines, m.buildPreview(m.next, false)...)
	lines = append(lines, panelSpacer())
	lines = append(lines, panelLabel("  LINES"))
	lines = append(lines, panelValue(fmt.Sprintf("  %d", m.lines)))
	lines = append(lines, panelSpacer())
	lines = append(lines, panelLabel("  LEVEL"))
	lines = append(lines, panelValue(fmt.Sprintf("  %d", m.level)))
	lines = append(lines, panelSpacer())
	lines = append(lines, panelLabel("  KEYS"))
	lines = append(lines, keyLine("←→", "Move"))
	lines = append(lines, keyLine("↑ ", "Rotate"))
	lines = append(lines, keyLine("↓ ", "Soft↓"))
	lines = append(lines, keyLine("X ", "Drop"))
	lines = append(lines, keyLine("C ", "Hold"))
	lines = append(lines, keyLine("P ", "Pause"))
	lines = append(lines, panelSpacer())
	return buildPanel(lines)
}
