package game

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Game struct {
	LevelChans   []chan *Level
	InputChan    chan *Input
	Levels       map[string]*Level
	CurrentLevel *Level
}

func NewGame(numWindows int) *Game {
	//if we're going to have some number of windows we're going to make one level channel for each window
	levelChans := make([]chan *Level, numWindows)
	for i := range levelChans {
		levelChans[i] = make(chan *Level)
	}
	inputChan := make(chan *Input)
	levels := loadLevels()

	game := &Game{levelChans, inputChan, levels, nil}
	game.loadWorldFile()
	game.CurrentLevel.lineOfSight()
	return game
}

type InputType int

const (
	None InputType = iota
	Up
	Down
	Left
	Right
	TakeAll
	TakeItem
	DropItem
	EquipItem
	QuitGame
	CloseWindow
	MouseClick
	Search // temp
)

// Tagged / Discriminatory Union / Sum Type
type Input struct {
	Typ          InputType
	Item         *Item
	LevelChannel chan *Level
}

// tile is alias for rune, lets us create an enum
type Tile struct {
	Rune        rune
	OverlayRune rune
	Visible     bool
	Seen        bool
}

const (
	StoneWall rune = '#'
	DirtFloor      = '.'
	CloseDoor      = '|'
	OpenDoor       = '/'
	UpStair        = 'u'
	DownStair      = 'd'
	Blank          = 0
	Pending        = -1
)

type Pos struct {
	X, Y int
}

type LevelPos struct {
	*Level
	Pos
}

type Entity struct {
	Pos
	Name string
	Rune rune
}

type Character struct {
	Entity
	Hitpoints    int
	Strength     int
	Speed        float64
	ActionPoints float64
	SightRange   int
	Items        []*Item
	Helmet       *Item
	Weapon       *Item
}

type Player struct {
	Character
}

type GameEvent int

const (
	Move GameEvent = iota
	DoorOpen
	Attack
	Hit
	Portal
	Pickup
	Drop
)

type Level struct {
	Map       [][]Tile
	Player    *Player
	Monsters  map[Pos]*Monster
	Items     map[Pos][]*Item
	Portals   map[Pos]*LevelPos
	Events    []string
	EventPos  int
	Debug     map[Pos]bool
	LastEvent GameEvent
}

func (level *Level) DropItem(itemToDrop *Item, character *Character) {
	pos := character.Pos
	items := character.Items

	for i, item := range items {
		if item == itemToDrop {
			character.Items = append(character.Items[:i], character.Items[i+1:]...)
			level.Items[pos] = append(level.Items[pos], item)
			level.AddEvent(character.Name + " dropped: " + item.Name)
			return
		}
	}

}

func (level *Level) MoveItem(itemToMove *Item, character *Character) {
	fmt.Println("Move Item!")
	pos := character.Pos
	items := level.Items[pos]

	for i, item := range items {
		if item == itemToMove {
			items = append(items[:i], items[i+1:]...)
			level.Items[pos] = items
			character.Items = append(character.Items, item)
			level.AddEvent(character.Name + " picked up: " + item.Name)
			return
		}
	}
	panic("Tried to move an item we were not on top of")
}

// Combine the two attack functions into 1 somehow
func (level *Level) Attack(c1, c2 *Character) {
	c1.ActionPoints--
	// base strength
	c1AttackPower := c1.Strength
	// with weapon
	if c1.Weapon != nil {
		c1AttackPower = int(float64(c1AttackPower) * c1.Weapon.power)
	}
	damage := c1AttackPower

	if c2.Helmet != nil {
		damage = int(float64(damage) * (1.0 - c2.Helmet.power))
	}
	c2.Hitpoints -= damage

	if c2.Hitpoints > 0 {
		level.AddEvent(c1.Name + "Attacked" + c2.Name + " for " + strconv.Itoa(damage))
	} else {
		level.AddEvent(c1.Name + " Killed " + c2.Name)
	}
}

func (level *Level) AddEvent(event string) {
	// initializes events to 0 starting point
	level.Events[level.EventPos] = event
	//increment pos
	level.EventPos++
	//wraparound
	if level.EventPos == len(level.Events) {
		level.EventPos = 0
	}
}

func (level *Level) lineOfSight() {
	pos := level.Player.Pos
	dist := level.Player.SightRange

	for y := pos.Y - dist; y <= pos.Y+dist; y++ {
		for x := pos.X - dist; x <= pos.X+dist; x++ {
			xDelta := pos.X - x
			yDelta := pos.Y - y
			d := math.Sqrt(float64(xDelta*xDelta + yDelta*yDelta))
			if d <= float64(dist) {
				level.bresenham(pos, Pos{x, y})
			}
		}
	}
}

// Reversing the order of the results when necessary
func (level *Level) bresenham(start Pos, end Pos) {
	// make([]slice) - allocating memory
	steep := math.Abs(float64(end.Y-start.Y)) > math.Abs(float64(end.X-start.X))

	if steep {
		start.X, start.Y = start.Y, start.X
		end.X, end.Y = end.Y, end.X
	}
	deltaY := int(math.Abs(float64(end.Y - start.Y)))
	err := 0

	y := start.Y
	ystep := 1
	if start.Y >= end.Y {
		ystep = -1
	}

	if start.X > end.X {

		deltaX := start.X - end.X

		for x := start.X; x > end.X; x-- {
			var pos Pos
			if steep {
				pos = Pos{y, x}
			} else {
				pos = Pos{x, y}
			}
			level.Map[pos.Y][pos.X].Visible = true
			level.Map[pos.Y][pos.X].Seen = true
			if !canSeeThrough(level, pos) {
				return
			}
			err += deltaY
			if 2*err >= deltaX {
				y += ystep
				err -= deltaX
			}
		}
	} else {

		deltaX := end.X - start.X
		for x := start.X; x < end.X; x++ {
			var pos Pos
			if steep {
				pos = Pos{y, x}
			} else {
				pos = Pos{x, y}
			}
			level.Map[pos.Y][pos.X].Visible = true
			level.Map[pos.Y][pos.X].Seen = true
			if !canSeeThrough(level, pos) {
				return
			}
			err += deltaY
			if 2*err >= deltaX {
				y += ystep
				err -= deltaX
			}
		}
	}
}

func (game *Game) loadWorldFile() {
	file, err := os.Open("game/maps/world.txt")
	if err != nil {
		panic(err)
	}
	csvReader := csv.NewReader(file)
	csvReader.FieldsPerRecord = -1
	csvReader.TrimLeadingSpace = true
	rows, err := csvReader.ReadAll()
	if err != nil {
		panic(err)
	}

	for rowIndex, row := range rows {
		// set current level
		if rowIndex == 0 {
			game.CurrentLevel = game.Levels[row[0]]
			if game.CurrentLevel == nil {
				fmt.Println("couldn't find currentlevel name in world file")
				panic(nil)
			}
			continue
		}
		levelWithPortal := game.Levels[row[0]]
		if levelWithPortal == nil {
			fmt.Println("couldn't find level name 1 in world file")
			panic(nil)
		}

		x, err := strconv.ParseInt(row[1], 10, 64)
		if err != nil {
			panic(err)
		}

		y, err := strconv.ParseInt(row[2], 10, 64)
		if err != nil {
			panic(err)
		}
		pos := Pos{int(x), int(y)}

		levelToTeleportTo := game.Levels[row[3]]
		if levelWithPortal == nil {
			fmt.Println("couldn't find level name 2 in world file")
			panic(nil)
		}

		x, err = strconv.ParseInt(row[4], 10, 64)
		if err != nil {
			panic(err)
		}

		y, err = strconv.ParseInt(row[5], 10, 64)
		if err != nil {
			panic(err)
		}
		posToTeleportTo := Pos{int(x), int(y)}
		levelWithPortal.Portals[pos] = &LevelPos{levelToTeleportTo, posToTeleportTo}
	}

}

// Todo take in a path
func loadLevels() map[string]*Level {

	player := &Player{}
	// TODO where should we initialize the player?
	player.Strength = 5
	player.Hitpoints = 20
	player.Name = "GOrillana"
	player.Rune = '@'
	player.Speed = 1.0
	player.ActionPoints = 0.0
	player.SightRange = 7

	levels := make(map[string]*Level)

	filenames, err := filepath.Glob("game/maps/*.map")
	if err != nil {
		panic("file error")
	}

	for _, filename := range filenames {
		fmt.Println("loading:", filename)
		extIndex := strings.LastIndex(filename, ".map")
		lastSlashIndex := strings.LastIndex(filename, "\\")
		levelName := filename[lastSlashIndex+1 : extIndex]
		fmt.Println("name: ", levelName)
		file, err := os.Open(filename)
		if err != nil {
			panic(err)
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		levelLines := make([]string, 0)
		longestRow := 0
		index := 0
		for scanner.Scan() {

			// array will have string for each row
			levelLines = append(levelLines, scanner.Text())
			if len(levelLines[index]) > longestRow {
				longestRow = len(levelLines[index])
			}
			index++
		}

		// level set to a blank/zeroed out level
		level := &Level{}
		level.Debug = make(map[Pos]bool)
		level.Events = make([]string, 10)
		level.Player = player
		level.Map = make([][]Tile, len(levelLines))
		// init monsters
		level.Monsters = make(map[Pos]*Monster)
		level.Items = make(map[Pos][]*Item)
		level.Portals = make(map[Pos]*LevelPos)

		// go through each row and make an array for the row
		for i := range level.Map {
			level.Map[i] = make([]Tile, longestRow)
		}

		// render tiles
		for y := 0; y < len(level.Map); y++ {
			line := levelLines[y]
			for x, c := range line {
				pos := Pos{x, y}
				var t Tile
				t.OverlayRune = Blank
				switch c {
				case ' ', '\t', '\n', '\r':
					t.Rune = Blank
				case '#':
					t.Rune = StoneWall
				case '|':
					t.OverlayRune = CloseDoor
					t.Rune = Pending
				case '/':
					t.OverlayRune = OpenDoor
					t.Rune = Pending
				case 'u':
					t.OverlayRune = UpStair
					t.Rune = Pending
				case 'd':
					t.OverlayRune = DownStair
					t.Rune = Pending
				case 's':
					level.Items[pos] = append(level.Items[pos], NewSword(pos))
					t.Rune = Pending
				case 'h':
					level.Items[pos] = append(level.Items[pos], NewHelmet(pos))
					t.Rune = Pending
				case '.':
					t.Rune = DirtFloor
				case '@':
					level.Player.X = x
					level.Player.Y = y
					t.Rune = Pending
				case 'B':
					level.Monsters[pos] = NewBat(pos)
					t.Rune = Pending
				case 'D':
					level.Monsters[pos] = NewDragon(pos)
					t.Rune = Pending
				case 'S':
					level.Monsters[pos] = NewSpider(pos)
					t.Rune = Pending
				default:
					panic("Invalid character in map")
				}
				level.Map[y][x] = t
			}
		}

		// we should use bfs to find first floor tile
		// go over map again (draw order)
		for y, row := range level.Map {
			for x, tile := range row {
				// fill in the player/pending square with similar tiles
				if tile.Rune == Pending {
					level.Map[y][x].Rune = level.bfsFloor(Pos{x, y})
				}
			}
		}
		levels[levelName] = level
	}
	return levels
}

func inRange(level *Level, pos Pos) bool {
	return pos.X < len(level.Map[0]) && pos.Y < len(level.Map) && pos.X >= 0 && pos.Y >= 0

}

// added function to simplify handleInput
// checks to see if a tile can be walked on
func canWalk(level *Level, pos Pos) bool {
	if inRange(level, pos) {
		t := level.Map[pos.Y][pos.X]
		switch t.Rune {
		case StoneWall, Blank:
			return false
		}
		switch t.OverlayRune {
		case CloseDoor:
			return false
		}
		_, exists := level.Monsters[pos]
		if exists {
			return false
		}
		return true
	}
	return false
}

func canSeeThrough(level *Level, pos Pos) bool {
	if inRange(level, pos) {
		t := level.Map[pos.Y][pos.X]
		switch t.Rune {
		case StoneWall, Blank:
			return false
		}
		switch t.OverlayRune {
		case CloseDoor:
			return false
		default:
			return true
		}
	}
	return false
}

func checkDoor(level *Level, pos Pos) {
	t := level.Map[pos.Y][pos.X]
	if t.OverlayRune == CloseDoor {
		level.Map[pos.Y][pos.X].OverlayRune = OpenDoor
		level.LastEvent = DoorOpen
		level.lineOfSight()
	}
}

func (game *Game) Move(to Pos) {
	level := game.CurrentLevel
	player := level.Player

	levelAndPos := level.Portals[to]
	if levelAndPos != nil {
		game.CurrentLevel = levelAndPos.Level
		game.CurrentLevel.Player.Pos = levelAndPos.Pos
	} else {
		player.Pos = to
		level.LastEvent = Move
		for y, row := range level.Map {
			for x := range row {
				level.Map[y][x].Visible = false
			}
		}

		level.lineOfSight()
		fmt.Println("Player:", player.Pos)
	}
}


func (game *Game) resolveMovement(pos Pos) {
	level := game.CurrentLevel
	monster, exists := level.Monsters[pos]
	if exists {
		level.Attack(&level.Player.Character, &monster.Character)
		level.LastEvent = Attack
		// monster dies
		if monster.Hitpoints <= 0 {
			monster.Kill(level)
		}
		// player dies
		if level.Player.Hitpoints <= 0 {
			//ui.DrawDeathRect("You have died")
			panic("ded")
		}
	} else if canWalk(level, pos) {
		game.Move(pos)
	} else {
		checkDoor(level, pos)
	}
}

func equip(c *Character, itemtoEquip *Item) {
	for i, item := range c.Items {
		if item == itemtoEquip {
			c.Items = append(c.Items[:i], c.Items[i+1:]...)
			if itemtoEquip.Typ == Helmet {
				c.Helmet = itemtoEquip
			} else if itemtoEquip.Typ == Weapon {
				c.Weapon = itemtoEquip
			}
			return
		}
	}
	panic("someone tried to equip a thing they don't have")
}

// allows user to use d-pad to move character
func (game *Game) handleInput(input *Input) {
	level := game.CurrentLevel
	p := level.Player
	switch input.Typ {
	case Up:
		newPos := Pos{p.X, p.Y - 1}
		game.resolveMovement(newPos)
	case Down:
		newPos := Pos{p.X, p.Y + 1}
		game.resolveMovement(newPos)
	case Left:
		newPos := Pos{p.X - 1, p.Y}
		game.resolveMovement(newPos)
	case Right:
		newPos := Pos{p.X + 1, p.Y}
		game.resolveMovement(newPos)
	case TakeAll:
		for _, item := range level.Items[p.Pos] {
			level.MoveItem(item, &level.Player.Character)
		}
		level.LastEvent = Pickup
	case TakeItem:
		level.MoveItem(input.Item, &level.Player.Character)
		level.LastEvent = Pickup
	case EquipItem:
		equip(&level.Player.Character, input.Item)
	case DropItem:
		level.DropItem(input.Item, &level.Player.Character)
		level.LastEvent = Drop

	case CloseWindow:
		close(input.LevelChannel)
		chanIndex := 0
		for i, c := range game.LevelChans {
			if c == input.LevelChannel {
				chanIndex = i
				break
			}
		}
		game.LevelChans = append(game.LevelChans[:chanIndex], game.LevelChans[chanIndex+1:]...)
	}
}

func getNeighbors(level *Level, pos Pos) []Pos {

	neighbors := make([]Pos, 0, 4)

	// look at player adjacent tiles
	left := Pos{pos.X - 1, pos.Y}
	right := Pos{pos.X + 1, pos.Y}
	up := Pos{pos.X, pos.Y - 1}
	down := Pos{pos.X, pos.Y + 1}

	if canWalk(level, right) {
		neighbors = append(neighbors, right)
	}
	if canWalk(level, left) {
		neighbors = append(neighbors, left)
	}
	if canWalk(level, up) {
		neighbors = append(neighbors, up)
	}
	if canWalk(level, down) {
		neighbors = append(neighbors, down)
	}

	return neighbors
}

// breadth-first search
func (level *Level) bfsFloor(start Pos) rune {
	//0 - how big to make slice at start
	//8 - optional, how much space to reserve; reserves 8 spaces
	frontier := make([]Pos, 0, 8)
	frontier = append(frontier, start)
	visited := make(map[Pos]bool)
	visited[start] = true

	for len(frontier) > 0 {
		current := frontier[0]
		currentTile := level.Map[current.Y][current.X]
		switch currentTile.Rune {
		// finds closest foor tile to something's that got a char/monst on top of it
		case DirtFloor:
			return DirtFloor
		default:
		}

		// start at second element to the end
		frontier = frontier[1:]
		for _, next := range getNeighbors(level, current) {
			if !visited[next] {
				frontier = append(frontier, next)
				visited[next] = true
			}
		}
	}
	return DirtFloor
}

func (level *Level) aStar(start Pos, goal Pos) []Pos {
	// priority queue
	frontier := make(pqueue, 0, 8)
	frontier = frontier.push(start, 1)
	cameFrom := make(map[Pos]Pos)
	cameFrom[start] = start
	costSoFar := make(map[Pos]int)
	costSoFar[start] = 0

	var current Pos
	for len(frontier) > 0 {

		frontier, current = frontier.pop()

		if current == goal {
			path := make([]Pos, 0)
			p := current
			for p != start {
				path = append(path, p)
				p = cameFrom[p]
			}
			path = append(path, p)

			for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
				path[i], path[j] = path[j], path[i]
			}

			return path
		}

		// second var after
		for _, next := range getNeighbors(level, current) {
			newCost := costSoFar[current] + 1 // always 1 for now
			// The thing, "exist" will verify if it's there or not; blank shows we don't care what "the thing" is but that it exists
			_, exists := costSoFar[next]
			if !exists || newCost < costSoFar[next] {
				costSoFar[next] = newCost
				xDist := int(math.Abs(float64(goal.X - next.X)))
				yDist := int(math.Abs(float64(goal.Y - next.Y)))
				priority := newCost + xDist + yDist
				frontier = frontier.push(next, priority)
				cameFrom[next] = current

			}
		}
	}
	return nil
}

// loads up game, called in main
func (game *Game) Run() {
	fmt.Println("Starting...")

	count := 0
	// go through all level channels and sent current game level
	for _, lchan := range game.LevelChans {
		lchan <- game.CurrentLevel
	}

	// infinite loop to run as long as we need
	for input := range game.InputChan {
		// check for quit
		if input.Typ == QuitGame {
			return
		}

		game.handleInput(input)

		//game.Level.AddEvent("Move: " + strconv.Itoa(count))
		count++

		for _, monster := range game.CurrentLevel.Monsters {
			monster.Update(game.CurrentLevel)
		}

		// if number of level channels is 0, all windows are closed, so quit
		if len(game.LevelChans) == 0 {
			return
		}
		for _, lchan := range game.LevelChans {
			lchan <- game.CurrentLevel
		}
	}

}

// inspired by Jack Mott on Youtube's GamewithGo series
