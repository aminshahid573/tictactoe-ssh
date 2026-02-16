package snake

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ─────────────────────────────────────────────────────────────────
//  CRITICAL: There is ONE ticker for the snake game lifetime.
//  It is started once when entering the game and NEVER started again.
//  Restarts only reset game state — they do NOT spawn a new ticker.
// ─────────────────────────────────────────────────────────────────

const uiTick = 60 * time.Millisecond

// Snake moves once every N ui-ticks (60ms base)
// Easy=360ms, Normal=240ms, Hard=120ms per step
var diffMoveEvery = [3]int{6, 4, 2}
var diffNames = [3]string{"Easy", "Normal", "Hard"}
var diffColors = [3]lipgloss.Color{"#44dd88", "#f0e040", "#ff4444"}

// ── Palette ──────────────────────────────────
var (
	colorBorder     = lipgloss.Color("#7b2fff")
	colorSnakeHead  = lipgloss.Color("#00ffcc")
	colorSnakeBody1 = lipgloss.Color("#00d4a8")
	colorSnakeBody2 = lipgloss.Color("#009e7e")
	colorFood       = lipgloss.Color("#ff2d78")
	colorFoodGlow   = lipgloss.Color("#ff6fa8")
	colorScore      = lipgloss.Color("#f0e040")
	colorSubtitle   = lipgloss.Color("#7b2fff")
	colorDead       = lipgloss.Color("#ff4040")
	colorDim        = lipgloss.Color("#3a3a5c")
	colorGhost      = lipgloss.Color("#2a2a45")
)

// ─────────────────────────────────────────────
//  Types
// ─────────────────────────────────────────────

// Point represents a coordinate on the game board.
type Point struct{ X, Y int }

// Direction represents the movement direction.
type Direction int

const (
	DirUp Direction = iota
	DirDown
	DirLeft
	DirRight
)

// GameState represents the state of the snake game.
type GameState int

const (
	StateMenu GameState = iota
	StatePlaying
	StatePaused
	StateGameOver
)

// TickMsg is the message sent on each UI tick.
type TickMsg struct{}

// ─────────────────────────────────────────────
//  Model
// ─────────────────────────────────────────────

// Model holds all state for the snake game.
type Model struct {
	TermW, TermH int

	// game board
	snake     []Point
	dir       Direction
	nextDir   Direction
	food      Point
	score     int
	highscore int
	State     GameState
	diff      int // 0=Easy 1=Normal 2=Hard

	// animation / movement counters
	uiFrame  int // incremented every uiTick
	moveAccu int // snake steps when this reaches diffMoveEvery[diff]
	foodAnim int

	// menu
	menuSel int

	// Whether the player wants to quit back to the game-select screen
	WantsQuit bool

	rng *rand.Rand
}

const boardW, boardH = 30, 20

// buildSnake resets only game-board state, keeping meta fields intact.
func (m *Model) buildSnake() {
	m.snake = []Point{
		{boardW/2 + 1, boardH / 2},
		{boardW / 2, boardH / 2},
		{boardW/2 - 1, boardH / 2},
	}
	m.dir = DirRight
	m.nextDir = DirRight
	m.score = 0
	m.moveAccu = 0
	m.food = m.spawnFood()
}

// InitialModel creates a fresh snake game model.
func InitialModel() Model {
	m := Model{
		State:   StateMenu,
		menuSel: 1,
		rng:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}
	m.buildSnake()
	return m
}

func (m *Model) spawnFood() Point {
	occ := map[Point]bool{}
	for _, p := range m.snake {
		occ[p] = true
	}
	for {
		p := Point{m.rng.Intn(boardW), m.rng.Intn(boardH)}
		if !occ[p] {
			return p
		}
	}
}

// ─────────────────────────────────────────────
//  Tick command — the ONE and ONLY ticker source
// ─────────────────────────────────────────────

// TickCmd creates a command that sends the next tick.
func TickCmd() tea.Cmd {
	return tea.Tick(uiTick, func(_ time.Time) tea.Msg { return TickMsg{} })
}

// ─────────────────────────────────────────────
//  Update
// ─────────────────────────────────────────────

// Update processes messages for the snake game. Returns the updated model
// and any commands that should be executed.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:
		k := msg.String()
		m.handleKey(k)
		return m, nil

	case TickMsg:
		// ── advance animation ──
		m.uiFrame++
		m.foodAnim = (m.uiFrame / 2) % 4

		// ── advance snake (only while playing) ──
		if m.State == StatePlaying {
			m.moveAccu++
			if m.moveAccu >= diffMoveEvery[m.diff] {
				m.moveAccu = 0
				if m.stepSnake() {
					m.State = StateGameOver
				}
			}
		}

		// Reschedule the ticker
		return m, TickCmd()
	}

	return m, nil
}

// handleKey is a plain method (no tea.Cmd) so it can never accidentally
// spawn a new ticker.
func (m *Model) handleKey(k string) {
	switch m.State {

	case StateMenu:
		switch k {
		case "up", "w", "k":
			m.menuSel = (m.menuSel + 2) % 3
		case "down", "s", "j":
			m.menuSel = (m.menuSel + 1) % 3
		case "1":
			m.menuSel = 0
		case "2":
			m.menuSel = 1
		case "3":
			m.menuSel = 2
		case "enter", " ":
			m.diff = m.menuSel
			m.buildSnake()
			m.State = StatePlaying
		case "q", "esc":
			m.WantsQuit = true
		}

	case StatePlaying:
		switch k {
		case "up", "w", "k":
			if m.dir != DirDown {
				m.nextDir = DirUp
			}
		case "down", "s", "j":
			if m.dir != DirUp {
				m.nextDir = DirDown
			}
		case "left", "a", "h":
			if m.dir != DirRight {
				m.nextDir = DirLeft
			}
		case "right", "d", "l":
			if m.dir != DirLeft {
				m.nextDir = DirRight
			}
		case "p", "escape":
			m.State = StatePaused
		}

	case StatePaused:
		switch k {
		case "p", "escape", "enter":
			m.State = StatePlaying
		case "q":
			m.WantsQuit = true
		}

	case StateGameOver:
		switch k {
		case "enter", " ", "r":
			m.buildSnake()
			m.State = StatePlaying
		case "m":
			hs := m.highscore
			tw, th := m.TermW, m.TermH
			*m = InitialModel()
			m.highscore = hs
			m.TermW, m.TermH = tw, th
		case "q":
			m.WantsQuit = true
		}
	}
}

// stepSnake advances the snake one cell. Returns true if the snake died.
func (m *Model) stepSnake() bool {
	m.dir = m.nextDir
	head := m.snake[0]
	switch m.dir {
	case DirUp:
		head.Y--
	case DirDown:
		head.Y++
	case DirLeft:
		head.X--
	case DirRight:
		head.X++
	}

	if head.X < 0 || head.X >= boardW || head.Y < 0 || head.Y >= boardH {
		return true
	}
	for _, p := range m.snake {
		if p == head {
			return true
		}
	}

	ate := head == m.food
	ns := make([]Point, 0, len(m.snake)+1)
	ns = append(ns, head)
	if ate {
		ns = append(ns, m.snake...)
		m.score += 10
		if m.score > m.highscore {
			m.highscore = m.score
		}
		m.food = m.spawnFood()
	} else {
		ns = append(ns, m.snake[:len(m.snake)-1]...)
	}
	m.snake = ns
	return false
}

// ─────────────────────────────────────────────
//  Styles
// ─────────────────────────────────────────────

func outerBox() lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorBorder).
		Padding(0, 1)
}

var (
	subtitleSty = lipgloss.NewStyle().Foreground(colorSubtitle)
	scoreSty    = lipgloss.NewStyle().Foreground(colorScore).Bold(true)
	dimSty      = lipgloss.NewStyle().Foreground(colorDim)
	deadSty     = lipgloss.NewStyle().Foreground(colorDead).Bold(true)
	pauseSty    = lipgloss.NewStyle().Foreground(lipgloss.Color("#ffdd44")).Bold(true)
	keySty      = lipgloss.NewStyle().Foreground(lipgloss.Color("#aaaaee")).Bold(true)
	helpSty     = lipgloss.NewStyle().Foreground(colorDim)
)

// ─────────────────────────────────────────────
//  Cell renderers
// ─────────────────────────────────────────────

func snakeCell(idx int, alive bool) string {
	if !alive {
		return lipgloss.NewStyle().Foreground(colorDead).Render("██")
	}
	if idx == 0 {
		return lipgloss.NewStyle().Foreground(colorSnakeHead).Render("██")
	}
	if idx%2 == 0 {
		return lipgloss.NewStyle().Foreground(colorSnakeBody1).Render("██")
	}
	return lipgloss.NewStyle().Foreground(colorSnakeBody2).Render("██")
}

func foodCell(anim int) string {
	glyphs := [4]string{"◆", "◈", "◇", "◈"}
	colors := [4]lipgloss.Color{colorFood, colorFoodGlow, colorFood, colorFoodGlow}
	return lipgloss.NewStyle().Foreground(colors[anim]).Bold(true).Render(" " + glyphs[anim])
}

func emptyCell() string {
	return lipgloss.NewStyle().Foreground(colorGhost).Render("··")
}

// ─────────────────────────────────────────────
//  View
// ─────────────────────────────────────────────

// View renders the snake game.
func (m Model) View() string {
	var content string
	switch m.State {
	case StateMenu:
		content = m.renderMenu()
	default:
		content = m.renderGame()
	}
	return content
}

// ── Menu ──────────────────────────────────────

func (m Model) renderMenu() string {
	art := [6]string{
		" ███████╗███╗   ██╗ █████╗ ██╗  ██╗███████╗",
		" ██╔════╝████╗  ██║██╔══██╗██║ ██╔╝██╔════╝",
		" ███████╗██╔██╗ ██║███████║█████╔╝ █████╗  ",
		" ╚════██║██║╚██╗██║██╔══██║██╔═██╗ ██╔══╝  ",
		" ███████║██║ ╚████║██║  ██║██║  ██╗███████╗",
		" ╚══════╝╚═╝  ╚═══╝╚═╝  ╚═╝╚═╝  ╚═╝╚══════╝",
	}
	pal := [7]lipgloss.Color{
		"#c084fc", "#a855f7", "#9333ea", "#a855f7",
		"#c084fc", "#e879f9", "#c084fc",
	}
	animSty := lipgloss.NewStyle().Foreground(pal[m.uiFrame%7]).Bold(true)

	var sb strings.Builder
	sb.WriteString("\n\n")
	for _, line := range art {
		sb.WriteString(animSty.Render(line))
		sb.WriteString("\n")
	}
	sb.WriteString("\n")
	sb.WriteString(subtitleSty.Render("  ▸  A neon cyberpunk experience  ◂"))
	sb.WriteString("\n\n")

	// ── Vertical difficulty list ──────────────
	sb.WriteString(dimSty.Render("  SELECT DIFFICULTY"))
	sb.WriteString("\n")
	sb.WriteString(dimSty.Render("  ─────────────────"))
	sb.WriteString("\n")

	for i, name := range diffNames {
		if i == m.menuSel {
			bullet := lipgloss.NewStyle().
				Foreground(diffColors[i]).
				Bold(true)
			label := lipgloss.NewStyle().
				Foreground(diffColors[i]).
				Bold(true).
				Render("[" + name + "]")
			line := bullet.Render("  ▶ ") + label
			sb.WriteString(line)
		} else {
			plain := lipgloss.NewStyle().Foreground(colorDim)
			sb.WriteString(plain.Render(fmt.Sprintf("    %s", name)))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("\n")
	sb.WriteString(keySty.Render("  ↑ ↓ to choose   ENTER to start"))
	sb.WriteString("\n\n")
	sb.WriteString(helpSty.Render("  Move: WASD / Arrows / HJKL"))
	sb.WriteString("\n")
	sb.WriteString(helpSty.Render("  Pause: P    Quit: Q"))
	sb.WriteString("\n")

	return outerBox().Render(sb.String())
}

// ── Game ──────────────────────────────────────

func (m Model) renderGame() string {
	snakeSet := map[Point]int{}
	for i, p := range m.snake {
		snakeSet[p] = i
	}
	alive := m.State != StateGameOver

	var board strings.Builder
	for y := 0; y < boardH; y++ {
		for x := 0; x < boardW; x++ {
			p := Point{x, y}
			if idx, ok := snakeSet[p]; ok {
				board.WriteString(snakeCell(idx, alive))
			} else if p == m.food {
				board.WriteString(foodCell(m.foodAnim))
			} else {
				board.WriteString(emptyCell())
			}
		}
		if y < boardH-1 {
			board.WriteString("\n")
		}
	}

	diffBadge := lipgloss.NewStyle().
		Foreground(diffColors[m.diff]).
		Bold(true).
		Render("[" + diffNames[m.diff] + "]")

	scoreBar := fmt.Sprintf(" %s  %s    %s  %s    %s  %s",
		dimSty.Render("SCORE"), scoreSty.Render(fmt.Sprintf("%06d", m.score)),
		dimSty.Render("BEST"), scoreSty.Render(fmt.Sprintf("%06d", m.highscore)),
		dimSty.Render("MODE"), diffBadge,
	)

	var status string
	switch m.State {
	case StatePaused:
		status = pauseSty.Render("  ⏸  PAUSED — press P or ESC to resume")
	case StateGameOver:
		status = deadSty.Render("  ✕  GAME OVER  ") +
			helpSty.Render("[ENTER] restart   [M] menu   [Q] quit")
	default:
		status = helpSty.Render("  WASD/Arrows: move   P: pause   Q: quit")
	}

	boardRendered := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(colorBorder).
		Render(board.String())

	inner := lipgloss.JoinVertical(lipgloss.Left,
		scoreBar,
		boardRendered,
		status,
	)
	return outerBox().Render(inner)
}
