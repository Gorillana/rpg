package game

type Monster struct {
	Pos
	Rune rune
	Character
}

// as opposed to Rat in Jack's video
func NewBat(p Pos) *Monster {
	monster := &Monster{}
	monster.Pos = p
	monster.Rune = 'B'
	monster.Name = "Bat"
	monster.Hitpoints = 50
	monster.Strength = 1
	monster.Speed = 1.5
	monster.ActionPoints = 0.0
	monster.SightRange = 10
	return monster
}

func (m *Monster) Kill(level *Level) {
	delete(level.Monsters, m.Pos)
	groundItems := level.Items[m.Pos]
	for _, item := range m.Items {
		item.Pos = m.Pos
		groundItems = append(groundItems, item)
	}
	level.Items[m.Pos] = groundItems
}

func NewSpider(p Pos) *Monster {
	//return &Monster{p, 'S', "Spider", 10, 10, 1.0, 0.0}
	monster := &Monster{}
	monster.Pos = p
	monster.Rune = 'S'
	monster.Name = "Spider"
	monster.Hitpoints = 100
	monster.Strength = 5
	monster.Speed = 1.1
	monster.ActionPoints = 0.0
	monster.SightRange = 10
	return monster
}

// New Boss monster added
func NewDragon(p Pos) *Monster {
	monster := &Monster{}
	monster.Pos = p
	monster.Rune = 'D'
	monster.Name = "Dragon"
	monster.Hitpoints = 300
	monster.Strength = 100
	monster.Speed = 0.8
	monster.ActionPoints = 0.0
	monster.SightRange = 5
	return monster
}

func (m *Monster) Update(level *Level) {
	m.ActionPoints += m.Speed
	playerPos := level.Player.Pos
	apInt := int(m.ActionPoints)
	positions := level.aStar(m.Pos, playerPos)

	// do we have any path to the goal?
	if len(positions) == 0 {
		m.Pass()
		return
	}

	moveIndex := 1

	for i := 0; i < apInt; i++ {
		if moveIndex < len(positions) {
			m.Move(positions[moveIndex], level)
			moveIndex++
			m.ActionPoints--
		}
	}
}

func (m *Monster) Pass() {
	m.ActionPoints -= m.Speed
}

func (m *Monster) Move(to Pos, level *Level) {
	_, exists := level.Monsters[to]

	// if there's a monster/player in the way
	if !exists && to != level.Player.Pos {
		delete(level.Monsters, m.Pos)
		level.Monsters[to] = m
		m.Pos = to
		return
	}

	if to == level.Player.Pos {
		level.AddEvent(m.Name + " Attacks Player")
		level.Attack(&m.Character, &level.Player.Character)
		if m.Hitpoints <= 0 {
			delete(level.Monsters, m.Pos)
		}
		if level.Player.Hitpoints <= 0 {
			//level.AddEvent("You have died")
			panic("ded")
		}
	}

}

// inspired by Jack Mott on Youtube's GamewithGo series
