package types

const (
	// Rows is the number of rows of cards in Codenames.
	Rows = 5
	// Columns is the number of columns of cards in Codenames.
	Columns = 5
	// Size is the total number of cards on a Codenames board.
	Size = Rows * Columns
)

// Board contains all of the information about a game of Codenames.
type Board struct {
	// Names is a list of the 25 words on the board.
	Codenames []Codename
}

// Codename is a single game card, and its corresponding affiliation.
type Codename struct {
	Name string
	// Team isn't populated by anything at the start, unless you're the
	// Spymaster.
	Team Team
}

// Team is the affiliation of a codename.
type Team int

const (
	// NoTeam means the codename doesn't belong to any agent (yet).
	NoTeam Team = iota
	// RedTeam means the codename belongs to an agent on the red team.
	RedTeam
	// BlueTeam means the codename belongs to an agent on the blue team.
	BlueTeam
	// Assassin means the codename belongs to the assassin.
	Assassin
)

// Role is what type of player in the game you are.
type Role int

const (
	// NoRole means the codename doesn't belong to any agent.
	NoRole Role = iota
	// Spymaster is the person giving the hints
	Spymaster
	// Operative is a field agent on a team, anyone who guesses the codenames on the board.
	Operative
)
